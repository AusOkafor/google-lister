package handlers

import (
	"fmt"
	"net/http"

	"lister/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChannelHandler struct {
	db     *gorm.DB
	logger interface{}
}

func NewChannelHandler(db *gorm.DB, logger interface{}) *ChannelHandler {
	return &ChannelHandler{
		db:     db,
		logger: logger,
	}
}

func (h *ChannelHandler) List(c *gin.Context) {
	var channels []models.Channel

	if err := h.db.Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": channels})
}

func (h *ChannelHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var channel models.Channel
	if err := h.db.First(&channel, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": channel})
}

func (h *ChannelHandler) Create(c *gin.Context) {
	var channel models.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": channel})
}

func (h *ChannelHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var channel models.Channel
	if err := h.db.First(&channel, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}

	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": channel})
}

func (h *ChannelHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&models.Channel{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *ChannelHandler) Connect(c *gin.Context) {
	var requestData struct {
		ChannelID   string `json:"channel_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Credentials struct {
			APIKey     string `json:"apiKey" binding:"required"`
			Secret     string `json:"secret" binding:"required"`
			MerchantID string `json:"merchantId"`
		} `json:"credentials" binding:"required"`
		Settings struct {
			AutoSync     bool   `json:"autoSync"`
			SyncInterval string `json:"syncInterval"`
			TestMode     bool   `json:"testMode"`
		} `json:"settings"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create or update the channel with the provided credentials
	channel := models.Channel{
		ID:   requestData.ChannelID,
		Name: requestData.Name,
		Type: models.ChannelTypeGoogleMerchantCenter, // Default type, can be made configurable
		Status: models.ChannelStatusActive,
		Config: `{"description": "` + requestData.Description + `", "autoSync": ` + 
			`"` + fmt.Sprintf("%t", requestData.Settings.AutoSync) + `", "syncInterval": "` + 
			requestData.Settings.SyncInterval + `", "testMode": "` + 
			fmt.Sprintf("%t", requestData.Settings.TestMode) + `"}`,
		Credentials: `{"apiKey": "` + requestData.Credentials.APIKey + 
			`", "secret": "` + requestData.Credentials.Secret + 
			`", "merchantId": "` + requestData.Credentials.MerchantID + `"}`,
	}

	// Check if channel already exists
	var existingChannel models.Channel
	if err := h.db.First(&existingChannel, "id = ?", requestData.ChannelID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new channel
			if err := h.db.Create(&channel).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel connection"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check channel existence"})
			return
		}
	} else {
		// Update existing channel
		existingChannel.Name = channel.Name
		existingChannel.Status = models.ChannelStatusActive
		existingChannel.Config = channel.Config
		existingChannel.Credentials = channel.Credentials
		
		if err := h.db.Save(&existingChannel).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel connection"})
			return
		}
		channel = existingChannel
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Channel connected successfully",
		"data":    channel,
	})
}

func (h *ChannelHandler) Available(c *gin.Context) {
	// Return available channel types that can be connected
	availableChannels := []map[string]interface{}{
		{
			"id":          "google-merchant-center",
			"name":        "Google Merchant Center",
			"type":        "GOOGLE_MERCHANT_CENTER",
			"description": "Connect to Google Merchant Center for product listings",
			"icon":        "google",
			"status":      "available",
		},
		{
			"id":          "bing-shopping",
			"name":        "Bing Shopping",
			"type":        "BING_SHOPPING",
			"description": "Connect to Bing Shopping for product listings",
			"icon":        "microsoft",
			"status":      "available",
		},
		{
			"id":          "meta-catalog",
			"name":        "Meta Catalog",
			"type":        "META_CATALOG",
			"description": "Connect to Meta Catalog for Facebook and Instagram shopping",
			"icon":        "facebook",
			"status":      "available",
		},
		{
			"id":          "pinterest-catalog",
			"name":        "Pinterest Catalog",
			"type":        "PINTEREST_CATALOG",
			"description": "Connect to Pinterest Catalog for product pins",
			"icon":        "pinterest",
			"status":      "available",
		},
		{
			"id":          "tiktok-shopping",
			"name":        "TikTok Shopping",
			"type":        "TIKTOK_SHOPPING",
			"description": "Connect to TikTok Shopping for product listings",
			"icon":        "tiktok",
			"status":      "available",
		},
	}

	c.JSON(http.StatusOK, gin.H{"data": availableChannels})
}

func (h *ChannelHandler) Connected(c *gin.Context) {
	var channels []models.Channel

	if err := h.db.Where("status = ?", models.ChannelStatusActive).Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connected channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": channels})
}

func (h *ChannelHandler) Test(c *gin.Context) {
	id := c.Param("id")

	var channel models.Channel
	if err := h.db.First(&channel, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}

	// TODO: Implement actual connection test logic
	// This would test the channel credentials and connection

	c.JSON(http.StatusOK, gin.H{
		"message": "Channel connection test successful",
		"data":    channel,
	})
}

func (h *ChannelHandler) Sync(c *gin.Context) {
	id := c.Param("id")

	var channel models.Channel
	if err := h.db.First(&channel, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channel"})
		return
	}

	// TODO: Implement actual sync logic
	// This would trigger the sync process for the channel

	c.JSON(http.StatusOK, gin.H{"message": "Sync started"})
}
