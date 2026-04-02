-- Purpose:
-- Seed a one-shot, high-fanout crawl dataset for beer-related keyword collection.
--
-- Goal:
-- - let ingest scheduler pick every due target in a single heartbeat
-- - fan out into a large number of external_tasks across TikTok / Facebook / YouTube
-- - generate enough raw batches and UAP output to target 10k+ UAP samples
--
-- Design choices:
-- - only 12 crawl_targets total, so all targets fit within default scheduler heartbeat_limit=20
-- - each target contains many beer keywords, so one scheduler tick can still create many tasks
-- - all targets are due immediately once after clean DB
-- - crawl_interval_minutes is set high (1440) so after the first claim, targets will not rerun soon
--
-- Expected runtime behavior with current ingest runtime:
-- - 3 ACTIVE crawl data sources are selected:
--   1. TikTok beer corpus
--   2. Facebook beer corpus
--   3. YouTube beer corpus
-- - 12 KEYWORD grouped targets are due immediately
-- - scheduler should create 12 scheduled_jobs in one tick
-- - runtime should fan out 216 external_tasks total
--   - 72 TikTok full_flow tasks
--   - 72 Facebook full_flow tasks
--   - 72 YouTube full_flow tasks
--
-- Important:
-- - this script only seeds data_sources + crawl_targets
-- - scheduled_jobs / external_tasks / raw_batches are created by runtime
-- - written for schema_ingest
-- - uses fixed UUIDs for repeatable inspection

BEGIN;

SET search_path TO schema_ingest;

-- -------------------------------------------------------------------
-- Fixed IDs for repeatable debugging
-- -------------------------------------------------------------------
-- project_id
-- 33333333-3333-3333-3333-333333333333
--
-- data_sources
-- 31000000-0000-0000-0000-000000000001 : TikTok ACTIVE NORMAL
-- 31000000-0000-0000-0000-000000000002 : Facebook ACTIVE NORMAL
-- 31000000-0000-0000-0000-000000000003 : YouTube ACTIVE NORMAL
--
-- crawl_targets
-- 32000000-0000-0000-0000-000000000001 .. 32000000-0000-0000-0000-000000000012

