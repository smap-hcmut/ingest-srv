# Phase 3 TikTok vNext Cutover Plan

## Mục tiêu

Nâng TikTok parser từ schema runtime hiện tại sang schema UAP vNext theo hướng trung tính hơn, nhưng vẫn giữ đầy đủ capability của TikTok.

## Phạm vi

- đổi UAP core field names cho TikTok
- thêm subtitle normalization thành text
- chuyển field TikTok-specific sang `platform_meta`
- chưa thêm Facebook/YouTube parser trong phase này

## File dự kiến tạo

- `internal/uap/usecase/subtitle_helper.go`
- `internal/uap/usecase/subtitle_helper_test.go`
- `internal/uap/usecase/tiktok_parser_test.go`

## File dự kiến sửa

- `internal/uap/types.go`
- `internal/uap/errors.go`
- `internal/uap/usecase/tiktok_parser.go`
- `internal/uap/usecase/helpers_common.go`
- `documents/resource/input-output/UAP_SPECIFICATION.md`

## Đổi schema chính

- `content.summary_title -> content.title`
- `content.subtitle_url -> content.subtitle`
- `content.tiktok_keywords -> content.keywords`
- `content.external_links -> content.links`
- `engagement.bookmarks -> engagement.saves`

## Field chuyển xuống `platform_meta`

- `music_title`
- `music_url`
- `is_shop_video`
- `video_resources`
- `sort_score`

## Việc cần làm

1. Đổi type struct trong `internal/uap/types.go`.
2. Sửa `marshalUAPRecord(...)` để chỉ xuất schema mới.
3. Viết subtitle helper cho TikTok:
   - ưu tiên `detail.subtitle_url`
   - fallback `detail.downloads.subtitle`
   - parse `WEBVTT`
   - bỏ timestamp, cue index, dòng trống
   - merge thành plain text
4. Sửa `mapTikTokPost(...)`:
   - `title`
   - `subtitle`
   - `keywords`
   - `links`
   - `saves`
5. Sửa `mapTikTokComment(...)` và `mapTikTokReply(...)` để đưa `sort_score` sang `platform_meta`.
6. Giữ policy:
   - chỉ emit `REPLY` khi raw có `reply_comments`
   - không fabricate từ `reply_count`

## Kết quả mong muốn

- TikTok output đã theo schema vNext
- subtitle không còn là URL trong core UAP
- field đặc thù TikTok không còn làm bẩn core schema

## Acceptance Criteria

1. TikTok parser vẫn ra `POST`, `COMMENT`, `REPLY`.
2. `content.subtitle` là text đã normalize.
3. `content.title`, `content.keywords`, `engagement.saves` hoạt động đúng.
4. `music_*`, `is_shop_video`, `sort_score` không còn nằm trong core object.
5. Artifact TikTok có `reply_comments` vẫn ra `uap_type=REPLY` đúng.

## Rủi ro

- Đây là phase breaking nhất vì đổi schema output.
- Nếu subtitle fetch fail làm fail cả batch thì rất nguy hiểm.
- Nếu cutover quá sớm mà downstream chưa đổi parser thì dễ vỡ contract.

## Test tối thiểu

1. Test parse `WEBVTT`.
2. Test wrong content-type nhưng body là subtitle thật.
3. Test TikTok parser với sample có `reply_comments`.
4. Test legacy keys không còn xuất hiện trong output.
