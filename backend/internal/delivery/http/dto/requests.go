package dto

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type CreateCustomerRequest struct {
	CompanyID string `json:"company_id" validate:"required,uuid"`
	FullName  string `json:"full_name" validate:"required,min=2,max=255"`
	IDNumber  string `json:"id_number" validate:"required"`
	IDType    string `json:"id_type" validate:"required,oneof=ktp passport sim"`
	Phone     string `json:"phone" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Address   string `json:"address" validate:"required"`
}

type UpdateCustomerRequest struct {
	FullName string `json:"full_name" validate:"required,min=2"`
	Phone    string `json:"phone" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Address  string `json:"address" validate:"required"`
}

type CreateCompanyRequest struct {
	Name               string `json:"name" validate:"required"`
	RegistrationNumber string `json:"registration_number" validate:"required"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email" validate:"required,email"`
}

type UpdateCompanyRequest struct {
	Name    string `json:"name" validate:"required"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
	Email   string `json:"email" validate:"required,email"`
}

type SubmitKYCRequest struct {
	CustomerID    string `json:"customer_id" validate:"required,uuid"`
	Notes         string `json:"notes"`
	IDDocumentURL string `json:"id_document_url"`
	SelfieURL     string `json:"selfie_url"`
}

type ReviewKYCRequest struct {
	Notes string `json:"notes"`
}

type SubmitKYBRequest struct {
	CompanyID        string `json:"company_id" validate:"required,uuid"`
	Notes            string `json:"notes"`
	BusinessDocURL   string `json:"business_doc_url"`
	TaxDocURL        string `json:"tax_doc_url"`
	DirectorIDDocURL string `json:"director_id_doc_url"`
}

type ReviewKYBRequest struct {
	Notes string `json:"notes"`
}

// SetInReviewRequest is the request body for the set-in-review action.
// Notes is optional; it lets the reviewer leave a comment when starting review.
type SetInReviewRequest struct {
	Notes string `json:"notes"`
}

// RequestDocsRequest is the request body for requesting additional documents.
// Notes is required so the reviewer communicates what documents are needed.
type RequestDocsRequest struct {
	Notes string `json:"notes" validate:"required"`
}

type ManualRiskOverrideRequest struct {
	RiskLevel string `json:"risk_level" validate:"required,oneof=low medium high critical"`
	Notes     string `json:"notes"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}
