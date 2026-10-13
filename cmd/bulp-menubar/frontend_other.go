//go:build !linux

package main

import (
	"context"
	"fmt"

	"bulp/internal/ui"
)

// Other platforms: no tray integration yet, just serve the panel.
type frontend struct{ done chan struct{} }

func newFrontend(*ui.Server) *frontend { return &frontend{done: make(chan struct{})} }

func (*frontend) lightsChanged([]ui.Light) {}
func (f *frontend) quit()                  { close(f.done) }
func (f *frontend) run(_ context.Context, url string) {
	fmt.Println("Open the control panel at", url)
	<-f.done
}
