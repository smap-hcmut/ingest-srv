-- =====================================================
-- Init Schema: ingest
-- Purpose: Initialize database schema for Ingest Service (V1)
-- Based on: ingest_project_schema_alignment_proposal.md v1.5
-- Tables: 7 | ENUMs: 11
-- Note: This is a one-shot init migration. Do not rerun manually.
-- =====================================================

BEGIN;

-- CREATE SCHEMA IF NOT EXISTS ingest;
-- ALTER SCHEMA ingest OWNER TO ingest_master;

CREATE TYPE ingest.source_type AS ENUM (
    'TIKTOK',
    'FACEBOOK',
    'YOUTUBE',
    'FILE_UPLOAD',
    'WEBHOOK'
);

CREATE TYPE ingest.source_category AS ENUM (
    'CRAWL',
    'PASSIVE'
);

CREATE TYPE ingest.source_status AS ENUM (
    'PENDING',
    'READY',
    'ACTIVE',
    'PAUSED',
    'FAILED',
    'COMPLETED',
    'ARCHIVED'
);

CREATE TYPE ingest.onboarding_status AS ENUM (
    'NOT_REQUIRED',
    'PENDING',
    'ANALYZING',
    'SUGGESTED',
    'CONFIRMED',
    'FAILED'
);

CREATE TYPE ingest.dryrun_status AS ENUM (
    'NOT_REQUIRED',
    'PENDING',
    'RUNNING',
    'SUCCESS',
    'WARNING',
    'FAILED'
);

CREATE TYPE ingest.crawl_mode AS ENUM (
    'SLEEP',
    'NORMAL',
    'CRISIS'
);

CREATE TYPE ingest.job_status AS ENUM (
    'PENDING',
    'RUNNING',
    'SUCCESS',
    'PARTIAL',
    'FAILED',
    'CANCELLED'
);

CREATE TYPE ingest.batch_status AS ENUM (
    'RECEIVED',
    'DOWNLOADED',
    'PARSED',
    'FAILED'
);

CREATE TYPE ingest.publish_status AS ENUM (
    'PENDING',
    'PUBLISHING',
    'SUCCESS',
    'FAILED'
);

CREATE TYPE ingest.trigger_type AS ENUM (
    'MANUAL',
    'SCHEDULED',
    'PROJECT_EVENT',
    'CRISIS_EVENT',
    'WEBHOOK_PUSH'
);

CREATE TYPE ingest.target_type AS ENUM (
    'KEYWORD',
    'PROFILE',
    'POST_URL'
);

CREATE OR REPLACE FUNCTION ingest.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE ingest.data_sources (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source_type ingest.source_type NOT NULL,
    source_category ingest.source_category NOT NULL,
    status ingest.source_status NOT NULL DEFAULT 'PENDING',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,      -- crawl options chung (không chứa targets)
    account_ref JSONB,                               -- DEPRECATED v1.5: dùng crawl_targets thay thế
    mapping_rules JSONB,
    onboarding_status ingest.onboarding_status NOT NULL DEFAULT 'NOT_REQUIRED',
    dryrun_status ingest.dryrun_status NOT NULL DEFAULT 'NOT_REQUIRED',
    dryrun_last_result_id UUID,
    crawl_mode ingest.crawl_mode,
    crawl_interval_minutes INTEGER,                  -- DEFAULT cho target mới, scheduling thực tế ở crawl_targets
    next_crawl_at TIMESTAMPTZ,                       -- DEPRECATED v1.5: dùng crawl_targets.next_crawl_at
    last_crawl_at TIMESTAMPTZ,                       -- DEPRECATED v1.5: dùng crawl_targets.last_crawl_at
    last_success_at TIMESTAMPTZ,                     -- DEPRECATED v1.5: dùng crawl_targets.last_success_at
    last_error_at TIMESTAMPTZ,                       -- DEPRECATED v1.5: dùng crawl_targets.last_error_at
    last_error_message TEXT,                          -- DEPRECATED v1.5: dùng crawl_targets.last_error_message
    webhook_id VARCHAR(255),
    webhook_secret_encrypted TEXT,
    created_by VARCHAR(255),
    activated_at TIMESTAMPTZ,
    paused_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_data_sources_type_category
        CHECK (
            (source_type IN ('TIKTOK', 'FACEBOOK', 'YOUTUBE') AND source_category = 'CRAWL') OR
            (source_type IN ('FILE_UPLOAD', 'WEBHOOK') AND source_category = 'PASSIVE')
        ),
    CONSTRAINT chk_data_sources_crawl_requirements
        CHECK (
            source_category <> 'CRAWL'
            OR crawl_mode IS NOT NULL
        ),
    CONSTRAINT chk_data_sources_webhook_ready_requirements
        CHECK (
            source_type <> 'WEBHOOK'
            OR status NOT IN ('READY', 'ACTIVE')
            OR (webhook_id IS NOT NULL AND webhook_secret_encrypted IS NOT NULL)
        ),
    CONSTRAINT chk_data_sources_completed_only_file_upload
        CHECK (
            status <> 'COMPLETED' OR source_type = 'FILE_UPLOAD'
        )
);

