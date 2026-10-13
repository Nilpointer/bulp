package main

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"elg/internal/ui"
)

// panel shows the control panel and owns the main loop.
type panel interface {
	// run blocks until quit, showing url when asked; with show the panel opens
	// right away. Must be called from main().
	run(ctx context.Context, url string, show bool)
	show() // open or raise the panel; any goroutine
	quit() // any goroutine
}

// browserPanel opens the control panel in a browser: a chromeless app window
// if a Chromium-based browser is installed, a tab in the default one otherwise.
type browserPanel struct {
	url      string
	done     chan struct{}
	quitOnce sync.Once
}

// newBrowserPanel creates a browser panel. closed runs when the page goes away
// for good; the page sends a beacon on unload, and a reload talks to the
// server again right away.
func newBrowserPanel(srv *ui.Server, closed func()) *browserPanel {
	srv.Closed = func() {
		closedAt := time.Now()
		time.AfterFunc(3*time.Second, func() {
			if !srv.LastRequest().After(closedAt) {
				closed()
			}
		})
	}
	return &browserPanel{done: make(chan struct{})}
}

func (p *browserPanel) run(ctx context.Context, url string, show bool) {
	p.url = url
	if show {
		p.show()
	}
	select {
	case <-p.done:
	case <-ctx.Done():
	}
}

func (p *browserPanel) quit() { p.quitOnce.Do(func() { close(p.done) }) }

// Chromium-based browsers can open a URL as a chromeless "app" window.
var appBrowsers = []string{
	"chromium", "chromium-browser", "google-chrome", "google-chrome-stable",
	"brave-browser", "brave", "microsoft-edge", "microsoft-edge-stable", "vivaldi",
}

func (p *browserPanel) show() {
	url := p.url
	for _, b := range appBrowsers {
		if path, err := exec.LookPath(b); err == nil {
			cmd := exec.Command(path, "--app="+url, "--window-size=389,560", "--class=elg-menubar")
			if err := cmd.Start(); err == nil {
				go cmd.Wait()
				return
			}
		}
	}
	// Fallback: whatever the desktop's default browser is.
	cmd := exec.Command("xdg-open", url)
	if err := cmd.Start(); err != nil {
		log.Printf("can't open control panel (%v); open %s in a browser", err, strings.TrimSpace(url))
		return
	}
	go cmd.Wait()
}
