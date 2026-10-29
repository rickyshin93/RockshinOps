#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import "events_darwin.h"

extern void rockshin_menubar_action(char *action);

static id MBMain(id (^work)(void)) {
    if (NSThread.isMainThread) return work();
    __block id value;
    dispatch_sync(dispatch_get_main_queue(), ^{ value = work(); });
    return value;
}
static NSString *MBPNG(NSImage *image) {
    if (!image) return @"";
    NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:image.TIFFRepresentation];
    NSData *data = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    return data ? [@"data:image/png;base64," stringByAppendingString:[data base64EncodedStringWithOptions:0]] : @"";
}
static CGRect MBRect(NSDictionary *window) {
    CGRect r = CGRectZero;
    CGRectMakeWithDictionaryRepresentation((__bridge CFDictionaryRef)window[(__bridge NSString *)kCGWindowBounds], &r);
    return r;
}
static NSArray *MBWindows(void) {
    return CFBridgingRelease(CGWindowListCopyWindowInfo(kCGWindowListOptionAll, kCGNullWindowID)) ?: @[];
}
static BOOL MBIceRunning(void) {
    return [NSRunningApplication runningApplicationsWithBundleIdentifier:@"com.jordanbaird.Ice"].count > 0;
}
static NSString *MBText(NSDictionary *strings, NSString *key) { return strings[key] ?: key; }
static NSDictionary *MBMove(NSDictionary *payload);
static CGRect MBWindowFrame(CGWindowID wid);
static BOOL MBWaitForFrameChange(CGWindowID wid, CGRect initial);
static NSImage *MBControlImage(void) {
    static NSImage *image;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        image=[NSImage imageWithSize:NSMakeSize(20,18) flipped:NO drawingHandler:^BOOL(NSRect rect){
            [NSColor.blackColor setFill];[NSColor.blackColor setStroke];
            NSBezierPath *path=[NSBezierPath bezierPath];
            path.lineWidth=1;path.lineJoinStyle=NSLineJoinStyleRound;
            [path moveToPoint:NSMakePoint(2.5,9)];[path lineToPoint:NSMakePoint(9.5,3)];
            [path lineToPoint:NSMakePoint(9.5,6)];[path lineToPoint:NSMakePoint(17.5,6)];
            [path lineToPoint:NSMakePoint(17.5,12)];[path lineToPoint:NSMakePoint(9.5,12)];
            [path lineToPoint:NSMakePoint(9.5,15)];[path closePath];[path fill];[path stroke];
            return YES;
        }];
        image.template=YES;
    });
    return image;
}
static NSSet *MBVisibleWindowIDs(pid_t pid) {
    NSMutableSet *ids=[NSMutableSet new];
    for(NSDictionary *w in MBWindows())if([w[(__bridge NSString *)kCGWindowOwnerPID] intValue]==pid && [w[(__bridge NSString *)kCGWindowIsOnscreen] boolValue]) [ids addObject:w[(__bridge NSString *)kCGWindowNumber]];
    return ids;
}
static void MBSaveStatusItemPosition(NSStatusItem *item) {
    if(!item)return;
    NSString *key=[@"NSStatusItem Preferred Position " stringByAppendingString:item.autosaveName];
    id position=[NSUserDefaults.standardUserDefaults objectForKey:key];
    // AppKit can clear its own key after removal; keep the snapshot under our own key.
    if(position)[NSUserDefaults.standardUserDefaults setObject:position forKey:[@"RockshinOps.SavedPosition." stringByAppendingString:item.autosaveName]];
}
static NSStatusItem *MBCreateStatusItem(NSString *name, CGFloat length, NSNumber *initialPosition) {
    NSUserDefaults *defaults=NSUserDefaults.standardUserDefaults;
    NSString *key=[@"NSStatusItem Preferred Position " stringByAppendingString:name];
    NSString *savedKey=[@"RockshinOps.SavedPosition." stringByAppendingString:name];
    id saved=[defaults objectForKey:savedKey];
    if(saved)[defaults setObject:saved forKey:key];
    else if(initialPosition && ![defaults objectForKey:key])[defaults setObject:initialPosition forKey:key];
    NSStatusItem *item=[NSStatusBar.systemStatusBar statusItemWithLength:length];
    item.autosaveName=name;
    if(saved)[defaults removeObjectForKey:savedKey];
    return item;
}

@interface MBPanel : NSPanel
@property NSUInteger visibilityRevision;
- (void)present;
- (void)dismiss;
@end
@implementation MBPanel
- (BOOL)canBecomeKeyWindow { return YES; }
- (void)present {
    self.visibilityRevision++;
    if(!self.visible)self.alphaValue=0;
    [self makeKeyAndOrderFront:nil];
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context){
        context.duration=NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceMotion?0:0.12;
        self.animator.alphaValue=1;
    } completionHandler:nil];
}
- (void)dismiss {
    if(!self.visible)return;
    NSUInteger revision=++self.visibilityRevision;
    [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context){
        context.duration=NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceMotion?0:0.1;
        self.animator.alphaValue=0;
    } completionHandler:^{if(self.visibilityRevision==revision){[self orderOut:nil];self.alphaValue=1;}}];
}
- (void)cancelOperation:(id)sender { [self dismiss]; }
@end

