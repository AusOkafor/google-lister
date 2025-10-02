package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Issue struct {
	ID           string        `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ProductID    string        `json:"product_id" gorm:"not null"`
	Channel      string        `json:"channel" gorm:"not null"`
	Code         string        `json:"code" gorm:"not null"`
	Severity     IssueSeverity `json:"severity" gorm:"not null"`
	Explanation  string        `json:"explanation" gorm:"not null"`
	SuggestedFix *string       `json:"suggested_fix"`
	Confidence   *float64      `json:"confidence"`
	IsResolved   bool          `json:"is_resolved" gorm:"default:false"`
	ResolvedAt   *time.Time    `json:"resolved_at"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`

	// Relations (temporarily disabled for migration)
	// Product Product `json:"product" gorm:"foreignKey:ProductID"`
}

type IssueSeverity string

const (
	IssueSeverityLow      IssueSeverity = "LOW"
	IssueSeverityMedium   IssueSeverity = "MEDIUM"
	IssueSeverityHigh     IssueSeverity = "HIGH"
	IssueSeverityCritical IssueSeverity = "CRITICAL"
)

func (i *Issue) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}
