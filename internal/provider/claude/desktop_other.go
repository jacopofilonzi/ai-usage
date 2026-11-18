//go:build !windows

package claude

// readDesktopHistory is nil outside Windows: the desktop app's history file has only been
// observed there, so the fallback is off rather than guessing a path.
var readDesktopHistory func() ([]byte, error)