CREATE TRIGGER trg_data_sources_set_updated_at
BEFORE UPDATE ON ingest.data_sources
FOR EACH ROW
EXECUTE FUNCTION ingest.set_updated_at();

CREATE TABLE ingest.dryrun_results (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES ingest.data_sources(id),
    project_id UUID NOT NULL,
    target_id UUID,                                  -- nullable: NULL cho FILE_UPLOAD/WEBHOOK
    job_id VARCHAR(255),
    status ingest.dryrun_status NOT NULL DEFAULT 'PENDING',
    sample_count INTEGER NOT NULL DEFAULT 0,
    total_found INTEGER,
    sample_data JSONB,
    warnings JSONB,
    error_message TEXT,
    requested_by VARCHAR(255),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_dryrun_results_sample_count_non_negative CHECK (sample_count >= 0),
    CONSTRAINT chk_dryrun_results_total_found_non_negative CHECK (total_found IS NULL OR total_found >= 0)
);

CREATE TABLE ingest.scheduled_jobs (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES ingest.data_sources(id),
    project_id UUID NOT NULL,
    target_id UUID,                                  -- nullable: NULL cho batch job, FILE_UPLOAD
    status ingest.job_status NOT NULL DEFAULT 'PENDING',
    trigger_type ingest.trigger_type NOT NULL,
    cron_expr VARCHAR(100),
    crawl_mode ingest.crawl_mode NOT NULL,
    scheduled_for TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    retry_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scheduled_jobs_retry_count_non_negative CHECK (retry_count >= 0)
);

