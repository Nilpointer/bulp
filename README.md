# bulps

> **AI generated:** this project was written with [Claude Code](https://claude.com/claude-code), Anthropic's
> AI coding assistant, directed by its author and tested against real lights.
> Expect the rough edges of a young project, and please report anything that doesn't work.

Control Elgato lights (Key Light, Ring Light, Light Strip, …) from Go, over their local HTTP API.

<p>
  <img src="docs/screenshots/panel-light.png" alt="Control panel in light mode: power, color temperature and brightness for an Elgato Ring Light" width="389">
  <img src="docs/screenshots/panel-dark.png" alt="The same control panel in dark mode" width="389">
</p>

Downloads (Linux AppImage and Flatpak, macOS app, CLI) are on the [releases page](https://github.com/Nilpointer/bulps/releases).

- `elgato/` – library: Bonjour discovery and an HTTP client for the lights' API (port 9123)
- `cmd/elg` – CLI
- `cmd/elg-menubar` – menu-bar app (macOS popover) / system-tray app (Linux)

## Build

Go is pinned with [mise](https://mise.jdx.dev):

```sh
mise run build         # bin/elg, bin/elg-menubar (cgo: Xcode CLT on macOS, gtk4 + webkitgtk-6.0 dev packages on Linux)
mise run build-linux   # static Linux binaries in bin/linux-{amd64,arm64}/ (no cgo: panel opens in a browser)
mise run test
mise run flatpak       # build + install the Flatpak locally, then: flatpak run de.codecauldron.Elg
mise run appimage      # build/Elgato_Lights-<version>-x86_64.AppImage (needs podman or docker)
mise run macos-app     # build/Elgato_Lights-<version>-macOS.zip, universal .app (on macOS)
```

The Flatpak (`packaging/flatpak/de.codecauldron.Elg.yml`) uses the GNOME 51 runtime, which ships GTK 4 and
WebKitGTK; one-time setup: `flatpak install flathub org.flatpak.Builder`. The CLI is included too:
`flatpak run --command=elg de.codecauldron.Elg list`. Desktop entry, AppStream metadata and icon live in
`packaging/`.

The AppImage doesn't bundle GTK or WebKit (~17 MB). It carries two builds of `elg-menubar`: one linked
against the system's GTK 4 + WebKitGTK 6.0 (built in an Ubuntu 24.04 container so it runs on glibc ≥ 2.34),
used when those libraries are installed (Ubuntu 24.04+, Debian 12+, current Fedora/Arch), and a static
one that opens the panel in a browser everywhere else. `AppRun` picks between them.

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

`-host a,b` skips Bonjour discovery if multicast is blocked. On Linux, discovery goes through the Avahi
daemon when it's running (much more reliable than querying directly), else plain multicast.

## macOS app downloads

`.github/workflows/macos.yml` builds the universal `.app` on a GitHub macOS runner for every `v*` tag and
attaches it to that release (or run it manually, optionally naming a tag to attach to). The app is
ad-hoc signed, not notarized: on first launch macOS refuses to open it. Allow it via System Settings →
Privacy & Security → "Open Anyway", or `xattr -dr com.apple.quarantine "Elgato Lights.app"`.
