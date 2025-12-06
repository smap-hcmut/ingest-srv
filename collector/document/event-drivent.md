# IMPLEMENTATION GUIDE: EVENT-DRIVEN CHOREOGRAPHY FOR SMAP

**Mục tiêu:** Thiết kế luồng dữ liệu tự hành (Autonomous Data Flow) từ Project → Collector → Analytics → Dashboard.

**Cập nhật:** 2025-12-06 - Đã implement và verify trên cả Project Service và Collector Service.

---

## Implementation Status

| Service               | Component                     | Status         | Verified   |
| --------------------- | ----------------------------- | -------------- | ---------- |
| **Project Service**   | Exchange Configuration        | ✅ Implemented | 2025-12-05 |
| **Project Service**   | ProjectCreatedEvent Publisher | ✅ Implemented | 2025-12-05 |
| **Project Service**   | Redis State Init              | ✅ Implemented | 2025-12-05 |
| **Project Service**   | Progress Webhook Handler      | ✅ Implemented | 2025-12-05 |
| **Collector Service** | Exchange Configuration        | ✅ Implemented | 2025-12-06 |
| **Collector Service** | ProjectCreatedEvent Consumer  | ✅ Implemented | 2025-12-06 |
| **Collector Service** | Redis State Management        | ✅ Implemented | 2025-12-06 |
| **Collector Service** | Progress Webhook Client       | ✅ Implemented | 2025-12-06 |
| **Collector Service** | User Mapping Storage          | ✅ Implemented | 2025-12-06 |
| **Collector Service** | Task Type Differentiation     | ✅ Implemented | 2025-12-06 |
| **Collector Service** | Dry-Run Result Handler        | ✅ Implemented | 2025-12-06 |
| **Collector Service** | Project Result Handler        | ✅ Implemented | 2025-12-06 |
| **Crawler (Worker)**  | TaskType in Result Meta       | ⏳ Pending     | -          |
| **Crawler (Worker)**  | DataCollectedEvent Publisher  | ⏳ Pending     | -          |
| Analytics Service     | DataCollectedEvent Consumer   | ⏳ Pending     | -          |
| WebSocket Service     | Redis Pub/Sub Subscriber      | ⏳ Pending     | -          |

**Collector Service Compliance:** Verified via `openspec/changes/review-event-driven-compliance/`

---

## Recent Changes (2025-12-06)

### Collector Service - Task Type Differentiation

**Problem:** Results handler luôn gọi `SendDryRunCallback()` cho mọi kết quả, không phân biệt dry-run vs project execution.

**Solution:** Thêm `task_type` field vào `CrawlerContentMeta` và routing logic trong `HandleResult()`:

```go
// internal/results/types.go
type CrawlerContentMeta struct {
    // ... existing fields
    TaskType string `json:"task_type,omitempty"` // "dryrun_keyword" or "research_and_crawl"
}

// internal/results/usecase/result.go
func (uc implUseCase) HandleResult(ctx context.Context, res models.CrawlerResult) error {
    taskType := uc.extractTaskType(ctx, res.Payload)

    switch taskType {
    case "dryrun_keyword":
        return uc.handleDryRunResult(ctx, res)      // → /internal/dryrun/callback
    case "research_and_crawl":
        return uc.handleProjectResult(ctx, res)     // → Update Redis + /internal/progress/callback
    default:
        return uc.handleDryRunResult(ctx, res)      // Backward compatibility
    }
}
```

**Files Changed:**

- `internal/results/types.go` - Added `TaskType` field
- `internal/results/usecase/new.go` - Added `stateUC`, `webhookUC` dependencies
- `internal/results/usecase/result.go` - Added routing logic and handlers
- `internal/consumer/server.go` - Updated constructor call

**Crawler Requirement:** Crawler cần include `task_type` trong `meta` của mỗi content item khi trả kết quả.

---

## 1. Thiết kế Hạ tầng Sự kiện (Event Infrastructure)

### 1.1. Cấu trúc Exchange & Routing Key

Sử dụng 1 Exchange chính: `smap.events` (Type: `topic`).

| Routing Key         | Ý nghĩa                       | Producer          | Consumers         |
| :------------------ | :---------------------------- | :---------------- | :---------------- |
| `project.created`   | Có dự án mới cần chạy         | Project Service   | Collector Service |
| `data.collected`    | Dữ liệu thô đã nằm trên MinIO | Crawler (Worker)  | Analytics Service |
| `analysis.finished` | Phân tích xong 1 bài          | Analytics Service | Insight Service   |
| `job.completed`     | Toàn bộ dự án đã xong         | Analytics Service | Notification      |

