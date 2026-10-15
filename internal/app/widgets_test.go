package app

import (
	"testing"

	"github.com/jacopofilonzi/ai-usage/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func screensFixture() []*application.Screen {
	return []*application.Screen{
		{ID: "65537", Name: `\\.\DISPLAY1`, IsPrimary: false},
		{ID: "65539", Name: `\\.\DISPLAY2`, IsPrimary: true},
	}
}

func TestTargetScreens(t *testing.T) {
	all := screensFixture()
	cases := map[string][]string{
		config.MonitorPrimary: {"65539"},
		config.MonitorAll:     {"65537", "65539"},
		`\\.\DISPLAY1`:        {"65537"},
		"65537":               {"65537"}, // raw Wails ID still accepted
		"gone":                {"65539"}, // unknown → primary
	}
	for monitor, want := range cases {
		got := targetScreens(all, monitor)
		if len(got) != len(want) {
			t.Fatalf("%q: got %d screens, want %v", monitor, len(got), want)
		}
		for i := range want {
			if got[i].ID != want[i] {
				t.Errorf("%q: screen %d = %s, want %s", monitor, i, got[i].ID, want[i])
			}
		}
	}
	if targetScreens(nil, config.MonitorAll) != nil {
		t.Error("no screens must give no widgets")
	}
	noPrimary := []*application.Screen{{ID: "a"}, {ID: "b"}}
	if got := targetScreens(noPrimary, config.MonitorPrimary); len(got) != 1 || got[0].ID != "a" {
		t.Errorf("without a primary flag the first screen is used: %v", got)
	}
}

func TestScreenKeyAndName(t *testing.T) {
	all := screensFixture()
	if k := screenKey(all[0], all); k != `\\.\DISPLAY1` {
		t.Errorf("key = %q", k)
	}
	dup := []*application.Screen{{ID: "1", Name: "DELL"}, {ID: "2", Name: "DELL"}}
	if k := screenKey(dup[1], dup); k != "2" {
		t.Errorf("duplicate names must fall back to the ID: %q", k)
	}
	if n := widgetName(`\\.\DISPLAY1`); n != "widget-____DISPLAY1" {
		t.Errorf("widgetName = %q", n)
	}
}

func TestWidgetBounds(t *testing.T) {
	// 1920×1040 work area (taskbar at the bottom) at 150%: 72 DIP = 108 px.
	work := application.Rect{X: 0, Y: 0, Width: 1920, Height: 1040}
	cases := map[config.Corner]application.Rect{
		config.TopLeft:     {X: 0, Y: 0, Width: 108, Height: 108},
		config.TopRight:    {X: 1812, Y: 0, Width: 108, Height: 108},
		config.BottomLeft:  {X: 0, Y: 932, Width: 108, Height: 108},
		config.BottomRight: {X: 1812, Y: 932, Width: 108, Height: 108},
	}
	for corner, want := range cases {
		if got := widgetBounds(work, 1.5, corner); got != want {
			t.Errorf("%s: %+v, want %+v", corner, got, want)
		}
	}
	// Secondary screen left of the primary, negative coordinates.
	got := widgetBounds(application.Rect{X: -1280, Y: 0, Width: 1280, Height: 1024}, 1, config.BottomRight)
	if want := (application.Rect{X: -72, Y: 952, Width: 72, Height: 72}); got != want {
		t.Errorf("negative origin: %+v, want %+v", got, want)
	}
}

func TestNextRevealed(t *testing.T) {
	// 1920×1080 screen at 150%, taskbar 60 px at the bottom: work area 1920×1020.
	screen := application.Rect{Width: 1920, Height: 1080}
	r := widgetBounds(application.Rect{Width: 1920, Height: 1020}, 1.5, config.BottomRight)
	wx, wy := r.X+r.Width, r.Y+r.Height

	// On the widget but not in the hot corner: stays hidden.
	if nextRevealed(false, wx-20, wy-20, r, screen, config.BottomRight, 1.5) {
		t.Error("a hidden widget must not appear when the cursor is merely on it")
	}
	// Pushed into the screen corner (on the taskbar): revealed.
	if !nextRevealed(false, 1919, 1079, r, screen, config.BottomRight, 1.5) {
		t.Error("the screen corner must reveal")
	}
	// Moving up from the taskbar onto the widget keeps it revealed.
	if !nextRevealed(true, wx-20, wy+30, r, screen, config.BottomRight, 1.5) {
		t.Error("the way from the screen corner to the widget must keep it revealed")
	}
	if !nextRevealed(true, wx-60, wy-60, r, screen, config.BottomRight, 1.5) {
		t.Error("on the disc must keep it revealed")
	}
	// Far away: hidden again.
	if nextRevealed(true, 900, 500, r, screen, config.BottomRight, 1.5) {
		t.Error("moving away must hide")
	}

	tl := application.Rect{X: 100, Y: 50, Width: 800, Height: 600}
	if !nextRevealed(false, 102, 52, widgetBounds(tl, 1, config.TopLeft), tl, config.TopLeft, 1) {
		t.Error("top-left hot corner")
	}
}

func TestOnDisc(t *testing.T) {
	r := widgetBounds(application.Rect{Width: 1920, Height: 1020}, 1, config.BottomRight)
	wx, wy := r.X+r.Width, r.Y+r.Height
	if !onDisc(wx-30, wy-30, r, config.BottomRight, 1) {
		t.Error("inside the radius")
	}
	if onDisc(r.X, r.Y, r, config.BottomRight, 1) {
		t.Error("the window's far corner is outside the disc")
	}
	if onDisc(wx-5, wy+2, r, config.BottomRight, 1) {
		t.Error("outside the window")
	}
}
