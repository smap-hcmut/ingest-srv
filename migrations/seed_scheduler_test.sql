-- Purpose:
-- Seed a broader scheduler test dataset for ingest-srv after the DB has been cleaned.
--
-- This script seeds:
-- - ACTIVE / PAUSED / ARCHIVED crawl data sources
-- - NORMAL / CRISIS / SLEEP crawl modes
-- - supported mappings that should dispatch successfully
-- - due targets that should be ignored by due query
-- - due targets that should be selected but fail at dispatch validation/mapping
-- - enough due targets to observe heartbeat_limit behavior by lowering the limit locally
--
-- Expected behavior with current scheduler/runtime:
-- - selected and dispatched successfully:
--   1. target tiktok_due_1 (fan-out many external_tasks from one grouped target)
--   2. target tiktok_due_null (fan-out many external_tasks from one grouped target)
--   3. target facebook_due_1
--   4. target tiktok_sleep_due
--   5. target tiktok_due_with_empty_values (runtime trims and skips empty keywords)
-- - ignored by due query:
--   6. target tiktok_future_1
--   7. target facebook_inactive
--   8. target paused_source_target
--   9. target archived_source_target
-- - selected but expected to fail during dispatch:
--  10. target youtube_due_unsupported
--  11. target facebook_missing_parse_ids
--  12. target tiktok_due_all_empty_keywords
-- - extra due targets to observe heartbeat_limit:
--  13. target tiktok_due_limit_probe_1
--  14. target tiktok_due_limit_probe_2
--
-- Notes:
-- - Scheduler currently uses crawl_targets.next_crawl_at, not data_sources.next_crawl_at.
-- - This script is written for schema_ingest.
-- - It uses fixed UUIDs so you can inspect the same records repeatedly.

BEGIN;

SET search_path TO schema_ingest;

-- -------------------------------------------------------------------
-- Fixed IDs for repeatable debugging
-- -------------------------------------------------------------------
-- project_id shared by all test sources
-- 11111111-1111-1111-1111-111111111111

-- data_sources
-- 10000000-0000-0000-0000-000000000001 : TikTok ACTIVE NORMAL
-- 10000000-0000-0000-0000-000000000002 : Facebook ACTIVE CRISIS
-- 10000000-0000-0000-0000-000000000003 : TikTok PAUSED NORMAL
-- 10000000-0000-0000-0000-000000000004 : TikTok ARCHIVED NORMAL
-- 10000000-0000-0000-0000-000000000005 : TikTok ACTIVE SLEEP
-- 10000000-0000-0000-0000-000000000006 : YouTube ACTIVE NORMAL

-- crawl_targets
-- 20000000-0000-0000-0000-000000000001 : TikTok due
-- 20000000-0000-0000-0000-000000000002 : TikTok future
-- 20000000-0000-0000-0000-000000000003 : TikTok null next_crawl_at
-- 20000000-0000-0000-0000-000000000004 : Facebook due valid parse_ids
-- 20000000-0000-0000-0000-000000000005 : Facebook inactive
-- 20000000-0000-0000-0000-000000000006 : Paused source target
-- 20000000-0000-0000-0000-000000000007 : Archived source target
-- 20000000-0000-0000-0000-000000000008 : TikTok SLEEP due
-- 20000000-0000-0000-0000-000000000009 : YouTube due unsupported mapping
-- 20000000-0000-0000-0000-000000000010 : Facebook due missing parse_ids
-- 20000000-0000-0000-0000-000000000011 : TikTok due with empty values between keywords
-- 20000000-0000-0000-0000-000000000012 : TikTok due all empty keywords
-- 20000000-0000-0000-0000-000000000013 : TikTok due limit probe 1
-- 20000000-0000-0000-0000-000000000014 : TikTok due limit probe 2

