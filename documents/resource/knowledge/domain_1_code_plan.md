# Domain 1: Indexing -- Chi tiết Plan Code

**Version:** 1.0  
**Last Updated:** 2026-02-16  
**Domain:** `internal/indexing` -- Vector Indexing

---

## I. TỔNG QUAN

### 1. Vai trò Domain

Domain Indexing là **write path** của hệ thống RAG Knowledge Service, chịu trách nhiệm:

- Nhận dữ liệu đã phân tích từ Analytics Service
- Tạo vector embeddings từ content
- Lưu trữ vào Qdrant Vector Database
- Tracking metadata trong PostgreSQL
- Invalidate cache khi cần

### 2. Input/Output

**Input:**

- Analytics data đã phân tích theo schema `analytics.post_analytics` (xem `documents/analytic_post_schema.md`)
- Nhận qua 2 phương thức:
  - **Kafka**: topic `analytics.batch.completed` → message chứa MinIO file URL
  - **HTTP**: `POST /internal/index` → body chứa MinIO file URL

**Output:**

- Vector points trong Qdrant collection `smap_analytics`
- Tracking records trong PostgreSQL `knowledge.indexed_documents`
- Cache invalidation signals

### 3. Kiến trúc tổng quan

```
┌─────────────────────────────────────────────────────────────────┐
│                    Analytics Service                            │
│  ┌──────────────────┐        ┌──────────────────┐              │
│  │ Batch Analyzer   │        │ Realtime Analyzer│              │
│  └────────┬─────────┘        └────────┬─────────┘              │
│           │                            │                         │
│           ▼                            ▼                         │
│   ┌──────────────┐            ┌──────────────┐                 │
│   │ Save to      │            │ Save to      │                 │
│   │ MinIO        │            │ MinIO        │                 │
│   └──────┬───────┘            └──────┬───────┘                 │
└──────────┼────────────────────────────┼─────────────────────────┘
           │                            │
           │ Publish Kafka              │ HTTP Call
           ▼                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Knowledge Service (Domain: Indexing)          │
│                                                                  │
│  ┌──────────────────┐        ┌──────────────────┐              │
│  │ Kafka Consumer   │        │ HTTP Handler     │              │
│  │ (delivery/kafka) │        │ (delivery/http)  │              │
│  └────────┬─────────┘        └────────┬─────────┘              │
│           │                            │                         │
│           └──────────┬─────────────────┘                        │
│                      ▼                                           │
│           ┌──────────────────┐                                  │
│           │   UseCase        │                                  │
│           │   Index()          │                                  │
│           └────────┬─────────┘                                  │
│                    │                                             │
│         ┌──────────┼──────────┐                                 │
│         ▼          ▼           ▼                                │
│    ┌────────┐ ┌────────┐ ┌─────────┐                           │
│    │ MinIO  │ │Voyage/ │ │ Qdrant  │                           │
│    │Download│ │OpenAI  │ │ Upsert  │                           │
│    └────────┘ └────────┘ └─────────┘                           │
│                    │                                             │
│                    ▼                                             │
│           ┌──────────────────┐                                  │
│           │   PostgreSQL     │                                  │
│           │  indexed_docs    │                                  │
│           └──────────────────┘                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## II. DATABASE SCHEMA

### 1. PostgreSQL Schema: `knowledge.indexed_documents`

```sql
-- =====================================================
-- Table: indexed_documents
-- Purpose: Tracking metadata của các documents đã index
-- =====================================================
CREATE TABLE knowledge.indexed_documents (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analytics_id    UUID NOT NULL,              -- FK → analytics.post_analytics.id
    project_id      UUID NOT NULL,              -- Thuộc project nào
    source_id       UUID NOT NULL,              -- Từ data source nào

    -- Qdrant Reference
    qdrant_point_id UUID NOT NULL,              -- ID của point trong Qdrant
    collection_name VARCHAR(100) NOT NULL,      -- Collection name (e.g., "smap_analytics")

    -- Content Hash (for deduplication)
    content_hash    VARCHAR(64) NOT NULL,       -- SHA-256 hash của content

    -- Indexing Status
    status          VARCHAR(20) NOT NULL        -- PENDING | INDEXED | FAILED | RE_INDEXING
                    DEFAULT 'PENDING',
    error_message   TEXT,                       -- Lỗi nếu status = FAILED
    retry_count     INT DEFAULT 0,              -- Số lần retry

    -- Batch Tracking
    batch_id        VARCHAR(100),               -- Thuộc batch nào (nullable)
    ingestion_method VARCHAR(20) NOT NULL,      -- 'kafka' | 'api'

    -- Performance Metrics
    embedding_time_ms   INT,                    -- Thời gian sinh embedding (ms)
    upsert_time_ms      INT,                    -- Thời gian upsert Qdrant (ms)
    total_time_ms       INT,                    -- Tổng thời gian xử lý (ms)

    -- Timestamps
    indexed_at      TIMESTAMPTZ,                -- Thời điểm index thành công
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- Indexes
-- =====================================================
CREATE UNIQUE INDEX idx_indexed_docs_analytics_id
    ON knowledge.indexed_documents(analytics_id);

CREATE INDEX idx_indexed_docs_project
    ON knowledge.indexed_documents(project_id);

CREATE INDEX idx_indexed_docs_batch
    ON knowledge.indexed_documents(batch_id);

CREATE INDEX idx_indexed_docs_status
    ON knowledge.indexed_documents(status);

CREATE INDEX idx_indexed_docs_content_hash
    ON knowledge.indexed_documents(content_hash);

CREATE INDEX idx_indexed_docs_created
    ON knowledge.indexed_documents(created_at DESC);
```

### 2. Dead Letter Queue (DLQ) Schema

```sql
-- =====================================================
-- Table: indexing_dlq
-- Purpose: Lưu các records lỗi sau khi retry hết
-- =====================================================
CREATE TABLE knowledge.indexing_dlq (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analytics_id    UUID NOT NULL,
    batch_id        VARCHAR(100),
    raw_payload     JSONB NOT NULL,             -- Lưu nguyên record gốc để debug
    error_message   TEXT NOT NULL,
    error_type      VARCHAR(50) NOT NULL,       -- PARSE_ERROR | EMBEDDING_ERROR | QDRANT_ERROR
    retry_count     INT DEFAULT 0,
    max_retries     INT DEFAULT 3,
    resolved        BOOLEAN DEFAULT false,      -- Admin đã xử lý chưa
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- Indexes
-- =====================================================
CREATE INDEX idx_indexing_dlq_analytics
    ON knowledge.indexing_dlq(analytics_id);

CREATE INDEX idx_indexing_dlq_batch
    ON knowledge.indexing_dlq(batch_id);

CREATE INDEX idx_indexing_dlq_resolved
    ON knowledge.indexing_dlq(resolved) WHERE resolved = false;

CREATE INDEX idx_indexing_dlq_error_type
    ON knowledge.indexing_dlq(error_type);
```

---

## III. QDRANT SCHEMA

### 1. Collection Configuration

```json
{
  "collection_name": "smap_analytics",
  "vectors": {
    "dense": {
      "size": 1536,
      "distance": "Cosine"
    }
  },
  "optimizers_config": {
    "indexing_threshold": 20000
  },
  "hnsw_config": {
    "m": 16,
    "ef_construct": 100
  }
}
```

### 2. Point Structure

```json
{
  "id": "analytics_550e8400-e29b-41d4-a716-446655440000",
  "vector": [0.123, -0.456, ...],  // 1536 dimensions từ Voyage AI
  "payload": {
    // Core Identity
    "project_id": "proj_vinfast_vf8_monitor",
    "source_id": "src_tiktok_crawl_batch_001",
    "analytics_id": "analytics_550e8400-...",

    // UAP Core
    "content": "Mình vừa lái thử VF8, xe đẹp nhưng pin yếu quá...",
    "content_created_at": 1707577800,  // Unix timestamp
    "ingested_at": 1708187123,
    "platform": "tiktok",

    // Sentiment
    "overall_sentiment": "MIXED",
    "overall_sentiment_score": 0.15,
    "sentiment_confidence": 0.87,

    // Aspects (ABSA)
    "aspects": [
      {
        "aspect": "DESIGN",
        "aspect_display_name": "Thiết kế",
        "sentiment": "POSITIVE",
        "sentiment_score": 0.85,
        "keywords": ["đẹp", "sang trọng"],
        "impact_score": 0.78
      },
      {
        "aspect": "BATTERY",
        "aspect_display_name": "Pin",
        "sentiment": "NEGATIVE",
        "sentiment_score": -0.72,
        "keywords": ["pin yếu", "sụt nhanh"],
        "impact_score": 0.85
      }
    ],

    // Keywords
    "keywords": ["VF8", "thiết kế", "pin", "lái thử"],

    // Risk
    "risk_level": "MEDIUM",
    "risk_score": 0.58,
    "requires_attention": true,

    // Engagement
    "engagement_score": 0.73,
    "virality_score": 0.65,
    "influence_score": 0.58,
    "reach_estimate": 45000,

    // Quality
    "content_quality_score": 0.82,
    "is_spam": false,
    "is_bot": false,
    "language": "vi",
    "toxicity_score": 0.05,

    // Metadata
    "metadata": {
      "author": "nguyen_van_a_2024",
      "author_display_name": "Nguyễn Văn A",
      "author_followers": 15000,
      "engagement": {
        "views": 45000,
        "likes": 3200,
        "comments": 156,
        "shares": 89
      },
      "video_url": "https://tiktok.com/@.../video/...",
      "hashtags": ["#VinFast", "#VF8", "#XeDien"],
      "location": "Hà Nội, Việt Nam"
    }
  }
}
```

---

## IV. CODEBASE STRUCTURE

```
internal/indexing/
├── delivery/
│   ├── http/
│   │   ├── new.go                    # Factory: New(l, uc, discord) Handler
│   │   ├── handlers.go               # Index, RetryFailed, Reconcile, GetStatistics handlers
│   │   ├── process_request.go        # processIndexReq, processRetryFailedReq, processReconcileReq
│   │   ├── presenters.go             # IndexReq/Resp, RetryFailedReq/Resp, ReconcileReq/Resp, StatisticsResp
│   │   ├── routes.go                 # RegisterRoutes
│   │   └── errors.go                 # mapError (errors.Is based)
│   └── kafka/
│       ├── type.go                   # BatchCompletedMessage, topic/group constants
│       └── consumer/
│           ├── new.go                # Factory: New(cfg) Consumer
│           ├── handler.go            # sarama.ConsumerGroupHandler impl
│           ├── consumer.go           # ConsumeBatchCompleted
│           ├── workers.go            # handleBatchCompletedMessage
│           ├── presenters.go         # toIndexInput mapper
│           └── error.go              # Consumer-specific errors
├── repository/
│   ├── interface.go                  # PostgresRepository (composed), QdrantRepository
│   ├── option.go                     # Options structs (Create/Get/List/Upsert/Update)
│   ├── errors.go                     # Repository errors
│   ├── postgre/
│   │   ├── new.go                    # Factory
│   │   ├── document.go               # Document CRUD (uses build functions)
│   │   ├── document_query.go         # Document query builders
│   │   ├── document_build.go         # buildCreateDocument, buildUpsertDocument
│   │   ├── dlq.go                    # DLQ CRUD
│   │   └── dlq_query.go              # DLQ query builders
│   └── qdrant/
│       ├── new.go                    # Factory
│       └── point.go                  # UpsertPoint
├── usecase/
│   ├── new.go                        # Factory: impl struct + New + Config
│   ├── index.go                      # Index, indexSingleRecord, batch, validation, embedding, qdrant, cache, DLQ
│   ├── retry_failed.go               # RetryFailed logic
│   ├── reconcile.go                  # Reconcile (stale pending → failed)
│   └── get_statistics.go             # GetStatistics
├── interface.go                      # UseCase interface
├── types.go                          # Input/Output structs, constants
└── errors.go                         # Domain errors
```

> **Note:** ID types sử dụng `string` thay vì `uuid.UUID`. UUID validation được thực hiện ở tầng Delivery (binding tag `uuid`), các tầng bên trong làm việc trên string.

---

## V. CHI TIẾT IMPLEMENTATION

### A. TYPES (`types.go`)

```go
package indexing

import (
	"time"
	"github.com/google/uuid"
)

// =====================================================
// Input Types
// =====================================================

// IndexFromFileInput - Input cho method IndexFromFile
type IndexFromFileInput struct {
	BatchID      string    // ID của batch (tracking)
	ProjectID    uuid.UUID // Project ID (scope)
	FileURL      string    // MinIO file URL (s3://bucket/path/file.jsonl)
	RecordCount  int       // Số records trong file (optional, for progress)
	IngestionMethod string // "kafka" hoặc "api"
}

// RetryFailedInput - Input cho retry failed records
type RetryFailedInput struct {
	MaxRetryCount int       // Chỉ retry records có retry_count < max
	Limit         int       // Số records retry mỗi lần
	ErrorTypes    []string  // Filter by error types
}

// ReconcileInput - Input cho reconcile job
type ReconcileInput struct {
	StaleDuration time.Duration // Records PENDING quá lâu (e.g., 10m)
	Limit         int           // Số records check mỗi lần
}

// =====================================================
// Output Types
// =====================================================

// IndexFromFileOutput - Kết quả index batch
type IndexFromFileOutput struct {
	BatchID       string
	TotalRecords  int
	Indexed       int
	Failed        int
	Skipped       int // Spam, bot, duplicate
	Duration      time.Duration
	FailedRecords []FailedRecord // Chi tiết records lỗi
}

// FailedRecord - Chi tiết 1 record lỗi
type FailedRecord struct {
	AnalyticsID  uuid.UUID
	ErrorType    string // PARSE_ERROR, EMBEDDING_ERROR, QDRANT_ERROR
	ErrorMessage string
}

// RetryFailedOutput - Kết quả retry
type RetryFailedOutput struct {
	TotalRetried int
	Succeeded    int
	Failed       int
	Duration     time.Duration
}

// ReconcileOutput - Kết quả reconcile
type ReconcileOutput struct {
	TotalChecked int
	Fixed        int // Updated to INDEXED
	Requeued     int // Pushed to retry queue
	Duration     time.Duration
}

// =====================================================
// Analytics Post (Input từ Analytics Service)
// =====================================================

// AnalyticsPost - Cấu trúc của 1 record trong file JSONL
// Mapping chính xác theo schema analytics.post_analytics
type AnalyticsPost struct {
	// Core Identity
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	SourceID  uuid.UUID `json:"source_id"`

	// UAP Core
	Content          string    `json:"content"`
	ContentCreatedAt time.Time `json:"content_created_at"`
	IngestedAt       time.Time `json:"ingested_at"`
	Platform         string    `json:"platform"`
	UAPMetadata      UAPMetadata `json:"uap_metadata"`

	// Sentiment
	OverallSentiment      string  `json:"overall_sentiment"`
	OverallSentimentScore float64 `json:"overall_sentiment_score"`
	SentimentConfidence   float64 `json:"sentiment_confidence"`
	SentimentExplanation  string  `json:"sentiment_explanation"`

	// ABSA
	Aspects []Aspect `json:"aspects"`

	// Keywords
	Keywords []string `json:"keywords"`

	// Risk
	RiskLevel          string       `json:"risk_level"`
	RiskScore          float64      `json:"risk_score"`
	RiskFactors        []RiskFactor `json:"risk_factors"`
	RequiresAttention  bool         `json:"requires_attention"`
	AlertTriggered     bool         `json:"alert_triggered"`

	// Engagement
	EngagementScore float64 `json:"engagement_score"`
	ViralityScore   float64 `json:"virality_score"`
	InfluenceScore  float64 `json:"influence_score"`
	ReachEstimate   int     `json:"reach_estimate"`

	// Quality
	ContentQualityScore float64 `json:"content_quality_score"`
	IsSpam              bool    `json:"is_spam"`
	IsBot               bool    `json:"is_bot"`
	Language            string  `json:"language"`
	LanguageConfidence  float64 `json:"language_confidence"`
	ToxicityScore       float64 `json:"toxicity_score"`
	IsToxic             bool    `json:"is_toxic"`
}

// UAPMetadata - Metadata từ UAP
type UAPMetadata struct {
	Author            string              `json:"author"`
	AuthorDisplayName string              `json:"author_display_name"`
	AuthorFollowers   int                 `json:"author_followers"`
	Engagement        EngagementMetadata  `json:"engagement"`
	VideoURL          string              `json:"video_url,omitempty"`
	Hashtags          []string            `json:"hashtags,omitempty"`
	Location          string              `json:"location,omitempty"`
}

// EngagementMetadata - Engagement trong metadata
type EngagementMetadata struct {
	Views    int `json:"views"`
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
}

// Aspect - ABSA aspect
type Aspect struct {
	Aspect            string    `json:"aspect"`
	AspectDisplayName string    `json:"aspect_display_name"`
	Sentiment         string    `json:"sentiment"`
	SentimentScore    float64   `json:"sentiment_score"`
	Confidence        float64   `json:"confidence"`
	Keywords          []string  `json:"keywords"`
	Mentions          []Mention `json:"mentions"`
	ImpactScore       float64   `json:"impact_score"`
	Explanation       string    `json:"explanation"`
}

// Mention - Text mention trong aspect
type Mention struct {
	Text     string `json:"text"`
	StartPos int    `json:"start_pos"`
	EndPos   int    `json:"end_pos"`
}

// RiskFactor - Risk factor
type RiskFactor struct {
	Factor      string `json:"factor"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}
```

### B. INTERFACE (`interface.go`)

```go
package indexing

import (
	"context"
	"github.com/google/uuid"
	"smap/knowledge-srv/internal/model"
)

// UseCase - Interface chính của domain indexing
type UseCase interface {
	// IndexFromFile - Index batch từ MinIO file
	// Context: request-scoped
	// Scope: service-to-service auth (không cần user context)
	IndexFromFile(ctx context.Context, sc model.Scope, input IndexFromFileInput) (IndexFromFileOutput, error)

	// RetryFailed - Retry các records FAILED
	// Context: background job
	RetryFailed(ctx context.Context, input RetryFailedInput) (RetryFailedOutput, error)

	// Reconcile - Reconcile records PENDING (background job)
	// Context: background job
	Reconcile(ctx context.Context, input ReconcileInput) (ReconcileOutput, error)

	// GetIndexingStats - Lấy thống kê indexing (cho monitoring)
	GetIndexingStats(ctx context.Context, projectID uuid.UUID) (IndexingStats, error)
}

// IndexingStats - Thống kê indexing
type IndexingStats struct {
	ProjectID      uuid.UUID
	TotalIndexed   int
	TotalFailed    int
	TotalPending   int
	LastIndexedAt  *time.Time
	AvgIndexTimeMs int
}
```

### C. ERRORS (`errors.go`)

```go
package indexing

import "errors"

// Domain errors - Theo convention, định nghĩa tại module root
var (
	// ErrFileNotFound - File không tồn tại trong MinIO
	ErrFileNotFound = errors.New("indexing: file not found")

	// ErrFileDownloadFailed - Không tải được file từ MinIO
	ErrFileDownloadFailed = errors.New("indexing: file download failed")

	// ErrFileParseFailed - Parse file thất bại (invalid JSONL)
	ErrFileParseFailed = errors.New("indexing: file parse failed")

	// ErrContentTooShort - Content quá ngắn (< 10 chars)
	ErrContentTooShort = errors.New("indexing: content too short")

	// ErrEmbeddingFailed - Không tạo được embedding
	ErrEmbeddingFailed = errors.New("indexing: embedding generation failed")

	// ErrQdrantUpsertFailed - Upsert vào Qdrant thất bại
	ErrQdrantUpsertFailed = errors.New("indexing: qdrant upsert failed")

	// ErrAlreadyIndexed - Record đã được index
	ErrAlreadyIndexed = errors.New("indexing: record already indexed")

	// ErrDuplicateContent - Content trùng lặp (same hash)
	ErrDuplicateContent = errors.New("indexing: duplicate content")

	// ErrInvalidAnalyticsData - Dữ liệu analytics không hợp lệ
	ErrInvalidAnalyticsData = errors.New("indexing: invalid analytics data")
)
```

### D. REPOSITORY INTERFACE (`repository/interface.go`)

```go
package repository

import (
	"context"
	"time"
	"github.com/google/uuid"
	"smap/knowledge-srv/internal/model"
)

// Repository - Composed interface
type Repository interface {
	IndexedDocumentRepository
	DLQRepository
}

// IndexedDocumentRepository - Operations cho indexed_documents
type IndexedDocumentRepository interface {
	// Create - Tạo mới tracking record (status = PENDING)
	Create(ctx context.Context, doc *model.IndexedDocument) error

	// GetByAnalyticsID - Lấy theo analytics_id (check duplicate)
	GetByAnalyticsID(ctx context.Context, analyticsID uuid.UUID) (*model.IndexedDocument, error)

	// ExistsByAnalyticsID - Check tồn tại
	ExistsByAnalyticsID(ctx context.Context, analyticsID uuid.UUID) (bool, error)

	// ExistsByContentHash - Check content trùng
	ExistsByContentHash(ctx context.Context, contentHash string) (bool, error)

	// UpdateStatus - Update status + metrics
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, metrics StatusMetrics) error

	// Upsert - Insert hoặc update (re-index case)
	Upsert(ctx context.Context, doc *model.IndexedDocument) error

	// ListByStatus - List theo status (for retry/reconcile)
	ListByStatus(ctx context.Context, status string, opts ListOptions) ([]model.IndexedDocument, error)

	// ListStale - List records PENDING quá lâu
	ListStale(ctx context.Context, staleDuration time.Duration, limit int) ([]model.IndexedDocument, error)

	// CountByProject - Thống kê per project
	CountByProject(ctx context.Context, projectID uuid.UUID) (ProjectStats, error)
}

// DLQRepository - Operations cho indexing_dlq
type DLQRepository interface {
	// Create - Tạo DLQ record
	Create(ctx context.Context, dlq *model.IndexingDLQ) error

	// ListUnresolved - List DLQ chưa resolve
	ListUnresolved(ctx context.Context, opts DLQListOptions) ([]model.IndexingDLQ, error)

	// MarkResolved - Đánh dấu đã xử lý
	MarkResolved(ctx context.Context, id uuid.UUID) error
}

// =====================================================
// Options & Filter structs
// =====================================================

// ListOptions - Options cho List query
type ListOptions struct {
	Limit      int
	Offset     int
	OrderBy    string // e.g., "created_at DESC"
	MaxRetry   int    // Filter retry_count < max (for RetryFailed)
	ErrorTypes []string // Filter by error types
}

// DLQListOptions - Options cho DLQ query
type DLQListOptions struct {
	Limit      int
	Offset     int
	ErrorTypes []string
}

// StatusMetrics - Metrics khi update status
type StatusMetrics struct {
	IndexedAt       *time.Time
	ErrorMessage    string
	RetryCount      int
	EmbeddingTimeMs int
	UpsertTimeMs    int
	TotalTimeMs     int
}

// ProjectStats - Thống kê per project
type ProjectStats struct {
	ProjectID      uuid.UUID
	TotalIndexed   int
	TotalFailed    int
	TotalPending   int
	LastIndexedAt  *time.Time
	AvgIndexTimeMs int
}
```

### E. REPOSITORY IMPLEMENTATION (`repository/postgre/indexed_document.go`)

```go
package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"smap/knowledge-srv/internal/indexing/repository"
	"smap/knowledge-srv/internal/model"
	"smap/knowledge-srv/sqlboiler" // Generated by sqlboiler
)

// Create - Tạo mới record
func (r *implRepository) Create(ctx context.Context, doc *model.IndexedDocument) error {
	dbDoc := r.toDBIndexedDocument(doc)
	return dbDoc.Insert(ctx, r.db, boil.Infer())
}

// GetByAnalyticsID - Lấy theo analytics_id
func (r *implRepository) GetByAnalyticsID(ctx context.Context, analyticsID uuid.UUID) (*model.IndexedDocument, error) {
	mods := r.buildGetByAnalyticsIDQuery(analyticsID)
	dbDoc, err := sqlboiler.IndexedDocuments(mods...).One(ctx, r.db)
	if err == sql.ErrNoRows {
		return nil, nil // Không tìm thấy → return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetByAnalyticsID: %w", err)
	}
	return r.toDomainIndexedDocument(dbDoc), nil
}

// ExistsByAnalyticsID - Check tồn tại
func (r *implRepository) ExistsByAnalyticsID(ctx context.Context, analyticsID uuid.UUID) (bool, error) {
	mods := r.buildExistsByAnalyticsIDQuery(analyticsID)
	return sqlboiler.IndexedDocuments(mods...).Exists(ctx, r.db)
}

// ExistsByContentHash - Check content hash
func (r *implRepository) ExistsByContentHash(ctx context.Context, contentHash string) (bool, error) {
	mods := r.buildExistsByContentHashQuery(contentHash)
	return sqlboiler.IndexedDocuments(mods...).Exists(ctx, r.db)
}

// UpdateStatus - Update status + metrics
func (r *implRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, metrics repository.StatusMetrics) error {
	mods := r.buildGetByIDQuery(id)
	dbDoc, err := sqlboiler.IndexedDocuments(mods...).One(ctx, r.db)
	if err != nil {
		return fmt.Errorf("UpdateStatus: %w", err)
	}

	// Update fields
	dbDoc.Status = status
	dbDoc.UpdatedAt = time.Now()
	dbDoc.RetryCount = metrics.RetryCount

	if metrics.IndexedAt != nil {
		dbDoc.IndexedAt.SetValid(*metrics.IndexedAt)
	}
	if metrics.ErrorMessage != "" {
		dbDoc.ErrorMessage.SetValid(metrics.ErrorMessage)
	}
	if metrics.EmbeddingTimeMs > 0 {
		dbDoc.EmbeddingTimeMs.SetValid(metrics.EmbeddingTimeMs)
	}
	if metrics.UpsertTimeMs > 0 {
		dbDoc.UpsertTimeMs.SetValid(metrics.UpsertTimeMs)
	}
	if metrics.TotalTimeMs > 0 {
		dbDoc.TotalTimeMs.SetValid(metrics.TotalTimeMs)
	}

	_, err = dbDoc.Update(ctx, r.db, boil.Infer())
	return err
}

// Upsert - Insert hoặc update
func (r *implRepository) Upsert(ctx context.Context, doc *model.IndexedDocument) error {
	dbDoc := r.toDBIndexedDocument(doc)
	return dbDoc.Upsert(ctx, r.db, true,
		[]string{"analytics_id"}, // Conflict columns
		boil.Infer(), boil.Infer())
}

// ListByStatus - List theo status
func (r *implRepository) ListByStatus(ctx context.Context, status string, opts repository.ListOptions) ([]model.IndexedDocument, error) {
	mods := r.buildListByStatusQuery(status, opts)
	dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("ListByStatus: %w", err)
	}
	return r.toDomainIndexedDocumentList(dbDocs), nil
}

// ListStale - List records PENDING quá lâu
func (r *implRepository) ListStale(ctx context.Context, staleDuration time.Duration, limit int) ([]model.IndexedDocument, error) {
	mods := r.buildListStaleQuery(staleDuration, limit)
	dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
	if err != nil {
		return nil, fmt.Errorf("ListStale: %w", err)
	}
	return r.toDomainIndexedDocumentList(dbDocs), nil
}

// CountByProject - Thống kê per project
func (r *implRepository) CountByProject(ctx context.Context, projectID uuid.UUID) (repository.ProjectStats, error) {
	// Raw SQL query for aggregation
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'INDEXED') as total_indexed,
			COUNT(*) FILTER (WHERE status = 'FAILED') as total_failed,
			COUNT(*) FILTER (WHERE status = 'PENDING') as total_pending,
			MAX(indexed_at) as last_indexed_at,
			AVG(total_time_ms) FILTER (WHERE status = 'INDEXED') as avg_index_time_ms
		FROM knowledge.indexed_documents
		WHERE project_id = $1
	`

	var stats repository.ProjectStats
	var lastIndexedAt sql.NullTime
	var avgIndexTimeMs sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, projectID).Scan(
		&stats.TotalIndexed,
		&stats.TotalFailed,
		&stats.TotalPending,
		&lastIndexedAt,
		&avgIndexTimeMs,
	)
	if err != nil {
		return repository.ProjectStats{}, fmt.Errorf("CountByProject: %w", err)
	}

	stats.ProjectID = projectID
	if lastIndexedAt.Valid {
		stats.LastIndexedAt = &lastIndexedAt.Time
	}
	if avgIndexTimeMs.Valid {
		stats.AvgIndexTimeMs = int(avgIndexTimeMs.Float64)
	}

	return stats, nil
}
```

### F. REPOSITORY QUERY BUILDER (`repository/postgre/indexed_document_query.go`)

```go
package postgre

import (
	"time"
	"github.com/google/uuid"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"smap/knowledge-srv/internal/indexing/repository"
	"smap/knowledge-srv/sqlboiler"
)

// buildGetByAnalyticsIDQuery - Query by analytics_id
func (r *implRepository) buildGetByAnalyticsIDQuery(analyticsID uuid.UUID) []qm.QueryMod {
	return []qm.QueryMod{
		qm.Where("analytics_id = ?", analyticsID),
	}
}

// buildExistsByAnalyticsIDQuery - Exists by analytics_id
func (r *implRepository) buildExistsByAnalyticsIDQuery(analyticsID uuid.UUID) []qm.QueryMod {
	return []qm.QueryMod{
		qm.Where("analytics_id = ?", analyticsID),
		qm.Select("1"),
	}
}

// buildExistsByContentHashQuery - Exists by content_hash
func (r *implRepository) buildExistsByContentHashQuery(contentHash string) []qm.QueryMod {
	return []qm.QueryMod{
		qm.Where("content_hash = ?", contentHash),
		qm.Select("1"),
	}
}

// buildGetByIDQuery - Query by id
func (r *implRepository) buildGetByIDQuery(id uuid.UUID) []qm.QueryMod {
	return []qm.QueryMod{
		qm.Where("id = ?", id),
	}
}

// buildListByStatusQuery - Query list by status
func (r *implRepository) buildListByStatusQuery(status string, opts repository.ListOptions) []qm.QueryMod {
	mods := []qm.QueryMod{
		qm.Where("status = ?", status),
	}

	// Filter by MaxRetry
	if opts.MaxRetry > 0 {
		mods = append(mods, qm.Where("retry_count < ?", opts.MaxRetry))
	}

	// Filter by ErrorTypes
	if len(opts.ErrorTypes) > 0 {
		mods = append(mods, qm.WhereIn("error_type IN ?", toInterfaceSlice(opts.ErrorTypes)...))
	}

	// OrderBy
	if opts.OrderBy != "" {
		mods = append(mods, qm.OrderBy(opts.OrderBy))
	} else {
		mods = append(mods, qm.OrderBy("created_at ASC"))
	}

	// Limit & Offset
	if opts.Limit > 0 {
		mods = append(mods, qm.Limit(opts.Limit))
	}
	if opts.Offset > 0 {
		mods = append(mods, qm.Offset(opts.Offset))
	}

	return mods
}

// buildListStaleQuery - Query stale PENDING records
func (r *implRepository) buildListStaleQuery(staleDuration time.Duration, limit int) []qm.QueryMod {
	staleThreshold := time.Now().Add(-staleDuration)
	mods := []qm.QueryMod{
		qm.Where("status = ?", "PENDING"),
		qm.Where("created_at < ?", staleThreshold),
		qm.OrderBy("created_at ASC"),
	}
	if limit > 0 {
		mods = append(mods, qm.Limit(limit))
	}
	return mods
}

// toInterfaceSlice - Helper: convert []string to []interface{}
func toInterfaceSlice(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}
```

### G. REPOSITORY BUILD MAPPER (`repository/postgre/indexed_document_build.go`)

```go
package postgre

import (
	"smap/knowledge-srv/internal/model"
	"smap/knowledge-srv/sqlboiler"
	"github.com/volatiletech/null/v8"
)

// toDomainIndexedDocument - DB → Domain
func (r *implRepository) toDomainIndexedDocument(db *sqlboiler.IndexedDocument) *model.IndexedDocument {
	if db == nil {
		return nil
	}

	doc := &model.IndexedDocument{
		ID:              db.ID,
		AnalyticsID:     db.AnalyticsID,
		ProjectID:       db.ProjectID,
		SourceID:        db.SourceID,
		QdrantPointID:   db.QdrantPointID,
		CollectionName:  db.CollectionName,
		ContentHash:     db.ContentHash,
		Status:          db.Status,
		RetryCount:      db.RetryCount,
		IngestionMethod: db.IngestionMethod,
		CreatedAt:       db.CreatedAt,
		UpdatedAt:       db.UpdatedAt,
	}

	// Nullable fields
	if db.BatchID.Valid {
		doc.BatchID = db.BatchID.String
	}
	if db.ErrorMessage.Valid {
		doc.ErrorMessage = db.ErrorMessage.String
	}
	if db.IndexedAt.Valid {
		doc.IndexedAt = &db.IndexedAt.Time
	}
	if db.EmbeddingTimeMs.Valid {
		doc.EmbeddingTimeMs = db.EmbeddingTimeMs.Int
	}
	if db.UpsertTimeMs.Valid {
		doc.UpsertTimeMs = db.UpsertTimeMs.Int
	}
	if db.TotalTimeMs.Valid {
		doc.TotalTimeMs = db.TotalTimeMs.Int
	}

	return doc
}

// toDBIndexedDocument - Domain → DB
func (r *implRepository) toDBIndexedDocument(doc *model.IndexedDocument) *sqlboiler.IndexedDocument {
	dbDoc := &sqlboiler.IndexedDocument{
		ID:              doc.ID,
		AnalyticsID:     doc.AnalyticsID,
		ProjectID:       doc.ProjectID,
		SourceID:        doc.SourceID,
		QdrantPointID:   doc.QdrantPointID,
		CollectionName:  doc.CollectionName,
		ContentHash:     doc.ContentHash,
		Status:          doc.Status,
		RetryCount:      doc.RetryCount,
		IngestionMethod: doc.IngestionMethod,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}

	// Nullable fields
	if doc.BatchID != "" {
		dbDoc.BatchID = null.StringFrom(doc.BatchID)
	}
	if doc.ErrorMessage != "" {
		dbDoc.ErrorMessage = null.StringFrom(doc.ErrorMessage)
	}
	if doc.IndexedAt != nil {
		dbDoc.IndexedAt = null.TimeFrom(*doc.IndexedAt)
	}
	if doc.EmbeddingTimeMs > 0 {
		dbDoc.EmbeddingTimeMs = null.IntFrom(doc.EmbeddingTimeMs)
	}
	if doc.UpsertTimeMs > 0 {
		dbDoc.UpsertTimeMs = null.IntFrom(doc.UpsertTimeMs)
	}
	if doc.TotalTimeMs > 0 {
		dbDoc.TotalTimeMs = null.IntFrom(doc.TotalTimeMs)
	}

	return dbDoc
}

// toDomainIndexedDocumentList - List DB → Domain
func (r *implRepository) toDomainIndexedDocumentList(dbDocs []*sqlboiler.IndexedDocument) []model.IndexedDocument {
	result := make([]model.IndexedDocument, len(dbDocs))
	for i, db := range dbDocs {
		if doc := r.toDomainIndexedDocument(db); doc != nil {
			result[i] = *doc
		}
	}
	return result
}
```

---

## VI. USECASE IMPLEMENTATION

### A. USECASE FACTORY (`usecase/new.go`)

```go
package usecase

import (
	"smap/knowledge-srv/internal/indexing"
	"smap/knowledge-srv/internal/indexing/repository"
	"smap/knowledge-srv/pkg/gemini"
	"smap/knowledge-srv/pkg/log"
	"smap/knowledge-srv/pkg/minio"
	"smap/knowledge-srv/pkg/qdrant"
	"smap/knowledge-srv/pkg/redis"
	"smap/knowledge-srv/pkg/voyage"
)

// implUseCase - Implementation của UseCase interface
type implUseCase struct {
	repo      repository.Repository
	qdrant    qdrant.IQdrant
	voyage    voyage.IVoyage
	redis     redis.IRedis
	minio     minio.IMinio
	l         log.Logger

	// Config
	collectionName string
	maxConcurrency int
	minContentLength int
	minQualityScore float64
}

// New - Factory function
func New(
	repo repository.Repository,
	qdrant qdrant.IQdrant,
	voyage voyage.IVoyage,
	redis redis.IRedis,
	minio minio.IMinio,
	l log.Logger,
	collectionName string,
) indexing.UseCase {
	return &implUseCase{
		repo:            repo,
		qdrant:          qdrant,
		voyage:          voyage,
		redis:           redis,
		minio:           minio,
		l:               l,
		collectionName:  collectionName,
		maxConcurrency:  10,          // Parallel workers
		minContentLength: 10,          // Min chars
		minQualityScore: 0.3,         // Min quality score
	}
}
```

### B. USECASE MAIN LOGIC (`usecase/index.go`)

```go
package usecase

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"golang.org/x/sync/errgroup"

	"smap/knowledge-srv/internal/indexing"
	"smap/knowledge-srv/internal/model"
)

