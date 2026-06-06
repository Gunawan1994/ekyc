package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type RiskAssessment struct {
	ID          uuid.UUID
	EntityType  string
	EntityID    uuid.UUID
	RiskLevel   RiskLevel
	RiskScore   int
	RiskFactors map[string]any
	AssessedBy  *uuid.UUID
	Notes       string
	AssessedAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RiskRepository interface {
	Create(ctx context.Context, r *RiskAssessment) error
	FindLatestByEntity(ctx context.Context, entityType string, entityID uuid.UUID) (*RiskAssessment, error)
	ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]RiskAssessment, error)
	UpdateKYCRisk(ctx context.Context, kycID uuid.UUID, level RiskLevel, score int) error
	UpdateKYBRisk(ctx context.Context, kybID uuid.UUID, level RiskLevel, score int) error
}
