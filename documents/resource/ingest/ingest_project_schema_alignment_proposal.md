# Đề xuất schema ingest-srv tương thích ingest-srv và project-srv

**Phiên bản:** 1.5  
**Ngày cập nhật:** 05/03/2026

## 0. Document Status

- **Status:** Derived
- **Canonical reference:** `/mnt/f/SMAP_v2/cross-service-docs/proposal_chuan_hoa_docs_3_service_v1.md`
- **RabbitMQ canonical:** `/mnt/f/SMAP_v2/scapper-srv/RABBITMQ.md`

### 0.1 Implementation Status Snapshot (schema)

| Hạng mục | Trạng thái |
|---|---|
| `data_sources`, `dryrun_results`, `scheduled_jobs`, `external_tasks`, `raw_batches`, `crawl_mode_changes` | Implemented trong migration v1 hiện tại |
| `crawl_targets` | Pending migration |
| `target_id` nullable cho `dryrun_results`, `scheduled_jobs`, `external_tasks` | Pending migration |

### 0.2 Deprecation Mapping

| Deprecated | Canonical |
|---|---|
| `/sources/*` | `/datasources/*` |
| `PUT /ingest/sources/{id}/crawl-mode` | `PUT /ingest/datasources/{id}/crawl-mode` |
| `ingest.data.first_batch` | `ingest.crawl.completed` |

## 1. Mục tiêu

Tài liệu này nhằm:

- Phân tích liên kết giữa `ingest-srv` và `project-srv`.
- Chốt ranh giới ownership giữa 2 service.
- Đề xuất schema `schema_ingest` đủ để phục vụ:
  - quản lý data source
  - onboarding / mapping
  - dry run
  - scheduler
  - tích hợp RabbitMQ với crawler bên thứ ba
  - publish UAP sang analysis
  - hỗ trợ adaptive crawl / crisis flow từ `project-srv`
- Loại bỏ các ambiguity còn lại để implementer có thể viết migration SQL, model và runtime flow mà không phải tự quyết định thêm.

---

## 2. Kết luận liên kết giữa 2 service

### 2.1 `project-srv` sở hữu

- `campaigns`
- `projects`
- `project_crisis_config`
- `crisis_alerts`
- quyết định nghiệp vụ liên quan đến activate project, pause/resume project, crisis detection và adaptive crawl ở mức business rule

### 2.2 `ingest-srv` sở hữu

- `data_sources`
- `crawl_targets`
- `mapping_rules`
- `dryrun_results`
- `scheduled_jobs`
- `external_tasks`
- `raw_batches`
- `crawl_mode_changes`
- webhook config
- toàn bộ trạng thái vận hành thực tế của source

### 2.3 Nguyên tắc liên kết

- `project_id` là logical foreign key, không dùng foreign key DB cross-service.
- `ingest-srv` là source of truth cho metadata và lifecycle của data source.
- `project-srv` chỉ gửi command/event hoặc gọi ingest API, không sở hữu bảng vật lý `data_sources`.

---

## 3. Contract liên service cần giữ

### 3.1 Kafka

Các event cần giữ giữa `project-srv` và `ingest-srv`:

- `project.activated`
- `project.paused`
- `project.resumed`
- `project.archived`
- `ingest.source.created`
- `ingest.source.activated`
- `ingest.source.paused`
- `ingest.source.resumed`
- `ingest.source.deleted`
- `ingest.dryrun.completed`
- `ingest.crawl.completed`
- `ingest.crawl_mode.changed`

### 3.2 HTTP

Contract HTTP quan trọng cần giữ:

- `PUT /ingest/datasources/{id}/crawl-mode`

### 3.3 RabbitMQ

- Tài liệu **canonical** cho queue/action/payload giao tiếp crawler là `/mnt/f/SMAP_v2/scapper-srv/RABBITMQ.md`.
- `ingest-srv` publish task sang crawler bên thứ ba theo contract canonical này.
- `ingest-srv` nhận kết quả crawl từ RabbitMQ, tải raw từ MinIO, rồi chuẩn hóa về UAP.
- `documents/resource/ingest-intergrate-3rdparty/RABBITMQ.md` là bản mirror (derived) để tiện theo dõi trong ingest repo, không phải nguồn sự thật gốc.

### 3.4 Analysis input

- Dùng 1 topic thống nhất để đẩy dữ liệu cho analysis: `smap.collector.output`.
- Không vận hành song song nhiều topic input nếu không có cơ chế compatibility rõ ràng.

---

## 4. Các mâu thuẫn hiện tại và cách chốt

### 4.1 Mâu thuẫn ownership source

Một số tài liệu trong `project-srv` mô tả như thể project quản lý source trực tiếp hoặc cập nhật source trong DB. Tuy nhiên:

