# Repository Layer Convention (`convention_repository.md`)

> **Role**: The Repository Layer handles **Data Access**.
> **Motto**: "One interface, multiple drivers, strict splitting, standard naming."

## 1. The "Split Pattern" (MANDATORY)

Repositories often become "God Objects" with 1000+ lines. To prevent this, strict file splitting is **ENFORCED**.

### 1.1 Per-entity files (PostgreSQL / SQLBoiler)

For each **entity** (table) in `repository/<driver>/` you have:

1. **`<entity>.go` (Coordinator)**
   - Methods gọi `buildXxxQuery(opt)` → driver → map sang `model.Entity`.
   - Ví dụ: `document.go` cho bảng indexed_documents, `dlq.go` cho indexing_dlq.

2. **`<entity>_query.go` (Builder)**
   - Chỉ build query: trả về `[]qm.QueryMod` (hoặc filter tương đương).
   - Không gọi DB, không map domain.
   - Ví dụ: `document_query.go`, `dlq_query.go`.

3. **Mapper**
   - Dùng `internal/model.NewXxxFromDB(dbRow)` (và `util.MapSlice` nếu cần).
   - Không bắt buộc file riêng `<entity>_mapper.go` nếu đã map trong `<entity>.go`.

---

## 2. Standard Method Names (MANDATORY)

Repository methods follow **standard verb + entity** naming. Khi một interface gộp nhiều entity (Document + DLQ), dùng **suffix entity** (CreateDocument, GetOneDLQ, ...) để tránh trùng tên.

### 2.1 CRUD (pattern: Verb + Entity)

| Verb           | Purpose             | Document (indexed_documents)                         | DLQ (indexing_dlq)       |
| -------------- | ------------------- | ---------------------------------------------------- | ------------------------ |
| Create         | Insert single       | CreateDocument(ctx, opt)                             | CreateDLQ(ctx, opt)      |
| Upsert         | Insert or update    | UpsertDocument(ctx, opt)                             | —                        |
| Detail         | Get by ID only      | DetailDocument(ctx, id)                              | —                        |
| GetOne         | Get by filters      | GetOneDocument(ctx, opt)                             | GetOneDLQ(ctx, opt)      |
| Get            | List + pagination   | GetDocuments(ctx, opt) → ([]model, paginator, error) | —                        |
| List           | List no pagination  | ListDocuments(ctx, opt)                              | ListDLQs(ctx, opt)       |
| Update\*       | Update by ID/status | UpdateDocumentStatus(ctx, opt)                       | MarkResolvedDLQ(ctx, id) |
| Delete/Deletes | Delete              | (nếu cần)                                            | —                        |

\* Có thể dùng tên cụ thể: UpdateDocumentStatus, MarkResolvedDLQ.

**Important Notes:**

- ✅ **Scope in Context**: `model.Scope` is extracted from `context.Context` using `scope.GetScopeFromContext()`, NOT passed as parameter
- ❌ **NO `Exists` methods**: Use `GetOne` and check if result is empty
- ✅ **Return entities**: `Create`, `Update`, `Upsert` return the modified entity (value type)
- ✅ **`Get` returns `paginator.Paginator`**: NOT `int` for total count

### 2.2 Specialized (theo entity)

Ví dụ Document: `CountDocumentsByProject(ctx, projectID string) (DocumentProjectStats, error)`.

### 2.3 Qdrant (vector store)

- `UpsertPoint(ctx, opt UpsertPointOptions) error` — collection name là const/config ở tầng repo qdrant, không truyền từ usecase.

**Tránh:** `ExistsByX` (dùng GetOne), `FindByX` (dùng GetOne/List), method đặc thù thay cho GetOne/List/Get.

---

## 3. Options Pattern (MANDATORY)

### 3.1 Flow: UseCase ↔ Repository

```
UseCase (domain models)
    ↓ Convert to Options
Repository Interface (Options)
    ↓ Use Options to build query
Driver (SQLBoiler/Mongo)
    ↓ Convert to domain models
UseCase (domain models)
```

### 3.2 Naming Convention

Options định nghĩa trong `repository/option.go`, suffix `Options`. Khi nhiều entity: dùng prefix/suffix entity (CreateDocumentOptions, GetOneDLQOptions, UpsertPointOptions).

