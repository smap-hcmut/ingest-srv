# SMAP Analytics Service — Analysis Document

## 1. Tổng quan

SMAP Analytics Service là một microservice event-driven, đóng vai trò **analytics engine** trong hệ thống SMAP. Service nhận dữ liệu bài viết mạng xã hội thô từ RabbitMQ (do crawler/n8n publish), chạy qua pipeline NLP 5 giai đoạn, và lưu kết quả phân tích vào PostgreSQL.

### Vai trò trong hệ thống

```
Crawler / n8n / API
        │
        ▼
   RabbitMQ (smap.events / data.collected)
        │
        ▼
   ┌─────────────────────────────────┐
   │   SMAP Analytics Service        │
   │   (Consumer → Pipeline → DB)    │
   └─────────────────────────────────┘
        │           │          │
        ▼           ▼          ▼
   PostgreSQL    Redis      MinIO
   (analyzed     (cache)    (batch raw
    _posts)                  payloads)
```

Service hoạt động như một **RabbitMQ consumer** đơn thuần — không expose REST API (API layer đang planned). Entry point duy nhất là consumer lắng nghe queue `analytics.data.collected` với routing key `data.collected` trên exchange `smap.events`.

---

## 2. Business Logic

### 2.1 Luồng xử lý chính

1. **Message Reception**: Consumer nhận message từ RabbitMQ queue `analytics.data.collected`.
2. **Validation & Parsing**: `AnalyticsHandler` validate JSON envelope, extract `EventMetadata` và `PostData`.
3. **Enrichment**: Pipeline enriches post data với metadata từ event (job_id, batch_index, task_type, brand_name, keyword, crawled_at, pipeline_version).
4. **5-Stage Pipeline**: Chạy tuần tự qua 5 giai đoạn phân tích.
5. **Persistence**: Kết quả được lưu vào `schema_analyst.analyzed_posts` qua async SQLAlchemy.
6. **Ack/Nack**: Message được ack nếu thành công, nack nếu lỗi.

### 2.2 Pipeline 5 giai đoạn

Mỗi giai đoạn có thể bật/tắt độc lập qua config:

#### Stage 1: Text Preprocessing

- Làm sạch và chuẩn hóa text từ bài viết và transcription.
- Loại bỏ URL, emoji, normalize Unicode (NFKC).
- Chuyển đổi teencode tiếng Việt (ko → không, dc → được, ...).
- Phát hiện số điện thoại VN, spam keywords.
- Tính toán stats: tổng độ dài, hashtag ratio, reduction ratio.
- Gộp caption + transcription + top comments (sắp xếp theo likes).

#### Stage 2: Intent Classification

- Phân loại ý định bài viết dựa trên regex pattern matching.
- 7 loại intent theo thứ tự ưu tiên:
  | Intent | Priority | Mô tả |
  |--------|----------|-------|
  | CRISIS | 10 | Tẩy chay, lừa đảo, phốt |
  | SEEDING | 9 | Quảng cáo ẩn (inbox giá, zalo, liên hệ mua) |
  | SPAM | 9 | Vay tiền, bán sim, tuyển dụng |
  | COMPLAINT | 7 | Phàn nàn, thất vọng, đừng mua |
  | LEAD | 5 | Hỏi giá, mua ở đâu, đặt cọc |
  | SUPPORT | 4 | Hỏi bảo hành, hướng dẫn sử dụng |
  | DISCUSSION | 1 | Thảo luận chung (default) |
- **Gatekeeping**: Nếu intent là SPAM hoặc SEEDING, `should_skip=true` → bỏ qua các stage AI tốn kém phía sau.

#### Stage 3: Keyword Extraction

- Trích xuất từ khóa và thực thể từ text.
- Hai nguồn:
  - **Dictionary matching** (DICT): So khớp với aspect dictionary (DESIGN, PERFORMANCE, PRICE, SERVICE, GENERAL).
  - **AI extraction** (AI): Sử dụng YAKE (statistical) + spaCy NER (entity recognition).
- Mỗi keyword có: `keyword`, `aspect`, `score`, `source`.
- Giới hạn tối đa 30 keywords (configurable).

#### Stage 4: Sentiment Analysis

- Phân tích cảm xúc sử dụng PhoBERT (ONNX runtime) — model tiếng Việt.
- Output:
  - **Overall sentiment**: label (POSITIVE/NEGATIVE/NEUTRAL), score (-1.0 → 1.0), confidence (0.0 → 1.0).
  - **Aspect-based sentiment**: Phân tích cảm xúc theo từng aspect (dựa trên keywords từ Stage 3).
  - **Probabilities**: Phân phối xác suất cho 3 class.
- Context windowing: Cắt text xung quanh keyword (100 ký tự) để phân tích aspect chính xác hơn.
- Thresholds: positive ≥ 0.25, negative ≤ -0.25, còn lại là neutral.

#### Stage 5: Impact Calculation

- Tính điểm ảnh hưởng (0-100) dựa trên công thức:

```
EngagementScore = views×0.01 + likes×1.0 + comments×2.0 + saves×3.0 + shares×5.0
ReachScore = log2(followers + 1) × (1.2 nếu verified)
RawImpact = (EngagementScore + ReachScore) × PlatformMultiplier × SentimentAmplifier
ImpactScore = (RawImpact / MaxRawScoreCeiling) × 100  (capped 0-100)
```