- `master_proposal.md` xác định `Data Source` do Ingest quản lý.
- `implementation_status.md` của `project-srv` cũng cho thấy phần data source còn thiếu và chưa nên nằm trong ownership chính của project.

### 4.2 Quyết định chốt

- `ingest-srv` là source of truth cho source metadata.
- `project-srv` chỉ:
  - quản lý project và crisis config
  - phát command/event
  - gọi ingest API khi cần đổi crawl mode hoặc đồng bộ workflow

### 4.3 Mâu thuẫn topic sang analysis

Hiện tài liệu có cả:

- `smap.collector.output`
- `analytics.uap.received`

Quyết định chốt:

- dùng `smap.collector.output` làm topic input chính cho analysis
- coi các tên topic khác là draft/legacy nếu chưa triển khai thực tế

---

## 5. Ownership được chốt

### 5.1 `project-srv`

Quản lý:

- campaign/project
- config trạng thái setup của project
- crisis configuration
- crisis alert
- business decision khi nào cần tăng/giảm crawl

### 5.2 `ingest-srv`

Quản lý:

- data source vật lý
- schedule vận hành
- dry run và onboarding
- webhook endpoint / secret
- task gửi crawler bên thứ ba
- raw storage tracking
- UAP publishing lineage

---

## 6. ENUM schema đề xuất

Các ENUM cần có trong `schema_ingest`:

### 6.1 `source_type`

- `TIKTOK`
- `FACEBOOK`
- `YOUTUBE`
- `FILE_UPLOAD`
- `WEBHOOK`

### 6.2 `source_category`

- `CRAWL`
- `PASSIVE`

### 6.3 `source_status`

- `PENDING`
- `READY`
- `ACTIVE`
- `PAUSED`
- `FAILED`
- `COMPLETED`
- `ARCHIVED`

Ghi chú:

- `COMPLETED` chỉ áp dụng cho source kiểu one-shot như `FILE_UPLOAD`.
- Với source kiểu `CRAWL`, source thường luân chuyển giữa `READY`, `ACTIVE`, `PAUSED`, `FAILED`, `ARCHIVED`.

### 6.4 `onboarding_status`

- `NOT_REQUIRED`
- `PENDING`
- `ANALYZING`
- `SUGGESTED`
- `CONFIRMED`
- `FAILED`

### 6.5 `dryrun_status`

- `NOT_REQUIRED`
- `PENDING`
- `RUNNING`
- `SUCCESS`
- `WARNING`
- `FAILED`

### 6.6 `crawl_mode`

- `SLEEP`
- `NORMAL`
- `CRISIS`

### 6.7 `job_status`

- `PENDING`
- `RUNNING`
- `SUCCESS`
- `PARTIAL`
- `FAILED`
- `CANCELLED`

### 6.8 `batch_status`

- `RECEIVED`
- `DOWNLOADED`
- `PARSED`
- `FAILED`

Ghi chú:

- `batch_status` chỉ phản ánh vòng đời của raw batch trong ingest pipeline.
- Trạng thái publish sang Kafka được tách riêng qua `publish_status` để không trộn concern parse và concern publish.

### 6.9 `publish_status`

- `PENDING`
- `PUBLISHING`
- `SUCCESS`
- `FAILED`

### 6.10 `target_type`

- `KEYWORD`
- `PROFILE`
- `POST_URL`

### 6.11 `trigger_type`

- `MANUAL`
- `SCHEDULED`
- `PROJECT_EVENT`
- `CRISIS_EVENT`
- `WEBHOOK_PUSH`

---

## 7. Các bảng cần có trong `schema_ingest`

### 7.1 `data_sources`

Dùng để lưu:

- source config
- source lifecycle
- crawl mode
- crawl interval
- onboarding state
- dryrun state
- webhook config
- operational state hiện tại

#### Các field quan trọng cần có

