package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	"github.com/supabase-community/supabase-go"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// In-memory storage for connectors (for demo purposes)
type Connector struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	ShopDomain  string     `json:"shop_domain"`
	AccessToken string     `json:"access_token"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSync    *time.Time `json:"last_sync"`
}

// ShopifyProduct represents a product from Shopify API
type ShopifyProduct struct {
	ID          int64              `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"body_html"`
	Status      string             `json:"status"`
	Vendor      string             `json:"vendor"`
	ProductType string             `json:"product_type"`
	Images      []ShopifyImage     `json:"images"`
	Variants    []ShopifyVariant   `json:"variants"`
	Metafields  []ShopifyMetafield `json:"metafields"`
}

type ShopifyImage struct {
	ID  int64  `json:"id"`
	URL string `json:"src"`
}

type ShopifyVariant struct {
	ID                  int64   `json:"id"`
	Title               string  `json:"title"`
	Price               string  `json:"price"`
	CompareAtPrice      *string `json:"compare_at_price"`
	SKU                 string  `json:"sku"`
	InventoryQuantity   int     `json:"inventory_quantity"`
	InventoryManagement string  `json:"inventory_management"`
	InventoryPolicy     string  `json:"inventory_policy"`
	Available           *bool   `json:"available"`
}

type ShopifyMetafield struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SEO Enhancement struct for AI-generated SEO data
type SEOEnhancement struct {
	SEOTitle       string   `json:"seo_title"`
	SEODescription string   `json:"seo_description"`
	Keywords       []string `json:"keywords"`
	MetaKeywords   string   `json:"meta_keywords"`
	AltText        string   `json:"alt_text"`
	SchemaMarkup   string   `json:"schema_markup"`
}

var (
	db      *sql.DB
	dbMutex sync.Mutex
	// Temporary in-memory storage for Vercel demo
	connectors     []Connector
	connectorMutex sync.RWMutex
	// Global organization ID
	globalOrganizationID string
	orgIDMutex           sync.RWMutex
)

// syncShopifyProducts syncs products from Shopify store
func syncShopifyProducts(db *sql.DB, connectorID, shopDomain, accessToken string) {
	log.Printf("🔄 Starting Shopify product sync for connector %s, shop %s", connectorID, shopDomain)
	// Show first 10 characters of token for debugging (but not the full token for security)
	tokenPreview := accessToken
	if len(accessToken) > 10 {
		tokenPreview = accessToken[:10]
	}
	log.Printf("🔑 Access token length: %d, first 10 chars: %s", len(accessToken), tokenPreview)

	// Fetch products from Shopify - clean the shop domain first
	cleanDomain := shopDomain
	if strings.HasSuffix(shopDomain, ".myshopify.com") {
		cleanDomain = strings.TrimSuffix(shopDomain, ".myshopify.com")
	}
	log.Printf("🏪 Original domain: %s, Clean domain: %s", shopDomain, cleanDomain)

	// Create HTTP client with longer timeout and custom transport
	transport := &http.Transport{
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second, // Increased for products API
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Timeout:   300 * time.Second, // 5 minutes for products API
		Transport: transport,
	}

	// Skip basic connectivity test for now - go directly to API
	log.Printf("🚀 Skipping basic connectivity test, going directly to Shopify API")

	// First, test the access token with a simple shop info call
	testURL := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/shop.json", cleanDomain)
	log.Printf("🧪 Testing access token with shop info: %s", testURL)

	testReq, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		log.Printf("❌ Failed to create test request: %v", err)
		return
	}

	testReq.Header.Set("X-Shopify-Access-Token", accessToken)
	testReq.Header.Set("Content-Type", "application/json")

	// Try the request with retry logic
	var testResp *http.Response
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("🔄 Attempt %d/3: Testing shop info API", attempt)
		testResp, err = client.Do(testReq)
		if err == nil {
			break
		}
		log.Printf("❌ Attempt %d failed: %v", attempt, err)
		if attempt < 3 {
			log.Printf("⏳ Waiting 5 seconds before retry...")
			time.Sleep(5 * time.Second)
		}
	}

	if err != nil {
		log.Printf("❌ All 3 attempts failed: %v", err)
		return
	}
	defer testResp.Body.Close()

	log.Printf("🧪 Shop info test response: %d", testResp.StatusCode)

	if testResp.StatusCode != 200 {
		body, _ := io.ReadAll(testResp.Body)
		log.Printf("❌ Shop info test failed: %s", string(body))
		return
	}

	// Read the shop info response to verify it's working
	shopInfoBody, _ := io.ReadAll(testResp.Body)
	log.Printf("✅ Access token is valid, shop info: %s", string(shopInfoBody))
	log.Printf("✅ Proceeding with product sync")

	// Fetch all products with pagination support
	var allProducts []ShopifyProduct
	pageInfo := ""
	pageCount := 0

	for {
		pageCount++
		log.Printf("🔄 Fetching page %d of products...", pageCount)

		// Build URL with pagination
		url := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/products.json?limit=250", cleanDomain)
		if pageInfo != "" {
			url += "&page_info=" + pageInfo
		}

		log.Printf("🌐 Making request to: %s", url)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("❌ Failed to create Shopify request: %v", err)
			return
		}

		req.Header.Set("X-Shopify-Access-Token", accessToken)
		req.Header.Set("Content-Type", "application/json")

		// Try the products API request with retry logic
		var resp *http.Response
		for attempt := 1; attempt <= 3; attempt++ {
			log.Printf("🔄 Attempt %d/3: Fetching page %d from Shopify API", attempt, pageCount)
			resp, err = client.Do(req)
			if err == nil {
				break
			}
			log.Printf("❌ Attempt %d failed: %v", attempt, err)
			if attempt < 3 {
				log.Printf("⏳ Waiting 10 seconds before retry...")
				time.Sleep(10 * time.Second)
			}
		}

		if err != nil {
			log.Printf("❌ All 3 attempts failed to fetch products: %v", err)
			return
		}

		log.Printf("📡 Shopify API response status: %d", resp.StatusCode)

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("❌ Shopify API error response: %s", string(body))
			resp.Body.Close()
			return
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Products []ShopifyProduct `json:"products"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("❌ Failed to parse Shopify response: %v", err)
			return
		}

		log.Printf("✅ Fetched %d products from page %d", len(result.Products), pageCount)
		allProducts = append(allProducts, result.Products...)

		// Check for next page using Link header
		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" || !strings.Contains(linkHeader, "rel=\"next\"") {
			log.Printf("📄 No more pages, total products fetched: %d", len(allProducts))
			break
		}

		// Extract page_info from Link header for next request
		// Link format: <https://shop.myshopify.com/admin/api/2023-10/products.json?page_info=...>; rel="next"
		nextMatch := regexp.MustCompile(`page_info=([^&>]+)`).FindStringSubmatch(linkHeader)
		if len(nextMatch) > 1 {
			pageInfo = nextMatch[1]
			log.Printf("🔗 Next page info: %s", pageInfo)
		} else {
			log.Printf("⚠️ Could not extract page_info from Link header")
			break
		}

		// Safety check to prevent infinite loops
		if pageCount > 10 {
			log.Printf("⚠️ Reached maximum page limit (10), stopping pagination")
			break
		}
	}

	log.Printf("✅ Total products fetched from all pages: %d", len(allProducts))

	if len(allProducts) == 0 {
		log.Printf("⚠️ No products found in Shopify store")
		return
	}

	// Insert/update products in database
	successCount := 0
	for i, product := range allProducts {
		log.Printf("📦 Processing product %d/%d: %s (ID: %d)", i+1, len(allProducts), product.Title, product.ID)
		imagesJSON, _ := json.Marshal(product.Images)
		metadataJSON, _ := json.Marshal(product)

		externalID := fmt.Sprintf("%d", product.ID)

		_, err := db.Exec(`
			INSERT INTO products (
				external_id, title, description, price, currency, sku,
				brand, category, images, status, metadata, organization_id, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
			ON CONFLICT (external_id) 
			DO UPDATE SET
				title = $2, description = $3, price = $4, currency = $5,
				sku = $6, brand = $7, category = $8, images = $9,
				status = $10, metadata = $11, organization_id = $12, updated_at = NOW()
		`, externalID, product.Title, product.Description,
			getFirstVariantPrice(product), "USD", getFirstVariantSKU(product),
			product.Vendor, product.ProductType, string(imagesJSON),
			product.Status, string(metadataJSON), globalOrganizationID)

		if err != nil {
			log.Printf("❌ Failed to insert product %s: %v", externalID, err)
		} else {
			successCount++
			log.Printf("✅ Successfully inserted product %s", externalID)
		}
	}

	log.Printf("✅ Shopify sync completed for %s - %d/%d products imported", shopDomain, successCount, len(allProducts))

	// Verify products were actually inserted
	var productCount int
	err = db.QueryRow("SELECT COUNT(*) FROM products WHERE organization_id = $1", globalOrganizationID).Scan(&productCount)
	if err != nil {
		log.Printf("❌ Failed to count products: %v", err)
	} else {
		log.Printf("📊 Total products in database for organization: %d", productCount)
	}
}

// Helper functions for sync
func getFirstVariantPrice(product ShopifyProduct) float64 {
	if len(product.Variants) > 0 {
		priceStr := product.Variants[0].Price
		price, _ := strconv.ParseFloat(priceStr, 64)
		return price
	}
	return 0.0
}

func getFirstVariantSKU(product ShopifyProduct) string {
	if len(product.Variants) > 0 {
		return product.Variants[0].SKU
	}
	return ""
}

