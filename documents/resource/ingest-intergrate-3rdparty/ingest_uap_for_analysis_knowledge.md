** Tai lieu nay chưa chính thức nên không dùng để lên proposal cho implement **
# UAP de xuat cho Ingest -> Analysis -> Knowledge (giai doan y tuong)

## 1. Muc tieu

Tai lieu nay de xuat 1 UAP day du cho module ingest khi:

- Ingest publish task qua ben thu 3 (crawler) de lay data.
- Dau vao thuc te hien co la 3 file JSON mau TikTok trong `phong_tmp/ingest`.
- Dau ra can phuc vu dong thoi:
  - `documents/analysis` (Analysis consume UAP).
  - `documents/knowledge` (Knowledge consume output sau Analysis, nen UAP can giu du metadata).

Tai lieu nay giu huong **backward-compatible voi UAP v1.0**, nhung bo sung quy uoc mapping ro rang cho post/comment/reply.

---

## 2. Tom tat du lieu dau vao da quet

### 2.1 Root cua 3 file JSON mau

Ca 3 file deu co root shape giong nhau:

- `keyword`
- `limit`
- `threshold`
- `comment_count`
- `total_posts_found`
- `total_posts_processed`
- `created_at`
- `posts[]`

### 2.2 Moi phan tu trong `posts[]`

Moi post co 3 block chinh:

- `search_info`: thong tin tim kiem nhanh.
- `detail`: thong tin day du cua video/post.
- `comments`: danh sach comment, co the co `reply_comments`.

### 2.3 Cac truong nghiep vu quan trong co san

- Dinh danh: `video_id`, `comment_id`, `reply_id`.
- Noi dung: `description`, `comments[].content`, `reply_comments[].content`.
- Tac gia: `author.uid`, `author.username`, `author.nickname`, `author.avatar`.
- Timestamps: `posted_at`, `commented_at`, `replied_at`.
- Tuong tac: `likes_count`, `comments_count`, `shares_count`, `views_count`, `bookmarks_count`.
- Context: `keyword`, `hashtags[]`.

---

## 3. UAP canonical de xuat

## 3.1 Nguyen tac

- 1 message UAP = 1 don vi phan tich text.
- Tach rieng `post`, `comment`, `reply` thanh cac record rieng de Analysis xu ly sentiment/risk theo dung cap.
- Dung `parent` de giu quan he cay thao luan.
- Day du metadata vao `raw.original_fields` va `ingest.trace.raw_ref` de debug/reprocess.

## 3.2 JSON shape chuan

```json
{
  "uap_version": "1.0",
  "event_id": "uuid-v4",
  "event_type": "data.collected",
  "event_created_at": "2026-02-24T14:55:02Z",
  "producer": "ingest-social",
  "ingest": {
    "project_id": "proj_xxx",
    "entity": {
      "entity_type": "topic",
      "entity_name": "vpbank bi phot",
      "brand": "VPBank"
    },
    "source": {
      "source_id": "src_tiktok_01",
      "source_type": "TIKTOK",
      "account_ref": {
        "id": "7580257698044085264",
        "name": "quybadep999"
      }
    },
    "batch": {
      "batch_id": "batch_20260224_145502",
      "batch_index": 0,
      "mode": "SCHEDULED_CRAWL",
      "received_at": "2026-02-24T14:55:02Z",
      "job_id": "job_20260224_tiktok_vpbank",
      "task_type": "crawl_keyword"
    },
    "trace": {
      "raw_ref": "minio://crawler-results/tiktok/2026-02-24/task_xxx/output.json",
      "mapping_id": "map_tiktok_v1"
    }
  },
  "content": {
    "doc_id": "tt_video_7602522309290691858",
    "doc_type": "post",
    "parent": {
      "parent_id": null,
      "parent_type": null
    },
    "url": "https://www.tiktok.com/@quybadep999/video/7602522309290691858",
    "language": "vi",
    "published_at": "2026-02-03T11:30:00Z",
    "author": {
      "author_id": "7580257698044085264",
      "display_name": "Quy Ba Dep",
      "username": "quybadep999",
      "avatar_url": "https://...",
      "followers": null,
      "is_verified": null,
      "author_type": "user"
    },
    "text": "VPBank - Hang tram khach hang ...",
    "description": null,
    "transcription": null,
    "hashtags": ["vpbank", "nganhangvpbank", "phot", "vayno", "luadao"],
    "attachments": [
      { "type": "video", "url": "https://...", "content": null },
      { "type": "image", "url": "https://...", "content": "cover" }
    ]
  },
  "signals": {
    "engagement": {
      "like_count": 88,
      "comment_count": 13,
      "share_count": 25,
      "view_count": 11700,
      "save_count": 11,
      "rating": null
    },
    "geo": {
      "country": null,
      "city": null
    }
  },
  "context": {
    "keywords_matched": ["vpbank bi phot", "vpbank", "phot", "luadao"],
    "campaign_id": null
  },
  "raw": {
    "original_fields": {
      "sample_file": "tiktok_vpbank_bi_phot_20260224_145502.json",
      "search_info": {},
      "detail": {},
      "comments_meta": {
        "total": 9,
        "cursor": 50,
        "has_more": false
      }
    }
  }
}
```

