package handler

import (
	"net/http"

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
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

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
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Products
		api.GET("/products", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"data":    []interface{}{},
				"message": "Products endpoint - ready for implementation",
			})
		})

		// Shopify routes
		shopify := api.Group("/shopify")
		{
			shopify.POST("/install", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Shopify install endpoint - ready for implementation",
				})
			})

			shopify.GET("/callback", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Shopify callback endpoint - ready for implementation",
				})
			})

			shopify.POST("/webhook", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Shopify webhook endpoint - ready for implementation",
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
