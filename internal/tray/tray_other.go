//go:build !windows

package tray

import (
	"github.com/jacopofilonzi/ai-usage/internal/i18n"
	"github.com/jacopofilonzi/ai-usage/internal/platform"
	"github.com/jacopofilonzi/ai-usage/internal/provider"
	"github.com/jacopofilonzi/ai-usage/internal/trayicon"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// setup: left click toggles the popover under the icon (Wails positions it), right click keeps
// the native menu.
func (t *Tray) setup(popover application.Window) {
	if popover == nil {
		return
	}
	t.st.AttachWindow(popover).WindowOffset(4)
	t.st.OnClick(t.st.ToggleWindow)
}

func (t *Tray) iconFor(s State) []byte {
	limits := s.Snapshot.Limits
	if len(limits) > trayicon.MaxRings {
		limits = limits[:trayicon.MaxRings]
	}
	stale := s.Snapshot.Status != provider.StatusOK
	return trayicon.Usage(platform.TrayIconSize(), limits, stale, s.Wake.Active)
}

// labelFor is the highest percentage, next to the icon, where the OS shows tray labels (macOS).
func (t *Tray) labelFor(s State) string {
	if !showLabel || len(s.Snapshot.Limits) == 0 {
		return defaultLabel
	}
	top := s.Snapshot.Limits[0].UsedPct
	for _, l := range s.Snapshot.Limits[1:] {
		top = max(top, l.UsedPct)
	}
	return i18n.Percent(top)
}
