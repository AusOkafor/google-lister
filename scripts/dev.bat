@echo off
echo Starting development environment...

echo Starting Docker services...
call scripts\docker-up.bat
if %errorlevel% neq 0 (
    echo Failed to start Docker services
    exit /b 1
)

echo.
echo Installing dependencies...
call scripts\deps.bat
if %errorlevel% neq 0 (
    echo Failed to install dependencies
    exit /b 1
)

echo.
echo Development environment ready!
echo.
echo Available services:
echo - API: http://localhost:8080
echo - Adminer (DB admin): http://localhost:8080
echo - Redis Commander: http://localhost:8081
echo.
echo To run the API server: scripts\run-api.bat
echo To run the worker: scripts\run-worker.bat
