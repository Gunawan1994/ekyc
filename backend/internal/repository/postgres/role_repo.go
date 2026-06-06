package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// roleRepo implements domain.RoleRepository backed by PostgreSQL.
type roleRepo struct {
	db *sqlx.DB
}

// NewRoleRepo constructs a roleRepo.
func NewRoleRepo(db *sqlx.DB) domain.RoleRepository {
	return &roleRepo{db: db}
}

// FindByName returns the role whose name matches exactly. Returns
// domain.ErrNotFound when no row exists.
func (r *roleRepo) FindByName(ctx context.Context, name domain.RoleName) (*domain.Role, error) {
	const q = `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE name = $1
		LIMIT 1`

	var row struct {
		ID          string `db:"id"`
		Name        string `db:"name"`
		Description string `db:"description"`
		CreatedAt   dbTime `db:"created_at"`
		UpdatedAt   dbTime `db:"updated_at"`
	}

	if err := r.db.QueryRowxContext(ctx, q, string(name)).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: role find by name: %w", err)
	}

	id, err := parseUUID(row.ID)
	if err != nil {
		return nil, fmt.Errorf("repo: role find by name – parse id: %w", err)
	}

	return &domain.Role{
		ID:          id,
		Name:        domain.RoleName(row.Name),
		Description: row.Description,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

// FindAll returns all roles ordered by name.
func (r *roleRepo) FindAll(ctx context.Context) ([]domain.Role, error) {
	const q = `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY name`

	type scanRow struct {
		ID          string `db:"id"`
		Name        string `db:"name"`
		Description string `db:"description"`
		CreatedAt   dbTime `db:"created_at"`
		UpdatedAt   dbTime `db:"updated_at"`
	}

	rows, err := r.db.QueryxContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("repo: role find all: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var sr scanRow
		if err := rows.StructScan(&sr); err != nil {
			return nil, fmt.Errorf("repo: role find all – scan: %w", err)
		}
		id, err := parseUUID(sr.ID)
		if err != nil {
			return nil, fmt.Errorf("repo: role find all – parse id: %w", err)
		}
		roles = append(roles, domain.Role{
			ID:          id,
			Name:        domain.RoleName(sr.Name),
			Description: sr.Description,
			CreatedAt:   sr.CreatedAt.Time,
			UpdatedAt:   sr.UpdatedAt.Time,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repo: role find all – rows: %w", err)
	}

	return roles, nil
}
