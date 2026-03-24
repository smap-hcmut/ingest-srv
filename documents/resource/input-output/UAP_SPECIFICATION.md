# SMAP Universal Analytics Profile (UAP) Specification - Runtime vNext

- Version: `runtime-vnext-v2.0`
- Last updated: `2026-03-24 18:35:00 +07:00`
- Status: `runtime-canonical`

## Mục tiêu

Tài liệu này mô tả schema UAP đang được `ingest-srv` xuất ra ở runtime sau Phase 3.

Nguyên tắc:

- ưu tiên field dùng chung cho TikTok, Facebook, YouTube
- field đặc thù nền tảng được đẩy xuống `platform_meta`
- chỉ emit `REPLY` khi raw có reply body thật
- `subtitle` là text đã normalize, không phải URL

## Runtime support hiện tại

- parser runtime hiện hỗ trợ `tiktok/full_flow` và `youtube/full_flow`
- `facebook/full_flow` vẫn chưa parse ở runtime
- schema runtime hiện tại đã dùng field names vNext

## 1. Enums

- `uap_type`: `POST`, `COMMENT`, `REPLY`
- `platform`: `tiktok`, `facebook`, `youtube`
- `media_type`: `video`, `image`, `carousel`

## 2. Object Shape

### 2.1 POST

```json
{
  "identity": {
    "uap_id": "tt_p_760990...",
    "origin_id": "760990...",
    "uap_type": "POST",
    "platform": "tiktok",
    "url": "https://...",
    "task_id": "f0e87d9e...",
    "project_id": "project-123"
  },

  "hierarchy": {
    "parent_id": null,
    "root_id": "tt_p_760990...",
    "depth": 0
  },

  "content": {
    "text": "Nội dung chính...",
    "title": "Đánh giá VinFast VF8",
    "subtitle": "Xin chào mọi người...",
    "hashtags": ["vf8", "review"],
    "keywords": ["xe điện", "vinfast"],
    "language": "vi",
    "links": ["https://..."]
  },

  "author": {
    "id": "7520108261...",
    "username": "tuyenduongxa",
    "nickname": "Tuyến đường xa",
    "avatar": "https://...",
    "profile_url": "",
    "is_verified": false
  },

  "engagement": {
    "likes": 2578,
    "comments_count": 180,
    "shares": 140,
    "views": 147800,
    "saves": 121,
    "reply_count": 450
  },

  "media": [
    {
      "type": "video",
      "url": "https://...",
      "download_url": "https://...",
      "thumbnail": "https://...",
      "duration": 84,
      "width": null,
      "height": null
    }
  ],

  "temporal": {
    "posted_at": "2026-02-23T03:44:32Z",
    "updated_at": "",
    "ingested_at": "2026-03-24T09:44:00Z"
  },

  "platform_meta": {
    "tiktok": {
      "music_title": "nhạc nền...",
      "music_url": "https://...",
      "is_shop_video": false
    }
  }
}
```

### 2.2 COMMENT / REPLY

```json
{
  "identity": {
    "uap_id": "tt_c_761173...",
    "origin_id": "761173...",
    "uap_type": "COMMENT",
    "platform": "tiktok",
    "url": "",
    "task_id": "f0e87d9e...",
    "project_id": "project-123"
  },

  "hierarchy": {
    "parent_id": "tt_p_760990...",
    "root_id": "tt_p_760990...",
    "depth": 1
  },

  "content": {
    "text": "Con này đẹp mà...",
    "title": "",
    "subtitle": "",
    "hashtags": [],
    "keywords": [],
    "language": "",
    "links": []
  },

  "author": {
    "id": "6830415126...",
    "username": "hanguyenhiep",
    "nickname": "Ha Nguyen Hiep",
    "avatar": "https://...",
    "profile_url": "",
    "is_verified": null
  },

  "engagement": {
    "likes": 47,
    "comments_count": null,
    "shares": null,
    "views": null,
    "saves": null,
    "reply_count": 7
  },

  "media": [],

  "temporal": {
    "posted_at": "2026-02-28T02:38:51Z",
    "updated_at": "",
    "ingested_at": "2026-03-24T09:44:00Z"
  },

  "platform_meta": {
    "tiktok": {
      "sort_score": 0.3481
    }
  }
}
```

## 3. Phân tích có thể làm được

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

## 4. Runtime mapping notes

### TikTok

- `title = detail.summary.title`
- `subtitle = download subtitle từ subtitle_url hoặc downloads.subtitle rồi normalize`
- `keywords = detail.summary.keywords`
- `saves = detail.bookmarks_count`
- `music_*`, `is_shop_video`, `sort_score` đi vào `platform_meta.tiktok`
- chỉ emit `REPLY` khi raw có `reply_comments`

### Facebook / YouTube

### YouTube

- `title = detail.title || video.title`
- `text = detail.description || video.description_snippet`
- `subtitle = transcript.full_text || join(transcript.segments[].text)`
- `keywords = detail.keywords`
- `links = extract từ description, description_snippet, transcript`
- `author.profile_url = detail.author_url || https://www.youtube.com/channel/<channel_id>`
- `author.username = extract handle từ author_url nếu có`
- `media.duration = parse từ duration_text`
- `media.width/height = detail.width/detail.height`
- `comments` map sang `COMMENT`
- `published_time_text` của comment chỉ giữ trong `platform_meta.youtube`, không map vào `temporal.posted_at`
- không emit `REPLY` khi raw chỉ có `reply_count`

### Facebook

- chưa parse ở runtime hiện tại
- sẽ được bổ sung ở phase tiếp theo
