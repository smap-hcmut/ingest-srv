# Kế hoạch chi tiết Phase 2 cho ingest-srv

**Ngày cập nhật:** 2026-03-06  
**Phiên bản:** v1.0  
**Phạm vi:** `ingest-srv/internal`  
**Tiền đề:** dựa trên code hiện tại, không giả định các module runtime Phase 3 đã sẵn sàng.

> **Trạng thái hiện tại:** phần lớn nội dung plan này đã được hiện thực trong code ngày 2026-03-06. Dry run hiện là control-plane only, `target_id` được hiểu là grouped target, và grouped-target create contract nằm ở `POST /datasources/:id/targets/keywords|profiles|posts`.

## 1. Kết luận ngắn

**Có thể bắt đầu Phase 2**, nhưng nên hiểu đúng là:

- có thể bắt đầu **thiết kế và hiện thực lifecycle API / crawl mode / dry run control plane**
- **không nên coi Phase 1 đã hoàn tất 100%**
- debt còn lại của Phase 1 hiện chủ yếu là **test coverage**, không còn là blocker kiến trúc cho Phase 2

Vì vậy, quyết định hợp lý là:

- bắt đầu Phase 2 ngay
- không mở rộng sang scheduler / RabbitMQ / parser runtime trong Phase 2
- giữ scope Phase 2 ở mức **control plane + state transition + persistence**

---

## 2. Mục tiêu của Phase 2

Biến `data_source` từ một record CRUD thành một thực thể có lifecycle rõ ràng trong hệ ingest.

Phase 2 phải giải quyết 3 nhóm chính:

1. lifecycle source
2. crawl mode update + audit trail
3. dry run API + persistence + readiness transition

---

## 3. Trạng thái hiện tại của code

### 3.1 Những gì đã sẵn sàng

- `datasource` CRUD public đã có
- `crawl_targets` CRUD public đã có
- boundary validation của CRUD đã được siết thêm
- model đã có đầy đủ enum lifecycle:
  - `PENDING`, `READY`, `ACTIVE`, `PAUSED`, `FAILED`, `COMPLETED`, `ARCHIVED`
- model đã có field phục vụ Phase 2:
  - `dryrun_status`
  - `dryrun_last_result_id`
  - `activated_at`
  - `paused_at`
  - `archived_at`
  - `crawl_mode`
- sqlboiler đã có bảng/model:
  - `dryrun_results`
  - `crawl_mode_changes`

### 3.2 Những gì còn thiếu

- chưa có lifecycle usecase cho `activate/pause/resume`
- chưa có repository cho `dryrun_results`
- chưa có repository cho `crawl_mode_changes`
- chưa có API dry run
- chưa có internal crawl mode endpoint thật
- chưa có state machine tập trung cho source lifecycle
- chưa có contract rõ ràng cho actor nào được activate source trong V1

---

## 4. Quyết định chốt trước khi hiện thực

Các quyết định dưới đây nên được coi là canonical cho Phase 2.

### 4.1 Activate không phải public user API

Để khớp với tài liệu ingest hiện tại:

- **không thêm public endpoint** để user tự `activate` source
- `activate` đi qua:
  - internal usecase
  - internal adapter / consumer từ `project-srv`
  - hoặc internal/admin handler nếu thật sự cần trong runtime, nhưng không public cho user-facing API

### 4.2 Archive là action riêng, delete là bước sau archive

Phase 1 hiện đã tách:

- `POST /api/v1/datasources/:id/archive` để chuyển datasource sang `ARCHIVED`
- `DELETE /api/v1/datasources/:id` chỉ để soft delete sau khi datasource đã `ARCHIVED`

Vì vậy trong Phase 2:

- giữ nguyên 2 route này
- chỉ chuẩn hóa state transition và side effect của archive/delete
- không nhập nhằng `DELETE` với `archive`

### 4.3 Dry run cho CRAWL đi theo target trước

Để tránh fan-out runtime quá sớm khi scheduler/external task chưa hoàn chỉnh:

- với source `CRAWL`, Phase 2 chỉ hỗ trợ dry run **theo 1 target**
- request dry run cho `CRAWL` phải có `target_id`
- dry run toàn source cho crawl sẽ để lại cho Phase 3 nếu cần

### 4.4 Crawl mode chỉ áp dụng cho source `CRAWL`

