# Implementation Tasks

## Phase 1: Config Layer

### 1.1 Config Structure

- [x] 1.1.1 Update `config/config.go` - Add CrawlLimitsConfig struct

  - Add `CrawlLimitsConfig` struct with 10 fields
  - Add `CrawlLimits CrawlLimitsConfig` field to main `Config` struct
  - Fields: DefaultLimitPerKeyword, DefaultMaxComments, DefaultMaxAttempts
  - Fields: DryRunLimitPerKeyword, DryRunMaxComments
  - Fields: MaxLimitPerKeyword, MaxMaxComments
  - Fields: IncludeComments, DownloadMedia

- [x] 1.1.2 Update `template.env` - Add env vars

  - Add DEFAULT_LIMIT_PER_KEYWORD=50
  - Add DEFAULT_MAX_COMMENTS=100
  - Add DEFAULT_MAX_ATTEMPTS=3
  - Add DRYRUN_LIMIT_PER_KEYWORD=3
  - Add DRYRUN_MAX_COMMENTS=5
  - Add MAX_LIMIT_PER_KEYWORD=500
  - Add MAX_MAX_COMMENTS=1000
  - Add INCLUDE_COMMENTS=true
  - Add DOWNLOAD_MEDIA=false

- [x] 1.1.3 Update `k8s/configmap.yaml` - Add config entries
  - Add all 9 env vars to ConfigMap data section

---

## Phase 2: Models Layer

### 2.1 Enhanced Crawler Response

- [x] 2.1.1 Update `internal/models/crawler_result.go` - Add enhanced structs
  - Add `EnhancedCrawlerResult` struct
  - Add `LimitInfo` struct (RequestedLimit, AppliedLimit, TotalFound, PlatformLimited)
  - Add `CrawlStats` struct (Successful, Failed, Skipped, CompletionRate)
  - Add `CrawlError` struct (Code, Message)
  - Add `IsRetryable()` method to CrawlError

### 2.2 Config-Based Transform Options

- [x] 2.2.1 Update `internal/models/event_transform.go` - Add config functions
  - Add `NewTransformOptionsFromConfig(cfg CrawlLimitsConfig) TransformOptions`
  - Add `NewDryRunOptionsFromConfig(cfg CrawlLimitsConfig) TransformOptions`
  - Add deprecation comment to `DefaultTransformOptions()`
  - Ensure hard limit validation in config functions

### 2.3 Hybrid State Structure

- [x] 2.3.1 Update `internal/models/event.go` - Update ProjectState
  - Add `TasksTotal`, `TasksDone`, `TasksErrors` fields
  - Add `ItemsExpected`, `ItemsActual`, `ItemsErrors` fields
  - Keep existing `AnalyzeTotal`, `AnalyzeDone`, `AnalyzeErrors` fields
  - Update `IsCrawlComplete()` to use task-level
  - Update `CrawlProgressPercent()` to use item-level with fallback
  - Add `TasksProgressPercent()` method
  - Add `ItemsProgressPercent()` method
  - Update `OverallProgressPercent()` method

---

## Phase 3: State Layer

### 3.1 State Types

- [x] 3.1.1 Update `internal/state/types.go` - Add field constants
  - Add `FieldTasksTotal`, `FieldTasksDone`, `FieldTasksErrors`
  - Add `FieldItemsExpected`, `FieldItemsActual`, `FieldItemsErrors`
  - Keep existing analyze field constants

### 3.2 State Interface

- [x] 3.2.1 Update `internal/state/interface.go` - Add new methods
  - Add `SetTasksTotal(ctx, projectID string, tasksTotal, itemsExpected int64) error`
  - Add `IncrementTasksDone(ctx, projectID string) error`
  - Add `IncrementTasksErrors(ctx, projectID string) error`
  - Add `IncrementItemsActualBy(ctx, projectID string, count int64) error`
  - Add `IncrementItemsErrorsBy(ctx, projectID string, count int64) error`

### 3.3 State Repository

