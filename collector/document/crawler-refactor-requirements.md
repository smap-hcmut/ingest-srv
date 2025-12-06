# Crawler Service - Refactor Requirements

**Cập nhật:** 2025-12-06

**Mục đích:** Document chi tiết những thay đổi cần thiết trên Crawler service để tương thích với Collector Service mới.

---

## 1. Tổng quan Thay đổi

### 1.1. Bối cảnh

Collector Service đã được cập nhật để phân biệt giữa:

- **Dry-run results** → Gửi về `/internal/dryrun/callback`
- **Project execution results** → Update Redis state + gửi về `/internal/progress/callback`

Để routing hoạt động đúng, Crawler cần include `task_type` trong metadata của kết quả trả về.

### 1.2. Tóm tắt Thay đổi

| Priority | Component                   | Change Type  | Effort |
| -------- | --------------------------- | ------------ | ------ |
| **P0**   | Result Meta - TaskType      | Required     | Low    |
| **P1**   | DataCollected Event         | Required     | Medium |
| **P2**   | MinIO Batch Upload          | Recommended  | Medium |
| **P3**   | Error Reporting Enhancement | Nice-to-have | Low    |

---

## 2. P0: Result Meta - TaskType (REQUIRED)

### 2.1. Vấn đề Hiện tại

Crawler trả về kết quả với `meta` không có `task_type`:

```json
{
  "success": true,
  "payload": [
    {
      "meta": {
        "id": "video_123",
        "platform": "tiktok",
        "job_id": "uuid",
        "crawled_at": "2025-12-06T10:00:00Z",
        "published_at": "2025-12-05T08:00:00Z"
        // ❌ Missing: task_type
      },
      "content": {...},
      "interaction": {...},
      "author": {...}
    }
  ]
}
```

### 2.2. Yêu cầu Thay đổi

Thêm `task_type` vào `meta` của mỗi content item:

```json
{
  "success": true,
  "payload": [
    {
      "meta": {
        "id": "video_123",
        "platform": "tiktok",
        "job_id": "uuid",
        "task_type": "dryrun_keyword",  // ✅ REQUIRED
        "crawled_at": "2025-12-06T10:00:00Z",
        "published_at": "2025-12-05T08:00:00Z"
      },
      "content": {...},
      "interaction": {...},
      "author": {...}
    }
  ]
}
```

### 2.3. Task Type Values

| Value                | Description                          | Routing                                 |
| -------------------- | ------------------------------------ | --------------------------------------- |
| `dryrun_keyword`     | Dry-run test keywords                | → `/internal/dryrun/callback`           |
| `research_and_crawl` | Project execution (brand/competitor) | → Redis + `/internal/progress/callback` |
| `research_keyword`   | Research only (không crawl content)  | → `/internal/dryrun/callback`           |
| `crawl_links`        | Crawl specific links                 | → `/internal/dryrun/callback`           |

### 2.4. Implementation Guide

**Python Example:**

```python
class CrawlerResult:
    def __init__(self, task_type: str, job_id: str, platform: str):
        self.task_type = task_type
        self.job_id = job_id
        self.platform = platform
        self.content_items = []

    def add_content(self, content_data: dict):
        content_data["meta"]["task_type"] = self.task_type
        self.content_items.append(content_data)

    def to_message(self) -> dict:
        return {
            "success": True,
            "payload": self.content_items
        }
```

**Go Example:**

```go
type ContentMeta struct {
    ID          string `json:"id"`
    Platform    string `json:"platform"`
    JobID       string `json:"job_id"`
    TaskType    string `json:"task_type"`  // NEW FIELD
    CrawledAt   string `json:"crawled_at"`
    PublishedAt string `json:"published_at"`
    // ... other fields
}
```

### 2.5. Backward Compatibility

Nếu `task_type` không có hoặc rỗng, Collector sẽ default về `handleDryRunResult()`. Tuy nhiên, điều này sẽ gây ra:

- Project execution results không update Redis state
- Progress không được track đúng
- User không nhận được real-time progress updates

**⚠️ QUAN TRỌNG:** Phải implement `task_type` trước khi deploy project execution feature.

---

## 3. P1: DataCollected Event (REQUIRED)

### 3.1. Bối cảnh

Theo architecture document, Crawler (không phải Collector) chịu trách nhiệm:

1. Upload batch data lên MinIO
2. Publish `data.collected` event với `minio_path`

Điều này tránh nghẽn message queue khi crawl nhiều bài.

### 3.2. Event Schema

```json
{
  "event_id": "evt_abc123",
  "timestamp": "2025-12-06T10:00:00Z",
  "payload": {
    "project_id": "proj_xyz",
    "job_id": "proj_xyz-brand-0",
    "platform": "tiktok",
    "minio_path": "crawl-results/proj_xyz/brand/batch_001.json",
    "content_count": 50,
    "batch_index": 1,
    "total_batches": 10
  }
}
```

### 3.3. RabbitMQ Configuration

```
Exchange: smap.events
Routing Key: data.collected
Queue: analytics.data.collected (Analytics Service sẽ consume)
```

### 3.4. Implementation Steps

1. **MinIO Client Setup**

   ```python
   from minio import Minio

   minio_client = Minio(
       endpoint=os.getenv("MINIO_ENDPOINT"),
       access_key=os.getenv("MINIO_ACCESS_KEY"),
       secret_key=os.getenv("MINIO_SECRET_KEY"),
       secure=True
   )
   ```

2. **Batch Upload Logic**

   ```python
   def upload_batch(project_id: str, platform: str, batch_index: int, content_items: list):
       bucket = "crawl-results"
       object_name = f"{project_id}/{platform}/batch_{batch_index:03d}.json"

       data = json.dumps(content_items).encode('utf-8')
       minio_client.put_object(
           bucket_name=bucket,
           object_name=object_name,
           data=io.BytesIO(data),
           length=len(data),
           content_type="application/json"
       )

       return f"{bucket}/{object_name}"
   ```

3. **Publish Event**
   ```python
   def publish_data_collected(project_id: str, job_id: str, platform: str,
                               minio_path: str, content_count: int,
                               batch_index: int, total_batches: int):
       event = {
           "event_id": str(uuid.uuid4()),
           "timestamp": datetime.utcnow().isoformat() + "Z",
           "payload": {
               "project_id": project_id,
               "job_id": job_id,
               "platform": platform,
               "minio_path": minio_path,
               "content_count": content_count,
               "batch_index": batch_index,
               "total_batches": total_batches
           }
       }

       channel.basic_publish(
           exchange="smap.events",
           routing_key="data.collected",
           body=json.dumps(event),
           properties=pika.BasicProperties(
               delivery_mode=2,  # persistent
               content_type="application/json"
           )
       )
   ```

### 3.5. Batch Size Recommendation

| Platform | Recommended Batch Size | Reason                        |
| -------- | ---------------------- | ----------------------------- |
| TikTok   | 50 items               | Smaller content, faster crawl |
| YouTube  | 20 items               | Larger content with comments  |

---

## 4. P2: MinIO Batch Upload (RECOMMENDED)

### 4.1. Current Flow (Không tối ưu)

```
Crawler → RabbitMQ (full content) → Collector → Project Service
```

**Vấn đề:**

- Message size lớn khi có nhiều content
- Queue có thể bị nghẽn
- Memory pressure trên Collector

### 4.2. Recommended Flow

```
Crawler → MinIO (batch upload) → RabbitMQ (minio_path only) → Analytics
```

**Lợi ích:**

- Message size nhỏ (~200 bytes vs ~50KB)
- Không nghẽn queue
- Analytics có thể parallel fetch từ MinIO

### 4.3. MinIO Bucket Structure

```
crawl-results/
├── {project_id}/
│   ├── brand/
│   │   ├── batch_001.json
│   │   ├── batch_002.json
│   │   └── ...
│   ├── {competitor_name}/
│   │   ├── batch_001.json
│   │   └── ...
│   └── metadata.json
└── dryrun/
    └── {job_id}.json
```

### 4.4. Metadata File

```json
{
  "project_id": "proj_xyz",
  "created_at": "2025-12-06T10:00:00Z",
  "platforms": ["tiktok", "youtube"],
  "brands": {
    "total_batches": 10,
    "total_items": 500
  },
  "competitors": {
    "toyota": { "total_batches": 5, "total_items": 250 },
    "honda": { "total_batches": 5, "total_items": 250 }
  }
}
```

---

