# Kế hoạch triển khai tiếp theo cho ingest-srv

**Ngày cập nhật:** 2026-03-06  
**Phiên bản:** v1.1  
**Phạm vi:** `ingest-srv/internal`  
**Nguyên tắc đánh giá:** dựa trên code hiện tại trong `internal`, không dựa vào giả định từ tài liệu cũ.
**Ghi nhớ:** bỏ qua các yêu cầu viết unit test, chỉ hiện thực code cho các phase dưới

> **Lưu ý:** file này phản ánh đánh giá trước khi grouped-target refactor và Phase 2 control-plane được hiện thực xong. Contract hiện tại dùng `values[]` và 3 create endpoints `keywords|profiles|posts`; không dùng file này làm API reference canon.
## 1. Kết luận ngắn

Hiện tại `ingest-srv` đã có nền tảng khá tốt cho domain `data_sources` và `crawl_targets`:
- đã có schema và sqlboiler cho các bảng ingest chính
- đã có model domain tương ứng
- đã có repository/usecase cho CRUD `data_sources`
- đã có repository/usecase cho CRUD `crawl_targets`
- HTTP route cho datasource đã được mount vào runtime server

Tuy nhiên service vẫn chưa ở trạng thái sẵn sàng để tích hợp ngay với `project-srv` và `scapper-srv`, vì còn thiếu các phần quan trọng:
- scheduler/runtime orchestration chưa có
- consumer vẫn là no-op
- chưa có test cho datasource module
- chưa có flow runtime: `dryrun -> scheduled_jobs -> external_tasks -> raw_batches -> publish UAP`

## 2. Mục tiêu triển khai tiếp theo

Mục tiêu ngắn hạn không phải là tích hợp liên service ngay, mà là:
- đóng vòng module `datasource + crawl_target`
- chuẩn hóa boundary API
- hoàn thiện business rule cốt lõi
- có test đủ tin cậy
- sau đó mới mở rộng sang runtime orchestration và integration

---

## 3. Phase 1: Đóng vòng CRUD cho chuẩn

Đây là phase ưu tiên cao nhất. Chưa xong phase này thì chưa nên chuyển sang tích hợp với service khác.

### 3.1 Mục tiêu
- hoàn thiện public CRUD cho `data_sources`
- hoàn thiện public CRUD cho `crawl_targets`
- chốt request/response/error handling đúng với runtime thật
- chặn các lỗi ownership và validation ngay tại boundary

### 3.2 Trạng thái đầu vào của code hiện tại

Phase 1 không bắt đầu từ số 0. Những phần đã có sẵn:

- HTTP server đã mount datasource routes vào `/api/v1`
- `data_sources` đã có create/list/detail/update/archive
- `crawl_targets` đã có create/list/update/delete
- usecase đã có một phần business rule quan trọng:
  - validate `source_type`
  - infer `source_category`
  - source `CRAWL` bắt buộc có `crawl_mode` + `crawl_interval_minutes`
  - chặn update `config/mapping_rules` khi source đang `ACTIVE`
  - target chỉ được tạo dưới source `CRAWL`

Những gap Phase 1 trong snapshot này hiện đã được xử lý ở code:

- đã có `GET /datasources/:id/targets/:target_id`
- ownership check target theo `(data_source_id, target_id)` đã được khóa
- validation ở delivery layer đã được siết thêm
- swagger/runtime path đã dùng `datasources`
- `mapError` không còn `panic` ở nhánh default

Gap còn lại chủ yếu là:

- chưa có test cho module datasource

### 3.3 Kế hoạch hiện thực chi tiết cho Phase 1

#### Workstream 1: Hoàn thiện surface API của `crawl_targets`

**Mục tiêu:**
- đóng đủ bộ CRUD public cho `crawl_targets`

**Việc cần làm:**
1. Bổ sung endpoint detail target:
- `GET /api/v1/datasources/:id/targets/:target_id`

