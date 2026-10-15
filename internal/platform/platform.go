// Package platform is the only place that branches on the OS: the platform name for the frontend
// and the few native calls the shell needs (cursor position, tray icon size). Everything here is
// cgo-free so it cross-compiles; OS-specific bodies live in _windows / _other files.
package platform

import "runtime"

// Name is runtime.GOOS: "windows", "darwin", "linux".
func Name() string { return runtime.GOOS }
