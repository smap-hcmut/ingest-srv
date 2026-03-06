# Domain 4: Report -- Chi tiết Plan Code

**Version:** 1.0  
**Last Updated:** 2026-02-16  
**Domain:** `internal/report` -- Report Generation

---

## I. TỔNG QUAN

### 1. Vai trò Domain

Domain Report chịu trách nhiệm **tạo báo cáo tự động** từ campaign data, xử lý **không đồng bộ** (async):

- Nhận yêu cầu generate report → trả ngay `report_id` → xử lý background
- Hỗ trợ 4 loại báo cáo: `SUMMARY`, `COMPARISON`, `TREND`, `ASPECT_DEEP_DIVE`
- Sử dụng **Map-Reduce Summarization** (Aggregate → Sample → Generate → Compile)
- Chống duplicate bằng `params_hash`
- Upload report file lên MinIO, cung cấp presigned download URL

### 2. Input/Output

**Input:**

- Report request từ HTTP API:
  - `POST /api/v1/reports/generate` → Tạo report (async)
  - `GET /api/v1/reports/{report_id}` → Kiểm tra trạng thái + metadata
  - `GET /api/v1/reports/{report_id}/download` → Tải file report

**Output:**

- Report metadata (status, file_url, timestamps)
- Report file PDF/Markdown lưu trên MinIO
- Presigned download URL cho client

### 3. Kiến trúc tổng quan

```
┌─────────────────────────────────────────────────────────────────┐
│                         Callers                                  │
│  ┌──────────────────┐                                           │
│  │ Web UI           │                                           │
│  │ (POST /generate) │                                           │
│  │ (GET /reports)   │                                           │
│  └────────┬─────────┘                                           │
└───────────┼──────────────────────────────────────────────────────┘
            ▼
┌─────────────────────────────────────────────────────────────────┐
│              Knowledge Service (Domain: Report)                  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                     UseCase                               │   │
│  │  Generate(ctx, sc, input) → (GenerateOutput, error)      │   │
│  │  GetReport(ctx, sc, input) → (ReportOutput, error)       │   │
│  │  DownloadReport(ctx, sc, input) → (DownloadOutput, error)│   │
│  └────────┬──────────┬──────────┬──────────┬────────────────┘   │
│           │          │          │          │                      │
│     ┌─────▼────┐ ┌───▼────┐ ┌──▼─────┐ ┌─▼──────────┐         │
│     │ search   │ │Gemini  │ │MinIO   │ │ PostgreSQL │         │
│     │ .UseCase │ │(LLM)   │ │(files) │ │ (reports)  │         │
│     └──────────┘ └────────┘ └────────┘ └────────────┘         │
└─────────────────────────────────────────────────────────────────┘
```

### 4. Dependencies

| Dependency       | Interface                       | Vai trò                                         |
| :--------------- | :------------------------------ | :---------------------------------------------- |
| `search.UseCase` | `search.UseCase`                | Search documents + Aggregate thống kê từ Qdrant |
| `pkg/gemini`     | `gemini.IGemini`                | Gọi LLM để sinh nội dung report sections        |
| `pkg/minio`      | `minio.IMinIO`                  | Upload report PDF, tạo presigned download URL   |
| PostgreSQL       | `repository.PostgresRepository` | Lưu report metadata + status tracking           |

> [!IMPORTANT]
> Domain report cần `search.UseCase` bổ sung method `Aggregate()` (hiện chưa có). Xem phần **IX. CROSS-DOMAIN CHANGES** bên dưới.

---

## II. DATABASE SCHEMA

### 1. Reports Table

```sql
-- =====================================================
-- Migration: 007 - Create reports table
-- Purpose: Tracking metadata và status của reports
-- Domain: Report (Async Report Generation)
-- Created: 2026-02-16
-- =====================================================

CREATE TABLE IF NOT EXISTS schema_knowledge.reports (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     UUID NOT NULL,              -- Thuộc campaign nào
    user_id         UUID NOT NULL,              -- Ai yêu cầu generate

    -- Report Configuration
    title           VARCHAR(500),               -- Tiêu đề report (auto-generated hoặc user đặt)
    report_type     VARCHAR(50) NOT NULL,       -- 'SUMMARY' | 'COMPARISON' | 'TREND' | 'ASPECT_DEEP_DIVE'
    params_hash     VARCHAR(64) NOT NULL,       -- SHA-256 hash of (campaign_id + report_type + filters)
    filters         JSONB,                      -- Search filters đã dùng: {sentiments, aspects, date_range, ...}

    -- Status Tracking
    status          VARCHAR(20) NOT NULL        -- PROCESSING | COMPLETED | FAILED
                    DEFAULT 'PROCESSING',
    error_message   TEXT,                       -- Lỗi nếu status = FAILED

    -- Output
    file_url        TEXT,                       -- MinIO path: s3://smap-reports/{id}.pdf
    file_size_bytes BIGINT,                     -- Kích thước file (bytes)
    file_format     VARCHAR(10) DEFAULT 'pdf',  -- 'pdf' | 'md'

    -- Performance Metrics
    total_docs_analyzed INT,                    -- Tổng số documents đã phân tích
    sections_count      INT,                    -- Số sections trong report
    generation_time_ms  BIGINT,                 -- Tổng thời gian generate (ms)

    -- Timestamps
    completed_at    TIMESTAMPTZ,                -- Thời điểm hoàn tất
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_reports_campaign ON schema_knowledge.reports(campaign_id);
CREATE INDEX idx_reports_user ON schema_knowledge.reports(user_id);
CREATE INDEX idx_reports_status ON schema_knowledge.reports(status);
CREATE INDEX idx_reports_params_hash ON schema_knowledge.reports(params_hash, status);
CREATE INDEX idx_reports_created ON schema_knowledge.reports(created_at DESC);
```

### 2. Migration Workflow

Thứ tự thực hiện khi triển khai:

