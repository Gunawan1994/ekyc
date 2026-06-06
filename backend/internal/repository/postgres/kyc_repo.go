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

// kycRepository implements domain.KYCRepository backed by PostgreSQL.
type kycRepository struct {
	db *sqlx.DB
}

// NewKYCRepository constructs a kycRepository.
func NewKYCRepository(db *sqlx.DB) domain.KYCRepository {
	return &kycRepository{db: db}
}

// kycSortCols maps caller-supplied sort column names to safe SQL identifiers.
var kycSortCols = map[string]string{
	"submitted_at": "submitted_at",
	"created_at":   "created_at",
	"updated_at":   "updated_at",
	"status":       "status",
}

// kycRow is the flat scan target for a kyc_verifications row.
type kycRow struct {
	ID              string     `db:"id"`
	CustomerID      string     `db:"customer_id"`
	CustomerName    string     `db:"customer_name"`
	ReviewerID      dbNullUUID `db:"reviewer_id"`
	SubmittedBy     string     `db:"submitted_by"`
	Status          string     `db:"status"`
	IDDocumentURL   string     `db:"id_document_url"`
	SelfieURL       string     `db:"selfie_url"`
	LivenessScore   float64    `db:"liveness_score"`
	FaceMatchScore  float64    `db:"face_match_score"`
	RiskLevel       string     `db:"risk_level"`
	RiskScore       int        `db:"risk_score"`
	RejectionReason string     `db:"rejection_reason"`
	Notes           string     `db:"notes"`
	SubmittedAt     dbTime     `db:"submitted_at"`
	ReviewedAt      dbNullTime `db:"reviewed_at"`
	CreatedAt       dbTime     `db:"created_at"`
	UpdatedAt       dbTime     `db:"updated_at"`
}

// toKYCVerification converts a scanned kycRow to the domain type.
func toKYCVerification(r kycRow) (domain.KYCVerification, error) {
	id, err := parseUUID(r.ID)
	if err != nil {
		return domain.KYCVerification{}, fmt.Errorf("repo: kyc – parse id: %w", err)
	}
	customerID, err := parseUUID(r.CustomerID)
	if err != nil {
		return domain.KYCVerification{}, fmt.Errorf("repo: kyc – parse customer_id: %w", err)
	}
	submittedBy, err := parseUUID(r.SubmittedBy)
	if err != nil {
		return domain.KYCVerification{}, fmt.Errorf("repo: kyc – parse submitted_by: %w", err)
	}

	return domain.KYCVerification{
		ID:              id,
		CustomerID:      customerID,
		CustomerName:    r.CustomerName,
		ReviewerID:      r.ReviewerID.Ptr(),
		SubmittedBy:     submittedBy,
		Status:          domain.VerificationStatus(r.Status),
		IDDocumentURL:   r.IDDocumentURL,
		SelfieURL:       r.SelfieURL,
		LivenessScore:   r.LivenessScore,
		FaceMatchScore:  r.FaceMatchScore,
		RiskLevel:       domain.RiskLevel(r.RiskLevel),
		RiskScore:       r.RiskScore,
		RejectionReason: r.RejectionReason,
		Notes:           r.Notes,
		SubmittedAt:     r.SubmittedAt.Time,
		ReviewedAt:      r.ReviewedAt.Ptr(),
		CreatedAt:       r.CreatedAt.Time,
		UpdatedAt:       r.UpdatedAt.Time,
	}, nil
}

// FindByID returns the KYC verification with the given id, or domain.ErrNotFound.
func (r *kycRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.KYCVerification, error) {
	const q = `
		SELECT kv.id, kv.customer_id, COALESCE(c.full_name, '') AS customer_name,
		       kv.reviewer_id, kv.submitted_by, kv.status,
		       kv.id_document_url, kv.selfie_url, kv.liveness_score, kv.face_match_score,
		       kv.risk_level, kv.risk_score,
		       kv.rejection_reason, kv.notes, kv.submitted_at, kv.reviewed_at,
		       kv.created_at, kv.updated_at
		FROM kyc_verifications kv
		LEFT JOIN customers c ON c.id = kv.customer_id
		WHERE kv.id = $1`

	var row kycRow
	if err := r.db.QueryRowxContext(ctx, q, id).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: kyc find by id: %w", err)
	}

	v, err := toKYCVerification(row)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// FindAll returns a paginated, optionally status-filtered list of KYC
// verifications ordered by submitted_at DESC by default, plus the total count.
func (r *kycRepository) FindAll(ctx context.Context, params domain.ListParams) ([]domain.KYCVerification, int64, error) {
	sortCol := allowedSort(params.SortBy, "submitted_at", kycSortCols)
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

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM kyc_verifications kv LEFT JOIN customers c ON c.id = kv.customer_id %s`, whereClause)
	var total int64
	if err := r.db.QueryRowxContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: kyc find all – count: %w", err)
	}

	args = append(args, limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT kv.id, kv.customer_id, COALESCE(c.full_name, '') AS customer_name,
		       kv.reviewer_id, kv.submitted_by, kv.status,
		       kv.id_document_url, kv.selfie_url, kv.liveness_score, kv.face_match_score,
		       kv.risk_level, kv.risk_score,
		       kv.rejection_reason, kv.notes, kv.submitted_at, kv.reviewed_at,
		       kv.created_at, kv.updated_at
		FROM kyc_verifications kv
		LEFT JOIN customers c ON c.id = kv.customer_id
		%s
		ORDER BY kv.%s %s
		LIMIT $%d OFFSET $%d`,
		whereClause,
		sortCol, sortDir,
		len(args)-1, len(args),
	)

	rows, err := r.db.QueryxContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: kyc find all – query: %w", err)
	}
	defer rows.Close()

	var verifications []domain.KYCVerification
	for rows.Next() {
		var row kycRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, fmt.Errorf("repo: kyc find all – scan: %w", err)
		}
		v, err := toKYCVerification(row)
		if err != nil {
			return nil, 0, err
		}
		verifications = append(verifications, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: kyc find all – rows: %w", err)
	}

	return verifications, total, nil
}

