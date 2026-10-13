// Command bulp controls Elgato lights on the local network.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"bulp/elgato"
)

const usage = `usage: bulp [flags] <command> [args]

commands:
  list                       discover lights on the network
  status                     show on/off, brightness and color of each light
  on | off | toggle          switch lights
  brightness <n|+n|-n>       set brightness 0–100, or adjust relative to current
  temp <kelvin>              set white color temperature (2900–7000)
  color <hue> <saturation>   set color on color lights (hue 0–360, sat 0–100)
  identify                   blink the light
  info                       show accessory info (model, firmware, Wi-Fi)
  battery                    show battery info (battery-powered lights only)
  settings [key=value ...]   show or change power-on/fade settings
  rename <name>              set the light's display name

flags:
`

type target struct {
	label  string
	client *elgato.Client
}

func main() {
	fs := flag.NewFlagSet("bulp", flag.ExitOnError)
	host := fs.String("host", os.Getenv("BULP_HOST"), "talk to this light directly (host[:port]), skipping discovery; env BULP_HOST")
	name := fs.String("name", "", "only act on lights whose name contains this text")
	wait := fs.Duration("discover", 2*time.Second, "how long to browse for lights")
	asJSON := fs.Bool("json", false, "print raw JSON")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage); fs.PrintDefaults() }
	fs.Parse(os.Args[1:])

	if fs.NArg() == 0 {
		fs.Usage()
		os.Exit(2)
	}
	cmd, args := fs.Arg(0), fs.Args()[1:]
	ctx := context.Background()

	if cmd == "list" {
		check(list(ctx, *wait, *asJSON))
		return
	}

	targets, err := resolve(ctx, *host, *name, *wait)
	check(err)

	failed := false
	for _, t := range targets {
		if err := run(ctx, t, cmd, args, *asJSON); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", t.label, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func resolve(ctx context.Context, host, name string, wait time.Duration) ([]target, error) {
	if host != "" {
		return []target{{host, elgato.NewClient(host)}}, nil
	}
	dctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	devices, err := elgato.Discover(dctx)
	if err != nil {
		return nil, err
	}
	var out []target
	for _, d := range devices {
		if name != "" && !strings.Contains(strings.ToLower(d.Instance), strings.ToLower(name)) {
			continue
		}
		out = append(out, target{d.Instance, d.Client()})
	}
	if len(out) == 0 {
		return nil, errors.New("no matching lights found (try -discover 5s or -host)")
	}
	return out, nil
}

func list(ctx context.Context, wait time.Duration, asJSON bool) error {
	dctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	devices, err := elgato.Discover(dctx)
	if err != nil {
		return err
	}
	if asJSON {
		return printJSON(devices)
	}
	for _, d := range devices {
		fmt.Printf("%-28s %-22s %s\n", d.Instance, d.Host, d.Model)
	}
	if len(devices) == 0 {
		fmt.Fprintln(os.Stderr, "no lights found")
	}
	return nil
}

