//go:build !windows && !darwin

package tray

// StatusNotifierItem hosts rarely show a label; Wails uses it as the item title (default "Wails").
const (
	showLabel    = false
	defaultLabel = "ai-usage"
)
