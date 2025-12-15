# **Tài liệu mô tả chi tiết hành vi Collector Service SMAP**

**Cập nhật:** 2025-12-15 (Two-Phase State)

---

## 1. Tổng quan Service

Collector Service là một middleware nằm giữa Project Service và các Crawler Worker, chịu trách nhiệm:

1. **Nhận và phân phối task** từ Project Service tới các Crawler Worker
2. **Quản lý trạng thái** thực thi project trong Redis
3. **Xử lý kết quả** từ Crawler và gửi webhook về Project Service
4. **Phân biệt loại task** để xử lý khác biệt (dry-run và thực thi project)

---

## 2. Tổng quan Kiến trúc

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              COLLECTOR SERVICE                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐          │
│  │   Dispatcher    │    │    Results      │    │      State      │          │
│  │    Consumer     │    │    Consumer     │    │     UseCase     │          │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘          │
│           │                      │                      │                   │
│           ▼                      ▼                      ▼                   │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐          │
│  │   Dispatcher    │    │     Results     │    │     Webhook     │          │
│  │    UseCase      │    │    UseCase      │    │     UseCase     │          │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘          │
│           │                      │                      │                   │
└───────────┼──────────────────────┼──────────────────────┼───────────────────┘
            │                      │                      │
            ▼                      ▼                      ▼
    ┌───────────────┐      ┌───────────────┐      ┌───────────────┐
    │   RabbitMQ    │      │   RabbitMQ    │      │     Redis     │
    │   (Outbound)  │      │   (Inbound)   │      │   (State)     │
    └───────────────┘      └───────────────┘      └───────────────┘
```

---

## 3. Hàng đợi Consumer

### 3.1. Dispatcher Consumer

| Queue                       | Exchange            | Routing Key       | Mục đích                          |
| --------------------------- | ------------------- | ----------------- | --------------------------------- |
| `collector.inbound.tasks`   | `collector.inbound` | `crawler.*`       | Nhận các task dry-run             |
| `collector.project.created` | `smap.events`       | `project.created` | Nhận sự kiện thực thi project mới |

### 3.2. Results Consumer

| Queue                  | Exchange          | Routing Key | Mục đích                    |
| ---------------------- | ----------------- | ----------- | --------------------------- |
| `results.inbound.data` | `results.inbound` | `#`         | Nhận kết quả trả về Crawler |

---

## 4. Loại Task & Chiến lược Xử lý

### 4.1. Khai báo hằng số loại task

```go
const (
    TaskTypeResearchKeyword  TaskType = "research_keyword"
    TaskTypeCrawlLinks       TaskType = "crawl_links"
    TaskTypeResearchAndCrawl TaskType = "research_and_crawl"
    TaskTypeDryRunKeyword    TaskType = "dryrun_keyword"
)
```

### 4.2. Ma trận chiến lược xử lý

| Loại Task            | Nguồn khởi tạo    | Hàm xử lý               | Webhook Endpoint              |
| -------------------- | ----------------- | ----------------------- | ----------------------------- |
| `dryrun_keyword`     | Project Service   | `handleDryRunResult()`  | `/internal/dryrun/callback`   |
| `research_and_crawl` | Project Execution | `handleProjectResult()` | `/internal/progress/callback` |
| `analyze_result`     | Analytics Service | `handleAnalyzeResult()` | `/internal/progress/callback` |
| Không xác định       | -                 | `handleDryRunResult()`  | `/internal/dryrun/callback`   |

---

## 5. Luồng chi tiết: Dry-Run

### 5.1. Sơ đồ luồng

```mermaid
sequenceDiagram
    participant PS as Project Service
    participant COL as Collector
    participant CR as Crawler
    participant PS2 as Project Service

    PS->>COL: POST /projects/dryrun\n(gửi collector.inbound)
    COL->>CR: Dispatch tới crawler
    CR-->>COL: Crawler trả kết quả
    COL->>PS2: handleDryRunResult()\nSendDryRunCallback()
    PS2-->>COL: Gửi user_noti:{userID}
```

