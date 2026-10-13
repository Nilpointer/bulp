---
name: bulp
description: Control Elgato lights (Key Light, Ring Light, Light Strip, …) on the local network with the bulp CLI - switch them on or off, change brightness, color temperature or color, or check their state. Use when the user asks to turn lights on/off, dim, brighten, warm up, cool down or color their lights, or asks what the lights are set to.
allowed-tools: Bash(bulp:*), Bash(flatpak run --command=bulp io.github.Nilpointer.Bulp:*)
---

# Controlling Elgato lights with bulp

## Find the command

Run `bulp` directly; don't probe for it first (users often allow only `bulp` commands). Only if the
shell reports `command not found`, use `flatpak run --command=bulp io.github.Nilpointer.Bulp` in its place
(Bulp installed as a Flatpak). If that fails too, tell the user to install Bulp
(https://github.com/Nilpointer/bulp/releases) and stop.

## Read the state first

For relative requests ("a bit dimmer", "warmer") or when unsure which lights exist, run:

```sh
bulp -json status
```

It prints one array with an entry per light: `name`, `host`, `on`, `brightness` (0–100), and
`kelvin` (white lights) or `hue`/`saturation` (color lights such as the Light Strip).

## Commands

| User says | Run |
|---|---|
| turn on / off / switch | `bulp on`, `bulp off`, `bulp toggle` |
| set to 40 % | `bulp brightness 40` |
| a bit dimmer / brighter | `bulp brightness -10` / `bulp brightness +10` |
| warmer / cozier | `bulp temp 3000` (range 2900–7000 K; lower is warmer) |
| cooler / daylight | `bulp temp 5600` |
| make it blue (color lights only) | `bulp color 220 90` (hue 0–360, saturation 0–100) |
| which light is which | `bulp identify` (the light blinks) |

Every command acts on all lights unless filtered:

- `-name <text>` picks lights whose name contains the text (case-insensitive), e.g.
  `bulp -name desk off`. Use it when the user names a light; check the names with `bulp -json status`.
- Add `-json` to get the resulting state back as the same array as `status`.

## Problems

- `no matching lights found`: retry once with `bulp -discover 5s status`. If that fails too, ask the
  user for the light's IP address and pass it: `bulp -host 192.168.1.20 status` (several:
  comma-separated, or set `BULP_HOST`).
- A line on stderr like `<light>: … connection refused` means that light is unreachable (off at the
  wall, or offline); the other lights were still changed. Exit status is 1 in that case.
- `color` only works on color lights; on white lights use `temp`. Color lights have no `kelvin`, so for
  "warmer" there set an amber hue (`bulp -name strip color 30 70`), for "cooler" a cool blue (`210 40`).

## Finish

Tell the user the new state in one line, taken from the command's output (e.g. "Desk Left: on, 40 %,
3500 K").
