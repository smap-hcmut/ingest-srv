# Design: Two-Phase State Structure

## Context

Collector Service hiện tại track progress của crawl phase thông qua Redis state với fields `total`, `done`, `errors`. Tuy nhiên, pipeline thực tế có 2 phases:

1. **Crawl Phase**: Crawler thu thập data từ platforms, upload lên MinIO
2. **Analyze Phase**: Analytics Service fetch data từ MinIO và phân tích

Crawler và Analytics hoạt động **song song** (không sequential):

- Crawler crawl batch 1 → Analytics analyze batch 1
- Crawler crawl batch 2 → Analytics analyze batch 2
- ...

**Stakeholders:**

- Project Service (nhận webhook, notify user)
- Analytics Service (publish analyze results)
- Frontend (hiển thị progress)

**Constraints:**

- Collector là single point cho state management (không để Analytics gọi webhook trực tiếp)
- Batch processing (không increment +1 mà +N)
- Redis HINCRBY là atomic (không cần lock)

## Goals / Non-Goals

### Goals

- Track cả crawl và analyze progress trong cùng state
- Support batch increment operations
- Simplify status model (single PROCESSING status)
- Collector là central hub cho progress tracking
- Accurate completion detection (cả 2 phases phải xong)

### Non-Goals

- Thay đổi flow của Crawler (vẫn upload MinIO, publish event)
- Thay đổi flow của Analytics (vẫn fetch từ MinIO)
- Implement retry logic cho Analytics
- Implement timeout detection

## Decisions

### Decision 1: Parallel Counters với Single Status

**What:** Sử dụng 6 counters riêng biệt (3 cho crawl, 3 cho analyze) nhưng chỉ 1 status field.

**Why:**

- Crawl và analyze chạy song song, không sequential
- Single status đơn giản hóa state machine
- Counters độc lập cho phép track chính xác từng phase

**Status values:**

- `INITIALIZING` - Vừa khởi tạo, chưa có task
- `PROCESSING` - Đang xử lý (crawl + analyze)
- `DONE` - Cả 2 phases hoàn thành
- `FAILED` - Lỗi nghiêm trọng

```go
const (
    ProjectStatusInitializing ProjectStatus = "INITIALIZING"
    ProjectStatusProcessing   ProjectStatus = "PROCESSING"
    ProjectStatusDone         ProjectStatus = "DONE"
    ProjectStatusFailed       ProjectStatus = "FAILED"
)
```

### Decision 2: Analyze Total Auto-Set

**What:** Collector tự động set `analyze_total` khi nhận crawl result thành công.

**Why:**

- Crawler biết số items trong batch
- Collector nhận payload với items
- Analytics chỉ cần gửi `success_count` và `error_count`

```go
func (uc implUseCase) handleProjectResult(ctx, res) error {
    itemCount := int64(len(res.Payload))

    if res.Success {
        uc.stateUC.IncrementCrawlDoneBy(ctx, projectID, itemCount)
        uc.stateUC.IncrementAnalyzeTotalBy(ctx, projectID, itemCount)  // Auto-set
    } else {
        uc.stateUC.IncrementCrawlErrorsBy(ctx, projectID, itemCount)
    }
}
```

### Decision 3: Batch Increment Methods

**What:** Thay `IncrementDone()` (+1) bằng `IncrementDoneBy(count)` (+N).

**Why:**

- Crawler và Analytics xử lý theo batch (50-100 items)
- Giảm số lần gọi Redis
- Atomic operation với HINCRBY

```go
type UseCase interface {
    // Crawl operations
    SetCrawlTotal(ctx, projectID string, total int64) error
    IncrementCrawlDoneBy(ctx, projectID string, count int64) error
    IncrementCrawlErrorsBy(ctx, projectID string, count int64) error

    // Analyze operations
    IncrementAnalyzeTotalBy(ctx, projectID string, count int64) error
    IncrementAnalyzeDoneBy(ctx, projectID string, count int64) error
    IncrementAnalyzeErrorsBy(ctx, projectID string, count int64) error
}
```

