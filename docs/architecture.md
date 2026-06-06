# eKYC & eKYB Platform — Architecture

## 1. High-Level Architecture

```mermaid
graph TB
    subgraph Clients
        WebApp[Web App<br/>React / Next.js]
        MobileApp[Mobile App<br/>Flutter / React Native]
        AdminPortal[Admin Portal<br/>React / Next.js]
    end

    subgraph API Gateway Layer
        NGINX[NGINX<br/>Reverse Proxy + Rate Limiter]
    end

    subgraph Backend Services
        AuthSvc[Auth Service<br/>JWT / OAuth2]
        KYCSvc[KYC Service<br/>Verification Engine]
        KYBSvc[KYB Service<br/>Business Verification]
        UserSvc[User Service<br/>Profile Management]
        FileSvc[File Service<br/>Document Storage]
        NotifSvc[Notification Service<br/>Email / SMS / Push]
        AuditSvc[Audit Service<br/>Event Logging]
    end

    subgraph Data Layer
        PG[(PostgreSQL<br/>Primary Database)]
        Redis[(Redis<br/>Cache + Token Store)]
        S3[(Object Storage<br/>S3 / MinIO)]
    end

    subgraph External Integrations
        OCR[OCR Provider<br/>ID Extraction]
        FaceMatch[Face Match<br/>Liveness Detection]
        SanctionDB[Sanctions DB<br/>AML / PEP Check]
        BizRegistry[Business Registry<br/>Company Lookup]
    end

    WebApp --> NGINX
    MobileApp --> NGINX
    AdminPortal --> NGINX

    NGINX --> AuthSvc
    NGINX --> KYCSvc
    NGINX --> KYBSvc
    NGINX --> UserSvc
    NGINX --> FileSvc
    NGINX --> NotifSvc

    AuthSvc --> Redis
    AuthSvc --> PG

    KYCSvc --> PG
    KYCSvc --> Redis
    KYCSvc --> FileSvc
    KYCSvc --> OCR
    KYCSvc --> FaceMatch
    KYCSvc --> SanctionDB
    KYCSvc --> AuditSvc

    KYBSvc --> PG
    KYBSvc --> Redis
    KYBSvc --> FileSvc
    KYBSvc --> BizRegistry
    KYBSvc --> SanctionDB
    KYBSvc --> AuditSvc

    UserSvc --> PG
    FileSvc --> S3
    NotifSvc --> PG
    AuditSvc --> PG
```

### ASCII Fallback (Service Boundaries)

```
+---------------------------------------------------------------+
|                          CLIENTS                              |
|   [Web App]         [Mobile App]         [Admin Portal]       |
+----------------------------+----------------------------------+
                             | HTTPS
+----------------------------v----------------------------------+
|              NGINX (Reverse Proxy / Rate Limiter)             |
+----+------+------+------+------+------+------+---------------+
     |      |      |      |      |      |      |
  Auth   KYC   KYB   User  File  Notif Audit  (Services)
     |      |      |      |      |      |      |
+----v------v------v------v------v------v------v---------------+
|                        DATA LAYER                             |
|           [PostgreSQL]    [Redis]    [S3/MinIO]               |
+---------------------------------------------------------------+
```

---

## 2. Service Boundaries

### 2.1 Frontend

| Component     | Technology       | Responsibility                                              |
|---------------|------------------|-------------------------------------------------------------|
| Web App       | Next.js 14+      | Customer-facing KYC/KYB submission portal                   |
| Admin Portal  | Next.js 14+      | Reviewer dashboard, case management, reporting              |
| Mobile App    | Flutter / RN     | Document capture, liveness check, submission                |

- All frontends communicate exclusively via REST API through NGINX.
- JWT access tokens stored in memory (not localStorage); refresh tokens in `httpOnly` cookies.
- Content Security Policy enforced; no inline scripts.

### 2.2 Backend Services

