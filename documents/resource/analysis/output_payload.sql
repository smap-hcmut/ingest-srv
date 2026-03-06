-- ============================================================================
-- SMAP Analytics Service — Output Payload Schema
-- ============================================================================
-- Table: schema_analyst.analyzed_posts
-- Mô tả cấu trúc dữ liệu output mà pipeline persist vào PostgreSQL.
-- Mỗi record = 1 bài viết đã phân tích đầy đủ (dữ liệu gốc + kết quả AI).
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS schema_analyst;

CREATE TABLE schema_analyst.analyzed_posts (

    -- ═══════════════════════════════════════════════════════════════════════
    -- IDENTIFIERS
    -- ═══════════════════════════════════════════════════════════════════════

    -- Primary key. Lấy từ meta.id của input — ID bài viết trên platform gốc
    id                      VARCHAR(255)    PRIMARY KEY,

    -- UUID project trên SMAP. Lấy từ payload.project_id hoặc extract từ job_id.
    -- Pipeline sanitize để đảm bảo UUID hợp lệ (loại bỏ suffix nếu có)
    project_id              VARCHAR(255)    NULL,

    -- Platform nguồn, normalized uppercase: TIKTOK, FACEBOOK, YOUTUBE, INSTAGRAM, UNKNOWN
    platform                VARCHAR(50)     NOT NULL DEFAULT 'UNKNOWN',

    -- ═══════════════════════════════════════════════════════════════════════
    -- TIMESTAMPS
    -- ═══════════════════════════════════════════════════════════════════════

    -- Thời điểm bài viết được đăng trên platform. Từ meta.published_at, fallback now() UTC
    published_at            TIMESTAMPTZ     NULL,

    -- Thời điểm pipeline hoàn thành phân tích. Luôn là now() UTC tại thời điểm xử lý
    analyzed_at             TIMESTAMPTZ     NULL,

    -- ═══════════════════════════════════════════════════════════════════════
    -- SENTIMENT ANALYSIS RESULTS (Stage 4 — PhoBERT ONNX)
    -- ═══════════════════════════════════════════════════════════════════════

    -- Label cảm xúc tổng thể: POSITIVE, NEGATIVE, NEUTRAL
    -- Thresholds: positive ≥ 0.25, negative ≤ -0.25
    overall_sentiment       VARCHAR(50)     NOT NULL DEFAULT 'NEUTRAL',

    -- Điểm cảm xúc: -1.0 (rất tiêu cực) → 1.0 (rất tích cực)
    -- Output trực tiếp từ PhoBERT model
    overall_sentiment_score FLOAT           NOT NULL DEFAULT 0.0,

    -- Độ tin cậy kết quả sentiment: 0.0 → 1.0
    overall_confidence      FLOAT           NOT NULL DEFAULT 0.0,

    -- ═══════════════════════════════════════════════════════════════════════
    -- INTENT CLASSIFICATION RESULTS (Stage 2 — Regex Pattern Matching)
    -- ═══════════════════════════════════════════════════════════════════════

    -- Ý định chính: CRISIS, SEEDING, SPAM, COMPLAINT, LEAD, SUPPORT, DISCUSSION
    -- Priority-based conflict resolution khi nhiều patterns match
    primary_intent          VARCHAR(50)     NOT NULL DEFAULT 'DISCUSSION',

    -- Độ tin cậy phân loại intent: 0.0 → 1.0
    -- Công thức: base 0.5 + 0.1 × số patterns matched, capped 1.0
    intent_confidence       FLOAT           NOT NULL DEFAULT 0.0,

    -- ═══════════════════════════════════════════════════════════════════════
    -- IMPACT CALCULATION RESULTS (Stage 5)
    -- ═══════════════════════════════════════════════════════════════════════

    -- Điểm ảnh hưởng tổng hợp: 0 → 100
    -- Formula: (EngagementScore + ReachScore) × PlatformMultiplier × SentimentAmplifier
    -- Normalized: (RawImpact / 100000) × 100, capped [0, 100]
    impact_score            FLOAT           NOT NULL DEFAULT 0.0,

    -- Mức rủi ro dựa trên impact_score và sentiment:
    --   CRITICAL: impact ≥ 70 AND sentiment = NEGATIVE
    --   HIGH:     impact ≥ 70
    --   MEDIUM:   impact ≥ 40
    --   LOW:      còn lại
    risk_level              VARCHAR(50)     NOT NULL DEFAULT 'LOW',

    -- Flag bài viết viral: true khi impact_score ≥ 70.0
    is_viral                BOOLEAN         NOT NULL DEFAULT FALSE,

    -- Flag Key Opinion Leader: true khi author.followers ≥ 50,000
    is_kol                  BOOLEAN         NOT NULL DEFAULT FALSE,

    -- ═══════════════════════════════════════════════════════════════════════
    -- BREAKDOWNS (JSONB — kết quả chi tiết từ AI modules)
    -- ═══════════════════════════════════════════════════════════════════════

    -- Phân tích cảm xúc theo aspect (từ Stage 3 keywords + Stage 4 sentiment)
    -- Format: {"DESIGN": {"label": "POSITIVE", "score": 0.6, "confidence": 0.8,
    --          "mentions": 2, "keywords": ["thiết kế"], "rating": 4}, ...}
    -- Aspects: DESIGN, PERFORMANCE, PRICE, SERVICE, GENERAL
    aspects_breakdown       JSONB           NOT NULL DEFAULT '{}',

    -- Danh sách từ khóa đã trích xuất (Stage 3). Tối đa 30 items
    -- Kết hợp dictionary matching (DICT) và AI (YAKE + spaCy NER)
    keywords                TEXT[]          NOT NULL DEFAULT '{}',

    -- Phân phối xác suất 3 class sentiment
    -- Format: {"POSITIVE": 0.75, "NEGATIVE": 0.15, "NEUTRAL": 0.10}
    sentiment_probabilities JSONB           NOT NULL DEFAULT '{}',

    -- Chi tiết tính toán impact score
    -- Format: {"engagement_score": 12500.0, "reach_score": 16.4,
    --          "platform_multiplier": 1.0, "sentiment_amplifier": 1.1,
    --          "raw_impact": 13768.0}
    impact_breakdown        JSONB           NOT NULL DEFAULT '{}',

    -- ═══════════════════════════════════════════════════════════════════════
    -- RAW METRICS (passthrough từ input interaction + author)
    -- ═══════════════════════════════════════════════════════════════════════

    -- Lượt xem. Từ interaction.views. Impact weight: ×0.01
    view_count              INTEGER         NOT NULL DEFAULT 0,

    -- Lượt thích. Từ interaction.likes. Impact weight: ×1.0
    like_count              INTEGER         NOT NULL DEFAULT 0,

    -- Số bình luận. Từ interaction.comments_count. Impact weight: ×2.0
    comment_count           INTEGER         NOT NULL DEFAULT 0,

    -- Lượt chia sẻ. Từ interaction.shares. Impact weight: ×5.0 (cao nhất)
    share_count             INTEGER         NOT NULL DEFAULT 0,

    -- Lượt lưu/bookmark. Từ interaction.saves. Impact weight: ×3.0
    save_count              INTEGER         NOT NULL DEFAULT 0,

    -- Số followers tác giả. Từ author.followers. Dùng tính is_kol và reach_score
    follower_count          INTEGER         NOT NULL DEFAULT 0,

    -- ═══════════════════════════════════════════════════════════════════════
    -- PROCESSING METADATA
    -- ═══════════════════════════════════════════════════════════════════════

    -- Thời gian xử lý pipeline (ms). Đo từ process() start → end
    processing_time_ms      INTEGER         NOT NULL DEFAULT 0,

    -- Version analytics model/pipeline. Configurable, default "1.0.0"
    model_version           VARCHAR(50)     NOT NULL DEFAULT '1.0.0',

    -- Trạng thái xử lý:
    --   "success"  — hoàn thành đầy đủ pipeline
    --   "error"    — lỗi xảy ra, record vẫn được lưu với default values
    --   "skipped"  — bỏ qua AI stages do spam/seeding intent
    processing_status       VARCHAR(50)     NOT NULL DEFAULT 'success',

    -- ═══════════════════════════════════════════════════════════════════════
    -- CRAWLER METADATA (Contract v2.0 — passthrough từ EventMetadata)
    -- ═══════════════════════════════════════════════════════════════════════

    -- ID crawl job. Passthrough từ EventMetadata.job_id
    job_id                  VARCHAR(255)    NULL,

    -- Index batch trong job (0-based). Passthrough từ EventMetadata.batch_index
    batch_index             INTEGER         NULL,

    -- Loại task crawl: "keyword_search", "profile_crawl", ...
    task_type               VARCHAR(50)     NULL,

    -- Từ khóa nguồn. Lấy từ EventMetadata.keyword
    keyword_source          VARCHAR(255)    NULL,

    -- Thời điểm crawl. Parse từ envelope.timestamp (ISO 8601)
    crawled_at              TIMESTAMPTZ     NULL,

    -- Version pipeline. Auto-generated: "crawler_{platform}_v3"
    pipeline_version        VARCHAR(50)     NULL,

    -- Tên thương hiệu đang theo dõi
    brand_name              VARCHAR(255)    NULL,

    -- Từ khóa tìm kiếm đã dùng để crawl
    keyword                 VARCHAR(255)    NULL,

    -- ═══════════════════════════════════════════════════════════════════════
    -- CONTENT FIELDS (passthrough từ input content)
    -- ═══════════════════════════════════════════════════════════════════════

    -- Nội dung text gốc (caption/body). Từ content.text
    content_text            TEXT            NULL,

    -- Transcript audio/video gốc. Từ content.transcription
    content_transcription   TEXT            NULL,

    -- Thời lượng media (giây). Từ content.duration
    media_duration          INTEGER         NULL,

    -- Danh sách hashtags gốc. Từ content.hashtags
    hashtags                TEXT[]          NULL,

    -- URL gốc bài viết trên platform. Từ meta.permalink
    permalink               TEXT            NULL,

    -- ═══════════════════════════════════════════════════════════════════════
    -- AUTHOR FIELDS (passthrough từ input author)
    -- ═══════════════════════════════════════════════════════════════════════

    -- ID tác giả trên platform
    author_id               VARCHAR(255)    NULL,

    -- Tên hiển thị tác giả
    author_name             VARCHAR(255)    NULL,

    -- Username/handle (@username)
    author_username         VARCHAR(255)    NULL,

    -- URL ảnh đại diện
    author_avatar_url       TEXT            NULL,

    -- Tài khoản xác minh. Nếu true, reach_score ×1.2 trong impact calculation
    author_is_verified      BOOLEAN         NOT NULL DEFAULT FALSE
);

-- ═══════════════════════════════════════════════════════════════════════════
-- INDEXES
-- ═══════════════════════════════════════════════════════════════════════════

-- Query theo project
CREATE INDEX idx_post_analytics_project_id  ON schema_analyst.analyzed_posts (project_id);

-- Query theo crawl job
CREATE INDEX idx_post_analytics_job_id      ON schema_analyst.analyzed_posts (job_id);

-- Query theo thời gian phân tích (time-series, dashboard)
CREATE INDEX idx_post_analytics_analyzed_at ON schema_analyst.analyzed_posts (analyzed_at);

-- Filter theo platform
CREATE INDEX idx_post_analytics_platform    ON schema_analyst.analyzed_posts (platform);

-- Filter bài viết rủi ro cao (alert system)
CREATE INDEX idx_post_analytics_risk_level  ON schema_analyst.analyzed_posts (risk_level);