### Decision 4: Completion Condition

**What:** Project complete khi CẢ HAI phases đều complete.

**Why:** User chỉ nên được notify khi có kết quả phân tích để xem.

```go
func (s *ProjectState) IsComplete() bool {
    crawlComplete := s.CrawlTotal > 0 &&
                     (s.CrawlDone + s.CrawlErrors) >= s.CrawlTotal

    analyzeComplete := s.AnalyzeTotal > 0 &&
                       (s.AnalyzeDone + s.AnalyzeErrors) >= s.AnalyzeTotal

    return crawlComplete && analyzeComplete
}
```

### Decision 5: Analyze Result Message Format

**What:** Analytics publish message với `task_type: "analyze_result"` và batch counts.

**Why:**

- Reuse existing results consumer infrastructure
- Task type routing đã có sẵn
- Batch counts cho phép increment by N

```json
{
  "success": true,
  "payload": {
    "project_id": "proj_xyz",
    "job_id": "proj_xyz-brand-0",
    "task_type": "analyze_result",
    "batch_size": 50,
    "success_count": 48,
    "error_count": 2
  }
}
```

### Decision 6: Webhook Format Update

**What:** Webhook payload bao gồm cả crawl và analyze progress.

**Why:**

- Project Service cần hiển thị progress của cả 2 phases
- Overall progress = (crawl + analyze) / 2

```go
type ProgressRequest struct {
    ProjectID              string        `json:"project_id"`
    UserID                 string        `json:"user_id"`
    Status                 string        `json:"status"`
    Crawl                  PhaseProgress `json:"crawl"`
    Analyze                PhaseProgress `json:"analyze"`
    OverallProgressPercent float64       `json:"overall_progress_percent"`
}

type PhaseProgress struct {
    Total           int64   `json:"total"`
    Done            int64   `json:"done"`
    Errors          int64   `json:"errors"`
    ProgressPercent float64 `json:"progress_percent"`
}
```

## Risks / Trade-offs

### Risk 1: Breaking webhook format

**Risk:** Project Service phải update handler cho format mới.
**Mitigation:** Coordinate deployment, update Project Service trước.

### Risk 2: Existing state keys incompatible

**Risk:** Redis keys với field names cũ sẽ không đọc được.
**Mitigation:** TTL 7 ngày, keys cũ sẽ tự expire. Hoặc migration script.

### Risk 3: Analytics Service chưa ready

**Risk:** Nếu Analytics chưa publish `analyze.result`, analyze counters sẽ không update.
**Mitigation:** Graceful degradation - crawl vẫn hoạt động, analyze counters = 0.

### Risk 4: Race condition

**Risk:** Concurrent updates từ multiple crawl/analyze results.
**Mitigation:** Redis HINCRBY là atomic, không cần lock.

## Migration Plan

### Phase 1: Update Collector (Week 1)

1. Update `internal/models/event.go` - struct + methods
2. Update `internal/state/` - interface + implementation
3. Update `internal/webhook/` - types + usecase
4. Update `internal/results/` - add analyze handler
5. Deploy Collector

### Phase 2: Update Analytics (Week 2)

1. Analytics publish `analyze.result` messages
2. Deploy Analytics

### Phase 3: Update Project Service (Week 2)

1. Update webhook handler for new format
2. Update GET /progress response
3. Deploy Project Service

### Phase 4: Cleanup (Week 3)

1. Remove deprecated code
2. Update documentation

### Rollback

- Revert Collector deployment
- Redis keys cũ vẫn còn (TTL 7 ngày)
- Analytics/Project Service không bị ảnh hưởng

## Open Questions

1. **Q:** Analytics có retry không? Nếu có thì track như thế nào?
   **A:** TBD - Có thể thêm `retry_count` field sau.

2. **Q:** Timeout detection - nếu Analytics không respond trong X phút?
   **A:** TBD - Có thể implement background job check sau.

3. **Q:** Partial failure - nếu crawl 100, chỉ 80 thành công?
   **A:** `analyze_total = 80` (chỉ count successful crawls).
