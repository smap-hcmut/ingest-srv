# Collector ↔ Crawler Contract Specification

## Tổng quan

Document này định nghĩa contract giữa Collector (Dispatcher) và Crawler services sau khi fix các vấn đề trong `collector-limit-config-analysis.md`.

**Mục đích:** Đảm bảo 2 services đồng bộ về format message, behavior, và expectations.

---

## 1. Message Flow Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         OUTBOUND: Collector → Crawler                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐         ┌─────────────┐         ┌──────────────┐          │
│  │   Collector  │ ──────► │  RabbitMQ   │ ──────► │   Crawler    │          │
│  │  (Dispatcher)│         │   Queues    │         │   (Worker)   │          │
│  └──────────────┘         └─────────────┘         └──────────────┘          │
│                                                                              │
│  Message: CollectorTask (TikTokCollectorTask / YouTubeCollectorTask)         │
│  Queue: tiktok.crawl / youtube.crawl                                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                         INBOUND: Crawler → Collector                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐         ┌─────────────┐         ┌──────────────┐          │
│  │   Crawler    │ ──────► │  RabbitMQ   │ ──────► │  Collector   │          │
│  │   (Worker)   │         │   Queues    │         │   (Results)  │          │
│  └──────────────┘         └─────────────┘         └──────────────┘          │
│                                                                              │
│  Message: CrawlerResult (Enhanced format)                                    │
│  Queue: collector.results                                                    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. OUTBOUND: Collector → Crawler

### 2.1 Message Format (CollectorTask)

**Base Task Structure:**

```json
{
  "job_id": "proj123-brand-0",
  "platform": "tiktok",
  "task_type": "research_and_crawl",
  "time_range": 30,
  "attempt": 1,
  "max_attempts": 3,
  "retry": false,
  "schema_version": 1,
  "trace_id": "trace-abc123",
  "routing_key": "tiktok.crawl",
  "emitted_at": "2024-01-15T10:30:00Z",
  "headers": {
    "x-schema-version": 1
  },
  "payload": { ... }
}
```

### 2.2 Payload by Task Type

#### 2.2.1 `research_and_crawl` (Production)

```json
{
  "payload": {
    "keywords": ["keyword1"],
    "limit_per_keyword": 50,
    "include_comments": true,
    "max_comments": 100,
    "download_media": false,
    "time_range": 30,
    "project_id": "proj123",
    "user_id": "user456",
    "keyword_source": "brand",
    "brand_name": "BrandX"
  }
}
```

#### 2.2.2 `dryrun_keyword` (Testing/Preview)

```json
{
  "payload": {
    "keywords": ["keyword1"],
    "limit_per_keyword": 3,
    "include_comments": true,
    "max_comments": 5,
    "download_media": false,
    "time_range": 30
  }
}
```

### 2.3 Field Definitions & Expectations

| Field               | Type     | Required | Description                   | Crawler Expectation               |
| ------------------- | -------- | -------- | ----------------------------- | --------------------------------- |
| `job_id`            | string   | ✅       | Unique identifier cho task    | Trả về trong response để tracking |
| `platform`          | string   | ✅       | "tiktok" hoặc "youtube"       | Xác định crawler nào xử lý        |
| `task_type`         | string   | ✅       | Loại task                     | Quyết định logic xử lý            |
| `keywords`          | string[] | ✅       | **Luôn chỉ 1 keyword**        | Search với keyword này            |
| `limit_per_keyword` | int      | ✅       | Số video tối đa cần crawl     | **PHẢI respect limit này**        |
| `include_comments`  | bool     | ✅       | Có crawl comments không       | Nếu true, crawl comments          |
| `max_comments`      | int      | ✅       | Số comments tối đa/video      | **PHẢI respect limit này**        |
| `download_media`    | bool     | ✅       | Có download video/audio không | Nếu true, download media          |
| `time_range`        | int      | ✅       | Số ngày filter (từ now)       | Filter videos trong range         |
| `max_attempts`      | int      | ✅       | Số lần retry tối đa           | Collector sẽ retry nếu fail       |
| `attempt`           | int      | ✅       | Lần thử hiện tại              | Để tracking retry count           |

### 2.4 Collector Guarantees (Outbound)

