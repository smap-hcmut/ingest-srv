# Yêu Cầu Refactor - Dịch vụ Crawler

**Cập nhật:** 2025-12-06

**Mục đích:** Tài liệu này mô tả chi tiết các thay đổi cần thiết trên dịch vụ Crawler để tương thích với Collector Service mới.

---

## 1. Tổng Quan Thay Đổi

### 1.1. Bối Cảnh

Collector Service đã được nâng cấp để phân biệt giữa:

- **Kết quả Dry-run** → Gửi về `/internal/dryrun/callback`
- **Kết quả thực thi dự án** → Cập nhật trạng thái Redis + gửi về `/internal/progress/callback`

Để routing hoạt động chính xác, Crawler cần bổ sung trường `task_type` trong metadata của kết quả trả về.

### 1.2. Tóm Tắt Thay Đổi

| Mức ưu tiên | Thành phần              | Loại thay đổi | Độ phức tạp |
| ----------- | ----------------------- | ------------- | ----------- |
| **P0**      | Result Meta - TaskType  | Bắt buộc      | Thấp        |
| **P1**      | Sự kiện DataCollected   | Bắt buộc      | Trung bình  |
| **P2**      | Upload MinIO theo batch | Khuyến nghị   | Trung bình  |
| **P3**      | Nâng cao báo lỗi        | Nên có        | Thấp        |

---

## 2. P0: Result Meta - TaskType (BẮT BUỘC)

### 2.1. Vấn Đề Hiện Tại

