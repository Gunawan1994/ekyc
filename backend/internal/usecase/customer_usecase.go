package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

const (
	customerCacheTTL    = 5 * time.Minute
	customerEntityType  = "customer"
	customerCachePrefix = "customer:"
)

// CreateCustomerInput holds the fields required to register a new customer.
type CreateCustomerInput struct {
	CompanyID uuid.UUID
	FullName  string
	IDNumber  string
	IDType    domain.IDType
	Phone     string
	Email     string
	Address   string
}

// UpdateCustomerInput holds the mutable fields for an existing customer.
type UpdateCustomerInput struct {
	ID       uuid.UUID
	FullName string
	Phone    string
	Email    string
	Address  string
}

// CustomerUsecase defines the application-level operations on Customer entities.
type CustomerUsecase interface {
	Create(ctx context.Context, input CreateCustomerInput, actorID uuid.UUID) (*domain.Customer, error)
	Update(ctx context.Context, input UpdateCustomerInput, actorID uuid.UUID) (*domain.Customer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	List(ctx context.Context, params domain.ListParams) ([]domain.Customer, int64, error)
	Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error
}

// customerCache is the subset of the Redis interface this usecase needs.
type customerCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type customerUsecase struct {
	customerRepo domain.CustomerRepository
	auditRepo    domain.AuditRepository
	cache        customerCache
}

// NewCustomerUsecase constructs a CustomerUsecase with all required dependencies.
func NewCustomerUsecase(
	customerRepo domain.CustomerRepository,
	auditRepo domain.AuditRepository,
	cache customerCache,
) CustomerUsecase {
	return &customerUsecase{
		customerRepo: customerRepo,
		auditRepo:    auditRepo,
		cache:        cache,
	}
}

// Create validates input, enforces IDNumber uniqueness within the company,
// persists the customer, and writes an audit log entry.
func (uc *customerUsecase) Create(ctx context.Context, input CreateCustomerInput, actorID uuid.UUID) (*domain.Customer, error) {
	if err := validateCreateCustomerInput(input); err != nil {
		return nil, err
	}

	// Enforce IDNumber uniqueness within the company scope.
	existing, _, err := uc.customerRepo.FindAll(ctx, domain.ListParams{
		Search:   input.IDNumber,
		PageSize: 1,
		Page:     1,
	})
	if err != nil {
		return nil, fmt.Errorf("create customer: check id uniqueness: %w", err)
	}
	for _, c := range existing {
		if c.IDNumber == input.IDNumber && c.CompanyID == input.CompanyID {
			return nil, domain.ErrAlreadyExists
		}
	}

	now := time.Now().UTC()
	customer := &domain.Customer{
		ID:          uuid.New(),
		CompanyID:   input.CompanyID,
		FullName:    input.FullName,
		IDNumber:    input.IDNumber,
		IDType:      input.IDType,
		PhoneNumber: input.Phone,
		Email:       input.Email,
		Address:     input.Address,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("create customer: persist: %w", err)
	}

	newVal, _ := json.Marshal(customer)
	_ = uc.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    actorID,
		Action:     string(domain.AuditActionCreate),
		EntityType: customerEntityType,
		EntityID:   customer.ID,
		NewValue:   string(newVal),
		CreatedAt:  now,
	})

	return customer, nil
}

// Update applies the mutable fields, invalidates the cache entry, and writes
// an audit log with old and new values.
func (uc *customerUsecase) Update(ctx context.Context, input UpdateCustomerInput, actorID uuid.UUID) (*domain.Customer, error) {
	if err := validateUpdateCustomerInput(input); err != nil {
		return nil, err
	}

	existing, err := uc.customerRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("update customer: find: %w", err)
	}

	oldVal, _ := json.Marshal(existing)

	updated := *existing
	updated.FullName = input.FullName
	updated.PhoneNumber = input.Phone
	updated.Email = input.Email
	updated.Address = input.Address
	updated.UpdatedAt = time.Now().UTC()

	if err := uc.customerRepo.Update(ctx, &updated); err != nil {
		return nil, fmt.Errorf("update customer: persist: %w", err)
	}

	_ = uc.cache.Delete(ctx, customerCacheKey(input.ID))

	newVal, _ := json.Marshal(&updated)
	_ = uc.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    actorID,
		Action:     string(domain.AuditActionUpdate),
		EntityType: customerEntityType,
		EntityID:   updated.ID,
		OldValue:   string(oldVal),
		NewValue:   string(newVal),
		CreatedAt:  time.Now().UTC(),
	})

	return &updated, nil
}

// GetByID returns a customer from the Redis cache when available, otherwise
// fetches from the repository and populates the cache with a 5-minute TTL.
func (uc *customerUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	key := customerCacheKey(id)

	if cached, err := uc.cache.Get(ctx, key); err == nil && cached != "" {
		var c domain.Customer
		if jsonErr := json.Unmarshal([]byte(cached), &c); jsonErr == nil {
			return &c, nil
		}
	}

	customer, err := uc.customerRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	if serialized, jsonErr := json.Marshal(customer); jsonErr == nil {
		_ = uc.cache.Set(ctx, key, string(serialized), customerCacheTTL)
	}

	return customer, nil
}

// List returns a paginated slice of customers matching the given params.
func (uc *customerUsecase) List(ctx context.Context, params domain.ListParams) ([]domain.Customer, int64, error) {
	customers, total, err := uc.customerRepo.FindAll(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list customers: %w", err)
	}
	return customers, total, nil
}

// Delete soft-deletes a customer, invalidates its cache entry, and writes an
// audit log entry.
func (uc *customerUsecase) Delete(ctx context.Context, id uuid.UUID, actorID uuid.UUID) error {
	if _, err := uc.customerRepo.FindByID(ctx, id); err != nil {
		return fmt.Errorf("delete customer: find: %w", err)
	}

	if err := uc.customerRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete customer: persist: %w", err)
	}

	_ = uc.cache.Delete(ctx, customerCacheKey(id))

	_ = uc.auditRepo.Create(ctx, &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    actorID,
		Action:     string(domain.AuditActionDelete),
		EntityType: customerEntityType,
		EntityID:   id,
		CreatedAt:  time.Now().UTC(),
	})

	return nil
}

func customerCacheKey(id uuid.UUID) string {
	return customerCachePrefix + id.String()
}

func validateCreateCustomerInput(input CreateCustomerInput) error {
	if input.CompanyID == uuid.Nil {
		return fmt.Errorf("%w: company_id is required", domain.ErrInvalidInput)
	}
	if input.FullName == "" {
		return fmt.Errorf("%w: full_name is required", domain.ErrInvalidInput)
	}
	if input.IDNumber == "" {
		return fmt.Errorf("%w: id_number is required", domain.ErrInvalidInput)
	}
	if input.IDType == "" {
		return fmt.Errorf("%w: id_type is required", domain.ErrInvalidInput)
	}
	switch input.IDType {
	case domain.IDTypeKTP, domain.IDTypePassport, domain.IDTypeSIM:
		// valid
	default:
		return fmt.Errorf("%w: id_type must be one of ktp, passport, sim", domain.ErrInvalidInput)
	}
	return nil
}

func validateUpdateCustomerInput(input UpdateCustomerInput) error {
	if input.ID == uuid.Nil {
		return fmt.Errorf("%w: id is required", domain.ErrInvalidInput)
	}
	if input.FullName == "" {
		return fmt.Errorf("%w: full_name is required", domain.ErrInvalidInput)
	}
	return nil
}