1. **Tạo migration file** `migrations/007_create_reports_table.sql` — chứa SQL ở trên
2. **User import migration vào DB** — chạy thủ công migration file trên PostgreSQL
3. **Chạy `make models`** — sinh SQLBoiler models vào `internal/sqlboiler/`
4. **Tạo model mapper** `internal/model/report.go` — chứa `NewReportFromDB()` + `ToDBReport()` (xem section VI)
5. **Tạo repo implementation** — dùng SQLBoiler models thay vì raw SQL

> [!NOTE]
> Migration file đặt trong `migrations/` theo số thứ tự tiếp theo (hiện có 006). Sau khi user xác nhận đã import DB thành công, mới chạy `make models`.

---

## III. DIRECTORY STRUCTURE

```
internal/report/
├── delivery/
│   └── http/
│       ├── new.go                  # Factory: Handler interface + New()
│       ├── handlers.go             # GenerateReport, GetReport, DownloadReport
│       ├── process_request.go      # Validate + extract Scope
│       ├── presenters.go           # Request/Response DTOs
│       ├── routes.go               # Route definitions
│       └── errors.go               # Error mapping
├── repository/
│   ├── interface.go                # ReportRepository interface
│   ├── options.go                  # Filter/Option structs
│   ├── errors.go                   # Sentinel errors
│   └── postgre/
│       ├── new.go                  # Factory
│       ├── report.go               # Create, GetByID, FindByParamsHash, UpdateCompleted, UpdateFailed
│       ├── report_query.go         # buildListQuery, buildFindByParamsHashQuery
│       └── report_build.go         # toDomain, toDB
├── usecase/
│   ├── new.go                      # implUseCase{repo, searchUC, gemini, minio, logger, cfg}
│   ├── report.go                   # Generate (async), GetReport, DownloadReport
│   ├── generator.go                # generateInBackground: Aggregate → Sample → Generate → Compile
│   ├── template.go                 # Report templates per type (SUMMARY, COMPARISON, TREND, ASPECT_DEEP_DIVE)
│   └── helpers.go                  # buildSectionPrompt, compileReport, generateParamsHash
├── interface.go                    # UseCase interface
├── types.go                        # GenerateInput, GenerateOutput, ReportOutput, etc.
└── errors.go                       # Domain errors
```

---

## IV. DOMAIN-LEVEL FILES

### A. TYPES (`types.go`)

```go
package report

import "time"

type GenerateInput struct {
	CampaignID string        // campaign scope
	Title      string        // optional, auto-generated nếu rỗng
	ReportType string        // SUMMARY | COMPARISON | TREND | ASPECT_DEEP_DIVE
	Filters    ReportFilters // optional search filters
}

type ReportFilters struct {
	Sentiments []string // filter by sentiments
	Aspects    []string // filter by aspects
	Platforms  []string // filter by platforms
	DateFrom   *int64   // unix timestamp
	DateTo     *int64   // unix timestamp
	RiskLevels []string // filter by risk levels
}

type GetReportInput struct {
	ReportID string
}

type DownloadReportInput struct {
	ReportID string
}

type GenerateOutput struct {
	ReportID string // ID of created report
	Status   string // "PROCESSING" | "COMPLETED"
	Message  string // human-readable status message
}

type ReportOutput struct {
	ID                string
	CampaignID        string
	UserID            string
	Title             string
	ReportType        string
	Status            string
	ErrorMessage      string
	FileFormat        string
	FileSizeBytes     int64
	TotalDocsAnalyzed int
	SectionsCount     int
	GenerationTimeMs  int64
	CompletedAt       *time.Time
	CreatedAt         time.Time
}

type DownloadOutput struct {
	DownloadURL string // presigned MinIO URL
	FileName    string // e.g. "report_rpt_001.pdf"
	FileSize    int64
	ExpiresAt   time.Time
}

type ReportSection struct {
	Title   string
	Content string
}

type SectionTemplate struct {
	Title      string
	PromptTmpl string // Go template string for LLM prompt
}
```

### B. ERRORS (`errors.go`)

```go
package report

import "errors"

var (
	ErrReportNotFound         = errors.New("report: report not found")
	ErrReportNotCompleted     = errors.New("report: report not yet completed")
	ErrCampaignNotFound       = errors.New("report: campaign not found")
	ErrInvalidReportType      = errors.New("report: invalid report type")
	ErrGenerationFailed       = errors.New("report: generation failed")
	ErrDuplicateProcessing    = errors.New("report: report is already being generated")
	ErrDownloadURLFailed      = errors.New("report: failed to generate download URL")
)
```

### C. INTERFACE (`interface.go`)

```go
package report

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	// Report Generation
	Generate(ctx context.Context, sc model.Scope, input GenerateInput) (GenerateOutput, error)

	// Report Retrieval
	GetReport(ctx context.Context, sc model.Scope, input GetReportInput) (ReportOutput, error)

	// Report Download
	DownloadReport(ctx context.Context, sc model.Scope, input DownloadReportInput) (DownloadOutput, error)
}
```

---

## V. REPOSITORY LAYER

### A. INTERFACE (`repository/interface.go`)

```go
package repository

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name PostgresRepository
type PostgresRepository interface {
	ReportRepository
}

type ReportRepository interface {
	CreateReport(ctx context.Context, opt CreateReportOptions) (model.Report, error)
	GetReportByID(ctx context.Context, id string) (model.Report, error)
	FindByParamsHash(ctx context.Context, opt FindByParamsHashOptions) (model.Report, error)
	UpdateCompleted(ctx context.Context, opt UpdateCompletedOptions) error
	UpdateFailed(ctx context.Context, opt UpdateFailedOptions) error
	ListReports(ctx context.Context, opt ListReportsOptions) ([]model.Report, error)
}
```

### B. OPTIONS (`repository/options.go`)

