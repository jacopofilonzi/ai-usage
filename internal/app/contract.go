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

// WidgetHover tells one widget window that the cursor entered or left its interactive area.
type WidgetHover struct {
	Window string `json:"window"` // the widget window's name; each widget filters on its own
	Hover  bool   `json:"hover"`
}

// Geometry of the corner widget, in DIP. The widget window is a Size×Size square whose outer corner
// sits on the screen corner (inside the work area). The interactive area is the quarter disc
// centred on that corner: the cursor enters at CollapsedRadius and, once expanded, leaves beyond
// ExpandedRadius. Go uses this for click-through; the frontend draws with the same numbers.
type WidgetGeometry struct {
	Corner          config.Corner `json:"corner"`
	Size            int           `json:"size"`
	CollapsedRadius int           `json:"collapsedRadius"`
	ExpandedRadius  int           `json:"expandedRadius"`
}

const (
	WidgetSize            = 200
	WidgetCollapsedRadius = 64
	WidgetExpandedRadius  = 196
)

// Event names. Payload types are registered in init so the binding generator types them.
const (
	EventUsageSnapshot  = "usage:snapshot"   // provider.Snapshot of the active provider
	EventWakeLockStatus = "wakelock:status"  // wakelock.Status
	EventSettings       = "settings:changed" // Settings, after any settings or provider change
	EventWidgetHover    = "widget:hover"     // WidgetHover
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
	application.RegisterEvent[WidgetHover](EventWidgetHover)
}
