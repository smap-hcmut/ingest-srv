# Implementation Plan

## 1. Document Message Contracts (Priority: Confirm với Crawler & Analytics)

- [x] 1.1 Create final Crawler → Collector contract document (FLAT format v3.0)

  - Define CrawlerResultMessage struct với tất cả fields ở root level
  - Bỏ payload - Crawler push content trực tiếp sang Analytics
  - Document fields: success, task_type, job_id, platform, requested_limit, applied_limit, total_found, platform_limited, successful, failed, skipped, error_code, error_message
  - Add JSON examples cho các scenarios: full success, platform limited, no results, task failed
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 1.2 Create final Analytics → Collector contract document (FLAT format)

  - Define AnalyzeResultMessage struct với tất cả fields ở root level
  - Document: task_type="analyze_result", project_id, job_id, batch_size, success_count, error_count
  - Add JSON examples cho success và partial failure scenarios
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 1.3 Update `document/collector-crawler-contract.md` với final contracts
  - Consolidate cả 2 contracts vào 1 document chính thức (Version 3.0)
  - Add validation rules và error handling expectations
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.1, 2.2, 2.3, 2.4_

## 2. Update Message Contract Models in Code

- [x] 2.1 Create CrawlerResultMessage struct in `internal/models/crawler_result.go`

  - FLAT struct với tất cả fields ở root level (no nested objects)
  - Add validation methods: Validate(), ExtractProjectID(), IsErrorRetryable()
  - Add conversion methods: ToLimitInfo(), ToStats() for backward compatibility
  - Mark EnhancedCrawlerResult as DEPRECATED
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2.2 Review and update AnalyzeResultPayload struct in `internal/results/types.go`

  - Ensure all required fields match contract document
  - Add TaskType constant validation
  - _Requirements: 2.1, 2.2_

- [x] 2.3 Write tests for CrawlerResultMessage validation

  - Test validation, ExtractProjectID, JSON unmarshaling
  - **Property 1: Message Structure Validation**
  - **Validates: Requirements 1.1, 1.4**

- [x] 2.4 Write tests for AnalyzeResultPayload validation
  - **Property 5: Analyze Payload Validation**
  - **Validates: Requirements 2.1, 2.2, 2.3**

## 3. Checkpoint - Contract Documents Ready for Review

- [x] 3. Checkpoint
  - Contract documents complete (Version 3.0 - FLAT format)
  - Code models updated: CrawlerResultMessage, AnalyzeResultPayload
  - Ready to share với Crawler và Analytics teams for confirmation
  - Ask the user if questions arise.

## 4. Refactor handleProjectResult for FLAT Format

- [x] 4.1 Update `handleProjectResult` in `internal/results/usecase/result.go`

  - Parse CrawlerResultMessage (flat format) thay vì EnhancedCrawlerResult
  - Extract project_id từ msg.JobID using ExtractProjectID()
  - Read stats trực tiếp từ msg.Successful, msg.Failed (không cần extract)
  - Added `parseCrawlerResultMessage()` and `extractTaskTypeFromRoot()` functions
  - Updated `CrawlerResult` struct to include FLAT format fields at root level
  - _Requirements: 1.1, 1.2, 1.3, 3.1, 3.2, 3.3_

- [x] 4.2 Remove extractLimitInfoAndStats and fallbackLimitInfoAndStats

  - Replaced with `parseCrawlerResultMessage()` - direct field mapping
  - Không cần extract nested objects nữa - fields đã ở root level
  - Không cần fallback logic - flat format là required
  - _Requirements: 1.5 (simplified)_

- [x] 4.3 Update state counter increments

  - Use msg.Successful for items_actual increment
  - Use msg.Failed for items_errors increment
  - Auto-increment analyze_total by msg.Successful
  - _Requirements: 3.2, 3.3, 4.1_

- [x] 4.4 Add platform limitation logging

  - Log warning when msg.PlatformLimited is true
  - Include msg.RequestedLimit và msg.TotalFound in log
  - _Requirements: 3.1_

- [x] 4.5 Write unit tests for refactored handleProjectResult
  - Created `result_project_test.go` with tests for FLAT format
  - Test with valid CrawlerResultMessage
  - Test state counter updates
  - Test platform limitation logging
  - Test validation (missing task_type, job_id, platform)
  - Updated existing tests in `result_routing_test.go` and `result_extraction_test.go`
  - **Property 6: Counter Update Consistency**
  - **Validates: Requirements 2.4, 4.1, 4.2**

## 5. Checkpoint - Crawl Phase Refactor Complete

- [x] 5. Checkpoint
  - handleProjectResult refactored for flat format ✅
  - All crawl phase tests pass ✅
  - CrawlerResult struct updated to support FLAT format fields at root level
  - Legacy structs (EnhancedCrawlerResult, LimitInfo, CrawlStats, CrawlError) can now be removed
  - Ask the user if questions arise.

## 6. Review handleAnalyzeResult Logic

- [x] 6.1 Review `handleAnalyzeResult` in `internal/results/usecase/result.go`

  - Verified task_type validation equals "analyze_result" via extractTaskType()
  - Verified project_id extraction directly from payload via extractAnalyzePayload()
  - _Requirements: 2.2, 2.3_

