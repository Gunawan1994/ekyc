package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// companyAllowedSortCols whitelists caller-supplied sort column names.
var companyAllowedSortCols = map[string]string{
	"name":            "name",
	"legal_name":      "legal_name",
	"status":          "status",
	"created_at":      "created_at",
	"updated_at":      "updated_at",
	"registration_no": "registration_no",
}

type companyRepository struct {
	db *sqlx.DB
}

// NewCompanyRepository constructs a companyRepository backed by PostgreSQL.
func NewCompanyRepository(db *sqlx.DB) domain.CompanyRepository {
	return &companyRepository{db: db}
}

// companyScanRow maps every column of the companies table.
type companyScanRow struct {
	ID             string     `db:"id"`
	UserID         string     `db:"user_id"`
	Name           string     `db:"name"`
	LegalName      string     `db:"legal_name"`
	RegistrationNo string     `db:"registration_no"`
	TaxID          string     `db:"tax_id"`
	Industry       string     `db:"industry"`
	Address        string     `db:"address"`
	City           string     `db:"city"`
	Province       string     `db:"province"`
	PostalCode     string     `db:"postal_code"`
	Country        string     `db:"country"`
	PhoneNumber    string     `db:"phone_number"`
	Email          string     `db:"email"`
	Website        string     `db:"website"`
	Status         string     `db:"status"`
	CreatedAt      dbTime     `db:"created_at"`
	UpdatedAt      dbTime     `db:"updated_at"`
	DeletedAt      dbNullTime `db:"deleted_at"`
}

// toCompany converts a scanned row into a domain.Company.
func toCompany(r companyScanRow) (*domain.Company, error) {
	id, err := parseUUID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}
	userID, err := parseUUID(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}

	return &domain.Company{
		ID:             id,
		UserID:         userID,
		Name:           r.Name,
		LegalName:      r.LegalName,
		RegistrationNo: r.RegistrationNo,
		TaxID:          r.TaxID,
		Industry:       r.Industry,
		Address:        r.Address,
		City:           r.City,
		Province:       r.Province,
		PostalCode:     r.PostalCode,
		Country:        r.Country,
		PhoneNumber:    r.PhoneNumber,
		Email:          r.Email,
		Website:        r.Website,
		Status:         domain.CompanyStatus(r.Status),
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
		DeletedAt:      r.DeletedAt.Ptr(),
	}, nil
}

const companySelectCols = `
	id, user_id, name, legal_name, registration_no, tax_id, industry,
	address, city, province, postal_code, country, phone_number, email,
	website, status, created_at, updated_at, deleted_at`

// FindByID returns the company with the given id. Returns domain.ErrNotFound
// when the company does not exist or has been soft-deleted.
func (repo *companyRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM companies
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`, companySelectCols)

	var row companyScanRow
	if err := repo.db.QueryRowxContext(ctx, q, id.String()).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: company find by id: %w", err)
	}

	company, err := toCompany(row)
	if err != nil {
		return nil, fmt.Errorf("repo: company find by id: %w", err)
	}
	return company, nil
}

// FindByUserID returns the company associated with the given user id. Returns
// domain.ErrNotFound when no non-deleted company exists for that user.
func (repo *companyRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*domain.Company, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM companies
		WHERE user_id = $1
		  AND deleted_at IS NULL
		LIMIT 1`, companySelectCols)

	var row companyScanRow
	if err := repo.db.QueryRowxContext(ctx, q, userID.String()).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: company find by user id: %w", err)
	}

	company, err := toCompany(row)
	if err != nil {
		return nil, fmt.Errorf("repo: company find by user id: %w", err)
	}
	return company, nil
}

