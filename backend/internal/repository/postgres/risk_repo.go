package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

type riskRepository struct {
	db *sqlx.DB
}

func NewRiskRepository(db *sqlx.DB) domain.RiskRepository {
	return &riskRepository{db: db}
}

type riskRow struct {
	ID          string     `db:"id"`
	EntityType  string     `db:"entity_type"`
	EntityID    string     `db:"entity_id"`
	RiskLevel   string     `db:"risk_level"`
	RiskScore   int        `db:"risk_score"`
	RiskFactors []byte     `db:"risk_factors"`
	AssessedBy  dbNullUUID `db:"assessed_by"`
	Notes       string     `db:"notes"`
	AssessedAt  dbTime     `db:"assessed_at"`
	CreatedAt   dbTime     `db:"created_at"`
	UpdatedAt   dbTime     `db:"updated_at"`
}

func toRiskAssessment(r riskRow) (domain.RiskAssessment, error) {
	id, err := parseUUID(r.ID)
	if err != nil {
		return domain.RiskAssessment{}, fmt.Errorf("repo: risk – parse id: %w", err)
	}
	entityID, err := parseUUID(r.EntityID)
	if err != nil {
		return domain.RiskAssessment{}, fmt.Errorf("repo: risk – parse entity_id: %w", err)
	}

	var factors map[string]any
	if len(r.RiskFactors) > 0 {
		if err := json.Unmarshal(r.RiskFactors, &factors); err != nil {
			factors = map[string]any{}
		}
	}

	return domain.RiskAssessment{
		ID:          id,
		EntityType:  r.EntityType,
		EntityID:    entityID,
		RiskLevel:   domain.RiskLevel(r.RiskLevel),
		RiskScore:   r.RiskScore,
		RiskFactors: factors,
		AssessedBy:  r.AssessedBy.Ptr(),
		Notes:       r.Notes,
		AssessedAt:  r.AssessedAt.Time,
		CreatedAt:   r.CreatedAt.Time,
		UpdatedAt:   r.UpdatedAt.Time,
	}, nil
}

func (r *riskRepository) Create(ctx context.Context, ra *domain.RiskAssessment) error {
	factors, err := json.Marshal(ra.RiskFactors)
	if err != nil {
		factors = []byte("{}")
	}

	const q = `
		INSERT INTO risk_assessments (
			id, entity_type, entity_id, risk_level, risk_score,
			risk_factors, assessed_by, notes, assessed_at,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
		RETURNING id`

	var returnedID string
	if err := r.db.QueryRowxContext(ctx, q,
		ra.ID, ra.EntityType, ra.EntityID,
		string(ra.RiskLevel), ra.RiskScore,
		factors, ra.AssessedBy, ra.Notes, ra.AssessedAt,
	).Scan(&returnedID); err != nil {
		return fmt.Errorf("repo: risk create: %w", err)
	}

	id, err := parseUUID(returnedID)
	if err != nil {
		return fmt.Errorf("repo: risk create – parse id: %w", err)
	}
	ra.ID = id
	return nil
}

func (r *riskRepository) FindLatestByEntity(ctx context.Context, entityType string, entityID uuid.UUID) (*domain.RiskAssessment, error) {
	const q = `
		SELECT id, entity_type, entity_id, risk_level, risk_score,
		       risk_factors, assessed_by, notes, assessed_at, created_at, updated_at
		FROM risk_assessments
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY assessed_at DESC
		LIMIT 1`

	var row riskRow
	if err := r.db.QueryRowxContext(ctx, q, entityType, entityID).StructScan(&row); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repo: risk find latest: %w", err)
	}

	ra, err := toRiskAssessment(row)
	if err != nil {
		return nil, err
	}
	return &ra, nil
}

func (r *riskRepository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.RiskAssessment, error) {
	const q = `
		SELECT id, entity_type, entity_id, risk_level, risk_score,
		       risk_factors, assessed_by, notes, assessed_at, created_at, updated_at
		FROM risk_assessments
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY assessed_at DESC`

	rows, err := r.db.QueryxContext(ctx, q, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("repo: risk list by entity: %w", err)
	}
	defer rows.Close()

	var result []domain.RiskAssessment
	for rows.Next() {
		var row riskRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("repo: risk list – scan: %w", err)
		}
		ra, err := toRiskAssessment(row)
		if err != nil {
			return nil, err
		}
		result = append(result, ra)
	}
	return result, rows.Err()
}

func (r *riskRepository) UpdateKYCRisk(ctx context.Context, kycID uuid.UUID, level domain.RiskLevel, score int) error {
	const q = `UPDATE kyc_verifications SET risk_level=$1, risk_score=$2, updated_at=NOW() WHERE id=$3`
	_, err := r.db.ExecContext(ctx, q, string(level), score, kycID)
	if err != nil {
		return fmt.Errorf("repo: update kyc risk: %w", err)
	}
	return nil
}

func (r *riskRepository) UpdateKYBRisk(ctx context.Context, kybID uuid.UUID, level domain.RiskLevel, score int) error {
	const q = `UPDATE kyb_verifications SET risk_level=$1, risk_score=$2, updated_at=NOW() WHERE id=$3`
	_, err := r.db.ExecContext(ctx, q, string(level), score, kybID)
	if err != nil {
		return fmt.Errorf("repo: update kyb risk: %w", err)
	}
	return nil
}
