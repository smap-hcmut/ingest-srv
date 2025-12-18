# Collector ↔ Services Contract Specification

## Tổng quan

Document này định nghĩa contract giữa Collector Service và các services khác:

1. **Crawler → Collector**: Kết quả crawl từ TikTok/YouTube (case `research_and_crawl`)
2. **Analytics → Collector**: Kết quả phân tích từ Analytics Service

**Mục đích:** Đảm bảo các services đồng bộ về format message, behavior, và expectations.

**Version:** 3.0 (Flat structure, no payload for research_and_crawl)

---

## 1. Message Flow Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PHASE 1: CRAWL (Crawler → Collector)                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐         ┌─────────────┐         ┌──────────────┐          │
│  │   Crawler    │ ──────► │  RabbitMQ   │ ──────► │  Collector   │          │
│  │   (Worker)   │         │   Queue     │         │   (Results)  │          │
│  └──────────────┘         └─────────────┘         └──────────────┘          │
│                                                                              │
│  Message: CrawlerResultMessage (flat, no payload)                            │
│  Queue: collector.results                                                    │
│                                                                              │
│  NOTE: Crawler push content trực tiếp sang Analytics, không qua Collector    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                   PHASE 2: ANALYZE (Analytics → Collector)                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐         ┌─────────────┐         ┌──────────────┐          │
│  │  Analytics   │ ──────► │  RabbitMQ   │ ──────► │  Collector   │          │
│  │   Service    │         │   Queue     │         │   (Results)  │          │
│  └──────────────┘         └─────────────┘         └──────────────┘          │
│                                                                              │
│  Message: AnalyzeResultMessage (flat)                                        │
│  Queue: collector.results                                                    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. CONTRACT 1: Crawler → Collector (CrawlerResultMessage)

### 2.1 Message Structure (FLAT - No Payload)

```json
{
  "success": true,
  "task_type": "research_and_crawl",
  "job_id": "proj123-brand-0",
  "platform": "tiktok",
  "requested_limit": 50,
  "applied_limit": 50,
  "total_found": 30,
  "platform_limited": true,
  "successful": 28,
  "failed": 2,
  "skipped": 0,
  "error_code": null,
  "error_message": null
}
```

### 2.2 Field Definitions

| Field              | Type        | Required | Description                                         |
| ------------------ | ----------- | -------- | --------------------------------------------------- |
| `success`          | bool        | ✅       | `true` nếu task hoàn thành (kể cả không có kết quả) |
| `task_type`        | string      | ✅       | `"research_and_crawl"` cho project execution        |
| `job_id`           | string      | ✅       | Job ID format: `{projectID}-{source}-{index}`       |
| `platform`         | string      | ✅       | `"tiktok"` hoặc `"youtube"`                         |
| `requested_limit`  | int         | ✅       | Limit được request từ Collector                     |
| `applied_limit`    | int         | ✅       | Limit thực tế Crawler áp dụng                       |
| `total_found`      | int         | ✅       | Số items tìm được trên platform (trước khi crawl)   |
| `platform_limited` | bool        | ✅       | `true` nếu `total_found < requested_limit`          |
| `successful`       | int         | ✅       | Số items crawl thành công                           |
| `failed`           | int         | ✅       | Số items crawl thất bại                             |
| `skipped`          | int         | ✅       | Số items bị skip (duplicate, filtered, etc.)        |
| `error_code`       | string/null | ❌       | Error code khi `success=false`                      |
| `error_message`    | string/null | ❌       | Error message khi `success=false`                   |

### 2.3 Response Scenarios

#### Scenario 1: Full Success (tìm đủ limit)

```json
{
  "success": true,
  "task_type": "research_and_crawl",
  "job_id": "proj123-brand-0",
  "platform": "tiktok",
  "requested_limit": 50,
  "applied_limit": 50,
  "total_found": 50,
  "platform_limited": false,
  "successful": 50,
  "failed": 0,
  "skipped": 0,
  "error_code": null,
  "error_message": null
}
```

#### Scenario 2: Platform Limited (tìm ít hơn limit)

```json
{
  "success": true,
  "task_type": "research_and_crawl",
  "job_id": "proj123-brand-0",
  "platform": "tiktok",
  "requested_limit": 50,
  "applied_limit": 50,
  "total_found": 30,
  "platform_limited": true,
  "successful": 28,
  "failed": 2,
  "skipped": 0,
  "error_code": null,
  "error_message": null
}
```