| Service              | Port  | Responsibility                                                        |
|----------------------|-------|-----------------------------------------------------------------------|
| Auth Service         | 8001  | Login, logout, token issuance, refresh, RBAC enforcement              |
| KYC Service          | 8002  | Individual identity verification workflow                             |
| KYB Service          | 8003  | Business entity verification workflow                                 |
| User Service         | 8004  | User profile, company profile, account management                     |
| File Service         | 8005  | Secure document upload, download, virus scanning                      |
| Notification Service | 8006  | Email, SMS, push notification dispatch                                |
| Audit Service        | 8007  | Immutable audit trail for all state-changing operations               |

All services are stateless and horizontally scalable. Inter-service communication uses internal REST calls with service-to-service JWTs.

### 2.3 PostgreSQL

- Single primary for writes; read replicas optional for reporting.
- Separate schemas per service domain: `auth`, `kyc`, `kyb`, `users`, `files`, `notifications`, `audit`.
- Migrations managed via `golang-migrate` or `flyway`.
- Row-level security (RLS) enabled on sensitive tables.
- PII columns encrypted at rest using `pgcrypto`.

### 2.4 Redis

| Key Pattern                      | Purpose                     | TTL          |
|----------------------------------|-----------------------------|--------------|
| `token:access:{userId}:{jti}`    | Access token store          | 15 minutes   |
| `token:refresh:{userId}:{jti}`   | Refresh token store         | 7 days       |
| `session:{sessionId}`            | Session metadata            | 30 minutes   |
| `cache:kyc:{applicationId}`      | KYC application cache       | 5 minutes    |
| `cache:user:{userId}`            | User profile cache          | 10 minutes   |
| `ratelimit:{ip}:{endpoint}`      | Rate limit counters         | 1 minute     |
| `lock:kyc:{applicationId}`       | Distributed lock for review | 30 seconds   |

---

## 3. API Standards

### 3.1 REST Conventions

- Base URL: `https://api.ekyc-platform.com/v1`
- All endpoints use kebab-case paths: `/kyc-applications`, `/kyb-applications`
- Versioning via URL prefix: `/v1/`, `/v2/`
- HTTP methods follow REST semantics:
  - `GET` — read, idempotent, no body
  - `POST` — create or trigger action
  - `PUT` — full replacement update
  - `PATCH` — partial update
  - `DELETE` — soft or hard delete
- Pagination via query params: `?page=1&limit=20&sort=created_at&order=desc`

### 3.2 JSON Envelope

Every API response uses a consistent envelope:

```json
{
  "success": true,
  "data": { },
  "error": null,
  "meta": {
    "request_id": "req_01J5XQZK3M9VBHF2N7D",
    "timestamp": "2026-06-05T10:00:00Z",
    "version": "1.0.0"
  }
}
```

**Success response:**

```json
{
  "success": true,
  "data": {
    "id": "kyc_01J5XQZK3M9VBHF2N7D",
    "status": "submitted",
    "created_at": "2026-06-05T10:00:00Z"
  },
  "error": null,
  "meta": {
    "request_id": "req_01J5XQZK3M9VBHF2N7D",
    "timestamp": "2026-06-05T10:00:00Z"
  }
}
```

**Error response:**

```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      {
        "field": "id_number",
        "message": "ID number must be 16 digits"
      }
    ]
  },
  "meta": {
    "request_id": "req_01J5XQZK3M9VBHF2N7D",
    "timestamp": "2026-06-05T10:00:00Z"
  }
}
```

**Paginated response:**

```json
{
  "success": true,
  "data": [],
  "error": null,
  "meta": {
    "request_id": "req_01J5XQZK3M9VBHF2N7D",
    "timestamp": "2026-06-05T10:00:00Z",
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 145,
      "total_pages": 8
    }
  }
}
```

### 3.3 Content Negotiation

- Request: `Content-Type: application/json`
- Response: `Content-Type: application/json; charset=utf-8`
- File uploads: `multipart/form-data`