// IndexFromFile - Main method: download file → parse → index batch
func (uc *implUseCase) IndexFromFile(
	ctx context.Context,
	sc model.Scope,
	input indexing.IndexFromFileInput,
) (indexing.IndexFromFileOutput, error) {
	startTime := time.Now()

	uc.l.Infof(ctx, "IndexFromFile: batch_id=%s, file_url=%s, count=%d",
		input.BatchID, input.FileURL, input.RecordCount)

	// Step 1: Download file từ MinIO
	uc.l.Infof(ctx, "Downloading file from MinIO: %s", input.FileURL)
	reader, err := uc.minio.Download(ctx, input.FileURL)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to download file: %v", err)
		return indexing.IndexFromFileOutput{}, fmt.Errorf("%w: %v", indexing.ErrFileDownloadFailed, err)
	}
	defer reader.Close()

	// Step 2: Parse JSONL → slice of AnalyticsPost
	uc.l.Infof(ctx, "Parsing JSONL file...")
	records, err := uc.parseJSONL(reader)
	if err != nil {
		uc.l.Errorf(ctx, "Failed to parse file: %v", err)
		return indexing.IndexFromFileOutput{}, fmt.Errorf("%w: %v", indexing.ErrFileParseFailed, err)
	}

	uc.l.Infof(ctx, "Parsed %d records from file", len(records))

	// Step 3: Process batch (parallel)
	result := uc.processBatch(ctx, input, records)

	// Step 4: Invalidate cache (nếu có records thành công)
	if result.Indexed > 0 {
		uc.l.Infof(ctx, "Invalidating search cache for project_id=%s", input.ProjectID)
		if err := uc.invalidateSearchCache(ctx, input.ProjectID); err != nil {
			uc.l.Warnf(ctx, "Failed to invalidate cache: %v", err)
		}
	}

	// Step 5: Return output
	result.BatchID = input.BatchID
	result.TotalRecords = len(records)
	result.Duration = time.Since(startTime)

	uc.l.Infof(ctx, "IndexFromFile completed: indexed=%d, failed=%d, skipped=%d, duration=%v",
		result.Indexed, result.Failed, result.Skipped, result.Duration)

	return result, nil
}

