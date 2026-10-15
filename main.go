// Command ai-usage shows the usage limits of an AI coding agent in the tray (and, on Windows, in a
// screen-corner widget) and can keep the computer awake. See internal/app for the wiring.
package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"

	"github.com/jacopofilonzi/ai-usage/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		slog.Error("open embedded frontend", "err", err)
		os.Exit(1)
	}
	if err := app.Run(dist); err != nil {
		slog.Error("ai-usage stopped", "err", err)
		os.Exit(1)
	}
}
