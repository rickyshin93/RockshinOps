#import <AppKit/AppKit.h>
#include <stdint.h>

extern void rockshin_launcher_open_app_completed(uintptr_t token, char *message);

void rockshin_launcher_open_app(const char *path, uintptr_t token) {
    @autoreleasepool {
        NSURL *url = [NSURL fileURLWithFileSystemRepresentation:path isDirectory:YES relativeToURL:nil];
        [NSWorkspace.sharedWorkspace openApplicationAtURL:url
                                            configuration:NSWorkspaceOpenConfiguration.configuration
                                        completionHandler:^(NSRunningApplication *app, NSError *error) {
            const char *message = error ? error.localizedDescription.UTF8String : app ? NULL : "macOS returned no running application";
            rockshin_launcher_open_app_completed(token, (char *)message);
        }];
    }
}
