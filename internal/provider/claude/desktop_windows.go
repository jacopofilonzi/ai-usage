package claude

import (
	"log/slog"
	"os"
	"path/filepath"
)

// readDesktopHistory reads the Claude desktop app's plan usage history. The Microsoft Store /
// MSIX build keeps its data in %LOCALAPPDATA%\Claude-Data\<package dir>\; older or non-packaged
// installs use %APPDATA%\Claude. When several exist, the most recently written file wins.
func readDesktopHistory() ([]byte, error) {
	var paths []string
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		root := filepath.Join(local, "Claude-Data")
		entries, err := os.ReadDir(root)
		if err != nil && !os.IsNotExist(err) {
			slog.Debug("claude: list desktop data dir", "err", err)
		}
		for _, e := range entries {
			if e.IsDir() {
				paths = append(paths, filepath.Join(root, e.Name(), desktopHistoryFile))
			}
		}
	}
	if roaming := os.Getenv("APPDATA"); roaming != "" {
		paths = append(paths, filepath.Join(roaming, "Claude", desktopHistoryFile))
	}
	return newestFile(paths)
}
