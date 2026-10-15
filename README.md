# AI Usage

See at a glance how much of your AI coding agent's usage limits you have left: the 5-hour and weekly
windows of Claude Code (and Codex), as rings in a corner of the screen (Windows) or in the menu bar
(macOS, Linux). It can also keep the computer awake while a long job or an agent is running.

- **One agent at a time.** Switch with the mouse wheel over the widget, the right-click menu, the tray menu or
  the settings.
- **Uses your agent's own login.** AI Usage reads the credentials the agent already keeps and never refreshes
  or stores them. If the token has expired, it shows the last known values until you open the agent again.
- **Keep awake** for 5–120 minutes or indefinitely, optionally keeping the display on.
- English and Italian.

Note: Claude Code's usage endpoint is not documented by Anthropic and may change.

## Install

Windows: run `ai-usage-<version>-windows-amd64-setup.exe` (per-user install, no admin rights needed), or
the standalone `ai-usage-<version>-windows-amd64.exe` without installing. To build them, see "Build" below.

## Use

Start `ai-usage`. On Windows a quarter-circle widget appears in the bottom-right corner: hover it for
details, scroll to change agent, right-click for the menu. On macOS and Linux the rings are the tray
icon: click it for details. Open **Settings** from the tray to choose the agent, the corner, the monitors
(one, or all), the refresh interval, start at login and the language.

## Development

Requirements: Go 1.26+, Node 24+, GNU make, the Wails CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29`).

```sh
make setup   # frontend dependencies
make dev     # run with hot reload
make check   # what must pass before a change is done
make build   # production build into bin/
```

## Build

On a new machine install only [Go](https://go.dev/dl/), then run the script for that OS: it installs what is
still missing (Node.js, the wails3 CLI, NSIS on Windows; Xcode command line tools on macOS; compiler and
GTK 4 / WebKitGTK 6.0 packages on Linux, via sudo), builds, and puts the results in `dist/<os>/`.

| OS | Command | Standalone | Installer |
| --- | --- | --- | --- |
| Windows | `scripts\build-windows.bat` | `ai-usage-<ver>-windows-amd64.exe` | `…-setup.exe` (per user, no admin) |
| macOS | `sh scripts/build-mac.sh` | `ai-usage-<ver>-macos-<arch>.zip` (the .app) | `.dmg` |
| Linux | `sh scripts/build-linux.sh` | binary and `.AppImage` | `.deb`, `.rpm`, Arch package |

Builds are not code-signed: Windows SmartScreen and macOS Gatekeeper warn on first launch.
The version comes from `info.version` in `build/config.yml`.

See [STRUCTURE.md](STRUCTURE.md) for how the code is organised and [AGENTS.md](AGENTS.md) for the rules.
