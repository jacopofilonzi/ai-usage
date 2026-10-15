package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

func decode(t *testing.T, data []byte, size int) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
		t.Fatalf("size = %v, want %d", b, size)
	}
	if dir := os.Getenv("TRAYICON_DUMP"); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, t.Name()+".png"), data, 0o644)
	}
	return img
}

func nrgba(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}

func near(a, b color.NRGBA) bool {
	d := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return d(a.R, b.R) < 12 && d(a.G, b.G) < 12 && d(a.B, b.B) < 12 && a.A > 230
}

func TestLevelColor(t *testing.T) {
	cases := map[float64]color.NRGBA{0: Green, 69.9: Green, 70: Amber, 89.9: Amber, 90: Red, 130: Red}
	for pct, want := range cases {
		if got := LevelColor(pct); got != want {
			t.Errorf("LevelColor(%v) = %v, want %v", pct, got, want)
		}
	}
}

// For a 44 px icon with 2 rings: outer ring spans radius ~13.6–20.2 around (22, 22).
const size = 44

func limitsAt(pct ...float64) []provider.Limit {
	var out []provider.Limit
	for _, p := range pct {
		out = append(out, provider.Limit{UsedPct: p})
	}
	return out
}

func TestUsageArcs(t *testing.T) {
	img := decode(t, Usage(size, limitsAt(30, 95), false, false), size)
	// Outer ring, 30%: lit just right of 12 o'clock, track at 6 o'clock.
	if c := nrgba(img, 24, 5); !near(c, Green) {
		t.Errorf("outer ring start = %v, want green", c)
	}
	if c := nrgba(img, 22, 38); near(c, Green) || c.A == 0 {
		t.Errorf("outer ring at 6 o'clock = %v, want the faint track", c)
	}
	// Inner ring, 95%: red almost all around, including 9 o'clock.
	if c := nrgba(img, 11, 22); !near(c, Red) {
		t.Errorf("inner ring at 9 o'clock = %v, want red", c)
	}
	// Centre hole is empty.
	if c := nrgba(img, 22, 22); c.A != 0 {
		t.Errorf("centre = %v, want transparent", c)
	}
}

func TestUsageStaleIsGrey(t *testing.T) {
	img := decode(t, Usage(size, limitsAt(95), true, false), size)
	if c := nrgba(img, 24, 4); !near(c, Grey) {
		t.Errorf("stale ring = %v, want grey", c)
	}
}

func TestUsageEmptyAndOver(t *testing.T) {
	decode(t, Usage(size, nil, false, false), size)
	decode(t, Usage(size, limitsAt(150, 10, 20, 30), false, false), size) // >100 and >3 limits
}

func TestWakeDot(t *testing.T) {
	off := decode(t, Usage(size, limitsAt(10), false, false), size)
	on := decode(t, Usage(size, limitsAt(10), false, true), size)
	x, y := size-8, 8 // centre of the dot
	if c := nrgba(on, x, y); !near(c, Accent) {
		t.Errorf("dot = %v, want accent", c)
	}
	if c := nrgba(off, x, y); near(c, Accent) {
		t.Errorf("dot drawn while the wake lock is off")
	}

	app := decode(t, App(32, true), 32)
	if c := nrgba(app, 32-6, 6); !near(c, Accent) {
		t.Errorf("app icon dot = %v, want accent", c)
	}
	if c := nrgba(decode(t, App(32, false), 32), 32-6, 6); c.A != 0 {
		t.Errorf("app icon top-right = %v, want transparent without wake lock", c)
	}
}

func TestAppSizes(t *testing.T) {
	for _, s := range []int{16, 20, 24, 32, 64} {
		img := decode(t, App(s, false), s)
		// The outer ring passes through the top-left area.
		if c := nrgba(img, s/8, s/8); c.A == 0 {
			t.Errorf("size %d: outer ring missing at the top-left", s)
		}
	}
}
