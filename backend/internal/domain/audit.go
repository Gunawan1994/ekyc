package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditAction classifies the kind of change recorded in an AuditLog entry.
type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionSubmit AuditAction = "submit"
	AuditActionReview AuditAction = "review"
)

// AuditLog records an immutable trail of actions performed on domain entities.
type AuditLog struct {
	ID         uuid.UUID
	ActorID    uuid.UUID
	ActorEmail string
	Action     string
	EntityType string
	EntityID   uuid.UUID
	OldValue   string
	NewValue   string
	IPAddress  string
	UserAgent  string
	CreatedAt  time.Time
}

// AuditListParams extends ListParams with audit-specific filters.
type AuditListParams struct {
	ListParams
	ActorID string
	Action  string
}

type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	FindByEntity(ctx context.Context, entityType string, entityID uuid.UUID, params ListParams) ([]AuditLog, int64, error)
	FindAll(ctx context.Context, params AuditListParams) ([]AuditLog, int64, error)
}
