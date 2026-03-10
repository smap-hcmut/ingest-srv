# System Module Convention

This document outlines the standard structure and conventions for modules based on the `internal/event` reference implementation and strict architectural rules.

## Module Structure

Each module (e.g., `internal/event`, `internal/user`) should follow this directory structure:

```
internal/<module_name>/
├── delivery/           # Delivery layer (HTTP, gRPC, Workers)
│   ├── http/           # HTTP handlers
│   │   ├── handlers.go         # API handlers
│   │   ├── process_request.go  # Input decoding & validation
│   │   ├── presenters.go       # Response formatting
│   │   └── routes.go           # Route detailed definitions
│   ├── job/            # Cron Job / Scheduler handlers
│   │   ├── handlers.go         # Job handlers
│   │   └── register.go         # Job registration
│   └── rabbitmq/       # Message queue consumers/producers
│       ├── consumer/           # RabbitMQ Consumers
│       │   ├── new.go          # Factory: NewConsumer
│       │   ├── consumer.go     # Manager: Starts/Configures consumers
│       │   ├── workers.go      # Logic: Actual message processing functions
│       │   └── <topic>.go      # (Optional) Separated workers if complex
│       ├── producer/           # RabbitMQ Producers
│       │   ├── new.go          # Factory: NewProducer
│       │   └── producer.go     # Publish methods
│       └── presenters.go       # MQ Message DTOs & Constants
│   └── kafka/          # (Proposed) Kafka Consumers/Producers
│       ├── consumer/
│       │   ├── new.go          # Factory: NewConsumerGroup
│       │   ├── handler.go      # Sarama ConsumerGroupHandler impl
│       │   ├── router.go       # Topic/Message routing to workers
│       │   └── workers.go      # Business logic: ProcessMessage
│       ├── producer/
│       │   ├── new.go          # Factory: NewProducer
│       │   └── producer.go     # Publish methods
│       ├── constants.go        # Topic names & Consumer Groups
│       └── messages.go         # Message DTO structs
├── repository/         # Data access layer
│   ├── mongo/          # MongoDB implementation
│   │   ├── <entity>.go       # Main CRUD implementation
│   │   ├── <entity>_build.go # Helper: Input -> Model, Model -> BSON
│   │   └── <entity>_query.go # Helper: Filter -> BSON M
│   ├── postgre/        # PostgreSQL implementation
│   ├── interface.go      # Repository interface definition
│   └── options.go      # Filter/Option structs for Repository methods
├── usecase/            # Business logic (implementation)
│   ├── <entity>.go     # Main logic (Create, Update, Delete)
│   ├── <concern>.go    # Logic separated by concern (e.g., session.go, ratelimit.go)
│   ├── job.go          # Logic for background jobs
│   ├── producer.go     # Logic for publishing events
│   ├── consumer.go     # Logic for handling consumed events
│   ├── helpers.go      # Private helper methods
│   └── types.go        # Private struct definitions (internal to usecase package)
├── interface.go        # UseCase interface definition
├── types.go            # ALL Input/Output structs for UseCase
└── errors.go           # Module-specific errors
```

## Detailed File Responsibilities

> [!NOTE]
> **Multiple Entities**: A module can (and often does) manage multiple related entities. For example, `internal/event` manages `Event`, `EventInstance`, and `RecurringTracking`.
>
> - **Repository**: You should have separate implementations (e.g., `repository/mongo/event.go`, `repository/mongo/recurring_tracking.go`).
> - **UseCase**: You should have separate logic files (e.g., `usecase/event.go`, `usecase/recurring_tracking.go`).
> - **Types**: Keep related types together or separate if too large, but ALWAYS in `types.go` (module-level) or `usecase/types.go` (if private to usecase package).

### Interface Conventions

**1. Repository Interface**: Use **Interface Composition** to group methods by entity.

```go
type Repository interface {
    EventRepository
    EventInstanceRepository
}

type EventRepository interface {
    Create(...)
    // ...
}
```

**2. UseCase Interface**: Use a **Single Flat Interface** but group methods with comments.

```go
type UseCase interface {
    // Event Logic
    Create(...)

    // Job Logic
    CheckNotifyEvent()
}
```

### 1. Delivery Layer

This layer handles all transport concerns (HTTP, Job, RabbitMQ). It **MUST NOT** contain business logic.

