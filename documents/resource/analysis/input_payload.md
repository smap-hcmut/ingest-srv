# Input Payload — SMAP Analytics Service

## Tổng quan

Service nhận message từ RabbitMQ queue `analytics.data.collected` (exchange: `smap.events`, routing key: `data.collected`). Message body là JSON UTF-8 theo format **Event Envelope** chứa metadata và payload dữ liệu bài viết.

---

## Cấu trúc Event Envelope

```jsonc
{
  // ─── Event-level metadata ───
  "event_id":    "string",       // UUID định danh event, dùng cho tracing/logging
  "event_type":  "string",       // Loại event, thường là "data.collected"
  "timestamp":   "string",       // ISO 8601 timestamp khi event được tạo (VD: "2025-01-15T10:30:00Z")

  // ─── Payload chính ───
  "payload": {
    // ─── Batch/Job context ───
    "project_id":    "string",   // UUID của project trong SMAP platform
    "job_id":        "string",   // UUID của crawl job (có thể chứa prefix project_id)
    "batch_index":   0,          // Thứ tự batch trong job (0-based)
    "content_count": 0,          // Số lượng bài viết trong batch
    "platform":      "string",   // Platform nguồn: "TIKTOK", "FACEBOOK", "YOUTUBE", "INSTAGRAM"
    "task_type":     "string",   // Loại task crawl: "keyword_search", "profile_crawl", ...
    "brand_name":    "string",   // Tên thương hiệu đang theo dõi
    "keyword":       "string",   // Từ khóa tìm kiếm đã dùng để crawl

    // ─── MinIO batch path (alternative) ───
    "minio_path":    "string",   // Path tới file batch trên MinIO (nếu dùng batch mode)

    // ─── Post data (inline mode) ───
    "meta": {
      "id":           "string",  // [REQUIRED] ID duy nhất của bài viết (từ platform)
      "platform":     "string",  // Platform nguồn (fallback nếu payload.platform không có)
      "permalink":    "string",  // URL gốc của bài viết trên platform
      "published_at": "string"   // ISO 8601 timestamp khi bài viết được đăng
    },

    "content": {
      "text":          "string", // Nội dung text/caption của bài viết
      "description":   "string", // Mô tả bài viết (fallback nếu text rỗng)
      "transcription": "string", // Transcript audio/video (nếu có)
      "hashtags":      ["string"], // Danh sách hashtags
      "duration":      0         // Thời lượng media tính bằng giây (video/audio)
    },

    "interaction": {
      "views":          0,       // Lượt xem
      "likes":          0,       // Lượt thích
      "comments_count": 0,       // Số lượng bình luận
      "shares":         0,       // Lượt chia sẻ
      "saves":          0        // Lượt lưu/bookmark
    },

    "author": {
      "id":          "string",   // ID tác giả trên platform
      "name":        "string",   // Tên hiển thị
      "username":    "string",   // Username/handle
      "avatar_url":  "string",   // URL ảnh đại diện
      "followers":   0,          // Số người theo dõi
      "is_verified": false       // Tài khoản đã xác minh
    },

    "comments": [                // Danh sách bình luận (optional)
      {
        "text":  "string",       // Nội dung bình luận
        "likes": 0               // Lượt thích bình luận
      }
    ]
  }
}
```

---

## Chi tiết từng field

### Event-level Fields

| Field | Type | Required | Mô tả |
|-------|------|----------|-------|
| `event_id` | string | Không | UUID định danh event. Dùng cho distributed tracing và log correlation. Default: `"unknown"` nếu thiếu. |
| `event_type` | string | Không | Loại event. Service hiện chỉ xử lý `"data.collected"`. |
| `timestamp` | string (ISO 8601) | Không | Thời điểm event được tạo bởi publisher. Được parse thành `crawled_at` trong pipeline. |

### Payload — Batch/Job Context

| Field | Type | Required | Mô tả |
|-------|------|----------|-------|
| `project_id` | string (UUID) | Không | ID project trên SMAP. Nếu không có, pipeline cố extract từ `job_id` (lấy 36 ký tự đầu nếu là UUID). |
| `job_id` | string | Không | ID của crawl job. Có thể có format `{project_uuid}-{suffix}`. |
| `batch_index` | integer | Không | Index của batch trong job, bắt đầu từ 0. Dùng để track tiến độ xử lý. |
| `content_count` | integer | Không | Tổng số bài viết trong batch. Metadata cho monitoring. |
| `platform` | string | Không | Platform nguồn dữ liệu. Giá trị hợp lệ: `TIKTOK`, `FACEBOOK`, `YOUTUBE`, `INSTAGRAM`. Được normalize thành uppercase. |
| `task_type` | string | Không | Loại task crawl đã tạo ra dữ liệu này (VD: `keyword_search`, `profile_crawl`). |
| `brand_name` | string | Không | Tên thương hiệu đang được theo dõi trong project. |
| `keyword` | string | Không | Từ khóa tìm kiếm đã dùng để crawl bài viết này. Cũng được dùng làm `keyword_source`. |
| `minio_path` | string | Không | Path tới file batch trên MinIO. Nếu có, service sẽ download và decompress (zstd) file thay vì dùng inline data. |

### Payload — Meta

