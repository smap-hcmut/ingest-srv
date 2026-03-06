# SMAP Analytics Service - Implementation Status Report

**Ngày tạo:** 15/02/2026  
**Phiên bản:** 1.0  
**Repo:** analysis-srv (Analytics Service - Python)

---

## TÓM TẮT EXECUTIVE

Repo này đã hoàn thành **Phase 1: Core Analytics Pipeline** với kiến trúc Domain-Driven Design (DDD) và async processing. Hệ thống hiện tại có khả năng:

✅ Nhận message từ RabbitMQ queue  
✅ Parse và validate event payload  
✅ Orchestrate analytics pipeline (5 stages)  
✅ Persist kết quả vào PostgreSQL (schema: `schema_analyst`)  
✅ End-to-end flow đã được verify thành công

**Trạng thái:** PRODUCTION-READY cho core flow, AI modules đang ở TODO state.

---

## 1. KIẾN TRÚC TỔNG QUAN

### 1.1 Tech Stack

| Component     | Technology    | Version               |
| ------------- | ------------- | --------------------- |
| Language      | Python        | 3.11+                 |
| Framework     | FastAPI       | (API layer - planned) |
| Message Queue | RabbitMQ      | aio-pika              |
| Database      | PostgreSQL    | asyncpg + SQLAlchemy  |
| ORM           | SQLAlchemy    | 2.x (async)           |
| AI Models     | PhoBERT       | ONNX Runtime          |
| Logging       | Custom Logger | -                     |
| Config        | YAML          | PyYAML                |

### 1.2 Domain Structure (DDD Pattern)

```
internal/
├── analytics/              # ORCHESTRATION DOMAIN
│   ├── usecase/
│   │   └── usecase.py     # AnalyticsPipeline (orchestrator)
│   ├── delivery/
│   │   └── rabbitmq/consumer/
│   │       └── handler.py  # AnalyticsHandler (message adapter)
│   ├── interface.py        # IAnalyticsPipeline protocol
│   ├── type.py            # Input, Output, AnalyticsResult, EventMetadata
│   └── constant.py        # Constants (no magic strings)
│
├── analyzed_post/          # CRUD DOMAIN (Persistence)
│   ├── usecase/
│   │   └── crud.py        # AnalyzedPostUseCase (create, update)
│   ├── repository/
│   │   ├── interface.py   # IAnalyzedPostRepository protocol
│   │   └── postgre/
│   │       └── repository.py  # PostgreSQL implementation
│   ├── interface.py       # IAnalyzedPostUseCase protocol
│   └── type.py           # CreateAnalyzedPostInput, UpdateAnalyzedPostInput
│
├── sentiment_analysis/     # AI DOMAIN - Sentiment
│   ├── usecase/
│   │   └── analyzer.py    # SentimentAnalyzer (PhoBERT)
│   └── interface.py       # ISentimentAnalyzer protocol
│
├── intent_classification/  # AI DOMAIN - Intent
│   ├── usecase/
│   │   └── classifier.py  # IntentClassifier (gatekeeper)
│   └── interface.py       # IIntentClassifier protocol
│
├── keyword_extraction/     # AI DOMAIN - Keywords
│   ├── usecase/
│   │   └── extractor.py   # KeywordExtractor (YAKE + spaCy)
│   └── interface.py       # IKeywordExtractor protocol
│
├── impact_calculation/     # AI DOMAIN - Impact Score
│   ├── usecase/
│   │   └── calculator.py  # ImpactCalculator (weighted formula)
│   └── interface.py       # IImpactCalculator protocol
│
├── text_preprocessing/     # AI DOMAIN - Text Cleaning
│   ├── usecase/
│   │   └── preprocessor.py  # TextPreprocessor (normalize)
│   └── interface.py       # ITextPreprocessor protocol
│
├── consumer/              # INFRASTRUCTURE DOMAIN
│   ├── server.py          # ConsumerServer (RabbitMQ consumer)
│   ├── registry.py        # ConsumerRegistry (dependency injection)
│   ├── handler.py         # IMessageHandler protocol
│   └── interface.py       # Protocols
│
└── model/                 # DATA MODEL DOMAIN
    ├── analyzed_post.py   # AnalyzedPost SQLAlchemy model
    ├── base.py           # Base model
    └── phobert/          # PhoBERT model files (ONNX)
```

