# SMAP Universal Analytics Profile (UAP) Specification - vNext

- Version: `vnext-v2.0-draft`
- Last updated: `2026-03-24 16:44:00 +07:00`
- Status: `proposed`

## Mục tiêu

Tài liệu này mô tả schema UAP đích cho các phase tiếp theo.

Nguyên tắc:

- ưu tiên field dùng chung cho TikTok, Facebook, YouTube
- field đặc thù nền tảng được đẩy xuống `platform_meta`
- chỉ emit `REPLY` khi raw có reply body thật
- `subtitle` là text đã normalize, không phải URL

## Runtime Note

- File này là spec đích cho implementation các phase sau.
- Runtime hiện tại vẫn đang dùng schema cũ ở [UAP_SPECIFICATION.md](/Users/phongdang/Documents/GitHub/SMAP/ingest-srv/documents/resource/input-output/UAP_SPECIFICATION.md).

## 1. Enums

- `uap_type`: `POST`, `COMMENT`, `REPLY`
- `platform`: `tiktok`, `facebook`, `youtube`
- `media_type`: `video`, `image`, `carousel`

## 2. Object Shape

### 2.1 POST

```json
{
  "crawl_keyword": "bia heineken",

  "identity": {
    "uap_id": "tt_p_760990...",          // ID duy nhất toàn hệ thống; dùng để join, dedupe, trace lineage
    "origin_id": "760990...",            // ID gốc từ nền tảng; dùng để backtrack raw và debug mapping
    "uap_type": "POST",                  // Loại bản ghi; dùng để tách luồng xử lý post/comment/reply
    "platform": "tiktok",                // Nền tảng nguồn; dùng để phân tích đa nền tảng
    "url": "https://...",                // URL canonical/public; dùng để drill-through từ dashboard
    "task_id": "f0e87d9e...",            // Task crawl sinh ra raw này; dùng để truy vết job
    "project_id": "project-123"          // Project sở hữu dữ liệu; dùng cho phân quyền và grouping
  },

  "hierarchy": {
    "parent_id": null,                   // POST không có cha; dùng để giữ tree conversation nhất quán
    "root_id": "tt_p_760990...",         // Root của toàn thread; dùng để group toàn bộ discussion
    "depth": 0                           // Độ sâu hội thoại; POST luôn là 0
  },

  "content": {
    "text": "Nội dung chính...",         // Nội dung text chính; dùng cho search, sentiment, topic mining
    "title": "Đánh giá VinFast VF8",     // Tiêu đề tốt nhất có thể; dùng cho summarization, headline analysis
    "subtitle": "Xin chào mọi người...", // Transcript/subtitle text đã clean; dùng cho NLP, RAG, ASR-free analysis
    "hashtags": ["vf8", "review"],       // Hashtag nền tảng; dùng cho trend tracking, clustering
    "keywords": ["xe điện", "vinfast"],  // Keyword semantic/AI extract; dùng cho topic detection nhanh
    "language": "vi",                    // Ngôn ngữ nội dung; dùng cho routing model, dashboard filter
    "links": ["https://..."]             // Link bóc ra từ text; dùng cho attribution, commerce, campaign tracing
  },

  "author": {
    "id": "7520108261...",               // ID gốc của author; dùng để join author records xuyên batch
    "username": "tuyenduongxa",          // Username/handle; dùng để hiển thị và QA identity
    "nickname": "Tuyến đường xa",        // Display name; dùng cho dashboard và human review
    "avatar": "https://...",             // Avatar URL; dùng cho UI, moderation review
    "profile_url": "https://...",        // Link profile; dùng để drill-through và identity QA
    "is_verified": false                 // Verified flag nếu có; dùng cho trust weighting
  },

  "engagement": {
    "likes": 2578,                       // Lượng thích/reaction gần nhất; dùng để đo resonance
    "comments_count": 180,               // Tổng comment cấp 1; dùng để đo discussion volume
    "shares": 140,                       // Lượng share; dùng để đo virality
    "views": 147800,                     // Lượt xem nếu có; dùng để tính conversion/engagement rate
    "saves": 121,                        // Save/bookmark trung tính nền tảng; dùng để đo intent sâu
    "reply_count": 450                   // Tổng reply nếu raw hỗ trợ; dùng để đo độ sâu thảo luận
  },

  "media": [
    {
      "type": "video",                   // Kiểu media; dùng để route pipeline ảnh/video
      "url": "https://...",              // URL playable/canonical; dùng để preview hoặc trace source
      "download_url": "https://...",     // URL tải trực tiếp nếu có; dùng cho archival/re-processing
      "thumbnail": "https://...",        // Thumbnail đại diện; dùng cho gallery/dashboard
      "duration": 84,                    // Thời lượng media; dùng cho watch-time segmentation
      "width": 1920,                     // Chiều rộng; dùng cho media QA, format analysis
      "height": 1080                     // Chiều cao; dùng cho aspect-ratio, creative analysis
    }
  ],

  "temporal": {
    "posted_at": "2026-02-23T03:44:32Z", // Thời điểm nền tảng ghi nhận đăng; dùng cho timeline analysis
    "updated_at": "2026-02-23T05:10:00Z",// Thời điểm cập nhật nếu có; dùng để phát hiện edit/update
    "ingested_at": "2026-03-24T09:44:00Z"// Thời điểm ingest; dùng cho freshness, SLA, lag analysis
  },

  "platform_meta": {
    "tiktok": {
      "music_title": "nhạc nền...",      // TikTok-specific; dùng cho trend âm thanh
      "music_url": "https://...",        // TikTok-specific; dùng để audit asset âm thanh
      "is_shop_video": false,            // TikTok-specific; dùng để nhận diện content commerce
      "video_resources": []              // TikTok-specific; dùng cho media QA/debug
    }
  }
}
```