#### 1.1 HTTP (`delivery/http`)

- **`handlers.go`**: The controller.
  - **Responsibility**: Coordinates the flow: Request -> Validation -> UseCase -> Response.
  - **Swagger**: Every handler method **MUST** have a full Swagger annotation block (Summary, Description, Params, Responses).
  - **Naming**: Simple method names (e.g., `Create`, `Detail`), as they are methods of a specific Handler struct.
  - **Error Handling**: Catch errors from UseCase and map them using `response.Error` or `response.ErrorWithMap`.

- **`process_request.go`**: Input Processing.
  - **Responsibility**:
    1.  Binds HTTP request (JSON, Query, URI) to local DTO structs.
    2.  Validates the DTO (using `binding` tags + `validate()` method).
    3.  Extracts User Context (JWT) into `models.Scope`.
    4.  Converts DTO to UseCase Input (`uc_types.go`) via a `toInput()` method.
  - **Pattern**: `func (h handler) process[Action]Request(c *gin.Context) ([Action]Req, models.Scope, error)`

- **`presenters.go`**: DTO Definitions & Mapping.
  - **Responsibility**: Defines all Request/Response private structs (e.g., `createReq`, `detailResp`).
  - **Rule**: Use **Pointers** (`*Struct`) for optional nested structs to handle `nil`/missing fields correctly.
  - **Request Methods**:
    - `func (r [Action]Req) validate() error`: Custom validation logic.
    - `func (r [Action]Req) toInput() [Module].[Action]Input`: Mapper from Delivery DTO to UseCase Input.
  - **Response Methods**:
    - `func (h handler) new[Action]Resp(output [Module].[Action]Output) [Action]Resp`: Mapper from UseCase Output to Delivery Response DTO.

- **`new.go`**: Factory.
  - **Responsibility**: Defines `Handler` interface and `New` factory function.
  - **Pattern**: `func New(l log.Logger, uc [Module].UseCase) Handler`

- **`routes.go`**: Routing.
  - **Responsibility**: Maps HTTP verbs and paths to Handler methods.
  - **Grouping**: Use `gin.RouterGroup` to group routes by resource.
  - **Middleware**: Inject `middleware.Middleware` and apply via `r.Use(mw.Auth())`.

- **`errors.go`**: Error Mapping (Crucial).
  - **Responsibility**: Maps UseCase errors to HTTP errors (using `pkg/errors`).
  - **Pattern**: `func (h handler) mapError(err error) error`.
  - **Rule**: **MUST** panic on unknown errors (`default: panic(err)`). This guarantees all domain errors are explicitly handled during development.
  - **Constants**: Define local `var` constants like `errEventNotFound = pkgErrors.NewHTTPError(141004, "Event not found")`.

#### 1.2 Job (`delivery/job`)

- **`handlers.go`**: Job Controllers.
  - **Responsibility**: Entry point for cron jobs.
  - **Context**: Must create a fresh background context (`context.Background()`), as jobs are independent of HTTP requests.
  - **Logging**: **MANDATORY** to log `Start` and `End` of the job schedule to trace execution.
  - **Error Handling**: Log errors but do **not** panic or crash the scheduler.
  - **Pattern**:
    ```go
    func (h handler) CheckNotifyEvent() {
        ctx := context.Background()
        h.l.Infof(ctx, "Start...")
        if err := h.uc.DoWork(ctx); err != nil {
            h.l.Errorf(ctx, "Error: %v", err)
        }
        h.l.Infof(ctx, "End...")
    }
    ```
- **`register.go`**: Job Registration.
  - **Responsibility**: Maps Handlers to Cron Expressions.
  - **Return**: `[]cron.JobInfo`.

#### 1.3 RabbitMQ (`delivery/rabbitmq`)

- **`consumer/new.go`**: Consumer Factory.
- **`consumer/workers.go`**: Message Processing Logic.
  - **Context**: Create `context.Background()` for each message.
  - **Responsibility**:
    1.  Unmarshal JSON body to DTO (`presenters.go`).
    2.  Handle Unmarshal errors: Log Warn + `Ack(false)` (poison message, discard).
    3.  Call UseCase.
    4.  Handle Business errors: Log Error + `Ack(false)` (retry/requeue depending on logic, usually defined by infrastructure).
  - **Scope**: Construct `models.Scope` manually if needed (e.g., from message ShopID).