2. Đồng bộ handler + presenter + request processor cho target detail

3. Đảm bảo response target nhất quán với các endpoint target còn lại

**File chính sẽ tác động:**
- `internal/datasource/delivery/http/routes.go`
- `internal/datasource/delivery/http/handlers.go`
- `internal/datasource/delivery/http/process_request.go`
- `internal/datasource/delivery/http/presenters.go`

**Done khi:**
- API target đủ `create/list/detail/update/delete`
- target detail trả đúng `target_id`, `data_source_id`, `target_type`, `values[]`, `interval`, `runtime timestamps`

#### Workstream 2: Khóa ownership check theo `data_source_id`

**Mục tiêu:**
- không cho thao tác target chéo datasource

**Việc cần làm:**
1. Tất cả luồng `detail/update/delete target` phải kiểm tra:
- `target_id` tồn tại
- `target.data_source_id == path.data_source_id`

2. Nếu không match:
- trả `not found` hoặc lỗi ownership tương đương
- không để lộ target của datasource khác

3. Quy tắc kiểm tra ownership phải nằm ở usecase hoặc repository query theo cặp:
- `(data_source_id, target_id)`

**File chính sẽ tác động:**
- `internal/datasource/types.go`
- `internal/datasource/interface.go`
- `internal/datasource/usecase/detail_target.go`
- `internal/datasource/usecase/update_target.go`
- `internal/datasource/usecase/delete_target.go`
- `internal/datasource/repository/interface.go`
- `internal/datasource/repository/option.go`
- `internal/datasource/repository/postgre/target.go`
- `internal/datasource/delivery/http/process_request.go`

**Done khi:**
- không thể dùng `target_id` hợp lệ của datasource A để đọc/sửa/xóa qua path datasource B

#### Workstream 3: Chuẩn hóa validation ở boundary HTTP

**Mục tiêu:**
- reject request sai càng sớm càng tốt

**Việc cần làm:**
1. Bổ sung validate cho request DTO:
- `createTargetReq`
- `updateTargetReq`
- `listTargetsReq` nếu có enum filter

2. Chuẩn hóa các rule:
- `target_type` phải thuộc `KEYWORD | PROFILE | POST_URL`
- `crawl_interval_minutes > 0` nếu user truyền lên
- payload JSON sai format phải fail rõ

3. Rà lại datasource request DTO:
- `source_type`
- `source_category`
- `crawl_mode`
- `crawl_interval_minutes`

**File chính sẽ tác động:**
- `internal/datasource/delivery/http/presenters.go`
- `internal/datasource/delivery/http/process_request.go`
- `internal/datasource/delivery/http/errors.go`

**Done khi:**
- request sai bị chặn ngay ở delivery layer với message/error code rõ ràng

#### Workstream 4: Chuẩn hóa error handling và swagger annotation

**Mục tiêu:**
- contract docs khớp runtime
- API không `panic` vì lỗi domain chưa map

**Việc cần làm:**
1. Sửa swagger annotation từ `/sources` sang `/datasources`

2. Rà lại toàn bộ annotation target endpoints cho đúng path runtime

3. Thay default branch của `mapError`:
- không `panic`
- trả internal error chuẩn

4. Kiểm tra lại mã lỗi HTTP cho datasource/target có nhất quán hay chưa

**File chính sẽ tác động:**
- `internal/datasource/delivery/http/handlers.go`
- `internal/datasource/delivery/http/errors.go`

**Done khi:**
- annotation, route runtime, docs examples dùng cùng một namespace `datasources`
- không còn `panic` do thiếu map error ở HTTP layer

<!-- #### Workstream 5: Hoàn thiện test cho module datasource

**Mục tiêu:**
- có safety net trước khi chuyển sang Phase 2

**Mức test tối thiểu cần có:**
1. Usecase tests
- create datasource success
- create crawl datasource thiếu crawl config -> fail
- update active source có đổi config/mapping -> fail
- create target dưới passive source -> fail
- update/delete/detail target sai datasource owner -> fail

