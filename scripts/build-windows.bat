@echo off
rem Builds AI Usage for Windows and its per-user NSIS installer:
rem   bin\ai-usage.exe and bin\ai-usage-amd64-installer.exe
rem Needs Go, Node, the wails3 CLI and NSIS (makensis; found in its default install folder too).
setlocal
cd /d "%~dp0.."

where makensis >nul 2>nul
if errorlevel 1 (
  if exist "%ProgramFiles(x86)%\NSIS\makensis.exe" (
    set "PATH=%PATH%;%ProgramFiles(x86)%\NSIS"
  ) else if exist "%ProgramFiles%\NSIS\makensis.exe" (
    set "PATH=%PATH%;%ProgramFiles%\NSIS"
  ) else (
    echo makensis not found: install NSIS ^(winget install NSIS.NSIS^)
    exit /b 1
  )
)

rem Per-user install: no UAC prompt, and it matches start at login, which is per user too.
wails3 task package INSTALL_SCOPE=user
if errorlevel 1 exit /b 1
echo.
echo Installer: bin\ai-usage-amd64-installer.exe
