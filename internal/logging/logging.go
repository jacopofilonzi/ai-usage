// Package logging sets up the default slog logger: a log file in the user cache dir
// (<user cache>/ai-usage/ai-usage.log, since a GUI build has no console), plus stderr when one is
// attached. The file is rotated once at startup when it grows past maxSize, keeping one previous
// file (ai-usage.log.1).
//
// Ported from github.com/jacopofilonzi/wake-lock (internal/logging).
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	appDir  = "ai-usage"
	logName = "ai-usage.log"
	maxSize = 1 << 20
)

// Setup installs the logger and returns the log file to close on exit.
func Setup() (io.Closer, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("locate user cache dir: %w", err)
	}
	f, err := open(filepath.Join(dir, appDir))
	if err != nil {
		return nil, err
	}
	var w io.Writer = f
	// A Windows GUI build has no valid stderr handle: Stat fails there.
	if _, err := os.Stderr.Stat(); err == nil {
		w = io.MultiWriter(f, os.Stderr)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, nil)))
	return f, nil
}

// open creates dir, rotates an oversized log and opens the log for appending.
func open(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	path := filepath.Join(dir, logName)
	if fi, err := os.Stat(path); err == nil && fi.Size() > maxSize {
		if err := os.Rename(path, path+".1"); err != nil {
			return nil, fmt.Errorf("rotate log: %w", err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log: %w", err)
	}
	return f, nil
}
