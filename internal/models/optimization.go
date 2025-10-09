package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// OptimizationType represents the type of optimization
type OptimizationType string

const (
	OptimizationTypeTitle       OptimizationType = "title"
	OptimizationTypeDescription OptimizationType = "description"
	OptimizationTypeCategory    OptimizationType = "category"
	OptimizationTypeImage       OptimizationType = "image"
	OptimizationTypeBulk        OptimizationType = "bulk"
)

// OptimizationStatus represents the status of an optimization
type OptimizationStatus string

const (
	OptimizationStatusPending  OptimizationStatus = "pending"
	OptimizationStatusApplied  OptimizationStatus = "applied"
	OptimizationStatusRejected OptimizationStatus = "rejected"
	OptimizationStatusFailed   OptimizationStatus = "failed"
)

// JSONB is a custom type for PostgreSQL JSONB columns
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// OptimizationHistory tracks all AI optimization attempts and results
type OptimizationHistory struct {
	ID                    uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ProductID             uuid.UUID          `gorm:"type:uuid;not null;index" json:"product_id"`
	OrganizationID        uuid.UUID          `gorm:"type:uuid;not null;index" json:"organization_id"`
	OptimizationType      OptimizationType   `gorm:"type:varchar(50);not null;index" json:"optimization_type"`
	OriginalValue         string             `gorm:"type:text" json:"original_value"`
	OptimizedValue        string             `gorm:"type:text" json:"optimized_value"`
	Status                OptimizationStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Score                 *int               `gorm:"type:integer" json:"score,omitempty"`
	ImprovementPercentage *float64           `gorm:"type:decimal(5,2)" json:"improvement_percentage,omitempty"`
	AIModel               string             `gorm:"type:varchar(50);not null" json:"ai_model"`
	Cost                  float64            `gorm:"type:decimal(10,4);default:0.0000" json:"cost"`
	TokensUsed            int                `gorm:"type:integer;default:0" json:"tokens_used"`
	Metadata              JSONB              `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	ErrorMessage          *string            `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt             time.Time          `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
	UpdatedAt             time.Time          `gorm:"type:timestamp with time zone;default:now()" json:"updated_at"`
	AppliedAt             *time.Time         `gorm:"type:timestamp with time zone" json:"applied_at,omitempty"`

	// Relations
	Product      *Product      `gorm:"foreignKey:ProductID;references:ID" json:"product,omitempty"`
	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID" json:"organization,omitempty"`
}

// TableName specifies the table name for OptimizationHistory
func (OptimizationHistory) TableName() string {
	return "optimization_history"
}

// BeforeCreate hook to generate UUID if not set
func (o *OptimizationHistory) BeforeCreate() error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// AISettings stores AI optimization configuration per organization
type AISettings struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;unique;not null" json:"organization_id"`

	// General Settings
	DefaultModel     string  `gorm:"type:varchar(50);default:'gpt-3.5-turbo'" json:"default_model"`
	AutoOptimize     bool    `gorm:"default:false" json:"auto_optimize"`
	AutoApply        bool    `gorm:"default:false" json:"auto_apply"`
	MaxCostPerMonth  float64 `gorm:"type:decimal(10,2);default:25.00" json:"max_cost_per_month"`
	Notifications    bool    `gorm:"default:true" json:"notifications"`

	// Model Parameters
	MaxTokens   int     `gorm:"type:integer;default:500" json:"max_tokens"`
	Temperature float64 `gorm:"type:decimal(3,2);default:0.70" json:"temperature"`
	TopP        float64 `gorm:"type:decimal(3,2);default:0.90" json:"top_p"`

	// Feature Toggles
	TitleOptimization       bool `gorm:"default:true" json:"title_optimization"`
	DescriptionOptimization bool `gorm:"default:true" json:"description_optimization"`
	CategoryOptimization    bool `gorm:"default:true" json:"category_optimization"`
	ImageOptimization       bool `gorm:"default:true" json:"image_optimization"`

	// Quality Settings
	MinScoreThreshold int  `gorm:"type:integer;default:80" json:"min_score_threshold"`
	RequireApproval   bool `gorm:"default:true" json:"require_approval"`
	MaxRetries        int  `gorm:"type:integer;default:3" json:"max_retries"`

	// Channel Settings
	GoogleOptimization    bool `gorm:"default:true" json:"google_optimization"`
	FacebookOptimization  bool `gorm:"default:true" json:"facebook_optimization"`
	InstagramOptimization bool `gorm:"default:true" json:"instagram_optimization"`

	// Language Settings
	DefaultLanguage  string `gorm:"type:varchar(10);default:'en'" json:"default_language"`
	FallbackLanguage string `gorm:"type:varchar(10);default:'en'" json:"fallback_language"`
	TranslateContent bool   `gorm:"default:false" json:"translate_content"`

	// Advanced Settings
	CustomPrompts      bool    `gorm:"default:false" json:"custom_prompts"`
	CustomInstructions *string `gorm:"type:text" json:"custom_instructions,omitempty"`
	DebugMode          bool    `gorm:"default:false" json:"debug_mode"`

	CreatedAt time.Time `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;default:now()" json:"updated_at"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID" json:"organization,omitempty"`
}

