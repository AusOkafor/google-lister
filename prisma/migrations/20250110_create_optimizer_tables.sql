-- Migration: Create AI Optimizer Tables
-- Created: 2025-01-10
-- Description: Creates tables for optimization history, AI settings, and AI credits

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- Table: optimization_history
-- Purpose: Track all AI optimization attempts and results
-- ============================================================================
CREATE TABLE IF NOT EXISTS optimization_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    organization_id UUID NOT NULL,
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
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    applied_at TIMESTAMP WITH TIME ZONE,
    
    -- Foreign key constraints
    CONSTRAINT fk_optimization_product FOREIGN KEY (product_id) 
        REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_optimization_organization FOREIGN KEY (organization_id) 
        REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for optimization_history
CREATE INDEX idx_optimization_history_product_id ON optimization_history(product_id);
CREATE INDEX idx_optimization_history_organization_id ON optimization_history(organization_id);
CREATE INDEX idx_optimization_history_type ON optimization_history(optimization_type);
CREATE INDEX idx_optimization_history_status ON optimization_history(status);
CREATE INDEX idx_optimization_history_created_at ON optimization_history(created_at DESC);
CREATE INDEX idx_optimization_history_org_created ON optimization_history(organization_id, created_at DESC);

-- ============================================================================
-- Table: ai_settings
-- Purpose: Store AI optimization settings per organization
-- ============================================================================
CREATE TABLE IF NOT EXISTS ai_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL UNIQUE,
    
    -- General Settings
    default_model VARCHAR(50) DEFAULT 'gpt-3.5-turbo',
    auto_optimize BOOLEAN DEFAULT FALSE,
    auto_apply BOOLEAN DEFAULT FALSE,
    max_cost_per_month DECIMAL(10,2) DEFAULT 25.00,
    notifications BOOLEAN DEFAULT TRUE,
    
    -- Model Parameters
    max_tokens INTEGER DEFAULT 500 CHECK (max_tokens > 0 AND max_tokens <= 4000),
    temperature DECIMAL(3,2) DEFAULT 0.70 CHECK (temperature >= 0 AND temperature <= 2),
    top_p DECIMAL(3,2) DEFAULT 0.90 CHECK (top_p >= 0 AND top_p <= 1),
    
    -- Feature Toggles
    title_optimization BOOLEAN DEFAULT TRUE,
    description_optimization BOOLEAN DEFAULT TRUE,
    category_optimization BOOLEAN DEFAULT TRUE,
    image_optimization BOOLEAN DEFAULT TRUE,
    
    -- Quality Settings
    min_score_threshold INTEGER DEFAULT 80 CHECK (min_score_threshold >= 0 AND min_score_threshold <= 100),
    require_approval BOOLEAN DEFAULT TRUE,
    max_retries INTEGER DEFAULT 3 CHECK (max_retries >= 0 AND max_retries <= 10),
    
    -- Channel Settings
    google_optimization BOOLEAN DEFAULT TRUE,
    facebook_optimization BOOLEAN DEFAULT TRUE,
    instagram_optimization BOOLEAN DEFAULT TRUE,
    
    -- Language Settings
    default_language VARCHAR(10) DEFAULT 'en',
    fallback_language VARCHAR(10) DEFAULT 'en',
    translate_content BOOLEAN DEFAULT FALSE,
    
    -- Advanced Settings
    custom_prompts BOOLEAN DEFAULT FALSE,
    custom_instructions TEXT,
    debug_mode BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_ai_settings_organization FOREIGN KEY (organization_id) 
        REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for ai_settings
CREATE INDEX idx_ai_settings_organization_id ON ai_settings(organization_id);

-- ============================================================================
-- Table: ai_credits
-- Purpose: Track AI credit usage and limits per organization
-- ============================================================================
CREATE TABLE IF NOT EXISTS ai_credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL UNIQUE,
    
    credits_remaining INTEGER DEFAULT 2500 CHECK (credits_remaining >= 0),
    credits_total INTEGER DEFAULT 2500 CHECK (credits_total >= 0),
    credits_used INTEGER DEFAULT 0 CHECK (credits_used >= 0),
    
    reset_date TIMESTAMP WITH TIME ZONE NOT NULL,
    last_reset_date TIMESTAMP WITH TIME ZONE,
    
    -- Cost tracking
    total_spent DECIMAL(10,4) DEFAULT 0.0000,
    monthly_spent DECIMAL(10,4) DEFAULT 0.0000,
    
    -- Usage statistics
    total_optimizations INTEGER DEFAULT 0,
    successful_optimizations INTEGER DEFAULT 0,
    failed_optimizations INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_ai_credits_organization FOREIGN KEY (organization_id) 
        REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for ai_credits
CREATE INDEX idx_ai_credits_organization_id ON ai_credits(organization_id);
CREATE INDEX idx_ai_credits_reset_date ON ai_credits(reset_date);

