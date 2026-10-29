#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <objc/message.h>
#import <objc/runtime.h>
#include <stdint.h>

static char resizeDriverKey;
static char responsiveResizeKey;
static char resizeTargetKey;
static IMP originalSetFrame;
static IMP originalPerformZoom;
static IMP originalZoom;

@interface RockshinOpsResizeTick : NSObject
@property(copy) void (^advance)(CFTimeInterval);
- (void)step:(CADisplayLink *)link API_AVAILABLE(macos(14.0));
@end

@implementation RockshinOpsResizeTick
- (void)step:(CADisplayLink *)link {
    void (^advance)(CFTimeInterval) = [self.advance copy];
    advance(link.targetTimestamp);
    [advance release];
}
- (void)dealloc {
    [_advance release];
    [super dealloc];
}
@end

static void setOriginalFrame(NSWindow *window, NSRect frame, BOOL display, BOOL animate) {
    ((void (*)(id, SEL, NSRect, BOOL, BOOL))originalSetFrame)(
        window, @selector(setFrame:display:animate:), frame, display, animate);
}

static void setAnimatedFrame(NSWindow *window, SEL selector, NSRect frame, BOOL display, BOOL animate) {
    if (!objc_getAssociatedObject(window, &responsiveResizeKey)) {
        setOriginalFrame(window, frame, display, animate);
        return;
    }
    id previous = objc_getAssociatedObject(window, &resizeDriverKey);
    [previous invalidate];
    objc_setAssociatedObject(window, &resizeDriverKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    objc_setAssociatedObject(window, &resizeTargetKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    if (!animate || NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceMotion) {
        setOriginalFrame(window, frame, display, NO);
        return;
    }

    NSRect start = window.frame;
    NSTimeInterval duration = [window animationResizeTime:frame];
    if (duration <= 0) {
        setOriginalFrame(window, frame, display, NO);
        return;
    }
    CFTimeInterval began = CACurrentMediaTime();
    void (^advance)(CFTimeInterval) = ^(CFTimeInterval timestamp) {
        double progress = MIN(1.0, (timestamp - began) / duration);
        double eased = progress * progress * (3.0 - 2.0 * progress);
        NSRect current = NSMakeRect(start.origin.x + (frame.origin.x - start.origin.x) * eased,
                                    start.origin.y + (frame.origin.y - start.origin.y) * eased,
                                    start.size.width + (frame.size.width - start.size.width) * eased,
                                    start.size.height + (frame.size.height - start.size.height) * eased);
        // Commit each native geometry/layout step before advancing to the next display frame.
        // Do not let implicit layer animations interpolate an already animated frame.
        [CATransaction begin];
        [CATransaction setDisableActions:YES];
        setOriginalFrame(window, current, display, NO);
        [window.contentView layoutSubtreeIfNeeded];
        [CATransaction commit];
        [CATransaction flush];
        if (progress >= 1.0) {
            [objc_getAssociatedObject(window, &resizeDriverKey) invalidate];
            objc_setAssociatedObject(window, &resizeDriverKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
            objc_setAssociatedObject(window, &resizeTargetKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
        }
    };
    id driver;
    if (@available(macOS 14.0, *)) {
        RockshinOpsResizeTick *tick = [RockshinOpsResizeTick new];
        tick.advance = advance;
        driver = [window displayLinkWithTarget:tick selector:@selector(step:)];
        [tick release];
    } else {
        // Keep the existing timer path on systems without AppKit display links.
        driver = [NSTimer timerWithTimeInterval:1.0 / 60.0 repeats:YES block:^(NSTimer *timer) {
            advance(CACurrentMediaTime());
        }];
    }
    objc_setAssociatedObject(window, &resizeTargetKey, [NSValue valueWithRect:frame], OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    objc_setAssociatedObject(window, &resizeDriverKey, driver, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    if (@available(macOS 14.0, *)) {
        [driver addToRunLoop:NSRunLoop.mainRunLoop forMode:NSRunLoopCommonModes];
    } else {
        [NSRunLoop.mainRunLoop addTimer:driver forMode:NSRunLoopCommonModes];
    }
}

static void finishResize(NSWindow *window) {
    NSValue *target = objc_getAssociatedObject(window, &resizeTargetKey);
    if (!target) return;
    NSRect frame = target.rectValue;
    [objc_getAssociatedObject(window, &resizeDriverKey) invalidate];
    objc_setAssociatedObject(window, &resizeDriverKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    objc_setAssociatedObject(window, &resizeTargetKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    setOriginalFrame(window, frame, YES, NO);
}

static void performZoom(NSWindow *window, SEL selector, id sender) {
    finishResize(window);
    ((void (*)(id, SEL, id))originalPerformZoom)(window, selector, sender);
}

static void zoom(NSWindow *window, SEL selector, id sender) {
    finishResize(window);
    ((void (*)(id, SEL, id))originalZoom)(window, selector, sender);
}

static IMP replaceWindowMethod(Class cls, SEL selector, IMP replacement) {
    Method method = class_getInstanceMethod(cls, selector);
    IMP original = method_getImplementation(method);
    if (!class_addMethod(cls, selector, replacement, method_getTypeEncoding(method))) {
        method_setImplementation(method, replacement);
    }
    return original;
}

static void enableResponsiveResize(NSWindow *window) {
    if (!originalSetFrame) {
        Class cls = object_getClass(window);
        originalSetFrame = replaceWindowMethod(cls, @selector(setFrame:display:animate:), (IMP)setAnimatedFrame);
        originalPerformZoom = replaceWindowMethod(cls, @selector(performZoom:), (IMP)performZoom);
        originalZoom = replaceWindowMethod(cls, @selector(zoom:), (IMP)zoom);
    }
    objc_setAssociatedObject(window, &responsiveResizeKey, @YES, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
}

@interface RockshinOpsDragStrip : NSView
@end

@implementation RockshinOpsDragStrip
- (void)mouseDown:(NSEvent *)event {
    NSWindow *window = self.window;
    if (event.clickCount == 2) {
        // Share AppKit's saved user frame with MyGo's title-bar regions and
        // the system zoom button, including after window-state restoration.
        [window performZoom:nil];
    } else {
        [window performWindowDragWithEvent:event];
    }
}
- (BOOL)acceptsFirstMouse:(NSEvent *)event { return YES; }
@end

void rockshinops_install_top_drag(uintptr_t handle) {
    NSWindow *window = (NSWindow *)handle;
    // AppKit's default zoom animation blocks WebKit's resize updates until it ends.
    // Advance the frame on the run loop so WebKit can lay out each intermediate size.
    enableResponsiveResize(window);
    NSView *content = window.contentView;
    // Native traffic lights start at y=20; the blank strip sits above them.
    NSRect frame = NSMakeRect(0, content.bounds.size.height - 20,
                             content.bounds.size.width, 20);
    RockshinOpsDragStrip *strip = [[RockshinOpsDragStrip alloc] initWithFrame:frame];
    strip.autoresizingMask = NSViewWidthSizable | NSViewMinYMargin;
    [content addSubview:strip positioned:NSWindowAbove relativeTo:nil];
    [strip release];
}
