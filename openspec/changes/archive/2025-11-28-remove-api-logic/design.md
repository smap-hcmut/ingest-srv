# Design: Remove API Logic

## Architecture Decision

### Current State
The SMAP Dispatcher Service maintains a dual-entry architecture:
```
┌─────────────────────────────────────────────────────────┐
│                  SMAP Dispatcher Service                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐              ┌──────────────┐       │
│  │  cmd/api     │              │ cmd/consumer │       │
│  │  (HTTP)      │              │  (RabbitMQ)  │       │
│  └──────┬───────┘              └──────┬───────┘       │
│         │                             │               │
│         │                             │               │
│  ┌──────▼────────┐            ┌───────▼────────┐     │
│  │ httpserver    │            │   consumer     │     │
│  │ + middleware  │            │   server       │     │
│  └───────────────┘            └───────┬────────┘     │
│                                       │               │
│                               ┌───────▼────────┐     │
│                               │  dispatcher    │     │
│                               │   usecase      │     │
│                               └────────────────┘     │
└─────────────────────────────────────────────────────────┘
```

### Target State
Simplified consumer-only architecture:
```
┌─────────────────────────────────────────────────────────┐
│                  SMAP Dispatcher Service                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│              ┌──────────────┐                          │
│              │ cmd/consumer │                          │
│              │  (RabbitMQ)  │                          │
│              └──────┬───────┘                          │
│                     │                                   │
│              ┌──────▼───────┐                          │
│              │   consumer   │                          │
│              │    server    │                          │
│              └──────┬───────┘                          │
│                     │                                   │
│              ┌──────▼───────┐                          │
│              │  dispatcher  │                          │
│              │   usecase    │                          │
│              └──────────────┘                          │
└─────────────────────────────────────────────────────────┘
```

## Rationale

### Why Remove API Entry Point?

1. **Single Responsibility**: The service's sole purpose is to consume crawl requests from RabbitMQ and dispatch them to platform workers. HTTP endpoints add unnecessary complexity.

2. **No Active Consumers**: The API endpoints (`/health`, `/ready`, `/live`, `/swagger`) are not actively used:
   - Health checks can be performed at the Kubernetes level via exec probes
   - Swagger documentation is not consumed by external services
   - No business logic exposed via HTTP

3. **Reduced Attack Surface**: Removing HTTP server eliminates potential security vulnerabilities from web-facing endpoints.

4. **Simplified Dependencies**: Eliminates need for:
   - Gin web framework
   - Swagger/Swaggo documentation
   - JWT authentication
   - WebSocket configuration
   - CORS middleware
   - HTTP-specific error handling

5. **Operational Simplicity**: 
   - Fewer configuration parameters
   - Single deployment artifact (consumer only)
   - Clearer service boundaries

## Component Analysis

### Components to Remove

#### 1. HTTP Server (`internal/httpserver/`)
**Purpose**: Gin-based HTTP server with health check endpoints and Swagger UI.

**Dependencies**:
- Gin framework
- MongoDB (for readiness checks)
- Discord (for error reporting)
- JWT, Encrypter, WebSocket config

**Removal Impact**: None - functionality not required for message-driven service.

#### 2. Middleware (`internal/middleware/`)
**Purpose**: HTTP middleware for CORS, error handling, locale, recovery.

**Dependencies**:
- Gin framework
- i18n package
- scope package (JWT)
- logger

**Removal Impact**: None - middleware only used by HTTP server.

#### 3. Configuration Structs
**Components**:
- `HTTPServerConfig`: Host, Port, Mode
- `WebSocketConfig`: Buffer sizes, timeouts
- `JWTConfig`: Secret key

**Removal Impact**: Simplifies configuration loading and reduces environment variables.

### Components to Preserve

#### 1. Consumer Entry Point (`cmd/consumer/`)
**Reason**: Core service functionality - must be preserved.

#### 2. Dispatcher Logic (`internal/dispatcher/`)
**Reason**: Business logic for task routing and payload mapping - unchanged.

#### 3. Utility Packages (`pkg/`)
**Reason**: May have future utility or used by consumer:
- `pkg/discord`: Used by consumer for error reporting
- `pkg/log`: Used by consumer for logging
- `pkg/rabbitmq`: Core dependency for consumer
- `pkg/mongo`: Used for persistence
- `pkg/encrypter`: Used for secure configuration
- Others: Preserved for potential future use

## Configuration Changes

### Before
```go
type Config struct {
    HTTPServer HTTPServerConfig
    Logger     LoggerConfig
    Mongo      MongoConfig
    JWT        JWTConfig
    Encrypter  EncrypterConfig
    InternalConfig InternalConfig
    WebSocket  WebSocketConfig
    RabbitMQConfig RabbitMQConfig
    Discord    DiscordConfig
}
```

### After
```go
type Config struct {
    Logger     LoggerConfig
    Mongo      MongoConfig
    Encrypter  EncrypterConfig
    InternalConfig InternalConfig
    RabbitMQConfig RabbitMQConfig
    Discord    DiscordConfig
}
```

## Deployment Changes

### Jenkins Pipeline
**Current**: Builds both API and consumer images (though Jenkinsfile only shows API build).

**After**: Build only consumer image.

**Changes Required**:
- Update `Jenkinsfile` to remove API build stages
- Keep consumer deployment logic
- Update image naming if needed

### Kubernetes Health Checks
**Current**: Likely using HTTP probes (`/health`, `/ready`, `/live`).

**After**: Use exec probes to check consumer process health.

**Example**:
```yaml
livenessProbe:
  exec:
    command:
    - pgrep
    - -f
    - consumer
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  exec:
    command:
    - pgrep
    - -f
    - consumer
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Risk Assessment

### Low Risk
- **Consumer Functionality**: No changes to core message consumption logic
- **RabbitMQ Integration**: No changes to message queue interactions
- **Worker Communication**: No impact on YouTube/TikTok worker communication

### Medium Risk
- **Monitoring**: If monitoring systems depend on HTTP health endpoints, they must be updated
- **Kubernetes Probes**: Deployment manifests must be updated to use exec probes

### Mitigation
- Update Kubernetes deployment manifests before deploying code changes
- Verify consumer process health checks work in staging environment
- Maintain rollback capability via Docker image versioning

## Trade-offs

### Pros
✅ Simplified architecture (single entry point)  
✅ Reduced dependencies (no Gin, Swagger, JWT)  
✅ Fewer configuration parameters  
✅ Smaller attack surface  
✅ Clearer service boundaries  
✅ Easier to maintain and debug  

### Cons
❌ No HTTP-based health checks (must use exec probes)  
❌ No Swagger documentation endpoint (can document in README)  
❌ Cannot easily test service via HTTP (must use RabbitMQ)  

### Conclusion
The benefits significantly outweigh the drawbacks. The service's message-driven nature makes HTTP endpoints unnecessary overhead.
