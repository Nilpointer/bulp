// Command bulp-menubar is a system-tray app for Elgato lights (Linux).
package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"bulp/internal/ui"
)

func main() {
	hosts := flag.String("host", "", "comma-separated light addresses (host[:port]); skips Bonjour discovery")
	discover := flag.Duration("discover", 3*time.Second, "how long each Bonjour scan runs")
	debugURL := flag.Bool("print-url", false, "print the UI URL (for debugging in a browser)")
	flag.Parse()

	var hostList []string
	for h := range strings.SplitSeq(*hosts, ",") {
		if h = strings.TrimSpace(h); h != "" {
			hostList = append(hostList, h)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := ui.New(hostList, *discover)
	f := newFrontend(srv)
	srv.OnChange = f.lightsChanged
	srv.Quit = f.quit
	url, err := srv.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if *debugURL {
		log.Printf("UI: %s", url)
	}
	f.run(ctx, url) // blocks until quit
}
