## MODIFIED Requirements

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

The Collector Service SHALL update project execution state in Redis DB 1 using the standardized key schema `smap:proj:{projectID}`.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

#### Scenario: Update total items count

- **WHEN** the Collector determines the total number of items to crawl
- **THEN** the service SHALL execute `HSET smap:proj:{projectID} total {count}`
- **AND** the service SHALL execute `HSET smap:proj:{projectID} status CRAWLING`
- **VERIFIED:** `internal/state/usecase/state.go` - `UpdateTotal()`

#### Scenario: Increment done counter

- **WHEN** an item is successfully crawled
- **THEN** the service SHALL execute `HINCRBY smap:proj:{projectID} done 1`
- **VERIFIED:** `internal/state/usecase/state.go` - `IncrementDone()`

#### Scenario: Increment errors counter

- **WHEN** an item fails to crawl
- **THEN** the service SHALL execute `HINCRBY smap:proj:{projectID} errors 1`
- **VERIFIED:** `internal/state/usecase/state.go` - `IncrementErrors()`

#### Scenario: Update status to DONE

- **WHEN** all items have been processed (done + errors >= total)
- **THEN** the service SHALL execute `HSET smap:proj:{projectID} status DONE`
- **VERIFIED:** `internal/state/usecase/state.go` - `CheckAndUpdateCompletion()`

#### Scenario: Update status to FAILED

- **WHEN** a fatal error occurs during crawling
- **THEN** the service SHALL execute `HSET smap:proj:{projectID} status FAILED`
- **VERIFIED:** `internal/state/usecase/state.go` - `UpdateStatus()`

---

### Requirement: Progress Webhook Notification

The Collector Service SHALL notify Project Service of crawling progress via the internal webhook endpoint `/internal/progress/callback`.

**Verification Status:** ✅ COMPLIANT (2025-12-06)

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
- **VERIFIED:** `pkg/project/client.go` - `SendProgressCallback()`

#### Scenario: Immediate webhook on total set

- **WHEN** the total items count is determined
- **THEN** the service SHALL immediately call the progress webhook
- **VERIFIED:** `internal/dispatcher/usecase/project_event_uc.go` - `notifyProgress()` after `UpdateTotal()`

#### Scenario: Immediate webhook on completion

- **WHEN** the crawling status changes to DONE or FAILED
- **THEN** the service SHALL immediately call the progress webhook
- **VERIFIED:** `internal/webhook/usecase/webhook.go` - `NotifyCompletion()`

#### Scenario: Webhook on platform completion

- **WHEN** a platform worker completes crawling all items
- **THEN** the service SHALL call the progress webhook with current project state
- **AND** the service SHALL always update Redis state before calling webhook
- **VERIFIED:** Flow in `HandleProjectCreatedEvent()` - state update → webhook notify

#### Scenario: Webhook client initialization

- **WHEN** the consumer service starts
- **THEN** the webhook client SHALL be initialized in cmd layer (not server layer)
- **AND** initialization failure SHALL cause service startup to fail
- **VERIFIED:** `cmd/consumer/main.go` - Redis init with `l.Fatalf()`

#### Scenario: Webhook failure handling

- **WHEN** the webhook call fails
- **THEN** the service SHALL log the error
- **AND** the service SHALL continue updating Redis state
- **AND** the service SHALL retry with exponential backoff (optional)
- **VERIFIED:** `pkg/project/client.go` - exponential backoff with `delay *= 2`

---

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
