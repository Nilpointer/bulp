// Package ui serves the menu-bar popover's web UI and a small JSON API
// on 127.0.0.1. Every route sits under a random per-launch token so other
// local processes and web pages can't drive the lights through it.
package ui

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"elg/elgato"
)

//go:embed web
var webFS embed.FS

// Light is what the UI sees for one device.
type Light struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Product  string             `json:"product,omitempty"`
	Host     string             `json:"host"`
	Firmware string             `json:"firmware,omitempty"`
	Serial   string             `json:"serial,omitempty"`
	RSSI     *int               `json:"rssi,omitempty"`
	State    *elgato.LightState `json:"state,omitempty"`
	Error    string             `json:"error,omitempty"`
}

type device struct {
	id, instance, host string
	client             *elgato.Client
	info               *elgato.AccessoryInfo
}

type Server struct {
	// OnChange is called with the current lights after every refresh.
	OnChange func(lights []Light)
	// Quit is called when the UI asks the app to exit.
	Quit func()

	fixedHosts []string
	discover   time.Duration

	scanned chan struct{} // closed after the first discovery pass

	mu      sync.Mutex
	devices map[string]*device
}

// New creates a server. If hosts is non-empty those lights are used and
// Bonjour discovery is skipped.
func New(hosts []string, discoverFor time.Duration) *Server {
	s := &Server{fixedHosts: hosts, discover: discoverFor, devices: map[string]*device{}, scanned: make(chan struct{})}
	for _, h := range hosts {
		s.devices[h] = &device{id: h, instance: h, host: h, client: elgato.NewClient(h)}
	}
	return s
}

// Start listens on a random localhost port, begins background discovery,
// and returns the URL the web view should load.
func (s *Server) Start(ctx context.Context) (string, error) {
	tok := make([]byte, 16)
	rand.Read(tok)
	prefix := "/" + hex.EncodeToString(tok)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(must(fs.Sub(webFS, "web"))))
	mux.HandleFunc("GET /api/lights", s.handleList)
	mux.HandleFunc("PUT /api/lights/{id}", s.handleSet)
	mux.HandleFunc("POST /api/lights/{id}/identify", s.handleIdentify)
	mux.HandleFunc("POST /api/rescan", s.handleRescan)
	mux.HandleFunc("POST /api/quit", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		if s.Quit != nil {
			go s.Quit()
		}
	})

	srv := &http.Server{
		Handler: http.StripPrefix(prefix, mux),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	go srv.Serve(ln)
	go s.discoverLoop(ctx)
	return "http://" + ln.Addr().String() + prefix + "/", nil
}

func (s *Server) discoverLoop(ctx context.Context) {
	if len(s.fixedHosts) > 0 {
		close(s.scanned)
		return
	}
	first := true
	for {
		s.rescan(ctx)
		if first {
			close(s.scanned)
			first = false
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(60 * time.Second):
		}
	}
}

func (s *Server) rescan(ctx context.Context) {
	if len(s.fixedHosts) > 0 {
		return
	}
	dctx, cancel := context.WithTimeout(ctx, s.discover)
	defer cancel()
	found, err := elgato.Discover(dctx)
	if err != nil {
		log.Printf("discovery: %v", err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range found {
		id := d.ID
		if id == "" {
			id = d.Instance
		}
		if cur, ok := s.devices[id]; ok && cur.host == d.Host {
			continue
		}
		s.devices[id] = &device{id: id, instance: d.Instance, host: d.Host, client: d.Client()}
	}
}

func (s *Server) snapshot() []*device {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*device, 0, len(s.devices))
	for _, d := range s.devices {
		out = append(out, d)
	}
	return out
}

func (s *Server) lookup(id string) *device {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.devices[id]
}

// Lights reads current state (and, once, accessory info) for every device in parallel.
func (s *Server) Lights(ctx context.Context) []Light {
	devs := s.snapshot()
	lights := make([]Light, len(devs))
	var wg sync.WaitGroup
	for i, d := range devs {
		wg.Go(func() {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			lights[i] = s.describe(ctx, d)
		})
	}
	wg.Wait()
	slices.SortFunc(lights, func(a, b Light) int { return strings.Compare(a.Name, b.Name) })
	s.notify(lights)
	return lights
}

func (s *Server) describe(ctx context.Context, d *device) Light {
	l := Light{ID: d.id, Name: d.instance, Host: d.host}
	if info, err := d.client.AccessoryInfo(ctx); err == nil {
		s.mu.Lock()
		d.info = info
		s.mu.Unlock()
	}
	s.mu.Lock()
	info := d.info
	s.mu.Unlock()
	if info != nil {
		switch {
		case info.DisplayName != "":
			l.Name = info.DisplayName
		case d.instance == d.host: // added via -host: no Bonjour name to show
			l.Name = info.ProductName
		}
		l.Product, l.Firmware, l.Serial = info.ProductName, info.FirmwareVersion, info.SerialNumber
		if info.WiFi != nil {
			l.RSSI = &info.WiFi.RSSI
		}
	}
	st, err := d.client.Lights(ctx)
	switch {
	case err != nil:
		l.Error = err.Error()
	case len(st.Lights) > 0:
		l.State = &st.Lights[0]
	}
	return l
}

func (s *Server) notify(lights []Light) {
	if s.OnChange != nil {
		s.OnChange(lights)
	}
}

// AnyOn reports whether at least one light is switched on.
func AnyOn(lights []Light) bool {
	return slices.ContainsFunc(lights, Light.IsOn)
}

func (l Light) IsOn() bool {
	return l.State != nil && l.State.On != nil && *l.State.On == 1
}

// SetOn switches one light on or off.
func (s *Server) SetOn(ctx context.Context, id string, on bool) error {
	d := s.lookup(id)
	if d == nil {
		return errors.New("unknown light")
	}
	v := 0
	if on {
		v = 1
	}
	_, err := d.client.SetLights(ctx, elgato.LightState{On: &v})
	return err
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	// Don't report "no lights" before the first scan has had a chance to find any.
	select {
	case <-s.scanned:
	case <-r.Context().Done():
		return
	}
	writeJSON(w, s.Lights(r.Context()))
}

func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	s.rescan(r.Context())
	writeJSON(w, s.Lights(r.Context()))
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	d := s.lookup(r.PathValue("id"))
	if d == nil {
		http.Error(w, "unknown light", http.StatusNotFound)
		return
	}
	var patch elgato.LightState
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st, err := d.client.SetLights(r.Context(), patch)
	if err == nil && len(st.Lights) == 0 {
		err = errors.New("light returned no state")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if s.OnChange != nil && patch.On != nil {
		go s.Lights(context.Background()) // let the menu-bar/tray icon catch up
	}
	writeJSON(w, st.Lights[0])
}

func (s *Server) handleIdentify(w http.ResponseWriter, r *http.Request) {
	d := s.lookup(r.PathValue("id"))
	if d == nil {
		http.Error(w, "unknown light", http.StatusNotFound)
		return
	}
	if err := d.client.Identify(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
