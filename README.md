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

Downloads are on the [releases page](https://github.com/Nilpointer/bulp/releases):

| Download | |
|---|---|
| **Flatpak** `io.github.Nilpointer.Bulp.flatpak` | `flatpak install --user io.github.Nilpointer.Bulp.flatpak`, then start Bulp from your app launcher. Pulls the GNOME runtime from Flathub. Includes the CLI: `flatpak run --command=bulp io.github.Nilpointer.Bulp`. |
| **AppImage** `Bulp-<version>-x86_64.AppImage` | `chmod +x` and run. Shows the panel in a native window if GTK 4 + WebKitGTK 6.0 are installed (Ubuntu 24.04+, Debian 12+, current Fedora/Arch), otherwise in your browser. |
| **CLI** `bulp-<os>-<arch>` | Single static binary for Linux, macOS and Windows (amd64/arm64); see [Linux](#cli-on-linux), [macOS](#cli-on-macos) and [Windows](#cli-on-windows) below. |

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

### CLI on Linux

Put the `bulp` binary into a directory that's already on your `PATH`, so your shell, scripts and agents
find it without any `PATH` changes. Download it first (on ARM machines use `bulp-linux-arm64`):

```sh
curl -fLo /tmp/bulp https://github.com/Nilpointer/bulp/releases/latest/download/bulp-linux-amd64
```

- **For all users: `/usr/local/bin`.** It's on the default `PATH` of every distribution and meant for
  software you install yourself:
  ```sh
  sudo install -m 755 /tmp/bulp /usr/local/bin/bulp
  ```
- **Just for you, without sudo: `~/.local/bin`.** Fedora, Ubuntu, Debian, openSUSE and most other
  distributions add it to your `PATH` once it exists (Ubuntu/Debian from the next login on):
  ```sh
  install -Dm 755 /tmp/bulp ~/.local/bin/bulp
  ```
  Check with `command -v bulp`. If that prints nothing even after logging in again, your distribution
  doesn't include `~/.local/bin` (stock Arch, for example); use `/usr/local/bin` instead.

Then run `bulp list`. If you installed the Flatpak, you can skip the download and make its built-in CLI
available as `bulp` with a small wrapper instead:

```sh
mkdir -p ~/.local/bin
printf '#!/bin/sh\nexec flatpak run --command=bulp io.github.Nilpointer.Bulp "$@"\n' > ~/.local/bin/bulp
chmod +x ~/.local/bin/bulp
```

(It starts more slowly than the plain binary, which matters little for occasional commands.)

### CLI on macOS

The tray app is Linux-only (use Elgato's Control Center app on macOS), but the `bulp` CLI runs on macOS,
for scripts and AI agents. Install it to `/usr/local/bin`, which is on the default `PATH`, so shells and
agents find it:

```sh
# Apple Silicon; on an Intel Mac use bulp-darwin-amd64
curl -fLo /tmp/bulp https://github.com/Nilpointer/bulp/releases/latest/download/bulp-darwin-arm64
sudo install -d /usr/local/bin                    # newer Macs may not have the folder yet
sudo install -m 755 /tmp/bulp /usr/local/bin/bulp
bulp list
```

- **Download with `curl`, not a browser.** Browsers mark the file as quarantined and macOS refuses to run
  the unsigned binary. If you did use a browser: `xattr -d com.apple.quarantine /usr/local/bin/bulp`.
- **Allow Local Network access** (macOS 15+) when macOS asks on the first `bulp list`. The prompt is for
  the app running `bulp` (Terminal, iTerm, an IDE or an agent's desktop app), and each needs it once.
- **Set `BULP_HOST`.** macOS has no Avahi, so discovery falls back to plain multicast, which often misses
  lights on Wi-Fi. Fixed addresses make every command reliable, and agents started from your shell
  inherit them: `echo 'export BULP_HOST=192.168.1.20' >> ~/.zshrc` (comma-separate several lights).

To let an agent use it, see [Scripts and AI agents](#scripts-and-ai-agents).

### CLI on Windows

As on macOS, only the CLI runs on Windows (the tray app is Linux-only; use Elgato's Control Center app).
In PowerShell, install it into your user's programs folder and add that to your `PATH`:

```powershell
# x64 PCs; on Windows on Arm use bulp-windows-arm64.exe
$dir = "$env:LOCALAPPDATA\Programs\bulp"
New-Item -ItemType Directory -Force $dir | Out-Null
Invoke-WebRequest https://github.com/Nilpointer/bulp/releases/latest/download/bulp-windows-amd64.exe -OutFile "$dir\bulp.exe"
Unblock-File "$dir\bulp.exe"
[Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", "User") + ";$dir", "User")
```

Open a new terminal (it picks up the new `PATH`), then run `bulp list`.

- **Allow network access** if Windows Defender Firewall asks on the first `bulp list`: discovery listens
  for the lights' multicast replies.
- **Set `BULP_HOST`** for reliable commands, since multicast discovery often misses lights on Wi-Fi.
  New terminals and agents started from them inherit it:
  `[Environment]::SetEnvironmentVariable("BULP_HOST", "192.168.1.20", "User")` (comma-separate several
  lights).
- The binary isn't code-signed. `Unblock-File` stops Windows from treating it as an untrusted download.
  If you downloaded it in a browser and SmartScreen still blocks it, choose **More info → Run anyway**.

To let an agent use it, see [Scripts and AI agents](#scripts-and-ai-agents).

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

### Scripts and AI agents

The `bulp` CLI is the interface for scripts and AI coding agents (Claude Code, Codex, …): it's
non-interactive, exits with status 0 on success and 1 on errors (message on stderr), and with `-json`
every command prints one array with an entry per light:

```sh
$ bulp -json status
[{"name": "Desk Left", "host": "192.168.1.20:9123", "on": true, "brightness": 40, "kelvin": 3500},
 {"name": "Shelf Strip", "host": "192.168.1.21:9123", "on": true, "brightness": 85, "hue": 265, "saturation": 80}]
```

- **Skip discovery:** `export BULP_HOST=192.168.1.20,192.168.1.21` (addresses from `bulp list`). Commands
  then don't browse the network first, and always hit the same lights.
- **Pick lights by name:** `-name desk` acts only on lights whose name contains "desk", with or without
  `BULP_HOST`.
- **Allow it without prompts:** the skill below does this for you while it's active. Without the skill,
  add `"Bash(bulp:*)"` to `permissions.allow` in `.claude/settings.json` (or `~/.claude/settings.json`).

What an agent runs for typical requests (the skill below teaches it the same mappings):

| User says | Command |
|---|---|
| turn on / off / switch | `bulp on`, `bulp off`, `bulp toggle` |
| set to 40 % | `bulp brightness 40` |
| a bit dimmer / brighter | `bulp brightness -10` / `bulp brightness +10` |
| warmer / cozier | `bulp temp 3000` (range 2900–7000 K; lower is warmer) |
| cooler / daylight | `bulp temp 5600` |
| make it blue (color lights only) | `bulp color 220 90` (hue 0–360, saturation 0–100) |
| warmer, on a color light | `bulp -name strip color 30 70` (amber; color lights have no Kelvin) |
| only the desk light | `bulp -name desk off` |
| which light is which | `bulp identify` (the light blinks) |

**Agent skill:** [`skills/bulp/SKILL.md`](skills/bulp/SKILL.md) teaches an agent when and how to use
`bulp` (finding the command, reading the state first, mapping "a bit warmer" to a command, handling
unreachable lights), so it works from any directory.

#### Claude Code plugin (recommended)

This repo is also a Claude Code plugin marketplace, so the skill installs and updates like any plugin.
You still need the `bulp` command itself ([Linux](#cli-on-linux), [macOS](#cli-on-macos),
[Windows](#cli-on-windows); on Linux the skill falls back to the Flatpak's CLI if `bulp` isn't on
your `PATH`).

**Install**, either inside Claude Code or from a terminal:

```text
/plugin install bulp --marketplace Nilpointer/bulp
```
```sh
claude plugin install bulp --marketplace Nilpointer/bulp
```

On Claude Code before 2.1.275, add the marketplace first: `claude plugin marketplace add Nilpointer/bulp`,
then `claude plugin install bulp@bulps`. The plugin is installed for your user in all projects. With
`--scope project` it's recorded in the project's `.claude/settings.json` instead, so everyone working on
that repository gets it.

**Use:** just ask, in any directory: "dim the lights to 40%", "make it warmer", "turn off the desk light",
"what are my lights set to?". Claude picks the skill up by itself. You can also call it directly as a
slash command, e.g. `/bulp:bulp status`. While the skill runs, `bulp` commands need no permission
prompts; the skill allows them itself.

**Manage it:**

| Task | Command |
|---|---|
| show what's installed | `claude plugin list`, or `/plugin` inside Claude Code |
| see what the plugin adds and its token cost | `claude plugin details bulp@bulps` |
| update to a new version | `claude plugin marketplace update bulps`, then `claude plugin update bulp@bulps`, then restart Claude Code |
| turn it off for a while | `claude plugin disable bulp@bulps` (back on: `claude plugin enable bulp@bulps`) |
| remove it | `claude plugin uninstall bulp@bulps`, and optionally `claude plugin marketplace remove bulps` |

If you copied the skill by hand earlier, delete that copy (`rm -r ~/.claude/skills/bulp`), or Claude sees
the skill twice (`bulp` and `bulp:bulp`).

#### Skill for other agents

For other agents that support the Agent Skills `SKILL.md` format, or to manage the skill by hand in
Claude Code, copy the folder into the agent's skills directory:

```sh
mkdir -p ~/.claude/skills && cp -r skills/bulp ~/.claude/skills/
# or without a clone of this repo:
mkdir -p ~/.claude/skills/bulp && curl -fLo ~/.claude/skills/bulp/SKILL.md https://raw.githubusercontent.com/Nilpointer/bulp/main/skills/bulp/SKILL.md
```

### Automating lights out

For a fixed schedule, cron is all you need. Use the standalone CLI binary from the releases page (e.g.
installed as `/usr/local/bin/bulp`), set `PATH` (cron's default doesn't include `/usr/local/bin`) and
`BULP_HOST`, so it doesn't depend on network discovery at night. `crontab -e`:

```cron
PATH=/usr/local/bin:/usr/bin:/bin
BULP_HOST=192.168.1.20,192.168.1.21
# Lights off at 23:00 every day, and at 18:30 on weekdays
0 23 * * *    bulp off
30 18 * * 1-5 bulp off
# Dim to 20 % and warm up at 21:00
0 21 * * *    bulp brightness 20 && bulp temp 2900
```

A systemd user timer works the same way (`bulp off` as the service's `ExecStart`) and also catches up
on runs missed while the computer was off (`Persistent=true`).

Reach for an agent only when the rule needs judgment or context a schedule can't express, like "lights
off, but keep the desk light on while I'm still in a video call". Run it headless from cron and allow it
nothing but `bulp` and the checks it needs (the `PATH` and `BULP_HOST` lines from above apply to the
agent's commands too):

```cron
0 23 * * * cd ~ && /home/you/.local/bin/claude -p "Turn off all lights. If Zoom or Teams is running, keep the desk light on." --allowedTools "Bash(bulp:*)" "Bash(pgrep:*)"
```

Whatever runs the schedule has to be on the same network as the lights: cloud-hosted agents and
scheduled cloud tasks can't reach them.

## Build from source

Go is pinned with [mise](https://mise.jdx.dev). The tray app needs cgo and the GTK 4 + WebKitGTK 6.0
development packages: `sudo dnf install gtk4-devel webkitgtk6.0-devel`
(Debian/Ubuntu: `libgtk-4-dev libwebkitgtk-6.0-dev`).

```sh
mise run build         # bin/bulp, bin/bulp-menubar
mise run build-linux   # static binaries in bin/linux-{amd64,arm64}/ (no cgo: the panel opens in a browser)
mise run test
mise run flatpak       # build + install the Flatpak locally, then: flatpak run io.github.Nilpointer.Bulp
mise run flatpak-bundle  # single-file build/io.github.Nilpointer.Bulp.flatpak
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

The Flatpak (`packaging/flatpak/io.github.Nilpointer.Bulp.yml`) uses the GNOME 51 runtime, which ships GTK 4 and
WebKitGTK; one-time setup: `flatpak install flathub org.flatpak.Builder`.

The AppImage doesn't bundle GTK or WebKit (~17 MB). It carries two builds of `bulp-menubar`: one linked
against the system's GTK 4 + WebKitGTK 6.0 (built in an Ubuntu 24.04 container so it runs on glibc ≥ 2.34)
and a static one that opens the panel in a browser; `AppRun` starts the first if its libraries load.

## License

[MIT](LICENSE)