- `id`: định danh duy nhất của source, dùng làm khóa chính và làm reference xuyên suốt toàn bộ flow.
- `project_id`: liên kết logic source này với project nào để `project-srv` có thể điều phối đúng phạm vi.
- `name`: tên hiển thị trên UI để user dễ phân biệt nhiều source trong cùng một project.
- `description`: mô tả bổ sung cho source, hữu ích khi source có cùng platform nhưng khác mục tiêu crawl.
- `source_type`: xác định platform hoặc loại nguồn (`TIKTOK`, `FILE_UPLOAD`...) để áp đúng logic ingest.
- `source_category`: phân biệt `CRAWL` và `PASSIVE`, vì hai nhóm này có lifecycle và scheduler khác nhau.
- `status`: trạng thái vận hành hiện tại của source để UI, scheduler và worker cùng hiểu chung một state.
- `config`: lưu cấu hình crawl options chung (không chứa targets — targets nằm ở bảng `crawl_targets`). Với FILE_UPLOAD/WEBHOOK lưu cấu hình parse.
- `account_ref`: **deprecated trong V1.4** — thông tin page/profile/keyword chuyển sang bảng `crawl_targets`.
- `mapping_rules`: lưu rule chuẩn hóa raw -> UAP; trong V1 field này là `JSONB`, đủ cho file upload và webhook mà chưa cần tách bảng riêng.
- `onboarding_status`: theo dõi tiến trình AI mapping/onboarding để biết source đã sẵn sàng hay chưa.
- `dryrun_status`: theo dõi kết quả dry run cho crawl source trước khi activate chính thức.
- `dryrun_last_result_id`: trỏ tới kết quả dry run gần nhất để UI/API truy xuất nhanh.
- `crawl_mode`: trạng thái crawl hiện tại (`SLEEP`, `NORMAL`, `CRISIS`) phục vụ adaptive crawl. Áp dụng ở **level datasource**, ảnh hưởng tất cả `crawl_targets` thuộc source này thông qua mode multiplier.
- `crawl_interval_minutes`: giá trị **default** dùng khi tạo `crawl_target` mới mà user không truyền interval riêng. Scheduling thực tế dựa trên `crawl_targets.crawl_interval_minutes`.
- `next_crawl_at`: **deprecated** — scheduling per-target nằm ở `crawl_targets.next_crawl_at`.
- `last_crawl_at`: **deprecated** — tracking per-target nằm ở `crawl_targets.last_crawl_at`.
- `last_success_at`: **deprecated** — tracking per-target nằm ở `crawl_targets.last_success_at`.
- `last_error_at`: **deprecated** — tracking per-target nằm ở `crawl_targets.last_error_at`.
- `last_error_message`: **deprecated** — tracking per-target nằm ở `crawl_targets.last_error_message`.
- `webhook_id`: định danh endpoint webhook public, dùng để route request vào đúng source.
- `webhook_secret_encrypted`: secret ký webhook phải được lưu mã hóa để verify chữ ký an toàn.
- `created_by`: truy vết user hoặc service actor đã tạo source để audit và phân quyền.
- `activated_at`: thời điểm source bắt đầu chạy thật, phục vụ lifecycle và báo cáo.
- `paused_at`: thời điểm source bị dừng để biết source đang dừng từ khi nào.
- `archived_at`: thời điểm archive để hỗ trợ soft-retention và cleanup.
- `created_at`: mốc tạo record.
- `updated_at`: mốc cập nhật gần nhất để đồng bộ UI và audit thay đổi.
- `deleted_at`: hỗ trợ soft delete thay vì xóa vật lý.

Ghi chú thiết kế:

- `project_id` xuất hiện ở `data_sources` và các bảng con là **intentional denormalization** để query theo project nhanh hơn.
- Không lưu `record_count`, `raw_batch_count` trong V1 để tránh counter bị stale; các số liệu này nên tính bằng query aggregation.
- Các field deprecated (`next_crawl_at`, `last_crawl_at`, `last_success_at`, `last_error_at`, `last_error_message`) vẫn giữ trong DB schema nhưng không dùng trong runtime scheduling. Có thể dùng làm summary aggregate nếu cần.

### 7.2 `dryrun_results`

Dùng để lưu:

- dry run theo source và project
- `sample_data`
- `warnings`
- `error_message`
- `status`

Khuyến nghị field chính:

- `id`: định danh kết quả dry run.
- `source_id`: gắn kết quả với source đã được test.
- `project_id`: hỗ trợ query theo project mà không cần join nhiều bước.
- `target_id`: liên kết với `crawl_target` cụ thể để biết dry run cho target nào; **nullable** vì FILE_UPLOAD/WEBHOOK không có target.
- `job_id`: liên kết với request, internal execution id hoặc event đã trigger dry run.
- `status`: kết quả cuối cùng của dry run (`SUCCESS`, `WARNING`, `FAILED`).
- `sample_count`: số mẫu thực tế lấy được để đánh giá source có usable hay không.
- `total_found`: tổng số item tìm thấy, giúp user biết source đang có dữ liệu hay không.
- `sample_data`: dữ liệu mẫu hiển thị cho user review.
- `warnings`: cảnh báo không chặn flow nhưng cần user chú ý.
- `error_message`: lỗi chính nếu dry run thất bại.
- `requested_by`: user hoặc system actor đã trigger dry run.
- `started_at`: thời điểm bắt đầu chạy.
- `completed_at`: thời điểm hoàn tất để đo thời gian chạy.
- `created_at`: thời điểm ghi record.

Ghi chú thiết kế:

- `target_id` phải set khi dry run cho một `crawl_target` cụ thể.
- `target_id = NULL` cho dry run FILE_UPLOAD hoặc WEBHOOK (không có target).

