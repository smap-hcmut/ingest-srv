# Kế hoạch chi tiết Phase 3 cho ingest-srv

**Ngày cập nhật:** 2026-03-07  
**Phiên bản:** v1.0  
**Phạm vi:** `ingest-srv/internal`  
**Tiền đề:** Phase 2 control-plane đã có `datasource` lifecycle, `crawl_mode` audit, grouped `crawl_targets`, và dry run validation-only.

## 1. Kết luận ngắn

Phase 3 nên được hiểu là **execution plane tối thiểu** cho ingest crawl:

1. scheduler pick grouped target đến hạn  
2. tạo `scheduled_jobs`  
3. tạo + publish `external_tasks` sang RabbitMQ  
4. consume kết quả crawler  
5. persist `raw_batches` để Phase 4 parse/publish UAP

Phase 3 **không nên** kéo luôn parser/UAP publish hay project lifecycle Kafka consumer vào cùng scope. Nếu làm vậy, rủi ro tích hợp sẽ tăng mạnh và debug sẽ rất khó.

## 2. Mục tiêu của Phase 3

Biến grouped `crawl_target` từ record control-plane thành **execution unit thực sự**:

- có scheduler heartbeat thật
- có claim/dispatch target đến hạn
- có tracking end-to-end qua `scheduled_jobs` và `external_tasks`
- có ingest consumer nhận response envelope từ crawler
- có `raw_batches` để bàn giao cho Phase 4

## 3. Current baseline

### 3.1 Đã có

- grouped `crawl_targets.values[]`
- datasource lifecycle `READY/ACTIVE/PAUSED/...`
- internal `crawl-mode` API + audit `crawl_mode_changes`
- dry run control-plane only
- migration/model/sqlboiler cho:
  - `scheduled_jobs`
  - `external_tasks`
  - `raw_batches`
- scheduler server chỉ có heartbeat log
- consumer hiện là no-op
- canonical request/completion wire contract nằm ở `/mnt/f/SMAP_v2/scapper-srv/RABBITMQ.md`
- canonical shared runtime boundary/idempotency nằm ở `/mnt/f/SMAP_v2/ingest-srv/documents/plan/scapper_ingest_shared_runtime_contract_proposal.md`

### 3.2 Chưa có

- query lấy due grouped targets
- claim logic để tránh scheduler bắn trùng
- repository/usecase cho `scheduled_jobs`
- repository/usecase cho `external_tasks`
- RabbitMQ publisher adapter theo grouped target
- consumer xử lý response envelope từ crawler
- create `raw_batches` khi nhận kết quả
- idempotency runtime cho `task_id` và `batch_id`

## 4. Quyết định chốt trước khi hiện thực

### 4.1 Execution unit là grouped target

Một `crawl_target` là một execution unit duy nhất:

- `KEYWORD` -> 1 logical dispatch của grouped target, nhưng runtime có thể fan-out thành nhiều `external_tasks`
- `PROFILE` -> 1 task nhận `urls[]` hoặc contract profile batch tương ứng
- `POST_URL` -> 1 task nhận `urls[]` / `parse_ids[]` / `video_ids[]` tùy platform adapter

Scheduler, `scheduled_jobs`, `external_tasks`, `raw_batches`, health timestamps và error tracking đều bám theo `target_id` của grouped target. Một `scheduled_job` có thể sinh nhiều `external_tasks`.

### 4.2 Phase 3 chỉ làm crawl runtime, chưa parse/publish UAP

Phase 3 dừng ở:

- publish RabbitMQ
- consume result
- lưu raw response vào MinIO
- tạo `raw_batches`

Không làm trong Phase 3:

- parser platform-specific
- mapping transforms đầy đủ
- publish Kafka UAP
- replay parser pipeline hoàn chỉnh

### 4.3 Response persistence strategy: 1 external task -> 1 raw batch

Mặc định, mỗi response envelope thành công từ crawler sẽ tạo:

- 1 `external_task` được complete
- 1 `raw_batch` chứa **full raw envelope/result**

Với grouped keyword fan-out, một `crawl_target` có thể sinh nhiều `external_tasks`; mỗi `external_task` thành công vẫn tạo đúng 1 `raw_batch`.

