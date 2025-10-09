package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lister/internal/config"
	"lister/internal/logger"
	"lister/internal/models"
	"lister/internal/worker/processors/ai"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OptimizerHandler handles AI optimization requests
type OptimizerHandler struct {
	db        *gorm.DB
	logger    *logger.Logger
	optimizer *ai.Optimizer
	config    *config.Config
}

// NewOptimizerHandler creates a new optimizer handler
func NewOptimizerHandler(db *gorm.DB, log *logger.Logger, cfg *config.Config) *OptimizerHandler {
	return &OptimizerHandler{
		db:        db,
		logger:    log,
		optimizer: ai.New(cfg, log),
		config:    cfg,
	}
}

// OptimizeTitle optimizes a product title
// POST /api/v1/optimizer/title
func (h *OptimizerHandler) OptimizeTitle(c *gin.Context) {
	var req models.OptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get organization ID from context (set by auth middleware)
	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		// For development, use a default org ID if not authenticated
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
		return
	}

	// Check AI credits
	if err := h.checkAndDeductCredits(orgUUID, 1); err != nil {
		h.logger.Info("Insufficient AI credits for organization: %s", organizationID)
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient AI credits", "details": err.Error()})
		return
	}

	// Get product
	productUUID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := h.db.First(&product, "id = ?", productUUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		h.logger.Error("Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		return
	}

	// Get AI settings
	settings, err := h.getAISettings(orgUUID)
	if err != nil {
		h.logger.Error("Failed to get AI settings: %v", err)
		// Use default settings
		settings = h.getDefaultAISettings(orgUUID)
	}

	// Prepare product data for optimization
	description := ""
	if product.Description != nil {
		description = *product.Description
	}
	brand := ""
	if product.Brand != nil {
		brand = *product.Brand
	}
	category := ""
	if product.Category != nil {
		category = *product.Category
	}

	productData := map[string]interface{}{
		"title":        product.Title,
		"description":  description,
		"brand":        brand,
		"category":     category,
		"keywords":     req.Keywords,
		"max_length":   req.MaxLength,
		"strategy":     req.Strategy,
		"instructions": req.CustomInstructions,
	}

	// Call AI optimizer
	startTime := time.Now()
	optimizedTitle, err := h.optimizer.OptimizeTitle(productData)
	duration := time.Since(startTime)

	// Calculate cost (approximate based on tokens)
	estimatedTokens := len(product.Title) + len(description) + 200
	cost := h.calculateCost(settings.DefaultModel, estimatedTokens)

	// Create optimization history record
	history := &models.OptimizationHistory{
		ProductID:        productUUID,
		OrganizationID:   orgUUID,
		OptimizationType: models.OptimizationTypeTitle,
		OriginalValue:    product.Title,
		OptimizedValue:   optimizedTitle,
		Status:           models.OptimizationStatusPending,
		AIModel:          settings.DefaultModel,
		Cost:             cost,
		TokensUsed:       estimatedTokens,
		Metadata: models.JSONB{
			"strategy":     req.Strategy,
			"keywords":     req.Keywords,
			"max_length":   req.MaxLength,
			"duration_ms":  duration.Milliseconds(),
			"instructions": req.CustomInstructions,
		},
	}

	if err != nil {
		h.logger.Error("Title optimization failed: %v", err)
		history.Status = models.OptimizationStatusFailed
		errorMsg := err.Error()
		history.ErrorMessage = &errorMsg

		// Save failed attempt
		if dbErr := h.db.Create(history).Error; dbErr != nil {
			h.logger.Error("Failed to save optimization history: %v", dbErr)
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Optimization failed",
			"details": err.Error(),
		})
		return
	}

	// Calculate score and improvement
	score := h.calculateTitleScore(optimizedTitle, product.Title)
	improvement := h.calculateImprovement(product.Title, optimizedTitle)
	history.Score = &score
	history.ImprovementPercentage = &improvement

	// Save optimization history
	if err := h.db.Create(history).Error; err != nil {
		h.logger.Error("Failed to save optimization history: %v", err)
	}

	// Update AI credits with cost
	h.updateCreditsCost(orgUUID, cost, true)

	// Prepare response
	response := models.OptimizationResponse{
		OptimizationID:   history.ID.String(),
		ProductID:        req.ProductID,
		OptimizationType: string(models.OptimizationTypeTitle),
		OriginalValue:    product.Title,
		OptimizedValue:   optimizedTitle,
		Score:            score,
		Improvement:      improvement,
		Cost:             cost,
		TokensUsed:       estimatedTokens,
		AIModel:          settings.DefaultModel,
		Status:           string(history.Status),
		Message:          "Title optimized successfully",
		Metadata: map[string]interface{}{
			"duration_ms":     duration.Milliseconds(),
			"character_count": len(optimizedTitle),
		},
	}

	c.JSON(http.StatusOK, response)
}

