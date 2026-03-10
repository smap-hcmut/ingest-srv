-- Purpose:
-- Seed grouped TikTok keyword targets for AI-oriented sample jobs.
--
-- This dataset is intended to create realistic scheduler/execution samples for:
-- 1. Brand & Core Products
-- 2. Competitors
-- 3. Features & Pain-points
-- 4. Industry Trends
--
-- Runtime expectation with current ingest scheduler:
-- - one ACTIVE TikTok crawl datasource is selected
-- - four KEYWORD grouped targets are due immediately
-- - scheduler will create 4 scheduled_jobs
-- - each target will fan out into many external_tasks (one full_flow task per keyword)
--
-- Fan-out expectation by target:
-- - brand_core_products      -> 5 external_tasks
-- - competitors             -> 7 external_tasks
-- - features_pain_points    -> 10 external_tasks
-- - industry_trends         -> 4 external_tasks
--
-- Notes:
-- - this script only seeds datasource + crawl_targets
-- - scheduled_jobs / external_tasks / raw_batches are created by runtime
-- - written for schema_ingest
-- - uses fixed UUIDs for repeatable inspection

BEGIN;

SET search_path TO schema_ingest;

-- -------------------------------------------------------------------
-- Fixed IDs for repeatable debugging
-- -------------------------------------------------------------------
-- project_id
-- 22222222-2222-2222-2222-222222222222
--
-- data_source
-- 21000000-0000-0000-0000-000000000001 : TikTok ACTIVE NORMAL
--
-- crawl_targets
-- 22000000-0000-0000-0000-000000000001 : Brand & Core Products
-- 22000000-0000-0000-0000-000000000002 : Competitors
-- 22000000-0000-0000-0000-000000000003 : Features & Pain-points
-- 22000000-0000-0000-0000-000000000004 : Industry Trends

-- -------------------------------------------------------------------
-- Data source
-- -------------------------------------------------------------------
INSERT INTO data_sources (
    id,
    project_id,
    name,
    description,
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
) VALUES (
    '21000000-0000-0000-0000-000000000001',
    '22222222-2222-2222-2222-222222222222',
    'AI Sample TikTok Keyword Corpus',
    'Seed datasource for AI sample jobs around VinFast brand, competitors, pain-points, and market trends.',
    'TIKTOK',
    'CRAWL',
    'ACTIVE',
    '{
      "seed_purpose": "ai_sample_jobs",
      "owner_team": "ai",
      "note": "Grouped TikTok full_flow keyword targets for scheduler fan-out"
    }'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    30,
    NOW(),
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE
SET
    project_id = EXCLUDED.project_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
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
    '22000000-0000-0000-0000-000000000001',
    '21000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "VinFast VF 8",
      "VF 7",
      "xe điện VinFast",
      "mua xe VF8",
      "trải nghiệm VF"
    ]'::jsonb,
    'AI Sample - Brand & Core Products',
    '{
      "seed_group": "brand_core_products",
      "seed_for_team": "ai",
      "business_goal": "sentiment_analysis",
      "notes": "Direct opinion mining around VinFast core products"
    }'::jsonb,
    TRUE,
    100,
    30,
    NOW() - INTERVAL '20 minutes',
    NOW(),
    NOW()
),
(
    '22000000-0000-0000-0000-000000000002',
    '21000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "BYD Sealion 6",
      "VF8 hay BYD",
      "Tesla",
      "Ioniq 5",
      "VF8 hay CX5",
      "Ford Territory",
      "CRV"
    ]'::jsonb,
    'AI Sample - Competitors',
    '{
      "seed_group": "competitors",
      "seed_for_team": "ai",
      "business_goal": "entity_extraction_and_comparison",
      "notes": "Direct and indirect competitor discovery keywords"
    }'::jsonb,
    TRUE,
    95,
    30,
    NOW() - INTERVAL '18 minutes',
    NOW(),
    NOW()
),
(
    '22000000-0000-0000-0000-000000000003',
    '21000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "chế độ cắm trại",
      "trợ lý ảo Vi Vi",
      "chi phí sạc điện rẻ",
      "tăng tốc nhanh",
      "sự tiện dụng xe điện",
      "lỗi phần mềm xe điện",
      "đợi sạc pin lâu",
      "xe điện bị ồn",
      "màn hình bị đơ",
      "không có nút vật lý"
    ]'::jsonb,
    'AI Sample - Features & Pain-points',
    '{
      "seed_group": "features_pain_points",
      "seed_for_team": "ai",
      "business_goal": "topic_modeling_and_usp_discovery",
      "notes": "Mixture of strengths and pain-points for theme clustering"
    }'::jsonb,
    TRUE,
    90,
    30,
    NOW() - INTERVAL '16 minutes',
    NOW(),
    NOW()
),
(
    '22000000-0000-0000-0000-000000000004',
    '21000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "có nên mua xe điện lúc này",
      "trạm sạc VinFast",
      "xe điện đi tỉnh",
      "chi phí nuôi xe điện vs xe xăng"
    ]'::jsonb,
    'AI Sample - Industry Trends',
    '{
      "seed_group": "industry_trends",
      "seed_for_team": "ai",
      "business_goal": "macro_barrier_and_trend_monitoring",
      "notes": "Market education and adoption barrier keywords"
    }'::jsonb,
    TRUE,
    85,
    30,
    NOW() - INTERVAL '14 minutes',
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
-- Optional inspection queries
-- -------------------------------------------------------------------
-- SELECT
--     ds.id AS source_id,
--     ds.name AS source_name,
--     ds.source_type,
--     ds.status,
--     ct.id AS target_id,
--     ct.label,
--     ct.priority,
--     ct.crawl_interval_minutes,
--     ct.next_crawl_at,
--     jsonb_array_length(ct.values) AS keyword_count
-- FROM schema_ingest.crawl_targets ct
-- JOIN schema_ingest.data_sources ds ON ds.id = ct.data_source_id
-- WHERE ds.id = '21000000-0000-0000-0000-000000000001'
-- ORDER BY ct.priority DESC, ct.created_at ASC;
