# Design: Config-Driven Limits & Hybrid State Tracking

## Context

Collector Service hiện tại có 3 vấn đề chính:

1. **Hardcoded limits** - Không thể thay đổi runtime
2. **State tracking không đồng nhất** - Mix task-level và item-level
3. **Thiếu thông tin từ Crawler** - Không biết platform có bị limit không

**Stakeholders:**

- Crawler Services (YouTube, TikTok) - Cần update response format
- Project Service - Cần handle new webhook format
- DevOps - Cần update K8s ConfigMap

**Constraints:**

- Backward compatible với Crawler response cũ (fallback logic)
- Không downtime khi deploy
- Redis HINCRBY là atomic (không cần lock)

## Goals / Non-Goals

### Goals

- Config-driven limits (có thể thay đổi qua env vars)
- Hybrid state tracking (task-level + item-level)
- Parse enhanced Crawler response với fallback
- Accurate progress tracking và completion detection
- Platform limitation logging/alerting

### Non-Goals

- Thay đổi Crawler internal logic
- Implement retry logic cho Crawler
- Implement timeout detection
- Change message queue topology

## Decisions

### Decision 1: Config Structure

**What:** Tạo `CrawlLimitsConfig` struct với 10 env vars.

**Why:**

- Tách biệt config cho production vs dry-run
- Hard limits để prevent abuse
- Feature flags cho optional features

```go
type CrawlLimitsConfig struct {
    // Default limits (production)
    DefaultLimitPerKeyword int `env:"DEFAULT_LIMIT_PER_KEYWORD" envDefault:"50"`
    DefaultMaxComments     int `env:"DEFAULT_MAX_COMMENTS" envDefault:"100"`
    DefaultMaxAttempts     int `env:"DEFAULT_MAX_ATTEMPTS" envDefault:"3"`

    // Dry-run limits (testing)
    DryRunLimitPerKeyword int `env:"DRYRUN_LIMIT_PER_KEYWORD" envDefault:"3"`
    DryRunMaxComments     int `env:"DRYRUN_MAX_COMMENTS" envDefault:"5"`

    // Hard limits (safety caps)
    MaxLimitPerKeyword int `env:"MAX_LIMIT_PER_KEYWORD" envDefault:"500"`
    MaxMaxComments     int `env:"MAX_MAX_COMMENTS" envDefault:"1000"`

    // Feature flags
    IncludeComments bool `env:"INCLUDE_COMMENTS" envDefault:"true"`
    DownloadMedia   bool `env:"DOWNLOAD_MEDIA" envDefault:"false"`
}
```

### Decision 2: Hybrid State Structure

**What:** Track cả task-level và item-level trong cùng state.

**Why:**

- Task-level cho completion check (đơn giản, reliable)
- Item-level cho progress display (chính xác hơn)
- Không cần biết số items expected trước

```go
type ProjectState struct {
    Status ProjectStatus `json:"status"`

    // Task-level (for completion)
    TasksTotal  int64 `json:"tasks_total"`   // Số tasks dispatch
    TasksDone   int64 `json:"tasks_done"`    // Số tasks hoàn thành
    TasksErrors int64 `json:"tasks_errors"`  // Số tasks failed

    // Item-level (for progress)
    ItemsExpected int64 `json:"items_expected"` // tasks × limit
    ItemsActual   int64 `json:"items_actual"`   // Actual crawled
    ItemsErrors   int64 `json:"items_errors"`   // Failed items

    // Analyze phase (unchanged)
    AnalyzeTotal  int64 `json:"analyze_total"`
    AnalyzeDone   int64 `json:"analyze_done"`
    AnalyzeErrors int64 `json:"analyze_errors"`
}
```

### Decision 3: Completion Logic

**What:** Completion dựa trên TASK-level, không phải item-level.

**Why:**

- Task-level đơn giản và reliable
- Không bị ảnh hưởng bởi platform limitation
- Mỗi response = 1 task complete

```go
func (s *ProjectState) IsCrawlComplete() bool {
    return s.TasksTotal > 0 &&
           (s.TasksDone + s.TasksErrors) >= s.TasksTotal
}
```

### Decision 4: Progress Calculation

**What:** Progress display dựa trên ITEM-level với fallback.

**Why:**

- Item-level chính xác hơn cho user
- Fallback về task-level nếu items không tracked