// OptimizeDescription optimizes a product description
// POST /api/v1/optimizer/description
func (h *OptimizerHandler) OptimizeDescription(c *gin.Context) {
	var req models.OptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	// Check credits
	if err := h.checkAndDeductCredits(orgUUID, 2); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient AI credits"})
		return
	}

	// Get product
	productUUID, _ := uuid.Parse(req.ProductID)
	var product models.Product
	if err := h.db.First(&product, "id = ?", productUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Get settings
	settings, _ := h.getAISettings(orgUUID)
	if settings == nil {
		settings = h.getDefaultAISettings(orgUUID)
	}

	// Get string values from pointers
	description := ""
	if product.Description != nil {
		description = *product.Description
	}
	brand := ""
	if product.Brand != nil {
		brand = *product.Brand
	}
	category := ""
	if product.Category != nil {
		category = *product.Category
	}

	// Prepare product data
	productData := map[string]interface{}{
		"title":           product.Title,
		"description":     description,
		"brand":           brand,
		"category":        category,
		"style":           req.Style,
		"length":          req.Length,
		"target_audience": req.TargetAudience,
		"instructions":    req.CustomInstructions,
	}

	// Optimize description
	startTime := time.Now()
	optimizedDesc, err := h.optimizer.OptimizeDescription(productData)
	duration := time.Since(startTime)

	estimatedTokens := len(description) + 300
	cost := h.calculateCost(settings.DefaultModel, estimatedTokens)

	// Create history record
	history := &models.OptimizationHistory{
		ProductID:        productUUID,
		OrganizationID:   orgUUID,
		OptimizationType: models.OptimizationTypeDescription,
		OriginalValue:    description,
		OptimizedValue:   optimizedDesc,
		Status:           models.OptimizationStatusPending,
		AIModel:          settings.DefaultModel,
		Cost:             cost,
		TokensUsed:       estimatedTokens,
		Metadata: models.JSONB{
			"style":           req.Style,
			"length":          req.Length,
			"target_audience": req.TargetAudience,
			"duration_ms":     duration.Milliseconds(),
		},
	}

	if err != nil {
		h.logger.Error("Description optimization failed: %v", err)
		history.Status = models.OptimizationStatusFailed
		errorMsg := err.Error()
		history.ErrorMessage = &errorMsg
		h.db.Create(history)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Optimization failed"})
		return
	}

	score := h.calculateDescriptionScore(optimizedDesc)
	improvement := h.calculateImprovement(description, optimizedDesc)
	history.Score = &score
	history.ImprovementPercentage = &improvement

	h.db.Create(history)
	h.updateCreditsCost(orgUUID, cost, true)

	response := models.OptimizationResponse{
		OptimizationID:   history.ID.String(),
		ProductID:        req.ProductID,
		OptimizationType: string(models.OptimizationTypeDescription),
		OriginalValue:    description,
		OptimizedValue:   optimizedDesc,
		Score:            score,
		Improvement:      improvement,
		Cost:             cost,
		TokensUsed:       estimatedTokens,
		AIModel:          settings.DefaultModel,
		Status:           string(history.Status),
		Message:          "Description optimized successfully",
	}

	c.JSON(http.StatusOK, response)
}