@interface MBItemButton : NSButton
@property BOOL hovered;
@property NSTrackingArea *hoverArea;
@end
@implementation MBItemButton
- (void)updateTrackingAreas {
    [super updateTrackingAreas];
    if(self.hoverArea)[self removeTrackingArea:self.hoverArea];
    self.hoverArea=[[NSTrackingArea alloc] initWithRect:NSZeroRect options:NSTrackingMouseEnteredAndExited|NSTrackingActiveAlways|NSTrackingInVisibleRect owner:self userInfo:nil];
    [self addTrackingArea:self.hoverArea];
}
- (void)mouseEntered:(NSEvent *)event {self.hovered=YES;self.needsDisplay=YES;}
- (void)mouseExited:(NSEvent *)event {self.hovered=NO;self.needsDisplay=YES;}
- (void)drawRect:(NSRect)dirty {
    if(self.hovered){[[NSColor.labelColor colorWithAlphaComponent:0.12] setFill];[[NSBezierPath bezierPathWithRoundedRect:self.bounds xRadius:6 yRadius:6] fill];}
    if(self.image && self.imagePosition==NSImageOnly) {
        // A capture already has the status item's point size. NSButton's image cell shrinks it.
        NSSize size=self.image.size;
        [self.image drawInRect:NSMakeRect((self.bounds.size.width-size.width)/2,(self.bounds.size.height-size.height)/2,size.width,size.height) fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
    } else [super drawRect:dirty];
}
@end

@interface MBAppearanceView : NSView
@property NSDictionary *style;
@end
@implementation MBAppearanceView
- (void)drawRect:(NSRect)dirty {
    NSDictionary *s = self.style;
    unsigned rgb = 0;
    [[NSScanner scannerWithString:[s[@"color"] substringFromIndex:1]] scanHexInt:&rgb];
    NSColor *color = [NSColor colorWithSRGBRed:((rgb>>16)&255)/255.0 green:((rgb>>8)&255)/255.0 blue:(rgb&255)/255.0 alpha:0.72];
    NSRect r = NSInsetRect(self.bounds, 2, 1);
    NSArray *rects = [s[@"shape"] isEqual:@"split"] ? @[[NSValue valueWithRect:NSMakeRect(2,1,r.size.width*0.35,r.size.height)], [NSValue valueWithRect:NSMakeRect(r.size.width*0.52,1,r.size.width*0.48,r.size.height)]] : @[[NSValue valueWithRect:r]];
    for (NSValue *v in rects) {
        NSBezierPath *path = [NSBezierPath bezierPathWithRoundedRect:v.rectValue xRadius:[s[@"shape"] isEqual:@"none"]?0:9 yRadius:[s[@"shape"] isEqual:@"none"]?0:9];
        [NSGraphicsContext saveGraphicsState];
        if ([s[@"shadow"] boolValue]) { NSShadow *shadow=[NSShadow new]; shadow.shadowColor=[NSColor colorWithWhite:0 alpha:0.32]; shadow.shadowBlurRadius=5; shadow.shadowOffset=NSMakeSize(0,-2); [shadow set]; }
        if ([s[@"tint"] isEqual:@"solid"]) { [color setFill]; [path fill]; }
        else if ([s[@"tint"] isEqual:@"gradient"]) { [[[NSGradient alloc] initWithStartingColor:color endingColor:[color colorWithAlphaComponent:0.1]] drawInBezierPath:path angle:0]; }
        else if (![s[@"shape"] isEqual:@"none"] || [s[@"shadow"] boolValue]) { [[NSColor.windowBackgroundColor colorWithAlphaComponent:0.7] setFill]; [path fill]; }
        if ([s[@"border"] boolValue]) { [[NSColor.labelColor colorWithAlphaComponent:0.3] setStroke]; path.lineWidth=1; [path stroke]; }
        [NSGraphicsContext restoreGraphicsState];
    }
}
@end

@interface MBManager : NSObject <NSSearchFieldDelegate>
@property NSDictionary *settings;
@property NSDictionary *strings;
@property NSStatusItem *control;
@property NSStatusItem *divider;
@property NSStatusItem *always;
@property BOOL expanded;
@property BOOL showingAll;
@property(nonatomic) BOOL menusHidden;
@property NSMenu *savedMenu;
@property NSRunningApplication *previousApp;
@property MBPanel *panel;
@property NSSearchField *search;
@property NSMutableDictionary<NSNumber *,NSImage *> *images;
@property NSMutableDictionary<NSNumber *,NSString *> *imageURLs;
@property BOOL moving;
@property BOOL activating;
@property NSDictionary *temporaryItem;
@property NSSet *temporaryWindows;
@property NSTimer *revealTimer;
@property CFAbsoluteTime revealDeadline;
@property BOOL capturing;
@property NSArray<NSDictionary *> *panelItems;
@property NSArray<NSButton *> *panelButtons;
@property NSArray<NSPanel *> *overlays;
@property id globalMonitor;
@property id localMonitor;
@property NSTimer *rehideTimer;
@property NSTimer *hoverTimer;
@property NSString *lastError;
@property BOOL searchMode;
- (NSDictionary *)state;
- (NSString *)apply:(NSDictionary *)payload;
- (void)stop;
- (void)showPanel:(BOOL)search;
- (void)collapse;
- (NSArray *)items;
- (void)refreshAppearance;
- (NSString *)restoreTemporaryItem;
@end

@implementation MBManager
- (instancetype)init {
    if ((self=[super init])) {
        _images=[NSMutableDictionary new]; _imageURLs=[NSMutableDictionary new]; _strings=@{}; _lastError=@"";
        NSUserDefaults *defaults=NSUserDefaults.standardUserDefaults;
        // Seed first-run positions; AppKit owns subsequent user ordering.
        if(![defaults objectForKey:@"NSStatusItem Preferred Position localops.storage"])
            [defaults setInteger:1 forKey:@"NSStatusItem Preferred Position localops.storage"];
        [NSNotificationCenter.defaultCenter addObserver:self selector:@selector(screenChanged:) name:NSApplicationDidChangeScreenParametersNotification object:nil];
        [NSWorkspace.sharedWorkspace.notificationCenter addObserver:self selector:@selector(screenChanged:) name:NSWorkspaceActiveSpaceDidChangeNotification object:nil];
        [NSDistributedNotificationCenter.defaultCenter addObserver:self selector:@selector(screenChanged:) name:@"AppleInterfaceThemeChangedNotification" object:nil];
    }
    return self;
}
- (void)screenChanged:(NSNotification *)note { [self collapse]; [self refreshAppearance]; }
- (CGFloat)hiddenLength {
    CGFloat width=0; for (NSScreen *screen in NSScreen.screens) width+=screen.frame.size.width;
    return width*2;
}
- (void)updateLengths {
    CGFloat hidden=self.expanded?20:[self hiddenLength],always=self.showingAll?20:[self hiddenLength];
    if(self.divider.length!=hidden)self.divider.length=hidden;
    if(self.always.length!=always)self.always.length=always;
    self.control.button.image=MBControlImage();
    self.control.button.accessibilityLabel=MBText(self.strings,@"展开或收起隐藏图标");
    self.control.button.toolTip=MBText(self.strings,@"展开或收起隐藏图标");
    NSString *style=self.settings[@"divider"];
    NSImage *mark=[style isEqual:@"none"]?nil:[NSImage imageWithSystemSymbolName:[style isEqual:@"dot"]?@"circle.fill":@"line.diagonal" accessibilityDescription:MBText(self.strings,@"分区边界")];
    self.divider.button.image=mark; self.always.button.image=mark;
}
- (void)setMenusHidden:(BOOL)hidden {
    if (hidden==_menusHidden) return;
    _menusHidden=hidden;
    if (hidden) {
        self.previousApp=NSWorkspace.sharedWorkspace.frontmostApplication;
        self.savedMenu=NSApp.mainMenu;
        NSApp.mainMenu=[NSMenu new];
        [NSApp activateIgnoringOtherApps:YES];
    } else {
        NSApp.mainMenu=self.savedMenu; self.savedMenu=nil;
        if (NSWorkspace.sharedWorkspace.frontmostApplication.processIdentifier==getpid() && self.previousApp.processIdentifier!=getpid()) [self.previousApp activateWithOptions:0];
        self.previousApp=nil;
    }
}
- (void)armRehide {
    [self.rehideTimer invalidate];
    double seconds=[self.settings[@"rehideDelay"] doubleValue];
    if (seconds>0) self.rehideTimer=[NSTimer scheduledTimerWithTimeInterval:seconds target:self selector:@selector(rehide:) userInfo:nil repeats:NO];
}
- (void)rehide:(NSTimer *)timer {
    // Never retract an item while its app's menu is tracking or a mouse drag is active.
    BOOL tracking=[NSRunLoop.currentRunLoop.currentMode isEqual:NSEventTrackingRunLoopMode];
    for (NSDictionary *w in MBWindows()) if ([w[(__bridge NSString *)kCGWindowNumber] integerValue]!=self.panel.windowNumber && [w[(__bridge NSString *)kCGWindowLayer] integerValue]==NSPopUpMenuWindowLevel && [w[(__bridge NSString *)kCGWindowIsOnscreen] boolValue]) tracking=YES;
    if (tracking || NSEvent.pressedMouseButtons) { [self armRehide]; return; }
    [self collapse];
}
- (void)collapse {
    self.expanded=NO; self.showingAll=NO;
    [self.panel dismiss]; [self.rehideTimer invalidate]; [self.hoverTimer invalidate];
    [self setMenusHidden:NO]; [self updateLengths];
}
- (void)expand {
    self.expanded=YES; [self updateLengths];
    if ([self.settings[@"hideAppMenus"] boolValue]) [self setMenusHidden:YES];
    [self armRehide];
}
- (void)clicked:(id)sender {
    if (NSApp.currentEvent.type==NSEventTypeRightMouseUp) { [self contextMenu]; return; }
    if (self.expanded || self.panel.visible) [self collapse];
    else if ([self.settings[@"usePanel"] boolValue]) [self showPanel:NO];
    else [self expand];
}
- (void)contextMenu {
    NSMenu *menu=[NSMenu new];
    NSArray *rows=@[
        @[@"打开 RockshinOps",@"open"], @[@"菜单栏设置…",@"menuBarSettings"], @[@"应用设置…",@"settings"], @[],
        @[@"打开展开栏",@"panel"], @[@"展开或收起隐藏图标",@"toggle"], @[@"搜索菜单栏图标",@"search"],
        @[@"显示全部分区",@"all"], @[@"刷新图标",@"refresh"], @[], @[@"退出 RockshinOps",@"quit"]
    ];
    for (NSArray *row in rows) {
        if (!row.count) { [menu addItem:NSMenuItem.separatorItem]; continue; }
        NSMenuItem *item=[[NSMenuItem alloc] initWithTitle:MBText(self.strings,row[0]) action:@selector(contextAction:) keyEquivalent:@""];
        item.representedObject=row[1]; item.target=self; [menu addItem:item];
    }
    [menu popUpMenuPositioningItem:nil atLocation:NSEvent.mouseLocation inView:nil];
}
- (void)contextAction:(NSMenuItem *)sender {
    NSString *action=sender.representedObject;
    if ([action isEqual:@"panel"]) [self showPanel:NO];
    else if ([action isEqual:@"toggle"]) { if(self.expanded || self.panel.visible) [self collapse]; else [self expand]; }
    else if ([action isEqual:@"search"]) [self showPanel:YES];
    else if ([action isEqual:@"all"]) { self.showingAll=YES; [self expand]; }
    else if ([action isEqual:@"refresh"]) {
        self.lastError=@"";
        if (!CGPreflightScreenCaptureAccess()) {
            self.lastError=MBText(self.strings,@"展开栏需要屏幕录制权限以显示真实菜单栏图标。");
            NSAlert *alert=[NSAlert new]; alert.messageText=self.lastError; [alert runModal];
        } else { if(self.panel.visible)[self renderPanel]; [self captureImages:NO]; }
    }
    else rockshin_menubar_action((char *)action.UTF8String);
}
- (BOOL)inMenuBar:(NSPoint)p {
    for (NSScreen *s in NSScreen.screens) if (NSPointInRect(p,NSMakeRect(s.frame.origin.x,NSMaxY(s.frame)-NSStatusBar.systemStatusBar.thickness,s.frame.size.width,NSStatusBar.systemStatusBar.thickness))) return YES;
    return NO;
}
- (void)event:(NSEvent *)event {
    if(self.moving || self.activating)return;
    NSPoint p=NSEvent.mouseLocation;
    BOOL atBar=[self inMenuBar:p];
    if (event.type==NSEventTypeLeftMouseDragged && atBar && (event.modifierFlags & NSEventModifierFlagCommand) && [self.settings[@"showAllOnDrag"] boolValue]) {
        self.showingAll=YES; [self expand];
    }
    if (event.type==NSEventTypeLeftMouseUp && self.showingAll) { [self armRehide]; }
    if (event.type==NSEventTypeRightMouseUp && atBar && [self.settings[@"secondaryMenu"] boolValue]) {
        // Only empty status-bar space; never steal another app's context menu.
        BOOL occupied=NO;
        for(NSDictionary *w in MBWindows()) { CGRect r=MBRect(w); CGFloat top=NSMaxY(NSScreen.screens.firstObject.frame); if ([w[(__bridge NSString *)kCGWindowLayer] integerValue]==NSStatusWindowLevel && CGRectContainsPoint(r,CGPointMake(p.x,top-p.y))) occupied=YES; }
        if(!occupied) [self contextMenu];
    }
    if(event.type==NSEventTypeMouseMoved && [self.settings[@"showOnHover"] boolValue]) {
        BOOL overControl=self.control.button.window && NSPointInRect(p,self.control.button.window.frame);
        if(overControl && !self.expanded && !self.panel.visible && !self.hoverTimer) self.hoverTimer=[NSTimer scheduledTimerWithTimeInterval:MAX(0.01,[self.settings[@"hoverDelay"] doubleValue]) target:self selector:@selector(hover:) userInfo:nil repeats:NO];
        if(!overControl) { [self.hoverTimer invalidate]; self.hoverTimer=nil; }
    }
    if(event.type==NSEventTypeLeftMouseDown && self.panel.visible && !NSPointInRect(p,self.panel.frame) && !atBar) [self collapse];
}
- (void)hover:(NSTimer *)timer { self.hoverTimer=nil; if([self.settings[@"usePanel"] boolValue]) [self showPanel:NO];else [self expand]; }
- (NSString *)apply:(NSDictionary *)payload {
    NSDictionary *settings=payload[@"settings"];
    if(self.activating || (self.temporaryItem && (![settings[@"enabled"] boolValue] || (![settings[@"alwaysHidden"] boolValue] && [self.temporaryItem[@"section"] isEqual:@"alwaysHidden"]))))return MBText(self.strings,@"请先关闭当前图标菜单，再操作其他图标。");
    if ([settings[@"enabled"] boolValue] && MBIceRunning()) return MBText(payload[@"strings"],@"请先退出 Ice，避免两个菜单栏管理器同时运行。");
    self.settings=settings; self.strings=payload[@"strings"] ?: @{};
    if (![settings[@"enabled"] boolValue]) { [self stop]; return nil; }
    if(!self.control) {
        self.control=MBCreateStatusItem(@"RockshinOps.MenuBar.Toggle",26,@0);
        self.control.button.target=self; self.control.button.action=@selector(clicked:);
        self.control.button.focusRingType=NSFocusRingTypeNone;
        ((NSButtonCell *)self.control.button.cell).highlightsBy=NSNoCellMask;
        [self.control.button sendActionOn:NSEventMaskLeftMouseUp|NSEventMaskRightMouseUp];
        self.divider=MBCreateStatusItem(@"RockshinOps.MenuBar.Hidden",20,@2);
        self.divider.button.toolTip=MBText(self.strings,@"按住 Command 拖动图标到分区边界左侧以隐藏");
        NSEventMask mask=NSEventMaskMouseMoved|NSEventMaskLeftMouseDown|NSEventMaskLeftMouseUp|NSEventMaskLeftMouseDragged|NSEventMaskRightMouseUp;
        __weak MBManager *weak=self;
        self.globalMonitor=[NSEvent addGlobalMonitorForEventsMatchingMask:mask handler:^(NSEvent *e){[weak event:e];}];
        self.localMonitor=[NSEvent addLocalMonitorForEventsMatchingMask:mask handler:^NSEvent *(NSEvent *e){[weak event:e];return e;}];
    }
    if([settings[@"alwaysHidden"] boolValue] && !self.always)self.always=MBCreateStatusItem(@"RockshinOps.MenuBar.AlwaysHidden",20,nil);
    if(![settings[@"alwaysHidden"] boolValue] && self.always) { MBSaveStatusItemPosition(self.always);[NSStatusBar.systemStatusBar removeStatusItem:self.always];self.always=nil; }
    [self updateLengths]; [self refreshAppearance];
    return nil;
}
- (void)stop {
    // Snapshot every position before removing any item shifts its neighbors.
    MBSaveStatusItemPosition(self.control);MBSaveStatusItemPosition(self.divider);MBSaveStatusItemPosition(self.always);
    [self collapse];
    if(self.control)[NSStatusBar.systemStatusBar removeStatusItem:self.control];
    if(self.divider)[NSStatusBar.systemStatusBar removeStatusItem:self.divider];
    if(self.always)[NSStatusBar.systemStatusBar removeStatusItem:self.always];
    self.control=nil;self.divider=nil;self.always=nil;
    if(self.globalMonitor) [NSEvent removeMonitor:self.globalMonitor]; if(self.localMonitor) [NSEvent removeMonitor:self.localMonitor];
    self.globalMonitor=nil; self.localMonitor=nil;
    for(NSPanel *p in self.overlays) [p orderOut:nil];self.overlays=@[];
}
- (NSArray *)items {
    NSMutableArray *items=[NSMutableArray new];
    // Read boundaries and items from one WindowServer transaction. AppKit frames can lead it by a turn.
    NSArray *windows=MBWindows();
    CGFloat hiddenEdge=-CGFLOAT_MAX,alwaysEdge=-CGFLOAT_MAX;
    for(NSDictionary *w in windows) {
        NSInteger wid=[w[(__bridge NSString *)kCGWindowNumber] integerValue];
        if(wid==self.divider.button.window.windowNumber)hiddenEdge=CGRectGetMaxX(MBRect(w));
        if(self.always && wid==self.always.button.window.windowNumber)alwaysEdge=CGRectGetMaxX(MBRect(w));
    }
    NSSet *boundaries=[NSSet setWithArray:@[@(self.divider.button.window.windowNumber),@(self.always.button.window.windowNumber)]];
    CGFloat screenWidth=0;for(NSScreen *s in NSScreen.screens) screenWidth=MAX(screenWidth,s.frame.size.width);
    for(NSDictionary *w in windows) {
        CGRect r=MBRect(w); NSNumber *wid=w[(__bridge NSString *)kCGWindowNumber];
        if([w[(__bridge NSString *)kCGWindowLayer] integerValue]!=NSStatusWindowLevel || r.size.width<=0 || r.size.width>screenWidth/2 || r.size.height>64 || [boundaries containsObject:wid]) continue;
        CGFloat cocoaY=NSMaxY(NSScreen.screens.firstObject.frame)-CGRectGetMidY(r);
        BOOL atMenuBar=NO;
        for(NSScreen *display in NSScreen.screens)if(fabs(cocoaY-(NSMaxY(display.frame)-r.size.height/2))<4)atMenuBar=YES;
        if(!atMenuBar)continue;
        pid_t pid=[w[(__bridge NSString *)kCGWindowOwnerPID] intValue];
        NSRunningApplication *app=[NSRunningApplication runningApplicationWithProcessIdentifier:pid];
        NSString *name=w[(__bridge NSString *)kCGWindowName];
        if(wid.integerValue==self.control.button.window.windowNumber)name=MBText(self.strings,@"展开或收起隐藏图标");
        else if([name hasPrefix:@"RockshinOps.MenuBar."])continue;
        NSString *owner=w[(__bridge NSString *)kCGWindowOwnerName] ?: app.localizedName ?: @"";
        if(!name.length || [name hasPrefix:@"Item-"]) name=owner;
        NSString *section=@"visible";
        // Boundaries only classify the display containing our controls; other displays stay visible.
        NSScreen *screen=self.divider.button.window.screen;
        BOOL sameDisplay=screen && fabs(cocoaY-(NSMaxY(screen.frame)-r.size.height/2))<4 && CGRectGetMidX(r)<=NSMaxX(screen.frame);
        if(self.control && sameDisplay && CGRectGetMidX(r)<hiddenEdge) section=@"hidden";
        if(self.always && sameDisplay && CGRectGetMidX(r)<alwaysEdge) section=@"alwaysHidden";
        BOOL movable=![name isEqual:@"Clock"] && ![name isEqual:@"Control Center"] && ![owner isEqual:@"Spotlight"];
        [items addObject:@{@"id":wid,@"pid":@(pid),@"name":name,@"owner":owner,@"section":section,@"image":self.imageURLs[wid] ?: @"",@"x":@(r.origin.x),@"width":@(r.size.width),@"movable":@(movable)}];
    }
    // WindowServer can omit status windows while switching sections or Spaces.
    // Keep the last successful preview for this manager's lifetime.
    [items sortUsingComparator:^NSComparisonResult(NSDictionary *a,NSDictionary *b){return [a[@"x"] compare:b[@"x"]];}];
    return items;
}
- (void)captureImages:(BOOL)missingOnly {
    if(self.capturing || self.moving || self.activating || !CGPreflightScreenCaptureAccess()) return;
    self.capturing=YES;
    __weak MBManager *weak=self;
    [SCShareableContent getShareableContentExcludingDesktopWindows:YES onScreenWindowsOnly:NO completionHandler:^(SCShareableContent *content,NSError *error){
        dispatch_async(dispatch_get_main_queue(),^{
            MBManager *self=weak; if(!self) return;
            if(error) {self.lastError=error.localizedDescription;self.capturing=NO;return;}
            NSMutableSet *ids=[NSMutableSet new];for(NSDictionary *i in [self items]) [ids addObject:i[@"id"]];
            dispatch_group_t group=dispatch_group_create();
            for(SCWindow *w in content.windows) {
                if(![ids containsObject:@(w.windowID)]) continue;
                if(missingOnly && [self.imageURLs[@(w.windowID)] length]) continue;
                SCContentFilter *filter=[[SCContentFilter alloc] initWithDesktopIndependentWindow:w];
                SCStreamConfiguration *config=[SCStreamConfiguration new];config.width=MAX(1,w.frame.size.width*2);config.height=MAX(1,w.frame.size.height*2);config.showsCursor=NO;
                if(@available(macOS 14.0,*)){config.ignoreShadowsSingleWindow=YES;config.shouldBeOpaque=NO;}
                dispatch_group_enter(group);
                [SCScreenshotManager captureImageWithFilter:filter configuration:config completionHandler:^(CGImageRef image,NSError *err){
                    NSImage *result=image?[[NSImage alloc] initWithCGImage:image size:w.frame.size]:nil;
                    NSString *url=MBPNG(result);
                    dispatch_async(dispatch_get_main_queue(),^{if(url.length){self.images[@(w.windowID)]=result;self.imageURLs[@(w.windowID)]=url;}else if(err)self.lastError=err.localizedDescription;dispatch_group_leave(group);});
                }];
            }
            dispatch_group_notify(group,dispatch_get_main_queue(),^{
                self.capturing=NO;
                for(NSButton *button in self.panelButtons)if(self.images[@(button.tag)]){button.image=self.images[@(button.tag)];if(!self.searchMode)button.title=@"";}
            });
        });
    }];
}
- (void)showPanel:(BOOL)search {
    if(!self.control) {self.lastError=MBText(self.strings,@"请先启用菜单栏管理");return;}
    self.searchMode=search;
    if(!CGPreflightScreenCaptureAccess()) {self.lastError=MBText(self.strings,@"展开栏需要屏幕录制权限以显示真实菜单栏图标。");return;}
    if(!self.panel) {
        self.panel=[[MBPanel alloc] initWithContentRect:NSMakeRect(0,0,600,70) styleMask:NSWindowStyleMaskBorderless|NSWindowStyleMaskNonactivatingPanel backing:NSBackingStoreBuffered defer:NO];
        self.panel.level=NSPopUpMenuWindowLevel;self.panel.collectionBehavior=NSWindowCollectionBehaviorCanJoinAllSpaces|NSWindowCollectionBehaviorFullScreenAuxiliary;
        self.panel.backgroundColor=NSColor.clearColor;self.panel.opaque=NO;self.panel.hasShadow=YES;
        self.panel.hidesOnDeactivate=NO;
    }
    self.panelItems=[self items];
    self.panel.appearance=self.control.button.effectiveAppearance;
    self.panel.title=MBText(self.strings,search?@"搜索菜单栏图标":@"打开展开栏");
    [self renderPanel];[self.panel present];[self captureImages:NO];
    if(search) [self.panel makeFirstResponder:self.search];
    [self armRehide];
}
- (void)renderPanel {
    NSString *query=self.searchMode ? (self.search.stringValue ?: @"") : @"";
    NSScreen *screen=self.control.button.window.screen ?: NSScreen.mainScreen;
    NSMutableArray *items=[NSMutableArray new];CGFloat naturalWidth=16;
    for(NSDictionary *item in self.panelItems) {
        if(!self.searchMode && ![item[@"section"] isEqual:@"hidden"])continue;
        if(query.length && [item[@"name"] rangeOfString:query options:NSCaseInsensitiveSearch].location==NSNotFound && [item[@"owner"] rangeOfString:query options:NSCaseInsensitiveSearch].location==NSNotFound)continue;
        [items addObject:item];naturalWidth+=MAX(32,MIN(160,[item[@"width"] doubleValue]+8))+2;
    }
    CGFloat width=MIN(self.searchMode?360:MAX(160,naturalWidth),screen.visibleFrame.size.width-24);
    CGFloat documentHeight=self.searchMode?MAX(52,items.count*36+16):52;
    CGFloat height=self.searchMode?MIN(documentHeight+44,MIN(460,screen.visibleFrame.size.height-24)):52;
    CGFloat documentWidth=self.searchMode?width:MAX(width,naturalWidth);
    NSVisualEffectView *content=[self.panel.contentView isKindOfClass:NSVisualEffectView.class]?(NSVisualEffectView *)self.panel.contentView:[NSVisualEffectView new];
    for(NSView *view in content.subviews.copy)if(view!=self.search)[view removeFromSuperview];
    content.material=NSVisualEffectMaterialPopover;content.blendingMode=NSVisualEffectBlendingModeBehindWindow;content.state=NSVisualEffectStateActive;
    content.wantsLayer=YES;content.layer.cornerRadius=10;content.layer.masksToBounds=YES;
    if(!content.maskImage) {
        // Layer clipping alone does not clip the compositor's behind-window material.
        NSImage *mask=[NSImage imageWithSize:NSMakeSize(21,21) flipped:NO drawingHandler:^BOOL(NSRect rect){
            [NSColor.blackColor setFill];[[NSBezierPath bezierPathWithRoundedRect:rect xRadius:10 yRadius:10] fill];return YES;
        }];
        mask.capInsets=NSEdgeInsetsMake(10,10,10,10);content.maskImage=mask;
    }
    content.layer.borderWidth=0.5;content.layer.borderColor=[NSColor.separatorColor colorWithAlphaComponent:0.35].CGColor;
    content.frame=NSMakeRect(0,0,width,height);
    NSScrollView *scroll=[[NSScrollView alloc] initWithFrame:NSMakeRect(0,0,width,height-(self.searchMode?44:0))];
    scroll.drawsBackground=NO;scroll.contentView.drawsBackground=NO;scroll.borderType=NSNoBorder;
    scroll.hasVerticalScroller=self.searchMode;scroll.hasHorizontalScroller=!self.searchMode;
    scroll.scrollerStyle=NSScrollerStyleOverlay;scroll.autohidesScrollers=YES;
    NSView *document=[[NSView alloc] initWithFrame:NSMakeRect(0,0,documentWidth,documentHeight)];
    NSMutableArray *buttons=[NSMutableArray new];CGFloat x=8;
    for(NSDictionary *item in items) {
        MBItemButton *button=[MBItemButton new];button.target=self;button.action=@selector(itemClicked:);button.tag=[item[@"id"] integerValue];
        button.focusRingType=NSFocusRingTypeNone;
        button.bordered=NO;button.bezelStyle=NSBezelStyleRecessed;button.font=[NSFont systemFontOfSize:13];button.imageScaling=NSImageScaleProportionallyDown;
        button.toolTip=[NSString stringWithFormat:@"%@ · %@",item[@"owner"],item[@"name"]];button.accessibilityLabel=button.toolTip;
        button.image=self.images[item[@"id"]] ?: [NSImage imageWithSystemSymbolName:@"app.dashed" accessibilityDescription:item[@"name"]];
        button.title=self.searchMode?item[@"name"]:@"";
        button.imagePosition=self.searchMode?NSImageLeft:NSImageOnly;
        CGFloat bw=self.searchMode?width-16:MAX(32,MIN(160,[item[@"width"] doubleValue]+8));
        button.frame=NSMakeRect(self.searchMode?8:x,self.searchMode?documentHeight-8-(buttons.count+1)*36:6,bw,self.searchMode?32:40);
        if(self.searchMode)button.alignment=NSTextAlignmentLeft;
        x+=bw+2;[buttons addObject:button];[document addSubview:button];
    }
    self.panelButtons=buttons;
    if(!buttons.count){NSTextField *empty=[NSTextField labelWithString:MBText(self.strings,@"此分区没有图标")];empty.textColor=NSColor.secondaryLabelColor;empty.font=[NSFont systemFontOfSize:12];empty.frame=NSMakeRect(12,18,width-24,18);[document addSubview:empty];}
    scroll.documentView=document;[content addSubview:scroll];
    [document scrollPoint:NSMakePoint(0,self.searchMode?MAX(0,documentHeight-scroll.frame.size.height):0)];
    if(self.searchMode){
        if(!self.search){self.search=[NSSearchField new];self.search.placeholderString=MBText(self.strings,@"搜索菜单栏图标");self.search.delegate=self;}
        self.search.frame=NSMakeRect(10,height-36,width-20,26);
        if(self.search.superview!=content)[content addSubview:self.search];
    } else {[self.search removeFromSuperview];self.search=nil;}
    NSRect anchor=self.control.button.window.frame;
    CGFloat panelX=MAX(NSMinX(screen.visibleFrame)+12,MIN(NSMidX(anchor)-width/2,NSMaxX(screen.visibleFrame)-width-12));
    [self.panel setFrame:NSMakeRect(panelX,NSMinY(anchor)-height-5,width,height) display:NO];
    if(self.panel.contentView!=content)self.panel.contentView=content;
    [self.panel invalidateShadow];
}
- (void)controlTextDidChange:(NSNotification *)note { [self renderPanel];[self armRehide]; }
- (BOOL)temporaryInterfaceVisible {
    pid_t pid=[self.temporaryItem[@"pid"] intValue];
    BOOL active=NSWorkspace.sharedWorkspace.frontmostApplication.processIdentifier==pid;
    for(NSDictionary *w in MBWindows()) {
        NSInteger layer=[w[(__bridge NSString *)kCGWindowLayer] integerValue];
        if([w[(__bridge NSString *)kCGWindowOwnerPID] intValue]==pid && [w[(__bridge NSString *)kCGWindowIsOnscreen] boolValue]
            && layer!=NSStatusWindowLevel && ![self.temporaryWindows containsObject:w[(__bridge NSString *)kCGWindowNumber]]
            && (layer==NSPopUpMenuWindowLevel || active))return YES;
    }
    return NO;
}
- (NSString *)restoreTemporaryItem {
    NSDictionary *payload=MBMain(^id{
        if(!self.temporaryItem)return nil;
        [self.revealTimer invalidate];self.revealTimer=nil;
        BOOL exists=NO,beforeExists=NO;
        for(NSDictionary *item in [self items]) {
            if([item[@"id"] isEqual:self.temporaryItem[@"id"]] && [item[@"pid"] isEqual:self.temporaryItem[@"pid"]])exists=YES;
            if([item[@"id"] isEqual:self.temporaryItem[@"before"]] && [item[@"section"] isEqual:self.temporaryItem[@"section"]])beforeExists=YES;
        }
        if(!exists){self.temporaryItem=nil;self.temporaryWindows=nil;return nil;}
        NSMutableDictionary *result=self.temporaryItem.mutableCopy;
        if(!beforeExists)result[@"before"]=@0;
        return result;
    });
    if(!payload)return nil;
    NSDictionary *result=MBMove(payload);
    return MBMain(^id{
        // A failed return is surfaced and leaves the item visible for manual recovery.
        self.temporaryItem=nil;self.temporaryWindows=nil;
        return [result[@"error"] length]?result[@"error"]:nil;
    });
}
- (void)checkTemporaryItem:(NSTimer *)timer {
    if(self.activating || self.moving || CFAbsoluteTimeGetCurrent()<self.revealDeadline || NSEvent.pressedMouseButtons || [self temporaryInterfaceVisible])return;
    [timer invalidate];self.revealTimer=nil;self.activating=YES;
    dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED,0),^{
        NSString *error=[self restoreTemporaryItem];
        MBMain(^id{self.activating=NO;if(error.length){self.lastError=error;NSAlert *alert=[NSAlert new];alert.messageText=error;[alert runModal];}return nil;});
    });
}
- (NSString *)showItem:(NSNumber *)wid click:(BOOL)openMenu {
    __block NSDictionary *selected=nil,*returnTo=nil;
    __block NSString *error=nil;
    __block CGWindowID boundary=0;
    __block CGRect boundaryFrame=CGRectNull;
    BOOL shown=NO;
    MBMain(^id{
        if(!self.control)error=MBText(self.strings,@"请先启用菜单栏管理");
        else if(!AXIsProcessTrusted())error=MBText(self.strings,@"需要辅助功能权限以操作菜单栏图标。");
        else if(self.activating || self.moving || (self.temporaryItem && [self temporaryInterfaceVisible]))error=MBText(self.strings,@"请先关闭当前图标菜单，再操作其他图标。");
        if(!error)self.activating=YES;
        return nil;
    });
    if(error)return error;
    @try {
        error=[self restoreTemporaryItem];
        if(error)return error;
        MBMain(^id{
            NSArray *items=[self items];
            for(NSUInteger i=0;i<items.count;i++)if([items[i][@"id"] isEqual:wid]) {
                selected=items[i];
                if(![selected[@"section"] isEqual:@"visible"]) {
                    NSNumber *before=@0;
                    if(i+1<items.count && [items[i+1][@"section"] isEqual:selected[@"section"]])before=items[i+1][@"id"];
                    returnTo=@{@"id":wid,@"pid":selected[@"pid"],@"section":selected[@"section"],@"before":before,@"temporary":@YES};
                }
                break;
            }
            if(!selected){error=MBText(self.strings,@"图标已经退出，请刷新后重试。");return nil;}
            [self.panel orderOut:nil];
            if(self.expanded || self.showingAll){boundary=(CGWindowID)self.divider.button.window.windowNumber;boundaryFrame=MBWindowFrame(boundary);}
            [self collapse];
            return nil;
        });
        if(error)return error;
        if(boundary)MBWaitForFrameChange(boundary,boundaryFrame);
        if(returnTo) {
            // Only the selected item crosses the divider; all other hidden items stay hidden.
            NSDictionary *result=MBMove(@{@"id":wid,@"pid":selected[@"pid"],@"section":@"visible",@"before":@0,@"temporary":@YES});
            MBMain(^id{self.temporaryItem=returnTo;return nil;});
            if([result[@"error"] length])return result[@"error"];
        }
        __block CGRect frame=CGRectZero;
        __block BOOL visible=NO;
        MBMain(^id{
            frame=MBWindowFrame(wid.unsignedIntValue);
            CGFloat top=NSMaxY(NSScreen.screens.firstObject.frame);
            NSPoint center=NSMakePoint(CGRectGetMidX(frame),top-CGRectGetMidY(frame));
            for(NSScreen *screen in NSScreen.screens)if(NSPointInRect(center,screen.frame))visible=YES;
            self.temporaryWindows=MBVisibleWindowIDs([selected[@"pid"] intValue]);
            return nil;
        });
        if(CGRectIsNull(frame) || CGRectIsEmpty(frame) || !visible)return MBText(self.strings,@"无法打开此图标，请展开分区后从系统菜单栏点击。");
        if(!openMenu){shown=YES;return nil;}
        CGEventSourceRef source=CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
        CGPoint point=CGPointMake(CGRectGetMidX(frame),CGRectGetMidY(frame));
        CGEventRef cursor=CGEventCreate(NULL);CGPoint previous=CGEventGetLocation(cursor);CFRelease(cursor);
        for(int i=0;i<2;i++){
            CGEventRef event=CGEventCreateMouseEvent(source,i==0?kCGEventLeftMouseDown:kCGEventLeftMouseUp,point,kCGMouseButtonLeft);
            CGEventSetIntegerValueField(event,kCGMouseEventClickState,1);CGEventPost(kCGHIDEventTap,event);CFRelease(event);
            [NSThread sleepForTimeInterval:0.08];
        }
        CGWarpMouseCursorPosition(previous);CFRelease(source);
        shown=YES;
        return nil;
    } @finally {
        if(!shown) {
            NSString *restoreError=[self restoreTemporaryItem];
            if(restoreError.length)MBMain(^id{self.lastError=restoreError;return nil;});
        }
        MBMain(^id{
            self.activating=NO;
            if(self.temporaryItem) {
                self.revealDeadline=CFAbsoluteTimeGetCurrent()+[self.settings[@"temporaryRevealSeconds"] doubleValue];
                self.revealTimer=[NSTimer timerWithTimeInterval:[self.settings[@"revealCheckInterval"] doubleValue] target:self selector:@selector(checkTemporaryItem:) userInfo:nil repeats:YES];
                [NSRunLoop.mainRunLoop addTimer:self.revealTimer forMode:NSRunLoopCommonModes];
            }
            return nil;
        });
    }
}
- (NSString *)activateItem:(NSNumber *)wid { return [self showItem:wid click:YES]; }
- (void)itemClicked:(NSButton *)sender {
    NSNumber *wid=@(sender.tag);
    dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED,0),^{
        NSString *error=[self showItem:wid click:NO];
        dispatch_async(dispatch_get_main_queue(),^{self.lastError=error ?: @"";if(error.length){NSAlert *alert=[NSAlert new];alert.messageText=error;[alert runModal];}});
    });
}
- (void)refreshAppearance {
    for(NSPanel *p in self.overlays)[p orderOut:nil];self.overlays=@[];
    if(!self.control)return;
    BOOL dark=[[NSApp.effectiveAppearance bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua,NSAppearanceNameDarkAqua]] isEqualToString:NSAppearanceNameDarkAqua];
    NSDictionary *style=self.settings[[self.settings[@"dynamic"] boolValue] && dark?@"dark":@"light"];
    if([style[@"tint"] isEqual:@"none"] && [style[@"shape"] isEqual:@"none"] && ![style[@"border"] boolValue] && ![style[@"shadow"] boolValue])return;
    NSMutableArray *panels=[NSMutableArray new];
    for(NSScreen *s in NSScreen.screens){
        if(NSMaxY(s.visibleFrame)>=NSMaxY(s.frame))continue;
        CGFloat height=MAX(NSStatusBar.systemStatusBar.thickness,s.safeAreaInsets.top);
        NSPanel *panel=[[NSPanel alloc] initWithContentRect:NSMakeRect(s.frame.origin.x,NSMaxY(s.frame)-height,s.frame.size.width,height) styleMask:NSWindowStyleMaskBorderless|NSWindowStyleMaskNonactivatingPanel backing:NSBackingStoreBuffered defer:NO];
        panel.level=NSStatusWindowLevel-1;panel.collectionBehavior=NSWindowCollectionBehaviorCanJoinAllSpaces;panel.ignoresMouseEvents=YES;panel.backgroundColor=NSColor.clearColor;panel.opaque=NO;panel.hasShadow=NO;
        MBAppearanceView *view=[[MBAppearanceView alloc] initWithFrame:NSMakeRect(0,0,s.frame.size.width,height)];view.style=style;panel.contentView=view;[panel orderFrontRegardless];[panels addObject:panel];
    }
    self.overlays=panels;
}
- (NSDictionary *)state {
    NSArray *items=[self items];
    for(NSDictionary *item in items)if(![item[@"image"] length]){[self captureImages:YES];break;}
    return @{@"capturing":@(self.capturing),@"supported":@YES,@"accessibility":(AXIsProcessTrusted()?@YES:@NO),@"screenRecording":(CGPreflightScreenCaptureAccess()?@YES:@NO),@"expanded":@(self.expanded),@"panelVisible":@(self.panel.visible),@"iceRunning":@(MBIceRunning()),@"items":items,@"error":self.lastError ?: @""};
}
@end