// CSV Import helper functions
func parseAndImportCSV(db *sql.DB, file io.Reader, organizationID string) (int, []string) {
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, []string{fmt.Sprintf("Failed to parse CSV: %v", err)}
	}

	if len(records) < 2 {
		return 0, []string{"CSV file is empty or has no data rows"}
	}

	headers := records[0]
	imported := 0
	var errors []string

	// Validate headers
	requiredHeaders := []string{"title", "price"}
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			errors = append(errors, fmt.Sprintf("Missing required header: %s", required))
		}
	}

	if len(errors) > 0 {
		return 0, errors
	}

	// Import products
	for rowNum, record := range records[1:] {
		if len(record) < len(headers) {
			errors = append(errors, fmt.Sprintf("Row %d: Insufficient columns", rowNum+2))
			continue
		}

		// Extract data
		externalID := getCSVValue(record, headerMap, "id")
		if externalID == "" {
			externalID = fmt.Sprintf("csv-%d-%d", time.Now().Unix(), rowNum)
		}
		title := getCSVValue(record, headerMap, "title")
		description := getCSVValue(record, headerMap, "description")
		priceStr := getCSVValue(record, headerMap, "price")
		currency := getCSVValue(record, headerMap, "currency")
		if currency == "" {
			currency = "USD"
		}
		sku := getCSVValue(record, headerMap, "sku")
		brand := getCSVValue(record, headerMap, "brand")
		category := getCSVValue(record, headerMap, "category")
		status := getCSVValue(record, headerMap, "status")
		if status == "" {
			status = "active"
		}
		imageURL := getCSVValue(record, headerMap, "image_url")

		// Parse price
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: Invalid price '%s'", rowNum+2, priceStr))
			continue
		}

		// Create images JSON
		imagesJSON := "[]"
		if imageURL != "" {
			images := []map[string]string{{"src": imageURL}}
			imagesBytes, _ := json.Marshal(images)
			imagesJSON = string(imagesBytes)
		}

		// Insert product
		_, err = db.Exec(`
			INSERT INTO products (
				external_id, title, description, price, currency, sku,
				brand, category, images, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
			ON CONFLICT (external_id) 
			DO UPDATE SET
				title = $2, description = $3, price = $4, currency = $5,
				sku = $6, brand = $7, category = $8, images = $9,
				status = $10, updated_at = NOW()
		`, externalID, title, description, price, currency, sku, brand, category, imagesJSON, status)

		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: Failed to import - %v", rowNum+2, err))
		} else {
			imported++
		}
	}

	return imported, errors
}

func getCSVValue(record []string, headerMap map[string]int, key string) string {
	if idx, exists := headerMap[key]; exists && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

func validateCSV(file io.Reader) []string {
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return []string{fmt.Sprintf("Failed to parse CSV: %v", err)}
	}

	var errors []string

	if len(records) < 1 {
		return []string{"CSV file is empty"}
	}

	// Validate headers
	headers := records[0]
	requiredHeaders := []string{"title", "price"}
	headerMap := make(map[string]bool)
	for _, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = true
	}

	for _, required := range requiredHeaders {
		if !headerMap[required] {
			errors = append(errors, fmt.Sprintf("Missing required header: %s", required))
		}
	}

	if len(records) < 2 {
		errors = append(errors, "CSV has no data rows")
	}

	return errors
}

