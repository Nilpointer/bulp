package tray

import (
	"bytes"
	"image/png"
	"os"
	"testing"
)

func TestIconsArePNG(t *testing.T) {
	for name, b := range map[string][]byte{"off": IconOff, "on": IconOn} {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if img.Bounds().Dx() != size {
			t.Fatalf("%s: got %v", name, img.Bounds())
		}
		if dir := os.Getenv("ICON_DUMP"); dir != "" {
			os.WriteFile(dir+"/icon-"+name+".png", b, 0o644)
		}
	}
}
