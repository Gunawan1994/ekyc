package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// customerAllowedSortCols whitelists caller-supplied sort column names.
var customerAllowedSortCols = map[string]string{
	"full_name":     "full_name",
	"email":         "email",
	"created_at":    "created_at",
	"updated_at":    "updated_at",
	"date_of_birth": "date_of_birth",
	"id_number":     "id_number",
}

type customerRepository struct {
	db *sqlx.DB
}

// NewCustomerRepository constructs a customerRepository backed by PostgreSQL.
func NewCustomerRepository(db *sqlx.DB) domain.CustomerRepository {
	return &customerRepository{db: db}
}

// customerScanRow maps every column of the customers table.
type customerScanRow struct {
	ID           string         `db:"id"`
	UserID       sql.NullString `db:"user_id"`
	CompanyID    string         `db:"company_id"`
	FullName     string     `db:"full_name"`
	DateOfBirth  dbTime     `db:"date_of_birth"`
	PlaceOfBirth string     `db:"place_of_birth"`
	Gender       string     `db:"gender"`
	Nationality  string     `db:"nationality"`
	IDType       string     `db:"id_type"`
	IDNumber     string     `db:"id_number"`
	Address      string     `db:"address"`
	City         string     `db:"city"`
	Province     string     `db:"province"`
	PostalCode   string     `db:"postal_code"`
	Country      string     `db:"country"`
	PhoneNumber  string     `db:"phone_number"`
	Email        string     `db:"email"`
	CreatedAt    dbTime     `db:"created_at"`
	UpdatedAt    dbTime     `db:"updated_at"`
	DeletedAt    dbNullTime `db:"deleted_at"`
}

// toCustomer converts a scanned row into a domain.Customer.
func toCustomer(r customerScanRow) (*domain.Customer, error) {
	id, err := parseUUID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}
	var userID uuid.UUID
	if r.UserID.Valid && r.UserID.String != "" {
		userID, err = parseUUID(r.UserID.String)
		if err != nil {
			return nil, fmt.Errorf("repo: %w", err)
		}
	}
	companyID, err := parseUUID(r.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}

	return &domain.Customer{
		ID:           id,
		UserID:       userID,
		CompanyID:    companyID,
		FullName:     r.FullName,
		DateOfBirth:  r.DateOfBirth.Time,
		PlaceOfBirth: r.PlaceOfBirth,
		Gender:       r.Gender,
		Nationality:  r.Nationality,
		IDType:       domain.IDType(r.IDType),
		IDNumber:     r.IDNumber,
		Address:      r.Address,
		City:         r.City,
		Province:     r.Province,
		PostalCode:   r.PostalCode,
		Country:      r.Country,
		PhoneNumber:  r.PhoneNumber,
		Email:        r.Email,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
		DeletedAt:    r.DeletedAt.Ptr(),
	}, nil
}

const customerSelectCols = `
	id, user_id, company_id, full_name, date_of_birth, place_of_birth,
	gender, nationality, id_type, id_number, address, city, province,
	postal_code, country, phone_number, email, created_at, updated_at, deleted_at`

