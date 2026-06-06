package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// kybRepository implements domain.KYBRepository backed by PostgreSQL.
type kybRepository struct {
	db *sqlx.DB
}

// NewKYBRepository constructs a kybRepository.
func NewKYBRepository(db *sqlx.DB) domain.KYBRepository {
	return &kybRepository{db: db}
}

// kybSortCols maps caller-supplied sort column names to safe SQL identifiers.
var kybSortCols = map[string]string{
	"submitted_at": "submitted_at",
	"created_at":   "created_at",
	"updated_at":   "updated_at",
	"status":       "status",
}

// kybRow is the flat scan target for a kyb_verifications row.
type kybRow struct {
	ID               string     `db:"id"`
	CompanyID        string     `db:"company_id"`
	CompanyName      string     `db:"company_name"`
	ReviewerID       dbNullUUID `db:"reviewer_id"`
	SubmittedBy      string     `db:"submitted_by"`
	Status           string     `db:"status"`
	BusinessDocURL   string     `db:"business_doc_url"`
	TaxDocURL        string     `db:"tax_doc_url"`
	DirectorIDDocURL string     `db:"director_id_doc_url"`
	RiskLevel        string     `db:"risk_level"`
	RiskScore        int        `db:"risk_score"`
	RejectionReason  string     `db:"rejection_reason"`
	Notes            string     `db:"notes"`
	SubmittedAt      dbTime     `db:"submitted_at"`
	ReviewedAt       dbNullTime `db:"reviewed_at"`
	CreatedAt        dbTime     `db:"created_at"`
	UpdatedAt        dbTime     `db:"updated_at"`
}

// toKYBVerification converts a scanned kybRow to the domain type.
func toKYBVerification(r kybRow) (domain.KYBVerification, error) {
	id, err := parseUUID(r.ID)
	if err != nil {
		return domain.KYBVerification{}, fmt.Errorf("repo: kyb – parse id: %w", err)
	}
	companyID, err := parseUUID(r.CompanyID)
	if err != nil {
		return domain.KYBVerification{}, fmt.Errorf("repo: kyb – parse company_id: %w", err)
	}
	submittedBy, err := parseUUID(r.SubmittedBy)
	if err != nil {
		return domain.KYBVerification{}, fmt.Errorf("repo: kyb – parse submitted_by: %w", err)
	}

	return domain.KYBVerification{
		ID:               id,
		CompanyID:        companyID,
		CompanyName:      r.CompanyName,
		ReviewerID:       r.ReviewerID.Ptr(),
		SubmittedBy:      submittedBy,
		Status:           domain.VerificationStatus(r.Status),
		BusinessDocURL:   r.BusinessDocURL,
		TaxDocURL:        r.TaxDocURL,
		DirectorIDDocURL: r.DirectorIDDocURL,
		RiskLevel:        domain.RiskLevel(r.RiskLevel),
		RiskScore:        r.RiskScore,
		RejectionReason:  r.RejectionReason,
		Notes:            r.Notes,
		SubmittedAt:      r.SubmittedAt.Time,
		ReviewedAt:       r.ReviewedAt.Ptr(),
		CreatedAt:        r.CreatedAt.Time,
		UpdatedAt:        r.UpdatedAt.Time,
	}, nil
}

// FindByID returns the KYB verification with the given id, or domain.ErrNotFound.
func (r *kybRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.KYBVerification, error) {
	const q = `
		SELECT kv.id, kv.company_id, COALESCE(co.name, '') AS company_name,
		       kv.reviewer_id, kv.submitted_by, kv.status,
		       kv.business_doc_url, kv.tax_doc_url, kv.director_id_doc_url,
		       kv.risk_level, kv.risk_score,
		       kv.rejection_reason, kv.notes, kv.submitted_at, kv.reviewed_at,
		       kv.created_at, kv.updated_at
		FROM kyb_verifications kv
		LEFT JOIN companies co ON co.id = kv.company_id
		WHERE kv.id = $1`

	var row kybRow
	if err := r.db.QueryRowxContext(ctx, q, id).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: kyb find by id: %w", err)
	}

	v, err := toKYBVerification(row)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// FindAll returns a paginated, optionally status-filtered list of KYB
