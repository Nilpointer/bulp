package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bulp/elgato"
)

func TestLightJSON(t *testing.T) {
	tg := target{"Desk", elgato.NewClient("192.0.2.1")}
	white := newLightJSON(tg, elgato.LightState{On: elgato.Ptr(0), Brightness: elgato.Ptr(40), Temperature: elgato.Ptr(284)})
	got, _ := json.Marshal(white)
	if want := `{"name":"Desk","host":"192.0.2.1:9123","on":false,"brightness":40,"kelvin":3521}`; string(got) != want {
		t.Errorf("white light:\n got %s\nwant %s", got, want)
	}

	color := newLightJSON(tg, elgato.LightState{On: elgato.Ptr(1), Brightness: elgato.Ptr(0), Hue: elgato.Ptr(190.0), Saturation: elgato.Ptr(95.0)})
	got, _ = json.Marshal(color)
	if want := `{"name":"Desk","host":"192.0.2.1:9123","on":true,"brightness":0,"hue":190,"saturation":95}`; string(got) != want {
		t.Errorf("color light:\n got %s\nwant %s", got, want)
	}
}

func TestNamedHosts(t *testing.T) {
	light := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"productName":"Elgato Key Light","displayName":"Desk Left"}`))
	}))
	defer light.Close()
	up := strings.TrimPrefix(light.URL, "http://")
	down := "127.0.0.1:1" // nothing listens here

	got := named(context.Background(), " "+up+", ,"+down)
	if len(got) != 2 {
		t.Fatalf("got %d targets, want 2", len(got))
	}
	if got[0].label != "Desk Left" || got[0].host() != up {
		t.Errorf("answering light: label %q host %q", got[0].label, got[0].host())
	}
	if got[1].label != down {
		t.Errorf("silent light should keep its address as name, got %q", got[1].label)
	}
}
