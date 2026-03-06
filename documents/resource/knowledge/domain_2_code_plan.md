# Domain 2: Search -- Chi tiết Plan Code

**Version:** 1.0  
**Last Updated:** 2026-02-16  
**Domain:** `internal/search` -- Semantic Search

---

## I. TỔNG QUAN

### 1. Vai trò Domain

Domain Search là **domain nền tảng** (foundation domain) của hệ thống RAG Knowledge Service, chịu trách nhiệm:

- Thực hiện semantic search trong Qdrant Vector Database
- Quản lý campaign scope filtering (resolve campaign → project_ids)
- Áp dụng multi-criteria payload filtering (sentiment, aspects, date range, platform)
- Quản lý 3 tầng cache (embedding, campaign projects, search results)
- Cung cấp aggregated search results cho downstream domains (`chat`, `report`)
- Áp dụng hallucination control qua configurable `min_score` threshold

### 2. Input/Output

**Input:**

- Search request từ 3 nguồn:
  - **HTTP API**: `POST /api/v1/search` → Public API, JWT auth
  - **chat.UseCase**: `search.UseCase.Search()` → Internal call từ RAG pipeline
  - **report.UseCase**: `search.UseCase.Search()` → Internal call từ report generation

**Output:**

- Structured search results bao gồm:
  - Ranked documents với relevance scores
  - Aggregations (by_sentiment, by_aspect, by_platform)
  - `NoRelevantContext` flag cho hallucination control

### 3. Kiến trúc tổng quan

```
┌─────────────────────────────────────────────────────────────────┐
│                      Callers                                     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │ HTTP Client      │  │ chat.UseCase     │  │ report.UC    │  │
│  │ (POST /search)   │  │ (RAG Pipeline)   │  │ (Generator)  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └──────┬───────┘  │
└───────────┼──────────────────────┼───────────────────┼──────────┘
            │                      │                    │
            └──────────┬───────────┘────────────────────┘
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Knowledge Service (Domain: Search)              │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    UseCase                                │   │
│  │   Search(ctx, sc, input) → (SearchOutput, error)         │   │
│  └────────┬──────────┬──────────┬──────────┬────────────────┘   │
│           │          │          │          │                      │
│     ┌─────▼────┐ ┌───▼────┐ ┌──▼─────┐ ┌─▼──────────┐         │
│     │ [Cache]  │ │Voyage  │ │Project │ │  Qdrant    │         │
│     │ 3-tier   │ │Embed   │ │Service │ │  Search    │         │
│     │ Redis    │ │Query   │ │Resolve │ │  + Filter  │         │
│     └──────────┘ └────────┘ └────────┘ └────────────┘         │
│                                                                  │
│  Stateless: Không có PostgreSQL table riêng                     │
│  Cache-only: 3 tầng Redis cache                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## II. CACHING STRATEGY (3 Tầng)

### Tầng 1: Embedding Cache

```
Key:       embedding:{sha256(query_text)}
Value:     []float32 (JSON serialized vector, 1536 dimensions)
TTL:       7 ngày
Size:      ~6KB per entry

Invalidation: Không cần (deterministic — cùng input = cùng output).
```

### Tầng 2: Campaign Projects Cache

```
Key:       campaign_projects:{campaign_id}
Value:     []string (JSON array of project_ids)
TTL:       10 phút

Invalidation: TTL-based. Hoặc active invalidation khi indexing domain
              thêm project mới vào campaign.
```

### Tầng 3: Search Results Cache

```
Key:       search:{campaign_id}:{sha256(query + filters_json)}
Value:     SearchOutput (JSON serialized)
TTL:       5 phút

Invalidation:
    - TTL-based: 5 phút tự expire
    - Active: indexing domain gọi InvalidateSearchCache khi index data mới
```

### Tổng hợp cache flow

```
POST /api/v1/search
    │
    ▼
[Tầng 3] Search results cache ──── HIT → return ngay (< 5ms)
    │ MISS
    ▼
[Tầng 2] Campaign projects cache ── HIT → dùng cached project_ids
    │ MISS → gọi Project Service (50ms)
    ▼
[Tầng 1] Embedding cache ────────── HIT → dùng cached vector (< 5ms)
    │ MISS → gọi Voyage API (200-800ms)
    ▼
Query Qdrant (100-300ms)
    │
    ▼
Aggregate results + SET cả 3 tầng cache → return response

Latency:
    - Full cache hit:  < 10ms
    - Partial hits:    200-500ms
    - Full cache miss: 500-1200ms
```

---

## III. CODEBASE STRUCTURE

```
internal/search/
├── delivery/
│   └── http/
│       ├── new.go                    # Factory: New(l, uc, discord) Handler
│       ├── handlers.go               # Search handler
│       ├── process_request.go        # processSearchReq: validate + extract Scope
│       ├── presenters.go             # SearchReq/Resp DTOs, AggregationResp
│       ├── routes.go                 # RegisterRoutes: /api/v1/search
│       └── errors.go                 # mapError (errors.Is based)
├── repository/
│   ├── interface.go                  # QdrantRepository, CacheRepository
│   ├── option.go                     # SearchOptions, SearchFilters
│   ├── errors.go                     # Repository-level errors
│   ├── qdrant/
│   │   ├── new.go                    # Factory: New(client, log, collectionName) QdrantRepository
│   │   ├── search.go                 # SearchPoints, CountByFilter
│   │   └── search_query.go           # buildSearchFilter: project_ids, sentiment, aspects, date range
│   └── redis/
│       ├── new.go                    # Factory: New(redis, log) CacheRepository
│       └── cache.go                  # GetEmbedding, SaveEmbedding, GetCampaignProjects, SaveCampaignProjects, GetSearchResults, SaveSearchResults, InvalidateSearchCache
├── usecase/
│   ├── new.go                        # Factory: implUseCase struct + New + Config
│   ├── search.go                     # Search() main method
│   └── helpers.go                    # resolveCampaignProjects, embedQuery, buildAggregations, generateCacheKey
├── interface.go                      # UseCase interface
├── types.go                          # SearchInput, SearchOutput, SearchResult, SearchFilters, Aggregation
└── errors.go                         # Domain errors
```

> **Note:** Search domain là **stateless** — không có PostgreSQL table riêng. Dữ liệu persistent nằm ở Qdrant (managed bởi indexing domain). Search chỉ **đọc** từ Qdrant và **cache** trong Redis.

---

## IV. CHI TIẾT IMPLEMENTATION

### A. TYPES (`types.go`)

```go
package search

// =====================================================
// Input Types
// =====================================================

