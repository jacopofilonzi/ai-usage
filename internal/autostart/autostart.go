// Package autostart registers the app to start at login.
//
// CONTRACT STUB (phase 0): ported from wake-lock by the wake-lock agent.
package autostart

// Enabled reports whether start at login is registered.
func Enabled() (bool, error) { return false, nil }

// Enable registers the running executable with args to start at login.
func Enable(args []string) error { return nil }

func Disable() error { return nil }
