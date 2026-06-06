# ingest-srv Agent Reference

Machine-readable service guide for new AI coding agents. Assumes prior reading of `/Workspaces/smap/AGENT.md` for cross-repo orientation.

---

## 1. Service Shape

### Entrypoint & Deployment

**Main entry:** `/cmd/server/main.go:main()` — monolithic process spawning three goroutines:
- HTTP API server (blocks until shutdown)
- RabbitMQ consumer goroutine (blocks until done)
- Scheduler goroutine (blocks until done)

**Init sequence** (cmd/server/main.go:47–196):
1. Load config from `INGEST_CONFIG_FILE` or `./config/ingest-config.yaml`
2. Initialize logger (Zap)
3. Connect PostgreSQL (fail-fast; required)
4. Connect Redis (fail-fast; required)
5. Connect MinIO (fail-fast; required)
6. Connect Kafka producer (fail-fast; required)
7. Connect RabbitMQ (fail-fast; required)
8. Spawn consumer goroutine
9. Spawn scheduler goroutine
10. Start HTTP server (blocking)

### Environment Variables & Config

**Config file:** `config/config.go` (Viper-driven, env overrides with `_` separator).

**Required env vars** (fail-fast on missing):

| Var | Purpose | Example |
|-----|---------|---------|
| `DATABASE_HOST` | PG host | `localhost` |
| `DATABASE_PORT` | PG port | `5432` |
| `DATABASE_USER` | PG user | `ingest_master` |
| `DATABASE_PASSWORD` | PG password | secret |
| `DATABASE_DBNAME` | PG database | `smap` |
| `RABBITMQ_URL` | RabbitMQ DSN | `amqp://user:pass@rabbit:5672/` |
| `KAFKA_BROKERS` | CSV Kafka brokers | `kafka:9092,kafka:9093` |
| `KAFKA_UAP_TOPIC` | Output topic for UAP | `smap.collector.output` |
| `MINIO_ENDPOINT` | MinIO S3 endpoint | `minio:9000` |
| `MINIO_BUCKET` | Bucket name for raw artifacts | `smap-raw` |
| `MICROSERVICE_PROJECT_BASE_URL` | project-srv base | `http://project-srv:8080/api/v1` |
| `INTERNAL_CONFIG_INTERNAL_KEY` | Service-to-service key | secret |

**Queue names** (RabbitMQ, from shared-libs/constants):
- `tiktok_tasks` — TikTok crawl task requests
- `facebook_tasks` — Facebook crawl task requests
- `youtube_tasks` — YouTube crawl task requests
- `ingest_task_completions` — Crawler completion responses
- `ingest_dryrun_completions` — Dry-run result queue

**Kafka topics**:
- `smap.collector.output` — UAP normalized records (partition key: `uap_id`)

---

## 2. Domain Entities

All entities live in PostgreSQL schema `ingest` (001_create_schema_ingest_v1.sql). 

| Entity | PG Table | Lifecycle States | Key Columns |
|--------|----------|------------------|-------------|
| **DataSource** | `ingest.data_sources` | PENDING→READY→ACTIVE/PAUSED→ARCHIVED | `id`, `project_id`, `source_type`, `status`, `crawl_mode`, `dryrun_status` |
| **CrawlTarget** | `ingest.crawl_targets` | (per-target scheduling, no explicit state machine) | `id`, `data_source_id`, `target_type`, `values`, `crawl_interval_minutes`, `next_crawl_at`, `is_active` |
| **DryrunResult** | `ingest.dryrun_results` | PENDING→RUNNING→SUCCESS/WARNING/FAILED | `id`, `source_id`, `target_id`, `job_id`, `status`, `sample_data` |
| **ScheduledJob** | `ingest.scheduled_jobs` | PENDING→RUNNING→SUCCESS/PARTIAL/FAILED/CANCELLED | `id`, `source_id`, `target_id`, `status`, `trigger_type`, `cron_expr` |
| **ExternalTask** | `ingest.external_tasks` | PENDING→RUNNING→SUCCESS/PARTIAL/FAILED/CANCELLED | `id`, `source_id`, `task_id` (correlation), `platform`, `status` |
| **RawBatch** | `ingest.raw_batches` | RECEIVED→DOWNLOADED→PARSED/FAILED (parse); PENDING→PUBLISHING→SUCCESS/FAILED (publish) | `id`, `source_id`, `external_task_id`, `batch_id` (dedup key), `status`, `publish_status` |
| **CrawlModeChange** | `ingest.crawl_mode_changes` | Audit-only; no state machine | `id`, `source_id`, `from_mode`, `to_mode`, `reason` |

