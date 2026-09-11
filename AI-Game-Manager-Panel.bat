@echo off
setlocal EnableExtensions DisableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
set "SCRIPT=%~dp0scripts\windows\AIGameManagerPanel.ps1"
if not exist "%SCRIPT%" (
  echo [ERROR] AGMP source tree is incomplete.
  echo [ERROR] Missing: scripts\windows\AIGameManagerPanel.ps1
  echo [HINT] Fully extract the source package or run AGMP-Sync.bat first.
  pause
  exit /b 2
)
where pwsh.exe >nul 2>nul
if not errorlevel 1 (
  pwsh.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT%" %*
  set "RC=%ERRORLEVEL%"
  if not "%RC%"=="0" pause
  exit /b %RC%
)
where powershell.exe >nul 2>nul
if errorlevel 1 (
  echo [ERROR] PowerShell was not found. Windows PowerShell 5.1 or PowerShell 7 is required.
  pause
  exit /b 90
)
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT%" %*
set "RC=%ERRORLEVEL%"
if not "%RC%"=="0" pause
exit /b %RC%
