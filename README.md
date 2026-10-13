# bulps

Control Elgato lights (Key Light, Ring Light, Light Strip, …) from Go, over their local HTTP API.

- `elgato/` – library: Bonjour discovery and an HTTP client for the lights' API (port 9123)
- `cmd/elg` – CLI
- `cmd/elg-menubar` – menu-bar app (macOS popover) / system-tray app (Linux)

## Build

Go is pinned with [mise](https://mise.jdx.dev):

```sh
mise run build         # bin/elg, bin/elg-menubar (cgo: Xcode CLT on macOS, gtk4 + webkitgtk-6.0 dev packages on Linux)
mise run build-linux   # static Linux binaries in bin/linux-{amd64,arm64}/ (no cgo: panel opens in a browser)
mise run test
```

## CLI

```sh
elg list                    # discover lights
elg status
elg on | off | toggle
elg brightness 40           # or +10 / -10
elg temp 4500               # Kelvin, 2900–7000
elg color 200 80            # hue, saturation (color lights)
elg identify | info | settings | rename "Desk"
```

Flags: `-name <substr>` to pick lights, `-host <ip>` (or `ELG_HOST`) to skip discovery, `-json`.

## Menu-bar / tray app

- **macOS:** menu-bar icon with a popover (brightness, color temperature, power, details).
- **Linux:** StatusNotifierItem tray icon. Left click opens the panel in a native GTK/WebKitGTK window;
  the right-click menu toggles lights. Without a tray host the app is just the panel window and exits
  when it's closed. It's a single-instance GtkApplication (`de.codecauldron.Elg`): launching it again raises the
  window. Builds without cgo open the panel in a browser instead
  (a Chromium-based app window if available, else the default browser).

Linux build deps: `sudo dnf install gtk4-devel webkitgtk6.0-devel` (Debian/Ubuntu: `libgtk-4-dev libwebkitgtk-6.0-dev`).

### Tray on GNOME

KDE, Cinnamon, MATE, Budgie, XFCE, Ubuntu's GNOME and waybar (`tray` module) show the icon out of the box.
Stock GNOME (e.g. Fedora Workstation, Arch) has no system tray; install the
[AppIndicator extension](https://extensions.gnome.org/extension/615/appindicator-support/):

```sh
sudo dnf install gnome-shell-extension-appindicator   # same package name on Arch (pacman) and Debian (apt)
# log out and back in, then:
gnome-extensions enable appindicatorsupport@rgcjonas.gmail.com
```

`-host a,b` skips Bonjour discovery if multicast is blocked.
