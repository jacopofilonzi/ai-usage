package app

import (
	"log/slog"
	"sync"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/config"
	"github.com/jacopofilonzi/ai-usage/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Corner widgets (Windows): one transparent, always-on-top, click-through window per target
// screen. The window ignores the mouse until the cursor reaches the quarter disc around its
// corner; a ticker watches the cursor because a click-through window gets no mouse messages.
// The ticker runs only while widgets exist and skips work while the cursor stands still.

const (
	hoverTick       = 80 * time.Millisecond
	displayDebounce = 500 * time.Millisecond
)

type widget struct {
	win    *application.WebviewWindow
	bounds application.Rect // physical
	corner config.Corner
	scale  float32
	hover  bool
}

type widgetManager struct {
	a *App
	// relayout is armed from the window procedure (UI thread), which must never wait for mu.
	relayout *time.Timer

	mu      sync.Mutex
	cfg     config.Config
	applied bool // cfg is valid
	widgets map[string]*widget
	stop    chan struct{} // closes the hover ticker; nil when it is not running
}

func newWidgetManager(a *App) *widgetManager {
	m := &widgetManager{a: a, widgets: map[string]*widget{}}
	m.relayout = time.AfterFunc(time.Hour, func() {
		m.mu.Lock()
		cfg, ok := m.cfg, m.applied
		m.mu.Unlock()
		if ok {
			slog.Info("display configuration changed: placing widgets again")
			m.apply(cfg)
			m.a.emitSettings() // the screen list changed
		}
	})
	m.relayout.Stop()
	return m
}

// screensChanged is called on the UI thread for every display/work-area change message.
func (m *widgetManager) screensChanged() {
	m.relayout.Reset(displayDebounce)
}

// apply creates, moves or closes widget windows so there is exactly one per target screen, on the
// configured corner.
func (m *widgetManager) apply(c config.Config) {
	if !m.a.isRunning() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg, m.applied = c, true

	all := m.a.wails.Screen.GetAll()
	want := map[string]*application.Screen{}
	for _, s := range targetScreens(all, c.Monitor) {
		want[widgetName(screenKey(s, all))] = s
	}
	for name, w := range m.widgets {
		if _, ok := want[name]; !ok {
			w.win.Close()
			delete(m.widgets, name)
		}
	}
	for name, s := range want {
		b := widgetBounds(s.PhysicalWorkArea, s.ScaleFactor, c.Corner)
		if w, ok := m.widgets[name]; ok {
			if w.bounds != b {
				w.win.SetPhysicalBounds(b)
			}
			w.bounds, w.corner, w.scale = b, c.Corner, s.ScaleFactor
			continue
		}
		m.widgets[name] = &widget{win: m.newWindow(name, s, b), bounds: b, corner: c.Corner, scale: s.ScaleFactor}
		slog.Info("widget created", "window", name, "screen", s.Name, "bounds", b, "scale", s.ScaleFactor)
	}

	if len(m.widgets) > 0 && m.stop == nil {
		m.stop = make(chan struct{})
		go m.track(m.stop)
	} else if len(m.widgets) == 0 && m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
}

func (m *widgetManager) newWindow(name string, s *application.Screen, b application.Rect) *application.WebviewWindow {
	w := m.a.wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             name,
		Title:            appName,
		Width:            WidgetSize,
		Height:           WidgetSize,
		URL:              "/?view=" + ViewWidget,
		Frameless:        true,
		DisableResize:    true,
		AlwaysOnTop:      true,
		Hidden:           true, // shown once placed, so it never flashes at the default position
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.RGBA{},
		// Created on the target screen so WebView2 starts at that screen's DPI.
		Screen:          s,
		InitialPosition: application.WindowXY,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
			// The decorations extend the DWM frame into the client area, which a transparent
			// window shows as a ghost caption and border.
			DisableFramelessWindowDecorations: true,
		},
	})
	w.SetPhysicalBounds(b)
	w.SetIgnoreMouseEvents(true)
	w.Show()
	return w
}

// close stops tracking; the windows go away with the app.
func (m *widgetManager) close() {
	m.relayout.Stop()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
}

type hoverChange struct {
	name  string
	win   *application.WebviewWindow
	hover bool
}

func (m *widgetManager) track(stop chan struct{}) {
	t := time.NewTicker(hoverTick)
	defer t.Stop()
	lastX, lastY, moved := 0, 0, true
	for {
		select {
		case <-stop:
			return
		case <-t.C:
		}
		x, y, ok := platform.CursorPos()
		if !ok {
			continue
		}
		moved = moved || x != lastX || y != lastY
		lastX, lastY = x, y
		if !moved {
			continue
		}
		moved = false

		var changes []hoverChange
		m.mu.Lock()
		for name, w := range m.widgets {
			if h := nextHover(w.hover, x, y, w.bounds, w.corner, w.scale); h != w.hover {
				w.hover = h
				changes = append(changes, hoverChange{name, w.win, h})
			}
		}
		m.mu.Unlock()
		for _, c := range changes {
			c.win.SetIgnoreMouseEvents(!c.hover)
			m.a.wails.Event.Emit(EventWidgetHover, WidgetHover{Window: c.name, Hover: c.hover})
		}
	}
}