Parser Phase 4 sẽ đọc raw batch này để tách tiếp thành nhiều post/comment/reply records.

### 4.4 Idempotency là bắt buộc từ ngày đầu

Phase 3 phải chốt 3 key:

- scheduler dedup: `(target_id, scheduled_for, trigger_type)`
- RabbitMQ dedup: `external_tasks.task_id`
- raw batch dedup:
  - key chính: `(source_id, batch_id)`
  - fallback: `checksum`

Không chấp nhận thiết kế “làm trước, idempotency thêm sau”.

### 4.5 Scheduler claim bằng DB, không giữ state trong memory

Không dùng in-memory registry làm source of truth cho due target.

Heartbeat scheduler phải:

- query target đến hạn từ DB
- claim/advance `next_crawl_at` bằng DB update có điều kiện
- rồi mới tạo `scheduled_job`

Mục tiêu là:

- restart process không mất state
- scale nhiều instance vẫn có đường để thêm locking sau

### 4.6 Platform/action support phải có runtime support flags

#### Nhóm enable ngay nếu contract canonical đã đủ rõ

- TikTok `KEYWORD` -> `full_flow`, fan-out 1 task cho mỗi keyword trong grouped target
- TikTok `POST_URL` -> `post_detail` với `urls[]`
- Facebook `POST_URL` -> `post_detail` nếu upstream đã có `parse_ids[]`/URL adapter rõ

#### Nhóm phải feature-flag hoặc block sớm nếu contract chưa chốt

- `PROFILE` cho mọi platform nếu scapper chưa có canonical action batch rõ
- Facebook/YouTube keyword search nếu upstream action còn `not yet` hoặc single-value only
- YouTube post/video grouped batch nếu canonical chưa support batch

Decision chốt:

- introducce một runtime support flag cho từng mapping:
  - `platform`
  - `target_type`
  - `action`
  - `is_enabled`
- flag được persist trong DB để restart service không mất state
- flag được đổi qua **admin-only endpoint**
- scheduler và manual trigger đều phải check flag trước khi build/publish task

Admin contract đề xuất:

- `GET /api/v1/ingest/admin/platform-action-flags`
- `PUT /api/v1/ingest/admin/platform-action-flags/{platform}/{target_type}/{action}`

Body tối thiểu:

```json
{
  "is_enabled": true,
  "reason": "enable youtube post batch after upstream release"
}
```

Rule runtime:

- nếu mapping không có trong support matrix -> fail `unsupported platform-action mapping`
- nếu mapping có nhưng flag `is_enabled = false` -> cũng fail `unsupported platform-action mapping`
- chỉ khi mapping tồn tại và flag `true` thì scheduler/manual trigger mới được publish

Default rollout cho Phase 3:

- tạo đủ support flags cho toàn bộ mapping mà Phase 3 biết tới
- **set tất cả flags = true trong Phase 3**
- cơ chế bật/tắt vẫn phải hiện thực ngay từ đầu để Phase 4+ không phải redesign

### 4.7 Manual trigger nội bộ thuộc Phase 3, nhưng chỉ là thin path

Phase 3 nên có internal/manual trigger tối thiểu:

- tạo 1 `scheduled_job` hoặc `external_task` manual
- dùng lại đúng publisher pipeline như scheduler

Không tạo một runtime path riêng biệt khác logic scheduler.

## 5. Kiến trúc Phase 3

### 5.1 Scheduler lane

`crawl_targets`  
-> claim due target  
-> create `scheduled_job`  
-> build request payload(s)  
-> create `external_tasks`  
-> publish RabbitMQ task(s)  
-> mark publish timestamps/status per task  
-> update target/source runtime timestamps

### 5.2 Consumer lane

RabbitMQ result envelope  
-> lookup `external_task` theo `task_id`  
-> idempotency check  
-> persist raw envelope vào MinIO  
-> create `raw_batch`  
-> update `external_task.status` + timestamps  
-> update target/source success or error snapshot

### 5.3 Out-of-scope lane

`raw_batches`  
-> parser  
-> transform  
-> publish UAP  

Lane này là Phase 4.

## 6. Workstreams

