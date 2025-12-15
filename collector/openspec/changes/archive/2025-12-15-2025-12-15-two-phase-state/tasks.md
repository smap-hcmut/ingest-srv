# Implementation Tasks

## 1. Models Update

- [x] 1.1 Update `internal/models/event.go` - ProjectState struct
  - Remove `Total`, `Done`, `Errors` fields
  - Add `CrawlTotal`, `CrawlDone`, `CrawlErrors` fields
  - Add `AnalyzeTotal`, `AnalyzeDone`, `AnalyzeErrors` fields
- [x] 1.2 Update `internal/models/event.go` - ProjectStatus constants
  - Remove `ProjectStatusCrawling`
  - Keep `ProjectStatusProcessing` (already exists)
- [x] 1.3 Add helper methods to ProjectState
  - `IsCrawlComplete() bool`
  - `IsAnalyzeComplete() bool`
  - `IsComplete() bool` - both phases complete
  - `CrawlProgressPercent() float64`
  - `AnalyzeProgressPercent() float64`
  - `OverallProgressPercent() float64`
- [x] 1.4 Update `NewProjectState()` function
  - Initialize all 6 counters to 0

## 2. State Interface Update

- [x] 2.1 Update `internal/state/interface.go`
  - Remove `UpdateTotal`, `IncrementDone`, `IncrementErrors`
  - Add `SetCrawlTotal(ctx, projectID, total) error`
  - Add `IncrementCrawlDoneBy(ctx, projectID, count) error`
  - Add `IncrementCrawlErrorsBy(ctx, projectID, count) error`
  - Add `IncrementAnalyzeTotalBy(ctx, projectID, count) error`
  - Add `IncrementAnalyzeDoneBy(ctx, projectID, count) error`
  - Add `IncrementAnalyzeErrorsBy(ctx, projectID, count) error`
  - Rename `CheckAndUpdateCompletion` to `CheckCompletion`

## 3. State Types Update

- [x] 3.1 Update `internal/state/types.go` - Redis field constants
  - Remove `FieldTotal`, `FieldDone`, `FieldErrors`
  - Add `FieldCrawlTotal`, `FieldCrawlDone`, `FieldCrawlErrors`
  - Add `FieldAnalyzeTotal`, `FieldAnalyzeDone`, `FieldAnalyzeErrors`

## 4. State UseCase Implementation

- [x] 4.1 Update `internal/state/usecase/state.go`
  - Implement `SetCrawlTotal` - set crawl_total + status=PROCESSING
  - Implement `IncrementCrawlDoneBy` - HINCRBY crawl_done
  - Implement `IncrementCrawlErrorsBy` - HINCRBY crawl_errors
  - Implement `IncrementAnalyzeTotalBy` - HINCRBY analyze_total
  - Implement `IncrementAnalyzeDoneBy` - HINCRBY analyze_done
  - Implement `IncrementAnalyzeErrorsBy` - HINCRBY analyze_errors
  - Update `CheckCompletion` - check both phases
- [x] 4.2 Update `internal/state/usecase/state_test.go`
  - Add tests for new methods
  - Update existing tests for renamed methods

## 5. State Repository Update

- [x] 5.1 Update `internal/state/repository/redis/redis.go`
  - Update `InitState` - use new field names
  - Update `GetState` - parse new field names

## 6. Webhook Types Update

- [x] 6.1 Update `internal/webhook/types.go`
  - Add `PhaseProgress` struct
  - Update `ProgressRequest` struct with Crawl, Analyze, OverallProgressPercent
  - Update `IsValid()` method
  - Remove old `ProgressPercent()` method

## 7. Webhook UseCase Update

- [x] 7.1 Update `internal/webhook/usecase/webhook.go`
  - Update `NotifyProgress` to build new format
  - Update `NotifyCompletion` to build new format
- [x] 7.2 Add helper function to build PhaseProgress from state

## 8. Results Types Update

- [x] 8.1 Update `internal/results/types.go`
  - Add `AnalyzeResultPayload` struct
    - `ProjectID string`
    - `JobID string`
    - `TaskType string`
    - `BatchSize int`
    - `SuccessCount int`
    - `ErrorCount int`

## 9. Results UseCase Update

- [x] 9.1 Update `internal/results/usecase/result.go` - HandleResult
  - Add case for `task_type: "analyze_result"`
  - Route to `handleAnalyzeResult` handler
- [x] 9.2 Implement `handleAnalyzeResult` handler
  - Extract `AnalyzeResultPayload` from message
  - Call `IncrementAnalyzeDoneBy(successCount)`
  - Call `IncrementAnalyzeErrorsBy(errorCount)`
  - Get state and notify progress
  - Check completion
- [x] 9.3 Update `handleProjectResult` handler
  - Count items in batch: `itemCount := len(payload)`
  - Call `IncrementCrawlDoneBy(itemCount)` instead of `IncrementDone()`
  - Call `IncrementAnalyzeTotalBy(itemCount)` for successful crawls
  - Call `IncrementCrawlErrorsBy(itemCount)` for failed crawls
- [x] 9.4 Add `extractAnalyzePayload` helper function
- [x] 9.5 Update `buildProgressRequest` to use new format

## 10. Dispatcher Update

- [x] 10.1 Update `internal/dispatcher/usecase/project_event.go`
  - Replace `UpdateTotal` with `SetCrawlTotal`

## 11. Testing

- [x] 11.1 Unit tests for `ProjectState` methods
  - `TestIsCrawlComplete`
  - `TestIsAnalyzeComplete`
  - `TestIsComplete`
  - `TestProgressPercent`
- [x] 11.2 Unit tests for state usecase
  - `TestSetCrawlTotal`
  - `TestIncrementCrawlDoneBy`
  - `TestIncrementAnalyzeDoneBy`
  - `TestCheckCompletion_BothPhasesComplete`
  - `TestCheckCompletion_OnlyCrawlComplete`
- [x] 11.3 Unit tests for results usecase
  - `TestHandleAnalyzeResult`
  - `TestHandleProjectResult_BatchIncrement`
- [x] 11.4 Unit tests for webhook format
  - `TestBuildProgressRequest_NewFormat`

## 12. Documentation

- [x] 12.1 Update `openspec/specs/event-infrastructure/spec.md`
  - Update Redis State Management requirements
  - Add Analyze Result Handler requirement
  - Update Webhook format requirement
- [x] 12.2 Update `README.md`
  - Update Redis State Management section
  - Add Analyze phase tracking documentation
- [x] 12.3 Update `document/collector-behavior.md`
  - Update state structure documentation
  - Add analyze result handling flow

## 13. Configuration & Deployment

- [x] 13.1 Verify `template.env` - No new config needed for two-phase state
  - Existing REDIS_STATE_DB config is sufficient
  - Existing PROJECT_SERVICE_URL and webhook configs are sufficient
- [x] 13.2 Verify `k8s/configmap.yaml` - No changes needed
  - Redis and Project Service configs already present
- [x] 13.3 Verify `k8s/secret.yaml` - No changes needed
  - Sensitive configs already present
- [x] 13.4 Update `document/collector-behavior.md`
  - Update document date
  - Update sequence diagram for two-phase flow

## Dependencies

### External (must be done by other teams)

- [ ] Analytics Service: Publish `analyze.result` messages
- [ ] Project Service: Update webhook handler for new format

### Deployment Order

1. Deploy Collector (this change)
2. Deploy Analytics Service (publish analyze.result)
3. Deploy Project Service (handle new webhook format)