// verifications ordered by submitted_at DESC by default, plus the total count.
func (r *kybRepository) FindAll(ctx context.Context, params domain.ListParams) ([]domain.KYBVerification, int64, error) {
	sortCol := allowedSort(params.SortBy, "submitted_at", kybSortCols)
	sortDir := normSortDir(params.SortDir)
	limit, offset := pageOffset(params.Page, params.PageSize)

	var where []string
	var args []any

	if params.Status != "" {
		args = append(args, params.Status)
		where = append(where, fmt.Sprintf("kv.status = $%d", len(args)))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM kyb_verifications kv LEFT JOIN companies co ON co.id = kv.company_id %s`, whereClause)
	var total int64
	if err := r.db.QueryRowxContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: kyb find all – count: %w", err)
	}

	args = append(args, limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT kv.id, kv.company_id, COALESCE(co.name, '') AS company_name,
		       kv.reviewer_id, kv.submitted_by, kv.status,
		       kv.business_doc_url, kv.tax_doc_url, kv.director_id_doc_url,
		       kv.risk_level, kv.risk_score,
		       kv.rejection_reason, kv.notes, kv.submitted_at, kv.reviewed_at,
		       kv.created_at, kv.updated_at
		FROM kyb_verifications kv
		LEFT JOIN companies co ON co.id = kv.company_id
		%s
		ORDER BY kv.%s %s
		LIMIT $%d OFFSET $%d`,
		whereClause,
		sortCol, sortDir,
		len(args)-1, len(args),
	)

	rows, err := r.db.QueryxContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: kyb find all – query: %w", err)
	}
	defer rows.Close()

	var verifications []domain.KYBVerification
	for rows.Next() {
		var row kybRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, fmt.Errorf("repo: kyb find all – scan: %w", err)
		}
		v, err := toKYBVerification(row)
		if err != nil {
			return nil, 0, err
		}
		verifications = append(verifications, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: kyb find all – rows: %w", err)
	}

	return verifications, total, nil
}

// Create inserts a new KYB verification row and writes the database-assigned
// id back into v.ID.
func (r *kybRepository) Create(ctx context.Context, v *domain.KYBVerification) error {
	const q = `
		INSERT INTO kyb_verifications (
			id, company_id, submitted_by, status,
			business_doc_url, tax_doc_url, director_id_doc_url,
			rejection_reason, notes, submitted_at,
			created_at, updated_at
		) VALUES (
			$1,  $2,  $3,  $4,
			$5,  $6,  $7,
			$8,  $9,  $10,
			NOW(), NOW()
		)
		RETURNING id`

	var returnedID string
	err := r.db.QueryRowxContext(ctx, q,
		v.ID,
		v.CompanyID,
		v.SubmittedBy,
		string(v.Status),
		v.BusinessDocURL,
		v.TaxDocURL,
		v.DirectorIDDocURL,
		v.RejectionReason,
		v.Notes,
		v.SubmittedAt,
	).Scan(&returnedID)
	if err != nil {
		return fmt.Errorf("repo: kyb create: %w", err)
	}

	id, err := parseUUID(returnedID)
	if err != nil {
		return fmt.Errorf("repo: kyb create – parse returned id: %w", err)
	}
	v.ID = id
	return nil
}

