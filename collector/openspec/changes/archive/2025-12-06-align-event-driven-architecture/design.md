# Design: Align Event-Driven Architecture

## Context

Collector Service hiện tại hoạt động như một dispatcher nhận `CrawlRequest` từ `collector.inbound` exchange và fan-out tới các worker queues (TikTok, YouTube). Tuy nhiên, document `event-drivent.md` định nghĩa một kiến trúc event-driven choreography chuẩn hóa cho toàn bộ hệ thống SMAP.

**Stakeholders:**
- Project Service (producer của `project.created`)
- Analytics Service (consumer của `data.collected`)
- Frontend (nhận progress updates qua WebSocket)

**Constraints:**
- Phải backward compatible trong giai đoạn migration
- Không được làm gián đoạn workers hiện tại (TikTok, YouTube scrapers)
- Redis state phải dùng DB 1 (DB 0 đã dùng cho job mapping)

## Goals / Non-Goals

### Goals
- Align với event-driven architecture document
- Support `ProjectCreatedEvent` schema mới
- Implement Redis state management cho progress tracking
- Implement progress webhook notification
- Publish `data.collected` event sau khi crawl xong

### Non-Goals
- Thay đổi worker queues (giữ nguyên `tiktok_exchange`, `youtube_exchange`)
- Implement WebSocket server (đã có ở Project Service)
- Thay đổi MinIO upload logic

## Decisions

### Decision 1: Dual Exchange Support
**What:** Collector sẽ consume từ cả `smap.events` (mới) và `collector.inbound` (cũ) trong giai đoạn migration.

**Why:** Đảm bảo backward compatibility, cho phép Project Service migrate dần sang schema mới.

**Alternatives considered:**
- Big bang migration: Rủi ro cao, downtime
- Chỉ support schema mới: Breaking change cho Project Service

### Decision 2: Schema Detection Strategy
**What:** Sử dụng field detection để phân biệt `ProjectCreatedEvent` và `CrawlRequest`.

**Why:** `ProjectCreatedEvent` có field `event_id` và `payload.project_id`, trong khi `CrawlRequest` có `job_id` và `task_type`.

```go
// Detection logic
if msg.EventID != "" && msg.Payload.ProjectID != "" {
    // ProjectCreatedEvent
} else if msg.JobID != "" && msg.TaskType != "" {
    // CrawlRequest (legacy)
}
```

### Decision 3: Redis State Module
**What:** Tạo module mới `internal/state/` với clean architecture.

**Why:** Tách biệt state management logic khỏi dispatcher, dễ test và maintain.

```
internal/state/
├── uc_interface.go    # StateUseCase interface
├── uc_types.go        # ProjectState, ProjectStatus
├── uc_errors.go       # State-specific errors
├── usecase/
│   ├── new.go
│   └── state_uc.go    # Business logic
└── repository/
    ├── repo_interface.go
    └── redis/
        └── state_repo.go  # Redis implementation
```

### Decision 4: Webhook Client Module
**What:** Tạo module mới `internal/webhook/` cho progress notification.

**Why:** Tách biệt webhook logic, support throttling và retry.

```go
type WebhookClient interface {
    NotifyProgress(ctx context.Context, req ProgressRequest) error
}

type ProgressRequest struct {
    ProjectID string
    UserID    string
    Status    string
    Total     int64
    Done      int64
    Errors    int64
}
```

### Decision 5: Throttling Implementation
**What:** Sử dụng in-memory map với mutex để track last notify time per project.

**Why:** Simple, không cần external dependency, đủ cho single instance.

```go
type throttler struct {
    mu       sync.Mutex
    lastCall map[string]time.Time
    interval time.Duration
}

func (t *throttler) ShouldNotify(projectID string) bool {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    last, ok := t.lastCall[projectID]
    if !ok || time.Since(last) > t.interval {
        t.lastCall[projectID] = time.Now()
        return true
    }
    return false
}
```

### Decision 6: Event Transformation
**What:** Transform `ProjectCreatedEvent` thành multiple `CrawlRequest` tasks.

**Why:** Giữ nguyên logic dispatcher hiện tại, chỉ thêm layer transformation.

```go
func TransformProjectEvent(event ProjectCreatedEvent) []CrawlRequest {
    var requests []CrawlRequest
    
    // Brand keywords
    for _, keyword := range event.Payload.BrandKeywords {
        requests = append(requests, CrawlRequest{
            JobID:    fmt.Sprintf("%s-brand-%s", event.Payload.ProjectID, keyword),
            TaskType: TaskTypeResearchAndCrawl,
            Payload: map[string]any{
                "keywords": []string{keyword},
                // ...
            },
        })
    }
    
    // Competitor keywords
    for competitor, keywords := range event.Payload.CompetitorKeywordsMap {
        // ...
    }
    
    return requests
}
```

## Risks / Trade-offs

### Risk 1: Dual schema complexity
**Risk:** Maintaining two schemas increases code complexity.
**Mitigation:** Set deprecation timeline (3 months), log warnings for legacy schema.

### Risk 2: Redis connection failure
**Risk:** State updates fail if Redis unavailable.
**Mitigation:** Log errors, continue processing (state is non-critical for crawling).

### Risk 3: Webhook endpoint unavailable
**Risk:** Progress notifications lost if Project Service down.
**Mitigation:** Retry with exponential backoff, log failures, Redis state still updated.

### Risk 4: Throttling memory leak
**Risk:** In-memory throttle map grows unbounded.
**Mitigation:** Add TTL cleanup goroutine, remove entries older than 1 hour.

## Migration Plan

### Phase 1: Add New Infrastructure (Week 1)
1. Add Redis state module
2. Add webhook client module
3. Add new event types
4. Add `smap.events` consumer (parallel to existing)

### Phase 2: Enable New Flow (Week 2)
1. Deploy Collector with dual support
2. Project Service starts publishing to `smap.events`
3. Monitor both flows

### Phase 3: Deprecate Legacy (Week 3-4)
1. Log deprecation warnings for `CrawlRequest`
2. Notify dependent services
3. Set deadline for migration

### Phase 4: Remove Legacy (Month 2)
1. Remove `collector.inbound` consumer
2. Remove `CrawlRequest` handling
3. Clean up code

### Rollback
- Revert to previous Collector version
- Project Service continues using `collector.inbound`
- No data loss (messages remain in queues)

## Open Questions

1. **Q:** Should we support partial migration (some projects use new schema, some use old)?
   **A:** Yes, schema detection handles this automatically.

2. **Q:** How to handle in-flight messages during deployment?
   **A:** RabbitMQ queues persist messages, new Collector picks up where old left off.

3. **Q:** Should throttle interval be configurable per project?
   **A:** No, global 5-second interval is sufficient for MVP.
