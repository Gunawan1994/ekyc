package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IDType string

const (
	IDTypeKTP      IDType = "ktp"
	IDTypePassport IDType = "passport"
	IDTypeSIM      IDType = "sim"
)

type Customer struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CompanyID    uuid.UUID
	FullName     string
	DateOfBirth  time.Time
	PlaceOfBirth string
	Gender       string
	Nationality  string
	IDType       IDType
	IDNumber     string
	Address      string
	City         string
	Province     string
	PostalCode   string
	Country      string
	PhoneNumber  string
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type CustomerRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	FindByCompanyID(ctx context.Context, companyID uuid.UUID, params ListParams) ([]Customer, int64, error)
	FindAll(ctx context.Context, params ListParams) ([]Customer, int64, error)
	Create(ctx context.Context, customer *Customer) error
	Update(ctx context.Context, customer *Customer) error
	Delete(ctx context.Context, id uuid.UUID) error
}
