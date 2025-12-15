# event-infrastructure Specification

## Purpose

Định nghĩa event-driven architecture cho SMAP Collector Service, bao gồm:

- Event consumption từ Project Service (`project.created`)
- Redis state management cho project execution tracking
- Progress webhook notification đến Project Service
- Event publishing cho downstream services (`data.collected`)

**Reference Document:** `document/event-drivent.md`

## Compliance Status

| Requirement                          | Status                    | Verified   |
| ------------------------------------ | ------------------------- | ---------- |
| SMAP Events Exchange Configuration   | ✅ Compliant              | 2025-12-06 |
| ProjectCreatedEvent Schema Support   | ✅ Compliant              | 2025-12-06 |
| Redis State Management               | ✅ Compliant              | 2025-12-06 |
| Progress Webhook Notification        | ✅ Compliant              | 2025-12-06 |
| Data Collected Event Publishing      | ⚠️ Crawler responsibility | -          |
| Backward Compatibility               | ✅ Compliant              | 2025-12-06 |
| Configuration                        | ✅ Compliant              | 2025-12-06 |
| External Dependencies Initialization | ✅ Compliant              | 2025-12-06 |

**Last Verified:** 2025-12-06 via `review-event-driven-compliance` proposal
## Requirements
### Requirement: SMAP Events Exchange Configuration

The Collector Service SHALL use the centralized `smap.events` topic exchange for receiving project execution events from Project Service.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

#### Scenario: Exchange declaration on startup

- **WHEN** the Collector Service starts
- **THEN** the service SHALL declare or verify the `smap.events` exchange exists with type `topic`
- **AND** the exchange SHALL be durable and not auto-delete
- **VERIFIED:** `internal/dispatcher/delivery/rabbitmq/constants.go` - `SMAPEventsExchangeArgs`

#### Scenario: Queue binding for project.created

- **WHEN** the Collector Service initializes its consumer
- **THEN** the service SHALL create queue `collector.project.created`
- **AND** bind it to `smap.events` exchange with routing key `project.created`
- **VERIFIED:** `internal/dispatcher/delivery/rabbitmq/consumer/project_consumer.go` - `ConsumeProjectEvents()`

---

### Requirement: ProjectCreatedEvent Schema Support

The Collector Service SHALL consume and process `ProjectCreatedEvent` messages following the standardized schema defined in the event-driven architecture document.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

#### Scenario: Parse ProjectCreatedEvent successfully

- **WHEN** a message arrives on `collector.project.created` queue
- **THEN** the service SHALL parse the message as `ProjectCreatedEvent` with fields:
  - `event_id` (string)
  - `timestamp` (RFC3339)
  - `payload.project_id` (string)
  - `payload.user_id` (string)
  - `payload.brand_name` (string)
  - `payload.brand_keywords` ([]string)
  - `payload.competitor_names` ([]string)
  - `payload.competitor_keywords_map` (map[string][]string)
  - `payload.date_range.from` (YYYY-MM-DD)
  - `payload.date_range.to` (YYYY-MM-DD)
- **VERIFIED:** `internal/models/event.go` - `ProjectCreatedEvent` struct

#### Scenario: Store project-user mapping

- **WHEN** a `ProjectCreatedEvent` is successfully parsed
- **THEN** the service SHALL store the mapping between `project_id` and `user_id`
- **AND** this mapping SHALL be used for progress notifications
- **VERIFIED:** `internal/state/usecase/state.go` - `StoreUserMapping()`

#### Scenario: Invalid event handling

- **WHEN** a message cannot be parsed as `ProjectCreatedEvent`
- **THEN** the service SHALL log the error with message details
- **AND** the service SHALL reject the message (no requeue)
- **VERIFIED:** `internal/dispatcher/delivery/rabbitmq/consumer/project_consumer.go` - `d.Ack(false)`

---

### Requirement: Redis State Management

The Collector Service SHALL update project execution state in Redis DB 1 using the standardized key schema `smap:proj:{projectID}` with **two-phase counters** for crawl and analyze progress.