2. Repository tests
- create/detail/list/update/archive datasource
- create/detail/list/update/delete target
- query ownership theo `(data_source_id, target_id)`

3. HTTP/API tests
- status code + body đúng cho datasource CRUD
- status code + body đúng cho target CRUD
- validation error đúng format

**File/dir dự kiến:**
- `internal/datasource/usecase/*_test.go`
- `internal/datasource/repository/postgre/*_test.go`
- `internal/datasource/delivery/http/*_test.go` -->

**Done khi:**
<!-- - module datasource có test cho happy path + guard path chính -->
- thay đổi ở Phase 2 không dễ làm vỡ CRUD nền

### 3.4 Thứ tự thực hiện trong Phase 1

Thứ tự nên làm:

1. Bổ sung target detail endpoint
2. Khóa ownership check cho target
3. Chuẩn hóa request validation
4. Sửa error mapping + swagger annotation
<!-- 5. Viết test usecase
6. Viết test repository
7. Viết test HTTP boundary -->

Lý do:
- target detail + ownership check là gap nghiệp vụ lớn nhất
- validation/error/doc drift nên xử lý trước khi viết test snapshot
<!-- - test để khóa behavior sau khi API surface đã ổn định -->

### 3.5 Ràng buộc business phải giữ nguyên trong Phase 1

- `data_source` tạo mới luôn ở `PENDING`
- source `CRAWL` bắt buộc có `crawl_mode` và `crawl_interval_minutes > 0`
- `source_category` có thể infer từ `source_type`
- source `ACTIVE` không cho sửa `config` và `mapping_rules`
- `crawl_target` chỉ được tạo dưới source `CRAWL`
- `crawl_target.crawl_interval_minutes` nếu không truyền thì kế thừa từ datasource hiện tại
- API path canonical là `datasources`, không dùng `sources`

### 3.6 Điều kiện hoàn tất phase 1
- `data_sources` CRUD chạy đủ qua API
- `crawl_targets` CRUD chạy đủ qua API
- `crawl_targets` có `detail` endpoint riêng
- target ownership theo datasource được kiểm tra đầy đủ
- route, swagger, response thực tế khớp nhau
- HTTP layer không còn `panic` ở nhánh map error mặc định
- có test cơ bản cho usecase/repository/API boundary

### 3.7 Deliverables của Phase 1

Sau khi kết thúc Phase 1, tối thiểu phải có:

1. Bộ endpoint public ổn định cho:
- `data_sources`
- `crawl_targets`

2. Bộ business rule CRUD đã khóa bằng test

3. Swagger/runtime path đồng nhất theo `datasources`

4. Một module datasource đủ chắc để làm nền cho:
- lifecycle API ở Phase 2
- dry run
- scheduler theo target
- integration với `project-srv` và `scapper-srv`

---

## 4. Phase 2: Bổ sung lifecycle API cho source

Sau khi CRUD cơ bản ổn định, phase tiếp theo là biến `data_source` từ một record CRUD thành một thực thể vận hành đúng nghĩa.

### 4.1 Mục tiêu
- đưa lifecycle source vào API và business flow
- chuẩn bị dữ liệu cho scheduler và project integration sau này

### 4.2 Việc cần làm
1. Bổ sung source lifecycle operations
- activate
- pause
- resume
- archive theo workflow rõ ràng

2. Bổ sung crawl mode operations
- update `crawl_mode`
- ghi log `crawl_mode_changes`
- áp dụng đúng default interval/multiplier theo mode

3. Bổ sung dry run API
- trigger dry run
- lấy latest result
- lấy history dry run

4. Chuẩn hóa state transition rule
- trạng thái nào được chuyển sang trạng thái nào
- khi nào reset `dryrun_status`
- khi nào source được phép activate