> **Lưu ý:** Crawler upload batch data lên MinIO trước, sau đó pub 1 event `data.collected` với `minio_path`. Analytics Service sẽ fetch data từ MinIO thay vì nhận trực tiếp qua message. Điều này tránh nghẽn khi có nhiều bài.

### 1.2. Exchange Configuration (Đã implement)

```go
// Project Service - Exchange declaration
exchange := "smap.events"
exchangeType := "topic"
routingKey := "project.created"
```

---

## 2. Project Execution Flow (Đã implement)

### 2.1. Tách biệt Create và Execute

**QUAN TRỌNG:** Flow đã được tách thành 2 bước riêng biệt:

| Step | Endpoint                   | Action                              |
| ---- | -------------------------- | ----------------------------------- |
| 1    | POST /projects             | Tạo project (PostgreSQL only)       |
| 2    | POST /projects/dryrun      | Optional: Test keywords             |
| 3    | POST /projects/:id/execute | Start execution (Redis + RabbitMQ)  |
| 4    | GET /projects/:id/progress | Monitor progress (polling fallback) |

**Lý do tách:**

- User có thể tạo project, chạy dry-run test keywords trước
- Sau khi hài lòng với keywords mới bấm Execute
- Tránh publish event khi user chưa sẵn sàng

### 2.2. Execute Endpoint Implementation

```go
// POST /projects/:id/execute
// internal/project/usecase/project.go

func (u *projectUseCase) Execute(ctx context.Context, projectID, userID string) error {
    // 1. Get project from PostgreSQL
    project, err := u.repo.GetByID(ctx, projectID)
    if err != nil {
        return err
    }

    // 2. Authorization check
    if project.CreatedBy != userID {
        return ErrUnauthorized
    }

    // 3. Check if already executing (prevent duplicate)
    state, _ := u.stateUC.GetState(ctx, projectID)
    if state != nil && state.Status != model.ProjectStatusDone &&
       state.Status != model.ProjectStatusFailed {
        return ErrProjectAlreadyExecuting
    }

    // 4. Initialize Redis state
    if err := u.stateUC.InitState(ctx, projectID); err != nil {
        return err
    }

    // 5. Publish event to RabbitMQ
    event := ToProjectCreatedEvent(project)
    return u.eventPublisher.PublishProjectCreated(ctx, event)
}
```

---

## 3. Event Schema (Đã implement)

### 3.1. `project.created` Event

**QUAN TRỌNG:** Event payload bao gồm `user_id` để Collector biết notify cho ai.

```go
// internal/project/delivery/rabbitmq/types.go

type ProjectCreatedEvent struct {
    EventID   string                `json:"event_id"`
    Timestamp time.Time             `json:"timestamp"`
    Payload   ProjectCreatedPayload `json:"payload"`
}

type ProjectCreatedPayload struct {
    ProjectID             string              `json:"project_id"`
    UserID                string              `json:"user_id"`  // For progress notifications
    BrandName             string              `json:"brand_name"`
    BrandKeywords         []string            `json:"brand_keywords"`
    CompetitorNames       []string            `json:"competitor_names"`
    CompetitorKeywordsMap map[string][]string `json:"competitor_keywords_map"`
    DateRange             DateRange           `json:"date_range"`
}

type DateRange struct {
    From string `json:"from"` // Format: YYYY-MM-DD
    To   string `json:"to"`   // Format: YYYY-MM-DD
}
```

**Example Event:**

```json
{
  "event_id": "evt_abc123",
  "timestamp": "2025-12-05T10:00:00Z",
  "payload": {
    "project_id": "proj_xyz",
    "user_id": "user_123",
    "brand_name": "VinFast",
    "brand_keywords": ["VinFast", "VF3", "VF8"],
    "competitor_names": ["Toyota", "Honda"],
    "competitor_keywords_map": {
      "Toyota": ["Toyota", "Vios", "Camry"],
      "Honda": ["Honda", "City", "Civic"]
    },
    "date_range": {
      "from": "2025-01-01",
      "to": "2025-02-01"
    }
  }
}
```

---

## 4. Redis State Management (Đã implement)

