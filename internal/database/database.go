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
