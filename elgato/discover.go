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
// de-duplicated by device ID and sorted by name. On Linux it asks the Avahi
// daemon when it can (more reliable, and it returns as soon as Avahi's cache
// has answered), otherwise it queries over multicast itself.
func Discover(ctx context.Context) ([]Device, error) {
	found, ok := discoverAvahi(ctx)
	if !ok {
		var err error
		if found, err = discoverZeroconf(ctx); err != nil {
			return nil, err
		}
	}

	byID := map[string]Device{}
	for _, d := range found {
		key := d.ID
		if key == "" {
			key = d.Instance
		}
		byID[key] = d
	}
	devices := make([]Device, 0, len(byID))
	for _, d := range byID {
		devices = append(devices, d)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Instance < devices[j].Instance })
	return devices, nil
}

func discoverZeroconf(ctx context.Context) ([]Device, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, err
	}
	entries := make(chan *zeroconf.ServiceEntry)
	if err := resolver.Browse(ctx, ServiceType, "local.", entries); err != nil {
		return nil, err
	}

	var found []Device
	for e := range entries { // closed by zeroconf when ctx is done
		var ip net.IP
		if len(e.AddrIPv4) > 0 {
			ip = e.AddrIPv4[0]
		} else if len(e.AddrIPv6) > 0 {
			ip = e.AddrIPv6[0]
		}
		found = append(found, newDevice(e.Instance, e.Text, ip, e.HostName, e.Port))
	}
	return found, nil
}

// newDevice builds a Device from a resolved service; hostName is used when
// there's no address.
func newDevice(instance string, txt []string, ip net.IP, hostName string, port int) Device {
	d := Device{Instance: unescape(instance)}
	for _, kv := range txt {
		k, v, _ := strings.Cut(kv, "=")
		switch k {
		case "md":
			d.Model = v
		case "id":
			d.ID = v
		}
	}
	if ip != nil {
		d.Host = net.JoinHostPort(ip.String(), strconv.Itoa(port))
	} else {
		d.Host = net.JoinHostPort(strings.TrimSuffix(hostName, "."), strconv.Itoa(port))
	}
	return d
}

// unescape turns DNS-SD escapes like "Elgato\ Ring\ Light" back into plain text.
func unescape(s string) string {
	return strings.ReplaceAll(s, `\`, "")
}
