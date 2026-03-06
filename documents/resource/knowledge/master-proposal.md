# Knowledge Service -- Domain Proposal

---

## Đề xuất các Domain cho Knowledge Service

Dựa trên phân tích chi tiết toàn bộ specs (API, kiến trúc, luồng dữ liệu, schema phân tích) và quy ước domain, đề xuất **4 domain chính** cho service này:

---

### Tổng quan Dependency giữa các Domain

```
                    ┌──────────────────────────────────────┐
                    │          DELIVERY (HTTP/Kafka)         │
                    └──────┬───────┬───────┬───────┬───────┘
                           │       │       │       │
                    ┌──────▼──┐ ┌──▼────┐ ┌▼─────┐ ┌▼──────────┐
                    │  chat   │ │search │ │report│ │ indexing   │
                    │ (RAG)   │ │       │ │      │ │            │
                    └────┬────┘ └───▲───┘ └──┬───┘ └────────────┘
                         │         │         │
                         └─────────┘─────────┘
                          (chat & report gọi search qua UseCase interface)

    Shared:  internal/model  (Scope, Entity models dùng chung)
    Pkg:     pkg/qdrant, pkg/gemini, pkg/voyage, pkg/redis,
             pkg/minio, pkg/projectsrv, pkg/kafka, ...
```

---

### Domain 1: `internal/indexing` -- Vector Indexing

**Vai trò:** Nhận analytics data đã phân tích từ Analytics Service, tạo vector embeddings, lưu vào Qdrant. Đây là **write path** của hệ thống RAG.

**Entity:** `IndexedDocument` (theo dõi metadata của các documents đã index)

#### 1.1 Định dạng đầu vào (Input Format) -- ĐÃ CHỐT

Dữ liệu đầu vào cho indexing domain là output của Analytics Service, đã chốt format tại:

- **Schema chi tiết:** `specs/ANALYTICS_SCHEMA_EXPLAINED.md`
- **Ví dụ JSON:** `specs/analytics_post_example.json`

Mỗi record `analytics.post_analytics` bao gồm các nhóm field chính:

| Nhóm Field         | Fields chính                                                              | Mục đích trong Indexing                       |
| ------------------ | ------------------------------------------------------------------------- | --------------------------------------------- |
| **Core Identity**  | `id`, `project_id`, `source_id`                                           | Tracking + Qdrant payload filter              |
| **UAP Core**       | `content`, `content_created_at`, `ingested_at`, `platform`                | `content` → sinh embedding; còn lại → payload |
| **Metadata**       | `uap_metadata` (author, engagement, hashtags, location...)                | Qdrant payload cho RAG context enrichment     |
| **Sentiment**      | `overall_sentiment`, `overall_sentiment_score`, `sentiment_confidence`    | Qdrant payload filter (POSITIVE/NEGATIVE/...) |
| **ABSA (Aspects)** | `aspects[]` (aspect, sentiment, score, keywords, mentions)                | Qdrant payload filter theo aspect             |
| **Keywords**       | `keywords[]`                                                              | Qdrant payload cho keyword search             |
| **Risk**           | `risk_level`, `risk_score`, `risk_factors[]`                              | Qdrant payload filter (HIGH/CRITICAL alerts)  |
| **Engagement**     | `engagement_score`, `virality_score`, `influence_score`, `reach_estimate` | Qdrant payload cho ranking/sorting            |
| **Quality**        | `content_quality_score`, `is_spam`, `is_bot`, `language`                  | Pre-filter: bỏ spam/bot trước khi index       |

**Lưu ý quan trọng -- Qdrant lưu CẢ dữ liệu đã phân tích:**

Mỗi point trong Qdrant gồm 2 phần:

- **Vector** (mảng 1536 số thực): chỉ được **sinh từ field `content`** (text gốc → Voyage/OpenAI embedding API → `[0.12, -0.45, ...]`). Đây là phần phục vụ **tìm kiếm theo ngữ nghĩa** (semantic similarity).
- **Payload** (JSON object): chứa **TẤT CẢ fields** từ analytics post, bao gồm cả kết quả đã phân tích (sentiment, aspects, scores, keywords, risk...) VÀ cả content gốc. Payload phục vụ **filtering** (lọc theo sentiment, aspect, date range...) và **trả về context** cho RAG.

```json
// Ví dụ 1 point trong Qdrant:
{
  "id": "analytics_001",
  "vector": [0.123, -0.456, ...],          // ← sinh từ content
  "payload": {                              // ← TẤT CẢ data đã phân tích
    "project_id": "proj_vf8",
    "content": "Xe đẹp nhưng pin yếu",     // content gốc (để trả về cho RAG context)
    "content_created_at": 1707577800,
    "platform": "tiktok",
    "overall_sentiment": "MIXED",           // ← ĐÃ phân tích
    "overall_sentiment_score": 0.15,        // ← ĐÃ phân tích
    "aspects": [                            // ← ĐÃ phân tích
      {"aspect": "DESIGN", "sentiment": "POSITIVE", "score": 0.85},
      {"aspect": "BATTERY", "sentiment": "NEGATIVE", "score": -0.72}
    ],
    "keywords": ["VF8", "thiết kế", "pin"],// ← ĐÃ phân tích
    "risk_level": "MEDIUM",                 // ← ĐÃ phân tích
    "engagement_score": 0.73,
    "metadata": {"author": "nguyen_van_a", "views": 45000}
  }
}
```

Tóm lại: Qdrant **CÓ chứa toàn bộ số liệu đã phân tích**. Field `content` chỉ đặc biệt ở chỗ nó được dùng thêm để sinh vector -- nhưng bản thân nó cũng nằm trong payload.

#### 1.2 Cách nhận data từ Analytics Service (2 phương thức)

Service hỗ trợ **2 cách** nhận data, hoạt động song song:

**Cách 1: Queue-based (Kafka Consumer) -- cho batch processing**

```
Analytics Service
    │
    │  1. Phân tích xong batch N records
    │  2. Ghi file JSON/JSONL lên MinIO
    │     Path: s3://smap-analytics/batches/{batch_id}.jsonl
    │  3. Publish message vào Kafka topic
    │
    ▼
Kafka topic: analytics.batch.completed
    Message: {
      "batch_id": "batch_001",
      "project_id": "proj_vf8",
      "file_url": "s3://smap-analytics/batches/batch_001.jsonl",
      "record_count": 500,
      "completed_at": "2026-02-15T10:46:15Z"
    }
    │
    ▼
Knowledge Service (Kafka Consumer)
    │  1. Consume message
    │  2. Download file từ MinIO qua file_url
    │  3. Parse JSONL → list analytics posts
    │  4. Với mỗi record: validate → embed → upsert Qdrant
    │  5. Lưu metadata vào knowledge.indexed_documents
    │  6. Ack message khi hoàn tất
```

**Cách 2: API-based (HTTP) -- cho single/realtime indexing**

```
Analytics Service
    │
    │  Phân tích xong 1 record hoặc small batch
    │
    ▼
POST /internal/index/by-file
    Headers: X-Service-Token: <secret>
    Body: {
      "batch_id": "batch_001",
      "project_id": "proj_vf8",
      "file_url": "s3://smap-analytics/batches/batch_001.jsonl",
      "record_count": 500
    }
    │
    ▼
Knowledge Service (HTTP Handler)
    │  1. Validate request + service token
    │  2. Download file từ MinIO qua file_url
    │  3. Parse → index (giống flow Kafka)
    │  4. Return response với kết quả
```

**So sánh 2 cách:**

| Tiêu chí          | Kafka Consumer                 | HTTP API                            |
| ----------------- | ------------------------------ | ----------------------------------- |
| **Phù hợp cho**   | Batch lớn (100-10000 records)  | Batch nhỏ, realtime (1-100 records) |
| **Retry**         | Tự động (Kafka consumer group) | Caller tự retry                     |
| **Back-pressure** | Tự nhiên (consumer lag)        | Cần rate limiting                   |
| **Coupling**      | Loose (async)                  | Tight (sync request-response)       |
| **Monitoring**    | Kafka lag metrics              | HTTP latency/error metrics          |

#### 1.3 Schema `knowledge.indexed_documents` -- Chi tiết

Table này **KHÔNG chứa nội dung analytics** (nội dung nằm ở Qdrant payload). Nó chỉ lưu **metadata tracking** để:

- Biết record nào đã index, record nào chưa (deduplication)
- Theo dõi trạng thái index (thành công/thất bại)
- Debug: khi nào index, mất bao lâu, có lỗi gì
- Thống kê: bao nhiêu docs đã index per project
- Hỗ trợ re-index: nếu cần index lại, biết cần xử lý records nào

