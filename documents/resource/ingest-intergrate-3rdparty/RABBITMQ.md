# RabbitMQ Contract Mirror (ingest-srv ↔ scapper-srv)

**Status:** Derived (Mirror)  
**Canonical source:** `/mnt/f/SMAP_v2/scapper-srv/RABBITMQ.md`  
**Ngày cập nhật:** 05/03/2026

> Tài liệu này là bản mirror để team ingest theo dõi nhanh. Khi có khác biệt, ưu tiên canonical tại `scapper-srv/RABBITMQ.md`.

---

## 1. Queues canonical

| Queue | Platform | Actions |
|---|---|---|
| `tiktok_tasks` | TikTok | `search`, `post_detail`, `comments`, `summary`, `comment_replies`, `cookie_check`, `full_flow` |
| `facebook_tasks` | Facebook | `search`, `posts`, `post_detail`, `comments`, `comments_graphql`, `comments_graphql_batch` |
| `youtube_tasks` | YouTube | `search`, `videos`, `video_detail`, `transcript`, `comments` |

Durable queue + persistent delivery.

---

## 2. Request payload (ingest -> scapper)

```json
{
  "task_id": "uuid-v4",
  "action": "string",
  "params": {},
  "created_at": "ISO-8601"
}
```

- `task_id` là correlation key bắt buộc.
- `action` phải thuộc action list theo queue platform.
- `params` theo định nghĩa của action tương ứng.

---

## 3. Response envelope chuẩn (scapper -> ingest)

```json
{
  "task_id": "uuid-v4",
  "queue": "tiktok_tasks|facebook_tasks|youtube_tasks",
  "platform": "tiktok|facebook|youtube",
  "action": "string",
  "status": "success|error",
  "result": {},
  "error": null,
  "completed_at": "ISO-8601",
  "metadata": {
    "crawler_version": "string",
    "duration_ms": 0
  }
}
```

Mục tiêu của envelope:
- Tương quan 1-1 với `external_tasks.task_id`.
- Chuẩn hóa đầu vào để ingest tạo `raw_batches` và parse pipeline.

---

## 4. Idempotency / Retry / DLQ

### 4.1 Idempotency

- Key chính: `task_id`.
- Response trùng `task_id` phải được xử lý idempotent:
  - không tạo `external_task` mới,
  - không tạo `raw_batch` mới cho cùng dữ liệu.

### 4.2 Retry

- Retry tại consumer ingest theo policy runtime.
- Lỗi tạm thời (network/decode/transient) được retry theo backoff.

### 4.3 DLQ

- Message quá ngưỡng retry phải đưa vào DLQ (khi hạ tầng DLQ sẵn sàng).
- Cần log đầy đủ `task_id`, `queue`, `action`, `error`, `attempt_count`.

---

## 5. Mapping deprecation

| Deprecated | Canonical |
|---|---|
| Ingest-only TikTok action list | Multi-platform queue/action tại `scapper-srv/RABBITMQ.md` |
| Contract nội bộ không có response envelope chuẩn | Dùng response envelope ở mục 3 |
