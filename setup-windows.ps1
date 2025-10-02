Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Lister - Next-Gen Product Listing Platform" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Setting up development environment..." -ForegroundColor Green

Write-Host ""
Write-Host "1. Starting Docker services..." -ForegroundColor Yellow
& "$PSScriptRoot\scripts\docker-up.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to start Docker services" -ForegroundColor Red
    Write-Host "Please make sure Docker Desktop is running" -ForegroundColor Yellow
    Read-Host "Press Enter to continue"
    exit 1
}

Write-Host ""
Write-Host "2. Installing Go dependencies..." -ForegroundColor Yellow
& "$PSScriptRoot\scripts\deps.ps1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to install dependencies" -ForegroundColor Red
    Read-Host "Press Enter to continue"
    exit 1
}

Write-Host ""
Write-Host "3. Setting up environment..." -ForegroundColor Yellow
if (!(Test-Path ".env")) {
    Copy-Item "env.example" ".env"
    Write-Host "Created .env file from template" -ForegroundColor Green
    Write-Host "Please edit .env with your configuration" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Setup completed successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Available services:" -ForegroundColor Cyan
Write-Host "- API: http://localhost:8080" -ForegroundColor White
Write-Host "- Adminer (DB admin): http://localhost:8080" -ForegroundColor White
Write-Host "- Redis Commander: http://localhost:8081" -ForegroundColor White
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Edit .env file with your configuration" -ForegroundColor White
Write-Host "2. Run: .\scripts\run-api.ps1" -ForegroundColor White
Write-Host "3. In another terminal: .\scripts\run-worker.ps1" -ForegroundColor White
Write-Host ""
Read-Host "Press Enter to continue"
