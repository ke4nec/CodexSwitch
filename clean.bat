@echo off
setlocal
cd /d "%~dp0"

set "REMOVE_ALL=0"

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="-All" (
  set "REMOVE_ALL=1"
)
shift
goto parse_args

:args_done
echo [STEP] cleaning generated artifacts

if exist "build" (
  echo [STEP] remove build
  rmdir /s /q "build"
)

if exist "frontend\dist" (
  echo [STEP] remove frontend\dist
  rmdir /s /q "frontend\dist"
)

if exist "frontend\wailsjs" (
  echo [STEP] remove frontend\wailsjs
  rmdir /s /q "frontend\wailsjs"
)

if exist "frontend\.vite" (
  echo [STEP] remove frontend\.vite
  rmdir /s /q "frontend\.vite"
)

if exist "coverage.out" (
  echo [STEP] remove coverage.out
  del /f /q "coverage.out"
)

if exist "frontend\package.json.md5" (
  echo [STEP] remove frontend\package.json.md5
  del /f /q "frontend\package.json.md5"
)

if "%REMOVE_ALL%"=="1" (
  if exist "frontend\node_modules" (
    echo [STEP] remove frontend\node_modules
    rmdir /s /q "frontend\node_modules"
  )
)

echo [STEP] clean complete
exit /b 0
