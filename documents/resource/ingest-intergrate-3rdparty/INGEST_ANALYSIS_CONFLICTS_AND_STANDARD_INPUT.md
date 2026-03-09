** Tai lieu nay chưa chính thức nên không dùng để lên proposal cho implement **
# Ingest -> Analysis: Conflict Report and Standard Input Proposal

## 1. Scope

Tai lieu nay chi danh gia conflict giua:

- Ingest docs: `documents/input-output/input/UAP_INPUT_SCHEMA.md`, `documents/input-output/input/input_explain.jsonc`, `documents/input-output/input/example_input_crawl.json`, `documents/input-output/input/example_input_csv.json`
- Analysis docs: `documents/analysis/input_payload.md`, `documents/analysis/input_payload.sql`, `documents/analysis/analysis.md`, `documents/analysis/status.md`

Khong bao gom phan conflict voi `knowledge`.

## 2. Conflict List (Ingest -> Analysis)

### 2.1 Transport and event contract conflict

1. Broker khong thong nhat:
- Ingest mo ta day vao Kafka.
- Analysis mo ta consume RabbitMQ (`analytics.data.collected`).
- Impact: khong co hop dong integration end-to-end o muc transport.

2. Event envelope khong thong nhat:
- Ingest UAP root: `uap_version`, `event_id`, `ingest`, `content`, `signals`, `context`, `raw`.
- Analysis root: `event_id`, `event_type`, `timestamp`, `payload`.
- Impact: analysis parser hien tai khong parse duoc UAP record truc tiep.

3. Event type khong thong nhat:
- Analysis ky vong `event_type = "data.collected"`.
- Ingest docs khong dinh nghia ro event_type cho UAP.
- Impact: routing/filtering theo event_type co the fail.

### 2.2 Payload shape conflict

4. UAP block `ingest.*` khong co slot tuong ung trong analysis input:
- UAP co `ingest.source.*`, `ingest.entity.*`, `ingest.trace.*`, `ingest.batch.*`.
- Analysis input chi co `project_id`, `job_id`, `batch_index`, `task_type`, `brand_name`, `keyword`, `minio_path`.
- Impact: mat context source/entity/trace neu khong map rieng.

5. Dinh danh noi dung khong thong nhat:
- UAP dung `content.doc_id`.
- Analysis bat buoc `payload.meta.id`.
- Impact: neu ingest khong map `doc_id -> meta.id` thi analysis reject message.

6. UAP text model khac analysis text model:
- UAP text trong `content.text` + `attachments[].content`.
- Analysis text model gom `content.text`, `content.description`, `content.transcription`, `comments[]`.
- Impact: mot phan text (attachment OCR/caption) khong duoc dua vao NLP neu khong mapping.

7. Comment structure conflict:
- UAP khong co `comments[]` top-level theo dinh dang analysis.
- Analysis preprocessing du kien dung top comments theo likes.
- Impact: mat du lieu context comment khi crawl co comment nhung ingest khong chuan hoa sang `comments[]`.

8. Signals conflict:
- UAP: `signals.engagement.like_count/comment_count/share_count/view_count/rating`.
- Analysis: `interaction.likes/comments_count/shares/views/saves`.
- Impact: can map field-by-field + `rating` khong co dich trong analysis current schema.

9. Geo conflict:
- UAP co `signals.geo.country/city`.
- Analysis input/output khong co field dia ly.
- Impact: mat thong tin dia ly ngay truoc pipeline.

10. Author conflict:
- UAP: `content.author.author_id/display_name/author_type`.
- Analysis: `author.id/name/username/avatar_url/followers/is_verified`.
- Impact: mapping 1-1 khong du, thieu `author_type`; co field analysis can bo sung ma UAP co the khong co.

11. Parent/hierarchy conflict:
- UAP co `content.parent.parent_id/parent_type`.
- Analysis khong co field parent.
- Impact: mat thread structure (post-comment relationship).

12. Language/doc_type conflict:
- UAP co `content.language`, `content.doc_type`.
- Analysis input khong co field tuong ung.
- Impact: mat metadata quan trong cho routing model va phan loai du lieu.

13. Context conflict:
- UAP co `context.keywords_matched`, `context.campaign_id`.
- Analysis input chi co `keyword` (single), khong co campaign_id.
- Impact: khong giu duoc monitoring context day du.

14. Provenance conflict:
- UAP co `raw.original_fields` va `ingest.trace.raw_ref/mapping_id`.
- Analysis input chi co `minio_path`.
- Impact: audit/debug lineage khong day du.