```go
package repository

import "encoding/json"

type CreateReportOptions struct {
	CampaignID string
	UserID     string
	Title      string
	ReportType string          // SUMMARY | COMPARISON | TREND | ASPECT_DEEP_DIVE
	ParamsHash string          // SHA-256 hash
	Filters    json.RawMessage // nullable JSON
}

type FindByParamsHashOptions struct {
	ParamsHash string
	Status     string // filter by status (e.g. "PROCESSING", "COMPLETED")
}

type UpdateCompletedOptions struct {
	ReportID          string
	FileURL           string
	FileSizeBytes     int64
	TotalDocsAnalyzed int
	SectionsCount     int
	GenerationTimeMs  int64
}

type UpdateFailedOptions struct {
	ReportID     string
	ErrorMessage string
}

type ListReportsOptions struct {
	CampaignID string
	UserID     string
	Status     string // optional filter
	Limit      int
	Offset     int
}
```

### C. ERRORS (`repository/errors.go`)

```go
package repository

import "errors"

var (
	ErrFailedToInsert = errors.New("failed to insert")
	ErrFailedToGet    = errors.New("failed to get")
	ErrFailedToList   = errors.New("failed to list")
	ErrFailedToUpdate = errors.New("failed to update")
	ErrNotFound       = errors.New("not found")
)
```

### D. REPOSITORY IMPLEMENTATION (`repository/postgre/report.go`)

```go
package postgre

import (
	"context"
	"database/sql"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/sqlboiler"
	"knowledge-srv/pkg/util"
)

func (r *implRepository) CreateReport(ctx context.Context, opt repository.CreateReportOptions) (model.Report, error) {
	dbReport := buildCreateReport(opt)

	if err := dbReport.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.CreateReport: %v", err)
		return model.Report{}, repository.ErrFailedToInsert
	}

	if rpt := model.NewReportFromDB(dbReport); rpt != nil {
		return *rpt, nil
	}
	return model.Report{}, nil
}

func (r *implRepository) GetReportByID(ctx context.Context, id string) (model.Report, error) {
	dbReport, err := sqlboiler.FindReport(ctx, r.db, id)
	if err == sql.ErrNoRows {
		return model.Report{}, repository.ErrNotFound
	}
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.GetReportByID: %v", err)
		return model.Report{}, repository.ErrFailedToGet
	}

	if rpt := model.NewReportFromDB(dbReport); rpt != nil {
		return *rpt, nil
	}
	return model.Report{}, nil
}

func (r *implRepository) FindByParamsHash(ctx context.Context, opt repository.FindByParamsHashOptions) (model.Report, error) {
	mods := r.buildFindByParamsHashQuery(opt)

	dbReport, err := sqlboiler.Reports(mods...).One(ctx, r.db)
	if err == sql.ErrNoRows {
		return model.Report{}, repository.ErrNotFound
	}
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.FindByParamsHash: %v", err)
		return model.Report{}, repository.ErrFailedToGet
	}

	if rpt := model.NewReportFromDB(dbReport); rpt != nil {
		return *rpt, nil
	}
	return model.Report{}, nil
}

func (r *implRepository) UpdateCompleted(ctx context.Context, opt repository.UpdateCompletedOptions) error {
	dbReport, err := sqlboiler.FindReport(ctx, r.db, opt.ReportID)
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateCompleted: %v", err)
		return repository.ErrFailedToUpdate
	}

	now := time.Now()
	dbReport.Status = "COMPLETED"
	dbReport.FileURL = null.StringFrom(opt.FileURL)
	dbReport.FileSizeBytes = null.Int64From(opt.FileSizeBytes)
	dbReport.TotalDocsAnalyzed = null.IntFrom(opt.TotalDocsAnalyzed)
	dbReport.SectionsCount = null.IntFrom(opt.SectionsCount)
	dbReport.GenerationTimeMS = null.Int64From(opt.GenerationTimeMs)
	dbReport.CompletedAt = null.TimeFrom(now)
	dbReport.UpdatedAt = null.TimeFrom(now)

	_, err = dbReport.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateCompleted: %v", err)
		return repository.ErrFailedToUpdate
	}
	return nil
}

func (r *implRepository) UpdateFailed(ctx context.Context, opt repository.UpdateFailedOptions) error {
	dbReport, err := sqlboiler.FindReport(ctx, r.db, opt.ReportID)
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateFailed: %v", err)
		return repository.ErrFailedToUpdate
	}

	dbReport.Status = "FAILED"
	dbReport.ErrorMessage = null.StringFrom(opt.ErrorMessage)
	dbReport.UpdatedAt = null.TimeFrom(time.Now())

	_, err = dbReport.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateFailed: %v", err)
		return repository.ErrFailedToUpdate
	}
	return nil
}

func (r *implRepository) ListReports(ctx context.Context, opt repository.ListReportsOptions) ([]model.Report, error) {
	mods := r.buildListReportsQuery(opt)

	dbReports, err := sqlboiler.Reports(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.ListReports: %v", err)
		return nil, repository.ErrFailedToList
	}

	return util.MapSlice(dbReports, model.NewReportFromDB), nil
}
```

### E. QUERY BUILDER (`repository/postgre/report_query.go`)

```go
package postgre

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	"knowledge-srv/internal/report/repository"
)

func (r *implRepository) buildFindByParamsHashQuery(opt repository.FindByParamsHashOptions) []qm.QueryMod {
	mods := []qm.QueryMod{
		qm.Where("params_hash = ?", opt.ParamsHash),
	}
	if opt.Status != "" {
		mods = append(mods, qm.Where("status = ?", opt.Status))
	}
	mods = append(mods, qm.OrderBy("created_at DESC"))
	mods = append(mods, qm.Limit(1))
	return mods
}

func (r *implRepository) buildListReportsQuery(opt repository.ListReportsOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	if opt.CampaignID != "" {
		mods = append(mods, qm.Where("campaign_id = ?", opt.CampaignID))
	}
	if opt.UserID != "" {
		mods = append(mods, qm.Where("user_id = ?", opt.UserID))
	}
	if opt.Status != "" {
		mods = append(mods, qm.Where("status = ?", opt.Status))
	}

	mods = append(mods, qm.OrderBy("created_at DESC"))

	if opt.Limit > 0 {
		mods = append(mods, qm.Limit(opt.Limit))
	}
	if opt.Offset > 0 {
		mods = append(mods, qm.Offset(opt.Offset))
	}

	return mods
}
```

