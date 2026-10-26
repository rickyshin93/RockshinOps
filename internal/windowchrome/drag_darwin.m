#import <Cocoa/Cocoa.h>
#include <stdint.h>

@interface LocalOpsDragStrip : NSView
@end

@implementation LocalOpsDragStrip
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

void localops_install_top_drag(uintptr_t handle) {
    NSWindow *window = (NSWindow *)handle;
    NSView *content = window.contentView;
    // Native traffic lights start at y=20; the blank strip sits above them.
    NSRect frame = NSMakeRect(0, content.bounds.size.height - 20,
                             content.bounds.size.width, 20);
    LocalOpsDragStrip *strip = [[LocalOpsDragStrip alloc] initWithFrame:frame];
    strip.autoresizingMask = NSViewWidthSizable | NSViewMinYMargin;
    [content addSubview:strip positioned:NSWindowAbove relativeTo:nil];
    [strip release];
}