- [x] 6.2 Review analyze counter increments

  - Verified analyze_done incremented by success_count via IncrementAnalyzeDoneBy()
  - Verified analyze_errors incremented by error_count via IncrementAnalyzeErrorsBy()
  - _Requirements: 4.2_

- [x] 6.3 Write unit tests for handleAnalyzeResult (if not exist)
  - Added TestHandleAnalyzeResult_MissingProjectID
  - Added TestHandleAnalyzeResult_NilPayload
  - Added TestHandleAnalyzeResult_InvalidPayloadFormat
  - Existing tests: TestHandleResult_RoutesAnalyzeResultCorrectly, TestHandleAnalyzeResult_OnlySuccessCount, TestHandleAnalyzeResult_OnlyErrorCount, TestHandleAnalyzeResult_ProjectCompletion
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

## 7. Review Completion Check Logic

- [x] 7.1 Review `IsCrawlComplete` in `internal/models/event.go`

  - Verified: tasks_done + tasks_errors >= tasks_total
  - Verified fallback to legacy crawl fields if tasks_total is 0
  - _Requirements: 3.4_

- [x] 7.2 Review `IsAnalyzeComplete` in `internal/models/event.go`

  - Verified: analyze_done + analyze_errors >= analyze_total
  - _Requirements: 4.3_

- [x] 7.3 Review `IsComplete` in `internal/models/event.go`

  - Verified: IsCrawlComplete() AND IsAnalyzeComplete()
  - _Requirements: 4.3, 4.4_

- [x] 7.4 Write property test for completion logic
  - Created `internal/models/event_test.go` with tests:
    - TestIsCrawlComplete_TaskLevel
    - TestIsCrawlComplete_LegacyFallback
    - TestIsAnalyzeComplete
    - TestIsComplete
  - **Property 7: Completion Logic**
  - **Validates: Requirements 3.4, 4.3, 4.4**

## 8. Review Legacy Field Updates

- [x] 8.1 Review `IncrementItemsActualBy` in `internal/state/usecase/state.go`

  - Verified also increments crawl_done for backward compatibility
  - _Requirements: 5.2_

- [x] 8.2 Review `IncrementItemsErrorsBy` in `internal/state/usecase/state.go`

  - Verified also increments crawl_errors for backward compatibility
  - _Requirements: 5.2_

- [x] 8.3 Write property test for legacy field consistency
  - Added tests in `internal/models/event_test.go`:
    - TestCrawlProgressPercent_ItemLevelPreferred (verifies fallback chain)
    - TestTasksProgressPercent
    - TestItemsProgressPercent
  - **Property 8: Legacy Field Consistency**
  - **Validates: Requirements 5.2**

## 9. Checkpoint - State Management Complete

- [x] 9. Checkpoint
  - All state management logic reviewed ✅
  - All tests pass ✅
  - handleAnalyzeResult, completion logic, legacy field updates verified

## 10. Review Progress Calculation and Webhook

- [x] 10.1 Review `CrawlProgressPercent` in `internal/models/event.go`

  - Verified uses item-level (items_actual/items_expected) when available
  - Verified fallback to task-level if items not tracked
  - _Requirements: 3.5_

- [x] 10.2 Review `OverallProgressPercent` in `internal/models/event.go`

  - Verified: (crawl_progress + analyze_progress) / 2
  - _Requirements: 6.2_

- [x] 10.3 Write property test for progress calculation

  - Added TestAnalyzeProgressPercent, TestOverallProgressPercent in event_test.go
  - **Property 9: Progress Calculation**
  - **Validates: Requirements 6.2, 6.3**

- [x] 10.4 Review `buildHybridProgressRequest` in `internal/results/usecase/result.go`

  - Verified includes both tasks and items progress (Tasks, Items structs)
  - Verified includes analyze phase fields (Analyze struct)
  - Verified includes legacy Crawl struct for backward compatibility
  - _Requirements: 6.1, 6.4_

- [x] 10.5 Write property test for progress webhook structure
  - Existing tests cover webhook structure via mock verification
  - **Property 10: Progress Webhook Structure**
  - **Validates: Requirements 6.1, 6.4**

## 11. Final Checkpoint - All Tests Pass

- [x] 11. Final Checkpoint
  - All tests pass ✅ (`go test ./...` successful)
  - All phases 1-10 completed

## 12. Final Documentation Update

- [x] 12.1 Update `document/collector-message-processing-detail.md`
  - Updated Section 3 (Message Contracts) with FLAT format v3.0
  - Updated Section 6 (Crawl Result Processing) with new flow diagram and code
  - Updated Section 11 with CrawlerResultMessage struct and validation
  - Sequence diagrams already present for both Crawler and Analytics flows
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 4.1, 4.2, 4.3, 4.4, 4.5, 6.1, 6.2, 6.3, 6.4_

---

## ✅ IMPLEMENTATION COMPLETE

All phases (1-12) have been completed:

- Contract documents updated to Version 3.0 FLAT format
- Code models updated (CrawlerResult, CrawlerResultMessage)
- handleProjectResult refactored for FLAT format
- handleAnalyzeResult reviewed and tested
- Completion logic verified (IsCrawlComplete, IsAnalyzeComplete, IsComplete)
- Legacy field updates verified (backward compatibility)
- Progress calculation verified
- All tests pass (`go test ./...`)
