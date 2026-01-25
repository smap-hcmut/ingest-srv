# Change: Verify Crawler-Collector Integration

## Why

Sau khi hoàn thành Collector Service và Crawler Service (TikTok & YouTube), cần verify rằng integration giữa hai services hoạt động đúng theo event-driven architecture đã định nghĩa. Document `collector-integration-checklist.md` đã được tạo để guide quá trình verification này.

## What Changes

Đây là proposal để **verify và document** integration compliance, không phải thêm tính năng mới:

1. **Task Type Routing Verification**

   - Verify dry-run results route đến `/internal/dryrun/callback`
   - Verify project execution results route đến Redis + `/internal/progress/callback`
   - Verify backward compatibility với legacy results (không có `task_type`)

2. **Error Handling Verification**

   - Verify error codes từ Crawler được propagate đúng
   - Verify error rate monitoring hoạt động

3. **Progress Tracking Verification**

   - Verify Redis state management (`smap:proj:{projectID}`)
   - Verify progress webhook notifications
   - Verify WebSocket notifications (nếu có)

4. **Data.Collected Event Verification**
   - Verify events published đến `smap.events` exchange
   - Verify batch upload đến MinIO

## Impact

- Affected specs: `event-infrastructure`
- Affected code:
  - `internal/results/usecase/result.go` (routing logic)
  - `internal/results/types.go` (TaskType field)
  - `internal/state/usecase/state.go` (Redis state)
  - `internal/webhook/usecase/webhook.go` (progress notifications)

## Verification Checklist Reference

Chi tiết verification steps được document trong `document/collector-integration-checklist.md`:

### Pre-Deployment Verification

- [ ] Verify `TaskType` field trong `CrawlerContentMeta`
- [ ] Verify routing logic trong `HandleResult()`
- [ ] Verify `handleDryRunResult()` và `handleProjectResult()` methods
- [ ] Verify backward compatibility default routing

### Task Type Routing

- [ ] Dry-run task verification (task_type: `dryrun_keyword`)
- [ ] Project execution task verification (task_type: `research_and_crawl`)
- [ ] Backward compatibility verification (missing task_type)

### Error Handling

- [ ] Error code propagation (CONTENT_REMOVED, CONTENT_NOT_FOUND, etc.)
- [ ] Error rate monitoring

### Progress Tracking

- [ ] Redis state updates (status, total, done, errors)
- [ ] Progress webhook calls
- [ ] State transitions (INITIALIZING → CRAWLING → DONE/FAILED)

### Data.Collected Events

- [ ] Event schema compliance
- [ ] MinIO batch file verification

## Success Criteria

Integration được coi là **PASS** khi:

1. Tất cả routing tests pass
2. Error handling hoạt động đúng
3. Progress tracking accurate
4. Data.collected events được publish đúng format
5. Không có data loss hoặc corruption