// parseJSONL - Parse JSONL file thành slice of AnalyticsPost
func (uc *implUseCase) parseJSONL(reader io.Reader) ([]indexing.AnalyticsPost, error) {
	var records []indexing.AnalyticsPost
	scanner := bufio.NewScanner(reader)

	// Increase buffer size for large lines (default 64KB → 1MB)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var record indexing.AnalyticsPost
		if err := json.Unmarshal(line, &record); err != nil {
			// Log warning nhưng KHÔNG fail toàn bộ file
			uc.l.Warnf(context.Background(), "Failed to parse line %d: %v", lineNum, err)
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return records, nil
}

// invalidateSearchCache - Invalidate search cache cho project
func (uc *implUseCase) invalidateSearchCache(ctx context.Context, projectID uuid.UUID) error {
	// Pattern: search:{project_id}:*
	pattern := fmt.Sprintf("search:*%s*", projectID.String())
	return uc.redis.DeleteByPattern(ctx, pattern)
}
```

### C. USECASE BATCH PROCESSING (`usecase/batch.go`)

```go
package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"smap/knowledge-srv/internal/indexing"
)

// processBatch - Xử lý batch song song với errgroup
func (uc *implUseCase) processBatch(
	ctx context.Context,
	input indexing.IndexFromFileInput,
	records []indexing.AnalyticsPost,
) indexing.IndexFromFileOutput {

	var (
		indexed  int
		failed   int
		skipped  int
		mu       sync.Mutex // Protect counters
		failedRecords []indexing.FailedRecord
	)

	// errgroup với limit concurrency
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(uc.maxConcurrency) // Max 10 workers song song

	for i := range records {
		record := records[i] // Capture loop variable

		g.Go(func() error {
			// Process single record
			result := uc.indexSingleRecord(gctx, input, record)

			// Update counters (thread-safe)
			mu.Lock()
			defer mu.Unlock()

			switch result.Status {
			case "indexed":
				indexed++
			case "skipped":
				skipped++
			case "failed":
				failed++
				failedRecords = append(failedRecords, indexing.FailedRecord{
					AnalyticsID:  record.ID,
					ErrorType:    result.ErrorType,
					ErrorMessage: result.ErrorMessage,
				})
			}

			return nil // KHÔNG return error để không stop errgroup
		})
	}

	// Wait all goroutines
	_ = g.Wait() // Không quan tâm error vì đã handle per-record

	return indexing.IndexFromFileOutput{
		Indexed:       indexed,
		Failed:        failed,
		Skipped:       skipped,
		FailedRecords: failedRecords,
	}
}

