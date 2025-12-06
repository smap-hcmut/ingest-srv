# Change: Align Collector Service với Event-Driven Architecture Document

## Why

Hiện tại Collector Service (SMAP Dispatcher) đang sử dụng cấu trúc exchange/routing key khác với document `event-drivent.md`. Document này định nghĩa chuẩn event-driven choreography cho toàn bộ hệ thống SMAP, nhưng implementation hiện tại chưa tuân thủ đầy đủ:

1. **Exchange naming không đồng nhất**: Document yêu cầu sử dụng `smap.events` (topic exchange), nhưng hiện tại đang dùng `collector.inbound`, `tiktok_exchange`, `youtube_exchange`
2. **Routing key không theo chuẩn**: Document định nghĩa `project.created`, `data.collected`, nhưng hiện tại dùng `crawler.#`, `tiktok.crawl`, `youtube.crawl`
3. **Event schema chưa chuẩn hóa**: Document yêu cầu `ProjectCreatedEvent` với `user_id`, `brand_keywords`, `competitor_keywords_map`, nhưng hiện tại dùng `CrawlRequest` với cấu trúc khác
4. **Thiếu Redis State Management**: Document yêu cầu Collector cập nhật Redis state (`smap:proj:{projectID}`), nhưng hiện tại chưa implement
5. **Thiếu Progress Webhook**: Document yêu cầu gọi webhook `/internal/progress/callback` để notify progress, nhưng hiện tại chỉ có dry-run callback

## What Changes

### **BREAKING** - Exchange & Routing Key Restructure
- Thay đổi inbound exchange từ `collector.inbound` sang `smap.events`
- Thay đổi routing key từ `crawler.#` sang `project.created`
- Giữ nguyên outbound exchanges cho workers (backward compatible)

### Event Schema Migration
- Thêm support cho `ProjectCreatedEvent` schema mới
- Backward compatible: vẫn support `CrawlRequest` schema cũ trong giai đoạn migration

### Redis State Management
- Thêm Redis client cho state tracking (DB 1)
- Implement state update functions: `UpdateTotal`, `IncrementDone`, `IncrementErrors`, `UpdateStatus`
- Key schema: `smap:proj:{projectID}`

### Progress Webhook Integration
- Thêm progress webhook client để gọi `/internal/progress/callback`
- Implement throttling (5 giây minimum)
- Gọi webhook khi: set total, status change, periodically during crawling

### Publish `data.collected` Event
- Sau khi crawl xong, publish event `data.collected` để Analytics Service consume

## Impact

- **Affected specs**: event-infrastructure (new)
- **Affected code**:
  - `internal/dispatcher/delivery/rabbitmq/constants.go` - Exchange/routing key changes
  - `internal/dispatcher/delivery/rabbitmq/consumer/` - New event schema handling
  - `internal/models/` - New event types
  - `config/config.go` - Redis state config
  - `internal/consumer/` - Redis client initialization
  - New module: `internal/state/` - Redis state management
  - New module: `internal/webhook/` - Progress webhook client
