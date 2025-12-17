# Change: Config-Driven Limits & Hybrid State Tracking

## Change ID

`config-driven-limits`

## Status

🟡 **Draft** - Pending review and implementation

## Why

Hiện tại Collector Service có các vấn đề nghiêm trọng về configuration và state tracking:

### 1. Hardcoded Limit Values

Các giá trị limit đang được hardcode trong code, không thể thay đổi runtime:

```go
// internal/models/event_transform.go
func DefaultTransformOptions() TransformOptions {
    return TransformOptions{
        MaxAttempts:     3,    // ❌ HARDCODE
        LimitPerKeyword: 50,   // ❌ HARDCODE
        MaxComments:     100,  // ❌ HARDCODE
    }
}

// internal/dispatcher/usecase/util.go
payload.LimitPerKeyword = 3   // ❌ HARDCODE dry-run
payload.MaxComments = 5       // ❌ HARDCODE dry-run
```

**Impact:**

- Phải rebuild và redeploy để thay đổi limits
- Không sync với K8s ConfigMap
- Khó maintain khi giá trị nằm rải rác nhiều file

### 2. State Tracking Không Chính Xác

State hiện tại đang MIX 2 concepts khác nhau:

| Concept      | `crawl_total`    | `crawl_done` |
| ------------ | ---------------- | ------------ |
| **Hiện tại** | Số TASKS         | Số ITEMS     |
| **Vấn đề**   | Không đồng nhất! |

**Ví dụ:**

- Dispatch 4 tasks (2 keywords × 2 platforms)
- `crawl_total = 4` (tasks)
- Crawler trả về 30 items → `crawl_done += 30` (items)
- `crawl_done (30) > crawl_total (4)` → Logic sai!

### 3. Thiếu Thông Tin từ Crawler Response

Crawler response hiện tại không có:

- `limit_info.requested_limit` - Limit được request
- `limit_info.total_found` - Số items tìm được
- `limit_info.platform_limited` - Platform có bị giới hạn không
- `stats.successful/failed/skipped` - Statistics chi tiết

**Impact:**

- Không biết platform có bị limit không
- Không track được actual vs expected items
- Không có metrics để alerting

## What Changes

### 1. Config-Driven Limits

**Thêm `CrawlLimitsConfig` vào config:**

```go
type CrawlLimitsConfig struct {
    // Default limits
    DefaultLimitPerKeyword int `env:"DEFAULT_LIMIT_PER_KEYWORD" envDefault:"50"`
    DefaultMaxComments     int `env:"DEFAULT_MAX_COMMENTS" envDefault:"100"`
    DefaultMaxAttempts     int `env:"DEFAULT_MAX_ATTEMPTS" envDefault:"3"`

    // Dry-run limits
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

### 2. Hybrid State Tracking

**Thêm task-level và item-level tracking:**

```go
type ProjectState struct {
    Status ProjectStatus `json:"status"`

    // Task-level tracking (for completion check)
    TasksTotal  int64 `json:"tasks_total"`
    TasksDone   int64 `json:"tasks_done"`
    TasksErrors int64 `json:"tasks_errors"`

    // Item-level tracking (for progress display)
    ItemsExpected int64 `json:"items_expected"`
    ItemsActual   int64 `json:"items_actual"`
    ItemsErrors   int64 `json:"items_errors"`

    // Analyze phase (unchanged)
    AnalyzeTotal  int64 `json:"analyze_total"`
    AnalyzeDone   int64 `json:"analyze_done"`
    AnalyzeErrors int64 `json:"analyze_errors"`
}
```

**Logic:**

- Completion check dựa trên TASK-level: `(tasks_done + tasks_errors) >= tasks_total`
- Progress display dựa trên ITEM-level: `items_actual / items_expected * 100`

### 3. Enhanced Crawler Response Handling

**Thêm structs cho enhanced response:**

```go
type EnhancedCrawlerResult struct {
    Success   bool        `json:"success"`
    TaskType  string      `json:"task_type,omitempty"`
    LimitInfo *LimitInfo  `json:"limit_info,omitempty"`
    Stats     *CrawlStats `json:"stats,omitempty"`
    Error     *CrawlError `json:"error,omitempty"`
    Payload   any         `json:"payload"`
}