-- -------------------------------------------------------------------
-- Data sources
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
) VALUES
(
    '31000000-0000-0000-0000-000000000001',
    '33333333-3333-3333-3333-333333333333',
    'Beer Massive Corpus - TikTok',
    'One-shot TikTok beer keyword corpus for large UAP sampling.',
    'TIKTOK',
    'CRAWL',
    'ACTIVE',
    '{
      "seed_purpose": "beer_massive_one_shot",
      "expected_runtime": "full_flow",
      "note": "Grouped beer keywords for one scheduler tick fan-out"
    }'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    1440,
    NOW(),
    NOW(),
    NOW()
),
(
    '31000000-0000-0000-0000-000000000002',
    '33333333-3333-3333-3333-333333333333',
    'Beer Massive Corpus - Facebook',
    'One-shot Facebook beer keyword corpus for large UAP sampling.',
    'FACEBOOK',
    'CRAWL',
    'ACTIVE',
    '{
      "seed_purpose": "beer_massive_one_shot",
      "expected_runtime": "full_flow",
      "note": "Grouped beer keywords for one scheduler tick fan-out"
    }'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    1440,
    NOW(),
    NOW(),
    NOW()
),
(
    '31000000-0000-0000-0000-000000000003',
    '33333333-3333-3333-3333-333333333333',
    'Beer Massive Corpus - YouTube',
    'One-shot YouTube beer keyword corpus for large UAP sampling.',
    'YOUTUBE',
    'CRAWL',
    'ACTIVE',
    '{
      "seed_purpose": "beer_massive_one_shot",
      "expected_runtime": "full_flow",
      "note": "Grouped beer keywords for one scheduler tick fan-out"
    }'::jsonb,
    'NOT_REQUIRED',
    'NOT_REQUIRED',
    'NORMAL',
    1440,
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
-- 12 targets x 18 keywords = 216 external tasks in one scheduler tick
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
    '32000000-0000-0000-0000-000000000001',
    '31000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "bia heineken",
      "bia tiger",
      "bia saigon",
      "bia 333",
      "bia larue",
      "bia sapporo",
      "bia budweiser",
      "bia corona",
      "bia strongbow",
      "bia thủ công",
      "craft beer vietnam",
      "bia ngon",
      "review bia",
      "uống bia chill",
      "mồi nhậu với bia",
      "quán bia ngon",
      "bia việt nam",
      "bia nhập khẩu"
    ]'::jsonb,
    'Beer TikTok - Brands & Discovery',
    '{
      "seed_group": "brands_discovery",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    120,
    1440,
    NOW() - INTERVAL '3 hours',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000002',
    '31000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "bia không cồn",
      "bia trái cây",
      "bia đen",
      "bia lúa mì",
      "bia ipa",
      "bia stout",
      "bia pilsner",
      "bia lager",
      "bia ale",
      "bia thủ công hà nội",
      "bia thủ công sài gòn",
      "beer tasting",
      "beer pairing",
      "bia với đồ nướng",
      "bia với hải sản",
      "bia với đồ cay",
      "beer lover",
      "beer flight"
    ]'::jsonb,
    'Beer TikTok - Styles & Pairing',
    '{
      "seed_group": "styles_pairing",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    119,
    1440,
    NOW() - INTERVAL '2 hours 58 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000003',
    '31000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "giá bia heineken",
      "giá bia tiger",
      "giá bia saigon special",
      "khuyến mãi bia",
      "mua bia ở đâu",
      "bia tết",
      "thùng bia",
      "lon bia",
      "chai bia",
      "bia draft",
      "bia tươi",
      "bia cho tiệc",
      "bia cho đám cưới",
      "bia cho bbq",
      "bia cho quán ăn",
      "beer deal",
      "beer promotion",
      "beer combo"
    ]'::jsonb,
    'Beer TikTok - Purchase Intent',
    '{
      "seed_group": "purchase_intent",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    118,
    1440,
    NOW() - INTERVAL '2 hours 56 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000004',
    '31000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "uống bia văn minh",
      "uống bia có trách nhiệm",
      "bia và lái xe",
      "say bia",
      "phân biệt các loại bia",
      "cách uống bia ngon",
      "nhiệt độ uống bia",
      "ly uống bia",
      "rót bia đúng cách",
      "bọt bia",
      "beer etiquette",
      "beer tips",
      "beer hack",
      "beer facts",
      "bia bao nhiêu calo",
      "bia và sức khỏe",
      "uống bia mập không",
      "beer myth"
    ]'::jsonb,
    'Beer TikTok - Education & Lifestyle',
    '{
      "seed_group": "education_lifestyle",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    117,
    1440,
    NOW() - INTERVAL '2 hours 54 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000005',
    '31000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "bia heineken",
      "bia tiger",
      "bia saigon",
      "bia 333",
      "bia larue",
      "bia sapporo",
      "bia budweiser",
      "bia corona",
      "bia ngon",
      "review bia",
      "uống bia chill",
      "mồi nhậu với bia",
      "quán bia ngon",
      "bia việt nam",
      "beer vietnam",
      "beer lover",
      "beer review",
      "beer night"
    ]'::jsonb,
    'Beer Facebook - Brands & Discovery',
    '{
      "seed_group": "brands_discovery",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    116,
    1440,
    NOW() - INTERVAL '2 hours 52 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000006',
    '31000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "bia thủ công",
      "craft beer vietnam",
      "bia ipa",
      "bia stout",
      "bia lager",
      "bia ale",
      "bia lúa mì",
      "bia đen",
      "bia trái cây",
      "bia không cồn",
      "beer tasting",
      "beer pairing",
      "bia với đồ nướng",
      "bia với hải sản",
      "bia với đồ cay",
      "beer flight",
      "beer menu",
      "beer culture"
    ]'::jsonb,
    'Beer Facebook - Styles & Pairing',
    '{
      "seed_group": "styles_pairing",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    115,
    1440,
    NOW() - INTERVAL '2 hours 50 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000007',
    '31000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "giá bia heineken",
      "giá bia tiger",
      "giá bia saigon special",
      "khuyến mãi bia",
      "mua bia ở đâu",
      "bia tết",
      "thùng bia",
      "lon bia",
      "chai bia",
      "bia draft",
      "bia tươi",
      "bia cho tiệc",
      "bia cho quán ăn",
      "beer deal",
      "beer promotion",
      "beer combo",
      "beer wholesale",
      "beer distributor"
    ]'::jsonb,
    'Beer Facebook - Purchase Intent',
    '{
      "seed_group": "purchase_intent",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    114,
    1440,
    NOW() - INTERVAL '2 hours 48 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000008',
    '31000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "uống bia văn minh",
      "uống bia có trách nhiệm",
      "bia và lái xe",
      "say bia",
      "phân biệt các loại bia",
      "cách uống bia ngon",
      "nhiệt độ uống bia",
      "ly uống bia",
      "rót bia đúng cách",
      "bọt bia",
      "beer etiquette",
      "beer tips",
      "beer facts",
      "bia bao nhiêu calo",
      "bia và sức khỏe",
      "uống bia mập không",
      "beer myth",
      "beer trivia"
    ]'::jsonb,
    'Beer Facebook - Education & Lifestyle',
    '{
      "seed_group": "education_lifestyle",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    113,
    1440,
    NOW() - INTERVAL '2 hours 46 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000009',
    '31000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "bia heineken",
      "bia tiger",
      "bia saigon",
      "bia 333",
      "bia larue",
      "bia sapporo",
      "bia budweiser",
      "bia corona",
      "bia ngon",
      "review bia",
      "beer review",
      "beer vlog",
      "uống bia chill",
      "mồi nhậu với bia",
      "quán bia ngon",
      "bia việt nam",
      "beer vietnam",
      "best beer"
    ]'::jsonb,
    'Beer YouTube - Brands & Discovery',
    '{
      "seed_group": "brands_discovery",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    112,
    1440,
    NOW() - INTERVAL '2 hours 44 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000010',
    '31000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "bia thủ công",
      "craft beer vietnam",
      "bia ipa",
      "bia stout",
      "bia lager",
      "bia ale",
      "bia lúa mì",
      "bia đen",
      "bia trái cây",
      "bia không cồn",
      "beer tasting",
      "beer pairing",
      "bia với đồ nướng",
      "bia với hải sản",
      "bia với đồ cay",
      "beer flight",
      "beer guide",
      "beer documentary"
    ]'::jsonb,
    'Beer YouTube - Styles & Pairing',
    '{
      "seed_group": "styles_pairing",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    111,
    1440,
    NOW() - INTERVAL '2 hours 42 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000011',
    '31000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "giá bia heineken",
      "giá bia tiger",
      "giá bia saigon special",
      "khuyến mãi bia",
      "mua bia ở đâu",
      "bia tết",
      "thùng bia",
      "lon bia",
      "chai bia",
      "bia draft",
      "bia tươi",
      "bia cho tiệc",
      "bia cho bbq",
      "beer deal",
      "beer promotion",
      "beer combo",
      "beer shop",
      "beer unboxing"
    ]'::jsonb,
    'Beer YouTube - Purchase Intent',
    '{
      "seed_group": "purchase_intent",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    110,
    1440,
    NOW() - INTERVAL '2 hours 40 minutes',
    NOW(),
    NOW()
),
(
    '32000000-0000-0000-0000-000000000012',
    '31000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "uống bia văn minh",
      "uống bia có trách nhiệm",
      "bia và lái xe",
      "say bia",
      "phân biệt các loại bia",
      "cách uống bia ngon",
      "nhiệt độ uống bia",
      "ly uống bia",
      "rót bia đúng cách",
      "bọt bia",
      "beer etiquette",
      "beer tips",
      "beer facts",
      "bia bao nhiêu calo",
      "bia và sức khỏe",
      "uống bia mập không",
      "beer myth",
      "beer science"
    ]'::jsonb,
    'Beer YouTube - Education & Lifestyle',
    '{
      "seed_group": "education_lifestyle",
      "seed_theme": "beer",
      "one_shot": true
    }'::jsonb,
    TRUE,
    109,
    1440,
    NOW() - INTERVAL '2 hours 38 minutes',
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
-- WHERE ds.project_id = '33333333-3333-3333-3333-333333333333'
-- ORDER BY ct.priority DESC, ct.created_at ASC;
--
-- SELECT source_type, COUNT(*) AS target_count
-- FROM schema_ingest.data_sources ds
-- JOIN schema_ingest.crawl_targets ct ON ct.data_source_id = ds.id
-- WHERE ds.project_id = '33333333-3333-3333-3333-333333333333'
-- GROUP BY source_type
-- ORDER BY source_type;