// SuggestCategory suggests product categories
// POST /api/v1/optimizer/category
func (h *OptimizerHandler) SuggestCategory(c *gin.Context) {
	var req models.OptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	// Check credits
	if err := h.checkAndDeductCredits(orgUUID, 1); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient AI credits"})
		return
	}

	// Get product
	productUUID, _ := uuid.Parse(req.ProductID)
	var product models.Product
	if err := h.db.First(&product, "id = ?", productUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Get settings
	settings, _ := h.getAISettings(orgUUID)
	if settings == nil {
		settings = h.getDefaultAISettings(orgUUID)
	}

	// Get string values from pointers
	description := ""
	if product.Description != nil {
		description = *product.Description
	}
	brand := ""
	if product.Brand != nil {
		brand = *product.Brand
	}
	category := ""
	if product.Category != nil {
		category = *product.Category
	}

	// Prepare product data
	productData := map[string]interface{}{
		"title":       product.Title,
		"description": description,
		"brand":       brand,
		"category":    category,
	}

	// Suggest category
	startTime := time.Now()
	suggestedCategory, err := h.optimizer.SuggestCategory(productData)
	duration := time.Since(startTime)

	estimatedTokens := 150
	cost := h.calculateCost(settings.DefaultModel, estimatedTokens)

	// Create history
	history := &models.OptimizationHistory{
		ProductID:        productUUID,
		OrganizationID:   orgUUID,
		OptimizationType: models.OptimizationTypeCategory,
		OriginalValue:    category,
		OptimizedValue:   suggestedCategory,
		Status:           models.OptimizationStatusPending,
		AIModel:          settings.DefaultModel,
		Cost:             cost,
		TokensUsed:       estimatedTokens,
		Metadata: models.JSONB{
			"duration_ms": duration.Milliseconds(),
		},
	}

	if err != nil {
		h.logger.Error("Category suggestion failed: %v", err)
		history.Status = models.OptimizationStatusFailed
		errorMsg := err.Error()
		history.ErrorMessage = &errorMsg
		h.db.Create(history)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Suggestion failed"})
		return
	}

	score := 85 // Default score for category
	history.Score = &score

	h.db.Create(history)
	h.updateCreditsCost(orgUUID, cost, true)

	// Generate multiple suggestions (mock for now)
	suggestions := []map[string]interface{}{
		{
			"category":   suggestedCategory,
			"confidence": 95,
			"channels":   []string{"Google Shopping", "Facebook", "Instagram"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"optimization_id":  history.ID.String(),
		"product_id":       req.ProductID,
		"current_category": product.Category,
		"suggestions":      suggestions,
		"cost":             cost,
		"message":          "Category suggestions generated successfully",
	})
}

// AnalyzeImages analyzes product images
// POST /api/v1/optimizer/image
func (h *OptimizerHandler) AnalyzeImages(c *gin.Context) {
	var req models.OptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	// Check credits (images cost more)
	if err := h.checkAndDeductCredits(orgUUID, 3); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient AI credits"})
		return
	}

	// Get product
	productUUID, _ := uuid.Parse(req.ProductID)
	var product models.Product
	if err := h.db.First(&product, "id = ?", productUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Get images (they're already a slice, no need to parse)
	images := product.Images
	if images == nil {
		images = []string{}
	}

	// For now, return mock analysis
	// TODO: Integrate with actual image analysis API
	analysis := map[string]interface{}{
		"overall_score": 85,
		"quality_metrics": map[string]int{
			"resolution":     92,
			"composition":    78,
			"lighting":       82,
			"color_accuracy": 88,
		},
		"images": []map[string]interface{}{
			{
				"url":             images[0],
				"score":           85,
				"issues":          []string{"Low contrast"},
				"recommendations": []string{"Increase contrast", "Better lighting"},
			},
		},
		"recommendations": []map[string]interface{}{
			{
				"type":        "quality",
				"title":       "Improve Image Quality",
				"priority":    "high",
				"description": "Enhance resolution and sharpness",
			},
		},
	}

	cost := h.calculateCost("gpt-4-vision", 1000)

	// Create history
	history := &models.OptimizationHistory{
		ProductID:        productUUID,
		OrganizationID:   orgUUID,
		OptimizationType: models.OptimizationTypeImage,
		OriginalValue:    fmt.Sprintf("%d images", len(images)),
		OptimizedValue:   "Image analysis completed",
		Status:           models.OptimizationStatusPending,
		AIModel:          "gpt-4-vision",
		Cost:             cost,
		TokensUsed:       1000,
		Metadata:         models.JSONB(analysis),
	}

	h.db.Create(history)
	h.updateCreditsCost(orgUUID, cost, true)

	c.JSON(http.StatusOK, gin.H{
		"optimization_id": history.ID.String(),
		"product_id":      req.ProductID,
		"analysis":        analysis,
		"cost":            cost,
		"message":         "Image analysis completed successfully",
	})
}

// BulkOptimize performs bulk optimization on multiple products
// POST /api/v1/optimizer/bulk
func (h *OptimizerHandler) BulkOptimize(c *gin.Context) {
	var req models.BulkOptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	// Check credits (bulk operations require more credits)
	creditsNeeded := len(req.ProductIDs) * 2
	if err := h.checkAndDeductCredits(orgUUID, creditsNeeded); err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":          "Insufficient AI credits",
			"credits_needed": creditsNeeded,
		})
		return
	}

	// Process each product
	results := make([]map[string]interface{}, 0)
	successCount := 0

	for _, productID := range req.ProductIDs {
		productUUID, err := uuid.Parse(productID)
		if err != nil {
			results = append(results, map[string]interface{}{
				"product_id": productID,
				"status":     "failed",
				"error":      "Invalid product ID",
			})
			continue
		}

		var product models.Product
		if err := h.db.First(&product, "id = ?", productUUID).Error; err != nil {
			results = append(results, map[string]interface{}{
				"product_id": productID,
				"status":     "failed",
				"error":      "Product not found",
			})
			continue
		}

		// Get string values from pointers
		description := ""
		if product.Description != nil {
			description = *product.Description
		}
		brand := ""
		if product.Brand != nil {
			brand = *product.Brand
		}
		category := ""
		if product.Category != nil {
			category = *product.Category
		}

		// Perform optimization based on type
		var optimizedValue string
		var optimizationErr error

		productData := map[string]interface{}{
			"title":       product.Title,
			"description": description,
			"brand":       brand,
			"category":    category,
		}

		switch req.OptimizationType {
		case models.OptimizationTypeTitle:
			optimizedValue, optimizationErr = h.optimizer.OptimizeTitle(productData)
		case models.OptimizationTypeDescription:
			optimizedValue, optimizationErr = h.optimizer.OptimizeDescription(productData)
		case models.OptimizationTypeCategory:
			optimizedValue, optimizationErr = h.optimizer.SuggestCategory(productData)
		default:
			optimizationErr = errors.New("unsupported optimization type")
		}

		if optimizationErr != nil {
			results = append(results, map[string]interface{}{
				"product_id": productID,
				"status":     "failed",
				"error":      optimizationErr.Error(),
			})
			continue
		}

		// Save optimization history
		history := &models.OptimizationHistory{
			ProductID:        productUUID,
			OrganizationID:   orgUUID,
			OptimizationType: req.OptimizationType,
			OriginalValue:    product.Title, // Adjust based on type
			OptimizedValue:   optimizedValue,
			Status:           models.OptimizationStatusPending,
			AIModel:          "gpt-3.5-turbo",
			Cost:             0.002,
			TokensUsed:       200,
		}

		h.db.Create(history)
		successCount++

		results = append(results, map[string]interface{}{
			"product_id":      productID,
			"status":          "success",
			"optimization_id": history.ID.String(),
			"optimized_value": optimizedValue,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"processed_products": len(req.ProductIDs),
		"success_count":      successCount,
		"failed_count":       len(req.ProductIDs) - successCount,
		"results":            results,
		"message":            "Bulk optimization completed",
	})
}

