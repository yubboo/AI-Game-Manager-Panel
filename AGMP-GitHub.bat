@echo off
setlocal EnableExtensions
set "SCRIPT=%~dp0push-agmp.ps1"
if not exist "%SCRIPT%" (
  echo [ERROR] push-agmp.ps1 was not found next to this BAT file.
  pause
  exit /b 1
)
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT%"
exit /b %ERRORLEVEL%
