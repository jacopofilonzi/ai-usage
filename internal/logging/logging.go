// Package logging sets up the default slog logger.
//
// CONTRACT STUB (phase 0): ported from wake-lock by the wake-lock agent.
package logging

import (
	"io"
)

// Setup installs the logger and returns the log file to close on exit.
func Setup() (io.Closer, error) { return io.NopCloser(nil), nil }
