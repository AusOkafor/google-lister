package handlers

import (
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
