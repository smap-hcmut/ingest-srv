** Tai lieu nay chưa chính thức nên không dùng để lên proposal cho implement **
# RabbitMQ Integration Specification
## Ingest-Social ↔ Crawler Services

**Version:** 1.0  
**Date:** 2026-02-11  
**Services:** ingest-social (Producer), youtube-crawler, tiktok-crawler, facebook-crawler (Consumers)

---

## Overview

Ingest-social service gửi crawl tasks đến các crawler services thông qua RabbitMQ. Crawlers lưu kết quả lên MinIO theo batch và trả về links để ingest-social tải về xử lý.

### Architecture Flow

```
Ingest-Social (Producer)
    ↓ publish task
[Exchange: social.direct]
    ↓ route by platform
[Queue: {platform}.{action}.{type}]
    ↓ consume
Crawler Services
    ↓ crawl & save to MinIO
    ↓ publish result
[Exchange: social.results]
    ↓ route by results.{platform}
[Queue: ingest.results.{platform}]
    ↓ consume
Ingest-Social downloads from MinIO
```

---

## RabbitMQ Components

### Exchanges

#### 1. `social.direct`
- **Type:** `direct`
- **Purpose:** Route crawl tasks từ ingest-social đến crawler tương ứng
- **Durable:** `true`
- **Auto-delete:** `false`

#### 2. `social.results`
- **Type:** `topic`
- **Purpose:** Route kết quả crawl từ crawlers về ingest-social
- **Durable:** `true`
- **Auto-delete:** `false`

---

### Queues Configuration

#### Ingest-Social Queues

| Queue Name                | Binding Exchange | Routing Key        | Purpose                          |
| ------------------------- | ---------------- | ------------------ | -------------------------------- |
| `ingest.results.youtube`  | `social.results` | `results.youtube`  | Nhận kết quả từ YouTube crawler  |
| `ingest.results.tiktok`   | `social.results` | `results.tiktok`   | Nhận kết quả từ TikTok crawler   |
| `ingest.results.facebook` | `social.results` | `results.facebook` | Nhận kết quả từ Facebook crawler |

**Properties:**
- **Durable:** `true`
- **Auto-delete:** `false`
- **Message TTL:** 86400000 (24 hours)
- **Max Length:** 10000

---

#### YouTube Crawler Queues

| Queue Name                  | Binding Exchange | Routing Key                 | Purpose                                |
| --------------------------- | ---------------- | --------------------------- | -------------------------------------- |
| `youtube.research.link`     | `social.direct`  | `youtube.research.link`     | Research links theo keywords           |
| `youtube.crawl.link`        | `social.direct`  | `youtube.crawl.link`        | Crawl theo links cụ thể                |
| `youtube.crawl.keyword`     | `social.direct`  | `youtube.crawl.keyword`     | Crawl theo keywords                    |
| `youtube.research.profile`  | `social.direct`  | `youtube.research.profile`  | Research profiles theo keywords        |
| `youtube.crawl.profile`     | `social.direct`  | `youtube.crawl.profile`     | Crawl profile metadata                 |
| `youtube.crawl.profile.all` | `social.direct`  | `youtube.crawl.profile.all` | Crawl toàn bộ posts/videos của profile |

**Properties:**
- **Durable:** `true`
- **Auto-delete:** `false`
- **Message TTL:** 3600000 (1 hour)
- **Max Length:** 5000
- **Prefetch Count:** 1 (consume 1 task at a time)

---

#### TikTok Crawler Queues

| Queue Name                 | Binding Exchange | Routing Key                | Purpose                                |
| -------------------------- | ---------------- | -------------------------- | -------------------------------------- |
| `tiktok.research.link`     | `social.direct`  | `tiktok.research.link`     | Research links theo keywords           |
| `tiktok.crawl.link`        | `social.direct`  | `tiktok.crawl.link`        | Crawl theo links cụ thể                |
| `tiktok.crawl.keyword`     | `social.direct`  | `tiktok.crawl.keyword`     | Crawl theo keywords                    |
| `tiktok.research.profile`  | `social.direct`  | `tiktok.research.profile`  | Research profiles theo keywords        |
| `tiktok.crawl.profile`     | `social.direct`  | `tiktok.crawl.profile`     | Crawl profile metadata                 |
| `tiktok.crawl.profile.all` | `social.direct`  | `tiktok.crawl.profile.all` | Crawl toàn bộ posts/videos của profile |

