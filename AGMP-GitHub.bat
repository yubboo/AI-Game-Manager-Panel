@echo off
setlocal EnableExtensions
set "SCRIPT=%~dp0push-agmp.ps1"
if not exist "%SCRIPT%" (
  echo [ERROR] push-agmp.ps1 was not found next to this BAT file.
  echo.
  pause
  exit /b 1
)
powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%SCRIPT%"
set "RC=%ERRORLEVEL%"
if "%RC%"=="0" (
  echo.
  echo [OK] AGMP GitHub helper finished successfully.
  echo This window will stay open so you can review the result.
) else (
  echo.
  echo [ERROR] AGMP GitHub helper exited with code %RC%.
  echo Please keep this window open and send the red PowerShell error if needed.
)
echo.
pause
exit /b %RC%
