#!/bin/sh
# Builds AI Usage for macOS (run on a Mac, for its own architecture) into dist/macos/:
#   ai-usage-<version>-macos-<arch>.zip   the standalone app (unzip and run, or drag to Applications)
#   ai-usage-<version>-macos-<arch>.dmg   the disk image installer
# Only Go must be installed beforehand: the Xcode command line tools are requested when missing,
# Node.js comes from Homebrew, the wails3 CLI from `go install`, the frontend packages from npm.
# The app is ad-hoc signed only: on another Mac, open it with right-click > Open the first time.
set -eu
cd "$(dirname "$0")/.."

command -v go >/dev/null || { echo "Go is required: https://go.dev/dl/"; exit 1; }

# cgo needs the Xcode command line tools.
if ! xcode-select -p >/dev/null 2>&1; then
  xcode-select --install || true
  echo "Install the Xcode command line tools in the window that opened, then run this script again."
  exit 1
fi

if ! command -v npm >/dev/null; then
  command -v brew >/dev/null || { echo "Node.js is required: install it from https://nodejs.org or Homebrew (https://brew.sh)."; exit 1; }
  brew install node
fi

# wails3 CLI, same version as go.mod (it prints its version on stderr).
WAILS_VERSION=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)
PATH="$PATH:$(go env GOPATH)/bin"
if [ "$(wails3 version 2>&1 || true)" != "$WAILS_VERSION" ]; then
  echo "Installing wails3 $WAILS_VERSION..."
  go install "github.com/wailsapp/wails/v3/cmd/wails3@$WAILS_VERSION"
fi

# Builds bin/ai-usage.app, then the .dmg from it.
wails3 task darwin:package:dmg

VERSION=$(node -p "require('fs').readFileSync('build/config.yml','utf8').match(/info:[^]*?version: .([0-9.]+)/)[1]")
ARCH=$(uname -m)
OUT=dist/macos
NAME="ai-usage-$VERSION-macos-$ARCH"
mkdir -p "$OUT"
rm -f "$OUT/$NAME.zip" "$OUT/$NAME.dmg"
ditto -c -k --keepParent bin/ai-usage.app "$OUT/$NAME.zip"
cp bin/*.dmg "$OUT/$NAME.dmg"
echo
echo "Done: $OUT/"
ls -1 "$OUT"