### Key Indexes (Performance)

- `idx_data_sources_project_deleted` — list sources per project
- `idx_data_sources_status_category` — filter by status/category
- `idx_crawl_targets_next_crawl_active` — scheduler's **main query** for due targets
- `idx_scheduled_jobs_status_scheduled_for` — scheduler filter
- `idx_external_tasks_task_id` — completion correlation (RabbitMQ task_id → external_task)
- `idx_raw_batches_external_task_id` — link raw batches to task
- `idx_raw_batches_publish_status_received_desc` — find batches needing publish
- **Unique:** `uq_raw_batches_source_batch` — dedup constraint `(source_id, batch_id)`
- **Unique:** `idx_data_sources_webhook_id_unique` — webhook routing

---

## 3. HTTP API

### Public Routes (Auth required, user-facing)

**Prefix:** `/api/v1`

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| `POST` | `/datasources` | `datasourceHandler.Create()` | Create new source |
| `GET` | `/datasources` | `datasourceHandler.List()` | List sources (paginated) |
| `GET` | `/datasources/:id` | `datasourceHandler.Detail()` | Get source + relations |
| `PUT` | `/datasources/:id` | `datasourceHandler.Update()` | Update config/mapping (not ACTIVE) |
| `DELETE` | `/datasources/:id` | `datasourceHandler.Delete()` | Archive source |
| `POST` | `/datasources/:id/activate` | `datasourceHandler.ActivateDataSource()` | Transition READY→ACTIVE |
| `POST` | `/datasources/:id/pause` | `datasourceHandler.PauseDataSource()` | Transition ACTIVE→PAUSED |
| `POST` | `/datasources/:id/resume` | `datasourceHandler.ResumeDataSource()` | Transition PAUSED→ACTIVE |
| `POST` | `/datasources/:id/targets/keywords` | `datasourceHandler.CreateKeywordTarget()` | Bulk create KEYWORD targets |
| `POST` | `/datasources/:id/targets/profiles` | `datasourceHandler.CreateProfileTarget()` | Bulk create PROFILE targets |
| `POST` | `/datasources/:id/targets/posts` | `datasourceHandler.CreatePostTarget()` | Bulk create POST_URL targets |
| `GET` | `/datasources/:id/targets` | `datasourceHandler.ListTargets()` | List targets of source |
| `GET` | `/datasources/:id/targets/:target_id` | `datasourceHandler.DetailTarget()` | Get one target |
| `PUT` | `/datasources/:id/targets/:target_id` | `datasourceHandler.UpdateTarget()` | Update target (interval, priority) |
| `DELETE` | `/datasources/:id/targets/:target_id` | `datasourceHandler.DeleteTarget()` | Soft delete target |
| `POST` | `/datasources/:id/targets/:target_id/activate` | `datasourceHandler.ActivateTarget()` | Set `is_active=true` |
| `POST` | `/datasources/:id/targets/:target_id/deactivate` | `datasourceHandler.DeactivateTarget()` | Set `is_active=false` |
| `POST` | `/dryrun/trigger` | `dryrunHandler.Trigger()` | Start async dry-run (external queue) |
| `GET` | `/dryrun/:source_id/latest` | `dryrunHandler.GetLatest()` | Get last dry-run result |
| `GET` | `/dryrun/:source_id/history` | `dryrunHandler.ListHistory()` | Paginated dry-run history |

