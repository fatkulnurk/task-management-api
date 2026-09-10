@echo off
setlocal
where go >nul 2>nul || (echo go not found in PATH & exit /b 1)

go test ./...
if errorlevel 1 exit /b 1

echo done
