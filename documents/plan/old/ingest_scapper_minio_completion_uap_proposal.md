# Ingest Proposal: `scapper -> MinIO -> completion envelope -> ingest -> UAP`

**Status:** Companion ingest-side proposal  
**Date:** 2026-03-08  
**Audience:** ingest runtime, scheduler, parser, production operations

## 0. Canonical References

This document keeps ingest-side implementation implications and sequencing.

Canonical wire/runtime contract now lives in:

- RabbitMQ wire contract: `/mnt/f/SMAP_v2/scapper-srv/RABBITMQ.md`
- Shared runtime boundary, MinIO, idempotency: `/mnt/f/SMAP_v2/ingest-srv/documents/plan/scapper_ingest_shared_runtime_contract_proposal.md`

## 1. Goal

Lock the production runtime contract between `ingest-srv` and `scapper-srv` so Phase 3 and Phase 4 can be implemented without redesigning the crawler handoff later.

Chosen direction:

- `ingest-srv` publishes crawl tasks to RabbitMQ
- `scapper-srv` executes the task
- `scapper-srv` uploads the raw result to MinIO
- `scapper-srv` publishes a small completion envelope to RabbitMQ queue `ingest_task_completions`
- `ingest-srv` resolves the envelope by `task_id`
- `ingest-srv` creates one `raw_batch` per successful `external_task`
- `ingest-srv` parses raw into UAP and publishes to Kafka for analysis

For grouped TikTok keyword targets, one target dispatch may create many `external_tasks`; each successful task still maps to exactly one `raw_batch`.

This document does **not** replace the canonical wire/runtime docs above. It explains how ingest should consume that contract.

## 2. Runtime Boundary

### 2.1 Control plane

Owned by `ingest-srv`:

- datasource lifecycle
- crawl target management
- dry run
- scheduler decisions
- support-flag gating

### 2.2 Execution plane

Owned jointly:

- `ingest-srv` creates `scheduled_jobs` and `external_tasks`
- `ingest-srv` publishes RabbitMQ tasks
- `scapper-srv` executes crawl
- `scapper-srv` uploads raw artifact to MinIO
- `scapper-srv` publishes completion envelope to `ingest_task_completions`
- `ingest-srv` creates `raw_batches`
- `ingest-srv` parses raw and emits UAP to Kafka

## 3. Outbound Task Dispatch

`ingest-srv` remains the source of truth for execution lineage.

Each published task must already have:

- `source_id`
- `project_id`
- optional `target_id`
- optional `scheduled_job_id`
- `external_task.id`
- `external_task.task_id`
- `platform`
- `task_type`
- `request_payload`

RabbitMQ request payload shape:

```json
{
  "task_id": "uuid-v4",
  "action": "search|post_detail|comments|summary|...",
  "params": {},
  "created_at": "2026-03-07T00:00:00Z"
}
```

`task_id` is the correlation key between outbound task and inbound completion envelope.

## 4. Completion Envelope From `scapper-srv`

`scapper-srv` must publish a **small message** to queue `ingest_task_completions` after raw upload succeeds.

### 4.1 Envelope shape

```json
{
  "task_id": "uuid-v4",
  "queue": "tiktok_tasks|facebook_tasks|youtube_tasks",
  "platform": "tiktok|facebook|youtube",
  "action": "string",
  "status": "success|error",
  "completed_at": "2026-03-07T00:00:15Z",
  "storage_bucket": "ingest-raw",
  "storage_path": "crawl-raw/tiktok/post_detail/2026/03/07/3f5d...json",
  "batch_id": "raw-tiktok-post_detail-3f5d...",
  "checksum": "sha256:...",
  "item_count": 2,
  "error": null,
  "metadata": {
    "crawler_version": "string",
    "duration_ms": 15234,
    "content_type": "application/json",
    "size_bytes": 1048576,
    "logical_run_id": "uuid-v4",
    "source_id": "optional echo",
    "target_id": "optional echo"
  }
}
```

### 4.2 Envelope rules

- `task_id` is mandatory in all cases
- `status=success` requires:
  - `storage_bucket`
  - `storage_path`
  - `batch_id`
- `status=error` may omit MinIO fields
- completion envelope must be published **only after** raw upload succeeds
- duplicate completion envelopes with the same `task_id` must be treated as duplicate delivery, not a new task result

## 5. MinIO Object Model

### 5.1 Raw crawl artifact

Recommended object path:

`crawl-raw/{platform}/{action}/{yyyy}/{mm}/{dd}/{task_id}.json`

Examples:

- `crawl-raw/tiktok/full_flow/2026/03/07/1111-2222.json`
- `crawl-raw/facebook/post_detail/2026/03/07/3333-4444.json`