// SearchInput - Input cho method Search
type SearchInput struct {
	CampaignID string        // Campaign scope (resolve → project_ids)
	Query      string        // Search query text
	Filters    SearchFilters // Optional filters
	Limit      int           // Max results (default 10, max 50)
	MinScore   float64       // Min relevance score (default 0.65)
}

// SearchFilters - Multi-criteria filters
type SearchFilters struct {
	Sentiments []string // Filter by overall_sentiment: POSITIVE, NEGATIVE, NEUTRAL, MIXED
	Aspects    []string // Filter by aspect name: DESIGN, BATTERY, PRICE, ...
	Platforms  []string // Filter by platform: tiktok, youtube, facebook, ...
	DateFrom   *int64   // Unix timestamp — start of date range
	DateTo     *int64   // Unix timestamp — end of date range
	RiskLevels []string // Filter by risk_level: LOW, MEDIUM, HIGH, CRITICAL
	MinEngagement *float64 // Min engagement_score
}

// =====================================================
// Output Types
// =====================================================

// SearchOutput - Kết quả search
type SearchOutput struct {
	Results            []SearchResult      // Ranked results
	TotalFound         int                 // Total matching documents
	Aggregations       Aggregations        // Aggregated stats
	NoRelevantContext  bool                // True khi không đủ relevant docs → hallucination control
	CacheHit           bool                // True nếu kết quả từ cache
	ProcessingTimeMs   int64               // Processing time
}

// SearchResult - 1 document tìm được
type SearchResult struct {
	ID                  string             // analytics_id (= Qdrant point ID)
	Score               float64            // Relevance score (0.0 - 1.0)
	Content             string             // Content text (truncated)
	ProjectID           string             // project_id
	Platform            string             // Platform name
	OverallSentiment    string             // POSITIVE | NEGATIVE | NEUTRAL | MIXED
	SentimentScore      float64            // -1.0 to 1.0
	Aspects             []AspectResult     // Aspects matched
	Keywords            []string           // Keywords
	RiskLevel           string             // LOW | MEDIUM | HIGH | CRITICAL
	EngagementScore     float64            // 0.0 to 1.0
	ContentCreatedAt    int64              // Unix timestamp
	Metadata            map[string]interface{} // Author, engagement, etc.
}

// AspectResult - Aspect trong search result
type AspectResult struct {
	Aspect            string  // Aspect name
	AspectDisplayName string  // Display name
	Sentiment         string  // Sentiment of this aspect
	SentimentScore    float64 // Score
	Keywords          []string
}

// Aggregations - Tổng hợp thống kê
type Aggregations struct {
	BySentiment  []SentimentAgg  // Phân bố sentiment
	ByAspect     []AspectAgg     // Phân bố aspects
	ByPlatform   []PlatformAgg   // Phân bố platforms
}

// SentimentAgg - Aggregation by sentiment
type SentimentAgg struct {
	Sentiment string // POSITIVE, NEGATIVE, NEUTRAL, MIXED
	Count     int
	Percentage float64
}

// AspectAgg - Aggregation by aspect
type AspectAgg struct {
	Aspect            string
	AspectDisplayName string
	Count             int
	AvgSentimentScore float64
}

// PlatformAgg - Aggregation by platform
type PlatformAgg struct {
	Platform string
	Count    int
	Percentage float64
}
```

### B. INTERFACE (`interface.go`)

```go
package search

import (
	"context"
	"knowledge-srv/internal/model"
)

// UseCase - Interface chính của domain search
// Domain này là nền tảng, được inject vào chat.UseCase và report.UseCase
type UseCase interface {
	// Search - Semantic search trong Qdrant với campaign scope + filters
	// Flow: resolve campaign → embed query → search Qdrant → aggregate → cache → return
	Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)
}
```

### C. ERRORS (`errors.go`)

```go
package search

import "errors"

// Domain errors
var (
	// ErrCampaignNotFound - Campaign không tồn tại hoặc user không có quyền
	ErrCampaignNotFound = errors.New("search: campaign not found")

	// ErrCampaignNoProjects - Campaign không có project nào
	ErrCampaignNoProjects = errors.New("search: campaign has no projects")

	// ErrQueryTooShort - Query text quá ngắn (< 3 chars)
	ErrQueryTooShort = errors.New("search: query too short")

	// ErrQueryTooLong - Query text quá dài (> 1000 chars)
	ErrQueryTooLong = errors.New("search: query too long")

	// ErrEmbeddingFailed - Không sinh được embedding cho query
	ErrEmbeddingFailed = errors.New("search: embedding generation failed")

	// ErrSearchFailed - Qdrant search thất bại
	ErrSearchFailed = errors.New("search: qdrant search failed")

	// ErrInvalidFilters - Filters không hợp lệ
	ErrInvalidFilters = errors.New("search: invalid filters")
)
```

### D. REPOSITORY INTERFACE (`repository/interface.go`)

```go
package repository

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
	"knowledge-srv/pkg/qdrant"
)

// QdrantRepository - Qdrant search operations
// Collection name là const trong implementation, không truyền từ usecase
type QdrantRepository interface {
	// SearchPoints - Vector search với filter
	SearchPoints(ctx context.Context, opt SearchPointsOptions) ([]qdrant.SearchResult, error)

	// CountByFilter - Đếm points theo filter (cho aggregation)
	CountByFilter(ctx context.Context, filter *pb.Filter) (uint64, error)
}

// CacheRepository - Redis cache operations (3 tầng)
type CacheRepository interface {
	// Tầng 1: Embedding Cache
	GetEmbedding(ctx context.Context, contentHash string) ([]float32, error)
	SaveEmbedding(ctx context.Context, contentHash string, vector []float32) error

	// Tầng 2: Campaign Projects Cache
	GetCampaignProjects(ctx context.Context, campaignID string) ([]string, error)
	SaveCampaignProjects(ctx context.Context, campaignID string, projectIDs []string) error

	// Tầng 3: Search Results Cache
	GetSearchResults(ctx context.Context, cacheKey string) ([]byte, error)
	SaveSearchResults(ctx context.Context, cacheKey string, data []byte) error

	// Invalidation
	InvalidateSearchCache(ctx context.Context, projectID string) error
}
```

### E. REPOSITORY OPTIONS (`repository/option.go`)

```go
package repository

import pb "github.com/qdrant/go-client/qdrant"