---

## 4. Folder Structure

```
kyc/
+-- cmd/
|   +-- api/                    # Main entry point per service
|   |   +-- auth/main.go
|   |   +-- kyc/main.go
|   |   +-- kyb/main.go
|   +-- migrate/main.go         # Database migration runner
+-- internal/
|   +-- auth/
|   |   +-- handler.go          # HTTP handlers
|   |   +-- service.go          # Business logic
|   |   +-- repository.go       # DB access layer
|   |   +-- model.go            # Domain models
|   |   +-- dto.go              # Request/response DTOs
|   +-- kyc/
|   |   +-- handler.go
|   |   +-- service.go
|   |   +-- repository.go
|   |   +-- model.go
|   |   +-- dto.go
|   |   +-- verifier/           # External verification adapters
|   |       +-- ocr.go
|   |       +-- face_match.go
|   |       +-- sanction.go
|   +-- kyb/
|   |   +-- handler.go
|   |   +-- service.go
|   |   +-- repository.go
|   |   +-- model.go
|   |   +-- dto.go
|   |   +-- verifier/
|   |       +-- biz_registry.go
|   |       +-- sanction.go
|   +-- user/
|   +-- file/
|   +-- notification/
|   +-- audit/
+-- pkg/
|   +-- middleware/
|   |   +-- auth.go             # JWT validation middleware
|   |   +-- rbac.go             # Role-based access control
|   |   +-- ratelimit.go        # Rate limiting middleware
|   |   +-- logger.go           # Request logging middleware
|   |   +-- requestid.go        # Request ID injection
|   +-- jwt/
|   |   +-- token.go            # Token generation/validation
|   |   +-- claims.go           # Custom claims struct
|   +-- redis/
|   |   +-- client.go           # Redis client wrapper
|   |   +-- token_store.go      # Token store operations
|   +-- postgres/
|   |   +-- client.go           # DB connection pool
|   +-- crypto/
|   |   +-- pii.go              # PII encryption helpers
|   +-- response/
|   |   +-- envelope.go         # JSON envelope builders
|   +-- errors/
|   |   +-- codes.go            # Error code constants
|   |   +-- handler.go          # Centralized error handler
|   +-- logger/
|       +-- logger.go           # Structured logger (zerolog/zap)
+-- migrations/
|   +-- 001_create_users.sql
|   +-- 002_create_kyc_applications.sql
+-- docs/
|   +-- architecture.md         # This file
|   +-- system-design.md
|   +-- api/
|       +-- openapi.yaml
+-- deployments/
|   +-- docker-compose.yml
|   +-- docker-compose.prod.yml
|   +-- nginx/
|       +-- nginx.conf
+-- scripts/
|   +-- seed.sh
|   +-- generate-keys.sh
+-- .env.example
+-- Dockerfile
+-- Makefile
+-- README.md
```

---

## 5. Security Architecture

### 5.1 Transport Security

- All traffic over TLS 1.2+ (TLS 1.3 preferred).
- HSTS header: `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload`
- NGINX terminates TLS; internal service communication uses mTLS in production.

### 5.2 Authentication Security

- Access tokens: short-lived (15 min), signed RS256 JWT.
- Refresh tokens: long-lived (7 days), stored in Redis with binding to device fingerprint.
- Token rotation on every refresh; old refresh token invalidated immediately.
- Refresh token stored in `httpOnly`, `Secure`, `SameSite=Strict` cookie.
- Access token stored only in memory (React state / service worker).

### 5.3 Secrets Management

- No secrets in source code or environment files committed to VCS.
- Production secrets via HashiCorp Vault or AWS Secrets Manager.
- `.env.example` documents required variables with no values.
- Startup validation: service fails fast if required secrets are missing.

### 5.4 Data Security

