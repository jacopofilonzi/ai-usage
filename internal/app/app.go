// Package app wires providers, poller, wake lock, tray and windows into the Wails application.
//
// CONTRACT STUB (phase 0): implemented by the desktop-shell agent.
package app

import (
	"io/fs"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App holds the running state shared by the bound services.
type App struct {
	wails *application.App
}

// Run builds the application around the embedded frontend assets and blocks until it quits.
func Run(assets fs.FS) error {
	a := &App{}
	a.wails = application.New(application.Options{
		Name: "ai-usage",
		Services: []application.Service{
			application.NewService(&UsageService{a}),
			application.NewService(&SettingsService{a}),
			application.NewService(&WakeLockService{a}),
			application.NewService(&WidgetService{a}),
		},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
	})
	a.wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  WindowSettings,
		Title: "ai-usage",
		URL:   "/?view=" + ViewSettings,
	})
	return a.wails.Run()
}