- [x] 3.3.1 Update `internal/state/repository/redis/redis.go` - Implement methods
  - Implement `SetTasksTotal` - HSET tasks_total, items_expected, status=PROCESSING
  - Implement `IncrementTasksDone` - HINCRBY tasks_done 1
  - Implement `IncrementTasksErrors` - HINCRBY tasks_errors 1
  - Implement `IncrementItemsActualBy` - HINCRBY items_actual N
  - Implement `IncrementItemsErrorsBy` - HINCRBY items_errors N
  - Update `GetState` to parse new fields

### 3.4 State UseCase

- [x] 3.4.1 Update `internal/state/usecase/state.go` - Implement methods
  - Implement all new interface methods
  - Update `CheckCompletion` to use task-level completion

---

## Phase 4: Dispatcher Layer

### 4.1 Dispatcher UseCase

- [x] 4.1.1 Update `internal/dispatcher/usecase/new.go` - Inject config

  - Add `crawlLimitsCfg config.CrawlLimitsConfig` field to `implUseCase`
  - Add `crawlLimitsCfg` parameter to `New()` function
  - Initialize field in constructor

- [x] 4.1.2 Update `internal/dispatcher/usecase/project_event.go` - Use config

  - Replace `models.DefaultTransformOptions()` with `models.NewTransformOptionsFromConfig(uc.crawlLimitsCfg)`
  - Calculate `itemsExpected = totalTasks * opts.LimitPerKeyword`
  - Replace `SetCrawlTotal` with `SetTasksTotal(ctx, projectID, totalTasks, itemsExpected)`

- [x] 4.1.3 Update `internal/dispatcher/usecase/util.go` - Use config for dry-run
  - Replace hardcoded `LimitPerKeyword = 3` with `uc.crawlLimitsCfg.DryRunLimitPerKeyword`
  - Replace hardcoded `MaxComments = 5` with `uc.crawlLimitsCfg.DryRunMaxComments`
  - Apply to both `mapYouTubePayload` and `mapTikTokPayload`

---

## Phase 5: Results Handler Layer

### 5.1 Results Types

- [x] 5.1.1 Update `internal/results/types.go` - Add types (if not in models)
  - Add `LimitInfo` struct (if not already in models)
  - Add `CrawlStats` struct (if not already in models)
  - Note: Types added to `internal/models/crawler_result.go` in Phase 2

### 5.2 Results UseCase

- [x] 5.2.1 Update `internal/results/usecase/result.go` - Add extraction functions

  - Add `extractLimitInfoAndStats(ctx, res) (*LimitInfo, *CrawlStats)`
  - Add `fallbackLimitInfoAndStats(ctx, res) (*LimitInfo, *CrawlStats)`
  - Implement fallback logic: count payload items if enhanced fields missing

- [x] 5.2.2 Update `internal/results/usecase/result.go` - Update handleProjectResult

  - Call `extractLimitInfoAndStats` at start
  - Update task-level: `IncrementTasksDone` or `IncrementTasksErrors`
  - Update item-level: `IncrementItemsActualBy(stats.Successful)`
  - Update item-level: `IncrementItemsErrorsBy(stats.Failed)`
  - Update analyze: `IncrementAnalyzeTotalBy(stats.Successful)`
  - Add platform limitation logging
  - Update progress webhook building

- [x] 5.2.3 Update `internal/results/usecase/result.go` - Update progress building
  - Rename `buildTwoPhaseProgressRequest` to `buildHybridProgressRequest`
  - Build `Tasks` progress from state
  - Build `Items` progress from state
  - Keep `Analyze` progress unchanged
  - Calculate `OverallProgressPercent`

---

## Phase 6: Webhook Layer

### 6.1 Webhook Types

- [x] 6.1.1 Update `internal/webhook/types.go` - Update ProgressRequest
  - Add `TaskProgress` struct (Total, Done, Errors, Percent)
  - Add `ItemProgress` struct (Expected, Actual, Errors, Percent)
  - Update `ProgressRequest` to include `Tasks TaskProgress`
  - Update `ProgressRequest` to include `Items ItemProgress`
  - Keep existing `Analyze PhaseProgress`
  - Update `OverallProgressPercent` calculation

