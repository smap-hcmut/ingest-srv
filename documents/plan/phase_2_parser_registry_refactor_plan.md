# Phase 2 Parser Registry Refactor Plan

## Mục tiêu

Refactor `ingest-srv` để mở rộng parser theo `platform + action` mà không phải tiếp tục hard-code TikTok trong flow chung.

Phase này chưa đổi schema UAP. Mục tiêu là đổi structure code, giữ output runtime hiện tại.

## Phạm vi

- refactor trong `internal/uap/usecase`
- refactor cách `execution` quyết định có parse UAP hay không
- không đổi field output JSON
- không thêm Facebook/YouTube parser đầy đủ ở phase này

## File dự kiến tạo

- `internal/uap/usecase/registry.go`
- `internal/uap/usecase/helpers_common.go`
- `internal/uap/usecase/tiktok_parser.go`

## File dự kiến sửa

- `internal/uap/usecase/usecase.go`
- `internal/uap/usecase/new.go`
- `internal/uap/interface.go`
- `internal/uap/usecase/usecase_test.go`
- `internal/execution/usecase/consumer.go`

## Thiết kế mục tiêu

Giữ một entrypoint:

- `ParseAndStoreRawBatch(...)`

Nhưng bên trong dispatch qua registry:

- `platform + action -> parse func`

Ví dụ:

- `tiktok/full_flow -> flattenTikTokFullFlow`

## Việc cần làm

1. Tách validate input chung ra khỏi validate TikTok-specific.
2. Tạo `parseKey`.
3. Tạo `parseFunc`.
4. Tạo `buildParseRegistry()`.
5. Tạo `resolveParser(...)`.
6. Tạo `SupportsParse(platform, action string) bool`.
7. Chuyển phần TikTok parser từ `helpers.go` sang file riêng `tiktok_parser.go`.
8. Để `usecase.go` chỉ còn orchestration:
   - validate common
   - claim raw batch
   - download raw
   - resolve parser
   - parse
   - publish
   - upload chunk
   - mark parsed
9. Sửa `execution/usecase/consumer.go` để không hard-code TikTok nữa.

## Kết quả mong muốn

- flow orchestration vẫn giữ nguyên
- parser logic được tách khỏi orchestration
- thêm platform mới sau này chỉ cần:
  - thêm parser file
  - đăng ký parser vào registry

## Acceptance Criteria

1. Output TikTok UAP không đổi so với trước phase này.
2. `execution` không còn hard-code `tiktok/full_flow`.
3. `usecase.go` không còn gọi trực tiếp `flattenTikTokFullFlow(...)`.
4. Parser TikTok vẫn parse được `POST`, `COMMENT`, `REPLY` như hiện tại.

## Rủi ro

- Refactor sai dễ làm vỡ flow publish/chunk upload dù schema không đổi.
- Nếu trộn helper chung và helper TikTok không rõ ràng, phase sau sẽ lại khó mở rộng.

## Test tối thiểu

1. Registry support:
   - `tiktok/full_flow = true`
   - unsupported target = false
2. Golden test cho TikTok parser trước và sau refactor phải tương đương.
3. `HandleCompletion` vẫn gọi parse path đúng với TikTok.
