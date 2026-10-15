package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const smCxSmIcon = 49 // SM_CXSMICON

type point struct{ X, Y int32 }

// CursorPos returns the cursor position in physical pixels (the process is per-monitor DPI aware,
// so GetCursorPos does not virtualise it). ok is false when the call fails, e.g. while the secure
// desktop is shown.
func CursorPos() (x, y int, ok bool) {
	var pt point
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y), r != 0
}

// TrayIconSize is the pixel size Windows draws notification-area icons at for the primary
// display's DPI (16 at 100%, 24 at 150%, 32 at 200%).
func TrayIconSize() int {
	r, _, _ := procGetSystemMetrics.Call(smCxSmIcon)
	if r == 0 {
		return 32
	}
	return int(r)
}
