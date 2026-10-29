package windowchrome

/*
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore
#include <stdint.h>
void localops_install_top_drag(uintptr_t handle);
*/
import "C"

// InstallTopDrag gives the 20-point empty strip a native drag and double-click zoom target. Call on
// the main thread; WKWebView's asynchronous DOM event route cannot own it.
func InstallTopDrag(handle uintptr) { C.localops_install_top_drag(C.uintptr_t(handle)) }