| #   | Guarantee                 | Description                                            |
| --- | ------------------------- | ------------------------------------------------------ |
| 1   | **1 keyword per message** | Mỗi message chỉ chứa đúng 1 keyword trong array        |
| 2   | **Limits từ config**      | Tất cả limit values được đọc từ config, không hardcode |
| 3   | **Unique job_id**         | Format: `{projectID}-{source}-{index}`                 |
| 4   | **Valid time_range**      | Luôn > 0, default 30 nếu không parse được              |
| 5   | **Idempotent dispatch**   | Cùng event sẽ tạo cùng job_ids                         |

### 2.5 Config Values (Collector sẽ gửi)

**Production (`research_and_crawl`):**

| Config Key                  | Default | Description       |
| --------------------------- | ------- | ----------------- |
| `DEFAULT_LIMIT_PER_KEYWORD` | 50      | Số video/keyword  |
| `DEFAULT_MAX_COMMENTS`      | 100     | Số comments/video |
| `DEFAULT_MAX_ATTEMPTS`      | 3       | Số lần retry      |
| `INCLUDE_COMMENTS`          | true    | Crawl comments    |
| `DOWNLOAD_MEDIA`            | false   | Download media    |

**Dry-run (`dryrun_keyword`):**

| Config Key                 | Default | Description                    |
| -------------------------- | ------- | ------------------------------ |
| `DRYRUN_LIMIT_PER_KEYWORD` | 3       | Số video/keyword (nhỏ để test) |
| `DRYRUN_MAX_COMMENTS`      | 5       | Số comments/video              |

---

## 3. INBOUND: Crawler → Collector

### 3.1 Current Response Format (Existing)

```json
{
  "success": true,
  "payload": [
    {
      "meta": {
        "id": "video123",
        "platform": "tiktok",
        "job_id": "proj123-brand-0",
        "task_type": "research_and_crawl",
        "crawled_at": "2024-01-15T10:35:00Z",
        "published_at": "2024-01-10T08:00:00Z",
        "permalink": "https://tiktok.com/@user/video/123",
        "keyword_source": "brand",
        "lang": "vi",
        "region": "VN",
        "pipeline_version": "1.0.0",
        "fetch_status": "success",
        "fetch_error": null
      },
      "content": { ... },
      "interaction": { ... },
      "author": { ... },
      "comments": [ ... ]
    }
  ]
}
```

### 3.2 Enhanced Response Format (Required)

**Collector cần thêm các fields sau để tracking chính xác:**

```json
{
  "success": true,
  "task_type": "research_and_crawl",

  "limit_info": {
    "requested_limit": 50,
    "applied_limit": 50,
    "total_found": 30,
    "platform_limited": true
  },

  "stats": {
    "successful": 28,
    "failed": 2,
    "skipped": 0,
    "completion_rate": 0.93
  },

  "payload": [ ... ]
}
```

### 3.3 New Fields Definition

| Field                         | Type  | Required | Description                            |
| ----------------------------- | ----- | -------- | -------------------------------------- |
| `limit_info.requested_limit`  | int   | ✅       | Limit được request từ Collector        |
| `limit_info.applied_limit`    | int   | ✅       | Limit thực tế áp dụng (sau khi cap)    |
| `limit_info.total_found`      | int   | ✅       | Số videos tìm được từ search           |
| `limit_info.platform_limited` | bool  | ✅       | `true` nếu platform trả về < requested |
| `stats.successful`            | int   | ✅       | Số videos crawl thành công             |
| `stats.failed`                | int   | ✅       | Số videos crawl thất bại               |
| `stats.skipped`               | int   | ✅       | Số videos bị skip (duplicate, etc.)    |
| `stats.completion_rate`       | float | ✅       | `successful / total_found`             |

### 3.4 Crawler Expectations (Response)

| #   | Expectation                 | Description                                                                |
| --- | --------------------------- | -------------------------------------------------------------------------- |
| 1   | **Trả về job_id**           | Phải include job_id trong `payload[].meta.job_id`                          |
| 2   | **Trả về task_type**        | Phải include task_type trong response root hoặc `payload[].meta.task_type` |
| 3   | **Trả về limit_info**       | **MỚI** - Phải include thông tin về limits                                 |
| 4   | **Trả về stats**            | **MỚI** - Phải include statistics                                          |
| 5   | **Respect limits**          | Không crawl nhiều hơn `limit_per_keyword`                                  |
| 6   | **Empty payload = success** | Nếu search không có kết quả, vẫn trả `success: true` với `payload: []`     |

