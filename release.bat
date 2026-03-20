@echo off
setlocal
cd /d "%~dp0"

set "SKIP_WAILS=0"
set "WAILS_ARGS="

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="-SkipWails" (
  set "SKIP_WAILS=1"
) else (
  set "WAILS_ARGS=%WAILS_ARGS% %~1"
)
shift
goto parse_args

:args_done
call go version >nul 2>&1 || (
  echo [ERROR] Missing go. Install Go 1.26+ and ensure it is on PATH.
  exit /b 1
)

where fnm >nul 2>&1 && (
  echo [STEP] fnm use --install-if-missing
  FOR /f "tokens=*" %%z IN ('fnm env --use-on-cd') DO CALL %%z
  call fnm use --install-if-missing || exit /b 1
)

call node -v >nul 2>&1 || (
  echo [ERROR] Missing node. Install Node.js 22 and ensure it is on PATH.
  exit /b 1
)

for /f "usebackq delims=" %%v in (`node -v`) do set "NODE_VERSION=%%v"
for /f "tokens=1 delims=." %%m in ("%NODE_VERSION:v=%") do set "NODE_MAJOR=%%m"
if not "%NODE_MAJOR%"=="22" (
  echo [ERROR] Node.js 22 is required for this project. Current version: %NODE_VERSION%
  echo         Hint: install fnm and keep the repo-level .node-version file.
  exit /b 1
)

call npm -v >nul 2>&1 || (
  echo [ERROR] Missing npm. Install Node.js 22+ and ensure it is on PATH.
  exit /b 1
)

if not exist "frontend\node_modules" (
  echo [STEP] npm install
  pushd "frontend"
  call npm install || (
    popd
    exit /b 1
  )
  popd
) else (
  echo [STEP] frontend dependencies already present
)

echo [STEP] go test ./...
call go test ./... || exit /b 1

echo [STEP] go build ./...
call go build ./... || exit /b 1

echo [STEP] npm run build
pushd "frontend"
call npm run build || (
  popd
  exit /b 1
)
popd

if "%SKIP_WAILS%"=="1" (
  echo [STEP] skip wails release build
  exit /b 0
)

call wails version >nul 2>&1 || (
  echo [ERROR] Missing wails. Install with:
  echo         go install github.com/wailsapp/wails/v2/cmd/wails@latest
  exit /b 1
)

echo [STEP] wails build -clean%WAILS_ARGS%
call wails build -clean%WAILS_ARGS%
exit /b %ERRORLEVEL%
