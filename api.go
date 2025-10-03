package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
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
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Price string `json:"price"`
	SKU   string `json:"sku"`
}

type ShopifyMetafield struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

var (
	db      *sql.DB
	dbMutex sync.Mutex
	// Temporary in-memory storage for Vercel demo
	connectors     []Connector
	connectorMutex sync.RWMutex
)

// initDB initializes the database connection
func initDB() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db != nil {
		return nil // Already initialized
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}

	// Test the connection
	if err = db.Ping(); err != nil {
		return err
	}

	// Create all required tables
	tables := []string{
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
	}

	// Execute all table creation statements
	for _, tableSQL := range tables {
		_, err = db.Exec(tableSQL)
		if err != nil {
			return fmt.Errorf("failed to create table: %v", err)
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
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse JSON response
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}

	if tokenResponse.AccessToken == "" {
		return "", fmt.Errorf("no access token in response: %s", string(body))
	}

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

	// Build API URL with proper format (fetch all products)
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

// Handler is the main entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// Initialize database connection
	if err := initDB(); err != nil {
		http.Error(w, fmt.Sprintf("Database initialization failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create a simple router
	router := gin.New()

	// Add basic middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Shopify-Topic, X-Shopify-Shop-Domain, X-Shopify-Hmac-Sha256")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Lister API is running",
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// App Proxy routes (for Custom Apps)
	proxy := router.Group("/api/v1/shopify/proxy")
	{
		// App Proxy Install
		proxy.GET("/install", func(c *gin.Context) {
			// Get shop domain from query params
			shop := c.Query("shop")
			if shop == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Missing shop parameter"})
				return
			}

			// Get Shopify credentials from environment
			clientID := os.Getenv("SHOPIFY_CLIENT_ID")
			if clientID == "" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Shopify client ID not configured"})
				return
			}

			// Generate OAuth URL for App Proxy
			scopes := "read_products,write_products,read_inventory,write_inventory,read_shop"
			state := fmt.Sprintf("%d", time.Now().Unix())

			// Clean the shop domain
			cleanDomain := shop
			if strings.HasSuffix(shop, ".myshopify.com") {
				cleanDomain = strings.TrimSuffix(shop, ".myshopify.com")
			}

			// App Proxy callback URL
			redirectURI := fmt.Sprintf("https://%s/apps/lister/api/callback", shop)

			authURL := fmt.Sprintf(
				"https://%s.myshopify.com/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s&state=%s",
				cleanDomain,
				clientID,
				scopes,
				redirectURI,
				state,
			)

			// Return HTML page with redirect
			c.Header("Content-Type", "text/html")
			c.String(200, `
				<!DOCTYPE html>
				<html>
				<head>
					<title>Installing Lister App</title>
				</head>
				<body>
					<h2>Installing Lister App...</h2>
					<p>Redirecting to Shopify for authentication...</p>
					<script>
						window.location.href = "%s";
					</script>
					<p><a href="%s">Click here if not redirected automatically</a></p>
				</body>
				</html>
			`, authURL, authURL)
		})

		// App Proxy Callback
		proxy.GET("/callback", func(c *gin.Context) {
			code := c.Query("code")
			state := c.Query("state")
			shop := c.Query("shop")

			if code == "" || state == "" || shop == "" {
				c.Header("Content-Type", "text/html")
				c.String(400, `
					<!DOCTYPE html>
					<html>
					<head><title>Error</title></head>
					<body>
						<h2>Installation Error</h2>
						<p>Missing required parameters. Please try again.</p>
					</body>
					</html>
				`)
				return
			}

			// Get Shopify credentials
			clientID := os.Getenv("SHOPIFY_CLIENT_ID")
			clientSecret := os.Getenv("SHOPIFY_CLIENT_SECRET")

			if clientID == "" || clientSecret == "" {
				c.Header("Content-Type", "text/html")
				c.String(500, `
					<!DOCTYPE html>
					<html>
					<head><title>Error</title></head>
					<body>
						<h2>Configuration Error</h2>
						<p>Shopify credentials not configured properly.</p>
					</body>
					</html>
				`)
				return
			}

			// For demo purposes, create a mock access token
			// In production, you would exchange the code for a real access token
			mockAccessToken := fmt.Sprintf("mock_token_%d", time.Now().Unix())
			connectorID := fmt.Sprintf("connector_%d", time.Now().Unix())

			// Save connector to Supabase database
			_, err := db.Exec(`
				INSERT INTO connectors (id, name, type, status, shop_domain, access_token, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				ON CONFLICT (id) DO UPDATE SET
					name = EXCLUDED.name,
					status = EXCLUDED.status,
					access_token = EXCLUDED.access_token
			`, connectorID, fmt.Sprintf("Shopify Store - %s", shop), "SHOPIFY", "ACTIVE", shop, mockAccessToken, time.Now())

			if err != nil {
				c.Header("Content-Type", "text/html")
				c.String(500, `
					<!DOCTYPE html>
					<html>
					<head><title>Database Error</title></head>
					<body>
						<h2>❌ Installation Failed</h2>
						<p>Failed to save connector to database.</p>
					</body>
					</html>
				`)
				return
			}

			// Success page
			c.Header("Content-Type", "text/html")
			c.String(200, `
				<!DOCTYPE html>
				<html>
				<head><title>Installation Successful</title></head>
				<body>
					<h2>✅ Lister App Installed Successfully!</h2>
					<p><strong>Shop:</strong> %s</p>
					<p><strong>Status:</strong> Connected</p>
					<p><strong>Connector ID:</strong> %s</p>
					<p>You can now close this window and return to your Shopify admin.</p>
					<p><a href="/api/v1/connectors">View Connectors</a></p>
				</body>
				</html>
			`, shop, connectorID)
		})
	}

	// API routes
	api := router.Group("/api/v1")
	{
		// Products Management
		products := api.Group("/products")
		{
			// List all products with pagination and filtering
			products.GET("/", func(c *gin.Context) {
				// Get query parameters
				page := c.DefaultQuery("page", "1")
				limit := c.DefaultQuery("limit", "20")
				search := c.Query("search")
				category := c.Query("category")
				status := c.DefaultQuery("status", "ACTIVE")
				
				// Convert to integers
				pageInt := 1
				limitInt := 20
				if p, err := fmt.Sscanf(page, "%d", &pageInt); err == nil && p == 1 {
					// Page converted successfully
				}
				if l, err := fmt.Sscanf(limit, "%d", &limitInt); err == nil && l == 1 {
					// Limit converted successfully
				}
				
				// Calculate offset
				offset := (pageInt - 1) * limitInt
				
				// Build query
				whereClause := "WHERE status = $1"
				args := []interface{}{status}
				argIndex := 2
				
				if search != "" {
					whereClause += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d OR brand ILIKE $%d)", argIndex, argIndex, argIndex)
					args = append(args, "%"+search+"%")
					argIndex++
				}
				
				if category != "" {
					whereClause += fmt.Sprintf(" AND category = $%d", argIndex)
					args = append(args, category)
					argIndex++
				}
				
				// Get total count
				countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products %s", whereClause)
				var totalCount int
				err := db.QueryRow(countQuery, args...).Scan(&totalCount)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count products"})
					return
				}
				
				// Get products
				query := fmt.Sprintf(`
					SELECT id, external_id, title, description, price, currency, sku, brand, category, 
						   images, variants, metadata, status, created_at, updated_at
					FROM products %s 
					ORDER BY created_at DESC 
					LIMIT $%d OFFSET $%d
				`, whereClause, argIndex, argIndex+1)
				
				args = append(args, limitInt, offset)
				
				rows, err := db.Query(query, args...)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query products"})
					return
				}
				defer rows.Close()
				
				var products []map[string]interface{}
				for rows.Next() {
					var id, externalID, title, description, sku, brand, category, status string
					var price float64
					var currency string
					var images []string
					var variants, metadata string
					var createdAt, updatedAt time.Time
					
					err := rows.Scan(&id, &externalID, &title, &description, &price, &currency, &sku, &brand, &category, 
									&images, &variants, &metadata, &status, &createdAt, &updatedAt)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan product"})
						return
					}
					
					products = append(products, map[string]interface{}{
						"id":          id,
						"external_id":  externalID,
						"title":       title,
						"description": description,
						"price":       price,
						"currency":    currency,
						"sku":         sku,
						"brand":       brand,
						"category":    category,
						"images":      images,
						"variants":    variants,
						"metadata":    metadata,
						"status":      status,
						"created_at":  createdAt,
						"updated_at":  updatedAt,
					})
				}
				
				// Calculate pagination info
				totalPages := (totalCount + limitInt - 1) / limitInt
				
				c.JSON(http.StatusOK, gin.H{
					"data": products,
					"pagination": gin.H{
						"page":         pageInt,
						"limit":        limitInt,
						"total":        totalCount,
						"total_pages":  totalPages,
						"has_next":     pageInt < totalPages,
						"has_prev":     pageInt > 1,
					},
					"message": "Products retrieved successfully",
				})
			})
			
			// Get single product by ID
			products.GET("/:id", func(c *gin.Context) {
				productID := c.Param("id")
				
				var id, externalID, title, description, sku, brand, category, status string
				var price float64
				var currency string
				var images []string
				var variants, metadata string
				var createdAt, updatedAt time.Time
				
				err := db.QueryRow(`
					SELECT id, external_id, title, description, price, currency, sku, brand, category, 
						   images, variants, metadata, status, created_at, updated_at
					FROM products 
					WHERE id = $1
				`, productID).Scan(&id, &externalID, &title, &description, &price, &currency, &sku, &brand, &category, 
									&images, &variants, &metadata, &status, &createdAt, &updatedAt)
				
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
					return
				}
				
				c.JSON(http.StatusOK, gin.H{
					"data": map[string]interface{}{
						"id":          id,
						"external_id":  externalID,
						"title":       title,
						"description": description,
						"price":       price,
						"currency":    currency,
						"sku":         sku,
						"brand":       brand,
						"category":    category,
						"images":      images,
						"variants":    variants,
						"metadata":    metadata,
						"status":      status,
						"created_at":  createdAt,
						"updated_at":  updatedAt,
					},
					"message": "Product retrieved successfully",
				})
			})
			
			// Update product
			products.PUT("/:id", func(c *gin.Context) {
				productID := c.Param("id")
				
				var request struct {
					Title       string  `json:"title"`
					Description string  `json:"description"`
					Price       float64 `json:"price"`
					SKU         string  `json:"sku"`
					Brand       string  `json:"brand"`
					Category    string  `json:"category"`
					Status      string  `json:"status"`
				}
				
				if err := c.ShouldBindJSON(&request); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				
				_, err := db.Exec(`
					UPDATE products 
					SET title = $1, description = $2, price = $3, sku = $4, brand = $5, category = $6, status = $7, updated_at = NOW()
					WHERE id = $8
				`, request.Title, request.Description, request.Price, request.SKU, request.Brand, request.Category, request.Status, productID)
				
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
					return
				}
				
				c.JSON(http.StatusOK, gin.H{
					"message": "Product updated successfully",
				})
			})
			
			// Get product statistics
			products.GET("/stats", func(c *gin.Context) {
				var stats struct {
					TotalProducts    int     `json:"total_products"`
					ActiveProducts  int     `json:"active_products"`
					AveragePrice    float64 `json:"average_price"`
					Categories      int     `json:"categories"`
					Brands          int     `json:"brands"`
				}
				
				// Get total products
				db.QueryRow("SELECT COUNT(*) FROM products").Scan(&stats.TotalProducts)
				
				// Get active products
				db.QueryRow("SELECT COUNT(*) FROM products WHERE status = 'ACTIVE'").Scan(&stats.ActiveProducts)
				
				// Get average price
				db.QueryRow("SELECT AVG(price) FROM products WHERE price > 0").Scan(&stats.AveragePrice)
				
				// Get unique categories
				db.QueryRow("SELECT COUNT(DISTINCT category) FROM products WHERE category IS NOT NULL AND category != ''").Scan(&stats.Categories)
				
				// Get unique brands
				db.QueryRow("SELECT COUNT(DISTINCT brand) FROM products WHERE brand IS NOT NULL AND brand != ''").Scan(&stats.Brands)
				
				c.JSON(http.StatusOK, gin.H{
					"data":    stats,
					"message": "Product statistics retrieved successfully",
				})
			})
		}

		// Connectors
		api.GET("/connectors", func(c *gin.Context) {
			// Query connectors from Supabase database
			rows, err := db.Query("SELECT id, name, type, status, shop_domain, created_at, last_sync FROM connectors ORDER BY created_at DESC")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query connectors"})
				return
			}
			defer rows.Close()

			var connectors []map[string]interface{}
			for rows.Next() {
				var id, name, connectorType, status, shopDomain string
				var createdAt time.Time
				var lastSync *time.Time

				err := rows.Scan(&id, &name, &connectorType, &status, &shopDomain, &createdAt, &lastSync)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan connector"})
					return
				}

				connectors = append(connectors, map[string]interface{}{
					"id":          id,
					"name":        name,
					"type":        connectorType,
					"status":      status,
					"shop_domain": shopDomain,
					"created_at":  createdAt,
					"last_sync":   lastSync,
				})
			}

			c.JSON(200, gin.H{
				"data":    connectors,
				"message": "Connectors retrieved successfully",
			})
		})

		// Shopify routes
		shopify := api.Group("/shopify")
		{
			// Shopify OAuth Install
			shopify.POST("/install", func(c *gin.Context) {
				var request struct {
					ShopDomain  string `json:"shop_domain" binding:"required"`
					RedirectURI string `json:"redirect_uri" binding:"required"`
				}

				if err := c.ShouldBindJSON(&request); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				// Get Shopify credentials from environment
				clientID := os.Getenv("SHOPIFY_CLIENT_ID")
				if clientID == "" {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Shopify client ID not configured"})
					return
				}

				// Generate OAuth URL
				scopes := "read_products,write_products,read_inventory,write_inventory,read_shop"
				state := fmt.Sprintf("%d", time.Now().Unix())

				// Clean the shop domain (remove .myshopify.com if present)
				cleanDomain := request.ShopDomain
				if strings.HasSuffix(request.ShopDomain, ".myshopify.com") {
					cleanDomain = strings.TrimSuffix(request.ShopDomain, ".myshopify.com")
				}

				authURL := fmt.Sprintf(
					"https://%s.myshopify.com/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s&state=%s",
					cleanDomain,
					clientID,
					scopes,
					request.RedirectURI,
					state,
				)

				c.JSON(http.StatusOK, gin.H{
					"auth_url": authURL,
					"state":    state,
					"message":  "Redirect user to the auth_url to complete OAuth flow",
				})
			})

			// Shopify OAuth Callback
			shopify.GET("/callback", func(c *gin.Context) {
				code := c.Query("code")
				state := c.Query("state")
				shop := c.Query("shop")

				if code == "" || state == "" || shop == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
					return
				}

				// Get Shopify credentials
				clientID := os.Getenv("SHOPIFY_CLIENT_ID")
				clientSecret := os.Getenv("SHOPIFY_CLIENT_SECRET")

				if clientID == "" || clientSecret == "" {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Shopify credentials not configured"})
					return
				}

				// Exchange authorization code for access token
				accessToken, err := exchangeCodeForToken(code, shop, clientID, clientSecret)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code for token"})
					return
				}

				connectorID := fmt.Sprintf("connector_%d", time.Now().Unix())

				// Save connector to Supabase database
				_, err = db.Exec(`
					INSERT INTO connectors (id, name, type, status, shop_domain, access_token, created_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					ON CONFLICT (id) DO UPDATE SET
						name = EXCLUDED.name,
						status = EXCLUDED.status,
						access_token = EXCLUDED.access_token
				`, connectorID, fmt.Sprintf("Shopify Store - %s", shop), "SHOPIFY", "ACTIVE", shop, accessToken, time.Now())

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save connector to database"})
					return
				}

				// Return success with connector info
				c.JSON(http.StatusOK, gin.H{
					"message":      "Shopify store connected successfully",
					"shop":         shop,
					"state":        state,
					"connector_id": connectorID,
					"note":         "Real access token obtained and stored",
				})
			})

			// Shopify Webhook
			shopify.POST("/webhook", func(c *gin.Context) {
				// Get webhook topic
				topic := c.GetHeader("X-Shopify-Topic")
				shopDomain := c.GetHeader("X-Shopify-Shop-Domain")
				_ = c.GetHeader("X-Shopify-Hmac-Sha256") // Signature validation placeholder

				if topic == "" || shopDomain == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required headers"})
					return
				}

				// Read the payload
				payload, err := c.GetRawData()
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read payload"})
					return
				}

				// Process webhook based on topic
				switch topic {
				case "products/create", "products/update":
					var productData map[string]interface{}
					if err := json.Unmarshal(payload, &productData); err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
						return
					}

					c.JSON(http.StatusOK, gin.H{
						"message":    "Product webhook processed",
						"topic":      topic,
						"shop":       shopDomain,
						"product_id": productData["id"],
					})

				case "products/delete":
					c.JSON(http.StatusOK, gin.H{
						"message": "Product delete webhook processed",
						"topic":   topic,
						"shop":    shopDomain,
					})

				default:
					c.JSON(http.StatusOK, gin.H{
						"message": "Webhook received but not processed",
						"topic":   topic,
					})
				}
			})

			// Product Sync
			shopify.POST("/:id/sync", func(c *gin.Context) {
				connectorID := c.Param("id")

				// Get connector from database
				var connector struct {
					ID          string
					ShopDomain  string
					AccessToken string
				}

				err := db.QueryRow("SELECT id, shop_domain, access_token FROM connectors WHERE id = $1", connectorID).Scan(
					&connector.ID, &connector.ShopDomain, &connector.AccessToken)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "Connector not found"})
					return
				}

				// Fetch products from Shopify
				products, err := fetchShopifyProducts(connector.ShopDomain, connector.AccessToken)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products from Shopify"})
					return
				}

				// Store products in database
				syncedCount := 0
				var errors []string

				for _, product := range products {
					// Extract first variant price and SKU
					var price float64
					var sku string
					if len(product.Variants) > 0 {
						// Convert price string to float
						if p, err := fmt.Sscanf(product.Variants[0].Price, "%f", &price); err == nil && p == 1 {
							// Price converted successfully
						}
						sku = product.Variants[0].SKU
					}

					// Extract image URLs and convert to PostgreSQL array format
					var imageURLs []string
					for _, img := range product.Images {
						imageURLs = append(imageURLs, img.URL)
					}

					// Convert variants to JSON
					variantsJSON, _ := json.Marshal(product.Variants)
					metafieldsJSON, _ := json.Marshal(product.Metafields)

					// Convert Go slice to PostgreSQL array format
					imageURLsArray := "{" + strings.Join(imageURLs, ",") + "}"

					_, err := db.Exec(`
						INSERT INTO products (connector_id, external_id, title, description, price, currency, sku, brand, category, images, variants, metadata, status)
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
					`, connectorID, fmt.Sprintf("%d", product.ID), product.Title, product.Description, price, "USD", sku, product.Vendor, product.ProductType,
						imageURLsArray, string(variantsJSON), string(metafieldsJSON), "ACTIVE")

					if err != nil {
						errors = append(errors, fmt.Sprintf("Product %s: %v", product.Title, err))
					} else {
						syncedCount++
					}
				}

				// Update connector last_sync
				db.Exec("UPDATE connectors SET last_sync = NOW() WHERE id = $1", connectorID)

				response := gin.H{
					"message":         "Product sync completed",
					"connector_id":    connectorID,
					"products_synced": syncedCount,
					"total_products":  len(products),
				}

				if len(errors) > 0 {
					response["errors"] = errors
					response["message"] = fmt.Sprintf("Product sync completed with %d errors", len(errors))
				}

				c.JSON(http.StatusOK, response)
			})
		}
	}

	// Serve the request
	router.ServeHTTP(w, r)
}

// This function is required by Vercel
func main() {
	// This won't be called in Vercel, but required for Go compilation
}
