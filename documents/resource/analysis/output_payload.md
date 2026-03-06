# Output Payload — SMAP Analytics Service

## Tổng quan

Sau khi xử lý qua pipeline 5 giai đoạn, kết quả được persist vào table `schema_analyst.analyzed_posts` trong PostgreSQL. Mỗi record đại diện cho một bài viết đã được phân tích đầy đủ, bao gồm cả dữ liệu gốc từ crawler và kết quả AI.

---

## Database Target

- Schema: `schema_analyst`
- Table: `analyzed_posts`
- Primary Key: `id` (string, từ `meta.id` của input)
- ORM: SQLAlchemy 2.x async

---

## Cấu trúc Output Record

### Identifiers

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `id` | VARCHAR(255) | Không | — | Primary key. Lấy từ `meta.id` của input payload. ID bài viết trên platform gốc. |
| `project_id` | VARCHAR(255) | Có | NULL | UUID project trên SMAP. Lấy từ `payload.project_id` hoặc extract từ `job_id`. Pipeline sanitize để đảm bảo UUID hợp lệ. |
| `platform` | VARCHAR(50) | Không | `"UNKNOWN"` | Platform nguồn, normalized thành uppercase: `TIKTOK`, `FACEBOOK`, `YOUTUBE`, `INSTAGRAM`, `UNKNOWN`. |

### Timestamps

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `published_at` | TIMESTAMPTZ | Có | NULL | Thời điểm bài viết được đăng trên platform. Lấy từ `meta.published_at`, fallback `now() UTC`. |
| `analyzed_at` | TIMESTAMPTZ | Có | NULL | Thời điểm pipeline hoàn thành phân tích. Luôn là `now() UTC` tại thời điểm xử lý. |

### Sentiment Analysis Results (Stage 4)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `overall_sentiment` | VARCHAR(50) | Không | `"NEUTRAL"` | Label cảm xúc tổng thể: `POSITIVE`, `NEGATIVE`, `NEUTRAL`. Phân loại dựa trên score với thresholds: positive ≥ 0.25, negative ≤ -0.25. |
| `overall_sentiment_score` | FLOAT | Không | `0.0` | Điểm cảm xúc từ -1.0 (rất tiêu cực) đến 1.0 (rất tích cực). Output trực tiếp từ PhoBERT model. |
| `overall_confidence` | FLOAT | Không | `0.0` | Độ tin cậy của kết quả sentiment (0.0 → 1.0). Phản ánh mức chắc chắn của model. |
| `sentiment_probabilities` | JSONB | Không | `{}` | Phân phối xác suất cho 3 class sentiment. Format: `{"POSITIVE": 0.75, "NEGATIVE": 0.15, "NEUTRAL": 0.10}`. |

### Intent Classification Results (Stage 2)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `primary_intent` | VARCHAR(50) | Không | `"DISCUSSION"` | Ý định chính của bài viết. Giá trị: `CRISIS`, `SEEDING`, `SPAM`, `COMPLAINT`, `LEAD`, `SUPPORT`, `DISCUSSION`. Xác định bằng regex pattern matching với priority-based conflict resolution. |
| `intent_confidence` | FLOAT | Không | `0.0` | Độ tin cậy phân loại intent (0.0 → 1.0). Tính bằng: base 0.5 + 0.1 × số patterns matched, capped tại 1.0. |

### Impact Calculation Results (Stage 5)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `impact_score` | FLOAT | Không | `0.0` | Điểm ảnh hưởng tổng hợp (0 → 100). Tính từ engagement, reach, platform multiplier, và sentiment amplifier. |
| `risk_level` | VARCHAR(50) | Không | `"LOW"` | Mức độ rủi ro: `CRITICAL` (impact≥70 + negative), `HIGH` (impact≥70), `MEDIUM` (impact≥40), `LOW`. |
| `is_viral` | BOOLEAN | Không | `false` | Flag bài viết viral. `true` khi `impact_score ≥ 70.0`. |
| `is_kol` | BOOLEAN | Không | `false` | Flag Key Opinion Leader. `true` khi `author.followers ≥ 50,000`. |
| `impact_breakdown` | JSONB | Không | `{}` | Chi tiết tính toán impact. Format: `{"engagement_score": 12500.0, "reach_score": 16.4, "platform_multiplier": 1.0, "sentiment_amplifier": 1.1, "raw_impact": 13768.0}`. |