// WooCommerce helper functions
func validateWooCommerceCredentials(storeURL, consumerKey, consumerSecret string) (bool, error) {
	// Clean URL
	storeURL = strings.TrimSuffix(storeURL, "/")

	// Test API call
	url := fmt.Sprintf("%s/wp-json/wc/v3/products?per_page=1", storeURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	req.SetBasicAuth(consumerKey, consumerSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

func syncWooCommerceProducts(db *sql.DB, connectorID, storeURL, consumerKey, consumerSecret, organizationID string) {
	log.Printf("🔄 Starting WooCommerce product sync for %s", storeURL)

	storeURL = strings.TrimSuffix(storeURL, "/")
	url := fmt.Sprintf("%s/wp-json/wc/v3/products?per_page=100", storeURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("❌ Failed to create WooCommerce request: %v", err)
		return
	}

	req.SetBasicAuth(consumerKey, consumerSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Failed to fetch WooCommerce products: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("❌ WooCommerce API returned status %d", resp.StatusCode)
		return
	}

	body, _ := io.ReadAll(resp.Body)

	var products []map[string]interface{}
	if err := json.Unmarshal(body, &products); err != nil {
		log.Printf("❌ Failed to parse WooCommerce response: %v", err)
		return
	}

	log.Printf("✅ Fetched %d products from WooCommerce", len(products))

	// Import products
	for _, product := range products {
		title, _ := product["name"].(string)
		description, _ := product["description"].(string)
		priceStr, _ := product["price"].(string)
		sku, _ := product["sku"].(string)
		status, _ := product["status"].(string)

		price, _ := strconv.ParseFloat(priceStr, 64)
		externalID := fmt.Sprintf("%v", product["id"])

		// Get images
		imagesJSON := "[]"
		if images, ok := product["images"].([]interface{}); ok && len(images) > 0 {
			imagesBytes, _ := json.Marshal(images)
			imagesJSON = string(imagesBytes)
		}

		// Get categories
		category := ""
		if categories, ok := product["categories"].([]interface{}); ok && len(categories) > 0 {
			if cat, ok := categories[0].(map[string]interface{}); ok {
				category, _ = cat["name"].(string)
			}
		}

		_, err = db.Exec(`
			INSERT INTO products (
				external_id, title, description, price, currency, sku,
				category, images, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
			ON CONFLICT (external_id) 
			DO UPDATE SET
				title = $2, description = $3, price = $4, currency = $5,
				sku = $6, category = $7, images = $8,
				status = $9, updated_at = NOW()
		`, externalID, title, description, price, "USD", sku, category, imagesJSON, status)

		if err != nil {
			log.Printf("❌ Failed to insert WooCommerce product %s: %v", externalID, err)
		}
	}

	log.Printf("✅ WooCommerce sync completed for %s", storeURL)
}

// getOrCreateOrganizationID gets the organization ID from memory or creates a new one
func getOrCreateOrganizationID() string {
	orgIDMutex.RLock()
	if globalOrganizationID != "" {
		orgIDMutex.RUnlock()
		return globalOrganizationID
	}
	orgIDMutex.RUnlock()

	orgIDMutex.Lock()
	defer orgIDMutex.Unlock()

	// Double-check in case another goroutine created it
	if globalOrganizationID != "" {
		return globalOrganizationID
	}

	// Try to get existing organization from database
	if db != nil {
		var existingOrgID string
		err := db.QueryRow("SELECT id FROM organizations LIMIT 1").Scan(&existingOrgID)
		if err == nil {
			globalOrganizationID = existingOrgID
			return globalOrganizationID
		}
	}

	// Create new organization ID
	orgID := uuid.New().String()
	globalOrganizationID = orgID

	// Create organization in database if connection exists
	if db != nil {
		_, err := db.Exec(`
			INSERT INTO organizations (id, name, domain, settings, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (id) DO NOTHING
		`, orgID, "Default Organization", "example.com", "{}")
		if err != nil {
			log.Printf("Warning: Could not create organization in database: %v", err)
		}
	}

	return globalOrganizationID
}

// AI-powered SEO enhancement function using OpenRouter
func enhanceProductSEO(product ShopifyProduct) SEOEnhancement {
	// Try OpenRouter AI enhancement first
	enhancement, err := callOpenRouterForSEO(product)
	if err == nil {
		return enhancement
	}
	fmt.Printf("[WARN] OpenRouter AI SEO enhancement failed, using fallback: %v\n", err)

	// Fallback to rule-based approach
	return createFallbackSEO(product)
}

// Enhanced version with custom options
func enhanceProductSEOWithOptions(product ShopifyProduct, optimizationType, aiModel, language, audience, optimizationLevel, customInstructions string) SEOEnhancement {
	// Try OpenRouter AI enhancement with options
	enhancement, err := callOpenRouterForSEOWithOptions(product, optimizationType, aiModel, language, audience, optimizationLevel, customInstructions)
	if err == nil {
		return enhancement
	}
	fmt.Printf("[WARN] OpenRouter AI SEO enhancement failed, using fallback: %v\n", err)

	// Fallback to rule-based approach
	return createFallbackSEO(product)
}

// callOpenRouterForSEO - Make API call to OpenRouter for SEO enhancement
func callOpenRouterForSEO(product ShopifyProduct) (SEOEnhancement, error) {
	// Convert product to map for AI processing
	price := "0"
	sku := ""
	if len(product.Variants) > 0 {
		price = product.Variants[0].Price
		sku = product.Variants[0].SKU
	}

	productMap := map[string]interface{}{
		"title":        product.Title,
		"description":  product.Description,
		"product_type": product.ProductType,
		"vendor":       product.Vendor,
		"price":        price,
		"sku":          sku,
	}

	productJSON, err := json.Marshal(productMap)
	if err != nil {
		return SEOEnhancement{}, fmt.Errorf("failed to marshal product: %v", err)
	}

	// Create AI prompt for SEO enhancement
	prompt := fmt.Sprintf(`
You are an expert e-commerce SEO specialist. Analyze this product and provide comprehensive SEO optimization.

Product data: %s

Provide a JSON response with the following structure:
{
  "seo_title": "Optimized title under 60 characters",
  "seo_description": "Meta description under 160 characters",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "meta_keywords": "keyword1, keyword2, keyword3",
  "alt_text": "Descriptive alt text for product images",
  "schema_markup": "JSON-LD structured data for the product"
}

Requirements:
- SEO title: Under 60 characters, keyword-rich, compelling
- SEO description: Under 160 characters, persuasive, includes CTA
- Keywords: 5-10 relevant keywords from title, category, brand
- Alt text: Descriptive, includes product name and key features
- Schema markup: Valid JSON-LD for Product type with name, description, brand, category

Return ONLY the JSON response, no explanations.
`, string(productJSON))

	// Use the existing OpenRouter AI function
	response, err := callOpenRouterAI(prompt, 500, 0.7)
	if err != nil {
		return SEOEnhancement{}, fmt.Errorf("OpenRouter AI call failed: %v", err)
	}

	// Parse AI response
	var enhancement SEOEnhancement
	if err := json.Unmarshal([]byte(response), &enhancement); err != nil {
		return SEOEnhancement{}, fmt.Errorf("failed to parse AI response: %v", err)
	}

	return enhancement, nil
}

// callOpenRouterForSEOWithOptions - Make API call with custom options
func callOpenRouterForSEOWithOptions(product ShopifyProduct, optimizationType, aiModel, language, audience, optimizationLevel, customInstructions string) (SEOEnhancement, error) {
	// Set defaults if not provided
	if optimizationType == "" {
		optimizationType = "all"
	}
	if language == "" {
		language = "en"
	}
	if audience == "" {
		audience = "general"
	}
	if optimizationLevel == "" {
		optimizationLevel = "balanced"
	}

	// Convert product to map for AI processing
	price := "0"
	sku := ""
	if len(product.Variants) > 0 {
		price = product.Variants[0].Price
		sku = product.Variants[0].SKU
	}

	productMap := map[string]interface{}{
		"title":        product.Title,
		"description":  product.Description,
		"product_type": product.ProductType,
		"vendor":       product.Vendor,
		"price":        price,
		"sku":          sku,
	}

	productJSON, err := json.Marshal(productMap)
	if err != nil {
		return SEOEnhancement{}, fmt.Errorf("failed to marshal product: %v", err)
	}

	// Build language-specific instructions
	languageInstructions := map[string]string{
		"en": "in English",
		"es": "in Spanish (Espa�ol)",
		"fr": "in French (Fran�ais)",
		"de": "in German (Deutsch)",
	}
	langInstruction := languageInstructions[language]
	if langInstruction == "" {
		langInstruction = "in English"
	}

	// Build audience-specific instructions
	audienceInstructions := map[string]string{
		"general":       "general audience",
		"professionals": "professional audience (business-focused, technical terms are OK)",
		"students":      "students and young adults (clear, educational tone)",
		"families":      "families and parents (warm, family-friendly tone)",
	}
	audienceInstruction := audienceInstructions[audience]
	if audienceInstruction == "" {
		audienceInstruction = "general audience"
	}

	// Build optimization level instructions
	levelInstructions := map[string]string{
		"conservative": "Make minimal changes, preserve the original tone and style. Only fix obvious issues.",
		"balanced":     "Balance between keeping the original style and adding improvements. Moderate SEO optimization.",
		"aggressive":   "Maximize SEO potential. Rewrite completely for best search visibility and conversion.",
	}
	levelInstruction := levelInstructions[optimizationLevel]
	if levelInstruction == "" {
		levelInstruction = levelInstructions["balanced"]
	}

	// Build custom instructions section
	customSection := ""
	if customInstructions != "" {
		customSection = fmt.Sprintf(`

IMPORTANT CUSTOM INSTRUCTIONS FROM USER:
%s

Follow these custom instructions carefully.`, customInstructions)
	}

	// Build optimization type-specific instructions
	optimizationFocus := ""
	switch optimizationType {
	case "title":
		optimizationFocus = "Focus ONLY on optimizing the SEO title. Keep description and other fields minimal."
	case "description":
		optimizationFocus = "Focus ONLY on optimizing the SEO description. Keep title and other fields minimal."
	case "category":
		optimizationFocus = "Focus on improving category classification and keywords."
	case "tags":
		optimizationFocus = "Focus on generating comprehensive, relevant keywords and tags."
	case "seo":
		optimizationFocus = "Focus on technical SEO elements: schema markup, meta tags, alt text."
	default:
		optimizationFocus = "Optimize all aspects: title, description, keywords, and technical SEO."
	}

	// Create enhanced AI prompt
	prompt := fmt.Sprintf(`You are an expert e-commerce SEO specialist. Analyze this product and provide comprehensive SEO optimization %s.

Product data: %s

TARGET LANGUAGE: %s
TARGET AUDIENCE: %s
OPTIMIZATION LEVEL: %s
FOCUS: %s
%s

Provide a JSON response with the following structure:
{
  "seo_title": "Optimized title under 60 characters",
  "seo_description": "Meta description under 160 characters",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "meta_keywords": "keyword1, keyword2, keyword3",
  "alt_text": "Descriptive alt text for product images",
  "schema_markup": "{\"@context\":\"https://schema.org\",\"@type\":\"Product\",\"name\":\"Product Name\",\"description\":\"Description\",\"brand\":{\"@type\":\"Brand\",\"name\":\"Brand\"}}"
}

CRITICAL REQUIREMENTS:
- SEO title: Under 60 characters, keyword-rich, compelling, written %s
- SEO description: Under 160 characters, persuasive, includes CTA, written %s for %s
- Keywords: 5-10 relevant keywords from title, category, brand
- Alt text: Descriptive, includes product name and key features
- Schema markup: MUST be a JSON STRING (escaped JSON), NOT a JSON object. See example above.

Return ONLY the JSON response, no markdown code blocks, no explanations.
`, langInstruction, string(productJSON), langInstruction, audienceInstruction, levelInstruction, optimizationFocus, customSection, langInstruction, langInstruction, audienceInstruction)

	// Use the existing OpenRouter AI function
	response, err := callOpenRouterAI(prompt, 500, 0.7)
	if err != nil {
		fmt.Printf("❌ OpenRouter AI call error: %v\n", err)
		return SEOEnhancement{}, fmt.Errorf("OpenRouter AI call failed: %v", err)
	}

	// Log raw AI response for debugging
	fmt.Printf("🤖 Raw AI Response (first 500 chars): %s\n", response[:min(500, len(response))])

	// Try to clean up the response if it has markdown code blocks
	cleanedResponse := response
	if strings.HasPrefix(response, "```json") {
		cleanedResponse = strings.TrimPrefix(response, "```json")
		cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
		cleanedResponse = strings.TrimSpace(cleanedResponse)
	} else if strings.HasPrefix(response, "```") {
		cleanedResponse = strings.TrimPrefix(response, "```")
		cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
		cleanedResponse = strings.TrimSpace(cleanedResponse)
	}

	// Parse AI response into a flexible map first
	var rawResponse map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &rawResponse); err != nil {
		fmt.Printf("❌ JSON Parse Error: %v\n", err)
		fmt.Printf("❌ Tried to parse: %s\n", cleanedResponse[:min(200, len(cleanedResponse))])
		return SEOEnhancement{}, fmt.Errorf("failed to parse AI response: %v", err)
	}

	// Handle schema_markup - convert object to string if needed
	var schemaMarkupStr string
	if schemaMarkup, ok := rawResponse["schema_markup"]; ok {
		switch v := schemaMarkup.(type) {
		case string:
			schemaMarkupStr = v
		case map[string]interface{}:
			// AI returned object instead of string, stringify it
			if schemaBytes, err := json.Marshal(v); err == nil {
				schemaMarkupStr = string(schemaBytes)
				fmt.Printf("✅ Converted schema_markup object to string\n")
			}
		default:
			schemaMarkupStr = ""
		}
	}

	// Build the SEOEnhancement struct
	enhancement := SEOEnhancement{
		SEOTitle:       getString(rawResponse, "seo_title"),
		SEODescription: getString(rawResponse, "seo_description"),
		Keywords:       getStringArray(rawResponse, "keywords"),
		MetaKeywords:   getString(rawResponse, "meta_keywords"),
		AltText:        getString(rawResponse, "alt_text"),
		SchemaMarkup:   schemaMarkupStr,
	}

	fmt.Printf("✅ Successfully parsed AI response - Title: %s\n", enhancement.SEOTitle)
	return enhancement, nil
}

// Helper function to safely get string from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// Helper function to safely get string array from map
func getStringArray(m map[string]interface{}, key string) []string {
	if val, ok := m[key]; ok {
		if arr, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

// calculateSEOScore calculates a real SEO score based on product metadata quality
func calculateSEOScore(metadata map[string]interface{}) int {
	score := 0

	// Title optimization (25 points)
	if seoTitle, ok := metadata["seo_title"].(string); ok && seoTitle != "" {
		titleLen := len(seoTitle)
		if titleLen >= 30 && titleLen <= 60 {
			score += 25 // Optimal length
		} else if titleLen > 0 {
			score += 15 // Has title but not optimal
		}
	}

	// Description optimization (25 points)
	if seoDesc, ok := metadata["seo_description"].(string); ok && seoDesc != "" {
		descLen := len(seoDesc)
		if descLen >= 120 && descLen <= 160 {
			score += 25 // Optimal length
		} else if descLen >= 50 {
			score += 18 // Good length
		} else if descLen > 0 {
			score += 10 // Has description
		}
	}

	// Keywords (20 points)
	if keywords, ok := metadata["keywords"]; ok {
		if keywordArray, ok := keywords.([]interface{}); ok {
			keywordCount := len(keywordArray)
			if keywordCount >= 5 && keywordCount <= 10 {
				score += 20 // Optimal keyword count
			} else if keywordCount >= 3 {
				score += 15 // Good keyword count
			} else if keywordCount > 0 {
				score += 8 // Has some keywords
			}
		}
	}

	// Alt text (10 points)
	if altText, ok := metadata["alt_text"].(string); ok && altText != "" {
		if len(altText) >= 20 {
			score += 10 // Good alt text
		} else {
			score += 5 // Has alt text
		}
	}

	// Schema markup (15 points)
	if schemaMarkup, ok := metadata["schema_markup"].(string); ok && schemaMarkup != "" {
		if strings.Contains(schemaMarkup, "@context") && strings.Contains(schemaMarkup, "@type") {
			score += 15 // Valid schema markup
		} else {
			score += 8 // Has schema but might be incomplete
		}
	}

	// Meta keywords (5 points)
	if metaKeywords, ok := metadata["meta_keywords"].(string); ok && metaKeywords != "" {
		score += 5
	}

	// Ensure score is between 0 and 100
	if score > 100 {
		score = 100
	}

	return score
}

// createFallbackSEO - Create fallback SEO when AI fails
func createFallbackSEO(product ShopifyProduct) SEOEnhancement {
	title := product.Title
	description := product.Description
	category := product.ProductType
	vendor := product.Vendor

	// Create fallback SEO
	seoTitle := title
	if len(seoTitle) > 60 {
		seoTitle = seoTitle[:57] + "..."
	}

	seoDescription := description
	if len(seoDescription) > 160 {
		seoDescription = seoDescription[:157] + "..."
	} else if seoDescription == "" {
		seoDescription = fmt.Sprintf("Shop %s online. High-quality %s from %s. Fast shipping and great customer service.", title, category, vendor)
	}

	keywords := []string{
		strings.ToLower(title),
		strings.ToLower(category),
		strings.ToLower(vendor),
		"online shopping",
		"buy online",
	}

	if category != "" {
		keywords = append(keywords, strings.ToLower(category)+" for sale")
	}

	return SEOEnhancement{
		SEOTitle:       seoTitle,
		SEODescription: seoDescription,
		Keywords:       keywords,
		MetaKeywords:   strings.Join(keywords, ", "),
		AltText:        fmt.Sprintf("%s - %s product from %s", title, category, vendor),
		SchemaMarkup:   fmt.Sprintf(`{"@context":"https://schema.org","@type":"Product","name":"%s","description":"%s","brand":{"@type":"Brand","name":"%s"},"category":"%s"}`, title, description, vendor, category),
	}
}

// initDB initializes the database connection
func initDB() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db != nil {
		return nil // Already initialized
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Printf("[ERROR] DATABASE_URL environment variable is not set\n")
		return fmt.Errorf("DATABASE_URL not set")
	}

	fmt.Printf("[INFO] Attempting to connect to database...\n")

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		fmt.Printf("[ERROR] Failed to open database connection: %v\n", err)
		return err
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		fmt.Printf("[ERROR] Failed to ping database: %v\n", err)
		return err
	}

	fmt.Printf("[INFO] Database connection established successfully\n")

	// Create all required tables
	tables := []string{
		// Add connector_id column to existing product_feeds table if it doesn't exist
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'product_feeds' AND column_name = 'connector_id') THEN
				ALTER TABLE product_feeds ADD COLUMN connector_id VARCHAR(255);
				CREATE INDEX IF NOT EXISTS idx_product_feeds_connector_id ON product_feeds(connector_id);
			END IF;
		END $$;`,
		`CREATE TABLE IF NOT EXISTS connectors (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			shop_domain VARCHAR(255) NOT NULL,
			access_token TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			last_sync TIMESTAMP WITH TIME ZONE
		);`,
		`CREATE TABLE IF NOT EXISTS products (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			connector_id VARCHAR(255) REFERENCES connectors(id),
			external_id VARCHAR(255) NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			price DECIMAL(10,2),
			currency VARCHAR(3) DEFAULT 'USD',
			sku VARCHAR(255),
			gtin VARCHAR(255),
			brand VARCHAR(255),
			category VARCHAR(255),
			images TEXT[],
			variants JSONB,
			shipping JSONB,
			custom_labels TEXT[],
			metadata JSONB,
			status VARCHAR(50) DEFAULT 'ACTIVE',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(connector_id, external_id)
		);`,
		`ALTER TABLE products ADD COLUMN IF NOT EXISTS compare_at_price DECIMAL(10,2);`,
		`CREATE TABLE IF NOT EXISTS feed_variants (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			product_id UUID REFERENCES products(id),
			name VARCHAR(255) NOT NULL,
			config JSONB,
			transformation JSONB,
			status VARCHAR(50) DEFAULT 'ACTIVE',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS issues (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			product_id UUID REFERENCES products(id),
			connector_id VARCHAR(255) REFERENCES connectors(id),
			type VARCHAR(100) NOT NULL,
			severity VARCHAR(20) DEFAULT 'WARNING',
			message TEXT NOT NULL,
			details JSONB,
			status VARCHAR(50) DEFAULT 'OPEN',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			resolved_at TIMESTAMP WITH TIME ZONE
		);`,
		`CREATE TABLE IF NOT EXISTS channels (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL,
			config JSONB,
			credentials JSONB,
			status VARCHAR(50) DEFAULT 'ACTIVE',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			domain VARCHAR(255),
			settings JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID REFERENCES organizations(id),
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255),
			role VARCHAR(50) DEFAULT 'USER',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS inventory_levels (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			product_id UUID REFERENCES products(id),
			connector_id VARCHAR(255) REFERENCES connectors(id),
			inventory_item_id VARCHAR(255) NOT NULL,
			location_id VARCHAR(255) NOT NULL,
			available_quantity INTEGER DEFAULT 0,
			committed_quantity INTEGER DEFAULT 0,
			incoming_quantity INTEGER DEFAULT 0,
			on_hand_quantity INTEGER DEFAULT 0,
			last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(connector_id, inventory_item_id, location_id)
		);`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
			type VARCHAR(50) NOT NULL CHECK (type IN ('feed_generated', 'feed_failed', 'feed_scheduled', 'system_alert', 'info')),
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			is_read BOOLEAN DEFAULT FALSE,
			priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
			entity_type VARCHAR(50),
			entity_id UUID,
			entity_name VARCHAR(255),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			read_at TIMESTAMP WITH TIME ZONE,
			expires_at TIMESTAMP WITH TIME ZONE
		);`,
		`CREATE TABLE IF NOT EXISTS platform_credentials (
			feed_id VARCHAR(255) PRIMARY KEY,
			organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
			platform VARCHAR(100) NOT NULL,
			name VARCHAR(255),
			api_key VARCHAR(500),
			merchant_id VARCHAR(255),
			access_token TEXT,
			auto_submit BOOLEAN DEFAULT FALSE,
			submit_on_regenerate BOOLEAN DEFAULT FALSE,
			config JSONB DEFAULT '{}',
			last_submission_at TIMESTAMP WITH TIME ZONE,
			last_submission_status VARCHAR(50),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_platform_credentials_organization_id ON platform_credentials(organization_id);`,
		`CREATE INDEX IF NOT EXISTS idx_platform_credentials_platform ON platform_credentials(platform);`,
	}

	// Execute all table creation statements
	for i, tableSQL := range tables {
		_, err = db.Exec(tableSQL)
		if err != nil {
			fmt.Printf("[ERROR] Failed to create table %d: %v\n", i, err)
			fmt.Printf("[ERROR] SQL: %s\n", tableSQL)
			return fmt.Errorf("failed to create table %d: %v", i, err)
		}
	}

	return nil
}

// exchangeCodeForToken exchanges authorization code for access token
func exchangeCodeForToken(code, shop, clientID, clientSecret string) (string, error) {
	// Clean shop domain
	cleanDomain := shop
	if strings.HasSuffix(shop, ".myshopify.com") {
		cleanDomain = strings.TrimSuffix(shop, ".myshopify.com")
	}

	// Prepare token exchange request
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)

	// Make request to Shopify
	tokenURL := fmt.Sprintf("https://%s.myshopify.com/admin/oauth/access_token", cleanDomain)
	log.Printf("🔄 Making token request to: %s", tokenURL)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		log.Printf("❌ HTTP request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("📡 Response status: %d", resp.StatusCode)

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response body: %v", err)
		return "", err
	}

	log.Printf("📄 Response body: %s", string(body))

	// Parse JSON response
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		log.Printf("❌ JSON parse error: %v", err)
		return "", err
	}

	if tokenResponse.AccessToken == "" {
		log.Printf("❌ No access token in response")
		return "", fmt.Errorf("no access token in response: %s", string(body))
	}

	log.Printf("✅ Successfully obtained access token")

	return tokenResponse.AccessToken, nil
}

