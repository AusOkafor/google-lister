-- ============================================================================
-- Product Feeds Tables for Supabase
-- Run this in Supabase SQL Editor
-- ============================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- Table: product_feeds
-- Purpose: Store feed configurations and metadata
-- ============================================================================
CREATE TABLE IF NOT EXISTS product_feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    
    -- Feed Configuration
    name VARCHAR(255) NOT NULL,
    channel VARCHAR(100) NOT NULL CHECK (channel IN ('Google Shopping', 'Facebook', 'Instagram', 'Amazon', 'eBay', 'Pinterest', 'TikTok Shop', 'Snapchat')),
    format VARCHAR(20) NOT NULL CHECK (format IN ('xml', 'csv', 'json', 'txt')),
    
    -- Feed Status
    status VARCHAR(20) NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive', 'generating', 'error', 'paused')),
    
    -- Feed Statistics
    products_count INTEGER DEFAULT 0 CHECK (products_count >= 0),
    last_generated TIMESTAMP WITH TIME ZONE,
    
    -- Configuration
    settings JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(organization_id, name)
);

-- ============================================================================
-- Table: feed_generation_history
-- Purpose: Track feed generation history and performance
-- ============================================================================
CREATE TABLE IF NOT EXISTS feed_generation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL,
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    
    -- Generation Details
    status VARCHAR(20) NOT NULL CHECK (status IN ('started', 'completed', 'failed', 'cancelled')),
    products_processed INTEGER DEFAULT 0,
    products_included INTEGER DEFAULT 0,
    products_excluded INTEGER DEFAULT 0,
    
    -- Performance Metrics
    generation_time_ms INTEGER,
    file_size_bytes BIGINT,
    error_message TEXT,
    
    -- File Information
    file_url TEXT,
    file_format VARCHAR(20),
    
    -- Timestamps
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Foreign key
    CONSTRAINT fk_feed_generation_history_feed FOREIGN KEY (feed_id) 
        REFERENCES product_feeds(id) ON DELETE CASCADE
);

-- ============================================================================
-- Table: feed_templates
-- Purpose: Store pre-configured feed templates for different channels
-- ============================================================================
CREATE TABLE IF NOT EXISTS feed_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    
    -- Template Configuration
    name VARCHAR(255) NOT NULL,
    description TEXT,
    channel VARCHAR(100) NOT NULL,
    format VARCHAR(20) NOT NULL,
    
    -- Template Settings
    field_mapping JSONB DEFAULT '{}',
    filters JSONB DEFAULT '{}',
    transformations JSONB DEFAULT '{}',
    
    -- Template Metadata
    is_system_template BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(organization_id, name)
);

-- ============================================================================
-- Indexes for performance
-- ============================================================================

-- Product feeds indexes
CREATE INDEX IF NOT EXISTS idx_product_feeds_organization_id ON product_feeds(organization_id);
CREATE INDEX IF NOT EXISTS idx_product_feeds_channel ON product_feeds(channel);
CREATE INDEX IF NOT EXISTS idx_product_feeds_status ON product_feeds(status);
CREATE INDEX IF NOT EXISTS idx_product_feeds_created_at ON product_feeds(created_at DESC);

-- Feed generation history indexes
CREATE INDEX IF NOT EXISTS idx_feed_generation_history_feed_id ON feed_generation_history(feed_id);
CREATE INDEX IF NOT EXISTS idx_feed_generation_history_organization_id ON feed_generation_history(organization_id);
CREATE INDEX IF NOT EXISTS idx_feed_generation_history_status ON feed_generation_history(status);
CREATE INDEX IF NOT EXISTS idx_feed_generation_history_started_at ON feed_generation_history(started_at DESC);

-- Feed templates indexes
CREATE INDEX IF NOT EXISTS idx_feed_templates_organization_id ON feed_templates(organization_id);
CREATE INDEX IF NOT EXISTS idx_feed_templates_channel ON feed_templates(channel);
CREATE INDEX IF NOT EXISTS idx_feed_templates_is_system_template ON feed_templates(is_system_template);

-- ============================================================================
-- Insert default system templates
-- ============================================================================

-- Google Shopping Template
INSERT INTO feed_templates (
    id,
    name,
    description,
    channel,
    format,
    field_mapping,
    filters,
    transformations,
    is_system_template
) VALUES (
    gen_random_uuid(),
    'Google Shopping Standard',
    'Standard Google Shopping feed template with required fields',
    'Google Shopping',
    'xml',
    '{"id": "external_id", "title": "title", "description": "description", "link": "product_url", "image_link": "main_image", "price": "price", "availability": "availability", "brand": "brand", "condition": "condition", "google_product_category": "category", "mpn": "sku"}',
    '{"status": "ACTIVE", "price_min": 0.01}',
    '{"price_format": "decimal", "description_max_length": 5000}',
    TRUE
) ON CONFLICT DO NOTHING;

-- Facebook Catalog Template
INSERT INTO feed_templates (
    id,
    name,
    description,
    channel,
    format,
    field_mapping,
    filters,
    transformations,
    is_system_template
) VALUES (
    gen_random_uuid(),
    'Facebook Catalog Standard',
    'Standard Facebook Catalog feed template',
    'Facebook',
    'csv',
    '{"id": "external_id", "title": "title", "description": "description", "link": "product_url", "image_link": "main_image", "price": "price", "availability": "availability", "brand": "brand", "condition": "condition", "product_type": "category", "sale_price": "compare_at_price"}',
    '{"status": "ACTIVE", "price_min": 0.01}',
    '{"price_format": "decimal", "description_max_length": 5000}',
    TRUE
) ON CONFLICT DO NOTHING;

-- Instagram Shopping Template
INSERT INTO feed_templates (
    id,
    name,
    description,
    channel,
    format,
    field_mapping,
    filters,
    transformations,
    is_system_template
) VALUES (
    gen_random_uuid(),
    'Instagram Shopping Standard',
    'Standard Instagram Shopping feed template',
    'Instagram',
    'json',
    '{"id": "external_id", "title": "title", "description": "description", "link": "product_url", "image_url": "main_image", "price": "price", "availability": "availability", "brand": "brand", "condition": "condition", "category": "category"}',
    '{"status": "ACTIVE", "price_min": 0.01}',
    '{"price_format": "decimal", "description_max_length": 5000}',
    TRUE
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE product_feeds IS 'Stores feed configurations and metadata for different sales channels';
COMMENT ON TABLE feed_generation_history IS 'Tracks feed generation history and performance metrics';
COMMENT ON TABLE feed_templates IS 'Pre-configured feed templates for different channels and formats';

-- Migration complete
SELECT 'Product Feeds tables created successfully! ✅' as status;