### F. BUILD HELPER (`repository/postgre/report_build.go`)

```go
package postgre

import (
	"github.com/aarondl/null/v8"

	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/sqlboiler"
)

func buildCreateReport(opt repository.CreateReportOptions) *sqlboiler.Report {
	dbReport := &sqlboiler.Report{
		CampaignID: opt.CampaignID,
		UserID:     opt.UserID,
		Title:      null.StringFrom(opt.Title),
		ReportType: opt.ReportType,
		ParamsHash: opt.ParamsHash,
		Status:     "PROCESSING",
		FileFormat: null.StringFrom("pdf"),
	}

	if len(opt.Filters) > 0 && string(opt.Filters) != "null" {
		dbReport.Filters = null.JSONFrom(opt.Filters)
	}

	return dbReport
}
```

---

## VI. MODEL ENTITY

### `internal/model/report.go`

```go
package model

import (
	"encoding/json"
	"time"

	"github.com/aarondl/null/v8"
	"knowledge-srv/internal/sqlboiler"
)

// Report represents a generated report record.
type Report struct {
	ID         string
	CampaignID string
	UserID     string

	// Report Configuration
	Title      string
	ReportType string // SUMMARY | COMPARISON | TREND | ASPECT_DEEP_DIVE
	ParamsHash string
	Filters    json.RawMessage

	// Status
	Status       string // PROCESSING | COMPLETED | FAILED
	ErrorMessage string

	// Output
	FileURL       string
	FileSizeBytes int64
	FileFormat    string

	// Metrics
	TotalDocsAnalyzed int
	SectionsCount     int
	GenerationTimeMs  int64

	// Timestamps
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewReportFromDB converts a SQLBoiler Report to model Report
func NewReportFromDB(db *sqlboiler.Report) *Report {
	if db == nil {
		return nil
	}

	rpt := &Report{
		ID:         db.ID,
		CampaignID: db.CampaignID,
		UserID:     db.UserID,
		ReportType: db.ReportType,
		ParamsHash: db.ParamsHash,
		Status:     db.Status,
	}

	// Handle nullable string fields
	if db.Title.Valid {
		rpt.Title = db.Title.String
	}
	if db.ErrorMessage.Valid {
		rpt.ErrorMessage = db.ErrorMessage.String
	}
	if db.FileURL.Valid {
		rpt.FileURL = db.FileURL.String
	}
	if db.FileFormat.Valid {
		rpt.FileFormat = db.FileFormat.String
	}

	// Handle nullable JSON
	if db.Filters.Valid {
		rpt.Filters = json.RawMessage(db.Filters.JSON)
	}

	// Handle nullable numeric fields
	if db.FileSizeBytes.Valid {
		rpt.FileSizeBytes = db.FileSizeBytes.Int64
	}
	if db.TotalDocsAnalyzed.Valid {
		rpt.TotalDocsAnalyzed = db.TotalDocsAnalyzed.Int
	}
	if db.SectionsCount.Valid {
		rpt.SectionsCount = db.SectionsCount.Int
	}
	if db.GenerationTimeMS.Valid {
		rpt.GenerationTimeMs = db.GenerationTimeMS.Int64
	}

	// Handle nullable time fields
	if db.CompletedAt.Valid {
		rpt.CompletedAt = &db.CompletedAt.Time
	}
	if db.CreatedAt.Valid {
		rpt.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		rpt.UpdatedAt = db.UpdatedAt.Time
	}

	return rpt
}

// ToDBReport converts model Report to SQLBoiler Report
func (r *Report) ToDBReport() *sqlboiler.Report {
	db := &sqlboiler.Report{
		ID:         r.ID,
		CampaignID: r.CampaignID,
		UserID:     r.UserID,
		ReportType: r.ReportType,
		ParamsHash: r.ParamsHash,
		Status:     r.Status,
	}

	if r.Title != "" {
		db.Title = null.StringFrom(r.Title)
	}
	if r.ErrorMessage != "" {
		db.ErrorMessage = null.StringFrom(r.ErrorMessage)
	}
	if r.FileURL != "" {
		db.FileURL = null.StringFrom(r.FileURL)
	}
	if r.FileFormat != "" {
		db.FileFormat = null.StringFrom(r.FileFormat)
	}
	if len(r.Filters) > 0 && string(r.Filters) != "null" {
		db.Filters = null.JSONFrom(r.Filters)
	}
	if r.FileSizeBytes > 0 {
		db.FileSizeBytes = null.Int64From(r.FileSizeBytes)
	}
	if r.TotalDocsAnalyzed > 0 {
		db.TotalDocsAnalyzed = null.IntFrom(r.TotalDocsAnalyzed)
	}
	if r.SectionsCount > 0 {
		db.SectionsCount = null.IntFrom(r.SectionsCount)
	}
	if r.GenerationTimeMs > 0 {
		db.GenerationTimeMS = null.Int64From(r.GenerationTimeMs)
	}
	if r.CompletedAt != nil {
		db.CompletedAt = null.TimeFrom(*r.CompletedAt)
	}
	db.CreatedAt = null.TimeFrom(r.CreatedAt)
	db.UpdatedAt = null.TimeFrom(r.UpdatedAt)

	return db
}
```

---

## VII. USECASE IMPLEMENTATION

### A. FACTORY (`usecase/new.go`)

