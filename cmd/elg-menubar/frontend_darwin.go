package main

import (
	"context"

	"elg/internal/menubar"
	"elg/internal/ui"
)

// macOS: NSStatusItem + NSPopover hosting a WKWebView.
type frontend struct{}

func newFrontend(*ui.Server) *frontend { return &frontend{} }

func (*frontend) lightsChanged(lights []ui.Light) { menubar.SetActive(ui.AnyOn(lights)) }
func (*frontend) quit()                           { menubar.Quit() }
func (*frontend) run(_ context.Context, url string) {
	menubar.Run(url, "lightbulb")
}
