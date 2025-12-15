# Flow State Redis trong Collector Service

**Cập nhật:** 2025-12-15

---

## 1. Key Schema

```
smap:proj:{projectID}      → Hash chứa state của project
smap:user:{projectID}      → String chứa user_id mapping
```

**TTL mặc định:** 7 ngày

---

## 2. Cấu trúc State (Hash Fields)

| Field    | Type   | Mô tả                                 |
| -------- | ------ | ------------------------------------- |
| `status` | String | INITIALIZING → CRAWLING → DONE/FAILED |
| `total`  | Int64  | Tổng số task cần xử lý                |
| `done`   | Int64  | Số task hoàn thành                    |
| `errors` | Int64  | Số task lỗi                           |

---

## 3. State Transition Diagram

```
┌─────────────────┐
│  INITIALIZING   │  ← InitState() khi nhận project.created event
└────────┬────────┘
         │ UpdateTotal()
         ▼
┌─────────────────┐
│    CRAWLING     │  ← Đang crawl, IncrementDone()/IncrementErrors()
└────────┬────────┘
         │ CheckAndUpdateCompletion()
         │ (done + errors >= total)
         ▼
┌─────────────────┐     ┌─────────────────┐
│      DONE       │ or  │     FAILED      │
└─────────────────┘     └─────────────────┘
```

---

## 4. Flow Chi Tiết

```
Step 1: Project Service gọi POST /projects/:id/execute
        ↓
Step 2: Project Service publish event "project.created" → RabbitMQ
        ↓
Step 3: Collector nhận event
        ├── InitState(projectID)           → HSET status=INITIALIZING, total=0, done=0, errors=0
        └── StoreUserMapping(projectID, userID) → SET smap:user:{projectID} = userID
        ↓
Step 4: Collector tính tổng số tasks cần dispatch
        └── UpdateTotal(projectID, total)  → HSET total={n}, status=CRAWLING
        ↓
Step 5: Collector dispatch tasks → Crawler Workers
        ↓
Step 6: Crawler trả kết quả về Collector
        ├── Success → IncrementDone()      → HINCRBY done 1
        └── Failed  → IncrementErrors()    → HINCRBY errors 1
        ↓
Step 7: Sau mỗi result, check completion
        └── CheckAndUpdateCompletion()
            ├── if (done + errors >= total && total > 0)
            │   └── UpdateStatus(DONE)     → HSET status=DONE
            └── else: continue
```

---

## 5. Ví dụ Redis Commands

```redis
# Step 3: Init state
HSET smap:proj:proj_abc status "INITIALIZING"
HSET smap:proj:proj_abc total 0
HSET smap:proj:proj_abc done 0
HSET smap:proj:proj_abc errors 0
EXPIRE smap:proj:proj_abc 604800   # 7 days

SET smap:user:proj_abc "user_123"
EXPIRE smap:user:proj_abc 604800

# Step 4: Update total (10 tasks)
HSET smap:proj:proj_abc total 10
HSET smap:proj:proj_abc status "CRAWLING"

# Step 6: Crawler results
HINCRBY smap:proj:proj_abc done 1    # task 1 success
HINCRBY smap:proj:proj_abc done 1    # task 2 success
HINCRBY smap:proj:proj_abc errors 1  # task 3 failed
...

# Step 7: Check completion (done=9, errors=1, total=10)
# 9 + 1 >= 10 → Complete!
HSET smap:proj:proj_abc status "DONE"
```

---

## 6. Completion Logic

```go
// IsComplete() trong models/event.go
func (s *ProjectState) IsComplete() bool {
    return s.Total > 0 && (s.Done + s.Errors) >= s.Total
}
```

Project được coi là **complete** khi:

- `total > 0` (đã set tổng số tasks)
- `done + errors >= total` (tất cả tasks đã có kết quả, dù success hay fail)

---

## 7. Kết hợp với Webhook

Sau mỗi lần update state, Collector gọi webhook về Project Service:

```
IncrementDone()/IncrementErrors()
        ↓
GetState() → lấy state hiện tại
        ↓
webhookUC.NotifyProgress() → POST /internal/progress/callback
        ↓
CheckAndUpdateCompletion()
        ↓ (nếu complete)
webhookUC.NotifyCompletion() → POST /internal/progress/callback (status=DONE)
```