- **`presenters.go`**: Message DTOs.
  - **Rule**: decoupled structs (don't reuse UseCase/Entity types).

#### 1.4 Kafka (`delivery/kafka`)

**Standard implementation using `sarama` library.**

- **Structure**:

  ```
  kafka/
  ├── consumer/
  │   ├── group.go       // Factory: NewConsumerGroup
  │   ├── handler.go     // sarama.ConsumerGroupHandler implementation
  │   └── worker.go      // Business logic: ProcessMessage
  ├── producer/
  │   ├── producer.go    // AsyncProducer implementation
  ├── models.go          // Message DTOs
  └── constants.go       // Topics, Consumer Group IDs
  ```

- **`consumer/group.go`**:
  - **Responsibility**: init `sarama.NewConsumerGroup`, handles the `Consume` loop in a goroutine.

- **`consumer/handler.go`**:
  - **Responsibility**: Implements `Setup`, `Cleanup`, `ConsumeClaim`.
  - **Logic**: Loops over `claim.Messages()`, calls `worker` for each message, marks message (`session.MarkMessage`) if successful.

- **`consumer/worker.go`**:
  - **Pattern**: `func (h *GroupHandler) process(ctx context.Context, msg *sarama.ConsumerMessage) error`
  - **Responsibility**:
    1.  Unmarshal `msg.Value` to DTO (`models.go`).
    2.  Call UseCase.
    3.  Return error to Handler (decide to retry or skip).

- **Clean Architecture Note**:
  - **NO** business logic in Delivery (e.g., batching, complicated buffering should be in UseCase or Repository if it's data-layer specific optimization).
  - **NO** direct Repository access. **MUST** go through UseCase.

### 2. UseCase Layer (`usecase/`, `interface.go`, `types.go`)

This layer contains the core business logic.

- **`interface.go`** (module root):
  - Defines the `UseCase` interface.
  - **Rule**: Methods must accept `context.Context` and `models.Scope` as the first two arguments.
  - **Pattern**: `Create(ctx context.Context, sc models.Scope, input CreateInput) (CreateOutput, error)`

- **`types.go`** (module root):
  - **Responsibility**: Defines Input/Output structs for UseCase methods.
  - **Rule**: **DECOUPLE** from Delivery Layer. Do not use HTTP-specific tags (`form`, `json` is okay for general serialization but avoid binding tags if possible).
  - **Naming**: `<Action>Input` and `<Action>Output` (e.g., `CreateEventInput`, `DetailOutput`).
  - **Converters**: Helper functions to convert Domain Models to UseCase DTOs (e.g., `EventToEventInstance`) should live here or in strict utility files.

- **`usecase/` (Implementation)**:
  - **`new.go`**:
    - Defines `implUseCase` struct (private) and `New` factory.
    - Injects dependencies: `Repository`, `Logger`, and other `UseCases`.
  - **`<entity>.go`** (e.g., `event.go`):
    - Implements business logic.
    - **Concurrency**: Use `golang.org/x/sync/errgroup` for parallel operations (e.g., fetching multiple related entities, sending notifications).
    - **Error Handling**:
      - **Log**: `uc.l.Errorf(ctx, "method: %v", err)` before returning.
      - **Return**: Return definition errors (from `uc_errors.go`) or wrap/return repository errors.
    - **Validation**: Perform **Business Validation** here (e.g., "User belongs to Branch", "Inventory check"). Simple validation (required fields) can happen in Delivery, but UseCase is the final gatekeeper.
  - **`utils.go`**:
    - Shared private helper methods for the UseCase implementation (e.g., `validateAssign`, `calculateNotifyTime`).

- **Transactions**:
  - Currently, explicit transaction propagation (`uow`) is not strictly enforced in the codebase scan, but critical multi-step writes should ideally be transactional. For now, multiple writes are often handled sequentially or via `errgroup` without rollback.

### 3. Repository Layer (`repository/`)

This layer handles data access. It **MUST** separate Execution, Query Building, and Data Mapping.

#### 3.1 Common Structure

Both MongoDB and PostgreSQL implementations **MUST** follow this file structure within `repository/<driver>/`:

- **`new.go`**: Factory function.
- **`<entity>.go`**: Execution logic (CRUD operations). Calls methods in `query.go` and `build.go`.
- **`query.go`**: Query Builders. Pure logic that returns query filters/modifiers.
  - Mongo: Returns `bson.M`.
  - Postgres: Returns `[]qm.QueryMod`.
- **`build.go`**: Data Mappers. Converts between Domain Models (Business layer) and DB Models (Driver specific).

#### 3.2 MongoDB (`repository/mongo/`)

- **Execution (`<entity>.go`)**:
  ```go
  func (r *implRepo) Detail(ctx context.Context, sc models.Scope, id string) (models.Event, error) {
      // 1. Build Query
      filter, err := r.buildDetailQuery(ctx, sc, id)
      if err != nil { return models.Event{}, err }
      // 2. Execute
      var doc models.Event
      err = r.col.FindOne(ctx, filter).Decode(&doc)
      // 3. Return
      return doc, err
  }
  ```
- **Query (`query.go`)**:
  - **Pattern**: `func (r *implRepo) build<Action>Query(...) (bson.M, error)`
  - **Responsibility**: Constructs `bson.M` filters, handling `models.Scope`.

#### 3.3 PostgreSQL (`repository/postgre/`) (Using `sqlboiler`)

- **Execution (`<entity>.go`)**:
  - **Context**: Must always pass `ctx` to `sqlboiler` methods.
  - **Pattern**:
    ```go
    func (r *implRepo) List(ctx context.Context, sc models.Scope, input repo.GlobalListInput) ([]models.User, error) {
        // 1. Build Query Modifiers
        mods := r.buildListQuery(sc, input)
        // 2. Execute (using sqlboiler generated code)
        rows, err := sqlboiler.Users(mods...).All(ctx, r.db)
        if err != nil { return nil, err }
        // 3. Convert (DB Model -> Domain Model)
        return r.toDomainList(rows), nil
    }
    ```
- **Query (`query.go`)**:
  - **Pattern**: `func (r *implRepo) build<Action>Query(sc models.Scope, ...) []qm.QueryMod`
  - **Responsibility**: Returns a slice of `qm.QueryMod`.
- **Build (`build.go`)**:
  - **Responsibility**:
    - `toDomain(db *sqlboiler.User) models.User`: Convert SQLBoiler struct to Domain struct.
    - `toDB(m models.User) *sqlboiler.User`: Convert Domain struct to SQLBoiler struct.
    - **Nullable Handling**: Handle `null.String`/`null.Time` conversions here.

#### 3.4 Raw SQL (Special Case)

- Only use Raw SQL (e.g., `db.ExecContext`) for bulk operations or complex queries.
- Logic must still be isolated in `query.go` (returning query string + args).

## File Responsibility Principle (MANDATORY)

> [!IMPORTANT]
> **Every file has ONE job.** Struct definitions and logic **MUST NOT** live in the same file. If a file exceeds 200 lines, review whether it should be split by concern.

### Rules

1. **Struct definitions go in dedicated type files.** Never define types (structs, interfaces) inline in logic files.

   | Layer                        | Type file                                       | Logic files                                   |
   | ---------------------------- | ----------------------------------------------- | --------------------------------------------- |
   | **UseCase** (module root)    | `types.go` — Input/Output structs               | —                                             |
   | **UseCase** (implementation) | `usecase/types.go` — private structs            | `usecase/<entity>.go`, `usecase/<concern>.go` |
   | **Delivery HTTP**            | `presenters.go` — all DTOs                      | `handlers.go`, `<resource>.go`                |
   | **Repository**               | `repository/options.go` — filter/option structs | `repository/<driver>/<entity>.go`             |

2. **Logic files contain ONLY method implementations.** No `type`, no `var` blocks (except very local sentinels).

3. **`new.go` is strictly a factory.** It contains:
   - The `impl` struct (private)
   - The `New()` constructor
   - Setter methods (if needed for optional dependencies)
   - **Nothing else.** No interfaces, no helper types, no constants.

4. **Split by concern, not by size.** When a module is complex, split logic files by business concern rather than arbitrarily. Each file name should clearly communicate its responsibility.

   ```text
   # ✅ Good: split by concern
   usecase/
   ├── event.go          # CRUD operations for Event
   ├── recurring.go      # Recurrence logic
   ├── notification.go   # Notification orchestration
   ├── helpers.go        # Shared private helpers
   └── new.go            # Factory only

   # ❌ Bad: everything in one file
   usecase/
   ├── event.go          # 800 lines, CRUD + recurrence + notification + helpers
   └── new.go
   ```

5. **Multi-entity repositories** — use `<entity>_query.go` and `<entity>_build.go`:

   ```text
   repository/postgre/
   ├── user.go             # User coordinator
   ├── user_query.go       # User query builder
   ├── user_build.go       # User data mapper
   ├── event.go            # Event coordinator
   ├── event_query.go      # Event query builder
   ├── event_build.go      # Event data mapper
   └── new.go              # Factory
   ```

---

## Naming Conventions & Rules

### Global Rules

1.  **Contextual Naming**: In `internal/user`, functions should be named `Create`, `Update`, `Detail` (NOT `CreateUser`, `UpdateUser`). The package name provides the context.
2.  **Type Centralization**:
    - ALL UseCase Input/Output types must be in `types.go` at the module root.
    - Logic files (e.g., `event.go`) must contain **logic only**, no struct definitions.
    - Private helper structs go in `usecase/types.go`.

### Module Interaction Rules

1.  **UseCase to UseCase Only**: Modules can only interact via their UseCase interfaces.
    - ✅ `event.UseCase` calls `user.UseCase`.
    - ❌ `event.UseCase` calls `user.Repository`.
    - ❌ `event.Repository` calls `user.Repository`.
2.  **No Repo exports**: Repositories should technically be internal or only exposed via interface for dependency injection in `main` or specific factory. Never import a repository directly into another module's logic.

## Folder Conventions

- **PostgreSQL**: If using Postgres, use `repository/postgre`.
- **MongoDB**: Use `repository/mongo`.
- **File Splitting**: Repositories MUST rely on `build.go` and `query.go` to keep the main implementation file clean.

## Code Examples

### Repository (`repository/mongo/query.go`)

```go
func (repo implRepository) buildFilter(ctx context.Context, sc models.Scope, f repository.Filter) (bson.M, error) {
    // Logic to build BSON filter
    filter := bson.M{}
    if f.ID != "" {
        filter["_id"] = f.ID
    }
    return filter, nil
}
```

### UseCase (`usecase/event.go`)

```go
// Only logic here, no struct definitions
func (uc implUseCase) Create(ctx context.Context, sc models.Scope, input event.CreateInput) (event.CreateEventOutput, error) {
    // Logic
}
```

### UseCase Types (`types.go`)

```go
type CreateInput struct {
    Title string
    // ...
}

type CreateEventOutput struct {
    Event models.Event
}
```

## Shared Packages (`pkg/`) Convention

The `pkg/` directory contains shared, domain-agnostic utilities and libraries. These are designed to be reusable across the entire system and potentially external projects.

### Core Principles

1.  **No Internal Dependencies**: Code in `pkg/` must **NEVER** import packages from `internal/`.
2.  **Stateless & Reusable**: Designed as libraries/helpers.
3.  **Domain Agnostic**: Logic here should not contain specific business rules of the application (e.g., "User" logic goes in `internal/user`, but "JWT parsing" goes in `pkg/jwt`).

### Common Packages & Responsibilities

| Package                                       | Responsibility                                                                                                     |
| :-------------------------------------------- | :----------------------------------------------------------------------------------------------------------------- |
| **`pkg/auth`**                                | Authentication utilities and middleware.                                                                           |
| **`pkg/compressor`**                          | Compression utilities (e.g., streaming compression) for handling large payloads.                                   |
| **`pkg/discord`**                             | Discord webhook integrations for system alerts/notifications.                                                      |
| **`pkg/email`**                               | Email sending utilities and template management.                                                                   |
| **`pkg/encrypter`**                           | Encryption and hashing utilities (e.g., password hashing, data encryption).                                        |
| **`pkg/errors`**                              | Defines standard system errors (`HTTPError`, `ValidationError`) mapped to HTTP status codes.                       |
| **`pkg/google`**                              | Google API client wrappers.                                                                                        |
| **`pkg/i18n` / `pkg/locale`**                 | Internationalization and localization helpers (translations, locale parsing).                                      |
| **`pkg/jwt`**                                 | JWT generation, parsing, and validation.                                                                           |
| **`pkg/kafka`**                               | Kafka message queue connection and producer/consumer wrappers.                                                     |
| **`pkg/log`**                                 | Structured logging wrapper (e.g., Zap). Provides consistent log formats and levels.                                |
| **`pkg/oauth`**                               | OAuth2 providers integration (Google, Azure, Okta, etc.).                                                          |
| **`pkg/paginator`**                           | Pagination logic and query parsing helpers.                                                                        |
| **`pkg/postgre` / `pkg/mongo` / `pkg/redis`** | Database/Cache connection initialization and low-level driver configuration. **NOT** for repo implementation.      |
| **`pkg/rabbitmq`**                            | RabbitMQ connection managers and low-level publishers/consumers.                                                   |
| **`pkg/response`**                            | Standardizes API JSON responses (`Success`, `Error`, `Paging`). Enforces consistent response structure.            |
| **`pkg/scope`**                               | Handles request context extraction. Standardizes how `UserID`, `ShopID`, and `Role` are passed through the system. |
| **`pkg/util`**                                | General purpose helpers (Slice manipulation, String formatting, Time conversion).                                  |

### Usage Rule

- **Importing**: Any module in `internal/` or `cmd/` can import any package from `pkg/`.
- **Modifying**: Changes in `pkg/` should be backward compatible as they may affect multiple services/modules.

> [!IMPORTANT]
> **MANDATORY**: Before implementing any generic utility or helper (e.g., error handling, HTTP response, JWT parsing, etc.), you **MUST** check the `pkg/` directory first.
> **Re-inventing the wheel is strictly prohibited.** If a utility exists in `pkg/`, you must use it. If it lacks functionality (e.g., `pkg/kafka` missing Async producer), **improve the existing `pkg`** instead of implementing a raw driver in `internal/`.
>
> **Infrastructure Rule**: `internal/` modules must **NEVER** import low-level drivers (e.g., `sarama`, `redis/v8`, `amqp`) directly in business logic using strict types. Always use the wrappers in `pkg/` or define an `interface` in `internal/` and implement the adapter in `delivery/`.

## Application Wiring (`cmd/api/main.go`)

The `main.go` file is the entry point and the **only** place where dependency injection should occur.

### Initialization Order

1.  **Configuration**: Load `config.yaml` and env vars.
2.  **Infrastructure**: Init Logger (`pkg/log`), Database (`pkg/postgre`, `pkg/mongo`), Redis, Kafka.
3.  **Core Utilities**: Init customized managers (e.g., `pkg/encrypter`, `pkg/jwt`, `pkg/discord`).
4.  **UseCases**: Init all UseCases, injecting Repositories and other UseCases.
    - _Rule_: Repositories are injected into UseCases here.
5.  **Delivery**: Init HTTP Server (`internal/httpserver`), injecting UseCases, Middlewares, and Core Utils.
6.  **Run**: Start server and listen for graceful shutdown signals.

### Rules

- **No Globals**: Avoid global variables for DB or Logger. Pass them as dependencies.
- **Graceful Shutdown**: Always implement signal handling to close DB connections and stop server gracefully.

## API Documentation (Swagger)

Swagger documentation is **MANDATORY** for all HTTP handlers.

### Location

- Place annotations directly above the handler method in `handlers.go`.

### Required Annotations

- `@Summary`: Short description.
- `@Description`: Detailed explanation.
- `@Tags`: Grouping (e.g., `Events`, `Users`).
- `@Accept`/`@Produce`: usually `json`.
- `@Param`: Body, Query, or Path parameters.
- `@Success`: Success response using the `presenters.go` struct.
- `@Failure`: Error response structure.
- `@Router`: Path and Verb.

### Example

```go
// @Summary Create event
// @Description Create a new event with optional recurrence
// @Tags Events
// @Accept json
// @Produce json
// @Param body body createReq true "Event Data"
// @Success 200 {object} detailResp
// @Failure 400 {object} response.Resp "Bad Request"
// @Router /api/v1/events [POST]
func (h handler) Create(c *gin.Context) { ... }
```

## Best Practices & Anti-Patterns

### 🟢 Do

- Use `pkg/response` for **ALL** API responses.
- Use `pkg/errors` for **ALL** custom errors.
- Validate all inputs in `process_request.go` using `req.validate()`.
- Use `models.Scope` to pass User ID and Context.

### 🔴 Don't

- **Don't** return GORM/Mongo models directly in API responses. Use Presenter DTOs.
- **Don't** put business logic in Handlers.
- **Don't** import `repository` package in `delivery` layer.