```sql
CREATE TABLE knowledge.indexed_documents (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analytics_id    UUID NOT NULL,              -- FK tới analytics.post_analytics.id
    project_id      UUID NOT NULL,              -- Thuộc project nào
    source_id       UUID NOT NULL,              -- Từ data source nào

    -- Qdrant Reference
    qdrant_point_id UUID NOT NULL,              -- ID của point trong Qdrant collection
    collection_name VARCHAR(100) NOT NULL,      -- Qdrant collection name (e.g. "smap_analytics")

    -- Content Hash (for deduplication)
    content_hash    VARCHAR(64) NOT NULL,       -- SHA-256 hash của content, tránh index trùng

    -- Indexing Status
    status          VARCHAR(20) NOT NULL        -- PENDING | INDEXED | FAILED | RE_INDEXING
                    DEFAULT 'PENDING',
    error_message   TEXT,                       -- Lỗi nếu status = FAILED
    retry_count     INT DEFAULT 0,              -- Số lần retry

    -- Batch Tracking
    batch_id        VARCHAR(100),               -- Thuộc batch nào (nullable nếu index đơn lẻ)
    ingestion_method VARCHAR(20) NOT NULL,      -- 'kafka' | 'api' (nhận qua queue hay HTTP)

    -- Performance Metrics
    embedding_time_ms   INT,                    -- Thời gian sinh embedding
    upsert_time_ms      INT,                    -- Thời gian upsert Qdrant
    total_time_ms       INT,                    -- Tổng thời gian xử lý

    -- Timestamps
    indexed_at      TIMESTAMPTZ,                -- Thời điểm index thành công
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE UNIQUE INDEX idx_indexed_docs_analytics_id ON knowledge.indexed_documents(analytics_id);
CREATE INDEX idx_indexed_docs_project ON knowledge.indexed_documents(project_id);
CREATE INDEX idx_indexed_docs_batch ON knowledge.indexed_documents(batch_id);
CREATE INDEX idx_indexed_docs_status ON knowledge.indexed_documents(status);
CREATE INDEX idx_indexed_docs_content_hash ON knowledge.indexed_documents(content_hash);
```

**Ví dụ sử dụng:**

```go
// Kiểm tra đã index chưa (deduplication)
exists, _ := repo.ExistsByAnalyticsID(ctx, analyticsID)
if exists {
    return ErrAlreadyIndexed
}

// Kiểm tra content trùng (cùng nội dung từ source khác)
exists, _ = repo.ExistsByContentHash(ctx, contentHash)
if exists {
    return ErrDuplicateContent
}

// Lấy danh sách records FAILED để retry
failedDocs, _ := repo.ListByStatus(ctx, "FAILED", repo.Filter{MaxRetry: 3})
for _, doc := range failedDocs {
    // retry indexing...
}

// Thống kê per project
stats, _ := repo.CountByProject(ctx, projectID)
// stats = {total: 1500, indexed: 1480, failed: 15, pending: 5}
```

#### 1.4 API Endpoints

- `POST /internal/index/by-file` -- Nhận link MinIO, tải + index batch (Internal API, service token auth)

#### 1.5 Cấu trúc

```
internal/indexing/
├── delivery/
│   ├── http/
│   │   ├── new.go              # Factory
│   │   ├── handlers.go         # IndexByFile handler
│   │   ├── process_request.go  # Validate IndexByFileRequest
│   │   ├── presenters.go       # IndexByFileReq/Resp DTOs
│   │   ├── routes.go           # /internal/index/by-file
│   │   └── errors.go           # EMBEDDING_FAILED, CONTENT_TOO_SHORT mapping
│   └── kafka/
│       └── consumer/
│           ├── new.go          # Factory: NewConsumerGroup
│           ├── handler.go      # sarama ConsumerGroupHandler impl
│           └── worker.go       # ProcessBatchCompleted: download MinIO → parse → index
├── repository/
│   ├── interface.go            # IndexedDocumentRepository interface
│   ├── options.go              # Filter structs
│   └── postgre/
│       ├── new.go
│       ├── indexed_document.go       # Create, ExistsByAnalyticsID, ExistsByContentHash, UpdateStatus, CountByProject
│       ├── indexed_document_query.go # buildListQuery, buildExistsQuery
│       └── indexed_document_build.go # toDomain, toDB
├── usecase/
│   ├── new.go                  # implUseCase{repo, qdrant, embedding, minio, redis, logger}
│   ├── index.go                # IndexFromFile: download MinIO → parse JSONL → index records
│   ├── index_record.go         # indexSingleRecord: validate → embed → upsert Qdrant → save metadata
│   ├── batch.go                # processBatch: xử lý song song với errgroup, error aggregation
│   └── helpers.go              # prepareQdrantPoint, generateContentHash, filterSpamBot
├── interface.go                # UseCase interface: IndexFromFile(), RetryFailed()
├── types.go                    # IndexFromFileInput, IndexFromFileOutput, IndexRecordInput, BatchResult
└── errors.go                   # ErrEmbeddingFailed, ErrContentTooShort, ErrAlreadyIndexed, ErrDuplicateContent
```

**Trách nhiệm cụ thể:**

1. Nhận batch info (qua Kafka message hoặc HTTP request) chứa `file_url` trỏ tới MinIO
2. Download file JSONL từ MinIO (`pkg/minio`)
3. Parse từng record theo format đã chốt (`analytics.post_analytics`)
4. Pre-filter: bỏ qua records có `is_spam=true`, `is_bot=true`, `content_quality_score < 0.3`
5. Dedup: kiểm tra `analytics_id` và `content_hash` trong `knowledge.indexed_documents`
6. Sinh embedding cho `content` qua `pkg/voyage` (check Redis cache trước)
7. Chuẩn bị Qdrant point: vector + payload (tất cả fields từ analytics post)
8. Upsert vào Qdrant collection
9. Lưu tracking record vào `knowledge.indexed_documents` (status: INDEXED)
10. Invalidate search cache trong Redis
11. (Optional) Publish event `DOCUMENT_INDEXED` qua Kafka

---

### Domain 2: `internal/search` -- Semantic Search

**Vai trò:** Thực hiện semantic search trong Qdrant với campaign scope filtering. Đây là **domain nền tảng** được dùng bởi cả `chat` và `report`.

**Entity:** Stateless (không lưu trữ dữ liệu persistent, chỉ cache)

**API Endpoints:**

- `POST /api/v1/search` -- Semantic search (Public API, JWT auth)

#### 2.1 Cấu trúc

```
internal/search/
├── delivery/
│   └── http/
│       ├── new.go
│       ├── handlers.go         # Search handler
│       ├── process_request.go  # Validate SearchRequest + extract Scope
│       ├── presenters.go       # SearchReq/Resp DTOs, aggregation DTOs
│       ├── routes.go           # /api/v1/search
│       └── errors.go           # CAMPAIGN_NOT_FOUND, NO_RESULTS_FOUND mapping
├── repository/
│   ├── interface.go            # SearchRepository interface (Qdrant operations)
│   ├── options.go              # SearchFilter, SearchOptions structs
│   └── qdrant/                 # Qdrant-specific implementation
│       ├── new.go
│       ├── search.go           # Thực hiện vector search
│       ├── search_query.go     # Build Qdrant filters (project_id, sentiment, aspects, date range)
│       └── search_build.go     # Map Qdrant ScoredPoint → domain SearchResult
├── usecase/
│   ├── new.go                  # implUseCase{repo, embedding, projectSrv, redis, logger}
│   ├── search.go               # Search: resolve campaign → embed query → search Qdrant → aggregate → return
│   ├── cache.go                # Caching logic: get/set/invalidate
│   └── helpers.go              # buildCampaignFilter, aggregateResults
├── interface.go                # UseCase interface: Search(ctx, sc, input) (SearchOutput, error)
├── types.go                    # SearchInput, SearchOutput, SearchResult, Aggregation, SearchFilters
└── errors.go                   # ErrCampaignNotFound, ErrNoResultsFound
```

#### 2.2 Chiến lược Caching chi tiết

Domain search quản lý **3 tầng cache** qua `pkg/redis`:

**Tầng 1: Embedding Cache -- cache vector đã sinh từ query text**

```
Mục đích:  Tránh gọi lại Voyage/OpenAI API cho cùng một câu query
Key:       embedding:{sha256(query_text)}
Value:     []float32 (JSON serialized vector, 1536 dimensions)
TTL:       7 ngày
Size:      ~6KB per entry (1536 floats × 4 bytes)

Flow:
    query text
        → hash = SHA256("VinFast bị đánh giá tiêu cực về gì?")
        → Redis GET embedding:{hash}
        → HIT?  → dùng cached vector
        → MISS? → gọi pkg/voyage.Embed(query)
                 → Redis SET embedding:{hash} = vector (TTL 7d)
                 → dùng vector mới

Invalidation: Không cần. Embedding là deterministic (cùng input → cùng output).
              TTL 7 ngày tự expire, đủ dài vì model embedding hiếm khi thay đổi.
```

**Tầng 2: Campaign Projects Cache -- cache mapping campaign → project_ids**

