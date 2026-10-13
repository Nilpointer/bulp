# bulps

Control Elgato lights (Key Light, Ring Light, Light Strip, …) from Go, over their local HTTP API.

- `elgato/` – library: Bonjour discovery and an HTTP client for the lights' API (port 9123)
- `cmd/elg` – CLI
- `cmd/elg-menubar` – menu-bar app (macOS popover) / system-tray app (Linux)

## Build

Go is pinned with [mise](https://mise.jdx.dev):

```sh
mise run build         # bin/elg, bin/elg-menubar (macOS needs cgo/Xcode CLT)
mise run build-linux   # static Linux binaries in bin/linux-{amd64,arm64}/
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
- **Linux:** StatusNotifierItem tray icon (KDE, most desktops; GNOME needs the AppIndicator extension).
  Left click opens the panel in a Chromium-based app window (falls back to the default browser);
  the right-click menu toggles lights.

`-host a,b` skips Bonjour discovery if multicast is blocked.