### Workstream 1: Repository foundation cho execution plane

**Mục tiêu**

- đủ data-access để Phase 3 không phải nhét SQL vào scheduler/consumer

**Việc cần làm**

1. Tạo module repository/usecase cho `scheduled_jobs`
2. Tạo module repository/usecase cho `external_tasks`
3. Tạo module repository/usecase cho `raw_batches`
4. Tạo module repository/usecase cho `platform_action_flags`
4. Thêm datasource repository functions:
   - list due grouped targets
   - claim target bằng conditional update
   - update target runtime fields:
     - `next_crawl_at`
     - `last_crawl_at`
     - `last_success_at`
     - `last_error_at`
     - `last_error_message`
5. Thêm datasource/source runtime snapshot update:
   - `last_crawl_at`
   - `last_success_at`
   - `last_error_at`
   - `last_error_message`

Schema tối thiểu cho `platform_action_flags`:

- `id`
- `platform`
- `target_type`
- `action`
- `is_enabled`
- `reason`
- `updated_by`
- `created_at`
- `updated_at`

Unique key:

- `(platform, target_type, action)`

**Done khi**

- scheduler, admin endpoint và consumer chỉ gọi usecase/repository rõ ràng, không viết SQL inline

### Workstream 2: Due-target selection và claim algorithm

**Mục tiêu**

- mỗi tick chỉ claim đúng target đến hạn
- hạn chế duplicate dispatch

**Việc cần làm**

1. Query due targets với điều kiện tối thiểu:
   - source `ACTIVE`
   - source `CRAWL`
   - target `is_active = true`
   - `next_crawl_at <= now`
2. Tính `effective_interval`:
   - `target.crawl_interval_minutes × mode_multiplier(source.crawl_mode)`
3. Claim target bằng update có điều kiện:
   - set `next_crawl_at = now + effective_interval`
   - set `last_crawl_at = now`
   - only if record vẫn còn đến hạn
4. Nếu claim fail do race:
   - skip target
   - không tạo job/task
5. Batch size mỗi tick:
   - chốt 1 config `heartbeat_limit`
   - tránh publish bùng nổ trong một tick

**Done khi**

- heartbeat thực sự tạo được danh sách target claimed
- cùng một target không bị dispatch trùng trong cùng một cửa sổ tick bình thường

### Workstream 3: Scheduled job materialization

**Mục tiêu**

- mọi lần scheduler quyết định chạy đều có audit/runtime record

**Việc cần làm**

1. Tạo `scheduled_job` cho mỗi grouped target đã claim
2. Snapshot tối thiểu vào job:
   - `source_id`
   - `project_id`
   - `target_id`
   - `trigger_type = SCHEDULED`
   - `crawl_mode`
   - `scheduled_for`
   - `payload` snapshot input scheduler
3. Lifecycle job tối thiểu:
   - `PENDING` khi vừa tạo
   - `RUNNING` khi bắt đầu publish
   - `SUCCESS` khi task publish thành công
   - `FAILED` khi build/publish fail

**Done khi**

- scheduler có audit trail đầy đủ qua `scheduled_jobs`

### Workstream 4: RabbitMQ publisher adapter

**Mục tiêu**

- convert grouped target thành request payload đúng contract scapper

**Việc cần làm**

1. Tạo module `internal/crawler` hoặc tương đương gồm:
   - queue resolver theo platform
   - action resolver theo `(source_type, target_type)`
   - payload builders
   - publish client
2. Chốt mapping tối thiểu:
- TikTok + `KEYWORD` -> `tiktok_tasks/full_flow`, mỗi keyword trong `target.values[]` sinh 1 `external_task` với `params.keyword`
   - TikTok + `POST_URL` -> `tiktok_tasks/post_detail` với `urls[] = target.values`
   - các mapping khác:
     - enable nếu canonical contract rõ
     - ngược lại fail fast với lỗi unsupported
3. Check `platform_action_flags` trước khi build/publish:
   - nếu `is_enabled = false` -> trả `unsupported platform-action mapping`
   - nếu không tìm thấy row -> cũng trả `unsupported platform-action mapping`
