Write-Host "Building Lister application..." -ForegroundColor Green

if (!(Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin"
}

Write-Host "Building API server..." -ForegroundColor Yellow
go build -o bin/api.exe cmd/api/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build API server" -ForegroundColor Red
    exit 1
}

Write-Host "Building worker..." -ForegroundColor Yellow
go build -o bin/worker.exe cmd/worker/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build worker" -ForegroundColor Red
    exit 1
}

Write-Host "Build completed successfully!" -ForegroundColor Green
Write-Host "API server: bin/api.exe" -ForegroundColor Cyan
Write-Host "Worker: bin/worker.exe" -ForegroundColor Cyan
