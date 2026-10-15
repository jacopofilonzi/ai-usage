package tray

import (
	"github.com/jacopofilonzi/ai-usage/internal/platform"
	"github.com/jacopofilonzi/ai-usage/internal/trayicon"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// setup: the usage lives in the corner widget and the tooltip, so left click opens the settings.
func (t *Tray) setup(_ application.Window) {
	open := t.act.OpenSettings
	t.st.OnClick(func() { go open() })
}

func (t *Tray) iconFor(s State) []byte {
	return trayicon.App(platform.TrayIconSize(), s.Wake.Active)
}

func (t *Tray) labelFor(State) string { return "" }
