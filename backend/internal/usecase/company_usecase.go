package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

const (
	companyCacheTTL    = 5 * time.Minute
	companyEntityType  = "company"
	companyCachePrefix = "company:"
)

// CreateCompanyInput holds the fields required to register a new company.
type CreateCompanyInput struct {
	UserID             uuid.UUID
	Name               string
	LegalName          string
	RegistrationNumber string
	TaxID              string
	Industry           string
	Address            string
	City               string
	Province           string
	PostalCode         string
	Country            string
	Phone              string
	Email              string
	Website            string
}

// UpdateCompanyInput holds the mutable fields for an existing company.
type UpdateCompanyInput struct {
	ID         uuid.UUID
	Name       string
	Industry   string
	Address    string
	City       string
	Province   string
	PostalCode string
	Country    string
	Phone      string
	Email      string
	Website    string
}

// CompanyUsecase defines the application-level operations on Company entities.
type CompanyUsecase interface {
	Create(ctx context.Context, input CreateCompanyInput, actorID uuid.UUID) (*domain.Company, error)
	Update(ctx context.Context, input UpdateCompanyInput, actorID uuid.UUID) (*domain.Company, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error)
	List(ctx context.Context, params domain.ListParams) ([]domain.Company, int64, error)
	Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
}

// companyCache is the subset of the Redis interface this usecase needs.
type companyCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type companyUsecase struct {
	companyRepo domain.CompanyRepository
	auditRepo   domain.AuditRepository
	cache       companyCache
}

// NewCompanyUsecase constructs a CompanyUsecase with all required dependencies.
func NewCompanyUsecase(
	companyRepo domain.CompanyRepository,
	auditRepo domain.AuditRepository,
	cache companyCache,
) CompanyUsecase {
	return &companyUsecase{
		companyRepo: companyRepo,
		auditRepo:   auditRepo,
		cache:       cache,
	}
}

// Create validates input, enforces RegistrationNumber uniqueness, sets status
// to "pending", persists the company, and writes an audit log entry.
func (uc *companyUsecase) Create(ctx context.Context, input CreateCompanyInput, actorID uuid.UUID) (*domain.Company, error) {
	if err := validateCreateCompanyInput(input); err != nil {
		return nil, err
	}

	// Enforce RegistrationNumber uniqueness via exact match.
	_, err := uc.companyRepo.FindByRegistrationNo(ctx, input.RegistrationNumber)
	if err == nil {
		return nil, domain.ErrAlreadyExists
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("create company: check registration uniqueness: %w", err)
	}

	now := time.Now().UTC()
	company := &domain.Company{
		ID:             uuid.New(),
		UserID:         input.UserID,
		Name:           input.Name,
		LegalName:      input.LegalName,
		RegistrationNo: input.RegistrationNumber,
		TaxID:          input.TaxID,
		Industry:       input.Industry,
		Address:        input.Address,
		City:           input.City,
		Province:       input.Province,
		PostalCode:     input.PostalCode,
		Country:        input.Country,
		PhoneNumber:    input.Phone,
		Email:          input.Email,
		Website:        input.Website,
		Status:         domain.CompanyStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.companyRepo.Create(ctx, company); err != nil {
		return nil, fmt.Errorf("create company: persist: %w", err)
	}

	newVal, _ := json.Marshal(company)
	_ = uc.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    actorID,
		Action:     string(domain.AuditActionCreate),
		EntityType: companyEntityType,
		EntityID:   company.ID,
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return company, nil
}

// Update applies the mutable fields, invalidates the cache entry, and writes
// an audit log with old and new values.
func (uc *companyUsecase) Update(ctx context.Context, input UpdateCompanyInput, actorID uuid.UUID) (*domain.Company, error) {
	if err := validateUpdateCompanyInput(input); err != nil {
		return nil, err
	}

	existing, err := uc.companyRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("update company: find: %w", err)
	}

	oldVal, _ := json.Marshal(existing)

	updated := *existing
	updated.Name = input.Name
	updated.Industry = input.Industry
	updated.Address = input.Address
	updated.City = input.City
	updated.Province = input.Province
	updated.PostalCode = input.PostalCode
	updated.Country = input.Country
	updated.PhoneNumber = input.Phone
	updated.Email = input.Email
	updated.Website = input.Website
	updated.UpdatedAt = time.Now().UTC()

	if err := uc.companyRepo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("update company: persist: %w", err)
	}

	_ = uc.cache.Delete(ctx, companyCacheKey(input.ID))

	newVal, _ := json.Marshal(&updated)
	_ = uc.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    actorID,
		Action:     string(domain.AuditActionUpdate),
		EntityType: companyEntityType,
		EntityID:   updated.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  time.Now().UTC(),
	})

	return &updated, nil
}

// GetByID returns a company from the Redis cache when available, otherwise
// fetches from the repository and populates the cache with a 5-minute TTL.
func (uc *companyUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	key := companyCacheKey(id)

	if cached, err := uc.cache.Get(ctx, key); err == nil && cached != "" {
		var c domain.Company
		if jsonErr := json.Unmarshal([]byte(cached), &c); jsonErr == nil {
			return &c, nil
		}
	}

	company, err := uc.companyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}

	if serialized, jsonErr := json.Marshal(company); jsonErr == nil {
		_ = uc.cache.Set(ctx, key, string(serialized), companyCacheTTL)
	}

	return company, nil
}

// List returns a paginated slice of companies matching the given params.
func (uc *companyUsecase) List(ctx context.Context, params domain.ListParams) ([]domain.Company, int64, error) {
	companies, total, err := uc.companyRepo.FindAll(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list companies: %w", err)
	}
	return companies, total, nil
}

// Delete soft-deletes a company, invalidates its cache entry, and writes an
// audit log entry.
func (uc *companyUsecase) Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	if _, err := uc.companyRepo.FindByID(ctx, id); err != nil {
		return fmt.Errorf("delete company: find: %w", err)
	}

	if err := uc.companyRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete company: persist: %w", err)
	}

	_ = uc.cache.Delete(ctx, companyCacheKey(id))

	_ = uc.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    actorID,
		Action:     string(domain.AuditActionDelete),
		EntityType: companyEntityType,
		EntityID:   id,
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}

func companyCacheKey(id uuid.UUID) string {
	return companyCachePrefix + id.String()
}

func validateCreateCompanyInput(input CreateCompanyInput) error {
	if input.UserID == uuid.Nil {
		return fmt.Errorf("%w: user_id is required", domain.ErrInvalidInput)
	}
	if input.Name == "" {
		return fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	if input.LegalName == "" {
		return fmt.Errorf("%w: legal_name is required", domain.ErrInvalidInput)
	}
	if input.RegistrationNumber == "" {
		return fmt.Errorf("%w: registration_number is required", domain.ErrInvalidInput)
	}
	return nil
}

func validateUpdateCompanyInput(input UpdateCompanyInput) error {
	if input.ID == uuid.Nil {
		return fmt.Errorf("%w: id is required", domain.ErrInvalidInput)
	}
	if input.Name == "" {
		return fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	return nil
}