```
Mục đích:  Tránh gọi Project Service mỗi lần search
Key:       campaign_projects:{campaign_id}
Value:     []string (JSON array of project_ids)
TTL:       10 phút

Flow:
    campaign_id
        → Redis GET campaign_projects:{campaign_id}
        → HIT?  → dùng cached project_ids
        → MISS? → gọi pkg/projectsrv.GetCampaignProjects(campaign_id)
                 → Redis SET campaign_projects:{campaign_id} = project_ids (TTL 10m)

Invalidation: TTL-based. 10 phút đủ ngắn để reflect campaign changes.
              Nếu cần realtime hơn, indexing domain invalidate key này
              khi nhận data từ project mới.
```

**Tầng 3: Search Results Cache -- cache kết quả search đã aggregated**

```
Mục đích:  Tránh query Qdrant lại cho cùng search request (query + filters)
Key:       search:{campaign_id}:{sha256(query + filters_json)}
Value:     SearchOutput (JSON serialized: results + aggregations)
TTL:       5 phút

Flow:
    SearchInput (campaign_id, query, filters)
        → cache_key = "search:{campaign_id}:{SHA256(query + JSON(filters))}"
        → Redis GET cache_key
        → HIT?  → deserialize và return SearchOutput
        → MISS? → embed query (Tầng 1 cache)
                 → resolve campaign projects (Tầng 2 cache)
                 → query Qdrant
                 → aggregate results
                 → Redis SET cache_key = SearchOutput (TTL 5m)
                 → return SearchOutput

Invalidation:
    - TTL-based: 5 phút tự expire
    - Active invalidation: khi indexing domain index data mới cho project_id,
      nó gọi Redis DEL search:{*} pattern matching project liên quan
      (thông qua event hoặc trực tiếp gọi redis SCAN + DEL)
```

**Tổng hợp cache flow trong 1 request:**

```
POST /api/v1/search
    │
    ▼
[Tầng 3] Kiểm tra search results cache ──── HIT → return ngay (< 5ms)
    │ MISS
    ▼
[Tầng 2] Kiểm tra campaign projects cache ── HIT → dùng cached project_ids
    │ MISS → gọi Project Service (50ms)
    ▼
[Tầng 1] Kiểm tra embedding cache ────────── HIT → dùng cached vector (< 5ms)
    │ MISS → gọi Voyage API (200-800ms)
    ▼
Query Qdrant với vector + filters (100-300ms)
    │
    ▼
Aggregate kết quả (5-10ms)
    │
    ▼
SET cả 3 tầng cache → return response

Latency:
    - Full cache hit:  < 10ms
    - Partial hits:    200-500ms (thường chỉ miss Tầng 3)
    - Full cache miss: 500-1200ms (first query ever)
```

#### 2.3 Trách nhiệm cụ thể

1. Nhận campaign_id → resolve project_ids (Tầng 2 cache → `pkg/projectsrv`)
2. Sinh embedding cho query (Tầng 1 cache → `pkg/voyage`)
3. Kiểm tra search results cache (Tầng 3)
4. Build Qdrant filters: project_id IN [...], sentiment, aspects, date range, platforms
5. Thực hiện vector search trong Qdrant
6. Tập hợp kết quả (by_sentiment, by_aspect, by_platform)
7. Cache kết quả (Tầng 3)
8. Trả về kết quả search đã cấu trúc

**Điểm quan trọng:** Interface UseCase của domain này được **`chat`** và **`report`** inject và gọi trực tiếp, theo đúng convention "UseCase to UseCase Only". Khi `chat` hay `report` gọi `search.UseCase.Search()`, chúng cũng hưởng lợi từ 3 tầng cache này.

---

### Domain 3: `internal/chat` -- RAG Chat Q&A

**Vai trò:** Core pipeline RAG -- chat hỏi đáp trong context campaign, quản lý lịch sử hội thoại, và tạo gợi ý thông minh. Đây là **domain giá trị nhất** của service.

**Entity:** `Conversation`, `Message`

**API Endpoints:**

- `POST /api/v1/chat` -- Chat với campaign (Public API, JWT auth)
- `GET /api/v1/conversations/{conversation_id}` -- Lấy lịch sử chat
- `GET /api/v1/campaigns/{campaign_id}/suggestions` -- Smart suggestions

#### 3.1 Schema lưu lịch sử Chat -- Chi tiết

```sql
-- =============================================
-- CONVERSATIONS: Mỗi cuộc hội thoại trong 1 campaign
-- =============================================
CREATE TABLE knowledge.conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     UUID NOT NULL,              -- Thuộc campaign nào
    user_id         UUID NOT NULL,              -- Ai tạo conversation
    title           VARCHAR(500),               -- Tiêu đề (tự sinh từ message đầu tiên, hoặc user đặt)

    -- State
    status          VARCHAR(20) NOT NULL        -- ACTIVE | ARCHIVED
                    DEFAULT 'ACTIVE',
    message_count   INT DEFAULT 0,              -- Đếm tổng số messages (cả user + assistant)

    -- Timestamps
    last_message_at TIMESTAMPTZ,                -- Thời điểm message cuối (sort conversations)
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conversations_campaign ON knowledge.conversations(campaign_id);
CREATE INDEX idx_conversations_user ON knowledge.conversations(user_id);
CREATE INDEX idx_conversations_last_msg ON knowledge.conversations(last_message_at DESC);

-- =============================================
-- MESSAGES: Từng tin nhắn trong conversation
-- =============================================
CREATE TABLE knowledge.messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES knowledge.conversations(id) ON DELETE CASCADE,

    -- Content
    role            VARCHAR(20) NOT NULL,       -- 'user' | 'assistant'
    content         TEXT NOT NULL,              -- Nội dung tin nhắn

    -- RAG Context (chỉ có ở assistant messages)
    citations       JSONB,                     -- Array of Citation objects (source, relevance_score, ...)
    search_metadata JSONB,                     -- {total_docs_searched, docs_used, processing_time_ms, model_used}
    suggestions     JSONB,                     -- Array of follow-up suggestion strings

    -- User Request Context (chỉ có ở user messages)
    filters_used    JSONB,                     -- SearchFilters mà user đã apply {sentiment, aspects, date_range, ...}

    -- Timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON knowledge.messages(conversation_id, created_at ASC);
```

**Ví dụ data:**

```sql
-- Conversation
INSERT INTO knowledge.conversations VALUES (
    'conv_123', 'camp_001', 'user_456',
    'Đánh giá tiêu cực VinFast',  -- title (auto-generated từ message đầu)
    'ACTIVE', 4,                    -- 4 messages (2 lượt hỏi-đáp)
    '2026-02-15T10:46:08Z',
    NOW(), NOW()
);

-- Message 1: User hỏi
INSERT INTO knowledge.messages VALUES (
    'msg_001', 'conv_123', 'user',
    'VinFast được đánh giá như thế nào?',
    NULL, NULL, NULL,                    -- assistant fields: null cho user message
    NULL,                                -- không có filter
    '2026-02-15T10:45:00Z'
);

-- Message 2: Assistant trả lời
INSERT INTO knowledge.messages VALUES (
    'msg_002', 'conv_123', 'assistant',
    'VinFast nhận được đánh giá trái chiều. Thiết kế được khen (85% positive)...',
    '[{"id":"analytics_001","content":"Xe đẹp...","relevance_score":0.92}]'::jsonb,
    '{"total_docs_searched":1500,"docs_used":10,"processing_time_ms":2340,"model_used":"gemini-pro"}'::jsonb,
    '["Cụ thể về pin thì sao?","So sánh với BYD"]'::jsonb,
    NULL,
    '2026-02-15T10:45:05Z'
);

-- Message 3: User hỏi tiếp (multi-turn)
INSERT INTO knowledge.messages VALUES (
    'msg_003', 'conv_123', 'user',
    'Cụ thể về pin thì sao?',
    NULL, NULL, NULL,
    '{"aspects":["BATTERY"]}'::jsonb,    -- user filter theo aspect
    '2026-02-15T10:46:00Z'
);

-- Message 4: Assistant trả lời với context từ lượt trước
INSERT INTO knowledge.messages VALUES (
    'msg_004', 'conv_123', 'assistant',
    'Về pin, VinFast nhận nhiều phản hồi tiêu cực. Cụ thể: sụt pin nhanh [1], dung lượng thấp [2]...',
    '[{"id":"analytics_005","content":"pin sụt nhanh...","relevance_score":0.95}]'::jsonb,
    '{"total_docs_searched":1500,"docs_used":10,"processing_time_ms":1890,"model_used":"gemini-pro"}'::jsonb,
    '["So sánh pin VF8 vs BYD","Xu hướng phản hồi pin theo thời gian"]'::jsonb,
    NULL,
    '2026-02-15T10:46:08Z'
);
```

#### 3.2 Multi-turn Conversation Flow -- Chi tiết

