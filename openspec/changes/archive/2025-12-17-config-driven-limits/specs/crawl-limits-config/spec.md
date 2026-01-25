# Spec: Crawl Limits Configuration

## Overview

Capability này định nghĩa cách Collector Service quản lý crawl limits thông qua configuration thay vì hardcode, và track state ở cả task-level và item-level.

---

## ADDED Requirements

### Requirement: Config-Driven Crawl Limits

THE Collector Service SHALL load crawl limit values from environment variables instead of hardcoded values.

#### Scenario: Load default production limits from config

**Given** the Collector Service starts
**When** it loads configuration
**Then** it SHALL read `DEFAULT_LIMIT_PER_KEYWORD` from environment (default: 50)
**And** it SHALL read `DEFAULT_MAX_COMMENTS` from environment (default: 100)
**And** it SHALL read `DEFAULT_MAX_ATTEMPTS` from environment (default: 3)

#### Scenario: Load dry-run limits from config

**Given** the Collector Service starts
**When** it loads configuration
**Then** it SHALL read `DRYRUN_LIMIT_PER_KEYWORD` from environment (default: 3)
**And** it SHALL read `DRYRUN_MAX_COMMENTS` from environment (default: 5)

#### Scenario: Load hard limits from config

**Given** the Collector Service starts
**When** it loads configuration
**Then** it SHALL read `MAX_LIMIT_PER_KEYWORD` from environment (default: 500)
**And** it SHALL read `MAX_MAX_COMMENTS` from environment (default: 1000)

#### Scenario: Load feature flags from config

**Given** the Collector Service starts
**When** it loads configuration
**Then** it SHALL read `INCLUDE_COMMENTS` from environment (default: true)
**And** it SHALL read `DOWNLOAD_MEDIA` from environment (default: false)

---

### Requirement: Use Config in Dispatch

THE Dispatcher SHALL use configured limit values when creating crawl tasks.

#### Scenario: Use production limits for research_and_crawl tasks

**Given** a ProjectCreatedEvent is received
**When** the Dispatcher creates CrawlRequest tasks
**Then** it SHALL set `limit_per_keyword` to the configured `DEFAULT_LIMIT_PER_KEYWORD` value
**And** it SHALL set `max_comments` to the configured `DEFAULT_MAX_COMMENTS` value
**And** it SHALL set `max_attempts` to the configured `DEFAULT_MAX_ATTEMPTS` value

#### Scenario: Use dry-run limits for dryrun_keyword tasks

**Given** a dry-run task is being processed
**When** the Dispatcher maps the payload
**Then** it SHALL set `limit_per_keyword` to the configured `DRYRUN_LIMIT_PER_KEYWORD` value
**And** it SHALL set `max_comments` to the configured `DRYRUN_MAX_COMMENTS` value
**And** it SHALL set `download_media` to false regardless of config

---

### Requirement: Hybrid State Tracking

THE Collector Service SHALL track project state at both task-level and item-level.

#### Scenario: Initialize hybrid state on dispatch

**Given** a ProjectCreatedEvent is processed
**When** the Dispatcher sets initial state
**Then** it SHALL set `tasks_total` to the number of tasks dispatched
**And** it SHALL set `items_expected` to `tasks_total × limit_per_keyword`
**And** it SHALL set status to `PROCESSING`

#### Scenario: Update task-level state on crawler response

**Given** a CrawlerResult is received
**When** the result is successful
**Then** it SHALL increment `tasks_done` by 1
**When** the result is failed
**Then** it SHALL increment `tasks_errors` by 1

#### Scenario: Update item-level state on crawler response

**Given** a CrawlerResult is received with stats
**When** the stats indicate successful items
**Then** it SHALL increment `items_actual` by `stats.successful`
**And** it SHALL increment `analyze_total` by `stats.successful`
**When** the stats indicate failed items
**Then** it SHALL increment `items_errors` by `stats.failed`

---

### Requirement: Task-Level Completion Check

THE Collector Service SHALL determine crawl completion based on task-level counters.

#### Scenario: Crawl phase complete

**Given** a project with `tasks_total = 4`
**When** `tasks_done + tasks_errors >= tasks_total`
**Then** the crawl phase SHALL be considered complete

#### Scenario: Crawl phase incomplete

**Given** a project with `tasks_total = 4`
**When** `tasks_done + tasks_errors < tasks_total`
**Then** the crawl phase SHALL NOT be considered complete

---

### Requirement: Item-Level Progress Display