- PII fields (full name, ID number, date of birth, address) encrypted at rest using AES-256 via `pgcrypto`.
- Documents (ID cards, selfies) stored in S3 with server-side encryption (SSE-S3 or SSE-KMS).
- S3 pre-signed URLs with 5-minute expiry for document access.
- Database connections use SSL mode `require` or `verify-full`.

### 5.5 Input Validation

- All request bodies validated against JSON schemas before handler execution.
- File uploads limited to: JPEG, PNG, PDF; max 10 MB per file.
- File content validated via magic bytes — not just the MIME type from the HTTP header.
- Virus scanning via ClamAV on all uploaded files before storage.

### 5.6 Security Headers

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 0
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
Content-Security-Policy: default-src 'self'; ...
```

### 5.7 Audit Trail

- All state-changing operations (create, update, status change) emit an audit event.
- Audit records are immutable — no UPDATE or DELETE on the audit table.
- Audit record includes: `actor_id`, `actor_role`, `action`, `resource_type`, `resource_id`, `before_state`, `after_state`, `ip_address`, `request_id`, `timestamp`.

---

## 6. Authentication Flow

### 6.1 Login Flow

```
Client              NGINX         Auth Service        Redis          PostgreSQL
  |                   |               |                 |                |
  |-- POST /v1/auth/login ----------->|                 |                |
  |   {email, password}               |-- query user ---|--------------->|
  |                                   |<-- user record -|----------------|
  |                                   |                 |                |
  |                                   | verify bcrypt   |                |
  |                                   |                 |                |
  |                                   | generate tokens |                |
  |                                   |  access (RS256, 15min)           |
  |                                   |  refresh (random, 7d)            |
  |                                   |                 |                |
  |                                   |-- SET token:access:{uid}:{jti}-->|
  |                                   |-- SET token:refresh:{uid}:{jti}->|
  |                                   |                 |                |
  |<-- 200 OK {access_token (body)} --|                 |                |
  |    Set-Cookie: refresh_token      |                 |                |
```

### 6.2 Token Refresh Flow

```
Client              Auth Service          Redis
  |                     |                   |
  |-- POST /v1/auth/refresh (cookie) ------>|
  |                     |-- GET token:refresh:{uid}:{jti} -->|
  |                     |<-- token data ----|
  |                     |                   |
  |                     | validate binding  |
  |                     |                   |
  |                     |-- DEL old token -->|
  |                     |-- SET new tokens ->|
  |                     |                   |
  |<-- 200 {access_token, new refresh cookie}
```

### 6.3 Request Authorization Flow

```
Client            Auth Middleware           Service
  |                    |                       |
  |-- GET /v1/kyc ---->|                       |
  |  Bearer {token}    | validate JWT sig      |
  |                    | check Redis exists     |
  |                    | check expiry          |
  |                    | extract claims        |
  |                    |  {userId, role, ...}  |
  |                    |-- inject ctx -------->|
  |                    |                       | handle request
  |<-- 200 OK ---------|<----------------------|
```

### 6.4 Logout Flow

```
Client              Auth Service          Redis
  |                     |                   |
  |-- POST /v1/auth/logout ---------------->|
  |                     |-- DEL token:access -->|
  |                     |-- DEL token:refresh ->|
  |<-- 200 OK (clear cookie) --------------|
