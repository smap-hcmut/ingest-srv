-- Purpose:
-- Seed a one-shot, high-fanout crawl dataset for blockchain-related keyword collection.
--
-- Goal:
-- - let ingest scheduler pick every due target in a single heartbeat
-- - fan out into a large number of external_tasks across TikTok / Facebook / YouTube
-- - generate enough raw batches and UAP output to target broad blockchain topic coverage
--
-- Design choices:
-- - only 12 crawl_targets total, so all targets fit within default scheduler heartbeat_limit=20
-- - each target contains many blockchain keywords, so one scheduler tick can still create many tasks
-- - all targets are due immediately once after clean DB
-- - crawl_interval_minutes is set high (1440) so after the first claim, targets will not rerun soon
--
-- Expected runtime behavior with current ingest runtime:
-- - 3 ACTIVE crawl data sources are selected:
--   1. TikTok blockchain corpus
--   2. Facebook blockchain corpus
--   3. YouTube blockchain corpus
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
-- - written for ingest
-- - uses fixed UUIDs for repeatable inspection

BEGIN;

SET search_path TO ingest;

-- -------------------------------------------------------------------
-- Fixed IDs for repeatable debugging
-- -------------------------------------------------------------------
-- project_id
-- 44444444-4444-4444-4444-444444444444
--
-- data_sources
-- 41000000-0000-0000-0000-000000000001 : TikTok ACTIVE NORMAL
-- 41000000-0000-0000-0000-000000000002 : Facebook ACTIVE NORMAL
-- 41000000-0000-0000-0000-000000000003 : YouTube ACTIVE NORMAL
--
-- crawl_targets
-- 42000000-0000-0000-0000-000000000001 .. 42000000-0000-0000-0000-000000000012

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
    '41000000-0000-0000-0000-000000000001',
    '44444444-4444-4444-4444-444444444444',
    'Blockchain Massive Corpus - TikTok',
    'One-shot TikTok blockchain keyword corpus for broad ecosystem sampling.',
    'TIKTOK',
    'CRAWL',
    'ACTIVE',
    '{
      "seed_purpose": "blockchain_massive_one_shot",
      "expected_runtime": "full_flow",
      "note": "Grouped blockchain keywords for one scheduler tick fan-out"
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
    '41000000-0000-0000-0000-000000000002',
    '44444444-4444-4444-4444-444444444444',
    'Blockchain Massive Corpus - Facebook',
    'One-shot Facebook blockchain keyword corpus for broad ecosystem sampling.',
    'FACEBOOK',
    'CRAWL',
    'ACTIVE',
    '{
      "seed_purpose": "blockchain_massive_one_shot",
      "expected_runtime": "full_flow",
      "note": "Grouped blockchain keywords for one scheduler tick fan-out"
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
    '41000000-0000-0000-0000-000000000003',
    '44444444-4444-4444-4444-444444444444',
    'Blockchain Massive Corpus - YouTube',
    'One-shot YouTube blockchain keyword corpus for broad ecosystem sampling.',
    'YOUTUBE',
    'CRAWL',
    'ACTIVE',
    '{
      "seed_purpose": "blockchain_massive_one_shot",
      "expected_runtime": "full_flow",
      "note": "Grouped blockchain keywords for one scheduler tick fan-out"
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
    '42000000-0000-0000-0000-000000000001',
    '41000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "blockchain là gì",
      "crypto là gì",
      "web3 là gì",
      "bitcoin",
      "ethereum",
      "solana",
      "bnb chain",
      "avalanche",
      "layer 1",
      "layer 2",
      "altcoin",
      "token ecosystem",
      "onchain trend",
      "blockchain việt nam",
      "crypto vietnam",
      "web3 vietnam",
      "coin narrative",
      "bull run crypto"
    ]'::jsonb,
    'Blockchain TikTok - Ecosystem Overview',
    '{
      "seed_group": "ecosystem_overview",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000002',
    '41000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "defi",
      "dex",
      "uniswap",
      "pancakeswap",
      "aave",
      "lending crypto",
      "yield farming",
      "staking",
      "liquidity pool",
      "airdrop",
      "memecoin",
      "spot trade crypto",
      "future trade crypto",
      "copy trade crypto",
      "gem crypto",
      "coin listing",
      "defi strategy",
      "token unlock"
    ]'::jsonb,
    'Blockchain TikTok - DeFi & Trading',
    '{
      "seed_group": "defi_trading",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000003',
    '41000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "ví lạnh crypto",
      "ví nóng crypto",
      "metamask",
      "trust wallet",
      "phantom wallet",
      "okx wallet",
      "private key",
      "seed phrase",
      "bridge crypto",
      "cross chain",
      "onchain wallet",
      "rpc blockchain",
      "explorer blockchain",
      "smart contract",
      "gas fee ethereum",
      "solana fee",
      "node validator",
      "restaking"
    ]'::jsonb,
    'Blockchain TikTok - Wallets & Infra',
    '{
      "seed_group": "wallets_infra",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000004',
    '41000000-0000-0000-0000-000000000001',
    'KEYWORD',
    '[
      "scam crypto",
      "rug pull",
      "hack bridge",
      "lừa đảo crypto",
      "bảo mật ví crypto",
      "kyc sàn",
      "regulation crypto",
      "etf bitcoin",
      "thuế crypto",
      "pháp lý crypto",
      "proof of reserve",
      "audit smart contract",
      "revoke approval",
      "phishing wallet",
      "security onchain",
      "stablecoin risk",
      "fraud token",
      "crypto education"
    ]'::jsonb,
    'Blockchain TikTok - Security & Regulation',
    '{
      "seed_group": "security_regulation",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000005',
    '41000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "bitcoin",
      "ethereum",
      "solana",
      "bnb chain",
      "ton blockchain",
      "sui blockchain",
      "aptos",
      "base chain",
      "arbitrum",
      "optimism",
      "layer 2",
      "real world asset crypto",
      "tokenization",
      "web3 startup",
      "crypto community",
      "blockchain ecosystem",
      "onchain adoption",
      "digital asset"
    ]'::jsonb,
    'Blockchain Facebook - Ecosystem Overview',
    '{
      "seed_group": "ecosystem_overview",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000006',
    '41000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "defi",
      "yield farming",
      "staking",
      "liquidity mining",
      "airdrop",
      "retroactive",
      "launchpool",
      "launchpad crypto",
      "copy trade crypto",
      "signal crypto",
      "altcoin season",
      "memecoin",
      "dex volume",
      "onchain gem",
      "wallet tracking",
      "smart money crypto",
      "token analysis",
      "portfolio crypto"
    ]'::jsonb,
    'Blockchain Facebook - DeFi & Trading',
    '{
      "seed_group": "defi_trading",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000007',
    '41000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "metamask",
      "trust wallet",
      "ledger nano",
      "trezor",
      "phantom wallet",
      "safe wallet",
      "seed phrase",
      "private key",
      "wallet connect",
      "bridge crypto",
      "cross chain bridge",
      "validator node",
      "rpc endpoint",
      "block explorer",
      "smart contract audit",
      "gas war",
      "onchain tool",
      "defi dashboard"
    ]'::jsonb,
    'Blockchain Facebook - Wallets & Infra',
    '{
      "seed_group": "wallets_infra",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000008',
    '41000000-0000-0000-0000-000000000002',
    'KEYWORD',
    '[
      "scam crypto",
      "rug pull",
      "honeypot token",
      "wallet drain",
      "phishing crypto",
      "revoke token approval",
      "proof of reserve",
      "stablecoin depeg",
      "regulation crypto",
      "crypto compliance",
      "kyc exchange",
      "aml crypto",
      "etf bitcoin",
      "fed crypto impact",
      "macro crypto",
      "black swan crypto",
      "risk management crypto",
      "crypto cảnh báo"
    ]'::jsonb,
    'Blockchain Facebook - Security & Regulation',
    '{
      "seed_group": "security_regulation",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000009',
    '41000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "bitcoin",
      "ethereum",
      "solana",
      "web3 là gì",
      "blockchain là gì",
      "defi là gì",
      "layer 2 là gì",
      "rollup",
      "zero knowledge proof",
      "zksync",
      "arbitrum",
      "base chain",
      "crypto documentary",
      "onchain analysis",
      "blockchain tutorial",
      "crypto explain",
      "tokenomics",
      "market cycle crypto"
    ]'::jsonb,
    'Blockchain YouTube - Ecosystem Overview',
    '{
      "seed_group": "ecosystem_overview",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000010',
    '41000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "defi strategy",
      "yield farming",
      "staking guide",
      "airdrops 2026",
      "airdrop tutorial",
      "perpetual trading",
      "futures crypto",
      "copy trade crypto",
      "smart money tracking",
      "wallet tracking",
      "dex screener",
      "onchain data",
      "token unlock",
      "altcoin analysis",
      "memecoin strategy",
      "crypto portfolio",
      "entry exit crypto",
      "market structure crypto"
    ]'::jsonb,
    'Blockchain YouTube - DeFi & Trading',
    '{
      "seed_group": "defi_trading",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000011',
    '41000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "metamask tutorial",
      "phantom wallet tutorial",
      "ledger setup",
      "cold wallet guide",
      "seed phrase backup",
      "bridge tutorial",
      "cross chain transfer",
      "gas fee explained",
      "smart contract basics",
      "solidity beginner",
      "validator explained",
      "node setup blockchain",
      "restaking explained",
      "liquid staking",
      "wallet security setup",
      "rpc explained",
      "block explorer tutorial",
      "onchain toolkit"
    ]'::jsonb,
    'Blockchain YouTube - Wallets & Infra',
    '{
      "seed_group": "wallets_infra",
      "seed_theme": "blockchain",
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
    '42000000-0000-0000-0000-000000000012',
    '41000000-0000-0000-0000-000000000003',
    'KEYWORD',
    '[
      "scam crypto",
      "rug pull explained",
      "wallet drain scam",
      "phishing metamask",
      "revoke approval guide",
      "stablecoin risk",
      "proof of reserve explained",
      "exchange collapse",
      "crypto regulation 2026",
      "bitcoin etf impact",
      "legal crypto vietnam",
      "tax crypto",
      "kyc explained",
      "aml explained",
      "security checklist crypto",
      "avoid scam token",
      "risk management crypto",
      "crypto beginner mistakes"
    ]'::jsonb,
    'Blockchain YouTube - Security & Regulation',
    '{
      "seed_group": "security_regulation",
      "seed_theme": "blockchain",
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
-- FROM ingest.crawl_targets ct
-- JOIN ingest.data_sources ds ON ds.id = ct.data_source_id
-- WHERE ds.project_id = '44444444-4444-4444-4444-444444444444'
-- ORDER BY ct.priority DESC, ct.created_at ASC;
--
-- SELECT source_type, COUNT(*) AS target_count
-- FROM ingest.data_sources ds
-- JOIN ingest.crawl_targets ct ON ct.data_source_id = ds.id
-- WHERE ds.project_id = '44444444-4444-4444-4444-444444444444'
-- GROUP BY source_type
-- ORDER BY source_type;