```go
func (s *ProjectState) CrawlProgressPercent() float64 {
    if s.ItemsExpected > 0 {
        return float64(s.ItemsActual+s.ItemsErrors) / float64(s.ItemsExpected) * 100
    }
    // Fallback to task-level
    if s.TasksTotal > 0 {
        return float64(s.TasksDone+s.TasksErrors) / float64(s.TasksTotal) * 100
    }
    return 0
}
```

### Decision 5: Enhanced Response Parsing với Fallback

**What:** Parse enhanced response, fallback về counting payload items.

**Why:**

- Backward compatible với Crawler cũ
- Không cần coordinate deployment
- Graceful degradation

```go
func (uc implUseCase) extractLimitInfoAndStats(ctx, res) (*LimitInfo, *CrawlStats) {
    // Try parse enhanced format
    if enhanced.LimitInfo != nil && enhanced.Stats != nil {
        return enhanced.LimitInfo, enhanced.Stats
    }

    // Fallback: count payload items
    itemCount := len(res.Payload)
    return &LimitInfo{
        RequestedLimit:  50, // default
        TotalFound:      itemCount,
        PlatformLimited: false,
    }, &CrawlStats{
        Successful:     itemCount,
        Failed:         0,
        CompletionRate: 1.0,
    }
}
```

### Decision 6: State Update Flow

**What:** Update cả task-level và item-level trong mỗi response.

**Why:**

- Consistent state
- Atomic operations với Redis HINCRBY

```go
func (uc implUseCase) handleProjectResult(ctx, res) error {
    limitInfo, stats := uc.extractLimitInfoAndStats(ctx, res)

    // Task-level: always 1 per response
    if res.Success {
        uc.stateUC.IncrementTasksDone(ctx, projectID)
    } else {
        uc.stateUC.IncrementTasksErrors(ctx, projectID)
    }

    // Item-level: from stats
    uc.stateUC.IncrementItemsActualBy(ctx, projectID, stats.Successful)
    uc.stateUC.IncrementItemsErrorsBy(ctx, projectID, stats.Failed)

    // Analyze total: successful items
    uc.stateUC.IncrementAnalyzeTotalBy(ctx, projectID, stats.Successful)

    // Log platform limitation
    if limitInfo.PlatformLimited {
        uc.l.Warnf(ctx, "Platform limited: requested=%d, found=%d",
            limitInfo.RequestedLimit, limitInfo.TotalFound)
    }

    // Send webhook + check completion
    // ...
}
```

### Decision 7: Webhook Format Update

**What:** Webhook bao gồm cả `tasks` và `items` progress.

**Why:**

- Project Service có thể hiển thị cả 2 metrics
- Backward compatible (thêm fields, không xóa)

```go
type ProgressRequest struct {
    ProjectID string `json:"project_id"`
    UserID    string `json:"user_id"`
    Status    string `json:"status"`

    Tasks   TaskProgress  `json:"tasks"`
    Items   ItemProgress  `json:"items"`
    Analyze PhaseProgress `json:"analyze"`

    OverallProgressPercent float64 `json:"overall_progress_percent"`
}

type TaskProgress struct {
    Total   int64   `json:"total"`
    Done    int64   `json:"done"`
    Errors  int64   `json:"errors"`
    Percent float64 `json:"percent"`
}

type ItemProgress struct {
    Expected int64   `json:"expected"`
    Actual   int64   `json:"actual"`
    Errors   int64   `json:"errors"`
    Percent  float64 `json:"percent"`
}
```

### Decision 8: Config Injection

**What:** Inject `CrawlLimitsConfig` vào dispatcher usecase.

**Why:**

- Dependency injection pattern
- Testable (có thể mock config)
- Single source of truth

```go
type implUseCase struct {
    // ... existing fields ...
    crawlLimitsCfg config.CrawlLimitsConfig
}

func New(
    // ... existing params ...
    crawlLimitsCfg config.CrawlLimitsConfig,
) UseCase {
    return &implUseCase{
        // ...
        crawlLimitsCfg: crawlLimitsCfg,
    }
}
```

## Risks / Trade-offs

### Risk 1: Breaking webhook format

**Risk:** Project Service phải update handler.
**Mitigation:** Thêm fields mới, không xóa fields cũ. Coordinate deployment.