### Keyword Extraction Results (Stage 3)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `keywords` | TEXT[] | Không | `[]` | Danh sách từ khóa đã trích xuất. Tối đa 30 items. Kết hợp từ dictionary matching và AI (YAKE + spaCy NER). |
| `aspects_breakdown` | JSONB | Không | `{}` | Phân tích cảm xúc theo aspect. Format: `{"DESIGN": {"label": "POSITIVE", "score": 0.6, "confidence": 0.8, "mentions": 2, "keywords": ["thiết kế", "ngoại thất"], "rating": 4}, ...}`. Aspects: `DESIGN`, `PERFORMANCE`, `PRICE`, `SERVICE`, `GENERAL`. |

### Raw Metrics (passthrough từ input)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `view_count` | INTEGER | Không | `0` | Lượt xem. Lấy từ `interaction.views`. |
| `like_count` | INTEGER | Không | `0` | Lượt thích. Lấy từ `interaction.likes`. |
| `comment_count` | INTEGER | Không | `0` | Số bình luận. Lấy từ `interaction.comments_count`. |
| `share_count` | INTEGER | Không | `0` | Lượt chia sẻ. Lấy từ `interaction.shares`. |
| `save_count` | INTEGER | Không | `0` | Lượt lưu. Lấy từ `interaction.saves`. |
| `follower_count` | INTEGER | Không | `0` | Số followers tác giả. Lấy từ `author.followers`. |

### Processing Metadata

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `processing_time_ms` | INTEGER | Không | `0` | Thời gian xử lý pipeline (milliseconds). Đo từ lúc bắt đầu `process()` đến khi hoàn thành. |
| `model_version` | VARCHAR(50) | Không | `"1.0.0"` | Version của analytics model/pipeline. Configurable. |
| `processing_status` | VARCHAR(50) | Không | `"success"` | Trạng thái xử lý: `success` (hoàn thành), `error` (lỗi, record vẫn được lưu), `skipped` (bỏ qua do spam/seeding). |

### Crawler Metadata (Contract v2.0)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `job_id` | VARCHAR(255) | Có | NULL | ID crawl job. Passthrough từ `EventMetadata.job_id`. Indexed cho query. |
| `batch_index` | INTEGER | Có | NULL | Index batch trong job. Passthrough từ `EventMetadata.batch_index`. |
| `task_type` | VARCHAR(50) | Có | NULL | Loại task crawl. Passthrough từ `EventMetadata.task_type`. |
| `keyword_source` | VARCHAR(255) | Có | NULL | Từ khóa nguồn. Lấy từ `EventMetadata.keyword`. |
| `crawled_at` | TIMESTAMPTZ | Có | NULL | Thời điểm crawl. Parse từ `envelope.timestamp` (ISO 8601). |
| `pipeline_version` | VARCHAR(50) | Có | NULL | Version pipeline. Auto-generated: `"crawler_{platform}_v3"`. |
| `brand_name` | VARCHAR(255) | Có | NULL | Tên thương hiệu. Passthrough từ `EventMetadata.brand_name`. |
| `keyword` | VARCHAR(255) | Có | NULL | Từ khóa tìm kiếm. Passthrough từ `EventMetadata.keyword`. |

### Content Fields (passthrough từ input)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `content_text` | TEXT | Có | NULL | Nội dung text gốc. Lấy từ `content.text`. |
| `content_transcription` | TEXT | Có | NULL | Transcript audio/video gốc. Lấy từ `content.transcription`. |
| `media_duration` | INTEGER | Có | NULL | Thời lượng media (giây). Lấy từ `content.duration`. |
| `hashtags` | TEXT[] | Có | NULL | Danh sách hashtags gốc. Lấy từ `content.hashtags`. |
| `permalink` | TEXT | Có | NULL | URL gốc bài viết. Lấy từ `meta.permalink`. |

### Author Fields (passthrough từ input)

| Column | Type | Nullable | Default | Mô tả |
|--------|------|----------|---------|-------|
| `author_id` | VARCHAR(255) | Có | NULL | ID tác giả. Lấy từ `author.id`. |
| `author_name` | VARCHAR(255) | Có | NULL | Tên hiển thị. Lấy từ `author.name`. |
| `author_username` | VARCHAR(255) | Có | NULL | Username/handle. Lấy từ `author.username`. |
| `author_avatar_url` | TEXT | Có | NULL | URL ảnh đại diện. Lấy từ `author.avatar_url`. |
| `author_is_verified` | BOOLEAN | Không | `false` | Tài khoản xác minh. Lấy từ `author.is_verified`. |