### 5.2. Hàm xử lý kết quả Dry-Run

```go
func (uc implUseCase) handleDryRunResult(ctx context.Context, res models.CrawlerResult) error {
    // 1. Tạo request callback với nội dung đã transform
    callbackReq, err := uc.buildCallbackRequest(ctx, res)

    // 2. Gửi webhook về Project Service
    err = uc.projectClient.SendDryRunCallback(ctx, callbackReq)

    // 3. Project Service sẽ sử dụng Redis Pub/Sub để đẩy lên WebSocket cho client
    return nil
}
```

### 5.3. Cấu trúc request callback

```json
{
  "job_id": "uuid",
  "status": "success",
  "platform": "tiktok",
  "payload": {
    "content": [
      {
        "meta": { "id": "...", "platform": "tiktok", "job_id": "..." },
        "content": { "text": "...", "hashtags": [...] },
        "interaction": { "views": 1000, "likes": 100 },
        "author": { "id": "...", "name": "...", "followers": 5000 },
        "comments": [...]
      }
    ]
  }
}
```

---

## 6. Luồng chi tiết: Project Execution

### 6.1. Sơ đồ luồng (Two-Phase)

```mermaid
sequenceDiagram
    participant PS as Project Service
    participant COL as Collector
    participant CR as Crawler
    participant AN as Analytics
    participant RED as Redis

    PS->>COL: phát project.created
    COL->>RED: Lưu user mapping
    COL->>RED: SetCrawlTotal()
    COL->>CR: Gửi task cho crawler

    Note over CR,COL: Phase 1: CRAWL
    CR-->>COL: Crawler trả kết quả
    COL->>RED: IncrementCrawlDoneBy(N)
    COL->>RED: IncrementAnalyzeTotalBy(N)
    COL->>PS: NotifyProgress (two-phase)

    Note over AN,COL: Phase 2: ANALYZE
    AN-->>COL: analyze_result
    COL->>RED: IncrementAnalyzeDoneBy(N)
    COL->>PS: NotifyProgress (two-phase)

    COL->>RED: CheckCompletion()
    COL->>PS: NotifyCompletion (khi cả 2 phase xong)
```

### 6.2. Hàm xử lý kết quả Project (Two-Phase)

```go
func (uc implUseCase) handleProjectResult(ctx context.Context, res models.CrawlerResult) error {
    // 1. Lấy project_id từ job_id
    projectID, err := uc.extractProjectID(ctx, res.Payload)

    // 2. Đếm số items trong batch
    itemCount := uc.countBatchItems(ctx, res.Payload)

    // 3. Cập nhật crawl counters trong Redis
    if res.Success {
        uc.stateUC.IncrementCrawlDoneBy(ctx, projectID, itemCount)
        // Auto-increment analyze_total cho mỗi crawl thành công
        uc.stateUC.IncrementAnalyzeTotalBy(ctx, projectID, itemCount)
    } else {
        uc.stateUC.IncrementCrawlErrorsBy(ctx, projectID, itemCount)
    }

    // 4. Lấy trạng thái và gửi webhook với two-phase format
    state, _ := uc.stateUC.GetState(ctx, projectID)
    userID, _ := uc.stateUC.GetUserID(ctx, projectID)
    progressReq := uc.buildTwoPhaseProgressRequest(projectID, userID, state)
    uc.webhookUC.NotifyProgress(ctx, progressReq)

    // 5. Kiểm tra hoàn thành (cả crawl và analyze)
    completed, _ := uc.stateUC.CheckCompletion(ctx, projectID)
    if completed {
        uc.webhookUC.NotifyCompletion(ctx, progressReq)
    }

    return nil
}
```

### 6.3. Hàm xử lý kết quả Analyze

