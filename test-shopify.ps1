Write-Host "Testing Shopify Integration..." -ForegroundColor Green
Write-Host ""

Write-Host "1. Testing server health..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/products" -Method GET
    Write-Host "✅ Server is running!" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Server not responding: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

Write-Host "2. Testing Shopify OAuth installation..." -ForegroundColor Yellow
try {
    $body = @{
        shop_domain = "test-shop"
        redirect_uri = "http://localhost:8081/api/v1/shopify/callback"
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/shopify/install" -Method POST -Body $body -ContentType "application/json"
    Write-Host "✅ OAuth installation endpoint working!" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ OAuth installation failed: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

Write-Host "3. Testing Shopify webhook..." -ForegroundColor Yellow
try {
    $webhookBody = @{
        id = 123
        title = "Test Product"
        body_html = "<p>Test Description</p>"
        vendor = "Test Vendor"
        product_type = "Test Type"
        handle = "test-product"
        status = "active"
        tags = "test, sample"
        variants = @(
            @{
                id = 456
                product_id = 123
                title = "Default Title"
                price = "29.99"
                sku = "TEST-SKU-001"
                position = 1
                inventory_quantity = 10
                weight = 0.5
                weight_unit = "kg"
            }
        )
        images = @(
            @{
                id = 789
                product_id = 123
                src = "https://example.com/image.jpg"
                alt = "Test Product Image"
            }
        )
        options = @(
            @{
                id = 101
                product_id = 123
                name = "Size"
                position = 1
                values = @("Small", "Medium", "Large")
            }
        )
        created_at = "2025-01-02T10:00:00Z"
        updated_at = "2025-01-02T10:00:00Z"
    } | ConvertTo-Json -Depth 5

    $headers = @{
        "Content-Type" = "application/json"
        "X-Shopify-Topic" = "products/create"
        "X-Shopify-Shop-Domain" = "test-shop.myshopify.com"
    }

    $response = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/shopify/webhook" -Method POST -Body $webhookBody -Headers $headers
    Write-Host "✅ Webhook endpoint working!" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 3
} catch {
    Write-Host "❌ Webhook test failed: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

Write-Host "4. Testing product sync..." -ForegroundColor Yellow
try {
    # First, let's check if we have any connectors
    $connectors = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/connectors" -Method GET
    if ($connectors.data -and $connectors.data.Count -gt 0) {
        $connectorId = $connectors.data[0].id
        Write-Host "Found connector: $connectorId" -ForegroundColor Cyan
        
        $response = Invoke-RestMethod -Uri "http://localhost:8081/api/v1/shopify/$connectorId/sync" -Method POST
        Write-Host "✅ Product sync working!" -ForegroundColor Green
        $response | ConvertTo-Json -Depth 3
    } else {
        Write-Host "⚠️ No connectors found. Create a connector first." -ForegroundColor Yellow
    }
} catch {
    Write-Host "❌ Product sync failed: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host ""

Write-Host "Testing complete!" -ForegroundColor Green
Read-Host "Press Enter to continue"
