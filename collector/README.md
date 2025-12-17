# SMAP API - Dispatcher Service

This repository houses the **Dispatcher Service** for the SMAP data collection system. It serves as the central coordinator that receives high-level crawl requests, validates them, and distributes granular tasks to platform-specific workers via RabbitMQ.

## 🏗 System Architecture

The service follows a **Producer-Consumer** and **Fan-out** architecture pattern. It acts as an intermediary between the request initiator (e.g., a frontend or scheduler) and the actual scraping workers.

```mermaid
graph LR
    A[BE Service] -- "CrawlRequest" --> B(RabbitMQ: Dispatch Queue)
    B --> C{SMAP Dispatcher}
    C -- "Validate & Map" --> C
    C -- "CollectorTask (YouTube)" --> D[RabbitMQ: YouTube Queue]
    C -- "CollectorTask (TikTok)" --> E[RabbitMQ: TikTok Queue]
    D --> F[YouTube Worker]
    E --> G[TikTok Worker]
```

### Core Component

**Dispatcher Consumer (`cmd/consumer`)**:

- The core worker process that consumes `CrawlRequest` messages from RabbitMQ.
- Validates incoming requests and executes the **Dispatch UseCase** to route tasks.
- Distributes platform-specific tasks to YouTube and TikTok workers.
- Built with **Go** and **RabbitMQ client**.

## Business Logic & Rules

The core logic resides in `internal/dispatcher`.

### 1. Dispatch Flow

The dispatch process (`internal/dispatcher/usecase/dispatch_uc.go`) follows these steps:

1.  **Receive Request**: Accepts a `CrawlRequest` containing `JobID`, `TaskType`, and a generic `Payload`.
2.  **Platform Selection**: Determines which platforms to target based on configuration (default: all enabled platforms).
3.  **Task Generation**: For each target platform:
    - Creates a `CollectorTask` with a new trace ID and metadata.
    - **Payload Mapping**: Converts the generic input payload into a strict, platform-specific struct (e.g., `YouTubeResearchKeywordPayload`) using the `Mapper` logic.
    - **Routing**: Determines the correct RabbitMQ routing key for the platform.
4.  **Publish**: Sends the `CollectorTask` to the platform's queue.

### 2. Supported Task Types

The system supports three primary task types (`internal/models/task.go`):

- `research_keyword`: Search for content based on keywords.
- `crawl_links`: Scrape specific video/profile URLs.
- `research_and_crawl`: A composite task to search and then immediately scrape results.

### 3. Supported Platforms

- **YouTube** (`PlatformYouTube`)
- **TikTok** (`PlatformTikTok`)

## 📐 Design Patterns

The project strictly follows **Clean Architecture** and **SOLID** principles:

- **Clean Architecture**:

  - `delivery/`: Transport layer (RabbitMQ consumers, HTTP handlers).
  - `usecase/`: Pure business logic (Dispatching, Mapping).
  - `models/`: Domain entities and DTOs.
  - **Benefit**: The business logic is decoupled from the transport mechanism. We could easily switch from RabbitMQ to Kafka without changing the core dispatch logic.

- **Dependency Injection (DI)**:

  - All components (UseCases, Repositories, Consumers) are initialized with their dependencies passed via constructors (`New...`).
  - **Benefit**: Makes testing easier (mocking dependencies) and dependencies explicit.

- **Strategy/Factory Pattern**:
  - The `mapPayload` function (`internal/dispatcher/usecase/mapper.go`) acts as a factory, selecting the correct payload structure and validation strategy based on the `Platform` and `TaskType`.

## 📂 Project Structure

```
smap-api/
├── cmd/
│   └── consumer/     # RabbitMQ Consumer entry point
├── config/           # Configuration loading (Env)
├── internal/
│   ├── dispatcher/   # CORE DOMAIN: Dispatch logic
│   │   ├── delivery/ # RabbitMQ consumers/producers
│   │   └── usecase/  # Business logic (Dispatch, Map)
│   ├── consumer/     # Consumer server implementation
│   └── models/       # Shared data structures (CrawlRequest, CollectorTask)
├── pkg/              # Shared utilities (Logger, RabbitMQ, Mongo, etc.)
└── ...
```

## 🚀 Getting Started

### Prerequisites

- Go 1.23+
- RabbitMQ
- MongoDB (for future persistence)
- Redis (optional, for caching)

### Configuration

Copy `env.template` to `.env` and configure:

```ini
# RabbitMQ
AMQP_URL=amqp://guest:guest@localhost:5672/

# Service
PORT=8080
MODE=debug
```

### Crawl Limits Configuration

The service supports configurable crawl limits to control resource usage:

| Environment Variable        | Default | Description                              |
| --------------------------- | ------- | ---------------------------------------- |
| `DEFAULT_LIMIT_PER_KEYWORD` | 50      | Default items per keyword for production |
| `DEFAULT_MAX_COMMENTS`      | 100     | Default max comments per item            |
| `DEFAULT_MAX_ATTEMPTS`      | 3       | Max retry attempts for failed tasks      |
| `DRYRUN_LIMIT_PER_KEYWORD`  | 3       | Items per keyword for dry-run/testing    |
| `DRYRUN_MAX_COMMENTS`       | 5       | Max comments per item for dry-run        |
| `MAX_LIMIT_PER_KEYWORD`     | 500     | Hard limit (safety cap) for items        |
| `MAX_MAX_COMMENTS`          | 1000    | Hard limit (safety cap) for comments     |
| `INCLUDE_COMMENTS`          | true    | Whether to include comments in crawl     |
| `DOWNLOAD_MEDIA`            | false   | Whether to download media files          |

