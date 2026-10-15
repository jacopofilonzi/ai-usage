package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// A session spanning midnight keeps writing into the previous day's directory, so the newest
	// file is not always in the newest directory: a couple of extra days cover that cheaply.
	dayDirsToScan = 3

	initialTail = 256 << 10
	maxTail     = 16 << 20
)

// latestRollout returns the most recently modified rollout-*.jsonl under sessionsDir, or "" when
// there is none (including when sessionsDir does not exist).
func latestRollout(ctx context.Context, sessionsDir string) (string, error) {
	years, err := numericSubdirs(sessionsDir)
	if err != nil {
		return "", err
	}
	var (
		best     string
		bestTime time.Time
		scanned  int
	)
	for _, y := range years {
		months, err := numericSubdirs(y)
		if err != nil {
			return "", err
		}
		for _, m := range months {
			days, err := numericSubdirs(m)
			if err != nil {
				return "", err
			}
			for _, d := range days {
				if err := ctx.Err(); err != nil {
					return "", fmt.Errorf("scan sessions: %w", err)
				}
				found, err := newestRolloutIn(d)
				if err != nil {
					return "", err
				}
				if found.path == "" {
					continue
				}
				if best == "" || found.mod.After(bestTime) {
					best, bestTime = found.path, found.mod
				}
				scanned++
				if scanned >= dayDirsToScan {
					return best, nil
				}
			}
		}
	}
	return best, nil
}

// numericSubdirs lists the subdirectories of dir whose names are numbers (YYYY, MM, DD), newest
// first. A missing dir yields no entries.
func numericSubdirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}
	type numbered struct {
		n    int
		name string
	}
	var dirs []numbered
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		dirs = append(dirs, numbered{n, e.Name()})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].n > dirs[j].n })
	out := make([]string, len(dirs))
	for i, d := range dirs {
		out[i] = filepath.Join(dir, d.name)
	}
	return out, nil
}

type fileMod struct {
	path string
	mod  time.Time
}

func newestRolloutIn(dir string) (fileMod, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fileMod{}, fmt.Errorf("list %s: %w", dir, err)
	}
	var best fileMod
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "rollout-") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		info, err := e.Info()
		if errors.Is(err, fs.ErrNotExist) {
			continue // removed (or compressed) between ReadDir and Info
		}
		if err != nil {
			return fileMod{}, fmt.Errorf("stat %s: %w", name, err)
		}
		if best.path == "" || info.ModTime().After(best.mod) {
			best = fileMod{filepath.Join(dir, name), info.ModTime()}
		}
	}
	return best, nil
}

// lastSnapshot returns the last rate-limit snapshot in the file, or nil when it has none. It reads
// only the tail, doubling it until a snapshot is found or the whole file was read.
func lastSnapshot(path string) (*snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	size := info.Size()

	for tail := int64(initialTail); ; tail *= 2 {
		start := max(size-tail, 0)
		buf := make([]byte, size-start)
		if _, err := f.ReadAt(buf, start); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read tail: %w", err)
		}
		if start > 0 {
			// The first line is probably cut in half: drop it.
			if i := bytes.IndexByte(buf, '\n'); i >= 0 {
				buf = buf[i+1:]
			} else {
				buf = nil
			}
		}
		if snap := scanBackwards(buf); snap != nil {
			return snap, nil
		}
		if start == 0 || tail >= maxTail {
			return nil, nil
		}
	}
}

// rolloutLine is the envelope of every rollout line.
type rolloutLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type tokenCountPayload struct {
	Type       string         `json:"type"`
	RateLimits *rateLimitsRaw `json:"rate_limits"`
}

type rateLimitsRaw struct {
	LimitID   string     `json:"limit_id"`
	Primary   *windowRaw `json:"primary"`
	Secondary *windowRaw `json:"secondary"`
}

type windowRaw struct {
	UsedPercent     float64 `json:"used_percent"`
	WindowMinutes   *int64  `json:"window_minutes"`
	ResetsAt        *int64  `json:"resets_at"`         // unix seconds (current versions)
	ResetsInSeconds *int64  `json:"resets_in_seconds"` // relative to the line timestamp (2025 versions)
}

// scanBackwards returns the snapshot of the last token_count line with rate limits, preferring the
// main Codex limit over other limit ids. A line that fails to parse (e.g. the one Codex is writing
// right now) is skipped, not fatal.
func scanBackwards(buf []byte) *snapshot {
	var fallback *snapshot
	for len(buf) > 0 {
		var line []byte
		if i := bytes.LastIndexByte(bytes.TrimRight(buf, "\r\n"), '\n'); i >= 0 {
			line, buf = buf[i+1:], buf[:i]
		} else {
			line, buf = buf, nil
		}
		// Cheap filter before decoding: most lines are messages and tool calls.
		if !bytes.Contains(line, []byte(`"token_count"`)) || !bytes.Contains(line, []byte(`"rate_limits"`)) {
			continue
		}
		var l rolloutLine
		if err := json.Unmarshal(line, &l); err != nil || l.Type != "event_msg" {
			continue
		}
		var p tokenCountPayload
		if err := json.Unmarshal(l.Payload, &p); err != nil || p.Type != "token_count" || p.RateLimits == nil {
			continue
		}
		if p.RateLimits.Primary == nil && p.RateLimits.Secondary == nil {
			continue
		}
		ts, _ := time.Parse(time.RFC3339Nano, l.Timestamp) // zero when missing: relative resets become unknown
		snap := &snapshot{at: ts, raw: *p.RateLimits}
		if isMainLimit(p.RateLimits.LimitID) {
			return snap
		}
		if fallback == nil {
			fallback = snap
		}
	}
	return fallback
}

// isMainLimit reports whether a snapshot is the account's general Codex limit. Newer versions tag
// snapshots with a limit_id and may also record per-model quotas; older ones have no id at all.
func isMainLimit(id string) bool {
	return id == "" || id == "codex"
}
