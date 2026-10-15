#!/bin/sh
# Builds AI Usage for Linux (run on Linux, for its own architecture) into dist/linux/:
#   ai-usage-<version>-linux-<arch>             standalone binary (needs GTK 4 + WebKitGTK 6.0 installed)
#   ai-usage-<version>-linux-<arch>.AppImage    portable, runs without installing
#   .deb, .rpm, Arch package                    installers
# Only Go must be installed beforehand: the C toolchain, GTK 4 / WebKitGTK 6.0 dev packages and
# Node.js are installed with the system package manager (apt, dnf or pacman, via sudo), the wails3
# CLI with `go install`, the frontend packages with npm.
set -eu
cd "$(dirname "$0")/.."

command -v go >/dev/null || { echo "Go is required: https://go.dev/dl/"; exit 1; }

SUDO=""
[ "$(id -u)" -ne 0 ] && SUDO=sudo
if command -v apt-get >/dev/null; then
  $SUDO apt-get update
  $SUDO apt-get install -y build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev nodejs npm file
elif command -v dnf >/dev/null; then
  $SUDO dnf install -y gcc pkgconf-pkg-config gtk4-devel webkitgtk6.0-devel nodejs npm file
elif command -v pacman >/dev/null; then
  $SUDO pacman -S --needed --noconfirm base-devel pkgconf gtk4 webkitgtk-6.0 nodejs npm file
else
  echo "Unknown package manager: install a C compiler, pkg-config, GTK 4 and WebKitGTK 6.0 dev packages"
  echo "and Node.js by hand (\`wails3 doctor\` lists what is missing), then run this script again."
  command -v npm >/dev/null || exit 1
fi

# wails3 CLI, same version as go.mod (it prints its version on stderr).
WAILS_VERSION=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v3)
PATH="$PATH:$(go env GOPATH)/bin"
if [ "$(wails3 version 2>&1 || true)" != "$WAILS_VERSION" ]; then
  echo "Installing wails3 $WAILS_VERSION..."
  go install "github.com/wailsapp/wails/v3/cmd/wails3@$WAILS_VERSION"
fi

# Builds bin/ai-usage, then the AppImage, .deb, .rpm and Arch packages.
wails3 task package

VERSION=$(node -p "require('fs').readFileSync('build/config.yml','utf8').match(/info:[^]*?version: .([0-9.]+)/)[1]")
ARCH=$(uname -m)
OUT=dist/linux
NAME="ai-usage-$VERSION-linux-$ARCH"
mkdir -p "$OUT"
cp bin/ai-usage "$OUT/$NAME"
for f in bin/*.AppImage; do [ -e "$f" ] && cp "$f" "$OUT/$NAME.AppImage"; done
for f in bin/*.deb bin/*.rpm bin/*.pkg.tar.zst; do [ -e "$f" ] && cp "$f" "$OUT/"; done
echo
echo "Done: $OUT/"
ls -1 "$OUT"