- `FILE_UPLOAD` và `WEBHOOK` không nhận `crawl_mode`
- đổi mode phải ghi `crawl_mode_changes`
- Phase 2 chỉ persist mode + audit trail
- việc áp multiplier vào scheduler runtime sẽ được làm thật ở Phase 3

### 4.5 READY là cổng vào duy nhất để ACTIVE

Nguồn chỉ được activate khi đang ở `READY`.

Phase 2 không cho phép:

- `PENDING -> ACTIVE`
- `FAILED -> ACTIVE`
- `ARCHIVED -> ACTIVE`
- `COMPLETED -> ACTIVE`

---

## 5. State machine cần chốt

### 5.1 Source lifecycle

| From | To | Điều kiện |
|---|---|---|
| `PENDING` | `READY` | dry run pass; implementation Phase 2 hiện trả `WARNING` |
| `READY` | `ACTIVE` | internal/project activation hợp lệ |
| `ACTIVE` | `PAUSED` | internal/project pause |
| `PAUSED` | `ACTIVE` | internal/project resume |
| `PENDING` | `FAILED` | dry run thất bại nghiêm trọng |
| `READY` | `FAILED` | dry run lại thất bại hoặc validation runtime fail |
| `FAILED` | `READY` | dry run lại thành công |
| `PENDING`,`READY`,`PAUSED`,`FAILED` | `ARCHIVED` | archive datasource |

Không hỗ trợ trong Phase 2:

- `ACTIVE -> READY`
- `ARCHIVED -> *`
- `COMPLETED -> ACTIVE/PAUSED`

### 5.2 Rule theo loại source

#### `CRAWL`

- cần `crawl_mode`
- cần `crawl_interval_minutes > 0`
- cần ít nhất 1 `crawl_target` active trước khi vào `READY` hoặc `ACTIVE`
- dry run theo `target_id`

#### `WEBHOOK`

- cần `webhook_id` + `webhook_secret_encrypted` trước khi `READY/ACTIVE`
- không có `crawl_mode`
- dry run không gắn `target_id`

#### `FILE_UPLOAD`

- không tham gia `ACTIVE/PAUSED` theo flow dài hạn như crawl
- Phase 2 chỉ cần không phá vỡ lifecycle hiện tại
- one-shot flow chi tiết giữ cho phase sau

---

## 6. API surface đề xuất cho Phase 2

## 6.1 Internal lifecycle entrypoints

Phase 2 nên tách rõ **public** và **internal**.

### Internal / orchestration-facing

- `PUT /api/v1/ingest/datasources/:id/crawl-mode`
- lifecycle `activate/pause/resume` đi qua usecase nội bộ trước; HTTP adapter cho các action này là tùy chọn nếu cần để debug/admin

Nếu thêm internal HTTP handlers để thao tác lifecycle, nên dùng namespace:

- `POST /api/v1/internal/datasources/:id/activate`
- `POST /api/v1/internal/datasources/:id/pause`
- `POST /api/v1/internal/datasources/:id/resume`

Các route trên là **internal-only**, không mount vào user-facing group.

### User-facing

- `POST /api/v1/datasources/:id/dryrun`
- `GET /api/v1/datasources/:id/dryrun/latest`
- `GET /api/v1/datasources/:id/dryrun/history`

## 6.2 Request/response tối thiểu

### `POST /datasources/:id/dryrun`

Request body đề xuất:

```json
{
  "target_id": "optional-for-passive-required-for-crawl",
  "sample_limit": 10,
  "force": false
}
```

Rule:

- `target_id` bắt buộc với source `CRAWL`
- `target_id` không được truyền với source `PASSIVE`
- `sample_limit` optional, phải `> 0` nếu truyền
- `force` chỉ dùng để cho phép re-run khi source đang có dry run gần nhất thành công nhưng user muốn kiểm tra lại

Response tối thiểu:

- trả `dryrun_result`
- kèm `data_source` snapshot rút gọn nếu cần UI refresh state

### `GET /datasources/:id/dryrun/latest`

Query params:

- `target_id` optional

Rule:

- nếu source là `CRAWL`, `target_id` nên cho phép để lấy latest theo target
- nếu không truyền `target_id`, ưu tiên latest theo source tổng quát

### `GET /datasources/:id/dryrun/history`

Query params:

- `target_id` optional
- `page`
- `limit`

Response:

- danh sách `dryrun_results`
- paginator chuẩn

### `PUT /ingest/datasources/:id/crawl-mode`

Request body đề xuất:

```json
{
  "crawl_mode": "NORMAL",
  "reason": "manual reset after crisis",
  "trigger_type": "MANUAL",
  "event_ref": "optional-correlation-id"
}
```