#### Scenario 3: No Results (keyword không có video)

```json
{
  "success": true,
  "task_type": "research_and_crawl",
  "job_id": "proj123-brand-0",
  "platform": "tiktok",
  "requested_limit": 50,
  "applied_limit": 50,
  "total_found": 0,
  "platform_limited": true,
  "successful": 0,
  "failed": 0,
  "skipped": 0,
  "error_code": null,
  "error_message": null
}
```

#### Scenario 4: Task Failed (error xảy ra)

```json
{
  "success": false,
  "task_type": "research_and_crawl",
  "job_id": "proj123-brand-0",
  "platform": "tiktok",
  "requested_limit": 50,
  "applied_limit": 50,
  "total_found": 0,
  "platform_limited": false,
  "successful": 0,
  "failed": 0,
  "skipped": 0,
  "error_code": "SEARCH_FAILED",
  "error_message": "TikTok API rate limited"
}
```

### 2.4 Error Codes

| Code                     | Description           | Retryable |
| ------------------------ | --------------------- | --------- |
| `SEARCH_FAILED`          | Search API failed     | ✅ Yes    |
| `RATE_LIMITED`           | Platform rate limit   | ✅ Yes    |
| `RATE_LIMITED_PERMANENT` | Permanent rate limit  | ❌ No     |
| `AUTH_FAILED`            | Authentication failed | ❌ No     |
| `INVALID_KEYWORD`        | Keyword không hợp lệ  | ❌ No     |
| `BLOCKED`                | IP/Account bị block   | ❌ No     |
| `TIMEOUT`                | Request timeout       | ✅ Yes    |

### 2.5 Validation Rules

1. **Platform Limited Logic**: `platform_limited = (total_found < requested_limit)`
2. **Job ID Format**: `{projectID}-{source}-{index}` (e.g., `proj123-brand-0`)
3. **Task Type**: PHẢI là `"research_and_crawl"` cho project execution
4. **Stats Consistency**: `successful + failed + skipped` = số items đã attempt

---

## 3. CONTRACT 2: Analytics → Collector (AnalyzeResultMessage)

### 3.1 Message Structure (FLAT)

```json
{
  "task_type": "analyze_result",
  "project_id": "proj123",
  "job_id": "proj123-analyze-batch-1",
  "batch_size": 10,
  "success_count": 8,
  "error_count": 2
}
```

### 3.2 Field Definitions

| Field           | Type   | Required | Description                                    |
| --------------- | ------ | -------- | ---------------------------------------------- |
| `task_type`     | string | ✅       | **PHẢI là `"analyze_result"`** để routing đúng |
| `project_id`    | string | ✅       | Project ID để identify state trong Redis       |
| `job_id`        | string | ✅       | Job ID để tracking và logging                  |
| `batch_size`    | int    | ✅       | Số items trong batch được gửi để analyze       |
| `success_count` | int    | ✅       | Số items analyze thành công                    |
| `error_count`   | int    | ✅       | Số items analyze thất bại                      |

### 3.3 Response Scenarios

#### Scenario 1: Full Success

```json
{
  "task_type": "analyze_result",
  "project_id": "proj123",
  "job_id": "proj123-analyze-batch-1",
  "batch_size": 10,
  "success_count": 10,
  "error_count": 0
}
```

#### Scenario 2: Partial Success

```json
{
  "task_type": "analyze_result",
  "project_id": "proj123",
  "job_id": "proj123-analyze-batch-1",
  "batch_size": 10,
  "success_count": 7,
  "error_count": 3
}
```

#### Scenario 3: All Failed

```json
{
  "task_type": "analyze_result",
  "project_id": "proj123",
  "job_id": "proj123-analyze-batch-1",
  "batch_size": 10,
  "success_count": 0,
  "error_count": 10
}
```

### 3.4 Validation Rules

1. **Task Type**: `task_type` PHẢI bằng `"analyze_result"`
2. **Project ID Required**: `project_id` PHẢI non-empty
3. **Count Consistency**: `success_count + error_count` SHOULD bằng `batch_size`
4. **Non-negative Counts**: Tất cả counts >= 0

---

## 4. Collector Processing Logic

### 4.1 Message Routing