// indexRecordResult - Kết quả xử lý 1 record
type indexRecordResult struct {
	Status       string // "indexed", "skipped", "failed"
	ErrorType    string
	ErrorMessage string
}
```

### D. USECASE SINGLE RECORD PROCESSING (`usecase/index_record.go`)

```go
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"smap/knowledge-srv/internal/indexing"
	"smap/knowledge-srv/internal/model"
)

// indexSingleRecord - Xử lý 1 record: validate → dedup → embed → upsert → save
func (uc *implUseCase) indexSingleRecord(
	ctx context.Context,
	input indexing.IndexFromFileInput,
	record indexing.AnalyticsPost,
) indexRecordResult {

	startTime := time.Now()

	// Step 1: Validate record
	if err := uc.validateAnalyticsPost(record); err != nil {
		return indexRecordResult{
			Status:       "skipped",
			ErrorType:    "VALIDATION_ERROR",
			ErrorMessage: err.Error(),
		}
	}

	// Step 2: Pre-filter (spam, bot, quality)
	if uc.shouldSkipRecord(record) {
		return indexRecordResult{Status: "skipped"}
	}

	// Step 3: Check duplicate
	contentHash := uc.generateContentHash(record.Content)

	// Check analytics_id đã index chưa
	exists, _ := uc.repo.ExistsByAnalyticsID(ctx, record.ID)
	isReindex := exists

	if !isReindex {
		// Check content hash (trùng content từ nguồn khác)
		contentExists, _ := uc.repo.ExistsByContentHash(ctx, contentHash)
		if contentExists {
			return indexRecordResult{
				Status:    "skipped",
				ErrorType: "DUPLICATE_CONTENT",
			}
		}
	}

	// Step 4: Create/Update tracking record (status = PENDING)
	pointID := record.ID // Dùng analytics_id làm Qdrant point ID
	trackingDoc := &model.IndexedDocument{
		ID:              uuid.New(),
		AnalyticsID:     record.ID,
		ProjectID:       record.ProjectID,
		SourceID:        record.SourceID,
		QdrantPointID:   pointID,
		CollectionName:  uc.collectionName,
		ContentHash:     contentHash,
		Status:          "PENDING",
		BatchID:         input.BatchID,
		IngestionMethod: input.IngestionMethod,
		RetryCount:      0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if isReindex {
		// Re-index case: upsert
		if err := uc.repo.Upsert(ctx, trackingDoc); err != nil {
			return indexRecordResult{
				Status:       "failed",
				ErrorType:    "DB_ERROR",
				ErrorMessage: err.Error(),
			}
		}
	} else {
		// First time: create
		if err := uc.repo.Create(ctx, trackingDoc); err != nil {
			return indexRecordResult{
				Status:       "failed",
				ErrorType:    "DB_ERROR",
				ErrorMessage: err.Error(),
			}
		}
	}

	// Step 5: Embed content
	embeddingStart := time.Now()
	vector, err := uc.embedContent(ctx, record.Content)
	embeddingTime := int(time.Since(embeddingStart).Milliseconds())

	if err != nil {
		// Update status = FAILED
		uc.updateFailedStatus(ctx, trackingDoc.ID, "EMBEDDING_ERROR", err.Error(), 0, 0)
		return indexRecordResult{
			Status:       "failed",
			ErrorType:    "EMBEDDING_ERROR",
			ErrorMessage: err.Error(),
		}
	}

	// Step 6: Prepare Qdrant point
	payload := uc.prepareQdrantPayload(record)

	// Step 7: Upsert to Qdrant
	upsertStart := time.Now()
	err = uc.upsertToQdrant(ctx, pointID, vector, payload)
	upsertTime := int(time.Since(upsertStart).Milliseconds())

	if err != nil {
		// Update status = FAILED
		uc.updateFailedStatus(ctx, trackingDoc.ID, "QDRANT_ERROR", err.Error(), embeddingTime, 0)
		return indexRecordResult{
			Status:       "failed",
			ErrorType:    "QDRANT_ERROR",
			ErrorMessage: err.Error(),
		}
	}

	// Step 8: Update status = INDEXED
	totalTime := int(time.Since(startTime).Milliseconds())
	now := time.Now()
	uc.repo.UpdateStatus(ctx, trackingDoc.ID, "INDEXED", repository.StatusMetrics{
		IndexedAt:       &now,
		EmbeddingTimeMs: embeddingTime,
		UpsertTimeMs:    upsertTime,
		TotalTimeMs:     totalTime,
	})

	return indexRecordResult{Status: "indexed"}
}

// updateFailedStatus - Update status = FAILED
func (uc *implUseCase) updateFailedStatus(
	ctx context.Context,
	docID uuid.UUID,
	errorType, errorMessage string,
	embeddingTime, upsertTime int,
) {
	uc.repo.UpdateStatus(ctx, docID, "FAILED", repository.StatusMetrics{
		ErrorMessage:    fmt.Sprintf("[%s] %s", errorType, errorMessage),
		RetryCount:      0, // Will be incremented on retry
		EmbeddingTimeMs: embeddingTime,
		UpsertTimeMs:    upsertTime,
	})
}
```

### E. USECASE VALIDATION (`usecase/validation.go`)

```go
package usecase

import (
	"fmt"
	"smap/knowledge-srv/internal/indexing"
)

// validateAnalyticsPost - Validate basic fields
func (uc *implUseCase) validateAnalyticsPost(record indexing.AnalyticsPost) error {
	if record.ID == uuid.Nil {
		return fmt.Errorf("missing analytics_id")
	}
	if record.ProjectID == uuid.Nil {
		return fmt.Errorf("missing project_id")
	}
	if record.SourceID == uuid.Nil {
		return fmt.Errorf("missing source_id")
	}
	if len(record.Content) < uc.minContentLength {
		return indexing.ErrContentTooShort
	}
	return nil
}

// shouldSkipRecord - Pre-filter: spam, bot, quality
func (uc *implUseCase) shouldSkipRecord(record indexing.AnalyticsPost) bool {
	// Skip spam
	if record.IsSpam {
		return true
	}
	// Skip bot
	if record.IsBot {
		return true
	}
	// Skip low quality
	if record.ContentQualityScore < uc.minQualityScore {
		return true
	}
	return false
}
```

### F. USECASE EMBEDDING (`usecase/embedding.go`)

```go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// embedContent - Sinh embedding với Redis cache
func (uc *implUseCase) embedContent(ctx context.Context, content string) ([]float32, error) {
	// Check cache
	cacheKey := uc.getEmbeddingCacheKey(content)

	// Try get from Redis
	cachedVector, err := uc.getEmbeddingFromCache(ctx, cacheKey)
	if err == nil && cachedVector != nil {
		uc.l.Debugf(ctx, "Embedding cache hit for key: %s", cacheKey)
		return cachedVector, nil
	}

	// Cache miss → Call Voyage API
	uc.l.Debugf(ctx, "Embedding cache miss, calling Voyage API")
	vector, err := uc.voyage.Embed(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", indexing.ErrEmbeddingFailed, err)
	}

	// Save to cache (TTL 7 days)
	if err := uc.saveEmbeddingToCache(ctx, cacheKey, vector, 7*24*time.Hour); err != nil {
		uc.l.Warnf(ctx, "Failed to save embedding to cache: %v", err)
	}

	return vector, nil
}

// getEmbeddingCacheKey - Generate cache key from content hash
func (uc *implUseCase) getEmbeddingCacheKey(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("embedding:%x", hash)
}

// getEmbeddingFromCache - Get embedding từ Redis
func (uc *implUseCase) getEmbeddingFromCache(ctx context.Context, key string) ([]float32, error) {
	data, err := uc.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var vector []float32
	if err := json.Unmarshal([]byte(data), &vector); err != nil {
		return nil, err
	}

	return vector, nil
}

// saveEmbeddingToCache - Save embedding to Redis
func (uc *implUseCase) saveEmbeddingToCache(ctx context.Context, key string, vector []float32, ttl time.Duration) error {
	data, err := json.Marshal(vector)
	if err != nil {
		return err
	}
	return uc.redis.Set(ctx, key, string(data), ttl)
}
```

### G. USECASE QDRANT OPERATIONS (`usecase/qdrant.go`)

```go
package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"smap/knowledge-srv/internal/indexing"
	"smap/knowledge-srv/pkg/qdrant"
)

