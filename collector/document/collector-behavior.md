# Collector Service - Behavior Specification

**Cập nhật:** 2025-12-18 (Config-Driven Limits & Hybrid State)

---

## 1. Tổng quan

Collector Service là middleware giữa Project Service và Crawler Workers, chịu trách nhiệm:

1. **Nhận và phân phối task** từ Project Service tới Crawler Workers
2. **Quản lý trạng thái** thực thi project trong Redis (Hybrid State)
3. **Xử lý kết quả** từ Crawler/Analytics và gửi webhook về Project Service
4. **Phân biệt loại task** để xử lý khác biệt (dry-run vs project execution)

---

## 2. Kiến trúc

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

## 3. Queue Configuration

### 3.1. Dispatcher Consumer

| Queue                       | Exchange            | Routing Key       | Mục đích                          |
| --------------------------- | ------------------- | ----------------- | --------------------------------- |
| `collector.inbound.tasks`   | `collector.inbound` | `crawler.*`       | Nhận các task dry-run             |
| `collector.project.created` | `smap.events`       | `project.created` | Nhận sự kiện thực thi project mới |

### 3.2. Results Consumer

| Queue                  | Exchange          | Routing Key | Mục đích                |
| ---------------------- | ----------------- | ----------- | ----------------------- |
| `results.inbound.data` | `results.inbound` | `#`         | Nhận kết quả từ Crawler |

---

## 4. Task Types & Routing

### 4.1. Task Type Constants

```go
const (
    TaskTypeResearchAndCrawl = "research_and_crawl"
    TaskTypeDryRunKeyword    = "dryrun_keyword"
    TaskTypeAnalyzeResult    = "analyze_result"
)
```

### 4.2. Routing Logic

```go
func HandleResult(ctx context.Context, res models.CrawlerResult) error {
    switch res.TaskType {
    case "dryrun_keyword":
        return handleDryRunResult(ctx, res)      // → /internal/dryrun/callback
    case "research_and_crawl":
        return handleProjectResult(ctx, res)     // → Update Redis + /internal/progress/callback
    case "analyze_result":
        return handleAnalyzeResult(ctx, res)     // → Update Redis + /internal/progress/callback
    default:
        return handleDryRunResult(ctx, res)      // Backward compatibility
    }
}
```

### 4.3. Ma trận xử lý

| Task Type            | Handler                 | Webhook Endpoint              | Redis Update |
| -------------------- | ----------------------- | ----------------------------- | ------------ |
| `dryrun_keyword`     | `handleDryRunResult()`  | `/internal/dryrun/callback`   | Không        |
| `research_and_crawl` | `handleProjectResult()` | `/internal/progress/callback` | Có           |
| `analyze_result`     | `handleAnalyzeResult()` | `/internal/progress/callback` | Có           |

---

## 5. Config-Driven Limits

### 5.1. Configuration Structure

```go
type CrawlLimitsConfig struct {
    // Default limits
    DefaultLimitPerKeyword int `env:"DEFAULT_LIMIT_PER_KEYWORD" envDefault:"50"`
    DefaultMaxComments     int `env:"DEFAULT_MAX_COMMENTS" envDefault:"100"`
    DefaultMaxAttempts     int `env:"DEFAULT_MAX_ATTEMPTS" envDefault:"3"`

    // Dry-run limits
    DryRunLimitPerKeyword int `env:"DRYRUN_LIMIT_PER_KEYWORD" envDefault:"3"`
    DryRunMaxComments     int `env:"DRYRUN_MAX_COMMENTS" envDefault:"5"`

    // Feature flags
    IncludeComments bool `env:"INCLUDE_COMMENTS" envDefault:"true"`
}
```

### 5.2. Environment Variables

```env
DEFAULT_LIMIT_PER_KEYWORD=50
DEFAULT_MAX_COMMENTS=100
DEFAULT_MAX_ATTEMPTS=3
DRYRUN_LIMIT_PER_KEYWORD=3
DRYRUN_MAX_COMMENTS=5
INCLUDE_COMMENTS=true
```

---

## 6. Redis State Management (Hybrid State)

### 6.1. Key Schema

```
smap:proj:{projectID}    # Project state (Hash)
smap:user:{projectID}    # User mapping (String)
```

### 6.2. State Fields

| Field            | Type   | Mô tả                                        |
| ---------------- | ------ | -------------------------------------------- |
| `status`         | String | INITIALIZING, PROCESSING, DONE, FAILED       |
| `tasks_total`    | Int64  | Tổng số tasks (keywords × platforms)         |
| `tasks_done`     | Int64  | Số tasks hoàn thành                          |
| `tasks_errors`   | Int64  | Số tasks failed                              |
| `items_expected` | Int64  | Số items dự kiến (tasks × limit_per_keyword) |
| `items_actual`   | Int64  | Số items crawl thành công                    |
| `items_errors`   | Int64  | Số items crawl thất bại                      |
| `analyze_total`  | Int64  | Tổng số items cần analyze                    |
| `analyze_done`   | Int64  | Số items analyze thành công                  |
| `analyze_errors` | Int64  | Số items analyze thất bại                    |

