# Testing Guide — eKYC Platform

## Overview

The platform uses a two-layer testing strategy that keeps each layer fast and independently runnable.

| Layer | Tooling | Runs against |
|-------|---------|-------------|
| Backend unit tests | `go test` + testify/mock | Pure Go; no external services |
| Backend integration tests | `go test` + testcontainers-go | Real Postgres + Redis in containers |
| Frontend component tests | Vitest + React Testing Library | jsdom (in-process) |
| Frontend API integration tests | Vitest + MSW (Mock Service Worker) | Intercepted HTTP in jsdom |

Coverage target for all layers: **80% lines, branches, functions, and statements minimum**.

---

## Backend — Unit Tests

### What they test

Unit tests isolate each usecase by substituting all repository dependencies with
`testify/mock` doubles. No database or Redis instance is needed.

Current test file: `backend/tests/unit/auth_usecase_test.go`

Scenarios covered:

| Test | What is asserted |
|------|-----------------|
| `TestLogin_Success` | Valid credentials return access + refresh token pair; token stored in mock Redis; audit log written |
| `TestLogin_WrongPassword` | Returns `ErrInvalidCredentials`; `StoreRefreshToken` never called |
| `TestLogin_UserNotFound` | Returns `ErrInvalidCredentials` (not `ErrNotFound`); prevents user enumeration |
| `TestLogin_InactiveUser` | Returns `ErrInvalidCredentials`; inactive account status not leaked |
| `TestRefreshToken_Success` | Token rotation: old token revoked, new pair issued, rotated token differs from original |
| `TestRefreshToken_Revoked` | Returns `ErrTokenRevoked`; new pair not issued |
| `TestLogout_Success` | Specific refresh token revoked by JTI; audit log written |

### How to run

```bash
cd backend
go test ./tests/unit/... -v -race
```

Run with coverage:

```bash
go test ./tests/unit/... -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Mock strategy

Mocks are hand-written structs that embed `mock.Mock` from `github.com/stretchr/testify/mock`.
Each test follows the Arrange-Act-Assert pattern:

1. Create mock instances and set expectations with `.On(...)`.
2. Call the usecase method under test.
3. Assert return values and verify expectations with `.AssertExpectations(t)`.

The JWT manager is not mocked — a real `jwtpkg.Manager` is used with a test-only
secret and short expiries (`15m` access, `7d` refresh). This keeps token parsing
behaviour correct without coupling tests to implementation details of JWT signing.

---

## Backend — Integration Tests

### Prerequisites

Integration tests spin up real containers using
[testcontainers-go](https://golang.testcontainers.org/). Docker must be running on
the host.

Required:
- Docker Engine 24+ (or Docker Desktop)
- `go test` 1.24+

### What they test

Integration tests verify the full stack from usecase through repository to the
real database and cache. They cover:

- SQL query correctness (pagination, filtering, soft-deletes)
- Postgres constraint enforcement (unique email, FK integrity)
- Redis token lifecycle (store, validate, revoke, TTL expiry)
- Migration idempotency

### Test container setup pattern

```go
package integration

import (
    "context"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
    "github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(t *testing.T) string {
    t.Helper()
    ctx := context.Background()

    pgContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("ekyc_test"),
        postgres.WithUsername("ekyc_user"),
        postgres.WithPassword("ekyc_pass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections"),
        ),
    )
    if err != nil {
        t.Fatalf("start postgres container: %v", err)
    }
    t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatalf("get connection string: %v", err)
    }
    return connStr
}

