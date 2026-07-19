@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start-Demo.ps1" -Restart
if errorlevel 1 (
  echo.
  echo Startup failed. Keep the error above and check the logs folder.
  pause
  exit /b 1
)
endlocal
