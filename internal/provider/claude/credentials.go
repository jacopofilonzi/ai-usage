package claude

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jacopofilonzi/ai-usage/internal/provider"
)

const credentialsFile = ".credentials.json"

// credentials is the part of Claude Code's credentials JSON this package needs. The refresh token
// is deliberately not decoded: it is never used.
type credentials struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   int64  `json:"expiresAt"` // ms since epoch; 0 when absent
}

// expired reports whether the token is past its expiry. A missing expiry is treated as valid and
// left to the server to judge.
func (c credentials) expired(now time.Time) bool {
	return c.ExpiresAt > 0 && !now.Before(time.UnixMilli(c.ExpiresAt))
}

func (p *Provider) loadCredentials(dir string) (credentials, error) {
	raw, err := p.readCredentials(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return credentials{}, fmt.Errorf("claude: no credentials: %w", provider.ErrNotLoggedIn)
	}
	if err != nil {
		return credentials{}, fmt.Errorf("claude: read credentials: %w", err)
	}
	return parseCredentials(raw)
}

// parseCredentials decodes the credentials JSON. Decoding errors are reported without the input,
// which contains secrets.
func parseCredentials(raw []byte) (credentials, error) {
	var doc struct {
		ClaudeAiOauth *credentials `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return credentials{}, fmt.Errorf("claude: credentials are not valid JSON (offset %d)", syntaxErr.Offset)
		}
		return credentials{}, errors.New("claude: credentials have an unexpected shape")
	}
	if doc.ClaudeAiOauth == nil || doc.ClaudeAiOauth.AccessToken == "" {
		return credentials{}, fmt.Errorf("claude: credentials have no OAuth token: %w", provider.ErrNotLoggedIn)
	}
	return *doc.ClaudeAiOauth, nil
}

// readCredentialsFile reads <dir>/.credentials.json, used on every OS (and as the macOS fallback).
func readCredentialsFile(dir string) ([]byte, error) {
	raw, err := os.ReadFile(filepath.Join(dir, credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", credentialsFile, err)
	}
	return raw, nil
}
