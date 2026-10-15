package app

import (
	"github.com/jacopofilonzi/ai-usage/internal/config"
	"github.com/jacopofilonzi/ai-usage/internal/provider"
	"github.com/jacopofilonzi/ai-usage/internal/wakelock"
)

// CONTRACT STUBS (phase 0): the method sets below are what the frontend binds to. The desktop-shell
// agent implements them; signatures change only together with the frontend.

// UsageService exposes the active provider's usage.
type UsageService struct{ a *App }

// Providers lists the detected providers, marking the active one.
func (s *UsageService) Providers() []ProviderInfo { return nil }

// Snapshot returns the active provider's latest snapshot.
func (s *UsageService) Snapshot() provider.Snapshot { return provider.Snapshot{} }

// SetProvider switches the shown provider and persists the choice.
func (s *UsageService) SetProvider(id string) error { return nil }

// Cycle switches to the next (step > 0) or previous (step < 0) detected provider: mouse wheel.
func (s *UsageService) Cycle(step int) error { return nil }

// Refresh polls the active provider now.
func (s *UsageService) Refresh() {}

// SettingsService backs the settings window.
type SettingsService struct{ a *App }

func (s *SettingsService) Get() Settings { return Settings{} }

// Save validates, applies (widget windows, poll interval, language) and persists c.
func (s *SettingsService) Save(c config.Config) error { return nil }

func (s *SettingsService) SetAutostart(on bool) error { return nil }

// Open shows and focuses the settings window.
func (s *SettingsService) Open() {}

func (s *SettingsService) Quit() {}

// WakeLockService controls the wake lock.
type WakeLockService struct{ a *App }

func (s *WakeLockService) Status() wakelock.Status { return wakelock.Status{} }

// Start keeps the computer awake for minutes; 0 = indefinitely.
func (s *WakeLockService) Start(minutes int) error { return nil }

func (s *WakeLockService) Stop() error { return nil }

// SetKeepUnlocked applies and persists the preference.
func (s *WakeLockService) SetKeepUnlocked(on bool) error { return nil }

// WidgetService serves the corner widget windows.
type WidgetService struct{ a *App }

func (s *WidgetService) Geometry() WidgetGeometry { return WidgetGeometry{} }