### Internal Routes (InternalAuth via X-Internal-Key, service-to-service)

**Prefix:** `/api/v1/internal`

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| `PUT` | `/datasources/:id/crawl-mode` | `datasourceHandler.UpdateCrawlMode()` | Switch SLEEP/NORMAL/CRISIS + audit |
| `GET` | `/projects/:project_id/activation-readiness` | `datasourceHandler.GetActivationReadiness()` | Check if project ready to activate |
| `POST` | `/projects/:project_id/activate` | `datasourceHandler.Activate()` | Activate all READY sources in project |
| `POST` | `/projects/:project_id/pause` | `datasourceHandler.Pause()` | Pause all ACTIVE sources in project |
| `POST` | `/projects/:project_id/resume` | `datasourceHandler.Resume()` | Resume all PAUSED sources in project |
| `POST` | `/projects/:project_id/crawl-mode` | `datasourceHandler.UpdateProjectCrawlMode()` | Change all sources' crawl mode |
| `POST` | `/datasources/:id/targets/:target_id/dispatch` | `executionHandler.DispatchTarget()` | Manually trigger target crawl |

### Health Endpoints (public)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/health` | Liveness check (always 200) |
| `GET` | `/live` | Kubernetes liveness probe |
| `GET` | `/ready` | Readiness check; 503 if deps missing |

All dependencies checked at `/ready`: PostgreSQL, Redis, MinIO, Kafka, RabbitMQ (fail-fast on any miss).

---

## 4. Dispatch Flow (Project Activation → Crawler Task)

### Entry Point
`POST /api/v1/internal/datasources/:id/targets/:target_id/dispatch` → `executionHandler.DispatchTarget()`

### Flow (internal/execution/usecase/usecase.go:17–134)

1. **Validate context** (validateDispatchContext):
   - Source must be CRAWL category, not ARCHIVED
   - Target must be `is_active=true`
   - Source must have `crawl_mode` set
   
2. **Build dispatch specs** (buildDispatchSpecs, helpers.go:78–200):
   - Parse source type + target type → **specs per keyword/profile**
   - Example: TikTok keyword → multiple `full_flow` actions (one per keyword)
   - Spec params include `limit`, `comment_count`, etc.

3. **Create scheduled_job** (repo.CreateScheduledJob):
   - Insert `scheduled_jobs` row with status=RUNNING
   - Snapshot input: `trigger_type`, `cron_expr`, payload

4. **Fan out external_tasks** (dispatchPrepared, usecase.go:148–220):
   - For each spec, generate `task_id` (UUID)
   - Call `repo.CreateExternalTask()` → insert `external_tasks` row
   - Resolve project's `domain_type_code` (call project-srv)
   - Status=PENDING initially

5. **Publish to RabbitMQ** (dispatchOneSpec, producer):
   - Call `execProducer.PublishDispatch()` 
   - Route to queue: `tiktok_tasks`, `facebook_tasks`, or `youtube_tasks`
   - Body: JSON with task_id, action, params
   - Persistence: amqp.Persistent flag set
   - Update external_task status → RUNNING, set published_at

6. **Return output**:
   - `DispatchTargetOutput`: `{ scheduled_job_id, status, task_count, published_count, failed_count, tasks[] }`
   - If all fail → status=FAILED, finalize job
   - If partial → status=PARTIAL
   - If all succeed → status=RUNNING

### Queue Names & Routing (constants)

- `tiktok_tasks` → TikTok crawler
- `facebook_tasks` → Facebook crawler  
- `youtube_tasks` → YouTube crawler

### Idempotency & Task ID

**Critical:** `task_id` (UUID) is correlation key. Duplicate `task_id` in same dispatch = error.
- RabbitMQ: message acknowledged only after DB update
- Completion consumer: matches completion by `task_id` → finds `external_task` (indexed)
- Dedup on `external_tasks.task_id` UNIQUE constraint

### Known Limits (types.go:89–103)

