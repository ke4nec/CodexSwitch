@echo off
setlocal
cd /d "%~dp0"

call go version >nul 2>&1 || (
  echo [ERROR] Missing go. Install Go 1.26+ and ensure it is on PATH.
  exit /b 1
)

call npm -v >nul 2>&1 || (
  echo [ERROR] Missing npm. Install Node.js 22+ and ensure it is on PATH.
  exit /b 1
)

echo [STEP] bootstrap start
call go mod tidy || exit /b 1

if /I "%~1"=="-ForceNpmInstall" goto install_frontend
if exist "frontend\node_modules" (
  echo [STEP] frontend dependencies already present
  goto done
)

:install_frontend
echo [STEP] npm install
pushd "frontend"
call npm install || (
  popd
  exit /b 1
)
popd

:done
echo [STEP] bootstrap complete
exit /b 0