### 7.3 `scheduled_jobs`

Dùng để lưu:

- từng job crawl được scheduler tạo ra
- status chạy
- scheduled time
- retry và error

Khuyến nghị field chính:

- `id`: định danh từng lần scheduler tạo job.
- `source_id`: biết job này chạy cho source nào.
- `project_id`: hỗ trợ lọc job theo project để debug crisis hoặc adaptive crawl.
- `target_id`: liên kết job với `crawl_target` cụ thể để biết job crawl cho target nào; **nullable** cho job không gắn target cụ thể (batch job, FILE_UPLOAD).
- `status`: trạng thái xử lý job runtime.
- `trigger_type`: job được sinh do lịch định kỳ, manual hay do crisis event.
- `cron_expr`: lưu cron gốc để audit lịch chạy mà user đã cấu hình.
- `crawl_mode`: snapshot mode tại thời điểm tạo job để đối soát khi mode thay đổi giữa chừng.
- `scheduled_for`: thời điểm job đáng ra phải chạy.
- `started_at`: thời điểm worker thực sự bắt đầu xử lý.
- `completed_at`: thời điểm hoàn tất job.
- `retry_count`: số lần retry đã thực hiện.
- `error_message`: lỗi cuối cùng nếu job fail.
- `payload`: snapshot input cấu hình gửi xuống worker hoặc crawler để replay và debug.
- `created_at`: thời điểm tạo job record.

### 7.4 `external_tasks`

Dùng để tracking:

- request publish qua RabbitMQ
- `task_id`
- `routing_key`
- request payload
- response lifecycle

Khuyến nghị field chính:

- `id`: định danh nội bộ của task tracking.
- `source_id`: biết task này sinh từ source nào.
- `project_id`: hỗ trợ lọc và audit theo project.
- `target_id`: liên kết task với `crawl_target` cụ thể để biết task crawl cho target nào; **nullable** vì FILE_UPLOAD hoặc batch task có thể không gắn target.
- `scheduled_job_id`: nối task RabbitMQ với job scheduler đã sinh ra nó; field này **nullable** để support manual trigger, dry run, replay hoặc emergency trigger không đi qua cron scheduler.
- `task_id`: định danh task mà bên thứ 3 cũng dùng để phản hồi; đây là field quan trọng nhất để correlate request và response.
- `platform`: platform đích của crawler.
- `task_type`: loại task cụ thể (`search`, `comments`, `full_flow`...).
- `routing_key`: phục vụ debug RabbitMQ routing.
- `request_payload`: lưu request đã publish để replay và audit.
- `status`: trạng thái end-to-end của task với bên thứ 3.
- `published_at`: thời điểm task được publish thành công.
- `response_received_at`: thời điểm ingest nhận được phản hồi đầu tiên.
- `completed_at`: thời điểm task khép lại hoàn toàn.
- `error_message`: lưu lỗi nếu publish fail hoặc crawler trả lỗi.
- `created_at`: thời điểm tạo tracking record.

Ghi chú thiết kế:

- Nếu task sinh từ scheduler thì phải set `scheduled_job_id`.
- Nếu task sinh từ manual trigger, dry run hoặc replay thì `scheduled_job_id = NULL`.
- `target_id` phải set khi task crawl cho một target cụ thể.

### 7.5 `raw_batches`

Dùng để tracking:

- raw file hoặc batch nhận từ crawler hoặc upload hoặc webhook
- MinIO bucket/path
- item_count
- batch status
- parse và publish timestamps

Khuyến nghị field chính:

- `id`: định danh nội bộ của batch raw.
- `source_id`: biết batch đến từ source nào.
- `project_id`: phục vụ lọc theo project.
- `external_task_id`: liên kết về task RabbitMQ đã tạo ra batch này; field này nullable vì `FILE_UPLOAD`, `WEBHOOK` hoặc manual ingest có thể tạo raw batch mà không đi qua RabbitMQ task.
- `batch_id`: mã batch do crawler hoặc ingest cấp, dùng để dedup và đối soát.
- `status`: trạng thái xử lý batch raw trong ingest pipeline; chỉ phản ánh raw lifecycle (`RECEIVED`, `DOWNLOADED`, `PARSED`, `FAILED`).
- `storage_bucket`: bucket MinIO đang giữ raw file.
- `storage_path`: path chính xác để worker tải về parse.
- `storage_url`: URL tham chiếu nếu cần truy cập trực tiếp hoặc chia sẻ nội bộ.
- `item_count`: số item trong batch để đối chiếu với số UAP publish ra.
- `size_bytes`: kích thước file phục vụ monitoring và planning.
- `checksum`: chống parse trùng, hỗ trợ integrity check.
- `received_at`: thời điểm ingest nhận batch.
- `parsed_at`: thời điểm parse xong batch raw.
- `publish_status`: trạng thái publish UAP của batch này sang Kafka; được tách riêng khỏi `batch_status` để phản ánh publish lifecycle (`PENDING`, `PUBLISHING`, `SUCCESS`, `FAILED`).
- `publish_record_count`: số record UAP đã publish từ batch này.
- `first_event_id`: event đầu tiên được sinh ra từ batch này để trace.
- `last_event_id`: event cuối cùng được sinh ra từ batch này để trace.
- `uap_published_at`: thời điểm batch đã được đẩy hết sang analysis.
- `error_message`: lỗi parse raw nếu có.
- `publish_error`: lỗi publish sang Kafka nếu có.
- `raw_metadata`: metadata phụ từ crawler response như `crawler_version`, `duration`, `partial_flag`.
- `created_at`: thời điểm tạo record.