### 6.3. Hybrid State Pipeline

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           HYBRID STATE PIPELINE                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Phase 1: CRAWL                           Phase 2: ANALYZE                  │
│  ┌─────────────────────────────────┐      ┌─────────────────────────┐       │
│  │ Task-Level (completion check):  │      │ analyze_total: 450      │       │
│  │   tasks_total: 10               │      │ analyze_done: 200       │       │
│  │   tasks_done: 10                │ ───► │ analyze_errors: 5       │       │
│  │   tasks_errors: 0               │      └─────────────────────────┘       │
│  │                                 │                                        │
│  │ Item-Level (progress display):  │                                        │
│  │   items_expected: 500           │                                        │
│  │   items_actual: 450             │                                        │
│  │   items_errors: 50              │                                        │
│  └─────────────────────────────────┘                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.4. State Transitions

```
INITIALIZING → PROCESSING (khi SetTasksTotal)
PROCESSING → DONE (khi crawl complete AND analyze complete)
PROCESSING → FAILED (khi gặp lỗi không thể phục hồi)

Crawl complete: tasks_done + tasks_errors >= tasks_total
Analyze complete: analyze_done + analyze_errors >= analyze_total
```

### 6.5. Completion Logic

```go
func (s *ProjectState) IsCrawlComplete() bool {
    return s.TasksTotal > 0 && (s.TasksDone + s.TasksErrors) >= s.TasksTotal
}

func (s *ProjectState) IsAnalyzeComplete() bool {
    return s.AnalyzeTotal > 0 && (s.AnalyzeDone + s.AnalyzeErrors) >= s.AnalyzeTotal
}

func (s *ProjectState) IsComplete() bool {
    return s.IsCrawlComplete() && s.IsAnalyzeComplete()
}
```

---

## 7. Message Processing

### 7.1. Crawl Result Processing (FLAT Format v3.0)

```go
func handleProjectResult(ctx context.Context, res CrawlerResultMessage) error {
    // 1. Extract project_id từ job_id
    projectID := extractProjectID(res.JobID)

    // 2. Update task-level counter
    if res.Success {
        stateUC.IncrementTasksDone(ctx, projectID)
    } else {
        stateUC.IncrementTasksErrors(ctx, projectID)
    }

    // 3. Update item-level counters
    if res.Successful > 0 {
        stateUC.IncrementItemsActualBy(ctx, projectID, res.Successful)
        stateUC.IncrementAnalyzeTotalBy(ctx, projectID, res.Successful)
    }
    if res.Failed > 0 {
        stateUC.IncrementItemsErrorsBy(ctx, projectID, res.Failed)
    }

    // 4. Send progress webhook
    state, _ := stateUC.GetState(ctx, projectID)
    userID, _ := stateUC.GetUserID(ctx, projectID)
    webhookUC.NotifyProgress(ctx, buildProgressRequest(projectID, userID, state))

    // 5. Check completion
    if completed, _ := stateUC.CheckCompletion(ctx, projectID); completed {
        webhookUC.NotifyCompletion(ctx, buildProgressRequest(projectID, userID, state))
    }

    return nil
}
```

### 7.2. Analyze Result Processing

```go
func handleAnalyzeResult(ctx context.Context, res AnalyzeResultMessage) error {
    projectID := res.ProjectID

    // Update analyze counters
    if res.SuccessCount > 0 {
        stateUC.IncrementAnalyzeDoneBy(ctx, projectID, res.SuccessCount)
    }
    if res.ErrorCount > 0 {
        stateUC.IncrementAnalyzeErrorsBy(ctx, projectID, res.ErrorCount)
    }

    // Send progress và check completion
    state, _ := stateUC.GetState(ctx, projectID)
    userID, _ := stateUC.GetUserID(ctx, projectID)
    webhookUC.NotifyProgress(ctx, buildProgressRequest(projectID, userID, state))

    if completed, _ := stateUC.CheckCompletion(ctx, projectID); completed {
        webhookUC.NotifyCompletion(ctx, buildProgressRequest(projectID, userID, state))
    }

    return nil
}
```

### 7.3. Job ID Format

| Loại       | Format                             | Ví dụ               |
| ---------- | ---------------------------------- | ------------------- |
| Brand      | `{projectID}-brand-{index}`        | `proj_abc-brand-0`  |
| Competitor | `{projectID}-{competitor}-{index}` | `proj_abc-toyota-0` |
| Dry-run    | `{uuid}`                           | `550e8400-e29b-...` |

---

## 8. Webhook Integration

### 8.1. Progress Webhook

```
POST /internal/progress/callback
Header: X-Internal-Key: {internal_key}
```

