//go:build !windows

package app

import "github.com/jacopofilonzi/ai-usage/internal/config"

// Corner widgets are Windows only: macOS and Linux show the usage in the tray icon and popover.
type widgetManager struct{}

func newWidgetManager(*App) *widgetManager { return &widgetManager{} }
func (*widgetManager) apply(config.Config) {}
func (*widgetManager) close()              {}
