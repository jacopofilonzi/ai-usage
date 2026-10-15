#!/bin/sh
# Builds AI Usage for Linux (run on Linux): the binary plus AppImage, .deb, .rpm and Arch packages in bin/.
# Needs a C compiler and the GTK/WebKitGTK dev packages (cgo), Go, Node and the wails3 CLI;
# `wails3 doctor` lists what is missing.
set -eu
cd "$(dirname "$0")/.."
wails3 task package
echo
echo "Packages are in bin/"
