//go:build !cgo

package main

import "elg/internal/ui"

// Without cgo there's no native window; closed runs when the page is closed.
func newPanel(srv *ui.Server, closed func()) panel { return newBrowserPanel(srv, closed) }
