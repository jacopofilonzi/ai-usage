package claude

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

const (
	keySession        = "five_hour"
	keyWeekly         = "seven_day"
	weeklyModelPrefix = "seven_day_"

	sessionMinutes = 5 * 60
	weeklyMinutes  = 7 * 24 * 60
)

// window is one usage window object. Other fields (dollar amounts, locked_reason, ...) are ignored.
type window struct {
	Utilization *float64 `json:"utilization"` // already 0–100 (observed: 15.0 alongside "percent":15)
	ResetsAt    *string  `json:"resets_at"`
}

// parseUsage turns the usage response into limits, in the stable order session, weekly,
// weekly_model (by model), other (by id).
func parseUsage(body []byte) ([]provider.Limit, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, fmt.Errorf("decode top-level object: %w", err)
	}
	limits := make([]provider.Limit, 0, 4)
	for key, raw := range top {
		w, ok := decodeWindow(raw)
		if !ok {
			continue
		}
		l := provider.Limit{ID: key, UsedPct: *w.Utilization, ResetsAt: parseResetsAt(key, w.ResetsAt)}
		switch {
		case key == keySession:
			l.Kind, l.WindowMinutes = provider.KindSession, sessionMinutes
		case key == keyWeekly:
			l.Kind, l.WindowMinutes = provider.KindWeekly, weeklyMinutes
		case strings.HasPrefix(key, weeklyModelPrefix) && len(key) > len(weeklyModelPrefix):
			l.Kind, l.WindowMinutes = provider.KindWeeklyModel, weeklyMinutes
			l.Model = strings.TrimPrefix(key, weeklyModelPrefix)
		default:
			l.Kind = provider.KindOther
		}
		limits = append(limits, l)
	}
	sort.Slice(limits, func(i, j int) bool {
		ri, rj := kindRank(limits[i].Kind), kindRank(limits[j].Kind)
		if ri != rj {
			return ri < rj
		}
		return limits[i].ID < limits[j].ID
	})
	return limits, nil
}

// decodeWindow reports whether raw is a window object with a utilization value. Nulls, arrays,
// scalars and objects without utilization (extra_usage, spend, ...) are not windows.
func decodeWindow(raw json.RawMessage) (window, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(trimmed, "{") {
		return window{}, false
	}
	var w window
	if err := json.Unmarshal(raw, &w); err != nil || w.Utilization == nil {
		return window{}, false
	}
	return w, true
}

// parseResetsAt tolerates a missing or malformed timestamp: the usage figure is still worth showing.
func parseResetsAt(key string, v *string) *time.Time {
	if v == nil || *v == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *v)
	if err != nil {
		slog.Debug("claude: unparseable resets_at", "window", key, "err", err)
		return nil
	}
	return &t
}

func kindRank(k provider.Kind) int {
	switch k {
	case provider.KindSession:
		return 0
	case provider.KindWeekly:
		return 1
	case provider.KindWeeklyModel:
		return 2
	default:
		return 3
	}
}