// SearchPointsOptions - Options cho SearchPoints
type SearchPointsOptions struct {
	Vector       []float32   // Query vector
	Limit        uint64      // Max results
	Filter       *pb.Filter  // Qdrant payload filter
	ScoreThreshold *float32  // Min score threshold (optional)
	WithPayload  []string    // Fields to include in payload (selective retrieval)
}
```

### F. REPOSITORY ERRORS (`repository/errors.go`)

```go
package repository

import "errors"

var (
	ErrFailedToSearch     = errors.New("repository: failed to search points")
	ErrFailedToCount      = errors.New("repository: failed to count points")
	ErrCacheMiss          = errors.New("repository: cache miss")
	ErrCacheSetFailed     = errors.New("repository: failed to set cache")
	ErrCacheDeleteFailed  = errors.New("repository: failed to delete cache")
)
```

---

## V. REPOSITORY IMPLEMENTATION

### A. QDRANT REPOSITORY (`repository/qdrant/new.go`)

```go
package qdrant

import (
	"knowledge-srv/internal/search/repository"
	pkgQdrant "knowledge-srv/pkg/qdrant"
	"knowledge-srv/pkg/log"
)

const collectionName = "smap_analytics"

// Payload fields cho search (selective retrieval — P3.2 Optimization)
var searchPayloadFields = []string{
	"content", "project_id", "platform",
	"overall_sentiment", "overall_sentiment_score",
	"aspects", "keywords",
	"risk_level", "risk_score",
	"engagement_score", "content_created_at",
	"metadata",
}

type implQdrantRepository struct {
	client pkgQdrant.IQdrant
	l      log.Logger
}

// New - Factory
func New(client pkgQdrant.IQdrant, l log.Logger) repository.QdrantRepository {
	return &implQdrantRepository{
		client: client,
		l:      l,
	}
}
```

### B. QDRANT SEARCH (`repository/qdrant/search.go`)

```go
package qdrant

import (
	"context"
	"fmt"

	"knowledge-srv/internal/search/repository"
	pkgQdrant "knowledge-srv/pkg/qdrant"
)

// SearchPoints - Thực hiện vector search với filter
func (r *implQdrantRepository) SearchPoints(ctx context.Context, opt repository.SearchPointsOptions) ([]pkgQdrant.SearchResult, error) {
	results, err := r.client.SearchWithFilter(ctx, collectionName, opt.Vector, opt.Limit, opt.Filter)
	if err != nil {
		r.l.Errorf(ctx, "search.repository.qdrant.SearchPoints: %v", err)
		return nil, fmt.Errorf("%w: %v", repository.ErrFailedToSearch, err)
	}
	return results, nil
}

// CountByFilter - Đếm points theo filter
func (r *implQdrantRepository) CountByFilter(ctx context.Context, filter *pb.Filter) (uint64, error) {
	count, err := r.client.CountPoints(ctx, collectionName)
	if err != nil {
		r.l.Errorf(ctx, "search.repository.qdrant.CountByFilter: %v", err)
		return 0, fmt.Errorf("%w: %v", repository.ErrFailedToCount, err)
	}
	return count, nil
}
```

### C. QDRANT QUERY BUILDER (`repository/qdrant/search_query.go`)

```go
package qdrant

import (
	pb "github.com/qdrant/go-client/qdrant"
)

// BuildProjectFilter - Filter by project_ids (IN clause)
func BuildProjectFilter(projectIDs []string) *pb.Condition {
	values := make([]*pb.Value, len(projectIDs))
	for i, id := range projectIDs {
		values[i] = &pb.Value{Kind: &pb.Value_StringValue{StringValue: id}}
	}
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_Field{
			Field: &pb.FieldCondition{
				Key: "project_id",
				Match: &pb.Match{
					MatchValue: &pb.Match_Keywords{
						Keywords: &pb.RepeatedStrings{Strings: projectIDs},
					},
				},
			},
		},
	}
}

// BuildSentimentFilter - Filter by sentiment values
func BuildSentimentFilter(sentiments []string) *pb.Condition {
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_Field{
			Field: &pb.FieldCondition{
				Key: "overall_sentiment",
				Match: &pb.Match{
					MatchValue: &pb.Match_Keywords{
						Keywords: &pb.RepeatedStrings{Strings: sentiments},
					},
				},
			},
		},
	}
}

// BuildPlatformFilter - Filter by platforms
func BuildPlatformFilter(platforms []string) *pb.Condition {
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_Field{
			Field: &pb.FieldCondition{
				Key: "platform",
				Match: &pb.Match{
					MatchValue: &pb.Match_Keywords{
						Keywords: &pb.RepeatedStrings{Strings: platforms},
					},
				},
			},
		},
	}
}

// BuildDateRangeFilter - Filter by date range (content_created_at)
func BuildDateRangeFilter(from, to *int64) []*pb.Condition {
	var conditions []*pb.Condition
	if from != nil {
		fromFloat := float64(*from)
		conditions = append(conditions, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "content_created_at",
					Range: &pb.Range{Gte: &fromFloat},
				},
			},
		})
	}
	if to != nil {
		toFloat := float64(*to)
		conditions = append(conditions, &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key: "content_created_at",
					Range: &pb.Range{Lte: &toFloat},
				},
			},
		})
	}
	return conditions
}

// BuildRiskLevelFilter - Filter by risk levels
func BuildRiskLevelFilter(riskLevels []string) *pb.Condition {
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_Field{
			Field: &pb.FieldCondition{
				Key: "risk_level",
				Match: &pb.Match{
					MatchValue: &pb.Match_Keywords{
						Keywords: &pb.RepeatedStrings{Strings: riskLevels},
					},
				},
			},
		},
	}
}

// BuildSearchFilter - Compose all filters into a single Qdrant filter
func BuildSearchFilter(projectIDs []string, filters SearchFilterInput) *pb.Filter {
	var must []*pb.Condition

	// Required: project_ids filter (always present)
	must = append(must, BuildProjectFilter(projectIDs))

	// Optional filters
	if len(filters.Sentiments) > 0 {
		must = append(must, BuildSentimentFilter(filters.Sentiments))
	}
	if len(filters.Platforms) > 0 {
		must = append(must, BuildPlatformFilter(filters.Platforms))
	}
	if filters.DateFrom != nil || filters.DateTo != nil {
		must = append(must, BuildDateRangeFilter(filters.DateFrom, filters.DateTo)...)
	}
	if len(filters.RiskLevels) > 0 {
		must = append(must, BuildRiskLevelFilter(filters.RiskLevels))
	}

	return &pb.Filter{Must: must}
}