Ghi chú thiết kế:

- Batch mới nhận về: `status = RECEIVED`, `publish_status = PENDING`.
- Parse thành công nhưng chưa publish: `status = PARSED`, `publish_status = PENDING`.
- Đang publish UAP: `status = PARSED`, `publish_status = PUBLISHING`.
- Publish thành công: `status = PARSED`, `publish_status = SUCCESS`.
- Parse fail: `status = FAILED`, `publish_status = PENDING`.
- Publish fail: `status = PARSED`, `publish_status = FAILED`.
- V1 không có `PARTIAL_SUCCESS`; nếu publish một phần thất bại thì toàn batch được xem là `FAILED` ở publish layer và xử lý theo hướng retry full batch.

### 7.6 `crawl_mode_changes`

Dùng để tracking:

- lịch sử chuyển crawl mode theo crisis feedback loop hoặc thao tác manual
- transition `NORMAL -> CRISIS -> NORMAL`
- actor, event nguồn và lý do đổi mode

Khuyến nghị field chính:

- `id`: định danh của một lần thay đổi mode.
- `source_id`: biết source nào bị đổi mode.
- `project_id`: hỗ trợ query theo project khi phân tích crisis trên toàn project.
- `trigger_type`: lý do thay đổi mode là `MANUAL`, `PROJECT_EVENT`, `CRISIS_EVENT` hay trigger khác.
- `from_mode`: mode trước khi thay đổi.
- `to_mode`: mode sau khi thay đổi.
- `from_interval_minutes`: interval cũ để biết hệ thống thay đổi mạnh tới đâu.
- `to_interval_minutes`: interval mới sau khi đổi mode.
- `reason`: diễn giải human-readable vì sao mode bị đổi.
- `event_ref`: reference tới event gốc, alert id hoặc request id đã trigger việc đổi mode; field này nullable vì manual change có thể không có nguồn event.
- `triggered_by`: user hoặc service actor đã tạo ra quyết định này.
- `triggered_at`: thời điểm quyết định đổi mode được áp dụng.
- `created_at`: thời điểm ghi log vào DB.

Ghi chú thiết kế:

- `event_ref = NULL` với manual change.
- `event_ref = event_id` nếu thay đổi đến từ Kafka event.
- `event_ref = alert_id` nếu thay đổi đến từ crisis alert record.
- `event_ref = request_id` nếu thay đổi đến từ admin hoặc internal API.

### 7.7 `crawl_targets`

Dùng để lưu:

- danh sách target crawl (keyword, profile, post URL) của mỗi data source
- tần suất crawl riêng per-target
- trạng thái scheduling per-target
- operational state per-target

Khuyến nghị field chính:

- `id`: định danh duy nhất per-target.
- `data_source_id`: liên kết target thuộc source nào.
- `target_type`: loại target (`KEYWORD`, `PROFILE`, `POST_URL`).
- `value`: giá trị target (từ khóa, URL profile, URL post).
- `label`: display label cho UI.
- `platform_meta`: metadata platform-specific dạng JSONB (profile_id, hashtag_id...).
- `is_active`: tạm tắt target mà không xóa, scheduler bỏ qua target `is_active = false`.
- `priority`: thứ tự ưu tiên khi crisis mode, target priority cao crawl trước.
- `crawl_interval_minutes`: tần suất crawl riêng cho target này; default kế thừa từ `data_sources.crawl_interval_minutes` khi tạo.
- `next_crawl_at`: mốc scheduler dùng để pick target nào đến hạn.
- `last_crawl_at`: lần crawl gần nhất per-target.
- `last_success_at`: lần thành công gần nhất per-target.
- `last_error_at`: lần lỗi gần nhất per-target.
- `last_error_message`: lỗi gần nhất per-target.
- `created_at`: mốc tạo record.
- `updated_at`: mốc cập nhật gần nhất.