### 2.3 Validation and semantics conflict

15. Rule required field conflict:
- Analysis rule bat buoc duy nhat: `meta.id`.
- UAP docs khong dat rule required tuong duong cho `content.doc_id` theo dang "must".
- Impact: ingest co the hop le theo UAP nhung van bi analysis reject.

16. Timestamp semantics conflict:
- UAP co `ingest.batch.received_at` (thoi diem he thong nhan) va `content.published_at` (business time).
- Analysis dung `timestamp` root de tao `crawled_at`.
- Impact: de nham lan system time vs crawl time vs event time.

17. Project ID semantics conflict:
- UAP project_id la string tu do.
- Analysis mo ta project_id la UUID va co logic extract tu `job_id`.
- Impact: validation/typing co the xung dot khi project_id khong theo UUID.

18. Enum scope conflict:
- UAP `source_type` ho tro `FILE_UPLOAD`, `WEBHOOK`.
- Analysis `platform` enum tap trung mang xa hoi (`TIKTOK/FACEBOOK/YOUTUBE/INSTAGRAM`).
- Impact: file/webhook data khong co gia tri platform tuong thich neu khong co mapping quy uoc.

19. Batch mode conflict:
- Analysis co `minio_path` + zstd decompress trong contract.
- UAP mo ta `trace.raw_ref` (URI), khong chot chung convention voi `minio_path`.
- Impact: consumer khong biet URI nao dung de tai file batch processing.

### 2.4 Internal inconsistency affecting integration

20. Docs analysis khong dong nhat ve muc do implement:
- `analysis.md`/`input_payload.md` mo ta full 5-stage pipeline.
- `status.md` noi AI modules con TODO.
- Impact: ingest team khong biet chinh xac field nao dang duoc su dung that su trong runtime.

21. Docs ingest co cho chua chot:
- `doc_type` trong example crawl dang de placeholder `"post | comment | csv_record | etc."`.
- Impact: enum validation khong the dong bo voi analysis.

22. UAP schema va examples lech nhe:
- `batch.batch_id` xuat hien trong examples, nhung phan schema table khong list ro.
- Impact: khong ro field nao la mandatory trong batch tracking.

## 3. Proposed Standard Input (Canonical Contract)

## 3.1 Ten contract

`IngestAnalysisEnvelope v1` (IAE v1)

Muc tieu:

- Ingest xuat 1 format duy nhat cho ca crawl, file upload, webhook.
- Analysis parse truc tiep, khong can suy doan field.
- Bao toan context UAP + tuong thich voi pipeline analysis hien tai.

## 3.2 Canonical JSON shape

```json
{
  "schema_version": "IAE/1.0",
  "event_id": "b6d6b1e2-9cf3-4e69-8fd0-5b1c8aab9f17",
  "event_type": "data.collected",
  "event_created_at": "2026-02-17T10:00:00Z",
  "producer": "ingest-worker",
  "payload": {
    "record_id": "fb_post_987654321",
    "project_id": "proj_vf8_monitor_01",

    "source": {
      "source_id": "src_fb_01",
      "source_type": "FACEBOOK",
      "platform": "FACEBOOK",
      "account_ref": {
        "id": "1234567890",
        "name": "VinFast Vietnam"
      }
    },

    "entity": {
      "entity_type": "product",
      "entity_name": "VF8",
      "brand_name": "VinFast"
    },

    "collection": {
      "job_id": "job_abc123",
      "batch_id": "batch_2026_02_07_001",
      "batch_index": 0,
      "mode": "SCHEDULED_CRAWL",
      "task_type": "keyword_search",
      "keyword": "vinfast vf8",
      "received_at": "2026-02-07T10:00:00Z",
      "raw_ref": "minio://raw/proj_vf8_monitor_01/facebook/2026-02-07/batch_001.jsonl",
      "mapping_id": "map_fb_default_v3"
    },

    "content": {
      "doc_type": "post",
      "parent": {
        "parent_id": null,
        "parent_type": null
      },
      "url": "https://facebook.com/.../posts/987654321",
      "language": "vi",
      "published_at": "2026-02-07T09:55:00Z",
      "text": "Xe di em nhung pin sut nhanh...",
      "description": null,
      "transcription": null,
      "hashtags": ["vf8", "pin", "gia"],
      "attachments": [
        {
          "type": "image",
          "url": "https://...",
          "content": "hinh anh ve gia o to"
        }
      ]
    },

    "author": {
      "author_id": "fb_user_abc",
      "display_name": "Nguyen A",
      "username": null,
      "avatar_url": null,
      "followers": null,
      "is_verified": null,
      "author_type": "user"
    },

    "interaction": {
      "views": 1111,
      "likes": 120,
      "comments_count": 34,
      "shares": 5,
      "saves": null,
      "rating": null
    },

    "comments": [
      {
        "text": "pin nhanh het",
        "likes": 20
      }
    ],

    "context": {
      "campaign_id": "id_feb_campaign_2026_01",
      "keywords_matched": ["vf8", "pin", "gia"]
    },

    "geo": {
      "country": "VN",
      "city": null
    },

    "raw": {
      "original_fields": {
        "post_id": "987654321",
        "message": "Xe di em..."
      }
    }
  }
}
```

