package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// auditRepository implements domain.AuditRepository backed by PostgreSQL.
type auditRepository struct {
	db *sqlx.DB
}

// NewAuditRepository constructs an auditRepository.
func NewAuditRepository(db *sqlx.DB) domain.AuditRepository {
	return &auditRepository{db: db}
}

// auditRow is the flat scan target for an audit_logs row.
type auditRow struct {
	ID         string `db:"id"`
	ActorID    string `db:"actor_id"`
	ActorEmail string `db:"actor_email"`
	Action     string `db:"action"`
	EntityType string `db:"entity_type"`
	EntityID   string `db:"entity_id"`
	OldValue   string `db:"old_value"`
	NewValue   string `db:"new_value"`
	IPAddress  string `db:"ip_address"`
	UserAgent  string `db:"user_agent"`
	CreatedAt  dbTime `db:"created_at"`
}

// toAuditLog converts a scanned auditRow to the domain type.
func toAuditLog(r auditRow) (domain.AuditLog, error) {
	id, err := parseUUID(r.ID)
	if err != nil {
		return domain.AuditLog{}, fmt.Errorf("repo: audit – parse id: %w", err)
	}
	actorID, err := parseUUID(r.ActorID)
	if err != nil {
		return domain.AuditLog{}, fmt.Errorf("repo: audit – parse actor_id: %w", err)
	}
	entityID, err := parseUUID(r.EntityID)
	if err != nil {
		return domain.AuditLog{}, fmt.Errorf("repo: audit – parse entity_id: %w", err)
	}

	return domain.AuditLog{
		ID:         id,
		ActorID:    actorID,
		ActorEmail: r.ActorEmail,
		Action:     r.Action,
		EntityType: r.EntityType,
		EntityID:   entityID,
		OldValue:   r.OldValue,
		NewValue:   r.NewValue,
		IPAddress:  r.IPAddress,
		UserAgent:  r.UserAgent,
		CreatedAt:  r.CreatedAt.Time,
	}, nil
}

// Create inserts an immutable audit log entry. created_at is set by the
// database via NOW() so that all timestamps originate from a single clock.
func (r *auditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	const q = `
		INSERT INTO audit_logs (
			id, actor_id, actor_email, action,
			entity_type, entity_id,
			old_value, new_value,
			ip_address, user_agent,
			created_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8,
			$9, $10,
			NOW()
		)`

	if _, err := r.db.ExecContext(ctx, q,
		log.ID,
		log.ActorID,
		log.ActorEmail,
		log.Action,
		log.EntityType,
		log.EntityID,
		log.OldValue,
		log.NewValue,
		log.IPAddress,
		log.UserAgent,
	); err != nil {
		return fmt.Errorf("repo: audit create: %w", err)
	}
	return nil
}

// FindByEntity returns audit log entries for the given entity type and id,
// ordered by created_at DESC, paginated via params, plus the total count.
func (r *auditRepository) FindByEntity(
	ctx context.Context,
	entityType string,
	entityID uuid.UUID,
	params domain.ListParams,
) ([]domain.AuditLog, int64, error) {
	limit, offset := pageOffset(params.Page, params.PageSize)

	var total int64
	if err := r.db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE entity_type = $1 AND entity_id = $2`,
		entityType, entityID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: audit find by entity – count: %w", err)
	}

	const q = `
		SELECT id, actor_id, actor_email, action,
		       entity_type, entity_id,
		       old_value, new_value,
		       ip_address, user_agent,
		       created_at
		FROM audit_logs
		WHERE entity_type = $1
		  AND entity_id   = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryxContext(ctx, q, entityType, entityID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: audit find by entity – query: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var row auditRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, fmt.Errorf("repo: audit find by entity – scan: %w", err)
		}
		entry, err := toAuditLog(row)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: audit find by entity – rows: %w", err)
	}

	return logs, total, nil
}

// FindAll returns a paginated list of audit log entries with optional filters
// on actor_id and action, ordered by created_at DESC.
func (r *auditRepository) FindAll(
	ctx context.Context,
	params domain.AuditListParams,
) ([]domain.AuditLog, int64, error) {
	limit, offset := pageOffset(params.Page, params.PageSize)

	// Build WHERE clauses dynamically based on optional filters.
	var (
		conditions []string
		args       []interface{}
		argIdx     = 1
	)

	if params.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, params.ActorID)
		argIdx++
	}
	if params.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, params.Action)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query.
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM audit_logs %s`, where)
	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowxContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repo: audit find all – count: %w", err)
	}

	// Data query with pagination appended.
	dataQ := fmt.Sprintf(`
		SELECT id, actor_id, actor_email, action,
		       entity_type, entity_id,
		       old_value, new_value,
		       ip_address, user_agent,
		       created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryxContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repo: audit find all – query: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var row auditRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, fmt.Errorf("repo: audit find all – scan: %w", err)
		}
		entry, err := toAuditLog(row)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repo: audit find all – rows: %w", err)
	}

	return logs, total, nil
}