```

---

## 7. Authorization Flow (RBAC)

### 7.1 Roles

| Role       | Description                                            |
|------------|--------------------------------------------------------|
| `admin`    | Platform administrator — full access to all resources  |
| `reviewer` | KYC/KYB reviewer — can view, approve, reject cases     |
| `company`  | Business account — submits KYB, manages its customers  |
| `customer` | End user — submits KYC, views own application status   |

### 7.2 Permission Matrix

| Resource                    | admin | reviewer | company | customer |
|-----------------------------|-------|----------|---------|----------|
| List all KYC applications   | yes   | yes      | no      | no       |
| View own KYC application    | yes   | yes      | no      | yes      |
| Submit KYC application      | no    | no       | no      | yes      |
| Approve/Reject KYC          | yes   | yes      | no      | no       |
| List all KYB applications   | yes   | yes      | no      | no       |
| Submit KYB application      | yes   | no       | yes     | no       |
| Approve/Reject KYB          | yes   | yes      | no      | no       |
| Manage users                | yes   | no       | no      | no       |
| View audit logs             | yes   | no       | no      | no       |
| View own audit logs         | yes   | yes      | yes     | yes      |

### 7.3 RBAC Middleware

```go
// Enforced as HTTP middleware — pseudocode
func RequireRole(roles ...string) Middleware {
    return func(ctx Context) error {
        claims := ctx.GetClaims()
        if !contains(roles, claims.Role) {
            return ErrForbidden
        }
        return next(ctx)
    }
}

// Usage
router.GET("/kyc-applications",    RequireRole("admin", "reviewer"), handler.ListAll)
router.GET("/kyc-applications/my", RequireRole("customer"),          handler.ListOwn)
```

### 7.4 Resource Ownership Enforcement

- `customer` role: all queries scoped with `WHERE user_id = $claims.UserID`.
- `company` role: all queries scoped with `WHERE company_id = $claims.CompanyID`.
- `reviewer` / `admin`: no ownership filter applied.

---

## 8. Error Handling Standards

### 8.1 Error Codes

| Code                        | HTTP Status | Description                                      |
|-----------------------------|-------------|--------------------------------------------------|
| `VALIDATION_ERROR`          | 400         | Request body/params failed schema validation     |
| `INVALID_FILE_TYPE`         | 400         | Uploaded file has unsupported type               |
| `FILE_TOO_LARGE`            | 400         | File exceeds maximum size limit                  |
| `UNAUTHENTICATED`           | 401         | No valid token provided                          |
| `TOKEN_EXPIRED`             | 401         | JWT or refresh token has expired                 |
| `TOKEN_REVOKED`             | 401         | Token has been revoked (logout or rotation)      |
| `FORBIDDEN`                 | 403         | Authenticated but lacks permission               |
| `NOT_FOUND`                 | 404         | Resource does not exist                          |
| `CONFLICT`                  | 409         | Duplicate resource or state conflict             |
| `UNPROCESSABLE`             | 422         | Business rule violation                          |
| `RATE_LIMITED`              | 429         | Too many requests                                |
| `INTERNAL_ERROR`            | 500         | Unexpected server-side error                     |
| `SERVICE_UNAVAILABLE`       | 503         | Downstream service (OCR, etc.) unavailable       |
| `KYC_ALREADY_SUBMITTED`     | 409         | Customer already has an active KYC application   |
| `KYC_NOT_REVIEWABLE`        | 422         | Application is not in a reviewable state         |
| `KYB_COMPANY_NOT_FOUND`     | 422         | Company not found in business registry           |
| `DOCUMENT_EXPIRED`          | 422         | Submitted ID document has expired                |
| `LIVENESS_CHECK_FAILED`     | 422         | Face liveness check did not pass                 |
| `SANCTION_MATCH`            | 422         | Subject found on sanctions or PEP list           |

### 8.2 Centralized Error Handler

All handlers return domain errors. A single middleware translates them to HTTP responses:

```go
// Domain error type
type AppError struct {
    Code    string
    Message string
    Details []FieldError
    Cause   error  // internal only — never sent to client
}