**Properties:** Same as YouTube Crawler Queues

---

#### Facebook Crawler Queues

| Queue Name                   | Binding Exchange | Routing Key                  | Purpose                                |
| ---------------------------- | ---------------- | ---------------------------- | -------------------------------------- |
| `facebook.research.link`     | `social.direct`  | `facebook.research.link`     | Research links theo keywords           |
| `facebook.crawl.link`        | `social.direct`  | `facebook.crawl.link`        | Crawl theo links cụ thể                |
| `facebook.crawl.keyword`     | `social.direct`  | `facebook.crawl.keyword`     | Crawl theo keywords                    |
| `facebook.research.profile`  | `social.direct`  | `facebook.research.profile`  | Research profiles theo keywords        |
| `facebook.crawl.profile`     | `social.direct`  | `facebook.crawl.profile`     | Crawl profile metadata                 |
| `facebook.crawl.profile.all` | `social.direct`  | `facebook.crawl.profile.all` | Crawl toàn bộ posts/videos của profile |

**Properties:** Same as YouTube Crawler Queues

---

## Message Formats

### Request Message (Ingest-Social → Crawler)

#### Message Structure

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "platform": "youtube",
  "task_type": "research_link",
  "payload": {
    "keywords": ["keyword1", "keyword2"],
    "links": ["https://youtube.com/watch?v=xxx"],
    "profile_links": ["https://youtube.com/@channel"]
  },
  "options": {
    "max_results": 100,
    "timeout": 300,
    "batch_size": 50,
    "priority": "normal"
  },
  "metadata": {
    "project_id": "proj-123",
    "user_id": "user-456"
  },
  "created_at": "2026-02-11T10:30:00Z"
}
```

#### Field Descriptions

| Field                   | Type             | Required | Description                               |
| ----------------------- | ---------------- | -------- | ----------------------------------------- |
| `task_id`               | string (UUID)    | ✅        | Unique ID của task                        |
| `platform`              | enum             | ✅        | `youtube`, `tiktok`, hoặc `facebook`      |
| `task_type`             | enum             | ✅        | Loại task (xem bảng dưới)                 |
| `payload`               | object           | ✅        | Data input cho task                       |
| `payload.keywords`      | string[]         | ⚠️        | Required cho research_*, crawl_keyword    |
| `payload.links`         | string[]         | ⚠️        | Required cho crawl_link                   |
| `payload.profile_links` | string[]         | ⚠️        | Required cho crawl_profile*               |
| `options`               | object           | ❌        | Tuỳ chọn crawl                            |
| `options.max_results`   | integer          | ❌        | Số lượng kết quả tối đa (default: 100)    |
| `options.timeout`       | integer          | ❌        | Timeout seconds (default: 300)            |
| `options.batch_size`    | integer          | ❌        | Items per batch (default: 50)             |
| `options.priority`      | enum             | ❌        | `low`, `normal`, `high` (default: normal) |
| `metadata`              | object           | ❌        | Metadata bổ sung                          |
| `created_at`            | string (ISO8601) | ✅        | Timestamp tạo task                        |

#### Task Types

| Task Type           | Routing Key Pattern            | Input             | Output                           |
| ------------------- | ------------------------------ | ----------------- | -------------------------------- |
| `research_link`     | `{platform}.research.link`     | `keywords[]`      | Links của posts/videos           |
| `crawl_link`        | `{platform}.crawl.link`        | `links[]`         | Raw data của posts/videos        |
| `crawl_keyword`     | `{platform}.crawl.keyword`     | `keywords[]`      | Raw data từ keyword search       |
| `research_profile`  | `{platform}.research.profile`  | `keywords[]`      | Links của profiles               |
| `crawl_profile`     | `{platform}.crawl.profile`     | `profile_links[]` | Metadata của profiles            |
| `crawl_profile_all` | `{platform}.crawl.profile.all` | `profile_links[]` | Toàn bộ posts/videos của profile |

---

### Response Message (Crawler → Ingest-Social)

#### Message Structure

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "platform": "youtube",
  "status": "success",
  "data": {
    "minio_bucket": "crawler-results",
    "batches": [
      {
        "batch_id": "batch-001",
        "minio_path": "youtube/2026-02-11/550e8400-e29b-41d4-a716-446655440000/batch_1.json",
        "minio_url": "s3://crawler-results/youtube/2026-02-11/550e8400-e29b-41d4-a716-446655440000/batch_1.json",
        "item_count": 50,
        "size_bytes": 1048576,
        "created_at": "2026-02-11T10:35:00Z"
      },
      {
        "batch_id": "batch-002",
        "minio_path": "youtube/2026-02-11/550e8400-e29b-41d4-a716-446655440000/batch_2.json",
        "minio_url": "s3://crawler-results/youtube/2026-02-11/550e8400-e29b-41d4-a716-446655440000/batch_2.json",
        "item_count": 45,
        "size_bytes": 987654,
        "created_at": "2026-02-11T10:35:30Z"
      }
    ],
    "total_items": 95,
    "total_batches": 2,
    "total_size_bytes": 2036230
  },
  "metadata": {
    "crawl_duration_seconds": 120,
    "batch_size": 50,
    "crawler_version": "1.0.0"
  },
  "error": null,
  "completed_at": "2026-02-11T10:35:30Z"
}
```

