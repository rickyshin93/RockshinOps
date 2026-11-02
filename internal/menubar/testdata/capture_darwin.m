#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#import <dlfcn.h>
#import <stdatomic.h>

static atomic_int batches, captures;
static int notifications;
static CGImageRef fixtureImage;
static BOOL fixturePermission=YES, fixtureAPI=YES;
static BOOL lateSlow;
static CGImageRef fixtureCapture(CGRect bounds,CFArrayRef windows,CGWindowImageOption options) {
    atomic_fetch_add(&captures,1);
    CGWindowID wid=(CGWindowID)(uintptr_t)CFArrayGetValueAtIndex(windows,0);
    double delay=wid==2?(lateSlow?0.8:0.30):0.02;
    [NSThread sleepForTimeInterval:delay];
    return wid==3?NULL:CGImageRetain(fixtureImage);
}
static void *fixtureSymbol(void *handle,const char *name) {
    atomic_fetch_add(&batches,1);
    return fixtureAPI?(void *)fixtureCapture:NULL;
}
#define dlsym fixtureSymbol
#define CGPreflightScreenCaptureAccess() fixturePermission
#ifndef MENU_NATIVE_SOURCE
#define MENU_NATIVE_SOURCE "../native_darwin.m"
#endif
#import MENU_NATIVE_SOURCE
void rockshin_menubar_action(char *action) {}
void rockshin_menubar_image(unsigned int wid, int pid, const char *image) { notifications++; }

@interface CaptureSubject : MBManager
@property CGFloat position;
@end
@implementation CaptureSubject
- (NSArray *)items {
    NSMutableArray *items=[NSMutableArray new];
    for(int wid=1;wid<=4;wid++)[items addObject:@{@"id":@(wid),@"pid":@42,@"name":[@(wid) stringValue],@"x":@(self.position+(wid==4?-9000:wid*24)),@"width":@24,@"section":wid==4?@"alwaysHidden":@"visible",@"image":self.imageURLs[@(wid)] ?: @""}];
    return items;
}
@end
static void spin(NSTimeInterval seconds) {
    NSDate *end=[NSDate dateWithTimeIntervalSinceNow:seconds];
    while(end.timeIntervalSinceNow>0)[NSRunLoop.currentRunLoop runMode:NSDefaultRunLoopMode beforeDate:[NSDate dateWithTimeIntervalSinceNow:0.001]];
}
static void waitForCapture(MBManager *subject) {
    CFAbsoluteTime start=CFAbsoluteTimeGetCurrent();
    while(subject.capturing && CFAbsoluteTimeGetCurrent()-start<2)spin(.001);
}
static int failures;
#define CHECK(condition,message) if(!(condition)){fprintf(stderr,"FAIL: %s\n",message);failures++;}
int main(void) { @autoreleasepool {
    [NSApplication sharedApplication];
    NSBitmapImageRep *rep=[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:48 pixelsHigh:48 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
    fixtureImage=rep.CGImage;
    CaptureSubject *subject=[CaptureSubject new];
    subject.settings=@{@"captureTimeoutSeconds":@0.5};
    NSButton *first=[NSButton new],*slow=[NSButton new];first.tag=1;slow.tag=2;subject.panelButtons=@[first,slow];
    CFAbsoluteTime start=CFAbsoluteTimeGetCurrent();
    [subject captureImages:YES];
    while(!first.image && CFAbsoluteTimeGetCurrent()-start<1)spin(0.001);
    double firstMS=(CFAbsoluteTimeGetCurrent()-start)*1000;
    printf("first_icon_ms=%.1f slow_still_pending=%d\n",firstMS,subject.capturing);
    CHECK(subject.capturing,"fast icon must display while slow capture is pending");
    waitForCapture(subject);
    CHECK(first.image && slow.image,"both successful captures must appear");
    CHECK([subject.imageURLs[@4] length]>0,"cold offscreen always-hidden icon must be captured without expanding");
    CHECK([subject.lastError isEqual:@"无法捕获菜单栏图标：3"],"capture errors must remain visible");
    int before=batches;
    for(int i=0;i<3;i++){[subject state];spin(.35);}
    printf("unchanged_missing_batches=%d\n",batches-before);
    CHECK(batches==before,"failed and absent windows must not be retried on unchanged polls");
    subject.position+=10;[subject state];waitForCapture(subject);
    CHECK(batches==before+1,"changed window geometry must retry missing captures");
    before=captures;[subject captureImages:NO];waitForCapture(subject);
    CHECK(captures==before+4,"explicit refresh must recapture cached images and retry failures");
    CHECK(notifications>=4,"successful images must be pushed before the next web poll");
    lateSlow=YES;[subject captureImages:NO];spin(.6);
    CHECK(!subject.capturing,"a stalled screenshot must not hold the capture lock");
    CHECK([subject.lastError isEqual:@"部分菜单栏图标捕获超时，可点击刷新重试。"],"timeout must be reported explicitly");
    int delivered=notifications;spin(.25);
    CHECK(notifications==delivered,"a late callback from an expired batch must not overwrite newer images");
    lateSlow=NO;before=captures;[subject captureImages:NO];
    waitForCapture(subject);
    fprintf(stderr,"retry_captures=%d still_capturing=%d\n",captures-before,subject.capturing);
    CHECK(captures==before+4 && !subject.capturing,"refresh must work after a stalled screenshot");
    before=batches;fixturePermission=NO;[subject captureImages:NO];spin(.05);
    CHECK(batches==before,"denied capture permission must not start discovery");
    fixturePermission=YES;[subject state];waitForCapture(subject);
    CHECK(batches==before+1,"restored capture permission must retry missing images");
    before=captures;[subject captureImages:NO];[subject captureImages:NO];[subject captureImages:NO];waitForCapture(subject);
    CHECK(captures==before+8 && !subject.capturing,"refresh clicks during capture must coalesce into one follow-up batch");
    fixtureAPI=NO;[subject captureImages:NO];
    CHECK(!subject.capturing && [subject.lastError isEqual:@"当前系统不支持菜单栏窗口捕获。"],"missing runtime API must fail explicitly");
    return failures?1:0;
} }
