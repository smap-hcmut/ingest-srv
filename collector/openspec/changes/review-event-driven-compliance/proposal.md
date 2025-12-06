# Change: Review Event-Driven Architecture Compliance

## Why

Sau khi implement 2 proposals trước đó (`align-event-driven-architecture` và `remove-throttling-simplify-progress`), cần review và xác nhận source code hiện tại đã tuân thủ đầy đủ các yêu cầu trong `document/event-drivent.md`.

**Mục tiêu:**

- Xác nhận tất cả requirements trong document đã được implement đúng
- Phát hiện bất kỳ gaps hoặc inconsistencies nào giữa document và implementation
- Đảm bảo code quality và maintainability

## Compliance Review Summary

### ✅ 1. Exchange & Routing Key Configuration (Section 1.1)

| Requirement   | Document                    | Implementation                                      | Status |
| ------------- | --------------------------- | --------------------------------------------------- | ------ |
| Exchange name | `smap.events`               | `ExchangeSMAPEvents = "smap.events"`                | ✅     |
| Exchange type | `topic`                     | `ExchangeTypeTopic`                                 | ✅     |
| Routing key   | `project.created`           | `RoutingKeyProjectCreated = "project.created"`      | ✅     |
| Queue name    | `collector.project.created` | `QueueProjectCreated = "collector.project.created"` | ✅     |

**Files:** `internal/dispatcher/delivery/rabbitmq/constants.go`

---

### ✅ 2. ProjectCreatedEvent Schema (Section 3.1)

| Field                             | Document            | Implementation                              | Status |
| --------------------------------- | ------------------- | ------------------------------------------- | ------ |
| `event_id`                        | string              | `EventID string`                            | ✅     |
| `timestamp`                       | time.Time           | `Timestamp time.Time`                       | ✅     |
| `payload.project_id`              | string              | `ProjectID string`                          | ✅     |
| `payload.user_id`                 | string              | `UserID string`                             | ✅     |
| `payload.brand_name`              | string              | `BrandName string`                          | ✅     |
| `payload.brand_keywords`          | []string            | `BrandKeywords []string`                    | ✅     |
| `payload.competitor_names`        | []string            | `CompetitorNames []string`                  | ✅     |
| `payload.competitor_keywords_map` | map[string][]string | `CompetitorKeywordsMap map[string][]string` | ✅     |
| `payload.date_range.from`         | YYYY-MM-DD          | `From string`                               | ✅     |
| `payload.date_range.to`           | YYYY-MM-DD          | `To string`                                 | ✅     |

**Files:** `internal/models/event.go`

---

### ✅ 3. Redis State Management (Section 4)

| Requirement   | Document                | Implementation                    | Status |
| ------------- | ----------------------- | --------------------------------- | ------ |
| Database      | DB 1                    | `cfg.Redis.StateDB` (default: 1)  | ✅     |
| Key schema    | `smap:proj:{projectID}` | `KeyPrefix = "smap:proj:"`        | ✅     |
| Field: status | String                  | `FieldStatus = "status"`          | ✅     |
| Field: total  | Int                     | `FieldTotal = "total"`            | ✅     |
| Field: done   | Int                     | `FieldDone = "done"`              | ✅     |
| Field: errors | Int                     | `FieldErrors = "errors"`          | ✅     |
| TTL           | 7 days                  | `DefaultTTL = 7 * 24 * time.Hour` | ✅     |

**Status Constants:**
| Status | Document | Implementation | Status |
|--------|----------|----------------|--------|
| INITIALIZING | ✓ | `ProjectStatusInitializing` | ✅ |
| CRAWLING | ✓ | `ProjectStatusCrawling` | ✅ |
| PROCESSING | ✓ | `ProjectStatusProcessing` | ✅ |
| DONE | ✓ | `ProjectStatusDone` | ✅ |
| FAILED | ✓ | `ProjectStatusFailed` | ✅ |

**Files:** `internal/state/types.go`, `internal/models/event.go`

---

### ✅ 4. Progress Webhook (Section 5)

| Requirement       | Document                      | Implementation                              | Status |
| ----------------- | ----------------------------- | ------------------------------------------- | ------ |
| Endpoint          | `/internal/progress/callback` | `progressCallbackEndpoint`                  | ✅     |
| Header            | `X-Internal-Key`              | `httpReq.Header.Set("X-Internal-Key", ...)` | ✅     |
| Field: project_id | string                        | `ProjectID string`                          | ✅     |
| Field: user_id    | string                        | `UserID string`                             | ✅     |
| Field: status     | string                        | `Status string`                             | ✅     |
| Field: total      | int64                         | `Total int64`                               | ✅     |
| Field: done       | int64                         | `Done int64`                                | ✅     |
| Field: errors     | int64                         | `Errors int64`                              | ✅     |
| Retry             | Exponential backoff           | `delay *= 2`                                | ✅     |