```go
package usecase

import (
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/gemini"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
)

type Config struct {
	MaxSampleDocsPerAspect int    // default 3
	MaxSections            int    // default 6
	MaxDocContentLen       int    // default 500 chars
	ReportBucketPath       string // default "smap-reports"
	DownloadURLExpiryMins  int    // default 60
	DedupWindowHours       int    // default 1 (return existing report if < 1h old)
}

type implUseCase struct {
	repo     repository.PostgresRepository
	searchUC search.UseCase
	gemini   gemini.IGemini
	minio    minio.IMinIO
	l        log.Logger
	cfg      Config
}

func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	gemini gemini.IGemini,
	minio minio.IMinIO,
	l log.Logger,
	cfg Config,
) report.UseCase {
	if cfg.MaxSampleDocsPerAspect <= 0 { cfg.MaxSampleDocsPerAspect = 3 }
	if cfg.MaxSections <= 0 { cfg.MaxSections = 6 }
	if cfg.MaxDocContentLen <= 0 { cfg.MaxDocContentLen = 500 }
	if cfg.ReportBucketPath == "" { cfg.ReportBucketPath = "smap-reports" }
	if cfg.DownloadURLExpiryMins <= 0 { cfg.DownloadURLExpiryMins = 60 }
	if cfg.DedupWindowHours <= 0 { cfg.DedupWindowHours = 1 }

	return &implUseCase{
		repo: repo, searchUC: searchUC, gemini: gemini,
		minio: minio, l: l, cfg: cfg,
	}
}
```

### B. REPORT LOGIC (`usecase/report.go`)

Core async flow — dedup by `params_hash`, background goroutine.

```go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
)

func (uc *implUseCase) Generate(ctx context.Context, sc model.Scope, input report.GenerateInput) (report.GenerateOutput, error) {
	// Step 0: Validate
	if err := uc.validateGenerateInput(input); err != nil {
		return report.GenerateOutput{}, err
	}

	// Step 1: Generate params hash for dedup
	paramsHash := uc.generateParamsHash(input)

	// Step 2: Check existing PROCESSING report
	existing, err := uc.repo.FindByParamsHash(ctx, repository.FindByParamsHashOptions{
		ParamsHash: paramsHash, Status: "PROCESSING",
	})
	if err == nil && existing.ID != "" {
		return report.GenerateOutput{
			ReportID: existing.ID,
			Status:   "PROCESSING",
			Message:  "Báo cáo đang được tạo, vui lòng đợi...",
		}, nil
	}

	// Step 3: Check recent COMPLETED report (< DedupWindowHours)
	recent, err := uc.repo.FindByParamsHash(ctx, repository.FindByParamsHashOptions{
		ParamsHash: paramsHash, Status: "COMPLETED",
	})
	if err == nil && recent.ID != "" && recent.CompletedAt != nil {
		if time.Since(*recent.CompletedAt) < time.Duration(uc.cfg.DedupWindowHours)*time.Hour {
			return report.GenerateOutput{
				ReportID: recent.ID,
				Status:   "COMPLETED",
				Message:  "Báo cáo đã có sẵn.",
			}, nil
		}
	}

	// Step 4: Generate title if empty
	title := input.Title
	if title == "" {
		title = uc.generateTitle(input)
	}

	// Step 5: Create report record
	filtersJSON, _ := json.Marshal(input.Filters)
	rpt, err := uc.repo.CreateReport(ctx, repository.CreateReportOptions{
		CampaignID: input.CampaignID,
		UserID:     sc.UserID,
		Title:      title,
		ReportType: input.ReportType,
		ParamsHash: paramsHash,
		Filters:    filtersJSON,
	})
	if err != nil {
		return report.GenerateOutput{}, fmt.Errorf("create report: %w", err)
	}

	// Step 6: Launch background generation
	go uc.generateInBackground(context.Background(), sc, rpt, input)

	return report.GenerateOutput{
		ReportID: rpt.ID,
		Status:   "PROCESSING",
		Message:  "Báo cáo đang được tạo...",
	}, nil
}

func (uc *implUseCase) GetReport(ctx context.Context, sc model.Scope, input report.GetReportInput) (report.ReportOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		return report.ReportOutput{}, report.ErrReportNotFound
	}

	return uc.toReportOutput(rpt), nil
}

func (uc *implUseCase) DownloadReport(ctx context.Context, sc model.Scope, input report.DownloadReportInput) (report.DownloadOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		return report.DownloadOutput{}, report.ErrReportNotFound
	}
	if rpt.Status != "COMPLETED" {
		return report.DownloadOutput{}, report.ErrReportNotCompleted
	}

	// Generate presigned URL from MinIO
	expiryDuration := time.Duration(uc.cfg.DownloadURLExpiryMins) * time.Minute
	downloadURL, err := uc.minio.PresignedGetObject(ctx, rpt.FileURL, expiryDuration)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.DownloadReport: presigned URL failed: %v", err)
		return report.DownloadOutput{}, report.ErrDownloadURLFailed
	}

	return report.DownloadOutput{
		DownloadURL: downloadURL,
		FileName:    fmt.Sprintf("report_%s.%s", rpt.ID, rpt.FileFormat),
		FileSize:    rpt.FileSizeBytes,
		ExpiresAt:   time.Now().Add(expiryDuration),
	}, nil
}

func (uc *implUseCase) validateGenerateInput(input report.GenerateInput) error {
	if input.CampaignID == "" {
		return report.ErrCampaignNotFound
	}
	validTypes := map[string]bool{
		"SUMMARY": true, "COMPARISON": true, "TREND": true, "ASPECT_DEEP_DIVE": true,
	}
	if !validTypes[input.ReportType] {
		return report.ErrInvalidReportType
	}
	return nil
}

func (uc *implUseCase) generateParamsHash(input report.GenerateInput) string {
	filtersJSON, _ := json.Marshal(input.Filters)
	data := fmt.Sprintf("%s:%s:%s", input.CampaignID, input.ReportType, string(filtersJSON))
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func (uc *implUseCase) generateTitle(input report.GenerateInput) string {
	typeNames := map[string]string{
		"SUMMARY":          "Báo cáo tổng hợp",
		"COMPARISON":       "Báo cáo so sánh",
		"TREND":            "Báo cáo xu hướng",
		"ASPECT_DEEP_DIVE": "Phân tích chi tiết khía cạnh",
	}
	return typeNames[input.ReportType]
}

func (uc *implUseCase) toReportOutput(rpt model.Report) report.ReportOutput {
	return report.ReportOutput{
		ID:                rpt.ID,
		CampaignID:        rpt.CampaignID,
		UserID:            rpt.UserID,
		Title:             rpt.Title,
		ReportType:        rpt.ReportType,
		Status:            rpt.Status,
		ErrorMessage:      rpt.ErrorMessage,
		FileFormat:        rpt.FileFormat,
		FileSizeBytes:     rpt.FileSizeBytes,
		TotalDocsAnalyzed: rpt.TotalDocsAnalyzed,
		SectionsCount:     rpt.SectionsCount,
		GenerationTimeMs:  rpt.GenerationTimeMs,
		CompletedAt:       rpt.CompletedAt,
		CreatedAt:         rpt.CreatedAt,
	}
}
```