func setupRedis(t *testing.T) string {
    t.Helper()
    ctx := context.Background()

    redisContainer, err := redis.Run(ctx, "redis:7-alpine")
    if err != nil {
        t.Fatalf("start redis container: %v", err)
    }
    t.Cleanup(func() { _ = redisContainer.Terminate(ctx) })

    addr, err := redisContainer.Endpoint(ctx, "")
    if err != nil {
        t.Fatalf("get redis endpoint: %v", err)
    }
    return addr
}
```

Apply migrations inside the test before each suite:

```go
func applyMigrations(t *testing.T, dsn string) {
    t.Helper()
    m, err := migrate.New("file://../../migrations", "postgres://"+dsn)
    if err != nil {
        t.Fatalf("create migrator: %v", err)
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        t.Fatalf("run migrations: %v", err)
    }
}
```

### How to run

```bash
cd backend
go test ./tests/integration/... -v -race -timeout 120s
```

The `-timeout 120s` flag is recommended because container startup adds latency on
cold runs.

### Adding new integration tests

Place files under `backend/tests/integration/`. Name test files with the pattern
`<domain>_repo_test.go` (e.g., `customer_repo_test.go`). Each file should call
`setupPostgres` and `setupRedis` in `TestMain` or via `t.Cleanup` to avoid
resource leaks.

---

## Frontend — Component Tests

### Tooling

| Package | Version | Role |
|---------|---------|------|
| `vitest` | 2.x | Test runner, watch mode, coverage |
| `@testing-library/react` | 16.x | DOM rendering helpers |
| `@testing-library/jest-dom` | 6.x | Custom DOM matchers |
| `@testing-library/user-event` | 14.x | Realistic user interaction simulation |
| `jsdom` | 24.x | In-process browser environment |
| `msw` | 2.x | HTTP request interception |
| `@vitest/coverage-v8` | 2.x | V8-based coverage reporting |

### Configuration

`frontend/vitest.config.ts` configures:

- Environment: `jsdom`
- Globals: enabled (no need to import `describe`, `it`, `expect`)
- Setup file: `tests/setup.ts`
- Test pattern: `tests/**/*.test.{ts,tsx}`
- Coverage source: `src/**/*.{ts,tsx}` (excludes `main.tsx`, type declarations)
- Coverage thresholds: 80% for lines, functions, branches, statements
- Path alias: `@` resolves to `src/`

### Global test setup (`tests/setup.ts`)

The setup file runs before every test file and:

1. Imports `@testing-library/jest-dom` to extend Vitest matchers.
2. Polyfills `window.matchMedia` and `window.scrollTo` (not implemented in jsdom).
3. Starts the MSW server with `onUnhandledRequest: 'error'` before all tests,
   resets handlers between tests with `server.resetHandlers()`, and closes the
   server after all tests.

### Current component tests

#### `tests/components/Badge.test.tsx`

Tests the `Badge` UI component (source: `src/components/ui/Badge.tsx`).

| Test | What is asserted |
|------|-----------------|
| Pending badge | Amber colour classes applied; label capitalised from status |
| Approved badge | Green colour classes applied |
| Rejected badge | Red colour classes applied |
| Custom label | `label` prop overrides the capitalised status |
| Unknown status | Falls back to gray colour classes |

#### `tests/components/Table.test.tsx`

Tests the generic `Table<T>` component (source: `src/components/ui/Table.tsx`).

| Test | What is asserted |
|------|-----------------|
| Column headers | Rendered from `columns` config |
| Row data | Each cell rendered from `data` array; `cell-{key}` test IDs |
| Custom cell renderer | `render` function in column config used when provided |
| Loading skeleton | 5 skeleton rows x column count; no empty-state shown; `aria-label="loading-row"` |
| Empty state | `emptyMessage` prop or default "No data available" shown when `data=[]` |
| Custom empty message | `emptyMessage` prop overrides default |

### How to run component tests

```bash
cd frontend

# Run once
npm run test