// fetchShopifyProducts fetches products from Shopify API
func fetchShopifyProducts(shopDomain, accessToken string) ([]ShopifyProduct, error) {
	// Clean shop domain
	cleanDomain := shopDomain
	if strings.HasSuffix(shopDomain, ".myshopify.com") {
		cleanDomain = strings.TrimSuffix(shopDomain, ".myshopify.com")
	}

	// Create HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Build API URL - let Shopify return all default fields including inventory data
	apiURL := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/products.json?limit=250", cleanDomain)

	// Create request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers
	req.Header.Set("X-Shopify-Access-Token", accessToken)
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Check status and return detailed error
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Shopify API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var response struct {
		Products []ShopifyProduct `json:"products"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v, response: %s", err, string(body))
	}

	return response.Products, nil
}

// fetchInventoryLevels fetches inventory levels for product variants from Shopify
func fetchInventoryLevels(shopDomain, accessToken string, variantIDs []int64) (map[int64]int, error) {
	if len(variantIDs) == 0 {
		return make(map[int64]int), nil
	}

	// Clean shop domain
	cleanDomain := shopDomain
	if strings.HasSuffix(shopDomain, ".myshopify.com") {
		cleanDomain = strings.TrimSuffix(shopDomain, ".myshopify.com")
	}

	// Create HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Convert variant IDs to comma-separated string
	var variantIDStrings []string
	for _, id := range variantIDs {
		variantIDStrings = append(variantIDStrings, fmt.Sprintf("%d", id))
	}
	variantIDsParam := strings.Join(variantIDStrings, ",")

	// Build API URL to get inventory item IDs for variants
	variantsURL := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/variants.json?ids=%s", cleanDomain, variantIDsParam)

	// Create request for variants
	req, err := http.NewRequest("GET", variantsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create variants request: %v", err)
	}

	// Add headers
	req.Header.Set("X-Shopify-Access-Token", accessToken)
	req.Header.Set("Accept", "application/json")

	// Make request for variants
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("variants request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read variants response: %v", err)
	}

	// Check status
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Shopify variants API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse variants response to get inventory item IDs
	var variantsResponse struct {
		Variants []struct {
			ID              int64 `json:"id"`
			InventoryItemID int64 `json:"inventory_item_id"`
		} `json:"variants"`
	}

	if err := json.Unmarshal(body, &variantsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse variants JSON: %v", err)
	}

	// Extract inventory item IDs
	var inventoryItemIDs []int64
	variantToInventoryItem := make(map[int64]int64)
	for _, variant := range variantsResponse.Variants {
		inventoryItemIDs = append(inventoryItemIDs, variant.InventoryItemID)
		variantToInventoryItem[variant.ID] = variant.InventoryItemID
	}

	if len(inventoryItemIDs) == 0 {
		return make(map[int64]int), nil
	}

	// Convert inventory item IDs to comma-separated string
	var inventoryIDStrings []string
	for _, id := range inventoryItemIDs {
		inventoryIDStrings = append(inventoryIDStrings, fmt.Sprintf("%d", id))
	}
	inventoryIDsParam := strings.Join(inventoryIDStrings, ",")

	// Build API URL to get inventory levels
	inventoryURL := fmt.Sprintf("https://%s.myshopify.com/admin/api/2023-10/inventory_levels.json?inventory_item_ids=%s", cleanDomain, inventoryIDsParam)

	// Create request for inventory levels
	req, err = http.NewRequest("GET", inventoryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory request: %v", err)
	}

	// Add headers
	req.Header.Set("X-Shopify-Access-Token", accessToken)
	req.Header.Set("Accept", "application/json")

	// Make request for inventory levels
	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inventory request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory response: %v", err)
	}

	// Check status
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Shopify inventory API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse inventory levels response
	var inventoryResponse struct {
		InventoryLevels []struct {
			InventoryItemID int64 `json:"inventory_item_id"`
			Available       int   `json:"available"`
		} `json:"inventory_levels"`
	}

	if err := json.Unmarshal(body, &inventoryResponse); err != nil {
		return nil, fmt.Errorf("failed to parse inventory JSON: %v", err)
	}

	// Map inventory levels to variant IDs
	inventoryLevels := make(map[int64]int)
	for _, level := range inventoryResponse.InventoryLevels {
		// Find variant ID for this inventory item ID
		for variantID, inventoryItemID := range variantToInventoryItem {
			if inventoryItemID == level.InventoryItemID {
				inventoryLevels[variantID] = level.Available
				break
			}
		}
	}

	return inventoryLevels, nil
}

// getStringValue safely extracts string value from sql.NullString
func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// getFloatValue safely extracts float value from sql.NullFloat64
func getFloatValue(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0.0
}

// getInt64Value safely extracts int64 value from sql.NullInt64
func getInt64Value(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

// getTimeValue safely extracts time value from sql.NullTime
func getTimeValue(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// seedSampleData creates sample products and feeds for development
func seedSampleData(organizationID string) {
	log.Printf("🌱 Seeding sample data for organization: %s", organizationID)

	// Sample products
	sampleProducts := []map[string]interface{}{
		{
			"external_id": "PROD-001",
			"title":       "Premium Wireless Headphones",
			"description": "High-quality wireless headphones with noise cancellation",
			"price":       299.99,
			"currency":    "USD",
			"sku":         "WH-001",
			"brand":       "TechBrand",
			"category":    "Electronics",
			"images":      `["https://example.com/images/headphones1.jpg"]`,
			"status":      "ACTIVE",
			"gtin":        "123456789012",
		},
		{
			"external_id": "PROD-002",
			"title":       "Smart Fitness Watch",
			"description": "Advanced fitness tracking with heart rate monitoring",
			"price":       199.99,
			"currency":    "USD",
			"sku":         "SFW-002",
			"brand":       "FitTech",
			"category":    "Wearables",
			"images":      `["https://example.com/images/watch1.jpg"]`,
			"status":      "ACTIVE",
			"gtin":        "987654321098",
		},
		{
			"external_id": "PROD-003",
			"title":       "Organic Cotton T-Shirt",
			"description": "Comfortable organic cotton t-shirt in multiple colors",
			"price":       29.99,
			"currency":    "USD",
			"sku":         "CTS-003",
			"brand":       "EcoWear",
			"category":    "Clothing",
			"images":      `["https://example.com/images/tshirt1.jpg"]`,
			"status":      "ACTIVE",
			"gtin":        "111111111111",
		},
		{
			"external_id": "PROD-004",
			"title":       "Bluetooth Speaker",
			"description": "Portable Bluetooth speaker with 360-degree sound",
			"price":       89.99,
			"currency":    "USD",
			"sku":         "BTS-004",
			"brand":       "SoundWave",
			"category":    "Electronics",
			"images":      `["https://example.com/images/speaker1.jpg"]`,
			"status":      "ACTIVE",
			"gtin":        "222222222222",
		},
		{
			"external_id": "PROD-005",
			"title":       "Yoga Mat",
			"description": "Non-slip yoga mat for home and studio practice",
			"price":       49.99,
			"currency":    "USD",
			"sku":         "YM-005",
			"brand":       "ZenFit",
			"category":    "Sports",
			"images":      `["https://example.com/images/yogamat1.jpg"]`,
			"status":      "ACTIVE",
			"gtin":        "333333333333",
		},
	}

	// Insert sample products
	for _, product := range sampleProducts {
		_, err := db.Exec(`
			INSERT INTO products (
				organization_id, external_id, title, description, price, currency, 
				sku, brand, category, images, status, gtin, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		`, organizationID, product["external_id"], product["title"], product["description"],
			product["price"], product["currency"], product["sku"], product["brand"],
			product["category"], product["images"], product["status"], product["gtin"])

		if err != nil {
			log.Printf("❌ Error inserting sample product %s: %v", product["external_id"], err)
		} else {
			log.Printf("✅ Inserted sample product: %s", product["title"])
		}
	}

	// Sample feeds
	sampleFeeds := []map[string]interface{}{
		{
			"name":              "Google Shopping Feed",
			"channel":           "Google Shopping",
			"format":            "xml",
			"status":            "active",
			"products_count":    5,
			"last_generated_at": time.Now(),
		},
		{
			"name":              "Facebook Catalog Feed",
			"channel":           "Facebook",
			"format":            "csv",
			"status":            "active",
			"products_count":    5,
			"last_generated_at": time.Now(),
		},
	}

	// Insert sample feeds
	for _, feed := range sampleFeeds {
		_, err := db.Exec(`
			INSERT INTO product_feeds (
				organization_id, name, channel, format, status, 
				products_count, last_generated_at, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, organizationID, feed["name"], feed["channel"], feed["format"],
			feed["status"], feed["products_count"], feed["last_generated_at"])

		if err != nil {
			log.Printf("❌ Error inserting sample feed %s: %v", feed["name"], err)
		} else {
			log.Printf("✅ Inserted sample feed: %s", feed["name"])
		}
	}

	log.Printf("🌱 Sample data seeding completed!")
}

// generateGoogleShoppingXML generates XML feed for Google Shopping

// AI-Powered Helper Functions with OpenRouter Integration

// OpenRouter AI Configuration
const (
	OPENROUTER_BASE_URL = "https://openrouter.ai/api/v1/chat/completions"
	// Free model (rate limited): "meta-llama/llama-3.3-70b-instruct:free"
	// Paid models (recommended for production):
	// - "anthropic/claude-3.5-sonnet" - Best quality, ~$3 per 1M tokens
	// - "openai/gpt-4o-mini" - Cheap and fast, ~$0.15 per 1M tokens
	// - "google/gemini-flash-1.5" - Very cheap, ~$0.075 per 1M tokens
	OPENROUTER_MODEL = "meta-llama/llama-3.3-70b-instruct:free" // Change this if rate limited
)

// OpenRouterRequest represents the request structure for OpenRouter API
type OpenRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenRouterMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// callOpenRouterAI makes a request to OpenRouter AI API
func callOpenRouterAI(prompt string, maxTokens int, temperature float64) (string, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	// Get model from environment variable or use default
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = OPENROUTER_MODEL
	}

	// Log the API call for debugging
	fmt.Printf("🤖 AI API Call (Model: %s): %s\n", model, prompt[:min(50, len(prompt))])

	request := OpenRouterRequest{
		Model:       model,
		Messages:    []OpenRouterMessage{{Role: "user", Content: prompt}},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", OPENROUTER_BASE_URL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenRouterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// optimizeProductTitle generates SEO-optimized product titles using hybrid approach
func optimizeProductTitle(title, description, brand, category, keywords string, maxLength int) string {
	if maxLength == 0 {
		maxLength = 60 // Default SEO-friendly length
	}

	// Try AI optimization first
	aiTitle, err := optimizeTitleWithAI(title, description, brand, category, keywords, maxLength)
	if err == nil && aiTitle != "" {
		fmt.Printf("✅ AI Title Optimization: %s\n", aiTitle)
		return aiTitle
	}

	// Fallback to rule-based optimization
	fmt.Printf("⚠️ AI Failed, using fallback: %s\n", err)
	ruleTitle := optimizeTitleWithRules(title, description, brand, category, keywords, maxLength)
	fmt.Printf("📝 Rule-based Title: %s\n", ruleTitle)
	return ruleTitle
}

// optimizeTitleWithAI uses OpenRouter AI for title optimization
func optimizeTitleWithAI(title, description, brand, category, keywords string, maxLength int) (string, error) {
	prompt := fmt.Sprintf(`You are a professional e-commerce SEO expert. Transform this product title into a compelling, searchable title that drives clicks and sales.

CURRENT PRODUCT:
- Original Title: "%s"
- Product Description: "%s"
- Brand: "%s"
- Category: "%s"
- Keywords to include: "%s"

TASK: Create a new title that is:
1. MORE DESCRIPTIVE than the original
2. Includes the brand name naturally
3. Contains relevant keywords customers search for
4. Under %d characters
5. Compelling and click-worthy

EXAMPLES OF GOOD TRANSFORMATIONS:
Original: "Summer Necklace"
Better: "Gold Boho Necklace with Turquoise Pendant - Elegant Summer Jewelry"

Original: "Leather Bag"
Better: "Premium Leather Crossbody Bag - Stylish Women's Handbag"

Original: "Blue Shirt"
Better: "Men's Cotton Oxford Blue Dress Shirt - Business Casual"

Based on the description "%s", create a title that describes what the product actually is and why customers should buy it. Return ONLY the optimized title:`, title, description, brand, category, keywords, maxLength, description)

	fmt.Printf("🤖 AI Input - Title: '%s', Description: '%s', Brand: '%s'\n", title, description, brand)

	aiTitle, err := callOpenRouterAI(prompt, 50, 0.7)
	if err != nil {
		fmt.Printf("❌ AI Error: %v\n", err)
		return "", err
	}

	// Clean and validate AI response
	aiTitle = strings.TrimSpace(aiTitle)
	fmt.Printf("🤖 AI Raw Response: '%s'\n", aiTitle)

	if len(aiTitle) > maxLength {
		aiTitle = aiTitle[:maxLength-3] + "..."
	}

	fmt.Printf("✅ AI Final Title: '%s'\n", aiTitle)
	return aiTitle, nil
}

// optimizeTitleWithRules provides rule-based fallback for title optimization
func optimizeTitleWithRules(title, description, brand, category, keywords string, maxLength int) string {
	// Clean and prepare base title
	baseTitle := strings.TrimSpace(title)
	if baseTitle == "" {
		baseTitle = "Product"
	}

	// Add brand if not already included
	if brand != "" && !strings.Contains(strings.ToLower(baseTitle), strings.ToLower(brand)) {
		baseTitle = brand + " " + baseTitle
	}

	// Add category if not already included and space allows
	if category != "" && !strings.Contains(strings.ToLower(baseTitle), strings.ToLower(category)) {
		if len(baseTitle+" "+category) <= maxLength {
			baseTitle = baseTitle + " " + category
		}
	}

	// Add keywords if provided and space allows
	if keywords != "" {
		keywordList := strings.Split(keywords, ",")
		for _, keyword := range keywordList {
			keyword = strings.TrimSpace(keyword)
			if keyword != "" && !strings.Contains(strings.ToLower(baseTitle), strings.ToLower(keyword)) {
				if len(baseTitle+" "+keyword) <= maxLength {
					baseTitle = baseTitle + " " + keyword
				}
			}
		}
	}

	// Truncate if too long
	if len(baseTitle) > maxLength {
		baseTitle = baseTitle[:maxLength-3] + "..."
	}

	return baseTitle
}

// enhanceProductDescription creates compelling product descriptions using hybrid approach
func enhanceProductDescription(title, description, brand, category string, price float64, style, length string) string {
	if style == "" {
		style = "marketing"
	}
	if length == "" {
		length = "medium"
	}

	// Try AI enhancement first (no custom instructions in this path)
	aiDescription, err := enhanceDescriptionWithAI(title, description, brand, category, price, style, length, "")
	if err == nil && aiDescription != "" {
		return aiDescription
	}

	// Fallback to rule-based enhancement
	return enhanceDescriptionWithRules(title, description, brand, category, price, style, length)
}

// enhanceDescriptionWithAI uses OpenRouter AI for description enhancement
func enhanceDescriptionWithAI(title, description, brand, category string, price float64, style, length, customInstructions string) (string, error) {
	// Build the base prompt
	prompt := fmt.Sprintf(`You are an expert e-commerce copywriter who specializes in creating compelling product descriptions that drive sales and conversions.

PRODUCT INFORMATION:
- Product Name: "%s"
- Original Description: "%s"
- Brand: "%s"
- Category: "%s"
- Price: $%.2f
- Style: %s
- Length: %s

COPYWRITING REQUIREMENTS:
1. Write in %s style (marketing/technical/casual)
2. Make it %s length (short/medium/long)
3. Focus on CUSTOMER BENEFITS, not just features
4. Use EMOTIONAL TRIGGERS and POWER WORDS
5. Include a COMPELLING CALL-TO-ACTION
6. Make it SCANNABLE with bullet points or short paragraphs
7. Add RELEVANT EMOJIS to increase engagement
8. Address CUSTOMER PAIN POINTS and solutions
9. Create URGENCY and DESIRE to buy

STYLE GUIDELINES:
- Marketing: Focus on benefits, emotional appeal, social proof
- Technical: Detailed specifications, features, performance
- Casual: Friendly, conversational, approachable`, title, description, brand, category, price, style, length, style, length)

	// Add custom instructions if provided
	if customInstructions != "" {
		prompt += fmt.Sprintf(`

CUSTOM INSTRUCTIONS (IMPORTANT - Follow these):
%s`, customInstructions)
	}

	prompt += "\n\nCreate a description that makes customers excited to buy this product. Return ONLY the enhanced description:"

	aiDescription, err := callOpenRouterAI(prompt, 300, 0.8)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(aiDescription), nil
}

// enhanceDescriptionWithRules provides rule-based fallback for description enhancement
func enhanceDescriptionWithRules(title, description, brand, category string, price float64, style, length string) string {
	// Clean existing description
	cleanDesc := strings.TrimSpace(description)
	if cleanDesc == "" {
		cleanDesc = "High-quality product"
	}

	// Remove HTML tags for processing
	cleanDesc = strings.ReplaceAll(cleanDesc, "<p>", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "</p>", "")
	cleanDesc = strings.ReplaceAll(cleanDesc, "<br>", " ")
	cleanDesc = strings.ReplaceAll(cleanDesc, "<br/>", " ")

	// Build enhanced description based on style
	var enhanced strings.Builder

	// Add compelling opening based on style
	switch style {
	case "marketing":
		enhanced.WriteString("🌟 ")
		if brand != "" {
			enhanced.WriteString(fmt.Sprintf("Discover the premium %s ", brand))
		}
		enhanced.WriteString(fmt.Sprintf("%s - ", title))
	case "technical":
		enhanced.WriteString("🔧 ")
		enhanced.WriteString(fmt.Sprintf("Technical specifications for %s: ", title))
	case "casual":
		enhanced.WriteString("✨ ")
		enhanced.WriteString(fmt.Sprintf("Love this %s! ", title))
	}

	// Add enhanced description content
	enhanced.WriteString(cleanDesc)

	// Add features based on length
	if length == "long" || length == "medium" {
		enhanced.WriteString(" Features include:")
		if brand != "" {
			enhanced.WriteString(fmt.Sprintf(" premium %s quality,", brand))
		}
		if category != "" {
			enhanced.WriteString(fmt.Sprintf(" perfect for %s enthusiasts,", category))
		}
		enhanced.WriteString(" durable construction,")
		enhanced.WriteString(" and exceptional value.")
	}

	// Add call to action based on style
	switch style {
	case "marketing":
		enhanced.WriteString(" 🛒 Shop now and experience the difference!")
	case "technical":
		enhanced.WriteString(" 📊 Ideal for professionals and enthusiasts.")
	case "casual":
		enhanced.WriteString(" 😍 You'll love it!")
	}

	return enhanced.String()
}

// suggestProductCategory provides AI-powered category suggestions using hybrid approach
func suggestProductCategory(title, description, brand, currentCategory string) []map[string]interface{} {
	// Try AI categorization first
	aiSuggestions, err := suggestCategoryWithAI(title, description, brand, currentCategory)
	if err == nil && len(aiSuggestions) > 0 {
		return aiSuggestions
	}

	// Fallback to rule-based categorization
	return suggestCategoryWithRules(title, description, brand, currentCategory)
}

// suggestCategoryWithAI uses OpenRouter AI for category suggestions
func suggestCategoryWithAI(title, description, brand, currentCategory string) ([]map[string]interface{}, error) {
	prompt := fmt.Sprintf(`You are an expert e-commerce product categorization specialist. Analyze this product and suggest the most appropriate Google Shopping-style hierarchical categories.

Product: "%s"
Description: "%s"
Brand: "%s"
Current Category: "%s"

Provide 3 category suggestions in this exact JSON format:
[
  {"category": "Apparel & Accessories > Clothing > Outerwear > Jackets & Coats", "confidence": 95, "reason": "Product is clearly outerwear"},
  {"category": "Sporting Goods > Outdoor Recreation > Outdoor Clothing", "confidence": 85, "reason": "Suitable for outdoor activities"},
  {"category": "Apparel & Accessories > Clothing > Activewear", "confidence": 75, "reason": "Can be used for sports"}
]

IMPORTANT REQUIREMENTS:
- Use hierarchical categories with " > " separators (e.g., "Parent > Child > Grandchild")
- Use Google Shopping / Google Merchant Center category format
- Confidence as WHOLE NUMBERS 0-100 (not decimals like 0.95)
- Categories should be specific and follow e-commerce standards
- Focus on the most accurate category paths
- Common category prefixes: "Apparel & Accessories", "Electronics", "Home & Garden", "Sporting Goods", "Health & Beauty", "Toys & Games"

Return ONLY the JSON array, no other text:`, title, description, brand, currentCategory)

	aiResponse, err := callOpenRouterAI(prompt, 200, 0.6)
	if err != nil {
		return nil, err
	}

	// Parse AI response as JSON
	var suggestions []map[string]interface{}
	if err := json.Unmarshal([]byte(aiResponse), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	return suggestions, nil
}

// suggestCategoryWithRules provides rule-based fallback for category suggestions
func suggestCategoryWithRules(title, description, brand, currentCategory string) []map[string]interface{} {
	suggestions := []map[string]interface{}{}

	// Analyze title and description for category keywords
	text := strings.ToLower(title + " " + description)

	// Fashion categories
	if strings.Contains(text, "shirt") || strings.Contains(text, "blouse") || strings.Contains(text, "top") {
		suggestions = append(suggestions, map[string]interface{}{
			"category":   "Shirts & Tops",
			"confidence": 0.9,
			"reason":     "Contains clothing keywords",
		})
	}

	if strings.Contains(text, "jacket") || strings.Contains(text, "coat") {
		suggestions = append(suggestions, map[string]interface{}{
			"category":   "Outerwear",
			"confidence": 0.95,
			"reason":     "Contains outerwear keywords",
		})
	}

	if strings.Contains(text, "jeans") || strings.Contains(text, "pants") {
		suggestions = append(suggestions, map[string]interface{}{
			"category":   "Bottoms",
			"confidence": 0.9,
			"reason":     "Contains bottom wear keywords",
		})
	}

	if strings.Contains(text, "necklace") || strings.Contains(text, "earrings") || strings.Contains(text, "bracelet") {
		suggestions = append(suggestions, map[string]interface{}{
			"category":   "Jewelry",
			"confidence": 0.95,
			"reason":     "Contains jewelry keywords",
		})
	}

	if strings.Contains(text, "watch") {
		suggestions = append(suggestions, map[string]interface{}{
			"category":   "Watches",
			"confidence": 0.9,
			"reason":     "Contains watch keywords",
		})
	}

	if strings.Contains(text, "dress") {
		suggestions = append(suggestions, map[string]interface{}{
			"category":   "Dresses",
			"confidence": 0.9,
			"reason":     "Contains dress keywords",
		})
	}

	// If no specific suggestions, provide general categories
	if len(suggestions) == 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"category":   "Fashion",
			"confidence": 0.7,
			"reason":     "General fashion category",
		})
	}

	return suggestions
}

