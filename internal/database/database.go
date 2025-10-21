package database

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB *gorm.DB
}

func New(databaseURL string) (*Database, error) {
	var db *gorm.DB
	var err error

	if strings.HasPrefix(databaseURL, "sqlite://") {
		// SQLite for development
		dbPath := strings.TrimPrefix(databaseURL, "sqlite://")
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	} else {
		// PostgreSQL for production
		db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create tables manually with raw SQL
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS products (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		external_id TEXT NOT NULL,
		sku TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		description TEXT,
		brand TEXT,
		gtin TEXT,
		mpn TEXT,
		category TEXT,
		price DECIMAL(10,2),
		currency TEXT DEFAULT 'USD',
		availability TEXT DEFAULT 'IN_STOCK',
		images TEXT,
		variants TEXT,
		shipping TEXT,
		tax_class TEXT,
		custom_labels TEXT,
		metadata TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS connectors (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT DEFAULT 'INACTIVE',
		config TEXT,
		credentials TEXT,
		last_sync TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS channels (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		status TEXT DEFAULT 'INACTIVE',
		config TEXT,
		credentials TEXT,
		last_sync TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS organizations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		plan TEXT DEFAULT 'free',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		organization_id UUID NOT NULL,
		role TEXT DEFAULT 'MEMBER',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS feed_variants (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		product_id UUID NOT NULL,
		transformation TEXT,
		status TEXT DEFAULT 'DRAFT',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS issues (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_id UUID NOT NULL,
		channel TEXT NOT NULL,
		code TEXT NOT NULL,
		severity TEXT NOT NULL,
		explanation TEXT NOT NULL,
		suggested_fix TEXT,
		confidence DECIMAL,
		is_resolved BOOLEAN DEFAULT false,
		resolved_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- AI Optimizer tables
	CREATE TABLE IF NOT EXISTS optimization_history (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		product_id UUID NOT NULL,
		organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
		optimization_type VARCHAR(50) NOT NULL CHECK (optimization_type IN ('title', 'description', 'category', 'image', 'bulk')),
		original_value TEXT,
		optimized_value TEXT,
		status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'applied', 'rejected', 'failed')),
		score INTEGER CHECK (score >= 0 AND score <= 100),
		improvement_percentage DECIMAL(5,2),
		ai_model VARCHAR(50) NOT NULL,
		cost DECIMAL(10,4) DEFAULT 0.0000,
		tokens_used INTEGER DEFAULT 0,
		metadata JSONB DEFAULT '{}',
		error_message TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		applied_at TIMESTAMPTZ
	);

	CREATE TABLE IF NOT EXISTS ai_credits (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid UNIQUE,
		credits_remaining INTEGER DEFAULT 2500 CHECK (credits_remaining >= 0),
		credits_total INTEGER DEFAULT 2500 CHECK (credits_total >= 0),
		credits_used INTEGER DEFAULT 0 CHECK (credits_used >= 0),
		reset_date TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 month',
		last_reset_date TIMESTAMPTZ,
		total_cost DECIMAL(10,4) DEFAULT 0.0000,
		total_optimizations INTEGER DEFAULT 0,
		successful_optimizations INTEGER DEFAULT 0,
		failed_optimizations INTEGER DEFAULT 0,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS ai_settings (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid UNIQUE,
		default_model VARCHAR(50) DEFAULT 'gpt-3.5-turbo',
		max_tokens INTEGER DEFAULT 500 CHECK (max_tokens > 0 AND max_tokens <= 4000),
		temperature DECIMAL(3,2) DEFAULT 0.70 CHECK (temperature >= 0 AND temperature <= 2),
		top_p DECIMAL(3,2) DEFAULT 0.90 CHECK (top_p >= 0 AND top_p <= 1),
		title_optimization BOOLEAN DEFAULT TRUE,
		description_optimization BOOLEAN DEFAULT TRUE,
		category_optimization BOOLEAN DEFAULT TRUE,
		image_optimization BOOLEAN DEFAULT FALSE,
		min_score_threshold INTEGER DEFAULT 80 CHECK (min_score_threshold >= 0 AND min_score_threshold <= 100),
		require_approval BOOLEAN DEFAULT TRUE,
		max_retries INTEGER DEFAULT 3 CHECK (max_retries >= 0 AND max_retries <= 10),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	`

	err = db.Exec(createTablesSQL).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
