# Delivery Layer Convention (`convention_delivery.md`)

> **Role**: The Delivery Layer is the **Entry Point**. It handles "How data gets IN and OUT".
> **Motto**: "Validate strictly, pass quickly, map errors."

## 1. The Request Lifecycle

Understand this flow. If you break it, you break the architecture.

```mermaid
graph TD
    A[Request (HTTP/MQ)] --> B{Handler};
    B -->|1. Bind & Validate| C[Process Request];
    C -->|Request DTO| B;
    B -->|2. Call Business Logic| D[UseCase];
    D -->|3. Return Domain Object| B;
    B -->|4. Map to Response| E[Presenter];
    E -->|Response DTO| F[Client];
```

## 2. Directory Structure & Naming (HTTP)

```text
internal/<module>/delivery/http/
├── handlers.go         # THE CONTROLLER. One handler per usecase action (Index, Create, ...).
├── process_request.go  # THE GATEKEEPER. processXxxReq(c) → (XxxReq, error). Bind, validate, scope from context.
├── presenters.go       # THE TRANSLATOR. DTOs: XxxReq, XxxResp; toInput(), newXxxResp().
├── routes.go           # THE MAP. RegisterRoutes(rg, mw). Paths simple: /index, /internal/...
├── errors.go           # THE JUDGE. mapError(domain err) → HTTP err.
└── new.go              # THE FACTORY. Handler interface + New(l, uc, discord).
```

### 2.1 HTTP Naming Convention

| What             | Convention                                      | Example                                    |
| ---------------- | ----------------------------------------------- | ------------------------------------------ |
| Handler method   | Simple, match usecase action                    | `Index`, `Create`, `Detail`, `RetryFailed` |
| Process request  | `processXxxReq(c *gin.Context) (XxxReq, error)` | `processIndexReq`                          |
| Request DTO      | PascalCase, suffix Req                          | `IndexReq`                                 |
| Response DTO     | PascalCase, suffix Resp                         | `IndexResp`                                |
| Response builder | `newXxxResp(output) XxxResp`                    | `newIndexResp`                             |
| Route path       | Simple, no redundant verb                       | `POST /internal/index`                     |

---

## 3. Strict Implementation Rules

### 3.1 `handlers.go` (The Controller)

**Goal**: Keep it simple. It should read like a recipe.

- **DO**: Call `processRequest`, check error, call `UseCase`, check error, return `Response`.
- **DON'T**: Write business logic here (e.g., "if user is active...").
- **DON'T**: Access the database directly.

**Standard Pattern** (scope in context, not param):

```go
// @Summary Full Swagger Documentation is MANDATORY
// @Router /api/v1/resource [POST]
func (h *handler) Create(c *gin.Context) {
    ctx := c.Request.Context()

    // 1. Process Request (Bind + Validate; scope from context)
    req, err := h.processCreateReq(c)
    if err != nil {
        response.Error(c, err, h.discord)
        return
    }

    // 2. Call UseCase
    output, err := h.uc.Create(ctx, req.toInput())
    if err != nil {
        h.l.Errorf(ctx, "uc.Create: %v", err)
        response.Error(c, h.mapError(err), h.discord)
        return
    }

    // 3. Response
    response.OK(c, h.newCreateResp(output))
}
```

### 3.2 `process_request.go` (The Gatekeeper)

**Goal**: Ensure garbage never reaches the UseCase.

- **Role**: Bind JSON/Query -> Validate format -> Extract Scope -> Convert to UseCase Input.
- **Validation**:
  - **Structural**: `email`, `required`, `uuid`, `min=0`. (DO THIS HERE).
  - **Business**: "Email already exists". (DO NOT DO THIS HERE. UseCase does this).

**Example** (scope from context):

```go
func (h *handler) processCreateReq(c *gin.Context) (CreateReq, error) {
    var req CreateReq
    if err := c.ShouldBindJSON(&req); err != nil {
        return req, err
    }
    if err := req.validate(); err != nil {
        return req, err
    }
    // Scope: from context (middleware/service auth). If missing, return error or set default in handler setup.
    sc := scope.GetScopeFromContext(c.Request.Context())
    if sc.UserID == "" {
        return req, errors.New("scope not found")
    }
    return req, nil
}
```

### 3.3 `presenters.go` (The Translator)

**Goal**: Keep various layers decoupled.

- **DTOs**: Define `private` structs for HTTP bodies.
- **Why?**: If DB model changes, API shouldn't break. If API changes, DB shouldn't break.
- **Pointers**: Use `*string`, `*int` for optional fields to distinguish `nil` (missing) vs `""` (empty).

```go
type CreateReq struct {
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"email"`
}

func (r CreateReq) validate() error { /* structural only */ return nil }

// Mapper: Request DTO -> UseCase Input
func (r CreateReq) toInput() indexing.CreateInput {
    return indexing.CreateInput{Name: r.Name, Email: r.Email}
}

