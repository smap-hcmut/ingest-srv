# Change: Remove Throttling & Simplify Progress Tracking

## Why

Throttling được thiết kế cho flow "mỗi item crawl xong → report 1 lần", nhưng flow thực tế là:

- Worker (YouTube/TikTok) crawl **tất cả items** rồi mới report **1 lần** khi hoàn thành
- Số lần gọi webhook = số platforms (2-3 lần max per project)
- Throttling không cần thiết và làm phức tạp code

## Architecture Issue: cmd vs server Separation

**Vấn đề hiện tại:** Code trong `internal/consumer/server.go` đang vi phạm nguyên tắc tách biệt:

```go
// ❌ SAI: server.go đang tự khởi tạo Redis với if/else
if srv.cfg.RedisStateConfig.Host != "" {
    redisClient, err = pkgRedis.Connect(...)
    if err != nil {
        srv.l.Warnf(ctx, "Failed to connect...") // Optional failure
    }
}
```

**Nguyên tắc đúng:**

| Layer       | Trách nhiệm                                          | Behavior                                |
| ----------- | ---------------------------------------------------- | --------------------------------------- |
| `cmd/`      | Khởi tạo external dependencies (Redis, RabbitMQ, DB) | Fail fast nếu không connect được        |
| `server.go` | Nhận dependencies đã sẵn sàng, chạy business logic   | Không có `if`, khởi tạo PHẢI thành công |

**Cần refactor:**

1. `cmd/consumer/main.go`: Khởi tạo Redis client, fail nếu lỗi
2. `internal/consumer/new.go`: Nhận `redis.Client` đã connect
3. `internal/consumer/server.go`: Không có logic khởi tạo external resources

## What Changes

- **REMOVE**: Webhook throttling logic (`throttler.go`, `ThrottleInterval`, cleanup goroutine)
- **REMOVE**: Throttle-related config (`WEBHOOK_THROTTLE_INTERVAL`, `WEBHOOK_CLEANUP_INTERVAL`, `WEBHOOK_MAX_THROTTLE_ENTRIES`)
- **SIMPLIFY**: Progress tracking dựa trên số platforms hoàn thành thay vì số items
- **SIMPLIFY**: Webhook client chỉ cần `NotifyProgress()` đơn giản, không cần `NotifyProgressImmediate()`
- **REFACTOR**: Di chuyển Redis initialization từ `server.go` lên `cmd/consumer/main.go`

## Impact

- Affected files:
  - `internal/webhook/client/throttler.go` (DELETE)
  - `internal/webhook/client/throttler_test.go` (DELETE)
  - `internal/webhook/client/http_client.go` (SIMPLIFY)
  - `internal/webhook/types.go` (SIMPLIFY)
  - `config/config.go` (REMOVE throttle configs)
  - `template.env` (REMOVE throttle env vars)
  - `cmd/consumer/main.go` (ADD Redis init)
  - `internal/consumer/new.go` (CHANGE Config struct)
  - `internal/consumer/server.go` (REMOVE Redis init logic)
  - `document/event-drivent.md` (UPDATE documentation)
  - `README.md` (UPDATE documentation)
