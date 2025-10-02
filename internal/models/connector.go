package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Connector struct {
	ID          string                 `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string                 `json:"name" gorm:"not null"`
	Type        ConnectorType          `json:"type" gorm:"not null"`
	Status      ConnectorStatus        `json:"status" gorm:"default:INACTIVE"`
	Config      map[string]interface{} `json:"config" gorm:"type:jsonb"`
	Credentials map[string]interface{} `json:"credentials" gorm:"type:jsonb"`
	LastSync    *time.Time             `json:"last_sync"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ConnectorType string

const (
	ConnectorTypeShopify     ConnectorType = "SHOPIFY"
	ConnectorTypeWooCommerce ConnectorType = "WOOCOMMERCE"
	ConnectorTypeMagento     ConnectorType = "MAGENTO"
	ConnectorTypeBigCommerce ConnectorType = "BIGCOMMERCE"
	ConnectorTypeCSV         ConnectorType = "CSV"
	ConnectorTypeAPI         ConnectorType = "API"
)

type ConnectorStatus string

const (
	ConnectorStatusActive   ConnectorStatus = "ACTIVE"
	ConnectorStatusInactive ConnectorStatus = "INACTIVE"
	ConnectorStatusError    ConnectorStatus = "ERROR"
	ConnectorStatusSyncing  ConnectorStatus = "SYNCING"
)

func (c *Connector) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