```
User gửi message lượt 2+:
POST /api/v1/chat
{
    "campaign_id": "camp_001",
    "conversation_id": "conv_123",       ← có conversation_id = multi-turn
    "message": "Cụ thể về pin thì sao?",
    "filters": { "aspects": ["BATTERY"] }
}
    │
    ▼
Step 1: Load conversation history từ PostgreSQL
    │   SELECT role, content FROM knowledge.messages
    │   WHERE conversation_id = 'conv_123'
    │   ORDER BY created_at ASC
    │   LIMIT 20  ← giới hạn context window
    │
    │   Result:
    │   [
    │     {role: "user",      content: "VinFast được đánh giá như thế nào?"},
    │     {role: "assistant", content: "VinFast nhận được đánh giá trái chiều..."},
    │   ]
    │
    ▼
Step 2: Search documents (gọi search.UseCase.Search())
    │   - Embed query "Cụ thể về pin thì sao?"
    │   - Filter: project_ids từ campaign + aspects = ["BATTERY"]
    │   - Top 10 relevant documents
    │
    ▼
Step 3: Build LLM Prompt (multi-turn aware)
    │
    │   ┌─── System Prompt ────────────────────────────────────┐
    │   │ Bạn là trợ lý phân tích dữ liệu SMAP.              │
    │   │ Trả lời dựa trên context. Trích dẫn bằng [1],[2]...│
    │   └──────────────────────────────────────────────────────┘
    │   ┌─── Context (từ Qdrant search results) ───────────────┐
    │   │ [1] "Pin sụt nhanh, đi 100km hết 25%..." (TikTok)   │
    │   │ [2] "Pin yếu hơn mình nghĩ..." (YouTube)            │
    │   │ ... (10 documents)                                    │
    │   └──────────────────────────────────────────────────────┘
    │   ┌─── Conversation History (từ PostgreSQL) ─────────────┐
    │   │ User: VinFast được đánh giá như thế nào?             │
    │   │ Assistant: VinFast nhận được đánh giá trái chiều...  │
    │   └──────────────────────────────────────────────────────┘
    │   ┌─── Current Question ─────────────────────────────────┐
    │   │ User: Cụ thể về pin thì sao?                        │
    │   └──────────────────────────────────────────────────────┘
    │
    ▼
Step 4: Gọi LLM (pkg/gemini) → sinh answer
    │
    ▼
Step 5: Lưu messages mới vào PostgreSQL
    │   INSERT message (role='user', content='Cụ thể về pin thì sao?', filters_used=...)
    │   INSERT message (role='assistant', content='Về pin, VinFast...', citations=..., suggestions=...)
    │   UPDATE conversation SET message_count += 2, last_message_at = NOW()
    │
    ▼
Step 6: Return ChatResponse
```

**Context Window Management:**

```
Vấn đề:  LLM có giới hạn token (e.g. Gemini Pro: 32K tokens).
          Conversation dài + search context có thể vượt quá.

Chiến lược:
    1. Conversation history: lấy tối đa 20 messages gần nhất
       (10 lượt hỏi-đáp). Nếu conversation dài hơn, cắt từ đầu.

    2. Search context: tối đa 10 documents, mỗi document truncate ở 500 chars.

    3. Priority: Search context > Recent history > Old history
       Nếu vẫn vượt token limit:
       - Giảm search docs: 10 → 5
       - Giảm history: 20 → 10 messages
       - Truncate history messages cũ (giữ message mới nhất đầy đủ)

    4. Token counting: dùng tiktoken hoặc estimate ~4 chars = 1 token
```

#### 3.3 Cấu trúc

```
internal/chat/
├── delivery/
│   └── http/
│       ├── new.go
│       ├── handlers.go         # Chat, GetConversation, GetSuggestions
│       ├── process_request.go  # Validate ChatRequest + extract Scope
│       ├── presenters.go       # ChatReq/Resp, ConversationResp, SuggestionResp DTOs
│       ├── routes.go           # /api/v1/chat, /api/v1/conversations/:id, /api/v1/campaigns/:id/suggestions
│       └── errors.go           # CAMPAIGN_NOT_FOUND, LLM_FAILED mapping
├── repository/
│   ├── interface.go            # ConversationRepository + MessageRepository (composed)
│   ├── options.go              # ConversationFilter, MessageFilter
│   └── postgre/
│       ├── new.go
│       ├── conversation.go           # Create, GetByID, ListByCampaign, UpdateLastMessage
│       ├── conversation_query.go     # buildListQuery (filter by campaign_id, user_id, status)
│       ├── conversation_build.go     # toDomain, toDB
│       ├── message.go                # Create, ListByConversation (ordered, limited)
│       ├── message_query.go          # buildListQuery (filter by conversation_id, limit, offset)
│       └── message_build.go          # toDomain, toDB
├── usecase/
│   ├── new.go                  # implUseCase{repo, searchUC, llm, projectSrv, redis, logger}
│   ├── chat.go                 # Chat: load history → search → build prompt → call LLM → save → return
│   ├── conversation.go         # GetConversation, ListConversations
│   ├── suggestion.go           # GetSuggestions: phân tích xu hướng dữ liệu campaign → sinh câu hỏi
│   ├── prompt.go               # buildSystemPrompt, buildContextBlock, buildHistoryBlock, manageTokenWindow
│   └── helpers.go              # extractCitations, generateTitle, formatContext
├── interface.go                # UseCase interface: Chat(), GetConversation(), ListConversations(), GetSuggestions()
├── types.go                    # ChatInput, ChatOutput, Citation, ConversationOutput, SuggestionOutput
└── errors.go                   # ErrConversationNotFound, ErrLLMFailed, ErrRateLimitExceeded
```

#### 3.4 Trách nhiệm cụ thể

1. **Quy trình chat (RAG Pipeline):**
   - Nhận message + campaign_id + optional conversation_id + optional filters
   - Nếu conversation_id rỗng → tạo conversation mới, auto-generate title từ message đầu
   - Nếu conversation_id có → load conversation history từ PostgreSQL (tối đa 20 messages gần nhất)
   - Gọi `search.UseCase.Search()` để lấy top N documents liên quan
   - Build prompt: System Prompt + Search Context + Conversation History + Current Question
   - Quản lý token window: đảm bảo tổng prompt không vượt quá giới hạn LLM
   - Gọi LLM (`pkg/gemini`) để sinh câu trả lời
   - Trích xuất citations từ answer + search results
   - Tạo gợi ý follow-up
   - Lưu 2 messages (user + assistant) vào PostgreSQL
   - Cập nhật conversation: message_count, last_message_at
   - (Optional) Publish event `CHAT_QUERY` qua Kafka

2. **Quản lý hội thoại:**
   - Tạo conversation (auto khi chat lần đầu)
   - Lấy chi tiết conversation + toàn bộ messages
   - List conversations theo campaign (sort by last_message_at DESC)
   - Archive conversation (status → ARCHIVED)

3. **Smart suggestions:**
   - Phân tích khía cạnh trending (nhiều negative mentions)
   - Phát hiện thay đổi sentiment > 10% trong 7 ngày
   - Gợi ý truy vấn so sánh
   - Gợi ý truy vấn insight

---

### Domain 4: `internal/report` -- Report Generation

**Vai trò:** Tạo báo cáo tự động từ campaign data (xử lý không đồng bộ). Support nhiều loại báo cáo: SUMMARY, COMPARISON, TREND, ASPECT_DEEP_DIVE.

**Entity:** `Report`

**API Endpoints:**

- `POST /api/v1/reports/generate` -- Tạo báo cáo (async, trả về report_id)
- `GET /api/v1/reports/{report_id}` -- Kiểm tra trạng thái + lấy thông tin report
- `GET /api/v1/reports/{report_id}/download` -- Tải file report

**Cấu trúc:**

```
internal/report/
├── delivery/
│   ├── http/
│   │   ├── new.go
│   │   ├── handlers.go         # GenerateReport, GetReport, DownloadReport
│   │   ├── process_request.go  # Validate ReportRequest
│   │   ├── presenters.go       # ReportReq/Resp DTOs
│   │   ├── routes.go           # /api/v1/reports/generate, /api/v1/reports/:id, /api/v1/reports/:id/download
│   │   └── errors.go
│   └── kafka/                  # (Optional) Publish REPORT_COMPLETED event
│       └── producer/
├── repository/
│   ├── interface.go            # ReportRepository interface
│   ├── options.go              # ReportFilter
│   └── postgre/
│       ├── new.go
│       ├── report.go           # CRUD reports
│       ├── report_query.go
│       └── report_build.go
├── usecase/
│   ├── new.go                  # implUseCase{repo, searchUC, llm, minio, projectSrv, logger}
│   ├── report.go               # GenerateReport (async), GetReport, DownloadReport
│   ├── generator.go            # Background job: search data → aggregate → LLM generate content → format
│   ├── template.go             # Report templates per type (SUMMARY, COMPARISON, TREND, ASPECT_DEEP_DIVE)
│   └── helpers.go              # convertToPDF, buildReportSections
├── interface.go                # UseCase interface: Generate(), GetReport()
├── types.go                    # GenerateInput, GenerateOutput, ReportOutput, ReportOptions
└── errors.go                   # ErrReportNotFound, ErrReportGenerationFailed
```