4. Phase 3 seed mặc định:
   - tạo row cho mọi mapping đã biết
   - set tất cả `is_enabled = true`
5. Request payload lưu vào `external_tasks.request_payload`
6. Sinh `task_id` UUID mới cho mỗi publish attempt
7. Khi publish success:
   - create/update `external_task`
   - set `published_at`
   - set status phù hợp
8. Khi publish fail:
   - mark `scheduled_job = FAILED`
   - update target/source `last_error_*`

**Done khi**

- scheduler có thể publish task thật sang RabbitMQ với payload đúng contract canonical

### Workstream 5: Consumer cho crawler result envelope

**Mục tiêu**

- nhận kết quả từ crawler và lưu raw cho phase sau

**Việc cần làm**

1. Tạo consumer handler thật cho result envelope:
   - parse JSON envelope
   - validate `task_id`, `status`, `completed_at`
2. Lookup `external_task` theo `task_id`
3. Nếu không tìm thấy:
   - log warning
   - DLQ hoặc skip theo config
4. Nếu duplicate response:
   - không tạo `raw_batch` trùng
   - không overwrite trạng thái thành công/lỗi một cách bừa bãi
5. Nếu response thành công:
   - upload full envelope/result lên MinIO
   - create `raw_batch`
   - update `external_task.status = SUCCESS`
   - set `response_received_at`, `completed_at`
   - update target/source `last_success_at`
6. Nếu response lỗi:
   - update `external_task.status = FAILED`
   - lưu `error_message`
   - update target/source `last_error_at`, `last_error_message`

**Done khi**

- có raw batch thật được tạo từ response crawler

### Workstream 6: Raw batch creation policy

**Mục tiêu**

- raw batch đủ thông tin cho parser phase sau

**Việc cần làm**

1. Batch ID policy:
   - ưu tiên `task_id` hoặc `result.metadata.batch_id` nếu crawler trả về
2. Lưu tối thiểu:
   - `source_id`
   - `project_id`
   - `external_task_id`
   - `batch_id`
   - `storage_bucket`
   - `storage_path`
   - `status = RECEIVED`
   - `publish_status = PENDING`
   - `checksum`
   - `item_count` nếu suy ra được
   - `raw_metadata`
3. Không parse ngay trong Phase 3
4. Nếu create raw batch fail sau khi đã nhận response:
   - giữ `external_task` ở trạng thái fail hoặc partial fail rõ ràng
   - không silent drop

**Done khi**

- raw batch record đủ sạch để parser phase sau chỉ việc consume

### Workstream 7: Manual/internal trigger

**Mục tiêu**

- cho phép bắn task thủ công để debug hoặc emergency run

**Việc cần làm**

1. Internal endpoint hoặc admin path tối thiểu:
   - `POST /api/v1/ingest/datasources/:id/trigger`
   - body có thể chứa `target_id`
2. Reuse chung publisher pipeline với scheduler
3. Tạo `scheduled_job` với `trigger_type = MANUAL` hoặc cho phép đi thẳng `external_task`
4. Không nhân bản logic build payload/publish

**Done khi**

- manual trigger dùng chung execution path với scheduler

### Workstream 8: Admin support-flag management

**Mục tiêu**

- admin bật/tắt từng platform-action mapping mà không cần redeploy

**Việc cần làm**

1. Tạo admin-only handlers:
   - list flags
   - update 1 flag
2. Authz:
   - chỉ admin/internal admin mới được gọi
3. Validate:
   - chỉ cho update mapping có trong support matrix
   - không cho tạo key rác ngoài matrix
4. Audit:
   - lưu `reason`
   - lưu `updated_by`

**Done khi**

- admin đổi flag qua endpoint và scheduler/manual trigger phản ánh ngay ở tick/lần gọi sau

### Workstream 9: Observability và operational guardrails

**Mục tiêu**

- runtime crawl phải debug được

**Việc cần làm**

1. Log correlation:
   - `source_id`
   - `target_id`
   - `scheduled_job_id`
   - `external_task_id`
   - `task_id`
2. Metric/log counters:
   - targets claimed
   - jobs created
   - publish success/fail
   - duplicate responses
   - raw batches created
3. Discord/reporting cho panic và fail nghiêm trọng
4. Config timeout / retry / batch limit