Crawler trả về kết quả với `meta` chưa có `task_type`:

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
        // ❌ Thiếu trường: task_type
      },
      "content": {...},
      "interaction": {...},
      "author": {...}
    }
  ]
}
```

### 2.2. Yêu Cầu Thay Đổi

Bổ sung trường `task_type` vào `meta` của từng content item:

```json
{
  "success": true,
  "payload": [
    {
      "meta": {
        "id": "video_123",
        "platform": "tiktok",
        "job_id": "uuid",
        "task_type": "dryrun_keyword",  // ✅ BẮT BUỘC
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

### 2.3. Giá Trị `task_type` Hỗ Trợ

| Giá trị              | Diễn giải                                   | Nơi chuyển tiếp                         |
| -------------------- | ------------------------------------------- | --------------------------------------- |
| `dryrun_keyword`     | Dry-run kiểm thử từ khoá                    | → `/internal/dryrun/callback`           |
| `research_and_crawl` | Thực thi dự án (brand/competitor)           | → Redis + `/internal/progress/callback` |
| `research_keyword`   | Chỉ research keyword (không crawl nội dung) | → `/internal/dryrun/callback`           |
| `crawl_links`        | Crawl theo link cụ thể                      | → `/internal/dryrun/callback`           |

### 2.4. Hướng Dẫn Triển Khai

**Ví dụ Python:**

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

**Ví dụ Go:**

```go
type ContentMeta struct {
    ID          string `json:"id"`
    Platform    string `json:"platform"`
    JobID       string `json:"job_id"`
    TaskType    string `json:"task_type"`  // Trường mới
    CrawledAt   string `json:"crawled_at"`
    PublishedAt string `json:"published_at"`
    // ... các field khác
}
```

### 2.5. Tương Thích Ngược

Nếu thiếu hoặc rỗng `task_type`, Collector sẽ mặc định gọi `handleDryRunResult()`. Tuy nhiên, hậu quả là:

- Kết quả dự án sẽ không update trạng thái Redis
- Tiến trình không được theo dõi chuẩn xác
- Người dùng không nhận được thông báo trạng thái thực

**⚠️ QUAN TRỌNG:** Phải bổ sung `task_type` trước khi triển khai tính năng thực thi dự án.

---

## 3. P1: Sự kiện DataCollected (BẮT BUỘC)

### 3.1. Bối Cảnh

Theo tài liệu kiến trúc, Crawler (không phải Collector) sẽ:

1. Upload dữ liệu (batch) lên MinIO
2. Publish sự kiện `data.collected` với trường `minio_path`

Việc này giúp tránh tắc nghẽn message queue khi số lượng bài lớn.

### 3.2. Mẫu Schema Sự Kiện

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

### 3.3. Cấu Hình RabbitMQ

```
Exchange: smap.events
Routing Key: data.collected
Queue: analytics.data.collected (Service Analytics sẽ consume)
```

### 3.4. Các Bước Triển Khai

1. **Khởi tạo MinIO Client**

   ```python
   from minio import Minio

   minio_client = Minio(
       endpoint=os.getenv("MINIO_ENDPOINT"),
       access_key=os.getenv("MINIO_ACCESS_KEY"),
       secret_key=os.getenv("MINIO_SECRET_KEY"),
       secure=True
   )
   ```

2. **Logic upload batch**

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

3. **Publish sự kiện**

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

### 3.5. Gợi Ý Kích Thước Batch

| Nền tảng | Batch size đề nghị | Lý do                       |
| -------- | ------------------ | --------------------------- |
| TikTok   | 50 items           | Nội dung nhỏ, crawl nhanh   |
| YouTube  | 20 items           | Nội dung lớn, nhiều comment |

---

## 4. P2: Upload MinIO theo batch (KHUYẾN NGHỊ)

### 4.1. Luồng Hiện Tại (Không tối ưu)

```
Crawler → RabbitMQ (full content) → Collector → Project Service
```

**Vấn đề:**

- Kích thước message lớn nếu nhiều content
- Dễ nghẽn queue
- Collector tốn nhiều bộ nhớ

### 4.2. Luồng Đề Xuất

```
Crawler → MinIO (batch upload) → RabbitMQ (chỉ minio_path) → Analytics
```

**Lợi ích:**

- Message size nhỏ (~200 bytes vs ~50KB)
- Không bị tắc queue
- Analytics có thể đồng thời fetch từ MinIO

### 4.3. Cấu Trúc Bucket MinIO

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

### 4.4. File metadata.json

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

## 5. P3: Nâng cao báo lỗi (NÊN CÓ)

### 5.1. Định Dạng Báo Lỗi Hiện Tại

```json
{
  "success": false,
  "payload": null
}
```

### 5.2. Định Dạng Báo Lỗi Đề Xuất

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

### 5.3. Các Loại Mã Lỗi

| Mã lỗi            | Ý nghĩa                    | Có thể retry? |
| ----------------- | -------------------------- | ------------- |
| `RATE_LIMITED`    | Nhà cung cấp giới hạn lượt | Có            |
| `AUTH_FAILED`     | Lỗi xác thực               | Không         |
| `CONTENT_REMOVED` | Nội dung đã bị xoá         | Không         |
| `NETWORK_ERROR`   | Lỗi kết nối mạng           | Có            |
| `PARSE_ERROR`     | Không parse được kết quả   | Không         |
| `TIMEOUT`         | Quá thời gian chờ          | Có            |

---

## 6. Checklist Kiểm Thử

### 6.1. Unit Tests

- [ ] `task_type` có trong mọi content item
- [ ] Đúng mapping `task_type` với loại task
- [ ] Upload MinIO đúng đường dẫn
- [ ] Sự kiện publish đúng schema

### 6.2. Integration Tests

- [ ] Luồng dry-run: Crawler → Collector → Project Service
- [ ] Luồng thực thi dự án: Crawler → Collector → Redis + Webhook
- [ ] Upload MinIO → Publish event → Analytics consume

### 6.3. E2E Tests

- [ ] Full dry-run với thông báo WebSocket
- [ ] Full project execution với theo dõi tiến trình
- [ ] Xử lý lỗi & logic retry

---

## 7. Kế Hoạch Migration

### Giai đoạn 1: Thêm TaskType (Tuần 1)

1. Thêm trường `task_type` vào result meta
2. Truyền `task_type` từ task đến kết quả
3. Deploy Crawler với trường mới
4. Xác thực Collector routing hoạt động đúng

### Giai đoạn 2: Kết nối MinIO (Tuần 2)

1. Setup MinIO client ở Crawler
2. Triển khai logic upload theo batch
3. Thêm publisher event `data.collected`
4. Kiểm thử với Service Analytics (mock)

### Giai đoạn 3: Full Integration (Tuần 3)

1. Deploy consumer phân tích dữ liệu Analytics Service
2. Bật cơ chế truyền event đầy đủ
3. Theo dõi & điều chỉnh batch size
4. Đo kiểm hiệu năng

---

## 8. Yêu Cầu Cấu Hình

### 8.1. Biến môi trường (Environment variables)

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

### 8.2. Quyền RabbitMQ

Crawler cần quyền publish lên:

- Exchange: `results.inbound` (đang sử dụng)
- Exchange: `smap.events` (mới - cho event data.collected)

---

## 9. Monitoring & Alerts

### 9.1. Các Chỉ Số Theo Dõi

| Metric                        | Giải thích             | Ngưỡng cảnh báo |
| ----------------------------- | ---------------------- | --------------- |
| `crawler_result_task_type`    | Đếm theo task_type     | -               |
| `crawler_minio_upload_time`   | Thời gian upload MinIO | > 5s            |
| `crawler_event_publish_error` | Số lỗi publish sự kiện | > 0             |
| `crawler_batch_size`          | Số item mỗi batch      | -               |

### 9.2. Mẫu Log

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

## 10. Tổng Kết

### Việc Cần Làm Ngay (P0)

1. ✅ Collector đã triển khai logic routing theo task_type
2. ⏳ **Crawler cần bổ sung `task_type` vào result meta**

### Việc Cần Làm Ngắn Hạn (P1)

1. ⏳ Triển khai upload batch MinIO
2. ⏳ Triển khai publisher event `data.collected`

### Việc Cần Làm Trung Hạn (P2-P3)

1. ⏳ Tối ưu batch size
2. ⏳ Nâng cao báo lỗi
3. ⏳ Thiết lập giám sát & cảnh báo