### 3.5 Response Scenarios

#### Scenario 1: Full Success (tìm đủ limit)

```json
{
  "success": true,
  "task_type": "research_and_crawl",
  "limit_info": {
    "requested_limit": 50,
    "applied_limit": 50,
    "total_found": 50,
    "platform_limited": false
  },
  "stats": {
    "successful": 50,
    "failed": 0,
    "skipped": 0,
    "completion_rate": 1.0
  },
  "payload": [
    /* 50 items */
  ]
}
```

#### Scenario 2: Platform Limited (tìm ít hơn limit)

```json
{
  "success": true,
  "task_type": "research_and_crawl",
  "limit_info": {
    "requested_limit": 50,
    "applied_limit": 50,
    "total_found": 30,
    "platform_limited": true
  },
  "stats": {
    "successful": 28,
    "failed": 2,
    "skipped": 0,
    "completion_rate": 0.93
  },
  "payload": [
    /* 28 items */
  ]
}
```

#### Scenario 3: No Results

```json
{
  "success": true,
  "task_type": "research_and_crawl",
  "limit_info": {
    "requested_limit": 50,
    "applied_limit": 50,
    "total_found": 0,
    "platform_limited": true
  },
  "stats": {
    "successful": 0,
    "failed": 0,
    "skipped": 0,
    "completion_rate": 0
  },
  "payload": []
}
```

#### Scenario 4: Task Failed

```json
{
  "success": false,
  "task_type": "research_and_crawl",
  "error": {
    "code": "SEARCH_FAILED",
    "message": "TikTok API rate limited"
  },
  "limit_info": {
    "requested_limit": 50,
    "applied_limit": 50,
    "total_found": 0,
    "platform_limited": false
  },
  "stats": {
    "successful": 0,
    "failed": 0,
    "skipped": 0,
    "completion_rate": 0
  },
  "payload": []
}
```

---

## 4. State Tracking Behavior

### 4.1 Collector State Updates

**Khi nhận response từ Crawler:**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  COLLECTOR STATE UPDATE LOGIC                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. Nhận CrawlerResult                                                       │
│                                                                              │
│  2. Extract stats:                                                           │
│     - successful = response.stats.successful                                 │
│     - failed = response.stats.failed                                         │
│     - platform_limited = response.limit_info.platform_limited                │
│                                                                              │
│  3. Update Redis state:                                                      │
│     - tasks_done += 1  (mỗi response = 1 task hoàn thành)                    │
│     - items_actual += successful                                             │
│     - items_errors += failed                                                 │
│                                                                              │
│  4. Check completion:                                                        │
│     - if (tasks_done + tasks_errors) >= tasks_total → crawl phase done       │
│                                                                              │
│  5. Log platform limitation (for monitoring):                                │
│     - if platform_limited: log warning với requested vs found                │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 State Structure (Redis)

```json
{
  "status": "PROCESSING",

  "tasks_total": 6,
  "tasks_done": 4,
  "tasks_errors": 0,

  "items_expected": 300,
  "items_actual": 180,
  "items_errors": 5,

  "analyze_total": 180,
  "analyze_done": 100,
  "analyze_errors": 2
}
```

| Field            | Description                       | Updated When           |
| ---------------- | --------------------------------- | ---------------------- |
| `tasks_total`    | Số tasks dispatch                 | Khi dispatch           |
| `tasks_done`     | Số tasks hoàn thành               | Mỗi response (success) |
| `tasks_errors`   | Số tasks failed                   | Mỗi response (failed)  |
| `items_expected` | `tasks_total × limit_per_keyword` | Khi dispatch           |
| `items_actual`   | Tổng `stats.successful`           | Mỗi response           |
| `items_errors`   | Tổng `stats.failed`               | Mỗi response           |

---

## 5. Error Handling Contract

### 5.1 Retry Logic (Collector side)

| Condition                                    | Action                                   |
| -------------------------------------------- | ---------------------------------------- |
| `success: false` + `attempt < max_attempts`  | Retry với `attempt += 1`                 |
| `success: false` + `attempt >= max_attempts` | Mark as failed, không retry              |
| `success: true` + `payload: []`              | **Không retry** - platform không có data |
| Network error                                | Retry với backoff                        |