### 5.2 Optional parsed artifact

Recommended path for parsed UAP batch artifact:

`uap-batches/{project_id}/{source_id}/{batch_id}/part-00001.jsonl`

This artifact is optional but recommended for replay, audit, and downstream troubleshooting.
For high-volume batches, parsed output may be chunked into multiple `part-xxxxx.jsonl` files.

### 5.3 Storage rules

- raw object is immutable after successful upload
- one successful `external_task` maps to one raw object
- checksum must be computed before completion publish
- object metadata should include:
  - `task_id`
  - `platform`
  - `action`
  - `batch_id`
  - `logical_run_id` if present

## 6. `raw_batch` Creation Policy

`ingest-srv` creates one `raw_batch` per successful `external_task`.

Minimum `raw_batches` fields:

- `source_id`
- `project_id`
- `external_task_id`
- `batch_id`
- `storage_bucket`
- `storage_path`
- `status = RECEIVED`
- `publish_status = PENDING`
- `checksum`
- `item_count`
- `raw_metadata`

Dedup rules:

- primary raw dedup key: `(source_id, batch_id)`
- duplicate detection fallback: `checksum`
- duplicate completion envelope must not create a second `raw_batch`

## 7. `POST_URL` Orchestration

For `POST_URL`, one logical run creates **two external tasks**:

- `post_detail`
- `comments`

Both tasks share lineage:

- `source_id`
- `target_id`
- `scheduled_job_id` or manual trigger lineage
- `logical_run_id`

Each task has its own:

- `task_id`
- `external_task`
- MinIO raw object
- `raw_batch`

This avoids coupling detail success and comments success into one crawler transaction.

### Why this model

- retry `comments` without re-running `post_detail`
- accept partial logical-run completion
- preserve clear observability
- match current crawler action split

Parser or downstream merge may use:

- `logical_run_id`
- `target_id`
- source post identifier from raw content

## 8. Parser and UAP Handoff

After `raw_batch` creation:

1. `ingest-srv` downloads raw from MinIO
2. parser converts raw records to UAP
3. `ingest-srv` publishes UAP to Kafka
4. optional UAP batch artifact is written to MinIO
5. `raw_batches.publish_status` is updated accordingly

Analysis remains a Kafka consumer of normalized UAP, not a direct consumer of crawler raw.

MinIO remains the system of record for:

- raw crawler artifacts
- optional UAP batch artifacts
- replay and forensic debugging

## 9. Failure Handling and Stability

### 9.1 Completion envelope received but object missing

- mark `external_task` failed or parked
- log `task_id`, `batch_id`, `storage_path`
- do not create `raw_batch`
- route to DLQ/parking path after retry budget is exhausted

### 9.2 Duplicate completion envelope

- lookup by `task_id`
- if already completed and `raw_batch` exists, ignore duplicate
- do not overwrite success/failure blindly

### 9.3 Upload succeeded but completion publish failed

- `scapper-srv` must retry completion publish
- replay must use same `task_id`, `batch_id`, `storage_path`
- ingest dedup rules make this safe

### 9.4 Unknown `task_id`

- do not create orphan `external_task`
- log warning with full envelope
- park or DLQ according to runtime config

### 9.5 Partial logical run success

Example:

- `post_detail` succeeds
- `comments` fails

Expected behavior:

- detail raw batch is accepted
- comments task is failed independently
- source/target runtime state records partial operational truth through per-task status and logs

## 10. Trace Model

Production logs and data lineage must make these keys easy to follow:

- `source_id`
- `project_id`
- `target_id`
- `scheduled_job_id`
- `external_task_id`
- `task_id`
- `batch_id`
- optional `logical_run_id`

Every log line in scheduler, publisher, consumer, parser should include the strongest available subset of these fields.

## 11. Debug and Fallback

### Local/dev

- `scapper-srv/output/*.json` may still exist
- `GET /api/v1/tasks/{task_id}/result` may remain for manual inspection

### Production

- MinIO raw object is the source of truth
- completion envelope is the only accepted runtime completion signal
- local output file is not part of the production contract

## 12. Review Scenarios

This contract should be reviewed against these scenarios:

1. keyword crawl success -> raw upload -> completion envelope -> raw batch creation
2. post-detail success
3. comments success
4. duplicate completion envelope for same `task_id`
5. success envelope but object missing in MinIO
6. upload success, completion publish retried
7. `POST_URL` logical run with `post_detail` success and `comments` failure
8. replay from `raw_batch` without re-calling crawler
9. Kafka UAP publish while keeping MinIO artifact for trace