**Done khi**

- sự cố publish/consume có trace id đủ để lần ra gốc

## 7. File/Module dự kiến tác động

### Tạo mới hoặc mở rộng mạnh

- `internal/scheduler/**`
- `internal/crawler/**`
- `internal/consumer/**`
- `internal/scheduledjob/**` hoặc module tương đương
- `internal/externaltask/**` hoặc module tương đương
- `internal/rawbatch/**` hoặc module tương đương

### Mở rộng module hiện có

- `internal/datasource/repository/**`
- `internal/datasource/usecase/**`
- `internal/model/**`
- `internal/httpserver/**`

## 8. Acceptance / Verification

Không bắt buộc unit test ở scope hiện tại nếu direction chưa đổi, nhưng Phase 3 tối thiểu phải verify được bằng manual/API/runtime check:

1. Có datasource `ACTIVE` với grouped target đến hạn
2. Scheduler tick claim đúng target
3. Tạo `scheduled_job` thành công
4. Tạo đủ `external_tasks` theo grouped target và publish RabbitMQ thành công
5. Consumer nhận response envelope theo `task_id`
6. Tạo `raw_batch` đúng source/target/task
7. Duplicate response cùng `task_id` không tạo `raw_batch` thứ hai
8. Target/source success timestamps được update khi crawl thành công
9. Target/source error timestamps + message được update khi crawl fail

## 9. Không làm trong Phase 3

- parse raw sang UAP
- publish Kafka cho analysis
- replay parser pipeline hoàn chỉnh
- project lifecycle Kafka consumer end-to-end
- webhook runtime receiver
- full adaptive interval automation beyond scheduler reading `crawl_mode`

Các phần trên là Phase 4 hoặc Phase 5.

## 10. Rủi ro và blocker phải chốt sớm

### 10.1 Scapper contract chưa đồng đều theo mọi platform-target combination

Hiện canonical RabbitMQ rõ nhất ở:

- TikTok `full_flow` với `keyword` và grouped-keyword fan-out
- TikTok `post_detail` với `urls[]`
- Facebook `post_detail` với `parse_ids[]`

Chưa đồng đều cho:

- grouped `PROFILE`
- keyword batch ở một số platform khác
- YouTube video batch detail

=> Phase 3 phải có bảng support matrix và fail-fast cho combination chưa support.

### 10.2 Consumer result transport chưa được chốt end-to-end

Plan này giả định ingest sẽ nhận được response envelope có:

- `task_id`
- `platform`
- `action`
- `status`
- `result` hoặc `error`
- `completed_at`

Nếu transport thực tế khác, phải khóa lại contract trước khi code consumer thật.

### 10.3 Idempotency index có thể cần migration bổ sung

Nếu chưa có unique/index phù hợp cho:

- `external_tasks.task_id`
- `raw_batches (source_id, batch_id)`

thì nên bổ sung migration đầu Phase 3.

## 11. Thứ tự hiện thực khuyến nghị

1. Repository + migration/index cho `scheduled_jobs` / `external_tasks` / `raw_batches`
2. Due-target query + claim algorithm trong scheduler
3. `scheduled_job` creation
4. RabbitMQ publisher adapter + support matrix
5. `external_task` persistence
6. Consumer result handler
7. MinIO upload + `raw_batch` creation
8. Admin support-flag management
9. Manual/internal trigger
10. Observability + hardening

Thứ tự này giúp tách rõ:

- dispatch plane
- message transport
- response persistence

và giảm thời gian debug khi tích hợp với `scapper-srv`.

## 12. Definition of Done cho Phase 3

Phase 3 được coi là xong khi:

- scheduler heartbeat không còn chỉ log, mà dispatch được grouped targets đến hạn
- mỗi dispatch tạo đủ `scheduled_job` và `external_tasks` theo contract fan-out
- task được publish RabbitMQ theo contract canonical
- consumer nhận response theo `task_id`
- raw response được lưu MinIO và tạo `raw_batch`
- duplicate response không tạo record trùng
- parser/UAP vẫn chưa cần hoàn tất, nhưng raw handoff sang Phase 4 đã rõ ràng