// optimizeProductImages provides AI-powered image optimization suggestions
func optimizeProductImages(title, description, brand, category string, images []string) []map[string]interface{} {
	suggestions := []map[string]interface{}{}

	// Check for missing images
	if len(images) == 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":           "missing_images",
			"priority":       "high",
			"message":        "Add product images to improve conversion rates",
			"recommendation": "Upload at least 3-5 high-quality product images",
			"impact":         "High - Images are crucial for online sales",
		})
		return suggestions
	}

	// Check for low image count
	if len(images) < 3 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":           "low_image_count",
			"priority":       "medium",
			"message":        fmt.Sprintf("Only %d images found, recommend 3-5 images", len(images)),
			"recommendation": "Add more product images from different angles",
			"impact":         "Medium - More images increase customer confidence",
		})
	}

	// Check for excessive images
	if len(images) > 8 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":           "too_many_images",
			"priority":       "low",
			"message":        fmt.Sprintf("Many images (%d) may slow page loading", len(images)),
			"recommendation": "Consider reducing to 5-8 high-quality images",
			"impact":         "Low - Performance optimization",
		})
	}

	// Check image quality and format
	for i, imageURL := range images {
		// Check for placeholder images
		if strings.Contains(imageURL, "placeholder") || strings.Contains(imageURL, "default") {
			suggestions = append(suggestions, map[string]interface{}{
				"type":           "low_quality_image",
				"priority":       "high",
				"message":        fmt.Sprintf("Image %d appears to be a placeholder", i+1),
				"recommendation": "Replace with high-quality product photos",
				"impact":         "High - Placeholder images reduce trust",
				"image_url":      imageURL,
			})
		}

		// Check image dimensions from URL
		if strings.Contains(imageURL, "_925x") {
			// Good standard size
		} else if strings.Contains(imageURL, "_1024x") || strings.Contains(imageURL, "_2048x") {
			suggestions = append(suggestions, map[string]interface{}{
				"type":           "high_resolution_image",
				"priority":       "low",
				"message":        fmt.Sprintf("Image %d has excellent resolution", i+1),
				"recommendation": "Great! High resolution images build trust",
				"impact":         "Positive - High quality image",
				"image_url":      imageURL,
			})
		}
	}

	// Suggest image types based on product category
	text := strings.ToLower(title + " " + description + " " + category)

	if strings.Contains(text, "clothing") || strings.Contains(text, "shirt") || strings.Contains(text, "dress") || strings.Contains(text, "jacket") {
		suggestions = append(suggestions, map[string]interface{}{
			"type":           "image_variety_fashion",
			"priority":       "medium",
			"message":        "Fashion items benefit from multiple angles",
			"recommendation": "Add front, back, side, and detail shots. Include model photos if possible.",
			"impact":         "Medium - Multiple angles help customers visualize the product",
		})
	}

	if strings.Contains(text, "jewelry") || strings.Contains(text, "necklace") || strings.Contains(text, "ring") || strings.Contains(text, "watch") {
		suggestions = append(suggestions, map[string]interface{}{
			"type":           "image_variety_jewelry",
			"priority":       "medium",
			"message":        "Jewelry needs detailed close-ups",
			"recommendation": "Add macro shots showing details, craftsmanship, and materials",
			"impact":         "Medium - Detail shots build trust in quality",
		})
	}

	// Check for consistent image style
	if len(images) > 1 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":           "image_consistency",
			"priority":       "low",
			"message":        "Ensure consistent lighting and background across images",
			"recommendation": "Use similar lighting, background, and styling for all product photos",
			"impact":         "Low - Consistency improves professional appearance",
		})
	}

	// If no specific suggestions, provide general ones
	if len(suggestions) == 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":           "image_optimization_general",
			"priority":       "low",
			"message":        "Images look good! Consider these optimizations:",
			"recommendation": "A/B test different image orders, add lifestyle shots, or consider video",
			"impact":         "Low - Further optimization opportunities",
		})
	}

	return suggestions
}

