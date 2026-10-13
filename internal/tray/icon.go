// Package tray provides the tray icons, drawn in code so no image assets are needed.
package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// PNG icons for the system tray: a light bulb, outlined when all lights are
// off and filled warm yellow when any light is on.
var (
	IconOff = bulb(color.NRGBA{0x9a, 0x9a, 0x9a, 0xff}, color.NRGBA{})
	IconOn  = bulb(color.NRGBA{0xf2, 0xb1, 0x2c, 0xff}, color.NRGBA{0xff, 0xd2, 0x5a, 0xff})
)

const size = 64

func bulb(stroke, fill color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			px, py := float64(x)+.5, float64(y)+.5
			outer, inner := glass(px, py, 0), glass(px, py, 4)
			base := (math.Abs(py-49) <= 2 || math.Abs(py-55) <= 2) && math.Abs(px-32) <= 10 ||
				math.Abs(py-60) <= 2 && math.Abs(px-32) <= 5
			switch {
			case outer && !inner, base:
				img.SetNRGBA(x, y, stroke)
			case inner && fill.A > 0:
				img.SetNRGBA(x, y, fill)
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

// glass reports whether (x, y) lies inside the bulb's glass shrunk by inset:
// a circle on top of a neck that narrows towards the base.
func glass(x, y, inset float64) bool {
	const cx, cy, r, neckTop, neckBottom = 32.0, 24.0, 18.0, 34.0, 45.0
	if math.Hypot(x-cx, y-cy) <= r-inset {
		return true
	}
	if y < neckTop || y > neckBottom-inset {
		return false
	}
	half := 15 - (y-neckTop)/(neckBottom-neckTop)*5 // 15 → 10
	return math.Abs(x-cx) <= half-inset
}
