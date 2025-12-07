# COLLECTOR SERVICE - POST-DEPLOYMENT INTEGRATION CHECKLIST

**Document Version:** 1.0
**Last Updated:** 2025-12-06
**Target Audience:** Collector Service Team, QA Engineers
**Purpose:** Verify Crawler-Collector Integration After Deployment

---

## Executive Summary

This document provides a comprehensive checklist to verify the integration between Crawler services (TikTok & YouTube) and Collector Service after the crawler refactor deployment. The crawler has implemented major changes to support event-driven architecture, and this checklist ensures all integration points function correctly.

**Crawler Changes Deployed:**

- Task type in result metadata (`task_type` field)
- Data.collected event publisher to `smap.events`
- Batch upload to MinIO (`crawl-results` bucket)
- Enhanced error reporting (17 error codes)
- Configuration externalization
- Retry logic with exponential backoff

**Integration Points to Verify:**

1. Result message routing (dry-run vs project execution)
2. Data.collected event consumption (future: Analytics Service)
3. Error handling and reporting
4. Progress tracking via Redis and webhooks

---

## Table of Contents

1. [Pre-Deployment Verification](#1-pre-deployment-verification)
2. [Task Type Routing Verification](#2-task-type-routing-verification)
3. [Error Handling Verification](#3-error-handling-verification)
4. [Progress Tracking Verification](#4-progress-tracking-verification)
5. [Data.Collected Event Verification](#5-datacollected-event-verification)
6. [Performance & Load Testing](#6-performance--load-testing)
7. [Rollback Procedures](#7-rollback-procedures)
8. [Sign-Off](#8-sign-off)

---

## 1. Pre-Deployment Verification

### 1.1. Environment Check

**Collector Service Configuration:**

- [x] Verify `internal/results/types.go` includes `TaskType` field in `CrawlerContentMeta`
- [x] Verify `internal/results/usecase/result.go` has routing logic based on `task_type`
- [x] Verify `handleDryRunResult()` method exists and routes to `/internal/dryrun/callback`
- [x] Verify `handleProjectResult()` method exists and routes to Redis + `/internal/progress/callback`
- [x] Verify backward compatibility: default routing when `task_type` is missing

**Check Configuration Files:**

```bash
# Verify Collector has latest code
cd /path/to/collector-service
git log --oneline -5

# Check for TaskType field
grep -n "TaskType" internal/results/types.go

# Check for routing logic
grep -n "handleDryRunResult\|handleProjectResult" internal/results/usecase/result.go
```

**Expected Output:**

```go
// internal/results/types.go
type CrawlerContentMeta struct {
    // ... other fields
    TaskType string `json:"task_type,omitempty"` //  Should be present
}

// internal/results/usecase/result.go
func (uc implUseCase) HandleResult(ctx context.Context, res models.CrawlerResult) error {
    taskType := uc.extractTaskType(ctx, res.Payload)

    switch taskType {
    case "dryrun_keyword":
        return uc.handleDryRunResult(ctx, res)
    case "research_and_crawl":
        return uc.handleProjectResult(ctx, res)
    default:
        return uc.handleDryRunResult(ctx, res)  // Backward compatibility
    }
}
```

**Verification Status:** ☐ PASS / ☐ FAIL

---

### 1.2. Crawler Service Verification

**Check Crawler Deployment:**

- [x] TikTok crawler deployed with refactor changes
- [x] YouTube crawler deployed with refactor changes
- [x] Result publisher configured: `result_exchange_name = "tiktok_exchange"`, `result_routing_key = "tiktok.res"`
- [x] Event publisher configured: `event_exchange_name = "smap.events"`, `event_routing_key = "data.collected"`
- [x] Batch storage configured: `minio_crawl_results_bucket = "crawl-results"`

**Check Crawler Configuration:**

```bash
# TikTok
cat tiktok/config/settings.py | grep -E "result_|event_|batch_"

# YouTube
cat youtube/config/settings.py | grep -E "result_|event_|batch_"
```

**Expected Output:**

```python
# Result Publisher Settings
result_publisher_enabled: bool = True
result_exchange_name: str = "tiktok_exchange"
result_routing_key: str = "tiktok.res"

# Event Publisher Settings
event_publisher_enabled: bool = True
event_exchange_name: str = "smap.events"
event_routing_key: str = "data.collected"

# Batch Upload Settings
batch_size: int = 50  # TikTok
minio_crawl_results_bucket: str = "crawl-results"
```

**Verification Status:** ☐ PASS / ☐ FAIL

---

## 2. Task Type Routing Verification

### 2.1. Dry-Run Task Verification

**Objective:** Verify dry-run results route to `/internal/dryrun/callback`

**Test Steps:**

**Step 1: Trigger Dry-Run Task**

```bash
# From Project Service or test script
curl -X POST http://project-service:8080/projects/dryrun \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "keywords": ["test keyword"],
    "platform": "tiktok"
  }'
```

**Step 2: Monitor Crawler Logs**

```bash
# TikTok crawler logs
docker logs tiktok-worker-1 --tail 100 -f | grep "task_type"

# Expected log output:
# INFO: Publishing result - job_id: 550e8400-..., task_type: dryrun_keyword
```

**Step 3: Monitor Collector Logs**

```bash
# Collector service logs
docker logs collector-service --tail 100 -f | grep -E "HandleResult|task_type|dryrun"

# Expected log output:
# INFO: HandleResult - task_type: dryrun_keyword
# INFO: Routing to handleDryRunResult
# INFO: Sending dry-run callback to /internal/dryrun/callback
```

**Step 4: Verify Callback Received**

```bash
# Project Service logs
docker logs project-service --tail 100 -f | grep "/internal/dryrun/callback"

# Expected log output:
# INFO: POST /internal/dryrun/callback - 200 OK
```

**Step 5: Verify Result in Database/UI**

- [x] Check Project Service database for dry-run results
- [x] Check UI shows dry-run results correctly
- [x] Verify NO Redis state update occurred (dry-run shouldn't affect project state)

**Checklist:**

- [x] Crawler includes `task_type: "dryrun_keyword"` in result meta
- [x] Collector routes to `handleDryRunResult()`
- [x] Callback sent to `/internal/dryrun/callback`
- [x] Project Service receives callback successfully
- [x] NO Redis state update
- [x] Results displayed in UI

**Test Data:**

```json
// Expected result format from crawler
{
  "success": true,
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "task_type": "dryrun_keyword",
  "platform": "tiktok",
  "keyword": "test keyword",
  "results_count": 10,
  "payload": [
    {
      "meta": {
        "id": "video123",
        "task_type": "dryrun_keyword",  //  VERIFY THIS
        "fetch_status": "success"
      },
      "content": { ... }
    }
  ]
}
```

**Verification Status:** ☐ PASS / ☐ FAIL

**Issues Found:** **\*\***\*\***\*\***\_\_\_\_**\*\***\*\***\*\***

---

### 2.2. Project Execution Task Verification

**Objective:** Verify project execution results route to Redis + `/internal/progress/callback`

**Test Steps:**

**Step 1: Create and Execute Project**

```bash
# Create project
PROJECT_ID=$(curl -X POST http://project-service:8080/projects \
  -H "Content-Type: application/json" \
  -H "Cookie: session=YOUR_SESSION" \
  -d '{
    "brand_name": "TestBrand",
    "brand_keywords": ["test keyword 1"],
    "competitor_names": [],
    "competitor_keywords_map": {},
    "date_range": {
      "from": "2025-12-01",
      "to": "2025-12-06"
    }
  }' | jq -r '.data.id')

echo "Created project: $PROJECT_ID"

# Execute project
curl -X POST http://project-service:8080/projects/$PROJECT_ID/execute \
  -H "Cookie: session=YOUR_SESSION"
```

**Step 2: Monitor Crawler Logs**

```bash
# Expected log output:
# INFO: Processing job - job_id: proj_test123-brand-0, task_type: research_and_crawl
# INFO: Publishing result - task_type: research_and_crawl
```

**Step 3: Monitor Collector Logs**

```bash
docker logs collector-service --tail 100 -f | grep -E "task_type|handleProjectResult|Redis|progress"

# Expected log output:
# INFO: HandleResult - task_type: research_and_crawl
# INFO: Routing to handleProjectResult
# INFO: Updating Redis state - project_id: proj_test123, done++
# INFO: Calling progress webhook - /internal/progress/callback
```

**Step 4: Verify Redis State Updates**

```bash
# Check Redis
redis-cli -n 1 HGETALL smap:proj:$PROJECT_ID

# Expected output:
# status: CRAWLING
# total: 100  # (or actual count)
# done: 1     # Incremented for each result
# errors: 0
```

**Step 5: Verify Progress Webhook**

```bash
# Project Service logs
docker logs project-service --tail 100 -f | grep "/internal/progress/callback"

# Expected log output:
# INFO: POST /internal/progress/callback - 200 OK
# Payload: {project_id: proj_test123, status: CRAWLING, total: 100, done: 1, errors: 0}
```

**Step 6: Verify WebSocket Notification (if applicable)**

```bash
# Check if user received progress update via WebSocket
# (Requires WebSocket client or browser devtools)
```

**Checklist:**

- [x] Crawler includes `task_type: "research_and_crawl"` in result meta
- [x] Collector routes to `handleProjectResult()`
- [x] Redis state updated correctly (done incremented)
- [x] Progress webhook sent to `/internal/progress/callback`
- [x] Project Service receives callback successfully
- [x] WebSocket notification sent to user (if implemented)
- [x] UI shows real-time progress

**Test Data:**

```json
// Expected result format from crawler
{
  "success": true,
  "job_id": "proj_test123-brand-0",
  "task_type": "research_and_crawl",  //  VERIFY THIS
  "platform": "tiktok",
  "keyword": "test keyword 1",
  "results_count": 50,
  "payload": [
    {
      "meta": {
        "id": "video456",
        "job_id": "proj_test123-brand-0",
        "task_type": "research_and_crawl",  //  VERIFY THIS
        "fetch_status": "success"
      },
      "content": { ... }
    }
  ]
}
```

**Verification Status:** ☐ PASS / ☐ FAIL

**Issues Found:** **\*\***\*\***\*\***\_\_\_\_**\*\***\*\***\*\***

---

### 2.3. Backward Compatibility Verification

**Objective:** Verify Collector handles old results without `task_type` field

**Test Steps:**

**Step 1: Send Legacy Result (Missing task_type)**

```bash
# Simulate old crawler result via RabbitMQ
python3 <<EOF
import pika
import json

connection = pika.BlockingConnection(pika.URLParameters('amqp://guest:guest@localhost:5672/'))
channel = connection.channel()

# Legacy result without task_type
legacy_result = {
    "success": True,
    "job_id": "legacy-job-123",
    "platform": "tiktok",
    "keyword": "legacy test",
    "payload": [{
        "meta": {
            "id": "video_legacy",
            # NO task_type field
            "fetch_status": "success"
        },
        "content": {"url": "https://tiktok.com/@test/video/legacy"}
    }]
}

channel.basic_publish(
    exchange='tiktok_exchange',
    routing_key='tiktok.res',
    body=json.dumps(legacy_result)
)

print("✓ Published legacy result")
connection.close()
EOF
```

**Step 2: Monitor Collector Logs**

```bash
docker logs collector-service --tail 50 -f

# Expected log output:
# WARN: task_type missing or empty, using default routing
# INFO: Routing to handleDryRunResult (backward compatibility)
# INFO: Sending dry-run callback
```

**Step 3: Verify Default Routing**

- [x] Collector routes to `handleDryRunResult()` by default
- [x] No errors or crashes
- [x] Callback sent successfully
- [x] Result processed correctly

**Checklist:**

- [x] Legacy results (no `task_type`) handled gracefully
- [x] Default routing to dry-run callback works
- [x] No errors in Collector logs
- [x] No service crashes or restarts

**Verification Status:** ☐ PASS / ☐ FAIL

---

## 3. Error Handling Verification

### 3.1. Error Code Propagation

**Objective:** Verify Collector receives and processes error codes from Crawler

**Test Steps:**

**Step 1: Trigger Error Scenario**

```bash
# Use a URL that will fail (deleted content, private content, etc.)
curl -X POST http://project-service:8080/projects/dryrun \
  -H "Content-Type: application/json" \
  -d '{
    "keywords": ["https://tiktok.com/@deleted/video/999999999"],
    "platform": "tiktok"
  }'
```

**Step 2: Monitor Crawler Logs**

```bash
# Expected log output:
# ERROR: Failed to fetch content - error_code: CONTENT_NOT_FOUND
# INFO: Publishing result with error - fetch_status: error, error_code: CONTENT_NOT_FOUND
```

**Step 3: Verify Error in Result**

```bash
# Collector logs
docker logs collector-service --tail 50 -f | grep -E "error_code|fetch_status"

# Expected: Result contains error information
```

**Step 4: Check Error Codes**

Common error codes to test:

| Error Code          | Test Scenario                | Expected Behavior  |
| ------------------- | ---------------------------- | ------------------ |
| `CONTENT_REMOVED`   | Deleted TikTok/YouTube video | Skip, log error    |
| `CONTENT_NOT_FOUND` | Invalid URL (404)            | Skip, log error    |
| `RATE_LIMITED`      | Too many requests            | Retry with backoff |
| `NETWORK_ERROR`     | Network issue                | Retry              |
| `PARSE_ERROR`       | Malformed response           | Skip, alert        |

**Checklist:**

- [x] Error codes present in result `meta.error_code`
- [x] Error messages present in `meta.fetch_error`
- [x] Error details present in `meta.error_details`
- [x] Collector logs error items appropriately
- [x] Error items NOT counted as successful crawls
- [x] Error metrics updated correctly

**Test Data:**

```json
{
  "payload": [
    {
      "meta": {
        "id": "video_error",
        "fetch_status": "error", //  VERIFY
        "fetch_error": "Content has been removed", //  VERIFY
        "error_code": "CONTENT_REMOVED", //  VERIFY
        "error_details": {
          "exception_type": "ContentRemovedError",
          "url": "https://tiktok.com/@test/video/999"
        }
      },
      "content": null,
      "author": null,
      "comments": []
    }
  ]
}
```

**Verification Status:** ☐ PASS / ☐ FAIL

---

### 3.2. Error Rate Monitoring

**Objective:** Verify error metrics are tracked correctly

**Test Steps:**

**Step 1: Generate Mix of Success/Error Results**

```bash
# Trigger crawl with mix of valid and invalid URLs
# (Use test script or manual execution)
```

**Step 2: Check Collector Metrics**

```bash
# If Collector exposes metrics endpoint
curl http://collector-service:9090/metrics | grep error

# Expected metrics:
# collector_items_error_total{platform="tiktok"} 5
# collector_error_rate{platform="tiktok"} 0.10  # 10%
```

**Step 3: Verify Error Distribution**

- [x] Errors categorized by `error_code`
- [x] Error count per category tracked
- [x] Error rate calculated correctly
- [x] Alerts triggered if error rate > threshold

**Checklist:**

- [x] Total error count tracked
- [x] Error rate percentage calculated
- [x] Error distribution by code available
- [x] Alerts configured and tested

**Verification Status:** ☐ PASS / ☐ FAIL

---

## 4. Progress Tracking Verification

### 4.1. Redis State Management

**Objective:** Verify Redis state is updated correctly during project execution

**Test Steps:**

**Step 1: Execute Small Project**

```bash
# Create project with 10-20 expected results
# (Use known keywords that return ~10 results)
```

**Step 2: Monitor Redis Updates**

```bash
# Watch Redis state in real-time
redis-cli -n 1
> SUBSCRIBE __keyspace@1__:smap:proj:*
# (In another terminal, execute project)

# Or poll state:
watch -n 1 'redis-cli -n 1 HGETALL smap:proj:YOUR_PROJECT_ID'
```

**Step 3: Verify State Transitions**

Expected state flow:

```
1. INITIALIZING (set by Project Service on /execute)
2. CRAWLING (set by Collector when total is known)
3. PROCESSING (optional, if analytics is running)
4. DONE (when done >= total && errors + done >= total)
```

**Step 4: Verify Field Updates**

| Field    | When Updated         | Updated By      | Verification                |
| -------- | -------------------- | --------------- | --------------------------- |
| `status` | Project start        | Project Service | Should be "INITIALIZING"    |
| `total`  | Search results found | Collector       | Should match expected count |
| `done`   | Each item processed  | Collector       | Should increment to total   |
| `errors` | Each error item      | Collector       | Should match error count    |
| `status` | All done             | Collector       | Should be "DONE"            |

**Checklist:**

- [x] Initial state set by Project Service (status=INITIALIZING, total=0, done=0, errors=0)
- [x] Total updated when Collector knows item count
- [x] Status changed to CRAWLING when total is set
- [x] Done incremented for each successful item
- [x] Errors incremented for each error item
- [x] Status changed to DONE when done >= total
- [x] Redis key has TTL (7 days)

**Verification Status:** ☐ PASS / ☐ FAIL

---

### 4.2. Progress Webhook Verification

**Objective:** Verify progress webhooks sent to Project Service

**Test Steps:**

**Step 1: Enable Webhook Logging**

```bash
# Project Service - enable debug logging for webhooks
export LOG_LEVEL=DEBUG
```

**Step 2: Execute Project**

```bash
# Execute project and monitor webhook calls
```

**Step 3: Verify Webhook Calls**

```bash
# Project Service logs
docker logs project-service --tail 200 -f | grep "/internal/progress/callback"

# Expected log pattern:
# POST /internal/progress/callback - Status: CRAWLING, Total: 100, Done: 1
# POST /internal/progress/callback - Status: CRAWLING, Total: 100, Done: 2
# ...
# POST /internal/progress/callback - Status: DONE, Total: 100, Done: 100
```

**Step 4: Verify Webhook Payload**

```json
{
  "project_id": "proj_test123",
  "user_id": "user_456",
  "status": "CRAWLING",
  "total": 100,
  "done": 50,
  "errors": 2
}
```

**Step 5: Verify Throttling**

- [x] Webhooks NOT sent for every single item (should be throttled)
- [x] Minimum interval between webhooks (e.g., 5 seconds)
- [x] Webhook always sent for important events (total set, status change to DONE/FAILED)

**Checklist:**

- [x] Webhook sent when total is set
- [x] Webhook sent periodically during crawling (throttled)
- [x] Webhook sent when status changes to DONE
- [x] Webhook sent when status changes to FAILED
- [x] Webhook payload correct (all fields present)
- [x] Webhook endpoint returns 200 OK
- [x] Throttling working correctly

**Verification Status:** ☐ PASS / ☐ FAIL

---

### 4.3. WebSocket Notification Verification (if applicable)

**Objective:** Verify users receive real-time progress via WebSocket

**Test Steps:**

**Step 1: Connect WebSocket Client**

```javascript
// Browser console or WebSocket client
const ws = new WebSocket("ws://project-service:8080/ws");
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log("Received:", message);
};
```

**Step 2: Execute Project**

```bash
# Execute project and monitor WebSocket messages
```

**Step 3: Verify Messages Received**

Expected message types:

```json
// Progress update
{
  "type": "project_progress",
  "payload": {
    "project_id": "proj_test123",
    "status": "CRAWLING",
    "total": 100,
    "done": 50,
    "errors": 2,
    "progress_percent": 50.0
  }
}

// Completion
{
  "type": "project_completed",
  "payload": {
    "project_id": "proj_test123",
    "status": "DONE",
    "total": 100,
    "done": 100,
    "errors": 5,
    "progress_percent": 100.0
  }
}
```

**Checklist:**

- [x] WebSocket connection established
- [x] Progress messages received during execution
- [x] Completion message received when done
- [x] Progress percentage calculated correctly
- [x] UI updates in real-time

**Verification Status:** ☐ PASS / ☐ FAIL

---

## 5. Data.Collected Event Verification

**Objective:** Verify `data.collected` events published to `smap.events` exchange

**Note:** This is for future Analytics Service integration. Collector Service does NOT consume these events.

**Test Steps:**

**Step 1: Set Up Event Consumer (Test)**

```python
# test_data_collected_consumer.py
import pika
import json

connection = pika.BlockingConnection(pika.URLParameters('amqp://guest:guest@localhost:5672/'))
channel = connection.channel()

# Declare exchange
channel.exchange_declare(exchange='smap.events', exchange_type='topic', durable=True)

# Declare test queue
channel.queue_declare(queue='test_data_collected', durable=True)

# Bind queue
channel.queue_bind(queue='test_data_collected', exchange='smap.events', routing_key='data.collected')

def callback(ch, method, properties, body):
    event = json.loads(body)
    print(f"✓ Received data.collected event:")
    print(f"  Event ID: {event['event_id']}")
    print(f"  Project ID: {event['payload'].get('project_id')}")
    print(f"  Platform: {event['payload']['platform']}")
    print(f"  MinIO Path: {event['payload']['minio_path']}")
    print(f"  Content Count: {event['payload']['content_count']}")
    print(f"  Batch Index: {event['payload']['batch_index']}")
    ch.basic_ack(delivery_tag=method.delivery_tag)

channel.basic_consume(queue='test_data_collected', on_message_callback=callback, auto_ack=False)
print('Waiting for data.collected events...')
channel.start_consuming()
```

**Step 2: Execute Project**

```bash
# Run test consumer in one terminal
python3 test_data_collected_consumer.py

# Execute project in another terminal
curl -X POST http://project-service:8080/projects/PROJECT_ID/execute
```

**Step 3: Verify Events Received**

Expected events for TikTok (batch_size=50):

- 100 items → 2 events (batch_000, batch_001)
- 150 items → 3 events (batch_000, batch_001, batch_002)

Expected events for YouTube (batch_size=20):

- 100 items → 5 events
- 50 items → 3 events (batch_000, batch_001, batch_002)

**Step 4: Verify Event Schema**

```json
{
  "event_id": "evt_abc123def456", //  Format: evt_{12_hex}
  "timestamp": "2025-12-06T10:30:00.000Z", //  ISO 8601 UTC
  "payload": {
    "project_id": "proj_test123", //  Extracted from job_id
    "job_id": "proj_test123-brand-0", //  Full job_id
    "platform": "tiktok", //  Platform name
    "minio_path": "crawl-results/tiktok/proj_test123/brand/batch_000.json", //  Full path
    "content_count": 50, //  Items in batch
    "batch_index": 1, //  1-based index
    "total_batches": 3 //  Optional
  }
}
```

**Step 5: Verify MinIO Batch Files**

```bash
# Check batch files exist in MinIO
mc ls myminio/crawl-results/tiktok/proj_test123/brand/

# Expected output:
# batch_000.json
# batch_001.json
# batch_002.json

# Download and verify content
mc cp myminio/crawl-results/tiktok/proj_test123/brand/batch_000.json ./
cat batch_000.json | jq '. | length'
# Expected: 50 (for TikTok)

# Verify item structure
cat batch_000.json | jq '.[0] | keys'
# Expected: ["meta", "content", "author", "comments"]
```

**Checklist:**

- [x] Events published to `smap.events` exchange
- [x] Routing key is `data.collected`
- [x] Event schema correct (event_id, timestamp, payload)
- [x] project_id extracted correctly (or null for dry-run)
- [x] minio_path correct format
- [x] content_count matches actual batch size
- [x] batch_index is 1-based and sequential
- [x] Batch files exist in MinIO at specified paths
- [x] Batch files contain correct number of items
- [x] Item structure correct (meta, content, author, comments)

**Verification Status:** ☐ PASS / ☐ FAIL

---

### 5.2. Batch Upload Verification

**Objective:** Verify batch files uploaded to MinIO correctly

**Test Steps:**

**Step 1: Execute Project and Monitor**

```bash
# Monitor crawler logs for batch uploads
docker logs tiktok-worker-1 --tail 200 -f | grep -E "batch|MinIO|upload"

# Expected log output:
# INFO: Batch full (50 items), uploading to MinIO
# INFO: Uploaded batch to crawl-results/tiktok/proj_test123/brand/batch_000.json
# INFO: Published data.collected event for batch 1
```

**Step 2: List MinIO Objects**

```bash
# Install MinIO client (mc)
mc alias set myminio http://minio:9000 ACCESS_KEY SECRET_KEY

# List crawl-results bucket
mc ls myminio/crawl-results/tiktok/ --recursive

# Expected structure:
# crawl-results/tiktok/proj_test123/brand/batch_000.json
# crawl-results/tiktok/proj_test123/brand/batch_001.json
# crawl-results/tiktok/proj_test123/competitor_toyota/batch_000.json
```

**Step 3: Verify Batch Content**

```bash
# Download batch file
mc cp myminio/crawl-results/tiktok/proj_test123/brand/batch_000.json ./batch_000.json

# Verify structure
cat batch_000.json | jq '
  {
    item_count: length,
    first_item_keys: .[0] | keys,
    fetch_statuses: [.[] | .meta.fetch_status] | unique,
    task_types: [.[] | .meta.task_type] | unique
  }
'

# Expected output:
# {
#   "item_count": 50,
#   "first_item_keys": ["meta", "content", "author", "comments"],
#   "fetch_statuses": ["success"] or ["success", "error"],
#   "task_types": ["research_and_crawl"]
# }
```

**Step 4: Verify Compression (if enabled)**

```bash
# Check file extension
mc ls myminio/crawl-results/tiktok/proj_test123/brand/

# If .zst extension:
mc cp myminio/crawl-results/tiktok/proj_test123/brand/batch_000.json.zst ./
zstd -d batch_000.json.zst
cat batch_000.json | jq '. | length'
```

**Checklist:**

- [x] Batch files created in MinIO
- [x] Path format correct: `{platform}/{project_id}/{subfolder}/batch_{index:03d}.json`
- [x] Batch size correct (TikTok: 50, YouTube: 20)
- [x] JSON array format
- [x] All items have required fields (meta, content, author, comments)
- [x] task_type present in all items
- [x] Compression working (if enabled)

**Verification Status:** ☐ PASS / ☐ FAIL

---

## 6. Performance & Load Testing

### 6.1. Throughput Verification

**Objective:** Verify system handles expected load

**Test Steps:**

**Step 1: Execute Large Project**

```bash
# Create project with high-volume keywords
# Expected results: 1000-2000 items
```

**Step 2: Monitor Performance Metrics**

| Metric                    | Target              | Actual | Status |
| ------------------------- | ------------------- | ------ | ------ |
| Crawler processing time   | < 30s per 50 items  | \_\_\_ | ☐      |
| Batch upload latency      | < 500ms             | \_\_\_ | ☐      |
| Event publish latency     | < 100ms             | \_\_\_ | ☐      |
| Collector processing time | < 1s per result     | \_\_\_ | ☐      |
| Redis update latency      | < 50ms              | \_\_\_ | ☐      |
| Webhook latency           | < 200ms             | \_\_\_ | ☐      |
| End-to-end latency        | < 60s per 100 items | \_\_\_ | ☐      |

**Step 3: Check Resource Usage**

```bash
# Crawler CPU/Memory
docker stats tiktok-worker-1 --no-stream

# Collector CPU/Memory
docker stats collector-service --no-stream

# Redis CPU/Memory
docker stats redis --no-stream
```

**Checklist:**

- [x] No timeouts or connection errors
- [x] No message loss
- [x] No data corruption
- [x] Resource usage within limits (CPU < 80%, Memory < 80%)
- [x] No service crashes or restarts

**Verification Status:** ☐ PASS / ☐ FAIL

---

### 6.2. Concurrent Execution Verification

**Objective:** Verify system handles multiple concurrent projects

**Test Steps:**

**Step 1: Execute 3-5 Projects Simultaneously**

```bash
# Execute multiple projects at once
for i in {1..5}; do
  curl -X POST http://project-service:8080/projects/project_$i/execute &
done
wait
```

**Step 2: Monitor System Behavior**

- [x] All projects execute successfully
- [x] No resource exhaustion
- [x] No deadlocks or race conditions
- [x] Progress tracking correct for all projects
- [x] No cross-project interference

**Checklist:**

- [x] All projects complete successfully
- [x] Redis state isolated per project
- [x] Webhooks sent to correct users
- [x] No message mixing between projects
- [x] Performance acceptable under load

**Verification Status:** ☐ PASS / ☐ FAIL

---

## 7. Rollback Procedures

### 7.1. Rollback Triggers

**Trigger rollback if:**

- [x] Error rate > 20% in production
- [x] Collector crashes or restarts repeatedly
- [x] Data loss detected
- [x] Progress tracking broken (Redis not updating)
- [x] Webhooks failing consistently
- [x] Critical bugs discovered

### 7.2. Rollback Steps

**Step 1: Disable New Routing Logic**

```go
// Quick fix in Collector code
func (uc implUseCase) HandleResult(ctx context.Context, res models.CrawlerResult) error {
    // ROLLBACK: Force all to dry-run handler (old behavior)
    return uc.handleDryRunResult(ctx, res)
}
```

**Step 2: Redeploy Collector**

```bash
# Deploy rollback version
kubectl rollout undo deployment/collector-service

# Or manually deploy previous version
docker pull collector-service:previous-version
docker stop collector-service
docker run -d collector-service:previous-version
```

**Step 3: Verify Rollback**

- [x] Collector running stable version
- [x] No errors in logs
- [x] Results processing correctly
- [x] System stable

**Step 4: Root Cause Analysis**

- [x] Review logs and metrics
- [x] Identify failure point
- [x] Document issues
- [x] Plan fixes

**Verification Status:** ☐ PASS / ☐ FAIL / ☐ N/A

---

## 8. Sign-Off

### 8.1. Test Summary

**Date:** **\*\*\*\***\_\_**\*\*\*\***
**Tested By:** **\*\*\*\***\_\_**\*\*\*\***
**Environment:** ☐ Development ☐ Staging ☐ Production

**Overall Status:**

- Task Type Routing: ☐ PASS / ☐ FAIL
- Error Handling: ☐ PASS / ☐ FAIL
- Progress Tracking: ☐ PASS / ☐ FAIL
- Data.Collected Events: ☐ PASS / ☐ FAIL
- Performance: ☐ PASS / ☐ FAIL

**Critical Issues Found:** **\*\***\*\***\*\***\_\_\_\_**\*\***\*\***\*\***

**Non-Critical Issues Found:** **\*\***\*\***\*\***\_\_\_\_**\*\***\*\***\*\***

---

### 8.2. Approvals

**QA Engineer:**

- Name: **\*\*\*\***\_\_**\*\*\*\***
- Signature: **\*\*\*\***\_\_**\*\*\*\***
- Date: **\*\*\*\***\_\_**\*\*\*\***

**Backend Lead:**

- Name: **\*\*\*\***\_\_**\*\*\*\***
- Signature: **\*\*\*\***\_\_**\*\*\*\***
- Date: **\*\*\*\***\_\_**\*\*\*\***

**DevOps Engineer:**

- Name: **\*\*\*\***\_\_**\*\*\*\***
- Signature: **\*\*\*\***\_\_**\*\*\*\***
- Date: **\*\*\*\***\_\_**\*\*\*\***

---

### 8.3. Deployment Decision

☐ **APPROVED FOR PRODUCTION** - All tests passed, no critical issues

☐ **APPROVED WITH CONDITIONS** - Minor issues found, can deploy with monitoring

- Conditions: **\*\***\*\***\*\***\_\_\_\_**\*\***\*\***\*\***

☐ **NOT APPROVED** - Critical issues found, rollback required

- Blocking Issues: **\*\***\*\***\*\***\_\_\_\_**\*\***\*\***\*\***

---

**End of Checklist**

**Document Version:** 1.0
**Last Updated:** 2025-12-06
**Next Review:** After Production Deployment
