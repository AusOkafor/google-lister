-- ============================================================================
-- Notifications System for Product Lister
-- Store webhook events and system notifications
-- Run this in Supabase SQL Editor
-- ============================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- Table: notifications
-- Purpose: Store system notifications from webhooks and other events
-- ============================================================================
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid,
    
    -- Notification Details
    type VARCHAR(50) NOT NULL CHECK (type IN ('feed_generated', 'feed_failed', 'feed_scheduled', 'system_alert', 'info')),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    
    -- Status
    is_read BOOLEAN DEFAULT FALSE,
    priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    
    -- Related Entity (optional)
    entity_type VARCHAR(50), -- 'feed', 'product', 'optimization', etc.
    entity_id UUID,
    entity_name VARCHAR(255),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    read_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Index for efficient queries
CREATE INDEX IF NOT EXISTS idx_notifications_org_created 
    ON notifications(organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread 
    ON notifications(organization_id, is_read, created_at DESC) 
    WHERE is_read = FALSE;

CREATE INDEX IF NOT EXISTS idx_notifications_entity 
    ON notifications(entity_type, entity_id);

-- ============================================================================
-- Function: Auto-expire old notifications
-- ============================================================================
CREATE OR REPLACE FUNCTION auto_expire_notifications()
RETURNS TRIGGER AS $$
BEGIN
    -- Auto-expire read notifications after 30 days
    IF NEW.is_read = TRUE AND NEW.expires_at IS NULL THEN
        NEW.expires_at = NOW() + INTERVAL '30 days';
    END IF;
    
    -- Auto-expire feed_generated notifications after 7 days
    IF NEW.type = 'feed_generated' AND NEW.expires_at IS NULL THEN
        NEW.expires_at = NOW() + INTERVAL '7 days';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_auto_expire_notifications
    BEFORE INSERT OR UPDATE ON notifications
    FOR EACH ROW
    EXECUTE FUNCTION auto_expire_notifications();

-- Migration complete
SELECT 'Notifications system created successfully! ✅' as status;

