package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
)

// AuthResponse is returned on successful login or token refresh.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// UserResponse is the public representation of a User.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// CustomerResponse is the public representation of a Customer.
type CustomerResponse struct {
	ID          uuid.UUID  `json:"id"`
	CompanyID   uuid.UUID  `json:"company_id"`
	FullName    string     `json:"full_name"`
	IDType      string     `json:"id_type"`
	IDNumber    string     `json:"id_number"`
	PhoneNumber string     `json:"phone"`
	Email       string     `json:"email"`
	Address     string     `json:"address"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// CompanyResponse is the public representation of a Company.
type CompanyResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	LegalName      string    `json:"legal_name"`
	RegistrationNo string    `json:"registration_number"`
	TaxID          string    `json:"tax_id"`
	Industry       string    `json:"industry"`
	Address        string    `json:"address"`
	City           string    `json:"city"`
	Province       string    `json:"province"`
	PostalCode     string    `json:"postal_code"`
	Country        string    `json:"country"`
	PhoneNumber    string    `json:"phone"`
	Email          string    `json:"email"`
	Website        string    `json:"website"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RiskAssessmentResponse is the public representation of a RiskAssessment.
type RiskAssessmentResponse struct {
	ID          uuid.UUID      `json:"id"`
	EntityType  string         `json:"entity_type"`
	EntityID    uuid.UUID      `json:"entity_id"`
	RiskLevel   string         `json:"risk_level"`
	RiskScore   int            `json:"risk_score"`
	RiskFactors map[string]any `json:"risk_factors"`
	AssessedBy  *uuid.UUID     `json:"assessed_by,omitempty"`
	Notes       string         `json:"notes,omitempty"`
	AssessedAt  time.Time      `json:"assessed_at"`
	CreatedAt   time.Time      `json:"created_at"`
}

// KYCResponse is the public representation of a KYCVerification.
type KYCResponse struct {
	ID              uuid.UUID  `json:"id"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	CustomerName    string     `json:"customer_name,omitempty"`
	ReviewerID      *uuid.UUID `json:"reviewer_id,omitempty"`
	SubmittedBy     uuid.UUID  `json:"submitted_by"`
	Status          string     `json:"status"`
	IDDocumentURL   string     `json:"id_document_url,omitempty"`
	SelfieURL       string     `json:"selfie_url,omitempty"`
	LivenessScore   float64    `json:"liveness_score"`
	FaceMatchScore  float64    `json:"face_match_score"`
	RiskLevel       string     `json:"risk_level"`
	RiskScore       int        `json:"risk_score"`
	RejectionReason string     `json:"rejection_reason,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// KYBResponse is the public representation of a KYBVerification.
type KYBResponse struct {
	ID               uuid.UUID  `json:"id"`
	CompanyID        uuid.UUID  `json:"company_id"`
	CompanyName      string     `json:"company_name,omitempty"`
	ReviewerID       *uuid.UUID `json:"reviewer_id,omitempty"`
	SubmittedBy      uuid.UUID  `json:"submitted_by"`
	Status           string     `json:"status"`
	BusinessDocURL   string     `json:"business_doc_url,omitempty"`
	TaxDocURL        string     `json:"tax_doc_url,omitempty"`
	DirectorIDDocURL string     `json:"director_id_doc_url,omitempty"`
	RiskLevel        string     `json:"risk_level"`
	RiskScore        int        `json:"risk_score"`
	RejectionReason  string     `json:"rejection_reason,omitempty"`
	Notes            string     `json:"notes,omitempty"`
	SubmittedAt      time.Time  `json:"submitted_at"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// AuditLogResponse is the public representation of an AuditLog entry.
type AuditLogResponse struct {
	ID         uuid.UUID `json:"id"`
	ActorID    uuid.UUID `json:"actor_id"`
	ActorEmail string    `json:"actor_email"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// DashboardStatsResponse is the public representation of DashboardStats.
type DashboardStatsResponse struct {
	TotalCustomers   int64 `json:"total_customers"`
	TotalCompanies   int64 `json:"total_companies"`
	TotalKYCPending  int64 `json:"total_kyc_pending"`
	TotalKYCApproved int64 `json:"total_kyc_approved"`
	TotalKYCRejected int64 `json:"total_kyc_rejected"`
	TotalKYBPending  int64 `json:"total_kyb_pending"`
	TotalKYBApproved int64 `json:"total_kyb_approved"`
	TotalKYBRejected int64 `json:"total_kyb_rejected"`
}

// ToAuthResponse maps a LoginOutput to an AuthResponse.
func ToAuthResponse(out *usecase.LoginOutput) AuthResponse {
	u := out.User
	roleName := ""
	if u.Role != nil {
		roleName = string(u.Role.Name)
	}
	return AuthResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		User: UserResponse{
			ID:        u.ID,
			Email:     u.Email,
			FullName:  u.FullName,
			Role:      roleName,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt,
		},
	}
}

// ToCustomerResponse maps a domain.Customer to a CustomerResponse.
func ToCustomerResponse(c *domain.Customer) CustomerResponse {
	return CustomerResponse{
		ID:          c.ID,
		CompanyID:   c.CompanyID,
		FullName:    c.FullName,
		IDType:      string(c.IDType),
		IDNumber:    c.IDNumber,
		PhoneNumber: c.PhoneNumber,
		Email:       c.Email,
		Address:     c.Address,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		DeletedAt:   c.DeletedAt,
	}
}

// ToCustomerResponses maps a slice of domain.Customer to a slice of CustomerResponse.
func ToCustomerResponses(customers []domain.Customer) []CustomerResponse {
	out := make([]CustomerResponse, len(customers))
	for i := range customers {
		out[i] = ToCustomerResponse(&customers[i])
	}
	return out
}

// ToCompanyResponse maps a domain.Company to a CompanyResponse.
func ToCompanyResponse(c *domain.Company) CompanyResponse {
	return CompanyResponse{
		ID:             c.ID,
		Name:           c.Name,
		LegalName:      c.LegalName,
		RegistrationNo: c.RegistrationNo,
		TaxID:          c.TaxID,
		Industry:       c.Industry,
		Address:        c.Address,
		City:           c.City,
		Province:       c.Province,
		PostalCode:     c.PostalCode,
		Country:        c.Country,
		PhoneNumber:    c.PhoneNumber,
		Email:          c.Email,
		Website:        c.Website,
		Status:         string(c.Status),
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// ToCompanyResponses maps a slice of domain.Company to a slice of CompanyResponse.
func ToCompanyResponses(companies []domain.Company) []CompanyResponse {
	out := make([]CompanyResponse, len(companies))
	for i := range companies {
		out[i] = ToCompanyResponse(&companies[i])
	}
	return out
}

// ToRiskAssessmentResponse maps a domain.RiskAssessment to a RiskAssessmentResponse.
func ToRiskAssessmentResponse(r *domain.RiskAssessment) RiskAssessmentResponse {
	return RiskAssessmentResponse{
		ID:          r.ID,
		EntityType:  r.EntityType,
		EntityID:    r.EntityID,
		RiskLevel:   string(r.RiskLevel),
		RiskScore:   r.RiskScore,
		RiskFactors: r.RiskFactors,
		AssessedBy:  r.AssessedBy,
		Notes:       r.Notes,
		AssessedAt:  r.AssessedAt,
		CreatedAt:   r.CreatedAt,
	}
}

// ToKYCResponse maps a domain.KYCVerification to a KYCResponse.
func ToKYCResponse(k *domain.KYCVerification) KYCResponse {
	return KYCResponse{
		ID:              k.ID,
		CustomerID:      k.CustomerID,
		CustomerName:    k.CustomerName,
		ReviewerID:      k.ReviewerID,
		SubmittedBy:     k.SubmittedBy,
		Status:          string(k.Status),
		IDDocumentURL:   k.IDDocumentURL,
		SelfieURL:       k.SelfieURL,
		LivenessScore:   k.LivenessScore,
		FaceMatchScore:  k.FaceMatchScore,
		RiskLevel:       string(k.RiskLevel),
		RiskScore:       k.RiskScore,
		RejectionReason: k.RejectionReason,
		Notes:           k.Notes,
		SubmittedAt:     k.SubmittedAt,
		ReviewedAt:      k.ReviewedAt,
		CreatedAt:       k.CreatedAt,
		UpdatedAt:       k.UpdatedAt,
	}
}

// ToKYCResponses maps a slice of domain.KYCVerification to a slice of KYCResponse.
func ToKYCResponses(records []domain.KYCVerification) []KYCResponse {
	out := make([]KYCResponse, len(records))
	for i := range records {
		out[i] = ToKYCResponse(&records[i])
	}
	return out
}

// ToKYBResponse maps a domain.KYBVerification to a KYBResponse.
func ToKYBResponse(k *domain.KYBVerification) KYBResponse {
	return KYBResponse{
		ID:               k.ID,
		CompanyID:        k.CompanyID,
		CompanyName:      k.CompanyName,
		ReviewerID:       k.ReviewerID,
		SubmittedBy:      k.SubmittedBy,
		Status:           string(k.Status),
		BusinessDocURL:   k.BusinessDocURL,
		TaxDocURL:        k.TaxDocURL,
		DirectorIDDocURL: k.DirectorIDDocURL,
		RiskLevel:        string(k.RiskLevel),
		RiskScore:        k.RiskScore,
		RejectionReason:  k.RejectionReason,
		Notes:            k.Notes,
		SubmittedAt:      k.SubmittedAt,
		ReviewedAt:       k.ReviewedAt,
		CreatedAt:        k.CreatedAt,
		UpdatedAt:        k.UpdatedAt,
	}
}

// ToKYBResponses maps a slice of domain.KYBVerification to a slice of KYBResponse.
func ToKYBResponses(records []domain.KYBVerification) []KYBResponse {
	out := make([]KYBResponse, len(records))
	for i := range records {
		out[i] = ToKYBResponse(&records[i])
	}
	return out
}

// ToDashboardStatsResponse maps a usecase.DashboardStats to a DashboardStatsResponse.
func ToDashboardStatsResponse(s *usecase.DashboardStats) DashboardStatsResponse {
	return DashboardStatsResponse{
		TotalCustomers:   s.TotalCustomers,
		TotalCompanies:   s.TotalCompanies,
		TotalKYCPending:  s.TotalKYCPending,
		TotalKYCApproved: s.TotalKYCApproved,
		TotalKYCRejected: s.TotalKYCRejected,
		TotalKYBPending:  s.TotalKYBPending,
		TotalKYBApproved: s.TotalKYBApproved,
		TotalKYBRejected: s.TotalKYBRejected,
	}
}

// ToAuditLogResponse maps a domain.AuditLog to an AuditLogResponse.
func ToAuditLogResponse(a *domain.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:         a.ID,
		ActorID:    a.ActorID,
		ActorEmail: a.ActorEmail,
		Action:     a.Action,
		EntityType: a.EntityType,
		EntityID:   a.EntityID,
		CreatedAt:  a.CreatedAt,
	}
}

// ToAuditLogResponses maps a slice of domain.AuditLog to a slice of AuditLogResponse.
func ToAuditLogResponses(logs []domain.AuditLog) []AuditLogResponse {
	out := make([]AuditLogResponse, len(logs))
	for i := range logs {
		out[i] = ToAuditLogResponse(&logs[i])
	}
	return out
}
