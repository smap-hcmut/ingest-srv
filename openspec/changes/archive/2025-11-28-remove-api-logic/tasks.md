# Tasks: Remove API Logic

## Phase 1: Remove API Entry Point
- [x] Delete `cmd/api/` directory (main.go, Dockerfile)
- [x] Delete `internal/httpserver/` directory (all files)
- [x] Delete `internal/middleware/` directory (all files)

## Phase 2: Clean Up Configuration
- [x] Remove `HTTPServerConfig` struct from `config/config.go`
- [x] Remove `WebSocketConfig` struct from `config/config.go`
- [x] Remove `JWTConfig` struct from `config/config.go`
- [x] Remove HTTP/WebSocket/JWT fields from `Config` struct in `config/config.go`
- [x] Remove HTTP/WebSocket/JWT environment variables from `template.env`

## Phase 3: Update Build Configuration
- [x] Remove `swagger` target from `Makefile`
- [x] Remove `run-api` target from `Makefile`
- [x] Update `Jenkinsfile` to remove API image build stages (file was already deleted)
- [x] Update `Jenkinsfile` to remove API image push stages (file was already deleted)
- [x] Update `Jenkinsfile` to keep only consumer deployment (file was already deleted)

## Phase 4: Update Documentation
- [x] Update `README.md` to remove API server references
- [x] Update `README.md` to reflect consumer-only architecture
- [x] Remove health check endpoint documentation from `README.md`
- [x] Update architecture diagram in `README.md` (if applicable)
- [x] Update `openspec/project.md` to reflect removal of API components

## Phase 5: Verification
- [x] Verify consumer builds successfully: `go build ./cmd/consumer`
- [x] Verify Docker Compose builds: `docker compose build`
- [x] Verify no import errors or missing dependencies
- [x] Test consumer runs and connects to RabbitMQ
- [x] Verify Jenkins pipeline builds consumer image successfully

## Notes
- **Preserve all `pkg/` packages** - Do not remove any utilities even if primarily used by API
- **Consumer unchanged** - No modifications to `cmd/consumer` or `internal/consumer`
- **Core logic intact** - No changes to `internal/dispatcher` or `internal/models`

