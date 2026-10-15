package app

import (
	"fmt"

	"github.com/jacopofilonzi/ai-usage/internal/autostart"
	"github.com/jacopofilonzi/ai-usage/internal/config"
	"github.com/jacopofilonzi/ai-usage/internal/provider"
	"github.com/jacopofilonzi/ai-usage/internal/wakelock"
)

// The method sets below are what the frontend binds to: signatures and doc comments end up in the
// generated bindings, so they change only together with the frontend. The bodies delegate to App.

// UsageService exposes the active provider's usage.
type UsageService struct{ a *App }

// Providers lists the detected providers, marking the active one.
func (s *UsageService) Providers() []ProviderInfo { return s.a.providerInfos() }

// Snapshot returns the active provider's latest snapshot.
func (s *UsageService) Snapshot() provider.Snapshot { return s.a.snapshot() }

// SetProvider switches the shown provider and persists the choice.
func (s *UsageService) SetProvider(id string) error { return s.a.setProvider(id) }

// Cycle switches to the next (step > 0) or previous (step < 0) detected provider: mouse wheel.
func (s *UsageService) Cycle(step int) error { return s.a.cycle(step) }

// Refresh polls the active provider now.
func (s *UsageService) Refresh() { s.a.poller.Refresh() }

// SettingsService backs the settings window.
type SettingsService struct{ a *App }

func (s *SettingsService) Get() Settings { return s.a.settingsSnapshot() }

// Save validates, applies (widget windows, poll interval, language) and persists c.
func (s *SettingsService) Save(c config.Config) error { return s.a.saveSettings(c) }

func (s *SettingsService) SetAutostart(on bool) error {
	var err error
	if on {
		err = autostart.Enable(nil)
	} else {
		err = autostart.Disable()
	}
	s.a.emitSettings() // also on failure: the frontend re-reads the real state
	if err != nil {
		return fmt.Errorf("set start at login: %w", err)
	}
	return nil
}

// Open shows and focuses the settings window.
func (s *SettingsService) Open() { s.a.openSettings() }

func (s *SettingsService) Quit() { s.a.wails.Quit() }

// WakeLockService controls the wake lock.
type WakeLockService struct{ a *App }

func (s *WakeLockService) Status() wakelock.Status { return s.a.wakeStatus() }

// Start keeps the computer awake for minutes; 0 = indefinitely.
func (s *WakeLockService) Start(minutes int) error { return s.a.startWake(minutes) }

func (s *WakeLockService) Stop() error { return s.a.stopWake() }

// SetKeepUnlocked applies and persists the preference.
func (s *WakeLockService) SetKeepUnlocked(on bool) error { return s.a.setKeepUnlocked(on) }

// WidgetService serves the corner widget windows.
type WidgetService struct{ a *App }

func (s *WidgetService) Geometry() WidgetGeometry { return s.a.geometry() }