**Trách nhiệm cụ thể:**

1. **Luồng Generate (Async):**
   - Validate request + tạo bản ghi report (status: PROCESSING)
   - Trả lại ngay report_id
   - Background goroutine:
     - Gọi `search.UseCase.Search()` nhiều lần để lấy data
     - Tổng hợp chỉ số (phân bố sentiment, breakdown aspects, top mentions)
     - Gọi LLM (`pkg/gemini`) để sinh nội dung các phần report
     - Chuyển sang PDF (dùng thư viện)
     - Upload lên MinIO (`pkg/minio`)
     - Cập nhật report record (status: COMPLETED, file_url)
     - (Optional) Publish event `REPORT_COMPLETED` và gọi webhook

2. **Quản lý report:**
   - Kiểm tra trạng thái (PROCESSING / COMPLETED / FAILED)
   - Lấy metadata report
   - Tạo presigned download URL từ MinIO

---

### Tổng kết Interaction Map

| Domain     | Gọi ai?                                                   | Ai gọi nó?                     | DB Tables                                       | External Deps                          |
| ---------- | --------------------------------------------------------- | ------------------------------ | ----------------------------------------------- | -------------------------------------- |
| `indexing` | `pkg/qdrant`, `pkg/voyage`, `pkg/redis`, `pkg/minio`      | Analytics Service (HTTP/Kafka) | `knowledge.indexed_documents`                   | Qdrant, Voyage, Redis, MinIO           |
| `search`   | `pkg/qdrant`, `pkg/voyage`, `pkg/redis`, `pkg/projectsrv` | `chat`, `report`, HTTP client  | (none - stateless, 3-layer cache only)          | Qdrant, Voyage, Redis, Project Service |
| `chat`     | `search.UseCase`, `pkg/gemini`, `pkg/redis`               | Web UI, Mobile App             | `knowledge.conversations`, `knowledge.messages` | Gemini (LLM), Redis                    |
| `report`   | `search.UseCase`, `pkg/gemini`, `pkg/minio`               | Web UI                         | `knowledge.reports`                             | Gemini (LLM), MinIO                    |

### Tổng kết DB Schema

```
knowledge schema:
├── knowledge.indexed_documents    ← Domain 1 (indexing) -- tracking metadata
├── knowledge.conversations        ← Domain 3 (chat) -- conversation state
├── knowledge.messages             ← Domain 3 (chat) -- message history + citations
└── knowledge.reports              ← Domain 4 (report) -- report metadata + status
```

### Nguyên tắc thiết kế:

1. **Separation of Concerns**: Mỗi domain có một trách nhiệm rõ ràng. `indexing` = write path, `search` = read/query, `chat` = RAG orchestration, `report` = async report generation.

2. **Search là domain nền tảng**: Cả `chat` và `report` đều depend vào interface `search.UseCase`. Điều này giúp tránh trùng lặp logic search/filter và đảm bảo nhất quán. 3 tầng cache (embedding, campaign projects, search results) được quản lý tập trung tại domain này.

3. **UseCase to UseCase Only**: Theo đúng convention -- `chat.UseCase` gọi `search.UseCase`, KHÔNG gọi trực tiếp `search.Repository` hay `pkg/qdrant`.

4. **Tầng pkg cho infrastructure**: Tất cả external clients (Qdrant, Gemini, Voyage, Redis, MinIO, Project Service) đều được wrap trong `pkg/` theo đúng `PKG_CONVENTION.md`. Domain chỉ tương tác với chúng qua interfaces.

5. **Báo cáo thực hiện async**: Report generation dùng goroutine + polling pattern, phù hợp với convention giao job.

6. **Dual ingestion**: Domain `indexing` hỗ trợ cả Kafka consumer (batch) và HTTP API (realtime), cả hai đều nhận MinIO file URL rồi download + parse + index.

7. **Analytics format đã chốt**: Input format cho indexing tuân theo `specs/analytics_post_example.json` và `specs/ANALYTICS_SCHEMA_EXPLAINED.md`. Knowledge Service KHÔNG phân tích sentiment/aspect -- chỉ nhận kết quả đã phân tích và index vào Qdrant.

---

---

## Bổ sung: Production Readiness -- Giải quyết các vấn đề thực tế

---

### P1. Data Consistency & Reliability (Domain Indexing)

#### P1.1 Dual-Write Problem: Qdrant + Postgres

**Vấn đề:** Lưu metadata vào Postgres (`knowledge.indexed_documents`) và vector vào Qdrant là 2 hệ thống riêng biệt. Nếu 1 trong 2 thất bại → data bị lệch.

**Giải pháp: Qdrant-first + Postgres-compensation**

```
Chiến lược: "Qdrant là source of truth cho vector, Postgres là audit log"

Flow cho mỗi record:
    │
    ▼
Step 1: INSERT vào Postgres với status = 'PENDING'
    │   (Nếu fail → skip record, log error. Chưa có gì trong Qdrant → safe.)
    │
    ▼
Step 2: Sinh embedding + Upsert vào Qdrant
    │
    ├── SUCCESS → Step 3a: UPDATE Postgres SET status = 'INDEXED', indexed_at = NOW()
    │                       (Nếu UPDATE fail → Postgres vẫn có record PENDING,
    │                        background job sẽ reconcile sau.)
    │
    └── FAIL    → Step 3b: UPDATE Postgres SET status = 'FAILED', error_message = '...'
                            retry_count += 1
                            (Record FAILED sẽ được retry bởi background job.)

Tại sao thứ tự này?
    - Postgres insert trước (nhẹ, nhanh, ít lỗi) để có "bằng chứng" record đang được xử lý.
    - Qdrant upsert sau (nặng hơn, phụ thuộc network + embedding API).
    - Nếu Qdrant fail: Postgres biết record đó PENDING/FAILED → retry.
    - Nếu Postgres update fail sau khi Qdrant success: worst case là
      Qdrant có data nhưng Postgres ghi PENDING. Background reconciler
      sẽ check Qdrant để verify và update lại.
```

**Background Reconciler Job** (chạy mỗi 5 phút):

```go
// usecase/reconcile.go
func (uc *implUseCase) Reconcile(ctx context.Context) error {
    // 1. Lấy tất cả records PENDING quá 10 phút (bị "treo")
    staleRecords, _ := uc.repo.ListStale(ctx, 10*time.Minute)

    for _, doc := range staleRecords {
        // 2. Kiểm tra record có tồn tại trong Qdrant không
        exists, _ := uc.qdrant.PointExists(ctx, doc.CollectionName, doc.QdrantPointID)

        if exists {
            // Qdrant có → Postgres chưa update → fix lại
            uc.repo.UpdateStatus(ctx, doc.ID, "INDEXED", time.Now())
        } else {
            // Qdrant không có → thực sự chưa index → retry
            uc.retryIndex(ctx, doc)
        }
    }
    return nil
}
```

#### P1.2 Partial Batch Failure + Dead Letter Queue (DLQ)

**Vấn đề:** Batch 500 records, record thứ 499 lỗi. Reject cả batch = 498 records delay vô lý. Bỏ qua = mất dữ liệu.

**Giải pháp: Per-record error isolation + DLQ table**

```
Chiến lược: "Fail từng record, không fail cả batch"

Batch 500 records
    │
    ▼
Xử lý song song (errgroup, concurrency = 10)
    │
    ├── Record 1:   ✅ INDEXED
    ├── Record 2:   ✅ INDEXED
    ├── ...
    ├── Record 499: ❌ FAILED (JSON parse error)
    │                   → Log error
    │                   → INSERT vào knowledge.indexed_documents (status='FAILED', error_message='...')
    │                   → Nếu retry_count >= 3: INSERT vào knowledge.indexing_dlq
    ├── Record 500: ✅ INDEXED
    │
    ▼
Kết quả batch: {total: 500, indexed: 498, failed: 2}
    │
    ▼
Ack Kafka message (vì batch đã "xử lý xong", dù có record lỗi)
    - Record lỗi ĐÃ ĐƯỢC tracking trong DB.
    - Không cần Kafka retry cả batch.
```

**DLQ Table:**

```sql
CREATE TABLE knowledge.indexing_dlq (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analytics_id    UUID NOT NULL,
    batch_id        VARCHAR(100),
    raw_payload     JSONB NOT NULL,             -- Lưu nguyên record gốc để debug/retry manual
    error_message   TEXT NOT NULL,
    error_type      VARCHAR(50) NOT NULL,       -- 'PARSE_ERROR' | 'EMBEDDING_ERROR' | 'QDRANT_ERROR'
    retry_count     INT DEFAULT 0,
    max_retries     INT DEFAULT 3,
    resolved        BOOLEAN DEFAULT false,      -- Admin đã xử lý chưa
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

**DLQ Xử lý:**

```
1. Automatic retry:  Background job retry records trong DLQ mỗi 15 phút
                     (chỉ retry nếu retry_count < max_retries VÀ error_type != 'PARSE_ERROR')
