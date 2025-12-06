# **Tài liệu mô tả chi tiết hành vi Collector Service SMAP**

**Cập nhật:** 2025-12-06

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

### 6.1. Sơ đồ luồng

```mermaid
sequenceDiagram
    participant PS as Project Service
    participant COL as Collector
    participant CR as Crawler
    participant RED as Redis

    PS->>COL: POST /projects/:id/execute\n(khởi tạo trạng thái Redis)
    PS->>COL: phát project.created
    COL->>RED: Lưu user mapping
    COL->>RED: Tính tổng số task\nUpdateTotal()
    COL->>CR: Gửi task cho crawler
    CR-->>COL: Crawler thực thi
    COL->>RED: handleProjectResult()\nIncrementDone()/IncrementErrors()
    COL->>RED: NotifyProgress()
    RED-->>COL:
    COL->>RED: Kiểm tra/đánh dấu hoàn thành
```

### 6.2. Hàm xử lý kết quả Project

```go
func (uc implUseCase) handleProjectResult(ctx context.Context, res models.CrawlerResult) error {
    // 1. Lấy project_id từ job_id
    // Định dạng: {projectID}-brand-{index} hoặc {projectID}-{competitor}-{index}
    projectID, err := uc.extractProjectID(ctx, res.Payload)

    // 2. Cập nhật trạng thái trong Redis
    if res.Success {
        uc.stateUC.IncrementDone(ctx, projectID)
    } else {
        uc.stateUC.IncrementErrors(ctx, projectID)
    }

    // 3. Lấy trạng thái hiện tại
    state, _ := uc.stateUC.GetState(ctx, projectID)
    userID, _ := uc.stateUC.GetUserID(ctx, projectID)

    // 4. Gửi webhook progress (không nguy hiểm nếu gặp lỗi)
    progressReq := webhook.ProgressRequest{
        ProjectID: projectID,
        UserID:    userID,
        Status:    string(state.Status),
        Total:     state.Total,
        Done:      state.Done,
        Errors:    state.Errors,
    }
    uc.webhookUC.NotifyProgress(ctx, progressReq)

    // 5. Kiểm tra hoàn thành
    completed, _ := uc.stateUC.CheckAndUpdateCompletion(ctx, projectID)
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

## 7. Quản lý trạng thái Redis

### 7.1. Định dạng key

```
smap:proj:{projectID}           # Trạng thái thực thi project (Hash)
smap:proj:{projectID}:user      # Ánh xạ user (String)
```

### 7.2. Các trường trạng thái

| Trường   | Kiểu   | Mô tả                                            |
| -------- | ------ | ------------------------------------------------ |
| `status` | String | INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED |
| `total`  | Int64  | Tổng số task cần xử lý                           |
| `done`   | Int64  | Số task đã hoàn thành                            |
| `errors` | Int64  | Số task bị lỗi                                   |

### 7.3. Chuyển trạng thái (state transition)

```
INITIALIZING → CRAWLING (khi set total)
CRAWLING → DONE (khi done + errors >= total)
CRAWLING → FAILED (khi gặp lỗi không thể phục hồi)
```

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

### 8.2. Callback Progress

```
POST /internal/progress/callback
Header: X-Internal-Key: {internal_key}

{
  "project_id": "uuid",
  "user_id": "uuid",
  "status": "CRAWLING" | "DONE" | "FAILED",
  "total": 100,
  "done": 50,
  "errors": 2
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