THE Collector Service SHALL calculate progress percentage based on item-level counters with fallback.

#### Scenario: Calculate progress from items

**Given** a project with `items_expected = 200` and `items_actual = 150`
**When** calculating crawl progress percentage
**Then** it SHALL return `(items_actual + items_errors) / items_expected * 100`

#### Scenario: Fallback to task-level progress

**Given** a project with `items_expected = 0` and `tasks_total = 4`
**When** calculating crawl progress percentage
**Then** it SHALL return `(tasks_done + tasks_errors) / tasks_total * 100`

---

### Requirement: Enhanced Crawler Response Parsing

THE Collector Service SHALL parse enhanced crawler response format with fallback to old format.

#### Scenario: Parse enhanced response with limit_info and stats

**Given** a CrawlerResult with `limit_info` and `stats` fields
**When** extracting statistics
**Then** it SHALL use `stats.successful` for successful item count
**And** it SHALL use `stats.failed` for failed item count
**And** it SHALL use `limit_info.platform_limited` for limitation detection

#### Scenario: Fallback to counting payload items

**Given** a CrawlerResult without `limit_info` or `stats` fields
**When** extracting statistics
**Then** it SHALL count items in `payload` array as successful count
**And** it SHALL set failed count to 0
**And** it SHALL set `platform_limited` to false

---

### Requirement: Platform Limitation Logging

THE Collector Service SHALL log warnings when platform limitation is detected.

#### Scenario: Log platform limitation

**Given** a CrawlerResult with `limit_info.platform_limited = true`
**When** processing the result
**Then** it SHALL log a WARNING with:

- `project_id`
- `limit_info.requested_limit`
- `limit_info.total_found`

---

### Requirement: Hybrid Webhook Format

THE Collector Service SHALL send progress webhooks with both task-level and item-level progress.

#### Scenario: Send progress webhook with hybrid format

**Given** a project state is updated
**When** sending progress webhook
**Then** it SHALL include `tasks` object with:

- `total`: tasks_total
- `done`: tasks_done
- `errors`: tasks_errors
- `percent`: calculated percentage
  **And** it SHALL include `items` object with:
- `expected`: items_expected
- `actual`: items_actual
- `errors`: items_errors
- `percent`: calculated percentage
  **And** it SHALL include `analyze` object (unchanged)
  **And** it SHALL include `overall_progress_percent`

---

## MODIFIED Requirements

### Requirement: State Interface Methods

THE State UseCase interface SHALL be extended with new methods for hybrid tracking.

#### Scenario: Set tasks total with items expected

**Given** a new project is being initialized
**When** calling `SetTasksTotal(ctx, projectID, tasksTotal, itemsExpected)`
**Then** it SHALL set `tasks_total` field in Redis
**And** it SHALL set `items_expected` field in Redis
**And** it SHALL set `status` to `PROCESSING`

#### Scenario: Increment task counters

**Given** a crawler result is received
**When** calling `IncrementTasksDone(ctx, projectID)`
**Then** it SHALL atomically increment `tasks_done` by 1
**When** calling `IncrementTasksErrors(ctx, projectID)`
**Then** it SHALL atomically increment `tasks_errors` by 1

#### Scenario: Increment item counters

**Given** a crawler result is received with stats
**When** calling `IncrementItemsActualBy(ctx, projectID, count)`
**Then** it SHALL atomically increment `items_actual` by count
**When** calling `IncrementItemsErrorsBy(ctx, projectID, count)`
**Then** it SHALL atomically increment `items_errors` by count

---

## REMOVED Requirements

### Requirement: Hardcoded Default Values

THE Collector Service SHALL NOT use hardcoded limit values in code.

#### Scenario: No hardcoded limits in DefaultTransformOptions

**Given** the `DefaultTransformOptions()` function exists
**When** it is called
**Then** it SHALL be marked as deprecated
**And** callers SHALL use `NewTransformOptionsFromConfig()` instead

#### Scenario: No hardcoded limits in dry-run handlers

**Given** a dry-run task is being processed
**When** mapping the payload
**Then** it SHALL NOT use hardcoded values like `LimitPerKeyword = 3`
**And** it SHALL use configured values from `CrawlLimitsConfig`

---

## Related Capabilities

- `event-infrastructure` - State management in Redis
- `http-server` - Webhook sending (if applicable)

## External Dependencies

- **Crawler Services**: Must update response format with `limit_info` and `stats`
- **Project Service**: Must handle new webhook format with `tasks` and `items`