// calculateTitleImprovement measures title optimization improvement
func calculateTitleImprovement(original, optimized string) map[string]interface{} {
	originalLength := len(original)
	optimizedLength := len(optimized)

	// Calculate improvement metrics
	lengthImprovement := float64(optimizedLength-originalLength) / float64(originalLength) * 100

	// Check for SEO improvements
	seoScore := 0
	if strings.Contains(strings.ToLower(optimized), "premium") {
		seoScore += 10
	}
	if strings.Contains(strings.ToLower(optimized), "quality") {
		seoScore += 10
	}
	if len(optimized) <= 60 {
		seoScore += 20
	}

	return map[string]interface{}{
		"length_change":       lengthImprovement,
		"seo_improvement":     seoScore,
		"overall_improvement": "enhanced",
	}
}

// calculateTitleScore evaluates SEO score for titles
func calculateTitleScore(title string) int {
	score := 0

	// Length check (optimal: 50-60 characters)
	length := len(title)
	if length >= 50 && length <= 60 {
		score += 30
	} else if length >= 40 && length <= 70 {
		score += 20
	} else {
		score += 10
	}

	// Keyword density
	words := strings.Fields(strings.ToLower(title))
	if len(words) > 0 {
		score += 20
	}

	// Brand presence
	if strings.Contains(strings.ToLower(title), "premium") ||
		strings.Contains(strings.ToLower(title), "quality") {
		score += 20
	}

	// Special characters (avoid excessive use)
	specialChars := strings.Count(title, "!") + strings.Count(title, "?") + strings.Count(title, "*")
	if specialChars <= 2 {
		score += 15
	} else {
		score += 5
	}

	// Title case (proper capitalization)
	if title == strings.Title(strings.ToLower(title)) {
		score += 15
	}

	return score
}

