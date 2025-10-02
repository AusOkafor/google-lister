package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"lister/internal/config"
	"lister/internal/logger"
	"lister/internal/models"
	"lister/internal/services/shopify"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ShopifyHandler struct {
	db           *gorm.DB
	logger       *logger.Logger
	config       *config.Config
	oauthService *shopify.OAuthService
}

func NewShopifyHandler(db *gorm.DB, logger *logger.Logger, config *config.Config) *ShopifyHandler {
	return &ShopifyHandler{
		db:           db,
		logger:       logger,
		config:       config,
		oauthService: shopify.NewOAuthService(config, logger),
	}
}

// Install initiates the Shopify OAuth flow
func (h *ShopifyHandler) Install(c *gin.Context) {
	var request struct {
		ShopDomain  string `json:"shop_domain" binding:"required"`
		RedirectURI string `json:"redirect_uri" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate OAuth URL
	authURL, state, err := h.oauthService.GenerateAuthURL(request.ShopDomain, request.RedirectURI)
	if err != nil {
		h.logger.Error("Failed to generate auth URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authorization URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
		"message":  "Redirect user to the auth_url to complete OAuth flow",
	})
}

// Callback handles the OAuth callback
func (h *ShopifyHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	shop := c.Query("shop")

	if code == "" || state == "" || shop == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
		return
	}

	// Exchange code for access token
	tokenResp, err := h.oauthService.ExchangeCodeForToken(shop, code)
	if err != nil {
		h.logger.Error("Failed to exchange code for token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange authorization code"})
		return
	}

	// Create Shopify client to get shop info
	client := shopify.NewClient(shop, tokenResp.AccessToken, h.logger)
	shopInfo, err := client.GetShopInfo()
	if err != nil {
		h.logger.Error("Failed to get shop info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get shop information"})
		return
	}

	// Save connector to database
	connector := &models.Connector{
		Name:   shopInfo.Name,
		Type:   "SHOPIFY",
		Status: "ACTIVE",
		Config: map[string]interface{}{
			"shop_domain": shop,
			"shop_id":     shopInfo.ID,
			"email":       shopInfo.Email,
			"currency":    shopInfo.Currency,
			"timezone":    shopInfo.Timezone,
		},
		Credentials: map[string]interface{}{
			"access_token": tokenResp.AccessToken,
			"scope":        tokenResp.Scope,
		},
	}

	if err := h.db.Create(connector).Error; err != nil {
		h.logger.Error("Failed to save connector: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save connector"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Shopify store connected successfully",
		"connector_id": connector.ID,
		"shop_name":    shopInfo.Name,
	})
}

// SyncProducts syncs products from Shopify
func (h *ShopifyHandler) SyncProducts(c *gin.Context) {
	connectorID := c.Param("id")

	var connector models.Connector
	if err := h.db.First(&connector, "id = ?", connectorID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Connector not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connector"})
		return
	}

	if connector.Type != "SHOPIFY" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Connector is not a Shopify connector"})
		return
	}

	// Extract credentials
	accessToken, ok := connector.Credentials["access_token"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid access token"})
		return
	}

	shopDomain, ok := connector.Config["shop_domain"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shop domain"})
		return
	}

	// Create Shopify client
	client := shopify.NewClient(shopDomain, accessToken, h.logger)
	transformer := shopify.NewTransformer()

	// Sync products
	var syncedCount int
	pageInfo := ""
	limit := 50

	for {
		productsResp, err := client.GetProducts(limit, pageInfo)
		if err != nil {
			h.logger.Error("Failed to fetch products: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products from Shopify"})
			return
		}

		// Transform and save each product
		for _, shopifyProduct := range productsResp.Products {
			canonicalProduct, err := transformer.TransformProduct(&shopifyProduct)
			if err != nil {
				h.logger.Error("Failed to transform product %d: %v", shopifyProduct.ID, err)
				continue
			}

			// Check if product already exists
			var existingProduct models.Product
			err = h.db.Where("external_id = ?", canonicalProduct.ExternalID).First(&existingProduct).Error

			if err == gorm.ErrRecordNotFound {
				// Create new product
				if err := h.db.Create(canonicalProduct).Error; err != nil {
					h.logger.Error("Failed to create product: %v", err)
					continue
				}
			} else if err == nil {
				// Update existing product
				canonicalProduct.ID = existingProduct.ID
				if err := h.db.Save(canonicalProduct).Error; err != nil {
					h.logger.Error("Failed to update product: %v", err)
					continue
				}
			} else {
				h.logger.Error("Database error: %v", err)
				continue
			}

			syncedCount++
		}

		// Check if there are more pages
		if productsResp.Link == nil {
			break
		}
		pageInfo = *productsResp.Link
	}

	// Update connector last sync time
	now := time.Now()
	connector.LastSync = &now
	h.db.Save(&connector)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Products synced successfully",
		"synced_count": syncedCount,
	})
}

// Webhook handles Shopify webhooks
func (h *ShopifyHandler) Webhook(c *gin.Context) {
	// Get webhook topic
	topic := c.GetHeader("X-Shopify-Topic")
	shopDomain := c.GetHeader("X-Shopify-Shop-Domain")
	signature := c.GetHeader("X-Shopify-Hmac-Sha256")

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

	// Validate webhook signature (implement proper HMAC validation)
	// For now, we'll skip validation in development
	if h.config.Env == "production" {
		// TODO: Implement proper webhook signature validation
		_ = signature // Suppress unused variable warning
	}

	// Process webhook based on topic
	switch topic {
	case "products/create", "products/update":
		err = h.handleProductWebhook(payload, shopDomain)
	case "products/delete":
		err = h.handleProductDeleteWebhook(payload, shopDomain)
	default:
		h.logger.Debug("Unhandled webhook topic: %s", topic)
		c.JSON(http.StatusOK, gin.H{"message": "Webhook received but not processed"})
		return
	}

	if err != nil {
		h.logger.Error("Failed to process webhook: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}

// handleProductWebhook processes product create/update webhooks
func (h *ShopifyHandler) handleProductWebhook(payload []byte, shopDomain string) error {
	var webhookProduct shopify.WebhookPayload
	if err := json.Unmarshal(payload, &webhookProduct); err != nil {
		return fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	// Convert webhook payload to Product struct
	product := &shopify.Product{
		ID:          webhookProduct.ID,
		Title:       webhookProduct.Title,
		BodyHTML:    webhookProduct.BodyHTML,
		Vendor:      webhookProduct.Vendor,
		ProductType: webhookProduct.ProductType,
		Handle:      webhookProduct.Handle,
		Status:      webhookProduct.Status,
		Tags:        webhookProduct.Tags,
		Variants:    webhookProduct.Variants,
		Images:      webhookProduct.Images,
		Options:     webhookProduct.Options,
		CreatedAt:   webhookProduct.CreatedAt,
		UpdatedAt:   webhookProduct.UpdatedAt,
		PublishedAt: webhookProduct.PublishedAt,
	}

	// Transform to canonical format
	transformer := shopify.NewTransformer()
	canonicalProduct, err := transformer.TransformProduct(product)
	if err != nil {
		return fmt.Errorf("failed to transform product: %w", err)
	}

	// Save or update product
	var existingProduct models.Product
	err = h.db.Where("external_id = ?", canonicalProduct.ExternalID).First(&existingProduct).Error

	if err == gorm.ErrRecordNotFound {
		// Create new product
		return h.db.Create(canonicalProduct).Error
	} else if err == nil {
		// Update existing product
		canonicalProduct.ID = existingProduct.ID
		return h.db.Save(canonicalProduct).Error
	}

	return err
}

// handleProductDeleteWebhook processes product delete webhooks
func (h *ShopifyHandler) handleProductDeleteWebhook(payload []byte, shopDomain string) error {
	var webhookProduct shopify.WebhookPayload
	if err := json.Unmarshal(payload, &webhookProduct); err != nil {
		return fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	// Delete the product
	externalID := fmt.Sprintf("shopify_%d", webhookProduct.ID)
	return h.db.Where("external_id = ?", externalID).Delete(&models.Product{}).Error
}
