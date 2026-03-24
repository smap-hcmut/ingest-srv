# UAP vNext Implementation Plan

## Summary

Kế hoạch này chỉ mô tả implementation, chưa hiện thực code.

Mục tiêu:

- giữ `ParseAndStoreRawBatch(...)` là entrypoint chung
- refactor nội bộ theo registry `platform + action -> parse func`
- tách logic parse theo file trong cùng package `usecase`
- đổi schema UAP sang vNext theo hướng cross-platform
- `ingest-srv` tự fetch, parse, clean subtitle TikTok
- `scapper-srv` YouTube `full_flow` phải có `transcript`

## Files To Create

### `ingest-srv/internal/uap/usecase/registry.go`

Chứa registry parser:

- `type parseKey struct { platform string; action string }`
- `type parseFunc ...`
- `buildParseRegistry()`
- `resolveParser(...)`
- `SupportsParse(...)`

Vì sao:

- thêm platform mới chỉ cần add parser mới vào registry
- không phải sửa orchestration chung

### `ingest-srv/internal/uap/usecase/helpers_common.go`

Chứa helper chung:

- validate input chung
- `toMap`, `toSlice`, `stringAt`, `intAt`, `floatAt`, `boolAt`, `stringSliceAt`
- `normalizeStringSlice`, `firstNonEmpty`, `firstNonZero`
- `buildUAPID`
- `extractLinks`
- `normalizePostedAt`
- `chunkRecords`, `marshalChunkJSONL`, `uploadChunk`
- `mergeRawMetadata`, `failRawBatch`
- pointer helpers, `readAllAndClose`

Vì sao:

- giữ `usecase.go` và parser files gọn
- tránh một file `helpers.go` phình to khi thêm platform

### `ingest-srv/internal/uap/usecase/subtitle_helper.go`

Chứa logic subtitle:

- `resolveTikTokSubtitleText(...)`
- `downloadSubtitleText(...)`
- `normalizeWEBVTT(...)`
- `normalizeTranscriptText(...)`

Vì sao:

- subtitle enrichment là concern riêng
- dễ test độc lập

### `ingest-srv/internal/uap/usecase/tiktok_parser.go`

Chứa:

- `flattenTikTokFullFlow(...)`
- `parseTikTokFullFlowInput(...)`
- `mapTikTokPost(...)`
- `mapTikTokComment(...)`
- `mapTikTokReply(...)`

### `ingest-srv/internal/uap/usecase/youtube_parser.go`

Chứa:

- `flattenYouTubeFullFlow(...)`
- `parseYouTubeFullFlowInput(...)`
- `mapYouTubePost(...)`
- `mapYouTubeComment(...)`

### `ingest-srv/internal/uap/usecase/facebook_parser.go`

Chứa:

- `flattenFacebookFullFlow(...)`
- `parseFacebookFullFlowInput(...)`
- `mapFacebookPost(...)`
- `mapFacebookComment(...)`

### Test files

- `ingest-srv/internal/uap/usecase/registry_test.go`
- `ingest-srv/internal/uap/usecase/tiktok_parser_test.go`
- `ingest-srv/internal/uap/usecase/youtube_parser_test.go`
- `ingest-srv/internal/uap/usecase/facebook_parser_test.go`
- `ingest-srv/internal/uap/usecase/subtitle_helper_test.go`

## Files To Modify

### `ingest-srv/internal/uap/types.go`

Sửa UAP schema sang vNext:

- `SummaryTitle -> Title`
- `SubtitleURL -> Subtitle`
- `TikTokKeywords -> Keywords`
- `ExternalLinks -> Links`
- `Bookmarks -> Saves`
- thêm `Author.ProfileURL`
- thêm `Media.Width`, `Media.Height`
- thêm `Temporal.UpdatedAt`
- thêm `PlatformMeta map[string]interface{}`
- bỏ khỏi core:
  - `MusicTitle`
  - `MusicURL`
  - `IsShopVideo`
  - `SortScore`

Vì sao:

- khớp spec vNext
- giảm field đặc thù TikTok trong core

### `ingest-srv/internal/uap/errors.go`

Thêm:

- `ErrUnsupportedParseTarget`

Vì sao:

- phân biệt invalid input với unsupported parser target

### `ingest-srv/internal/uap/interface.go`

Thêm:

- `SupportsParse(platform, action string) bool`

Vì sao:

- execution layer không nên hard-code TikTok

### `ingest-srv/internal/uap/usecase/new.go`

Sửa `implUseCase`:

- thêm `parsers map[parseKey]parseFunc`
- thêm `subtitleHTTPClient *http.Client`
- khởi tạo registry trong `New(...)`

Vì sao:

- parser dispatch phải được wiring tại constructor

### `ingest-srv/internal/uap/usecase/usecase.go`

Refactor `ParseAndStoreRawBatch(...)`:

- chỉ validate common input
- resolve parser từ registry
- gọi parser tương ứng thay vì hard-code TikTok
- giữ nguyên flow claim, download, publish, upload, mark parsed

Vì sao:

- orchestration chung vẫn reuse được cho mọi platform

### `ingest-srv/internal/execution/usecase/consumer.go`

Sửa `shouldParseUAP(...)` thành:

- `return uc.parser != nil && uc.parser.SupportsParse(task.Platform, task.TaskType)`

Vì sao:

- bỏ hard-code TikTok ở execution layer

### `ingest-srv/internal/uap/usecase/usecase_test.go`

Giữ lại các test chung:

- chunking
- metadata merge
- publisher payload

Điều chỉnh:

- bỏ dependency vào `flattenTikTokFullFlow` trong file này
- chuyển test parser TikTok sang file riêng

### `ingest-srv/documents/resource/input-output/UAP_SPECIFICATION.md`

Cập nhật thành canonical spec theo vNext.

### `scapper-srv/app/handlers/youtube.py`

Đảm bảo `handle_full_flow(...)` trả thêm `transcript`.

Vì sao:

- parser YouTube ở ingest sẽ dựa vào raw này

## Mapping Rules

### TikTok

- `content.text = detail.description || post.description`
- `content.title = detail.summary.title`
- `content.subtitle = resolveTikTokSubtitleText(detail)`
- `content.keywords = detail.summary.keywords`
- `content.links = extractLinks(detail.description, post.description)`
- `engagement.saves = detail.bookmarks_count`
- `platform_meta.tiktok`:
  - `music_title`
  - `music_url`
  - `is_shop_video`
  - `video_resources`
- `platform_meta.tiktok.sort_score` cho COMMENT
- chỉ tạo `REPLY` khi raw có `reply_comments`

### YouTube

- `content.text = detail.description`
- `content.title = detail.title || video.title`
- `content.subtitle = transcript.full_text || join(segments[].text)`
- `content.keywords = detail.keywords`
- `content.links = extractLinks(detail.description)`
- `author.profile_url = detail.author_url`
- `media.width = detail.width`
- `media.height = detail.height`
- `temporal.posted_at = detail.date_published || detail.upload_date`
- không tạo `REPLY` nếu chỉ có `reply_count`

### Facebook

- `content.text = post.message`
- `content.title = ""`
- `content.subtitle = ""`
- `content.links = extractLinks(post.message)`
- `author.profile_url = post.author.url`
- `comment.author.profile_url = comment.author.profile_url`
- `engagement.likes = reaction_count`
- `media` map từ attachments
- không map `accessibility_caption` vào `content.subtitle`
- không tạo `REPLY` khi `replies` rỗng hoặc `null`

## Subtitle Fetch Policy

TikTok subtitle fetch làm ở `ingest-srv`:

- ưu tiên `detail.subtitle_url`
- fallback `detail.downloads.subtitle`
- timeout `10s`
- max body `1 MiB`
- nếu lỗi download hoặc parse:
  - log warning
  - để `content.subtitle = ""`
  - không fail cả batch

Normalize:

- bỏ `WEBVTT`
- bỏ cue index
- bỏ timestamp
- bỏ dòng trống
- join text còn lại bằng space

Vì sao:

- giảm thay đổi phía `scapper-srv`
- vẫn đạt `content.subtitle` text theo vNext

## Test Plan

### Registry

- support:
  - `tiktok/full_flow`
  - `youtube/full_flow`
  - `facebook/full_flow`
- unsupported target trả `false`

### TikTok parser

- parse được POST, COMMENT, REPLY khi raw có `reply_comments`
- map `title`, `subtitle`, `keywords`, `saves`
- field TikTok-specific đi vào `platform_meta`
- fetch subtitle lỗi không fail batch

### Subtitle helper

- parse `WEBVTT` chuẩn
- parse body text dù header content-type sai
- timeout, `404`, body quá lớn trả empty string

### YouTube parser

- parse POST + COMMENT
- map transcript sang `content.subtitle`
- không emit `REPLY` nếu không có reply bodies

### Facebook parser

- parse POST + COMMENT
- map `author.profile_url`
- không map `accessibility_caption` sang subtitle
- không emit `REPLY` nếu `replies` trống

### Marshal output

- chỉ có field vNext
- không còn legacy keys:
  - `subtitle_url`
  - `summary_title`
  - `tiktok_keywords`
  - `external_links`
  - `bookmarks`
  - `music_title`
  - `music_url`
  - `is_shop_video`
  - `sort_score`

## Assumptions

- Đây là breaking cutover, không dual-write field cũ và mới.
- `scapper-srv` YouTube `full_flow` đã hoặc sẽ được merge thêm `transcript`.
- `ingest-srv` được phép outbound fetch subtitle URL của TikTok.
- Reply policy là strict:
  - có body thật mới emit `REPLY`
  - nếu chỉ có `reply_count` thì giữ ở engagement.
