-- =====================================================
-- Seed Defaults: ingest
-- Purpose: Store system default crawl intervals and mode multipliers
-- Note: Used as fallback config for scheduler/adaptive crawl
-- Multiplier: effective_interval = target.crawl_interval_minutes × multiplier
-- =====================================================

BEGIN;

CREATE TABLE IF NOT EXISTS ingest.crawl_mode_defaults (
    mode ingest.crawl_mode PRIMARY KEY,
    interval_minutes INTEGER NOT NULL,
    mode_multiplier NUMERIC(4,2) NOT NULL DEFAULT 1.0,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_crawl_mode_defaults_interval_positive
        CHECK (interval_minutes > 0),
    CONSTRAINT chk_crawl_mode_defaults_multiplier_positive
        CHECK (mode_multiplier > 0)
);

CREATE TRIGGER trg_crawl_mode_defaults_set_updated_at
BEFORE UPDATE ON ingest.crawl_mode_defaults
FOR EACH ROW
EXECUTE FUNCTION ingest.set_updated_at();

INSERT INTO ingest.crawl_mode_defaults (
    mode,
    interval_minutes,
    mode_multiplier,
    description
) VALUES
    ('SLEEP',  60, 5.0, 'Low-frequency mode: effective = target_interval × 5.0'),
    ('NORMAL', 11, 1.0, 'Standard mode: effective = target_interval × 1.0'),
    ('CRISIS',  2, 0.2, 'Crisis escalation mode: effective = target_interval × 0.2')
ON CONFLICT (mode) DO UPDATE
SET
    interval_minutes = EXCLUDED.interval_minutes,
    mode_multiplier = EXCLUDED.mode_multiplier,
    description = EXCLUDED.description,
    updated_at = NOW();

COMMIT;