### 4.3 Điều kiện hoàn tất phase 2
- source có lifecycle rõ ràng qua API
- crawl mode có API và audit trail
- dry run có thể trigger và truy vấn kết quả

---

## 5. Phase 3: Runtime orchestration nội bộ

Đây là phase chuyển service từ CRUD/lifecycle sang ingest runtime thực tế.

### 5.1 Mục tiêu
- scheduler hoạt động theo `crawl_targets`
- sinh job/task/batch đúng chuỗi xử lý
- chuẩn bị nền để nối với `scapper-srv`

### 5.2 Việc cần làm
1. Hoàn thiện scheduler
- query các `crawl_targets` đến hạn
- áp mode hiện tại của source
- áp `crawl_mode_defaults`
- tạo `scheduled_jobs`
- cập nhật `next_crawl_at`

2. Tạo dispatcher/external task flow
- từ `scheduled_jobs` sinh `external_tasks`
- snapshot payload gửi ra ngoài
- chuẩn hóa correlation theo `task_id`

3. Hoàn thiện raw batch flow
- nhận kết quả crawl
- lưu raw file/payload
- tạo `raw_batches`
- parse raw thành data chuẩn
- publish UAP

4. Cập nhật trạng thái runtime
- `last_crawl_at`
- `last_success_at`
- `last_error_at`
- `last_error_message`
- trạng thái publish batch

### 5.3 Điều kiện hoàn tất phase 3
- scheduler không còn là heartbeat-only
- internal runtime tables được dùng thật
- có trace đầy đủ từ target -> job -> task -> batch

---

## 6. Phase 4: Tích hợp liên service

Chỉ nên làm phase này sau khi 3 phase trên đã ổn định.

### 6.1 Tích hợp với `scapper-srv`
1. Publish RabbitMQ theo contract chính thức
- queue/action/payload thống nhất
- correlation bằng `task_id`

2. Nhận và xử lý kết quả crawl
- idempotency
- retry/replay
- mapping raw result về `raw_batches`

3. Theo dõi lỗi tích hợp
- timeout
- duplicate message
- malformed payload
- partial failure

### 6.2 Tích hợp với `project-srv`
1. Nhận lifecycle command/event
- activate/pause/resume/archive project liên quan đến source

2. Đồng bộ crawl mode/adaptive mode
- crisis event từ project
- project event làm thay đổi chế độ crawl

3. Trả event/status ngược về project nếu cần
- source state changed
- dry run completed
- crawl completed
- crawl mode changed

### 6.3 Điều kiện hoàn tất phase 4
- RabbitMQ integration chạy thật với `scapper-srv`
- project lifecycle có thể điều khiển ingest flow
- event/API contract giữa các service khớp runtime

---

## 7. Thứ tự ưu tiên đề xuất

Thứ tự triển khai nên là:

1. Hoàn thiện CRUD `data_sources`
2. Hoàn thiện CRUD `crawl_targets`
3. Bổ sung ownership check + validation + test
4. Chuẩn hóa swagger/runtime/error handling
5. Bổ sung lifecycle API cho source
6. Bổ sung dry run
7. Hoàn thiện scheduler
8. Hoàn thiện external task + raw batch flow
9. Tích hợp `scapper-srv`
10. Tích hợp `project-srv`
11. Hoàn thiện crisis/adaptive automation

---

## 8. Kết luận triển khai

Từ trạng thái code hiện tại, bước đúng nhất tiếp theo là:
- **không nhảy ngay sang integration**
- **không mở rộng thêm nhiều domain mới**
- **tập trung khóa chặt module `datasource + crawl_target` trước**

Nếu làm đúng thứ tự này, `ingest-srv` sẽ có một nền đủ chắc để:
- mở rộng sang dry run
- chạy scheduler thật
- dispatch task sang `scapper-srv`
- và sau đó mới tích hợp với `project-srv` mà không bị vỡ boundary ngay từ đầu.