// GetHistory retrieves optimization history
// GET /api/v1/optimizer/history
func (h *OptimizerHandler) GetHistory(c *gin.Context) {
	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	productID := c.Query("product_id")
	optimizationType := c.Query("type")
	status := c.Query("status")

	// Build query
	query := h.db.Model(&models.OptimizationHistory{}).Where("organization_id = ?", orgUUID)

	if productID != "" {
		query = query.Where("product_id = ?", productID)
	}
	if optimizationType != "" {
		query = query.Where("optimization_type = ?", optimizationType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get history records
	var history []models.OptimizationHistory
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&history).Error; err != nil {
		h.logger.Error("Failed to fetch optimization history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": history,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetAnalytics retrieves optimization analytics
// GET /api/v1/optimizer/analytics
func (h *OptimizerHandler) GetAnalytics(c *gin.Context) {
	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	// Get overall analytics
	var analytics struct {
		TotalOptimizations int64
		AppliedCount       int64
		PendingCount       int64
		FailedCount        int64
		AvgScore           sql.NullFloat64
		AvgImprovement     sql.NullFloat64
		TotalCost          float64
		TotalTokens        int64
	}

	h.db.Model(&models.OptimizationHistory{}).
		Where("organization_id = ?", orgUUID).
		Select(`
			COUNT(*) as total_optimizations,
			COUNT(*) FILTER (WHERE status = 'applied') as applied_count,
			COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
			AVG(score) FILTER (WHERE status = 'applied') as avg_score,
			AVG(improvement_percentage) FILTER (WHERE status = 'applied') as avg_improvement,
			SUM(cost) as total_cost,
			SUM(tokens_used) as total_tokens
		`).
		Scan(&analytics)

	// Get by type
	var byType []struct {
		OptimizationType string
		Count            int64
		AvgScore         sql.NullFloat64
		TotalCost        float64
	}

	h.db.Model(&models.OptimizationHistory{}).
		Where("organization_id = ?", orgUUID).
		Select("optimization_type, COUNT(*) as count, AVG(score) as avg_score, SUM(cost) as total_cost").
		Group("optimization_type").
		Scan(&byType)

	c.JSON(http.StatusOK, gin.H{
		"overview": gin.H{
			"total_optimizations": analytics.TotalOptimizations,
			"applied_count":       analytics.AppliedCount,
			"pending_count":       analytics.PendingCount,
			"failed_count":        analytics.FailedCount,
			"avg_score":           analytics.AvgScore.Float64,
			"avg_improvement":     analytics.AvgImprovement.Float64,
			"total_cost":          analytics.TotalCost,
			"total_tokens":        analytics.TotalTokens,
			"success_rate":        float64(analytics.AppliedCount) / float64(analytics.TotalOptimizations) * 100,
		},
		"by_type": byType,
	})
}

// GetSettings retrieves AI settings for the organization
// GET /api/v1/optimizer/settings
func (h *OptimizerHandler) GetSettings(c *gin.Context) {
	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	settings, err := h.getAISettings(orgUUID)
	if err != nil {
		h.logger.Error("Failed to get AI settings: %v", err)
		// Return default settings
		settings = h.getDefaultAISettings(orgUUID)
	}

	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// UpdateSettings updates AI settings for the organization
// PUT /api/v1/optimizer/settings
func (h *OptimizerHandler) UpdateSettings(c *gin.Context) {
	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	var req models.AISettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate settings
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.OrganizationID = orgUUID

	// Upsert settings
	var existing models.AISettings
	result := h.db.Where("organization_id = ?", orgUUID).First(&existing)

	if result.Error == nil {
		// Update existing
		req.ID = existing.ID
		if err := h.db.Save(&req).Error; err != nil {
			h.logger.Error("Failed to update AI settings: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
			return
		}
	} else {
		// Create new
		if err := h.db.Create(&req).Error; err != nil {
			h.logger.Error("Failed to create AI settings: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create settings"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully",
		"data":    req,
	})
}

// GetCredits retrieves AI credits information
// GET /api/v1/optimizer/credits
func (h *OptimizerHandler) GetCredits(c *gin.Context) {
	organizationID := c.GetString("organization_id")
	if organizationID == "" {
		organizationID = "00000000-0000-0000-0000-000000000000"
	}

	orgUUID, _ := uuid.Parse(organizationID)

	var credits models.AICredits
	if err := h.db.Where("organization_id = ?", orgUUID).First(&credits).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Initialize credits for organization
			credits = models.AICredits{
				OrganizationID:   orgUUID,
				CreditsRemaining: 2500,
				CreditsTotal:     2500,
				ResetDate:        time.Now().AddDate(0, 1, 0),
			}
			h.db.Create(&credits)
		} else {
			h.logger.Error("Failed to get AI credits: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch credits"})
			return
		}
	}

	// Check if credits should be reset
	if credits.ShouldReset() {
		credits.Reset()
		h.db.Save(&credits)
	}

	c.JSON(http.StatusOK, gin.H{"data": credits})
}

// ApplyOptimization applies an optimization to a product
// POST /api/v1/optimizer/:id/apply
func (h *OptimizerHandler) ApplyOptimization(c *gin.Context) {
	optimizationID := c.Param("id")

	orgID := c.GetString("organization_id")
	if orgID == "" {
		orgID = "00000000-0000-0000-0000-000000000000"
	}

	var history models.OptimizationHistory
	if err := h.db.First(&history, "id = ?", optimizationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Optimization not found"})
		return
	}

	// Update product with optimized value
	var product models.Product
	if err := h.db.First(&product, "id = ?", history.ProductID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Apply based on type
	switch history.OptimizationType {
	case models.OptimizationTypeTitle:
		product.Title = history.OptimizedValue
	case models.OptimizationTypeDescription:
		descValue := history.OptimizedValue
		product.Description = &descValue
	case models.OptimizationTypeCategory:
		catValue := history.OptimizedValue
		product.Category = &catValue
	}

	if err := h.db.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply optimization"})
		return
	}

	// Update history status
	now := time.Now()
	history.Status = models.OptimizationStatusApplied
	history.AppliedAt = &now
	h.db.Save(&history)

	// Update credits statistics
	h.updateCreditsSuccess(history.OrganizationID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Optimization applied successfully",
		"data":    history,
	})
}

// Helper methods

func (h *OptimizerHandler) getAISettings(organizationID uuid.UUID) (*models.AISettings, error) {
	var settings models.AISettings
	if err := h.db.Where("organization_id = ?", organizationID).First(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func (h *OptimizerHandler) getDefaultAISettings(organizationID uuid.UUID) *models.AISettings {
	return &models.AISettings{
		OrganizationID:          organizationID,
		DefaultModel:            "gpt-3.5-turbo",
		MaxTokens:               500,
		Temperature:             0.7,
		TopP:                    0.9,
		TitleOptimization:       true,
		DescriptionOptimization: true,
		CategoryOptimization:    true,
		ImageOptimization:       true,
		MinScoreThreshold:       80,
		RequireApproval:         true,
		MaxRetries:              3,
	}
}

func (h *OptimizerHandler) checkAndDeductCredits(organizationID uuid.UUID, amount int) error {
	var credits models.AICredits
	if err := h.db.Where("organization_id = ?", organizationID).First(&credits).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Initialize credits
			credits = models.AICredits{
				OrganizationID:   organizationID,
				CreditsRemaining: 2500,
				CreditsTotal:     2500,
				ResetDate:        time.Now().AddDate(0, 1, 0),
			}
			h.db.Create(&credits)
			return nil
		}
		return err
	}

	// Check if reset is needed
	if credits.ShouldReset() {
		credits.Reset()
	}

	// Deduct credits
	if err := credits.DeductCredits(amount); err != nil {
		return err
	}

	credits.TotalOptimizations++
	return h.db.Save(&credits).Error
}

func (h *OptimizerHandler) updateCreditsCost(organizationID uuid.UUID, cost float64, success bool) {
	var credits models.AICredits
	if err := h.db.Where("organization_id = ?", organizationID).First(&credits).Error; err != nil {
		return
	}

	credits.AddCost(cost)
	if success {
		credits.SuccessfulOptimizations++
	} else {
		credits.FailedOptimizations++
	}

	h.db.Save(&credits)
}

func (h *OptimizerHandler) updateCreditsSuccess(organizationID uuid.UUID) {
	var credits models.AICredits
	if err := h.db.Where("organization_id = ?", organizationID).First(&credits).Error; err != nil {
		return
	}

	credits.SuccessfulOptimizations++
	h.db.Save(&credits)
}

func (h *OptimizerHandler) calculateCost(model string, tokens int) float64 {
	rates := map[string]float64{
		"gpt-4":         0.03 / 1000,
		"gpt-4-vision":  0.04 / 1000,
		"gpt-3.5-turbo": 0.002 / 1000,
		"claude-3":      0.015 / 1000,
		"dall-e":        0.04 / 1000,
	}

	rate, exists := rates[model]
	if !exists {
		rate = 0.002 / 1000 // Default rate
	}

	return float64(tokens) * rate
}

func (h *OptimizerHandler) calculateTitleScore(optimized, original string) int {
	score := 0

	// Length check (50-60 optimal for SEO)
	optLen := len(optimized)
	if optLen >= 50 && optLen <= 60 {
		score += 25
	} else if optLen > 30 && optLen < 80 {
		score += 15
	} else {
		score += 5
	}

	// Check if title is different from original
	if strings.ToLower(optimized) != strings.ToLower(original) {
		score += 15
	}

	// Check for keywords (simple heuristic)
	words := strings.Fields(optimized)
	if len(words) >= 5 {
		score += 20
	}

	// Check for capital letters (proper formatting)
	if optimized != strings.ToUpper(optimized) && optimized != strings.ToLower(optimized) {
		score += 15
	}

	// Check for special characters (moderate use)
	specialCount := strings.Count(optimized, "-") + strings.Count(optimized, "|") + strings.Count(optimized, "·")
	if specialCount > 0 && specialCount <= 3 {
		score += 10
	}

	// Check for numbers (product specs)
	hasNumbers := strings.ContainsAny(optimized, "0123456789")
	if hasNumbers {
		score += 15
	}

	// Ensure score is between 0-100
	if score > 100 {
		score = 100
	}

	return score
}

func (h *OptimizerHandler) calculateDescriptionScore(description string) int {
	score := 0

	// Length check
	length := len(description)
	if length >= 150 && length <= 300 {
		score += 30
	} else if length > 100 && length < 500 {
		score += 20
	} else {
		score += 10
	}

	// Sentence count
	sentences := strings.Count(description, ".") + strings.Count(description, "!") + strings.Count(description, "?")
	if sentences >= 3 && sentences <= 8 {
		score += 20
	}

	// Check for bullets or lists
	hasBullets := strings.Contains(description, "•") || strings.Contains(description, "-") || strings.Contains(description, "*")
	if hasBullets {
		score += 15
	}

	// Check for key product terms
	hasFeatures := strings.Contains(strings.ToLower(description), "feature") ||
		strings.Contains(strings.ToLower(description), "benefit") ||
		strings.Contains(strings.ToLower(description), "quality")
	if hasFeatures {
		score += 15
	}

	// Check for call to action
	hasCTA := strings.Contains(strings.ToLower(description), "buy") ||
		strings.Contains(strings.ToLower(description), "order") ||
		strings.Contains(strings.ToLower(description), "get") ||
		strings.Contains(strings.ToLower(description), "shop")
	if hasCTA {
		score += 20
	}

	if score > 100 {
		score = 100
	}

	return score
}

func (h *OptimizerHandler) calculateImprovement(original, optimized string) float64 {
	if original == "" {
		return 100.0
	}

	// Simple improvement calculation based on length and quality indicators
	improvementFactor := 1.0

	// Length improvement
	if len(optimized) > len(original) {
		improvementFactor += 0.1
	}

	// Quality indicators
	if strings.Contains(optimized, "|") || strings.Contains(optimized, "·") {
		improvementFactor += 0.05
	}

	if len(strings.Fields(optimized)) > len(strings.Fields(original)) {
		improvementFactor += 0.1
	}

	// Calculate percentage
	improvement := (improvementFactor - 1.0) * 100
	if improvement > 100 {
		improvement = 100
	}
	if improvement < 0 {
		improvement = 0
	}

	return improvement
}
