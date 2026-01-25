# Implementation Tasks

## 1. Event Types & Models
- [x] 1.1 Create `ProjectCreatedEvent` struct in `internal/models/event.go`
  - Include `EventID`, `Timestamp`, `Payload` with all required fields
  - Add `DateRange` struct with `From`, `To` fields
- [x] 1.2 Create `DataCollectedEvent` struct for publishing
  - Include `EventID`, `Timestamp`, `Payload` with MinIO path info
- [x] 1.3 Add `ProjectStatus` constants (INITIALIZING, CRAWLING, PROCESSING, DONE, FAILED)
- [x] 1.4 Create event transformation functions
  - `TransformProjectEventToRequests()` - Convert ProjectCreatedEvent to []CrawlRequest

## 2. Redis State Module
- [x] 2.1 Create `internal/state/uc_interface.go`
  - Define `StateUseCase` interface with methods: `UpdateTotal`, `IncrementDone`, `IncrementErrors`, `UpdateStatus`, `GetState`
- [x] 2.2 Create `internal/state/uc_types.go`
  - Define `ProjectState` struct with Status, Total, Done, Errors fields
- [x] 2.3 Create `internal/state/uc_errors.go`
  - Define state-specific errors
- [x] 2.4 Create `internal/state/repository/repo_interface.go`
  - Define `StateRepository` interface for Redis operations
- [x] 2.5 Create `internal/state/repository/redis/state_repo.go`
  - Implement Redis HSET, HINCRBY operations
  - Key schema: `smap:proj:{projectID}`
- [x] 2.6 Create `internal/state/usecase/state_uc.go`
  - Implement business logic for state transitions
  - Add completion check: done + errors >= total

## 3. Webhook Client Module
- [x] 3.1 Create `internal/webhook/client_interface.go`
  - Define `WebhookClient` interface with `NotifyProgress` method
- [x] 3.2 Create `internal/webhook/types.go`
  - Define `ProgressRequest` struct
- [x] 3.3 Create `internal/webhook/client/http_client.go`
  - Implement HTTP client for `/internal/progress/callback`
  - Include `X-Internal-Key` header
- [x] 3.4 Create `internal/webhook/client/throttler.go`
  - Implement in-memory throttling with configurable interval
  - Add cleanup goroutine for stale entries

## 4. Configuration Updates
- [x] 4.1 Update `config/config.go`
  - Add `RedisStateConfig` with Host, DB fields
  - Add `WebhookConfig` with ThrottleInterval field
- [x] 4.2 Update `template.env`
  - Add `REDIS_STATE_HOST`, `REDIS_STATE_DB`
  - Add `WEBHOOK_THROTTLE_INTERVAL`

## 5. RabbitMQ Infrastructure Updates
- [x] 5.1 Update `internal/dispatcher/delivery/rabbitmq/constants.go`
  - Add `ExchangeSMAPEvents = "smap.events"`
  - Add `QueueProjectCreated = "collector.project.created"`
  - Add `RoutingKeyProjectCreated = "project.created"`
  - Add `RoutingKeyDataCollected = "data.collected"`
- [x] 5.2 Create new consumer for `smap.events` exchange
  - `internal/dispatcher/delivery/rabbitmq/consumer/project_consumer.go`
  - Bind to `project.created` routing key
- [x] 5.3 Create producer for `data.collected` event
  - `internal/dispatcher/delivery/rabbitmq/producer/event_producer.go`

## 6. Dispatcher UseCase Updates
- [x] 6.1 Update `internal/dispatcher/uc_interface.go`
  - Add `HandleProjectCreatedEvent(ctx, event) error` method
- [x] 6.2 Create `internal/dispatcher/usecase/project_event_uc.go`
  - Implement event handling logic
  - Transform event to CrawlRequests
  - Dispatch to workers
- [x] 6.3 Add schema detection in consumer
  - Try parse as ProjectCreatedEvent first
  - Fallback to CrawlRequest with deprecation warning

## 7. Progress Tracking Integration
- [x] 7.1 Update dispatcher to call state updates
  - Call `UpdateTotal` when total items determined
  - Call `IncrementDone` after each successful crawl
  - Call `IncrementErrors` after each failed crawl
- [x] 7.2 Integrate webhook notifications
  - Immediate notify on total set
  - Immediate notify on status change (DONE, FAILED)
  - Throttled notify during crawling
- [x] 7.3 Add project-user mapping storage
  - Store in Redis or in-memory map
  - Used for webhook notifications

## 8. Consumer Server Updates
- [x] 8.1 Update `internal/consumer/new.go`
  - Add Redis state client dependency
  - Add webhook client dependency
- [x] 8.2 Update `internal/consumer/server.go`
  - Initialize Redis state client
  - Initialize webhook client
  - Start new project event consumer
- [x] 8.3 Update `cmd/consumer/main.go`
  - Add Redis connection for state DB

## 9. Data Collected Event Publishing
- [x] 9.1 Update results usecase to publish `data.collected`
  - After successful MinIO upload
  - Include project_id, user_id, minio_path, item_count, platform
- [x] 9.2 Add event producer to results module
  - Added `EventPublisher` interface to `internal/results/uc_interface.go`
  - Implemented `PublishDataCollected` in `internal/results/usecase/event_publisher.go`
  - Added `NewUseCaseWithEventProducer` constructor in `internal/results/usecase/new.go`
  - Reuses `EventProducer` from `internal/dispatcher/delivery/rabbitmq/producer/`

## 10. Testing
- [x] 10.1 Unit tests for event transformation
  - `internal/models/event_transform_test.go`
- [x] 10.2 Unit tests for state usecase
  - `internal/state/usecase/state_uc_test.go`
- [x] 10.3 Unit tests for webhook client with throttling
  - `internal/webhook/client/throttler_test.go`
- [ ] 10.4 Integration tests for new consumer (deferred - requires running infrastructure)
- [ ] 10.5 Integration tests for progress flow (deferred - requires running infrastructure)

## 11. Documentation
- [x] 11.1 Update README with new architecture
  - Added Event-Driven Architecture section
  - Added SMAP Events Exchange documentation
  - Added Redis State Management documentation
- [x] 11.2 Update `document/event-drivent.md` with Collector implementation details
  - Added Section 14: Collector Service Implementation Status
  - Added Go code examples for all components
  - Updated checklist with implementation status
- [x] 11.3 Migration guide (included in event-drivent.md Section 14)