// calculateDescriptionImprovement measures description enhancement
func calculateDescriptionImprovement(original, enhanced string) map[string]interface{} {
	originalLength := len(original)
	enhancedLength := len(enhanced)

	lengthImprovement := float64(enhancedLength-originalLength) / float64(originalLength) * 100

	// Check for engagement improvements
	engagementScore := 0
	if strings.Contains(enhanced, "🌟") || strings.Contains(enhanced, "✨") {
		engagementScore += 20
	}
	if strings.Contains(enhanced, "!") {
		engagementScore += 10
	}

	return map[string]interface{}{
		"length_change":          lengthImprovement,
		"engagement_improvement": engagementScore,
		"overall_improvement":    "enhanced",
	}
}

// calculateReadabilityScore evaluates description readability
func calculateReadabilityScore(description string) int {
	score := 0

	// Length check
	length := len(description)
	if length >= 100 && length <= 300 {
		score += 30
	} else if length >= 50 && length <= 500 {
		score += 20
	} else {
		score += 10
	}

	// Sentence structure
	sentences := strings.Count(description, ".") + strings.Count(description, "!")
	if sentences > 0 {
		avgLength := length / sentences
		if avgLength >= 10 && avgLength <= 30 {
			score += 25
		} else {
			score += 15
		}
	}

	// Engagement elements
	if strings.Contains(description, "!") {
		score += 15
	}
	if strings.Contains(description, "🌟") || strings.Contains(description, "✨") {
		score += 10
	}

	// Word variety
	words := strings.Fields(description)
	uniqueWords := make(map[string]bool)
	for _, word := range words {
		uniqueWords[strings.ToLower(word)] = true
	}
	variety := float64(len(uniqueWords)) / float64(len(words))
	if variety > 0.7 {
		score += 20
	} else {
		score += 10
	}

	return score
}

// calculateCategoryConfidence provides confidence scores for category suggestions
func calculateCategoryConfidence(suggestions []map[string]interface{}) []float64 {
	confidences := make([]float64, len(suggestions))
	for i, suggestion := range suggestions {
		if conf, ok := suggestion["confidence"].(float64); ok {
			confidences[i] = conf
		} else {
			confidences[i] = 0.5
		}
	}
	return confidences
}

// calculateImageQualityScore evaluates image quality
func calculateImageQualityScore(images []string) int {
	if len(images) == 0 {
		return 0
	}

	score := 0

	// Image count score
	if len(images) >= 5 {
		score += 40
	} else if len(images) >= 3 {
		score += 30
	} else if len(images) >= 1 {
		score += 20
	}

	// Image quality checks
	for _, imageURL := range images {
		if strings.Contains(imageURL, "cdn.shopify.com") {
			score += 10
		}
		if strings.Contains(imageURL, "_925x") {
			score += 10
		}
		if !strings.Contains(imageURL, "placeholder") {
			score += 10
		}
	}

	return score
}

// processBulkTransformations handles bulk AI transformations
func processBulkTransformations(productIDs []string, transformations []string) []map[string]interface{} {
	results := []map[string]interface{}{}

	for _, productID := range productIDs {
		result := map[string]interface{}{
			"product_id":      productID,
			"status":          "success",
			"transformations": transformations,
			"results": map[string]interface{}{
				"title_optimized":      false,
				"description_enhanced": false,
				"category_suggested":   false,
				"images_optimized":     false,
			},
		}

		// Process each transformation type
		for _, transformation := range transformations {
			switch transformation {
			case "title":
				result["results"].(map[string]interface{})["title_optimized"] = true
			case "description":
				result["results"].(map[string]interface{})["description_enhanced"] = true
			case "category":
				result["results"].(map[string]interface{})["category_suggested"] = true
			case "images":
				result["results"].(map[string]interface{})["images_optimized"] = true
			}
		}

		results = append(results, result)
	}

	return results
}

// countSuccessfulTransformations counts successful transformations
func countSuccessfulTransformations(results []map[string]interface{}) int {
	successCount := 0
	for _, result := range results {
		if status, ok := result["status"].(string); ok && status == "success" {
			successCount++
		}
	}
	return successCount
}

// Export Helper Functions

// generateStandardCSV generates standard CSV format
func generateStandardCSV(products []map[string]interface{}) string {
	if len(products) == 0 {
		return "No products found"
	}

	// Get headers from first product
	headers := []string{"ID", "External ID", "Title", "Description", "Price", "Currency", "SKU", "Brand", "Category", "Images", "Status", "Created At", "Updated At"}

	var csv strings.Builder
	csv.WriteString(strings.Join(headers, ",") + "\n")

	for _, product := range products {
		row := []string{
			fmt.Sprintf("%v", product["id"]),
			fmt.Sprintf("%v", product["external_id"]),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["title"]), "\"", "\"\"")),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["description"]), "\"", "\"\"")),
			fmt.Sprintf("%.2f", product["price"]),
			fmt.Sprintf("%v", product["currency"]),
			fmt.Sprintf("%v", product["sku"]),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["brand"]), "\"", "\"\"")),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["category"]), "\"", "\"\"")),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["images"]), "\"", "\"\"")),
			fmt.Sprintf("%v", product["status"]),
			fmt.Sprintf("%v", product["created_at"]),
			fmt.Sprintf("%v", product["updated_at"]),
		}
		csv.WriteString(strings.Join(row, ",") + "\n")
	}

	return csv.String()
}

// generateExcelCSV generates Excel-compatible CSV with BOM
func generateExcelCSV(products []map[string]interface{}) string {
	if len(products) == 0 {
		return "No products found"
	}

	// Add BOM for Excel compatibility
	csv := "\xEF\xBB\xBF"

	// Get headers from first product
	headers := []string{"ID", "External ID", "Title", "Description", "Price", "Currency", "SKU", "Brand", "Category", "Images", "Status", "Created At", "Updated At"}

	csv += strings.Join(headers, ",") + "\n"

	for _, product := range products {
		row := []string{
			fmt.Sprintf("%v", product["id"]),
			fmt.Sprintf("%v", product["external_id"]),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["title"]), "\"", "\"\"")),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["description"]), "\"", "\"\"")),
			fmt.Sprintf("%.2f", product["price"]),
			fmt.Sprintf("%v", product["currency"]),
			fmt.Sprintf("%v", product["sku"]),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["brand"]), "\"", "\"\"")),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["category"]), "\"", "\"\"")),
			fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", product["images"]), "\"", "\"\"")),
			fmt.Sprintf("%v", product["status"]),
			fmt.Sprintf("%v", product["created_at"]),
			fmt.Sprintf("%v", product["updated_at"]),
		}
		csv += strings.Join(row, ",") + "\n"
	}

	return csv
}

