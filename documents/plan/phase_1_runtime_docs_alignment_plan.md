# Phase 1 Runtime Docs Alignment Plan

## Trạng thái

- Status: implemented
- Implemented at: 2026-03-24
- Scope delivered: docs only, no runtime code change

## Artifact tham chiếu

- `documents/resource/input-output/UAP_SPECIFICATION.md`
- `documents/resource/input-output/UAP_VALIDATION_REPORT_2026-03-24.md`
- `documents/resource/input-output/UAP_VNEXT_PROPOSAL_2026-03-24.md`
- `documents/resource/input-output/UAP_VNEXT_IMPLEMENTATION_PLAN.md`
- `scapper-srv/output/validation_uap/tiktok_full_flow_threshold03_validation_20260324_072909.json`

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

## Kết quả đã hiện thực

1. `UAP_SPECIFICATION.md` đã được đổi sang framing `Current Runtime`.
2. Spec runtime đã ghi rõ:
   - parser hiện chỉ hỗ trợ `tiktok/full_flow`
   - `facebook/full_flow` và `youtube/full_flow` vẫn ở mức proposal
   - schema hiện tại vẫn là schema cũ, chưa phải vNext
3. `UAP_VALIDATION_REPORT` đã được cập nhật bằng rerun TikTok mới:
   - `keyword=vinfast vf8`
   - `total_comments=157`
   - `reply_body_count=85`
   - `reply_support=supported now`
4. `UAP_VNEXT_PROPOSAL` và `UAP_VNEXT_IMPLEMENTATION_PLAN` đã được sửa wording để phản ánh:
   - TikTok đã có evidence live cho `reply_comments`
   - nhưng chưa thể claim `REPLY` cross-platform

## Những gì phase này cố ý không làm

- không sửa runtime code
- không đổi schema output hiện tại của `ingest-srv`
- không thêm parser Facebook/YouTube
- không biến `UAP_SPECIFICATION.md` thành canonical vNext spec

## Acceptance Criteria

1. `UAP_SPECIFICATION.md` có section nói rõ đây là current runtime.
2. Report có nhắc artifact TikTok mới với `reply_body_count > 0`.
3. Không còn câu nào khẳng định “TikTok hiện không ra reply” như một kết luận tuyệt đối.
4. Có phân biệt rõ:
   - current runtime
   - validated raw capability
   - future vNext design

## Acceptance review

1. Done
   - `UAP_SPECIFICATION.md` đã có section `Current Runtime`
2. Done
   - report đã nhắc artifact TikTok rerun mới với `reply_body_count > 0`
3. Done
   - report không còn chốt tuyệt đối rằng TikTok hiện không ra reply
4. Done
   - 4 file hiện đã tách khá rõ giữa runtime hiện tại, validation evidence, và vNext design

## Kết luận phase

Phase 1 đã hoàn thành.

Từ thời điểm này:

- `UAP_SPECIFICATION.md` nên được hiểu là runtime contract hiện tại
- `UAP_VNEXT_PROPOSAL` là schema đích
- `UAP_VNEXT_IMPLEMENTATION_PLAN` là plan tổng thể để hiện thực schema đích

Phase tiếp theo nên bắt đầu là:

- `phase_2_parser_registry_refactor_plan.md`

## Rủi ro

- Nếu phase này làm không kỹ, team sẽ tiếp tục nhầm giữa schema runtime và schema đích.
- Nếu wording quá mơ hồ, developer có thể triển khai nhầm theo doc proposal thay vì runtime.

## Gợi ý review

Review theo 3 câu hỏi:

1. Runtime hiện parse được gì thật?
2. Validation đã chứng minh được gì?
3. vNext sẽ đổi gì nhưng chưa làm?