#### Field Descriptions

| Field                       | Type             | Required | Description                   |
| --------------------------- | ---------------- | -------- | ----------------------------- |
| `task_id`                   | string (UUID)    | ✅        | Task ID từ request            |
| `platform`                  | enum             | ✅        | Platform đã crawl             |
| `status`                    | enum             | ✅        | `success`, `error`, `partial` |
| `data`                      | object           | ✅        | Kết quả crawl                 |
| `data.minio_bucket`         | string           | ✅        | MinIO bucket name             |
| `data.batches`              | array            | ✅        | Danh sách batch files         |
| `data.batches[].batch_id`   | string           | ✅        | Unique batch ID               |
| `data.batches[].minio_path` | string           | ✅        | Relative path trong bucket    |
| `data.batches[].minio_url`  | string           | ✅        | Full S3 URL                   |
| `data.batches[].item_count` | integer          | ✅        | Số items trong batch          |
| `data.batches[].size_bytes` | integer          | ✅        | Kích thước file               |
| `data.batches[].created_at` | string           | ✅        | Timestamp tạo batch           |
| `data.total_items`          | integer          | ✅        | Tổng số items crawled         |
| `data.total_batches`        | integer          | ✅        | Tổng số batches               |
| `data.total_size_bytes`     | integer          | ✅        | Tổng kích thước               |
| `metadata`                  | object           | ❌        | Metadata bổ sung              |
| `error`                     | string/null      | ❌        | Error message nếu có          |
| `completed_at`              | string (ISO8601) | ✅        | Timestamp hoàn thành          |

#### Status Values

| Status    | Description                                              |
| --------- | -------------------------------------------------------- |
| `success` | Task hoàn thành thành công                               |
| `partial` | Task hoàn thành nhưng không đầy đủ (một số items failed) |
| `error`   | Task failed hoàn toàn                                    |

---

## MinIO Storage

### Bucket Structure

**Bucket Name:** `crawler-results`

**Path Pattern:**
```
{platform}/{date}/{task_id}/batch_{n}.json
```

**Examples:**
```
crawler-results/
├── youtube/
│   └── 2026-02-11/
│       └── 550e8400-e29b-41d4-a716-446655440000/
│           ├── batch_1.json
│           ├── batch_2.json
│           └── batch_3.json
├── tiktok/
│   └── 2026-02-11/
│       └── 660e9511-f30c-52e5-b827-557766551111/
│           └── batch_1.json
└── facebook/
    └── 2026-02-11/
        └── 770ea622-g41d-63f6-c938-668877662222/
            ├── batch_1.json
            └── batch_2.json
```