Ghi chú thiết kế:

- Scheduler query `crawl_targets.next_crawl_at WHERE is_active = true` thay vì `data_sources.next_crawl_at`.
- Effective interval tại runtime = `target.crawl_interval_minutes × mode_multiplier(source.crawl_mode)`.
- Mode multiplier: `NORMAL = 1.0`, `CRISIS = 0.2`, `SLEEP = 5.0`.
- `crawl_mode` nằm ở **level datasource** (ảnh hưởng tất cả targets cùng lúc), không per-target.
- Khi tạo target mới: `crawl_interval_minutes = input.interval ?? source.crawl_interval_minutes ?? crawl_mode_defaults(mode)`.
- `crawl_mode_defaults` chỉ dùng làm default/fallback để seed cấu hình; runtime scheduling không dùng bảng này để ghi đè interval đã chốt trên target.
- Index đề xuất: `(data_source_id)` và `(next_crawl_at) WHERE is_active = true`.

> **Ghi chú về API contract**: CRUD endpoints cho `crawl_targets` (`POST/GET/PUT/DELETE /datasources/:id/targets`) được mô tả chi tiết trong `ingest_plan.md` (Module 1 – Data Source Management). Schema proposal chỉ định nghĩa cấu trúc bảng và business rules, không định nghĩa HTTP contract.

### 7.8 Các bảng defer khỏi V1

Các bảng sau **không cần tạo trong V1**, nhưng có thể bổ sung ở V2 khi nghiệp vụ thực sự cần:

- `source_onboarding_snapshots`: chỉ cần khi AI mapping preview/history được triển khai đầy đủ.
- `webhook_deliveries`: chỉ cần khi webhook source đã chạy production.
- `source_event_outbox`: chỉ cần khi ingest cần guarantee mạnh hơn cho event publishing.

---

## 8. JSON config shape theo từng loại source

> **Ghi chú V1.4:** Targets (keywords, profile links, post links) đã chuyển sang bảng `crawl_targets`. `config` chỉ chứa crawl options chung.

### 8.1 Crawl

Khuyến nghị dùng các field sau trong `config` (chỉ crawl options chung, **không chứa targets**):

- `max_results`
- `timeout_seconds`
- `batch_size`
- `comment_limit`
- `include_replies`
- `language_filter`
- `filters`
- `platform_options`

Targets (keywords, profile links, post URLs) được quản lý qua bảng `crawl_targets` với CRUD riêng.

### 8.2 File upload

Khuyến nghị dùng các field sau trong `config`:

- `file_type`
- `delimiter`
- `sheet_name`
- `header_row`
- `encoding`
- `sample_ref`
- `mapping_version`

### 8.3 Webhook

Khuyến nghị dùng các field sau trong `config`:

- `payload_schema`
- `signature_header`
- `signature_algo`
- `secret_ref`
- `accept_batch`
- `mapping_version`

### 8.4 `mapping_rules` JSON shape

Trong V1, `mapping_rules` được lưu trực tiếp trong `data_sources.mapping_rules` dưới dạng `JSONB`.

#### JSON shape đề xuất

```json
{
  "version": "1.0",
  "source_type": "FILE_UPLOAD",
  "fields": [
    {
      "source_field": "Ý kiến khách hàng",
      "target_path": "content.text",
      "required": true,
      "transforms": []
    },
    {
      "source_field": "Ngày gửi",
      "target_path": "content.published_at",
      "required": true,
      "transforms": [
        {
          "type": "datetime_parse",
          "input_format": "DD/MM/YYYY HH:mm",
          "timezone": "Asia/Ho_Chi_Minh"
        }
      ]
    },
    {
      "source_field": "Tên KH",
      "target_path": "content.author.display_name",
      "required": false,
      "transforms": [
        {
          "type": "trim"
        }
      ]
    }
  ],
  "defaults": {
    "content.doc_type": "feedback",
    "content.author.author_type": "customer",
    "ingest.source.source_type": "FILE_UPLOAD"
  },
  "validation": {
    "drop_if_missing_required": true,
    "allow_unknown_fields": true,
    "on_required_transform_error": "DROP_RECORD",
    "on_optional_transform_error": "SET_NULL_AND_WARN"
  }
}
```

#### Rule semantics

- `version`: version của rule để support evolve format sau này.
- `source_type`: loại source mà rule này áp dụng.
- `fields[]`: danh sách mapping chính từ input source sang target UAP.
- `source_field`: cột, header hoặc input key gốc.
- `target_path`: đường dẫn field đích trong UAP.
- `required`: có bắt buộc map thành công không.
- `transforms[]`: chuỗi transform được áp theo thứ tự khai báo.
- `defaults`: giá trị mặc định nếu source không có field tương ứng.
- `validation`: rule xử lý record lỗi hoặc record thiếu dữ liệu.