// SearchFilterInput - Internal struct for query builder
type SearchFilterInput struct {
	Sentiments []string
	Platforms  []string
	DateFrom   *int64
	DateTo     *int64
	RiskLevels []string
}
```

### D. REDIS CACHE REPOSITORY (`repository/redis/new.go`)

```go
package redis

import (
	"knowledge-srv/internal/search/repository"
	pkgRedis "knowledge-srv/pkg/redis"
	"knowledge-srv/pkg/log"
)

type implCacheRepository struct {
	redis pkgRedis.IRedis
	l     log.Logger
}

// New - Factory
func New(redis pkgRedis.IRedis, l log.Logger) repository.CacheRepository {
	return &implCacheRepository{
		redis: redis,
		l:     l,
	}
}
```

### E. REDIS CACHE IMPLEMENTATION (`repository/redis/cache.go`)

```go
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// =====================================================
// Tầng 1: Embedding Cache (TTL 7 days)
// =====================================================

func (r *implCacheRepository) GetEmbedding(ctx context.Context, contentHash string) ([]float32, error) {
	key := fmt.Sprintf("embedding:%s", contentHash)
	data, err := r.redis.GetClient().Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var vector []float32
	if err := json.Unmarshal([]byte(data), &vector); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.GetEmbedding: unmarshal error: %v", err)
		return nil, err
	}
	return vector, nil
}

func (r *implCacheRepository) SaveEmbedding(ctx context.Context, contentHash string, vector []float32) error {
	key := fmt.Sprintf("embedding:%s", contentHash)
	data, err := json.Marshal(vector)
	if err != nil {
		return err
	}
	if err := r.redis.GetClient().Set(ctx, key, data, 7*24*time.Hour).Err(); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.SaveEmbedding: %v", err)
		return err
	}
	return nil
}

// =====================================================
// Tầng 2: Campaign Projects Cache (TTL 10 min)
// =====================================================

func (r *implCacheRepository) GetCampaignProjects(ctx context.Context, campaignID string) ([]string, error) {
	key := fmt.Sprintf("campaign_projects:%s", campaignID)
	data, err := r.redis.GetClient().Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var projectIDs []string
	if err := json.Unmarshal([]byte(data), &projectIDs); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.GetCampaignProjects: unmarshal error: %v", err)
		return nil, err
	}
	return projectIDs, nil
}

func (r *implCacheRepository) SaveCampaignProjects(ctx context.Context, campaignID string, projectIDs []string) error {
	key := fmt.Sprintf("campaign_projects:%s", campaignID)
	data, err := json.Marshal(projectIDs)
	if err != nil {
		return err
	}
	if err := r.redis.GetClient().Set(ctx, key, data, 10*time.Minute).Err(); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.SaveCampaignProjects: %v", err)
		return err
	}
	return nil
}

// =====================================================
// Tầng 3: Search Results Cache (TTL 5 min)
// =====================================================

func (r *implCacheRepository) GetSearchResults(ctx context.Context, cacheKey string) ([]byte, error) {
	data, err := r.redis.GetClient().Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}
	return []byte(data), nil
}

func (r *implCacheRepository) SaveSearchResults(ctx context.Context, cacheKey string, data []byte) error {
	if err := r.redis.GetClient().Set(ctx, cacheKey, data, 5*time.Minute).Err(); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.SaveSearchResults: %v", err)
		return err
	}
	return nil
}

// =====================================================
// Cache Invalidation
// =====================================================

func (r *implCacheRepository) InvalidateSearchCache(ctx context.Context, projectID string) error {
	pattern := fmt.Sprintf("search:*%s*", projectID)
	client := r.redis.GetClient()

	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			r.l.Errorf(ctx, "search.repository.redis.InvalidateSearchCache: scan error: %v", err)
			return err
		}
		if len(keys) > 0 {
			pipe := client.Pipeline()
			for _, key := range keys {
				pipe.Del(ctx, key)
			}
			if _, err := pipe.Exec(ctx); err != nil && err != goredis.Nil {
				r.l.Errorf(ctx, "search.repository.redis.InvalidateSearchCache: pipeline error: %v", err)
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
```

---

## VI. USECASE IMPLEMENTATION

### A. USECASE FACTORY (`usecase/new.go`)

```go
package usecase

import (
	"knowledge-srv/internal/search"
	"knowledge-srv/internal/search/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/projectsrv"
	"knowledge-srv/pkg/voyage"
)

// Config - Cấu hình UseCase
type Config struct {
	MinScore       float64 // Min relevance score threshold (default 0.65)
	MaxResults     int     // Max results per query (default 10, max 50)
	MinQueryLength int     // Min query length in chars (default 3)
	MaxQueryLength int     // Max query length in chars (default 1000)
}

// DefaultConfig - Cấu hình mặc định
func DefaultConfig() Config {
	return Config{
		MinScore:       0.65,
		MaxResults:     10,
		MinQueryLength: 3,
		MaxQueryLength: 1000,
	}
}

// implUseCase - Implementation của UseCase interface
type implUseCase struct {
	qdrantRepo  repository.QdrantRepository
	cacheRepo   repository.CacheRepository
	voyage      voyage.IVoyage
	projectSrv  projectsrv.IProject
	l           log.Logger
	cfg         Config
}

// New - Factory function
func New(
	qdrantRepo repository.QdrantRepository,
	cacheRepo repository.CacheRepository,
	voyage voyage.IVoyage,
	projectSrv projectsrv.IProject,
	l log.Logger,
	cfg Config,
) search.UseCase {
	return &implUseCase{
		qdrantRepo: qdrantRepo,
		cacheRepo:  cacheRepo,
		voyage:     voyage,
		projectSrv: projectSrv,
		l:          l,
		cfg:        cfg,
	}
}
```

### B. USECASE MAIN LOGIC (`usecase/search.go`)

```go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/search"
	"knowledge-srv/internal/search/repository"
	qdrantRepo "knowledge-srv/internal/search/repository/qdrant"
)

// Search - Main search method
// Flow: check cache → resolve campaign → embed query → search Qdrant → filter by score → aggregate → cache → return
func (uc *implUseCase) Search(ctx context.Context, sc model.Scope, input search.SearchInput) (search.SearchOutput, error) {
	startTime := time.Now()

	// Step 0: Validate input
	if err := uc.validateInput(input); err != nil {
		return search.SearchOutput{}, err
	}

	// Apply defaults
	limit := input.Limit
	if limit <= 0 {
		limit = uc.cfg.MaxResults
	}
	if limit > 50 {
		limit = 50
	}
	minScore := input.MinScore
	if minScore <= 0 {
		minScore = uc.cfg.MinScore
	}

	// Step 1: Check Tầng 3 — Search Results Cache
	cacheKey := uc.generateCacheKey(input)
	cachedData, err := uc.cacheRepo.GetSearchResults(ctx, cacheKey)
	if err == nil && cachedData != nil {
		var cached search.SearchOutput
		if err := json.Unmarshal(cachedData, &cached); err == nil {
			cached.CacheHit = true
			cached.ProcessingTimeMs = time.Since(startTime).Milliseconds()
			uc.l.Debugf(ctx, "search.usecase.Search: cache hit")
			return cached, nil
		}
	}

	// Step 2: Resolve campaign → project_ids (Tầng 2 cache)
	projectIDs, err := uc.resolveCampaignProjects(ctx, input.CampaignID)
	if err != nil {
		return search.SearchOutput{}, err
	}

	// Step 3: Embed query (Tầng 1 cache)
	vector, err := uc.embedQuery(ctx, input.Query)
	if err != nil {
		return search.SearchOutput{}, err
	}

	// Step 4: Build Qdrant filter
	filterInput := qdrantRepo.SearchFilterInput{
		Sentiments: input.Filters.Sentiments,
		Platforms:  input.Filters.Platforms,
		DateFrom:   input.Filters.DateFrom,
		DateTo:     input.Filters.DateTo,
		RiskLevels: input.Filters.RiskLevels,
	}
	filter := qdrantRepo.BuildSearchFilter(projectIDs, filterInput)

	// Step 5: Search Qdrant
	scoreThreshold := float32(minScore)
	searchOpts := repository.SearchPointsOptions{
		Vector:         vector,
		Limit:          uint64(limit),
		Filter:         filter,
		ScoreThreshold: &scoreThreshold,
	}

	qdrantResults, err := uc.qdrantRepo.SearchPoints(ctx, searchOpts)
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.Search: Qdrant search failed: %v", err)
		return search.SearchOutput{}, fmt.Errorf("%w: %v", search.ErrSearchFailed, err)
	}

	// Step 6: Filter by min score + map to domain results
	var results []search.SearchResult
	for _, r := range qdrantResults {
		if float64(r.Score) < minScore {
			continue
		}
		results = append(results, uc.mapQdrantResult(r))
	}

	// Step 7: Hallucination control — NO relevant context flag
	noRelevantContext := len(results) == 0

	// Step 8: Build aggregations
	aggregations := uc.buildAggregations(results)

	// Step 9: Build output
	output := search.SearchOutput{
		Results:           results,
		TotalFound:        len(results),
		Aggregations:      aggregations,
		NoRelevantContext: noRelevantContext,
		CacheHit:          false,
		ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
	}

	// Step 10: Cache results (Tầng 3)
	if data, err := json.Marshal(output); err == nil {
		if err := uc.cacheRepo.SaveSearchResults(ctx, cacheKey, data); err != nil {
			uc.l.Warnf(ctx, "search.usecase.Search: Failed to cache results: %v", err)
		}
	}

	uc.l.Infof(ctx, "search.usecase.Search: query=%q, results=%d, no_context=%v, duration=%dms",
		input.Query, len(results), noRelevantContext, output.ProcessingTimeMs)

	return output, nil
}

