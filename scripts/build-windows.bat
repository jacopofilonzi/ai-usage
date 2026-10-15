@echo off
rem Builds AI Usage for Windows into dist\windows\:
rem   ai-usage-<version>-windows-amd64.exe        standalone executable (no install needed)
rem   ai-usage-<version>-windows-amd64-setup.exe  per-user installer (no admin rights)
rem Only Go must be installed beforehand: Node.js and NSIS are installed with winget when missing,
rem the wails3 CLI with `go install`, the frontend packages with npm.
setlocal EnableExtensions
cd /d "%~dp0.."

where go >nul 2>nul || (echo Go is required: https://go.dev/dl/ & exit /b 1)

rem --- Node.js ---
where npm >nul 2>nul
if errorlevel 1 (
  call :winget OpenJS.NodeJS.LTS || exit /b 1
  set "PATH=%PATH%;%ProgramFiles%\nodejs"
)
where npm >nul 2>nul || (echo Node.js was installed: open a new terminal and run this script again. & exit /b 1)

rem --- NSIS (makensis) ---
call :findnsis
if errorlevel 1 (
  call :winget NSIS.NSIS || exit /b 1
  call :findnsis || (echo NSIS was installed but makensis was not found. & exit /b 1)
)

rem --- wails3 CLI, same version as go.mod ---
for /f "delims=" %%v in ('go list -m -f "{{.Version}}" github.com/wailsapp/wails/v3') do set "WAILS_VERSION=%%v"
for /f "delims=" %%p in ('go env GOPATH') do set "PATH=%PATH%;%%p\bin"
set "HAVE_WAILS="
for /f "delims=" %%v in ('wails3 version 2^>^&1') do if "%%v"=="%WAILS_VERSION%" set "HAVE_WAILS=1"
if not defined HAVE_WAILS (
  echo Installing wails3 %WAILS_VERSION%...
  go install github.com/wailsapp/wails/v3/cmd/wails3@%WAILS_VERSION% || exit /b 1
)

rem --- build: executable + installer (npm install runs inside the task) ---
rem Per-user install: no UAC prompt, and it matches start at login, which is per user too.
wails3 task package INSTALL_SCOPE=user || exit /b 1

for /f "delims=" %%v in ('node -p "require('fs').readFileSync('build/config.yml','utf8').match(/info:[^]*?version: .([0-9.]+)/)[1]"') do set "VERSION=%%v"
if not exist dist\windows mkdir dist\windows
copy /y bin\ai-usage.exe "dist\windows\ai-usage-%VERSION%-windows-amd64.exe" >nul || exit /b 1
copy /y bin\ai-usage-amd64-installer.exe "dist\windows\ai-usage-%VERSION%-windows-amd64-setup.exe" >nul || exit /b 1
echo.
echo Done: dist\windows\
dir /b dist\windows
exit /b 0

:winget
where winget >nul 2>nul || (echo winget is missing: install %1 by hand. & exit /b 1)
echo Installing %1...
winget install -e --id %1 --accept-source-agreements --accept-package-agreements
exit /b %errorlevel%

:findnsis
where makensis >nul 2>nul && exit /b 0
if exist "%ProgramFiles(x86)%\NSIS\makensis.exe" (set "PATH=%PATH%;%ProgramFiles(x86)%\NSIS" & exit /b 0)
if exist "%ProgramFiles%\NSIS\makensis.exe" (set "PATH=%PATH%;%ProgramFiles%\NSIS" & exit /b 0)
exit /b 1
