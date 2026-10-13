//go:build darwin

// Package menubar shows a macOS menu-bar icon with a popover hosting a web view.
package menubar

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#include <stdlib.h>
#include "menubar.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
)

func init() {
	// Cocoa must run on the main thread; keep main() pinned to it.
	runtime.LockOSThread()
}

// Run shows the menu-bar item and blocks running the Cocoa event loop.
// The popover loads url. symbol is an SF Symbol name for the icon.
// Must be called from main().
func Run(url, symbol string) {
	cu, cs := C.CString(url), C.CString(symbol)
	defer C.free(unsafe.Pointer(cu))
	defer C.free(unsafe.Pointer(cs))
	C.mb_run(cu, cs)
}

// SetActive switches the icon between its filled and outline variant.
// Safe to call from any goroutine.
func SetActive(active bool) {
	C.mb_set_active(C.bool(active))
}

// Quit terminates the app. Safe to call from any goroutine.
func Quit() {
	C.mb_quit()
}
