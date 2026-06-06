# Database Design — eKYC Platform

## Overview

The eKYC platform uses **PostgreSQL** as its primary relational database. The schema is managed
through numbered migration files executed by [golang-migrate](https://github.com/golang-migrate/migrate).
Redis is used as a supplementary store for short-lived refresh tokens and response caches; it holds
no long-term business data and is not covered by these migration files.

### PostgreSQL extensions

Migration `000001` enables two extensions that every subsequent migration depends on:

| Extension   | Purpose |
|-------------|---------|
| `uuid-ossp` | `uuid_generate_v4()` — default PK generation for all tables |
| `pgcrypto`  | Server-side cryptographic helpers (available for future use) |

---

## Enum Types

Three custom enum types are defined before the tables that reference them.

### `company_status`

Defined in migration `000004`.

| Value      | Meaning |
|------------|---------|
| `pending`  | Company registered but not yet reviewed |
| `active`   | Company approved and operational |
| `inactive` | Company suspended or deactivated |

### `id_type`

Defined in migration `000005`.

| Value      | Meaning |
|------------|---------|
| `ktp`      | Indonesian national identity card (Kartu Tanda Penduduk) |
| `passport` | International travel passport |
| `sim`      | Indonesian driver's licence (Surat Izin Mengemudi) |

### `verification_status`

Defined in migration `000006`.

| Value      | Meaning |
|------------|---------|
| `pending`  | Verification submitted, awaiting review |
| `approved` | Verification passed review |
| `rejected` | Verification failed review |

---

## Tables

### `roles`

Stores the fixed set of roles that govern access control across the platform.
Three roles are seeded at startup: `admin`, `company`, and `customer`.

| Column        | Type          | Constraints                       | Description |
|---------------|---------------|-----------------------------------|-------------|
| `id`          | `UUID`        | PK, default `uuid_generate_v4()`  | Surrogate primary key |
| `name`        | `VARCHAR(50)` | NOT NULL, UNIQUE                  | Role identifier (e.g. `admin`) |
| `description` | `TEXT`        | NOT NULL, default `''`            | Human-readable description |
| `created_at`  | `TIMESTAMPTZ` | NOT NULL, default `NOW()`         | Record creation timestamp |
| `updated_at`  | `TIMESTAMPTZ` | NOT NULL, default `NOW()`         | Last modification timestamp |

**Indexes**

| Index name       | Columns | Reason |
|------------------|---------|--------|
| `idx_roles_name` | `name`  | Lookup by role name at login and seeding |

---

### `users`

Platform accounts for administrators, company representatives, and customers.
Each user belongs to exactly one role. Soft delete is supported via `deleted_at`.

| Column          | Type           | Constraints                       | Description |
|-----------------|----------------|-----------------------------------|-------------|
| `id`            | `UUID`         | PK, default `uuid_generate_v4()`  | Surrogate primary key |
| `role_id`       | `UUID`         | NOT NULL, FK -> `roles(id)`       | Assigned role |
| `email`         | `VARCHAR(255)` | NOT NULL, UNIQUE                  | Login credential and contact address |
| `password_hash` | `VARCHAR(255)` | NOT NULL                          | bcrypt hash (cost 12) of the user's password |
| `full_name`     | `VARCHAR(255)` | NOT NULL                          | Display name |
| `is_active`     | `BOOLEAN`      | NOT NULL, default `TRUE`          | Flag for enabling or disabling login |
| `created_at`    | `TIMESTAMPTZ`  | NOT NULL, default `NOW()`         | Record creation timestamp |
| `updated_at`    | `TIMESTAMPTZ`  | NOT NULL, default `NOW()`         | Last modification timestamp |
| `deleted_at`    | `TIMESTAMPTZ`  | NULLABLE                          | Soft delete timestamp; NULL means not deleted |

**Indexes**

| Index name             | Columns      | Reason |
|------------------------|--------------|--------|
| `idx_users_email`      | `email`      | Login lookup; also enforces uniqueness |
| `idx_users_role_id`    | `role_id`    | Filter users by role |
| `idx_users_deleted_at` | `deleted_at` | Efficient exclusion of soft-deleted rows |
| `idx_users_is_active`  | `is_active`  | Filter active users in list queries |

---

### `companies`

Represents a business entity registered on the platform. A company is created
by a user with the `company` role. Soft delete is supported.

| Column                | Type             | Constraints                       | Description |
|-----------------------|------------------|-----------------------------------|-------------|
| `id`                  | `UUID`           | PK, default `uuid_generate_v4()`  | Surrogate primary key |
| `user_id`             | `UUID`           | NULLABLE, FK -> `users(id)`       | Owning user account; nullable to allow orphan migration |
| `name`                | `VARCHAR(255)`   | NOT NULL                          | Trading name |
| `registration_number` | `VARCHAR(100)`   | NOT NULL, UNIQUE                  | Official company registration number |
| `address`             | `TEXT`           | NOT NULL, default `''`            | Registered address |
| `phone`               | `VARCHAR(50)`    | NOT NULL, default `''`            | Contact phone |
| `email`               | `VARCHAR(255)`   | NOT NULL, default `''`            | Contact email |
| `status`              | `company_status` | NOT NULL, default `pending`       | Lifecycle status |
| `created_at`          | `TIMESTAMPTZ`    | NOT NULL, default `NOW()`         | Record creation timestamp |
| `updated_at`          | `TIMESTAMPTZ`    | NOT NULL, default `NOW()`         | Last modification timestamp |
| `deleted_at`          | `TIMESTAMPTZ`    | NULLABLE                          | Soft delete timestamp |

**Indexes**

| Index name                          | Columns               | Reason |
|-------------------------------------|-----------------------|--------|
| `idx_companies_user_id`             | `user_id`             | Find all companies owned by a user |
| `idx_companies_status`              | `status`              | Filter by lifecycle status on admin dashboards |
| `idx_companies_registration_number` | `registration_number` | Uniqueness enforcement and direct lookup |
| `idx_companies_deleted_at`          | `deleted_at`          | Efficient exclusion of soft-deleted rows |

---

### `customers`

An individual end-customer registered by a company. Each customer carries an
official identity document reference. Soft delete is supported.

| Column       | Type           | Constraints                       | Description |
|--------------|----------------|-----------------------------------|-------------|
| `id`         | `UUID`         | PK, default `uuid_generate_v4()`  | Surrogate primary key |
| `user_id`    | `UUID`         | NULLABLE, FK -> `users(id)`       | Linked user account when the customer also has a login |
| `company_id` | `UUID`         | NOT NULL, FK -> `companies(id)`   | Owning company |
| `full_name`  | `VARCHAR(255)` | NOT NULL                          | Customer's legal full name |
| `id_number`  | `VARCHAR(100)` | NOT NULL, UNIQUE                  | Identity document number |
| `id_type`    | `id_type`      | NOT NULL, default `ktp`           | Type of identity document |
| `phone`      | `VARCHAR(50)`  | NOT NULL, default `''`            | Contact phone |
| `email`      | `VARCHAR(255)` | NOT NULL, default `''`            | Contact email |
| `address`    | `TEXT`         | NOT NULL, default `''`            | Residential address |
| `created_at` | `TIMESTAMPTZ`  | NOT NULL, default `NOW()`         | Record creation timestamp |
| `updated_at` | `TIMESTAMPTZ`  | NOT NULL, default `NOW()`         | Last modification timestamp |
| `deleted_at` | `TIMESTAMPTZ`  | NULLABLE                          | Soft delete timestamp |

**Indexes**

| Index name                 | Columns      | Reason |
|----------------------------|--------------|--------|
| `idx_customers_company_id` | `company_id` | List all customers belonging to a company |
| `idx_customers_id_number`  | `id_number`  | Uniqueness enforcement and document lookup |
| `idx_customers_deleted_at` | `deleted_at` | Efficient exclusion of soft-deleted rows |

---

### `kyc_verifications`

Records each Know Your Customer identity verification attempt for an individual
customer. Multiple verifications may exist per customer (one per submission cycle).
No soft delete — records are immutable once created.

| Column         | Type                 | Constraints                       | Description |
|----------------|----------------------|-----------------------------------|-------------|
| `id`           | `UUID`               | PK, default `uuid_generate_v4()`  | Surrogate primary key |
| `customer_id`  | `UUID`               | NOT NULL, FK -> `customers(id)`   | Customer being verified |
| `submitted_by` | `UUID`               | NOT NULL, FK -> `users(id)`       | User who submitted the verification |
| `reviewed_by`  | `UUID`               | NULLABLE, FK -> `users(id)`       | Admin user who reviewed; NULL until a decision is made |
| `status`       | `verification_status`| NOT NULL, default `pending`       | Current review state |
| `notes`        | `TEXT`               | NOT NULL, default `''`            | Reviewer notes or rejection reason |
| `submitted_at` | `TIMESTAMPTZ`        | NOT NULL, default `NOW()`         | When the verification was submitted |
| `reviewed_at`  | `TIMESTAMPTZ`        | NULLABLE                          | When the review decision was made |
| `created_at`   | `TIMESTAMPTZ`        | NOT NULL, default `NOW()`         | Record creation timestamp |
| `updated_at`   | `TIMESTAMPTZ`        | NOT NULL, default `NOW()`         | Last modification timestamp |

**Indexes**

| Index name              | Columns              | Reason |
|-------------------------|----------------------|--------|
| `idx_kyc_customer_id`   | `customer_id`        | List all verifications for a customer |
| `idx_kyc_status`        | `status`             | Filter pending/approved/rejected on admin dashboard |
| `idx_kyc_submitted_at`  | `submitted_at DESC`  | Time-ordered listing for review queues |

---

### `kyb_verifications`

Records each Know Your Business verification attempt for a company entity.
Mirrors the structure of `kyc_verifications` but targets companies instead of
individual customers. No soft delete — records are immutable once created.

| Column         | Type                 | Constraints                       | Description |
|----------------|----------------------|-----------------------------------|-------------|
| `id`           | `UUID`               | PK, default `uuid_generate_v4()`  | Surrogate primary key |
| `company_id`   | `UUID`               | NOT NULL, FK -> `companies(id)`   | Company being verified |
| `submitted_by` | `UUID`               | NOT NULL, FK -> `users(id)`       | User who submitted the verification |
| `reviewed_by`  | `UUID`               | NULLABLE, FK -> `users(id)`       | Admin user who reviewed; NULL until a decision is made |
| `status`       | `verification_status`| NOT NULL, default `pending`       | Current review state |
| `notes`        | `TEXT`               | NOT NULL, default `''`            | Reviewer notes or rejection reason |
| `submitted_at` | `TIMESTAMPTZ`        | NOT NULL, default `NOW()`         | When the verification was submitted |
| `reviewed_at`  | `TIMESTAMPTZ`        | NULLABLE                          | When the review decision was made |
| `created_at`   | `TIMESTAMPTZ`        | NOT NULL, default `NOW()`         | Record creation timestamp |
| `updated_at`   | `TIMESTAMPTZ`        | NOT NULL, default `NOW()`         | Last modification timestamp |

**Indexes**

| Index name             | Columns              | Reason |
|------------------------|----------------------|--------|
| `idx_kyb_company_id`   | `company_id`         | List all verifications for a company |
| `idx_kyb_status`       | `status`             | Filter by status on admin dashboard |
| `idx_kyb_submitted_at` | `submitted_at DESC`  | Time-ordered listing for review queues |

---

### `audit_logs`

Immutable, append-only trail of every significant action performed by any actor
in the system. No updates or deletes are performed on this table.

| Column        | Type           | Constraints                       | Description |
|---------------|----------------|-----------------------------------|-------------|
| `id`          | `UUID`         | PK, default `uuid_generate_v4()`  | Surrogate primary key |
| `actor_id`    | `UUID`         | NOT NULL                          | UUID of the user who performed the action |
| `actor_email` | `VARCHAR(255)` | NOT NULL                          | Email snapshot at time of action (denormalized for history) |
| `action`      | `VARCHAR(50)`  | NOT NULL                          | Action kind: `create`, `update`, `delete`, `submit`, `review` |
| `entity_type` | `VARCHAR(100)` | NOT NULL                          | Logical entity name (e.g. `user`, `company`, `kyc_verification`) |
| `entity_id`   | `UUID`         | NOT NULL                          | Primary key of the affected entity |
| `old_value`   | `TEXT`         | NOT NULL, default `''`            | JSON snapshot of the entity before the change |
| `new_value`   | `TEXT`         | NOT NULL, default `''`            | JSON snapshot of the entity after the change |
| `ip_address`  | `VARCHAR(45)`  | NOT NULL, default `''`            | Client IP address (supports IPv6) |
| `user_agent`  | `TEXT`         | NOT NULL, default `''`            | HTTP User-Agent header |
| `created_at`  | `TIMESTAMPTZ`  | NOT NULL, default `NOW()`         | When the event occurred |

> Note: `audit_logs` is defined in the domain model (`internal/domain/audit.go`)
> but does not yet have a numbered migration file. A future migration
> `000007_create_audit_logs` should add this table.

**Recommended indexes**

| Index name               | Columns                   | Reason |
|--------------------------|---------------------------|--------|
| `idx_audit_actor_id`     | `actor_id`                | Audit history for a specific user |
| `idx_audit_entity`       | `entity_type, entity_id`  | All events for a given record |
| `idx_audit_created_at`   | `created_at DESC`         | Time-ordered event stream |

---

### `refresh_tokens` (Redis)

Refresh tokens are **not stored in PostgreSQL**. They are stored in Redis under
the key pattern `refresh:{userID}:{tokenID}` as a SHA-256 hex digest of the raw
token value. The key's TTL equals the token's expiry duration, so revocation by
expiry is automatic.

| Redis field   | Value |
|---------------|-------|
| Key pattern   | `refresh:{userID}:{tokenID}` |
| Stored value  | `hex(sha256(rawToken))` |
| Expiry        | Configured refresh token TTL (set at write time) |

Explicit revocation uses `DEL` on a single key for per-session logout, or an
iterative `SCAN` + `DEL` loop matching `refresh:{userID}:*` to revoke all
sessions for a user without blocking the Redis server.

---

## Foreign Key Relationships

| Child table           | Column         | Parent table  | Parent column | Notes |
|-----------------------|----------------|---------------|---------------|-------|
| `users`               | `role_id`      | `roles`       | `id`          | NOT NULL; role must exist |
| `companies`           | `user_id`      | `users`       | `id`          | NULLABLE; allows company with no owning user |
| `customers`           | `user_id`      | `users`       | `id`          | NULLABLE; set when customer has a login |
| `customers`           | `company_id`   | `companies`   | `id`          | NOT NULL; every customer belongs to a company |
| `kyc_verifications`   | `customer_id`  | `customers`   | `id`          | NOT NULL |
| `kyc_verifications`   | `submitted_by` | `users`       | `id`          | NOT NULL |
| `kyc_verifications`   | `reviewed_by`  | `users`       | `id`          | NULLABLE; set after review |
| `kyb_verifications`   | `company_id`   | `companies`   | `id`          | NOT NULL |
| `kyb_verifications`   | `submitted_by` | `users`       | `id`          | NOT NULL |
| `kyb_verifications`   | `reviewed_by`  | `users`       | `id`          | NULLABLE; set after review |

---

## Soft Delete Strategy

The following tables support soft delete through a nullable `deleted_at` column:

- `users`
- `companies`
- `customers`

**Convention:**

- `deleted_at IS NULL` — the record is live and should appear in normal queries.
- `deleted_at IS NOT NULL` — the record is logically deleted; repository methods
  exclude it by appending `WHERE deleted_at IS NULL`.

A dedicated B-tree index on each `deleted_at` column makes these partial scans
efficient even on large tables.

Verification tables (`kyc_verifications`, `kyb_verifications`) and `audit_logs`
are intentionally excluded from soft delete because they form an immutable
compliance record. Once a verification or audit entry is created it must not be
hidden or removed.

---

## Index Strategy Summary

Indexes are created for three categories of access pattern:

**1. Unique business key lookup**

`users.email`, `companies.registration_number`, `customers.id_number`,
`roles.name`. These back uniqueness constraints and support direct row retrieval
by a known identifier without a sequential scan.

**2. Foreign key traversal**

`users.role_id`, `companies.user_id`, `customers.company_id`,
`kyc_verifications.customer_id`, `kyb_verifications.company_id`. PostgreSQL does
not automatically create indexes on FK columns; these indexes prevent full-table
scans when following parent-to-child relationships or listing children.

**3. Filtered list queries**

`users.is_active`, `companies.status`, `kyc_verifications.status`,
`kyb_verifications.status`, and all `deleted_at` columns. Dashboard and queue
screens filter heavily on these low-cardinality columns. An index reduces the
scan to the matching subset of rows.

Time-ordered indexes on `kyc_verifications.submitted_at DESC` and
`kyb_verifications.submitted_at DESC` support review queues that need the
most-recently submitted records first without an in-memory sort.

---

## Migration Strategy

Migrations are managed by [golang-migrate](https://github.com/golang-migrate/migrate)
using numbered SQL files in `backend/migrations/`.

### File naming convention

```
{sequence}_{description}.up.sql
{sequence}_{description}.down.sql
```

Each migration has a paired `down` file that reverses the change, enabling
deterministic rollback to any previous state.

### Execution order

| Migration file                    | Operation |
|-----------------------------------|-----------|
| `000001_create_extensions`        | Enable `uuid-ossp` and `pgcrypto` |
| `000002_create_roles`             | Create `roles` table |
| `000003_create_users`             | Create `users` table (depends on `roles`) |
| `000004_create_companies`         | Create `company_status` enum and `companies` table |
| `000005_create_customers`         | Create `id_type` enum and `customers` table |
| `000006_create_kyc_kyb`           | Create `verification_status` enum, `kyc_verifications`, and `kyb_verifications` |

### Running migrations

```bash
# Apply all pending migrations
migrate -path backend/migrations -database "$DATABASE_URL" up

# Roll back one step
migrate -path backend/migrations -database "$DATABASE_URL" down 1
```

In the Docker Compose setup the backend entrypoint (`entrypoint.sh`) runs
migrations automatically before starting the server process.

### Seeding

After migrations complete, `backend/seeders/seeder.go` seeds the three system
roles (`admin`, `company`, `customer`) and the default admin user
(`admin@example.com`). All seed operations use `ON CONFLICT ... DO NOTHING`
and are safe to run multiple times without producing duplicate rows.