### Batch File Format

```json
{
  "batch_id": "batch-001",
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "platform": "youtube",
  "batch_index": 1,
  "items": [
    {
      "id": "video_id_123",
      "type": "video",
      "url": "https://youtube.com/watch?v=xxx",
      "data": {
        "title": "Video Title",
        "description": "Video description",
        "author": "Channel Name",
        "views": 1000000,
        "likes": 50000,
        "comments": 1500,
        "published_at": "2026-02-10T15:00:00Z"
      }
    },
    {
      "id": "video_id_456",
      "type": "video",
      "url": "https://youtube.com/watch?v=yyy",
      "data": {
        "title": "Another Video",
        "description": "Another description",
        "author": "Channel Name 2",
        "views": 500000,
        "likes": 25000,
        "comments": 800,
        "published_at": "2026-02-09T10:30:00Z"
      }
    }
  ],
  "crawled_at": "2026-02-11T10:35:00Z"
}
```

### MinIO Configuration

| Setting        | Value                          |
| -------------- | ------------------------------ |
| **Endpoint**   | `minio:9000` (internal)        |
| **Access Key** | From environment variable      |
| **Secret Key** | From environment variable      |
| **Bucket**     | `crawler-results`              |
| **Region**     | `us-east-1`                    |
| **Retention**  | 7 days (auto-delete old files) |

---

## Implementation Guide

### Ingest-Social Service

#### Publishing Tasks

```python
import json
import uuid
from datetime import datetime

def publish_crawl_task(platform, task_type, payload, options=None):
    """
    Publish a crawl task to RabbitMQ
    
    Args:
        platform: 'youtube', 'tiktok', or 'facebook'
        task_type: 'research_link', 'crawl_link', etc.
        payload: dict with keywords, links, or profile_links
        options: optional dict with max_results, timeout, batch_size
    """
    # Create message
    message = {
        "task_id": str(uuid.uuid4()),
        "platform": platform,
        "task_type": task_type,
        "payload": payload,
        "options": options or {},
        "created_at": datetime.utcnow().isoformat() + "Z"
    }
    
    # Routing key: {platform}.{action}.{type}
    routing_key = f"{platform}.{task_type}"
    
    # Publish to exchange
    channel.basic_publish(
        exchange='social.direct',
        routing_key=routing_key,
        body=json.dumps(message),
        properties=pika.BasicProperties(
            content_type='application/json',
            delivery_mode=2  # persistent
        )
    )
    
    return message["task_id"]

# Example usage
task_id = publish_crawl_task(
    platform="youtube",
    task_type="research_link",
    payload={"keywords": ["AI tutorial", "machine learning"]},
    options={"max_results": 50, "batch_size": 25}
)
```

#### Consuming Results

```python
def on_result_message(ch, method, properties, body):
    """
    Handle result message from crawler
    """
    result = json.loads(body)
    
    task_id = result["task_id"]
    platform = result["platform"]
    status = result["status"]
    
    if status == "success":
        # Download batches from MinIO
        for batch in result["data"]["batches"]:
            minio_path = batch["minio_path"]
            download_and_process_batch(minio_path)
    elif status == "error":
        # Handle error
        print(f"Task {task_id} failed: {result['error']}")
    
    # Acknowledge message
    ch.basic_ack(delivery_tag=method.delivery_tag)

# Subscribe to results
channel.basic_consume(
    queue=f'ingest.results.{platform}',
    on_message_callback=on_result_message
)
```

---

### Crawler Service

#### Consuming Tasks

