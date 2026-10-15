package claude

import (
	"bytes"
	"context"
	"log/slog"
	"os/exec"
	"time"
)

// keychainService is the generic-password service name Claude Code uses on macOS.
const keychainService = "Claude Code-credentials"

// readCredentials prefers the Keychain, where Claude Code stores its token on macOS, and falls back
// to the file (older versions, or CLAUDE_CONFIG_DIR setups that still write it).
func readCredentials(dir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// -w prints only the password (the JSON). Stderr is discarded: it never holds the secret but
	// is not useful either; the exit status is enough to decide on the fallback.
	out, err := exec.CommandContext(ctx, "security", "find-generic-password", "-s", keychainService, "-w").Output()
	if err == nil {
		if raw := bytes.TrimSpace(out); len(raw) > 0 {
			return raw, nil
		}
	} else {
		// Exit status 44 means "item not found", the normal case when the file is used instead.
		slog.Debug("claude: keychain lookup failed, falling back to file", "err", err)
	}
	return readCredentialsFile(dir)
}