#### Error behavior khi transform fail

- Nếu input field thiếu:
  - `required = true` và không có default -> drop record
  - `required = false` -> set `null`
- Nếu input field có giá trị nhưng transform fail:
  - `required = true` -> drop record
  - `required = false` -> set `null` và ghi warning
- Nếu một field có nhiều transform:
  - dừng pipeline transform của field đó ngay tại bước fail đầu tiên
  - sau đó áp rule theo `required`
- Không dùng silent skip, vì sẽ làm mơ hồ giữa trường hợp field thực sự không có dữ liệu và trường hợp transform xử lý lỗi.

Ví dụ:

- `datetime_parse(\"abc\")` cho field `content.published_at`:
  - nếu field này là `required` -> drop record
  - nếu field này là `optional` -> set `published_at = null`, giữ record và ghi warning

#### Transform types cần hỗ trợ ở V1

- `trim`
- `lowercase`
- `uppercase`
- `datetime_parse`
- `number_parse`
- `boolean_parse`
- `array_split`
- `static_value`
- `concat`
- `regex_extract`

---

## 9. Cách schema hỗ trợ các luồng nghiệp vụ

### 9.1 Create source

- User tạo source.
- Ingest lưu vào `data_sources`.
- Source mới bắt đầu ở trạng thái `PENDING`.

### 9.2 Passive source onboarding

- Với `FILE_UPLOAD` hoặc `WEBHOOK`, ingest thực hiện onboarding bằng AI mapping.
- Trong V1, mapping được lưu trực tiếp vào `data_sources.mapping_rules`.
- Khi cần lưu history preview hoặc confirm đầy đủ, V2 mới bổ sung `source_onboarding_snapshots`.
- Sau confirm, source chuyển sang trạng thái sẵn sàng.

### 9.3 Dry run cho crawl source

- Crawl source thực hiện dry run **per-target**: mỗi `crawl_target` có thể chạy dry run riêng.
- Kết quả lưu ở `dryrun_results` với `target_id` gắn với target cụ thể.
- Trong runtime, dry run có thể tạo `external_tasks` với `scheduled_job_id = NULL` và `target_id` set.
- Dry run thành công thì source được chuyển sang `READY`.

### 9.4 Activate project

- `project-srv` phát `project.activated`.
- `ingest-srv` nhận event.
- Tất cả source hợp lệ thuộc project chuyển sang `ACTIVE`.

### 9.5 Scheduler và job runtime

- Scheduler query `crawl_targets.next_crawl_at WHERE is_active = true` để tìm target đến hạn.
- Tính effective interval: `target.crawl_interval_minutes × mode_multiplier(source.crawl_mode)`.
- Tạo `scheduled_jobs` cho từng lần crawl per-target.
- Worker publish task sang RabbitMQ và ghi `external_tasks` với `target_id` set.
- Nếu task sinh từ scheduler thì `external_tasks.scheduled_job_id` phải được set.

### 9.6 Manual trigger hoặc replay

- API hoặc internal action có thể tạo `external_tasks` trực tiếp.
- Trường hợp này không bắt buộc phải có `scheduled_jobs`.
- `external_tasks.scheduled_job_id = NULL`.
- Replay chỉ re-run parse/publish trên `raw_batches` đã tồn tại, không tạo raw batch mới.
- Replay mặc định chỉ cho batch lỗi (`status = FAILED` hoặc `publish_status = FAILED`).
- Replay batch `SUCCESS` yêu cầu `force = true` và phải ghi audit rõ đây là replay cưỡng bức.

### 9.7 Nhận raw từ crawler và publish sang analysis

- Ingest nhận result từ RabbitMQ.
- Lưu tracking raw vào `raw_batches`.
- Worker parse raw thành nhiều UAP record.
- Khi bắt đầu publish sang Kafka, cập nhật `raw_batches.publish_status = PUBLISHING`.
- Publish thành công thì cập nhật `publish_status = SUCCESS`, `publish_record_count`, `first_event_id`, `last_event_id`, `uap_published_at`.
- Nếu publish lỗi thì giữ `status = PARSED` và set `publish_status = FAILED` để có thể retry.
- RabbitMQ duplicate response theo cùng `task_id` phải xử lý idempotent: không tạo `external_tasks`/`raw_batches` mới.
- Raw batch idempotency key chính: `(source_id, batch_id)`; fallback detect duplicate: `checksum`.
- Webhook idempotency key chính: `webhook_id + external_event_id`; fallback: `webhook_id + checksum(payload)`.

### 9.8 Crisis mode change

