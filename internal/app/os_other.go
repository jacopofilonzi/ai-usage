//go:build !windows

package app

import "github.com/wailsapp/wails/v3/pkg/application"

// newPopoverWindow creates the window the tray icon toggles on macOS/Linux: frameless, hidden,
// gone as soon as it loses focus. The tray positions it under the icon.
func (a *App) newPopoverWindow() *application.WebviewWindow {
	w := a.wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:            WindowPopover,
		Title:           appName,
		Width:           320,
		Height:          260,
		URL:             "/?view=" + ViewPopover,
		Frameless:       true,
		DisableResize:   true,
		AlwaysOnTop:     true,
		Hidden:          true,
		HideOnFocusLost: true,
		HideOnEscape:    true,
		BackgroundType:  application.BackgroundTypeTransparent,
		Mac: application.MacWindow{
			// Shown over full-screen apps and on every Space, like a menu bar extra.
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary,
		},
	})
	hideOnClose(w)
	return w
}

// platformOptions: nothing beyond the shared options on macOS/Linux.
func (a *App) platformOptions(*application.Options) {}