CREATE TABLE ingest.external_tasks (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES ingest.data_sources(id),
    project_id UUID NOT NULL,
    domain_type_code VARCHAR(50) NOT NULL DEFAULT '_default',
    target_id UUID,                                  -- nullable: NULL cho FILE_UPLOAD hoặc batch task
    scheduled_job_id UUID REFERENCES ingest.scheduled_jobs(id),
    task_id UUID NOT NULL UNIQUE,
    platform VARCHAR(50) NOT NULL,
    task_type VARCHAR(100) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status ingest.job_status NOT NULL DEFAULT 'PENDING',
    published_at TIMESTAMPTZ,
    response_received_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ingest.raw_batches (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES ingest.data_sources(id),
    project_id UUID NOT NULL,
    domain_type_code VARCHAR(50) NOT NULL DEFAULT '_default',
    external_task_id UUID REFERENCES ingest.external_tasks(id),
    batch_id VARCHAR(255) NOT NULL,
    status ingest.batch_status NOT NULL DEFAULT 'RECEIVED',
    storage_bucket VARCHAR(100) NOT NULL,
    storage_path TEXT NOT NULL,
    storage_url TEXT,
    item_count INTEGER,
    size_bytes BIGINT,
    checksum VARCHAR(128),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    parsed_at TIMESTAMPTZ,
    publish_status ingest.publish_status NOT NULL DEFAULT 'PENDING',
    publish_record_count INTEGER NOT NULL DEFAULT 0,
    first_event_id UUID,
    last_event_id UUID,
    uap_published_at TIMESTAMPTZ,
    error_message TEXT,
    publish_error TEXT,
    raw_metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_raw_batches_source_batch UNIQUE (source_id, batch_id),
    CONSTRAINT chk_raw_batches_item_count_non_negative CHECK (item_count IS NULL OR item_count >= 0),
    CONSTRAINT chk_raw_batches_size_bytes_non_negative CHECK (size_bytes IS NULL OR size_bytes >= 0),
    CONSTRAINT chk_raw_batches_publish_record_count_non_negative CHECK (publish_record_count >= 0)
);

CREATE TABLE ingest.crawl_mode_changes (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES ingest.data_sources(id),
    project_id UUID NOT NULL,
    trigger_type ingest.trigger_type NOT NULL,
    from_mode ingest.crawl_mode NOT NULL,
    to_mode ingest.crawl_mode NOT NULL,
    from_interval_minutes INTEGER NOT NULL,
    to_interval_minutes INTEGER NOT NULL,
    reason TEXT,
    event_ref VARCHAR(255),
    triggered_by VARCHAR(255),
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_crawl_mode_changes_from_interval_non_negative CHECK (from_interval_minutes >= 0),
    CONSTRAINT chk_crawl_mode_changes_to_interval_non_negative CHECK (to_interval_minutes >= 0)
);

-- =====================================================
-- Table: crawl_targets
-- Per-target crawl schedule (keyword, profile, post URL)
-- =====================================================
CREATE TABLE ingest.crawl_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_source_id UUID NOT NULL REFERENCES ingest.data_sources(id) ON DELETE CASCADE,
    target_type ingest.target_type NOT NULL,
    values JSONB NOT NULL,
    label TEXT,
    platform_meta JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    priority INTEGER NOT NULL DEFAULT 0,

    -- Per-target crawl schedule
    crawl_interval_minutes INTEGER NOT NULL DEFAULT 11,
    next_crawl_at TIMESTAMPTZ,
    last_crawl_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_error_at TIMESTAMPTZ,
    last_error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_crawl_targets_interval_positive CHECK (crawl_interval_minutes > 0)
);

CREATE TRIGGER trg_crawl_targets_set_updated_at
BEFORE UPDATE ON ingest.crawl_targets
FOR EACH ROW
EXECUTE FUNCTION ingest.set_updated_at();

-- FK back-references from dryrun_results / scheduled_jobs / external_tasks → crawl_targets
ALTER TABLE ingest.dryrun_results
    ADD CONSTRAINT fk_dryrun_results_target
    FOREIGN KEY (target_id) REFERENCES ingest.crawl_targets(id) ON DELETE SET NULL;

ALTER TABLE ingest.scheduled_jobs
    ADD CONSTRAINT fk_scheduled_jobs_target
    FOREIGN KEY (target_id) REFERENCES ingest.crawl_targets(id) ON DELETE SET NULL;

ALTER TABLE ingest.external_tasks
    ADD CONSTRAINT fk_external_tasks_target
    FOREIGN KEY (target_id) REFERENCES ingest.crawl_targets(id) ON DELETE SET NULL;

ALTER TABLE ingest.data_sources
    ADD CONSTRAINT fk_data_sources_dryrun_last_result
    FOREIGN KEY (dryrun_last_result_id)
    REFERENCES ingest.dryrun_results(id);

CREATE INDEX idx_data_sources_project_deleted
    ON ingest.data_sources (project_id, deleted_at);

CREATE INDEX idx_data_sources_status_category
    ON ingest.data_sources (status, source_category);

CREATE INDEX idx_data_sources_project_source_type
    ON ingest.data_sources (project_id, source_type);

-- DEPRECATED: scheduler now uses crawl_targets.next_crawl_at
-- Kept for backward-compat summary queries
CREATE INDEX idx_data_sources_next_crawl_active
    ON ingest.data_sources (next_crawl_at)
    WHERE deleted_at IS NULL AND status = 'ACTIVE' AND source_category = 'CRAWL';