## 3.3 Field explanation (what + why)

### Root fields

| Field | Required | Type | Y nghia |
|---|---|---|---|
| `schema_version` | Yes | string | Version contract de backward-compatible parsing. |
| `event_id` | Yes | UUID string | ID su kien de trace qua ingest -> queue -> analysis. |
| `event_type` | Yes | string | Loai su kien. Chot `data.collected` de tuong thich analysis. |
| `event_created_at` | Yes | ISO8601 | Thoi diem message duoc ingest publish. |
| `producer` | No | string | Ten service publish (vd `ingest-worker`). |
| `payload` | Yes | object | Toan bo du lieu business cua 1 record. |

### Payload identity and source

| Field | Required | Type | Y nghia |
|---|---|---|---|
| `payload.record_id` | Yes | string | Dinh danh noi dung goc. Mapping truc tiep sang `meta.id` cua analysis. |
| `payload.project_id` | Yes | string | Scope project. |
| `payload.source.source_id` | No | string | ID cau hinh source trong ingest. |
| `payload.source.source_type` | Yes | enum | Nguon logic: FACEBOOK/TIKTOK/YOUTUBE/FILE_UPLOAD/WEBHOOK. |
| `payload.source.platform` | Yes | enum | Platform de impact calculator dung truc tiep. Gia tri khuyen nghi: TIKTOK/FACEBOOK/YOUTUBE/INSTAGRAM/OTHER. |
| `payload.source.account_ref` | No | object | Ten/ID page, kenh, file. |

### Payload business context

| Field | Required | Type | Y nghia |
|---|---|---|---|
| `payload.entity.entity_type` | No | enum | product/campaign/service/competitor/topic. |
| `payload.entity.entity_name` | No | string | Ten doi tuong theo doi. |
| `payload.entity.brand_name` | No | string | Thuong hieu. Mapping sang `brand_name` analysis. |
| `payload.collection.job_id` | No | string | ID dot crawl/thu thap. |
| `payload.collection.batch_id` | No | string | ID lo batch de trace. |
| `payload.collection.batch_index` | No | int | Thu tu item trong batch. |
| `payload.collection.mode` | No | enum | SCHEDULED_CRAWL/MANUAL_UPLOAD/WEBHOOK. |
| `payload.collection.task_type` | No | string | keyword_search/profile_crawl/upload... |
| `payload.collection.keyword` | No | string | Tu khoa theo doi/thu thap. |
| `payload.collection.received_at` | No | ISO8601 | Thoi diem ingest nhan du lieu (system time). |
| `payload.collection.raw_ref` | No | URI | Link raw de debug/reprocess. |
| `payload.collection.mapping_id` | No | string | Rule map file upload da dung. |

### Payload content

| Field | Required | Type | Y nghia |
|---|---|---|---|
| `payload.content.doc_type` | Yes | enum | post/comment/video/news/feedback. |
| `payload.content.parent` | No | object | Giu quan he post-comment. |
| `payload.content.url` | No | string | Permalink goc. |
| `payload.content.language` | No | string | Ma ngon ngu (vi/en/...). |
| `payload.content.published_at` | No | ISO8601 | Business time cua content. |
| `payload.content.text` | Cond. | string | Van ban chinh cho NLP. |
| `payload.content.description` | No | string/null | Fallback text phu. |
| `payload.content.transcription` | No | string/null | Transcript tu media. |
| `payload.content.hashtags` | No | string[] | Hashtag da tach. |
| `payload.content.attachments` | No | array | Anh/video/link va content OCR/caption. |

`Cond.`: bat buoc co it nhat 1 trong `text`, `description`, `transcription`, `attachments[].content`, `comments[].text`.