#### Scenario: Initialize state with two-phase counters

- **WHEN** a `ProjectCreatedEvent` is received
- **THEN** the service SHALL initialize Redis hash with fields:
  - `status` = "INITIALIZING"
  - `crawl_total` = 0
  - `crawl_done` = 0
  - `crawl_errors` = 0
  - `analyze_total` = 0
  - `analyze_done` = 0
  - `analyze_errors` = 0

#### Scenario: Set crawl total and transition to PROCESSING

- **WHEN** the Collector determines the total number of items to crawl
- **THEN** the service SHALL execute `HSET smap:proj:{projectID} crawl_total {count}`
- **AND** the service SHALL execute `HSET smap:proj:{projectID} status PROCESSING`

#### Scenario: Increment crawl counters by batch size

- **WHEN** a crawl batch result is received with N items
- **THEN** the service SHALL execute `HINCRBY smap:proj:{projectID} crawl_done {N}` for successful items
- **OR** the service SHALL execute `HINCRBY smap:proj:{projectID} crawl_errors {N}` for failed items

#### Scenario: Auto-set analyze total on successful crawl

- **WHEN** a successful crawl batch result is received with N items
- **THEN** the service SHALL execute `HINCRBY smap:proj:{projectID} analyze_total {N}`

#### Scenario: Increment analyze counters by batch counts

- **WHEN** an analyze batch result is received with success_count and error_count
- **THEN** the service SHALL execute `HINCRBY smap:proj:{projectID} analyze_done {success_count}`
- **AND** the service SHALL execute `HINCRBY smap:proj:{projectID} analyze_errors {error_count}`

#### Scenario: Two-phase completion detection

- **WHEN** checking for project completion
- **THEN** the service SHALL verify BOTH conditions:
  - `crawl_done + crawl_errors >= crawl_total` (crawl complete)
  - `analyze_done + analyze_errors >= analyze_total` (analyze complete)
- **AND** only when BOTH are true, the service SHALL set `status = DONE`

---

### Requirement: Progress Webhook Notification

The Collector Service SHALL notify Project Service of progress via webhook with **two-phase progress format**.

#### Scenario: Webhook request format with two-phase progress

- **WHEN** the service needs to notify progress
- **THEN** the service SHALL send POST request to `{PROJECT_SERVICE_URL}/internal/progress/callback`
- **AND** include JSON body with fields:
  - `project_id` (string)
  - `user_id` (string)
  - `status` (string: INITIALIZING, PROCESSING, DONE, FAILED)
  - `crawl.total` (int64)
  - `crawl.done` (int64)
  - `crawl.errors` (int64)
  - `crawl.progress_percent` (float64)
  - `analyze.total` (int64)
  - `analyze.done` (int64)
  - `analyze.errors` (int64)
  - `analyze.progress_percent` (float64)
  - `overall_progress_percent` (float64)

#### Scenario: Overall progress calculation

- **WHEN** calculating overall progress percent
- **THEN** the service SHALL compute `(crawl_progress + analyze_progress) / 2`
- **WHERE** `crawl_progress = (crawl_done + crawl_errors) / crawl_total * 100`
- **AND** `analyze_progress = (analyze_done + analyze_errors) / analyze_total * 100`

---

### Requirement: Data Collected Event Publishing

The Crawler (Worker) Service SHALL publish `data.collected` event after successfully storing crawled data to MinIO.

> **Note:** This requirement applies to Crawler/Worker services (YouTube, TikTok), NOT Collector Service. Collector only dispatches tasks; Crawlers upload data to MinIO and publish events.

#### Scenario: Publish data.collected event

- **WHEN** crawled data is successfully uploaded to MinIO by a Crawler
- **THEN** the Crawler SHALL publish to `smap.events` exchange with routing key `data.collected`
- **AND** the event payload SHALL include:
  - `event_id` (string)
  - `timestamp` (RFC3339)
  - `payload.project_id` (string)
  - `payload.user_id` (string)
  - `payload.minio_path` (string) - Path to batch data in MinIO
  - `payload.item_count` (int) - Number of items in batch
  - `payload.platform` (string) - youtube or tiktok