```go
// Route message based on task_type
switch msg.TaskType {
case "research_and_crawl":
    return handleCrawlerResult(ctx, msg)
case "analyze_result":
    return handleAnalyzeResult(ctx, msg)
default:
    return ErrUnknownTaskType
}
```

### 4.2 Crawler Result Processing

```go
func handleCrawlerResult(ctx context.Context, msg CrawlerResultMessage) error {
    // 1. Extract project_id từ job_id
    projectID := extractProjectID(msg.JobID)  // "proj123-brand-0" → "proj123"

    // 2. Update task-level counter
    if msg.Success {
        stateUC.IncrementTasksDone(ctx, projectID)
    } else {
        stateUC.IncrementTasksErrors(ctx, projectID)
    }

    // 3. Update item-level counters
    if msg.Successful > 0 {
        stateUC.IncrementItemsActualBy(ctx, projectID, msg.Successful)
        stateUC.IncrementAnalyzeTotalBy(ctx, projectID, msg.Successful)
    }
    if msg.Failed > 0 {
        stateUC.IncrementItemsErrorsBy(ctx, projectID, msg.Failed)
    }

    // 4. Log platform limitation warning
    if msg.PlatformLimited {
        log.Warnf("Platform limited: requested=%d, found=%d",
            msg.RequestedLimit, msg.TotalFound)
    }

    // 5. Send progress webhook và check completion
    sendProgressWebhook(ctx, projectID)
    checkAndUpdateCompletion(ctx, projectID)
}
```

### 4.3 Analytics Result Processing

```go
func handleAnalyzeResult(ctx context.Context, msg AnalyzeResultMessage) error {
    // 1. Validate task_type
    if msg.TaskType != "analyze_result" {
        return ErrInvalidTaskType
    }

    // 2. Validate project_id
    if msg.ProjectID == "" {
        return ErrMissingProjectID
    }

    // 3. Update analyze counters
    if msg.SuccessCount > 0 {
        stateUC.IncrementAnalyzeDoneBy(ctx, msg.ProjectID, msg.SuccessCount)
    }
    if msg.ErrorCount > 0 {
        stateUC.IncrementAnalyzeErrorsBy(ctx, msg.ProjectID, msg.ErrorCount)
    }

    // 4. Send progress webhook và check completion
    sendProgressWebhook(ctx, msg.ProjectID)
    checkAndUpdateCompletion(ctx, msg.ProjectID)
}
```

---

## 5. State Tracking

### 5.1 Two-Phase Processing Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TWO-PHASE PROCESSING                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  PHASE 1: CRAWL                          PHASE 2: ANALYZE                    │
│  ┌────────────────────────┐              ┌────────────────────────┐          │
│  │ Crawler → Collector    │              │ Analytics → Collector  │          │
│  │                        │              │                        │          │
│  │ Update:                │              │ Update:                │          │
│  │ - tasks_done += 1      │              │ - analyze_done +=      │          │
│  │ - items_actual +=      │              │     success_count      │          │
│  │     successful         │              │ - analyze_errors +=    │          │
│  │ - items_errors +=      │              │     error_count        │          │
│  │     failed             │              │                        │          │
│  │ - analyze_total +=     │              │                        │          │
│  │     successful         │              │                        │          │
│  └────────────────────────┘              └────────────────────────┘          │
│                                                                              │
│  Completion Check:                                                           │
│  - Crawl: tasks_done + tasks_errors >= tasks_total                           │
│  - Analyze: analyze_done + analyze_errors >= analyze_total                   │
│  - Project DONE: Crawl Complete AND Analyze Complete                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 5.2 Redis State Structure

```json
{
  "status": "PROCESSING",
  "tasks_total": 6,
  "tasks_done": 4,
  "tasks_errors": 0,
  "items_expected": 300,
  "items_actual": 180,
  "items_errors": 5,
  "analyze_total": 180,
  "analyze_done": 100,
  "analyze_errors": 2
}
```

### 5.3 Completion Logic

```go
func IsComplete(state *ProjectState) bool {
    crawlComplete := (state.TasksDone + state.TasksErrors >= state.TasksTotal)
    analyzeComplete := (state.AnalyzeDone + state.AnalyzeErrors >= state.AnalyzeTotal)
    return crawlComplete && analyzeComplete
}
```

---

