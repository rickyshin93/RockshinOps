#import "events_darwin.h"

typedef struct {
    CGEventRef event;
    pid_t pid;
    int64_t marker;
    CFMachPortRef processTap;
    CFMachPortRef sessionTap;
    BOOL delivered;
} MBDelivery;

static CGEventRef MBProcessEvent(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *context) {
    MBDelivery *delivery=context;
    if(type==kCGEventNull && CGEventGetIntegerValueField(event,kCGEventSourceUserData)==delivery->marker) {
        CGEventTapEnable(delivery->processTap,false);
        CGEventPost(kCGSessionEventTap,delivery->event);
        return NULL;
    }
    return event;
}
static CGEventRef MBSessionEvent(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *context) {
    MBDelivery *delivery=context;
    if(type==CGEventGetType(delivery->event) && CGEventGetIntegerValueField(event,kCGEventSourceUserData)==delivery->marker) {
        CGEventTapEnable(delivery->sessionTap,false);
        CGEventPostToPid(delivery->pid,delivery->event);
        delivery->delivered=YES;
        CFRunLoopStop(CFRunLoopGetCurrent());
    }
    return event;
}
BOOL MBDeliverMenuBarEvent(CGEventRef event, pid_t pid) {
    MBDelivery delivery={.event=event,.pid=pid,.marker=(int64_t)(uintptr_t)event};
    CGEventSetIntegerValueField(event,kCGEventSourceUserData,delivery.marker);
    delivery.processTap=CGEventTapCreateForPid(pid,kCGTailAppendEventTap,kCGEventTapOptionDefault,CGEventMaskBit(kCGEventNull),MBProcessEvent,&delivery);
    delivery.sessionTap=CGEventTapCreate(kCGSessionEventTap,kCGTailAppendEventTap,kCGEventTapOptionListenOnly,CGEventMaskBit(CGEventGetType(event)),MBSessionEvent,&delivery);
    if(!delivery.processTap || !delivery.sessionTap) {
        if(delivery.processTap){CFMachPortInvalidate(delivery.processTap);CFRelease(delivery.processTap);}
        if(delivery.sessionTap){CFMachPortInvalidate(delivery.sessionTap);CFRelease(delivery.sessionTap);}
        return NO;
    }
    CFRunLoopRef loop=CFRunLoopGetCurrent();
    CFRunLoopSourceRef processSource=CFMachPortCreateRunLoopSource(NULL,delivery.processTap,0);
    CFRunLoopSourceRef sessionSource=CFMachPortCreateRunLoopSource(NULL,delivery.sessionTap,0);
    CFRunLoopAddSource(loop,processSource,kCFRunLoopDefaultMode);
    CFRunLoopAddSource(loop,sessionSource,kCFRunLoopDefaultMode);
    CGEventRef wake=CGEventCreate(NULL);
    CGEventSetIntegerValueField(wake,kCGEventSourceUserData,delivery.marker);
    CGEventPostToPid(pid,wake);
    CFRelease(wake);
    // A bounded protocol acknowledgement, not a delay added to successful delivery.
    CFRunLoopRunInMode(kCFRunLoopDefaultMode,0.25,false);
    CFRunLoopRemoveSource(loop,processSource,kCFRunLoopDefaultMode);
    CFRunLoopRemoveSource(loop,sessionSource,kCFRunLoopDefaultMode);
    CFMachPortInvalidate(delivery.processTap);CFMachPortInvalidate(delivery.sessionTap);
    CFRelease(processSource);CFRelease(sessionSource);
    CFRelease(delivery.processTap);CFRelease(delivery.sessionTap);
    return delivery.delivered;
}
