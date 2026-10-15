package app

import "github.com/wailsapp/wails/v3/pkg/application"

// newPopoverWindow: Windows has the corner widgets instead of a tray popover.
func (a *App) newPopoverWindow() *application.WebviewWindow { return nil }

// Window messages that mean the monitors or their work areas changed. Wails re-reads its screen
// list on the same messages (on the main thread, right after the interceptor), but emits no event
// for it, so the widgets listen here and re-place themselves after a short debounce.
const (
	wmSettingChange = 0x001A // WM_SETTINGCHANGE
	wmDisplayChange = 0x007E // WM_DISPLAYCHANGE
	spiSetWorkArea  = 0x002F // SPI_SETWORKAREA
)

func (a *App) platformOptions(o *application.Options) {
	o.Windows.WndProcInterceptor = func(_ uintptr, msg uint32, wParam, _ uintptr) (uintptr, bool) {
		if msg == wmDisplayChange || (msg == wmSettingChange && wParam == spiSetWorkArea) {
			a.widgets.screensChanged()
		}
		return 0, false
	}
}
