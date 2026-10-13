// Package elgato talks to Elgato lights (Key Light, Ring Light, Light Strip, …)
// over their local HTTP API on port 9123.
package elgato

// LightState is one light's state as reported by /elgato/lights.
// Fields are pointers so a PUT only sends what the caller set.
// White lights use Temperature; color lights (Light Strip) use Hue/Saturation.
type LightState struct {
	On          *int     `json:"on,omitempty"`          // 0 or 1
	Brightness  *int     `json:"brightness,omitempty"`  // 0–100
	Temperature *int     `json:"temperature,omitempty"` // mireds, 143–344
	Hue         *float64 `json:"hue,omitempty"`         // 0–360
	Saturation  *float64 `json:"saturation,omitempty"`  // 0–100
}

type Lights struct {
	NumberOfLights int          `json:"numberOfLights"`
	Lights         []LightState `json:"lights"`
}

// Settings is /elgato/lights/settings. Pointers again, for partial updates.
type Settings struct {
	PowerOnBehavior       *int `json:"powerOnBehavior,omitempty"` // 1 = restore last, 2 = use powerOn* values
	PowerOnBrightness     *int `json:"powerOnBrightness,omitempty"`
	PowerOnTemperature    *int `json:"powerOnTemperature,omitempty"`
	SwitchOnDurationMs    *int `json:"switchOnDurationMs,omitempty"`
	SwitchOffDurationMs   *int `json:"switchOffDurationMs,omitempty"`
	ColorChangeDurationMs *int `json:"colorChangeDurationMs,omitempty"`
}

type WiFiInfo struct {
	SSID         string `json:"ssid"`
	FrequencyMHz int    `json:"frequencyMHz"`
	RSSI         int    `json:"rssi"`
}

type AccessoryInfo struct {
	ProductName         string    `json:"productName"`
	HardwareBoardType   int       `json:"hardwareBoardType"`
	HardwareRevision    float64   `json:"hardwareRevision"`
	MACAddress          string    `json:"macAddress"`
	FirmwareBuildNumber int       `json:"firmwareBuildNumber"`
	FirmwareVersion     string    `json:"firmwareVersion"`
	SerialNumber        string    `json:"serialNumber"`
	DisplayName         string    `json:"displayName"`
	Features            []string  `json:"features"`
	WiFi                *WiFiInfo `json:"wifi-info,omitempty"`
}

// Name returns the user-set display name, falling back to the product name.
func (a AccessoryInfo) Name() string {
	if a.DisplayName != "" {
		return a.DisplayName
	}
	return a.ProductName
}

// BatteryInfo is /elgato/battery-info (battery-powered lights such as Key Light Mini).
// Field names come from the Control Center binary; units are as reported by the device.
type BatteryInfo struct {
	PowerSource           float64 `json:"powerSource"`
	Level                 float64 `json:"level"`
	Status                float64 `json:"status"`
	CurrentBatteryVoltage float64 `json:"currentBatteryVoltage"`
	InputChargeVoltage    float64 `json:"inputChargeVoltage"`
	InputChargeCurrent    float64 `json:"inputChargeCurrent"`
}

// Temperature range supported by Elgato white lights, in mireds.
const (
	MinMired = 143 // ≈ 7000 K
	MaxMired = 344 // ≈ 2900 K
)

// KelvinToMired converts a color temperature in Kelvin to the device's
// mired value, clamped to the supported range.
func KelvinToMired(k int) int {
	if k <= 0 {
		return MaxMired
	}
	return clamp((1_000_000+k/2)/k, MinMired, MaxMired)
}

// MiredToKelvin converts a device mired value to Kelvin.
func MiredToKelvin(m int) int {
	if m <= 0 {
		return 0
	}
	return (1_000_000 + m/2) / m
}

func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}

// Ptr returns a pointer to v, handy for building partial updates.
func Ptr[T any](v T) *T { return &v }
