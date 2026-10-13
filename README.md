# Bulp

> **AI generated:** this project was written with [Claude Code](https://claude.com/claude-code), Anthropic's
> AI coding assistant, directed by its author and tested against real lights.
> Expect the rough edges of a young project, and please report anything that doesn't work.

Bulp controls Elgato lights (Key Light, Key Light Air/Mini/Neo, Ring Light, Light Strip, …) from the Linux
desktop: a tray icon to switch lights on and off, and a control panel for brightness, color temperature
and color. It talks to the lights directly over your local network; no account, no cloud.
There's also a command-line tool, `bulp`.

<p>
  <img src="docs/screenshots/panel-light.png" alt="Bulp's control panel in light mode, listing several lights with power buttons and sliders for color temperature, brightness, hue and saturation" width="389">
  <img src="docs/screenshots/panel-dark.png" alt="The same control panel in dark mode" width="389">
</p>

Bulp is an unofficial app, not affiliated with or endorsed by Elgato or Corsair. On macOS and Windows, use
Elgato's official Control Center app (the `bulp` CLI works there too).

## Install

Downloads are on the [releases page](https://github.com/Nilpointer/bulps/releases):

| Download | |
|---|---|
| **Flatpak** `de.codecauldron.Bulp.flatpak` | `flatpak install --user de.codecauldron.Bulp.flatpak`, then start Bulp from your app launcher. Pulls the GNOME runtime from Flathub. Includes the CLI: `flatpak run --command=bulp de.codecauldron.Bulp`. |
| **AppImage** `Bulp-<version>-x86_64.AppImage` | `chmod +x` and run. Shows the panel in a native window if GTK 4 + WebKitGTK 6.0 are installed (Ubuntu 24.04+, Debian 12+, current Fedora/Arch), otherwise in your browser. |
| **CLI** `bulp-<os>-<arch>` | Single static binary for Linux and macOS (amd64/arm64). |

### Tray icon on GNOME

KDE, Cinnamon, MATE, Budgie, XFCE, Ubuntu's GNOME and waybar (`tray` module) show the tray icon out of the
box. Stock GNOME (e.g. Fedora Workstation, Arch) has no system tray; install the
[AppIndicator extension](https://extensions.gnome.org/extension/615/appindicator-support/):

```sh
sudo dnf install gnome-shell-extension-appindicator   # same package name on Arch (pacman) and Debian (apt)
# log out and back in, then:
gnome-extensions enable appindicatorsupport@rgcjonas.gmail.com
```

Without a tray, Bulp runs as just the control panel window and quits when you close it.

## Using Bulp

- **Tray icon:** left click opens the control panel; the right-click menu switches each light on or off.
  The icon lights up while any light is on.
- **Control panel:** power, color temperature and brightness per light; hue and saturation for color
  lights such as the Light Strip. The power button at the top switches all lights at once; the sliders
  icon on a light shows its details (model, firmware, serial number, address, Wi-Fi signal) and an
  Identify button that makes the light blink.
- **One instance:** launching Bulp again brings up the panel of the running instance.
- **Finding lights:** lights are found automatically via Bonjour/mDNS, through the Avahi daemon when it's
  running (much more reliable with lights on Wi-Fi). If multicast is blocked on your network, pass the
  addresses: `bulp-menubar -host 192.168.1.20,192.168.1.21`.

## CLI

```sh
bulp list                    # discover lights
bulp status
bulp on | off | toggle
bulp brightness 40           # or +10 / -10
bulp temp 4500               # Kelvin, 2900–7000
bulp color 200 80            # hue, saturation (color lights)
bulp identify | info | settings | rename "Desk"
```

Flags: `-name <substr>` to pick lights, `-host <ip>` (or `BULP_HOST`) to skip discovery, `-json`.

## Build from source

Go is pinned with [mise](https://mise.jdx.dev). The tray app needs cgo and the GTK 4 + WebKitGTK 6.0
development packages: `sudo dnf install gtk4-devel webkitgtk6.0-devel`
(Debian/Ubuntu: `libgtk-4-dev libwebkitgtk-6.0-dev`).

```sh
mise run build         # bin/bulp, bin/bulp-menubar
mise run build-linux   # static binaries in bin/linux-{amd64,arm64}/ (no cgo: the panel opens in a browser)
mise run test
mise run flatpak       # build + install the Flatpak locally, then: flatpak run de.codecauldron.Bulp
mise run flatpak-bundle  # single-file build/de.codecauldron.Bulp.flatpak
mise run appimage      # build/Bulp-<version>-x86_64.AppImage (needs podman or docker)
mise run release       # all of the above release files + CLI binaries + SHA256SUMS in build/release/
```

### Layout

- `elgato/` – library: Bonjour discovery and an HTTP client for the lights' API (port 9123)
- `cmd/bulp` – CLI
- `cmd/bulp-menubar` – tray app; `internal/ui` serves the control panel (HTML/JS) on 127.0.0.1,
  `internal/webview` shows it in a GTK window
- `packaging/` – desktop entry, AppStream metadata, icon, Flatpak manifest, AppImage build

### Packaging notes

The Flatpak (`packaging/flatpak/de.codecauldron.Bulp.yml`) uses the GNOME 51 runtime, which ships GTK 4 and
WebKitGTK; one-time setup: `flatpak install flathub org.flatpak.Builder`.

The AppImage doesn't bundle GTK or WebKit (~17 MB). It carries two builds of `bulp-menubar`: one linked
against the system's GTK 4 + WebKitGTK 6.0 (built in an Ubuntu 24.04 container so it runs on glibc ≥ 2.34)
and a static one that opens the panel in a browser; `AppRun` starts the first if its libraries load.

## License

[MIT](LICENSE)