Rule:

- chỉ chấp nhận source `CRAWL`
- `crawl_mode` phải thuộc `SLEEP|NORMAL|CRISIS`
- `trigger_type` phải map vào `model.TriggerType`
- phải persist `crawl_mode_changes`

---

## 7. Workstreams hiện thực

## 7.1 Workstream A: Chốt state machine trong datasource domain

**Mục tiêu:**

- có một nơi duy nhất quyết định source được chuyển trạng thái nào

**Việc cần làm:**

1. Mở rộng `internal/datasource/interface.go`
- thêm usecase methods cho:
  - `Activate`
  - `Pause`
  - `Resume`
  - `UpdateCrawlMode`

2. Bổ sung input/output types trong `internal/datasource/types.go`

3. Tạo helper/state guard trong usecase:
- validate transition
- validate precondition theo source type
- validate source category tương ứng

4. Giữ `Archive` hiện có nhưng đưa vào cùng hệ rule transition

**File chính sẽ tác động:**

- `internal/datasource/interface.go`
- `internal/datasource/types.go`
- `internal/datasource/errors.go`
- `internal/datasource/usecase/*.go`

**Done khi:**

- mọi transition lifecycle đều đi qua cùng một bộ rule
- không còn logic lifecycle rải rác ở handler/repository

## 7.2 Workstream B: Repository cho lifecycle + crawl mode audit

**Mục tiêu:**

- đủ data access để persist lifecycle và audit crawl mode

**Việc cần làm:**

1. Mở rộng datasource repository:
- update status
- update dryrun fields
- read source cùng targets khi cần precondition check

2. Tạo repository cho `crawl_mode_changes`
- create audit record
- list/history nếu cần debug nội bộ

3. Nếu thiếu, bổ sung query helper:
- load source + active target count
- load source + ownership context

**File chính sẽ tác động:**

- `internal/datasource/repository/interface.go`
- `internal/datasource/repository/option.go`
- `internal/datasource/repository/postgre/*.go`

**Done khi:**

- đổi mode luôn tạo được audit log
- lifecycle usecase không cần tự chạm sqlboiler trực tiếp

## 7.3 Workstream C: Internal lifecycle adapters

**Mục tiêu:**

- có entrypoint rõ cho system/project trigger lifecycle

**Việc cần làm:**

1. Giữ user-facing router sạch, không public activation

2. Implement internal crawl mode endpoint thật:
- request DTO
- response DTO
- error mapping

3. Nếu cần để unblock tích hợp sớm, thêm internal handlers cho:
- activate
- pause
- resume

4. Nếu chưa mount internal lifecycle routes ở HTTP phase này, vẫn phải chốt service interface để consumer phase sau dùng ngay

**File chính sẽ tác động:**

- `internal/httpserver/handler.go`
- `internal/datasource/delivery/http/*.go`
- hoặc adapter nội bộ mới nếu tách namespace riêng

**Done khi:**

- internal/system actor có thể đổi mode
- lifecycle có entrypoint rõ ràng, không phải gọi chéo handler CRUD

## 7.4 Workstream D: Dry run module foundation

**Mục tiêu:**

- tạo control plane cho dry run mà chưa kéo toàn bộ runtime Phase 3 vào

**Việc cần làm:**

1. Tạo module `internal/dryrun`
- `interface.go`
- `types.go`
- `errors.go`
- `repository/*`
- `usecase/*`
- `delivery/http/*`

2. Repository cho `dryrun_results`
- create
- update status/result
- detail by id
- latest by source/target
- history by source/target

3. Dry run usecase:
- validate source existence
- validate source type/category
- validate `target_id` rule
- tạo vòng đời `PENDING -> RUNNING -> SUCCESS/WARNING/FAILED`
- update `data_sources.dryrun_status`
- update `data_sources.dryrun_last_result_id`

4. Tạo executor abstraction:
- `Executor` interface
- phase đầu có thể dùng implementation tối thiểu theo source type
- không hardcode RabbitMQ/scheduler vào usecase

**File chính sẽ tác động:**

- `internal/dryrun/**`
- `internal/datasource/repository/interface.go`
- `internal/datasource/repository/postgre/*.go`

**Done khi:**

- có thể trigger dry run và query latest/history
- source state được cập nhật đúng theo kết quả dry run

## 7.5 Workstream E: Delivery contract cho dry run

**Mục tiêu:**

