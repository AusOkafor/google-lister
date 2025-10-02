@echo off
echo ========================================
echo Lister - Next-Gen Product Listing Platform
echo ========================================
echo.

echo Setting up development environment...

echo.
echo 1. Starting Docker services...
call scripts\docker-up.bat
if %errorlevel% neq 0 (
    echo Failed to start Docker services
    echo Please make sure Docker Desktop is running
    pause
    exit /b 1
)

echo.
echo 2. Installing Go dependencies...
call scripts\deps.bat
if %errorlevel% neq 0 (
    echo Failed to install dependencies
    pause
    exit /b 1
)

echo.
echo 3. Setting up environment...
if not exist .env (
    copy env.example .env
    echo Created .env file from template
    echo Please edit .env with your configuration
)

echo.
echo ========================================
echo Setup completed successfully!
echo ========================================
echo.
echo Available services:
echo - API: http://localhost:8080
echo - Adminer (DB admin): http://localhost:8080
echo - Redis Commander: http://localhost:8081
echo.
echo Next steps:
echo 1. Edit .env file with your configuration
echo 2. Run: scripts\run-api.bat
echo 3. In another terminal: scripts\run-worker.bat
echo.
pause