### 2.2 COMMENT / REPLY

```json
{
  "crawl_keyword": "bia heineken",
  
  "identity": {
    "uap_id": "tt_c_761173...",          // ID UAP duy nhất; dùng để join, dedupe, trace
    "origin_id": "761173...",            // ID gốc từ nền tảng; dùng để backtrack raw comment/reply
    "uap_type": "COMMENT",               // COMMENT hoặc REPLY; dùng để route downstream logic
    "platform": "tiktok",                // Nền tảng nguồn; dùng cho phân tích đa nền tảng
    "url": "",                           // Có thể rỗng nếu nền tảng không có permalink comment
    "task_id": "f0e87d9e...",            // Task crawl tương ứng; dùng để audit
    "project_id": "project-123"          // Project sở hữu dữ liệu
  },

  "hierarchy": {
    "parent_id": "tt_p_760990...",       // Cha trực tiếp; COMMENT trỏ POST, REPLY trỏ COMMENT
    "root_id": "tt_p_760990...",         // Root thread; dùng để gom toàn bộ conversation
    "depth": 1                           // COMMENT=1, REPLY>=2; dùng cho tree analytics
  },

  "content": {
    "text": "Con này đẹp mà...",         // Nội dung bình luận; dùng cho sentiment, issue mining
    "title": "",                         // Thường rỗng ở comment/reply; giữ nhất quán schema
    "subtitle": "",                      // Không áp dụng cho comment/reply; giữ nhất quán schema
    "hashtags": [],                      // Có thể rỗng; vẫn cho phép nếu comment chứa hashtag
    "keywords": [],                      // Có thể để rỗng ở phase đầu
    "language": "vi",                    // Ngôn ngữ comment nếu suy ra được
    "links": []                          // Link bóc từ comment/reply; dùng cho spam/commercial detection
  },

  "author": {
    "id": "6830415126...",               // ID author comment/reply; dùng để phân tích actor
    "username": "hanguyenhiep",          // Username/handle
    "nickname": "Ha Nguyen Hiep",        // Display name
    "avatar": "https://...",             // Avatar phục vụ UI/review
    "profile_url": "https://...",        // Nếu raw có; dùng để drill-through
    "is_verified": false                 // Verified nếu raw hỗ trợ
  },

  "engagement": {
    "likes": 47,                         // Like trên comment/reply; dùng để tìm top response
    "comments_count": null,              // Không áp dụng ở comment/reply; để null
    "shares": null,                      // Không áp dụng ở comment/reply; để null
    "views": null,                       // Không áp dụng ở comment/reply; để null
    "saves": null,                       // Không áp dụng ở comment/reply; để null
    "reply_count": 7                     // Chỉ meaningful cho COMMENT; dùng để đo branching
  },

  "media": [],

  "temporal": {
    "posted_at": "2026-02-28T02:38:51Z", // Thời điểm comment/reply được tạo
    "updated_at": "",                    // Để trống nếu raw không có
    "ingested_at": "2026-03-24T09:44:00Z"
  },

  "platform_meta": {
    "tiktok": {
      "sort_score": 0.3481               // Điểm rank comment của TikTok; dùng để tìm comment ảnh hưởng mạnh
    }
  }
}
```

## 3. Field Change Summary So Với Runtime Cũ