// FindByRegistrationNo returns the company with the given registration number.
// Returns domain.ErrNotFound when no non-deleted company exists with that number.
func (repo *companyRepository) FindByRegistrationNo(ctx context.Context, registrationNo string) (*domain.Company, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM companies
		WHERE registration_no = $1
		  AND deleted_at IS NULL
		LIMIT 1`, companySelectCols)

	var row companyScanRow
	if err := repo.db.QueryRowxContext(ctx, q, registrationNo).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: company find by registration no: %w", err)
	}

	company, err := toCompany(row)
	if err != nil {
		return nil, fmt.Errorf("repo: company find by registration no: %w", err)
	}
	return company, nil
}

// FindAll returns a paginated, optionally filtered list of non-deleted companies
// along with the total count before pagination is applied.
func (repo *companyRepository) FindAll(ctx context.Context, params domain.ListParams) ([]domain.Company, int64, error) {
	sortCol := allowedSort(params.SortBy, "created_at", companyAllowedSortCols)
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
			fmt.Sprintf("(name ILIKE $%d OR legal_name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx+1, argIdx+2),
		)
		pattern := "%" + params.Search + "%"
		args = append(args, pattern, pattern, pattern)
		argIdx += 3
	}

	if params.Status != "" {
		filterParts = append(filterParts, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(filterParts, " AND ")

	// Total count
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM companies %s`, where)
	var total int64
	if err := repo.db.QueryRowxContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: company find all count: %w", err)
	}

	// Paginated data
	dataQ := fmt.Sprintf(`
		SELECT %s
		FROM companies
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		companySelectCols, where,
		sortCol, sortDir,
		argIdx, argIdx+1,
	)
	dataArgs := append(args, limit, offset)

	rows, err := repo.db.QueryxContext(ctx, dataQ, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: company find all query: %w", err)
	}
	defer rows.Close()

	var companies []domain.Company
	for rows.Next() {
		var sr companyScanRow
		if err := rows.StructScan(&sr); err != nil {
			return nil, 0, fmt.Errorf("repo: company find all scan: %w", err)
		}
		c, err := toCompany(sr)
		if err != nil {
			return nil, 0, fmt.Errorf("repo: company find all: %w", err)
		}
		companies = append(companies, *c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: company find all rows: %w", err)
	}

	return companies, total, nil
}

// Create inserts a new company row and populates company.ID from the RETURNING clause.
func (repo *companyRepository) Create(ctx context.Context, company *domain.Company) error {
	const q = `
		INSERT INTO companies (
			id, user_id, name, legal_name, registration_no, tax_id, industry,
			address, city, province, postal_code, country, phone_number, email,
			website, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18
		)
		RETURNING id`

	var returnedID string
	err := repo.db.QueryRowxContext(ctx, q,
		company.ID.String(),
		company.UserID.String(),
		company.Name,
		company.LegalName,
		company.RegistrationNo,
		company.TaxID,
		company.Industry,
		company.Address,
		company.City,
		company.Province,
		company.PostalCode,
		company.Country,
		company.PhoneNumber,
		company.Email,
		company.Website,
		string(company.Status),
		company.CreatedAt,
		company.UpdatedAt,
	).Scan(&returnedID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("repo: company create: %w", err)
	}

	parsed, err := parseUUID(returnedID)
	if err != nil {
		return fmt.Errorf("repo: company create parse returned id: %w", err)
	}
	company.ID = parsed
	return nil
}

// Update persists mutable company fields for a non-deleted company.
// Returns domain.ErrNotFound when the company does not exist or is deleted.
func (repo *companyRepository) Update(ctx context.Context, company *domain.Company) error {
	const q = `
		UPDATE companies
		SET name            = $1,
		    legal_name      = $2,
		    registration_no = $3,
		    tax_id          = $4,
		    industry        = $5,
		    address         = $6,
		    city            = $7,
		    province        = $8,
		    postal_code     = $9,
		    country         = $10,
		    phone_number    = $11,
		    email           = $12,
		    website         = $13,
		    status          = $14,
		    updated_at      = NOW()
		WHERE id = $15
		  AND deleted_at IS NULL`

	res, err := repo.db.ExecContext(ctx, q,
		company.Name,
		company.LegalName,
		company.RegistrationNo,
		company.TaxID,
		company.Industry,
		company.Address,
		company.City,
		company.Province,
		company.PostalCode,
		company.Country,
		company.PhoneNumber,
		company.Email,
		company.Website,
		string(company.Status),
		company.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("repo: company update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: company update rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete soft-deletes the company by setting deleted_at to the current timestamp.
// Returns domain.ErrNotFound when the company does not exist or is already deleted.
func (repo *companyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE companies
		SET deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL`

	res, err := repo.db.ExecContext(ctx, q, id.String())
	if err != nil {
		return fmt.Errorf("repo: company delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: company delete rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