### Risk 2: Redis field names change

**Risk:** Existing state keys incompatible.
**Mitigation:** TTL 7 ngày, keys cũ tự expire. Hoặc migration script.

### Risk 3: Crawler chưa update response

**Risk:** Fallback logic không có đủ thông tin.
**Mitigation:** Fallback vẫn hoạt động, chỉ thiếu `platform_limited` info.

### Risk 4: Config không sync với K8s

**Risk:** Env vars không được set trong K8s.
**Mitigation:** Update ConfigMap cùng lúc với code deployment.

### Risk 5: Hard limits quá thấp/cao

**Risk:** Limits không phù hợp với use case.
**Mitigation:** Configurable qua env vars, có thể adjust runtime.

## Data Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           DISPATCH FLOW                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. Receive ProjectCreatedEvent                                              │
│     └── payload: { keywords: ["kw1", "kw2"], ... }                           │
│                                                                              │
│  2. Load config                                                              │
│     └── opts = NewTransformOptionsFromConfig(cfg.CrawlLimits)                │
│     └── opts.LimitPerKeyword = 50 (from config)                              │
│                                                                              │
│  3. Transform to CrawlRequests                                               │
│     └── 2 keywords × 2 platforms = 4 tasks                                   │
│                                                                              │
│  4. Set state                                                                │
│     └── tasks_total = 4                                                      │
│     └── items_expected = 4 × 50 = 200                                        │
│                                                                              │
│  5. Dispatch to queues                                                       │
│     └── 4 messages to RabbitMQ                                               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                           RESULT HANDLING FLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. Receive CrawlerResult                                                    │
│     └── { success: true, limit_info: {...}, stats: {...}, payload: [...] }  │
│                                                                              │
│  2. Extract limit_info and stats (with fallback)                             │
│     └── stats.successful = 30                                                │
│     └── stats.failed = 2                                                     │
│     └── limit_info.platform_limited = true                                   │
│                                                                              │
│  3. Update task-level state                                                  │
│     └── tasks_done += 1                                                      │
│                                                                              │
│  4. Update item-level state                                                  │
│     └── items_actual += 30                                                   │
│     └── items_errors += 2                                                    │
│     └── analyze_total += 30                                                  │
│                                                                              │
│  5. Log platform limitation                                                  │
│     └── WARN: Platform limited: requested=50, found=32                       │
│                                                                              │
│  6. Send progress webhook                                                    │
│     └── { tasks: {...}, items: {...}, analyze: {...} }                       │
│                                                                              │
│  7. Check completion                                                         │
│     └── if tasks_done + tasks_errors >= tasks_total → complete               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Migration Plan

### Phase 1: Config + Models (Day 1)

1. Add `CrawlLimitsConfig` to `config/config.go`
2. Update `template.env` with new env vars
3. Update `k8s/configmap.yaml`
4. Add enhanced response structs to `models/crawler_result.go`
5. Add config-based functions to `models/event_transform.go`
6. Update `ProjectState` in `models/event.go`

### Phase 2: State Layer (Day 1-2)

7. Update `state/types.go` with new field constants
8. Update `state/interface.go` with new methods
9. Implement new methods in `state/repository/`
10. Implement new methods in `state/usecase/`

### Phase 3: Dispatcher (Day 2)

11. Inject config into `dispatcher/usecase/new.go`
12. Use config in `dispatcher/usecase/project_event.go`
13. Use config for dry-run in `dispatcher/usecase/util.go`

### Phase 4: Results Handler (Day 2-3)

14. Update `results/usecase/result.go` with enhanced parsing + fallback
15. Update webhook building with hybrid state

### Phase 5: Testing (Day 3)

16. Unit tests for new functions
17. Integration tests with mock crawler response

### Deployment Order

1. **Collector** (with fallback logic)
2. **Crawler** (with enhanced response)
3. **Project Service** (with new webhook handler)

## Open Questions

1. **Q:** Bao lâu cần support fallback logic?
   **A:** Recommend 2 weeks sau khi Crawler deploy.

2. **Q:** Có cần alert khi `platform_limited: true`?
   **A:** Yes, log WARN level. Có thể thêm Discord alert sau.

3. **Q:** Hard limits có cần validate?
   **A:** Yes, trong `NewTransformOptionsFromConfig()` nên cap values.
