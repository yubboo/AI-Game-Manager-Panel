@echo off
setlocal EnableExtensions
cd /d "%~dp0"
echo ================================================================
echo   AGMP 0.4.0 - TypeScript Agent + Rust Native
echo ================================================================
where node.exe >nul 2>nul || (echo [ERROR] Node.js 22+ not found.& pause& exit /b 1)
where cargo.exe >nul 2>nul || (echo [ERROR] Rust Cargo not found. Install Rust before local source launch.& pause& exit /b 1)
if not exist "crates\target\release\agmp-native.exe" (
  echo [1/3] Building Rust Native Runtime...
  cargo build --manifest-path crates\Cargo.toml --release || (pause& exit /b 1)
)
if not exist "node_modules\typescript\package.json" (
  echo [2/3] Installing TypeScript Core dependencies...
  call npm install || (pause& exit /b 1)
)
echo [3/3] Starting AGMP TypeScript Core at http://127.0.0.1:32123
start "" http://127.0.0.1:32123
node --experimental-strip-types apps\server\src\main.ts
pause
