-- ============================================================================
-- Feed Automation Tables for Supabase
-- Adds auto-regeneration scheduling and webhook support
-- Run this in Supabase SQL Editor
-- ============================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- Table: feed_schedules
-- Purpose: Track auto-regeneration schedules for feeds
-- ============================================================================
CREATE TABLE IF NOT EXISTS feed_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL,
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    
    -- Schedule Configuration
    enabled BOOLEAN DEFAULT TRUE,
    interval_hours INTEGER NOT NULL CHECK (interval_hours > 0),
    next_run_at TIMESTAMP WITH TIME ZONE,
    last_run_at TIMESTAMP WITH TIME ZONE,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'failed')),
    consecutive_failures INTEGER DEFAULT 0,
    last_error TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_schedule_feed FOREIGN KEY (feed_id)
        REFERENCES product_feeds(id) ON DELETE CASCADE,
    
    UNIQUE(feed_id)
);

-- Index for efficient scheduling queries
CREATE INDEX IF NOT EXISTS idx_feed_schedules_next_run 
    ON feed_schedules(next_run_at) 
    WHERE enabled = TRUE AND status = 'active';

-- ============================================================================
-- Table: feed_webhooks
-- Purpose: Store webhook configurations and delivery tracking
-- ============================================================================
CREATE TABLE IF NOT EXISTS feed_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL,
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    
    -- Webhook Configuration
    url TEXT NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    events TEXT[] DEFAULT ARRAY['feed.generated', 'feed.failed'],
    secret VARCHAR(255), -- For webhook signature verification
    
    -- Delivery Settings
    retry_count INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 30,
    
    -- Status
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    total_deliveries INTEGER DEFAULT 0,
    successful_deliveries INTEGER DEFAULT 0,
    failed_deliveries INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_webhook_feed FOREIGN KEY (feed_id)
        REFERENCES product_feeds(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_feed_webhooks_feed_id ON feed_webhooks(feed_id);
CREATE INDEX IF NOT EXISTS idx_feed_webhooks_enabled ON feed_webhooks(enabled) WHERE enabled = TRUE;

-- ============================================================================
-- Table: webhook_deliveries
-- Purpose: Log webhook delivery attempts for debugging and monitoring
-- ============================================================================
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL,
    feed_id UUID NOT NULL,
    
    -- Delivery Details
    event VARCHAR(50) NOT NULL,
    payload JSONB,
    
    -- Response
    status_code INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    
    -- Result
    success BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    retry_attempt INTEGER DEFAULT 0,
    
    -- Timestamp
    delivered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_delivery_webhook FOREIGN KEY (webhook_id)
        REFERENCES feed_webhooks(id) ON DELETE CASCADE,
    CONSTRAINT fk_delivery_feed FOREIGN KEY (feed_id)
        REFERENCES product_feeds(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_feed_id ON webhook_deliveries(feed_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_delivered_at ON webhook_deliveries(delivered_at DESC);

-- ============================================================================
-- Table: platform_credentials
-- Purpose: Store encrypted API credentials for platform submissions
-- ============================================================================
CREATE TABLE IF NOT EXISTS platform_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL,
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    
    -- Platform
    platform VARCHAR(50) NOT NULL, -- 'google_shopping', 'facebook', 'instagram'
    
    -- Credentials (should be encrypted in production)
    api_key TEXT,
    merchant_id VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Settings
    auto_submit BOOLEAN DEFAULT FALSE,
    submit_on_regenerate BOOLEAN DEFAULT FALSE,
    
    -- Status
    last_submission_at TIMESTAMP WITH TIME ZONE,
    last_submission_status VARCHAR(20),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT fk_credentials_feed FOREIGN KEY (feed_id)
        REFERENCES product_feeds(id) ON DELETE CASCADE,
    
    UNIQUE(feed_id, platform)
);

CREATE INDEX IF NOT EXISTS idx_platform_credentials_feed_id ON platform_credentials(feed_id);
CREATE INDEX IF NOT EXISTS idx_platform_credentials_platform ON platform_credentials(platform);

-- ============================================================================
-- Function: Update next_run_at when interval changes
-- ============================================================================
CREATE OR REPLACE FUNCTION update_next_run_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.enabled = TRUE AND NEW.status = 'active' THEN
        NEW.next_run_at = NOW() + (NEW.interval_hours || ' hours')::INTERVAL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_next_run_at
    BEFORE INSERT OR UPDATE ON feed_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_next_run_at();

-- Migration complete
SELECT 'Feed automation tables created successfully! ✅' as status;

