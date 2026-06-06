package usecase

import (
	"context"
	"fmt"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

// UserUsecase defines application-level read operations on User entities.
type UserUsecase interface {
	List(ctx context.Context, params domain.ListParams) ([]domain.User, int64, error)
}

type userUsecase struct {
	userRepo domain.UserRepository
}

// NewUserUsecase constructs a UserUsecase with the required repository.
func NewUserUsecase(userRepo domain.UserRepository) UserUsecase {
	return &userUsecase{userRepo: userRepo}
}

// List returns a paginated slice of users matching the given params.
func (uc *userUsecase) List(ctx context.Context, params domain.ListParams) ([]domain.User, int64, error) {
	users, total, err := uc.userRepo.FindAll(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}
