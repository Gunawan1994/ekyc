package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RoleName string

const (
	// Legacy roles — kept for backward compatibility.
	RoleAdmin    RoleName = "admin"
	RoleCompany  RoleName = "company"
	RoleCustomer RoleName = "customer"

	// New 5-role system.
	RoleSuperAdmin        RoleName = "super_admin"
	RoleRiskAnalyst       RoleName = "risk_analyst"
	RoleComplianceOfficer RoleName = "compliance_officer"
	RoleReviewer          RoleName = "reviewer"
	RoleCompanyUser       RoleName = "company_user"
)

type Role struct {
	ID          uuid.UUID
	Name        RoleName
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type User struct {
	ID           uuid.UUID
	RoleID       uuid.UUID
	Role         *Role
	Email        string
	PasswordHash string
	FullName     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type RoleRepository interface {
	FindByName(ctx context.Context, name RoleName) (*Role, error)
	FindAll(ctx context.Context) ([]Role, error)
}

type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindAll(ctx context.Context, params ListParams) ([]User, int64, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
