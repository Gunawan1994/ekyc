package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

const kycEntityType = "kyc_verification"

// SubmitKYCInput carries the data required to open a new KYC verification.
type SubmitKYCInput struct {
	CustomerID    uuid.UUID
	SubmittedBy   uuid.UUID
	Notes         string
	IDDocumentURL string
	SelfieURL     string
}

// ReviewKYCInput carries the data required to approve or reject an existing
// KYC verification.  Status is overwritten by Approve/Reject; callers should
// not set it directly.
type ReviewKYCInput struct {
	ID         uuid.UUID
	ReviewerID uuid.UUID
	Status     domain.VerificationStatus
	Notes      string
}

// SetInReviewKYCInput carries the data required to move a KYC verification
// into the in_review state.
type SetInReviewKYCInput struct {
	ID         uuid.UUID
	ReviewerID uuid.UUID
	Notes      string
}

// RequestAdditionalDocsKYCInput carries the data required to request
// additional documents from the submitter.
type RequestAdditionalDocsKYCInput struct {
	ID         uuid.UUID
	ReviewerID uuid.UUID
	Notes      string
}

// KYCUsecase defines the application-level operations for KYC verifications.
type KYCUsecase interface {
	Submit(ctx context.Context, input SubmitKYCInput) (*domain.KYCVerification, error)
	Approve(ctx context.Context, input ReviewKYCInput) (*domain.KYCVerification, error)
	Reject(ctx context.Context, input ReviewKYCInput) (*domain.KYCVerification, error)
	// SetInReview transitions a pending KYC verification to in_review.
	SetInReview(ctx context.Context, input SetInReviewKYCInput) (*domain.KYCVerification, error)
	// RequestAdditionalDocs transitions a pending or in_review KYC verification
	// to additional_docs_required so the submitter knows to upload more docs.
	RequestAdditionalDocs(ctx context.Context, input RequestAdditionalDocsKYCInput) (*domain.KYCVerification, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.KYCVerification, error)
	List(ctx context.Context, params domain.ListParams) ([]domain.KYCVerification, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type kycUsecase struct {
	kycRepo      domain.KYCRepository
	customerRepo domain.CustomerRepository
	auditRepo    domain.AuditRepository
	riskUC       RiskUsecase
}

// NewKYCUsecase constructs a KYCUsecase with the provided dependencies.
func NewKYCUsecase(
	kycRepo domain.KYCRepository,
	customerRepo domain.CustomerRepository,
	auditRepo domain.AuditRepository,
	riskUC RiskUsecase,
) KYCUsecase {
	return &kycUsecase{
		kycRepo:      kycRepo,
		customerRepo: customerRepo,
		auditRepo:    auditRepo,
		riskUC:       riskUC,
	}
}

// Submit creates a new KYC verification in Pending status for the given
// customer.  It returns domain.ErrNotFound when the customer does not exist.
func (u *kycUsecase) Submit(ctx context.Context, input SubmitKYCInput) (*domain.KYCVerification, error) {
	if _, err := u.customerRepo.FindByID(ctx, input.CustomerID); err != nil {
		return nil, fmt.Errorf("kyc submit – customer lookup: %w", err)
	}

	now := time.Now().UTC()
	kyc := &domain.KYCVerification{
		ID:            uuid.New(),
		CustomerID:    input.CustomerID,
		SubmittedBy:   input.SubmittedBy,
		Status:        domain.VerificationStatusPending,
		Notes:         input.Notes,
		IDDocumentURL: input.IDDocumentURL,
		SelfieURL:     input.SelfieURL,
		SubmittedAt:   now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := u.kycRepo.Create(ctx, kyc); err != nil {
		return nil, fmt.Errorf("kyc submit – create: %w", err)
	}

	// Auto-calculate risk on submit; non-fatal if it fails.
	if ra, err := u.riskUC.CalculateKYCRisk(ctx, CalculateKYCRiskInput{
		KYCID:          kyc.ID,
		IDDocumentURL:  kyc.IDDocumentURL,
		SelfieURL:      kyc.SelfieURL,
		LivenessScore:  kyc.LivenessScore,
		FaceMatchScore: kyc.FaceMatchScore,
	}); err == nil {
		kyc.RiskLevel = ra.RiskLevel
		kyc.RiskScore = ra.RiskScore
	}

	newVal, _ := json.Marshal(kyc)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.SubmittedBy,
		Action:     string(domain.AuditActionSubmit),
		EntityType: kycEntityType,
		EntityID:   kyc.ID,
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyc, nil
}

// Approve transitions a Pending or InReview KYC verification to Approved.
// It returns domain.ErrInvalidStatus when the current status is not Pending or InReview.
func (u *kycUsecase) Approve(ctx context.Context, input ReviewKYCInput) (*domain.KYCVerification, error) {
	input.Status = domain.VerificationStatusApproved
	return u.review(ctx, input)
}

// Reject transitions a Pending or InReview KYC verification to Rejected.
// It returns domain.ErrInvalidStatus when the current status is not Pending or InReview.
func (u *kycUsecase) Reject(ctx context.Context, input ReviewKYCInput) (*domain.KYCVerification, error) {
	input.Status = domain.VerificationStatusRejected
	return u.review(ctx, input)
}

// review is the shared implementation for Approve and Reject.
func (u *kycUsecase) review(ctx context.Context, input ReviewKYCInput) (*domain.KYCVerification, error) {
	kyc, err := u.kycRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("kyc review – find: %w", err)
	}

	if kyc.Status != domain.VerificationStatusPending && kyc.Status != domain.VerificationStatusInReview {
		return nil, fmt.Errorf("kyc review – current status %q: %w", kyc.Status, domain.ErrInvalidStatus)
	}

	oldVal, _ := json.Marshal(kyc)

	if err := u.kycRepo.UpdateStatus(ctx, kyc.ID, input.Status, input.ReviewerID, "", input.Notes); err != nil {
		return nil, fmt.Errorf("kyc review – update status: %w", err)
	}

	// Reflect the update in the returned struct without an extra round-trip.
	now := time.Now().UTC()
	kyc.Status = input.Status
	kyc.ReviewerID = &input.ReviewerID
	kyc.Notes = input.Notes
	kyc.ReviewedAt = &now
	kyc.UpdatedAt = now

	newVal, _ := json.Marshal(kyc)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.ReviewerID,
		Action:     string(domain.AuditActionReview),
		EntityType: kycEntityType,
		EntityID:   kyc.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyc, nil
}

// SetInReview transitions a pending KYC verification to in_review and assigns
// the reviewer.  Returns domain.ErrInvalidStatus when not in pending state.
func (u *kycUsecase) SetInReview(ctx context.Context, input SetInReviewKYCInput) (*domain.KYCVerification, error) {
	kyc, err := u.kycRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("kyc set in review – find: %w", err)
	}

	if kyc.Status != domain.VerificationStatusPending {
		return nil, fmt.Errorf("kyc set in review – current status %q: %w", kyc.Status, domain.ErrInvalidStatus)
	}

	oldVal, _ := json.Marshal(kyc)

	if err := u.kycRepo.UpdateStatusFrom(
		ctx,
		kyc.ID,
		domain.VerificationStatusPending,
		domain.VerificationStatusInReview,
		&input.ReviewerID,
		input.Notes,
	); err != nil {
		return nil, fmt.Errorf("kyc set in review – update status: %w", err)
	}

	now := time.Now().UTC()
	kyc.Status = domain.VerificationStatusInReview
	kyc.ReviewerID = &input.ReviewerID
	kyc.Notes = input.Notes
	kyc.UpdatedAt = now

	newVal, _ := json.Marshal(kyc)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.ReviewerID,
		Action:     string(domain.AuditActionReview),
		EntityType: kycEntityType,
		EntityID:   kyc.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyc, nil
}

// RequestAdditionalDocs transitions a pending or in_review KYC verification to
// additional_docs_required.  Returns domain.ErrInvalidStatus when the current
// status is not pending or in_review.
func (u *kycUsecase) RequestAdditionalDocs(ctx context.Context, input RequestAdditionalDocsKYCInput) (*domain.KYCVerification, error) {
	kyc, err := u.kycRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("kyc request additional docs – find: %w", err)
	}

	if kyc.Status != domain.VerificationStatusPending && kyc.Status != domain.VerificationStatusInReview {
		return nil, fmt.Errorf("kyc request additional docs – current status %q: %w", kyc.Status, domain.ErrInvalidStatus)
	}

	oldVal, _ := json.Marshal(kyc)

	if err := u.kycRepo.UpdateStatusFrom(
		ctx,
		kyc.ID,
		kyc.Status,
		domain.VerificationStatusAdditionalDocsRequired,
		&input.ReviewerID,
		input.Notes,
	); err != nil {
		return nil, fmt.Errorf("kyc request additional docs – update status: %w", err)
	}

	now := time.Now().UTC()
	kyc.Status = domain.VerificationStatusAdditionalDocsRequired
	kyc.ReviewerID = &input.ReviewerID
	kyc.Notes = input.Notes
	kyc.UpdatedAt = now

	newVal, _ := json.Marshal(kyc)
	_ = u.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    input.ReviewerID,
		Action:     string(domain.AuditActionReview),
		EntityType: kycEntityType,
		EntityID:   kyc.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return kyc, nil
}

// GetByID returns a single KYC verification by its ID.
func (u *kycUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.KYCVerification, error) {
	kyc, err := u.kycRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("kyc get by id: %w", err)
	}
	return kyc, nil
}

// List returns a paginated slice of KYC verifications together with the total
// count of matching records.
func (u *kycUsecase) List(ctx context.Context, params domain.ListParams) ([]domain.KYCVerification, int64, error) {
	verifications, total, err := u.kycRepo.FindAll(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("kyc list: %w", err)
	}
	return verifications, total, nil
}

// Delete permanently removes a KYC verification record.
func (u *kycUsecase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := u.kycRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("kyc delete: %w", err)
	}
	return nil
}
