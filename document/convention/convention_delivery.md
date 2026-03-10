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

## 2. Directory Structure & Responsibilities

```text
internal/<module>/delivery/http/
├── handlers.go         # THE CONTROLLER. Coordinates flow.
├── process_request.go  # THE GATEKEEPER. Binding & Validation.
├── presenters.go       # THE TRANSLATOR. Request/Response Structs.
├── routes.go           # THE MAP. URL definitions.
├── errors.go           # THE JUDGE. Error mapping.
└── new.go              # THE FACTORY. Dependency Injection.
```

---

## 3. Strict Implementation Rules

### 3.1 `handlers.go` (The Controller)

**Goal**: Keep it simple. It should read like a recipe.

- **DO**: Call `processRequest`, check error, call `UseCase`, check error, return `Response`.
- **DON'T**: Write business logic here (e.g., "if user is active...").
- **DON'T**: Access the database directly.

**Standard Pattern**:

```go
// @Summary Full Swagger Documentation is MANDATORY
// @Router /api/v1/resource [POST]
func (h handler) Create(c *gin.Context) {
    // 1. Process Request (Bind + Validate)
    input, sc, err := h.processCreateRequest(c)
    if err != nil {
        response.Error(c, err)
        return
    }

    // 2. Call UseCase
    output, err := h.uc.Create(c.Request.Context(), sc, input)
    if err != nil {
        // Log generic errors, but let mapError decide the status code
        h.l.Errorf(c.Request.Context(), "uc.Create: %v", err)
        response.Error(c, h.mapError(err))
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

**Example**:

```go
func (h handler) processCreateRequest(c *gin.Context) (uc.CreateInput, models.Scope, error) {
    // 1. Scope (User Context)
    sc, err := pkgScope.GetScope(c)
    if err != nil {
        return uc.CreateInput{}, models.Scope{}, pkgErrors.ErrUnauthorized
    }

    // 2. Bind & Validate DTO
    var req createReq // Defined in presenters.go
    if err := c.ShouldBindJSON(&req); err != nil {
        return uc.CreateInput{}, models.Scope{}, pkgErrors.ErrInvalidBody
    }
    if err := req.validate(); err != nil { // Custom validation logic
        return uc.CreateInput{}, models.Scope{}, err
    }

    // 3. Convert to UseCase Input
    return req.toInput(), sc, nil
}
```

### 3.3 `presenters.go` (The Translator)

**Goal**: Keep various layers decoupled.

- **DTOs**: Define `private` structs for HTTP bodies.
- **Why?**: If DB model changes, API shouldn't break. If API changes, DB shouldn't break.
- **Pointers**: Use `*string`, `*int` for optional fields to distinguish `nil` (missing) vs `""` (empty).

```go
type createReq struct {
    Name  string `json:"name" binding:"required"`
    Email string `json:"email" binding:"email"`
}

// Mapper: Request DTO -> UseCase Input
func (r createReq) toInput() event.CreateInput {
    return event.CreateInput{
        Name:  r.Name,
        Email: r.Email,
    }
}
```

### 3.4 `errors.go` (The Judge)

**Goal**: Translate Domain Errors to HTTP Status Codes.

- **Pattern**: Use `pkg/errors` and `switch` statements.
- **Unknown Errors**: `panic` in DEV/TEST (to catch bugs), `500` in PROD.

```go
func (h handler) mapError(err error) error {
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

## 5. Job & MQ Delivery

- **Jobs**:
  - **Context**: ALWAYS `context.Background()`.
  - **Logging**: ALWAYS log `Start` and `End` of the job.
  - **Panic**: NEVER panic. Catch all errors.
- **MQ Consumers**:
  - **Ack/Nack**:
    - `Unmarshal Error` -> **ACK** (Discard poison message).
    - `Business Error` -> **ACK** (Usually, unless retry strategy exists).
    - `Transient Error` (DB Timeout) -> **NACK** (Retry).