// validateInput - Validate search input
func (uc *implUseCase) validateInput(input search.SearchInput) error {
	if input.CampaignID == "" {
		return search.ErrCampaignNotFound
	}
	if len(input.Query) < uc.cfg.MinQueryLength {
		return search.ErrQueryTooShort
	}
	if len(input.Query) > uc.cfg.MaxQueryLength {
		return search.ErrQueryTooLong
	}
	return nil
}
```

### C. USECASE HELPERS (`usecase/helpers.go`)

```go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"knowledge-srv/internal/search"
	pkgQdrant "knowledge-srv/pkg/qdrant"
)

// resolveCampaignProjects - Resolve campaign_id → project_ids (Tầng 2 cache)
func (uc *implUseCase) resolveCampaignProjects(ctx context.Context, campaignID string) ([]string, error) {
	// Check cache
	projectIDs, err := uc.cacheRepo.GetCampaignProjects(ctx, campaignID)
	if err == nil && len(projectIDs) > 0 {
		uc.l.Debugf(ctx, "search.usecase.resolveCampaignProjects: cache hit, %d projects", len(projectIDs))
		return projectIDs, nil
	}

	// Cache miss → call Project Service
	campaign, err := uc.projectSrv.GetCampaign(ctx, campaignID)
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.resolveCampaignProjects: GetCampaign failed: %v", err)
		return nil, fmt.Errorf("%w: %v", search.ErrCampaignNotFound, err)
	}

	if len(campaign.ProjectIDs) == 0 {
		return nil, search.ErrCampaignNoProjects
	}

	// Save to cache
	if err := uc.cacheRepo.SaveCampaignProjects(ctx, campaignID, campaign.ProjectIDs); err != nil {
		uc.l.Warnf(ctx, "search.usecase.resolveCampaignProjects: cache save failed: %v", err)
	}

	return campaign.ProjectIDs, nil
}

// embedQuery - Embed query text (Tầng 1 cache)
func (uc *implUseCase) embedQuery(ctx context.Context, query string) ([]float32, error) {
	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(query)))

	// Check cache
	cachedVector, err := uc.cacheRepo.GetEmbedding(ctx, contentHash)
	if err == nil && cachedVector != nil {
		uc.l.Debugf(ctx, "search.usecase.embedQuery: cache hit")
		return cachedVector, nil
	}

	// Cache miss → call Voyage API
	vectors, err := uc.voyage.Embed(ctx, []string{query})
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.embedQuery: Voyage embed failed: %v", err)
		return nil, fmt.Errorf("%w: %v", search.ErrEmbeddingFailed, err)
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("%w: empty vector returned", search.ErrEmbeddingFailed)
	}

	vector := vectors[0]

	// Save to cache
	if err := uc.cacheRepo.SaveEmbedding(ctx, contentHash, vector); err != nil {
		uc.l.Warnf(ctx, "search.usecase.embedQuery: cache save failed: %v", err)
	}

	return vector, nil
}

// generateCacheKey - Generate Tầng 3 cache key
func (uc *implUseCase) generateCacheKey(input search.SearchInput) string {
	filterJSON, _ := json.Marshal(input.Filters)
	raw := fmt.Sprintf("%s:%s:%s:%d:%.2f", input.CampaignID, input.Query, string(filterJSON), input.Limit, input.MinScore)
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("search:%s:%x", input.CampaignID, hash)
}