// TableName specifies the table name for AISettings
func (AISettings) TableName() string {
	return "ai_settings"
}

// BeforeCreate hook to generate UUID if not set
func (s *AISettings) BeforeCreate() error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// Validate validates the AI settings
func (s *AISettings) Validate() error {
	if s.MaxTokens < 1 || s.MaxTokens > 4000 {
		return errors.New("max_tokens must be between 1 and 4000")
	}
	if s.Temperature < 0 || s.Temperature > 2 {
		return errors.New("temperature must be between 0 and 2")
	}
	if s.TopP < 0 || s.TopP > 1 {
		return errors.New("top_p must be between 0 and 1")
	}
	if s.MinScoreThreshold < 0 || s.MinScoreThreshold > 100 {
		return errors.New("min_score_threshold must be between 0 and 100")
	}
	if s.MaxRetries < 0 || s.MaxRetries > 10 {
		return errors.New("max_retries must be between 0 and 10")
	}
	return nil
}

// AICredits manages AI credit allocation and usage tracking
type AICredits struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID `gorm:"type:uuid;unique;not null" json:"organization_id"`

	// Credit Management
	CreditsRemaining int `gorm:"type:integer;default:2500" json:"credits_remaining"`
	CreditsTotal     int `gorm:"type:integer;default:2500" json:"credits_total"`
	CreditsUsed      int `gorm:"type:integer;default:0" json:"credits_used"`

	// Reset Management
	ResetDate     time.Time  `gorm:"type:timestamp with time zone;not null" json:"reset_date"`
	LastResetDate *time.Time `gorm:"type:timestamp with time zone" json:"last_reset_date,omitempty"`

	// Cost Tracking
	TotalSpent   float64 `gorm:"type:decimal(10,4);default:0.0000" json:"total_spent"`
	MonthlySpent float64 `gorm:"type:decimal(10,4);default:0.0000" json:"monthly_spent"`

	// Usage Statistics
	TotalOptimizations      int `gorm:"type:integer;default:0" json:"total_optimizations"`
	SuccessfulOptimizations int `gorm:"type:integer;default:0" json:"successful_optimizations"`
	FailedOptimizations     int `gorm:"type:integer;default:0" json:"failed_optimizations"`

	CreatedAt time.Time `gorm:"type:timestamp with time zone;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;default:now()" json:"updated_at"`

	// Relations
	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID" json:"organization,omitempty"`
}

// TableName specifies the table name for AICredits
func (AICredits) TableName() string {
	return "ai_credits"
}

// BeforeCreate hook to generate UUID and set reset date if not set
func (c *AICredits) BeforeCreate() error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.ResetDate.IsZero() {
		c.ResetDate = time.Now().AddDate(0, 1, 0) // Reset in 1 month
	}
	return nil
}

// HasCredits checks if the organization has remaining credits
func (c *AICredits) HasCredits() bool {
	return c.CreditsRemaining > 0
}