### C. BACKGROUND GENERATOR (`usecase/generator.go`)

**Map-Reduce pattern:** Aggregate → Sample → Generate → Compile

```go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/search"
)

func (uc *implUseCase) generateInBackground(ctx context.Context, sc model.Scope, rpt model.Report, input report.GenerateInput) {
	startTime := time.Now()

	defer func() {
		if r := recover(); r != nil {
			uc.l.Errorf(ctx, "report.usecase.generateInBackground: panic: %v", r)
			uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
				ReportID: rpt.ID, ErrorMessage: fmt.Sprintf("panic: %v", r),
			})
		}
	}()

	// Phase 1: AGGREGATE — Get statistics from Qdrant via search.UseCase
	searchInput := search.SearchInput{
		CampaignID: input.CampaignID,
		Query:      uc.getAggregateQuery(input.ReportType),
		Limit:      50,
		Filters: search.SearchFilters{
			Sentiments: input.Filters.Sentiments,
			Aspects:    input.Filters.Aspects,
			Platforms:  input.Filters.Platforms,
			DateFrom:   input.Filters.DateFrom,
			DateTo:     input.Filters.DateTo,
		},
	}
	searchOutput, err := uc.searchUC.Search(ctx, sc, searchInput)
	if err != nil {
		uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID: rpt.ID, ErrorMessage: fmt.Sprintf("search failed: %v", err),
		})
		return
	}

	// Phase 2: SAMPLE — Get representative documents per aspect
	samples := map[string][]search.SearchResult{}
	for _, aspect := range uc.getTargetAspects(searchOutput, input) {
		posResults, _ := uc.searchUC.Search(ctx, sc, search.SearchInput{
			CampaignID: input.CampaignID,
			Query:      aspect,
			Limit:      uc.cfg.MaxSampleDocsPerAspect,
			Filters: search.SearchFilters{
				Aspects:    []string{aspect},
				Sentiments: []string{"POSITIVE"},
			},
		})
		negResults, _ := uc.searchUC.Search(ctx, sc, search.SearchInput{
			CampaignID: input.CampaignID,
			Query:      aspect,
			Limit:      uc.cfg.MaxSampleDocsPerAspect,
			Filters: search.SearchFilters{
				Aspects:    []string{aspect},
				Sentiments: []string{"NEGATIVE"},
			},
		})
		samples[aspect] = append(posResults.Results, negResults.Results...)
	}

	// Phase 3: GENERATE — LLM generates each section
	templates := getTemplates(input.ReportType)
	sections := make([]report.ReportSection, 0, len(templates))
	for _, tmpl := range templates {
		prompt := uc.buildSectionPrompt(tmpl, searchOutput, samples)
		content, err := uc.gemini.Generate(ctx, prompt)
		if err != nil {
			uc.l.Errorf(ctx, "report.usecase.generateInBackground: LLM failed for section %s: %v", tmpl.Title, err)
			content = fmt.Sprintf("*Section generation failed: %v*", err)
		}
		sections = append(sections, report.ReportSection{Title: tmpl.Title, Content: content})
	}

	// Phase 4: COMPILE — Assemble markdown → upload to MinIO
	markdown := uc.compileReport(rpt.Title, sections, searchOutput)

	objectPath := fmt.Sprintf("%s/%s.md", uc.cfg.ReportBucketPath, rpt.ID)
	fileSize, err := uc.minio.Upload(ctx, objectPath, []byte(markdown))
	if err != nil {
		uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID: rpt.ID, ErrorMessage: fmt.Sprintf("upload failed: %v", err),
		})
		return
	}

	// Update status → COMPLETED
	uc.repo.UpdateCompleted(ctx, repository.UpdateCompletedOptions{
		ReportID:          rpt.ID,
		FileURL:           objectPath,
		FileSizeBytes:     fileSize,
		TotalDocsAnalyzed: searchOutput.TotalFound,
		SectionsCount:     len(sections),
		GenerationTimeMs:  time.Since(startTime).Milliseconds(),
	})
}

func (uc *implUseCase) getAggregateQuery(reportType string) string {
	switch reportType {
	case "SUMMARY":
		return "tổng quan đánh giá phân tích"
	case "COMPARISON":
		return "so sánh phân tích"
	case "TREND":
		return "xu hướng thay đổi theo thời gian"
	case "ASPECT_DEEP_DIVE":
		return "phân tích chi tiết từng khía cạnh"
	default:
		return "phân tích tổng quan"
	}
}

func (uc *implUseCase) getTargetAspects(output search.SearchOutput, input report.GenerateInput) []string {
	if len(input.Filters.Aspects) > 0 {
		return input.Filters.Aspects
	}
	// Extract from aggregations
	aspects := make([]string, 0)
	for _, a := range output.Aggregations.ByAspect {
		aspects = append(aspects, a.AspectDisplayName)
	}
	if len(aspects) > 5 {
		aspects = aspects[:5]
	}
	return aspects
}

func (uc *implUseCase) compileReport(title string, sections []report.ReportSection, output search.SearchOutput) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", title))
	b.WriteString(fmt.Sprintf("*Generated at: %s*\n\n", time.Now().Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("**Total documents analyzed: %d**\n\n---\n\n", output.TotalFound))

	for _, section := range sections {
		b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n---\n\n", section.Title, section.Content))
	}

	return b.String()
}
```

