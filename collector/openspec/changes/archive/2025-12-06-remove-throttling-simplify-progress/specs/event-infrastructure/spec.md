## ADDED Requirements

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

## MODIFIED Requirements

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
