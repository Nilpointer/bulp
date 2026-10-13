package main

import (
	"context"
	"log"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"

	"elg/internal/tray"
	"elg/internal/ui"
)

// Linux: a StatusNotifierItem tray icon (D-Bus, no cgo). Tray menus can't hold
// sliders, so the full control panel opens in a small app-style browser window.
type frontend struct {
	srv *ui.Server
	url string

	mu      sync.Mutex
	ids     []string                     // light IDs the menu was last built for
	items   map[string]*systray.MenuItem // per-light on/off checkboxes
	menuGen chan struct{}                // closed when the menu is rebuilt
}

func newFrontend(srv *ui.Server) *frontend {
	return &frontend{srv: srv, items: map[string]*systray.MenuItem{}}
}

func (f *frontend) quit() { systray.Quit() }

func (f *frontend) run(ctx context.Context, url string) {
	f.url = url
	systray.Run(func() {
		systray.SetIcon(tray.IconOff)
		systray.SetTitle("Elgato Lights")
		systray.SetTooltip("Elgato Lights")
		systray.SetOnTapped(f.openPanel) // left click
		f.buildMenu(nil)
		go f.poll(ctx)
	}, nil)
}

// poll keeps the tray icon and menu in sync with the lights.
func (f *frontend) poll(ctx context.Context) {
	for {
		f.srv.Lights(ctx) // triggers lightsChanged
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func (f *frontend) lightsChanged(lights []ui.Light) {
	if ui.AnyOn(lights) {
		systray.SetIcon(tray.IconOn)
	} else {
		systray.SetIcon(tray.IconOff)
	}

	ids := make([]string, len(lights))
	for i, l := range lights {
		ids[i] = l.ID
	}
	f.mu.Lock()
	changed := !slices.Equal(ids, f.ids)
	f.mu.Unlock()
	if changed {
		f.buildMenu(lights)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range lights {
		if it := f.items[l.ID]; it != nil {
			it.SetTitle(l.Name)
			if l.IsOn() {
				it.Check()
			} else {
				it.Uncheck()
			}
		}
	}
}

// buildMenu recreates the menu; systray can't insert items in the middle,
// so a changed set of lights means starting over.
func (f *frontend) buildMenu(lights []ui.Light) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.menuGen != nil {
		close(f.menuGen) // stop click listeners of the previous menu
	}
	gen := make(chan struct{})
	f.menuGen = gen
	systray.ResetMenu()
	f.items = map[string]*systray.MenuItem{}
	f.ids = f.ids[:0]

	open := systray.AddMenuItem("Open Control Center", "Show the control panel")
	go f.onClick(gen, open, f.openPanel)
	systray.AddSeparator()

	if len(lights) == 0 {
		systray.AddMenuItem("Searching for lights…", "").Disable()
	}
	for _, l := range lights {
		it := systray.AddMenuItemCheckbox(l.Name, "Turn on/off", l.IsOn())
		if l.State == nil {
			it.Disable()
		}
		f.items[l.ID] = it
		f.ids = append(f.ids, l.ID)
		id := l.ID
		go f.onClick(gen, it, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := f.srv.SetOn(ctx, id, !it.Checked()); err != nil {
				log.Printf("%s: %v", id, err)
			}
			f.srv.Lights(ctx) // refresh checkmarks and icon
		})
	}

	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "")
	go f.onClick(gen, quit, systray.Quit)
}

func (f *frontend) onClick(gen chan struct{}, it *systray.MenuItem, fn func()) {
	for {
		select {
		case <-gen:
			return
		case <-it.ClickedCh:
			fn()
		}
	}
}

// Chromium-based browsers can open a URL as a chromeless "app" window.
var appBrowsers = []string{
	"chromium", "chromium-browser", "google-chrome", "google-chrome-stable",
	"brave-browser", "brave", "microsoft-edge", "microsoft-edge-stable", "vivaldi",
}

func (f *frontend) openPanel() {
	for _, b := range appBrowsers {
		if path, err := exec.LookPath(b); err == nil {
			cmd := exec.Command(path, "--app="+f.url, "--window-size=389,560", "--class=elg-menubar")
			if err := cmd.Start(); err == nil {
				go cmd.Wait()
				return
			}
		}
	}
	// Fallback: whatever the desktop's default browser is.
	cmd := exec.Command("xdg-open", f.url)
	if err := cmd.Start(); err != nil {
		log.Printf("can't open control panel (%v); open %s in a browser", err, strings.TrimSpace(f.url))
		return
	}
	go cmd.Wait()
}
