package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"lister/internal/api/handlers"
	"lister/internal/api/middleware"
	"lister/internal/config"
	"lister/internal/database"
	"lister/internal/logger"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config *config.Config
	logger *logger.Logger
	db     *database.Database
	router *gin.Engine
	server *http.Server
}

func New(cfg *config.Config, logger *logger.Logger, db *database.Database) *Server {
	// Set Gin mode
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())

	// Initialize handlers
	productHandler := handlers.NewProductHandler(db.DB, logger)
	connectorHandler := handlers.NewConnectorHandler(db.DB, logger)
	channelHandler := handlers.NewChannelHandler(db.DB, logger)
	issueHandler := handlers.NewIssueHandler(db.DB, logger)
	shopifyHandler := handlers.NewShopifyHandler(db.DB, logger, cfg)

	// Routes
	v1 := router.Group("/api/v1")
	{
		// Products
		products := v1.Group("/products")
		{
			products.GET("", productHandler.List)
			products.GET("/:id", productHandler.Get)
			products.POST("", productHandler.Create)
			products.PUT("/:id", productHandler.Update)
			products.DELETE("/:id", productHandler.Delete)
		}

		// Connectors
		connectors := v1.Group("/connectors")
		{
			connectors.GET("", connectorHandler.List)
			connectors.GET("/:id", connectorHandler.Get)
			connectors.POST("", connectorHandler.Create)
			connectors.PUT("/:id", connectorHandler.Update)
			connectors.DELETE("/:id", connectorHandler.Delete)
			connectors.POST("/:id/sync", connectorHandler.Sync)
		}

		// Channels
		channels := v1.Group("/channels")
		{
			channels.GET("", channelHandler.List)
			channels.GET("/:id", channelHandler.Get)
			channels.POST("", channelHandler.Create)
			channels.PUT("/:id", channelHandler.Update)
			channels.DELETE("/:id", channelHandler.Delete)
			channels.POST("/:id/sync", channelHandler.Sync)
		}

		// Issues
		issues := v1.Group("/issues")
		{
			issues.GET("", issueHandler.List)
			issues.GET("/:id", issueHandler.Get)
			issues.POST("/:id/resolve", issueHandler.Resolve)
		}

		// Shopify Integration
		shopify := v1.Group("/shopify")
		{
			shopify.POST("/install", shopifyHandler.Install)
			shopify.GET("/callback", shopifyHandler.Callback)
			shopify.POST("/:id/sync", shopifyHandler.SyncProducts)
			shopify.POST("/webhook", shopifyHandler.Webhook)
		}
	}

	return &Server{
		config: cfg,
		logger: logger,
		db:     db,
		router: router,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%s", s.config.APIHost, s.config.APIPort)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting server on " + addr)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down server...")
	return s.server.Shutdown(ctx)
}

// GetRouter returns the Gin router for Vercel
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
