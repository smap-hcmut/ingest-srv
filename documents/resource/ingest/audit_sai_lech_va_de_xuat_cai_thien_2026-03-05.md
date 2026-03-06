# Audit sai lệch & đề xuất cải thiện trước khi triển khai

**Phạm vi:** `project-srv`, `ingest-srv`, `scapper-srv`  
**Ngày rà soát:** 05/03/2026  
**Nguyên tắc đối chiếu:** `runtime code` > `migration/model` > `documents`

---

## 1) Kết luận nhanh

Hệ thống hiện tại **chưa sẵn sàng triển khai liên thông 3 service** theo thiết kế trong tài liệu.  
Các sai lệch lớn tập trung ở: **contract API**, **contract event/queue**, **schema ingest thực tế chưa theo proposal**, và **tài liệu chưa đồng bộ với code**.

---

## 2) Các điểm đang sai/lệch và đề xuất cải thiện

| # | Hạng mục | Hiện trạng sai/lệch | Tác động | Đề xuất cải thiện | Ưu tiên |
|---|---|---|---|---|---|
| 1 | Contract kết quả crawl giữa `ingest-srv` và `scapper-srv` | `scapper-srv` hiện xử lý queue và ghi file vào `output/`; chưa có contract trả kết quả chuẩn để `ingest-srv` consume end-to-end như proposal | Không thể hoàn tất pipeline ingest -> parse -> publish UAP ổn định | Chốt 1 contract chính thức 2 chiều: queue kết quả + payload + tham chiếu MinIO + idempotency key + retry/DLQ | **CRITICAL** |
| 2 | Schema ingest theo proposal vs DB thực tế | Proposal/plan nói có `crawl_targets`, `target_id` cho dryrun/job/task; migration `001` hiện chưa có | Không triển khai được scheduler per-target, dryrun per-target, trace theo target | Tạo migration bổ sung `crawl_targets`; thêm `target_id` nullable cho `dryrun_results`, `scheduled_jobs`, `external_tasks`; regen sqlboiler/model | **CRITICAL** |
| 3 | API ingest runtime chưa mở các endpoint đã thiết kế | Runtime mới mount `/api/v1/ingest/ping`; route datasource chưa được map vào HTTP server | Service không dùng được theo tài liệu tích hợp | Mount đầy đủ routes datasource/internal; xuất OpenAPI/Swagger để khóa contract | **HIGH** |
| 4 | Namespace API chưa thống nhất (`sources` vs `datasources`) | Tài liệu project cũ còn dùng `/sources`; ingest docs/code dùng `datasources`; annotation handler vẫn `@Router /sources` | Dễ tích hợp sai endpoint, sai SDK client | Chốt canonical: `datasources`; cập nhật toàn bộ docs + swagger annotation + examples | **HIGH** |
| 5 | Event lifecycle project↔ingest mới ở mức tài liệu | Có mô tả `project.activated/paused/resumed/archived` và `ingest.*`; code producer/consumer chưa chạy nghiệp vụ thật | Luồng activate/pause/resume/archive không vận hành | Implement tối thiểu: producer project + consumer ingest + payload schema versioned + test tích hợp | **HIGH** |
| 6 | Topic/event naming chưa khóa cứng | Config runtime đang dùng topic chung (`project.events`, `ingest.events`), trong docs có thêm tên canonical nghiệp vụ | Có nguy cơ publish/consume lệch topic | Lập bảng map topic chuẩn theo môi trường; validate config khi boot service | **HIGH** |
| 7 | Migration `project-srv` chưa nhất quán schema qualification | `000002_add_crisis_config.sql` dùng định danh chưa prefix schema rõ như `init_schema.sql` | Rủi ro lệch DDL theo môi trường/search_path | Chuẩn hóa migration với `schema_project.*` đầy đủ, kiểm tra lại trên DB sạch | **HIGH** |
| 8 | README `project-srv` chưa phản ánh đúng API/runtime hiện tại | README còn mô tả endpoint/trạng thái cũ khác code hiện tại | Team khác đọc sai, gọi sai API | Cập nhật README theo route thực tế + link swagger hiện hành | **MEDIUM** |
| 9 | Thiếu OpenAPI cho ingest | Ingest chưa có artifact swagger/openapi chính thức | Không có nguồn contract machine-readable cho client/gateway | Sinh swagger từ annotation đã chuẩn hóa + thêm bước CI check drift | **MEDIUM** |
| 10 | Thiếu tài liệu chính sách retry/idempotency ở mức implementable | Có nêu định hướng nhưng chưa thành quy chuẩn runtime bắt buộc | Dễ phát sinh duplicate/mất dữ liệu khi lỗi MQ/webhook/replay | Chốt policy: key, TTL, hành vi duplicate, trạng thái retry, dead-letter flow | **MEDIUM** |
| 11 | Tài liệu ownership còn lẫn ở vài nơi | Có đoạn mô tả khiến hiểu nhầm project quản lý source vật lý | Chồng chéo trách nhiệm và tranh chấp thiết kế | Chốt rõ: `ingest-srv` owner `data_sources/crawl_targets`; `project-srv` chỉ logical reference + command/event | **MEDIUM** |

---

## 3) Các “cổng chặn” bắt buộc trước khi code business flow

### Gate A — Contract tích hợp ingest ↔ scapper
- Chốt queue request/response, payload schema version, idempotency/retry/DLQ.
- Viết test hợp đồng tối thiểu cho 3 queue: `tiktok_tasks`, `facebook_tasks`, `youtube_tasks`.

### Gate B — Khớp schema ingest với proposal đã chọn
- Bổ sung `crawl_targets` và các cột `target_id` theo luồng per-target.
- Regenerate `sqlboiler`, update model/usecase/repository tương ứng.

### Gate C — Khớp API runtime với tài liệu
- Mount đầy đủ endpoint ingest đã thống nhất.
- Đồng bộ namespace `datasources` trên docs, swagger, code annotation.

### Gate D — Khớp event project↔ingest
- Implement producer/consumer cho lifecycle events tối thiểu.
- Chuẩn hóa event payload và versioning; có test publish/consume cơ bản.

### Gate E — Chuẩn hóa tài liệu kỹ thuật cốt lõi
- Cập nhật README/service docs theo runtime thật.
- Bổ sung OpenAPI ingest và tài liệu lỗi chuẩn (error catalog).

---

## 4) Lộ trình cải thiện đề xuất (ngắn gọn)

1. **Tuần 1:** Chốt contract MQ + event payload + topic map, cập nhật docs canonical.  
2. **Tuần 2:** Cập nhật migration/schema ingest theo per-target, regen model, chạy kiểm tra DB sạch.  
3. **Tuần 3:** Mở endpoint ingest runtime + sinh OpenAPI + integration test cơ bản project↔ingest↔scapper.  
4. **Tuần 4:** Hoàn thiện retry/idempotency runtime + runbook xử lý lỗi + khóa release checklist.

---

## 5) Kết luận

Nếu chưa đóng các điểm **CRITICAL/HIGH** ở trên, việc triển khai nghiệp vụ sẽ có rủi ro lớn: lệch contract, không chạy end-to-end, và khó truy vết lỗi dữ liệu.  
Khuyến nghị: hoàn tất 5 gate trước khi bắt đầu implement sâu business modules.