```go
func (uc implUseCase) handleAnalyzeResult(ctx context.Context, res models.CrawlerResult) error {
    // 1. Extract analyze payload
    payload, _ := uc.extractAnalyzePayload(ctx, res.Payload)
    projectID := payload.ProjectID

    // 2. Cập nhật analyze counters
    if payload.SuccessCount > 0 {
        uc.stateUC.IncrementAnalyzeDoneBy(ctx, projectID, payload.SuccessCount)
    }
    if payload.ErrorCount > 0 {
        uc.stateUC.IncrementAnalyzeErrorsBy(ctx, projectID, payload.ErrorCount)
    }

    // 3. Gửi progress webhook và kiểm tra hoàn thành
    state, _ := uc.stateUC.GetState(ctx, projectID)
    userID, _ := uc.stateUC.GetUserID(ctx, projectID)
    progressReq := uc.buildTwoPhaseProgressRequest(projectID, userID, state)
    uc.webhookUC.NotifyProgress(ctx, progressReq)

    completed, _ := uc.stateUC.CheckCompletion(ctx, projectID)
    if completed {
        uc.webhookUC.NotifyCompletion(ctx, progressReq)
    }

    return nil
}
```

### 6.3. Định dạng Job ID

| Loại       | Định dạng                          | Ví dụ                                  |
| ---------- | ---------------------------------- | -------------------------------------- |
| Brand      | `{projectID}-brand-{index}`        | `proj_abc-brand-0`                     |
| Competitor | `{projectID}-{competitor}-{index}` | `proj_abc-toyota-0`                    |
| Dry-run    | `{uuid}`                           | `550e8400-e29b-41d4-a716-446655440000` |

---

## 7. Quản lý trạng thái Redis (Two-Phase State)

### 7.1. Định dạng key

```
smap:proj:{projectID}           # Trạng thái thực thi project (Hash)
smap:user:{projectID}           # Ánh xạ user (String)
```

### 7.2. Các trường trạng thái (Two-Phase)

| Trường           | Kiểu   | Mô tả                                  |
| ---------------- | ------ | -------------------------------------- |
| `status`         | String | INITIALIZING, PROCESSING, DONE, FAILED |
| `crawl_total`    | Int64  | Tổng số task crawl cần xử lý           |
| `crawl_done`     | Int64  | Số task crawl đã hoàn thành            |
| `crawl_errors`   | Int64  | Số task crawl bị lỗi                   |
| `analyze_total`  | Int64  | Tổng số task analyze cần xử lý         |
| `analyze_done`   | Int64  | Số task analyze đã hoàn thành          |
| `analyze_errors` | Int64  | Số task analyze bị lỗi                 |

### 7.3. Two-Phase Pipeline

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           TWO-PHASE PIPELINE                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Phase 1: CRAWL                      Phase 2: ANALYZE                       │
│  ┌─────────────────────────┐         ┌─────────────────────────┐            │
│  │ crawl_total: 100        │         │ analyze_total: 98       │            │
│  │ crawl_done: 98          │  ───►   │ analyze_done: 45        │            │
│  │ crawl_errors: 2         │         │ analyze_errors: 1       │            │
│  └─────────────────────────┘         └─────────────────────────┘            │
│                                                                             │
│  Crawler → Collector                 Analytics → Collector                  │
│  (research_and_crawl)                (analyze_result)                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.4. Chuyển trạng thái (state transition)

```
INITIALIZING → PROCESSING (khi set crawl_total)
PROCESSING → DONE (khi crawl complete AND analyze complete)
PROCESSING → FAILED (khi gặp lỗi không thể phục hồi)

Crawl complete: crawl_done + crawl_errors >= crawl_total
Analyze complete: analyze_done + analyze_errors >= analyze_total
```

### 7.5. Auto-increment analyze_total

Khi crawl thành công, `analyze_total` tự động tăng:

- Mỗi item crawl thành công = 1 item cần analyze
- `IncrementCrawlDoneBy(N)` → `IncrementAnalyzeTotalBy(N)`

---

## 8. Tích hợp Webhook

### 8.1. Callback Dry-Run

```
POST /internal/dryrun/callback
Header: Authorization: {internal_key}

{
  "job_id": "uuid",
  "status": "success" | "failed",
  "platform": "tiktok" | "youtube",
  "payload": {
    "content": [...],
    "errors": [...]
  }
}
```

