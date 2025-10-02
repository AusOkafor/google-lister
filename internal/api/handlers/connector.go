package handlers

import (
	"net/http"

	"lister/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConnectorHandler struct {
	db     *gorm.DB
	logger interface{}
}

func NewConnectorHandler(db *gorm.DB, logger interface{}) *ConnectorHandler {
	return &ConnectorHandler{
		db:     db,
		logger: logger,
	}
}

func (h *ConnectorHandler) List(c *gin.Context) {
	var connectors []models.Connector

	if err := h.db.Find(&connectors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connectors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": connectors})
}

func (h *ConnectorHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var connector models.Connector
	if err := h.db.First(&connector, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Connector not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connector"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": connector})
}

func (h *ConnectorHandler) Create(c *gin.Context) {
	var connector models.Connector
	if err := c.ShouldBindJSON(&connector); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Create(&connector).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create connector"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": connector})
}

func (h *ConnectorHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var connector models.Connector
	if err := h.db.First(&connector, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Connector not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connector"})
		return
	}

	if err := c.ShouldBindJSON(&connector); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Save(&connector).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update connector"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": connector})
}

func (h *ConnectorHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.Delete(&models.Connector{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete connector"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *ConnectorHandler) Sync(c *gin.Context) {
	id := c.Param("id")

	var connector models.Connector
	if err := h.db.First(&connector, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Connector not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connector"})
		return
	}

	// TODO: Implement actual sync logic
	// This would trigger the sync process for the connector

	c.JSON(http.StatusOK, gin.H{"message": "Sync started"})
}