```go
// repository/option.go — Document
type CreateDocumentOptions struct { AnalyticsID, ProjectID, SourceID, QdrantPointID, CollectionName, ContentHash, Status, ... }
type UpsertDocumentOptions struct { ... }
type GetOneDocumentOptions struct { AnalyticsID string; ContentHash string }  // AND khi cả hai có
type GetDocumentsOptions struct { Status, ProjectID, BatchID, ErrorTypes, MaxRetry, StaleBefore, Limit, Offset, OrderBy }
type ListDocumentsOptions struct { Status, ProjectID, BatchID, ErrorTypes, MaxRetry, StaleBefore, Limit, OrderBy }
type UpdateDocumentStatusOptions struct { ID, Status, Metrics DocumentStatusMetrics }

// DLQ
type CreateDLQOptions struct { ... }
type GetOneDLQOptions struct { ID, AnalyticsID, ContentHash }
type ListDLQOptions struct { ProjectID, ErrorTypes, ResolvedOnly, UnresolvedOnly, Limit, OrderBy }

// Qdrant
type UpsertPointOptions struct { PointID string; Vector []float32; Payload map[string]interface{} }
```

### 3.3 Rules

- ✅ **DO**: Use `Options` for ALL operations (Create, Upsert, GetOne, Get, List)
- ✅ **DO**: Return `model.Entity` from repository (NOT SQLBoiler types)
- ✅ **DO**: Apply all filters provided in Options (AND condition)
- ❌ **DON'T**: Pass `model.Entity` as parameter to repository
- ❌ **DON'T**: Pass individual fields as separate parameters
- ❌ **DON'T**: Return SQLBoiler/Mongo types from repository interface
- ❌ **DON'T**: Add business logic in repository (validation, if/else for filter selection)

**Important**:

- **UseCase → Repository**: Always use `Options` structs
- **Repository → UseCase**: Always return `model.Entity` types
- Repository is a "dumb" data access layer. If `Options` has multiple filters, apply ALL of them with AND. Business logic (e.g., "only use one filter") belongs in UseCase layer.

---

## 4. PostgreSQL Implementation (SQLBoiler)

We use **SQLBoiler**. It generates type-safe Go structs from the DB schema.

### 4.1 Directory Structure

```text
repository/
├── interface.go   # PostgresRepository (Document + DLQ), QdrantRepository
├── option.go      # Tất cả Options (CreateDocumentOptions, UpsertPointOptions, ...)
├── errors.go      # ErrFailedToInsert, ErrFailedToGet, ... (domain repo errors)
├── postgre/
│   ├── new.go           # New(db, log) PostgresRepository
│   ├── document.go      # CreateDocument, DetailDocument, GetOneDocument, GetDocuments, ListDocuments, UpsertDocument, UpdateDocumentStatus, CountDocumentsByProject
│   ├── document_query.go # buildGetOneQuery, buildGetCountQuery, buildGetQuery, buildListQuery
│   ├── dlq.go           # CreateDLQ, GetOneDLQ, ListDLQs, MarkResolvedDLQ
│   └── dlq_query.go     # buildGetOneDLQQuery, buildListDLQQuery
└── qdrant/
    ├── new.go    # New(client, log) QdrantRepository
    └── point.go  # UpsertPoint (collection name const trong package)
```

### 4.2 Method Implementation Pattern

#### Example: CreateDocument

```go
// document.go
func (r *implPostgresRepository) CreateDocument(ctx context.Context, opt repo.CreateDocumentOptions) (model.IndexedDocument, error) {
    dbDoc := &sqlboiler.IndexedDocument{
        AnalyticsID: opt.AnalyticsID,
        ProjectID:   opt.ProjectID,
        // ... map from opt
    }
    if err := dbDoc.Insert(ctx, r.db, boil.Infer()); err != nil {
        return model.IndexedDocument{}, repo.ErrFailedToInsert
    }
    if doc := model.NewIndexedDocumentFromDB(dbDoc); doc != nil {
        return *doc, nil
    }
    return model.IndexedDocument{}, nil
}
```

#### Example: DetailDocument (Get by ID only)

```go
// document.go
func (r *implPostgresRepository) DetailDocument(ctx context.Context, id string) (model.IndexedDocument, error) {
    dbDoc, err := sqlboiler.FindIndexedDocument(ctx, r.db, id)
    if err == sql.ErrNoRows {
        return model.IndexedDocument{}, nil
    }
    if err != nil {
        return model.IndexedDocument{}, repo.ErrFailedToGet
    }
    if doc := model.NewIndexedDocumentFromDB(dbDoc); doc != nil {
        return *doc, nil
    }
    return model.IndexedDocument{}, nil
}
```

#### Example: GetOneDocument + buildGetOneQuery

```go
// document.go
func (r *implPostgresRepository) GetOneDocument(ctx context.Context, opt repo.GetOneDocumentOptions) (model.IndexedDocument, error) {
    mods := r.buildGetOneQuery(opt)
    dbDoc, err := sqlboiler.IndexedDocuments(mods...).One(ctx, r.db)
    if err == sql.ErrNoRows {
        return model.IndexedDocument{}, nil
    }
    // ...
}

// document_query.go
func (r *implPostgresRepository) buildGetOneQuery(opt repo.GetOneDocumentOptions) []qm.QueryMod {
    var mods []qm.QueryMod
    if opt.AnalyticsID != "" {
        mods = append(mods, qm.Where("analytics_id = ?", opt.AnalyticsID))
    }
    if opt.ContentHash != "" {
        mods = append(mods, qm.Where("content_hash = ?", opt.ContentHash))
    }
    return mods
}
```

