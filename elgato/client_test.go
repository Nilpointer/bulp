package elgato

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeLight mimics a Ring Light's HTTP API, using responses captured from a real device.
func fakeLight(t *testing.T) (*Client, *Lights) {
	state := &Lights{NumberOfLights: 1, Lights: []LightState{{On: Ptr(0), Brightness: Ptr(11), Temperature: Ptr(344)}}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /elgato/lights", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(state)
	})
	mux.HandleFunc("PUT /elgato/lights", func(w http.ResponseWriter, r *http.Request) {
		var in Lights
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Lights) != 1 {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		cur, upd := &state.Lights[0], in.Lights[0]
		if upd.On != nil {
			cur.On = upd.On
		}
		if upd.Brightness != nil {
			cur.Brightness = upd.Brightness
		}
		if upd.Temperature != nil {
			cur.Temperature = upd.Temperature
		}
		json.NewEncoder(w).Encode(state)
	})
	mux.HandleFunc("GET /elgato/accessory-info", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"productName":"Elgato Ring Light","hardwareBoardType":201,"hardwareRevision":0.2,"macAddress":"00:11:22:33:44:55","firmwareBuildNumber":151,"firmwareVersion":"1.0.4","serialNumber":"X","displayName":"","features":["lights"],"wifi-info":{"ssid":"net","frequencyMHz":2400,"rssi":-64}}`))
	})
	mux.HandleFunc("POST /elgato/identify", func(w http.ResponseWriter, r *http.Request) {})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return NewClient(strings.TrimPrefix(srv.URL, "http://")), state
}

func TestPartialUpdateKeepsOtherFields(t *testing.T) {
	c, state := fakeLight(t)
	ctx := context.Background()

	got, err := c.SetLights(ctx, LightState{On: Ptr(1)})
	if err != nil {
		t.Fatal(err)
	}
	l := got.Lights[0]
	if *l.On != 1 || *l.Brightness != 11 || *l.Temperature != 344 {
		t.Fatalf("unexpected state after turning on: %+v", l)
	}
	if _, err := c.SetLights(ctx, LightState{On: Ptr(0)}); err != nil {
		t.Fatal(err)
	}
	if *state.Lights[0].On != 0 {
		t.Fatal("off was not sent (zero value dropped?)")
	}
}

func TestAccessoryInfo(t *testing.T) {
	c, _ := fakeLight(t)
	a, err := c.AccessoryInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if a.Name() != "Elgato Ring Light" || a.WiFi == nil || a.WiFi.RSSI != -64 {
		t.Fatalf("bad decode: %+v", a)
	}
}

func TestNotSupported(t *testing.T) {
	c, _ := fakeLight(t)
	if _, err := c.BatteryInfo(context.Background()); !errors.Is(err, ErrNotSupported) {
		t.Fatalf("want ErrNotSupported, got %v", err)
	}
	if err := c.Identify(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestKelvinMired(t *testing.T) {
	for k, want := range map[int]int{2900: 344, 7000: 143, 4000: 250, 1000: 344, 10000: 143} {
		if got := KelvinToMired(k); got != want {
			t.Errorf("KelvinToMired(%d) = %d, want %d", k, got, want)
		}
	}
	if got := MiredToKelvin(250); got != 4000 {
		t.Errorf("MiredToKelvin(250) = %d", got)
	}
}
