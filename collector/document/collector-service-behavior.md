# SMAP Collector Service - Detailed Behavior Documentation

**Cập nhật:** 2025-12-06

---

## 1. Tổng quan Service

Collector Service là middleware giữa Project Service và Crawler Workers, chịu trách nhiệm:

1. **Nhận và dispatch tasks** từ Project Service đến các Crawler Workers
2. **Quản lý state** của project execution trong Redis
3. **Xử lý kết quả** từ Crawler và gửi webhook về Project Service
4. **Phân biệt task types** để xử lý khác nhau (dry-run vs project execution)

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              COLLECTOR SERVICE                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │   Dispatcher    │    │     Results     │    │      State      │         │
│  │    Consumer     │    │    Consumer     │    │     UseCase     │         │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘         │
│           │                      │                      │                   │
│           ▼                      ▼                      ▼                   │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐         │
│  │   Dispatcher    │    │     Results     │    │     Webhook     │         │
│  │    UseCase      │    │    UseCase      │    │     UseCase     │         │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘         │
│           │                      │                      │                   │
└───────────┼──────────────────────┼──────────────────────┼───────────────────┘
            │                      │                      │
            ▼                      ▼                      ▼
    ┌───────────────┐      ┌───────────────┐      ┌───────────────┐
    │   RabbitMQ    │      │   RabbitMQ    │      │    Redis      │
    │   (Outbound)  │      │   (Inbound)   │      │   (State)     │
    └───────────────┘      └───────────────┘      └───────────────┘
```

---

## 3. Consumer Queues

### 3.1. Dispatcher Consumer

| Queue                       | Exchange            | Routing Key       | Purpose                       |
| --------------------------- | ------------------- | ----------------- | ----------------------------- |
| `collector.inbound.tasks`   | `collector.inbound` | `crawler.*`       | Nhận dry-run tasks            |
| `collector.project.created` | `smap.events`       | `project.created` | Nhận project execution events |

### 3.2. Results Consumer

| Queue                  | Exchange          | Routing Key | Purpose                 |
| ---------------------- | ----------------- | ----------- | ----------------------- |
| `results.inbound.data` | `results.inbound` | `#`         | Nhận kết quả từ Crawler |

---

## 4. Task Types & Handling Strategy

### 4.1. Task Type Constants

```go
const (
    TaskTypeResearchKeyword  TaskType = "research_keyword"
    TaskTypeCrawlLinks       TaskType = "crawl_links"
    TaskTypeResearchAndCrawl TaskType = "research_and_crawl"
    TaskTypeDryRunKeyword    TaskType = "dryrun_keyword"
)
```

### 4.2. Handling Strategy Matrix

| Task Type            | Source            | Handler                 | Webhook Endpoint              |
| -------------------- | ----------------- | ----------------------- | ----------------------------- |
| `dryrun_keyword`     | Project Service   | `handleDryRunResult()`  | `/internal/dryrun/callback`   |
| `research_and_crawl` | Project Execution | `handleProjectResult()` | `/internal/progress/callback` |
| Unknown              | -                 | `handleDryRunResult()`  | `/internal/dryrun/callback`   |

---

## 5. Detailed Flow: Dry-Run

### 5.1. Flow Diagram

```
Project Service          Collector              Crawler              Project Service
      │                      │                     │                       │
      │ POST /projects/dryrun│                     │                       │
      │ (pub to collector.inbound)                 │                       │
      │─────────────────────▶│                     │                       │
      │                      │                     │                       │
      │                      │ Dispatch to crawler │                       │
      │                      │────────────────────▶│                       │
      │                      │                     │                       │
      │                      │                     │ Crawl & return result │
      │                      │◀────────────────────│                       │
      │                      │                     │                       │
      │                      │ handleDryRunResult()│                       │
      │                      │ SendDryRunCallback()│                       │
      │                      │─────────────────────────────────────────────▶│
      │                      │                     │                       │
      │                      │                     │       Publish to      │
      │                      │                     │    user_noti:{userID} │
      │◀───────────────────────────────────────────────────────────────────│
      │                      │                     │                       │
```

### 5.2. Dry-Run Result Handling

```go
func (uc implUseCase) handleDryRunResult(ctx context.Context, res models.CrawlerResult) error {
    // 1. Build callback request với content đã transform
    callbackReq, err := uc.buildCallbackRequest(ctx, res)

    // 2. Gửi webhook đến Project Service
    err = uc.projectClient.SendDryRunCallback(ctx, callbackReq)

    // 3. Project Service sẽ publish qua Redis Pub/Sub đến WebSocket
    return nil
}
```

### 5.3. Callback Request Structure

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

## 6. Detailed Flow: Project Execution

### 6.1. Flow Diagram

