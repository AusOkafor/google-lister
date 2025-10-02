@echo off
echo Testing Shopify Integration...
echo.

echo 1. Testing server health...
curl -X GET http://localhost:8081/api/v1/products
echo.
echo.

echo 2. Testing Shopify OAuth installation...
curl -X POST http://localhost:8081/api/v1/shopify/install ^
  -H "Content-Type: application/json" ^
  -d "{\"shop_domain\": \"test-shop\", \"redirect_uri\": \"http://localhost:8081/api/v1/shopify/callback\"}"
echo.
echo.

echo 3. Testing Shopify webhook...
curl -X POST http://localhost:8081/api/v1/shopify/webhook ^
  -H "Content-Type: application/json" ^
  -H "X-Shopify-Topic: products/create" ^
  -H "X-Shopify-Shop-Domain: test-shop.myshopify.com" ^
  -d "{\"id\": 123, \"title\": \"Test Product\", \"body_html\": \"<p>Test Description</p>\", \"vendor\": \"Test Vendor\", \"product_type\": \"Test Type\", \"handle\": \"test-product\", \"status\": \"active\", \"tags\": \"test, sample\", \"variants\": [{\"id\": 456, \"product_id\": 123, \"title\": \"Default Title\", \"price\": \"29.99\", \"sku\": \"TEST-SKU-001\", \"position\": 1, \"inventory_quantity\": 10, \"weight\": 0.5, \"weight_unit\": \"kg\"}], \"images\": [{\"id\": 789, \"product_id\": 123, \"src\": \"https://example.com/image.jpg\", \"alt\": \"Test Product Image\"}], \"options\": [{\"id\": 101, \"product_id\": 123, \"name\": \"Size\", \"position\": 1, \"values\": [\"Small\", \"Medium\", \"Large\"]}], \"created_at\": \"2025-01-02T10:00:00Z\", \"updated_at\": \"2025-01-02T10:00:00Z\"}"
echo.
echo.

echo Testing complete!
pause
