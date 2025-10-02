package handlers

import (
	"net/http"
	"strconv"

	"lister/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type IssueHandler struct {
	db     *gorm.DB
	logger interface{}
}

func NewIssueHandler(db *gorm.DB, logger interface{}) *IssueHandler {
	return &IssueHandler{
		db:     db,
		logger: logger,
	}
}

func (h *IssueHandler) List(c *gin.Context) {
	var issues []models.Issue

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Filters
	severity := c.Query("severity")
	channel := c.Query("channel")
	resolved := c.Query("resolved")

	query := h.db.Model(&models.Issue{})

	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	if channel != "" {
		query = query.Where("channel = ?", channel)
	}

	if resolved != "" {
		if resolved == "true" {
			query = query.Where("is_resolved = ?", true)
		} else if resolved == "false" {
			query = query.Where("is_resolved = ?", false)
		}
	}

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Find(&issues).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch issues"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": issues,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func (h *IssueHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var issue models.Issue
	if err := h.db.First(&issue, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch issue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": issue})
}

func (h *IssueHandler) Resolve(c *gin.Context) {
	id := c.Param("id")

	var issue models.Issue
	if err := h.db.First(&issue, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch issue"})
		return
	}

	issue.IsResolved = true
	if err := h.db.Save(&issue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve issue"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": issue})
}