### D. TEMPLATES (`usecase/template.go`)

```go
package usecase

import "knowledge-srv/internal/report"

// getTemplates returns section templates for each report type
func getTemplates(reportType string) []report.SectionTemplate {
	switch reportType {
	case "SUMMARY":
		return []report.SectionTemplate{
			{Title: "Tóm tắt điều hành", PromptTmpl: summaryExecutiveTmpl},
			{Title: "Phân tích cảm xúc", PromptTmpl: summarySentimentTmpl},
			{Title: "Phân tích theo khía cạnh", PromptTmpl: summaryAspectTmpl},
			{Title: "Nhận xét và đề xuất", PromptTmpl: summaryRecommendationTmpl},
		}

	case "COMPARISON":
		return []report.SectionTemplate{
			{Title: "Tóm tắt điều hành", PromptTmpl: comparisonExecutiveTmpl},
			{Title: "So sánh cảm xúc", PromptTmpl: comparisonSentimentTmpl},
			{Title: "So sánh theo khía cạnh", PromptTmpl: comparisonAspectTmpl},
			{Title: "Kết luận", PromptTmpl: comparisonConclusionTmpl},
		}

	case "TREND":
		return []report.SectionTemplate{
			{Title: "Tổng quan xu hướng", PromptTmpl: trendOverviewTmpl},
			{Title: "Biến động cảm xúc theo thời gian", PromptTmpl: trendSentimentTmpl},
			{Title: "Biến động theo khía cạnh", PromptTmpl: trendAspectTmpl},
			{Title: "Dự đoán và cảnh báo", PromptTmpl: trendPredictionTmpl},
		}

	case "ASPECT_DEEP_DIVE":
		return []report.SectionTemplate{
			{Title: "Tổng quan khía cạnh", PromptTmpl: deepDiveOverviewTmpl},
			{Title: "Phân tích tích cực", PromptTmpl: deepDivePositiveTmpl},
			{Title: "Phân tích tiêu cực", PromptTmpl: deepDiveNegativeTmpl},
			{Title: "Mẫu đánh giá tiêu biểu", PromptTmpl: deepDiveSampleTmpl},
			{Title: "Đề xuất hành động", PromptTmpl: deepDiveActionTmpl},
		}

	default:
		return []report.SectionTemplate{
			{Title: "Tổng quan", PromptTmpl: summaryExecutiveTmpl},
		}
	}
}

// Template strings — Vietnamese prompts for LLM
const (
	summaryExecutiveTmpl = `Dựa trên dữ liệu phân tích bên dưới, viết phần "Tóm tắt điều hành" cho báo cáo.
Bao gồm: tổng quan sentiment, top aspects nổi bật, điểm đáng chú ý.
Viết ngắn gọn, chuyên nghiệp, dạng bullet points.

Thống kê: {{.Stats}}
Mẫu nội dung: {{.Samples}}`

	summarySentimentTmpl = `Dựa trên dữ liệu, viết phần "Phân tích cảm xúc".
Phân tích phân bố POSITIVE/NEGATIVE/NEUTRAL/MIXED, giải thích nguyên nhân dựa trên mẫu.

Thống kê: {{.Stats}}
Mẫu nội dung: {{.Samples}}`

	summaryAspectTmpl = `Dựa trên dữ liệu, viết phần "Phân tích theo khía cạnh".
Với mỗi aspect: tóm tắt sentiment, trích dẫn ví dụ tiêu biểu.

Thống kê: {{.Stats}}
Mẫu nội dung: {{.Samples}}`

	summaryRecommendationTmpl = `Dựa trên toàn bộ phân tích, viết phần "Nhận xét và đề xuất".
Gợi ý hành động cụ thể dựa trên dữ liệu.

Thống kê: {{.Stats}}
Mẫu nội dung: {{.Samples}}`

	comparisonExecutiveTmpl   = summaryExecutiveTmpl
	comparisonSentimentTmpl   = summarySentimentTmpl
	comparisonAspectTmpl      = summaryAspectTmpl
	comparisonConclusionTmpl  = summaryRecommendationTmpl

	trendOverviewTmpl    = summaryExecutiveTmpl
	trendSentimentTmpl   = summarySentimentTmpl
	trendAspectTmpl      = summaryAspectTmpl
	trendPredictionTmpl  = summaryRecommendationTmpl

	deepDiveOverviewTmpl  = summaryExecutiveTmpl
	deepDivePositiveTmpl  = summarySentimentTmpl
	deepDiveNegativeTmpl  = summarySentimentTmpl
	deepDiveSampleTmpl    = summaryAspectTmpl
	deepDiveActionTmpl    = summaryRecommendationTmpl
)
```

### E. HELPERS (`usecase/helpers.go`)

