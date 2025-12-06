## ADDED Requirements

### Requirement: Task Type Routing

The Collector Service SHALL route crawler results to appropriate handlers based on the `task_type` field in the result payload.

**Verification Status:** ✅ COMPLIANT (2025-12-06)
**Verified by:** Unit tests in `internal/results/usecase/result_routing_test.go`

#### Scenario: Route dry-run results

- **WHEN** a crawler result arrives with `task_type: "dryrun_keyword"`
- **THEN** the service SHALL route to `handleDryRunResult()` handler
- **AND** the service SHALL send callback to `/internal/dryrun/callback`
- **AND** the service SHALL NOT update Redis state
- **VERIFIED:** `TestHandleResult_RoutesDryRunCorrectly` - PASS

#### Scenario: Route project execution results

- **WHEN** a crawler result arrives with `task_type: "research_and_crawl"`
- **THEN** the service SHALL route to `handleProjectResult()` handler
- **AND** the service SHALL update Redis state (increment done or errors)
- **AND** the service SHALL send progress webhook to `/internal/progress/callback`
- **VERIFIED:** `TestHandleResult_RoutesProjectExecutionCorrectly` - PASS

#### Scenario: Backward compatibility for missing task_type

- **WHEN** a crawler result arrives without `task_type` field
- **THEN** the service SHALL default to `handleDryRunResult()` handler
- **AND** the service SHALL log a warning about missing task_type
- **AND** the service SHALL NOT crash or reject the message
- **VERIFIED:** `TestHandleResult_BackwardCompatibility` - PASS

#### Scenario: Extract task_type from payload

- **WHEN** the service receives a crawler result
- **THEN** the service SHALL extract `task_type` from the first item in payload array
- **AND** the service SHALL use `meta.task_type` field from `CrawlerContent`
- **VERIFIED:** `TestExtractTaskType_DryRunKeyword`, `TestExtractTaskType_ResearchAndCrawl` - PASS

---

### Requirement: Error Code Handling

The Collector Service SHALL properly handle and propagate error information from crawler results.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

#### Scenario: Process error result for project execution

- **WHEN** a project execution result has `Success: false`
- **THEN** the service SHALL increment the errors counter in Redis
- **AND** the service SHALL log the error
- **AND** the service SHALL continue processing
- **VERIFIED:** `TestHandleResult_ProjectExecutionWithErrors` - PASS

#### Scenario: Error fields in result metadata

- **WHEN** a crawler result item has an error
- **THEN** the item SHALL contain:
  - `meta.fetch_status: "error"`
  - `meta.fetch_error: "<error message>"`
- **VERIFIED:** `internal/results/types.go` - `FetchStatus`, `FetchError` fields exist

---

### Requirement: Project ID Extraction

The Collector Service SHALL extract project_id from job_id for project execution results.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

#### Scenario: Extract project_id from job_id

- **WHEN** processing a project execution result
- **THEN** the service SHALL extract project_id from job_id format `{projectID}-brand-{index}`
- **AND** the service SHALL use the extracted project_id for Redis state updates
- **VERIFIED:** `TestExtractProjectID_WithBrandSuffix` - PASS

#### Scenario: Handle job_id without brand suffix

- **WHEN** job_id does not contain `-brand-` suffix
- **THEN** the service SHALL use the entire job_id as project_id
- **AND** the service SHALL NOT fail or crash
- **VERIFIED:** `TestExtractProjectID_WithoutBrandSuffix` - PASS

#### Scenario: Handle complex project_id with hyphens

- **WHEN** project_id itself contains hyphens (e.g., `proj-abc-123-brand-5`)
- **THEN** the service SHALL correctly extract `proj-abc-123` as project_id
- **VERIFIED:** `TestExtractProjectID_ComplexProjectID` - PASS

---

### Requirement: Completion Detection

The Collector Service SHALL detect when a project execution is complete and update status accordingly.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

#### Scenario: Check completion after each result

- **WHEN** a project execution result is processed
- **THEN** the service SHALL check if `done + errors >= total`
- **AND** if complete, the service SHALL update status to DONE
- **AND** the service SHALL send completion webhook
- **VERIFIED:** `internal/results/usecase/result.go` - `CheckAndUpdateCompletion()` called after each result

#### Scenario: Send completion notification

- **WHEN** project status changes to DONE
- **THEN** the service SHALL call `NotifyCompletion()` webhook
- **AND** the webhook payload SHALL include final state (total, done, errors)
- **VERIFIED:** `internal/webhook/usecase/webhook.go` - `NotifyCompletion()` implementation
