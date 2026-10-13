package elgato

import (
	"context"
	"net"

	"github.com/godbus/dbus/v5"
)

const avahi = "org.freedesktop.Avahi"

// discoverAvahi browses via the Avahi daemon on the system bus. Its cache and
// retries make it far more reliable than querying ourselves: the lights sit on
// Wi-Fi, where multicast packets get dropped. ok is false if Avahi can't be
// reached (not running, or a sandbox doesn't allow it).
func discoverAvahi(ctx context.Context) (found []Device, ok bool) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, false
	}
	defer conn.Close()

	sigs := make(chan *dbus.Signal, 64)
	conn.Signal(sigs)
	if err := conn.AddMatchSignal(dbus.WithMatchSender(avahi), dbus.WithMatchInterface(avahi+".ServiceBrowser")); err != nil {
		return nil, false
	}
	server := conn.Object(avahi, "/")
	var browser dbus.ObjectPath
	// Interface and protocol -1: all of them. Signals that arrive before the
	// path is known wait in sigs.
	if err := server.CallWithContext(ctx, avahi+".Server.ServiceBrowserNew", 0,
		int32(-1), int32(-1), ServiceType, "local", uint32(0)).Store(&browser); err != nil {
		return nil, false
	}
	defer conn.Object(avahi, browser).Call(avahi+".ServiceBrowser.Free", 0)

	resolved := map[string]bool{} // an item shows up once per interface and protocol
	for {
		select {
		case <-ctx.Done():
			return found, true
		case s := <-sigs:
			if s.Path != browser {
				continue
			}
			switch s.Name {
			case avahi + ".ServiceBrowser.ItemNew":
				iface, proto, name := s.Body[0].(int32), s.Body[1].(int32), s.Body[2].(string)
				if resolved[name] {
					continue
				}
				if d, err := resolveAvahi(ctx, server, iface, proto, name); err == nil {
					resolved[name] = true
					found = append(found, d)
				}
			case avahi + ".ServiceBrowser.AllForNow":
				// Avahi has answered from its cache and the network; if that
				// turned up nothing, keep listening until ctx is done.
				if len(found) > 0 {
					return found, true
				}
			case avahi + ".ServiceBrowser.Failure":
				return found, len(found) > 0
			}
		}
	}
}

func resolveAvahi(ctx context.Context, server dbus.BusObject, iface, proto int32, name string) (Device, error) {
	var (
		rIface, rProto, aProto int32
		rName, rType, rDomain  string
		host, addr             string
		port                   uint16
		txt                    [][]byte
		flags                  uint32
	)
	// aprotocol 0: prefer an IPv4 address.
	err := server.CallWithContext(ctx, avahi+".Server.ResolveService", 0,
		iface, proto, name, ServiceType, "local", int32(0), uint32(0)).
		Store(&rIface, &rProto, &rName, &rType, &rDomain, &host, &aProto, &addr, &port, &txt, &flags)
	if err != nil {
		return Device{}, err
	}
	text := make([]string, len(txt))
	for i, b := range txt {
		text[i] = string(b)
	}
	return newDevice(name, text, net.ParseIP(addr), host, int(port)), nil
}