```go
package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"knowledge-srv/internal/report"
	"knowledge-srv/internal/search"
)

type promptData struct {
	Stats   string
	Samples string
}

func (uc *implUseCase) buildSectionPrompt(
	tmpl report.SectionTemplate,
	output search.SearchOutput,
	samples map[string][]search.SearchResult,
) string {
	statsJSON, _ := json.MarshalIndent(output.Aggregations, "", "  ")
	samplesStr := uc.formatSamples(samples)

	prompt := tmpl.PromptTmpl
	prompt = strings.ReplaceAll(prompt, "{{.Stats}}", string(statsJSON))
	prompt = strings.ReplaceAll(prompt, "{{.Samples}}", samplesStr)

	return prompt
}

func (uc *implUseCase) formatSamples(samples map[string][]search.SearchResult) string {
	var b strings.Builder
	for aspect, docs := range samples {
		b.WriteString(fmt.Sprintf("\n--- %s ---\n", aspect))
		for i, doc := range docs {
			content := doc.Content
			if len(content) > uc.cfg.MaxDocContentLen {
				content = content[:uc.cfg.MaxDocContentLen] + "..."
			}
			b.WriteString(fmt.Sprintf("[%d] \"%s\" (Sentiment: %s, Platform: %s)\n",
				i+1, content, doc.OverallSentiment, doc.Platform))
		}
	}
	return b.String()
}
```

---

## VIII. DELIVERY LAYER

### A. ROUTES (`delivery/http/routes.go`)

```go
package http

import (
	"github.com/gin-gonic/gin"
	"knowledge-srv/internal/report"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/middleware"
)

type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

func New(l log.Logger, uc report.UseCase) Handler {
	return &handler{l: l, uc: uc}
}

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	api := r.Group("/api/v1")
	api.Use(mw.JWTAuth())
	{
		api.POST("/reports/generate", h.GenerateReport)
		api.GET("/reports/:report_id", h.GetReport)
		api.GET("/reports/:report_id/download", h.DownloadReport)
	}
}
```

### B. PRESENTERS (`delivery/http/presenters.go`)

Request/Response DTOs follow the same pattern as domain 2 and 3 plans. Key DTOs:

- `generateReportReq` → `report.GenerateInput` via `toInput()`
- `generateReportResp` ← `report.GenerateOutput` via `newGenerateResp()`
- `reportResp` ← `report.ReportOutput`
- `downloadResp` ← `report.DownloadOutput`

### C. ERRORS (`delivery/http/errors.go`)

Maps domain errors to HTTP errors using `pkg/errors.NewHTTPError`:

| Domain Error            | HTTP Code | Message                           |
| ----------------------- | --------- | --------------------------------- |
| `ErrReportNotFound`     | 404       | "Report not found"                |
| `ErrReportNotCompleted` | 400       | "Report is not yet completed"     |
| `ErrCampaignNotFound`   | 404       | "Campaign not found"              |
| `ErrInvalidReportType`  | 400       | "Invalid report type"             |
| `ErrGenerationFailed`   | 500       | "Report generation failed"        |
| `ErrDownloadURLFailed`  | 500       | "Failed to generate download URL" |

---

## IX. CROSS-DOMAIN CHANGES

> [!WARNING]
> Domain Report phụ thuộc vào một số thay đổi ở domain Search và pkg/minio. Cần implement các thay đổi này **trước** khi triển khai report domain.

### 1. `search.UseCase` — Bổ sung `Aggregate()` (Phase 2 của master proposal)

Hiện tại `search.UseCase` chỉ có `Search()`. Master proposal yêu cầu thêm `Aggregate()`:

```go
// internal/search/interface.go — THAY ĐỔI
type UseCase interface {
	Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)
	Aggregate(ctx context.Context, input AggregateInput) (AggregateOutput, error) // MỚI
}
```

```go
// internal/search/types.go — BỔ SUNG
type AggregateInput struct {
	CampaignID string
	GroupBy    []string     // ["overall_sentiment", "aspects.aspect", "platform"]
	DateRange  *DateRange
}

type AggregateOutput struct {
	TotalDocs   int
	BySentiment map[string]int
	ByAspect    map[string]AspectStats
	ByPlatform  map[string]int
	TopAspects  []string
	TrendByWeek []WeeklyTrend
}
```

> [!NOTE]
> Trong Phase 1, report domain có thể dùng `Search()` với limit lớn (50 results) thay vì `Aggregate()`. `Aggregate()` sẽ được triển khai song song khi optimize performance.

### 2. `pkg/minio` — Bổ sung `PresignedGetObject()`

```go
// pkg/minio/interface.go — BỔ SUNG
type IMinIO interface {
	// Existing methods...
	Upload(ctx context.Context, path string, data []byte) (int64, error)
	Download(ctx context.Context, path string) ([]byte, error)

	// MỚI: Generate presigned download URL
	PresignedGetObject(ctx context.Context, objectPath string, expiry time.Duration) (string, error)
}
```

---

## X. TESTING STRATEGY

### Unit Tests (UseCase)

| File                | Key Test Cases                                                                                |
| ------------------- | --------------------------------------------------------------------------------------------- |
| `report_test.go`    | Generate_New, Generate_DedupProcessing, Generate_DedupCompleted, InvalidReportType, GetReport |
| `generator_test.go` | FullFlow, SearchFailed, LLMFailed, UploadFailed, PanicRecovery                                |
| `helpers_test.go`   | BuildSectionPrompt, FormatSamples, GenerateParamsHash, GenerateTitle                          |

### Integration Tests (Repository)

| File             | Key Test Cases                                                                |
| ---------------- | ----------------------------------------------------------------------------- |
| `report_test.go` | Create, GetByID, FindByParamsHash, UpdateCompleted, UpdateFailed, ListReports |

---

## XI. WIRING (`internal/httpserver/domain_report.go`)

```go
func (s *HTTPServer) initReportDomain() {
	repo := reportRepo.New(s.db, s.logger)
	uc := reportUC.New(repo, s.searchUC, s.gemini, s.minio, s.logger, reportUC.Config{
		MaxSampleDocsPerAspect: 3,
		MaxSections:            6,
		MaxDocContentLen:       500,
		ReportBucketPath:       "smap-reports",
		DownloadURLExpiryMins:  60,
		DedupWindowHours:       1,
	})
	handler := reportHTTP.New(s.logger, uc)
	handler.RegisterRoutes(s.router, s.mw)
}
```
