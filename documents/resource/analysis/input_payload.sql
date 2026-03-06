-- ============================================================================
-- SMAP Analytics Service — Input Payload Schema
-- ============================================================================
-- Mô tả cấu trúc JSON message mà service nhận từ RabbitMQ.
-- Queue: analytics.data.collected
-- Exchange: smap.events
-- Routing Key: data.collected
--
-- Đây KHÔNG phải table thực tế trong DB, mà là mô tả cấu trúc input
-- dưới dạng SQL-like schema để dễ tham chiếu.
-- ============================================================================

-- ─── Event Envelope (root level) ───────────────────────────────────────────
-- JSON message body gửi qua RabbitMQ
CREATE TYPE input_event_envelope AS (
    -- UUID định danh event, dùng cho tracing/logging. Default: "unknown"
    event_id        TEXT,

    -- Loại event. Service chỉ xử lý "data.collected"
    event_type      TEXT,

    -- ISO 8601 timestamp khi event được tạo. Parse thành crawled_at trong pipeline
    timestamp       TEXT,

    -- Payload chính chứa dữ liệu bài viết và metadata
    payload         input_payload
);

-- ─── Payload ───────────────────────────────────────────────────────────────
CREATE TYPE input_payload AS (
    -- ─── Batch/Job Context ───
    -- UUID project trên SMAP platform. Nếu thiếu, extract từ job_id
    project_id      TEXT,

    -- ID crawl job. Format có thể là "{project_uuid}-{suffix}"
    job_id          TEXT,

    -- Index batch trong job (0-based). Track tiến độ xử lý
    batch_index     INTEGER,

    -- Tổng số bài viết trong batch. Metadata cho monitoring
    content_count   INTEGER,

    -- Platform nguồn: "TIKTOK", "FACEBOOK", "YOUTUBE", "INSTAGRAM"
    platform        TEXT,

    -- Loại task crawl: "keyword_search", "profile_crawl", ...
    task_type       TEXT,

    -- Tên thương hiệu đang theo dõi
    brand_name      TEXT,

    -- Từ khóa tìm kiếm đã dùng để crawl. Cũng dùng làm keyword_source
    keyword         TEXT,

    -- Path tới file batch trên MinIO (alternative cho inline mode)
    minio_path      TEXT,

    -- ─── Inline Post Data ───
    -- Metadata bài viết [REQUIRED: meta.id]
    meta            input_meta,

    -- Nội dung bài viết
    content         input_content,

    -- Số liệu tương tác
    interaction     input_interaction,

    -- Thông tin tác giả
    author          input_author,

    -- Danh sách bình luận
    comments        input_comment[]
);

-- ─── Meta ──────────────────────────────────────────────────────────────────
CREATE TYPE input_meta AS (
    -- [REQUIRED] ID duy nhất bài viết từ platform. Dùng làm PK trong DB
    id              TEXT,

    -- Platform nguồn (fallback nếu payload.platform không có)
    platform        TEXT,

    -- URL gốc bài viết trên platform
    permalink       TEXT,

    -- ISO 8601 timestamp khi bài viết được đăng. Default: now() UTC
    published_at    TEXT
);

-- ─── Content ───────────────────────────────────────────────────────────────
CREATE TYPE input_content AS (
    -- Nội dung chính (caption/body). Input chính cho NLP pipeline
    text            TEXT,

    -- Mô tả bài viết. Fallback nếu text rỗng
    description     TEXT,

    -- Transcript audio/video. Nối với text thành full_text cho phân tích
    transcription   TEXT,

    -- Danh sách hashtags
    hashtags        TEXT[],

    -- Thời lượng media (giây) cho video/audio
    duration        INTEGER
);

-- ─── Interaction ───────────────────────────────────────────────────────────
CREATE TYPE input_interaction AS (
    -- Lượt xem. Impact weight: ×0.01. Default: 0
    views           INTEGER,

    -- Lượt thích. Impact weight: ×1.0. Default: 0
    likes           INTEGER,

    -- Số bình luận. Impact weight: ×2.0. Default: 0
    comments_count  INTEGER,

    -- Lượt chia sẻ. Impact weight: ×5.0 (cao nhất). Default: 0
    shares          INTEGER,

    -- Lượt lưu/bookmark. Impact weight: ×3.0. Default: 0
    saves           INTEGER
);

-- ─── Author ────────────────────────────────────────────────────────────────
CREATE TYPE input_author AS (
    -- ID tác giả trên platform
    id              TEXT,

    -- Tên hiển thị
    name            TEXT,

    -- Username/handle (@username)
    username        TEXT,

    -- URL ảnh đại diện
    avatar_url      TEXT,

    -- Số người theo dõi. Dùng tính is_kol (≥50,000) và reach_score. Default: 0
    followers       INTEGER,

    -- Tài khoản xác minh. Nếu true, reach_score ×1.2. Default: false
    is_verified     BOOLEAN
);

-- ─── Comment ───────────────────────────────────────────────────────────────
CREATE TYPE input_comment AS (
    -- Nội dung bình luận. Gộp vào full_text cho phân tích
    text            TEXT,

    -- Lượt thích bình luận. Dùng sắp xếp ưu tiên. Default: 0
    likes           INTEGER
);

-- ============================================================================
-- VALIDATION RULES
-- ============================================================================
-- 1. envelope.payload          MUST exist
-- 2. payload.minio_path OR payload.meta  MUST exist (ít nhất 1)
-- 3. payload.meta.id           MUST exist (field bắt buộc duy nhất)
-- 4. Tất cả field khác         OPTIONAL với default values hợp lý
-- ============================================================================
