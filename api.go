package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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

var (
	connectors       []Connector
	connectorMutex   sync.RWMutex
	connectorCounter int
)

// Handler is the main entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
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

			// Create and save connector
			connectorMutex.Lock()
			connectorCounter++
			connector := Connector{
				ID:          fmt.Sprintf("connector_%d", connectorCounter),
				Name:        fmt.Sprintf("Shopify Store - %s", shop),
				Type:        "SHOPIFY",
				Status:      "ACTIVE",
				ShopDomain:  shop,
				AccessToken: mockAccessToken,
				CreatedAt:   time.Now(),
			}
			connectors = append(connectors, connector)
			connectorMutex.Unlock()

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
			`, shop, connector.ID)
		})
	}

	// API routes
	api := router.Group("/api/v1")
	{
		// Products
		api.GET("/products", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"data":    []interface{}{},
				"message": "Products endpoint - ready for database integration",
			})
		})

		// Connectors
		api.GET("/connectors", func(c *gin.Context) {
			connectorMutex.RLock()
			defer connectorMutex.RUnlock()

			// Return connectors without sensitive data
			var safeConnectors []map[string]interface{}
			for _, conn := range connectors {
				safeConnectors = append(safeConnectors, map[string]interface{}{
					"id":          conn.ID,
					"name":        conn.Name,
					"type":        conn.Type,
					"status":      conn.Status,
					"shop_domain": conn.ShopDomain,
					"created_at":  conn.CreatedAt,
					"last_sync":   conn.LastSync,
				})
			}

			c.JSON(200, gin.H{
				"data":    safeConnectors,
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

				// For demo purposes, create a mock access token
				// In production, you would exchange the code for a real access token
				mockAccessToken := fmt.Sprintf("mock_token_%d", time.Now().Unix())

				// Create and save connector
				connectorMutex.Lock()
				connectorCounter++
				connector := Connector{
					ID:          fmt.Sprintf("connector_%d", connectorCounter),
					Name:        fmt.Sprintf("Shopify Store - %s", shop),
					Type:        "SHOPIFY",
					Status:      "ACTIVE",
					ShopDomain:  shop,
					AccessToken: mockAccessToken,
					CreatedAt:   time.Now(),
				}
				connectors = append(connectors, connector)
				connectorMutex.Unlock()

				// Return success with connector info
				c.JSON(http.StatusOK, gin.H{
					"message":      "Shopify store connected successfully",
					"shop":         shop,
					"state":        state,
					"connector_id": connector.ID,
					"note":         "Token exchange implementation needed",
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

				c.JSON(http.StatusOK, gin.H{
					"message":      "Product sync initiated",
					"connector_id": connectorID,
					"note":         "Database integration needed for full sync",
				})
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
