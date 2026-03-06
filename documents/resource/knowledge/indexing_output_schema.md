# Analytics Indexing Service Output Schema

**Service:** Indexing Internal API  
**Endpoint:** `POST /internal/index`  
**Input:** Batch file (JSONL) containing `AnalyticsPost` objects (as described in `indexing_input_schema.md`)  
**Output:** JSON response describing the indexing result.

---

## 1. Overview

The service processes a batch of analyzed posts, indexes them into the Vector Database (Qdrant) and Relational Database (PostgreSQL), and returns a summary of the operation.

## 2. Response Schema

```json
{
  "batch_id": "batch_20260217_001",
  "total_records": 100,
  "indexed": 95,
  "failed": 3,
  "skipped": 2,
  "duration_ms": 1540,
  "failed_records": [
    {
      "analytics_id": "analytics_uuid_1",
      "error_type": "EMBEDDING_ERROR",
      "error_message": "Failed to generate embedding: timeout"
    }
  ]
}
```

### 2.1 Fields Description

| Field | Type | Description |
| :--- | :--- | :--- |
| `batch_id` | `string` | Unique identifier for the indexing batch. Passed in the request or generated. |
| `total_records` | `integer` | Total number of records found in the input file. |
| `indexed` | `integer` | Number of records successfully indexed (Projected to DB & Qdrant). |
| `failed` | `integer` | Number of records that failed processing due to errors (DB, Embedding, Qdrant). |
| `skipped` | `integer` | Number of records skipped due to validation (spam, bot, low quality, duplicates). |
| `duration_ms` | `integer` | Total processing time in milliseconds. |
| `failed_records` | `array` | List of details for failed records. Omitted if empty. |

### 2.2 Failed Record Object

| Field | Type | Description |
| :--- | :--- | :--- |
| `analytics_id` | `string` | The ID of the `AnalyticsPost` that failed. |
| `error_type` | `string` | Category of error (e.g., `EMBEDDING_ERROR`, `QDRANT_ERROR`, `DB_ERROR`). |
| `error_message` | `string` | Detailed error message. |

## 3. Error Types

- `VALIDATION_ERROR`: Record missing required fields or content too short.
- `DUPLICATE_CONTENT`: Record with same content hash already exists.
- `EMBEDDING_ERROR`: Failed to call Embedding Service.
- `QDRANT_ERROR`: Failed to upsert points to Vector DB.
- `DB_ERROR`: Failed to read/write to PostgreSQL.

## 4. Status Codes

- `200 OK`: Batch processed (even if some records failed).
- `400 Bad Request`: Invalid request format or file URL.
- `500 Internal Server Error`: Critical system failure (e.g., cannot download file).