### 8.2. Progress Request Format

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "PROCESSING",
  "tasks": {
    "total": 10,
    "done": 8,
    "errors": 0,
    "percent": 80.0
  },
  "items": {
    "expected": 500,
    "actual": 400,
    "errors": 20,
    "percent": 84.0
  },
  "analyze": {
    "total": 400,
    "done": 200,
    "errors": 5,
    "progress_percent": 51.25
  },
  "overall_progress_percent": 67.625
}
```

### 8.3. Dry-Run Callback

```
POST /internal/dryrun/callback
Header: Authorization: {internal_key}
```

```json
{
  "job_id": "uuid",
  "status": "success",
  "platform": "tiktok",
  "payload": {
    "content": [...]
  }
}
```

---

## 9. Progress Calculation

```go
// Crawl Progress (item-level preferred)
func CrawlProgressPercent(state *ProjectState) float64 {
    if state.ItemsExpected > 0 {
        return float64(state.ItemsActual + state.ItemsErrors) / float64(state.ItemsExpected) * 100
    }
    if state.TasksTotal > 0 {
        return float64(state.TasksDone + state.TasksErrors) / float64(state.TasksTotal) * 100
    }
    return 0
}

// Analyze Progress
func AnalyzeProgressPercent(state *ProjectState) float64 {
    if state.AnalyzeTotal <= 0 {
        return 0
    }
    return float64(state.AnalyzeDone + state.AnalyzeErrors) / float64(state.AnalyzeTotal) * 100
}

// Overall Progress
func OverallProgressPercent(state *ProjectState) float64 {
    return (CrawlProgressPercent(state) + AnalyzeProgressPercent(state)) / 2
}
```

---

## 10. Event-Driven Flow

### 10.1. Project Execution Flow

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌───────┐     ┌─────────┐
│ Client  │     │ Project │     │Collector│     │ Redis │     │ Crawler │
└────┬────┘     └────┬────┘     └────┬────┘     └───┬───┘     └────┬────┘
     │               │               │              │              │
     │ POST /execute │               │              │              │
     │──────────────>│               │              │              │
     │               │               │              │              │
     │               │ Init state    │              │              │
     │               │──────────────────────────────>              │
     │               │               │              │              │
     │               │ Publish project.created      │              │
     │               │──────────────>│              │              │
     │               │               │              │              │
     │               │               │ Store user   │              │
     │               │               │ mapping      │              │
     │               │               │─────────────>│              │
     │               │               │              │              │
     │               │               │ Set tasks_total             │
     │               │               │─────────────>│              │
     │               │               │              │              │
     │               │               │ Dispatch tasks              │
     │               │               │─────────────────────────────>
     │               │               │              │              │
```

### 10.2. Result Processing Flow

```
┌─────────┐     ┌─────────┐     ┌───────┐     ┌─────────┐
│ Crawler │     │Collector│     │ Redis │     │ Project │
└────┬────┘     └────┬────┘     └───┬───┘     └────┬────┘
     │               │              │              │
     │ CrawlerResult │              │              │
     │──────────────>│              │              │
     │               │              │              │
     │               │ HINCRBY      │              │
     │               │ tasks_done   │              │
     │               │─────────────>│              │
     │               │              │              │
     │               │ HINCRBY      │              │
     │               │ items_actual │              │
     │               │─────────────>│              │
     │               │              │              │
     │               │ POST /progress/callback     │
     │               │─────────────────────────────>
     │               │              │              │
     │               │ Check completion            │
     │               │─────────────>│              │
     │               │              │              │
```

---

## 11. Configuration

```env
# Project Service
PROJECT_SERVICE_URL=http://project-service:8080
PROJECT_INTERNAL_KEY=your-internal-key

# Redis
REDIS_HOST=localhost:6379
REDIS_STATE_DB=1

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/

# Crawl Limits
DEFAULT_LIMIT_PER_KEYWORD=50
DEFAULT_MAX_COMMENTS=100
DRYRUN_LIMIT_PER_KEYWORD=3
DRYRUN_MAX_COMMENTS=5
```

---

## 12. State UseCase Interface

```go
type UseCase interface {
    // Init
    InitState(ctx context.Context, projectID string) error

    // Task-Level
    SetTasksTotal(ctx context.Context, projectID string, tasksTotal, itemsExpected int64) error
    IncrementTasksDone(ctx context.Context, projectID string) error
    IncrementTasksErrors(ctx context.Context, projectID string) error

    // Item-Level
    IncrementItemsActualBy(ctx context.Context, projectID string, count int64) error
    IncrementItemsErrorsBy(ctx context.Context, projectID string, count int64) error

    // Analyze Phase
    IncrementAnalyzeTotalBy(ctx context.Context, projectID string, count int64) error
    IncrementAnalyzeDoneBy(ctx context.Context, projectID string, count int64) error
    IncrementAnalyzeErrorsBy(ctx context.Context, projectID string, count int64) error

    // Status & State
    UpdateStatus(ctx context.Context, projectID string, status models.ProjectStatus) error
    GetState(ctx context.Context, projectID string) (*models.ProjectState, error)
    CheckCompletion(ctx context.Context, projectID string) (bool, error)

    // User Mapping
    StoreUserMapping(ctx context.Context, projectID, userID string) error
    GetUserID(ctx context.Context, projectID string) (string, error)
}
```
