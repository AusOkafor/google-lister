package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID           string              `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ExternalID   string              `json:"external_id" gorm:"not null"`
	SKU          string              `json:"sku" gorm:"unique;not null"`
	Title        string              `json:"title" gorm:"not null"`
	Description  *string             `json:"description"`
	Brand        *string             `json:"brand"`
	GTIN         *string             `json:"gtin"`
	MPN          *string             `json:"mpn"`
	Category     *string             `json:"category"`
	Price        float64             `json:"price" gorm:"type:decimal(10,2)"`
	CompareAtPrice *float64          `json:"compare_at_price" gorm:"type:decimal(10,2)"`
	Currency     string              `json:"currency" gorm:"default:USD"`
	Availability string              `json:"availability" gorm:"default:IN_STOCK"`
	Images       []string            `json:"images" gorm:"type:jsonb"`
	Variants     []ProductVariant    `json:"variants" gorm:"type:jsonb"`
	Shipping     *ShippingInfo       `json:"shipping" gorm:"type:jsonb"`
	TaxClass     *string             `json:"tax_class"`
	CustomLabels []string            `json:"custom_labels" gorm:"type:jsonb"`
	Metadata     map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`

	// Relations (temporarily disabled for migration)
	// Issues       []Issue       `json:"issues" gorm:"foreignKey:ProductID"`
	// FeedVariants []FeedVariant `json:"feed_variants" gorm:"foreignKey:ProductID"`
}

type ProductVariant struct {
	ID         string                 `json:"id"`
	SKU        string                 `json:"sku"`
	Price      float64                `json:"price"`
	Attributes map[string]interface{} `json:"attributes"`
}

type ShippingInfo struct {
	Weight        *float64    `json:"weight"`
	Dimensions    *Dimensions `json:"dimensions"`
	ShippingLabel *string     `json:"shipping_label"`
}

type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

type ProductAvailability string

const (
	AvailabilityInStock    ProductAvailability = "IN_STOCK"
	AvailabilityOutOfStock ProductAvailability = "OUT_OF_STOCK"
	AvailabilityPreorder   ProductAvailability = "PREORDER"
	AvailabilityBackorder  ProductAvailability = "BACKORDER"
)

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}