type LimitInfo struct {
    RequestedLimit  int  `json:"requested_limit"`
    AppliedLimit    int  `json:"applied_limit"`
    TotalFound      int  `json:"total_found"`
    PlatformLimited bool `json:"platform_limited"`
}

type CrawlStats struct {
    Successful     int     `json:"successful"`
    Failed         int     `json:"failed"`
    Skipped        int     `json:"skipped"`
    CompletionRate float64 `json:"completion_rate"`
}
```

**Fallback logic:** Nếu Crawler chưa update, Collector sẽ fallback về counting payload items.

### 4. Updated Webhook Format

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "PROCESSING",
  "tasks": {
    "total": 4,
    "done": 3,
    "errors": 0,
    "percent": 75.0
  },
  "items": {
    "expected": 200,
    "actual": 150,
    "errors": 5,
    "percent": 77.5
  },
  "analyze": {
    "total": 150,
    "done": 100,
    "errors": 2,
    "progress_percent": 68.0
  },
  "overall_progress_percent": 73.5
}
```

## Impact

### Affected Specs

- `event-infrastructure` - Update state management requirements

### Affected Code

| File                                           | Change                                       |
| ---------------------------------------------- | -------------------------------------------- |
| `config/config.go`                             | Thêm `CrawlLimitsConfig` struct              |
| `template.env`                                 | Thêm env vars cho crawl limits               |
| `k8s/configmap.yaml`                           | Thêm config entries                          |
| `internal/models/event_transform.go`           | Thêm config-based functions                  |
| `internal/models/crawler_result.go`            | Thêm enhanced response structs               |
| `internal/models/event.go`                     | Update `ProjectState` với hybrid fields      |
| `internal/state/interface.go`                  | Thêm new methods                             |
| `internal/state/types.go`                      | Thêm new field constants                     |
| `internal/state/repository/*.go`               | Implement new methods                        |
| `internal/state/usecase/*.go`                  | Implement new methods                        |
| `internal/dispatcher/usecase/new.go`           | Inject config                                |
| `internal/dispatcher/usecase/project_event.go` | Use config, set items_expected               |
| `internal/dispatcher/usecase/util.go`          | Use config for dry-run                       |
| `internal/results/usecase/result.go`           | Parse enhanced response, update hybrid state |
| `internal/webhook/types.go`                    | Update ProgressRequest                       |

### External Dependencies

| Service                  | Change Required                                  |
| ------------------------ | ------------------------------------------------ |
| Crawler (YouTube/TikTok) | Trả về `limit_info` và `stats` trong response    |
| Project Service          | Handle new webhook format với `tasks` và `items` |

### Breaking Changes

- Webhook format thay đổi (Project Service phải update handler)
- Redis field names thay đổi (existing state keys sẽ incompatible)

## Migration Path

### Phase 1: Collector Update (Day 1-2)

1. Deploy Collector với:
   - Config-driven limits
   - Hybrid state tracking
   - Fallback logic cho old Crawler response

### Phase 2: Crawler Update (Day 3-4)

1. Crawler update response format với `limit_info` và `stats`
2. Deploy Crawler

### Phase 3: Project Service Update (Day 5)

1. Update webhook handler cho new format
2. Deploy Project Service

### Rollback Plan

- Revert Collector deployment
- Redis keys cũ sẽ tự expire (TTL 7 ngày)
- Crawler/Project Service không bị ảnh hưởng nếu chưa deploy

## Open Questions

1. **Backward compatibility period:** Bao lâu cần support cả old và new format?
2. **Alerting:** Có cần alert khi `platform_limited: true`?
3. **Hard limits:** Có cần validate limits không vượt quá hard limits?

## Reference Documents

- `document/collector-implementation-plan.md` - Detailed implementation plan
- `document/collector-limit-config-analysis.md` - Problem analysis
- `document/collector-crawler-contract.md` - Contract specification
- `document/crawler-response-update-request.md` - Crawler update request
