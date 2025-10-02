package processors

import (
	"lister/internal/config"
	"lister/internal/logger"
	"lister/internal/worker/processors/ai"
	"lister/internal/worker/processors/export"
	"lister/internal/worker/processors/validation"
)

type EventProcessor struct {
	config      *config.Config
	logger      *logger.Logger
	validator   *validation.Validator
	aiOptimizer *ai.Optimizer
	exporter    *export.Exporter
}

func NewEventProcessor(cfg *config.Config, logger *logger.Logger) *EventProcessor {
	return &EventProcessor{
		config:      cfg,
		logger:      logger,
		validator:   validation.New(cfg, logger),
		aiOptimizer: ai.New(cfg, logger),
		exporter:    export.New(cfg, logger),
	}
}

func (ep *EventProcessor) Process(event interface{}) error {
	// TODO: Implement event processing logic
	// This would handle different event types:
	// - product.created
	// - product.updated
	// - product.deleted
	// - sync.requested
	// - validation.required
	// - export.required

	ep.logger.Debug("Processing event: %+v", event)

	// For now, just log the event
	ep.logger.Info("Event processed successfully")

	return nil
}
