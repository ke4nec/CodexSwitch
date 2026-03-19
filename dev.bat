@echo off
setlocal
cd /d "%~dp0"

set "CHECK_ONLY=0"
set "WAILS_ARGS="

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="-CheckOnly" (
  set "CHECK_ONLY=1"
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

if "%CHECK_ONLY%"=="1" (
  echo [STEP] dev checks complete
  exit /b 0
)

call wails version >nul 2>&1 || (
  echo [ERROR] Missing wails. Install with:
  echo         go install github.com/wailsapp/wails/v2/cmd/wails@latest
  exit /b 1
)

echo [STEP] wails dev%WAILS_ARGS%
call wails dev%WAILS_ARGS%
exit /b %ERRORLEVEL%
