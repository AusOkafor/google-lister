package main

import (
	"log"

	"lister/internal/api"
	"lister/internal/config"
	"lister/internal/database"
	"lister/internal/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel)

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database:", err)
	}

	// Initialize API server
	server := api.New(cfg, logger, db)

	// Start server
	logger.Info("Starting API server on port " + cfg.APIPort)
	if err := server.Start(); err != nil {
		logger.Fatal("Failed to start server:", err)
	}
}
