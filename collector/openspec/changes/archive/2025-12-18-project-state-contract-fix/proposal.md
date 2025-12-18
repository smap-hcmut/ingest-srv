# Change: Project State Contract Fix

## Change ID

`project-state-contract-fix`

## Status

✅ **Completed** - Archived 2025-12-18

## Why

Các vấn đề chính cần giải quyết:

1. **Message Contract chưa được xác định rõ ràng** - Struct giữa các service chưa được define chính xác
2. **Collector chưa handle đúng case platform limitation** - Khi crawler trả về ít bài hơn limit
3. **Collector chưa tính toán tổng analyze result** - Chưa xử lý đúng việc tính completion

### Current Issues

```
Crawler → Collector:
- CrawlerResult thiếu limit_info, stats
- Collector assumes all items succeeded when success=true

Analytics → Collector:
- Cần ensure Analytics sends correct format
- Collector cần properly aggregate success_count/error_count
```

## What Changes

### 1. FLAT Format v3.0 cho Crawler → Collector

```go
type CrawlerResultMessage struct {
    Success         bool    `json:"success"`
    TaskType        string  `json:"task_type"`
    JobID           string  `json:"job_id"`
    Platform        string  `json:"platform"`
    RequestedLimit  int     `json:"requested_limit"`
    AppliedLimit    int     `json:"applied_limit"`
    TotalFound      int     `json:"total_found"`
    PlatformLimited bool    `json:"platform_limited"`
    Successful      int     `json:"successful"`
    Failed          int     `json:"failed"`
    Skipped         int     `json:"skipped"`
    ErrorCode       *string `json:"error_code,omitempty"`
    ErrorMessage    *string `json:"error_message,omitempty"`
}
```

**Note:** No payload - Crawler pushes content directly to Analytics.

### 2. FLAT Format cho Analytics → Collector

```go
type AnalyzeResultPayload struct {
    ProjectID    string `json:"project_id"`
    JobID        string `json:"job_id"`
    TaskType     string `json:"task_type"`     // "analyze_result"
    BatchSize    int    `json:"batch_size"`
    SuccessCount int    `json:"success_count"`
    ErrorCount   int    `json:"error_count"`
}
```

### 3. Two-Phase Completion Logic

- Crawl complete: `tasks_done + tasks_errors >= tasks_total`
- Analyze complete: `analyze_done + analyze_errors >= analyze_total`
- Project complete: Both phases complete

## Impact

### Affected Specs

- `crawl-limits-config` - State tracking requirements
- `event-infrastructure` - State management

### Affected Code

| File                                     | Change                         |
| ---------------------------------------- | ------------------------------ |
| `internal/models/crawler_result.go`      | CrawlerResultMessage struct    |
| `internal/results/usecase/result.go`     | handleProjectResult refactored |
| `internal/models/event.go`               | Completion logic               |
| `document/collector-crawler-contract.md` | Contract v3.0                  |

## Reference Documents

- `document/collector-crawler-contract.md` - Contract specification v3.0
- `document/collector-behavior.md` - Behavior specification