```go
TikTokFullFlowLimit = 5           // max posts crawled per keyword
TikTokFullFlowCommentCount = 8    // comments per post
FacebookFullFlowLimit = 5         // max posts per keyword
FacebookPageFullFlowCount = 2     // posts crawled from page
YouTubeFullFlowLimit = 5          // max videos per keyword
```

These are **snapshot defaults** sent in task payload. Crawler may override if config allows.

---

## 5. Dry-Run Flow

### Trigger (HTTP)

`POST /api/v1/dryrun/trigger` → `dryrunHandler.Trigger()`

**Input:**
- `source_id` (required)
- `target_id` (required for CRAWL sources, forbidden for PASSIVE)
- `sample_limit` (optional; clamped to 10 default, max 100)

### Flow (internal/dryrun/usecase/usecase.go:16–114)

1. **Validate state**:
   - Source in PENDING or READY (not ACTIVE/PAUSED/etc.)
   - Per-source dedup: check no dry-run already RUNNING for this source

2. **Build spec** (buildDispatchSpec):
   - Route by source_type + target_type
   - Include `sample_limit` in params

3. **Create result** (repo.CreateResult):
   - Insert `dryrun_results` with status=RUNNING, job_id=UUID

4. **Update source**:
   - Set `dryrun_status=RUNNING`, `dryrun_last_result_id`

5. **Publish to RabbitMQ**:
   - Route to platform-specific queue (`tiktok_tasks`, etc.)
   - Action: `dryrun_sample` (not `full_flow`)

6. **Consume completion** (dryrun/delivery/rabbitmq/consumer/consumer.go:24):
   - Listen on `ingest_dryrun_completions` queue
   - Message → HandleDryrunCompletion usecase
   - Update `dryrun_results.status` → SUCCESS/WARNING/FAILED
   - If SUCCESS/WARNING → transition source PENDING→READY
   - If FAILED → source stays PENDING

### Result Storage & Lifetime

- **Store:** `dryrun_results` table
- **Sample data:** `dryrun_results.sample_data` (JSONB, raw crawl output)
- **Warnings:** `dryrun_results.warnings` (JSONB, parsed issues)
- **Lifetime:** Indefinite; no auto-purge. Query by source_id + created_at DESC for history.

---

## 6. Crawler Completion Consumer

**Queue:** `ingest_task_completions` (RabbitMQ)

**Consumer:** `internal/execution/delivery/rabbitmq/consumer/workers.go:handleCompletionWorker()`

### Flow

1. **Receive message** (amqp.Delivery):
   - Unmarshal JSON → `CompletionMessage`
   - Extract: `task_id`, `status`, `storage_bucket`, `storage_path`, `batch_id`, `checksum`, `item_count`, `error`

2. **Lookup external_task**:
   - Query `external_tasks WHERE task_id = :task_id` (indexed)
   - If not found → **Nack (discard)** — task unknown, skip

3. **Create raw_batch** (repo.CreateRawBatch):
   - Insert `raw_batches` row
   - Status: RECEIVED
   - Dedup on `(source_id, batch_id)` — if exists, merge counts

4. **Update external_task**:
   - Status → SUCCESS/FAILED
   - response_received_at = now
   - error_message (if applicable)

5. **Trigger parse** (internal/uap/usecase):
   - If raw_batch status=RECEIVED, download from MinIO
   - Parse raw JSON → UAP records
   - Update batch status → PARSED
   - Publish to Kafka `smap.collector.output`

6. **Ack behavior**:
   - Success: `delivery.Ack(false)` → message removed
   - Discardable error (task not found): `Ack(false)` → skip without retry
   - Transient error: `Nack(false, true)` → requeue

---

## 7. UAP Publish (Kafka Output)

**Topic:** `smap.collector.output` (per Kafka config)

**Producer:** `internal/uap/delivery/kafka/producer/producer.go:Publish()`

### Message Schema

See `documents/resource/input-output/UAP_SPECIFICATION_VNEXT.md` for full spec (POST/COMMENT/REPLY).

