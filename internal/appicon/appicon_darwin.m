#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#include <stdlib.h>
#include <string.h>

static void *png_of(NSImage *icon, int pixels, size_t *length) {
	NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:pixels pixelsHigh:pixels
		bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
	if (!icon || !rep) return NULL;
	[NSGraphicsContext saveGraphicsState];
	[NSGraphicsContext setCurrentContext:[NSGraphicsContext graphicsContextWithBitmapImageRep:rep]];
	[icon drawInRect:NSMakeRect(0, 0, pixels, pixels) fromRect:NSZeroRect operation:NSCompositingOperationCopy fraction:1];
	[NSGraphicsContext restoreGraphicsState];
	NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
	if (!png) return NULL;
	void *out = malloc(png.length);
	memcpy(out, png.bytes, png.length);
	*length = png.length;
	return out;
}

// Returns malloc'ed PNG bytes of the Finder icon for path, or NULL.
void *rockshinops_app_icon_png(const char *path, int pixels, size_t *length) {
	@autoreleasepool {
		NSImage *icon = path ? [[NSWorkspace sharedWorkspace] iconForFile:[NSString stringWithUTF8String:path]] : [NSImage imageNamed:NSImageNameComputer];
		return png_of(icon, pixels, length);
	}
}

// Returns malloc'ed PNG bytes of the generic icon of a folder, or of files
// with the extension ext (any file when ext is NULL).
void *rockshinops_type_icon_png(const char *ext, int folder, int pixels, size_t *length) {
	@autoreleasepool {
		UTType *type = folder ? UTTypeFolder : ext ? [UTType typeWithFilenameExtension:[NSString stringWithUTF8String:ext]] : UTTypeData;
		return png_of([[NSWorkspace sharedWorkspace] iconForContentType:type ?: UTTypeData], pixels, length);
	}
}
