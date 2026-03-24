# Phase 1 Runtime Docs Alignment Plan

## Mục tiêu

Làm rõ boundary giữa:

- runtime hiện tại của `ingest-srv`
- validation findings đã test
- hướng vNext sẽ làm sau

Sau phase này, người đọc doc không bị hiểu nhầm rằng `ingest-srv` đã parse đủ 3 nền tảng hoặc đã dùng schema vNext.

## Phạm vi

Chỉ sửa tài liệu, không sửa code runtime.

## Tài liệu cần chốt

- `documents/resource/input-output/UAP_SPECIFICATION.md`
- `documents/resource/input-output/UAP_VALIDATION_REPORT_2026-03-24.md`
- `documents/resource/input-output/UAP_VNEXT_PROPOSAL_2026-03-24.md`
- `documents/resource/input-output/UAP_VNEXT_IMPLEMENTATION_PLAN.md`

## Việc cần làm

1. Chốt `UAP_SPECIFICATION.md` là spec của runtime hiện tại.
2. Ghi rõ runtime hiện chỉ parse `tiktok/full_flow`.
3. Ghi rõ `facebook/full_flow` và `youtube/full_flow` mới ở mức proposal.
4. Cập nhật validation report bằng artifact TikTok rerun mới có `COMMENT` và `REPLY`.
5. Sửa wording trong proposal và implementation plan để phản ánh:
   - TikTok đã có evidence live cho `reply_comments`
   - nhưng chưa nên claim `REPLY` cross-platform
6. Đảm bảo không còn mâu thuẫn giữa:
   - spec runtime hiện tại
   - report validation
   - proposal vNext
   - implementation plan vNext

## Kết quả mong muốn

- `UAP_SPECIFICATION.md` không còn bị hiểu như canonical cross-platform spec.
- `UAP_VALIDATION_REPORT` phản ánh đúng kết quả test mới nhất.
- `UAP_VNEXT_PROPOSAL` và `UAP_VNEXT_IMPLEMENTATION_PLAN` vẫn giữ vai trò tài liệu đích.

## Acceptance Criteria

1. `UAP_SPECIFICATION.md` có section nói rõ đây là current runtime.
2. Report có nhắc artifact TikTok mới với `reply_body_count > 0`.
3. Không còn câu nào khẳng định “TikTok hiện không ra reply” như một kết luận tuyệt đối.
4. Có phân biệt rõ:
   - current runtime
   - validated raw capability
   - future vNext design

## Rủi ro

- Nếu phase này làm không kỹ, team sẽ tiếp tục nhầm giữa schema runtime và schema đích.
- Nếu wording quá mơ hồ, developer có thể triển khai nhầm theo doc proposal thay vì runtime.

## Gợi ý review

Review theo 3 câu hỏi:

1. Runtime hiện parse được gì thật?
2. Validation đã chứng minh được gì?
3. vNext sẽ đổi gì nhưng chưa làm?
