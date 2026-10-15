//go:build !windows

package platform

// CursorPos is only needed for the Windows corner widgets.
func CursorPos() (x, y int, ok bool) { return 0, 0, false }

// TrayIconSize is the pixel size of the menu bar / tray icon: macOS sizes the image to the bar
// thickness (22 pt), so 2× keeps it sharp on retina; Linux tray hosts scale the pixmap.
func TrayIconSize() int { return 44 }