### 4.1. Database Selection

- **DB 0:** Job mapping, Pub/Sub (mainRedisClient)
- **DB 1:** Project state tracking (stateRedisClient)

### 4.2. Key Schema

```
smap:proj:{projectID}
```

### 4.3. Hash Fields

| Field    | Type   | Description                                      | Writer            |
| :------- | :----- | :----------------------------------------------- | :---------------- |
| `status` | String | INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED | Project/Collector |
| `total`  | Int    | Tổng số items cần xử lý                          | Collector         |
| `done`   | Int    | Số items đã xong                                 | Collector         |
| `errors` | Int    | Số items bị lỗi                                  | Collector         |

### 4.4. Status Constants (Đã define)

```go
// internal/model/state.go

type ProjectStatus string

const (
    ProjectStatusInitializing ProjectStatus = "INITIALIZING"
    ProjectStatusCrawling     ProjectStatus = "CRAWLING"
    ProjectStatusProcessing   ProjectStatus = "PROCESSING"
    ProjectStatusDone         ProjectStatus = "DONE"
    ProjectStatusFailed       ProjectStatus = "FAILED"
)

type ProjectState struct {
    Status ProjectStatus `json:"status"`
    Total  int64         `json:"total"`
    Done   int64         `json:"done"`
    Errors int64         `json:"errors"`
}
```

### 4.5. State Initialization (Project Service)

```go
// Khi user gọi POST /projects/:id/execute
// Project Service khởi tạo state với TTL 7 ngày

key := fmt.Sprintf("smap:proj:%s", projectID)
pipe := redis.Pipeline()
pipe.HSet(key, "status", "INITIALIZING")
pipe.HSet(key, "total", "0")
pipe.HSet(key, "done", "0")
pipe.HSet(key, "errors", "0")
pipe.Expire(key, 7 * 24 * time.Hour)  // 7 days TTL
pipe.Exec()
```

---

## 5. Progress Webhook (Đã implement)

### 5.1. Endpoint

```
POST /internal/progress/callback
Header: X-Internal-Key: {internal_key}
```

### 5.2. Request Schema

```go
// internal/webhook/type.go

type ProgressCallbackRequest struct {
    ProjectID string `json:"project_id" binding:"required"`
    UserID    string `json:"user_id" binding:"required"`
    Status    string `json:"status" binding:"required"`
    Total     int64  `json:"total"`
    Done      int64  `json:"done"`
    Errors    int64  `json:"errors"`
}
```

**Example Request:**

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "CRAWLING",
  "total": 1000,
  "done": 150,
  "errors": 2
}
```

### 5.3. Project Service Handler

```go
// internal/webhook/usecase/webhook.go

func (uc *usecase) HandleProgressCallback(ctx context.Context, req ProgressCallbackRequest) error {
    channel := fmt.Sprintf("user_noti:%s", req.UserID)

    // Calculate progress
    var progressPercent float64
    if req.Total > 0 {
        progressPercent = float64(req.Done) / float64(req.Total) * 100
    }

    // Determine message type
    messageType := MessageTypeProjectProgress
    status := model.ProjectStatus(req.Status)
    if status == model.ProjectStatusDone || status == model.ProjectStatusFailed {
        messageType = MessageTypeProjectCompleted
    }

    // Publish to Redis Pub/Sub
    message := map[string]interface{}{
        "type": messageType,
        "payload": map[string]interface{}{
            "project_id":       req.ProjectID,
            "status":           req.Status,
            "total":            req.Total,
            "done":             req.Done,
            "errors":           req.Errors,
            "progress_percent": progressPercent,
        },
    }

    return uc.redisClient.Publish(ctx, channel, message)
}
```

---

## 6. WebSocket Message Types (Đã define)

### 6.1. Constants

```go
// internal/webhook/type.go

const (
    MessageTypeProjectProgress  = "project_progress"
    MessageTypeProjectCompleted = "project_completed"
    MessageTypeDryRunResult     = "dryrun_result"
)
```

### 6.2. Message Format

**Progress Update:**

```json
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_xyz",
    "status": "CRAWLING",
    "total": 1000,
    "done": 150,
    "errors": 2,
    "progress_percent": 15.0
  }
}
```

**Completion:**

```json
{
  "type": "project_completed",
  "payload": {
    "project_id": "proj_xyz",
    "status": "DONE",
    "total": 1000,
    "done": 1000,
    "errors": 5,
    "progress_percent": 100.0
  }
}
```

### 6.3. Redis Pub/Sub Channel

```
user_noti:{user_id}
```

---

## 7. Collector Service - Implementation Guide

### 7.1. Consume `project.created` Event

```python
# Exchange: smap.events
# Routing Key: project.created
# Queue: collector.project.created (tự đặt tên)

