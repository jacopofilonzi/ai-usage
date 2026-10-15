// Package autostart registers the app to start at user login, per user and without admin rights.
// Enable registers the running executable plus the given args, so the app re-registers itself
// from wherever it is installed.
//
//   - Windows: value "ai-usage" in HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
//   - macOS: LaunchAgent ~/Library/LaunchAgents/com.jacopofilonzi.ai-usage.plist (RunAtLoad).
//   - Linux: XDG autostart entry $XDG_CONFIG_HOME/autostart/ai-usage.desktop.
//
// Enabled and Disable live in the OS files (file.go is shared by the file-based ones); each OS
// provides register(cmdline) for the full command line, executable first.
//
// Ported from github.com/jacopofilonzi/wake-lock (internal/autostart), with the OS-visible names
// changed from wake-lock to ai-usage and the executable resolved here instead of by the caller.
package autostart

import (
	"fmt"
	"os"
)

// Enable registers the running executable with args to start at login. Calling it again
// replaces the previous registration.
func Enable(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	return register(commandLine(exe, args))
}

func commandLine(exe string, args []string) []string {
	return append([]string{exe}, args...)
}