// prepareQdrantPayload - Chuẩn bị payload từ AnalyticsPost
func (uc *implUseCase) prepareQdrantPayload(record indexing.AnalyticsPost) map[string]interface{} {
	payload := map[string]interface{}{
		// Core Identity
		"analytics_id": record.ID.String(),
		"project_id":   record.ProjectID.String(),
		"source_id":    record.SourceID.String(),

		// UAP Core
		"content":            uc.truncateContent(record.Content, 1000), // Truncate to 1000 chars
		"content_created_at": record.ContentCreatedAt.Unix(),
		"ingested_at":        record.IngestedAt.Unix(),
		"platform":           record.Platform,

		// Sentiment
		"overall_sentiment":       record.OverallSentiment,
		"overall_sentiment_score": record.OverallSentimentScore,
		"sentiment_confidence":    record.SentimentConfidence,

		// Keywords
		"keywords": record.Keywords,

		// Risk
		"risk_level":         record.RiskLevel,
		"risk_score":         record.RiskScore,
		"requires_attention": record.RequiresAttention,

		// Engagement
		"engagement_score": record.EngagementScore,
		"virality_score":   record.ViralityScore,
		"influence_score":  record.InfluenceScore,
		"reach_estimate":   record.ReachEstimate,

		// Quality
		"content_quality_score": record.ContentQualityScore,
		"is_spam":               record.IsSpam,
		"is_bot":                record.IsBot,
		"language":              record.Language,
		"toxicity_score":        record.ToxicityScore,
	}

	// Aspects (array of objects)
	if len(record.Aspects) > 0 {
		aspects := make([]map[string]interface{}, len(record.Aspects))
		for i, aspect := range record.Aspects {
			aspects[i] = map[string]interface{}{
				"aspect":            aspect.Aspect,
				"aspect_display_name": aspect.AspectDisplayName,
				"sentiment":         aspect.Sentiment,
				"sentiment_score":   aspect.SentimentScore,
				"keywords":          aspect.Keywords,
				"impact_score":      aspect.ImpactScore,
			}
		}
		payload["aspects"] = aspects
	}

	// Metadata (nested object)
	metadata := map[string]interface{}{
		"author":              record.UAPMetadata.Author,
		"author_display_name": record.UAPMetadata.AuthorDisplayName,
		"author_followers":    record.UAPMetadata.AuthorFollowers,
		"engagement": map[string]interface{}{
			"views":    record.UAPMetadata.Engagement.Views,
			"likes":    record.UAPMetadata.Engagement.Likes,
			"comments": record.UAPMetadata.Engagement.Comments,
			"shares":   record.UAPMetadata.Engagement.Shares,
		},
	}
	if record.UAPMetadata.VideoURL != "" {
		metadata["video_url"] = record.UAPMetadata.VideoURL
	}
	if len(record.UAPMetadata.Hashtags) > 0 {
		metadata["hashtags"] = record.UAPMetadata.Hashtags
	}
	if record.UAPMetadata.Location != "" {
		metadata["location"] = record.UAPMetadata.Location
	}
	payload["metadata"] = metadata

	return payload
}

