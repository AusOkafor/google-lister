Write-Host "Starting Docker services..." -ForegroundColor Green
docker-compose up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to start Docker services" -ForegroundColor Red
    exit 1
}

Write-Host "Docker services started successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Services available at:" -ForegroundColor Cyan
Write-Host "- API: http://localhost:8080" -ForegroundColor White
Write-Host "- Adminer (DB admin): http://localhost:8080" -ForegroundColor White
Write-Host "- Redis Commander: http://localhost:8081" -ForegroundColor White
