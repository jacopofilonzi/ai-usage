#!/bin/sh
# Builds AI Usage for macOS (run on a Mac): bin/ai-usage.app and a bin/ai-usage.dmg.
# Needs Xcode command line tools (cgo), Go, Node and the wails3 CLI.
# The app is ad-hoc signed only: on another Mac, open it with right-click > Open the first time.
set -eu
cd "$(dirname "$0")/.."
wails3 task darwin:package:dmg
echo
echo "App and disk image are in bin/"
