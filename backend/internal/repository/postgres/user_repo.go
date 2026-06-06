package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// userAllowedSortCols whitelists caller-supplied sort column names and maps
// them to fully-qualified SQL identifiers to prevent SQL injection.
var userAllowedSortCols = map[string]string{
	"email":      "u.email",
	"full_name":  "u.full_name",
	"created_at": "u.created_at",
	"updated_at": "u.updated_at",
	"is_active":  "u.is_active",
}

type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository constructs a userRepository backed by PostgreSQL.
func NewUserRepository(db *sqlx.DB) domain.UserRepository {
	return &userRepository{db: db}
}

// userScanRow is used to scan joined user + role columns in a single query.
type userScanRow struct {
	ID           string     `db:"id"`
	RoleID       string     `db:"role_id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	FullName     string     `db:"full_name"`
	IsActive     bool       `db:"is_active"`
	CreatedAt    dbTime     `db:"created_at"`
	UpdatedAt    dbTime     `db:"updated_at"`
	DeletedAt    dbNullTime `db:"deleted_at"`
	// Joined role fields aliased to avoid column name collision with u.role_id
	RoleIDJoined    string `db:"role_id_joined"`
	RoleName        string `db:"role_name"`
	RoleDescription string `db:"role_description"`
}

// toUser converts a scanned row into a domain.User.
func toUser(r userScanRow) (*domain.User, error) {
	id, err := parseUUID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}
	roleID, err := parseUUID(r.RoleID)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}
	roleIDJoined, err := parseUUID(r.RoleIDJoined)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}

	return &domain.User{
		ID:           id,
		RoleID:       roleID,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		FullName:     r.FullName,
		IsActive:     r.IsActive,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
		DeletedAt:    r.DeletedAt.Ptr(),
		Role: &domain.Role{
			ID:          roleIDJoined,
			Name:        domain.RoleName(r.RoleName),
			Description: r.RoleDescription,
		},
	}, nil
}

const userSelectCols = `
	u.id,
	u.role_id,
	u.email,
	u.password_hash,
	u.full_name,
	u.is_active,
	u.created_at,
	u.updated_at,
	u.deleted_at,
	r.id          AS role_id_joined,
	r.name        AS role_name,
	r.description AS role_description`

const userJoin = `FROM users u JOIN roles r ON r.id = u.role_id`

// FindByID returns the user with the given id. Returns domain.ErrNotFound when
// no active (non-deleted) user exists.
func (repo *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	q := fmt.Sprintf(`
		SELECT %s
		%s
		WHERE u.id = $1
		  AND u.deleted_at IS NULL
		LIMIT 1`, userSelectCols, userJoin)

	var row userScanRow
	if err := repo.db.QueryRowxContext(ctx, q, id.String()).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: user find by id: %w", err)
	}

	user, err := toUser(row)
	if err != nil {
		return nil, fmt.Errorf("repo: user find by id: %w", err)
	}
	return user, nil
}

// FindByEmail returns the active user with the given email address. Returns
// domain.ErrNotFound when no row matches.
func (repo *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := fmt.Sprintf(`
		SELECT %s
		%s
		WHERE u.email = $1
		  AND u.deleted_at IS NULL
		LIMIT 1`, userSelectCols, userJoin)

	var row userScanRow
	if err := repo.db.QueryRowxContext(ctx, q, email).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: user find by email: %w", err)
	}

	user, err := toUser(row)
	if err != nil {
		return nil, fmt.Errorf("repo: user find by email: %w", err)
	}
	return user, nil
}

// FindAll returns a paginated, optionally filtered list of active users along
// with the total count of matching rows before pagination is applied.
func (repo *userRepository) FindAll(ctx context.Context, params domain.ListParams) ([]domain.User, int64, error) {
	sortCol := allowedSort(params.SortBy, "u.created_at", userAllowedSortCols)
	sortDir := normSortDir(params.SortDir)
	limit, offset := pageOffset(params.Page, params.PageSize)

	var (
		args        []any
		filterParts []string
		argIdx      = 1
	)

	filterParts = append(filterParts, "u.deleted_at IS NULL")

	if params.Search != "" {
		filterParts = append(filterParts,
			fmt.Sprintf("(u.email ILIKE $%d OR u.full_name ILIKE $%d)", argIdx, argIdx+1),
		)
		pattern := "%" + params.Search + "%"
		args = append(args, pattern, pattern)
		argIdx += 2
	}

	if params.Status != "" {
		filterParts = append(filterParts, fmt.Sprintf("u.is_active = $%d", argIdx))
		args = append(args, params.Status == "active")
		argIdx++
	}

	where := "WHERE " + strings.Join(filterParts, " AND ")

	// Total count
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s`, where)
	var total int64
	if err := repo.db.QueryRowxContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: user find all count: %w", err)
	}

	// Paginated data
	dataQ := fmt.Sprintf(`
		SELECT %s
		%s
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		userSelectCols, userJoin, where,
		sortCol, sortDir,
		argIdx, argIdx+1,
	)
	dataArgs := append(args, limit, offset)

	rows, err := repo.db.QueryxContext(ctx, dataQ, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: user find all query: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var sr userScanRow
		if err := rows.StructScan(&sr); err != nil {
			return nil, 0, fmt.Errorf("repo: user find all scan: %w", err)
		}
		u, err := toUser(sr)
		if err != nil {
			return nil, 0, fmt.Errorf("repo: user find all: %w", err)
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: user find all rows: %w", err)
	}

	return users, total, nil
}

// Create inserts a new user row and populates user.ID from the RETURNING clause.
func (repo *userRepository) Create(ctx context.Context, user *domain.User) error {
	const q = `
		INSERT INTO users (id, role_id, email, password_hash, full_name, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var returnedID string
	err := repo.db.QueryRowxContext(ctx, q,
		user.ID.String(),
		user.RoleID.String(),
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&returnedID)
	if err != nil {
		return fmt.Errorf("repo: user create: %w", err)
	}

	parsed, err := parseUUID(returnedID)
	if err != nil {
		return fmt.Errorf("repo: user create parse returned id: %w", err)
	}
	user.ID = parsed
	return nil
}

// Update persists full_name and is_active changes for a non-deleted user.
// Returns domain.ErrNotFound when the user does not exist or is already deleted.
func (repo *userRepository) Update(ctx context.Context, user *domain.User) error {
	const q = `
		UPDATE users
		SET full_name  = $1,
		    is_active  = $2,
		    updated_at = NOW()
		WHERE id = $3
		  AND deleted_at IS NULL`

	res, err := repo.db.ExecContext(ctx, q, user.FullName, user.IsActive, user.ID.String())
	if err != nil {
		return fmt.Errorf("repo: user update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: user update rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdatePassword sets a new bcrypt password hash for the identified user.
// Returns domain.ErrNotFound when the user does not exist or is already deleted.
func (repo *userRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const q = `
		UPDATE users
		SET password_hash = $1,
		    updated_at    = NOW()
		WHERE id = $2
		  AND deleted_at IS NULL`

	res, err := repo.db.ExecContext(ctx, q, passwordHash, id.String())
	if err != nil {
		return fmt.Errorf("repo: user update password: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: user update password rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete soft-deletes the user by setting deleted_at to the current timestamp.
// Returns domain.ErrNotFound when the user does not exist or is already deleted.
func (repo *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE users
		SET deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL`

	res, err := repo.db.ExecContext(ctx, q, id.String())
	if err != nil {
		return fmt.Errorf("repo: user delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: user delete rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