### Payload author and signals

| Field | Required | Type | Y nghia |
|---|---|---|---|
| `payload.author.author_id` | No | string | ID tac gia nguon. |
| `payload.author.display_name` | No | string | Ten hien thi. |
| `payload.author.username` | No | string | Handle/username. |
| `payload.author.avatar_url` | No | string | Anh dai dien. |
| `payload.author.followers` | No | int/null | So follower de tinh reach/is_kol. |
| `payload.author.is_verified` | No | bool/null | Tai khoan xac minh. |
| `payload.author.author_type` | No | enum | user/page/customer/... |
| `payload.interaction.views` | No | int/null | Luot xem. |
| `payload.interaction.likes` | No | int/null | Luot thich. |
| `payload.interaction.comments_count` | No | int/null | So binh luan. |
| `payload.interaction.shares` | No | int/null | Luot chia se. |
| `payload.interaction.saves` | No | int/null | Luot save. |
| `payload.interaction.rating` | No | float/null | Rating cho feedback/file upload. |

### Payload supplementary blocks

| Field | Required | Type | Y nghia |
|---|---|---|---|
| `payload.comments[]` | No | array | Danh sach comment da chuan hoa (text + likes). |
| `payload.context.campaign_id` | No | string | Lien ket campaign marketing. |
| `payload.context.keywords_matched` | No | string[] | Keyword matched trong monitoring. |
| `payload.geo.country` | No | string | Quoc gia. |
| `payload.geo.city` | No | string/null | Thanh pho. |
| `payload.raw.original_fields` | No | object | Ban sao field goc de audit/debug. |

## 3.4 Validation rules (recommended)

1. Must have: `schema_version`, `event_id`, `event_type`, `event_created_at`, `payload`, `payload.record_id`, `payload.project_id`, `payload.source.source_type`, `payload.source.platform`, `payload.content.doc_type`.
2. Must have at least one analyzable text source:
- `payload.content.text`, or
- `payload.content.description`, or
- `payload.content.transcription`, or
- any `payload.content.attachments[].content`, or
- any `payload.comments[].text`.
3. Numeric interaction fields neu null thi analysis map ve `0`.
4. `event_created_at`, `payload.collection.received_at`, `payload.content.published_at` dung ISO8601 UTC.
5. `payload.record_id` phai stable va unique trong `source_type`.

## 3.5 Direct mapping to current Analysis contract

De chay voi analysis parser hien tai, map IAE v1 -> payload cu nhu sau:

- `payload.record_id` -> `payload.meta.id`
- `payload.source.platform` -> `payload.platform`
- `payload.content.url` -> `payload.meta.permalink`
- `payload.content.published_at` -> `payload.meta.published_at`
- `payload.content.text` -> `payload.content.text`
- `payload.content.description` -> `payload.content.description`
- `payload.content.transcription` -> `payload.content.transcription`
- `payload.content.hashtags` -> `payload.content.hashtags`
- `payload.interaction.views` -> `payload.interaction.views`
- `payload.interaction.likes` -> `payload.interaction.likes`
- `payload.interaction.comments_count` -> `payload.interaction.comments_count`
- `payload.interaction.shares` -> `payload.interaction.shares`
- `payload.interaction.saves` -> `payload.interaction.saves`
- `payload.author.author_id` -> `payload.author.id`
- `payload.author.display_name` -> `payload.author.name`
- `payload.author.username` -> `payload.author.username`
- `payload.author.avatar_url` -> `payload.author.avatar_url`
- `payload.author.followers` -> `payload.author.followers`
- `payload.author.is_verified` -> `payload.author.is_verified`
- `payload.comments` -> `payload.comments`
- `payload.collection.job_id` -> `payload.job_id`
- `payload.collection.batch_index` -> `payload.batch_index`
- `payload.collection.task_type` -> `payload.task_type`
- `payload.entity.brand_name` -> `payload.brand_name`
- `payload.collection.keyword` -> `payload.keyword`
- `payload.collection.raw_ref` -> `payload.minio_path` (neu su dung batch file mode)
- `event_created_at` -> `timestamp`

## 4. Recommended decision to unblock team

1. Chot IAE v1 la contract chung giua ingest va analysis.
2. Trong giai doan chuyen doi, ingest publish dong thoi:
- IAE v1 (canonical), va
- adapter payload cu cho analysis runtime neu can.
3. Sau khi analysis parser ho tro IAE v1 native, bo payload cu.

