// Package config persists user preferences as JSON in <user config dir>/ai-usage/config.json.
//
// Only preferences live here, never runtime state (the wake lock always starts inactive, start at
// login is read from the OS). The file is durable user data: field names stay stable, and unknown,
// missing or invalid fields fall back to the defaults instead of failing.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const AppDir = "ai-usage"

type Corner string

const (
	TopLeft     Corner = "top-left"
	TopRight    Corner = "top-right"
	BottomLeft  Corner = "bottom-left"
	BottomRight Corner = "bottom-right"
)

// Monitor values besides a screen ID.
const (
	MonitorPrimary = "primary"
	MonitorAll     = "all"
)

// Language values besides a language code ("en", "it").
const LanguageSystem = "system"

type Config struct {
	Provider     string `json:"provider"`      // Provider.ID(); "" = first detected
	Corner       Corner `json:"corner"`        // widget corner (Windows)
	AlwaysShow   bool   `json:"always_show"`   // widget always drawn; otherwise revealed by the hot corner (Windows)
	Monitor      string `json:"monitor"`       // MonitorPrimary, MonitorAll or a screen ID (Windows)
	Language     string `json:"language"`      // LanguageSystem, "en", "it"
	PollSeconds  int    `json:"poll_seconds"`  // 0 = provider default
	KeepUnlocked bool   `json:"keep_unlocked"` // wake lock also keeps the display on
}

func Default() Config {
	return Config{Corner: BottomRight, Monitor: MonitorPrimary, Language: LanguageSystem}
}

// Normalize replaces invalid values with the defaults.
func (c Config) Normalize() Config {
	d := Default()
	switch c.Corner {
	case TopLeft, TopRight, BottomLeft, BottomRight:
	default:
		c.Corner = d.Corner
	}
	if c.Monitor == "" {
		c.Monitor = d.Monitor
	}
	switch c.Language {
	case LanguageSystem, "en", "it":
	default:
		c.Language = d.Language
	}
	if c.PollSeconds < 0 {
		c.PollSeconds = 0
	} else if c.PollSeconds > 0 && c.PollSeconds < 30 {
		c.PollSeconds = 30 // the Claude endpoint is rate limited
	}
	return c
}

// DefaultPath returns the config file location for the current user.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config dir: %w", err)
	}
	return filepath.Join(dir, AppDir, "config.json"), nil
}

// Load reads the config at path; a missing file yields the defaults.
func Load(path string) (Config, error) {
	c := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return Default(), fmt.Errorf("parse config %s: %w", path, err)
	}
	return c.Normalize(), nil
}

// Save writes the config atomically (temp file + rename) so a crash never leaves it half written.
func Save(path string, c Config) error {
	data, err := json.MarshalIndent(c.Normalize(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*.json")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}
