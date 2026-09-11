@echo off
setlocal EnableExtensions DisableDelayedExpansion
chcp 65001 >nul
cd /d "%~dp0"
where pwsh.exe >nul 2>nul
if not errorlevel 1 (
  pwsh.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\windows\AIGameManagerPanel.ps1" %*
  exit /b %errorlevel%
)
where powershell.exe >nul 2>nul
if errorlevel 1 (
  echo [ERROR] PowerShell was not found. Windows PowerShell 5.1 or PowerShell 7 is required.
  exit /b 90
)
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\windows\AIGameManagerPanel.ps1" %*
exit /b %errorlevel%