// FindByID returns the customer with the given id. Returns domain.ErrNotFound
// when the customer does not exist or has been soft-deleted.
func (repo *customerRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM customers
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`, customerSelectCols)

	var row customerScanRow
	if err := repo.db.QueryRowxContext(ctx, q, id.String()).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: customer find by id: %w", err)
	}

	customer, err := toCustomer(row)
	if err != nil {
		return nil, fmt.Errorf("repo: customer find by id: %w", err)
	}
	return customer, nil
}

// FindByCompanyID returns a paginated list of non-deleted customers belonging to
// the given company, along with the total count before pagination is applied.
func (repo *customerRepository) FindByCompanyID(
	ctx context.Context,
	companyID uuid.UUID,
	params domain.ListParams,
) ([]domain.Customer, int64, error) {
	sortCol := allowedSort(params.SortBy, "created_at", customerAllowedSortCols)
	sortDir := normSortDir(params.SortDir)
	limit, offset := pageOffset(params.Page, params.PageSize)

	var (
		args        []any
		filterParts []string
		argIdx      = 1
	)

	filterParts = append(filterParts, fmt.Sprintf("company_id = $%d", argIdx))
	args = append(args, companyID.String())
	argIdx++

	filterParts = append(filterParts, "deleted_at IS NULL")

	if params.Search != "" {
		filterParts = append(filterParts,
			fmt.Sprintf("(full_name ILIKE $%d OR email ILIKE $%d OR id_number ILIKE $%d)", argIdx, argIdx+1, argIdx+2),
		)
		pattern := "%" + params.Search + "%"
		args = append(args, pattern, pattern, pattern)
		argIdx += 3
	}

	where := "WHERE " + strings.Join(filterParts, " AND ")

	// Total count
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM customers %s`, where)
	var total int64
	if err := repo.db.QueryRowxContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: customer find by company id count: %w", err)
	}

	// Paginated data
	dataQ := fmt.Sprintf(`
		SELECT %s
		FROM customers
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		customerSelectCols, where,
		sortCol, sortDir,
		argIdx, argIdx+1,
	)
	dataArgs := append(args, limit, offset)

	rows, err := repo.db.QueryxContext(ctx, dataQ, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: customer find by company id query: %w", err)
	}
	defer rows.Close()

	var customers []domain.Customer
	for rows.Next() {
		var sr customerScanRow
		if err := rows.StructScan(&sr); err != nil {
			return nil, 0, fmt.Errorf("repo: customer find by company id scan: %w", err)
		}
		c, err := toCustomer(sr)
		if err != nil {
			return nil, 0, fmt.Errorf("repo: customer find by company id: %w", err)
		}
		customers = append(customers, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: customer find by company id rows: %w", err)
	}

	return customers, total, nil
}

// FindAll returns a paginated, optionally filtered list of non-deleted customers.
// When params.Status holds a valid UUID string it is treated as a company_id filter,
// enabling the admin view to scope results to a specific company.
func (repo *customerRepository) FindAll(ctx context.Context, params domain.ListParams) ([]domain.Customer, int64, error) {
	sortCol := allowedSort(params.SortBy, "created_at", customerAllowedSortCols)
	sortDir := normSortDir(params.SortDir)
	limit, offset := pageOffset(params.Page, params.PageSize)

	var (
		args        []any
		filterParts []string
		argIdx      = 1
	)

	filterParts = append(filterParts, "deleted_at IS NULL")

	if params.Search != "" {
		filterParts = append(filterParts,
			fmt.Sprintf("(full_name ILIKE $%d OR email ILIKE $%d OR id_number ILIKE $%d)", argIdx, argIdx+1, argIdx+2),
		)
		pattern := "%" + params.Search + "%"
		args = append(args, pattern, pattern, pattern)
		argIdx += 3
	}

	// company_id filter: treat Status as a UUID when it parses as one.
	if params.Status != "" {
		if _, uuidErr := uuid.Parse(params.Status); uuidErr == nil {
			filterParts = append(filterParts, fmt.Sprintf("company_id = $%d", argIdx))
			args = append(args, params.Status)
			argIdx++
		}
	}

	where := "WHERE " + strings.Join(filterParts, " AND ")

	// Total count
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM customers %s`, where)
	var total int64
	if err := repo.db.QueryRowxContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: customer find all count: %w", err)
	}

	// Paginated data
	dataQ := fmt.Sprintf(`
		SELECT %s
		FROM customers
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		customerSelectCols, where,
		sortCol, sortDir,
		argIdx, argIdx+1,
	)
	dataArgs := append(args, limit, offset)

	rows, err := repo.db.QueryxContext(ctx, dataQ, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: customer find all query: %w", err)
	}
	defer rows.Close()

	var customers []domain.Customer
	for rows.Next() {
		var sr customerScanRow
		if err := rows.StructScan(&sr); err != nil {
			return nil, 0, fmt.Errorf("repo: customer find all scan: %w", err)
		}
		c, err := toCustomer(sr)
		if err != nil {
			return nil, 0, fmt.Errorf("repo: customer find all: %w", err)
		}
		customers = append(customers, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: customer find all rows: %w", err)
	}

	return customers, total, nil
}

// Create inserts a new customer row and populates customer.ID from the RETURNING clause.
func (repo *customerRepository) Create(ctx context.Context, customer *domain.Customer) error {
	const q = `
		INSERT INTO customers (
			id, user_id, company_id, full_name, date_of_birth, place_of_birth,
			gender, nationality, id_type, id_number, address, city, province,
			postal_code, country, phone_number, email, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19
		)
		RETURNING id`

	var userIDArg interface{}
	if customer.UserID != uuid.Nil {
		userIDArg = customer.UserID.String()
	}

	var returnedID string
	err := repo.db.QueryRowxContext(ctx, q,
		customer.ID.String(),
		userIDArg,
		customer.CompanyID.String(),
		customer.FullName,
		customer.DateOfBirth,
		customer.PlaceOfBirth,
		customer.Gender,
		customer.Nationality,
		string(customer.IDType),
		customer.IDNumber,
		customer.Address,
		customer.City,
		customer.Province,
		customer.PostalCode,
		customer.Country,
		customer.PhoneNumber,
		customer.Email,
		customer.CreatedAt,
		customer.UpdatedAt,
	).Scan(&returnedID)
	if err != nil {
		return fmt.Errorf("repo: customer create: %w", err)
	}

	parsed, err := parseUUID(returnedID)
	if err != nil {
		return fmt.Errorf("repo: customer create parse returned id: %w", err)
	}
	customer.ID = parsed
	return nil
}

// Update persists all mutable customer fields for a non-deleted customer.
// Returns domain.ErrNotFound when the customer does not exist or is deleted.
func (repo *customerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	const q = `
		UPDATE customers
		SET full_name      = $1,
		    date_of_birth  = $2,
		    place_of_birth = $3,
		    gender         = $4,
		    nationality    = $5,
		    id_type        = $6,
		    id_number      = $7,
		    address        = $8,
		    city           = $9,
		    province       = $10,
		    postal_code    = $11,
		    country        = $12,
		    phone_number   = $13,
		    email          = $14,
		    updated_at     = NOW()
		WHERE id = $15
		  AND deleted_at IS NULL`

	res, err := repo.db.ExecContext(ctx, q,
		customer.FullName,
		customer.DateOfBirth,
		customer.PlaceOfBirth,
		customer.Gender,
		customer.Nationality,
		string(customer.IDType),
		customer.IDNumber,
		customer.Address,
		customer.City,
		customer.Province,
		customer.PostalCode,
		customer.Country,
		customer.PhoneNumber,
		customer.Email,
		customer.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("repo: customer update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: customer update rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete soft-deletes the customer by setting deleted_at to the current timestamp.
// Returns domain.ErrNotFound when the customer does not exist or is already deleted.
func (repo *customerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE customers
		SET deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL`

	res, err := repo.db.ExecContext(ctx, q, id.String())
	if err != nil {
		return fmt.Errorf("repo: customer delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: customer delete rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
