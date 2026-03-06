# Analytics Schema Explained - Giải thích Chi tiết

**Schema:** `analytics.post_analytics` (tương đương `schema_analyst.analyzed_posts_v2`)

---

## 1. TỔNG QUAN SCHEMA

### 1.1 Mục đích

Schema này lưu trữ KẾT QUẢ phân tích của Analytics Pipeline, bao gồm:

- ✅ Sentiment analysis (cảm xúc tổng thể)
- ✅ Aspect-based sentiment analysis (cảm xúc theo khía cạnh)
- ✅ Keyword extraction (từ khóa)
- ✅ Risk assessment (đánh giá rủi ro)
- ✅ Engagement metrics (chỉ số tương tác)

### 1.2 Vị trí trong Data Flow

```
Raw Data (Excel/TikTok/...)
    ↓
UAP (Unified Analytics Payload)
    ↓
Analytics Pipeline (Sentiment + Aspect + Keyword)
    ↓
analytics.post_analytics ← BẠN Ở ĐÂY
    ↓
Qdrant Vector DB (for RAG)
```

---

## 2. CÁC NHÓM FIELDS

### 2.1 Core Identity Fields

```json
{
  "id": "analytics_550e8400-...", // UUID của record analytics
  "project_id": "proj_vinfast_vf8_monitor", // Thuộc project nào (Tầng 2)
  "source_id": "src_tiktok_crawl_..." // Từ data source nào (Tầng 1)
}
```

**Ý nghĩa:**

- `id`: Định danh duy nhất của bản ghi phân tích
- `project_id`: Dùng để filter trong Campaign (RAG scope)
- `source_id`: Truy vết nguồn gốc data

**Sử dụng trong RAG:**

```go
// Filter by campaign projects
qdrant.Search(ctx, &qdrant.SearchPoints{
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            {
                Field: "project_id",
                Match: &qdrant.Match{
                    Keywords: campaignProjectIDs, // ["proj_vf8", "proj_byd"]
                },
            },
        },
    },
})
```

---

### 2.2 UAP Core Fields (Dữ liệu gốc)

```json
{
  "content": "Mình vừa lái thử VF8...", // [QUAN TRỌNG] Text để phân tích
  "content_created_at": "2026-02-10T14:30:00Z", // Thời gian sự kiện xảy ra
  "ingested_at": "2026-02-15T10:45:23Z", // Thời gian SMAP thu thập
  "platform": "tiktok" // Nguồn: tiktok, youtube, excel...
}
```

**Ý nghĩa:**

- `content`: Text chính để AI phân tích và RAG search
- `content_created_at`: **Business time** - dùng cho trend charts, temporal queries
- `ingested_at`: **System time** - dùng cho debug, latency measurement
- `platform`: Phân biệt nguồn data

**Sử dụng trong RAG:**

```go
// Temporal query: "Xu hướng tuần này"
startOfWeek := time.Now().AddDate(0, 0, -7).Unix()
qdrant.Search(ctx, &qdrant.SearchPoints{
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            {
                Field: "content_created_at",
                Range: &qdrant.Range{
                    Gte: float64(startOfWeek),
                },
            },
        },
    },
})
```

---

### 2.3 Metadata (Thông tin bổ sung)

```json
{
  "uap_metadata": {
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
```

**Ý nghĩa:**

- Schema-less JSON - linh hoạt cho từng platform
- Chứa thông tin context phong phú cho RAG
- Engagement metrics để đánh giá influence

**Sử dụng trong RAG:**

```
User: "Những người có ảnh hưởng nói gì về VF8?"

RAG:
1. Search vectors
2. Filter: author_followers > 10000
3. Generate answer với context từ influencers
```

---

### 2.4 Sentiment Analysis (Cảm xúc tổng thể)

```json
{
  "overall_sentiment": "MIXED", // POSITIVE | NEGATIVE | NEUTRAL | MIXED
  "overall_sentiment_score": 0.15, // -1.0 (rất tiêu cực) → +1.0 (rất tích cực)
  "sentiment_confidence": 0.87, // 0.0 → 1.0 (độ tin cậy)
  "sentiment_explanation": "Người dùng có cảm xúc trái chiều..."
}
```