**Partition key:** `uap_id` (identity.uap_id) — ensures same post+replies go to same partition.

**Required fields** (per POST):
```json
{
  "domain_type_code": "crypto",        // snapshot from project
  "crawl_keyword": "bia heineken",
  "identity": {
    "uap_id": "tt_p_760990...",       // must be unique
    "origin_id": "760990...",         // platform ID
    "uap_type": "POST",               // POST/COMMENT/REPLY
    "platform": "tiktok",             // tiktok/facebook/youtube
    "url": "https://...",
    "task_id": "...",                 // external_task.task_id for lineage
    "project_id": "project-123"
  },
  "hierarchy": {
    "parent_id": null,
    "root_id": "tt_p_760990...",
    "depth": 0
  },
  "content": {
    "text": "...",
    "title": "...",
    "subtitle": "...",                // normalized transcript (not URL)
    "hashtags": [...],
    "keywords": [...],
    "language": "vi",
    "links": [...]
  },
  "author": { "id": "...", "username": "...", ... },
  "engagement": { "likes": 2578, "comments_count": 180, ... },
  "media": [...],
  "temporal": {
    "posted_at": "2026-02-23T03:44:32Z",
    "updated_at": "2026-02-23T05:10:00Z",
    "ingested_at": "2026-03-24T09:44:00Z"      // snapshot ingest time
  },
  "platform_meta": { "tiktok": { "music_title": "...", ... } }
}
```

**Important:**
- **Do NOT change UAP schema without coordinating analysis-srv consumer** — downstream uses these fields
- Always include `ingested_at`, `content_created_at`, attribution fields
- `domain_type_code` allows analysis to apply domain-specific rules without project service call

---

## 8. Scheduler & Periodic Dispatch

**Entrypoint:** `internal/execution/delivery/job/handlers.go:DispatchDueTargets()`

**Trigger:** Cron expression (default `HeartbeatCron` from config, e.g., `*/2 * * * *` for every 2 min)

### Flow (internal/execution/usecase/cron.go:12–200)

1. **Query due targets** (repo.ListDueTargets):
   ```sql
   SELECT * FROM crawl_targets 
   WHERE next_crawl_at <= :now 
     AND source.status = 'ACTIVE'
   LIMIT :limit  -- default 1
   ORDER BY priority DESC, next_crawl_at ASC
   ```

2. **For each due target**:
   - **Validate scheduled context** (validateScheduledDispatchContext):
     - Source status must be ACTIVE
     - Call project-srv to verify project still ACTIVE
     - If project inactive → skip, don't dispatch
   - **Build specs** (buildDispatchSpecs)
   - **Claim target** (repo.ClaimTarget — atomic, prevents race):
     - CAS update `next_crawl_at = now + effective_interval`
     - If fails → another scheduler instance claimed it, skip
   - **Dispatch** as per dispatch flow above
   - **Update crawl_targets.next_crawl_at** with new interval

3. **Return metrics**:
   - `due_count` — targets queried
   - `claimed_count` — successfully locked
   - `dispatched_count` — tasks published
   - `skipped_race_count` — lost race or validation failed
   - `failed_count` — publish errors

### Known Issues & Leaks

**E2E Test Project Leak:** If `project-srv` returns ACTIVE status for test projects that should be archived, those test project's sources can dispatch forever (scheduler doesn't check if project is test/ephemeral).

**Patch:** Check `project.is_test` flag before allowing scheduled dispatch (requires project-srv contract change).

### Effective Interval Calculation

```
effective_interval = target.crawl_interval_minutes * mode_multiplier
```

Where `mode_multiplier`:
- CRISIS mode: 0.2 (denser, 5x more frequent)
- NORMAL mode: 1.0 (baseline)
- SLEEP mode: 5.0 (sparser, 5x less frequent)

Example: 60-min interval in CRISIS = 12 min next crawl.

---

## 9. Project-Srv Integration (pkg/microservice/project)

**Client:** `pkg/microservice/project/usecase.go`

