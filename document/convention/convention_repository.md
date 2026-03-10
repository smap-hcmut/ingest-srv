# Repository Layer Convention (`convention_repository.md`)

> **Role**: The Repository Layer handles **Data Access**.
> **Motto**: "One interface, multiple drivers, strict splitting."

## 1. The "Split Pattern" (MANDATORY)

Repositories often become "God Objects" with 1000+ lines. To prevent this, strict file splitting is **ENFORCED**.

### The Triad

For every entity (e.g., `User`), you MUST have 3 files in `repository/<driver>/`:

1.  **`<entity>.go` (The Coordinator)**:
    - **Responsibility**: Orchestrates the flow.
    - **Code**: calls `buildQuery` -> calls Driver -> calls `toDomain`.
2.  **`query.go` (The Builder)**:
    - **Responsibility**: Pure logic. Constructs filters (`bson.M` or `qm.QueryMod`).
    - **Code**: `if id != "" { filter["_id"] = id }`
3.  **`build.go` (The Mapper)**:
    - **Responsibility**: Converts Data Models <-> Domain Models.
    - **Code**: `return models.User{ ID: dbUser.ID }`

---

## 2. PostgreSQL Implementation (SQLBoiler)

We use **SQLBoiler**. It generates type-safe Go structs from the DB schema.

### 2.1 Directory Structure

```text
repository/postgre/
├── user.go         # Coordinator: Find, Create
├── query.go        # Builder: Helper functions returning []qm.QueryMod
└── build.go        # Mapper: sqlboiler.User <-> models.User
```

### 2.2 Critical Rules

- **Always use `qm`**: Never write raw SQL strings unless absolutely necessary (bulk insert).
- **Context**: All DB calls must accept `context.Context`.
- **No Logic**: Do not put business logic (e.g., "defaults") in the repo.

### 2.3 Example: `query.go`

```go
// repo/postgre/query.go
func (r *implRepo) buildListQuery(sc models.Scope, filter repo.Filter) []qm.QueryMod {
    // 1. Base Tenant Filter (Security)
    mods := []qm.QueryMod{
        sqlboiler.UserWhere.ShopID.EQ(sc.ShopID),
    }

    // 2. Dynamic Filters
    if filter.Role != "" {
        mods = append(mods, sqlboiler.UserWhere.Role.EQ(filter.Role))
    }
    if filter.Search != "" {
        // Use ILIKE for case-insensitive search
        mods = append(mods, sqlboiler.UserWhere.Email.ILIKE("%"+filter.Search+"%"))
    }

    return mods
}
```

---

## 3. MongoDB Implementation

### 3.1 Rules

- **Driver**: Use official `mongo-driver`.
- **BSON**: Use `bson.M` for queries, `bson.D` for sorts (order matters).
- **Validation**: Handle `ErrNoDocuments` and return `domain.ErrNotFound` or `nil` (depending on requirement).

### 3.2 Example: `entity.go`

```go
// repo/mongo/event.go
func (r *implRepo) Detail(ctx context.Context, sc models.Scope, id string) (models.Event, error) {
    // 1. Build Query (Logic in query.go)
    filter, err := r.buildDetailQuery(ctx, sc, id)
    if err != nil {
        return models.Event{}, err
    }

    // 2. Execute
    var doc dbEvent
    err = r.col.FindOne(ctx, filter).Decode(&doc)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return models.Event{}, pkgErrors.ErrNotFound
        }
        return models.Event{}, err
    }

    // 3. Map (Logic in build.go)
    return r.toDomain(doc), nil
}
```

---

## 4. Interface Composition

Don't create a massive `Repository` interface. Split it by entity.

```go
// repo_interface.go

// The Main Interface (injected into UseCase)
type Repository interface {
    UserRepository
    EventRepository
}

// Sub-Interface: User
type UserRepository interface {
    CreateUser(ctx context.Context, u models.User) error
    GetUser(ctx context.Context, id string) (models.User, error)
}
```

---

## 5. Intern Checklist (Read before PR)

- [ ] **Split Check**: Did I split my code into `entity.go`, `query.go`, `build.go`?
- [ ] **Context Check**: Am I passing `ctx` to the driver? (Not `context.Background()` inside the method!)
- [ ] **Tenant Check**: Did I enforce `sc.ShopID` in the query builder? **CRITICAL SECURITY RISK IF MISSED.**
- [ ] **Mapping Check**: Did I handle `null` fields correctly? (e.g., `null.String` -> `*string`).
- [ ] **Leak Check**: Am I returning `bson.M` or `sqlboiler.User` out of the interface? **FORBIDDEN**. Return `internal/models`.
