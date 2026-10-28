package main

/*
#cgo LDFLAGS: -framework CFNetwork -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <CFNetwork/CFNetwork.h>
#include <stdlib.h>

// Return direct (0), HTTP (1), SOCKS (2), or unsupported/error (-1).
static int updateProxy(const char *raw, char *host, size_t hostSize, int *port) {
    CFStringRef text = CFStringCreateWithCString(NULL, raw, kCFStringEncodingUTF8);
    CFURLRef url = text ? CFURLCreateWithString(NULL, text, NULL) : NULL;
    CFDictionaryRef settings = CFNetworkCopySystemProxySettings();
    CFArrayRef proxies = url && settings ? CFNetworkCopyProxiesForURL(url, settings) : NULL;
    int result = -1;
    if (proxies && CFArrayGetCount(proxies) > 0) {
        CFDictionaryRef proxy = CFArrayGetValueAtIndex(proxies, 0);
        CFStringRef type = CFDictionaryGetValue(proxy, kCFProxyTypeKey);
        if (type && CFEqual(type, kCFProxyTypeNone)) result = 0;
        else if (type && (CFEqual(type, kCFProxyTypeHTTP) || CFEqual(type, kCFProxyTypeHTTPS) || CFEqual(type, kCFProxyTypeSOCKS)) &&
                 !CFDictionaryContainsKey(proxy, kCFProxyUsernameKey) && !CFDictionaryContainsKey(proxy, kCFProxyPasswordKey)) {
            CFStringRef name = CFDictionaryGetValue(proxy, kCFProxyHostNameKey);
            CFNumberRef number = CFDictionaryGetValue(proxy, kCFProxyPortNumberKey);
            if (name && number && CFStringGetCString(name, host, hostSize, kCFStringEncodingUTF8) && CFNumberGetValue(number, kCFNumberIntType, port))
                result = CFEqual(type, kCFProxyTypeSOCKS) ? 2 : 1;
        }
    }
    if (proxies) CFRelease(proxies);
    if (settings) CFRelease(settings);
    if (url) CFRelease(url);
    if (text) CFRelease(text);
    return result;
}
*/
import "C"

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"unsafe"
)

func systemUpdateProxy(target *url.URL) (*url.URL, error) {
	raw := C.CString(target.String())
	defer C.free(unsafe.Pointer(raw))
	var host [1024]C.char
	var port C.int
	kind := C.updateProxy(raw, &host[0], C.size_t(len(host)), &port)
	switch kind {
	case 0:
		return nil, nil
	case 1, 2:
		scheme := "http"
		if kind == 2 {
			scheme = "socks5"
		}
		return &url.URL{Scheme: scheme, Host: net.JoinHostPort(C.GoString(&host[0]), strconv.Itoa(int(port)))}, nil
	default:
		return nil, fmt.Errorf("cannot use the macOS system proxy for updates: unsupported or unavailable proxy configuration")
	}
}
