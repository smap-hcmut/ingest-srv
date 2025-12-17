# Project Context

## Purpose

SMAP Collector Service (Dispatcher) là service trung tâm trong hệ thống SMAP data collection. Service này nhận các crawl request từ Project Service (qua RabbitMQ event `project.created`) hoặc từ legacy API, validate và phân phối các task chi tiết đến các platform-specific workers (YouTube, TikTok) thông qua RabbitMQ.

**Core Responsibilities:**

- Consume `project.created` events từ `smap.events` exchange
- Transform project events thành crawl tasks cho từng platform
- Dispatch tasks đến YouTube và TikTok workers
- Quản lý state tracking trong Redis (DB 1)
- Gọi progress webhook để notify Project Service
- Nhận results từ workers và cập nhật state

> **Note:** `data.collected` event được publish bởi Crawler (Worker) services, không phải Collector.

## Tech Stack

- **Language**: Go 1.25.4
- **Message Queue**: RabbitMQ (amqp091-go)
- **State Management**: Redis (go-redis/v9)
- **Logging**: Zap (go.uber.org/zap)
- **Configuration**: caarlos0/env/v9
- **Testing**: testify
- **Future**: MongoDB (for persistence)

## Project Conventions

### Code Style

- **Naming**:
  - Packages: lowercase, single word
  - Types: PascalCase (e.g., `CrawlRequest`, `CollectorTask`)
  - Functions: PascalCase for exported, camelCase for private
  - Constants: PascalCase for exported (e.g., `PlatformYouTube`)
- **File Organization**:
  - One main type per file
  - Test files: `*_test.go`
  - Constructor functions: `new.go` hoặc `New*` functions
- **Error Handling**:
  - Return errors explicitly, không panic trừ khi fatal
  - Use custom error types trong `*_errors.go` files
  - Context-aware logging với `log.Logger` interface
- **Comments**:
  - Vietnamese comments cho business logic
  - English comments cho technical details
  - Package-level comments mô tả purpose

### Architecture Patterns

**Clean Architecture Layers:**

```
cmd/consumer/          # Entry point
  ↓
internal/consumer/     # Server orchestration
  ↓
internal/dispatcher/   # Core domain
  ├── delivery/        # Transport (RabbitMQ consumers/producers)
  ├── usecase/         # Business logic (Dispatch, Map, Event handling)
  └── models/          # Domain entities
  ↓
pkg/                   # Infrastructure (Logger, RabbitMQ, Redis, etc.)
```

**Key Patterns:**

- **Dependency Injection**: Tất cả dependencies được inject qua constructors (`New*` functions)
- **Interface Segregation**: Mỗi layer định nghĩa interfaces riêng (e.g., `dispatcher.UseCase`, `state.UseCase`)
- **Strategy Pattern**: Payload mapping dựa trên Platform + TaskType
- **Factory Pattern**: `NewUseCase`, `NewUseCaseWithDeps` cho optional dependencies
- **Event-Driven Architecture**:
  - Consume: `project.created` từ `smap.events` exchange
  - State updates: Redis hash `smap:proj:{projectID}`
  - Progress notifications: HTTP webhook đến Project Service
  - Note: `data.collected` được publish bởi Crawler services

**Module Structure:**

- `internal/dispatcher/`: Core dispatch logic, event handling
- `internal/state/`: Redis state management (status, progress tracking)
- `internal/results/`: Result processing từ workers
- `internal/webhook/`: Progress webhook client
- `internal/models/`: Shared domain models (Task, Event, State)
- `pkg/`: Reusable utilities (log, rabbitmq, redis, etc.)

### Testing Strategy

- **Unit Tests**:
  - Test files: `*_test.go` trong cùng package
  - Mock interfaces: Manual mocks (e.g., `mockStateRepository`)
  - Testify: `assert`, `require` packages
- **Test Coverage**:
  - Focus on business logic (usecase layer)
  - Mock external dependencies (RabbitMQ, Redis, HTTP clients)
- **Test Examples**:
  - `pkg/compressor/compressor_test.go`: Compression/decompression
  - `internal/state/usecase/state_uc_test.go`: State management với mocks
  - `internal/models/event_transform_test.go`: Event transformation

### Git Workflow

- **Branching**: Feature branches từ `master-collector`
- **Commits**: Conventional commits với clear messages
- **Code Review**: Required trước khi merge
- **OpenSpec**:
  - Changes trong `openspec/changes/` cho proposals
  - Specs trong `openspec/specs/` cho deployed capabilities
  - Archive completed changes vào `openspec/changes/archive/`

## Domain Context

**SMAP System Overview:**
SMAP là hệ thống data collection và analysis cho social media platforms (YouTube, TikTok). Flow chính:

1. **Project Creation** (Project Service): User tạo project với keywords
2. **Project Execution** (Project Service): Publish `project.created` event
3. **Task Dispatch** (Collector Service): Transform project → crawl tasks → workers
4. **Data Collection** (Workers): Crawl data từ platforms, upload MinIO
5. **Data Event** (Workers): Publish `data.collected` event
6. **Analysis** (Analytics Service): Fetch data từ MinIO và process

**Event-Driven Choreography:**

- Exchange: `smap.events` (topic)
- Routing Keys:
  - `project.created`: Project Service → Collector Service
  - `data.collected`: Crawler (Worker) → Analytics Service
  - `analysis.finished`: Analytics Service → Insight Service
  - `job.completed`: Analytics Service → Notification

**State Management (Hybrid State):**

- Redis DB 1: Project execution state
- Key schema: `smap:proj:{projectID}`
- Task-level fields: `tasks_total`, `tasks_done`, `tasks_errors` (for completion check)
- Item-level fields: `items_expected`, `items_actual`, `items_errors` (for progress display)
- Analyze fields: `analyze_total`, `analyze_done`, `analyze_errors`
- Legacy fields: `crawl_total`, `crawl_done`, `crawl_errors` (backward compatibility)
- Status flow: `INITIALIZING` → `PROCESSING` → `DONE`/`FAILED`

**Crawl Limits Configuration:**

- All limits are configurable via environment variables
- Default limits: `DEFAULT_LIMIT_PER_KEYWORD=50`, `DEFAULT_MAX_COMMENTS=100`
- Dry-run limits: `DRYRUN_LIMIT_PER_KEYWORD=3`, `DRYRUN_MAX_COMMENTS=5`
- Hard limits (safety caps): `MAX_LIMIT_PER_KEYWORD=500`, `MAX_MAX_COMMENTS=1000`
- Feature flags: `INCLUDE_COMMENTS=true`, `DOWNLOAD_MEDIA=false`

**Task Types:**

- `research_keyword`: Search content by keywords
- `crawl_links`: Scrape specific URLs
- `research_and_crawl`: Composite task (search + scrape)
- `dryrun_keyword`: Test keywords without actual crawling

**Supported Platforms:**

- YouTube (`PlatformYouTube`)
- TikTok (`PlatformTikTok`)

## Important Constraints

- **Redis Database Separation**:
  - DB 0: Reserved for job mapping và Pub/Sub (không dùng trong Collector)
  - DB 1: Project state tracking (status, progress)
- **Event Schema Compliance**:
  - Phải tuân thủ event schema định nghĩa trong `document/event-drivent.md`
  - `ProjectCreatedEvent` phải có `user_id` trong payload
- **Backward Compatibility**:
  - Legacy `collector.inbound` exchange vẫn được support (deprecated)
  - Support cả `CrawlRequest` (legacy) và `ProjectCreatedEvent` (new)
- **Webhook Throttling**:
  - Removed (workers report once per platform completion)
  - Chỉ 2-3 webhook calls per project maximum
- **State TTL**:
  - Default 7 days (168 hours)
  - Configurable via `REDIS_STATE_TTL_HOURS`
- **Fail-Fast Principle**:
  - RabbitMQ connection: Fail fast nếu không connect được
  - Redis connection: Fail fast nếu không connect được
  - Webhook client: Fail fast nếu không initialize được

## External Dependencies

**Message Queue:**

- **RabbitMQ**:
  - Exchange: `smap.events` (topic) - Event-driven architecture
  - Exchange: `collector.inbound` (topic) - Legacy support
  - Exchange: `collector.youtube`, `collector.tiktok` - Outbound to workers
  - Connection: `RABBITMQ_URL` env variable

**State Storage:**

- **Redis**:
  - Host: `REDIS_STATE_HOST` (default: `localhost:6379`)
  - DB: `REDIS_STATE_DB` (default: `1`)
  - Pool size: `REDIS_STATE_POOL_SIZE` (default: `10`)
  - TTL: `REDIS_STATE_TTL_HOURS` (default: `168` = 7 days)

**External Services:**

- **Project Service**:
  - Base URL: `PROJECT_SERVICE_URL` (default: `http://localhost:8080`)
  - Internal Key: `PROJECT_INTERNAL_KEY` (required)
  - Webhook endpoint: `POST /internal/progress/callback`
  - Timeout: `PROJECT_TIMEOUT` seconds (default: `10`)
  - Retry: `WEBHOOK_RETRY_ATTEMPTS` (default: `5`), `WEBHOOK_RETRY_DELAY` seconds (default: `1`)

**Monitoring & Notifications:**

- **Discord Webhook**:
  - Report bug ID: `DISCORD_REPORT_BUG_ID`
  - Report bug token: `DISCORD_REPORT_BUG_TOKEN`
  - Optional: Warn nếu không initialize được

**Future Dependencies:**

- **MongoDB**: For persistence (not yet implemented)
- **MinIO**: Data storage (referenced in `data.collected` events, managed by workers)