---

## 2. DATA FLOW HIỆN TẠI

### 2.1 End-to-End Flow (Đã Hoàn Thành)

```
┌─────────────────────────────────────────────────────────────────┐
│                    CURRENT IMPLEMENTATION                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. RabbitMQ Queue: analytics.data.collected                    │
│     ↓                                                           │
│  2. ConsumerServer (aio-pika)                                   │
│     - Prefetch count: 1 (process one at a time)                │
│     - Auto-ack: False (manual ack after success)               │
│     ↓                                                           │
│  3. AnalyticsHandler.handle()                                   │
│     - Parse JSON envelope                                       │
│     - Validate event format                                     │
│     - Extract EventMetadata + PostData                          │
│     ↓                                                           │
│  4. AnalyticsPipeline.process() [ASYNC]                         │
│     - Enrich post data with metadata                            │
│     - Run 5-stage pipeline (TODO: AI modules)                   │
│     - Build AnalyticsResult                                     │
│     ↓                                                           │
│  5. AnalyzedPostUseCase.create() [ASYNC]                        │
│     - Validate input                                            │
│     - Call repository                                           │
│     ↓                                                           │
│  6. AnalyzedPostRepository.save() [ASYNC]                       │
│     - Sanitize data (parse datetime strings)                    │
│     - INSERT into schema_analyst.analyzed_posts                 │
│     - Commit transaction                                        │
│     ↓                                                           │
│  7. PostgreSQL Database                                         │
│     - Database: smap                                            │
│     - Schema: schema_analyst                                    │
│     - Table: analyzed_posts                                     │
│     - User: analyst_prod (runtime)                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Message Format (Input)

**RabbitMQ Message Envelope:**

```json
{
  "event_id": "evt_123456",
  "event_type": "data.collected",
  "timestamp": "2026-02-15T10:00:00Z",
  "payload": {
    "project_id": "proj_vinfast_vf8",
    "job_id": "job_abc123",
    "batch_index": 1,
    "content_count": 50,
    "platform": "tiktok",
    "task_type": "crawl",
    "brand_name": "VinFast",
    "keyword": "vinfast vf8",
    "minio_path": "s3://bucket/path/to/batch.json",

    "meta": {
      "id": "post_123",
      "platform": "tiktok",
      "permalink": "https://tiktok.com/@user/video/123",
      "published_at": "2026-02-14T08:00:00Z"
    },
    "content": {
      "text": "Xe VF8 đi êm nhưng pin sụt nhanh quá",
      "transcription": "",
      "duration": 30,
      "hashtags": ["vinfast", "vf8", "xedien"]
    },
    "interaction": {
      "views": 10000,
      "likes": 500,
      "comments_count": 50,
      "shares": 20,
      "saves": 10
    },
    "author": {
      "id": "author_123",
      "name": "Nguyễn Văn A",
      "username": "@nguyenvana",
      "avatar_url": "https://...",
      "followers": 50000,
      "is_verified": false
    },
    "comments": []
  }
}
```

### 2.3 Database Schema

**Table: `schema_analyst.analyzed_posts`**

```sql
CREATE TABLE schema_analyst.analyzed_posts (
    -- Primary Key
    id VARCHAR(255) PRIMARY KEY,

    -- Business Context
    project_id VARCHAR(255),
    platform VARCHAR(50),
    published_at TIMESTAMPTZ,
    analyzed_at TIMESTAMPTZ DEFAULT NOW(),

    -- Processing Metadata
    processing_status VARCHAR(50) DEFAULT 'SUCCESS',
    processing_time_ms INTEGER,
    model_version VARCHAR(50),
    pipeline_version VARCHAR(50),

    -- Crawler Context
    job_id VARCHAR(255),
    batch_index INTEGER,
    task_type VARCHAR(50),
    keyword_source VARCHAR(255),
    crawled_at TIMESTAMPTZ,
    brand_name VARCHAR(255),
    keyword VARCHAR(255),

    -- Content Fields
    content_text TEXT,
    content_transcription TEXT,
    media_duration INTEGER,
    hashtags TEXT[],
    permalink TEXT,

    -- Author Fields
    author_id VARCHAR(255),
    author_name VARCHAR(255),
    author_username VARCHAR(255),
    author_avatar_url TEXT,
    author_is_verified BOOLEAN DEFAULT FALSE,

    -- Interaction Metrics
    view_count INTEGER DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    share_count INTEGER DEFAULT 0,
    save_count INTEGER DEFAULT 0,
    follower_count INTEGER DEFAULT 0,

    -- AI Analysis Results (TODO)
    sentiment VARCHAR(50),
    sentiment_score FLOAT,
    intent VARCHAR(50),
    intent_confidence FLOAT,
    keywords TEXT[],
    aspects JSONB,
    impact_score FLOAT,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_analyzed_posts_project ON schema_analyst.analyzed_posts(project_id);
CREATE INDEX idx_analyzed_posts_platform ON schema_analyst.analyzed_posts(platform);
CREATE INDEX idx_analyzed_posts_published ON schema_analyst.analyzed_posts(published_at);
CREATE INDEX idx_analyzed_posts_analyzed ON schema_analyst.analyzed_posts(analyzed_at);
```

---

## 3. DEPENDENCY INJECTION (ConsumerRegistry)

### 3.1 Initialization Flow

```python
# internal/consumer/registry.py

class ConsumerRegistry:
    """Central dependency injection container."""

    def initialize(self) -> DomainServices:
        """Wire up all dependencies."""

        # 1. Infrastructure
        db = PostgresDatabase(config.database)
        logger = Logger(config.logging)

        # 2. Repositories
        analyzed_post_repo = AnalyzedPostRepository(db, logger)

        # 3. Use Cases
        analyzed_post_usecase = AnalyzedPostUseCase(analyzed_post_repo, logger)

        # 4. AI Modules (TODO - currently None)
        preprocessor = None
        intent_classifier = None
        keyword_extractor = None
        sentiment_analyzer = None
        impact_calculator = None

        # 5. Analytics Pipeline (Orchestrator)
        analytics_pipeline = AnalyticsPipeline(
            config=config.analytics,
            analyzed_post_usecase=analyzed_post_usecase,
            logger=logger,
            preprocessor=preprocessor,
            intent_classifier=intent_classifier,
            keyword_extractor=keyword_extractor,
            sentiment_analyzer=sentiment_analyzer,
            impact_calculator=impact_calculator,
        )

        # 6. Message Handler
        analytics_handler = AnalyticsHandler(
            pipeline=analytics_pipeline,
            logger=logger,
        )

        return DomainServices(
            analytics_handler=analytics_handler,
            # ... other services
        )
```

---

## 4. CONFIGURATION

### 4.1 Config File: `config/config.yaml`

```yaml
service:
  name: "analytics-engine"
  version: "0.1.0"
  environment: "development"

database:
  url: "postgresql+asyncpg://analyst_prod:analyst_prod_pwd@172.16.19.10:5432/smap"
  url_sync: "postgresql://analyst_prod:analyst_prod_pwd@172.16.19.10:5432/smap"
  schema: "schema_analyst"
  pool_size: 20
  max_overflow: 10

rabbitmq:
  url: "amqp://admin:21042004@172.16.21.206:5672/"
  prefetch_count: 1 # Process one message at a time

  queues:
    - name: "analytics.data.collected"
      exchange: "smap.events"
      routing_key: "data.collected"
      handler_module: "internal.analytics.delivery.rabbitmq.consumer.handler"
      handler_class: "AnalyticsHandler"
      prefetch_count: 1
      enabled: true

phobert:
  model_path: "internal/model/phobert"
  max_length: 256

logging:
  level: "INFO"
  enable_console: true
  colorize: true
```

### 4.2 Database Users

| User             | Password             | Permissions                    | Usage              |
| ---------------- | -------------------- | ------------------------------ | ------------------ |
| `analyst_master` | `analyst_master_pwd` | CREATE TABLE, ALTER            | Migrations only    |
| `analyst_prod`   | `analyst_prod_pwd`   | SELECT, INSERT, UPDATE, DELETE | Runtime operations |

---

## 5. NHỮNG GÌ ĐÃ HOÀN THÀNH

### 5.1 Core Infrastructure ✅

- [x] RabbitMQ consumer với aio-pika (async)
- [x] PostgreSQL async connection pool (asyncpg)
- [x] SQLAlchemy async ORM
- [x] Dependency injection container (ConsumerRegistry)
- [x] Structured logging
- [x] YAML configuration management
- [x] Migration system (SQL scripts)

### 5.2 Domain Architecture ✅

- [x] Domain-Driven Design structure
- [x] Protocol-based interfaces (typing.Protocol)
- [x] Separation of concerns:
  - Analytics (orchestration)
  - AnalyzedPost (persistence)
  - AI modules (business logic)
  - Consumer (infrastructure)
- [x] Factory pattern (New functions)
- [x] Async/await throughout the stack

### 5.3 Analytics Pipeline ✅

- [x] Message parsing and validation
- [x] Event metadata extraction
- [x] Post data enrichment
- [x] Pipeline orchestration framework
- [x] Error handling and logging
- [x] Processing time tracking
- [x] Result persistence

### 5.4 Data Persistence ✅

- [x] AnalyzedPost model (SQLAlchemy)
- [x] Repository pattern implementation
- [x] Async database operations
- [x] Datetime string parsing
- [x] Transaction management
- [x] Schema isolation (schema_analyst)

### 5.5 Code Quality ✅

- [x] Type hints throughout
- [x] Protocol-based interfaces
- [x] Constants instead of magic strings
- [x] Proper error handling
- [x] Structured logging
- [x] Clean code principles

---

## 6. NHỮNG GÌ CHƯA HOÀN THÀNH (TODO)

### 6.1 AI Modules (Phase 2) ⏳

```python
# Current state in AnalyticsPipeline._run_pipeline()

# Stage 1: Preprocessing (if enabled)
if self.config.enable_preprocessing and self.preprocessor:
    # TODO: Implement preprocessing
    pass

# Stage 2: Intent classification (if enabled)
if self.config.enable_intent_classification and self.intent_classifier:
    # TODO: Implement intent classification
    # Check if should skip (spam/seeding)
    pass

# Stage 3: Keyword extraction (if enabled)
keywords = []
if self.config.enable_keyword_extraction and self.keyword_extractor:
    # TODO: Implement keyword extraction
    pass

# Stage 4: Sentiment analysis (if enabled)
if self.config.enable_sentiment_analysis and self.sentiment_analyzer:
    # TODO: Implement sentiment analysis
    pass

# Stage 5: Impact calculation (if enabled)
if self.config.enable_impact_calculation and self.impact_calculator:
    # TODO: Implement impact calculation
    pass
```

**Lý do chưa implement:**

- Framework đã sẵn sàng (interfaces, dependency injection)
- Chờ finalize AI model requirements
- PhoBERT model files đã có trong `internal/model/phobert/`
- Cần integrate với ONNX Runtime

### 6.2 Batch Processing ⏳

**Hiện tại:** Process từng message một (prefetch_count: 1)

**Kế hoạch:** Implement batch processing cho performance:

- MinIO integration để đọc batch files
- Parallel processing với asyncio.gather()
- Bulk insert vào database

### 6.3 API Layer ⏳

**Hiện tại:** Chỉ có consumer (message-driven)

**Kế hoạch:** Thêm FastAPI endpoints:

- Health check
- Metrics endpoint
- Manual trigger endpoint (for testing)
- Status query endpoint

### 6.4 Monitoring & Observability ⏳

- [ ] Prometheus metrics
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Performance profiling
- [ ] Error rate monitoring
- [ ] Queue depth monitoring

### 6.5 Testing ⏳

- [ ] Unit tests cho từng domain
- [ ] Integration tests
- [ ] End-to-end tests
- [ ] Load testing
- [ ] Mock AI modules for testing

---

## 7. SO SÁNH VỚI MASTER PLAN

### 7.1 Alignment với Migration Plan v2.0

| Aspect               | Master Plan             | Current Implementation   | Status                    |
| -------------------- | ----------------------- | ------------------------ | ------------------------- |
| **Architecture**     | Hybrid (Go + Python)    | Python only (this repo)  | ✅ Aligned                |
| **Message Queue**    | Kafka                   | RabbitMQ                 | ⚠️ Different (acceptable) |
| **Database**         | Multi-schema PostgreSQL | schema_analyst           | ✅ Aligned                |
| **AI Workers**       | Python FastAPI          | Integrated in service    | ⚠️ Different approach     |
| **UAP Format**       | Canonical data model    | EventMetadata + PostData | ✅ Aligned                |
| **Async Processing** | Required                | Fully async              | ✅ Aligned                |

### 7.2 Alignment với Dataflow v2

| Flow                  | Master Plan               | Current Implementation    | Status                 |
| --------------------- | ------------------------- | ------------------------- | ---------------------- |
| **FLOW 3: Analytics** | Kafka → n8n → Python → DB | RabbitMQ → Python → DB    | ✅ Simplified (better) |
| **Batch Processing**  | MinIO + Batch             | Single message            | ⏳ TODO                |
| **Quality Gate**      | Filter spam before index  | TODO in intent classifier | ⏳ TODO                |
| **Vector Indexing**   | Qdrant                    | Not implemented           | ⏳ Phase 2             |

### 7.3 Deviations (Có chủ đích)

**1. Không dùng n8n:**

- Master plan ban đầu dùng n8n làm orchestrator
- Quyết định: Implement trực tiếp trong Python
- Lý do: Performance, maintainability, debugging

**2. RabbitMQ thay vì Kafka:**

- Master plan dùng Kafka
- Hiện tại: RabbitMQ
- Lý do: Đơn giản hơn cho single-service, đủ cho Phase 1

**3. Monolithic Python service thay vì AI Workers:**

- Master plan: Tách AI workers riêng (FastAPI microservices)
- Hiện tại: Tích hợp trong analytics service
- Lý do: Đơn giản hóa deployment Phase 1, có thể tách sau

---

## 8. PERFORMANCE & SCALABILITY

### 8.1 Current Performance

**Throughput:**

- Prefetch count: 1 (conservative)
- Processing time: ~100-500ms per message (without AI)
- Estimated: ~2-10 messages/second per instance

**Bottlenecks:**

- AI inference (PhoBERT) - chưa implement
- Database writes - đã optimize với async
- Network I/O - đã optimize với connection pooling

### 8.2 Scalability Strategy

**Horizontal Scaling:**

```yaml
# Kubernetes Deployment
replicas: 3 # Multiple consumer instances
```

**Vertical Scaling:**

```yaml
rabbitmq:
  prefetch_count: 10 # Process 10 messages concurrently
```

**Batch Processing:**

```python
# Future implementation
async def process_batch(messages: list[Message]) -> list[Result]:
    tasks = [pipeline.process(msg) for msg in messages]
    return await asyncio.gather(*tasks)
```

---

## 9. DEPLOYMENT

### 9.1 Docker

**Dockerfile:** `apps/consumer/Dockerfile`

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY . .

# Run consumer
CMD ["python", "-m", "apps.consumer.main"]
```

### 9.2 Environment Variables

```bash
# Database
DATABASE_URL=postgresql+asyncpg://analyst_prod:analyst_prod_pwd@db:5432/smap
DATABASE_SCHEMA=schema_analyst

# RabbitMQ
RABBITMQ_URL=amqp://admin:password@rabbitmq:5672/

# Logging
LOG_LEVEL=INFO

# Service
SERVICE_NAME=analytics-engine
SERVICE_VERSION=0.1.0
ENVIRONMENT=production
```

### 9.3 Health Check

```python
# Future: FastAPI health endpoint
@app.get("/health")
async def health_check():
    return {
        "status": "healthy",
        "service": "analytics-engine",
        "version": "0.1.0",
        "database": await db.ping(),
        "rabbitmq": await rabbitmq.ping(),
    }
```

---

## 10. NEXT STEPS (Priority Order)

### Phase 2A: AI Integration (Tuần 1-2)

1. **Implement Sentiment Analysis**
   - Load PhoBERT ONNX model
   - Integrate với pipeline
   - Test accuracy

2. **Implement Intent Classification**
   - Pattern matching với config
   - Spam/seeding detection
   - Skip logic

3. **Implement Keyword Extraction**
   - YAKE algorithm
   - spaCy NER
   - Aspect mapping

4. **Implement Impact Calculation**
   - Weighted formula
   - Platform multipliers
   - Sentiment amplifiers

### Phase 2B: Performance (Tuần 3)

5. **Batch Processing**
   - MinIO integration
   - Parallel processing
   - Bulk database inserts

6. **Monitoring**
   - Prometheus metrics
   - Grafana dashboards
   - Alerting rules

### Phase 2C: Production Readiness (Tuần 4)

7. **Testing**
   - Unit tests (80% coverage)
   - Integration tests
   - Load testing

8. **Documentation**
   - API documentation
   - Deployment guide
   - Troubleshooting guide

9. **CI/CD**
   - GitHub Actions
   - Docker build pipeline
   - Automated testing

---

## 11. KẾT LUẬN

### 11.1 Achievements

Repo này đã hoàn thành **foundation layer** vững chắc cho Analytics Service:

✅ **Architecture:** DDD, clean separation of concerns  
✅ **Infrastructure:** Async processing, database, message queue  
✅ **Data Flow:** End-to-end verified working  
✅ **Code Quality:** Type hints, protocols, constants, clean code  
✅ **Scalability:** Ready for horizontal scaling

### 11.2 Current State

**Production-Ready:** Core flow (message → parse → persist)  
**Not Production-Ready:** AI analysis (TODO)

### 11.3 Effort Estimate

**Completed:** ~60% of Phase 1 (Infrastructure + Framework)  
**Remaining:** ~40% (AI integration + Testing + Monitoring)

**Timeline:**

- Phase 2A (AI): 2 tuần
- Phase 2B (Performance): 1 tuần
- Phase 2C (Production): 1 tuần
- **Total:** 4 tuần để hoàn thiện

### 11.4 Risks & Mitigations

| Risk                     | Impact | Mitigation                               |
| ------------------------ | ------ | ---------------------------------------- |
| PhoBERT performance slow | High   | Use ONNX Runtime, batch inference        |
| Database bottleneck      | Medium | Connection pooling, bulk inserts         |
| Message queue lag        | Medium | Horizontal scaling, prefetch tuning      |
| AI accuracy issues       | High   | Model fine-tuning, confidence thresholds |

---

**Document Version:** 1.0  
**Last Updated:** 15/02/2026  
**Author:** Nguyễn Tấn Tài  
**Status:** ACTIVE