// upsertToQdrant - Upsert point vào Qdrant
func (uc *implUseCase) upsertToQdrant(
	ctx context.Context,
	pointID uuid.UUID,
	vector []float32,
	payload map[string]interface{},
) error {
	point := qdrant.Point{
		ID:      pointID.String(),
		Vector:  vector,
		Payload: payload,
	}

	err := uc.qdrant.UpsertPoints(ctx, uc.collectionName, []qdrant.Point{point})
	if err != nil {
		return fmt.Errorf("%w: %v", indexing.ErrQdrantUpsertFailed, err)
	}

	return nil
}

// truncateContent - Truncate content to max length
func (uc *implUseCase) truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}
```

### H. USECASE HELPERS (`usecase/helpers.go`)

```go
package usecase

import (
	"crypto/sha256"
	"fmt"
)

// generateContentHash - Generate SHA-256 hash của content
func (uc *implUseCase) generateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
```

---

## VII. DELIVERY LAYER

### A. HTTP DELIVERY (`delivery/http/handlers.go`)

```go
package http

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"smap/knowledge-srv/internal/indexing"
	"smap/knowledge-srv/internal/model"
	"smap/knowledge-srv/pkg/log"
	"smap/knowledge-srv/pkg/response"
)

// handler - HTTP handler implementation
type handler struct {
	l  log.Logger
	uc indexing.UseCase
}