```python
def on_task_message(ch, method, properties, body):
    """
    Handle crawl task from ingest-social
    """
    task = json.loads(body)
    
    task_id = task["task_id"]
    task_type = task["task_type"]
    payload = task["payload"]
    options = task.get("options", {})
    
    try:
        # Execute crawl
        results = crawl_based_on_task_type(task_type, payload, options)
        
        # Save to MinIO in batches
        batch_info = save_batches_to_minio(task_id, results, options.get("batch_size", 50))
        
        # Publish result
        publish_result(task_id, "success", batch_info)
        
        # Acknowledge
        ch.basic_ack(delivery_tag=method.delivery_tag)
        
    except Exception as e:
        # Publish error result
        publish_result(task_id, "error", None, error=str(e))
        ch.basic_nack(delivery_tag=method.delivery_tag, requeue=False)

# Subscribe to task queue
channel.basic_qos(prefetch_count=1)
channel.basic_consume(
    queue=f'{platform}.{task_type}',
    on_message_callback=on_task_message
)
```

#### Saving Batches to MinIO

```python
from minio import Minio
import json

def save_batches_to_minio(task_id, items, batch_size):
    """
    Save crawled items to MinIO in batches
    
    Returns: list of batch info dicts
    """
    minio_client = Minio(
        "minio:9000",
        access_key=os.getenv("MINIO_ACCESS_KEY"),
        secret_key=os.getenv("MINIO_SECRET_KEY"),
        secure=False
    )
    
    bucket = "crawler-results"
    platform = get_current_platform()
    date = datetime.utcnow().strftime("%Y-%m-%d")
    
    batch_infos = []
    
    # Split into batches
    for i, batch_items in enumerate(chunk_list(items, batch_size), 1):
        batch_id = f"batch-{i:03d}"
        
        # Create batch object
        batch_data = {
            "batch_id": batch_id,
            "task_id": task_id,
            "platform": platform,
            "batch_index": i,
            "items": batch_items,
            "crawled_at": datetime.utcnow().isoformat() + "Z"
        }
        
        # Upload to MinIO
        minio_path = f"{platform}/{date}/{task_id}/batch_{i}.json"
        batch_json = json.dumps(batch_data, ensure_ascii=False)
        
        minio_client.put_object(
            bucket,
            minio_path,
            io.BytesIO(batch_json.encode('utf-8')),
            len(batch_json),
            content_type='application/json'
        )
        
        # Record batch info
        batch_infos.append({
            "batch_id": batch_id,
            "minio_path": minio_path,
            "minio_url": f"s3://{bucket}/{minio_path}",
            "item_count": len(batch_items),
            "size_bytes": len(batch_json),
            "created_at": datetime.utcnow().isoformat() + "Z"
        })
    
    return batch_infos
```

#### Publishing Results

```python
def publish_result(task_id, status, batch_info, error=None):
    """
    Publish crawl result back to ingest-social
    """
    platform = get_current_platform()
    
    message = {
        "task_id": task_id,
        "platform": platform,
        "status": status,
        "data": {
            "minio_bucket": "crawler-results",
            "batches": batch_info or [],
            "total_items": sum(b["item_count"] for b in batch_info) if batch_info else 0,
            "total_batches": len(batch_info) if batch_info else 0,
            "total_size_bytes": sum(b["size_bytes"] for b in batch_info) if batch_info else 0
        },
        "error": error,
        "completed_at": datetime.utcnow().isoformat() + "Z"
    }
    
    # Publish to results exchange
    channel.basic_publish(
        exchange='social.results',
        routing_key=f'results.{platform}',
        body=json.dumps(message),
        properties=pika.BasicProperties(
            content_type='application/json',
            delivery_mode=2
        )
    )
```

---

## Error Handling

### Retry Strategy

| Error Type         | Retry | Max Retries | Backoff     |
| ------------------ | ----- | ----------- | ----------- |
| Network timeout    | ✅ Yes | 3           | Exponential |
| Rate limit         | ✅ Yes | 5           | Fixed (60s) |
| Invalid input      | ❌ No  | 0           | N/A         |
| Server error (5xx) | ✅ Yes | 3           | Exponential |
| Not found (404)    | ❌ No  | 0           | N/A         |

### Dead Letter Queue

**Queue:** `{platform}.{task_type}.dlq`

Messages move to DLQ after:
- Max retries exceeded
- Message expired (TTL)
- Negative acknowledgment without requeue

---

