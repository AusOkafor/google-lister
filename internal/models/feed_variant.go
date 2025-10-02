package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FeedVariant struct {
	ID             string                 `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name           string                 `json:"name" gorm:"not null"`
	ProductID      string                 `json:"product_id" gorm:"not null"`
	Transformation string `json:"transformation" gorm:"type:text"`
	Status         VariantStatus          `json:"status" gorm:"default:DRAFT"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`

	// Relations (temporarily disabled for migration)
	// Product Product `json:"product" gorm:"foreignKey:ProductID"`
}

type VariantStatus string

const (
	VariantStatusDraft    VariantStatus = "DRAFT"
	VariantStatusActive   VariantStatus = "ACTIVE"
	VariantStatusArchived VariantStatus = "ARCHIVED"
)

func (fv *FeedVariant) BeforeCreate(tx *gorm.DB) error {
	if fv.ID == "" {
		fv.ID = uuid.New().String()
	}
	return nil
}
