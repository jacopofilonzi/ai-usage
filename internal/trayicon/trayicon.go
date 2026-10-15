// Package trayicon draws the tray icons as PNG (what Wails' SetIcon takes on every OS: Windows
// turns it into an HICON, macOS into an NSImage sized to the menu bar, Linux into an SNI pixmap).
//
// Pure Go and cgo-free: shapes are point-in-shape tests, rasterised with 4×4 supersampling for
// anti-aliasing. Two icons:
//
//   - App (Windows): a static gauge glyph, two quarter rings around the bottom-left corner like
//     the corner widget, plus an accent dot top-right while the wake lock is active.
//   - Usage (macOS/Linux): the usage itself, one concentric ring per limit (outermost = first,
//     up to 3), arc length = UsedPct clockwise from 12 o'clock, coloured by level; grey when the
//     values are stale; a dot top-right while the wake lock is active.
//
// Callers pick the pixel size: the OS small-icon size on Windows, 2× the menu bar height on
// macOS (Wails sizes the image to the bar thickness, so 44 px renders sharp on retina).
package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

// Level colours, shared with the frontend palette.
var (
	Green  = color.NRGBA{0x22, 0xc5, 0x5e, 0xff}
	Amber  = color.NRGBA{0xf5, 0x9e, 0x0b, 0xff}
	Red    = color.NRGBA{0xef, 0x44, 0x44, 0xff}
	Grey   = color.NRGBA{0x9c, 0xa3, 0xaf, 0xff}
	Accent = color.NRGBA{0x3b, 0x82, 0xf6, 0xff}
	track  = color.NRGBA{0x9c, 0xa3, 0xaf, 0x60}
)

// MaxRings is how many limits the usage icon shows.
const MaxRings = 3

// LevelColor maps a used percentage to the ring colour: green < 70, amber < 90, red ≥ 90.
func LevelColor(pct float64) color.NRGBA {
	switch {
	case pct >= 90:
		return Red
	case pct >= 70:
		return Amber
	default:
		return Green
	}
}

// shape reports whether a point (in pixels, origin top-left) is inside.
type shape func(x, y float64) bool

type layer struct {
	in    shape
	col   color.NRGBA
	erase bool // clears what is below instead of painting
}

const samples = 4

// render rasterises layers in order into a size×size image.
func render(size int, layers []layer) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for py := range size {
		for px := range size {
			var r, g, b, a float64 // premultiplied sums over the samples
			for sy := range samples {
				for sx := range samples {
					x := float64(px) + (float64(sx)+0.5)/samples
					y := float64(py) + (float64(sy)+0.5)/samples
					var cr, cg, cb, ca float64
					for _, l := range layers {
						if !l.in(x, y) {
							continue
						}
						if l.erase {
							cr, cg, cb, ca = 0, 0, 0, 0
							continue
						}
						la := float64(l.col.A) / 255
						cr = float64(l.col.R)/255*la + cr*(1-la)
						cg = float64(l.col.G)/255*la + cg*(1-la)
						cb = float64(l.col.B)/255*la + cb*(1-la)
						ca = la + ca*(1-la)
					}
					r, g, b, a = r+cr, g+cg, b+cb, a+ca
				}
			}
			n := float64(samples * samples)
			r, g, b, a = r/n, g/n, b/n, a/n
			if a == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(math.Round(r / a * 255)),
				G: uint8(math.Round(g / a * 255)),
				B: uint8(math.Round(b / a * 255)),
				A: uint8(math.Round(a * 255)),
			})
		}
	}
	return img
}

func encode(img image.Image) []byte {
	var buf bytes.Buffer
	// Encoding an in-memory NRGBA cannot fail.
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func disc(cx, cy, r float64) shape {
	return func(x, y float64) bool { return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r }
}

// arc is the part of the ring [r1, r2] around (cx, cy) between angles from and to (radians,
// 0 = 12 o'clock, growing clockwise; to may exceed 2π for a full ring).
func arc(cx, cy, r1, r2, from, to float64) shape {
	return func(x, y float64) bool {
		dx, dy := x-cx, y-cy
		d2 := dx*dx + dy*dy
		if d2 < r1*r1 || d2 > r2*r2 {
			return false
		}
		if to-from >= 2*math.Pi {
			return true
		}
		a := math.Atan2(dx, -dy) // clockwise from 12 o'clock
		if a < 0 {
			a += 2 * math.Pi
		}
		return a >= from && a <= to
	}
}

// wakeDot is the accent dot in the top-right corner, with a transparent gap around it so it
// reads on top of the rings.
func wakeDot(size float64) []layer {
	r := size * 0.15
	cx, cy := size-r-size*0.02, r+size*0.02
	return []layer{
		{in: disc(cx, cy, r+size*0.06), erase: true},
		{in: disc(cx, cy, r), col: Accent},
	}
}

// App draws the Windows tray icon.
func App(size int, awake bool) []byte {
	s := float64(size)
	m := s * 0.06    // margin
	cx, cy := m, s-m // the corner the rings wrap
	w := s * 0.2     // ring width
	outer := s - 2*m // outer radius reaches the opposite edges
	inner := outer - w - s*0.1
	q := math.Pi / 2 // the quarter from 12 to 3 o'clock
	layers := []layer{
		{in: arc(cx, cy, outer-w, outer, 0, q), col: track},
		{in: arc(cx, cy, outer-w, outer, 0, q*0.65), col: Accent},
		{in: arc(cx, cy, inner-w, inner, 0, q), col: track},
		{in: arc(cx, cy, inner-w, inner, 0, q*0.4), col: Green},
	}
	if awake {
		layers = append(layers, wakeDot(s)...)
	}
	return encode(render(size, layers))
}

// Usage draws the macOS/Linux tray icon from the active provider's limits.
func Usage(size int, limits []provider.Limit, stale, awake bool) []byte {
	s := float64(size)
	c := s / 2
	outer := c - s*0.04
	n := min(len(limits), MaxRings)
	rings := max(n, 1) // with no data, one empty track
	gap := s * 0.04
	hole := s * 0.1
	w := min((outer-hole)/float64(rings)-gap, s*0.14) // a lone ring stays a ring, not a pie
	var layers []layer
	for i := range rings {
		r2 := outer - float64(i)*(w+gap)
		r1 := r2 - w
		layers = append(layers, layer{in: arc(c, c, r1, r2, 0, 2*math.Pi), col: track})
		if i >= n {
			continue
		}
		pct := limits[i].UsedPct
		col := LevelColor(pct)
		if stale {
			col = Grey
		}
		if frac := math.Min(math.Max(pct, 0), 100) / 100; frac > 0 {
			layers = append(layers, layer{in: arc(c, c, r1, r2, 0, frac*2*math.Pi), col: col})
		}
	}
	if awake {
		layers = append(layers, wakeDot(s)...)
	}
	return encode(render(size, layers))
}
