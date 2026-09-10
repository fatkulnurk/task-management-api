@echo off
setlocal
where docker >nul 2>nul || (echo docker not found in PATH & exit /b 1)

docker run --rm -v "%CD%:/src" -w /src -v taskmanagement-gomod:/go/pkg/mod golang:1.27 go test -race ./...
if errorlevel 1 exit /b 1

echo done
