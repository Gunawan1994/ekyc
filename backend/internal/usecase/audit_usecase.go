package usecase

import (
	"context"
	"fmt"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// AuditUsecase defines the application-level operations on AuditLog entities.
type AuditUsecase interface {
	ListAuditLogs(ctx context.Context, params domain.AuditListParams) ([]domain.AuditLog, int64, error)
}

type auditUsecase struct {
	auditRepo domain.AuditRepository
}

// NewAuditUsecase constructs an AuditUsecase with all required dependencies.
func NewAuditUsecase(auditRepo domain.AuditRepository) AuditUsecase {
	return &auditUsecase{auditRepo: auditRepo}
}

// ListAuditLogs returns a paginated, optionally filtered list of audit log entries.
func (uc *auditUsecase) ListAuditLogs(ctx context.Context, params domain.AuditListParams) ([]domain.AuditLog, int64, error) {
	logs, total, err := uc.auditRepo.FindAll(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, total, nil
}