def handle_project_created(event):
    payload = event['payload']

    project_id = payload['project_id']
    user_id = payload['user_id']  # LƯU LẠI để gọi webhook
    brand_keywords = payload['brand_keywords']
    competitor_keywords_map = payload['competitor_keywords_map']
    date_range = payload['date_range']

    # Store mapping project_id -> user_id
    store_project_user_mapping(project_id, user_id)

    # Dispatch crawlers
    dispatch_crawlers(project_id, brand_keywords, competitor_keywords_map, date_range)
```

### 7.2. Update Redis State

```python
def update_project_state(project_id, field, value):
    key = f"smap:proj:{project_id}"

    if field == "total":
        redis.hset(key, "total", value)
        redis.hset(key, "status", "CRAWLING")
    elif field == "done":
        redis.hincrby(key, "done", 1)
    elif field == "errors":
        redis.hincrby(key, "errors", 1)
    elif field == "status":
        redis.hset(key, "status", value)
```

### 7.3. Call Progress Webhook

```python
PROJECT_SERVICE_URL = "http://project-service:8080"
INTERNAL_KEY = os.getenv("PROJECT_INTERNAL_KEY")

def notify_progress(project_id, user_id):
    # Get current state from Redis
    key = f"smap:proj:{project_id}"
    state = redis.hgetall(key)

    # Call Project Service webhook
    response = requests.post(
        f"{PROJECT_SERVICE_URL}/internal/progress/callback",
        headers={"X-Internal-Key": INTERNAL_KEY},
        json={
            "project_id": project_id,
            "user_id": user_id,
            "status": state["status"],
            "total": int(state["total"]),
            "done": int(state["done"]),
            "errors": int(state["errors"])
        }
    )
    return response.status_code == 200
```

### 7.4. Throttling (Khuyến nghị)

```python
# Không gọi webhook mỗi item - throttle tối thiểu 5 giây
last_notify_time = {}
THROTTLE_INTERVAL = 5  # seconds

def should_notify(project_id):
    now = time.time()
    last = last_notify_time.get(project_id, 0)
    if now - last > THROTTLE_INTERVAL:
        last_notify_time[project_id] = now
        return True
    return False

def on_item_crawled(project_id, user_id, is_error=False):
    # Always update Redis
    if is_error:
        update_project_state(project_id, "errors", 1)
    update_project_state(project_id, "done", 1)

    # Throttle webhook calls
    if should_notify(project_id):
        notify_progress(project_id, user_id)
```

### 7.5. When to Call Webhook

| Event             | Redis Update                                 | Webhook Call |
| ----------------- | -------------------------------------------- | ------------ |
| Found total items | `HSET total {count}`, `HSET status CRAWLING` | Immediately  |
| Crawled 1 item    | `HINCRBY done 1`                             | Throttled    |
| Item failed       | `HINCRBY errors 1`                           | Throttled    |
| All done          | `HSET status DONE`                           | Immediately  |
| Fatal error       | `HSET status FAILED`                         | Immediately  |

---

## 8. Clean Architecture (Quy ước)

### 8.1. Layer Structure

```
Project UseCase Layer (Orchestration)
  - Orchestrate flow: PostgreSQL -> Redis State -> RabbitMQ
  - Quyết định WHEN gọi state operations
       |
       v
State UseCase Layer (Business Logic)
  - Completion check: done >= total && total > 0
  - Status transitions: INITIALIZING -> CRAWLING -> PROCESSING -> DONE
  - Progress calculation
       |
       v
State Repository Layer (Data Access ONLY)
  - CHỈ chứa Redis CRUD operations
  - Biết về key schema: smap:proj:{id}
  - KHÔNG chứa business logic
       |
       v
pkg/redis (Infrastructure Layer)
  - CHỈ chứa Redis connection logic
  - Generic operations: HSet, HGet, HIncrBy, Pipeline
