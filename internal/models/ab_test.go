package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ABTest struct {
	ID          string       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string       `json:"name" gorm:"not null"`
	ProductID   string       `json:"product_id" gorm:"not null"`
	VariantAID  string       `json:"variant_a_id" gorm:"not null"`
	VariantBID  string       `json:"variant_b_id" gorm:"not null"`
	Status      ABTestStatus `json:"status" gorm:"default:ACTIVE"`
	Impressions int          `json:"impressions" gorm:"default:0"`
	Clicks      int          `json:"clicks" gorm:"default:0"`
	Conversions int          `json:"conversions" gorm:"default:0"`
	ROAS        *float64     `json:"roas" gorm:"type:decimal(10,4)"`
	Winner      *string      `json:"winner"`
	Confidence  *float64     `json:"confidence"`
	StartedAt   time.Time    `json:"started_at"`
	EndedAt     *time.Time   `json:"ended_at"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`

	// Relations
	Product  Product     `json:"product" gorm:"foreignKey:ProductID"`
	VariantA FeedVariant `json:"variant_a" gorm:"foreignKey:VariantAID"`
	VariantB FeedVariant `json:"variant_b" gorm:"foreignKey:VariantBID"`
}

type ABTestStatus string

const (
	ABTestStatusActive    ABTestStatus = "ACTIVE"
	ABTestStatusPaused    ABTestStatus = "PAUSED"
	ABTestStatusCompleted ABTestStatus = "COMPLETED"
	ABTestStatusCancelled ABTestStatus = "CANCELLED"
)

func (ab *ABTest) BeforeCreate(tx *gorm.DB) error {
	if ab.ID == "" {
		ab.ID = uuid.New().String()
	}
	return nil
}