- **Platform multipliers**: TikTok=1.0, Facebook=1.2, YouTube=1.5, Instagram=1.1.
- **Sentiment amplifiers**: Negative=1.5, Neutral=1.0, Positive=1.1.
- **Risk level**:
  | Điều kiện | Risk Level |
  |-----------|------------|
  | impact ≥ 70 AND sentiment = NEGATIVE | CRITICAL |
  | impact ≥ 70 | HIGH |
  | impact ≥ 40 | MEDIUM |
  | Còn lại | LOW |
- **Flags**:
  - `is_viral`: impact_score ≥ 70.
  - `is_kol`: followers ≥ 50,000.

### 2.3 Error Handling

- Lỗi xử lý được catch và log, record vẫn được lưu với `processing_status="error"`.
- Retry: configurable max 3 lần, exponential backoff (1s → 60s).
- Graceful shutdown: SIGTERM/SIGINT → hoàn thành message đang xử lý trước khi tắt.

### 2.4 Data Enrichment

Pipeline enriches raw post data với metadata từ event envelope:

| Field được inject  | Nguồn                                         |
| ------------------ | --------------------------------------------- |
| `project_id`       | `payload.project_id` hoặc extract từ `job_id` |
| `job_id`           | `payload.job_id`                              |
| `batch_index`      | `payload.batch_index`                         |
| `task_type`        | `payload.task_type`                           |
| `brand_name`       | `payload.brand_name`                          |
| `keyword`          | `payload.keyword`                             |
| `keyword_source`   | `payload.keyword` (alias)                     |
| `crawled_at`       | Parse từ `envelope.timestamp`                 |
| `pipeline_version` | Auto-generated: `crawler_{platform}_v3`       |

---

## 3. Kiến trúc Domain-Driven Design

```
internal/
├── analytics/           # Orchestrator — điều phối pipeline
│   ├── delivery/        # Infrastructure adapter (RabbitMQ handler)
│   ├── usecase/         # Core business logic (AnalyticsPipeline)
│   ├── type.py          # Domain types (PostData, EventMetadata, AnalyticsResult, Input, Output)
│   └── interface.py     # Contracts (IAnalyticsPipeline)
│
├── text_preprocessing/  # Stage 1 — Làm sạch text
├── intent_classification/ # Stage 2 — Phân loại ý định
├── keyword_extraction/  # Stage 3 — Trích xuất từ khóa
├── sentiment_analysis/  # Stage 4 — Phân tích cảm xúc (PhoBERT)
├── impact_calculation/  # Stage 5 — Tính điểm ảnh hưởng
│
├── analyzed_post/       # Repository layer — CRUD cho analyzed_posts
│   ├── repository/      # PostgreSQL repository implementation
│   └── usecase/         # Business logic cho persistence
│
├── consumer/            # Infrastructure — RabbitMQ consumer server
│   ├── registry.py      # DI container — khởi tạo tất cả domain services
│   └── server.py        # Consumer server lifecycle
│
└── model/               # SQLAlchemy ORM models
    └── analyzed_post.py # AnalyzedPost table definition
```

### Dependency Flow

```
ConsumerServer
  → ConsumerRegistry (DI)
    → AnalyticsHandler (delivery adapter)
      → AnalyticsPipeline (orchestrator)
        → TextPreprocessing
        → IntentClassification
        → KeywordExtraction
        → SentimentAnalysis
        → ImpactCalculation
        → AnalyzedPostUseCase
          → AnalyzedPostRepository (PostgreSQL)
```

---

## 4. Tech Stack

| Component          | Technology             | Vai trò                       |
| ------------------ | ---------------------- | ----------------------------- |
| Language           | Python 3.12+           | Backend                       |
| Package Manager    | uv                     | Dependencies                  |
| Message Queue      | RabbitMQ (aio-pika)    | Event-driven consumer         |
| Database           | PostgreSQL 15+         | Lưu kết quả phân tích         |
| ORM                | SQLAlchemy 2.x (async) | Async persistence             |
| Cache              | Redis                  | Cache (optional)              |
| Object Storage     | MinIO                  | Batch raw payloads (optional) |
| Sentiment Model    | PhoBERT (ONNX Runtime) | Phân tích cảm xúc tiếng Việt  |
| Keyword Extraction | YAKE + spaCy           | Statistical keywords + NER    |
| Compression        | Zstandard (zstd)       | Nén dữ liệu batch             |
| Config             | YAML + env vars        | Viper-style config loader     |

---

## 5. Tính năng chính

- **Event-driven**: Consume `data.collected` từ RabbitMQ; hỗ trợ inline payload hoặc `minio_path` batch.
- **Modular pipeline**: Mỗi stage bật/tắt độc lập qua config.
- **Idempotency**: Xử lý dựa trên unique Post ID.
- **Graceful degradation**: Lỗi được log và persist với status="error", không mất dữ liệu.
- **Horizontal scaling**: Stateless design, scale bằng cách tăng consumer instances.
- **Vietnamese NLP**: PhoBERT ONNX cho sentiment, teencode normalization, Vietnamese regex patterns cho intent.
- **Configurable**: YAML config với env var override cho mọi parameter (weights, thresholds, model paths, queue settings).
