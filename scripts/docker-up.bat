@echo off
echo Starting Docker services...
docker-compose up -d
if %errorlevel% neq 0 (
    echo Failed to start Docker services
    exit /b 1
)

echo Docker services started successfully!
echo.
echo Services available at:
echo - API: http://localhost:8080
echo - Adminer (DB admin): http://localhost:8080
echo - Redis Commander: http://localhost:8081
