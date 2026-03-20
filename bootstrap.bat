@echo off
setlocal
cd /d "%~dp0"

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
