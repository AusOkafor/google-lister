package worker

import (
	"context"
	"encoding/json"
	"time"

	"lister/internal/config"
	"lister/internal/logger"
	"lister/internal/worker/processors"

	"github.com/segmentio/kafka-go"
)

type Worker struct {
	config    *config.Config
	logger    *logger.Logger
	reader    *kafka.Reader
	processor *processors.EventProcessor
}

func New(cfg *config.Config, logger *logger.Logger) *Worker {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{cfg.KafkaBrokers},
		GroupID:        "lister-worker",
		Topic:          "product-events",
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
	})

	processor := processors.NewEventProcessor(cfg, logger)

	return &Worker{
		config:    cfg,
		logger:    logger,
		reader:    reader,
		processor: processor,
	}
}

func (w *Worker) Start() {
	w.logger.Info("Worker started, listening for events...")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		message, err := w.reader.ReadMessage(ctx)
		cancel()

		if err != nil {
			w.logger.Error("Failed to read message: %v", err)
			continue
		}

		w.logger.Debug("Received message: %s", string(message.Value))

		// Parse event
		var event Event
		if err := json.Unmarshal(message.Value, &event); err != nil {
			w.logger.Error("Failed to parse event: %v", err)
			continue
		}

		// Process event
		if err := w.processor.Process(event); err != nil {
			w.logger.Error("Failed to process event: %v", err)
			continue
		}

		w.logger.Debug("Event processed successfully")
	}
}

func (w *Worker) Stop() {
	w.logger.Info("Stopping worker...")
	w.reader.Close()
}

type Event struct {
	Type      string                 `json:"type"`
	ProductID string                 `json:"product_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}
