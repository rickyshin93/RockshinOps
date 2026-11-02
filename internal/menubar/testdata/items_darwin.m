#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>

static NSArray *fixtureWindows;
static CFArrayRef fixtureWindowList(CGWindowListOption option, CGWindowID wid) {
    return (__bridge_retained CFArrayRef)fixtureWindows;
}
#define CGWindowListCopyWindowInfo fixtureWindowList
#import "../native_darwin.m"
void rockshin_menubar_action(char *action) {}
void rockshin_menubar_image(unsigned int wid, int pid, const char *image) {}

static NSDictionary *window(NSInteger wid, NSString *title, CGFloat x, BOOL onScreen) {
    NSMutableDictionary *value=[@{
        (__bridge NSString *)kCGWindowNumber:@(wid),
        (__bridge NSString *)kCGWindowOwnerPID:@(getpid()),
        (__bridge NSString *)kCGWindowOwnerName:@"Fixture App",
        (__bridge NSString *)kCGWindowLayer:@(NSStatusWindowLevel),
        (__bridge NSString *)kCGWindowIsOnscreen:@(onScreen),
        (__bridge NSString *)kCGWindowBounds:CFBridgingRelease(CGRectCreateDictionaryRepresentation(CGRectMake(x,0,24,24)))
    } mutableCopy];
    if(title)value[(__bridge NSString *)kCGWindowName]=title;
    return value;
}

int main(void) { @autoreleasepool {
    [NSApplication sharedApplication];
    fixtureWindows=@[
        window(1,@"Item-0",100,YES),
        window(2,@"",200,YES), // secondary-display replica
        window(3,@"",300,NO), // offscreen replica
        window(4,@"Item-0",-500,NO), // real hidden item, same owner/name
        window(5,nil,400,YES) // title unavailable without capture permission
    ];
    NSArray *items=[[MBManager new] items];
    NSArray *ids=[items valueForKey:@"id"];
    if(![ids isEqual:@[@4,@1,@5]]) {
        NSLog(@"FAIL: expected real/hidden/permission-redacted items only, got %@",ids);
        return 1;
    }
    puts("PASS: replicas excluded; real hidden and permission-redacted items retained");
    return 0;
} }
