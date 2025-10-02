package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// Kafka
	KafkaBrokers string

	// API Configuration
	APIPort string
	APIHost string

	// JWT
	JWTSecret string

	// Encryption
	EncryptionKey string

	// External APIs
	OpenAIAPIKey    string
	AnthropicAPIKey string

	// Google Merchant Center
	GoogleClientID     string
	GoogleClientSecret string

	// Shopify
	ShopifyClientID     string
	ShopifyClientSecret string

	// Environment
	Env      string
	LogLevel string
}

func Load() (*Config, error) {
	// Load .env file
	godotenv.Load()

	return &Config{
		DatabaseURL:         getEnv("DATABASE_URL", "postgresql://lister:lister@localhost:5432/lister?schema=public"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
		KafkaBrokers:        getEnv("KAFKA_BROKERS", "localhost:9092"),
		APIPort:             getEnv("API_PORT", "8080"),
		APIHost:             getEnv("API_HOST", "0.0.0.0"),
		JWTSecret:           getEnv("JWT_SECRET", "your-jwt-secret-key-here"),
		EncryptionKey:       getEnv("ENCRYPTION_KEY", "your-32-byte-encryption-key-here"),
		OpenAIAPIKey:        getEnv("OPENAI_API_KEY", ""),
		AnthropicAPIKey:     getEnv("ANTHROPIC_API_KEY", ""),
		GoogleClientID:      getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:  getEnv("GOOGLE_CLIENT_SECRET", ""),
		ShopifyClientID:     getEnv("SHOPIFY_CLIENT_ID", ""),
		ShopifyClientSecret: getEnv("SHOPIFY_CLIENT_SECRET", ""),
		Env:                 getEnv("ENV", "development"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