#### Example: GetDocuments (with pagination)

```go
// document.go
func (r *implPostgresRepository) GetDocuments(ctx context.Context, opt repo.GetDocumentsOptions) ([]model.IndexedDocument, paginator.Paginator, error) {
    countMods := r.buildGetCountQuery(opt)
    total, err := sqlboiler.IndexedDocuments(countMods...).Count(ctx, r.db)
    // ...
    mods := r.buildGetQuery(opt)
    dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
    // ...
    pag := paginator.Paginator{ Total: int64(total), Count: int64(len(dbDocs)), PerPage: int64(opt.Limit), CurrentPage: (opt.Offset/opt.Limit)+1 }
    return util.MapSlice(dbDocs, model.NewIndexedDocumentFromDB), pag, nil
}

// indexed_document_query.go
func (r *implRepository) buildGetQuery(opt GetOptions) []qm.QueryMod {
    mods := []qm.QueryMod{}

    // Filters
    if opt.Status != "" {
        mods = append(mods, qm.Where("status = ?", opt.Status))
    }
    if opt.ProjectID != "" {
        mods = append(mods, qm.Where("project_id = ?", opt.ProjectID))
    }

    // Sorting
    if opt.OrderBy != "" {
        mods = append(mods, qm.OrderBy(opt.OrderBy))
    } else {
        mods = append(mods, qm.OrderBy("created_at DESC"))
    }

    // Pagination (REQUIRED)
    if opt.Limit > 0 {
        mods = append(mods, qm.Limit(opt.Limit))
    }
    if opt.Offset > 0 {
        mods = append(mods, qm.Offset(opt.Offset))
    }

    return mods
}
```

#### Example: ListDocuments (no pagination)

```go
// document.go
func (r *implPostgresRepository) ListDocuments(ctx context.Context, opt repo.ListDocumentsOptions) ([]model.IndexedDocument, error) {
    mods := r.buildListQuery(opt)
    dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
    if err != nil {
        return nil, repo.ErrFailedToList
    }
    return util.MapSlice(dbDocs, model.NewIndexedDocumentFromDB), nil
}

// document_query.go
func (r *implPostgresRepository) buildListQuery(opt repo.ListDocumentsOptions) []qm.QueryMod {
    var mods []qm.QueryMod
    if opt.Status != "" { mods = append(mods, qm.Where("status = ?", opt.Status)) }
    if opt.Limit > 0 { mods = append(mods, qm.Limit(opt.Limit)) }
    return mods
}
```

### 4.3 Critical Rules

- **Always use `qm`**: Never write raw SQL strings unless absolutely necessary (bulk insert, aggregations).
- **Context**: All DB calls must accept `context.Context`.
- **No Logic**: Do not put business logic (e.g., "defaults") in the repo.
- **Error Wrapping**: Use `fmt.Errorf("MethodName: %w", err)` for context.

---

## 5. Existence Checks: Use GetOne, NOT Exists

### ❌ WRONG: Creating Exists methods

```go
// DON'T do this
func (r *implRepository) ExistsByAnalyticsID(ctx context.Context, analyticsID string) (bool, error) {
    return sqlboiler.IndexedDocuments(
        qm.Where("analytics_id = ?", analyticsID),
    ).Exists(ctx, r.db)
}
```

### ✅ CORRECT: Use GetOne

```go
// In UseCase
doc, err := repo.GetOneDocument(ctx, repository.GetOneDocumentOptions{
    AnalyticsID: analyticsID,
})
if err != nil {
    return err
}
if doc.ID != "" {
    // Record exists
    return ErrDuplicate
}
// Record does not exist, proceed...
```

### Why?

1. **Simplicity**: One method (`GetOne`) instead of multiple `Exists` methods
2. **Flexibility**: If you need the data later, you already have it
3. **Performance**: Modern databases optimize `SELECT *` with `LIMIT 1` very well
4. **Consistency**: All queries go through the same pattern

### Performance Comparison

| Approach   | Query                        | Performance | Use Case                |
| ---------- | ---------------------------- | ----------- | ----------------------- |
| `Exists()` | `SELECT EXISTS(...)`         | ⚡ 1-2ms    | Only check existence    |
| `GetOne()` | `SELECT * WHERE ... LIMIT 1` | ⚡ 1-3ms    | Check + might need data |

**Difference: Negligible (< 1ms)** for indexed columns. Use `GetOne` for simplicity.

