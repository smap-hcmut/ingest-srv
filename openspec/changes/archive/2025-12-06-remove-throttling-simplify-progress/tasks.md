# Implementation Tasks

## 1. Remove Throttler Module

- [x] 1.1 Delete `internal/webhook/client/throttler.go`
- [x] 1.2 Delete `internal/webhook/client/throttler_test.go`

## 2. Simplify Webhook Client

- [x] 2.1 Update `internal/webhook/client/http_client.go`
  - Remove `throttler` field from `httpClient` struct
  - Remove `NewThrottler()` call in constructor
  - Simplify `NotifyProgress()` - remove throttle check, always send
  - Remove `NotifyProgressImmediate()` method (merge into `NotifyProgress`)
  - Remove `throttler.Stop()` from `Close()` method
- [x] 2.2 Update `internal/webhook/client_interface.go`
  - Remove `NotifyProgressImmediate()` from `Client` interface
  - Keep only `NotifyProgress()` and `NotifyCompletion()`

## 3. Remove Throttle Configuration

- [x] 3.1 Update `config/config.go`
  - Remove `ThrottleIntervalSec` from `WebhookConfig`
  - Remove `CleanupIntervalMin` from `WebhookConfig`
  - Remove `MaxThrottleEntries` from `WebhookConfig`
  - Remove `GetThrottleInterval()` method
  - Remove `GetCleanupInterval()` method
- [x] 3.2 Update `template.env`
  - Remove `WEBHOOK_THROTTLE_INTERVAL`
  - Remove `WEBHOOK_CLEANUP_INTERVAL`
  - Remove `WEBHOOK_MAX_THROTTLE_ENTRIES`

## 4. Update Webhook Types

- [x] 4.1 Update `internal/webhook/types.go`
  - Remove `ThrottleInterval` from `Config` struct
  - Remove `CleanupInterval` from `Config` struct
  - Remove `MaxThrottleEntries` from `Config` struct

## 5. Refactor cmd/server Separation (Architecture Fix)

- [x] 5.1 Update `cmd/consumer/main.go`
  - Add Redis client initialization (fail fast if error)
  - Add State UseCase initialization
  - Add Webhook Client initialization
  - Pass initialized dependencies to `consumer.New()`
- [x] 5.2 Update `internal/consumer/new.go`
  - Change `Config` struct to receive initialized dependencies:
    - `StateUseCase state.UseCase` (required, not optional)
    - `WebhookClient webhook.Client` (required, not optional)
  - Remove `RedisStateConfig` from Config (moved to cmd)
  - Remove `WebhookConfig` from Config (moved to cmd)
- [x] 5.3 Update `internal/consumer/server.go`
  - Remove ALL Redis initialization logic (if/else blocks)
  - Remove ALL Webhook client initialization logic
  - Simply use `srv.cfg.StateUseCase` and `srv.cfg.WebhookClient` directly
  - No conditional checks - dependencies MUST be provided

## 6. Update Dispatcher UseCase

- [x] 6.1 Update `internal/dispatcher/usecase/project_event_uc.go`
  - Replace `NotifyProgressImmediate()` calls with `NotifyProgress()`
  - Simplify notification logic

## 7. Update Documentation

- [x] 7.1 Update `document/event-drivent.md`
  - Remove Section 7.4 Throttling
  - Update Section 7.5 "When to Call Webhook" - remove throttle references
  - Update Section 14.2 Go Implementation - remove throttle code examples
  - Update Section 14.3 Configuration - remove throttle env vars
- [x] 7.2 Update `README.md`
  - No throttle-related documentation found (skipped)

## 8. Verify & Test

- [x] 8.1 Run `go build ./...` to verify compilation
- [x] 8.2 Run existing tests (remove throttler tests)
- [x] 8.3 Verify webhook client still works correctly
