package shopify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"lister/internal/config"
	"lister/internal/logger"
)

type ShopifyConnector struct {
	config *config.Config
	logger *logger.Logger
	client *http.Client
}

func New(cfg *config.Config, logger *logger.Logger) *ShopifyConnector {
	return &ShopifyConnector{
		config: cfg,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (sc *ShopifyConnector) SyncProducts(shopDomain, accessToken string) error {
	// TODO: Implement Shopify product sync
	// This would:
	// - Fetch products from Shopify API
	// - Transform to canonical format
	// - Publish events to Kafka
	// - Handle pagination and rate limiting

	sc.logger.Info("Syncing products from Shopify store: %s", shopDomain)

	// For now, just log the sync request
	sc.logger.Debug("Shopify sync completed")

	return nil
}

func (sc *ShopifyConnector) HandleWebhook(payload []byte) error {
	// TODO: Implement Shopify webhook handling
	// This would:
	// - Parse webhook payload
	// - Determine event type (product created/updated/deleted)
	// - Transform to canonical format
	// - Publish event to Kafka

	var webhook WebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	sc.logger.Debug("Received Shopify webhook: %s", webhook.Topic)

	return nil
}

type WebhookPayload struct {
	Topic   string                 `json:"topic"`
	Data    map[string]interface{} `json:"data"`
	Created time.Time              `json:"created_at"`
}