- `project-srv` gửi command hoặc event đổi crawl mode.
- Ingest cập nhật `data_sources.crawl_mode` (ở level datasource, ảnh hưởng tất cả targets).
- Scheduler tự động tính lại effective interval cho mỗi target bằng `target.crawl_interval_minutes × mode_multiplier`.
- Mode multiplier: `NORMAL = 1.0`, `CRISIS = 0.2`, `SLEEP = 5.0`.
- Đồng thời ghi `crawl_mode_changes` để lưu transition và nguyên nhân đổi mode.
- Nếu có event nguồn thì ghi vào `event_ref` để trace ngược.

---

## 10. Test cases bắt buộc

### 10.1 Source lifecycle

- tạo `TIKTOK` source -> `PENDING`
- thêm crawl targets vào source
- dry run per-target success -> `READY`
- activate -> `ACTIVE` và tất cả targets có `next_crawl_at`
- pause/resume source hoạt động đúng (tất cả targets dừng/tiếp)
- `FILE_UPLOAD` source có thể đi tới `COMPLETED`
- `CRAWL` source không dùng `COMPLETED`

### 10.2 `external_tasks.scheduled_job_id`

- scheduled crawl tạo `external_task` với `scheduled_job_id != NULL`
- manual trigger tạo `external_task` với `scheduled_job_id = NULL`
- dry run tạo `external_task` với `scheduled_job_id = NULL`
- replay tạo `external_task` với `scheduled_job_id = NULL`

### 10.3 `mapping_rules`

- validate JSON shape hợp lệ
- parse `datetime_parse`
- apply `defaults`
- reject record nếu thiếu field required và `drop_if_missing_required = true`
- allow unknown fields nếu `allow_unknown_fields = true`
- required field transform fail -> drop record
- optional field transform fail -> set `null` và ghi warning

### 10.4 `crawl_mode_changes`

- mode change manual có `event_ref = NULL`
- mode change từ crisis event có `event_ref = event_id`
- mode change từ project event có `event_ref = event_id` hoặc `alert_id`
- verify `from_mode -> to_mode` được lưu đúng

### 10.5 `raw_batches`

- parse fail:
  - `status = FAILED`
  - `publish_status = PENDING`
- parse success, chưa publish:
  - `status = PARSED`
  - `publish_status = PENDING`
- publish running:
  - `status = PARSED`
  - `publish_status = PUBLISHING`
- publish success:
  - `publish_status = SUCCESS`
  - `publish_record_count > 0`
  - `first_event_id`, `last_event_id` được set
- publish fail:
  - `status = PARSED`
  - `publish_status = FAILED`
  - `publish_error` có giá trị

### 10.6 Query by `project_id`

- list all source runtime state by `project_id`
- list all jobs by `project_id`
- list all external tasks by `project_id`
- list all raw batches by `project_id`
- verify denormalized query không cần join bắt buộc

---

### 10.7 `crawl_targets`

- tạo target KEYWORD cho TIKTOK source -> `crawl_interval_minutes` kế thừa từ datasource nếu không truyền
- tạo target với interval riêng -> dùng giá trị truyền vào
- tạm tắt target (`is_active = false`) -> scheduler bỏ qua target này
- bật lại target -> `next_crawl_at` được tính lại
- effective interval = `crawl_interval_minutes × mode_multiplier`
- NORMAL mode: multiplier = 1.0
- CRISIS mode: multiplier = 0.2 -> interval giảm 5 lần
- SLEEP mode: multiplier = 5.0 -> interval tăng 5 lần
- dry run per-target lưu `dryrun_results.target_id` đúng

---

## 11. Assumptions và default values

- `ingest-srv` là owner duy nhất của `data_sources` và `crawl_targets`.
- `project_id` là logical reference, không có DB FK cross-service.
- `project_id` được lặp lại ở bảng con để tối ưu filter và query.
- `mapping_rules` chỉ reuse ở mức source-specific trong V1.
- `NOT_REQUIRED` được giữ trong enum, không dùng `NULL`.
- third-party contract chính thức chỉ là `documents/resource/ingest-intergrate-3rdparty/RABBITMQ.md`.
- topic input analysis chính là `smap.collector.output`.
- default crawl interval (dùng làm default cho `data_sources.crawl_interval_minutes`):
  - `SLEEP = 60`
  - `NORMAL = 11`
  - `CRISIS = 2`
- mode multiplier (dùng để tính effective interval tại runtime):
  - `NORMAL = 1.0`
  - `CRISIS = 0.2`
  - `SLEEP = 5.0`
- `crawl_mode_defaults` là default/fallback config, không phải runtime policy override interval của target.
- `crawl_mode` ở level datasource (ảnh hưởng tất cả `crawl_targets`).
- `crawl_interval_minutes` trên `data_sources` là default; per-target override nằm ở `crawl_targets`.
- `account_ref` deprecated từ V1.4; targets lưu ở `crawl_targets`.
