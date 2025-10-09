-- ============================================================================
-- AI Optimizer Tables for Supabase
-- Run this in Supabase SQL Editor
-- ============================================================================

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- Table: optimization_history
-- Purpose: Track all AI optimization attempts and results
-- ============================================================================
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
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    applied_at TIMESTAMP WITH TIME ZONE,
    
    -- Foreign key to products
    CONSTRAINT fk_optimization_product FOREIGN KEY (product_id) 
        REFERENCES products(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_optimization_history_product_id ON optimization_history(product_id);
CREATE INDEX IF NOT EXISTS idx_optimization_history_organization_id ON optimization_history(organization_id);
CREATE INDEX IF NOT EXISTS idx_optimization_history_type ON optimization_history(optimization_type);
CREATE INDEX IF NOT EXISTS idx_optimization_history_status ON optimization_history(status);
CREATE INDEX IF NOT EXISTS idx_optimization_history_created_at ON optimization_history(created_at DESC);

-- ============================================================================
-- Table: ai_credits
-- Purpose: Track AI credit usage (simplified - one row for all)
-- ============================================================================
CREATE TABLE IF NOT EXISTS ai_credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid UNIQUE,
    
    credits_remaining INTEGER DEFAULT 2500 CHECK (credits_remaining >= 0),
    credits_total INTEGER DEFAULT 2500 CHECK (credits_total >= 0),
    credits_used INTEGER DEFAULT 0 CHECK (credits_used >= 0),
    
    reset_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() + INTERVAL '1 month',
    last_reset_date TIMESTAMP WITH TIME ZONE,
    
    -- Cost tracking
    total_cost DECIMAL(10,4) DEFAULT 0.0000,
    
    -- Usage statistics
    total_optimizations INTEGER DEFAULT 0,
    successful_optimizations INTEGER DEFAULT 0,
    failed_optimizations INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- Table: ai_settings
-- Purpose: Store AI optimization settings
-- ============================================================================
CREATE TABLE IF NOT EXISTS ai_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID DEFAULT '00000000-0000-0000-0000-000000000000'::uuid UNIQUE,
    
    -- General Settings
    default_model VARCHAR(50) DEFAULT 'gpt-3.5-turbo',
    max_tokens INTEGER DEFAULT 500 CHECK (max_tokens > 0 AND max_tokens <= 4000),
    temperature DECIMAL(3,2) DEFAULT 0.70 CHECK (temperature >= 0 AND temperature <= 2),
    top_p DECIMAL(3,2) DEFAULT 0.90 CHECK (top_p >= 0 AND top_p <= 1),
    
    -- Feature Toggles
    title_optimization BOOLEAN DEFAULT TRUE,
    description_optimization BOOLEAN DEFAULT TRUE,
    category_optimization BOOLEAN DEFAULT TRUE,
    image_optimization BOOLEAN DEFAULT FALSE,
    
    -- Quality Settings
    min_score_threshold INTEGER DEFAULT 80 CHECK (min_score_threshold >= 0 AND min_score_threshold <= 100),
    require_approval BOOLEAN DEFAULT TRUE,
    max_retries INTEGER DEFAULT 3 CHECK (max_retries >= 0 AND max_retries <= 10),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- Initialize default data
-- ============================================================================

-- Insert default AI credits
INSERT INTO ai_credits (
    organization_id,
    credits_remaining,
    credits_total,
    reset_date
) VALUES (
    '00000000-0000-0000-0000-000000000000'::uuid,
    2500,
    2500,
    NOW() + INTERVAL '1 month'
) ON CONFLICT (organization_id) DO NOTHING;

-- Insert default AI settings
INSERT INTO ai_settings (
    organization_id
) VALUES (
    '00000000-0000-0000-0000-000000000000'::uuid
) ON CONFLICT (organization_id) DO NOTHING;

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE optimization_history IS 'Tracks all AI optimization attempts with results and metadata';
COMMENT ON TABLE ai_credits IS 'Manages AI credit allocation and usage tracking';
COMMENT ON TABLE ai_settings IS 'Stores AI optimization configuration';

-- Migration complete
SELECT 'AI Optimizer tables created successfully! ✅' as status;