// Create inserts a new KYC verification row and writes the database-assigned
// id back into v.ID.
func (r *kycRepository) Create(ctx context.Context, v *domain.KYCVerification) error {
	const q = `
		INSERT INTO kyc_verifications (
			id, customer_id, submitted_by, status,
			id_document_url, selfie_url, liveness_score, face_match_score,
			rejection_reason, notes, submitted_at,
			created_at, updated_at
		) VALUES (
			$1,  $2,  $3,  $4,
			$5,  $6,  $7,  $8,
			$9,  $10, $11,
			NOW(), NOW()
		)
		RETURNING id`

	var returnedID string
	err := r.db.QueryRowxContext(ctx, q,
		v.ID,
		v.CustomerID,
		v.SubmittedBy,
		string(v.Status),
		v.IDDocumentURL,
		v.SelfieURL,
		v.LivenessScore,
		v.FaceMatchScore,
		v.RejectionReason,
		v.Notes,
		v.SubmittedAt,
	).Scan(&returnedID)
	if err != nil {
		return fmt.Errorf("repo: kyc create: %w", err)
	}

	id, err := parseUUID(returnedID)
	if err != nil {
		return fmt.Errorf("repo: kyc create – parse returned id: %w", err)
	}
	v.ID = id
	return nil
}

// UpdateStatus transitions a pending verification to the given status.
// Returns domain.ErrInvalidStatus when the record is not in pending state,
// and domain.ErrNotFound when no record with that id exists.
func (r *kycRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.VerificationStatus,
	reviewerID uuid.UUID,
	reason, notes string,
) error {
	const q = `
		UPDATE kyc_verifications
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
		return fmt.Errorf("repo: kyc update status: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: kyc update status – rows affected: %w", err)
	}
	if n == 0 {
		var exists bool
		if err := r.db.QueryRowxContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM kyc_verifications WHERE id = $1)`, id,
		).Scan(&exists); err != nil {
			return fmt.Errorf("repo: kyc update status – existence check: %w", err)
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
func (r *kycRepository) UpdateStatusFrom(
	ctx context.Context,
	id uuid.UUID,
	fromStatus, toStatus domain.VerificationStatus,
	reviewerID *uuid.UUID,
	notes string,
) error {
	const q = `
		UPDATE kyc_verifications
		SET    status      = $1,
		       reviewer_id = COALESCE($2, reviewer_id),
		       notes       = $3,
		       updated_at  = NOW()
		WHERE  id     = $4
		  AND  status = $5`

	result, err := r.db.ExecContext(ctx, q, string(toStatus), reviewerID, notes, id, string(fromStatus))
	if err != nil {
		return fmt.Errorf("repo: kyc update status from: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: kyc update status from – rows affected: %w", err)
	}
	if n == 0 {
		var exists bool
		if err := r.db.QueryRowxContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM kyc_verifications WHERE id = $1)`, id,
		).Scan(&exists); err != nil {
			return fmt.Errorf("repo: kyc update status from – existence check: %w", err)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrInvalidStatus
	}
	return nil
}

// FindByCustomerID returns all KYC verifications for a customer ordered by
// created_at DESC, with pagination support, plus the total matching count.
func (r *kycRepository) FindByCustomerID(
	ctx context.Context,
	customerID uuid.UUID,
	params domain.ListParams,
) ([]domain.KYCVerification, int64, error) {
	limit, offset := pageOffset(params.Page, params.PageSize)

	var total int64
	if err := r.db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM kyc_verifications WHERE customer_id = $1`, customerID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: kyc find by customer id – count: %w", err)
	}

	const q = `
		SELECT kv.id, kv.customer_id, COALESCE(c.full_name, '') AS customer_name,
		       kv.reviewer_id, kv.submitted_by, kv.status,
		       kv.id_document_url, kv.selfie_url, kv.liveness_score, kv.face_match_score,
		       kv.risk_level, kv.risk_score,
		       kv.rejection_reason, kv.notes, kv.submitted_at, kv.reviewed_at,
		       kv.created_at, kv.updated_at
		FROM kyc_verifications kv
		LEFT JOIN customers c ON c.id = kv.customer_id
		WHERE kv.customer_id = $1
		ORDER BY kv.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryxContext(ctx, q, customerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: kyc find by customer id – query: %w", err)
	}
	defer rows.Close()

	var verifications []domain.KYCVerification
	for rows.Next() {
		var row kycRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, fmt.Errorf("repo: kyc find by customer id – scan: %w", err)
		}
		v, err := toKYCVerification(row)
		if err != nil {
			return nil, 0, err
		}
		verifications = append(verifications, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: kyc find by customer id – rows: %w", err)
	}

	return verifications, total, nil
}

// CountByStatus returns the number of KYC verifications with the given status.
func (r *kycRepository) CountByStatus(ctx context.Context, status domain.VerificationStatus) (int64, error) {
	const q = `SELECT COUNT(*) FROM kyc_verifications WHERE status = $1`

	var count int64
	if err := r.db.QueryRowxContext(ctx, q, string(status)).Scan(&count); err != nil {
		return 0, fmt.Errorf("repo: kyc count by status: %w", err)
	}
	return count, nil
}

// Delete hard-deletes a KYC verification by id. Returns domain.ErrNotFound when
// no record with that id exists.
func (r *kycRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM kyc_verifications WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("repo: kyc delete: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("repo: kyc delete – rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
