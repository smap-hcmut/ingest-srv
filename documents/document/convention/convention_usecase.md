# UseCase Layer Convention (`convention_usecase.md`)

> **Role**: The UseCase Layer is the **Brain**. It houses all Business Logic.
> **Motto**: "Pure Go. No Frameworks. Strict Typing."

## 1. The Logic Flow

```mermaid
graph TD
    A[Input DTO] --> B{Validation};
    B -->|Structural| C[Fail Fast];
    B -->|Business| D[Check Logic];
    D -->|Valid| E[Repository / Domain Service];
    E -->|Data| F[Construct Output];
    F --> G[Output DTO];
```

---

## 2. Core Principles

1.  **Framework Agnostic**: This layer **MUST NOT** import `gin`, `sqlboiler`, `mongo-driver`, or `amqp`. It only knows about _Interfaces_ and _Standard Library_.
2.  **Interface Driven**: Dependencies are injected.
3.  **Strict Inputs/Outputs**: Every public method uses a dedicated struct in `types.go`.
4.  **One File, One Concern**: Logic files contain ONLY logic. Struct definitions live in type files.

---

## 2.1 File Structure & Splitting Rules

The UseCase layer follows the **File Responsibility Principle**: every file has exactly one job.

### Module Root Files

| File           | Contents                                                              |
| -------------- | --------------------------------------------------------------------- |
| `interface.go` | `UseCase` interface definition                                        |
| `types.go`     | ALL public Input/Output structs (e.g., `CreateInput`, `DetailOutput`) |
| `errors.go`    | Module-specific sentinel errors (e.g., `ErrUserNotFound`)             |

### Implementation Files (`usecase/`)

| File           | Contents                                                                                                               |
| -------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `new.go`       | **Factory ONLY**: `implUseCase` struct + `New()` + optional setter methods. **No interfaces, no types, no constants.** |
| `<entity>.go`  | Interface method implementations for a specific entity (e.g., `event.go`, `user.go`)                                   |
| `<concern>.go` | Logic separated by business concern (e.g., `session.go`, `notification.go`, `oauth.go`)                                |
| `helpers.go`   | Shared private helper methods used across multiple logic files                                                         |
| `types.go`     | Private struct definitions internal to the usecase package (NOT public Input/Output types)                             |

### Example: Complex Module

```text
internal/authentication/
├── interface.go            # UseCase interface
├── types.go                # OAuthCallbackInput, TokenValidationResult, etc.
├── errors.go               # ErrUserNotFound, ErrDomainNotAllowed, etc.
└── usecase/
    ├── new.go              # implUsecase struct + New() + setters
    ├── authentication.go   # GetCurrentUser, Logout, ValidateToken
    ├── oauth.go            # ProcessOAuthCallback, InitiateOAuthLogin
    ├── helpers.go          # createSession, generateToken (private)
    ├── session.go          # SessionManager (concern-based split)
    └── types.go            # Private types used only within usecase package
```

### Splitting Decision Tree

```mermaid
graph TD
    A["New struct/interface?"] -->|Public Input/Output| B["types.go (module root)"]
    A -->|Private helper struct| C["usecase/types.go"]
    A -->|Dependency interface| D["Import from pkg/ or define in interface.go"]
    E["New method?"] -->|Interface method| F["usecase/<entity>.go"]
    E -->|Private helper| G["usecase/helpers.go"]
    E -->|Distinct business concern| H["usecase/<concern>.go"]
```

---

## 3. Implementation Details

### 3.1 Business vs. Structural Validation

Distinguish these two.

- **Structural (Delivery Layer)**: Simple format checks.
  - _Eg: "Is email valid?", "Is Age > 0?"_ - **Do this in Delivery.**
- **Business (UseCase Layer)**: Logic that requires knowledge of the system state.
  - _Eg: "Does this user exist?", "Is the shop active?", "Does user have permission X?"_ - **Do this in UseCase.**

### 3.2 The `New` Factory (`new.go`)

`new.go` is **strictly a factory**. It contains ONLY:

- The `implUseCase` struct (private)
- The `New()` constructor
- Optional setter methods for late-bound dependencies

> [!CAUTION]
> **NEVER** define interfaces, constants, helper types, or business logic in `new.go`. If you need a dependency interface, import it from `pkg/` or define it in the module's `interface.go`.

```go
// ✅ Correct: new.go contains ONLY struct + factory + setters
type implUseCase struct {
    repo   Repository
    l      pkgLog.Logger
    userUC user.UseCase
}

func New(repo Repository, l pkgLog.Logger, userUC user.UseCase) UseCase {
    return &implUseCase{
        repo:   repo,
        l:      l,
        userUC: userUC,
    }
}

func (u *implUseCase) SetAuditPublisher(p audit.Publisher) {
    u.auditPublisher = p
}
```

```go
// ❌ Wrong: defining interfaces inside new.go
type OAuthProvider interface { ... }  // Move to pkg/oauth or interface.go
type RateLimiter interface { ... }    // Move to pkg/ or separate file
```

### 3.3 Concurrency (Using `errgroup`)

Use `errgroup` for parallel processing. **NEVER** use naked `go func()` in UseCase without a wait mechanism (unless it's fire-and-forget, which should go to a Job/MQ).

```go
import "golang.org/x/sync/errgroup"

func (uc *implUseCase) Dashboard(ctx context.Context, id string) (DashboardOut, error) {
    g, ctx := errgroup.WithContext(ctx)
    var user models.User
    var stats models.Stats

    // 1. Fetch User
    g.Go(func() error {
        var err error
        user, err = uc.repo.GetUser(ctx, id)
        return err
    })

    // 2. Fetch Stats
    g.Go(func() error {
        var err error
        stats, err = uc.repo.GetStats(ctx, id)
        return err
    })

    // 3. Wait
    if err := g.Wait(); err != nil {
        uc.l.Errorf(ctx, "Dashboard failed: %v", err)
        return DashboardOut{}, err
    }

    return DashboardOut{User: user, Stats: stats}, nil
}
```

---

## 4. Anti-Patterns (The "Strictly Forbidden" List)

- ❌ **Forbidden Imports**:
  - `github.com/gin-gonic/gin` (This belongs in Delivery)
  - `github.com/volatiletech/sqlboiler` (This belongs in Repository)
- ❌ **Returning Models**:
  - Returning `models.User` directly exposes DB structure. Wrap it in `UserOutput`.
- ❌ **Silent Logs**:
  - `if err != nil { log.Error(err); return nil }` -> **WRONG**. Return the error!
- ❌ **Defining structs in logic files**:
  - All public types go in `types.go` (module root). All private types go in `usecase/types.go`.
- ❌ **God files**:
  - A single `usecase.go` with 800+ lines handling multiple concerns. Split by entity/concern.
- ❌ **Defining interfaces in `new.go`**:
  - `new.go` is factory-only. Dependency interfaces belong in `pkg/` or `interface.go`.

---

## 5. Intern Checklist (Read before PR)

- [ ] **Import Check**: Did I import `gin` or `http`? **Remove it.**
- [ ] **Type Check**: Am I using `CreateInput` and `CreateOutput` structs from `types.go`?
- [ ] **Error Check**: Am I wrapping errors or using `errors.go` definitions?
- [ ] **Logic Check**: Is this logic re-usable? If I call this from a CLI or Job, will it work? (If it relies on HTTP headers, the answer is NO).
- [ ] **File Split Check**: Does my logic file contain any `type` declarations? **Move them to `types.go`.**
- [ ] **`new.go` Check**: Does `new.go` contain anything other than struct + factory + setters? **Move it out.**
