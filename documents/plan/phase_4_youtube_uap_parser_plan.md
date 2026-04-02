# Phase 4 YouTube UAP Parser Plan

## Mục tiêu

Thêm parser YouTube `full_flow` vào `ingest-srv` theo schema UAP vNext.

## Phạm vi

- parser YouTube trong `ingest-srv`
- mapping transcript sang `content.subtitle`
- chưa hỗ trợ `REPLY` vì raw hiện chỉ có `reply_count`

## Phụ thuộc trước khi làm

`scapper-srv` phải đảm bảo `youtube/full_flow` trả thêm `transcript` trong từng entry.

Expected shape:

- `video`
- `detail`
- `comments`
- `transcript`

## File dự kiến tạo

- `internal/uap/usecase/youtube_parser.go`
- `internal/uap/usecase/youtube_parser_test.go`

## File dự kiến sửa

- `internal/uap/types.go`
- `internal/uap/usecase/registry.go`
- `documents/resource/input-output/UAP_SPECIFICATION.md`

## Mapping chính

- `content.text = detail.description`
- `content.title = detail.title || video.title`
- `content.subtitle = transcript.full_text || join(segments[].text)`
- `content.keywords = detail.keywords`
- `content.links = extractLinks(detail.description)`
- `author.profile_url = detail.author_url`
- `media.width = detail.width`
- `media.height = detail.height`
- `temporal.posted_at = detail.date_published || detail.upload_date`

## Việc cần làm

1. Định nghĩa raw input structs cho YouTube.
2. Viết `parseYouTubeFullFlowInput(...)`.
3. Viết `mapYouTubePost(...)`.
4. Viết `mapYouTubeComment(...)`.
5. Đăng ký parser vào registry:
   - `youtube/full_flow`
6. Giữ policy:
   - không emit `REPLY` nếu raw chưa có body replies
   - chỉ giữ `reply_count`

## Kết quả mong muốn

- `ingest-srv` parse được YouTube `full_flow`
- transcript vào `content.subtitle`
- tận dụng được các field cross-platform đã validate

## Acceptance Criteria

1. `youtube/full_flow` được `SupportsParse(...) = true`.
2. Parser ra `POST` + `COMMENT`.
3. `content.subtitle` có text nếu transcript hiện diện.
4. `author.profile_url`, `media.width`, `media.height` map đúng.
5. Không emit `REPLY` từ `reply_count` thuần.

## Rủi ro

- Nếu `scapper-srv` chưa merge `transcript` vào `full_flow`, parser ingest sẽ thiếu input quan trọng.
- Raw YouTube có thể không đều ở `transcript` hoặc `comments`.

## Test tối thiểu

1. Sample có transcript đầy đủ.
2. Sample transcript thiếu `full_text` nhưng có `segments`.
3. Sample chỉ có `reply_count`, không có body replies.