CREATE UNIQUE INDEX idx_data_sources_webhook_id_unique
    ON ingest.data_sources (webhook_id)
    WHERE webhook_id IS NOT NULL;

CREATE INDEX idx_data_sources_config_gin
    ON ingest.data_sources USING GIN (config);

CREATE INDEX idx_data_sources_mapping_rules_gin
    ON ingest.data_sources USING GIN (mapping_rules);

CREATE INDEX idx_dryrun_results_project_created_desc
    ON ingest.dryrun_results (project_id, created_at DESC);

CREATE INDEX idx_dryrun_results_source_created_desc
    ON ingest.dryrun_results (source_id, created_at DESC);

CREATE INDEX idx_dryrun_results_job_id
    ON ingest.dryrun_results (job_id);

CREATE INDEX idx_scheduled_jobs_status_scheduled_for
    ON ingest.scheduled_jobs (status, scheduled_for);

CREATE INDEX idx_scheduled_jobs_source_scheduled_for_desc
    ON ingest.scheduled_jobs (source_id, scheduled_for DESC);

CREATE INDEX idx_scheduled_jobs_project_scheduled_for_desc
    ON ingest.scheduled_jobs (project_id, scheduled_for DESC);

CREATE INDEX idx_external_tasks_source_created_desc
    ON ingest.external_tasks (source_id, created_at DESC);

CREATE INDEX idx_external_tasks_project_created_desc
    ON ingest.external_tasks (project_id, created_at DESC);

CREATE INDEX idx_external_tasks_domain_type_created_desc
    ON ingest.external_tasks (domain_type_code, created_at DESC);

CREATE INDEX idx_external_tasks_scheduled_job_id
    ON ingest.external_tasks (scheduled_job_id);

CREATE INDEX idx_external_tasks_status_created_desc
    ON ingest.external_tasks (status, created_at DESC);

CREATE INDEX idx_raw_batches_source_received_desc
    ON ingest.raw_batches (source_id, received_at DESC);

CREATE INDEX idx_raw_batches_project_received_desc
    ON ingest.raw_batches (project_id, received_at DESC);

CREATE INDEX idx_raw_batches_domain_type_received_desc
    ON ingest.raw_batches (domain_type_code, received_at DESC);

CREATE INDEX idx_raw_batches_external_task_id
    ON ingest.raw_batches (external_task_id);

CREATE INDEX idx_raw_batches_publish_status_received_desc
    ON ingest.raw_batches (publish_status, received_at DESC);

CREATE INDEX idx_raw_batches_checksum
    ON ingest.raw_batches (checksum)
    WHERE checksum IS NOT NULL;

CREATE INDEX idx_crawl_mode_changes_source_triggered_desc
    ON ingest.crawl_mode_changes (source_id, triggered_at DESC);

CREATE INDEX idx_crawl_mode_changes_project_triggered_desc
    ON ingest.crawl_mode_changes (project_id, triggered_at DESC);

-- crawl_targets indexes
CREATE INDEX idx_crawl_targets_data_source
    ON ingest.crawl_targets (data_source_id);

CREATE INDEX idx_crawl_targets_next_crawl_active
    ON ingest.crawl_targets (next_crawl_at)
    WHERE is_active = true;

CREATE INDEX idx_crawl_targets_type
    ON ingest.crawl_targets (data_source_id, target_type);

-- target_id indexes on referencing tables
CREATE INDEX idx_dryrun_results_target
    ON ingest.dryrun_results (target_id)
    WHERE target_id IS NOT NULL;

CREATE INDEX idx_scheduled_jobs_target
    ON ingest.scheduled_jobs (target_id)
    WHERE target_id IS NOT NULL;

CREATE INDEX idx_external_tasks_target
    ON ingest.external_tasks (target_id)
    WHERE target_id IS NOT NULL;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA ingest TO ingest_master;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA ingest TO ingest_master;

COMMIT;