**Ý nghĩa:**

- `overall_sentiment`: Label phân loại (dễ filter)
- `overall_sentiment_score`: Numeric score (dễ aggregate, sort)
- `sentiment_confidence`: Đánh giá độ tin cậy của model
- `sentiment_explanation`: Human-readable explanation

**Sử dụng trong RAG:**

```go
// Query: "Tìm feedback tiêu cực về VinFast"
qdrant.Search(ctx, &qdrant.SearchPoints{
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            {
                Field: "overall_sentiment",
                Match: &qdrant.Match{
                    Keyword: "NEGATIVE",
                },
            },
        },
    },
})
```

**Dashboard Usage:**

```sql
-- Sentiment distribution
SELECT
    overall_sentiment,
    COUNT(*) as count,
    AVG(overall_sentiment_score) as avg_score
FROM analytics.post_analytics
WHERE project_id = 'proj_vf8'
GROUP BY overall_sentiment;

-- Result:
-- POSITIVE: 450 posts, avg_score: 0.72
-- NEGATIVE: 320 posts, avg_score: -0.65
-- NEUTRAL:  180 posts, avg_score: 0.05
-- MIXED:    150 posts, avg_score: 0.12
```

---

### 2.5 Aspect-Based Sentiment Analysis (ABSA)

```json
{
  "aspects": [
    {
      "aspect": "DESIGN", // Aspect ID (enum)
      "aspect_display_name": "Thiết kế", // Tên hiển thị
      "sentiment": "POSITIVE", // Cảm xúc về aspect này
      "sentiment_score": 0.85, // Score cụ thể
      "confidence": 0.92, // Độ tin cậy
      "keywords": ["đẹp", "sang trọng", "nội thất"],
      "mentions": [
        // Vị trí xuất hiện trong text
        {
          "text": "Thiết kế xe rất đẹp và sang trọng",
          "start_pos": 35,
          "end_pos": 68
        }
      ],
      "impact_score": 0.78, // Mức độ ảnh hưởng
      "explanation": "Người dùng rất hài lòng với thiết kế..."
    },
    {
      "aspect": "BATTERY",
      "aspect_display_name": "Pin/Năng lượng",
      "sentiment": "NEGATIVE",
      "sentiment_score": -0.72,
      "keywords": ["pin", "sụt nhanh", "100km", "25% pin"],
      "impact_score": 0.85
    }
  ]
}
```

**Ý nghĩa:**

- Phân tích chi tiết cảm xúc theo TỪNG khía cạnh
- Mỗi aspect có sentiment riêng (có thể khác overall_sentiment)
- `mentions`: Trích xuất chính xác đoạn text liên quan
- `impact_score`: Đánh giá mức độ quan trọng của aspect này

**Aspect Categories (Ví dụ cho Automotive):**

```
DESIGN       - Thiết kế
BATTERY      - Pin/Năng lượng
PRICE        - Giá cả
SERVICE      - Dịch vụ
PERFORMANCE  - Hiệu suất
COMFORT      - Tiện nghi
SAFETY       - An toàn
TECHNOLOGY   - Công nghệ
QUALITY      - Chất lượng
```

**Sử dụng trong RAG:**

```go
// Query: "VinFast bị phàn nàn về gì?"
qdrant.Search(ctx, &qdrant.SearchPoints{
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            {
                Field: "project_id",
                Match: "proj_vf8",
            },
            {
                // Filter aspects array: tìm aspect có sentiment NEGATIVE
                Field: "aspects[].sentiment",
                Match: "NEGATIVE",
            },
        },
    },
})

// RAG sẽ trả về:
// "VinFast nhận nhiều phàn nàn về:
//  1. PIN (45% negative) - Sụt nhanh, dung lượng thấp
//  2. GIÁ (30% negative) - Đắt so với đối thủ
//  3. DỊCH VỤ (15% negative) - Chậm trễ"
```

**Dashboard - Aspect Heatmap:**

