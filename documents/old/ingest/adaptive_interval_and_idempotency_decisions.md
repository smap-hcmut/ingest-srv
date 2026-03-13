# Quyết định bổ sung: Adaptive Interval và Idempotency

**Phiên bản:** 1.0  
**Ngày cập nhật:** 03/03/2026

---

## 1. Mục tiêu

Tài liệu này chốt 2 điểm còn mơ hồ trong thiết kế ingest:

- runtime policy cho adaptive crawl interval
- idempotency rules cho RabbitMQ, webhook và replay

Phạm vi tài liệu này chỉ nhằm giúp implementer không phải tự quyết định thêm trong lúc code.

---

## 2. Adaptive Interval: Chọn `multiplier`, không dùng `absolute mode interval`

## 2.1 Hai cách tiếp cận

### Cách A: `absolute mode interval`

Mỗi mode có một interval cố định:

- `NORMAL = 11 phút`
- `CRISIS = 2 phút`
- `SLEEP = 60 phút`

Ví dụ:

- target A có interval gốc `10 phút`
- target B có interval gốc `30 phút`

Khi source chuyển sang `CRISIS`:

- A -> `2 phút`
- B -> `2 phút`

### Cách B: `multiplier`

Mỗi target có interval gốc riêng. Mode chỉ nhân hệ số lên interval đó:

- `NORMAL = 1.0`
- `CRISIS = 0.2`
- `SLEEP = 5.0`

Ví dụ:

- target A = `10 phút`
- target B = `30 phút`

Khi source chuyển sang `CRISIS`:

- A -> `10 x 0.2 = 2 phút`
- B -> `30 x 0.2 = 6 phút`

---

## 2.2 Quyết định chốt

Chọn **`multiplier`** làm runtime policy chính.

### Lý do

- Hệ thống đã tách `crawl_targets`, nên mỗi target cần giữ interval base riêng.
- Nếu dùng `absolute mode interval`, lợi ích của per-target scheduling giảm đi nhiều.
- `multiplier` phù hợp hơn với requirement:
  - keyword crawl nhanh hơn profile
  - profile crawl nhanh hơn post URL hoặc ngược lại
  - khi vào `CRISIS`, tất cả cùng tăng tốc nhưng vẫn giữ tương quan ưu tiên ban đầu

---

## 2.3 Rule runtime

### Effective interval

```text
effective_interval = crawl_target.crawl_interval_minutes x mode_multiplier(data_source.crawl_mode)
```

### Mode multiplier mặc định

- `NORMAL = 1.0`
- `CRISIS = 0.2`
- `SLEEP = 5.0`

### Ý nghĩa của `crawl_mode_defaults`

Bảng `crawl_mode_defaults` không còn là nơi quyết định interval runtime chính.

Nó chỉ dùng cho:

- seed default hệ thống
- fallback khi tạo datasource mới hoặc target mới
- reset về mặc định nếu chưa có interval riêng

Không dùng `crawl_mode_defaults` để ghi đè toàn bộ interval runtime của mọi target khi source đổi mode.

---

## 2.4 Trade-off so với `absolute mode interval`

### `multiplier` - Ưu điểm

- giữ được per-target frequency
- phù hợp với thiết kế `crawl_targets`
- linh hoạt hơn khi scale nhiều loại target

### `multiplier` - Nhược điểm

- khó explain hơn cho non-technical user
- cần quy tắc làm tròn khi tính interval
- cần min/max guard để tránh interval quá nhỏ hoặc quá lớn

### `absolute mode interval` - Khi nào hợp lý hơn

Chỉ nên dùng nếu:

- hệ thống không có `crawl_targets`
- toàn bộ source luôn crawl cùng một nhịp
- business ưu tiên đơn giản hơn là granular control

---

## 2.5 Guard rails đề xuất

Để implement an toàn, nên có thêm rule:

- làm tròn xuống hoặc lên theo phút nguyên
- clamp interval tối thiểu, ví dụ `>= 1 phút`
- clamp interval tối đa, ví dụ `<= 1440 phút`

Ví dụ:

- target interval `3 phút`, mode `CRISIS = 0.2`
- kết quả toán học = `0.6 phút`
- runtime phải clamp thành `1 phút`

---

## 3. Idempotency Rules

## 3.1 Tại sao cần

Ingest làm việc với message queue và webhook nên duplicate delivery là bình thường:

- RabbitMQ có thể redeliver message
- crawler có thể gửi response trùng
- webhook provider có thể retry cùng payload
- user hoặc internal API có thể bấm replay nhiều lần

Nếu không có idempotency rules, hệ thống có thể:

- tạo `external_tasks` trùng
- tạo `raw_batches` trùng
- publish UAP trùng sang Analysis
- gây nhiễu analytics và khó audit

---

## 3.2 RabbitMQ idempotency

### Idempotency key chính

- `external_tasks.task_id`

### Quyết định chốt

- `task_id` là correlation id duy nhất với crawler bên thứ 3
- nếu nhận response trùng cho cùng `task_id`, ingest không được tạo `external_task` mới
- nếu response trùng dẫn đến cùng `batch_id`, ingest không được tạo `raw_batch` mới

### Rule xử lý

1. Tìm `external_task` theo `task_id`
2. Nếu không có task -> reject hoặc log error bất thường
3. Nếu đã có task và response đến lại:
   - không insert task mới
   - chỉ merge/update an toàn nếu cần
4. Nếu `raw_batch` của response đó đã tồn tại -> bỏ qua create mới

---

## 3.3 Raw batch idempotency

### Idempotency key ưu tiên

- `(source_id, batch_id)`

### Fallback key

- `checksum`

### Quyết định chốt

- `UNIQUE (source_id, batch_id)` là dedup key chính
- nếu provider không trả `batch_id` đủ tin cậy, dùng `checksum` để hỗ trợ detect duplicate

### Rule xử lý

- cùng `source_id` + `batch_id` -> coi là cùng một raw batch
- không parse/publish lại như batch mới nếu chỉ là duplicate delivery

---

## 3.4 Webhook idempotency

### Idempotency key ưu tiên

- `webhook_id + external_event_id`

### Fallback key

- `webhook_id + checksum(payload)`

### Quyết định chốt

- nếu webhook payload có `external_event_id`, dùng nó làm dedup key
- nếu không có, tính checksum trên payload chuẩn hóa
- duplicate webhook không được tạo `raw_batch` mới

### Rule xử lý

1. Verify signature
2. Tính idempotency key
3. Check đã nhận payload này chưa
4. Nếu đã nhận -> trả success idempotent, không xử lý lại như dữ liệu mới

---

## 3.5 Replay idempotency

### Câu hỏi chính

Replay có phải tạo dữ liệu mới hay chỉ chạy lại pipeline trên dữ liệu cũ?

### Quyết định chốt

- replay không tạo `raw_batch` mới
- replay chỉ re-run parse/publish trên batch đã tồn tại
- replay mặc định chỉ cho phép khi:
  - `raw_batches.status = FAILED`
  - hoặc `raw_batches.publish_status = FAILED`

### Force replay

Nếu muốn replay cả batch đã `SUCCESS`:

- phải có cờ `force = true`
- phải được coi là thao tác đặc biệt của internal/admin flow

### Rule xử lý

- replay thường -> chỉ cho batch lỗi
- replay force -> cho batch thành công nhưng phải log audit rõ đây là replay cưỡng bức

---

## 3.6 UAP publish idempotency

### Quyết định chốt

V1 chưa giải quyết exactly-once end-to-end.

Cách tiếp cận pragmatic:

- cố gắng không tạo duplicate từ phía ingest bằng `task_id`, `batch_id`, `checksum`
- nếu replay force được gọi thì duplicate publish có thể xảy ra và phải được coi là hành vi có chủ đích

### Ghi chú

Nếu downstream Analysis cần chống duplicate mạnh hơn, nên bổ sung thêm:

- `event_id` deterministic
- hoặc `trace.raw_ref + record position` làm natural dedup key

Điểm này có thể đưa sang V2.

---

## 4. Tóm tắt quyết định

## 4.1 Adaptive interval

Chốt:

- dùng `multiplier`
- không dùng `absolute mode interval` làm runtime policy chính
- `crawl_mode_defaults` chỉ là default/fallback config

## 4.2 Idempotency

Chốt:

- RabbitMQ: key chính = `task_id`
- Raw batch: key chính = `(source_id, batch_id)`
- Webhook: key chính = `webhook_id + external_event_id`, fallback = checksum
- Replay: chỉ replay batch lỗi, `SUCCESS` cần `force = true`

---

## 5. Khuyến nghị cập nhật doc chính

Sau tài liệu này, nên đồng bộ lại vào:

- `ingest_project_schema_alignment_proposal.md`
- `ingest_plan.md`

Các điểm nên update:

1. Ghi rõ `multiplier` là runtime policy canonical
2. Ghi rõ `crawl_mode_defaults` chỉ là default/fallback
3. Thêm mục idempotency rules cho RabbitMQ/webhook/replay
4. Thêm guard rails cho effective interval và replay force
