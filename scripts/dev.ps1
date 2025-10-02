Write-Host "Starting development environment..." -ForegroundColor Green

Write-Host "Starting Docker services..." -ForegroundColor Yellow
& "$PSScriptRoot\docker-up.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to start Docker services" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Installing dependencies..." -ForegroundColor Yellow
& "$PSScriptRoot\deps.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to install dependencies" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Development environment ready!" -ForegroundColor Green
Write-Host ""
Write-Host "Available services:" -ForegroundColor Cyan
Write-Host "- API: http://localhost:8080" -ForegroundColor White
Write-Host "- Adminer (DB admin): http://localhost:8080" -ForegroundColor White
Write-Host "- Redis Commander: http://localhost:8081" -ForegroundColor White
Write-Host ""
Write-Host "To run the API server: .\scripts\run-api.ps1" -ForegroundColor Yellow
Write-Host "To run the worker: .\scripts\run-worker.ps1" -ForegroundColor Yellow