// Middleware: AppError.Code -> HTTP status + JSON envelope
func ErrorHandler(err error, ctx Context) {
    var appErr *AppError
    if !errors.As(err, &appErr) {
        appErr = &AppError{
            Code:    "INTERNAL_ERROR",
            Message: "An unexpected error occurred",
        }
    }
    ctx.JSON(statusFor(appErr.Code), envelope.Error(appErr))
    log.Error().
        Str("request_id", ctx.RequestID()).
        Str("code", appErr.Code).
        Err(appErr.Cause).
        Msg("request error")
}
```

### 8.3 Never Expose Internals

- `Cause` (Go error, SQL error, stack trace) is logged only — never returned to the client.
- Error messages are user-friendly, not implementation details.
- 500 responses always return a generic message; detail stays in logs.

---

## 9. Logging Standards

### 9.1 Request ID

Every inbound request receives a unique `request_id` (ULID / UUID v7):

```
X-Request-ID: req_01J5XQZK3M9VBHF2N7D
```

- NGINX generates it if the client does not supply one.
- All services propagate it in downstream/inter-service calls.
- All log entries include `request_id`.
- Response header echoes the request ID for client-side correlation.

### 9.2 Structured Log Format (JSON)

```json
{
  "level": "info",
  "time": "2026-06-05T10:00:00.123Z",
  "service": "kyc-service",
  "request_id": "req_01J5XQZK3M9VBHF2N7D",
  "user_id": "usr_01J5XQZK3M9VBHF2N7D",
  "method": "POST",
  "path": "/v1/kyc-applications",
  "status": 201,
  "latency_ms": 142,
  "ip": "10.0.1.5",
  "message": "KYC application created"
}
```

### 9.3 Log Levels

| Level | When to Use                                                                     |
|-------|---------------------------------------------------------------------------------|
| ERROR | Unexpected failures, unhandled panics, external service failures                |
| WARN  | Recoverable issues, rate limit hits, validation failures on sensitive paths     |
| INFO  | Request/response cycle, significant state changes, service startup/shutdown     |
| DEBUG | Detailed flow tracing, SQL queries — disabled in production                     |

### 9.4 PII in Logs

- Never log raw PII values (ID numbers, full names, DOB, addresses).
- Log only resource IDs and masked representations where needed.
- Log aggregation pipeline (Loki, Elasticsearch) applies configurable retention policies.

### 9.5 Request Logging Middleware

Logs the following at `INFO` level for every request:
`request_id`, `method`, `path`, `query_params` (sanitized), `status`, `latency_ms`, `user_id`, `role`, `ip`.

---

## 10. Redis Strategy

### 10.1 Token Store

Tokens are stored in Redis as the authoritative source for token validity. A structurally valid JWT is rejected if not found in Redis, which enables instant revocation without waiting for token expiry.

```
Key:   token:access:{userId}:{jti}
Value: {"user_id": "...", "role": "...", "company_id": "...", "issued_at": "..."}
TTL:   15 minutes (matches JWT exp)

Key:   token:refresh:{userId}:{jti}
Value: {"user_id": "...", "device_id": "...", "issued_at": "..."}
TTL:   7 days
```

On logout or token rotation: immediately `DEL` both keys.
On password change or suspicious activity: invalidate all tokens for the user via a set index (`user:tokens:{userId}` holding all active JTIs).

### 10.2 Application Cache (Cache-Aside)

```
Key:   cache:kyc:{applicationId}
Value: serialized KYC application JSON
TTL:   5 minutes

Key:   cache:user:{userId}
Value: serialized user profile JSON
TTL:   10 minutes
```

Pattern:
1. Check Redis — return on hit.
2. On miss: query PostgreSQL, write result to Redis with TTL, return.
3. On write (update / delete): immediately `DEL` the key.

### 10.3 Rate Limiting

```
Key:   ratelimit:{strategy}:{identifier}
       e.g. ratelimit:ip:192.168.1.1
            ratelimit:user:usr_xxx:login
Value: counter (INCR)
TTL:   1 minute standard; 15 minutes for login / auth endpoints
```

Sliding window counters implemented with Redis + Lua script for atomic INCR and TTL management.

### 10.4 Distributed Locks

Prevents concurrent review of the same case:

```
Key:   lock:kyc:{applicationId}
Value: {reviewer_id, acquired_at}
TTL:   30 seconds (auto-released if reviewer disconnects)
```

Acquired via `SET key value NX PX 30000`. Heartbeat from the reviewer client extends TTL while the case is open.

### 10.5 Connection Configuration

- Min pool: 5 connections
- Max pool: 50 connections
- Idle timeout: 5 minutes
- Sentinel or Cluster mode in production for high availability

---

## 11. Deployment Architecture

### 11.1 Docker Compose (Development / Staging)

```yaml
# deployments/docker-compose.yml
version: "3.9"

