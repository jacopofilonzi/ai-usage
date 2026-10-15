// Contract between the Go side and the frontend: data types, event names, window names and the
// widget geometry. The bound services (services.go) and the frontend both depend on this file, so
// a change here is a change on both sides in the same commit.
package app

import (
	"github.com/jacopofilonzi/ai-usage/internal/config"
	"github.com/jacopofilonzi/ai-usage/internal/provider"
	"github.com/jacopofilonzi/ai-usage/internal/wakelock"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ProviderInfo is one installed (detected) provider.
type ProviderInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// ScreenInfo is one monitor, for the settings window.
type ScreenInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Primary bool   `json:"primary"`
	Width   int    `json:"width"`  // DIP
	Height  int    `json:"height"` // DIP
}

// Settings is everything the settings window shows.
type Settings struct {
	Config    config.Config  `json:"config"`
	Autostart bool           `json:"autostart"` // read from the OS
	Platform  string         `json:"platform"`  // runtime.GOOS
	Version   string         `json:"version"`
	Providers []ProviderInfo `json:"providers"`
	Screens   []ScreenInfo   `json:"screens"`
}

// WidgetReveal tells one widget window that it was revealed (cursor pushed into the screen
// corner) or hidden again (cursor moved away). With config.AlwaysShow the widget is drawn anyway.
type WidgetReveal struct {
	Window   string `json:"window"` // the widget window's name; each widget filters on its own
	Revealed bool   `json:"revealed"`
}

// Geometry of the corner widget, in DIP. The widget window is a Size×Size square whose outer corner
// sits on the configured corner of the screen's work area; the widget is the quarter disc of
// Radius centred on that corner. The frontend draws with these numbers; Go uses them for:
//   - reveal: the cursor gets within WidgetHotCorner of the screen's real corner (which, with the
//     taskbar on that edge, is on the taskbar), and stays revealed while within
//     Radius+WidgetLeaveMargin of the widget's corner;
//   - click-through: the window takes the mouse (wheel, right-click) only while the widget is
//     shown and the cursor is on the disc, so a hidden widget never blocks what is under it.
type WidgetGeometry struct {
	Corner config.Corner `json:"corner"`
	Size   int           `json:"size"`
	Radius int           `json:"radius"`
}

const (
	WidgetSize        = 72
	WidgetRadius      = 64
	WidgetHotCorner   = 8
	WidgetLeaveMargin = 24
)

// Event names. Payload types are registered in init so the binding generator types them.
const (
	EventUsageSnapshot  = "usage:snapshot"   // provider.Snapshot of the active provider
	EventWakeLockStatus = "wakelock:status"  // wakelock.Status
	EventSettings       = "settings:changed" // Settings, after any settings or provider change
	EventWidgetReveal   = "widget:reveal"    // WidgetReveal
)

// Window names and the frontend view each one loads ("/?view=<view>").
const (
	WindowSettings     = "settings"
	WindowPopover      = "popover" // macOS/Linux: attached to the tray icon
	WidgetWindowPrefix = "widget-" // + screen ID; Windows only

	ViewWidget   = "widget"
	ViewPopover  = "popover"
	ViewSettings = "settings"
)

// ContextMenuWidget is the native context menu name; the widget sets
// `--custom-contextmenu: widget` in CSS so right-click opens it.
const ContextMenuWidget = "widget"

func init() {
	application.RegisterEvent[provider.Snapshot](EventUsageSnapshot)
	application.RegisterEvent[wakelock.Status](EventWakeLockStatus)
	application.RegisterEvent[Settings](EventSettings)
	application.RegisterEvent[WidgetReveal](EventWidgetReveal)
}