---

### Requirement: Backward Compatibility with CrawlRequest

The Collector Service SHALL maintain backward compatibility with the existing `CrawlRequest` schema during the migration period.

#### Scenario: Detect message schema

- **WHEN** a message arrives on the inbound queue
- **THEN** the service SHALL attempt to parse as `ProjectCreatedEvent` first
- **AND** if parsing fails, the service SHALL attempt to parse as `CrawlRequest`

#### Scenario: Process legacy CrawlRequest

- **WHEN** a message is successfully parsed as `CrawlRequest`
- **THEN** the service SHALL process it using the existing dispatcher logic
- **AND** the service SHALL log a deprecation warning

---

### Requirement: Configuration for Event-Driven Architecture

The Collector Service SHALL support configuration for the event-driven architecture components.

#### Scenario: Redis state configuration

- **WHEN** the service starts
- **THEN** the service SHALL read configuration:
  - `REDIS_HOST` - Redis server address
  - `REDIS_STATE_DB` - Database number for state (default: 1)

#### Scenario: Project service configuration

- **WHEN** the service starts
- **THEN** the service SHALL read configuration:
  - `PROJECT_SERVICE_URL` - Base URL for Project Service
  - `PROJECT_INTERNAL_KEY` - Internal API key for authentication

### Requirement: External Dependencies Initialization

The Collector Service SHALL initialize all external dependencies (Redis, Webhook Client) in the cmd layer.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

#### Scenario: Redis initialization in cmd

- **WHEN** the consumer service starts
- **THEN** Redis client SHALL be initialized in cmd/consumer/main.go
- **AND** connection failure SHALL cause immediate service termination
- **VERIFIED:** `cmd/consumer/main.go` - `pkgRedis.Connect()` with `l.Fatalf()`

#### Scenario: Server receives initialized dependencies

- **WHEN** the server.Run() is called
- **THEN** all dependencies (StateUseCase, WebhookClient) SHALL already be initialized
- **AND** server SHALL NOT contain conditional initialization logic
- **VERIFIED:** `internal/consumer/server.go` - receives `Config` with pre-initialized `RedisClient`

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

### Requirement: Analyze Result Handler

The Collector Service SHALL consume and process analyze results from Analytics Service to track analyze phase progress.

#### Scenario: Route analyze result messages

- **WHEN** a message arrives with `task_type: "analyze_result"`
- **THEN** the service SHALL route to `handleAnalyzeResult()` handler

#### Scenario: Parse analyze result payload

- **WHEN** processing an analyze result message
- **THEN** the service SHALL extract fields:
  - `project_id` (string)
  - `job_id` (string)
  - `task_type` (string) = "analyze_result"
  - `batch_size` (int)
  - `success_count` (int)
  - `error_count` (int)

#### Scenario: Update analyze counters from result

- **WHEN** an analyze result is successfully parsed
- **THEN** the service SHALL increment `analyze_done` by `success_count`
- **AND** the service SHALL increment `analyze_errors` by `error_count`
- **AND** the service SHALL call progress webhook
- **AND** the service SHALL check for completion

---

### Requirement: Simplified Status Model

The Collector Service SHALL use a simplified status model with single PROCESSING status for both crawl and analyze phases.

#### Scenario: Status values

- **THE** system SHALL support exactly four status values:
  - `INITIALIZING` - Project just started, no tasks yet
  - `PROCESSING` - Crawl and/or analyze in progress
  - `DONE` - Both crawl and analyze phases complete
  - `FAILED` - Fatal error occurred

#### Scenario: Status transition from INITIALIZING

- **WHEN** `SetCrawlTotal()` is called with total > 0
- **THEN** the service SHALL transition status from `INITIALIZING` to `PROCESSING`

#### Scenario: Status transition to DONE

- **WHEN** both crawl and analyze phases are complete
- **THEN** the service SHALL transition status to `DONE`