services:
  nginx:
    image: nginx:1.25-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - auth-service
      - kyc-service
      - kyb-service

  auth-service:
    build:
      context: ..
      dockerfile: Dockerfile
      args:
        SERVICE: auth
    environment:
      - DATABASE_URL=postgres://kyc_user:${DB_PASSWORD}@postgres:5432/kyc_db?sslmode=disable
      - REDIS_URL=redis://:${REDIS_PASSWORD}@redis:6379
      - JWT_PRIVATE_KEY_FILE=/run/secrets/jwt_private_key
      - JWT_PUBLIC_KEY_FILE=/run/secrets/jwt_public_key
    secrets:
      - jwt_private_key
      - jwt_public_key
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  kyc-service:
    build:
      context: ..
      dockerfile: Dockerfile
      args:
        SERVICE: kyc
    environment:
      - DATABASE_URL=postgres://kyc_user:${DB_PASSWORD}@postgres:5432/kyc_db?sslmode=disable
      - REDIS_URL=redis://:${REDIS_PASSWORD}@redis:6379
      - OCR_PROVIDER_URL=${OCR_PROVIDER_URL}
      - OCR_API_KEY=${OCR_API_KEY}
      - FACE_MATCH_URL=${FACE_MATCH_URL}
      - SANCTION_API_KEY=${SANCTION_API_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  kyb-service:
    build:
      context: ..
      dockerfile: Dockerfile
      args:
        SERVICE: kyb
    environment:
      - DATABASE_URL=postgres://kyc_user:${DB_PASSWORD}@postgres:5432/kyc_db?sslmode=disable
      - REDIS_URL=redis://:${REDIS_PASSWORD}@redis:6379
      - BIZ_REGISTRY_URL=${BIZ_REGISTRY_URL}
      - SANCTION_API_KEY=${SANCTION_API_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: kyc_db
      POSTGRES_USER: kyc_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kyc_user -d kyc_db"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7.2-alpine
    command: >
      redis-server
      --requirepass ${REDIS_PASSWORD}
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
      --save ""
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ACCESS_KEY}
      MINIO_ROOT_PASSWORD: ${MINIO_SECRET_KEY}
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"

volumes:
  pg_data:
  redis_data:
  minio_data:

secrets:
  jwt_private_key:
    file: ./secrets/jwt_private.pem
  jwt_public_key:
    file: ./secrets/jwt_public.pem
```

### 11.2 Production Deployment Notes

- Services run as containers in Kubernetes (EKS / GKE / self-hosted k8s).
- NGINX replaced by cloud load balancer (ALB / GCP LB) + Ingress controller.
- PostgreSQL uses a managed service (RDS / Cloud SQL) with automated backups and point-in-time recovery.
- Redis uses a managed service (ElastiCache / Memorystore) in Cluster mode.
- Secrets managed via AWS Secrets Manager, GCP Secret Manager, or HashiCorp Vault.
- Container images scanned for CVEs before deployment (Trivy / Snyk).
- Zero-downtime deployments via rolling updates with readiness/liveness probes.
- Horizontal Pod Autoscaling based on CPU/memory and request throughput.

### 11.3 Environment Segregation

| Environment | Purpose                    | Data             |
|-------------|----------------------------|------------------|
| local       | Developer workstation      | Seeded fake data |
| staging     | Integration + QA testing   | Anonymized copy  |
| production  | Live traffic               | Real PII         |

All environments are network-isolated. Production requires MFA for all administrative access. No developer has direct production database access — changes go through the migration pipeline.