// mapQdrantResult - Map Qdrant SearchResult → domain SearchResult
func (uc *implUseCase) mapQdrantResult(r pkgQdrant.SearchResult) search.SearchResult {
	result := search.SearchResult{
		ID:       r.ID,
		Score:    float64(r.Score),
		Metadata: r.Payload,
	}

	// Extract typed fields from payload
	if v, ok := r.Payload["content"].(string); ok {
		result.Content = v
	}
	if v, ok := r.Payload["project_id"].(string); ok {
		result.ProjectID = v
	}
	if v, ok := r.Payload["platform"].(string); ok {
		result.Platform = v
	}
	if v, ok := r.Payload["overall_sentiment"].(string); ok {
		result.OverallSentiment = v
	}
	if v, ok := r.Payload["overall_sentiment_score"].(float64); ok {
		result.SentimentScore = v
	}
	if v, ok := r.Payload["risk_level"].(string); ok {
		result.RiskLevel = v
	}
	if v, ok := r.Payload["engagement_score"].(float64); ok {
		result.EngagementScore = v
	}
	if v, ok := r.Payload["content_created_at"].(float64); ok {
		result.ContentCreatedAt = int64(v)
	}
	if v, ok := r.Payload["keywords"].([]interface{}); ok {
		for _, k := range v {
			if s, ok := k.(string); ok {
				result.Keywords = append(result.Keywords, s)
			}
		}
	}
	// Aspects extraction
	if v, ok := r.Payload["aspects"].([]interface{}); ok {
		for _, a := range v {
			if m, ok := a.(map[string]interface{}); ok {
				aspect := search.AspectResult{}
				if s, ok := m["aspect"].(string); ok {
					aspect.Aspect = s
				}
				if s, ok := m["aspect_display_name"].(string); ok {
					aspect.AspectDisplayName = s
				}
				if s, ok := m["sentiment"].(string); ok {
					aspect.Sentiment = s
				}
				if f, ok := m["sentiment_score"].(float64); ok {
					aspect.SentimentScore = f
				}
				result.Aspects = append(result.Aspects, aspect)
			}
		}
	}

	return result
}

// buildAggregations - Tổng hợp thống kê từ results
func (uc *implUseCase) buildAggregations(results []search.SearchResult) search.Aggregations {
	total := len(results)
	if total == 0 {
		return search.Aggregations{}
	}

	// Sentiment aggregation
	sentimentCounts := make(map[string]int)
	for _, r := range results {
		if r.OverallSentiment != "" {
			sentimentCounts[r.OverallSentiment]++
		}
	}
	var bySentiment []search.SentimentAgg
	for s, c := range sentimentCounts {
		bySentiment = append(bySentiment, search.SentimentAgg{
			Sentiment:  s,
			Count:      c,
			Percentage: float64(c) / float64(total) * 100,
		})
	}

	// Platform aggregation
	platformCounts := make(map[string]int)
	for _, r := range results {
		if r.Platform != "" {
			platformCounts[r.Platform]++
		}
	}
	var byPlatform []search.PlatformAgg
	for p, c := range platformCounts {
		byPlatform = append(byPlatform, search.PlatformAgg{
			Platform:   p,
			Count:      c,
			Percentage: float64(c) / float64(total) * 100,
		})
	}

	// Aspect aggregation
	aspectData := make(map[string]struct {
		DisplayName string
		Count       int
		TotalScore  float64
	})
	for _, r := range results {
		for _, a := range r.Aspects {
			d := aspectData[a.Aspect]
			d.DisplayName = a.AspectDisplayName
			d.Count++
			d.TotalScore += a.SentimentScore
			aspectData[a.Aspect] = d
		}
	}
	var byAspect []search.AspectAgg
	for name, d := range aspectData {
		byAspect = append(byAspect, search.AspectAgg{
			Aspect:            name,
			AspectDisplayName: d.DisplayName,
			Count:             d.Count,
			AvgSentimentScore: d.TotalScore / float64(d.Count),
		})
	}

	return search.Aggregations{
		BySentiment: bySentiment,
		ByAspect:    byAspect,
		ByPlatform:  byPlatform,
	}
}
```

---

## VII. DELIVERY LAYER

### A. HTTP HANDLER (`delivery/http/handlers.go`)

```go
package http

import (
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

type handler struct {
	l  log.Logger
	uc search.UseCase
}

// Search - Handler cho POST /api/v1/search
func (h *handler) Search(c *gin.Context) {
	ctx := c.Request.Context()

	// Process request
	req, sc, err := h.processSearchRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "processSearchRequest failed: %v", err)
		response.Error(c, err)
		return
	}

	// Convert to UseCase input
	input := req.toInput()

	// Call UseCase
	output, err := h.uc.Search(ctx, sc, input)
	if err != nil {
		h.l.Errorf(ctx, "Search failed: %v", err)
		response.Error(c, h.mapError(err))
		return
	}

	// Return response
	resp := h.newSearchResp(output)
	response.Success(c, resp)
}
```

### B. HTTP PRESENTERS (`delivery/http/presenters.go`)

```go
package http

import "knowledge-srv/internal/search"

// =====================================================
// Request DTOs
// =====================================================

type searchReq struct {
	CampaignID string           `json:"campaign_id" binding:"required"`
	Query      string           `json:"query" binding:"required,min=3,max=1000"`
	Filters    *searchFilterReq `json:"filters,omitempty"`
	Limit      int              `json:"limit,omitempty"`
	MinScore   float64          `json:"min_score,omitempty"`
}

type searchFilterReq struct {
	Sentiments    []string `json:"sentiments,omitempty"`
	Aspects       []string `json:"aspects,omitempty"`
	Platforms     []string `json:"platforms,omitempty"`
	DateFrom      *int64   `json:"date_from,omitempty"`
	DateTo        *int64   `json:"date_to,omitempty"`
	RiskLevels    []string `json:"risk_levels,omitempty"`
	MinEngagement *float64 `json:"min_engagement,omitempty"`
}

func (r searchReq) toInput() search.SearchInput {
	input := search.SearchInput{
		CampaignID: r.CampaignID,
		Query:      r.Query,
		Limit:      r.Limit,
		MinScore:   r.MinScore,
	}
	if r.Filters != nil {
		input.Filters = search.SearchFilters{
			Sentiments:    r.Filters.Sentiments,
			Aspects:       r.Filters.Aspects,
			Platforms:     r.Filters.Platforms,
			DateFrom:      r.Filters.DateFrom,
			DateTo:        r.Filters.DateTo,
			RiskLevels:    r.Filters.RiskLevels,
			MinEngagement: r.Filters.MinEngagement,
		}
	}
	return input
}