-- ============================================================================
-- Function: Update updated_at timestamp
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
CREATE TRIGGER update_optimization_history_updated_at
    BEFORE UPDATE ON optimization_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ai_settings_updated_at
    BEFORE UPDATE ON ai_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ai_credits_updated_at
    BEFORE UPDATE ON ai_credits
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Function: Initialize AI credits for new organizations
-- ============================================================================
CREATE OR REPLACE FUNCTION initialize_ai_credits_for_organization()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO ai_credits (
        organization_id,
        credits_remaining,
        credits_total,
        credits_used,
        reset_date,
        last_reset_date
    ) VALUES (
        NEW.id,
        2500,
        2500,
        0,
        NOW() + INTERVAL '1 month',
        NOW()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to auto-initialize AI credits
CREATE TRIGGER trigger_initialize_ai_credits
    AFTER INSERT ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION initialize_ai_credits_for_organization();

-- ============================================================================
-- Function: Initialize AI settings for new organizations
-- ============================================================================
CREATE OR REPLACE FUNCTION initialize_ai_settings_for_organization()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO ai_settings (organization_id)
    VALUES (NEW.id)
    ON CONFLICT (organization_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to auto-initialize AI settings
CREATE TRIGGER trigger_initialize_ai_settings
    AFTER INSERT ON organizations
    FOR EACH ROW
    EXECUTE FUNCTION initialize_ai_settings_for_organization();

-- ============================================================================
-- Function: Reset monthly AI credits
-- Purpose: Called by a scheduled job to reset credits monthly
-- ============================================================================
CREATE OR REPLACE FUNCTION reset_monthly_ai_credits()
RETURNS void AS $$
BEGIN
    UPDATE ai_credits
    SET 
        credits_remaining = credits_total,
        credits_used = 0,
        monthly_spent = 0.0000,
        last_reset_date = NOW(),
        reset_date = NOW() + INTERVAL '1 month'
    WHERE reset_date <= NOW();
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Views for Analytics
-- ============================================================================

-- View: Optimization Analytics Summary
CREATE OR REPLACE VIEW v_optimization_analytics AS
SELECT 
    organization_id,
    COUNT(*) as total_optimizations,
    COUNT(*) FILTER (WHERE status = 'applied') as applied_count,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    AVG(score) FILTER (WHERE status = 'applied') as avg_score,
    AVG(improvement_percentage) FILTER (WHERE status = 'applied') as avg_improvement,
    SUM(cost) as total_cost,
    SUM(tokens_used) as total_tokens,
    MAX(created_at) as last_optimization_date
FROM optimization_history
GROUP BY organization_id;

-- View: Optimization by Type
CREATE OR REPLACE VIEW v_optimization_by_type AS
SELECT 
    organization_id,
    optimization_type,
    COUNT(*) as count,
    AVG(score) as avg_score,
    SUM(cost) as total_cost
FROM optimization_history
GROUP BY organization_id, optimization_type;

-- View: Daily Optimization Stats
CREATE OR REPLACE VIEW v_daily_optimization_stats AS
SELECT 
    organization_id,
    DATE(created_at) as date,
    COUNT(*) as optimizations,
    COUNT(*) FILTER (WHERE status = 'applied') as applied,
    SUM(cost) as daily_cost
FROM optimization_history
GROUP BY organization_id, DATE(created_at);

-- ============================================================================
-- Seed Data for Existing Organizations
-- ============================================================================

-- Initialize AI settings for existing organizations
INSERT INTO ai_settings (organization_id)
SELECT id FROM organizations
WHERE id NOT IN (SELECT organization_id FROM ai_settings)
ON CONFLICT (organization_id) DO NOTHING;

-- Initialize AI credits for existing organizations
INSERT INTO ai_credits (
    organization_id,
    credits_remaining,
    credits_total,
    reset_date,
    last_reset_date
)
SELECT 
    id,
    2500,
    2500,
    NOW() + INTERVAL '1 month',
    NOW()
FROM organizations
WHERE id NOT IN (SELECT organization_id FROM ai_credits)
ON CONFLICT (organization_id) DO NOTHING;

-- ============================================================================
-- Comments for Documentation
-- ============================================================================

COMMENT ON TABLE optimization_history IS 'Tracks all AI optimization attempts with results and metadata';
COMMENT ON TABLE ai_settings IS 'Stores AI optimization configuration per organization';
COMMENT ON TABLE ai_credits IS 'Manages AI credit allocation and usage tracking';

COMMENT ON COLUMN optimization_history.optimization_type IS 'Type of optimization: title, description, category, image, or bulk';
COMMENT ON COLUMN optimization_history.status IS 'Current status: pending, applied, rejected, or failed';
COMMENT ON COLUMN optimization_history.score IS 'Quality score of the optimization (0-100)';
COMMENT ON COLUMN optimization_history.improvement_percentage IS 'Calculated improvement over original';
COMMENT ON COLUMN optimization_history.metadata IS 'Additional JSON data including suggestions, reasoning, etc.';

COMMENT ON COLUMN ai_settings.default_model IS 'Default AI model to use (gpt-3.5-turbo, gpt-4, claude-3, etc.)';
COMMENT ON COLUMN ai_settings.temperature IS 'AI temperature parameter (0.0-2.0, controls randomness)';
COMMENT ON COLUMN ai_settings.top_p IS 'AI top_p parameter (0.0-1.0, nucleus sampling)';
COMMENT ON COLUMN ai_settings.min_score_threshold IS 'Minimum score required to auto-apply optimizations';

COMMENT ON COLUMN ai_credits.credits_remaining IS 'Number of AI credits remaining for current period';
COMMENT ON COLUMN ai_credits.reset_date IS 'Date when credits will reset';
COMMENT ON COLUMN ai_credits.monthly_spent IS 'Total cost spent in current month';

-- Migration complete
SELECT 'AI Optimizer tables created successfully' as status;

