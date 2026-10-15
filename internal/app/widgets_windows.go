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
// screen. Unless config.AlwaysShow, the widget is hidden until the cursor is pushed into the
// screen corner (see nextRevealed); the window takes the mouse only while the widget is shown and
// the cursor is on its disc. A ticker watches the cursor because a click-through window gets no
// mouse messages; it runs only while widgets exist and skips work while the cursor stands still.

const (
	hoverTick       = 80 * time.Millisecond
	displayDebounce = 500 * time.Millisecond
)

type widget struct {
	win         *application.WebviewWindow
	bounds      application.Rect // physical
	screen      application.Rect // physical bounds of the whole screen (the hot corner)
	corner      config.Corner
	scale       float32
	revealed    bool
	interactive bool // not click-through
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
			w.bounds, w.screen, w.corner, w.scale = b, s.PhysicalBounds, c.Corner, s.ScaleFactor
			continue
		}
		m.widgets[name] = &widget{win: m.newWindow(name, s, b), bounds: b, screen: s.PhysicalBounds, corner: c.Corner, scale: s.ScaleFactor}
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

type widgetChange struct {
	name                  string
	win                   *application.WebviewWindow
	revealed, interactive bool
	revealChanged         bool
	interactiveChanged    bool
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

		var changes []widgetChange
		m.mu.Lock()
		always := m.cfg.AlwaysShow
		for name, w := range m.widgets {
			r := nextRevealed(w.revealed, x, y, w.bounds, w.screen, w.corner, w.scale)
			i := (always || r) && onDisc(x, y, w.bounds, w.corner, w.scale)
			if r != w.revealed || i != w.interactive {
				changes = append(changes, widgetChange{name, w.win, r, i, r != w.revealed, i != w.interactive})
				w.revealed, w.interactive = r, i
			}
		}
		m.mu.Unlock()
		for _, c := range changes {
			if c.interactiveChanged {
				c.win.SetIgnoreMouseEvents(!c.interactive)
			}
			if c.revealChanged {
				m.a.wails.Event.Emit(EventWidgetReveal, WidgetReveal{Window: c.name, Revealed: c.revealed})
			}
		}
	}
}