---

## Phase 7: Consumer/Main Integration

### 7.1 Consumer Update

- [x] 7.1.1 Update `cmd/consumer/main.go` or `internal/consumer/server.go`
  - Pass `cfg.CrawlLimits` to dispatcher usecase constructor
  - Ensure config is loaded and passed correctly
  - Updated `internal/consumer/new.go` to add `CrawlLimitsConfig` to Config struct
  - Updated `internal/consumer/server.go` to pass config to NewUseCaseWithDeps
  - Updated `cmd/consumer/main.go` to pass `cfg.CrawlLimits` to consumer.Config

---

## Phase 8: Testing

### 8.1 Unit Tests - Models

- [x] 8.1.1 Test `NewTransformOptionsFromConfig`

  - Test with default config values
  - Test with custom config values
  - Test hard limit validation

- [x] 8.1.2 Test `NewDryRunOptionsFromConfig`

  - Test dry-run specific values
  - Test DownloadMedia always false

- [x] 8.1.3 Test `ProjectState` methods
  - Test `IsCrawlComplete` with task-level
  - Test `CrawlProgressPercent` with item-level
  - Test `CrawlProgressPercent` fallback to task-level
  - Test `OverallProgressPercent`

### 8.2 Unit Tests - State

- [x] 8.2.1 Test state usecase methods
  - Test `SetTasksTotal`
  - Test `IncrementTasksDone`
  - Test `IncrementTasksErrors`
  - Test `IncrementItemsActualBy`
  - Test `IncrementItemsErrorsBy`
  - Test `CheckCompletion` with hybrid state

### 8.3 Unit Tests - Results

- [x] 8.3.1 Test `extractLimitInfoAndStats`

  - Test with enhanced response (limit_info + stats present)
  - Test with old response (fallback)
  - Test with empty payload

- [x] 8.3.2 Test `handleProjectResult`
  - Test task-level increment on success
  - Test task-level increment on failure
  - Test item-level increment from stats
  - Test platform limitation logging

### 8.4 Integration Tests

- [ ] 8.4.1 Test full dispatch flow

  - Test config values are used in dispatch
  - Test state is set correctly

- [ ] 8.4.2 Test full result handling flow
  - Test with mock enhanced crawler response
  - Test with mock old crawler response
  - Test webhook is sent with correct format

---

## Phase 9: Documentation

### 9.1 Update Documentation

- [x] 9.1.1 Update `README.md`

  - Add section about crawl limits configuration
  - Document env vars

- [x] 9.1.2 Update `document/collector-behavior.md`

  - Update state structure documentation
  - Update flow diagrams

- [x] 9.1.3 Update `openspec/project.md`
  - Add CrawlLimits to config section
  - Update state management section

---

## Dependencies

### External (must be done by other teams)

- [ ] Crawler (YouTube): Update response format with `limit_info` and `stats`
- [ ] Crawler (TikTok): Update response format with `limit_info` and `stats`
- [ ] Project Service: Update webhook handler for new format

### Deployment Order

1. **Collector** (this change) - with fallback logic
2. **Crawler** (YouTube + TikTok) - with enhanced response
3. **Project Service** - with new webhook handler

---

## Verification Checklist

### Pre-deployment

- [x] All unit tests pass
- [ ] Integration tests pass
- [x] Config values are correct in K8s ConfigMap
- [x] Env vars are documented

### Post-deployment

- [ ] Verify config is loaded correctly (check logs)
- [ ] Verify state is tracked correctly (check Redis)
- [ ] Verify webhook format is correct (check Project Service logs)
- [ ] Verify fallback works with old Crawler response
- [ ] Verify platform limitation is logged

### After Crawler Update

- [ ] Verify enhanced response is parsed correctly
- [ ] Verify `platform_limited` is logged when true
- [ ] Verify stats are accurate