// IndexByFile - Handler cho POST /internal/index/by-file
// @Summary Index batch từ MinIO file
// @Description Internal API cho Analytics Service trigger indexing
// @Tags Indexing (Internal)
// @Accept json
// @Produce json
// @Param body body indexByFileReq true "Index request"
// @Success 200 {object} indexByFileResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /internal/index/by-file [post]
func (h *handler) IndexByFile(c *gin.Context) {
	ctx := c.Request.Context()

	// Process request
	req, sc, err := h.processIndexByFileRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "processIndexByFileRequest failed: %v", err)
		response.Error(c, err)
		return
	}

	// Convert to UseCase input
	input := req.toInput()

	// Call UseCase
	output, err := h.uc.IndexFromFile(ctx, sc, input)
	if err != nil {
		h.l.Errorf(ctx, "IndexFromFile failed: %v", err)
		response.Error(c, h.mapError(err))
		return
	}

	// Return response
	resp := h.newIndexByFileResp(output)
	response.Success(c, resp)
}
```

### B. HTTP PRESENTERS (`delivery/http/presenters.go`)

```go
package http

import (
	"time"
	"github.com/google/uuid"
	"smap/knowledge-srv/internal/indexing"
)

// =====================================================
// Request DTOs
// =====================================================

// indexByFileReq - Request body cho IndexByFile
type indexByFileReq struct {
	BatchID     string `json:"batch_id" binding:"required"`
	ProjectID   string `json:"project_id" binding:"required,uuid"`
	FileURL     string `json:"file_url" binding:"required"`
	RecordCount int    `json:"record_count"`
}

// validate - Custom validation
func (r indexByFileReq) validate() error {
	// FileURL phải bắt đầu bằng s3://
	if len(r.FileURL) < 6 || r.FileURL[:5] != "s3://" {
		return errors.New("file_url must start with s3://")
	}
	return nil
}

// toInput - Convert to UseCase input
func (r indexByFileReq) toInput() indexing.IndexFromFileInput {
	projectID, _ := uuid.Parse(r.ProjectID)
	return indexing.IndexFromFileInput{
		BatchID:         r.BatchID,
		ProjectID:       projectID,
		FileURL:         r.FileURL,
		RecordCount:     r.RecordCount,
		IngestionMethod: "api",
	}
}

// =====================================================
// Response DTOs
// =====================================================

// indexByFileResp - Response cho IndexByFile
type indexByFileResp struct {
	BatchID       string              `json:"batch_id"`
	TotalRecords  int                 `json:"total_records"`
	Indexed       int                 `json:"indexed"`
	Failed        int                 `json:"failed"`
	Skipped       int                 `json:"skipped"`
	DurationMs    int64               `json:"duration_ms"`
	FailedRecords []failedRecordResp  `json:"failed_records,omitempty"`
}

// failedRecordResp - Chi tiết record lỗi
type failedRecordResp struct {
	AnalyticsID  string `json:"analytics_id"`
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
}

// newIndexByFileResp - Convert UseCase output to response
func (h *handler) newIndexByFileResp(output indexing.IndexFromFileOutput) indexByFileResp {
	resp := indexByFileResp{
		BatchID:      output.BatchID,
		TotalRecords: output.TotalRecords,
		Indexed:      output.Indexed,
		Failed:       output.Failed,
		Skipped:      output.Skipped,
		DurationMs:   output.Duration.Milliseconds(),
	}

	// Map failed records
	if len(output.FailedRecords) > 0 {
		resp.FailedRecords = make([]failedRecordResp, len(output.FailedRecords))
		for i, fr := range output.FailedRecords {
			resp.FailedRecords[i] = failedRecordResp{
				AnalyticsID:  fr.AnalyticsID.String(),
				ErrorType:    fr.ErrorType,
				ErrorMessage: fr.ErrorMessage,
			}
		}
	}

	return resp
}
```

### C. HTTP PROCESS REQUEST (`delivery/http/process_request.go`)

```go
package http

import (
	"github.com/gin-gonic/gin"
	"smap/knowledge-srv/internal/model"
	"smap/knowledge-srv/pkg/scope"
)

// processIndexByFileRequest - Validate + extract scope
func (h *handler) processIndexByFileRequest(c *gin.Context) (indexByFileReq, model.Scope, error) {
	var req indexByFileReq

	// Bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, model.Scope{}, err
	}

	// Custom validate
	if err := req.validate(); err != nil {
		return req, model.Scope{}, err
	}

	// Extract scope (service-to-service, không cần user)
	sc := scope.Extract(c)

	return req, sc, nil
}
```

### D. HTTP ERROR MAPPING (`delivery/http/errors.go`)

```go
package http

import (
	"errors"
	"smap/knowledge-srv/internal/indexing"
	pkgErrors "smap/knowledge-srv/pkg/errors"
)

// Error codes
var (
	errFileNotFound = pkgErrors.NewHTTPError(
		pkgErrors.CodeBadRequest,
		"File not found in MinIO",
	)

	errFileDownloadFailed = pkgErrors.NewHTTPError(
		pkgErrors.CodeInternalError,
		"Failed to download file from MinIO",
	)

	errFileParseFailed = pkgErrors.NewHTTPError(
		pkgErrors.CodeBadRequest,
		"Failed to parse JSONL file",
	)

	errEmbeddingFailed = pkgErrors.NewHTTPError(
		pkgErrors.CodeInternalError,
		"Failed to generate embedding",
	)

	errQdrantFailed = pkgErrors.NewHTTPError(
		pkgErrors.CodeInternalError,
		"Failed to upsert to Qdrant",
	)
)

// mapError - Map domain errors to HTTP errors
func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, indexing.ErrFileNotFound):
		return errFileNotFound
	case errors.Is(err, indexing.ErrFileDownloadFailed):
		return errFileDownloadFailed
	case errors.Is(err, indexing.ErrFileParseFailed):
		return errFileParseFailed
	case errors.Is(err, indexing.ErrEmbeddingFailed):
		return errEmbeddingFailed
	case errors.Is(err, indexing.ErrQdrantUpsertFailed):
		return errQdrantFailed
	default:
		// Unknown error → panic (theo convention)
		panic(err)
	}
}
```

### E. HTTP ROUTES (`delivery/http/routes.go`)

```go
package http

import (
	"github.com/gin-gonic/gin"
	"smap/knowledge-srv/pkg/middleware"
)

// RegisterRoutes - Register HTTP routes
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	// Internal API group (service-to-service auth)
	internal := r.Group("/internal")
	internal.Use(mw.ServiceAuth()) // Service token authentication
	{
		internal.POST("/index/by-file", h.IndexByFile)
	}
}
```

### F. KAFKA CONSUMER (`delivery/kafka/consumer/handler.go`)

```go
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Shopify/sarama"
	"smap/knowledge-srv/internal/indexing"
	"smap/knowledge-srv/internal/model"
	"smap/knowledge-srv/pkg/log"
)

// GroupHandler - Implements sarama.ConsumerGroupHandler
type GroupHandler struct {
	uc indexing.UseCase
	l  log.Logger
}

// Setup - Runs when consumer group rebalances
func (h *GroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.l.Infof(context.Background(), "Kafka consumer setup")
	return nil
}

// Cleanup - Runs after session ends
func (h *GroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.l.Infof(context.Background(), "Kafka consumer cleanup")
	return nil
}

// ConsumeClaim - Main consumption loop
func (h *GroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		ctx := context.Background()

		h.l.Infof(ctx, "Received Kafka message: topic=%s, partition=%d, offset=%d",
			msg.Topic, msg.Partition, msg.Offset)

		// Process message
		err := h.processMessage(ctx, msg)
		if err != nil {
			h.l.Errorf(ctx, "Failed to process message: %v", err)
			// KHÔNG return error → tiếp tục consume
			// Message sẽ KHÔNG được mark → retry sau (tùy config)
		} else {
			// Mark message as processed
			session.MarkMessage(msg, "")
		}
	}

	return nil
}