```sql
-- Aspect sentiment breakdown
SELECT
    aspect,
    aspect_display_name,
    sentiment,
    COUNT(*) as mention_count,
    AVG(sentiment_score) as avg_score,
    AVG(impact_score) as avg_impact
FROM analytics.post_analytics,
     jsonb_to_recordset(aspects) AS a(
         aspect text,
         aspect_display_name text,
         sentiment text,
         sentiment_score float,
         impact_score float
     )
WHERE project_id = 'proj_vf8'
GROUP BY aspect, aspect_display_name, sentiment
ORDER BY mention_count DESC;

-- Result:
-- DESIGN    | Thiết kế      | POSITIVE | 450 | 0.78 | 0.75
-- BATTERY   | Pin           | NEGATIVE | 320 | -0.72| 0.85
-- PRICE     | Giá cả        | NEGATIVE | 280 | -0.65| 0.72
-- SERVICE   | Dịch vụ       | POSITIVE | 250 | 0.68 | 0.65
```

---

### 2.6 Keywords (Từ khóa)

```json
{
  "keywords": [
    "VF8",
    "thiết kế",
    "đẹp",
    "sang trọng",
    "pin",
    "sụt nhanh",
    "giá",
    "cao"
  ]
}
```

**Ý nghĩa:**

- Trích xuất từ khóa quan trọng từ content
- Dùng cho word cloud, trending topics
- Hỗ trợ RAG tìm kiếm chính xác hơn

**Sử dụng trong RAG:**

```sql
-- Top keywords trong campaign
SELECT
    keyword,
    COUNT(*) as frequency
FROM analytics.post_analytics,
     unnest(keywords) AS keyword
WHERE project_id IN ('proj_vf8', 'proj_byd')
GROUP BY keyword
ORDER BY frequency DESC
LIMIT 20;
```

---

### 2.7 Risk Assessment (Đánh giá rủi ro)

```json
{
  "risk_level": "MEDIUM", // LOW | MEDIUM | HIGH | CRITICAL
  "risk_score": 0.58, // 0.0 → 1.0
  "risk_factors": [
    {
      "factor": "NEGATIVE_BATTERY_FEEDBACK",
      "severity": "HIGH",
      "description": "Phản hồi tiêu cực về pin có thể ảnh hưởng..."
    },
    {
      "factor": "PRICE_CONCERN",
      "severity": "MEDIUM",
      "description": "Lo ngại về giá so với đối thủ"
    }
  ],
  "requires_attention": true, // Cần chú ý?
  "alert_triggered": false // Đã trigger alert chưa?
}
```

**Ý nghĩa:**

- Tự động đánh giá mức độ rủi ro của post
- `risk_factors`: Liệt kê các yếu tố gây rủi ro
- `requires_attention`: Flag để filter posts cần xem xét
- `alert_triggered`: Tránh spam alerts

**Crisis Detection Logic:**

```
Risk Score = weighted_sum(
    negative_sentiment_score * 0.4,
    high_engagement * 0.3,
    influencer_author * 0.2,
    trending_keywords * 0.1
)

if risk_score > 0.7:
    risk_level = CRITICAL
    trigger_alert()
```

**Sử dụng trong Dashboard:**

```sql
-- Crisis monitor
SELECT
    content,
    overall_sentiment_score,
    risk_score,
    uap_metadata->>'engagement'->>'views' as views
FROM analytics.post_analytics
WHERE project_id = 'proj_vf8'
  AND risk_level IN ('HIGH', 'CRITICAL')
  AND content_created_at > NOW() - INTERVAL '24 hours'
ORDER BY risk_score DESC
LIMIT 10;
```

---

### 2.8 Engagement Metrics (Chỉ số tương tác)

```json
{
  "engagement_score": 0.73, // Tổng hợp engagement
  "virality_score": 0.65, // Khả năng viral
  "influence_score": 0.58, // Mức độ ảnh hưởng
  "reach_estimate": 45000 // Ước tính reach
}
```

**Ý nghĩa:**

- Đánh giá mức độ lan truyền của post
- Dùng để prioritize posts quan trọng
- Kết hợp với sentiment để đánh giá impact

**Calculation:**

```
engagement_score = (likes + comments*2 + shares*3) / views

virality_score = shares / views

influence_score = author_followers / 1000000 * engagement_score
```

---

### 2.9 Quality Metrics (Chỉ số chất lượng)