```

### 8.2. Domain Types Location

```
internal/model/state.go
├── ProjectStatus (type + constants)
└── ProjectState (struct)
```

---

## 9. API Endpoints Summary

### 9.1. Public Endpoints (Cookie Auth)

| Method | Endpoint                 | Description        |
| ------ | ------------------------ | ------------------ |
| POST   | `/projects`              | Create project     |
| POST   | `/projects/:id/execute`  | Start execution    |
| GET    | `/projects/:id/progress` | Get progress       |
| POST   | `/projects/dryrun`       | Start dry-run test |

### 9.2. Internal Endpoints (X-Internal-Key)

| Method | Endpoint                      | Description      |
| ------ | ----------------------------- | ---------------- |
| POST   | `/internal/dryrun/callback`   | Dry-run webhook  |
| POST   | `/internal/progress/callback` | Progress webhook |

---

## 10. Error Codes

| Code  | Message                   | HTTP Status |
| ----- | ------------------------- | ----------- |
| 30004 | Project not found         | 404         |
| 30005 | Unauthorized              | 403         |
| 30007 | Invalid date range        | 400         |
| 30008 | Project already executing | 409         |

---

## 11. Configuration

### 11.1. Project Service

```env
# Redis
REDIS_HOST=localhost:6379
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Internal API
INTERNAL_KEY=your-internal-key
```

### 11.2. Collector Service (Cần config)

```env
# Project Service
PROJECT_SERVICE_URL=http://project-service:8080
PROJECT_INTERNAL_KEY=your-internal-key

# Redis (same instance as Project Service)
REDIS_HOST=localhost:6379
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

---

## 12. Checklist cho Collector Service

### 12.1. Event Consumption

- [ ] Declare queue `collector.project.created`
- [ ] Bind to exchange `smap.events` với routing key `project.created`
- [ ] Parse `ProjectCreatedEvent` schema
- [ ] Extract và lưu `user_id` từ payload

### 12.2. Redis State Updates

- [ ] Connect to Redis DB 1
- [ ] Implement `update_project_state()` function
- [ ] Update `total` khi biết tổng số items
- [ ] Update `done` sau mỗi item (HINCRBY)
- [ ] Update `errors` khi có lỗi (HINCRBY)
- [ ] Update `status` khi hoàn thành/lỗi

### 12.3. Progress Webhook

- [ ] Implement `notify_progress()` function
- [ ] Add throttling (5 giây minimum)
- [ ] Call webhook khi:
  - Set total items
  - Status change (CRAWLING, DONE, FAILED)
  - Periodically during crawling (throttled)

### 12.4. Error Handling

- [ ] Retry webhook calls với exponential backoff
- [ ] Log errors với project_id, user_id
- [ ] Vẫn update Redis state dù webhook fail

---

## 13. Sequence Diagram

```
Client          Project         Collector       Redis           WebSocket
  |                |                |              |                |
  | POST /execute  |                |              |                |
  |--------------->|                |              |                |
  |                |                |              |                |
  |                | Init state     |              |                |
  |                |------------------------------>|                |
  |                |                |              |                |
  |                | Publish project.created       |                |
  |                |--------------->|              |                |
  |                |                |              |                |
  |  200 OK        |                |              |                |
  |<---------------|                |              |                |
  |                |                |              |                |
  |                |                | Update state |                |
  |                |                |------------->|                |
  |                |                |              |                |
  |                | POST /progress/callback       |                |
  |                |<---------------|              |                |
  |                |                |              |                |
  |                | Publish user_noti:{user_id}   |                |
  |                |------------------------------>|                |
  |                |                |              |                |
  |                |                |              | Subscribe      |
  |                |                |              |<---------------|
  |                |                |              |                |
  |                |                |              | Message        |
  |                |                |              |--------------->|
  |                |                |              |                |
  | WebSocket message                              |                |
  |<---------------------------------------------------------------|
  |                |                |              |                |
```

---

## 14. Tổng kết

Với thiết kế này, hệ thống SMAP đạt được:

1. **Tách biệt Create/Execute:** User có thể test keywords trước khi chạy thật
2. **User tracking:** `user_id` trong event payload cho phép notify đúng user
3. **Real-time updates:** WebSocket qua Redis Pub/Sub
4. **Fallback polling:** API `/progress` cho trường hợp WebSocket fail
5. **Throttling:** Tránh spam webhook, tối ưu performance
6. **Clean Architecture:** Tách biệt rõ ràng giữa các layers