| Runtime cũ | vNext | Mục đích đổi |
| :--- | :--- | :--- |
| `content.summary_title` | `content.title` | Trung tính hơn giữa TikTok, YouTube, Facebook |
| `content.subtitle_url` | `content.subtitle` | Dùng text normalized thay vì URL để NLP/RAG trực tiếp |
| `content.tiktok_keywords` | `content.keywords` | Bỏ bias TikTok-only |
| `content.external_links` | `content.links` | Tên ngắn gọn, dùng chung đa nền tảng |
| `engagement.bookmarks` | `engagement.saves` | Trung tính hơn giữa nền tảng |
| `content.music_title` | `platform_meta.tiktok.music_title` | Chuyển field đặc thù xuống meta |
| `content.music_url` | `platform_meta.tiktok.music_url` | Chuyển field đặc thù xuống meta |
| `content.is_shop_video` | `platform_meta.tiktok.is_shop_video` | Chuyển field đặc thù xuống meta |
| `engagement.sort_score` | `platform_meta.tiktok.sort_score` | Chuyển field đặc thù xuống meta |

## 4. Có Thể Analyse Được Gì

| Nhóm phân tích | Field chính | Có thể phân tích gì |
| :--- | :--- | :--- |
| Topic mining | `content.text`, `content.title`, `content.keywords`, `hashtags` | Chủ đề nổi bật, clustering nội dung, keyword heatmap |
| Transcript intelligence | `content.subtitle` | Tóm tắt video, semantic search, quote extraction, RAG không cần ASR |
| Conversation analysis | `uap_type`, `hierarchy.root_id`, `hierarchy.parent_id`, `hierarchy.depth`, `engagement.reply_count` | Cây hội thoại, độ sâu tranh luận, thread reconstruction |
| Sentiment & issue detection | `content.text`, `content.subtitle` | Sentiment, complaint mining, crisis signal detection |
| Creator analysis | `author.id`, `author.username`, `author.profile_url`, `author.is_verified` | Phân nhóm creator, actor mapping, trust weighting |
| Virality analysis | `likes`, `comments_count`, `shares`, `views`, `saves` | Engagement rate, resonance, viral candidate scoring |
| Content lifecycle | `posted_at`, `updated_at`, `ingested_at` | Lag analysis, freshness, edit detection, timeline trend |
| Commerce / attribution | `content.links`, `platform_meta.tiktok.is_shop_video` | Link-out detection, affiliate/commercial content tagging |
| Media analysis | `media.type`, `duration`, `width`, `height`, `thumbnail` | Creative format analysis, asset QA, portrait/landscape segmentation |
| Platform-specific insight | `platform_meta.*` | TikTok trend audio, TikTok comment ranking, raw debug enrichment |

## 5. Mapping Rules By Platform

### TikTok

- `title = detail.summary.title`
- `subtitle = download subtitle từ subtitle_url hoặc downloads.subtitle rồi normalize`
- `keywords = detail.summary.keywords`
- `saves = detail.bookmarks_count`
- `music_*`, `is_shop_video`, `video_resources`, `sort_score` đi vào `platform_meta.tiktok`
- chỉ emit `REPLY` khi raw có `reply_comments`

### YouTube

- `title = detail.title || video.title`
- `subtitle = transcript.full_text || join(segments[].text)`
- `keywords = detail.keywords`
- `profile_url = detail.author_url`
- `width/height = detail.width/detail.height`
- chưa emit `REPLY` nếu raw chỉ có `reply_count`

### Facebook

- `title` thường rỗng
- `subtitle` để rỗng cho tới khi có transcript field đủ tin cậy
- `profile_url = post.author.url` hoặc `comment.author.profile_url`
- `updated_at = updated_time` nếu raw có
- không map `accessibility_caption` vào `subtitle` ở giai đoạn này

## 6. Reply Policy

- Chỉ emit `uap_type=REPLY` khi raw có reply body thật.
- Nếu raw chỉ có `reply_count`, giữ ở `engagement.reply_count`.
- Không fabricate reply records từ count.

## 7. Ghi chú triển khai

1. `platform_meta` là optional object, không bắt buộc lúc nào cũng có.
2. `subtitle` phải là plain text normalized, không giữ timestamp/cue marker.
3. `updated_at` có thể rỗng ở các nền tảng không hỗ trợ.
4. `profile_url` có thể rỗng nếu raw không trả ra.
5. Khi cutover sang vNext, cần cập nhật downstream consumer cùng lúc vì đây là breaking schema change.