### 5.2 Error Codes (Crawler should return)

| Code              | Description           | Retryable             |
| ----------------- | --------------------- | --------------------- |
| `SEARCH_FAILED`   | Search API failed     | ✅ Yes                |
| `RATE_LIMITED`    | Platform rate limit   | ✅ Yes (with backoff) |
| `AUTH_FAILED`     | Authentication failed | ❌ No                 |
| `INVALID_KEYWORD` | Keyword không hợp lệ  | ❌ No                 |
| `CRAWL_PARTIAL`   | Một số videos fail    | ✅ Partial success    |
| `TIMEOUT`         | Request timeout       | ✅ Yes                |

---

## 6. Backward Compatibility

### 6.1 Migration Path

**Phase 1: Collector update (không break Crawler)**

- Collector gửi message format cũ (không thay đổi)
- Collector đọc response format cũ + mới (fallback logic)

**Phase 2: Crawler update**

- Crawler trả về response format mới với `limit_info` và `stats`
- Collector đọc full response mới

**Phase 3: Cleanup**

- Remove fallback logic trong Collector

### 6.2 Fallback Logic (Collector)

```go
// Nếu response không có limit_info (old format)
if response.LimitInfo == nil {
    // Fallback: đếm items trong payload
    itemCount := len(response.Payload)
    stats = Stats{
        Successful:     itemCount,
        Failed:         0,
        Skipped:        0,
        CompletionRate: 1.0,
    }
}
```

---

## 7. Summary Checklist

### Crawler cần implement:

- [ ] Respect `limit_per_keyword` - không crawl nhiều hơn
- [ ] Respect `max_comments` - không crawl nhiều comments hơn
- [ ] Trả về `limit_info` trong response
- [ ] Trả về `stats` trong response
- [ ] Set `platform_limited: true` khi search trả về < requested
- [ ] Include `job_id` và `task_type` trong response
- [ ] Return `success: true` với `payload: []` khi không có kết quả

### Collector sẽ:

- [ ] Gửi limits từ config (không hardcode)
- [ ] Gửi đúng 1 keyword per message
- [ ] Đọc `limit_info` và `stats` từ response
- [ ] Track state ở cả task-level và item-level
- [ ] Log warning khi `platform_limited: true`
- [ ] Retry logic dựa trên error codes

---

## 8. Example Full Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  EXAMPLE: 1 Project, 2 keywords, 2 platforms                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  INPUT:                                                                      │
│  - project_id: "proj123"                                                     │
│  - brand_keywords: ["iphone", "samsung"]                                     │
│  - limit_per_keyword: 50 (from config)                                       │
│                                                                              │
│  DISPATCH (4 messages):                                                      │
│  ├── TikTok: { job_id: "proj123-brand-0", keywords: ["iphone"], limit: 50 }  │
│  ├── TikTok: { job_id: "proj123-brand-1", keywords: ["samsung"], limit: 50 } │
│  ├── YouTube: { job_id: "proj123-brand-0", keywords: ["iphone"], limit: 50 } │
│  └── YouTube: { job_id: "proj123-brand-1", keywords: ["samsung"], limit: 50 }│
│                                                                              │
│  STATE AFTER DISPATCH:                                                       │
│  - tasks_total: 4                                                            │
│  - items_expected: 200 (4 × 50)                                              │
│                                                                              │
│  RESPONSES:                                                                  │
│  ├── TikTok "iphone": found=50, successful=48, failed=2                      │
│  ├── TikTok "samsung": found=30, successful=30, failed=0 (platform_limited)  │
│  ├── YouTube "iphone": found=50, successful=50, failed=0                     │
│  └── YouTube "samsung": found=50, successful=45, failed=5                    │
│                                                                              │
│  FINAL STATE:                                                                │
│  - tasks_total: 4                                                            │
│  - tasks_done: 4                                                             │
│  - items_expected: 200                                                       │
│  - items_actual: 173 (48+30+50+45)                                           │
│  - items_errors: 7 (2+0+0+5)                                                 │
│  - completion: 86.5% (173/200)                                               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Related Documents

- `collector-limit-config-analysis.md` - Phân tích vấn đề và giải pháp Collector
- `crawler-limit-optimization.md` - Phân tích vấn đề và giải pháp Crawler
- `event-drivent.md` - Event flow documentation
