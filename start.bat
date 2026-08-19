@echo off
chcp 65001 >nul
echo 正在启动 VIBE，请稍候...
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0start.ps1" %*
echo.
pause
