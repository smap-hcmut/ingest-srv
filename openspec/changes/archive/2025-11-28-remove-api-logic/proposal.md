# Remove API Logic

## Change ID
`remove-api-logic`

## Status
🟢 **Complete** - Implemented and verified

## Why
The SMAP Dispatcher Service currently maintains two entry points (API and Consumer), but the API entry point is no longer necessary. The service's core responsibility is consuming messages from RabbitMQ and dispatching tasks to platform workers. The HTTP API endpoints (`/health`, `/ready`, `/live`, `/swagger`) are not actively used, and health checks can be handled at the Kubernetes level. Removing the API entry point will simplify the architecture, reduce dependencies (Gin, Swagger, JWT, WebSocket), minimize the attack surface, and make the service easier to maintain.

## What Changes
- Removed `cmd/api/` directory and HTTP server entry point
- Removed `internal/httpserver/` and `internal/middleware/` packages
- Removed HTTP, WebSocket, and JWT configuration from `config/config.go`
- Removed unused environment variables from `template.env`
- Updated `Makefile` to remove API-related targets
- Updated documentation (`README.md`, `openspec/project.md`) to reflect consumer-only architecture
- Additionally cleaned up unused dependencies: MongoDB, MinIO, PostgreSQL, Encrypter, InternalConfig
- Added Discord webhook integration to consumer for error reporting

## Overview
Remove the HTTP API entry point (`cmd/api`) and all related HTTP server logic from the collector service. The service will operate exclusively as a RabbitMQ consumer, simplifying the architecture and removing unnecessary dependencies.

## Motivation

The SMAP Dispatcher Service currently maintains two entry points:
1. **API Server** (`cmd/api`) - HTTP server with health checks and Swagger documentation
2. **Consumer** (`cmd/consumer`) - RabbitMQ consumer for task dispatching

Based on the service's current architecture and operational requirements, the API entry point is no longer necessary because:

- **Primary Function**: The service's core responsibility is consuming messages from RabbitMQ and dispatching tasks to platform-specific workers
- **No External HTTP Traffic**: The service does not receive external HTTP requests; all interactions occur via message queues
- **Kubernetes Health Checks**: Health/readiness/liveness checks can be handled at the infrastructure level (Kubernetes probes on the consumer process)
- **Reduced Complexity**: Removing HTTP server logic eliminates dependencies on Gin, Swagger, JWT, WebSocket, and related middleware
- **Simplified Configuration**: Fewer environment variables and configuration structs to maintain

## Scope

### What Will Be Removed

#### Entry Points
- `cmd/api/` - Entire API entry point directory
  - `cmd/api/main.go` - API server entry point
  - `cmd/api/Dockerfile` - API Docker build configuration

#### Internal Packages
- `internal/httpserver/` - HTTP server implementation
  - `handler.go` - Health check handlers
  - `httpserver.go` - Server lifecycle management
  - `new.go` - Server initialization and configuration
- `internal/middleware/` - Gin middleware (only used by API)
  - `cors.go` - CORS middleware
  - `errors.go` - Error handling middleware
  - `locale.go` - Localization middleware
  - `middleware.go` - Middleware composition
  - `new.go` - Middleware initialization
  - `recovery.go` - Panic recovery middleware

#### Configuration
- `config/config.go` - Remove HTTP/WebSocket/JWT-related config structs:
  - `HTTPServerConfig`
  - `WebSocketConfig`
  - `JWTConfig`
- `env.template` - Remove environment variables:
  - `HOST`, `APP_PORT`, `API_MODE`
  - `WS_*` (WebSocket configuration)
  - `JWT_SECRET`

#### Build & Deployment
- `Makefile` - Remove `run-api` and `swagger` targets
- `Jenkinsfile` - Update to build only consumer image (remove API build/push stages)
- `docker-compose.yml` - Already only contains consumer service (no changes needed)

### What Will Be Preserved

#### Packages in `pkg/`
The following packages will **NOT** be removed even if they were primarily used by the API, as they may have future utility or are used elsewhere:
- `pkg/response/` - Response formatting utilities
- `pkg/scope/` - JWT scope utilities
- `pkg/i18n/` - Internationalization
- `pkg/discord/` - Discord webhooks (used by consumer for error reporting)
- `pkg/encrypter/` - Encryption utilities (used by consumer)
- `pkg/log/` - Logging (used by consumer)
- `pkg/mongo/` - MongoDB client (used by consumer)
- `pkg/rabbitmq/` - RabbitMQ client (used by consumer)
- All other `pkg/` utilities

#### Core Service Logic
- `cmd/consumer/` - Consumer entry point (unchanged)
- `internal/dispatcher/` - Core dispatch logic (unchanged)
- `internal/consumer/` - Consumer server implementation (unchanged)
- `internal/models/` - Domain models (unchanged)

## Impact Assessment

### Breaking Changes
- **API Endpoints**: All HTTP endpoints (`/health`, `/ready`, `/live`, `/swagger`) will be removed
- **Swagger Documentation**: API documentation will no longer be available via HTTP
- **Jenkins Pipeline**: CI/CD pipeline must be updated to remove API image build stages

### Non-Breaking Changes
- **Consumer Functionality**: No impact on core message consumption and task dispatching
- **RabbitMQ Integration**: No changes to message queue interactions
- **Worker Communication**: No impact on communication with YouTube/TikTok workers

### Migration Path
1. **Kubernetes Health Checks**: Update deployment manifests to use `exec` probes instead of HTTP probes (if currently using HTTP)
2. **Monitoring**: Ensure monitoring systems do not depend on HTTP health endpoints
3. **Documentation**: Update README and project documentation to reflect consumer-only architecture

## Dependencies
None - this is a removal change with no external dependencies.

## Alternatives Considered

### Keep Minimal API for Health Checks
**Rejected** - Kubernetes supports exec-based health checks that can verify the consumer process is running. Adding HTTP server complexity solely for health checks is unnecessary overhead.

### Move Health Checks to Consumer
**Rejected** - The consumer's health is implicitly verified by its ability to consume messages. Explicit health check endpoints are not required for a message-driven service.

## Open Questions
None - the scope is well-defined and the change is straightforward.
