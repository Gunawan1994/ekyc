package seeders

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// Seeder holds a database connection and runs idempotent seed operations.
type Seeder struct {
	db *sqlx.DB
}

// NewSeeder constructs a Seeder with the given sqlx database connection.
func NewSeeder(db *sqlx.DB) *Seeder {
	return &Seeder{db: db}
}

// Run executes all seeders in order. Each seeder is idempotent.
func (s *Seeder) Run(ctx context.Context) error {
	log.Println("[seeder] starting seed operations")

	if err := s.seedRoles(ctx); err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}

	if err := s.seedAdminUser(ctx); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	if err := s.seedSuperAdminUser(ctx); err != nil {
		return fmt.Errorf("seed super admin user: %w", err)
	}

	log.Println("[seeder] all seed operations completed successfully")
	return nil
}

// seedRoles inserts all system roles if they do not already exist.
// Includes the original three legacy roles for backward compatibility plus the
// new five-role system. Uses ON CONFLICT (name) DO NOTHING so it is safe to
// run multiple times.
func (s *Seeder) seedRoles(ctx context.Context) error {
	type roleRow struct {
		id          uuid.UUID
		name        string
		description string
	}

	roles := []roleRow{
		// ── Legacy roles (backward compatibility) ──────────────────────────
		{
			id:          uuid.New(),
			name:        "admin",
			description: "System administrator (legacy alias — maps to super_admin behaviour)",
		},
		{
			id:          uuid.New(),
			name:        "company",
			description: "Company representative (legacy alias — maps to company_user behaviour)",
		},
		{
			id:          uuid.New(),
			name:        "customer",
			description: "End customer (legacy)",
		},
		// ── New 5-role system ───────────────────────────────────────────────
		{
			id:          uuid.New(),
			name:        "super_admin",
			description: "Super administrator with full platform access",
		},
		{
			id:          uuid.New(),
			name:        "risk_analyst",
			description: "Analyses risk metrics and views dashboard statistics",
		},
		{
			id:          uuid.New(),
			name:        "compliance_officer",
			description: "Reviews and approves or rejects KYC/KYB verifications",
		},
		{
			id:          uuid.New(),
			name:        "reviewer",
			description: "Reviews KYC/KYB submissions and makes approval decisions",
		},
		{
			id:          uuid.New(),
			name:        "company_user",
			description: "Company user who submits KYB verifications",
		},
	}

	now := time.Now().UTC()

	for _, r := range roles {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO roles (id, name, description, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (name) DO NOTHING`,
			r.id, r.name, r.description, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert role %q: %w", r.name, err)
		}
		log.Printf("[seeder] role %q ensured", r.name)
	}

	return nil
}

// seedAdminUser inserts the default admin user if admin@example.com does not exist.
// The password "Admin123!" is hashed with bcrypt cost 12 before storage.
func (s *Seeder) seedAdminUser(ctx context.Context) error {
	const adminEmail = "admin@example.com"

	// Check whether the admin user already exists.
	var existingID uuid.UUID
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id FROM users WHERE email = $1 LIMIT 1`,
		adminEmail,
	).Scan(&existingID)

	if err == nil {
		// Row found — admin already seeded.
		log.Printf("[seeder] admin user %q already exists, skipping", adminEmail)
		return nil
	}

	// Any error other than "no rows" is a real database error.
	if err != sql.ErrNoRows {
		return fmt.Errorf("check admin user existence: %w", err)
	}

	// Hash the default admin password.
	const defaultPassword = "Admin123!"
	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 12)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	// Resolve the admin role ID.
	var adminRoleID uuid.UUID
	err = s.db.QueryRowContext(
		ctx,
		`SELECT id FROM roles WHERE name = $1 LIMIT 1`,
		"admin",
	).Scan(&adminRoleID)
	if err != nil {
		return fmt.Errorf("get admin role id: %w", err)
	}

	now := time.Now().UTC()
	adminID := uuid.New()

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, role_id, email, password_hash, full_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (email) DO NOTHING`,
		adminID,
		adminRoleID,
		adminEmail,
		string(hashed),
		"Administrator",
		true,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	log.Printf("[seeder] admin user %q created with id %s", adminEmail, adminID)
	return nil
}

// seedSuperAdminUser inserts the default super_admin user if
// superadmin@example.com does not already exist.
// The password "SuperAdmin123!" is hashed with bcrypt cost 12 before storage.
func (s *Seeder) seedSuperAdminUser(ctx context.Context) error {
	const superAdminEmail = "superadmin@example.com"

	// Check whether the super admin user already exists.
	var existingID uuid.UUID
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id FROM users WHERE email = $1 LIMIT 1`,
		superAdminEmail,
	).Scan(&existingID)

	if err == nil {
		// Row found — super admin already seeded.
		log.Printf("[seeder] super admin user %q already exists, skipping", superAdminEmail)
		return nil
	}

	// Any error other than "no rows" is a real database error.
	if err != sql.ErrNoRows {
		return fmt.Errorf("check super admin user existence: %w", err)
	}

	// Hash the default super admin password.
	const defaultPassword = "SuperAdmin123!"
	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 12)
	if err != nil {
		return fmt.Errorf("hash super admin password: %w", err)
	}

	// Resolve the super_admin role ID.
	var superAdminRoleID uuid.UUID
	err = s.db.QueryRowContext(
		ctx,
		`SELECT id FROM roles WHERE name = $1 LIMIT 1`,
		"super_admin",
	).Scan(&superAdminRoleID)
	if err != nil {
		return fmt.Errorf("get super_admin role id: %w", err)
	}

	now := time.Now().UTC()
	superAdminID := uuid.New()

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, role_id, email, password_hash, full_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (email) DO NOTHING`,
		superAdminID,
		superAdminRoleID,
		superAdminEmail,
		string(hashed),
		"Super Administrator",
		true,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert super admin user: %w", err)
	}

	log.Printf("[seeder] super admin user %q created with id %s", superAdminEmail, superAdminID)
	return nil
}

// Run is the package-level convenience function that creates a Seeder and
// executes all seed operations against the provided database connection.
func Run(db *sqlx.DB) error {
	s := NewSeeder(db)
	return s.Run(context.Background())
}