**Files:** `internal/webhook/types.go`, `pkg/project/client.go`

---

### ✅ 5. Clean Architecture (Section 8)

| Layer      | Document               | Implementation                                         | Status |
| ---------- | ---------------------- | ------------------------------------------------------ | ------ |
| cmd/       | Khởi tạo external deps | `cmd/consumer/main.go`                                 | ✅     |
| server.go  | Nhận deps đã sẵn sàng  | `internal/consumer/server.go`                          | ✅     |
| UseCase    | Business logic         | `internal/state/usecase/`, `internal/webhook/usecase/` | ✅     |
| Repository | Data access only       | `internal/state/repository/redis/`                     | ✅     |

**Fail-Fast Pattern:**

- Redis: `l.Fatalf(ctx, "failed to connect to Redis: %v", err)` ✅
- RabbitMQ: `l.Fatalf(ctx, "failed to connect to RabbitMQ: %v", err)` ✅

**Files:** `cmd/consumer/main.go`, `internal/consumer/new.go`, `internal/consumer/server.go`

---

### ✅ 6. User Mapping Storage

| Requirement   | Document             | Implementation          | Status |
| ------------- | -------------------- | ----------------------- | ------ |
| Store mapping | project_id → user_id | `StoreUserMapping()`    | ✅     |
| Key schema    | -                    | `smap:user:{projectID}` | ✅     |
| Retrieve      | GetUserID            | `GetUserID()`           | ✅     |

**Files:** `internal/state/usecase/state.go`, `internal/state/types.go`

---

### ✅ 7. Event Consumer Implementation

| Requirement             | Document            | Implementation                   | Status |
| ----------------------- | ------------------- | -------------------------------- | ------ |
| Consume project.created | ✓                   | `ConsumeProjectEvents()`         | ✅     |
| Parse event             | ✓                   | `json.Unmarshal(d.Body, &event)` | ✅     |
| Validate event          | ✓                   | `event.IsValid()`                | ✅     |
| Handle invalid          | Reject (no requeue) | `d.Ack(false)`                   | ✅     |
| Handle error            | Nack with requeue   | `d.Nack(false, true)`            | ✅     |

**Files:** `internal/dispatcher/delivery/rabbitmq/consumer/project_consumer.go`

---

### ✅ 8. Throttling Removed (Per Proposal 2)

| Requirement    | Document                         | Implementation                 | Status |
| -------------- | -------------------------------- | ------------------------------ | ------ |
| No throttling  | Workers report once per platform | No throttler code              | ✅     |
| Simple webhook | `NotifyProgress()` only          | No `NotifyProgressImmediate()` | ✅     |

---

## What Changes

**Không có thay đổi code cần thiết.** Source code hiện tại đã tuân thủ đầy đủ `document/event-drivent.md`.

### Documentation Updates (Optional)

- Cập nhật `openspec/specs/event-infrastructure/spec.md` với verification status
- Thêm compliance checklist vào `document/event-drivent.md`

## Impact

- **Affected specs**: event-infrastructure (verification only)
- **Affected code**: None - compliance verified
- **Risk**: None - review only

## Verification Checklist

### Exchange & Routing

- [x] `smap.events` exchange declared as topic
- [x] `collector.project.created` queue bound to `project.created` routing key
- [x] Legacy `collector.inbound` exchange still supported (deprecated)

### Event Schema

- [x] `ProjectCreatedEvent` struct matches document schema
- [x] `user_id` included in payload for progress notifications
- [x] `IsValid()` method validates required fields

### Redis State

- [x] Using Redis DB 1 for state
- [x] Key schema `smap:proj:{projectID}` implemented
- [x] All hash fields (status, total, done, errors) implemented
- [x] 7-day TTL configured

### Progress Webhook

- [x] Endpoint `/internal/progress/callback` used
- [x] `X-Internal-Key` header included
- [x] Request body matches document schema
- [x] Exponential backoff retry implemented

### Clean Architecture

- [x] Redis initialized in `cmd/consumer/main.go`
- [x] Fail-fast on connection errors
- [x] Server receives initialized dependencies
- [x] No conditional initialization in server layer

### Throttling

- [x] Throttling code removed
- [x] Simple webhook notification pattern