- API dry run có contract rõ, không phải “stub route”

**Việc cần làm:**

1. Mount route dry run vào `/api/v1/datasources/:id/...`

2. Thêm request validation tại delivery layer:
- `target_id`
- `sample_limit`
- `force`

3. Thiết kế response:
- `dryrun_result`
- metadata/paginator

4. Chuẩn hóa error mapping:
- source not found
- target not found
- invalid target/source combination
- dry run not allowed in current state

**Done khi:**

- dry run API có thể dùng độc lập với UI/internal caller
- route, swagger và response khớp nhau

## 7.6 Workstream F: Readiness và lifecycle side effects

**Mục tiêu:**

- mọi state transition cập nhật field liên quan đồng bộ

**Việc cần làm:**

1. Khi dry run pass (`WARNING` trong implementation Phase 2 hiện tại)
- source -> `READY`
- set `dryrun_status`
- set `dryrun_last_result_id`

2. Khi dry run `FAILED`
- source giữ `PENDING` hoặc chuyển `FAILED` theo rule đã chốt

3. Khi `Activate`
- source -> `ACTIVE`
- set `activated_at`

4. Khi `Pause`
- source -> `PAUSED`
- set `paused_at`

5. Khi `Resume`
- source -> `ACTIVE`
- clear/giữ `paused_at` theo policy thống nhất

6. Khi `UpdateCrawlMode`
- update `data_sources.crawl_mode`
- append `crawl_mode_changes`

**Done khi:**

- API response và DB state không lệch nhau sau transition

---

## 8. Thứ tự hiện thực đề xuất

1. Chốt state machine + domain errors
2. Mở rộng datasource repository cho lifecycle
3. Implement `UpdateCrawlMode` + audit trail
4. Mount internal crawl mode endpoint
5. Tạo `dryrun` repository + usecase
6. Mount dry run APIs
7. Gắn readiness transition giữa dry run và datasource status
8. Sau cùng mới cân nhắc internal activate/pause/resume HTTP adapters nếu thực sự cần

Lý do:

- crawl mode ít phụ thuộc nhất và giúp khóa contract nội bộ sớm
- dry run cần state machine rõ trước khi viết
- activation path không nên public hóa vội nếu chưa chốt orchestration entrypoint

---

## 9. Ràng buộc nghiệp vụ phải giữ trong Phase 2

- namespace chuẩn vẫn là `datasources`
- user không tự activate source qua public API trong V1
- `READY` là điều kiện vào `ACTIVE`
- `crawl_mode` chỉ áp dụng cho source `CRAWL`
- đổi `crawl_mode` phải ghi `crawl_mode_changes`
- dry run của `CRAWL` đi theo `target_id`
- dry run result phải cập nhật `dryrun_status` và `dryrun_last_result_id`
- không trộn scheduler/external task runtime thật vào Phase 2 nếu chưa có abstraction rõ

---

## 10. Phạm vi không làm trong Phase 2

- scheduler pick target đến hạn
- tạo `scheduled_jobs`
- tạo `external_tasks`
- publish RabbitMQ thật
- consume RabbitMQ result thật
- parse raw -> publish UAP
- project lifecycle Kafka consumer end-to-end

Các phần trên thuộc Phase 3 hoặc Phase 4.

---

## 11. Điều kiện hoàn tất Phase 2

- source có state machine rõ ràng ở usecase layer
- có lifecycle operations tối thiểu cho `activate/pause/resume` ở service boundary phù hợp
- internal crawl mode API chạy thật
- đổi crawl mode tạo được audit record ở `crawl_mode_changes`
- dry run có thể:
  - trigger
  - lưu kết quả
  - lấy latest
  - lấy history
- dry run cập nhật đúng `READY/PENDING/FAILED` và `dryrun_last_result_id`
- không làm rò public API vượt quá canonical rules của ingest docs

---

## 12. Khuyến nghị trước khi code

Trước khi hiện thực, nên coi plan này là baseline và chốt thêm 2 điểm để tránh sửa nửa chừng:

1. có hay không mount internal HTTP cho `activate/pause/resume` ngay trong Phase 2  
2. dry run `FAILED` sẽ giữ source ở `PENDING` hay chuyển hẳn `FAILED` trong version đầu

Nếu chưa chốt hai điểm trên, phần còn lại của plan vẫn đủ ổn để bắt đầu từ:

- `UpdateCrawlMode`
- `crawl_mode_changes` repository
- `dryrun_results` repository
