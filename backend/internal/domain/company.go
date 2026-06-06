package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CompanyStatus string

const (
	CompanyStatusActive   CompanyStatus = "active"
	CompanyStatusInactive CompanyStatus = "inactive"
	CompanyStatusPending  CompanyStatus = "pending"
)

type Company struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Name           string
	LegalName      string
	RegistrationNo string
	TaxID          string
	Industry       string
	Address        string
	City           string
	Province       string
	PostalCode     string
	Country        string
	PhoneNumber    string
	Email          string
	Website        string
	Status         CompanyStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type CompanyRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Company, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*Company, error)
	FindByRegistrationNo(ctx context.Context, registrationNo string) (*Company, error)
	FindAll(ctx context.Context, params ListParams) ([]Company, int64, error)
	Create(ctx context.Context, company *Company) error
	Update(ctx context.Context, company *Company) error
	Delete(ctx context.Context, id uuid.UUID) error
}
