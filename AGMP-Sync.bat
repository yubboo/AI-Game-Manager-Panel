@echo off
setlocal EnableExtensions
set "SCRIPT=%~dp0sync-agmp.ps1"
if not exist "%SCRIPT%" (
  echo [ERROR] sync-agmp.ps1 was not found next to this BAT file.
  echo.
  pause
  exit /b 1
)
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT%"
set "RC=%ERRORLEVEL%"
echo.
if not "%RC%"=="0" (
  echo [ERROR] AGMP Sync helper exited with code %RC%.
  echo Please keep this window open and send the red PowerShell error if needed.
) else (
  echo [OK] AGMP Sync helper finished.
)
echo.
pause
exit /b %RC%
