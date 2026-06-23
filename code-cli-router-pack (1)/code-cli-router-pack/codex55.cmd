@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0code-router.ps1" -Profile codex55 %*
