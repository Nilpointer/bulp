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
	"sync"
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

// host is the light's address as host:port.
func (t target) host() string { return strings.TrimPrefix(t.client.BaseURL, "http://") }

// With -json every command prints one array with an entry per light, so
// scripts and agents can tell the lights apart.

// lightJSON is the entry for commands that report a light's state; the
// color temperature is in Kelvin, like the temp command takes it.
type lightJSON struct {
	Name       string   `json:"name"`
	Host       string   `json:"host"`
	On         *bool    `json:"on,omitempty"`
	Brightness *int     `json:"brightness,omitempty"`
	Kelvin     *int     `json:"kelvin,omitempty"`
	Hue        *float64 `json:"hue,omitempty"`
	Saturation *float64 `json:"saturation,omitempty"`
}

func newLightJSON(t target, s elgato.LightState) lightJSON {
	j := lightJSON{Name: t.label, Host: t.host(), Brightness: s.Brightness, Hue: s.Hue, Saturation: s.Saturation}
	if s.On != nil {
		j.On = elgato.Ptr(*s.On == 1)
	}
	if s.Temperature != nil {
		j.Kelvin = elgato.Ptr(elgato.MiredToKelvin(*s.Temperature))
	}
	return j
}

// detailJSON is the entry for everything else; commands without output
// (identify, rename, changing settings) list just the lights they acted on.
type detailJSON struct {
	Name     string                `json:"name"`
	Host     string                `json:"host"`
	Info     *elgato.AccessoryInfo `json:"info,omitempty"`
	Battery  *elgato.BatteryInfo   `json:"battery,omitempty"`
	Settings *elgato.Settings      `json:"settings,omitempty"`
}

func main() {
	fs := flag.NewFlagSet("bulp", flag.ExitOnError)
	host := fs.String("host", os.Getenv("BULP_HOST"), "talk to these lights directly (host[:port], comma-separated), skipping discovery; env BULP_HOST")
	name := fs.String("name", "", "only act on lights whose name contains this text")
	wait := fs.Duration("discover", 2*time.Second, "how long to browse for lights")
	asJSON := fs.Bool("json", false, "print JSON: one array with an entry per light")
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
	results := []any{} // printed as [] rather than null if every light failed
	for _, t := range targets {
		res, err := run(ctx, t, cmd, args, *asJSON)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", t.label, err)
			failed = true
			continue
		}
		results = append(results, res)
	}
	if *asJSON {
		check(printJSON(results))
	}
	if failed {
		os.Exit(1)
	}
}

