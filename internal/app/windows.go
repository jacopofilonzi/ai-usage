package app

import (
	"log/slog"
	"strings"

	"github.com/jacopofilonzi/ai-usage/internal/autostart"
	"github.com/jacopofilonzi/ai-usage/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// newSettingsWindow creates the settings window, hidden. Closing it only hides it: the app keeps
// running in the tray.
func (a *App) newSettingsWindow() *application.WebviewWindow {
	w := a.wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      WindowSettings,
		Title:     appName,
		Width:     520,
		Height:    680,
		MinWidth:  420,
		MinHeight: 480,
		URL:       "/?view=" + ViewSettings,
		Hidden:    true,
	})
	hideOnClose(w)
	return w
}

func hideOnClose(w *application.WebviewWindow) {
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		w.Hide()
	})
}

// openSettings re-detects the agents (the user may have just installed one), then shows and
// focuses the settings window.
func (a *App) openSettings() {
	a.detect()
	a.settings.Show()
	a.settings.Focus()
}

// popoverWindow returns the popover as a Window, or a nil interface when there is none.
func (a *App) popoverWindow() application.Window {
	if a.popover == nil {
		return nil
	}
	return a.popover
}

func (a *App) settingsSnapshot() Settings {
	on, err := autostart.Enabled()
	if err != nil {
		slog.Error("read start at login", "err", err)
	}
	return Settings{
		Config:    a.config(),
		Autostart: on,
		Platform:  platform.Name(),
		Version:   Version,
		Providers: a.providerInfos(),
		Screens:   a.screens(),
	}
}

func (a *App) screens() []ScreenInfo {
	out := []ScreenInfo{}
	if !a.isRunning() {
		return out
	}
	all := a.wails.Screen.GetAll()
	for _, s := range all {
		out = append(out, ScreenInfo{
			ID:      screenKey(s, all),
			Name:    strings.TrimPrefix(s.Name, `\\.\`), // Windows device names: \\.\DISPLAY1
			Primary: s.IsPrimary,
			Width:   s.Bounds.Width,
			Height:  s.Bounds.Height,
		})
	}
	return out
}

// screenKey identifies a screen in config.Monitor. Wails' IDs on Windows derive from the HMONITOR
// handle, which changes across reboots and reconnections, so the device name is preferred when it
// is unique.
func screenKey(s *application.Screen, all []*application.Screen) string {
	if s.Name == "" {
		return s.ID
	}
	for _, o := range all {
		if o != s && o.Name == s.Name {
			return s.ID
		}
	}
	return s.Name
}