type CreateResp struct { ... }
func (h *handler) newCreateResp(output indexing.CreateOutput) CreateResp { ... }
```

### 3.4 `errors.go` (The Judge)

**Goal**: Translate Domain Errors to HTTP Status Codes.

- **Pattern**: Use `pkg/errors` and `switch` statements.
- **Unknown Errors**: `panic` in DEV/TEST (to catch bugs), `500` in PROD.

```go
func (h *handler) mapError(err error) error {
    switch {
    case errors.Is(err, uc.ErrUserNotFound):
        return pkgErrors.ErrNotFound // 404
    case errors.Is(err, uc.ErrEmailDuplicate):
        return pkgErrors.NewHTTPError(409, "Email already exists")
    default:
        // Critical: Force developers to handle errors during development!
        if config.IsDev() {
            panic(err)
        }
        return pkgErrors.ErrInternalServerError // 500
    }
}
```

---

## 4. Intern Checklist (Read before PR)

- [ ] **Swagger Check**: Did I add a full Swagger block? (Summary, Param, Success, Failure).
- [ ] **Validation Check**: Did I restrict inputs? (e.g., `limit` max 100, `offset` min 0).
- [ ] **Business Logic Check**: Did I accidentally put logic in the handler? (e.g., `if status == "active"`). **MOVE IT TO USECASE**.
- [ ] **Error Map Check**: Did I map the UseCase error in `errors.go`? Or it falls to default 500?
- [ ] **Naming Check**: Are handler methods simple? (`Create`, `Detail` - NOT `CreateUser`).

---

## 5. Kafka (MQ) Delivery — Same Idea as HTTP

**Principle**: Consumer layer = **thin**, and **caller depends on interface** (like `http.Handler`). Receive message → normalize → pass input to usecase. **No business logic** in consumer.

### 5.1 Directory Structure & Interface

```text
internal/<module>/delivery/kafka/
├── type.go (or messages.go)   # Shared message DTOs (wire format). Optional if only consumer uses them.
internal/<module>/delivery/kafka/consumer/
├── new.go         # Consumer interface + concrete consumer struct + New(cfg) (Consumer, error)
├── consumer.go    # Start consuming: ConsumeXxx(ctx, topic) error — like "register topic"
├── handler.go     # Sarama ConsumerGroupHandler: Setup, Cleanup, ConsumeClaim → dispatch to handleXxxMessage
├── workers.go     # handleXxxMessage(msg): unmarshal, validate format, set scope, toInput(), uc.Call()
└── presenters.go  # Message DTO → UseCase input: toIndexInput(msg) IndexInput
```

- **Interface** (in `new.go`): Define `Consumer` with `ConsumeXxx(ctx, topic) error` and `Close() error`. `New(cfg) (Consumer, error)` returns the **interface**, not `*Consumer`. Caller (e.g. `internal/consumer`) depends on `Consumer`, so tests and wiring stay clean.
- **Concrete type**: Use lowercase `consumer` (unexported struct) that implements `Consumer`. Same pattern as HTTP: `Handler` interface, `handler` struct, `New(...) Handler`.

### 5.2 Naming Convention (Kafka)

| What            | Convention                                   | Example                       |
| --------------- | -------------------------------------------- | ----------------------------- |
| Interface       | `Consumer`                                   | Same as HTTP’s `Handler`      |
| Start consuming | `ConsumeXxx(ctx, topic) error`               | `ConsumeBatchCompleted`       |
| Message handler | `handleXxxMessage(msg) error`                | `handleBatchCompletedMessage` |
| Message → Input | `toXxxInput(msg) UseCaseInput` in presenters | `toIndexInput`                |
| Message DTO     | In `delivery/kafka` or consumer `presenters` | `BatchCompletedMessage`       |

### 5.3 Responsibilities (Thin Layer)

Consumer **only**:

1. **Receive** message (Sarama/consumer API).
2. **Unmarshal** into delivery DTO (e.g. `BatchCompletedMessage`).
3. **Validate format** (required fields). Skip/ACK invalid messages.
4. **Set scope** in context (e.g. system scope).
5. **Map to usecase input** via presenters (e.g. `toIndexInput(message)`).
6. **Call one usecase method** (e.g. `uc.Index(ctx, input)`).
7. **Log / ACK or NACK** per strategy.

**Naming**: `handleXxxMessage` = delivery handler (not business “process”). Business logic lives in **usecase** only.

**Ack/Nack**:

- `Unmarshal Error` → **ACK** (discard poison).
- `Business Error` (from usecase) → **ACK** (or NACK if retry).
- `Transient Error` (e.g. DB timeout) → **NACK** (retry).

---

## 6. Jobs (Cron / Background)

- **Context**: ALWAYS `context.Background()`.
- **Logging**: ALWAYS log Start and End of the job.
- **Panic**: NEVER panic. Catch all errors.
