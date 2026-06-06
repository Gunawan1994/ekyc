# Local Development Setup — eKYC Platform

This guide covers running the platform locally without Docker. If you prefer
containers, use `docker compose up` from the repo root instead.

---

## Prerequisites

| Tool | Minimum version | Install reference |
|------|----------------|-------------------|
| Go | 1.24 | https://go.dev/dl/ |
| Node.js | 20 | https://nodejs.org/ or `nvm install 20` |
| PostgreSQL | 16 | https://www.postgresql.org/download/ |
| Redis | 7 | https://redis.io/docs/install/install-redis/ |
| golang-migrate CLI | latest | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |

Verify installed versions:

```bash
go version          # go1.24.x or later
node --version      # v20.x.x or later
psql --version      # PostgreSQL 16.x
redis-cli --version # Redis server 7.x
migrate --version   # migrate vX.x.x
```

---

## 1. Clone the repository

```bash
git clone <repo-url>
cd kyc
```

---

## 2. Backend setup

### 2a. Configure environment

The backend reads configuration from a `config.env` file in the working directory
(or from environment variables with the same names). A template is provided at
the repo root.

```bash
cd backend
cp ../env.example config.env
```

Open `config.env` and update the values for local development:

```env
# Application
APP_NAME=ekyc-platform
APP_ENV=development
APP_PORT=8080
APP_HOST=0.0.0.0

# Database — point to your local Postgres instance
DB_HOST=localhost
DB_PORT=5432
DB_USER=ekyc_user
DB_PASSWORD=ekyc_pass
DB_NAME=ekyc_db
DB_SSLMODE=disable
DB_MAXCONNS=25
DB_MINCONNS=5

# Redis — point to your local Redis instance
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT — change to a random secret of at least 32 characters
JWT_SECRET=change-this-to-a-secure-random-secret-min-32-chars
JWT_ACCESSTOKENEXPIRY=15m
JWT_REFRESHTOKENEXPIRY=168h

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

Key differences from the Docker defaults:
- `DB_HOST=localhost` (not `postgres`)
- `REDIS_HOST=localhost` (not `redis`)
- `REDIS_PASSWORD=` (empty unless your local Redis has a password configured)

### 2b. Create the database

```bash
# Connect as a superuser and create the role + database
psql -U postgres -c "CREATE USER ekyc_user WITH PASSWORD 'ekyc_pass';"
psql -U postgres -c "CREATE DATABASE ekyc_db OWNER ekyc_user;"
psql -U postgres -d ekyc_db -c "GRANT ALL PRIVILEGES ON DATABASE ekyc_db TO ekyc_user;"
```

Or use `createdb` if your system PATH includes Postgres utilities:

```bash
createdb -U postgres -O ekyc_user ekyc_db
```

### 2c. Run migrations

Migrations live in `backend/migrations/`. Run them from the `backend/` directory:

```bash
migrate \
  -path ./migrations \
  -database "postgres://ekyc_user:ekyc_pass@localhost:5432/ekyc_db?sslmode=disable" \
  up
```

Expected output: a series of migration steps with no errors. The migrations
create the `uuid-ossp` and `pgcrypto` extensions and the following tables:
`roles`, `users`, `companies`, `customers`, `kyc_verifications`,
`kyb_verifications`, `audit_logs`, and `refresh_tokens`.

To roll back all migrations:

```bash
migrate \
  -path ./migrations \
  -database "postgres://ekyc_user:ekyc_pass@localhost:5432/ekyc_db?sslmode=disable" \
  down
```

### 2d. Download Go modules

```bash
go mod download
```

### 2e. Start the backend server

```bash
go run cmd/server/main.go
```

On first run the seeder creates a default admin user. Subsequent runs skip
seeding if the data already exists.

The server listens on `http://localhost:8080`. Verify it is running:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

Swagger UI is available at `http://localhost:8080/swagger/index.html`.

---

## 3. Frontend setup

### 3a. Install dependencies

```bash
cd frontend
npm install
```

### 3b. Configure environment

Create a `.env.local` file (Vite loads this automatically and it is gitignored):

```env
VITE_API_URL=http://localhost:8080
```

### 3c. Start the development server

```bash
npm run dev
```

The frontend is available at `http://localhost:3000`. Both services must be
running for end-to-end flows to work.

### 3d. Build for production (optional)

```bash
npm run build
npm run preview
```

The production build is output to `frontend/dist/`.

---

## 4. Verify the setup

With both services running:

1. Open `http://localhost:3000` — the login page should render.
2. Log in with the seeded admin account (check `backend/seeders/seeder.go` for
   the default credentials).
3. The dashboard should load and display statistics fetched from
   `http://localhost:8080/api/v1/dashboard/stats`.

---

## 5. Recommended VSCode extensions

Install these from the Extensions panel (`Ctrl+Shift+X`) or via the command line:

```bash
code --install-extension golang.go
code --install-extension dbaeumer.vscode-eslint
code --install-extension esbenp.prettier-vscode
code --install-extension bradlc.vscode-tailwindcss
```

| Extension | ID | Purpose |
|-----------|-----|---------|
| Go | `golang.go` | Go language server, debugging, test runner |
| ESLint | `dbaeumer.vscode-eslint` | Inline lint feedback for TypeScript/React |
| Prettier | `esbenp.prettier-vscode` | Auto-format on save |
| Tailwind CSS IntelliSense | `bradlc.vscode-tailwindcss` | Class autocomplete and hover docs |

Recommended workspace `settings.json` additions:

```json
{
  "editor.formatOnSave": true,
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "[go]": {
    "editor.defaultFormatter": "golang.go"
  },
  "go.toolsManagement.autoUpdate": true,
  "tailwindCSS.includeLanguages": {
    "typescript": "html",
    "typescriptreact": "html"
  }
}
```

---

## 6. Running tests locally

See [testing.md](./testing.md) for the full test guide. Quick reference:

```bash
# Backend unit tests (no Docker required)
cd backend && go test ./tests/unit/... -v -race

# Frontend tests
cd frontend && npm run test

# Frontend tests with coverage
cd frontend && npm run test:coverage
```

---

## 7. Common issues

### `DB_USER is required` on startup

The config validator requires `DB_USER`, `DB_PASSWORD`, `DB_NAME`, and
`JWT_SECRET` to be non-empty. Ensure `config.env` exists in the `backend/`
directory and all four values are set.

### `dial tcp [::1]:5432: connect: connection refused`

Postgres is not running. Start it:

```bash
# macOS (Homebrew)
brew services start postgresql@16

# Linux (systemd)
sudo systemctl start postgresql
```

### `NOAUTH Authentication required` from Redis

Your local Redis instance has a password configured. Set `REDIS_PASSWORD` in
`config.env` to match, or disable `requirepass` in `redis.conf` for development.

### `migrate: error: Dirty database version`

A previous migration run failed mid-way. Force the version and re-run:

```bash
migrate \
  -path ./migrations \
  -database "postgres://ekyc_user:ekyc_pass@localhost:5432/ekyc_db?sslmode=disable" \
  force <version_number>

migrate \
  -path ./migrations \
  -database "postgres://ekyc_user:ekyc_pass@localhost:5432/ekyc_db?sslmode=disable" \
  up
```

Replace `<version_number>` with the version shown in the error message.

### Frontend cannot reach the API

Check that:
1. The backend is running on port 8080.
2. `VITE_API_URL=http://localhost:8080` is set in `frontend/.env.local`.
3. The Vite dev server was restarted after adding `.env.local`.