```json
{
  "content_quality_score": 0.82, // Chất lượng content
  "is_spam": false, // Spam detection
  "is_bot": false, // Bot detection
  "language": "vi", // Ngôn ngữ
  "language_confidence": 0.98, // Độ tin cậy
  "toxicity_score": 0.05, // Độ toxic
  "is_toxic": false // Có toxic không?
}
```

**Ý nghĩa:**

- Filter spam/bot để có data sạch
- Toxicity detection để moderate content
- Language detection để route đúng model

**Sử dụng:**

```sql
-- Clean data for analysis
SELECT *
FROM analytics.post_analytics
WHERE is_spam = false
  AND is_bot = false
  AND content_quality_score > 0.5
  AND toxicity_score < 0.3;
```

---

## 3. SỬ DỤNG TRONG RAG

### 3.1 Vector Payload Structure

Khi index vào Qdrant, payload sẽ là:

```json
{
  "id": "analytics_550e8400-...",
  "vector": [0.123, -0.456, ...],  // 1536 dimensions
  "payload": {
    "project_id": "proj_vf8",
    "source_id": "src_tiktok_...",
    "content": "Mình vừa lái thử VF8...",
    "content_created_at": 1707577800,  // Unix timestamp
    "platform": "tiktok",

    "overall_sentiment": "MIXED",
    "overall_sentiment_score": 0.15,

    "aspects": [
      {"aspect": "DESIGN", "sentiment": "POSITIVE", "score": 0.85},
      {"aspect": "BATTERY", "sentiment": "NEGATIVE", "score": -0.72}
    ],

    "keywords": ["VF8", "thiết kế", "pin", "giá"],

    "risk_level": "MEDIUM",
    "engagement_score": 0.73,

    "metadata": {
      "author": "nguyen_van_a_2024",
      "views": 45000,
      "likes": 3200
    }
  }
}
```

### 3.2 RAG Query Examples

#### Example 1: Simple sentiment query

```
User: "VinFast được đánh giá tích cực về gì?"

RAG Process:
1. Embed query → vector
2. Search Qdrant:
   - Vector similarity
   - Filter: project_id = "proj_vf8"
   - Filter: overall_sentiment = "POSITIVE"
3. Get top 10 results
4. Generate answer from results

Answer: "VinFast được đánh giá tích cực về:
1. THIẾT KẾ (85% positive) - Đẹp, sang trọng, nội thất cao cấp
2. DỊCH VỤ (68% positive) - Nhân viên nhiệt tình, bảo hành tốt
3. CÔNG NGHỆ (72% positive) - Tính năng hiện đại, màn hình lớn"
```

#### Example 2: Aspect-specific query

```
User: "Tại sao VinFast bị chê về pin?"

RAG Process:
1. Embed query → vector
2. Search Qdrant:
   - Vector similarity
   - Filter: project_id = "proj_vf8"
   - Filter: aspects[].aspect = "BATTERY"
   - Filter: aspects[].sentiment = "NEGATIVE"
3. Get top 10 results
4. Extract mentions from aspects
5. Generate answer

Answer: "VinFast bị chê về pin vì:
1. Sụt pin nhanh - 'đi 100km hết 25% pin' [1]
2. Dung lượng thấp - 'pin yếu hơn kỳ vọng' [2]
3. Sạc chậm - 'sạc đầy mất 8 tiếng' [3]

Có 320 mentions tiêu cực về pin với avg score -0.72"
```

#### Example 3: Comparison query (Campaign-scoped)

```
User: "So sánh VinFast và BYD về giá"

Campaign: "So sánh Xe điện Q1/2026"
Projects: ["proj_vf8", "proj_byd"]

RAG Process:
1. Embed query → vector
2. Search Qdrant:
   - Vector similarity
   - Filter: project_id IN ["proj_vf8", "proj_byd"]
   - Filter: aspects[].aspect = "PRICE"
3. Group results by project_id
4. Aggregate sentiment scores
5. Generate comparison

Answer: "So sánh giá:

VinFast VF8:
- Giá: 1.2 tỷ
- Sentiment: 65% negative
- Phản hồi: 'Đắt so với đối thủ', 'Giá cao'

BYD Seal:
- Giá: 900 triệu
- Sentiment: 45% negative
- Phản hồi: 'Giá hợp lý', 'Rẻ hơn VinFast'

Kết luận: BYD được đánh giá tốt hơn về giá"
```