static MBManager *manager;
static NSDictionary *MBRequest(NSDictionary *request) {
    if(!manager)manager=[MBManager new];
    NSString *action=request[@"action"];id payload=request[@"payload"];
    if(![action isEqual:@"state"])manager.lastError=@"";
    if([action isEqual:@"apply"])manager.lastError=[manager apply:payload] ?: @"";
    else if([action isEqual:@"stop"])[manager stop];
    else if([action isEqual:@"permission"]){
        if([payload isEqual:@"accessibility"]){AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)@{(__bridge NSString *)kAXTrustedCheckOptionPrompt:@YES});}
        else if([payload isEqual:@"screenRecording"]){CGRequestScreenCaptureAccess();}
    }
    else if([action isEqual:@"refresh"])[manager captureImages:NO];
    else if(![action isEqual:@"state"] && !manager.control)manager.lastError=MBText(manager.strings,@"请先启用菜单栏管理");
    else if([action isEqual:@"toggle"])[manager clicked:nil];
    else if([action isEqual:@"expand"])[manager expand];
    else if([action isEqual:@"collapse"])[manager collapse];
    else if([action isEqual:@"all"]){manager.showingAll=YES;[manager expand];}
    else if([action isEqual:@"panel"])[manager showPanel:NO];
    else if([action isEqual:@"search"])[manager showPanel:YES];
    else if([action isEqual:@"menus"])[manager setMenusHidden:!manager.menusHidden];
    return [manager state];
}
// Run gesture timing off the AppKit thread. Re-resolve the window immediately before sending events.
static CGRect MBWindowFrame(CGWindowID wid) {
    // IncludingWindow omits offscreen status windows; explicit IDs retain hidden sections.
    const void *value=(const void *)(uintptr_t)wid;
    CFArrayRef ids=CFArrayCreate(NULL,&value,1,NULL);
    NSArray *windows=CFBridgingRelease(CGWindowListCreateDescriptionFromArray(ids));
    CFRelease(ids);
    for(NSDictionary *window in windows)if([window[(__bridge NSString *)kCGWindowNumber] unsignedIntValue]==wid)return MBRect(window);
    return CGRectNull;
}
static BOOL MBWaitForFrameChange(CGWindowID wid, CGRect initial) {
    CFAbsoluteTime deadline=CFAbsoluteTimeGetCurrent()+0.25;
    do {
        CGRect frame=MBWindowFrame(wid);
        if(!CGRectIsNull(frame) && !CGRectEqualToRect(frame,initial))return YES;
        [NSThread sleepForTimeInterval:0.004];
    } while(CFAbsoluteTimeGetCurrent()<deadline);
    return NO;
}
static CGEventRef MBMoveEvent(CGEventSourceRef source, CGEventType type, CGPoint point, CGWindowID window, pid_t pid) {
    CGEventRef event=CGEventCreateMouseEvent(source,type,point,kCGMouseButtonLeft);
    if(!event)return NULL;
    CGEventSetFlags(event,type==kCGEventLeftMouseDown?kCGEventFlagMaskCommand:0);
    CGEventSetIntegerValueField(event,kCGEventTargetUnixProcessID,pid);
    CGEventSetIntegerValueField(event,kCGMouseEventWindowUnderMousePointer,window);
    CGEventSetIntegerValueField(event,kCGMouseEventWindowUnderMousePointerThatCanHandleThisEvent,window);
    CGEventSetIntegerValueField(event,(CGEventField)0x33,window);
    return event;
}
static NSDictionary *MBMove(NSDictionary *payload) {
    __block NSString *error=nil;
    __block CGPoint target;
    __block CGRect original=CGRectNull;
    __block CGWindowID destination=0;
    CGWindowID wid=[payload[@"id"] unsignedIntValue];pid_t pid=[payload[@"pid"] intValue];
    MBMain(^id{
        if(!manager.control)error=MBText(manager.strings,@"请先启用菜单栏管理");
        else if(!AXIsProcessTrusted())error=MBText(manager.strings,@"需要辅助功能权限以操作菜单栏图标。");
        else if(MBIceRunning())error=MBText(manager.strings,@"请先退出 Ice，避免两个菜单栏管理器同时运行。");
        else if(![payload[@"temporary"] boolValue] && (manager.activating || manager.temporaryItem))error=MBText(manager.strings,@"请先关闭当前图标菜单，再操作其他图标。");
        else if(manager.moving || NSEvent.pressedMouseButtons)error=MBText(manager.strings,@"移动菜单栏图标前请松开鼠标。");
        else if([payload[@"section"] isEqual:@"alwaysHidden"] && !manager.always)error=MBText(manager.strings,@"请先启用始终隐藏分区。");
        if(error)return nil;
        NSArray *windows=MBWindows(),*items=[manager items];
        NSDictionary *selected=nil,*before=nil;
        for(NSDictionary *w in windows) {
            if([w[(__bridge NSString *)kCGWindowNumber] unsignedIntValue]==wid && [w[(__bridge NSString *)kCGWindowOwnerPID] intValue]==pid && [w[(__bridge NSString *)kCGWindowLayer] integerValue]==NSStatusWindowLevel)selected=w;
            if([w[(__bridge NSString *)kCGWindowNumber] isEqual:payload[@"before"]])before=w;
        }
        if(!selected){error=MBText(manager.strings,@"图标已经退出，请刷新后重试。");return nil;}
        original=MBRect(selected);
        if(before) {CGRect r=MBRect(before);target=CGPointMake(CGRectGetMinX(r),CGRectGetMidY(r));destination=[payload[@"before"] unsignedIntValue];}
        else {
            destination=(CGWindowID)([payload[@"section"] isEqual:@"alwaysHidden"]?manager.always.button.window.windowNumber:manager.divider.button.window.windowNumber);
            CGRect boundary=MBWindowFrame(destination);
            if(CGRectIsNull(boundary)){error=MBText(manager.strings,@"图标已经退出，请刷新后重试。");return nil;}
            target=CGPointMake([payload[@"section"] isEqual:@"visible"]?CGRectGetMaxX(boundary):CGRectGetMinX(boundary),CGRectGetMidY(boundary));
            for(NSDictionary *item in items)if([item[@"id"] unsignedIntValue]==wid && [item[@"section"] isEqual:payload[@"section"]]) {
                NSDictionary *last=nil;
                for(NSDictionary *other in items)if([other[@"section"] isEqual:payload[@"section"]] && [other[@"id"] unsignedIntValue]!=wid)last=other;
                if(last){destination=[last[@"id"] unsignedIntValue];CGRect r=MBWindowFrame(destination);target=CGPointMake(CGRectGetMaxX(r),CGRectGetMidY(r));}
            }
        }
        manager.moving=YES;[manager.rehideTimer invalidate];return nil;
    });
    if(!error) {
        CGEventSourceRef source=CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
        CGEventRef cursor=CGEventCreate(NULL);CGPoint previous=CGEventGetLocation(cursor);CFRelease(cursor);
        CGEventSourceSetLocalEventsSuppressionInterval(source,0);
        CGEventSourceSetLocalEventsFilterDuringSuppressionState(source,kCGEventFilterMaskPermitAllEvents,kCGEventSuppressionStateRemoteMouseDrag);
        CGEventSourceSetLocalEventsFilterDuringSuppressionState(source,kCGEventFilterMaskPermitAllEvents,kCGEventSuppressionStateSuppressionInterval);
        CGEventRef down=MBMoveEvent(source,kCGEventLeftMouseDown,CGPointMake(20000,20000),wid,pid);
        CGEventRef up=MBMoveEvent(source,kCGEventLeftMouseUp,target,destination,pid);
        BOOL received=NO,settled=NO;
        if(down && up) {
            // Do not change the global cursor hide count while routing events to another app.
            @try {
                received=MBDeliverMenuBarEvent(down,pid);
                if(received)MBWaitForFrameChange(wid,original);
                CGRect dragging=MBWindowFrame(wid);
                BOOL released=MBDeliverMenuBarEvent(up,pid);
                received=received && released;
                if(received)settled=MBWaitForFrameChange(wid,dragging);
                if(!received || !settled) {
                    // Always release the drag at its original location after a failed acknowledgement.
                    CGEventRef release=MBMoveEvent(source,kCGEventLeftMouseUp,CGPointMake(CGRectGetMidX(original),CGRectGetMidY(original)),wid,pid);
                    if(release){CGEventPostToPid(pid,release);CGEventPost(kCGSessionEventTap,release);CFRelease(release);}
                }
            } @finally {
                CGWarpMouseCursorPosition(previous);
            }
        }
        if(down)CFRelease(down);if(up)CFRelease(up);if(source)CFRelease(source);
        MBMain(^id{
            if(!received)error=MBText(manager.strings,@"系统未确认移动事件，请检查辅助功能权限后重试。");
            else {
                NSDictionary *moved=nil;for(NSDictionary *item in [manager items])if([item[@"id"] unsignedIntValue]==wid)moved=item;
                if(!moved)error=MBText(manager.strings,@"图标已经退出，请刷新后重试。");
                else if(![moved[@"section"] isEqual:payload[@"section"]])error=MBText(manager.strings,@"macOS 未将图标移入目标分区，请在菜单栏中按住 Command 拖动。");
            }
            manager.moving=NO;if(manager.expanded || manager.panel.visible)[manager armRehide];return nil;
        });
    }
    return MBMain(^id{manager.lastError=error ?: @"";return [manager state];});
}
char *rockshin_menubar_request(const char *request) {
    @autoreleasepool {
        NSData *data=[[NSString stringWithUTF8String:request] dataUsingEncoding:NSUTF8StringEncoding];
        NSDictionary *input=[NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
        NSDictionary *result;
        if([input[@"action"] isEqual:@"stop"]) {
            NSString *error=[manager restoreTemporaryItem];
            result=MBMain(^id{NSDictionary *state=MBRequest(input);if(error.length){manager.lastError=error;return [manager state];}return state;});
        }
        else if([input[@"action"] isEqual:@"move"]) result=MBMove(input[@"payload"]);
        else if([input[@"action"] isEqual:@"activate"]) {
            NSString *error=[manager activateItem:input[@"payload"]];
            result=MBMain(^id{manager.lastError=error ?: @"";return [manager state];});
        }
        else result=MBMain(^id{return MBRequest(input);});
        NSData *output=[NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
        return strdup([[NSString alloc] initWithData:output encoding:NSUTF8StringEncoding].UTF8String);
    }
}
