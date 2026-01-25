# Requirements Document

## Introduction

Tài liệu này mô tả các yêu cầu để sửa chữa và cải thiện flow xử lý message giữa Collector Service với Crawler và Analytics Service trong case chạy project. Các vấn đề chính bao gồm:

1. **Message Contract chưa được xác định rõ ràng** - Struct giữa các service chưa được define chính xác, dẫn đến có field thừa hoặc không được sử dụng đúng cách
2. **Collector chưa handle đúng case platform limitation** - Khi crawler trả về ít bài hơn limit do giới hạn nền tảng
3. **Collector chưa tính toán tổng analyze result** - Chưa xử lý đúng việc tính tổng success/error từ Analytics để xác định completion

## Glossary

- **Collector Service**: Middleware service quản lý state và điều phối tasks giữa Project Service, Crawler Workers và Analytics Service
- **Crawler Worker**: Service crawl data từ TikTok/YouTube platforms
- **Analytics Service**: Service phân tích data đã crawl được
- **Project State**: Trạng thái execution của project được lưu trong Redis
- **Task-Level Tracking**: Đếm số lượng tasks (mỗi response từ Crawler = 1 task)
- **Item-Level Tracking**: Đếm số lượng items thực tế crawl được
- **Platform Limitation**: Giới hạn của platform (TikTok/YouTube) khiến không thể crawl đủ số lượng yêu cầu
- **Two-Phase Processing**: Quy trình 2 giai đoạn: Crawl → Analyze
- **CrawlerResult**: Message format từ Crawler gửi về Collector
- **AnalyzeResultPayload**: Message format từ Analytics gửi về Collector

## Requirements

### Requirement 1: Crawler → Collector Message Contract

**User Story:** As a system developer, I want a clearly defined message contract between Crawler and Collector, so that both services can communicate correctly without ambiguity.

#### Acceptance Criteria

1. WHEN Crawler completes a crawl task THEN the Collector SHALL receive a CrawlerResultMessage containing `success`, `task_type`, `job_id`, `platform`, `requested_limit`, `applied_limit`, `total_found`, `platform_limited`, `successful`, `failed`, `skipped` fields at root level (FLAT format, no nested objects)
2. WHEN Crawler encounters platform limitation THEN the CrawlerResultMessage SHALL include `platform_limited = true` and `total_found` reflecting actual items found
3. WHEN Crawler successfully crawls items THEN the message SHALL contain accurate `successful`, `failed`, `skipped` counts at root level
4. WHEN Crawler sends result THEN the message SHALL contain `job_id` and `platform` at root level for routing and tracking (no payload required - Crawler pushes content directly to Analytics)
5. WHEN Crawler task fails THEN the message SHALL include `error_code` and `error_message` fields at root level

### Requirement 2: Analytics → Collector Message Contract

**User Story:** As a system developer, I want a clearly defined message contract between Analytics and Collector, so that analyze results can be properly tracked and aggregated.

#### Acceptance Criteria

1. WHEN Analytics completes analyzing a batch THEN the Collector SHALL receive an AnalyzeResultPayload containing `project_id`, `job_id`, `task_type`, `batch_size`, `success_count`, and `error_count` fields
2. WHEN Collector receives AnalyzeResultPayload THEN the Collector SHALL validate that `task_type` equals "analyze_result" for proper routing
3. WHEN Collector receives AnalyzeResultPayload THEN the Collector SHALL extract `project_id` directly from payload instead of parsing from `job_id`
4. WHEN `success_count` or `error_count` is greater than zero THEN the Collector SHALL increment corresponding Redis counters atomically

### Requirement 3: Platform Limitation Handling

**User Story:** As a system operator, I want Collector to properly handle cases where Crawler returns fewer items than requested due to platform limitations, so that project completion can be accurately determined.

#### Acceptance Criteria

1. WHEN Crawler returns `limit_info.platform_limited = true` THEN the Collector SHALL log a warning with `requested_limit` and `total_found` values
2. WHEN Crawler returns fewer items than `limit_per_keyword` THEN the Collector SHALL update `items_actual` with actual count from `stats.successful` instead of expected count
3. WHEN updating item-level counters THEN the Collector SHALL use `stats.successful` and `stats.failed` from CrawlerResult instead of assuming all items succeeded
4. WHEN calculating crawl progress THEN the Collector SHALL use task-level completion (`tasks_done + tasks_errors >= tasks_total`) for determining phase completion
5. WHEN displaying progress to user THEN the Collector SHALL use item-level progress (`items_actual / items_expected`) for more accurate representation

### Requirement 4: Analyze Phase Completion Tracking

**User Story:** As a system operator, I want Collector to accurately track and aggregate analyze results, so that project completion status can be correctly determined.

#### Acceptance Criteria

1. WHEN Crawler successfully crawls N items THEN the Collector SHALL auto-increment `analyze_total` by N (each crawled item = 1 item to analyze)
2. WHEN Analytics reports `success_count` and `error_count` THEN the Collector SHALL increment `analyze_done` by `success_count` and `analyze_errors` by `error_count`
3. WHEN checking project completion THEN the Collector SHALL verify both conditions: `IsCrawlComplete()` AND `IsAnalyzeComplete()`
4. WHEN `analyze_done + analyze_errors >= analyze_total` AND `tasks_done + tasks_errors >= tasks_total` THEN the Collector SHALL update project status to DONE
5. WHEN project status changes to DONE THEN the Collector SHALL send completion webhook to Project Service

### Requirement 5: State Consistency and Atomic Updates

**User Story:** As a system developer, I want Redis state updates to be atomic and consistent, so that concurrent updates from multiple Crawler/Analytics responses do not cause race conditions.

#### Acceptance Criteria

1. WHEN updating multiple counters THEN the Collector SHALL use Redis HINCRBY for atomic increments
2. WHEN updating task-level counters THEN the Collector SHALL also update legacy `crawl_done`/`crawl_errors` fields for backward compatibility
3. WHEN initializing project state THEN the Collector SHALL set both `tasks_total` and `items_expected` in a single Redis operation
4. WHEN checking completion THEN the Collector SHALL read state and update status atomically to prevent race conditions
5. WHEN state key does not exist THEN the Collector SHALL return appropriate error instead of creating invalid state

### Requirement 6: Progress Webhook Accuracy

**User Story:** As a Project Service consumer, I want to receive accurate progress information from Collector, so that users can see real-time progress of their project execution.

#### Acceptance Criteria

1. WHEN sending progress webhook THEN the Collector SHALL include both task-level (`tasks`) and item-level (`items`) progress in the payload
2. WHEN calculating `overall_progress_percent` THEN the Collector SHALL average crawl progress and analyze progress: `(crawl_progress + analyze_progress) / 2`
3. WHEN crawl phase has platform limitations THEN the progress webhook SHALL reflect actual items crawled, not expected items
4. WHEN analyze phase is in progress THEN the progress webhook SHALL include `analyze.total`, `analyze.done`, `analyze.errors`, and `analyze.progress_percent`
5. WHEN project completes THEN the Collector SHALL send a final webhook with `status = "DONE"` and `overall_progress_percent = 100`