// =====================================================
// Response DTOs
// =====================================================

type searchResp struct {
	Results           []searchResultResp `json:"results"`
	TotalFound        int                `json:"total_found"`
	Aggregations      aggregationsResp   `json:"aggregations"`
	NoRelevantContext bool               `json:"no_relevant_context"`
	CacheHit          bool               `json:"cache_hit"`
	ProcessingTimeMs  int64              `json:"processing_time_ms"`
}

type searchResultResp struct {
	ID               string             `json:"id"`
	Score            float64            `json:"score"`
	Content          string             `json:"content"`
	ProjectID        string             `json:"project_id"`
	Platform         string             `json:"platform"`
	OverallSentiment string             `json:"overall_sentiment"`
	SentimentScore   float64            `json:"sentiment_score"`
	Aspects          []aspectResultResp `json:"aspects,omitempty"`
	Keywords         []string           `json:"keywords,omitempty"`
	RiskLevel        string             `json:"risk_level"`
	EngagementScore  float64            `json:"engagement_score"`
	ContentCreatedAt int64              `json:"content_created_at"`
}

type aspectResultResp struct {
	Aspect            string  `json:"aspect"`
	AspectDisplayName string  `json:"aspect_display_name"`
	Sentiment         string  `json:"sentiment"`
	SentimentScore    float64 `json:"sentiment_score"`
}

type aggregationsResp struct {
	BySentiment []sentimentAggResp `json:"by_sentiment"`
	ByAspect    []aspectAggResp    `json:"by_aspect"`
	ByPlatform  []platformAggResp  `json:"by_platform"`
}

type sentimentAggResp struct {
	Sentiment  string  `json:"sentiment"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type aspectAggResp struct {
	Aspect            string  `json:"aspect"`
	AspectDisplayName string  `json:"aspect_display_name"`
	Count             int     `json:"count"`
	AvgSentimentScore float64 `json:"avg_sentiment_score"`
}

type platformAggResp struct {
	Platform   string  `json:"platform"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

func (h *handler) newSearchResp(output search.SearchOutput) searchResp {
	resp := searchResp{
		TotalFound:        output.TotalFound,
		NoRelevantContext: output.NoRelevantContext,
		CacheHit:          output.CacheHit,
		ProcessingTimeMs:  output.ProcessingTimeMs,
	}

	// Map results
	resp.Results = make([]searchResultResp, len(output.Results))
	for i, r := range output.Results {
		result := searchResultResp{
			ID:               r.ID,
			Score:            r.Score,
			Content:          r.Content,
			ProjectID:        r.ProjectID,
			Platform:         r.Platform,
			OverallSentiment: r.OverallSentiment,
			SentimentScore:   r.SentimentScore,
			RiskLevel:        r.RiskLevel,
			EngagementScore:  r.EngagementScore,
			ContentCreatedAt: r.ContentCreatedAt,
			Keywords:         r.Keywords,
		}
		for _, a := range r.Aspects {
			result.Aspects = append(result.Aspects, aspectResultResp{
				Aspect:            a.Aspect,
				AspectDisplayName: a.AspectDisplayName,
				Sentiment:         a.Sentiment,
				SentimentScore:    a.SentimentScore,
			})
		}
		resp.Results[i] = result
	}

	// Map aggregations
	resp.Aggregations.BySentiment = make([]sentimentAggResp, len(output.Aggregations.BySentiment))
	for i, s := range output.Aggregations.BySentiment {
		resp.Aggregations.BySentiment[i] = sentimentAggResp{
			Sentiment: s.Sentiment, Count: s.Count, Percentage: s.Percentage,
		}
	}
	resp.Aggregations.ByAspect = make([]aspectAggResp, len(output.Aggregations.ByAspect))
	for i, a := range output.Aggregations.ByAspect {
		resp.Aggregations.ByAspect[i] = aspectAggResp{
			Aspect: a.Aspect, AspectDisplayName: a.AspectDisplayName,
			Count: a.Count, AvgSentimentScore: a.AvgSentimentScore,
		}
	}
	resp.Aggregations.ByPlatform = make([]platformAggResp, len(output.Aggregations.ByPlatform))
	for i, p := range output.Aggregations.ByPlatform {
		resp.Aggregations.ByPlatform[i] = platformAggResp{
			Platform: p.Platform, Count: p.Count, Percentage: p.Percentage,
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
	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/scope"
)

func (h *handler) processSearchRequest(c *gin.Context) (searchReq, model.Scope, error) {
	var req searchReq

	if err := c.ShouldBindJSON(&req); err != nil {
		return req, model.Scope{}, err
	}

	sc := scope.Extract(c)
	return req, sc, nil
}
```

### D. HTTP ERROR MAPPING (`delivery/http/errors.go`)

```go
package http

import (
	"errors"
	"knowledge-srv/internal/search"
	pkgErrors "knowledge-srv/pkg/errors"
)

var (
	errCampaignNotFound = pkgErrors.NewHTTPError(
		pkgErrors.CodeNotFound, "Campaign not found",
	)
	errCampaignNoProjects = pkgErrors.NewHTTPError(
		pkgErrors.CodeBadRequest, "Campaign has no projects",
	)
	errQueryTooShort = pkgErrors.NewHTTPError(
		pkgErrors.CodeBadRequest, "Query too short (min 3 characters)",
	)
	errQueryTooLong = pkgErrors.NewHTTPError(
		pkgErrors.CodeBadRequest, "Query too long (max 1000 characters)",
	)
	errEmbeddingFailed = pkgErrors.NewHTTPError(
		pkgErrors.CodeInternalError, "Failed to generate query embedding",
	)
	errSearchFailed = pkgErrors.NewHTTPError(
		pkgErrors.CodeInternalError, "Search failed",
	)
	errInvalidFilters = pkgErrors.NewHTTPError(
		pkgErrors.CodeBadRequest, "Invalid search filters",
	)
)

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, search.ErrCampaignNotFound):
		return errCampaignNotFound
	case errors.Is(err, search.ErrCampaignNoProjects):
		return errCampaignNoProjects
	case errors.Is(err, search.ErrQueryTooShort):
		return errQueryTooShort
	case errors.Is(err, search.ErrQueryTooLong):
		return errQueryTooLong
	case errors.Is(err, search.ErrEmbeddingFailed):
		return errEmbeddingFailed
	case errors.Is(err, search.ErrSearchFailed):
		return errSearchFailed
	case errors.Is(err, search.ErrInvalidFilters):
		return errInvalidFilters
	default:
		panic(err)
	}
}
```

### E. HTTP ROUTES (`delivery/http/routes.go`)

```go
package http

