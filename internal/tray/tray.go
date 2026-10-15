// Package tray owns the system tray icon and its menu, plus the widget's native context menu
// (same entries), on every OS.
//
// The menu is described as plain items (menu.go) and only rebuilt when what it shows changes,
// because a rebuild replaces the native menu and drops a click on a menu that is open at that
// moment. The icon, tooltip and macOS label are likewise only pushed when they change.
//
// Native click callbacks may run on the UI thread while another goroutine holds mu and waits for
// that thread (every Wails setter is InvokeSync), so callbacks never take mu: they hand off to a
// goroutine. Right click keeps Wails' default, the native menu.
//
// Per-OS behaviour lives in tray_windows.go / tray_other.go: on Windows the icon is the app glyph,
// left click opens the settings and the tooltip carries the percentages; on macOS/Linux the icon
// draws the usage rings and left click toggles the popover attached to the tray.
package tray

import (
	"bytes"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Tray is safe for concurrent use.
type Tray struct {
	st          *application.SystemTray
	contextName string
	act         Actions

	mu        sync.Mutex
	state     State
	hasState  bool
	menuKey   string
	trayMenu  *application.Menu
	ctxMenu   *application.ContextMenu
	icon      []byte
	tooltip   string
	label     string
	firstSync bool
}

// New creates the tray icon. It must be called once the Wails app is running (the tray is created
// natively at once). popover may be nil; it is used on macOS/Linux only. contextMenu is the name
// the widget's CSS refers to with --custom-contextmenu.
func New(app *application.App, act Actions, popover application.Window, contextMenu string) *Tray {
	t := &Tray{st: app.SystemTray.New(), contextName: contextMenu, act: act, firstSync: true}
	t.setup(popover)
	return t
}

// Update shows s. Cheap when nothing visible changed.
func (t *Tray) Update(s State) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state, t.hasState = s, true
	t.sync(time.Now())
}

// Refresh re-renders the last state at the current time (the wake-lock line shows time left).
func (t *Tray) Refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.hasState {
		t.sync(time.Now())
	}
}

// sync pushes whatever changed to the native tray. Caller holds mu.
func (t *Tray) sync(now time.Time) {
	s := t.state
	items := spec(s, now, t.act)
	if k := key(items); k != t.menuKey {
		t.menuKey = k

		menu := application.NewMenu()
		fill(menu, items)
		t.st.SetMenu(menu)
		release(t.trayMenu)
		t.trayMenu = menu

		// NewContextMenu registers itself under the name, replacing the previous one.
		ctx := application.NewContextMenu(t.contextName)
		fill(ctx.Menu, items)
		ctx.Update()
		if t.ctxMenu != nil {
			release(t.ctxMenu.Menu)
		}
		t.ctxMenu = ctx
	}

	if icon := t.iconFor(s); !bytes.Equal(icon, t.icon) {
		t.icon = icon
		t.st.SetIcon(icon)
	}
	if tip := s.Msg.Tooltip(s.activeName(), s.Snapshot); tip != t.tooltip || t.firstSync {
		t.tooltip = tip
		t.st.SetTooltip(tip)
	}
	if label := t.labelFor(s); label != t.label {
		t.label = label
		t.st.SetLabel(label)
	}
	t.firstSync = false
}