---

## 6. Exists vs GetOne vs Detail

- **Không dùng method `Exists`**: Dùng `GetOne` rồi kiểm tra entity rỗng (ID == "" hoặc so sánh với zero value).
- **`DetailDocument(ctx, id)`**: Lấy theo primary key (FindX). Trả về `model.IndexedDocument` (value); not found → zero value.
- **`GetOneDocument(ctx, opt)`**: Lấy theo filter (AnalyticsID, ContentHash, ...). Trả về value; not found → zero value.

Performance: Detail (PK) và GetOne (index) đều nhanh; không cần thêm Exists.

---

## 7. Interface Composition

Tách theo entity; gộp Postgres (Document + DLQ) thành một interface, Qdrant riêng. Usecase có thể nhận **một** `PostgresRepository` và **một** `QdrantRepository`, hoặc một interface `Repository` embed cả hai (tùy wiring).

```go
// repository/interface.go

// PostgresRepository — Document + DLQ (một New cho postgre)
type PostgresRepository interface {
    DocumentRepository
    DLQRepository
}

type DocumentRepository interface {
    GetDocuments(ctx context.Context, opt GetDocumentsOptions) ([]model.IndexedDocument, paginator.Paginator, error)
    DetailDocument(ctx context.Context, id string) (model.IndexedDocument, error)
    ListDocuments(ctx context.Context, opt ListDocumentsOptions) ([]model.IndexedDocument, error)
    GetOneDocument(ctx context.Context, opt GetOneDocumentOptions) (model.IndexedDocument, error)
    CreateDocument(ctx context.Context, opt CreateDocumentOptions) (model.IndexedDocument, error)
    UpdateDocumentStatus(ctx context.Context, opt UpdateDocumentStatusOptions) (model.IndexedDocument, error)
    UpsertDocument(ctx context.Context, opt UpsertDocumentOptions) (model.IndexedDocument, error)
    CountDocumentsByProject(ctx context.Context, projectID string) (DocumentProjectStats, error)
}

type DLQRepository interface {
    CreateDLQ(ctx context.Context, opt CreateDLQOptions) (model.IndexingDLQ, error)
    GetOneDLQ(ctx context.Context, opt GetOneDLQOptions) (model.IndexingDLQ, error)
    ListDLQs(ctx context.Context, opt ListDLQOptions) ([]model.IndexingDLQ, error)
    MarkResolvedDLQ(ctx context.Context, id string) error
}

type QdrantRepository interface {
    UpsertPoint(ctx context.Context, opt UpsertPointOptions) error
}
```

- **Factory**: `postgre.New(db, log)` → `PostgresRepository`; `qdrant.New(client, log)` → `QdrantRepository`. Collection name (Qdrant) là const trong `qdrant` package.
- **errors.go**: Định nghĩa domain repo errors (ErrFailedToInsert, ErrFailedToGet, ...) dùng trong postgre/qdrant impl.

---

## 8. Common Patterns

### 8.1 Mapper Helper

Dùng `internal/model.NewIndexedDocumentFromDB(dbDoc)` và `pkg/util.MapSlice(dbDocs, model.NewIndexedDocumentFromDB)` thay vì viết tay `toDomainList`. Return type là value (`model.IndexedDocument`), không pointer.

### 8.2 Transaction Support (Optional)

```go
type Repository interface {
    WithTx(tx *sql.Tx) Repository
}

func (r *implRepository) WithTx(tx *sql.Tx) Repository {
    return &implRepository{
        db: tx, // Use transaction instead of DB
    }
}
```

---

## 9. Intern Checklist (Read before PR)

- [ ] **Naming**: Method names theo chuẩn (CreateDocument, DetailDocument, GetOneDocument, GetDocuments, ListDocuments, UpsertDocument, UpdateDocumentStatus, CountDocumentsByProject; CreateDLQ, GetOneDLQ, ListDLQs, MarkResolvedDLQ; UpsertPoint).
- [ ] **No Exists**: Không tạo method Exists; dùng GetOne rồi kiểm tra entity rỗng.
- [ ] **Options**: Tất cả Options trong `option.go` (CreateDocumentOptions, UpsertPointOptions, ...).
- [ ] **Split**: Code tách `document.go` + `document_query.go`, `dlq.go` + `dlq_query.go`; qdrant `point.go`.
- [ ] **Context**: Mọi call driver đều truyền `ctx`.
- [ ] **Not found**: GetOne/Detail trả về value rỗng (model.IndexedDocument{}), không return error.
- [ ] **Mapping**: Nullable dùng `null.v8` / `model.NewXxxFromDB`; không return sqlboiler/bson ra ngoài interface.
- [ ] **Errors**: Dùng repo errors (ErrFailedToInsert, ...) hoặc wrap với tên method.