**Endpoints called** (via HTTP):

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/internal/projects/:project_id` | GET | Fetch project detail + domain_type_code |

**Timeout:** `MICROSERVICE_PROJECT_TIMEOUT_MS` (default if not set)

**Fail-fast behavior:**
- If project-srv unreachable → dispatch fails
- If project status not ACTIVE → skip dispatch
- If domain_type_code unresolvable → dispatch fails

**Retry policy:** Single attempt, no backoff (fail-fast per design).

---

## 10. Persistence & Schema

**Schema name:** `ingest` (PostgreSQL)

**Migrations:** `migrations/` directory

| Migration | Purpose | Key Changes |
|-----------|---------|-------------|
| `001_create_schema_ingest_v1.sql` | Init schema | Create 7 tables, 11 ENUMs, triggers, indexes |
| `002_seed_ingest_defaults.sql` | Seed defaults | (if any) |
| `003_refactor_crawl_targets_to_grouped_values.sql` | Add crawl_targets | Refactor from source.values to per-target.values |
| `004_add_domain_type_to_runtime.sql` | Add domain snapshot | external_tasks.domain_type_code, raw_batches.domain_type_code |

**Dedup constraints:**

- `external_tasks.task_id` — UNIQUE (idempotency on RabbitMQ task_id)
- `raw_batches (source_id, batch_id)` — UNIQUE (dedup raw from same source)

---

## 11. Health & Probes

**Liveness:** `/health` — always returns 200 with `{ status: "healthy" }`

**Readiness:** `/ready` — returns:
- **200** if all deps ready: PostgreSQL, Redis, MinIO, Kafka, RabbitMQ
- **503** if any dep not ready, includes per-dep error details

**Fail-fast startup:**
- If PostgreSQL unavailable at init → fatal
- If Redis unavailable at init → fatal
- If MinIO unavailable at init → fatal
- If Kafka unavailable at init → fatal
- If RabbitMQ unavailable at init → fatal

---

## 12. Known Bugs & Fragile Spots

### 1. Dispatch Limits (types.go:89–103)

Current values:
```go
TikTokFullFlowLimit = 5
FacebookFullFlowLimit = 5
YouTubeFullFlowLimit = 5
```

**Known issue:** These were historically clamped from 50/100 to 12/30 due to crawler load spikes. **Verify current values at runtime** — may be tuned per platform in config (TBD).

### 2. Passive Onboarding (TODO markers)

- `internal/datasource/usecase/datasource.go` — FILE_UPLOAD/WEBHOOK flows incomplete
- Mapping rules validation not enforced
- Recommend skipping PASSIVE source features until completed

### 3. Dry-Run Not Mandatory (model/ingest_types.go:123)

```go
func IsDryrunRequired(sourceType SourceType, targetType TargetType) bool {
    return false  // Dry-run is optional; not blocking
}
```

Source can activate from PENDING directly (dry-run optional for MVP).

### 4. E2E Test Project Dispatch Leak

- Scheduler doesn't verify if project is test/ephemeral
- Test projects can remain ACTIVE indefinitely, dispatching crawl tasks
- **Mitigation:** Check project flags or add `scheduled_job.is_test` marker

### 5. RabbitMQ Dead-Letter Handling

- If message Nack'd 3+ times → goes to dead-letter queue
- No automatic requeue from DLQ (manual intervention needed)
- Monitor RabbitMQ console for `ingest_task_completions.dlx` queue

### 6. Task Retry Cap

- external_tasks: no max retry count enforced
- If crawler keeps failing, task stays in DB forever
- Recommend cleanup job for tasks in FAILED state older than 30 days

### 7. UAP Publish Partial Failures

- Batch publish to Kafka is all-or-nothing
- If 1 of 1000 UAP records fails → entire batch fails
- **Workaround:** Retry full batch; individual records not re-parsed

---

## 13. Dev / Build / Image Tag

### Run Locally

```bash
make run-api       # HTTP server only
make run-consumer  # Consumer workers
make run-scheduler # Scheduler
make run           # All three (needs dev.env)
```

### Build Docker Image

```bash
docker build -t ingest-srv:latest .
docker run -e INGEST_CONFIG_FILE=/etc/smap/ingest-config.yaml \
  -v /path/to/config:/etc/smap \
  ingest-srv:latest
