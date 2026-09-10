@echo off
setlocal
where docker >nul 2>nul || (echo docker not found in PATH & exit /b 1)

docker run --rm --network taskmanagement_default -e "TEST_DATABASE_URL=root:rootpassword@tcp(mysql:3306)/?multiStatements=true" -v "%CD%:/src" -w /src -v taskmanagement-gomod:/go/pkg/mod golang:1.27 go test -race -tags=integration ./internal/modules/tasks/repository/...
if errorlevel 1 exit /b 1

echo done
