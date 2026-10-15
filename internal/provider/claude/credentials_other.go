//go:build !darwin

package claude

// readCredentials reads the credentials file: outside macOS Claude Code keeps it only on disk.
func readCredentials(dir string) ([]byte, error) {
	return readCredentialsFile(dir)
}