## 6. Progress Webhook Payload

```json
{
  "project_id": "proj123",
  "status": "PROCESSING",
  "overall_progress_percent": 65.5,
  "crawl": {
    "tasks": { "total": 6, "done": 4, "errors": 0, "percent": 66.67 },
    "items": { "expected": 300, "actual": 180, "errors": 5, "percent": 60.0 }
  },
  "analyze": {
    "total": 180,
    "done": 100,
    "errors": 2,
    "percent": 55.56
  }
}
```

---

## 7. Example Full Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  EXAMPLE: 1 Project, 2 keywords, 2 platforms                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  INPUT:                                                                      │
│  - project_id: "proj123"                                                     │
│  - brand_keywords: ["iphone", "samsung"]                                     │
│  - limit_per_keyword: 50                                                     │
│                                                                              │
│  DISPATCH (4 tasks):                                                         │
│  - tasks_total: 4                                                            │
│  - items_expected: 200 (4 × 50)                                              │
│                                                                              │
│  CRAWLER MESSAGES:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │ {                                                                    │     │
│  │   "success": true,                                                   │     │
│  │   "task_type": "research_and_crawl",                                 │     │
│  │   "job_id": "proj123-brand-0",                                       │     │
│  │   "platform": "tiktok",                                              │     │
│  │   "requested_limit": 50, "applied_limit": 50,                        │     │
│  │   "total_found": 50, "platform_limited": false,                      │     │
│  │   "successful": 48, "failed": 2, "skipped": 0                        │     │
│  │ }                                                                    │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│  → tasks_done=1, items_actual=48, items_errors=2, analyze_total=48           │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │ {                                                                    │     │
│  │   "success": true,                                                   │     │
│  │   "task_type": "research_and_crawl",                                 │     │
│  │   "job_id": "proj123-brand-1",                                       │     │
│  │   "platform": "tiktok",                                              │     │
│  │   "requested_limit": 50, "applied_limit": 50,                        │     │
│  │   "total_found": 30, "platform_limited": true,                       │     │
│  │   "successful": 30, "failed": 0, "skipped": 0                        │     │
│  │ }                                                                    │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│  → tasks_done=2, items_actual=78, items_errors=2, analyze_total=78           │
│                                                                              │
│  ... (2 more crawler messages for YouTube)                                   │
│                                                                              │
│  STATE AFTER CRAWL COMPLETE:                                                 │
│  - tasks_total: 4, tasks_done: 4, tasks_errors: 0                            │
│  - items_expected: 200, items_actual: 173, items_errors: 7                   │
│  - analyze_total: 173                                                        │
│                                                                              │
│  ANALYTICS MESSAGES:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │ {                                                                    │     │
│  │   "task_type": "analyze_result",                                     │     │
│  │   "project_id": "proj123",                                           │     │
│  │   "job_id": "proj123-analyze-batch-1",                               │     │
│  │   "batch_size": 50, "success_count": 48, "error_count": 2            │     │
│  │ }                                                                    │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│  → analyze_done=48, analyze_errors=2                                         │
│                                                                              │
│  ... (more analytics messages)                                               │
│                                                                              │
│  FINAL STATE:                                                                │
│  - tasks_done: 4, tasks_errors: 0                                            │
│  - items_actual: 173, items_errors: 7                                        │
│  - analyze_done: 171, analyze_errors: 2                                      │
│  - Status: DONE                                                              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Summary Checklist

### Crawler Team:

- [ ] Gửi flat message với tất cả fields ở root level
- [ ] Include `job_id` và `platform` trực tiếp trong message
- [ ] Set `task_type = "research_and_crawl"`
- [ ] Set `platform_limited = true` khi `total_found < requested_limit`
- [ ] **KHÔNG cần gửi payload content** - push trực tiếp sang Analytics

### Analytics Team:

- [ ] Gửi flat message với `task_type = "analyze_result"`
- [ ] Include `project_id` trực tiếp (không để Collector parse từ job_id)
- [ ] Report `success_count` và `error_count` chính xác

### Collector:

- [ ] Parse flat message format
- [ ] Extract `project_id` từ `job_id` (format: `{projectID}-{source}-{index}`)
- [ ] Update state counters atomically
- [ ] Check completion dựa trên cả Crawl và Analyze phases
- [ ] Log warning khi `platform_limited = true`