```
Project Service          Collector              Crawler              Redis
      │                      │                     │                   │
      │ POST /projects/:id/execute                 │                   │
      │ (init Redis state)   │                     │                   │
      │──────────────────────────────────────────────────────────────▶│
      │                      │                     │                   │
      │ pub project.created  │                     │                   │
      │─────────────────────▶│                     │                   │
      │                      │                     │                   │
      │                      │ Store user mapping  │                   │
      │                      │─────────────────────────────────────────▶│
      │                      │                     │                   │
      │                      │ Calculate total tasks                   │
      │                      │ UpdateTotal()       │                   │
      │                      │─────────────────────────────────────────▶│
      │                      │                     │                   │
      │                      │ Dispatch tasks      │                   │
      │                      │────────────────────▶│                   │
      │                      │                     │                   │
      │                      │                     │ Crawl each item   │
      │                      │◀────────────────────│                   │
      │                      │                     │                   │
      │                      │ handleProjectResult()                   │
      │                      │ IncrementDone() or IncrementErrors()    │
      │                      │─────────────────────────────────────────▶│
      │                      │                     │                   │
      │                      │ NotifyProgress()    │                   │
      │◀─────────────────────│                     │                   │
      │                      │                     │                   │
      │                      │ CheckAndUpdateCompletion()              │
      │                      │─────────────────────────────────────────▶│
      │                      │                     │                   │
```

### 6.2. Project Result Handling

```go
func (uc implUseCase) handleProjectResult(ctx context.Context, res models.CrawlerResult) error {
    // 1. Extract project_id từ job_id
    // Format: {projectID}-brand-{index} hoặc {projectID}-{competitor}-{index}
    projectID, err := uc.extractProjectID(ctx, res.Payload)

    // 2. Update Redis state
    if res.Success {
        uc.stateUC.IncrementDone(ctx, projectID)
    } else {
        uc.stateUC.IncrementErrors(ctx, projectID)
    }

    // 3. Get current state
    state, _ := uc.stateUC.GetState(ctx, projectID)
    userID, _ := uc.stateUC.GetUserID(ctx, projectID)

    // 4. Send progress webhook (non-fatal if fails)
    progressReq := webhook.ProgressRequest{
        ProjectID: projectID,
        UserID:    userID,
        Status:    string(state.Status),
        Total:     state.Total,
        Done:      state.Done,
        Errors:    state.Errors,
    }
    uc.webhookUC.NotifyProgress(ctx, progressReq)

    // 5. Check completion
    completed, _ := uc.stateUC.CheckAndUpdateCompletion(ctx, projectID)
    if completed {
        uc.webhookUC.NotifyCompletion(ctx, progressReq)
    }

    return nil
}
```

### 6.3. Job ID Format

| Type       | Format                             | Example                                |
| ---------- | ---------------------------------- | -------------------------------------- |
| Brand      | `{projectID}-brand-{index}`        | `proj_abc-brand-0`                     |
| Competitor | `{projectID}-{competitor}-{index}` | `proj_abc-toyota-0`                    |
| Dry-run    | `{uuid}`                           | `550e8400-e29b-41d4-a716-446655440000` |

---

## 7. Redis State Management

### 7.1. Key Schema

```
smap:proj:{projectID}           # Project execution state (Hash)
smap:proj:{projectID}:user      # User mapping (String)
```

### 7.2. State Fields

| Field    | Type   | Description                                      |
| -------- | ------ | ------------------------------------------------ |
| `status` | String | INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED |
| `total`  | Int64  | Tổng số tasks cần xử lý                          |
| `done`   | Int64  | Số tasks đã hoàn thành                           |
| `errors` | Int64  | Số tasks bị lỗi                                  |

### 7.3. State Transitions

```
INITIALIZING → CRAWLING (khi set total)
CRAWLING → DONE (khi done + errors >= total)
CRAWLING → FAILED (khi có fatal error)
```

---

## 8. Webhook Integration

### 8.1. Dry-Run Callback

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

### 8.2. Progress Callback

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

## 9. Error Handling

### 9.1. Error Types

| Error             | Description                    | Retry? |
| ----------------- | ------------------------------ | ------ |
| `ErrInvalidInput` | Permanent error (4xx)          | No     |
| `ErrTemporary`    | Temporary error (5xx, network) | Yes    |

### 9.2. Webhook Error Handling

```go
func (uc implUseCase) handleWebhookError(ctx context.Context, jobID, platform string, err error) error {
    // 4xx errors → ErrInvalidInput (no retry)
    // 5xx/network errors → ErrTemporary (retry)
    // Timeout → ErrTemporary (retry)
    // Unauthorized → ErrInvalidInput (no retry)
}
```

---

## 10. Data Transformation

### 10.1. Crawler → Project Content Mapping

```
CrawlerContent              →    project.Content
├── Meta                    →    ContentMeta
│   ├── ID                  →    ID
│   ├── Platform            →    Platform
│   ├── JobID               →    JobID
│   ├── TaskType            →    (used for routing, not mapped)
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

### 10.2. Timestamp Parsing

Supported formats:

- RFC3339: `2025-12-06T10:00:00Z`
- RFC3339Nano: `2025-12-06T10:00:00.123456789Z`
- Without timezone: `2025-12-06T10:00:00.123456`
- Without fractional: `2025-12-06T10:00:00`

---

## 11. Dependencies

### 11.1. Internal Dependencies

```go
type implUseCase struct {
    l             log.Logger
    projectClient project.Client    // HTTP client to Project Service
    stateUC       state.UseCase     // Redis state management
    webhookUC     webhook.UseCase   // Webhook notifications
}
```

### 11.2. External Services

| Service         | Protocol | Purpose                      |
| --------------- | -------- | ---------------------------- |
| Project Service | HTTP     | Webhooks (dry-run, progress) |
| Redis           | Redis    | State management             |
| RabbitMQ        | AMQP     | Message queue                |

---

## 12. Configuration

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