---

## 4. Truong bat buoc de phuc vu ca Analysis va Knowledge

## 4.1 Bat buoc cho Analysis consume on-time

- `uap_version`
- `event_id`
- `ingest.project_id`
- `ingest.source.source_type`
- `ingest.batch.received_at`
- `content.doc_id`
- `content.doc_type`
- `content.published_at`
- `content.text` (hoac `description`/`transcription`, nhung khuyen nghi luon co `text`)

## 4.2 Bat buoc de khong mat thong tin cho Knowledge (qua output Analysis)

- `ingest.source.source_id`
- `content.url`
- `content.author.author_id`
- `content.author.display_name`
- `content.author.username`
- `content.hashtags`
- `signals.engagement.like_count`
- `signals.engagement.comment_count`
- `signals.engagement.share_count`
- `signals.engagement.view_count`
- `context.keywords_matched`
- `ingest.trace.raw_ref`

Neu thieu cac field tren, Analysis van chay duoc nhung Knowledge/RAG se mat nhieu kha nang filter theo metadata.

---

## 5. Mapping chi tiet tu JSON mau TikTok -> UAP

## 5.1 Mapping cap root file

| JSON mau | UAP |
|---|---|
| `keyword` | `context.keywords_matched[0]` + `ingest.entity.entity_name` |
| `created_at` | `event_created_at` + `ingest.batch.received_at` |
| `limit` | `raw.original_fields.limit` |
| `threshold` | `raw.original_fields.threshold` |
| `comment_count` | `raw.original_fields.comment_count_request` |
| `total_posts_found` | `raw.original_fields.total_posts_found` |
| `total_posts_processed` | `raw.original_fields.total_posts_processed` |

## 5.2 Mapping cap post (`posts[i].detail` uu tien, fallback `search_info`)

| TikTok JSON | UAP |
|---|---|
| `detail.video_id` | `content.doc_id = "tt_video_{video_id}"` |
| `detail.url` | `content.url` |
| `detail.description` | `content.text` |
| `detail.posted_at` | `content.published_at` |
| `detail.author.uid` | `content.author.author_id` |
| `detail.author.nickname` | `content.author.display_name` |
| `detail.author.username` | `content.author.username` |
| `detail.author.avatar` | `content.author.avatar_url` |
| `detail.hashtags[]` | `content.hashtags[]` |
| `detail.likes_count` | `signals.engagement.like_count` |
| `detail.comments_count` | `signals.engagement.comment_count` |
| `detail.shares_count` | `signals.engagement.share_count` |
| `detail.views_count` | `signals.engagement.view_count` |
| `detail.bookmarks_count` | `signals.engagement.save_count` |
| `detail.play_url/download_url` | `content.attachments[] (type=video)` |
| `detail.cover_url/origin_cover_url` | `content.attachments[] (type=image)` |
| `detail.subtitle_url` | `content.transcription` (neu da parse text), neu chua parse thi dua vao `raw` |

## 5.3 Mapping cap comment (`posts[i].comments.comments[j]`)

| TikTok JSON | UAP |
|---|---|
| `comment_id` | `content.doc_id = "tt_comment_{comment_id}"` |
| `content` | `content.text` |
| `commented_at` | `content.published_at` |
| `author.*` | `content.author.*` |
| `likes_count` | `signals.engagement.like_count` |
| `reply_count` | `signals.engagement.comment_count` |
| post `video_id` | `content.parent.parent_id = "tt_video_{video_id}"` |
| parent type | `content.parent.parent_type = "post"` |
| record type | `content.doc_type = "comment"` |

