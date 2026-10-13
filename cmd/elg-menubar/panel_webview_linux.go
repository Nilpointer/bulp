//go:build cgo

package main

import (
	"context"
	"log"
	"os"

	"elg/internal/ui"
	"elg/internal/webview"
)

// appID names the app to the desktop (Wayland, .desktop file, Flatpak).
const appID = "de.codecauldron.Elg"

// webviewPanel is a native GTK window with WebKitGTK.
type webviewPanel struct{}

func (webviewPanel) show() { webview.Show() }
func (webviewPanel) quit() { webview.Quit() }
func (webviewPanel) run(ctx context.Context, url string, show bool) {
	go func() {
		<-ctx.Done()
		webview.Quit()
	}()
	webview.Run(url, show)
}

// newPanel prefers the native window; closed runs when the user closes it.
// If the app is already running, that instance shows its window and this one exits.
func newPanel(srv *ui.Server, closed func()) panel {
	switch webview.Init(appID, closed) {
	case webview.Primary:
		return webviewPanel{}
	case webview.AlreadyRunning:
		os.Exit(0)
	}
	log.Print("can't start GTK; opening the control panel in a browser instead")
	return newBrowserPanel(srv, closed)
}