-- -------------------------------------------------------------------
-- Data sources
-- -------------------------------------------------------------------
INSERT INTO data_sources (
    id,
    project_id,
    name,
    source_type,
    source_category,
    status,
    config,
    onboarding_status,
    dryrun_status,
    crawl_mode,
    crawl_interval_minutes,
    activated_at,
    created_at,
    updated_at
) VALUES
(
    '10000000-0000-0000-0000-000000000001',
    '11111111-1111-1111-1111-111111111111',
    'Scheduler Test TikTok Active',
    'TIKTOK',
    'CRAWL',
    'ACTIVE',
    '{}'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    11,
    NOW(),
    NOW(),
    NOW()
),
(
    '10000000-0000-0000-0000-000000000002',
    '11111111-1111-1111-1111-111111111111',
    'Scheduler Test Facebook Active',
    'FACEBOOK',
    'CRAWL',
    'ACTIVE',
    '{}'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'CRISIS',
    11,
    NOW(),
    NOW(),
    NOW()
),
(
    '10000000-0000-0000-0000-000000000003',
    '11111111-1111-1111-1111-111111111111',
    'Scheduler Test TikTok Paused',
    'TIKTOK',
    'CRAWL',
    'PAUSED',
    '{}'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    11,
    NOW(),
    NOW(),
    NOW()
),
(
    '10000000-0000-0000-0000-000000000004',
    '11111111-1111-1111-1111-111111111111',
    'Scheduler Test TikTok Archived',
    'TIKTOK',
    'CRAWL',
    'ARCHIVED',
    '{}'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    11,
    NOW(),
    NOW(),
    NOW()
),
(
    '10000000-0000-0000-0000-000000000005',
    '11111111-1111-1111-1111-111111111111',
    'Scheduler Test TikTok Sleep',
    'TIKTOK',
    'CRAWL',
    'ACTIVE',
    '{}'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'SLEEP',
    11,
    NOW(),
    NOW(),
    NOW()
),
(
    '10000000-0000-0000-0000-000000000006',
    '11111111-1111-1111-1111-111111111111',
    'Scheduler Test YouTube Unsupported',
    'YOUTUBE',
    'CRAWL',
    'ACTIVE',
    '{}'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    11,
    NOW(),
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE
SET
    project_id = EXCLUDED.project_id,
    name = EXCLUDED.name,
    source_type = EXCLUDED.source_type,
    source_category = EXCLUDED.source_category,
    status = EXCLUDED.status,
    config = EXCLUDED.config,
    onboarding_status = EXCLUDED.onboarding_status,
    dryrun_status = EXCLUDED.dryrun_status,
    crawl_mode = EXCLUDED.crawl_mode,
    crawl_interval_minutes = EXCLUDED.crawl_interval_minutes,
    activated_at = EXCLUDED.activated_at,
    updated_at = NOW();

-- -------------------------------------------------------------------
-- Crawl targets
-- -------------------------------------------------------------------
INSERT INTO crawl_targets (
    id,
    data_source_id,
    target_type,
    values,
    label,
    platform_meta,
    is_active,
    priority,
    crawl_interval_minutes,
    next_crawl_at,
    created_at,
    updated_at
) VALUES
(
    '20000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '["vinfast vf8 review","vinfast ec34 review","vinfast battery update"]'::jsonb,
    'TikTok due keyword',
    NULL,
    TRUE,
    100,
    11,
    NOW() - INTERVAL '15 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '["vinfast ec34 update"]'::jsonb,
    'TikTok future keyword',
    NULL,
    TRUE,
    90,
    11,
    NOW() + INTERVAL '3 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000003',
    '10000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '["vinfast battery lease","vinfast charging speed"]'::jsonb,
    'TikTok null next_crawl_at keyword',
    NULL,
    TRUE,
    80,
    11,
    NULL,
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000004',
    '10000000-0000-0000-0000-000000000002',
    'POST_URL',
    '["https://www.facebook.com/story.php?story_fbid=pfbid02XYZ456&id=61550000000000"]'::jsonb,
    'Facebook due post detail',
    '{"parse_ids":["pfbid02XYZ456"]}'::jsonb,
    TRUE,
    95,
    10,
    NOW() - INTERVAL '2 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000005',
    '10000000-0000-0000-0000-000000000002',
    'POST_URL',
    '["https://www.facebook.com/story.php?story_fbid=pfbid02INACTIVE&id=61550000000000"]'::jsonb,
    'Facebook inactive target',
    '{"parse_ids":["pfbid02INACTIVE"]}'::jsonb,
    FALSE,
    70,
    10,
    NOW() - INTERVAL '1 minute',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000006',
    '10000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '["vinfast paused source"]'::jsonb,
    'Paused source target',
    NULL,
    TRUE,
    60,
    11,
    NOW() - INTERVAL '1 minute',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000007',
    '10000000-0000-0000-0000-000000000004',
    'KEYWORD',
    '["vinfast archived source"]'::jsonb,
    'Archived source target',
    NULL,
    TRUE,
    65,
    11,
    NOW() - INTERVAL '1 minute',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000008',
    '10000000-0000-0000-0000-000000000005',
    'KEYWORD',
    '["vinfast sleep mode monitor"]'::jsonb,
    'TikTok sleep due keyword',
    NULL,
    TRUE,
    85,
    11,
    NOW() - INTERVAL '4 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000009',
    '10000000-0000-0000-0000-000000000006',
    'PROFILE',
    '["https://www.youtube.com/@vinfast"]'::jsonb,
    'YouTube due unsupported mapping',
    NULL,
    TRUE,
    88,
    11,
    NOW() - INTERVAL '6 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000010',
    '10000000-0000-0000-0000-000000000002',
    'POST_URL',
    '["https://www.facebook.com/story.php?story_fbid=pfbid02MISSING&id=61550000000000"]'::jsonb,
    'Facebook due missing parse_ids',
    '{}'::jsonb,
    TRUE,
    92,
    10,
    NOW() - INTERVAL '3 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000011',
    '10000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '["   ","vinfast plant opening","", "vinfast warranty"]'::jsonb,
    'TikTok due keyword with empty values',
    NULL,
    TRUE,
    87,
    11,
    NOW() - INTERVAL '7 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000012',
    '10000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '["   ",""]'::jsonb,
    'TikTok due all empty keywords',
    NULL,
    TRUE,
    86,
    11,
    NOW() - INTERVAL '8 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000013',
    '10000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '["vinfast factory update","vinfast dealer expansion"]'::jsonb,
    'TikTok due limit probe 1',
    NULL,
    TRUE,
    84,
    11,
    NOW() - INTERVAL '9 minutes',
    NOW(),
    NOW()
),
(
    '20000000-0000-0000-0000-000000000014',
    '10000000-0000-0000-0000-000000000005',
    'KEYWORD',
    '["vinfast crisis monitoring","vinfast support issue"]'::jsonb,
    'TikTok due limit probe 2',
    NULL,
    TRUE,
    83,
    11,
    NOW() - INTERVAL '10 minutes',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE
SET
    data_source_id = EXCLUDED.data_source_id,
    target_type = EXCLUDED.target_type,
    values = EXCLUDED.values,
    label = EXCLUDED.label,
    platform_meta = EXCLUDED.platform_meta,
    is_active = EXCLUDED.is_active,
    priority = EXCLUDED.priority,
    crawl_interval_minutes = EXCLUDED.crawl_interval_minutes,
    next_crawl_at = EXCLUDED.next_crawl_at,
    updated_at = NOW();

COMMIT;

-- -------------------------------------------------------------------
-- Quick inspection query
-- -------------------------------------------------------------------
-- SELECT
--     ds.name AS datasource_name,
--     ds.source_type,
--     ds.status AS datasource_status,
--     ds.crawl_mode,
--     ct.id AS target_id,
--     ct.label,
--     ct.target_type,
--     ct.is_active,
--     ct.priority,
--     ct.crawl_interval_minutes,
--     ct.next_crawl_at
-- FROM schema_ingest.crawl_targets ct
-- JOIN schema_ingest.data_sources ds ON ds.id = ct.data_source_id
-- ORDER BY ct.next_crawl_at ASC NULLS FIRST, ct.priority DESC;
