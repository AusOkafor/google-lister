package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
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

				authURL := fmt.Sprintf(
					"https://%s.myshopify.com/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s&state=%s",
					request.ShopDomain,
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

				// Exchange code for token (simplified for demo)
				c.JSON(http.StatusOK, gin.H{
					"message": "Shopify store connected successfully",
					"shop":    shop,
					"state":   state,
					"note":    "Token exchange implementation needed",
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