// processMessage - Process 1 Kafka message
func (h *GroupHandler) processMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	// Parse message
	var batchMsg BatchCompletedMessage
	if err := json.Unmarshal(msg.Value, &batchMsg); err != nil {
		h.l.Warnf(ctx, "Failed to unmarshal message (poison message): %v", err)
		// Poison message → skip (không retry)
		return nil
	}

	// Convert to UseCase input
	input := indexing.IndexFromFileInput{
		BatchID:         batchMsg.BatchID,
		ProjectID:       batchMsg.ProjectID,
		FileURL:         batchMsg.FileURL,
		RecordCount:     batchMsg.RecordCount,
		IngestionMethod: "kafka",
	}

	// Call UseCase
	sc := model.Scope{} // Service-to-service, không có user context
	output, err := h.uc.IndexFromFile(ctx, sc, input)
	if err != nil {
		return fmt.Errorf("IndexFromFile failed: %w", err)
	}

	h.l.Infof(ctx, "Batch indexed: batch_id=%s, indexed=%d, failed=%d, skipped=%d",
		output.BatchID, output.Indexed, output.Failed, output.Skipped)

	return nil
}

// BatchCompletedMessage - Kafka message DTO
type BatchCompletedMessage struct {
	BatchID     string    `json:"batch_id"`
	ProjectID   uuid.UUID `json:"project_id"`
	FileURL     string    `json:"file_url"`
	RecordCount int       `json:"record_count"`
	CompletedAt time.Time `json:"completed_at"`
}
```

---

## VIII. TESTING STRATEGY

### A. Unit Tests

**Target:** UseCase logic

**Files:**

- `usecase/index_test.go`
- `usecase/batch_test.go`
- `usecase/validation_test.go`
- `usecase/embedding_test.go`

**Mocking:**

- Repository: Mock interface
- Qdrant: Mock interface
- Voyage: Mock interface
- Redis: Mock interface
- MinIO: Mock interface

**Test cases:**

```go
// index_test.go
func TestIndexFromFile_Success(t *testing.T)
func TestIndexFromFile_FileNotFound(t *testing.T)
func TestIndexFromFile_ParseError(t *testing.T)
func TestIndexFromFile_PartialFailure(t *testing.T)

// validation_test.go
func TestValidateAnalyticsPost_MissingID(t *testing.T)
func TestValidateAnalyticsPost_ContentTooShort(t *testing.T)
func TestShouldSkipRecord_Spam(t *testing.T)
func TestShouldSkipRecord_Bot(t *testing.T)
func TestShouldSkipRecord_LowQuality(t *testing.T)

// embedding_test.go
func TestEmbedContent_CacheHit(t *testing.T)
func TestEmbedContent_CacheMiss(t *testing.T)
func TestEmbedContent_APIFailed(t *testing.T)
```

### B. Integration Tests

**Target:** Repository + PostgreSQL

**Files:**

- `repository/postgre/indexed_document_test.go`
- `repository/postgre/dlq_test.go`

**Setup:**

- Docker Compose: PostgreSQL test instance
- Test data: Fixtures

**Test cases:**

```go
func TestCreate_Success(t *testing.T)
func TestGetByAnalyticsID_Found(t *testing.T)
func TestGetByAnalyticsID_NotFound(t *testing.T)
func TestExistsByAnalyticsID(t *testing.T)
func TestExistsByContentHash(t *testing.T)
func TestUpdateStatus(t *testing.T)
func TestUpsert_Insert(t *testing.T)
func TestUpsert_Update(t *testing.T)
func TestListByStatus(t *testing.T)
func TestListStale(t *testing.T)
func TestCountByProject(t *testing.T)
```

### C. End-to-End Tests

**Target:** Full flow HTTP → UseCase → Qdrant

**Files:**

- `e2e/indexing_test.go`

**Setup:**

- Docker Compose: PostgreSQL + Qdrant + Redis + MinIO
- Mock Analytics file in MinIO

**Test cases:**

```go
func TestE2E_IndexByFile_HTTP(t *testing.T)
func TestE2E_IndexByFile_Kafka(t *testing.T)
func TestE2E_Reconcile(t *testing.T)
func TestE2E_RetryFailed(t *testing.T)
```

---

## IX. MONITORING & OBSERVABILITY

### A. Metrics (Prometheus)

```go
// metrics.go
var (
	indexingTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "knowledge_indexing_total",
			Help: "Total number of indexing operations",
		},
		[]string{"status"}, // indexed, failed, skipped
	)

	indexingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "knowledge_indexing_duration_seconds",
			Help:    "Duration of indexing operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"}, // embed, upsert, total
	)

	batchSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "knowledge_indexing_batch_size",
			Help:    "Number of records per batch",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
		},
	)
)
```

### B. Logging

**Log levels:**

- **INFO**: Batch start/end, summary stats
- **WARN**: Skipped records, cache miss
- **ERROR**: Failed records, external API errors

**Log fields:**

```go
h.l.Infof(ctx, "IndexFromFile completed",
	log.Field("batch_id", input.BatchID),
	log.Field("total", result.TotalRecords),
	log.Field("indexed", result.Indexed),
	log.Field("failed", result.Failed),
	log.Field("skipped", result.Skipped),
	log.Field("duration_ms", result.Duration.Milliseconds()),
)
```

### C. Tracing (OpenTelemetry)

**Spans:**

- `indexing.IndexFromFile`
  - `indexing.downloadFile`
  - `indexing.parseJSONL`
  - `indexing.processBatch`
    - `indexing.indexSingleRecord` (per record)
      - `indexing.embedContent`
      - `indexing.upsertToQdrant`

---

## X. DEPLOYMENT

### A. Environment Variables

```bash
# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_NAME=knowledge
DB_USER=postgres
DB_PASSWORD=postgres

# Qdrant
QDRANT_URL=http://localhost:6333
QDRANT_COLLECTION=smap_analytics

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false

# Voyage AI
VOYAGE_API_KEY=pa-xxx
VOYAGE_MODEL=voyage-2

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_CONSUMER_GROUP=knowledge-indexing
KAFKA_TOPIC_ANALYTICS_BATCH=analytics.batch.completed

# Indexing Config
INDEXING_MAX_CONCURRENCY=10
INDEXING_MIN_CONTENT_LENGTH=10
INDEXING_MIN_QUALITY_SCORE=0.3
```

### B. Docker Compose (Development)

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: knowledge
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
    volumes:
      - qdrant_data:/qdrant/storage

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    ports:
      - "9092:9092"
    environment:
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
    depends_on:
      - zookeeper

  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    ports:
      - "2181:2181"
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

volumes:
  postgres_data:
  qdrant_data:
  minio_data:
```

---

## XI. CHECKLIST IMPLEMENTATION

### Phase 1: Foundation (Week 1)

- [ ] Setup database schema (PostgreSQL)
  - [ ] `knowledge.indexed_documents`
  - [ ] `knowledge.indexing_dlq`
  - [ ] Indexes
- [ ] Setup Qdrant collection
  - [ ] Collection config
  - [ ] Test connection
- [ ] Implement `pkg/` dependencies
  - [ ] `pkg/qdrant`
  - [ ] `pkg/voyage`
  - [ ] `pkg/minio`
  - [ ] `pkg/redis`
- [ ] Define domain types
  - [ ] `types.go`
  - [ ] `errors.go`
  - [ ] `interface.go`
- [ ] Implement repository
  - [ ] Interface
  - [ ] PostgreSQL implementation
  - [ ] Query builders
  - [ ] Mappers

### Phase 2: Core Logic (Week 2)

- [ ] Implement UseCase
  - [ ] `index.go` - IndexFromFile
  - [ ] `batch.go` - processBatch
  - [ ] `index_record.go` - indexSingleRecord
  - [ ] `validation.go` - validation logic
  - [ ] `embedding.go` - embedding with cache
  - [ ] `qdrant.go` - Qdrant operations
  - [ ] `helpers.go` - utility functions
- [ ] Unit tests
  - [ ] UseCase tests (mocked dependencies)
  - [ ] Validation tests
  - [ ] Embedding cache tests

### Phase 3: Delivery Layer (Week 3)

- [ ] HTTP delivery
  - [ ] Handlers
  - [ ] Presenters
  - [ ] Routes
  - [ ] Error mapping
- [ ] Kafka consumer
  - [ ] Consumer group setup
  - [ ] Message handler
  - [ ] Worker logic
- [ ] Integration tests
  - [ ] Repository tests (real DB)
  - [ ] HTTP endpoint tests
  - [ ] Kafka consumer tests

### Phase 4: Production Readiness (Week 4)

- [ ] Implement Reconcile job
  - [ ] `usecase/reconcile.go`
  - [ ] Background scheduler
- [ ] Implement RetryFailed job
  - [ ] `usecase/retry.go`
  - [ ] DLQ logic
- [ ] Monitoring
  - [ ] Prometheus metrics
  - [ ] Structured logging
  - [ ] OpenTelemetry tracing
- [ ] Documentation
  - [ ] API docs (Swagger)
  - [ ] README
  - [ ] Deployment guide
- [ ] E2E tests
  - [ ] Full flow tests
  - [ ] Load tests

---

## XII. APPENDIX

### A. Sample Analytics Post JSON

Xem file: `documents/analytics_post_example.json`

### B. Reference Documents

- Master proposal: `documents/master-proposal.md`
- Analytics schema: `documents/analytic_post_schema.md`
- Domain convention: `documents/domain_convention/convention.md`
- Pkg convention: `documents/pkg_convention.md`

---

**END OF DOCUMENT**
