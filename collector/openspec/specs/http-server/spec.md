# HTTP Server Capability

## REMOVED Requirements

### Requirement: HTTP Health Check Endpoints
The system shall remove HTTP health check endpoints (`/health`, `/ready`, `/live`) as the service operates exclusively as a RabbitMQ consumer and health checks will be performed via Kubernetes exec probes.

#### Scenario: Health Check Endpoint Removed
**Given** the collector service is running as a consumer-only service  
**When** an HTTP request is made to `/health`  
**Then** the endpoint does not exist (connection refused)

#### Scenario: Readiness Check Endpoint Removed
**Given** the collector service is running as a consumer-only service  
**When** an HTTP request is made to `/ready`  
**Then** the endpoint does not exist (connection refused)

#### Scenario: Liveness Check Endpoint Removed
**Given** the collector service is running as a consumer-only service  
**When** an HTTP request is made to `/live`  
**Then** the endpoint does not exist (connection refused)

### Requirement: Swagger Documentation Endpoint
The system shall remove the Swagger documentation endpoint (`/swagger/*`) as API documentation will be maintained in README.md.

#### Scenario: Swagger UI Removed
**Given** the collector service is running as a consumer-only service  
**When** a user navigates to `/swagger/index.html`  
**Then** the endpoint does not exist (connection refused)

### Requirement: HTTP Server Configuration
The system shall remove HTTP server configuration (HOST, APP_PORT, API_MODE environment variables) as no HTTP server will be running.

#### Scenario: HTTP Server Not Started
**Given** the collector service is configured without HTTP server settings  
**When** the service starts  
**Then** no HTTP server is listening on any port  
**And** only the RabbitMQ consumer is active

### Requirement: HTTP Middleware
The system shall remove HTTP middleware (CORS, error recovery, locale, logging) as no HTTP endpoints exist.

#### Scenario: No HTTP Middleware Loaded
**Given** the collector service is running  
**When** the service initializes  
**Then** no HTTP middleware is loaded or configured

### Requirement: JWT Authentication
The system shall remove JWT authentication configuration and logic as no HTTP authentication is required.

#### Scenario: JWT Configuration Removed
**Given** the collector service configuration is loaded  
**When** the service starts  
**Then** no JWT secret key is required  
**And** no JWT validation occurs

### Requirement: WebSocket Configuration
The system shall remove WebSocket configuration as WebSocket support was never implemented and is not required.

#### Scenario: WebSocket Configuration Removed
**Given** the collector service configuration is loaded  
**When** the service starts  
**Then** no WebSocket configuration is loaded  
**And** no WebSocket connections are accepted


