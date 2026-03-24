# Phase 5 Facebook UAP Parser Plan

## Mục tiêu

Thêm parser Facebook `full_flow` vào `ingest-srv` theo schema UAP vNext, nhưng chỉ map những field đã được validate là đủ ổn định.

## Phạm vi

- parser Facebook trong `ingest-srv`
- map `POST` + `COMMENT`
- không map transcript/subtitle từ `accessibility_caption`
- không emit `REPLY` nếu raw chưa có reply bodies thật

## File dự kiến tạo

- `internal/uap/usecase/facebook_parser.go`
- `internal/uap/usecase/facebook_parser_test.go`

## File dự kiến sửa

- `internal/uap/types.go`
- `internal/uap/usecase/registry.go`
- `documents/resource/input-output/UAP_SPECIFICATION.md`

## Mapping chính

- `content.text = post.message`
- `content.title = ""`
- `content.subtitle = ""`
- `content.links = extractLinks(post.message)`
- `author.profile_url = post.author.url`
- `comment.author.profile_url = comment.author.profile_url`
- `engagement.likes = reaction_count`
- `engagement.comments_count = comment_count`
- `engagement.shares = share_count`
- `temporal.posted_at = created_time`
- `temporal.updated_at = updated_time` nếu có
- `media` map từ attachments

## Điều cố ý chưa làm

- không map `accessibility_caption` vào `content.subtitle`
- không fabricate `REPLY` từ `reply_count`
- không kéo thêm `post_detail` enrich nếu chưa thật sự cần cho UAP

## Việc cần làm

1. Định nghĩa raw input structs cho Facebook `full_flow`.
2. Viết `parseFacebookFullFlowInput(...)`.
3. Viết `mapFacebookPost(...)`.
4. Viết `mapFacebookComment(...)`.
5. Map attachment thành `media[]`:
   - `Photo -> image`
   - `Video -> video`
6. Đăng ký parser vào registry:
   - `facebook/full_flow`
7. Giữ policy:
   - chỉ emit `REPLY` khi raw có body replies thật

## Kết quả mong muốn

- `ingest-srv` parse được Facebook `full_flow`
- có `author.profile_url`, `updated_at`, attachment media
- không overfit vào field mơ hồ như `accessibility_caption`

## Acceptance Criteria

1. `facebook/full_flow` được `SupportsParse(...) = true`.
2. Parser ra `POST` + `COMMENT`.
3. `author.profile_url` map đúng ở post và comment.
4. `content.subtitle` vẫn rỗng nếu chỉ có `accessibility_caption`.
5. Không emit `REPLY` khi `replies` rỗng hoặc `null`.

## Rủi ro

- Raw Facebook có thể thay đổi shape giữa photo/video post.
- `attachments` không phải lúc nào cũng đầy đủ.
- Nếu cố map quá nhiều field đặc thù sẽ làm core UAP lệch khỏi mục tiêu cross-platform.

## Test tối thiểu

1. Sample có photo attachment.
2. Sample có video attachment.
3. Sample có comment với `reply_count` nhưng không có body replies.
4. Sample có `updated_time`.