## 5.4 Mapping cap reply (`reply_comments[k]`)

| TikTok JSON | UAP |
|---|---|
| `reply_id` | `content.doc_id = "tt_reply_{reply_id}"` |
| `content` | `content.text` |
| `replied_at` | `content.published_at` |
| `author.*` | `content.author.*` |
| `likes_count` | `signals.engagement.like_count` |
| parent comment id | `content.parent.parent_id = "tt_comment_{comment_id}"` |
| parent type | `content.parent.parent_type = "comment"` |
| record type | `content.doc_type = "comment"` (giu compatibility v1.0) |

---

## 6. Quy tac chuan hoa quan trong

## 6.1 ID strategy

- Post: `tt_video_{video_id}`
- Comment: `tt_comment_{comment_id}`
- Reply: `tt_reply_{reply_id}`
- `event_id` luon UUID v4 moi cho moi UAP message.

## 6.2 Timestamp strategy

- `content.published_at`: lay tu nen tang (`posted_at`, `commented_at`, `replied_at`).
- `ingest.batch.received_at`: lay thoi diem ingest nhan du lieu (`created_at` file ket qua hoac worker completed time).
- `event_created_at`: thoi diem publish vao bus noi bo.

## 6.3 Null/default strategy

- Metric khong co -> `null`, khong tu dong gan `0` neu chua chac.
- Chuoi rong -> `null`.
- Arrays khong co du lieu -> `[]`.

## 6.4 Text strategy cho AI

- `content.text` uu tien:
  1. post/comment/reply text goc.
  2. neu rong thi fallback `summary.desc`.
- Khuyen nghi khong nhung URL dai vao `content.text`; de URL trong `attachments` hoac `raw`.

## 6.5 Metadata strategy cho Knowledge

- Dua day du hashtags, author, engagement vao UAP de Analysis co the map sang `uap_metadata`.
- Giu `context.keywords_matched` de Knowledge/RAG filter theo chu de monitor.

---

## 7. Quy uoc publish trong module ingest (de nghi)

## 7.1 Luong giao tiep

1. Ingest -> ben thu 3: RabbitMQ task (`social.direct` / routing key theo platform-task).
2. Ben thu 3 -> Ingest: result message + raw file path (MinIO/S3).
3. Ingest transform raw thanh UAP record.
4. Ingest publish UAP vao topic noi bo cho Analysis (theo docs Analysis hien tai: `smap.collector.output`).

## 7.2 Quy uoc dong goi message

- 1 message = 1 UAP record (de retry/dedup de hon).
- Kafka message key de nghi:
  - `"{project_id}:{content.doc_id}"`.
- Header de nghi:
  - `schema_version=1.0`
  - `source_type=TIKTOK`
  - `doc_type=post|comment`

## 7.3 Thu tu publish

1. Publish post record truoc.
2. Publish comment records sau.
3. Publish reply records cuoi.

Muc dich: downstream co parent context som hon de join/hien thi timeline.

---

## 8. Checklist implementation cho ingest

- [ ] Parse duoc ca `detail` va fallback `search_info`.
- [ ] Tao UAP record rieng cho post/comment/reply.
- [ ] Gan `parent` dung cap.
- [ ] Co `ingest.trace.raw_ref` de truy vet.
- [ ] Validate required fields truoc khi publish.
- [ ] Log `event_id`, `doc_id`, `project_id` cho moi message.
- [ ] Co dead-letter flow cho message map loi schema.

---

## 9. Ket luan de dung ngay o giai doan y tuong

Voi 3 JSON mau hien tai, cach map toi uu la:

- Chuan hoa thanh UAP v1.0 nhung bo sung quy uoc `event_type`, `event_created_at`, `producer`, `batch_index`, `save_count`, `author.username`.
- Tach 3 cap record (post/comment/reply) de Analysis danh gia sentiment/risk chinh xac hon.
- Giu metadata author/hashtags/engagement/keywords/trace ngay tu ingest de output cua Analysis day du cho Knowledge index va RAG filter.

File nay co the duoc dung lam base contract cho sprint implement module ingest.