func run(ctx context.Context, t target, cmd string, args []string, asJSON bool) error {
	c := t.client
	switch cmd {
	case "status":
		l, err := c.Lights(ctx)
		if err != nil {
			return err
		}
		if asJSON {
			return printJSON(l)
		}
		for _, s := range l.Lights {
			fmt.Printf("%-28s %s\n", t.label, describe(s))
		}
		return nil

	case "on", "off":
		return set(ctx, t, elgato.LightState{On: elgato.Ptr(boolInt(cmd == "on"))}, asJSON)

	case "toggle":
		l, err := c.Lights(ctx)
		if err != nil {
			return err
		}
		on := len(l.Lights) > 0 && l.Lights[0].On != nil && *l.Lights[0].On == 1
		return set(ctx, t, elgato.LightState{On: elgato.Ptr(boolInt(!on))}, asJSON)

	case "brightness":
		if len(args) != 1 {
			return errors.New("usage: brightness <0-100|+n|-n>")
		}
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid brightness %q", args[0])
		}
		if args[0][0] == '+' || args[0][0] == '-' {
			l, err := c.Lights(ctx)
			if err != nil {
				return err
			}
			if len(l.Lights) > 0 && l.Lights[0].Brightness != nil {
				n += *l.Lights[0].Brightness
			}
		}
		return set(ctx, t, elgato.LightState{Brightness: elgato.Ptr(max(0, min(n, 100)))}, asJSON)

	case "temp":
		if len(args) != 1 {
			return errors.New("usage: temp <kelvin>")
		}
		k, err := strconv.Atoi(strings.TrimSuffix(strings.ToUpper(args[0]), "K"))
		if err != nil {
			return fmt.Errorf("invalid temperature %q", args[0])
		}
		return set(ctx, t, elgato.LightState{Temperature: elgato.Ptr(elgato.KelvinToMired(k))}, asJSON)

	case "color":
		if len(args) != 2 {
			return errors.New("usage: color <hue 0-360> <saturation 0-100>")
		}
		h, err1 := strconv.ParseFloat(args[0], 64)
		s, err2 := strconv.ParseFloat(args[1], 64)
		if err1 != nil || err2 != nil {
			return errors.New("hue and saturation must be numbers")
		}
		return set(ctx, t, elgato.LightState{Hue: &h, Saturation: &s}, asJSON)

	case "identify":
		return c.Identify(ctx)

	case "info":
		a, err := c.AccessoryInfo(ctx)
		if err != nil {
			return err
		}
		if asJSON {
			return printJSON(a)
		}
		fmt.Printf("%s\n  product   %s (board %d, rev %g)\n  serial    %s\n  mac       %s\n  firmware  %s (build %d)\n  features  %s\n",
			t.label, a.ProductName, a.HardwareBoardType, a.HardwareRevision, a.SerialNumber,
			a.MACAddress, a.FirmwareVersion, a.FirmwareBuildNumber, strings.Join(a.Features, ", "))
		if a.WiFi != nil {
			fmt.Printf("  wifi      %s, %d MHz, %d dBm\n", a.WiFi.SSID, a.WiFi.FrequencyMHz, a.WiFi.RSSI)
		}
		return nil

	case "battery":
		b, err := c.BatteryInfo(ctx)
		if err != nil {
			return err
		}
		if asJSON {
			return printJSON(b)
		}
		fmt.Printf("%-28s level %g%%, status %g, power source %g, %gV\n",
			t.label, b.Level, b.Status, b.PowerSource, b.CurrentBatteryVoltage)
		return nil

	case "settings":
		if len(args) == 0 {
			s, err := c.Settings(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", t.label)
			return printJSON(s)
		}
		s, err := parseSettings(args)
		if err != nil {
			return err
		}
		return c.SetSettings(ctx, s)

	case "rename":
		if len(args) != 1 {
			return errors.New("usage: rename <name>")
		}
		return c.SetDisplayName(ctx, args[0])
	}
	return fmt.Errorf("unknown command %q (run bulp -h)", cmd)
}

func set(ctx context.Context, t target, s elgato.LightState, asJSON bool) error {
	l, err := t.client.SetLights(ctx, s)
	if err != nil {
		return err
	}
	if asJSON {
		return printJSON(l)
	}
	for _, s := range l.Lights {
		fmt.Printf("%-28s %s\n", t.label, describe(s))
	}
	return nil
}

func describe(s elgato.LightState) string {
	var parts []string
	if s.On != nil {
		parts = append(parts, map[bool]string{true: "on ", false: "off"}[*s.On == 1])
	}
	if s.Brightness != nil {
		parts = append(parts, fmt.Sprintf("%3d%%", *s.Brightness))
	}
	if s.Temperature != nil {
		parts = append(parts, fmt.Sprintf("%dK", elgato.MiredToKelvin(*s.Temperature)))
	}
	if s.Hue != nil && s.Saturation != nil {
		parts = append(parts, fmt.Sprintf("hue %g sat %g", *s.Hue, *s.Saturation))
	}
	return strings.Join(parts, "  ")
}

var settingKeys = map[string]func(*elgato.Settings, int){
	"powerOnBehavior":       func(s *elgato.Settings, v int) { s.PowerOnBehavior = &v },
	"powerOnBrightness":     func(s *elgato.Settings, v int) { s.PowerOnBrightness = &v },
	"powerOnTemperature":    func(s *elgato.Settings, v int) { s.PowerOnTemperature = &v },
	"switchOnDurationMs":    func(s *elgato.Settings, v int) { s.SwitchOnDurationMs = &v },
	"switchOffDurationMs":   func(s *elgato.Settings, v int) { s.SwitchOffDurationMs = &v },
	"colorChangeDurationMs": func(s *elgato.Settings, v int) { s.ColorChangeDurationMs = &v },
}

func parseSettings(args []string) (elgato.Settings, error) {
	var s elgato.Settings
	for _, a := range args {
		k, v, ok := strings.Cut(a, "=")
		apply, known := settingKeys[k]
		if !ok || !known {
			return s, fmt.Errorf("bad setting %q; keys: powerOnBehavior, powerOnBrightness, powerOnTemperature, switchOnDurationMs, switchOffDurationMs, colorChangeDurationMs", a)
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return s, fmt.Errorf("setting %s: %q is not a number", k, v)
		}
		apply(&s, n)
	}
	return s, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "bulp:", err)
		os.Exit(1)
	}
}
