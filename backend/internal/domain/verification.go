package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type VerificationStatus string

const (
	VerificationStatusPending                VerificationStatus = "pending"
	VerificationStatusInReview               VerificationStatus = "in_review"
	VerificationStatusApproved               VerificationStatus = "approved"
	VerificationStatusRejected               VerificationStatus = "rejected"
	VerificationStatusAdditionalDocsRequired VerificationStatus = "additional_docs_required"
)

// KYCVerification represents an individual customer identity verification record.
type KYCVerification struct {
	ID              uuid.UUID
	CustomerID      uuid.UUID
	CustomerName    string // denormalized from customers.full_name
	ReviewerID      *uuid.UUID
	SubmittedBy     uuid.UUID
	Status          VerificationStatus
	IDDocumentURL   string
	SelfieURL       string
	LivenessScore   float64
	FaceMatchScore  float64
	RiskLevel       RiskLevel
	RiskScore       int
	RejectionReason string
	Notes           string
	SubmittedAt     time.Time
	ReviewedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// KYBVerification represents a business entity verification record.
type KYBVerification struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	CompanyName      string // denormalized from companies.name
	ReviewerID       *uuid.UUID
	SubmittedBy      uuid.UUID
	Status           VerificationStatus
	BusinessDocURL   string
	TaxDocURL        string
	DirectorIDDocURL string
	RiskLevel        RiskLevel
	RiskScore        int
	RejectionReason  string
	Notes            string
	SubmittedAt      time.Time
	ReviewedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type KYCRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*KYCVerification, error)
	FindByCustomerID(ctx context.Context, customerID uuid.UUID, params ListParams) ([]KYCVerification, int64, error)
	FindAll(ctx context.Context, params ListParams) ([]KYCVerification, int64, error)
	Create(ctx context.Context, verification *KYCVerification) error
	// UpdateStatus transitions a record from pending to the target status.
	UpdateStatus(ctx context.Context, id uuid.UUID, status VerificationStatus, reviewerID uuid.UUID, reason, notes string) error
	// UpdateStatusFrom transitions a record from fromStatus to toStatus.
	UpdateStatusFrom(ctx context.Context, id uuid.UUID, fromStatus, toStatus VerificationStatus, reviewerID *uuid.UUID, notes string) error
	CountByStatus(ctx context.Context, status VerificationStatus) (int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type KYBRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*KYBVerification, error)
	FindByCompanyID(ctx context.Context, companyID uuid.UUID, params ListParams) ([]KYBVerification, int64, error)
	FindAll(ctx context.Context, params ListParams) ([]KYBVerification, int64, error)
	Create(ctx context.Context, verification *KYBVerification) error
	// UpdateStatus transitions a record from pending to the target status.
	UpdateStatus(ctx context.Context, id uuid.UUID, status VerificationStatus, reviewerID uuid.UUID, reason, notes string) error
	// UpdateStatusFrom transitions a record from fromStatus to toStatus.
	UpdateStatusFrom(ctx context.Context, id uuid.UUID, fromStatus, toStatus VerificationStatus, reviewerID *uuid.UUID, notes string) error
	CountByStatus(ctx context.Context, status VerificationStatus) (int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
