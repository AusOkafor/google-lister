@echo off
echo Installing Go dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo Failed to tidy modules
    exit /b 1
)

go mod download
if %errorlevel% neq 0 (
    echo Failed to download modules
    exit /b 1
)

echo Dependencies installed successfully!
