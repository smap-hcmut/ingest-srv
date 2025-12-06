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

#### Scenario: Webhook on platform completion

- **WHEN** a platform worker completes crawling all items
- **THEN** the service SHALL call the progress webhook with current project state
- **AND** the service SHALL always update Redis state before calling webhook

#### Scenario: Webhook client initialization

- **WHEN** the consumer service starts
- **THEN** the webhook client SHALL be initialized in cmd layer (not server layer)
- **AND** initialization failure SHALL cause service startup to fail

#### Scenario: Webhook failure handling

- **WHEN** the webhook call fails
- **THEN** the service SHALL log the error
- **AND** the service SHALL continue updating Redis state
- **AND** the service SHALL retry with exponential backoff (optional)

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

#### Scenario: Redis initialization in cmd

- **WHEN** the consumer service starts
- **THEN** Redis client SHALL be initialized in cmd/consumer/main.go
- **AND** connection failure SHALL cause immediate service termination

#### Scenario: Server receives initialized dependencies

- **WHEN** the server.Run() is called
- **THEN** all dependencies (StateUseCase, WebhookClient) SHALL already be initialized
- **AND** server SHALL NOT contain conditional initialization logic
