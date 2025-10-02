Write-Host "Installing Go dependencies..." -ForegroundColor Green

Write-Host "Tidying modules..." -ForegroundColor Yellow
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to tidy modules" -ForegroundColor Red
    exit 1
}

Write-Host "Downloading modules..." -ForegroundColor Yellow
go mod download
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to download modules" -ForegroundColor Red
    exit 1
}

Write-Host "Dependencies installed successfully!" -ForegroundColor Green
