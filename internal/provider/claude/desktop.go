package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

// Desktop fallback.
//
// When Claude Code runs inside the Claude desktop app, the app owns sign-in: it refreshes the OAuth
// token itself and hands it to the CLI in memory (CLAUDE_CODE_SDK_HAS_HOST_AUTH_REFRESH), so
// <config dir>/.credentials.json is never refreshed and soon holds an expired token, or none at
// all. The desktop token itself is encrypted with the OS keystore and is deliberately not read.
//
// The desktop app does, however, record the plan usage it fetches in a plain JSON file with no
// secrets in it, plan-usage-history.json in its data dir (observed 2026-11, desktop app on
// Windows, see testdata/plan_usage_history.json):
//
//	{"version":2,"samples":[{"t":<ms epoch>,"org":"<uuid>","u":{"fh":<0–100>,"sd":<0–100>}},…]}
//
// "fh" is the five-hour window and "sd" the seven-day one, the same figures as the usage
// endpoint's five_hour / seven_day utilization. It is sampled every 5–15 minutes while the app runs
// and has no reset times. Poll uses the newest sample only when the token route fails and the
// sample is recent enough to still be meaningful.

const (
	desktopHistoryFile = "plan-usage-history.json"
	// maxDesktopAge is how old the newest sample may be: the app samples at least every ~15 minutes
	// while it is open, so anything older means it is closed and the figures are not current.
	maxDesktopAge = 20 * time.Minute
	// A sample slightly in the future is tolerated (clock skew between processes is not expected,
	// but a file written by a machine with another clock could be synced over).
	maxDesktopSkew = time.Minute
)

// errNoDesktopUsage means the desktop app has no usable, recent sample.
var errNoDesktopUsage = errors.New("no recent Claude desktop usage sample")

type desktopHistory struct {
	Samples []desktopSample `json:"samples"`
}

type desktopSample struct {
	T int64 `json:"t"` // ms since epoch
	U struct {
		FiveHour *float64 `json:"fh"`
		SevenDay *float64 `json:"sd"`
	} `json:"u"`
}

// desktopUsage returns the limits in the newest desktop sample, or an error wrapping
// errNoDesktopUsage when there is none recent enough.
func (p *Provider) desktopUsage() ([]provider.Limit, error) {
	if p.readDesktopHistory == nil {
		return nil, errNoDesktopUsage
	}
	raw, err := p.readDesktopHistory()
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errNoDesktopUsage
	}
	if err != nil {
		return nil, fmt.Errorf("claude: read desktop usage history: %w", err)
	}
	return parseDesktopHistory(raw, p.now())
}

func parseDesktopHistory(raw []byte, now time.Time) ([]provider.Limit, error) {
	var h desktopHistory
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, fmt.Errorf("claude: decode desktop usage history: %w", err)
	}
	var latest *desktopSample
	for i := range h.Samples {
		if latest == nil || h.Samples[i].T > latest.T {
			latest = &h.Samples[i]
		}
	}
	if latest == nil {
		return nil, errNoDesktopUsage
	}
	at := time.UnixMilli(latest.T)
	if age := now.Sub(at); age > maxDesktopAge || age < -maxDesktopSkew {
		return nil, fmt.Errorf("claude: newest desktop sample is from %s: %w", at.UTC().Format(time.RFC3339), errNoDesktopUsage)
	}
	limits := make([]provider.Limit, 0, 2)
	if v := latest.U.FiveHour; v != nil {
		limits = append(limits, provider.Limit{ID: keySession, Kind: provider.KindSession, WindowMinutes: sessionMinutes, UsedPct: *v})
	}
	if v := latest.U.SevenDay; v != nil {
		limits = append(limits, provider.Limit{ID: keyWeekly, Kind: provider.KindWeekly, WindowMinutes: weeklyMinutes, UsedPct: *v})
	}
	if len(limits) == 0 {
		return nil, errNoDesktopUsage
	}
	return limits, nil
}

// newestFile reads the most recently modified of paths, skipping the ones that do not exist. It
// returns an error wrapping fs.ErrNotExist when none does.
func newestFile(paths []string) ([]byte, error) {
	var best string
	var bestMod time.Time
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if best == "" || info.ModTime().After(bestMod) {
			best, bestMod = path, info.ModTime()
		}
	}
	if best == "" {
		return nil, fmt.Errorf("%s: %w", desktopHistoryFile, fs.ErrNotExist)
	}
	f, err := os.Open(best)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filepath.Base(best), err)
	}
	defer func() { _ = f.Close() }()
	raw, err := io.ReadAll(io.LimitReader(f, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filepath.Base(best), err)
	}
	if len(raw) > maxBodyBytes {
		return nil, fmt.Errorf("%s too large", desktopHistoryFile)
	}
	return raw, nil
}