---

## Database Indexes

| Index Name | Column(s) | Mục đích |
|------------|-----------|----------|
| `idx_post_analytics_project_id` | `project_id` | Query theo project |
| `idx_post_analytics_job_id` | `job_id` | Query theo crawl job |
| `idx_post_analytics_analyzed_at` | `analyzed_at` | Query theo thời gian phân tích |
| `idx_post_analytics_platform` | `platform` | Filter theo platform |
| `idx_post_analytics_risk_level` | `risk_level` | Filter bài viết rủi ro cao |

---

## Ví dụ output record (JSON representation)

```json
{
  "id": "7312345678901234567",
  "project_id": "proj-uuid-1234-5678-abcdef",
  "platform": "TIKTOK",

  "published_at": "2025-01-14T08:00:00+00:00",
  "analyzed_at": "2025-01-15T10:30:05+00:00",

  "overall_sentiment": "POSITIVE",
  "overall_sentiment_score": 0.72,
  "overall_confidence": 0.89,
  "sentiment_probabilities": {
    "POSITIVE": 0.89,
    "NEGATIVE": 0.04,
    "NEUTRAL": 0.07
  },

  "primary_intent": "DISCUSSION",
  "intent_confidence": 0.6,

  "impact_score": 78.5,
  "risk_level": "HIGH",
  "is_viral": true,
  "is_kol": true,
  "impact_breakdown": {
    "engagement_score": 12890.0,
    "reach_score": 16.4,
    "platform_multiplier": 1.0,
    "sentiment_amplifier": 1.1,
    "raw_impact": 14196.0
  },

  "keywords": ["VinFast", "VF8", "pin", "review", "thiết kế"],
  "aspects_breakdown": {
    "PERFORMANCE": {
      "label": "POSITIVE",
      "score": 0.65,
      "confidence": 0.82,
      "mentions": 2,
      "keywords": ["pin", "chạy êm"],
      "rating": 4
    },
    "DESIGN": {
      "label": "POSITIVE",
      "score": 0.58,
      "confidence": 0.75,
      "mentions": 1,
      "keywords": ["thiết kế"],
      "rating": 4
    }
  },

  "view_count": 150000,
  "like_count": 8500,
  "comment_count": 320,
  "share_count": 450,
  "save_count": 1200,
  "follower_count": 85000,

  "processing_time_ms": 245,
  "model_version": "1.0.0",
  "processing_status": "success",

  "job_id": "proj-uuid-1234-5678-abcdef-job-001",
  "batch_index": 0,
  "task_type": "keyword_search",
  "keyword_source": "VinFast VF8",
  "crawled_at": "2025-01-15T10:30:00+00:00",
  "pipeline_version": "crawler_tiktok_v3",
  "brand_name": "VinFast",
  "keyword": "VinFast VF8",

  "content_text": "Review VinFast VF8 sau 1 năm sử dụng, pin vẫn rất tốt #vinfast #vf8",
  "content_transcription": "Xin chào mọi người, hôm nay mình sẽ review chiếc VinFast VF8...",
  "media_duration": 180,
  "hashtags": ["vinfast", "vf8", "review"],
  "permalink": "https://www.tiktok.com/@user123/video/7312345678901234567",

  "author_id": "user123",
  "author_name": "Reviewer Auto",
  "author_username": "reviewer_auto",
  "author_avatar_url": "https://p16-sign.tiktokcdn.com/avatar/user123.jpeg",
  "author_is_verified": true
}
```

## Ví dụ output record khi lỗi

```json
{
  "id": "post-error-example",
  "project_id": null,
  "platform": "UNKNOWN",
  "analyzed_at": "2025-01-15T10:30:05+00:00",
  "overall_sentiment": "NEUTRAL",
  "overall_sentiment_score": 0.0,
  "overall_confidence": 0.0,
  "primary_intent": "DISCUSSION",
  "intent_confidence": 0.0,
  "impact_score": 0.0,
  "risk_level": "LOW",
  "is_viral": false,
  "is_kol": false,
  "processing_status": "error",
  "processing_time_ms": 0,
  "model_version": "1.0.0"
}
```
