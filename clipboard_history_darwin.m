#import <AppKit/AppKit.h>
#import <ImageIO/ImageIO.h>
#include <stdbool.h>

static NSPasteboardType const historyRestoreType = @"dev.rockshinops.clipboard-history-restore";

int64_t rockshin_clipboard_change_count(void) {
    @autoreleasepool {
        return NSPasteboard.generalPasteboard.changeCount;
    }
}

static char *copy_data(NSData *data, int *length) {
    char *result = malloc(data.length);
    if (result == NULL) return NULL;
    memcpy(result, data.bytes, data.length);
    *length = (int)data.length;
    return result;
}

static char *read_clipboard_change(NSPasteboard *board, int64_t *change, int maxText, int maxImage, int *length,
                                   char **format, bool *accessRequired, bool *tooLarge) {
    if (@available(macOS 15.4, *)) {
        NSPasteboardAccessBehavior behavior = board.accessBehavior;
        // Let the system show its first prompt and wait for an explicit permission choice.
        if (behavior == NSPasteboardAccessBehaviorAsk || behavior == NSPasteboardAccessBehaviorAlwaysDeny) {
            *accessRequired = true;
            return NULL;
        }
    }

    int64_t current = board.changeCount;
    if (current == *change) return NULL;
    NSArray<NSPasteboardItem *> *items = board.pasteboardItems;
    *change = current;
    if (items.count != 1) return NULL;
    NSPasteboardItem *item = items.firstObject;
    NSArray<NSPasteboardType> *types = item.types;
    if ([types containsObject:historyRestoreType] ||
        [types containsObject:@"org.nspasteboard.ConcealedType"] ||
        [types containsObject:@"org.nspasteboard.TransientType"] ||
        [types containsObject:NSPasteboardTypeFileURL] ||
        [types containsObject:@"NSFilenamesPboardType"]) {
        return NULL;
    }

    NSPasteboardType imageTypes[] = {NSPasteboardTypePNG, @"public.jpeg", NSPasteboardTypeTIFF};
    char *imageFormats[] = {"png", "jpeg", "tiff"};
    bool imageTooLarge = false;
    for (size_t i = 0; i < sizeof(imageTypes) / sizeof(imageTypes[0]); i++) {
        if (![types containsObject:imageTypes[i]]) continue;
        NSData *data = [item dataForType:imageTypes[i]];
        if (board.changeCount != current) return NULL;
        if (data.length == 0) continue;
        if (data.length > (NSUInteger)maxImage) {
            imageTooLarge = true;
            continue;
        }
        *format = imageFormats[i];
        return copy_data(data, length);
    }
    if (imageTooLarge) {
        *tooLarge = true;
        return NULL;
    }
    if (![types containsObject:NSPasteboardTypeString]) return NULL;
    NSData *text = [item dataForType:NSPasteboardTypeString];
    if (board.changeCount != current || text.length == 0) return NULL;
    if (text.length > (NSUInteger)maxText) {
        *tooLarge = true;
        return NULL;
    }
    *format = "txt";
    return copy_data(text, length);
}

char *rockshin_clipboard_read(int64_t *change, int maxText, int maxImage, int *length,
                             char **format, bool *accessRequired, bool *tooLarge) {
    @autoreleasepool {
        return read_clipboard_change(NSPasteboard.generalPasteboard, change, maxText, maxImage, length, format, accessRequired, tooLarge);
    }
}

bool rockshin_clipboard_write(const char *format, const void *bytes, int length) {
    @autoreleasepool {
        NSPasteboardType type = nil;
        if (strcmp(format, "txt") == 0) type = NSPasteboardTypeString;
        else if (strcmp(format, "png") == 0) type = NSPasteboardTypePNG;
        else if (strcmp(format, "jpeg") == 0) type = @"public.jpeg";
        else if (strcmp(format, "tiff") == 0) type = NSPasteboardTypeTIFF;
        if (type == nil) return false;

        NSPasteboardItem *item = [[NSPasteboardItem alloc] init];
        // Mark the same item so every recorder can skip the restore without missing later external copies.
        bool written = [item setData:[NSData dataWithBytes:bytes length:(NSUInteger)length] forType:type] &&
                       [item setData:[NSData data] forType:historyRestoreType];
        if (written) {
            NSPasteboard *board = NSPasteboard.generalPasteboard;
            [board clearContents];
            written = [board writeObjects:@[item]];
        }
        [item release];
        return written;
    }
}

char *rockshin_clipboard_preview(const char *path, int pixels, int *length) {
    @autoreleasepool {
        NSURL *url = [NSURL fileURLWithFileSystemRepresentation:path isDirectory:NO relativeToURL:nil];
        CGImageSourceRef source = CGImageSourceCreateWithURL((CFURLRef)url, NULL);
        if (source == NULL) return NULL;
        NSDictionary *options = @{
            (NSString *)kCGImageSourceCreateThumbnailFromImageAlways: @YES,
            (NSString *)kCGImageSourceCreateThumbnailWithTransform: @YES,
            (NSString *)kCGImageSourceThumbnailMaxPixelSize: @(pixels)
        };
        CGImageRef image = CGImageSourceCreateThumbnailAtIndex(source, 0, (CFDictionaryRef)options);
        CFRelease(source);
        if (image == NULL) return NULL;
        NSMutableData *png = [NSMutableData data];
        CGImageDestinationRef destination = CGImageDestinationCreateWithData((CFMutableDataRef)png, CFSTR("public.png"), 1, NULL);
        if (destination == NULL) {
            CGImageRelease(image);
            return NULL;
        }
        CGImageDestinationAddImage(destination, image, NULL);
        bool encoded = CGImageDestinationFinalize(destination);
        CFRelease(destination);
        CGImageRelease(image);
        if (!encoded || png.length == 0) return NULL;
        return copy_data(png, length);
    }
}