| Field | Type | Required | Mô tả |
|-------|------|----------|-------|
| `meta.id` | string | **Có** | ID duy nhất của bài viết. Đây là field bắt buộc duy nhất — pipeline sẽ raise `ValueError` nếu thiếu. Dùng làm primary key trong DB. |
| `meta.platform` | string | Không | Platform nguồn. Fallback nếu `payload.platform` không có. |
| `meta.permalink` | string | Không | URL gốc bài viết trên platform (VD: `https://tiktok.com/@user/video/123`). |
| `meta.published_at` | string (ISO 8601) | Không | Thời điểm bài viết được đăng. Nếu thiếu, pipeline dùng `datetime.now(UTC)`. |

### Payload — Content

| Field | Type | Required | Mô tả |
|-------|------|----------|-------|
| `content.text` | string | Không | Nội dung chính (caption/body) của bài viết. Đây là input chính cho NLP pipeline. |
| `content.description` | string | Không | Mô tả bài viết. Dùng làm fallback nếu `text` rỗng. |
| `content.transcription` | string | Không | Transcript từ audio/video. Được nối với `text` thành `full_text` cho phân tích. |
| `content.hashtags` | string[] | Không | Danh sách hashtags. Được lưu nguyên vào DB. |
| `content.duration` | integer | Không | Thời lượng media (giây). Metadata cho video/audio posts. |

### Payload — Interaction

| Field | Type | Required | Mô tả |
|-------|------|----------|-------|
| `interaction.views` | integer | Không | Lượt xem. Weight trong impact: ×0.01. Default: 0. |
| `interaction.likes` | integer | Không | Lượt thích. Weight trong impact: ×1.0. Default: 0. |
| `interaction.comments_count` | integer | Không | Số bình luận. Weight trong impact: ×2.0. Default: 0. |
| `interaction.shares` | integer | Không | Lượt chia sẻ. Weight trong impact: ×5.0 (cao nhất). Default: 0. |
| `interaction.saves` | integer | Không | Lượt lưu/bookmark. Weight trong impact: ×3.0. Default: 0. |

### Payload — Author

| Field | Type | Required | Mô tả |
|-------|------|----------|-------|
| `author.id` | string | Không | ID tác giả trên platform. Lưu vào `author_id` trong DB. |
| `author.name` | string | Không | Tên hiển thị của tác giả. |
| `author.username` | string | Không | Username/handle (VD: `@username`). |
| `author.avatar_url` | string | Không | URL ảnh đại diện. |
| `author.followers` | integer | Không | Số người theo dõi. Dùng để tính `is_kol` (≥50,000) và `reach_score` trong impact. Default: 0. |
| `author.is_verified` | boolean | Không | Tài khoản đã xác minh. Nếu `true`, reach_score được nhân ×1.2. Default: `false`. |

### Payload — Comments

| Field | Type | Required | Mô tả |
|-------|------|----------|-------|
| `comments` | array | Không | Danh sách bình luận. Preprocessing lấy tối đa 5 comments (sắp xếp theo likes giảm dần). |
| `comments[].text` | string | Không | Nội dung bình luận. Được gộp vào `full_text` cho phân tích. |
| `comments[].likes` | integer | Không | Lượt thích bình luận. Dùng để sắp xếp ưu tiên. Default: 0. |

---

## Validation Rules

1. `payload` phải tồn tại trong envelope.
2. Payload phải có ít nhất một trong hai: `minio_path` (batch mode) hoặc `meta` (inline mode).
3. `meta.id` là field bắt buộc duy nhất — thiếu sẽ raise `ValueError("post_data.meta.id is required")`.
4. Tất cả field khác đều optional với default values hợp lý.

---

## Ví dụ payload đầy đủ

```json
{
  "event_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "event_type": "data.collected",
  "timestamp": "2025-01-15T10:30:00Z",
  "payload": {
    "project_id": "proj-uuid-1234-5678-abcdef",
    "job_id": "proj-uuid-1234-5678-abcdef-job-001",
    "batch_index": 0,
    "content_count": 50,
    "platform": "TIKTOK",
    "task_type": "keyword_search",
    "brand_name": "VinFast",
    "keyword": "VinFast VF8",
    "meta": {
      "id": "7312345678901234567",
      "platform": "TIKTOK",
      "permalink": "https://www.tiktok.com/@user123/video/7312345678901234567",
      "published_at": "2025-01-14T08:00:00Z"
    },
    "content": {
      "text": "Review VinFast VF8 sau 1 năm sử dụng, pin vẫn rất tốt #vinfast #vf8",
      "transcription": "Xin chào mọi người, hôm nay mình sẽ review chiếc VinFast VF8...",
      "hashtags": ["vinfast", "vf8", "review"],
      "duration": 180
    },
    "interaction": {
      "views": 150000,
      "likes": 8500,
      "comments_count": 320,
      "shares": 450,
      "saves": 1200
    },
    "author": {
      "id": "user123",
      "name": "Reviewer Auto",
      "username": "reviewer_auto",
      "avatar_url": "https://p16-sign.tiktokcdn.com/avatar/user123.jpeg",
      "followers": 85000,
      "is_verified": true
    },
    "comments": [
      { "text": "VF8 chạy êm quá, mình cũng muốn mua", "likes": 45 },
      { "text": "Giá bao nhiêu vậy bạn?", "likes": 23 },
      { "text": "Pin có bền không sau 1 năm?", "likes": 18 }
    ]
  }
}
```

## Ví dụ payload tối thiểu

```json
{
  "payload": {
    "meta": {
      "id": "post-unique-id-123"
    }
  }
}
```
