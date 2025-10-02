package main

import (
	"context"
	"log"
	"net/http"

	"lister/internal/api"
	"lister/internal/config"
	"lister/internal/database"
	"lister/internal/logger"

	"github.com/gin-gonic/gin"
)

// Handler is the main entry point for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
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

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Serve the request
	server.Router.ServeHTTP(w, r)
}

// This function is required by Vercel
func main() {
	// This won't be called in Vercel, but required for Go compilation
}