## 5. P3: Error Reporting Enhancement (NICE-TO-HAVE)

### 5.1. Current Error Format

```json
{
  "success": false,
  "payload": null
}
```

### 5.2. Enhanced Error Format

```json
{
  "success": false,
  "payload": [],
  "error": {
    "code": "RATE_LIMITED",
    "message": "TikTok API rate limit exceeded",
    "details": {
      "retry_after": 3600,
      "platform": "tiktok",
      "job_id": "uuid"
    }
  }
}
```

### 5.3. Error Codes

| Code              | Description                 | Retryable |
| ----------------- | --------------------------- | --------- |
| `RATE_LIMITED`    | Platform rate limit         | Yes       |
| `AUTH_FAILED`     | Authentication failed       | No        |
| `CONTENT_REMOVED` | Content no longer available | No        |
| `NETWORK_ERROR`   | Network connectivity issue  | Yes       |
| `PARSE_ERROR`     | Failed to parse response    | No        |
| `TIMEOUT`         | Request timeout             | Yes       |

---

## 6. Testing Checklist

### 6.1. Unit Tests

- [ ] `task_type` được include trong mọi content item
- [ ] `task_type` mapping đúng với task được dispatch
- [ ] MinIO upload thành công với đúng path
- [ ] Event publish với đúng schema

### 6.2. Integration Tests

- [ ] Dry-run flow: Crawler → Collector → Project Service
- [ ] Project execution flow: Crawler → Collector → Redis + Webhook
- [ ] MinIO batch upload → Event publish → Analytics consume

### 6.3. E2E Tests

- [ ] Full dry-run cycle với WebSocket notification
- [ ] Full project execution với progress tracking
- [ ] Error handling và retry logic

---

## 7. Migration Plan

### Phase 1: TaskType Implementation (Week 1)

1. Add `task_type` field to result meta
2. Propagate `task_type` from task to result
3. Deploy Crawler with new field
4. Verify Collector routing works correctly

### Phase 2: MinIO Integration (Week 2)

1. Setup MinIO client in Crawler
2. Implement batch upload logic
3. Implement `data.collected` event publisher
4. Test with Analytics Service (mock)

### Phase 3: Full Integration (Week 3)

1. Deploy Analytics Service consumer
2. Enable full event-driven flow
3. Monitor and tune batch sizes
4. Performance testing

---

## 8. Configuration Requirements

### 8.1. Environment Variables

```env
# MinIO
MINIO_ENDPOINT=minio.example.com:9000
MINIO_ACCESS_KEY=your-access-key
MINIO_SECRET_KEY=your-secret-key
MINIO_BUCKET=crawl-results
MINIO_SECURE=true

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Batch Settings
BATCH_SIZE_TIKTOK=50
BATCH_SIZE_YOUTUBE=20
```

### 8.2. RabbitMQ Permissions

Crawler cần quyền publish đến:

- Exchange: `results.inbound` (existing)
- Exchange: `smap.events` (new - for data.collected)

---

## 9. Monitoring & Alerts

### 9.1. Metrics to Track

| Metric                        | Description            | Alert Threshold |
| ----------------------------- | ---------------------- | --------------- |
| `crawler_result_task_type`    | Count by task_type     | -               |
| `crawler_minio_upload_time`   | Upload latency         | > 5s            |
| `crawler_event_publish_error` | Event publish failures | > 0             |
| `crawler_batch_size`          | Items per batch        | -               |

### 9.2. Log Format

```json
{
  "level": "info",
  "timestamp": "2025-12-06T10:00:00Z",
  "service": "crawler",
  "event": "result_published",
  "job_id": "uuid",
  "task_type": "research_and_crawl",
  "platform": "tiktok",
  "content_count": 50,
  "minio_path": "crawl-results/proj_xyz/brand/batch_001.json"
}
```

---

## 10. Summary

### Immediate Actions (P0)

1. ✅ Collector đã implement task_type routing
2. ⏳ **Crawler cần thêm `task_type` vào result meta**

### Short-term Actions (P1)

1. ⏳ Implement MinIO batch upload
2. ⏳ Implement `data.collected` event publisher

### Medium-term Actions (P2-P3)

1. ⏳ Optimize batch sizes
2. ⏳ Enhanced error reporting
3. ⏳ Monitoring & alerting setup
