package validation

import (
	"lister/internal/config"
	"lister/internal/logger"
)

type Validator struct {
	config *config.Config
	logger *logger.Logger
}

func New(cfg *config.Config, logger *logger.Logger) *Validator {
	return &Validator{
		config: cfg,
		logger: logger,
	}
}

func (v *Validator) ValidateProduct(product interface{}) error {
	// TODO: Implement product validation logic
	// This would check:
	// - Required fields (title, price, etc.)
	// - Channel-specific requirements (Google, Bing, etc.)
	// - Data quality (image URLs, GTIN format, etc.)
	// - Policy compliance (title length, description content, etc.)

	v.logger.Debug("Validating product: %+v", product)

	return nil
}

func (v *Validator) ValidateChannel(channel string, product interface{}) error {
	// TODO: Implement channel-specific validation
	// Each channel has different requirements:
	// - Google: GTIN, MPN, category, etc.
	// - Bing: Similar to Google but with some differences
	// - Meta: Different image requirements, etc.

	v.logger.Debug("Validating product for channel %s: %+v", channel, product)

	return nil
}
