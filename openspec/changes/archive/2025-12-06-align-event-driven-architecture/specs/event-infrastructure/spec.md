# Event Infrastructure Specification

## ADDED Requirements

### Requirement: SMAP Events Exchange Configuration
The Collector Service SHALL use the centralized `smap.events` topic exchange for receiving project execution events from Project Service.

#### Scenario: Exchange declaration on startup
- **WHEN** the Collector Service starts
- **THEN** the service SHALL declare or verify the `smap.events` exchange exists with type `topic`
- **AND** the exchange SHALL be durable and not auto-delete

#### Scenario: Queue binding for project.created
- **WHEN** the Collector Service initializes its consumer
- **THEN** the service SHALL create queue `collector.project.created`
- **AND** bind it to `smap.events` exchange with routing key `project.created`

---

### Requirement: ProjectCreatedEvent Schema Support
The Collector Service SHALL consume and process `ProjectCreatedEvent` messages following the standardized schema defined in the event-driven architecture document.

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

#### Scenario: Store project-user mapping
- **WHEN** a `ProjectCreatedEvent` is successfully parsed
- **THEN** the service SHALL store the mapping between `project_id` and `user_id`
- **AND** this mapping SHALL be used for progress notifications

#### Scenario: Invalid event handling
- **WHEN** a message cannot be parsed as `ProjectCreatedEvent`
- **THEN** the service SHALL log the error with message details
- **AND** the service SHALL reject the message (no requeue)

---

### Requirement: Redis State Management
The Collector Service SHALL update project execution state in Redis DB 1 using the standardized key schema `smap:proj:{projectID}`.

#### Scenario: Update total items count
- **WHEN** the Collector determines the total number of items to crawl
- **THEN** the service SHALL execute `HSET smap:proj:{projectID} total {count}`
- **AND** the service SHALL execute `HSET smap:proj:{projectID} status CRAWLING`

#### Scenario: Increment done counter
- **WHEN** an item is successfully crawled
- **THEN** the service SHALL execute `HINCRBY smap:proj:{projectID} done 1`

#### Scenario: Increment errors counter
- **WHEN** an item fails to crawl
- **THEN** the service SHALL execute `HINCRBY smap:proj:{projectID} errors 1`

#### Scenario: Update status to DONE
- **WHEN** all items have been processed (done + errors >= total)
- **THEN** the service SHALL execute `HSET smap:proj:{projectID} status DONE`

#### Scenario: Update status to FAILED
- **WHEN** a fatal error occurs during crawling
- **THEN** the service SHALL execute `HSET smap:proj:{projectID} status FAILED`

---

### Requirement: Progress Webhook Notification
The Collector Service SHALL notify Project Service of crawling progress via the internal webhook endpoint `/internal/progress/callback`.

#### Scenario: Webhook request format
- **WHEN** the service needs to notify progress
- **THEN** the service SHALL send POST request to `{PROJECT_SERVICE_URL}/internal/progress/callback`
- **AND** include header `X-Internal-Key: {INTERNAL_KEY}`
- **AND** include JSON body with fields:
  - `project_id` (string)
  - `user_id` (string)
  - `status` (string: CRAWLING, DONE, FAILED)
  - `total` (int64)
  - `done` (int64)
  - `errors` (int64)

#### Scenario: Immediate webhook on total set
- **WHEN** the total items count is determined
- **THEN** the service SHALL immediately call the progress webhook

#### Scenario: Immediate webhook on completion
- **WHEN** the crawling status changes to DONE or FAILED
- **THEN** the service SHALL immediately call the progress webhook

#### Scenario: Throttled webhook during crawling
- **WHEN** items are being crawled
- **THEN** the service SHALL call the progress webhook at most once every 5 seconds
- **AND** the service SHALL always update Redis state regardless of throttling

#### Scenario: Webhook failure handling
- **WHEN** the webhook call fails
- **THEN** the service SHALL log the error
- **AND** the service SHALL continue updating Redis state
- **AND** the service SHALL retry with exponential backoff (optional)

---

### Requirement: Data Collected Event Publishing
The Collector Service SHALL publish `data.collected` event after successfully storing crawled data to MinIO.

#### Scenario: Publish data.collected event
- **WHEN** crawled data is successfully uploaded to MinIO
- **THEN** the service SHALL publish to `smap.events` exchange with routing key `data.collected`
- **AND** the event payload SHALL include:
  - `event_id` (string)
  - `timestamp` (RFC3339)
  - `payload.project_id` (string)
  - `payload.user_id` (string)
  - `payload.minio_path` (string)
  - `payload.item_count` (int)
  - `payload.platform` (string)

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

#### Scenario: Webhook throttle configuration
- **WHEN** the service starts
- **THEN** the service SHALL read configuration:
  - `WEBHOOK_THROTTLE_INTERVAL` - Minimum seconds between webhook calls (default: 5)