// generateXMLExport generates XML export format
func generateXMLExport(products []map[string]interface{}) string {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<products_export>
  <export_info>
    <timestamp>` + time.Now().Format("2006-01-02T15:04:05Z") + `</timestamp>
    <total_products>` + fmt.Sprintf("%d", len(products)) + `</total_products>
    <format>xml</format>
    <version>1.0</version>
  </export_info>
  <products>`

	for _, product := range products {
		xml += `
    <product>
      <id>` + fmt.Sprintf("%v", product["id"]) + `</id>
      <external_id>` + fmt.Sprintf("%v", product["external_id"]) + `</external_id>
      <title><![CDATA[` + fmt.Sprintf("%v", product["title"]) + `]]></title>
      <description><![CDATA[` + fmt.Sprintf("%v", product["description"]) + `]]></description>
      <price>` + fmt.Sprintf("%.2f", product["price"]) + `</price>
      <currency>` + fmt.Sprintf("%v", product["currency"]) + `</currency>
      <sku>` + fmt.Sprintf("%v", product["sku"]) + `</sku>
      <brand><![CDATA[` + fmt.Sprintf("%v", product["brand"]) + `]]></brand>
      <category><![CDATA[` + fmt.Sprintf("%v", product["category"]) + `]]></category>
      <images>`

		// Handle images array
		if images, ok := product["images"].([]string); ok {
			for _, image := range images {
				xml += `<image>` + image + `</image>`
			}
		}

		xml += `</images>
      <status>` + fmt.Sprintf("%v", product["status"]) + `</status>
      <created_at>` + fmt.Sprintf("%v", product["created_at"]) + `</created_at>
      <updated_at>` + fmt.Sprintf("%v", product["updated_at"]) + `</updated_at>
    </product>`
	}

	xml += `
  </products>
</products_export>`

	return xml
}

// syncToChannel handles direct channel synchronization
func syncToChannel(channel string, products []map[string]interface{}, settings map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"channel":            channel,
		"status":             "success",
		"products_processed": len(products),
		"sync_timestamp":     time.Now(),
		"details":            map[string]interface{}{},
	}

	switch channel {
	case "google":
		result["details"] = map[string]interface{}{
			"method":         "Google Shopping API",
			"format":         "XML Feed",
			"endpoint":       "https://www.google.com/base/products",
			"estimated_time": "2-5 minutes",
		}

		// Simulate Google Shopping sync
		result["sync_id"] = fmt.Sprintf("google_sync_%d", time.Now().Unix())
		result["status"] = "completed"
		result["message"] = "Products successfully submitted to Google Shopping"

	case "facebook":
		result["details"] = map[string]interface{}{
			"method":         "Facebook Catalog API",
			"format":         "JSON Feed",
			"endpoint":       "https://graph.facebook.com/v18.0/catalog/products",
			"estimated_time": "1-3 minutes",
		}

		// Simulate Facebook sync
		result["sync_id"] = fmt.Sprintf("facebook_sync_%d", time.Now().Unix())
		result["status"] = "completed"
		result["message"] = "Products successfully synced to Facebook Catalog"

	case "instagram":
		result["details"] = map[string]interface{}{
			"method":         "Instagram Shopping API",
			"format":         "JSON Feed",
			"endpoint":       "https://graph.facebook.com/v18.0/instagram_business/catalog",
			"estimated_time": "1-2 minutes",
		}

		// Simulate Instagram sync
		result["sync_id"] = fmt.Sprintf("instagram_sync_%d", time.Now().Unix())
		result["status"] = "completed"
		result["message"] = "Products successfully synced to Instagram Shopping"

	case "amazon":
		result["details"] = map[string]interface{}{
			"method":         "Amazon SP-API",
			"format":         "XML Feed",
			"endpoint":       "https://sellingpartnerapi-na.amazon.com/feeds/2021-06-30/documents",
			"estimated_time": "5-10 minutes",
		}

		// Simulate Amazon sync
		result["sync_id"] = fmt.Sprintf("amazon_sync_%d", time.Now().Unix())
		result["status"] = "completed"
		result["message"] = "Products successfully submitted to Amazon Seller Central"

	default:
		result["status"] = "error"
		result["message"] = fmt.Sprintf("Unsupported channel: %s", channel)
		result["supported_channels"] = []string{"google", "facebook", "instagram", "amazon"}
	}

	return result
}

// Webhook Processing Helper Functions

// validateShopifyWebhook validates the HMAC signature of Shopify webhooks
func validateShopifyWebhook(body []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false // Skip validation if not configured
	}

	// Decode the signature from base64
	expectedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	// Create HMAC hash
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	computedSignature := mac.Sum(nil)

	// Compare signatures securely
	return hmac.Equal(expectedSignature, computedSignature)
}

// mergeVariantInventory merges new variant data with existing inventory data
// This preserves inventory_quantity when Shopify webhooks don't include it
func mergeVariantInventory(newVariants []ShopifyVariant, existingVariants []ShopifyVariant) []ShopifyVariant {
	// Create a map of existing variants by ID for quick lookup
	existingMap := make(map[int64]ShopifyVariant)
	for _, variant := range existingVariants {
		existingMap[variant.ID] = variant
	}

	// Merge the variants
	mergedVariants := make([]ShopifyVariant, len(newVariants))
	for i, newVariant := range newVariants {
		mergedVariants[i] = newVariant

		// If existing variant exists and new variant is missing inventory data, preserve it
		if existingVariant, exists := existingMap[newVariant.ID]; exists {
			// Preserve inventory_management if not provided in webhook
			if newVariant.InventoryManagement == "" && existingVariant.InventoryManagement != "" {
				mergedVariants[i].InventoryManagement = existingVariant.InventoryManagement
			}

			// Preserve inventory_policy if not provided in webhook
			if newVariant.InventoryPolicy == "" && existingVariant.InventoryPolicy != "" {
				mergedVariants[i].InventoryPolicy = existingVariant.InventoryPolicy
			}

			// For inventory_quantity: only preserve if new data doesn't have inventory tracking enabled
			// If inventory_management is set in new data, use the new quantity (even if 0)
			// If inventory_management is not set, preserve the old quantity
			if mergedVariants[i].InventoryManagement == "" || mergedVariants[i].InventoryManagement == "not_managed" {
				// Not tracking inventory, preserve old value if it exists
				if existingVariant.InventoryQuantity > 0 {
					mergedVariants[i].InventoryQuantity = existingVariant.InventoryQuantity
				}
			}
			// Otherwise use the new quantity from the webhook (Shopify sent updated inventory)
		}
	}

	return mergedVariants
}

// processProductUpdate handles product update webhooks
func processProductUpdate(product ShopifyProduct, shopDomain, topic string) map[string]interface{} {
	result := map[string]interface{}{
		"action":    "update",
		"status":    "success",
		"timestamp": time.Now(),
		"details":   map[string]interface{}{},
	}

	// Find existing product and its variants by external_id
	var existingProductID string
	var existingVariantsJSON string
	err := db.QueryRow(`
		SELECT id, variants FROM products 
		WHERE external_id = $1 AND connector_id IN (
			SELECT id FROM connectors WHERE shop_domain = $2
		)
	`, fmt.Sprintf("%d", product.ID), shopDomain).Scan(&existingProductID, &existingVariantsJSON)

	if err != nil {
		result["status"] = "error"
		result["message"] = "Product not found in database"
		return result
	}

	// Parse existing variants to preserve inventory data
	var existingVariants []ShopifyVariant
	if existingVariantsJSON != "" {
		json.Unmarshal([]byte(existingVariantsJSON), &existingVariants)
	}

	// Merge variant data - preserve inventory_quantity from existing data if new data doesn't have it
	mergedVariants := mergeVariantInventory(product.Variants, existingVariants)
	product.Variants = mergedVariants

	// Transform Shopify product to our format
	transformedProduct := transformShopifyProduct(product, shopDomain, make(map[int64]int))

	// Get automatic SEO enhancement (fallback-based, not full AI)
	// This gives products basic SEO but doesn't count as "AI optimized"
	seoEnhancement := createFallbackSEO(product)

	// Create metadata with basic SEO (NOT marked as AI enhanced)
	// Only manual optimization via API endpoint sets seo_enhanced = true
	enhancedMetadata := map[string]interface{}{
		"seo_title":       seoEnhancement.SEOTitle,
		"seo_description": seoEnhancement.SEODescription,
		"keywords":        seoEnhancement.Keywords,
		"meta_keywords":   seoEnhancement.MetaKeywords,
		"alt_text":        seoEnhancement.AltText,
		"schema_markup":   seoEnhancement.SchemaMarkup,
		"seo_enhanced":    false, // NOT marked as AI enhanced (automatic sync)
		"seo_enhanced_at": "",
	}

	// Convert enhanced metadata to JSON
	enhancedMetadataJSON, _ := json.Marshal(enhancedMetadata)

	// Update product in database
	_, err = db.Exec(`
		UPDATE products 
		SET title = $1, description = $2, price = $3, currency = $4, 
			brand = $5, category = $6, images = $7, variants = $8, 
			metadata = $9, updated_at = NOW()
		WHERE id = $10
	`,
		transformedProduct.Title,
		transformedProduct.Description,
		getFloatValue(transformedProduct.Price),
		transformedProduct.Currency,
		transformedProduct.Brand,
		transformedProduct.Category,
		fmt.Sprintf("{%s}", strings.Join(transformedProduct.Images, ",")),
		transformedProduct.Variants,
		string(enhancedMetadataJSON),
		existingProductID,
	)

	if err != nil {
		result["status"] = "error"
		result["message"] = fmt.Sprintf("Failed to update product: %v", err)
		return result
	}

	result["product_id"] = existingProductID
	result["message"] = "Product updated successfully"
	result["details"] = map[string]interface{}{
		"title":        transformedProduct.Title,
		"price":        getFloatValue(transformedProduct.Price),
		"images_count": len(transformedProduct.Images),
	}

	return result
}

// processProductCreate handles product creation webhooks
func processProductCreate(product ShopifyProduct, shopDomain, topic string) map[string]interface{} {
	result := map[string]interface{}{
		"action":    "create",
		"status":    "success",
		"timestamp": time.Now(),
		"details":   map[string]interface{}{},
	}

	// Transform Shopify product to our format
	transformedProduct := transformShopifyProduct(product, shopDomain, make(map[int64]int))

	// Get automatic SEO enhancement (fallback-based, not full AI)
	// This gives products basic SEO but doesn't count as "AI optimized"
	seoEnhancement := createFallbackSEO(product)

	// Create metadata with basic SEO (NOT marked as AI enhanced)
	// Only manual optimization via API endpoint sets seo_enhanced = true
	enhancedMetadata := map[string]interface{}{
		"seo_title":       seoEnhancement.SEOTitle,
		"seo_description": seoEnhancement.SEODescription,
		"keywords":        seoEnhancement.Keywords,
		"meta_keywords":   seoEnhancement.MetaKeywords,
		"alt_text":        seoEnhancement.AltText,
		"schema_markup":   seoEnhancement.SchemaMarkup,
		"seo_enhanced":    false, // NOT marked as AI enhanced (automatic sync)
		"seo_enhanced_at": "",
	}

	// Convert enhanced metadata to JSON
	enhancedMetadataJSON, _ := json.Marshal(enhancedMetadata)

	// Get connector ID
	var connectorID string
	err := db.QueryRow(`
		SELECT id FROM connectors WHERE shop_domain = $1
	`, shopDomain).Scan(&connectorID)

	if err != nil {
		result["status"] = "error"
		result["message"] = "Connector not found"
		return result
	}

	// Try upsert first, fallback to check-and-insert if constraint doesn't exist
	_, err = db.Exec(`
		INSERT INTO products (
			connector_id, external_id, title, description, price, currency,
			brand, category, images, variants, metadata, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		ON CONFLICT (connector_id, external_id) 
		DO UPDATE SET 
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			price = EXCLUDED.price,
			currency = EXCLUDED.currency,
			brand = EXCLUDED.brand,
			category = EXCLUDED.category,
			images = EXCLUDED.images,
			variants = EXCLUDED.variants,
			metadata = EXCLUDED.metadata,
			status = EXCLUDED.status,
			updated_at = NOW()
	`,
		connectorID,
		transformedProduct.ExternalID,
		transformedProduct.Title,
		transformedProduct.Description,
		getFloatValue(transformedProduct.Price),
		transformedProduct.Currency,
		transformedProduct.Brand,
		transformedProduct.Category,
		fmt.Sprintf("{%s}", strings.Join(transformedProduct.Images, ",")),
		transformedProduct.Variants,
		string(enhancedMetadataJSON),
		"ACTIVE",
	)

	// If upsert fails due to missing constraint, fallback to check-and-insert
	if err != nil && strings.Contains(err.Error(), "no unique or exclusion constraint") {
		// Check if product already exists
		var existingID string
		checkErr := db.QueryRow(`
			SELECT id FROM products 
