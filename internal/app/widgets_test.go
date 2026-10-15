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
	// 1920×1040 work area (taskbar at the bottom) at 150%.
	work := application.Rect{X: 0, Y: 0, Width: 1920, Height: 1040}
	cases := map[config.Corner]application.Rect{
		config.TopLeft:     {X: 0, Y: 0, Width: 300, Height: 300},
		config.TopRight:    {X: 1620, Y: 0, Width: 300, Height: 300},
		config.BottomLeft:  {X: 0, Y: 740, Width: 300, Height: 300},
		config.BottomRight: {X: 1620, Y: 740, Width: 300, Height: 300},
	}
	for corner, want := range cases {
		if got := widgetBounds(work, 1.5, corner); got != want {
			t.Errorf("%s: %+v, want %+v", corner, got, want)
		}
	}
	// Secondary screen left of the primary, negative coordinates.
	got := widgetBounds(application.Rect{X: -1280, Y: 0, Width: 1280, Height: 1024}, 1, config.BottomRight)
	if want := (application.Rect{X: -200, Y: 824, Width: 200, Height: 200}); got != want {
		t.Errorf("negative origin: %+v, want %+v", got, want)
	}
}

func TestNextHover(t *testing.T) {
	r := widgetBounds(application.Rect{Width: 1920, Height: 1040}, 1.5, config.BottomRight)
	cx, cy := r.X+r.Width, r.Y+r.Height // the corner, just outside the window

	// 50 DIP from the corner (75 px at 150%): inside the collapsed radius.
	near := 53 // ≈ 50·1.5/√2
	if !nextHover(false, cx-near, cy-near, r, config.BottomRight, 1.5) {
		t.Error("cursor near the corner must enter")
	}
	// 100 DIP away: outside the collapsed radius, inside the expanded one.
	mid := 106 // ≈ 100·1.5/√2
	if nextHover(false, cx-mid, cy-mid, r, config.BottomRight, 1.5) {
		t.Error("must not enter beyond the collapsed radius")
	}
	if !nextHover(true, cx-mid, cy-mid, r, config.BottomRight, 1.5) {
		t.Error("must stay within the expanded radius")
	}
	// Window's far corner: beyond the expanded radius.
	if nextHover(true, r.X, r.Y, r, config.BottomRight, 1.5) {
		t.Error("must leave beyond the expanded radius")
	}
	// Just below the work area (on the taskbar) is outside the window.
	if nextHover(true, cx-5, cy+2, r, config.BottomRight, 1.5) {
		t.Error("cursor outside the window must not hover")
	}

	tl := widgetBounds(application.Rect{X: 100, Y: 50, Width: 800, Height: 600}, 1, config.TopLeft)
	if !nextHover(false, 110, 60, tl, config.TopLeft, 1) {
		t.Error("top-left corner")
	}
}