## Monitoring & Metrics

### Key Metrics to Track

#### Ingest-Social
- Tasks published per platform
- Tasks completed success rate
- Average task completion time
- MinIO download failures

#### Crawlers
- Tasks consumed per queue
- Crawl success rate
- Average items per task
- Batch upload failures
- Queue depth

---

## Examples

### Example 1: Research YouTube Links

**Ingest-Social publishes:**
```json
{
  "task_id": "aaa-111-bbb",
  "platform": "youtube",
  "task_type": "research_link",
  "payload": {
    "keywords": ["AI tutorial", "python programming"]
  },
  "options": {
    "max_results": 20
  },
  "created_at": "2026-02-11T10:00:00Z"
}
```
**Routing Key:** `youtube.research_link`

**YouTube Crawler returns:**
```json
{
  "task_id": "aaa-111-bbb",
  "platform": "youtube",
  "status": "success",
  "data": {
    "minio_bucket": "crawler-results",
    "batches": [
      {
        "batch_id": "batch-001",
        "minio_path": "youtube/2026-02-11/aaa-111-bbb/batch_1.json",
        "minio_url": "s3://crawler-results/youtube/2026-02-11/aaa-111-bbb/batch_1.json",
        "item_count": 20,
        "size_bytes": 524288
      }
    ],
    "total_items": 20,
    "total_batches": 1
  },
  "completed_at": "2026-02-11T10:02:30Z"
}
```

---

### Example 2: Crawl TikTok Profile

**Ingest-Social publishes:**
```json
{
  "task_id": "ccc-222-ddd",
  "platform": "tiktok",
  "task_type": "crawl_profile_all",
  "payload": {
    "profile_links": ["https://tiktok.com/@username"]
  },
  "options": {
    "batch_size": 50
  },
  "created_at": "2026-02-11T11:00:00Z"
}
```
**Routing Key:** `tiktok.crawl.profile.all`

**TikTok Crawler returns:**
```json
{
  "task_id": "ccc-222-ddd",
  "platform": "tiktok",
  "status": "success",
  "data": {
    "minio_bucket": "crawler-results",
    "batches": [
      {
        "batch_id": "batch-001",
        "minio_path": "tiktok/2026-02-11/ccc-222-ddd/batch_1.json",
        "minio_url": "s3://crawler-results/tiktok/2026-02-11/ccc-222-ddd/batch_1.json",
        "item_count": 50,
        "size_bytes": 1048576
      },
      {
        "batch_id": "batch-002",
        "minio_path": "tiktok/2026-02-11/ccc-222-ddd/batch_2.json",
        "minio_url": "s3://crawler-results/tiktok/2026-02-11/ccc-222-ddd/batch_2.json",
        "item_count": 32,
        "size_bytes": 671088
      }
    ],
    "total_items": 82,
    "total_batches": 2
  },
  "completed_at": "2026-02-11T11:05:45Z"
}
```

---

## Checklist for Integration

### Ingest-Social Service
- [ ] Connect to RabbitMQ
- [ ] Declare exchange `social.direct`
- [ ] Declare exchange `social.results`
- [ ] Create result queues: `ingest.results.{youtube|tiktok|facebook}`
- [ ] Bind result queues to `social.results` exchange
- [ ] Implement task publishing with proper routing keys
- [ ] Implement result consumption
- [ ] Connect to MinIO
- [ ] Implement batch download from MinIO
- [ ] Implement error handling & retries
- [ ] Add monitoring & logging

### Crawler Services (Each Platform)
- [ ] Connect to RabbitMQ
- [ ] Create task queues for all 6 task types
- [ ] Bind task queues to `social.direct` exchange
- [ ] Implement task consumption with prefetch=1
- [ ] Implement crawling logic for each task type
- [ ] Connect to MinIO
- [ ] Implement batch upload to MinIO
- [ ] Implement result publishing
- [ ] Implement error handling & retries
- [ ] Add monitoring & logging

---

## Contact & Support

**Document Owner:** DevOps Team  
**Last Updated:** 2026-02-11  
**Version:** 1.0