2. Manual review:    Dashboard hiển thị DLQ records để dev/ops review
3. PARSE_ERROR:      Không auto-retry (data bị hỏng). Cần Analytics Service gửi lại.
4. EMBEDDING_ERROR:  Auto-retry (transient: API rate limit, timeout)
5. QDRANT_ERROR:     Auto-retry (transient: connection timeout, cluster issue)
```

#### P1.3 Update Propagation (Re-analysis)

**Vấn đề:** Analytics Service chạy lại phân tích, sentiment đổi từ POSITIVE sang NEGATIVE. Cách cập nhật trong Qdrant?

**Giải pháp: `analytics_id` = Qdrant Point ID (Idempotent Upsert)**

```
Quy tắc cốt lõi:
    Qdrant Point ID = analytics_id (UUID từ analytics.post_analytics)

Khi index lần đầu:
    Qdrant.Upsert(point_id = "analytics_001", vector = [...], payload = {...})
    → Tạo mới point

Khi Analytics Service re-analyze cùng bài post:
    Analytics ID vẫn là "analytics_001" (không đổi)
    Nhưng sentiment đổi: POSITIVE → NEGATIVE
    │
    ▼
    Knowledge Service nhận record update
    │
    ▼
    Qdrant.Upsert(point_id = "analytics_001", vector = [...mới...], payload = {...mới...})
    → Qdrant tự GHI ĐÈ point cũ (Upsert = Insert or Update)
    → Không cần DELETE + INSERT
    → Không tạo rác

Cập nhật Postgres:
    UPDATE knowledge.indexed_documents
    SET status = 'INDEXED',
        content_hash = '{new_hash}',
        indexed_at = NOW(),
        updated_at = NOW()
    WHERE analytics_id = 'analytics_001';

Flow detect update:
    - Khi nhận record, check ExistsByAnalyticsID()
    - Nếu tồn tại → đây là RE-INDEX (update), không phải insert mới
    - Vẫn chạy cùng flow: embed → upsert → update metadata
    - content_hash thay đổi? → Cần re-embed
    - content_hash không đổi? → Chỉ update payload (skip embedding, tiết kiệm API call)
```

```go
// usecase/index_record.go
func (uc *implUseCase) indexSingleRecord(ctx context.Context, record AnalyticsPost) error {
    contentHash := sha256(record.Content)
    pointID := record.AnalyticsID  // <-- DÙNG analytics_id LÀM point ID

    existing, _ := uc.repo.GetByAnalyticsID(ctx, record.AnalyticsID)

    var vector []float32
    if existing != nil && existing.ContentHash == contentHash {
        // Content không đổi → skip embedding, chỉ update payload
        // (ví dụ: sentiment thay đổi nhưng text vẫn vậy)
        vector = nil // Qdrant hỗ trợ update payload without vector
        uc.qdrant.SetPayload(ctx, collection, pointID, newPayload)
    } else {
        // Content đổi hoặc record mới → cần embedding mới
        vector, _ = uc.embedding.Embed(ctx, record.Content)
        uc.qdrant.Upsert(ctx, collection, pointID, vector, newPayload)
    }

    // Update hoặc Create tracking record
    uc.repo.Upsert(ctx, IndexedDocument{
        AnalyticsID:   record.AnalyticsID,
        QdrantPointID: pointID,
        ContentHash:   contentHash,
        Status:        "INDEXED",
        // ...
    })

    return nil
}
```

---

### P2. Chất lượng RAG & Search (Domain Search/Chat)

#### P2.1 Hybrid Search: Semantic + Keyword (BM25)

**Vấn đề:** Pure vector search tệ với exact match (mã sản phẩm "VF8-ECO-2024", tên riêng, số liệu cụ thể). Embedding model thường không encode chính xác các mã/ký hiệu.

**Giải pháp: Qdrant Hybrid Search (Dense + Sparse vectors)**

```
Qdrant hỗ trợ native Hybrid Search từ v1.7+:
    - Dense vector:  Voyage/OpenAI embedding (semantic meaning)
    - Sparse vector: BM25-like sparse embedding (keyword exact match)

Khi INDEX:
    Mỗi point có 2 vectors:
    {
        "id": "analytics_001",
        "vector": {
            "dense": [0.123, -0.456, ...],         // 1536 dims, từ Voyage
            "sparse": {                              // Sparse vector, từ SPLADE/BM25
                "indices": [102, 3547, 9821, ...],   // Token IDs có trong content
                "values":  [0.8, 1.2, 0.5, ...]      // TF-IDF weights
            }
        },
        "payload": { ... }
    }

Khi SEARCH:
    Query: "VF8-ECO-2024 pin yếu"

    1. Dense search:  Tìm docs có ý nghĩa tương tự "pin yếu" (semantic)
    2. Sparse search: Tìm docs chứa exact text "VF8-ECO-2024" (keyword)
    3. Fusion:        Kết hợp 2 kết quả bằng Reciprocal Rank Fusion (RRF)

    Qdrant API:
    POST /collections/smap_analytics/points/query
    {
        "prefetch": [
            { "query": dense_vector,  "using": "dense",  "limit": 20 },
            { "query": sparse_vector, "using": "sparse", "limit": 20 }
        ],
        "query": { "fusion": "rrf" },  // Reciprocal Rank Fusion
        "limit": 10
    }
```

**Tác động đến codebase:**

```
Thay đổi:
    1. pkg/qdrant:   Thêm method HybridSearch() bên cạnh Search()
    2. pkg/voyage:   Thêm SparseEmbed() hoặc dùng thư viện SPLADE riêng
    3. Indexing:     Khi index, sinh CẢ dense + sparse vectors
    4. Search UC:    Mặc định dùng HybridSearch thay vì Search
    5. Collection:   Setup collection với 2 named vectors ("dense" + "sparse")

Cấu hình Collection:
    PUT /collections/smap_analytics
    {
        "vectors": {
            "dense": { "size": 1536, "distance": "Cosine" },
            "sparse": { "datatype": "float32", "modifier": "idf" }  // Qdrant sparse vector
        }
    }
```

#### P2.2 Hallucination Control (Score Threshold)

**Vấn đề:** User hỏi "Cách nấu phở" trong campaign xe điện. Search vẫn trả về top 10 docs (dù score thấp). LLM cố bịa câu trả lời từ context không liên quan.

**Giải pháp: Configurable `min_score` + "No context" fallback**

```
Config:
    search:
      min_score: 0.65           # Ngưỡng tối thiểu similarity score
      min_relevant_docs: 3      # Ít nhất 3 docs vượt ngưỡng mới đưa cho LLM

Flow trong search.UseCase:
    1. Query Qdrant → nhận 10 results với scores
    2. Filter: chỉ giữ results có score >= min_score
    3. Nếu filtered results < min_relevant_docs:
       → Return SearchOutput{Results: [], NoRelevantContext: true}
    4. Nếu đủ:
       → Return SearchOutput{Results: filteredResults}

Flow trong chat.UseCase:
    searchOutput := uc.searchUC.Search(ctx, input)

    if searchOutput.NoRelevantContext {
        // Không đưa context cho LLM. Trả lời thẳng:
        return ChatOutput{
            Answer: "Tôi không tìm thấy thông tin liên quan đến câu hỏi này " +
                    "trong dữ liệu campaign hiện tại. Bạn có thể thử hỏi về " +
                    "các chủ đề khác liên quan đến [campaign topics].",
            Citations: nil,
        }
    }

    // Có context đủ tốt → tiếp tục RAG pipeline bình thường
```

#### P2.3 Citation Accuracy (Post-processing Verification)

**Vấn đề:** LLM hay bịa citation `[1], [2]` không khớp với document thực tế.

**Giải pháp: Structured output + Post-processing verification**

```
Chiến lược 2 lớp:

Lớp 1 - Prompt Engineering (phòng ngừa):
    System prompt yêu cầu LLM trả về answer trong format JSON:
    {
        "answer": "VinFast bị chê về pin {cite:1} và giá {cite:2}...",
        "used_citations": [1, 2, 5]  // Danh sách index đã dùng
    }

    Kèm instruction: "CHỈ trích dẫn index [N] nếu bạn thực sự dùng
    thông tin từ document [N] trong context. KHÔNG bịa index."

