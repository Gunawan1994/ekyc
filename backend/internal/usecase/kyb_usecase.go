package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

const kybEntityType = "kyb_verification"

// SubmitKYBInput carries the data required to open a new KYB verification.
type SubmitKYBInput struct {
	CompanyID        uuid.UUID
	SubmittedBy      uuid.UUID
	Notes            string
	BusinessDocURL   string
	TaxDocURL        string
	DirectorIDDocURL string
}

// ReviewKYBInput carries the data required to approve or reject an existing
// KYB verification.  Status is overwritten by Approve/Reject; callers should
// not set it directly.
type ReviewKYBInput struct {
	ID         uuid.UUID
	ReviewerID uuid.UUID
	Status     domain.VerificationStatus
	Notes      string
}

// SetInReviewKYBInput carries the data required to move a KYB verification
// into the in_review state.
type SetInReviewKYBInput struct {
	ID         uuid.UUID
	ReviewerID uuid.UUID
	Notes      string
}

// RequestAdditionalDocsKYBInput carries the data required to request
// additional documents from the submitter.
type RequestAdditionalDocsKYBInput struct {
	ID         uuid.UUID
	ReviewerID uuid.UUID
	Notes      string
}

// KYBUsecase defines the application-level operations for KYB verifications.
type KYBUsecase interface {
	Submit(ctx context.Context, input SubmitKYBInput) (*domain.KYBVerification, error)
	Approve(ctx context.Context, input ReviewKYBInput) (*domain.KYBVerification, error)
	Reject(ctx context.Context, input ReviewKYBInput) (*domain.KYBVerification, error)
	// SetInReview transitions a pending KYB verification to in_review.
	SetInReview(ctx context.Context, input SetInReviewKYBInput) (*domain.KYBVerification, error)
	// RequestAdditionalDocs transitions a pending or in_review KYB verification
	// to additional_docs_required so the submitter knows to upload more docs.
	RequestAdditionalDocs(ctx context.Context, input RequestAdditionalDocsKYBInput) (*domain.KYBVerification, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.KYBVerification, error)
	List(ctx context.Context, params domain.ListParams) ([]domain.KYBVerification, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type kybUsecase struct {
	kybRepo     domain.KYBRepository
	companyRepo domain.CompanyRepository
	auditRepo   domain.AuditRepository
	riskUC      RiskUsecase
}

// NewKYBUsecase constructs a KYBUsecase with the provided dependencies.
func NewKYBUsecase(
	kybRepo domain.KYBRepository,
	companyRepo domain.CompanyRepository,
	auditRepo domain.AuditRepository,
	riskUC RiskUsecase,
) KYBUsecase {
	return &kybUsecase{
		kybRepo:     kybRepo,
		companyRepo: companyRepo,
		auditRepo:   auditRepo,
		riskUC:      riskUC,
	}
}

// Submit creates a new KYB verification in Pending status for the given
// company.  It returns domain.ErrNotFound when the company does not exist.
func (u *kybUsecase) Submit(ctx context.Context, input SubmitKYBInput) (*domain.KYBVerification, error) {
	company, err := u.companyRepo.FindByID(ctx, input.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("kyb submit – company lookup: %w", err)
	}

	now := time.Now().UTC()
	kyb := &domain.KYBVerification{
		ID:               uuid.New(),
		CompanyID:        input.CompanyID,
		SubmittedBy:      input.SubmittedBy,
		Status:           domain.VerificationStatusPending,
		Notes:            input.Notes,
		BusinessDocURL:   input.BusinessDocURL,
		TaxDocURL:        input.TaxDocURL,
		DirectorIDDocURL: input.DirectorIDDocURL,
		SubmittedAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := u.kybRepo.Create(ctx, kyb); err != nil {
		return nil, fmt.Errorf("kyb submit – create: %w", err)
	}

	// Auto-calculate risk on submit; non-fatal if it fails.
	if ra, err := u.riskUC.CalculateKYBRisk(ctx, CalculateKYBRiskInput{
		KYBID:            kyb.ID,
		BusinessDocURL:   kyb.BusinessDocURL,
		TaxDocURL:        kyb.TaxDocURL,
		DirectorIDDocURL: kyb.DirectorIDDocURL,
		CompanyCreatedAt: company.CreatedAt,
		CompanyIndustry:  company.Industry,
		CompanyStatus:    string(company.Status),
	}); err == nil {
		kyb.RiskLevel = ra.RiskLevel
		kyb.RiskScore = ra.RiskScore
	}

	newVal, _ := json.Marshal(kyb)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.SubmittedBy,
		Action:     string(domain.AuditActionSubmit),
		EntityType: kybEntityType,
		EntityID:   kyb.ID,
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyb, nil
}

// Approve transitions a Pending or InReview KYB verification to Approved.
// It returns domain.ErrInvalidStatus when the current status is not Pending or InReview.
func (u *kybUsecase) Approve(ctx context.Context, input ReviewKYBInput) (*domain.KYBVerification, error) {
	input.Status = domain.VerificationStatusApproved
	return u.review(ctx, input)
}

// Reject transitions a Pending or InReview KYB verification to Rejected.
// It returns domain.ErrInvalidStatus when the current status is not Pending or InReview.
func (u *kybUsecase) Reject(ctx context.Context, input ReviewKYBInput) (*domain.KYBVerification, error) {
	input.Status = domain.VerificationStatusRejected
	return u.review(ctx, input)
}

// review is the shared implementation for Approve and Reject.
func (u *kybUsecase) review(ctx context.Context, input ReviewKYBInput) (*domain.KYBVerification, error) {
	kyb, err := u.kybRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("kyb review – find: %w", err)
	}

	if kyb.Status != domain.VerificationStatusPending && kyb.Status != domain.VerificationStatusInReview {
		return nil, fmt.Errorf("kyb review – current status %q: %w", kyb.Status, domain.ErrInvalidStatus)
	}

	oldVal, _ := json.Marshal(kyb)

	if err := u.kybRepo.UpdateStatus(ctx, kyb.ID, input.Status, input.ReviewerID, "", input.Notes); err != nil {
		return nil, fmt.Errorf("kyb review – update status: %w", err)
	}

	// Reflect the update in the returned struct without an extra round-trip.
	now := time.Now().UTC()
	kyb.Status = input.Status
	kyb.ReviewerID = &input.ReviewerID
	kyb.Notes = input.Notes
	kyb.ReviewedAt = &now
	kyb.UpdatedAt = now

	newVal, _ := json.Marshal(kyb)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.ReviewerID,
		Action:     string(domain.AuditActionReview),
		EntityType: kybEntityType,
		EntityID:   kyb.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyb, nil
}

// SetInReview transitions a pending KYB verification to in_review and assigns
// the reviewer.  Returns domain.ErrInvalidStatus when not in pending state.
func (u *kybUsecase) SetInReview(ctx context.Context, input SetInReviewKYBInput) (*domain.KYBVerification, error) {
	kyb, err := u.kybRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("kyb set in review – find: %w", err)
	}

	if kyb.Status != domain.VerificationStatusPending {
		return nil, fmt.Errorf("kyb set in review – current status %q: %w", kyb.Status, domain.ErrInvalidStatus)
	}

	oldVal, _ := json.Marshal(kyb)

	if err := u.kybRepo.UpdateStatusFrom(
		ctx,
		kyb.ID,
		domain.VerificationStatusPending,
		domain.VerificationStatusInReview,
		&input.ReviewerID,
		input.Notes,
	); err != nil {
		return nil, fmt.Errorf("kyb set in review – update status: %w", err)
	}

	now := time.Now().UTC()
	kyb.Status = domain.VerificationStatusInReview
	kyb.ReviewerID = &input.ReviewerID
	kyb.Notes = input.Notes
	kyb.UpdatedAt = now

	newVal, _ := json.Marshal(kyb)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.ReviewerID,
		Action:     string(domain.AuditActionReview),
		EntityType: kybEntityType,
		EntityID:   kyb.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyb, nil
}

// RequestAdditionalDocs transitions a pending or in_review KYB verification to
// additional_docs_required.  Returns domain.ErrInvalidStatus when the current
// status is not pending or in_review.
func (u *kybUsecase) RequestAdditionalDocs(ctx context.Context, input RequestAdditionalDocsKYBInput) (*domain.KYBVerification, error) {
	kyb, err := u.kybRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("kyb request additional docs – find: %w", err)
	}

	if kyb.Status != domain.VerificationStatusPending && kyb.Status != domain.VerificationStatusInReview {
		return nil, fmt.Errorf("kyb request additional docs – current status %q: %w", kyb.Status, domain.ErrInvalidStatus)
	}

	oldVal, _ := json.Marshal(kyb)

	if err := u.kybRepo.UpdateStatusFrom(
		ctx,
		kyb.ID,
		kyb.Status,
		domain.VerificationStatusAdditionalDocsRequired,
		&input.ReviewerID,
		input.Notes,
	); err != nil {
		return nil, fmt.Errorf("kyb request additional docs – update status: %w", err)
	}

	now := time.Now().UTC()
	kyb.Status = domain.VerificationStatusAdditionalDocsRequired
	kyb.ReviewerID = &input.ReviewerID
	kyb.Notes = input.Notes
	kyb.UpdatedAt = now

	newVal, _ := json.Marshal(kyb)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.ReviewerID,
		Action:     string(domain.AuditActionReview),
		EntityType: kybEntityType,
		EntityID:   kyb.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyb, nil
}

// GetByID returns a single KYB verification by its ID.
func (u *kybUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.KYBVerification, error) {
	kyb, err := u.kybRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("kyb get by id: %w", err)
	}
	return kyb, nil
}

// List returns a paginated slice of KYB verifications together with the total
// count of matching records.
func (u *kybUsecase) List(ctx context.Context, params domain.ListParams) ([]domain.KYBVerification, int64, error) {
	verifications, total, err := u.kybRepo.FindAll(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("kyb list: %w", err)
	}
	return verifications, total, nil
}

// Delete permanently removes a KYB verification record.
func (u *kybUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := u.kybRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("kyb delete: %w", err)
	}
	return nil
}