# Watch mode (re-runs on file changes)
npm run test:watch
```

---

## Frontend — API Integration Tests

MSW intercepts all `fetch` calls in jsdom so tests run without a real backend.

### MSW handler setup (`tests/mocks/handlers.ts`)

Default handlers registered for the shared server instance in `tests/mocks/server.ts`:

| Route | Method | Behaviour |
|-------|--------|-----------|
| `/api/v1/auth/login` | POST | Returns mock tokens + user for `admin@example.com` / `Admin123!`; returns 401 for any other credentials |
| `/api/v1/dashboard/stats` | GET | Returns fixed stat counters (1240 customers, 87 companies, 34 pending KYC, etc.) |

Handler groups are exported individually (`authHandlers`, `dashboardHandlers`) so
individual test files can import only what they need, or add per-test overrides
with `server.use(...)`.

### Per-test handler overrides

Use `server.use(...)` inside a test to override the default handler for that test
only. The setup file calls `server.resetHandlers()` in `afterEach`, so overrides
do not leak between tests:

```ts
server.use(
  http.post('/api/v1/auth/login', () =>
    HttpResponse.json(
      { success: false, data: null, message: 'Invalid email or password' },
      { status: 401 },
    ),
  ),
)
```

### Current API integration tests

#### `tests/auth/LoginPage.test.tsx`

Tests the `LoginPage` feature component.

| Test | What is asserted |
|------|-----------------|
| Renders form | Email input, password input, submit button visible; labels associated |
| Empty submit | Validation errors shown for both fields |
| Clear on type | Error message clears when user starts typing in that field |
| Valid submit | No validation errors; API called |
| Successful login | `toast.success` called; `access_token` persisted to `localStorage` |
| Failed login (401) | `toast.error` called with server message |

#### `tests/dashboard/DashboardPage.test.tsx`

Tests the `DashboardPage` feature component with TanStack Query.

A fresh `QueryClient` is created per test with `retry: false` and `staleTime: 0`
to ensure predictable fetch behaviour.

| Test | What is asserted |
|------|-----------------|
| Loading state | Skeleton cards visible; stats grid not yet rendered |
| Successful fetch | 8 stat cards rendered with values from MSW mock |
| Card titles | All 8 human-readable labels present |
| API error (500) | Error state message shown; stats grid absent |

### How to run API integration tests

```bash
cd frontend

# Run once
npm run test

# With coverage report
npm run test:coverage
```

Coverage HTML report is written to `frontend/coverage/index.html`.

---

## Running All Tests

### Backend (from `backend/`)

```bash
# Unit tests only (no Docker required)
go test ./tests/unit/... -v -race

# Integration tests (requires Docker)
go test ./tests/integration/... -v -race -timeout 120s

# All backend tests with coverage
go test ./... -v -race -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Frontend (from `frontend/`)

```bash
# All tests, single run
npm run test

# All tests with coverage (enforces 80% thresholds)
npm run test:coverage
```

### Full project (from repo root)

```bash
(cd backend && go test ./... -race) && (cd frontend && npm run test)
```

---

## Coverage Targets

| Layer | Lines | Functions | Branches | Statements |
|-------|-------|-----------|----------|------------|
| Backend | 80%+ | 80%+ | 80%+ | 80%+ |
| Frontend | 80%+ | 80%+ | 80%+ | 80%+ |

Frontend thresholds are enforced by Vitest in `vitest.config.ts`. A build that
falls below any threshold exits with a non-zero status code, making CI failures
visible immediately.

Backend coverage is checked via `go tool cover`. Add a CI step that parses
`go tool cover -func=coverage.out` output and fails the build if any package
falls below 80%.

---

## Adding New Tests

### Backend usecase unit test

1. Add a new `_test.go` file under `backend/tests/unit/`.
2. Define mocks for every interface the usecase depends on.
3. Write one `Test<Name>_<Scenario>` function per behaviour.
4. Run `go test ./tests/unit/... -v -race` to verify.

### Frontend component test

1. Create `tests/components/<ComponentName>.test.tsx`.
2. Import the component from `@/components/...` once the source file exists.
3. Use `render`, `screen`, and `userEvent` from React Testing Library.
4. For components that make API calls, use the shared MSW server with handler
   overrides for error cases.
5. Run `npm run test:watch` for rapid iteration.