**Note:** Hard limits (`MAX_*`) are safety caps that cannot be exceeded even if default values are set higher.

### Running the Service

**Start the Dispatcher Consumer:**
This process listens for incoming requests from RabbitMQ and dispatches them to platform workers.

```bash
go run cmd/consumer/main.go
```

**Or use the Makefile:**

```bash
make run-consumer
```

## 🔌 Integration Guide

### Event-Driven Architecture

The service now supports event-driven choreography with the Project Service. See `document/event-drivent.md` for full details.

#### SMAP Events Exchange

- **Exchange**: `smap.events` (Type: `topic`)
- **Consumed Routing Key**: `project.created`

> **Note:** `data.collected` event is published by Crawler (Worker) services, not Collector.

#### Project Created Event (Consumed)

```json
{
  "event_id": "evt_abc123",
  "timestamp": "2025-12-05T10:00:00Z",
  "payload": {
    "project_id": "proj_xyz",
    "user_id": "user_123",
    "brand_name": "VinFast",
    "brand_keywords": ["VinFast", "VF3"],
    "competitor_names": ["Toyota"],
    "competitor_keywords_map": {
      "Toyota": ["Toyota", "Vios"]
    },
    "date_range": {
      "from": "2025-01-01",
      "to": "2025-02-01"
    }
  }
}
```

#### Data Collected Event (Published)

```json
{
  "event_id": "evt_def456",
  "timestamp": "2025-12-05T11:00:00Z",
  "payload": {
    "project_id": "proj_xyz",
    "user_id": "user_123",
    "minio_path": "/data/proj_xyz/output.json",
    "item_count": 150,
    "platform": "youtube"
  }
}
```

#### Redis State Management (Hybrid State)

The service updates project state in Redis (DB 1) with key schema `smap:proj:{projectID}`.

**Hybrid State Pipeline:**

```
Phase 1: CRAWL                              Phase 2: ANALYZE
┌─────────────────────────────────┐         ┌─────────────────────────┐
│ Task-Level (completion check):  │         │ analyze_total: 450      │
│   tasks_total: 10               │         │ analyze_done: 200       │
│   tasks_done: 10                │  ────►  │ analyze_errors: 5       │
│   tasks_errors: 0               │         └─────────────────────────┘
│                                 │
│ Item-Level (progress display):  │
│   items_expected: 500           │
│   items_actual: 450             │
│   items_errors: 50              │
└─────────────────────────────────┘
Crawler → Collector                         Analytics → Collector
```

| Field            | Type   | Description                                |
| ---------------- | ------ | ------------------------------------------ |
| `status`         | String | INITIALIZING, PROCESSING, DONE, FAILED     |
| `tasks_total`    | Int    | Total crawl tasks (keywords × platforms)   |
| `tasks_done`     | Int    | Crawl tasks completed                      |
| `tasks_errors`   | Int    | Crawl tasks failed                         |
| `items_expected` | Int    | Expected items (tasks × limit_per_keyword) |
| `items_actual`   | Int    | Actual items crawled successfully          |
| `items_errors`   | Int    | Items failed to crawl                      |
| `analyze_total`  | Int    | Total analyze tasks (auto-set on crawl)    |
| `analyze_done`   | Int    | Analyze tasks completed                    |
| `analyze_errors` | Int    | Analyze tasks failed                       |

**Completion Logic:**

- Crawl complete: `tasks_done + tasks_errors >= tasks_total` (task-level)
- Analyze complete: `analyze_done + analyze_errors >= analyze_total`
- Project complete: Both phases complete

**Progress Display:**

- Uses item-level for accurate progress: `(items_actual + items_errors) / items_expected`
- Falls back to task-level if items not tracked

#### Progress Webhook (Hybrid Format)

The service calls Project Service webhook to notify progress:

```
POST /internal/progress/callback
Header: X-Internal-Key: {internal_key}
```

```json
{
  "project_id": "proj_xyz",
  "user_id": "user_123",
  "status": "PROCESSING",
  "tasks": {
    "total": 10,
    "done": 8,
    "errors": 0,
    "percent": 80.0
  },
  "items": {
    "expected": 500,
    "actual": 400,
    "errors": 20,
    "percent": 84.0
  },
  "crawl": {
    "total": 10,
    "done": 8,
    "errors": 0,
    "progress_percent": 84.0
  },
  "analyze": {
    "total": 400,
    "done": 200,
    "errors": 5,
    "progress_percent": 51.25
  },
  "overall_progress_percent": 67.625
}
```

#### Analyze Result Handler

The service consumes analyze results from Analytics Service:

```json
{
  "project_id": "proj_xyz",
  "job_id": "proj_xyz-analyze-0",
  "task_type": "analyze_result",
  "batch_size": 50,
  "success_count": 48,
  "error_count": 2
}
```

### RabbitMQ Connection (Legacy Inbound)

External services (e.g., API Gateway, Scheduler) can still publish `CrawlRequest` messages to the **Inbound** exchange.

- **Exchange**: `collector.inbound` (Type: `topic`)
- **Routing Key**: `crawler.#` (e.g., `crawler.request`)
- **Queue**: `collector.inbound.queue`

> [!NOTE]
> The `collector.tiktok` and `collector.youtube` exchanges are **internal** and managed by the dispatcher. Do not publish to them directly.

#### Payload Example (`CrawlRequest`)

```json
{
  "job_id": "job_12345",
  "task_type": "research_keyword",
  "payload": {
    "keyword": "golang tutorial",
    "limit": 10
  },
  "time_range": 7,
  "attempt": 1,
  "max_attempts": 3,
  "emitted_at": "2023-10-27T10:00:00Z"
}
```
