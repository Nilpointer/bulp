package elgato

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

const DefaultPort = 9123

// Client controls a single light.
type Client struct {
	BaseURL string // e.g. http://192.168.1.20:9123
	HTTP    *http.Client
}

// NewClient returns a client for host, which may be "host" or "host:port".
func NewClient(host string) *Client {
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, strconv.Itoa(DefaultPort))
	}
	return &Client{
		BaseURL: "http://" + host,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}
}

// ErrNotSupported is returned when the light answers 404 for an endpoint,
// e.g. battery-info on a mains-powered light.
var ErrNotSupported = errors.New("endpoint not supported by this light")

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotSupported
	}
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, bytes.TrimSpace(msg))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Lights(ctx context.Context) (*Lights, error) {
	var l Lights
	return &l, c.do(ctx, http.MethodGet, "/elgato/lights", nil, &l)
}

// SetLights applies state to every light on the device (Elgato devices
// expose exactly one) and returns the resulting state.
func (c *Client) SetLights(ctx context.Context, state LightState) (*Lights, error) {
	in := Lights{NumberOfLights: 1, Lights: []LightState{state}}
	var out Lights
	return &out, c.do(ctx, http.MethodPut, "/elgato/lights", in, &out)
}

func (c *Client) Settings(ctx context.Context) (*Settings, error) {
	var s Settings
	return &s, c.do(ctx, http.MethodGet, "/elgato/lights/settings", nil, &s)
}

func (c *Client) SetSettings(ctx context.Context, s Settings) error {
	return c.do(ctx, http.MethodPut, "/elgato/lights/settings", s, nil)
}

func (c *Client) AccessoryInfo(ctx context.Context) (*AccessoryInfo, error) {
	var a AccessoryInfo
	return &a, c.do(ctx, http.MethodGet, "/elgato/accessory-info", nil, &a)
}

func (c *Client) SetDisplayName(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPut, "/elgato/accessory-info", map[string]string{"displayName": name}, nil)
}

func (c *Client) BatteryInfo(ctx context.Context) (*BatteryInfo, error) {
	var b BatteryInfo
	return &b, c.do(ctx, http.MethodGet, "/elgato/battery-info", nil, &b)
}

// Identify makes the light blink so it can be found physically.
func (c *Client) Identify(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/elgato/identify", nil, nil)
}
