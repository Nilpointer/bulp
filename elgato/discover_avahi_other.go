//go:build !linux

package elgato

import "context"

func discoverAvahi(context.Context) ([]Device, bool) { return nil, false }
