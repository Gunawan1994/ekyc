# Video
## Demo Video

[![Demo Video](https://drive.google.com/thumbnail?id=14XlBNi9p6CVabD9rs201JTojefhKv61n&sz=w1000)](https://drive.google.com/file/d/14XlBNi9p6CVabD9rs201JTojefhKv61n/view?usp=sharing)

# eKYC & eKYB Platform

![Go 1.24](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go) ![React 18](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat&logo=postgresql) ![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis) ![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)

## Overview

An intermediary platform for Electronic Know Your Customer (eKYC) and Electronic Know Your Business (eKYB) verification. Companies onboard their customers or business entities through the platform, submit them for identity and compliance verification, and administrators review each submission to approve or reject it.

The platform provides a clear separation between company-facing workflows (submitting and tracking verifications) and admin-facing workflows (reviewing, approving, and rejecting submissions with audit trails).

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend language | Go 1.24 |
| Backend framework | Echo |
| Database | PostgreSQL 16 |
| Cache / session store | Redis 7 |
| Authentication | JWT |
| API documentation | Swagger (OpenAPI 2.0) |
| Frontend framework | React 18 + TypeScript |
| Styling | TailwindCSS |
| Data fetching | TanStack Query |
| HTTP client | Axios |
| Infrastructure | Docker, Docker Compose |

## Quick Start

### Prerequisites

- Docker 24+ and Docker Compose v2

### Run with Docker Compose

```bash
git clone <repo-url>
cd ekyc-platform
cp .env.example .env
# Optional: edit .env to change secrets and configuration
docker compose up --build
```

All services (database, cache, backend, frontend) start together. Database migrations and seed data run automatically on first boot.

### Access Points

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| Swagger UI | http://localhost:8080/swagger/index.html |

### Default Admin Credentials

| Field | Value |
|-------|-------|
| Email | admin@example.com |
| Password | Admin123! |

> Change these credentials immediately in any non-local environment.

## Project Structure

```
ekyc-platform/
├── backend/                  # Go API server
│   ├── cmd/server/           # Entry point (main.go)
│   ├── internal/
│   │   ├── config/           # Environment configuration
│   │   ├── domain/           # Entities and repository interfaces
│   │   ├── usecase/          # Business logic layer
│   │   ├── repository/       # Data access implementations
│   │   └── delivery/         # HTTP handlers and middleware
│   ├── migrations/           # SQL migration files
│   └── seeders/              # Database seed data
├── frontend/                 # React SPA
│   └── src/
│       ├── api/              # Axios API clients
│       ├── auth/             # Authentication context and hooks
│       ├── components/       # Shared UI components
│       └── features/         # Page-level feature modules
├── docs/                     # Architecture and design documentation
└── docker-compose.yaml
```

## API Overview

The REST API is versioned under `/api/v1`. Full interactive documentation is available at the Swagger UI URL above.

| Prefix | Description |
|--------|-------------|
| `/api/v1/auth` | Login, logout, token refresh |
| `/api/v1/dashboard` | Summary statistics and activity feeds |
| `/api/v1/customers` | Customer record management |
| `/api/v1/companies` | Company account management |
| `/api/v1/kyc` | eKYC submission and review workflows |
| `/api/v1/kyb` | eKYB submission and review workflows |

## Roles and Permissions

| Role | Capabilities |
|------|-------------|
| Admin | Full platform access; can approve or reject any KYC/KYB submission; manage companies and users |
| Company | Manage own customer records; submit KYC/KYB verifications; track submission status |
| Customer | View own verification status and submitted data |

## Development (Local)

For running services individually without Docker, configuring environment variables, and setting up a local database, see [docs/local-setup.md](docs/local-setup.md).

## Documentation

| Document | Description |
|----------|-------------|
| [docs/architecture.md](docs/architecture.md) | System architecture and component relationships |
| [docs/local-setup.md](docs/local-setup.md) | Local development setup without Docker |
| [docs/api-design.md](docs/api-design.md) | API conventions and response formats |
| [docs/deployment.md](docs/deployment.md) | Production deployment guide |

## Testing

```bash
# Backend
cd backend && go test ./tests/...

# Frontend
cd frontend && npm test
```

Backend tests cover use cases, repository integrations, and HTTP handler behavior. Frontend tests cover component rendering and API integration logic.

## License

MIT
