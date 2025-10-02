@echo off
echo Building Lister application...

if not exist bin mkdir bin

go build -o bin/api.exe cmd/api/main.go
if %errorlevel% neq 0 (
    echo Failed to build API server
    exit /b 1
)

go build -o bin/worker.exe cmd/worker/main.go
if %errorlevel% neq 0 (
    echo Failed to build worker
    exit /b 1
)

echo Build completed successfully!
echo API server: bin/api.exe
echo Worker: bin/worker.exe