Lớp 2 - Post-processing (xác minh):
    // usecase/helpers.go → extractCitations()
    func extractCitations(answer string, searchResults []SearchResult) []Citation {
        // 1. Parse tất cả {cite:N} hoặc [N] trong answer
        citedIndices := parseCitationIndices(answer)  // e.g. [1, 2, 5]

        // 2. Verify: index có tồn tại trong search results không?
        validCitations := []Citation{}
        for _, idx := range citedIndices {
            if idx >= 1 && idx <= len(searchResults) {
                result := searchResults[idx-1]
                validCitations = append(validCitations, Citation{
                    Index:          idx,
                    AnalyticsID:    result.ID,
                    Content:        result.Content,
                    RelevanceScore: result.Score,
                    Source:         result.Source,
                })
            }
            // Index ngoài range → bỏ qua (LLM bịa)
        }

        // 3. Clean answer: xóa citation không hợp lệ khỏi text
        cleanedAnswer := removeBadCitations(answer, validCitations)

        return validCitations
    }

Kết quả:
    - Citation [1] trỏ đúng document [1] trong search results → giữ
    - Citation [8] nhưng chỉ có 5 documents → xóa khỏi answer
    - LLM viết "[nguồn: TikTok]" thay vì [N] → không match pattern → bỏ qua
```

---

### P3. Performance & Scalability (Domain Chat)

#### P3.1 Streaming Response (SSE) cho Chat API

**Vấn đề:** RAG latency 3-8 giây. User tưởng app treo. Chatbot hiện đại đều streaming từng token.

**Giải pháp: Server-Sent Events (SSE) cho Chat endpoint**

```
API thay đổi:
    POST /api/v1/chat
    Headers:
        Accept: text/event-stream    ← Client yêu cầu streaming
        hoặc
        Accept: application/json     ← Client yêu cầu response đầy đủ (giữ backward compat)

SSE Flow:
    Client                              Server
      │                                   │
      │  POST /api/v1/chat                │
      │  Accept: text/event-stream        │
      │──────────────────────────────────▶│
      │                                   │── Search Qdrant (200ms)
      │                                   │── Build prompt
      │  event: status                    │
      │  data: {"step":"searching"}       │
      │◀──────────────────────────────────│
      │                                   │── LLM starts generating...
      │  event: token                     │
      │  data: {"text":"VinFast "}        │
      │◀──────────────────────────────────│
      │  event: token                     │
      │  data: {"text":"nhận "}           │
      │◀──────────────────────────────────│
      │  event: token                     │
      │  data: {"text":"nhiều "}          │
      │◀──────────────────────────────────│
      │  ...                              │── (streaming từng token)
      │                                   │
      │  event: citations                 │
      │  data: [{"id":"a_001",...}]       │── Gửi citations sau khi answer xong
      │◀──────────────────────────────────│
      │  event: done                      │
      │  data: {"message_id":"msg_456"}   │── Signal hoàn tất
      │◀──────────────────────────────────│
```

**Tác động đến codebase:**

```
Thay đổi:
    1. pkg/gemini:  Thêm method GenerateStream() trả về channel thay vì string
       func (g *geminiImpl) GenerateStream(ctx, prompt) (<-chan StreamChunk, error)

    2. chat.UseCase:  Thêm method ChatStream() bên cạnh Chat()
       - Search + build prompt: giống Chat()
       - Gọi gemini.GenerateStream() thay vì Generate()
       - Trả về channel cho delivery layer

    3. chat delivery/http:
       - handlers.go:  Thêm handler ChatSSE()
       - Check Accept header → route tới ChatSSE hoặc Chat
       - ChatSSE: set Content-Type: text/event-stream, flush từng chunk

    4. Lưu message:
       - Accumulate full answer từ stream
       - Sau khi stream done → save full answer + citations vào PostgreSQL
       - (Không save từng token)

Cấu trúc file mới:
    internal/chat/
    └── usecase/
        ├── chat.go              # Chat() - trả về đầy đủ (giữ nguyên)
        └── chat_stream.go       # ChatStream() - trả về channel

    internal/chat/
    └── delivery/http/
        ├── handlers.go          # Chat handler (JSON)
        └── handlers_stream.go   # ChatSSE handler (SSE)
```

#### P3.2 Qdrant Payload Optimization

**Vấn đề:** Lưu toàn bộ analytics post vào payload. Bài post dài 5000 từ + metadata → payload nặng. Search trả 50 records = network I/O lớn.

**Giải pháp: Selective field retrieval + content truncation**

```
Chiến lược:

1. Khi INDEX: vẫn lưu TẤT CẢ fields vào payload (không mất data).
   Nhưng content lưu tối đa 1000 ký tự đầu (đủ cho RAG context).
   Content đầy đủ vẫn nằm ở Analytics Service DB nếu cần.

2. Khi SEARCH: chỉ fetch fields cần thiết, không fetch toàn bộ payload.

   Qdrant hỗ trợ `with_payload` selector:

   // Cho RAG search (chat): cần content + metadata cho LLM context
   POST /collections/smap_analytics/points/query
   {
       "query": vector,
       "with_payload": {
           "include": [
               "content", "project_id", "platform",
               "overall_sentiment", "aspects", "keywords",
               "metadata.author", "content_created_at"
           ]
       },
       "limit": 10
   }

   // Cho filter-only search (aggregation): không cần content
   POST /collections/smap_analytics/points/query
   {
       "query": vector,
       "with_payload": {
           "include": ["project_id", "overall_sentiment", "aspects"]
       },
       "limit": 50
   }

3. Tác động codebase:
   - search repository/qdrant: thêm param SelectFields vào SearchOptions
   - search usecase: chọn fields phù hợp theo use case (chat vs aggregation)
   - indexing usecase: truncate content tại 1000 chars khi prepare payload
```

---

### P4. Operational & Maintainability (Vận hành)

#### P4.1 Qdrant Collection Versioning (Model Drift)

**Vấn đề:** Đổi embedding model (Voyage → OpenAI). Vector cũ và mới không tương thích. Cần re-index zero-downtime.

**Giải pháp: Collection aliasing + background re-index**

```
Cấu trúc:
    Qdrant Collections:
    ├── smap_analytics_v1    ← Voyage AI vectors (hiện tại)
    ├── smap_analytics_v2    ← OpenAI vectors (đang re-index)
    └── smap_analytics       ← ALIAS → trỏ tới v1 (hoặc v2 sau khi xong)

Flow đổi model:
    Phase 1: Chuẩn bị
        1. Tạo collection mới: smap_analytics_v2 (size phù hợp model mới)
        2. Config trong service: embedding_model = "openai" (cho writes mới)
        3. Alias vẫn trỏ v1 (reads vẫn dùng v1)

    Phase 2: Dual-write
        4. Index data MỚI vào CẢ v1 và v2 (song song)
        5. Background job: re-index toàn bộ data cũ vào v2
           - Query knowledge.indexed_documents để lấy list analytics_ids
           - Với mỗi record: lấy content từ Qdrant v1 payload
                             → embed bằng model mới
                             → upsert vào v2
           - Track progress trong DB

    Phase 3: Switch
        6. Khi v2 đã index xong 100% data:
           - Update alias: smap_analytics → v2
           - Stop dual-write (chỉ write v2)
           - Verify: so sánh count v1 vs v2

    Phase 4: Cleanup
        7. Giữ v1 thêm 7 ngày (rollback nếu cần)
        8. Xóa v1 sau 7 ngày

Zero downtime:
    - Reads luôn đi qua alias → tự động switch khi alias thay đổi
    - Writes dual-write trong giai đoạn chuyển tiếp
    - Không cần restart service
```

**Config trong service:**

```yaml
qdrant:
  collection_alias: "smap_analytics" # Reads luôn dùng alias
  write_collections: # Writes có thể ghi nhiều collection
    - "smap_analytics_v2"
  migration:
    enabled: true
    source_collection: "smap_analytics_v1"
    target_collection: "smap_analytics_v2"
    batch_size: 100
    concurrency: 5
```

#### P4.2 Schema Evolution (Thêm fields mới)

**Vấn đề:** Analytics Service thêm field `toxicity_score`. Cần update cho 1 triệu documents cũ.

**Giải pháp: Flexible payload + Backfill job**

```
Tại sao Qdrant không bị ảnh hưởng nặng?
    - Qdrant payload là schemaless JSON. Thêm field mới = thêm key vào JSON.
    - Documents cũ không có field mới → filter trên field đó = bỏ qua docs cũ.
    - KHÔNG cần xóa trắng index.

Khi thêm field mới (ví dụ toxicity_score):

    1. Update code indexing:
       - Thêm toxicity_score vào payload khi index records MỚI.
       - Records mới sẽ có field, records cũ không có → OK.

    2. Backfill job (nếu cần field cho records cũ):
       - Chạy background job đọc từ knowledge.indexed_documents
       - Với mỗi record: gọi Qdrant.SetPayload() để thêm field mới
       - KHÔNG cần re-embed (vector không thay đổi)
       - KHÔNG cần xóa point cũ

       Qdrant API:
       POST /collections/smap_analytics/points/payload
       {
           "payload": { "toxicity_score": 0.05 },
           "points": ["analytics_001", "analytics_002", ...]  // batch update
       }

    3. Payload mapping linh hoạt trong code:
       - Dùng map[string]interface{} cho "dynamic fields" thay vì struct cứng
       - Có config map biết fields nào cần index vào payload