### 8.2. Callback Progress (Two-Phase Format)

```
POST /internal/progress/callback
Header: X-Internal-Key: {internal_key}

{
  "project_id": "uuid",
  "user_id": "uuid",
  "status": "PROCESSING" | "DONE" | "FAILED",
  "crawl": {
    "total": 100,
    "done": 80,
    "errors": 2,
    "progress_percent": 82.0
  },
  "analyze": {
    "total": 78,
    "done": 45,
    "errors": 1,
    "progress_percent": 59.0
  },
  "overall_progress_percent": 70.5
}
```

---

## 9. Xử lý lỗi

### 9.1. Các loại lỗi

| Lỗi               | Diễn giải                   | Có retry không? |
| ----------------- | --------------------------- | --------------- |
| `ErrInvalidInput` | Lỗi không phục hồi (4xx)    | Không           |
| `ErrTemporary`    | Lỗi tạm thời (5xx, network) | Có              |

### 9.2. Xử lý lỗi khi gửi webhook

```go
func (uc implUseCase) handleWebhookError(ctx context.Context, jobID, platform string, err error) error {
    // Lỗi 4xx → ErrInvalidInput (không retry)
    // Lỗi 5xx/network → ErrTemporary (retry)
    // Timeout → ErrTemporary (retry)
    // Unauthorized → ErrInvalidInput (không retry)
}
```

---

## 10. Chuyển đổi dữ liệu

### 10.1. Mapping dữ liệu từ Crawler → Project Content

```
CrawlerContent              →    project.Content
├── Meta                    →    ContentMeta
│   ├── ID                  →    ID
│   ├── Platform            →    Platform
│   ├── JobID               →    JobID
│   ├── TaskType            →    (chỉ dùng cho định tuyến, không map)
│   ├── CrawledAt (string)  →    CrawledAt (time.Time)
│   └── PublishedAt (string)→    PublishedAt (time.Time)
├── Content                 →    ContentData
│   ├── Text                →    Text
│   ├── Duration            →    Duration
│   ├── Hashtags            →    Hashtags
│   ├── Title (YouTube)     →    Title
│   └── Media               →    Media
├── Interaction             →    ContentInteraction
│   ├── Views               →    Views
│   ├── Likes               →    Likes
│   └── CommentsCount       →    CommentsCount
├── Author                  →    ContentAuthor
│   ├── ID                  →    ID
│   ├── Name                →    Name
│   ├── Followers           →    Followers
│   └── Country (YouTube)   →    Country
└── Comments                →    []Comment
```

### 10.2. Parse timestamp

Hỗ trợ các format sau:

- RFC3339: `2025-12-06T10:00:00Z`
- RFC3339Nano: `2025-12-06T10:00:00.123456789Z`
- Không timezone: `2025-12-06T10:00:00.123456`
- Không thập phân: `2025-12-06T10:00:00`

---

## 11. Phụ thuộc hệ thống

### 11.1. Phụ thuộc nội bộ

```go
type implUseCase struct {
    l             log.Logger
    projectClient project.Client    // HTTP client tới Project Service
    stateUC       state.UseCase     // Quản lý state Redis
    webhookUC     webhook.UseCase   // Gửi webhook
}
```

### 11.2. Dịch vụ bên ngoài

| Dịch vụ         | Giao thức | Mục đích                    |
| --------------- | --------- | --------------------------- |
| Project Service | HTTP      | Webhook (dry-run, progress) |
| Redis           | Redis     | Quản lý trạng thái          |
| RabbitMQ        | AMQP      | Message queue               |

---

## 12. Cấu hình

```env
# Project Service
PROJECT_SERVICE_URL=http://project-service:8080
PROJECT_INTERNAL_KEY=your-internal-key

# Redis
REDIS_HOST=localhost:6379
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Dispatcher Options
DISPATCHER_SCHEMA_VERSION=1
DISPATCHER_DEFAULT_MAX_ATTEMPTS=3
```
