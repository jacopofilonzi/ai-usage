package app

import (
	"math"
	"strings"

	"github.com/jacopofilonzi/ai-usage/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Geometry of the corner widgets, shared by every OS so it is tested everywhere; only Windows
// creates widget windows (widgets_windows.go).
//
// Placement and hover work in physical pixels: the window is placed with SetPhysicalBounds on the
// screen's physical work area, and the cursor comes from GetCursorPos, also physical. Distances are
// converted to DIP by the screen's scale factor before comparing them with the contract radii.
// This avoids Wails' DIP layout of mixed-DPI setups, which is an approximation of the real one.

// targetScreens picks the screens that get a widget for config.Monitor: "all", "primary", or a
// screen key; an unknown key falls back to the primary screen.
func targetScreens(all []*application.Screen, monitor string) []*application.Screen {
	if len(all) == 0 {
		return nil
	}
	if monitor == config.MonitorAll {
		return all
	}
	if monitor != config.MonitorPrimary {
		for _, s := range all {
			if screenKey(s, all) == monitor || s.ID == monitor {
				return []*application.Screen{s}
			}
		}
	}
	for _, s := range all {
		if s.IsPrimary {
			return []*application.Screen{s}
		}
	}
	return all[:1]
}

// widgetName is the window name for a screen: WidgetWindowPrefix + the key, reduced to
// [A-Za-z0-9_-] (Windows device names are \\.\DISPLAY1).
func widgetName(key string) string {
	var b strings.Builder
	b.WriteString(WidgetWindowPrefix)
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// widgetBounds is the widget's physical rectangle: WidgetSize DIP square with its outer corner on
// the given corner of the work area.
func widgetBounds(work application.Rect, scale float32, corner config.Corner) application.Rect {
	if scale <= 0 {
		scale = 1
	}
	size := int(math.Round(float64(WidgetSize) * float64(scale)))
	r := application.Rect{X: work.X, Y: work.Y, Width: size, Height: size}
	if corner == config.TopRight || corner == config.BottomRight {
		r.X = work.X + work.Width - size
	}
	if corner == config.BottomLeft || corner == config.BottomRight {
		r.Y = work.Y + work.Height - size
	}
	return r
}

// cornerPoint is the outer corner of the widget rectangle, the centre of its quarter disc.
func cornerPoint(r application.Rect, corner config.Corner) (x, y float64) {
	x, y = float64(r.X), float64(r.Y)
	if corner == config.TopRight || corner == config.BottomRight {
		x += float64(r.Width)
	}
	if corner == config.BottomLeft || corner == config.BottomRight {
		y += float64(r.Height)
	}
	return x, y
}

// nextHover applies the hysteresis: the cursor (physical pixel px, py) enters within
// WidgetCollapsedRadius of the corner and leaves beyond WidgetExpandedRadius, both in DIP, and
// only counts while inside the widget window.
func nextHover(hover bool, px, py int, r application.Rect, corner config.Corner, scale float32) bool {
	if px < r.X || py < r.Y || px >= r.X+r.Width || py >= r.Y+r.Height {
		return false
	}
	if scale <= 0 {
		scale = 1
	}
	cx, cy := cornerPoint(r, corner)
	d := math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy) / float64(scale)
	if hover {
		return d <= WidgetExpandedRadius
	}
	return d <= WidgetCollapsedRadius
}

func (a *App) geometry() WidgetGeometry {
	return WidgetGeometry{
		Corner:          a.config().Corner,
		Size:            WidgetSize,
		CollapsedRadius: WidgetCollapsedRadius,
		ExpandedRadius:  WidgetExpandedRadius,
	}
}
