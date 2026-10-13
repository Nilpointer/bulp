package main

import (
	"context"
	"log"
	"slices"
	"sync"
	"time"

	"fyne.io/systray"
	"github.com/godbus/dbus/v5"

	"bulp/internal/tray"
	"bulp/internal/ui"
)

// Linux: a StatusNotifierItem tray icon (D-Bus, no cgo). Tray menus can't hold
// sliders, so the full control panel opens in its own window (see panel).
// Without a tray host (stock GNOME) the app is just that window.
type frontend struct {
	srv        *ui.Server
	panel      panel
	windowOnly bool

	mu      sync.Mutex
	ids     []string                     // light IDs the menu was last built for
	items   map[string]*systray.MenuItem // per-light on/off checkboxes
	menuGen chan struct{}                // closed when the menu is rebuilt
}

func newFrontend(srv *ui.Server) *frontend {
	f := &frontend{srv: srv, items: map[string]*systray.MenuItem{}}
	// With a tray icon, closing the panel just hides it.
	f.panel = newPanel(srv, func() {
		if f.windowOnly {
			f.quit()
		}
	})
	if !trayAvailable() {
		log.Print("no system tray found (on GNOME, install the AppIndicator extension: " +
			"https://extensions.gnome.org/extension/615/appindicator-support/); showing just the control panel")
		f.windowOnly = true
	}
	return f
}

func (f *frontend) quit() { f.panel.quit() }

func (f *frontend) run(ctx context.Context, url string) {
	if !f.windowOnly {
		start, end := systray.RunWithExternalLoop(func() {
			systray.SetIcon(tray.IconOff)
			systray.SetTitle("Bulp")
			systray.SetTooltip("Bulp")
			systray.SetOnTapped(f.openPanel) // left click
			f.buildMenu(nil)
			go f.poll(ctx)
		}, nil)
		start()
		defer end()
	}
	f.panel.run(ctx, url, f.windowOnly)
}

func (f *frontend) openPanel() { f.panel.show() }

// trayAvailable reports whether a StatusNotifierWatcher (the tray host) is
// running or can be started on the session bus.
func trayAvailable() bool {
	conn, err := dbus.SessionBus()
	if err != nil {
		return false
	}
	const watcher = "org.kde.StatusNotifierWatcher"
	var running bool
	if conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, watcher).Store(&running) == nil && running {
		return true
	}
	var activatable []string
	conn.BusObject().Call("org.freedesktop.DBus.ListActivatableNames", 0).Store(&activatable)
	return slices.Contains(activatable, watcher)
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
	if f.windowOnly {
		return
	}
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
	go f.onClick(gen, quit, f.quit)
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