// UpdateStatus transitions a pending verification to the given status.
// Returns domain.ErrInvalidStatus when the record is not in pending state,
// and domain.ErrNotFound when no record with that id exists.
func (r *kybRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.VerificationStatus,
	reviewerID uuid.UUID,
	reason, notes string,
) error {
	const q = `
		UPDATE kyb_verifications
		SET    status           = $1,
		       reviewer_id      = $2,
		       reviewed_at      = NOW(),
		       rejection_reason = $3,
		       notes            = $4,
		       updated_at       = NOW()
		WHERE  id     = $5
		  AND  status = 'pending'`

	result, err := r.db.ExecContext(ctx, q, string(status), reviewerID, reason, notes, id)
	if err != nil {
		return fmt.Errorf("repo: kyb update status: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: kyb update status – rows affected: %w", err)
	}
	if n == 0 {
		var exists bool
		if err := r.db.QueryRowxContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM kyb_verifications WHERE id = $1)`, id,
		).Scan(&exists); err != nil {
			return fmt.Errorf("repo: kyb update status – existence check: %w", err)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrInvalidStatus
	}
	return nil
}

// UpdateStatusFrom transitions a verification from fromStatus to toStatus.
// reviewerID is optional (pass nil to keep the existing reviewer_id value).
// Returns domain.ErrInvalidStatus when the current status does not match
// fromStatus, and domain.ErrNotFound when the record does not exist.
func (r *kybRepository) UpdateStatusFrom(
	ctx context.Context,
	id uuid.UUID,
	fromStatus, toStatus domain.VerificationStatus,
	reviewerID *uuid.UUID,
	notes string,
) error {
	const q = `
		UPDATE kyb_verifications
		SET    status      = $1,
		       reviewer_id = COALESCE($2, reviewer_id),
		       notes       = $3,
		       updated_at  = NOW()
		WHERE  id     = $4
		  AND  status = $5`

	result, err := r.db.ExecContext(ctx, q, string(toStatus), reviewerID, notes, id, string(fromStatus))
	if err != nil {
		return fmt.Errorf("repo: kyb update status from: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: kyb update status from – rows affected: %w", err)
	}
	if n == 0 {
		var exists bool
		if err := r.db.QueryRowxContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM kyb_verifications WHERE id = $1)`, id,
		).Scan(&exists); err != nil {
			return fmt.Errorf("repo: kyb update status from – existence check: %w", err)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrInvalidStatus
	}
	return nil
}

// FindByCompanyID returns all KYB verifications for a company ordered by
// created_at DESC, with pagination support, plus the total matching count.
func (r *kybRepository) FindByCompanyID(
	ctx context.Context,
	companyID uuid.UUID,
	params domain.ListParams,
) ([]domain.KYBVerification, int64, error) {
	limit, offset := pageOffset(params.Page, params.PageSize)

	var total int64
	if err := r.db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM kyb_verifications WHERE company_id = $1`, companyID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: kyb find by company id – count: %w", err)
	}

	const q = `
		SELECT kv.id, kv.company_id, COALESCE(co.name, '') AS company_name,
		       kv.reviewer_id, kv.submitted_by, kv.status,
		       kv.business_doc_url, kv.tax_doc_url, kv.director_id_doc_url,
		       kv.risk_level, kv.risk_score,
		       kv.rejection_reason, kv.notes, kv.submitted_at, kv.reviewed_at,
		       kv.created_at, kv.updated_at
		FROM kyb_verifications kv
		LEFT JOIN companies co ON co.id = kv.company_id
		WHERE kv.company_id = $1
		ORDER BY kv.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryxContext(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: kyb find by company id – query: %w", err)
	}
	defer rows.Close()

	var verifications []domain.KYBVerification
	for rows.Next() {
		var row kybRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, fmt.Errorf("repo: kyb find by company id – scan: %w", err)
		}
		v, err := toKYBVerification(row)
		if err != nil {
			return nil, 0, err
		}
		verifications = append(verifications, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: kyb find by company id – rows: %w", err)
	}

	return verifications, total, nil
}

// CountByStatus returns the number of KYB verifications with the given status.
func (r *kybRepository) CountByStatus(ctx context.Context, status domain.VerificationStatus) (int64, error) {
	const q = `SELECT COUNT(*) FROM kyb_verifications WHERE status = $1`

	var count int64
	if err := r.db.QueryRowxContext(ctx, q, string(status)).Scan(&count); err != nil {
		return 0, fmt.Errorf("repo: kyb count by status: %w", err)
	}
	return count, nil
}

// Delete hard-deletes a KYB verification by id. Returns domain.ErrNotFound when
// no record with that id exists.
func (r *kybRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM kyb_verifications WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("repo: kyb delete: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: kyb delete – rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
