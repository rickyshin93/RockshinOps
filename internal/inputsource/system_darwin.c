#include <Carbon/Carbon.h>
#include <stdlib.h>

typedef struct {
    TISInputSourceRef previous;
    TISInputSourceRef selected;
} KeyboardSelection;

static Boolean isSystemKeyboard(TISInputSourceRef source) {
    if (!source) return false;
    CFStringRef identifier = TISGetInputSourceProperty(source, kTISPropertyInputSourceID);
    return identifier && CFStringHasPrefix(identifier, CFSTR("com.apple.keylayout.")) &&
        CFEqual(TISGetInputSourceProperty(source, kTISPropertyInputSourceType), kTISTypeKeyboardLayout) &&
        CFEqual(TISGetInputSourceProperty(source, kTISPropertyInputSourceIsASCIICapable), kCFBooleanTrue) &&
        CFEqual(TISGetInputSourceProperty(source, kTISPropertyInputSourceIsEnabled), kCFBooleanTrue) &&
        CFEqual(TISGetInputSourceProperty(source, kTISPropertyInputSourceIsSelectCapable), kCFBooleanTrue);
}

int32_t rockshinops_use_system_keyboard(void **selection) {
    TISInputSourceRef previous = TISCopyCurrentKeyboardInputSource();
    if (!previous) return paramErr;
    TISInputSourceRef selected = TISCopyCurrentASCIICapableKeyboardLayoutInputSource();
    if (!isSystemKeyboard(selected)) {
        if (selected) CFRelease(selected);
        selected = NULL;
        CFArrayRef sources = TISCreateInputSourceList(NULL, false);
        for (CFIndex i = 0; sources && i < CFArrayGetCount(sources); i++) {
            TISInputSourceRef source = (TISInputSourceRef)CFArrayGetValueAtIndex(sources, i);
            if (isSystemKeyboard(source)) { selected = (TISInputSourceRef)CFRetain(source); break; }
        }
        if (sources) CFRelease(sources);
    }
    if (!selected) { CFRelease(previous); return paramErr; }
    KeyboardSelection *saved = malloc(sizeof(*saved));
    if (!saved) { CFRelease(previous); CFRelease(selected); return memFullErr; }
    OSStatus status = TISSelectInputSource(selected);
    if (status != noErr) {
        CFRelease(previous); CFRelease(selected); free(saved);
        return status;
    }
    *saved = (KeyboardSelection){previous, selected};
    *selection = saved;
    return noErr;
}

int32_t rockshinops_restore_keyboard(void *selection) {
    if (!selection) return noErr;
    KeyboardSelection *saved = selection;
    TISInputSourceRef current = TISCopyCurrentKeyboardInputSource();
    if (!current) return paramErr;
    // A deliberate user switch, or another app's input context, takes precedence.
    Boolean unchanged = CFEqual(TISGetInputSourceProperty(current, kTISPropertyInputSourceID),
                                TISGetInputSourceProperty(saved->selected, kTISPropertyInputSourceID));
    OSStatus status = unchanged ? TISSelectInputSource(saved->previous) : noErr;
    CFRelease(current);
    if (status == noErr) {
        CFRelease(saved->previous); CFRelease(saved->selected); free(saved);
    }
    return status;
}