func resolve(ctx context.Context, hosts, name string, wait time.Duration) ([]target, error) {
	var all []target
	if hosts != "" {
		all = named(ctx, hosts)
	} else {
		dctx, cancel := context.WithTimeout(ctx, wait)
		defer cancel()
		devices, err := elgato.Discover(dctx)
		if err != nil {
			return nil, err
		}
		for _, d := range devices {
			all = append(all, target{d.Instance, d.Client()})
		}
	}
	var out []target
	for _, t := range all {
		if name == "" || strings.Contains(strings.ToLower(t.label), strings.ToLower(name)) {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no matching lights found (try -discover 5s or -host)")
	}
	return out, nil
}

// named makes targets for comma-separated addresses, named like discovery
// would name them (asking the lights in parallel); a light that doesn't
// answer keeps its address as name, and the command reports its error.
func named(ctx context.Context, hosts string) []target {
	var out []target
	for h := range strings.SplitSeq(hosts, ",") {
		if h = strings.TrimSpace(h); h != "" {
			out = append(out, target{h, elgato.NewClient(h)})
		}
	}
	var wg sync.WaitGroup
	for i := range out {
		wg.Go(func() {
			ictx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			if info, err := out[i].client.AccessoryInfo(ictx); err == nil && info.Name() != "" {
				out[i].label = info.Name()
			}
		})
	}
	wg.Wait()
	return out
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

// run executes cmd on one light. It prints text output itself and returns
// the light's entry for -json.
func run(ctx context.Context, t target, cmd string, args []string, asJSON bool) (any, error) {
	c := t.client
	done := detailJSON{Name: t.label, Host: t.host()}
	switch cmd {
	case "status":
		l, err := c.Lights(ctx)
		if err != nil {
			return nil, err
		}
		return report(t, l, asJSON)

	case "on", "off":
		return set(ctx, t, elgato.LightState{On: elgato.Ptr(boolInt(cmd == "on"))}, asJSON)

	case "toggle":
		l, err := c.Lights(ctx)
		if err != nil {
			return nil, err
		}
		on := len(l.Lights) > 0 && l.Lights[0].On != nil && *l.Lights[0].On == 1
		return set(ctx, t, elgato.LightState{On: elgato.Ptr(boolInt(!on))}, asJSON)

	case "brightness":
		if len(args) != 1 {
			return nil, errors.New("usage: brightness <0-100|+n|-n>")
		}
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return nil, fmt.Errorf("invalid brightness %q", args[0])
		}
		if args[0][0] == '+' || args[0][0] == '-' {
			l, err := c.Lights(ctx)
			if err != nil {
				return nil, err
			}
			if len(l.Lights) > 0 && l.Lights[0].Brightness != nil {
				n += *l.Lights[0].Brightness
			}
		}
		return set(ctx, t, elgato.LightState{Brightness: elgato.Ptr(max(0, min(n, 100)))}, asJSON)

	case "temp":
		if len(args) != 1 {
			return nil, errors.New("usage: temp <kelvin>")
		}
		k, err := strconv.Atoi(strings.TrimSuffix(strings.ToUpper(args[0]), "K"))
		if err != nil {
			return nil, fmt.Errorf("invalid temperature %q", args[0])
		}
		return set(ctx, t, elgato.LightState{Temperature: elgato.Ptr(elgato.KelvinToMired(k))}, asJSON)

	case "color":
		if len(args) != 2 {
			return nil, errors.New("usage: color <hue 0-360> <saturation 0-100>")
		}
		h, err1 := strconv.ParseFloat(args[0], 64)
		s, err2 := strconv.ParseFloat(args[1], 64)
		if err1 != nil || err2 != nil {
			return nil, errors.New("hue and saturation must be numbers")
		}
		return set(ctx, t, elgato.LightState{Hue: &h, Saturation: &s}, asJSON)

	case "identify":
		return done, c.Identify(ctx)

	case "info":
		a, err := c.AccessoryInfo(ctx)
		if err != nil {
			return nil, err
		}
		done.Info = a
		if asJSON {
			return done, nil
		}
		fmt.Printf("%s\n  product   %s (board %d, rev %g)\n  serial    %s\n  mac       %s\n  firmware  %s (build %d)\n  features  %s\n",
			t.label, a.ProductName, a.HardwareBoardType, a.HardwareRevision, a.SerialNumber,
			a.MACAddress, a.FirmwareVersion, a.FirmwareBuildNumber, strings.Join(a.Features, ", "))
		if a.WiFi != nil {
			fmt.Printf("  wifi      %s, %d MHz, %d dBm\n", a.WiFi.SSID, a.WiFi.FrequencyMHz, a.WiFi.RSSI)
		}
		return done, nil

	case "battery":
		b, err := c.BatteryInfo(ctx)
		if err != nil {
			return nil, err
		}
		done.Battery = b
		if !asJSON {
			fmt.Printf("%-28s level %g%%, status %g, power source %g, %gV\n",
				t.label, b.Level, b.Status, b.PowerSource, b.CurrentBatteryVoltage)
		}
		return done, nil

	case "settings":
		if len(args) == 0 {
			s, err := c.Settings(ctx)
			if err != nil {
				return nil, err
			}
			done.Settings = s
			if !asJSON {
				fmt.Printf("%s\n", t.label)
				return done, printJSON(s)
			}
			return done, nil
		}
		s, err := parseSettings(args)
		if err != nil {
			return nil, err
		}
		return done, c.SetSettings(ctx, s)

	case "rename":
		if len(args) != 1 {
			return nil, errors.New("usage: rename <name>")
		}
		return done, c.SetDisplayName(ctx, args[0])
	}
	return nil, fmt.Errorf("unknown command %q (run bulp -h)", cmd)
}

func set(ctx context.Context, t target, s elgato.LightState, asJSON bool) (any, error) {
	l, err := t.client.SetLights(ctx, s)
	if err != nil {
		return nil, err
	}
	return report(t, l, asJSON)
}

// report prints a light's state as text, or returns its -json entry.
func report(t target, l *elgato.Lights, asJSON bool) (any, error) {
	if len(l.Lights) == 0 {
		return nil, errors.New("light returned no state")
	}
	if !asJSON {
		fmt.Printf("%-28s %s\n", t.label, describe(l.Lights[0]))
	}
	return newLightJSON(t, l.Lights[0]), nil
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