#### Example 4: Temporal query

```
User: "Xu hướng đánh giá về VinFast tuần này"

RAG Process:
1. Calculate time range: last 7 days
2. Search Qdrant:
   - Vector similarity
   - Filter: project_id = "proj_vf8"
   - Filter: content_created_at >= start_of_week
3. Aggregate by day
4. Detect trend

Answer: "Xu hướng tuần này (09/02 - 15/02):
- Sentiment trung bình: 0.25 (tăng từ 0.15 tuần trước)
- Positive mentions: tăng 15%
- Trending topics: 'bảo hành tốt', 'dịch vụ cải thiện'
- Negative mentions: giảm 10%
- Cảnh báo: Vẫn còn phàn nàn về pin"
```

---

## 4. DATABASE SCHEMA (PostgreSQL)

```sql
CREATE TABLE analytics.post_analytics (
    -- Identity
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL,
    source_id UUID NOT NULL,

    -- UAP Core
    content TEXT NOT NULL,
    content_created_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL,
    platform VARCHAR(50),
    uap_metadata JSONB,

    -- Sentiment
    overall_sentiment VARCHAR(20),
    overall_sentiment_score FLOAT,
    sentiment_confidence FLOAT,
    sentiment_explanation TEXT,

    -- ABSA
    aspects JSONB,  -- Array of aspect objects

    -- Keywords
    keywords TEXT[],

    -- Entities
    -- Entities
    -- (Removed: Not currently in schema)

    -- Risk
    risk_level VARCHAR(20),
    risk_score FLOAT,
    risk_factors JSONB,
    requires_attention BOOLEAN DEFAULT false,
    alert_triggered BOOLEAN DEFAULT false,

    -- Engagement
    engagement_score FLOAT,
    virality_score FLOAT,
    influence_score FLOAT,
    reach_estimate INTEGER,

    -- Quality
    content_quality_score FLOAT,
    is_spam BOOLEAN DEFAULT false,
    is_bot BOOLEAN DEFAULT false,
    language VARCHAR(10),
    language_confidence FLOAT,
    toxicity_score FLOAT,
    is_toxic BOOLEAN DEFAULT false,

    -- Timestamps
    analyzed_at TIMESTAMPTZ DEFAULT NOW(),
    indexed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Processing
    -- (Removed: processing_metadata not currently in schema)
);

-- Indexes
CREATE INDEX idx_post_analytics_project ON analytics.post_analytics(project_id);
CREATE INDEX idx_post_analytics_source ON analytics.post_analytics(source_id);
CREATE INDEX idx_post_analytics_created ON analytics.post_analytics(content_created_at);
CREATE INDEX idx_post_analytics_sentiment ON analytics.post_analytics(overall_sentiment);
CREATE INDEX idx_post_analytics_risk ON analytics.post_analytics(risk_level);
CREATE INDEX idx_post_analytics_attention ON analytics.post_analytics(requires_attention) WHERE requires_attention = true;

-- GIN index for JSONB
CREATE INDEX idx_post_analytics_aspects ON analytics.post_analytics USING GIN (aspects);
CREATE INDEX idx_post_analytics_metadata ON analytics.post_analytics USING GIN (uap_metadata);
```

---

## 5. TÓM TẮT

### Key Takeaways

1. **Schema này là OUTPUT của Analytics Pipeline** - chứa KẾT QUẢ phân tích, không phải raw data

2. **Có đủ metadata cho RAG** - project_id, sentiment, aspects, keywords để filter chính xác

3. **Aspect-based sentiment là CORE** - cho phép phân tích chi tiết theo từng khía cạnh

4. **Risk assessment tự động** - phát hiện crisis sớm

5. **Ready for Qdrant** - structure phù hợp để index vào vector DB

### Data Flow Summary

```
Raw Data → UAP → Analytics Pipeline → analytics.post_analytics → Qdrant → RAG
```

### Next Steps

1. Implement Analytics Pipeline để tạo ra data theo schema này
2. Setup Qdrant indexing từ analytics.post_analytics
3. Implement RAG query với filters phong phú
4. Build Dashboard từ aggregated data
