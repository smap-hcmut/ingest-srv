# event-infrastructure Specification Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
