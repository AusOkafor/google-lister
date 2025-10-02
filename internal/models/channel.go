package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Channel struct {
	ID          string                 `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string                 `json:"name" gorm:"not null"`
	Type        ChannelType            `json:"type" gorm:"not null"`
	Status      ChannelStatus          `json:"status" gorm:"default:INACTIVE"`
	Config      string `json:"config" gorm:"type:text"`
	Credentials string `json:"credentials" gorm:"type:text"`
	LastSync    *time.Time             `json:"last_sync"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ChannelType string

const (
	ChannelTypeGoogleMerchantCenter ChannelType = "GOOGLE_MERCHANT_CENTER"
	ChannelTypeBingShopping         ChannelType = "BING_SHOPPING"
	ChannelTypeMetaCatalog          ChannelType = "META_CATALOG"
	ChannelTypePinterestCatalog     ChannelType = "PINTEREST_CATALOG"
	ChannelTypeTikTokShopping       ChannelType = "TIKTOK_SHOPPING"
)

type ChannelStatus string

const (
	ChannelStatusActive   ChannelStatus = "ACTIVE"
	ChannelStatusInactive ChannelStatus = "INACTIVE"
	ChannelStatusError    ChannelStatus = "ERROR"
	ChannelStatusSyncing  ChannelStatus = "SYNCING"
)

func (c *Channel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
