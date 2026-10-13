package elgato

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
)

const ServiceType = "_elg._tcp"

// Device is a light found via Bonjour/mDNS.
type Device struct {
	Instance string // e.g. "Elgato Key Light 1A2B"
	Host     string // host:port, using an IPv4 address when available
	Model    string // TXT "md"
	ID       string // TXT "id" (MAC address)
}

func (d Device) Client() *Client { return NewClient(d.Host) }

// Discover browses for lights until ctx is done and returns what it found,
// de-duplicated by device ID and sorted by name.
func Discover(ctx context.Context) ([]Device, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, err
	}
	entries := make(chan *zeroconf.ServiceEntry)
	if err := resolver.Browse(ctx, ServiceType, "local.", entries); err != nil {
		return nil, err
	}

	found := map[string]Device{}
	for e := range entries { // closed by zeroconf when ctx is done
		d := Device{Instance: unescape(e.Instance)}
		for _, kv := range e.Text {
			k, v, _ := strings.Cut(kv, "=")
			switch k {
			case "md":
				d.Model = v
			case "id":
				d.ID = v
			}
		}
		var ip net.IP
		if len(e.AddrIPv4) > 0 {
			ip = e.AddrIPv4[0]
		} else if len(e.AddrIPv6) > 0 {
			ip = e.AddrIPv6[0]
		}
		if ip != nil {
			d.Host = net.JoinHostPort(ip.String(), strconv.Itoa(e.Port))
		} else {
			d.Host = net.JoinHostPort(strings.TrimSuffix(e.HostName, "."), strconv.Itoa(e.Port))
		}
		key := d.ID
		if key == "" {
			key = d.Instance
		}
		found[key] = d
	}

	devices := make([]Device, 0, len(found))
	for _, d := range found {
		devices = append(devices, d)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Instance < devices[j].Instance })
	return devices, nil
}

// unescape turns DNS-SD escapes like "Elgato\ Ring\ Light" back into plain text.
func unescape(s string) string {
	return strings.ReplaceAll(s, `\`, "")
}
