//go:build linux || darwin

package autostart

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Unix platforms register autostart with a file; path() and render() are per OS.

// Enabled reports whether start at login is registered.
func Enabled() (bool, error) {
	p, err := path()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", p, err)
	}
	return true, nil
}

func register(cmdline []string) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte(render(cmdline)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", p, err)
	}
	return nil
}

// Disable removes the registration; it is not an error if there is none.
func Disable() error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", p, err)
	}
	return nil
}
