#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

// Draws "symbol text  symbol text" at 2x into a bitmap 18pt tall and returns malloc'ed PNG bytes, or NULL.
// Only alpha matters: the status item shows it as a template image tinted by the system.
void *localops_tray_png(const char **symbols, const char **texts, int count, size_t *length) {
	@autoreleasepool {
		const CGFloat scale = 2, height = 18 * scale, iconGap = 3 * scale, itemGap = 8 * scale;
		CGFloat size = [NSFont menuBarFontOfSize:0].pointSize * scale;
		NSDictionary *attributes = @{NSFontAttributeName: [NSFont monospacedDigitSystemFontOfSize:size weight:NSFontWeightRegular], NSForegroundColorAttributeName: NSColor.blackColor};
		NSImageSymbolConfiguration *config = [NSImageSymbolConfiguration configurationWithPointSize:size weight:NSFontWeightRegular];
		NSMutableArray<NSImage *> *icons = [NSMutableArray array];
		NSMutableArray<NSAttributedString *> *labels = [NSMutableArray array];
		CGFloat width = 0;
		for (int i = 0; i < count; i++) {
			NSImage *icon = [[NSImage imageWithSystemSymbolName:[NSString stringWithUTF8String:symbols[i]] accessibilityDescription:nil] imageWithSymbolConfiguration:config];
			if (!icon) return NULL;
			NSAttributedString *label = [[NSAttributedString alloc] initWithString:[NSString stringWithUTF8String:texts[i]] attributes:attributes];
			[icons addObject:icon];
			[labels addObject:label];
			width += (i ? itemGap : 0) + icon.size.width + (label.length ? iconGap + label.size.width : 0);
		}
		NSInteger pixels = (NSInteger)ceil(width);
		NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:pixels pixelsHigh:(NSInteger)height
			bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
		if (!rep || pixels == 0) return NULL;
		[NSGraphicsContext saveGraphicsState];
		[NSGraphicsContext setCurrentContext:[NSGraphicsContext graphicsContextWithBitmapImageRep:rep]];
		CGFloat x = 0;
		for (int i = 0; i < count; i++) {
			NSImage *icon = icons[i];
			NSAttributedString *label = labels[i];
			if (i) x += itemGap;
			[icon drawInRect:NSMakeRect(x, round((height - icon.size.height) / 2), icon.size.width, icon.size.height) fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1];
			x += icon.size.width;
			if (label.length) {
				x += iconGap;
				[label drawAtPoint:NSMakePoint(x, round((height - label.size.height) / 2))];
				x += label.size.width;
			}
		}
		[NSGraphicsContext restoreGraphicsState];
		NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
		if (!png) return NULL;
		void *out = malloc(png.length);
		memcpy(out, png.bytes, png.length);
		*length = png.length;
		return out;
	}
}