import (
	"github.com/gin-gonic/gin"
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/middleware"
)

type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

func New(l log.Logger, uc search.UseCase) Handler {
	return &handler{l: l, uc: uc}
}

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	api := r.Group("/api/v1")
	api.Use(mw.JWTAuth())
	{
		api.POST("/search", h.Search)
	}
}
```

---

## VIII. TESTING STRATEGY

### A. Unit Tests

**Target:** UseCase logic

**Files:**

- `usecase/search_test.go`
- `usecase/helpers_test.go`

**Mocking:** QdrantRepository, CacheRepository, IVoyage, IProject

**Test cases:**

```go
// search_test.go
func TestSearch_FullCacheHit(t *testing.T)         // Tầng 3 cache hit
func TestSearch_PartialCacheHit(t *testing.T)      // Tầng 1+2 hit, Tầng 3 miss
func TestSearch_FullCacheMiss(t *testing.T)         // All miss → full pipeline
func TestSearch_NoResults(t *testing.T)             // NoRelevantContext = true
func TestSearch_ScoreFiltering(t *testing.T)        // Min score threshold
func TestSearch_CampaignNotFound(t *testing.T)
func TestSearch_QueryTooShort(t *testing.T)
func TestSearch_QueryTooLong(t *testing.T)
func TestSearch_EmbeddingFailed(t *testing.T)
func TestSearch_QdrantFailed(t *testing.T)

// helpers_test.go
func TestResolveCampaignProjects_CacheHit(t *testing.T)
func TestResolveCampaignProjects_CacheMiss(t *testing.T)
func TestEmbedQuery_CacheHit(t *testing.T)
func TestEmbedQuery_CacheMiss(t *testing.T)
func TestGenerateCacheKey_Deterministic(t *testing.T)
func TestBuildAggregations(t *testing.T)
func TestMapQdrantResult(t *testing.T)
```

### B. Integration Tests

**Target:** Repository + Qdrant + Redis

**Files:**

- `repository/qdrant/search_test.go`
- `repository/redis/cache_test.go`

**Test cases:**

```go
// qdrant/search_test.go
func TestSearchPoints_WithFilter(t *testing.T)
func TestSearchPoints_NoFilter(t *testing.T)
func TestSearchPoints_EmptyResults(t *testing.T)
func TestCountByFilter(t *testing.T)

// redis/cache_test.go
func TestEmbeddingCache_RoundTrip(t *testing.T)
func TestCampaignProjectsCache_RoundTrip(t *testing.T)
func TestSearchResultsCache_RoundTrip(t *testing.T)
func TestSearchResultsCache_TTLExpiry(t *testing.T)
func TestInvalidateSearchCache(t *testing.T)
```

### C. End-to-End Tests

**Target:** Full flow HTTP → UseCase → Qdrant

**Files:**

- `e2e/search_test.go`

**Test cases:**

```go
func TestE2E_Search_FullPipeline(t *testing.T)
func TestE2E_Search_WithFilters(t *testing.T)
func TestE2E_Search_CacheInvalidation(t *testing.T)
func TestE2E_Search_HallucinationControl(t *testing.T)
```

---

## IX. MONITORING & OBSERVABILITY

### A. Metrics (Prometheus)

```go
var (
	searchTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "knowledge_search_total",
			Help: "Total number of search operations",
		},
		[]string{"status", "cache_hit"}, // status: success/error, cache_hit: true/false
	)

	searchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "knowledge_search_duration_seconds",
			Help:    "Duration of search operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"step"}, // embed, resolve_campaign, qdrant, aggregate, total
	)

	searchResultCount = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "knowledge_search_result_count",
			Help:    "Number of results returned per search",
			Buckets: []float64{0, 1, 5, 10, 20, 50},
		},
	)

	cacheHitRate = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "knowledge_cache_hits_total",
			Help: "Cache hit/miss by tier",
		},
		[]string{"tier", "result"}, // tier: embedding/campaign/search, result: hit/miss
	)
)
```

### B. Logging

| Level     | When                                             |
| --------- | ------------------------------------------------ |
| **INFO**  | Search completed (query, result count, duration) |
| **WARN**  | Cache save failure, no relevant context          |
| **ERROR** | Qdrant/Voyage/ProjectSrv failures                |
| **DEBUG** | Cache hits, filter details                       |

### C. Tracing (OpenTelemetry)

```
search.Search
  ├── search.checkResultsCache
  ├── search.resolveCampaignProjects
  │   └── projectsrv.GetCampaign (optional)
  ├── search.embedQuery
  │   └── voyage.Embed (optional)
  ├── search.qdrantSearch
  ├── search.filterAndMap
  ├── search.buildAggregations
  └── search.cacheResults
```

---

## X. DEPLOYMENT

### Environment Variables

```bash
# Search Config
SEARCH_MIN_SCORE=0.65
SEARCH_MAX_RESULTS=10
SEARCH_MIN_QUERY_LENGTH=3
SEARCH_MAX_QUERY_LENGTH=1000

# Dependencies (shared with indexing)
QDRANT_URL=http://localhost:6333
QDRANT_COLLECTION=smap_analytics
REDIS_ADDR=localhost:6379
VOYAGE_API_KEY=pa-xxx
PROJECT_SERVICE_URL=http://project-srv:8080
```

---

## XI. CHECKLIST IMPLEMENTATION

### Phase 1: Foundation (Week 1)

- [ ] Domain scaffolding
  - [ ] `interface.go`, `types.go`, `errors.go`
  - [ ] Repository interfaces + options
- [ ] Repository implementation
  - [ ] Qdrant repository (search, query builder)
  - [ ] Redis cache repository (3 tầng)

### Phase 2: UseCase (Week 1-2)

- [ ] UseCase factory + config
- [ ] Main search logic
- [ ] Helpers (embed query, resolve campaign, aggregations, cache key)
- [ ] Score threshold + NoRelevantContext (hallucination control)

### Phase 3: Delivery (Week 2)

- [ ] HTTP handler + routes
- [ ] Request validation + presenters
- [ ] Error mapping

### Phase 4: Testing (Week 2-3)

- [ ] Unit tests: UseCase
- [ ] Integration tests: Repository
- [ ] E2E tests: Full pipeline

### Phase 5: Wiring (Week 3)

- [ ] `internal/httpserver/domain_search.go` — wire dependencies
- [ ] Verify search works end-to-end with indexed data
- [ ] Performance benchmark: cache hit vs miss latency
