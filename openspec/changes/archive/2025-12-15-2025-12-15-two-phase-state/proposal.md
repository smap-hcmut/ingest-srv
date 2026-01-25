# Change: Two-Phase State Structure for Crawl + Analyze Pipeline

## Change ID

`two-phase-state`

## Status

🟡 **Draft** - Pending review and implementation

## Why

Hiện tại Collector Service chỉ track **crawl progress**, không track **analyze progress**. Điều này gây ra các vấn đề:

1. **Không biết khi nào phân tích xong**: State hiện tại chỉ có `total`, `done`, `errors` cho crawl phase. Khi crawl xong (`status=DONE`), thực tế data vẫn đang được Analytics Service phân tích.

2. **User nhận thông báo sai thời điểm**: Project Service notify user khi `status=DONE`, nhưng lúc đó chưa có kết quả phân tích để xem.

3. **Flow thực tế không được reflect**:

   ```
   Crawler → (upload MinIO) → Analytics → (analyze) → Done
                ↓
           Collector (chỉ track đến đây)
   ```

4. **Batch processing không được support**: Crawler và Analytics xử lý theo batch (50-100 items), nhưng state chỉ có `IncrementDone()` (+1).

## What Changes

### State Structure Refactor

**Current:**

```go
type ProjectState struct {
    Status ProjectStatus  // INITIALIZING, CRAWLING, DONE, FAILED
    Total  int64
    Done   int64
    Errors int64
}
```

**New:**

```go
type ProjectState struct {
    Status ProjectStatus  // INITIALIZING, PROCESSING, DONE, FAILED

    // Crawl counters
    CrawlTotal  int64
    CrawlDone   int64
    CrawlErrors int64

    // Analyze counters
    AnalyzeTotal  int64
    AnalyzeDone   int64
    AnalyzeErrors int64
}
```

### Status Simplification

- Remove `CRAWLING` status
- Use single `PROCESSING` status for both crawl and analyze phases
- Completion = crawl complete AND analyze complete

### Batch Increment Methods

**Current:**

```go
IncrementDone(ctx, projectID) error      // +1
IncrementErrors(ctx, projectID) error    // +1
```

**New:**

```go
IncrementCrawlDoneBy(ctx, projectID, count) error      // +N
IncrementCrawlErrorsBy(ctx, projectID, count) error    // +N
IncrementAnalyzeTotalBy(ctx, projectID, count) error   // +N
IncrementAnalyzeDoneBy(ctx, projectID, count) error    // +N
IncrementAnalyzeErrorsBy(ctx, projectID, count) error  // +N
```

### New Analyze Result Handler

Collector sẽ consume `analyze.result` messages từ Analytics Service:

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

### Updated Webhook Format

**Current:**

```json
{
  "project_id": "proj_xyz",
  "status": "CRAWLING",
  "total": 100,
  "done": 50,
  "errors": 2
}
```

**New:**

```json
{
  "project_id": "proj_xyz",
  "status": "PROCESSING",
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

## Impact

### Affected Specs

- `event-infrastructure` - Update state management requirements

### Affected Code

| File                                           | Change                                      |
| ---------------------------------------------- | ------------------------------------------- |
| `internal/models/event.go`                     | Update `ProjectState` struct, add methods   |
| `internal/state/interface.go`                  | Update interface with `IncrementBy` methods |
| `internal/state/types.go`                      | Update Redis field constants                |
| `internal/state/usecase/state.go`              | Implement new methods                       |
| `internal/state/repository/redis/redis.go`     | Update field names                          |
| `internal/webhook/types.go`                    | Update `ProgressRequest` struct             |
| `internal/webhook/usecase/webhook.go`          | Update progress building                    |
| `internal/results/types.go`                    | Add `AnalyzeResultPayload`                  |
| `internal/results/usecase/result.go`           | Add `handleAnalyzeResult` handler           |
| `internal/dispatcher/usecase/project_event.go` | Use `SetCrawlTotal`                         |

### External Dependencies

| Service           | Change Required                   |
| ----------------- | --------------------------------- |
| Analytics Service | Publish `analyze.result` messages |
| Project Service   | Handle new webhook format         |

### Breaking Changes

- Webhook format changes (Project Service must update handler)
- Redis field names change (existing state keys will be incompatible)

## Migration Path

1. **Phase 1 - Collector Update**: Deploy Collector với state structure mới
2. **Phase 2 - Analytics Integration**: Analytics Service publish `analyze.result` messages
3. **Phase 3 - Project Service Update**: Handle new webhook format, update UI

### Rollback Plan

- Revert Collector deployment
- Redis keys cũ sẽ tự expire (TTL 7 ngày)
- Analytics/Project Service không bị ảnh hưởng nếu chưa deploy

## Open Questions

1. **Retry logic**: Analytics có retry không? Nếu có thì track như thế nào?
2. **Timeout**: Nếu Analytics không respond trong X phút, có tự động FAILED không?
3. **Backward compatibility**: Có cần support cả format cũ và mới trong giai đoạn migration?

## Reference Documents

- `document/project-state-proposal.md` - Detailed proposal
- `document/project-state-changes.md` - Code diff details
- `document/integration-analytics-service.md` - Analytics integration guide
