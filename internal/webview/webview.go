//go:build linux && cgo

// Package webview shows the control panel in a native GTK window hosting a
// WebKitGTK web view (Linux counterpart of the macOS popover). It runs as a
// single-instance GtkApplication: launching the app again raises the window.
package webview

/*
#cgo pkg-config: gtk4 webkitgtk-6.0
#include <stdlib.h>
#include "webview.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
)

func init() {
	// GTK must run on the main thread; keep main() pinned to it.
	runtime.LockOSThread()
}

// Status is the outcome of Init.
type Status int

const (
	Unavailable    Status = iota // GTK can't start (no display); don't use the package
	Primary                      // this process runs the app
	AlreadyRunning               // another instance runs and was asked to show its window
)

var onClose func()

// Init starts GTK and registers the application under appID (reverse DNS,
// matching the .desktop file). closed runs on the GTK thread whenever the user
// closes the window, which only hides it. Must be called from main().
func Init(appID string, closed func()) Status {
	onClose = closed
	cid := C.CString(appID)
	defer C.free(unsafe.Pointer(cid))
	return Status(C.wv_init(cid))
}

// Run blocks running the application until Quit. The window shows url; with
// show it opens right away, otherwise on Show or a second launch.
// Must be called from main() after Init returned Primary.
func Run(url string, show bool) {
	cu := C.CString(url)
	defer C.free(unsafe.Pointer(cu))
	C.wv_run(cu, C.bool(show))
}

// Show opens or raises the window. Safe to call from any goroutine.
func Show() { C.wv_show() }

// Quit stops Run. Safe to call from any goroutine.
func Quit() { C.wv_quit() }

//export goWindowClosed
func goWindowClosed() {
	if onClose != nil {
		onClose()
	}
}