// DeductCredits deducts credits from the remaining balance
func (c *AICredits) DeductCredits(amount int) error {
	if amount < 0 {
		return errors.New("amount must be positive")
	}
	if c.CreditsRemaining < amount {
		return errors.New("insufficient credits")
	}
	c.CreditsRemaining -= amount
	c.CreditsUsed += amount
	return nil
}

// AddCost adds to the cost tracking
func (c *AICredits) AddCost(cost float64) error {
	if cost < 0 {
		return errors.New("cost must be positive")
	}
	c.TotalSpent += cost
	c.MonthlySpent += cost
	return nil
}

// ShouldReset checks if credits should be reset based on reset date
func (c *AICredits) ShouldReset() bool {
	return time.Now().After(c.ResetDate)
}

// Reset resets the monthly credits
func (c *AICredits) Reset() {
	now := time.Now()
	c.CreditsRemaining = c.CreditsTotal
	c.CreditsUsed = 0
	c.MonthlySpent = 0
	c.LastResetDate = &now
	c.ResetDate = now.AddDate(0, 1, 0) // Reset in 1 month
}

// OptimizationAnalytics represents aggregated analytics data
type OptimizationAnalytics struct {
	OrganizationID       uuid.UUID `json:"organization_id"`
	TotalOptimizations   int       `json:"total_optimizations"`
	AppliedCount         int       `json:"applied_count"`
	PendingCount         int       `json:"pending_count"`
	FailedCount          int       `json:"failed_count"`
	AvgScore             *float64  `json:"avg_score,omitempty"`
	AvgImprovement       *float64  `json:"avg_improvement,omitempty"`
	TotalCost            float64   `json:"total_cost"`
	TotalTokens          int       `json:"total_tokens"`
	LastOptimizationDate time.Time `json:"last_optimization_date"`
}

// OptimizationByType represents optimization statistics by type
type OptimizationByType struct {
	OrganizationID   uuid.UUID        `json:"organization_id"`
	OptimizationType OptimizationType `json:"optimization_type"`
	Count            int              `json:"count"`
	AvgScore         *float64         `json:"avg_score,omitempty"`
	TotalCost        float64          `json:"total_cost"`
}

// OptimizationRequest represents a request to optimize a product
type OptimizationRequest struct {
	ProductID          string   `json:"product_id" binding:"required"`
	OptimizationType   string   `json:"optimization_type" binding:"required"`
	Strategy           string   `json:"strategy,omitempty"`
	Style              string   `json:"style,omitempty"`
	Length             string   `json:"length,omitempty"`
	TargetAudience     string   `json:"target_audience,omitempty"`
	Language           string   `json:"language,omitempty"`
	Keywords           string   `json:"keywords,omitempty"`
	MaxLength          int      `json:"max_length,omitempty"`
	CustomInstructions string   `json:"custom_instructions,omitempty"`
}

// BulkOptimizationRequest represents a bulk optimization request
type BulkOptimizationRequest struct {
	ProductIDs       []string          `json:"product_ids" binding:"required"`
	OptimizationType OptimizationType  `json:"optimization_type" binding:"required"`
	TargetAudience   string            `json:"target_audience,omitempty"`
	Language         string            `json:"language,omitempty"`
	Tone             string            `json:"tone,omitempty"`
	IncludeKeywords  bool              `json:"include_keywords"`
	AutoApply        bool              `json:"auto_apply"`
	Settings         map[string]interface{} `json:"settings,omitempty"`
}

// OptimizationResponse represents the response from an optimization
type OptimizationResponse struct {
	OptimizationID    string                 `json:"optimization_id"`
	ProductID         string                 `json:"product_id"`
	OptimizationType  string                 `json:"optimization_type"`
	OriginalValue     string                 `json:"original_value"`
	OptimizedValue    string                 `json:"optimized_value"`
	Score             int                    `json:"score"`
	Improvement       float64                `json:"improvement"`
	Suggestions       []string               `json:"suggestions,omitempty"`
	Reasoning         []string               `json:"reasoning,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Cost              float64                `json:"cost"`
	TokensUsed        int                    `json:"tokens_used"`
	AIModel           string                 `json:"ai_model"`
	Status            string                 `json:"status"`
	Message           string                 `json:"message"`
}

