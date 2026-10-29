#import <ApplicationServices/ApplicationServices.h>

// Delivers one status-item event through the owning app and session stream, with acknowledgement.
BOOL MBDeliverMenuBarEvent(CGEventRef event, pid_t pid);
