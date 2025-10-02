package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"lister/internal/config"
	"lister/internal/logger"
	"lister/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel)

	// Initialize worker
	w := worker.New(cfg, logger)

	// Start worker
	logger.Info("Starting worker...")
	go w.Start()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")
	w.Stop()
}