Indexing UseCase:
    // Phần cứng: fields luôn có
    payload := map[string]interface{}{
        "project_id":              record.ProjectID,
        "content":                 truncate(record.Content, 1000),
        "overall_sentiment":       record.OverallSentiment,
        "overall_sentiment_score": record.OverallSentimentScore,
        "aspects":                 record.Aspects,
        "keywords":                record.Keywords,
        // ...
    }

    // Phần mềm: fields có thể thêm/bớt theo config hoặc theo data
    if record.ToxicityScore != nil {
        payload["toxicity_score"] = *record.ToxicityScore
    }
    if record.Entities != nil {
        payload["entities"] = record.Entities
    }
    // → Records cũ (không có toxicity_score) vẫn index bình thường
    // → Records mới (có toxicity_score) tự động thêm field
```

---

### P5. Report Generation (Domain Report)

#### P5.1 Report Locking (Chống duplicate)

**Vấn đề:** User bấm "Generate" 5 lần liên tiếp → 5 background jobs chạy giống nhau, tốn 5x tiền LLM.

**Giải pháp: Dedup by params hash + return existing**

```go
// usecase/report.go
func (uc *implUseCase) Generate(ctx context.Context, sc model.Scope, input GenerateInput) (GenerateOutput, error) {
    // 1. Tạo hash từ parameters (campaign_id + report_type + filters + date range)
    paramsHash := sha256(fmt.Sprintf("%s:%s:%s",
        input.CampaignID, input.ReportType, jsonMarshal(input.Filters)))

    // 2. Check xem đã có report PROCESSING với cùng params chưa
    existing, _ := uc.repo.FindByParamsHash(ctx, paramsHash, "PROCESSING")
    if existing != nil {
        // Đã có job đang chạy → trả về ID cũ, KHÔNG tạo job mới
        return GenerateOutput{
            ReportID: existing.ID,
            Status:   "PROCESSING",
            Message:  "Báo cáo đang được tạo, vui lòng đợi...",
        }, nil
    }

    // 3. Check report COMPLETED gần đây (< 1 giờ) với cùng params
    recent, _ := uc.repo.FindByParamsHash(ctx, paramsHash, "COMPLETED")
    if recent != nil && time.Since(recent.CompletedAt) < 1*time.Hour {
        // Report mới tạo < 1 giờ trước → trả về luôn, không tạo lại
        return GenerateOutput{
            ReportID: recent.ID,
            Status:   "COMPLETED",
            Message:  "Báo cáo đã có sẵn.",
        }, nil
    }

    // 4. Tạo report record mới
    report := Report{
        CampaignID: input.CampaignID,
        ReportType: input.ReportType,
        ParamsHash: paramsHash,
        Status:     "PROCESSING",
    }
    uc.repo.Create(ctx, &report)

    // 5. Chạy background job
    go uc.generateInBackground(context.Background(), report)

    return GenerateOutput{
        ReportID: report.ID,
        Status:   "PROCESSING",
    }, nil
}
```

**Thêm `params_hash` vào reports table:**

```sql
ALTER TABLE knowledge.reports ADD COLUMN params_hash VARCHAR(64) NOT NULL;
CREATE INDEX idx_reports_params_hash ON knowledge.reports(params_hash, status);
```

#### P5.2 Map-Reduce Summarization (Cho báo cáo lớn)

**Vấn đề:** Report cần tổng hợp 1000+ bài post. Context window LLM không chứa hết. RAG top-10 không đủ đại diện.

**Giải pháp: 3-phase Map-Reduce thay vì RAG thuần túy**

```
Chiến lược: "Aggregate numbers from Qdrant, Sample docs for LLM"

Phase 1: AGGREGATE (Qdrant aggregation, không cần LLM)
    │
    │  Dùng Qdrant Count/Scroll API để lấy thống kê:
    │  - Tổng số posts per project
    │  - Phân bố sentiment: {POSITIVE: 450, NEGATIVE: 320, NEUTRAL: 180, MIXED: 150}
    │  - Phân bố aspects: {DESIGN: 500, BATTERY: 320, PRICE: 280, ...}
    │  - Average scores per aspect
    │  - Trends by date (count per week)
    │  → Kết quả: JSON chứa toàn bộ thống kê định lượng
    │
    ▼
Phase 2: SAMPLE (Lấy mẫu đại diện, không cần LLM)
    │
    │  Với mỗi nhóm quan trọng, lấy 3-5 bài tiêu biểu:
    │  - Top 3 bài POSITIVE có engagement_score cao nhất (aspect: DESIGN)
    │  - Top 3 bài NEGATIVE có risk_score cao nhất (aspect: BATTERY)
    │  - Top 3 bài NEGATIVE có engagement_score cao nhất (aspect: PRICE)
    │  - ...
    │  → Kết quả: 20-30 bài tiêu biểu (fit trong context window)
    │
    ▼
Phase 3: GENERATE (LLM tạo nội dung report)
    │
    │  Gửi cho LLM:
    │  - Thống kê tổng hợp từ Phase 1 (numbers, percentages)
    │  - Bài mẫu từ Phase 2 (representative examples)
    │  - Template cho loại report (SUMMARY/COMPARISON/TREND/...)
    │  → LLM sinh nội dung dựa trên DATA THỰC, không bịa
    │
    │  Nếu report phức tạp (COMPARISON), chia thành sections:
    │  - Section 1: Executive Summary → 1 LLM call
    │  - Section 2: Sentiment Analysis → 1 LLM call
    │  - Section 3: Aspect Comparison → 1 LLM call
    │  - Section 4: Recommendations → 1 LLM call
    │  → 4 calls nhỏ thay vì 1 call khổng lồ (map-reduce)
    │
    ▼
Phase 4: COMPILE
    │  Ghép các sections → Format markdown → Convert PDF → Upload MinIO
```

```go
// usecase/generator.go
func (uc *implUseCase) generateInBackground(ctx context.Context, report Report) {
    // Phase 1: Aggregate
    stats, _ := uc.searchUC.Aggregate(ctx, AggregateInput{
        CampaignID: report.CampaignID,
        GroupBy:    []string{"overall_sentiment", "aspects.aspect", "platform"},
        DateRange:  report.Filters.DateRange,
    })

    // Phase 2: Sample representative docs
    samples := map[string][]SearchResult{}
    for _, aspect := range stats.TopAspects {
        // Lấy top 3 positive + top 3 negative cho mỗi aspect
        posDocs, _ := uc.searchUC.Search(ctx, SearchInput{
            CampaignID: report.CampaignID,
            Filters:    SearchFilters{Aspects: []string{aspect}, Sentiment: "POSITIVE"},
            Limit:      3,
            SortBy:     "engagement_score",
        })
        negDocs, _ := uc.searchUC.Search(ctx, SearchInput{
            CampaignID: report.CampaignID,
            Filters:    SearchFilters{Aspects: []string{aspect}, Sentiment: "NEGATIVE"},
            Limit:      3,
            SortBy:     "risk_score",
        })
        samples[aspect] = append(posDocs.Results, negDocs.Results...)
    }

    // Phase 3: Generate sections (map)
    sections := []ReportSection{}
    for _, sectionTemplate := range reportTemplates[report.ReportType] {
        prompt := buildSectionPrompt(sectionTemplate, stats, samples)
        content, _ := uc.llm.Generate(ctx, prompt)
        sections = append(sections, ReportSection{
            Title:   sectionTemplate.Title,
            Content: content,
        })
    }

    // Phase 4: Compile (reduce)
    markdown := compileReport(report.Title, sections, stats)
    pdf, _ := convertToPDF(markdown)
    fileURL, _ := uc.minio.Upload(ctx, pdf, reportPath)

    // Update status
    uc.repo.UpdateCompleted(ctx, report.ID, fileURL)
}
```

**Thêm method `Aggregate()` vào search.UseCase interface:**

```go
// internal/search/interface.go
type UseCase interface {
    Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)
    Aggregate(ctx context.Context, input AggregateInput) (AggregateOutput, error)  // MỚI
}

// internal/search/types.go
type AggregateInput struct {
    CampaignID string
    GroupBy    []string        // ["overall_sentiment", "aspects.aspect", "platform"]
    DateRange  *DateRange
}

type AggregateOutput struct {
    TotalDocs       int
    BySentiment     map[string]int          // {"POSITIVE": 450, "NEGATIVE": 320, ...}
    ByAspect        map[string]AspectStats  // {"DESIGN": {pos: 400, neg: 100, avg: 0.72}, ...}
    ByPlatform      map[string]int          // {"tiktok": 800, "youtube": 400, ...}
    TopAspects      []string                // Sorted by mention count
    TrendByWeek     []WeeklyTrend           // [{week: "2026-W06", pos: 50, neg: 30}, ...]
}
```
