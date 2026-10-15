package autostart

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "ai-usage"
)

// Enabled reports whether start at login is registered.
func Enabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	_, _, err = k.GetStringValue(valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Run value: %w", err)
	}
	return true, nil
}

func register(cmdline []string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if err := k.SetStringValue(valueName, windows.ComposeCommandLine(cmdline)); err != nil {
		return fmt.Errorf("write Run value: %w", err)
	}
	return nil
}

// Disable removes the registration; it is not an error if there is none.
func Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if err := k.DeleteValue(valueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete Run value: %w", err)
	}
	return nil
}