```

### Image Tag Convention

- `latest` → main branch
- `v1.0.0` → git tag
- `{git-branch}` → feature branches (CI)

---

## 14. Critical Pitfalls for New Agents

### Do NOT:
1. **Change UAP schema** without coordinating with analysis-srv consumer team
2. **Remove idempotency** on `external_tasks.task_id` — RabbitMQ guarantees at-least-once delivery
3. **Bypass scheduled-job state checks** — race conditions will corrupt next_crawl_at
4. **Assume project-srv is always available** — implement timeout + fail-fast correctly
5. **Modify crawl_mode defaults** without load testing crawler infrastructure
6. **Delete raw_batches or dryrun_results directly** — soft-delete only, audit trail needed

### DO:
1. **Test dispatch idempotency** — send same task_id twice, verify only one crawl created
2. **Check project status before dispatch** — scheduler already does this; keep it
3. **Validate crawl_interval_minutes > 0** — DB constraint enforced, but verify in code
4. **Log task_id for every dispatch** — enables RabbitMQ audit trail
5. **Use internal/model enums** — SourceStatus, JobStatus, etc.; no string literals
6. **Test `/ready` endpoint** — confirm all deps up before declaring service ready

---

## 15. Quick Reference: File Locations

| Concern | File |
|---------|------|
| Main entry | `cmd/server/main.go` |
| Config types | `config/config.go` |
| Models & enums | `internal/model/*.go` |
| HTTP routes | `internal/datasource/delivery/http/routes.go`, `internal/execution/delivery/http/routes.go` |
| Dispatch logic | `internal/execution/usecase/usecase.go`, `usecase/helpers.go` |
| RabbitMQ producer | `internal/execution/delivery/rabbitmq/producer/producer.go` |
| RabbitMQ consumer | `internal/execution/delivery/rabbitmq/consumer/workers.go` |
| Dry-run flow | `internal/dryrun/usecase/usecase.go` |
| Scheduler job | `internal/execution/delivery/job/handlers.go` |
| UAP publish | `internal/uap/delivery/kafka/producer/producer.go` |
| Project client | `pkg/microservice/project/*.go` |
| Database schema | `migrations/001_create_schema_ingest_v1.sql` |
| UAP spec | `documents/resource/input-output/UAP_SPECIFICATION_VNEXT.md` |
| Health checks | `internal/httpserver/health.go` |

---

## 16. Checklist for Common Tasks

### Adding a New Platform (e.g., TikTok variant)

- [ ] Define `SourceType.TIKTOK_LIVE` in model/ingest_types.go
- [ ] Add `migration/` entry for new source_type enum value
- [ ] Add platform queue constants (if needed)
- [ ] Implement `buildDispatchSpecs` case in execution/usecase/helpers.go
- [ ] Register RabbitMQ exchange/queue in execution/rabbitmq/constants.go
- [ ] Add UAP parser in internal/uap/usecase
- [ ] Update project-srv contract for new domain_type_code (if applicable)
- [ ] Test end-to-end: create source → dispatch → consume → publish

### Changing Crawl Mode Multipliers

- [ ] Update constants in execution/types.go
- [ ] Audit cron schedule (HeartbeatCron) to ensure sufficient precision
- [ ] Test with live crawl targets in staging
- [ ] Check no targets' next_crawl_at calculations overflow

### Debugging RabbitMQ Task Loss

- [ ] Check `external_tasks` table for task_id (indexed query)
- [ ] Verify message in RabbitMQ queue or DLQ (RabbitMQ console)
- [ ] Check consumer logs for Nack/Ack pattern
- [ ] Verify database transaction isolation (serialization anomaly possible)

---

**Document version:** 2026-Q2 | **Last updated:** 2026-06-06
