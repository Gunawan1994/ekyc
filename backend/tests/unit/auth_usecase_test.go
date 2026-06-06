package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	jwtpkg "github.com/monarchintiteknologi/ekyc-platform/internal/pkg/jwt"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
)

// ---------------------------------------------------------------------------
// Mock: domain.UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindAll(ctx context.Context, params domain.ListParams) ([]domain.User, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]domain.User), args.Get(1).(int64), args.Error(2)
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// ---------------------------------------------------------------------------
// Mock: usecase.TokenRepository
// ---------------------------------------------------------------------------

type mockTokenRepo struct {
	mock.Mock
}

func (m *mockTokenRepo) StoreRefreshToken(ctx context.Context, userID, tokenID uuid.UUID, hash string, expiry time.Duration) error {
	return m.Called(ctx, userID, tokenID, hash, expiry).Error(0)
}

func (m *mockTokenRepo) ValidateRefreshToken(ctx context.Context, userID, tokenID uuid.UUID, hash string) error {
	return m.Called(ctx, userID, tokenID, hash).Error(0)
}

func (m *mockTokenRepo) RevokeRefreshToken(ctx context.Context, userID, tokenID uuid.UUID) error {
	return m.Called(ctx, userID, tokenID).Error(0)
}

func (m *mockTokenRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return m.Called(ctx, userID).Error(0)
}

// ---------------------------------------------------------------------------
// Mock: domain.AuditRepository (shared across all auth tests)
// ---------------------------------------------------------------------------

type mockAuditRepo struct {
	mock.Mock
}

func (m *mockAuditRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	return m.Called(ctx, log).Error(0)
}

func (m *mockAuditRepo) FindByEntity(ctx context.Context, entityType string, entityID uuid.UUID, params domain.ListParams) ([]domain.AuditLog, int64, error) {
	args := m.Called(ctx, entityType, entityID, params)
	return args.Get(0).([]domain.AuditLog), args.Get(1).(int64), args.Error(2)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newJWTManager returns a JWT manager suitable for tests (short expiries).
func newJWTManager() *jwtpkg.Manager {
	return jwtpkg.NewManager("test-secret-key-for-unit-tests", 15*time.Minute, 7*24*time.Hour)
}

// bcryptHash produces a bcrypt hash of password at minimum cost (fastest for tests).
func bcryptHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcryptHash: %v", err)
	}
	return string(hash)
}

// ---------------------------------------------------------------------------
// TestLogin_Success
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	password := "correct-password"

	user := &domain.User{
		ID:           userID,
		Email:        "alice@example.com",
		PasswordHash: bcryptHash(t, password),
		IsActive:     true,
	}

	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	auditRepo := new(mockAuditRepo)
	jwtMgr := newJWTManager()

	userRepo.On("FindByEmail", ctx, user.Email).Return(user, nil)
	tokenRepo.On(
		"StoreRefreshToken",
		ctx,
		userID,
		mock.AnythingOfType("uuid.UUID"),
		mock.AnythingOfType("string"),
		jwtMgr.RefreshTokenExpiry(),
	).Return(nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).Return(nil)

	uc := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtMgr, auditRepo)

	// Act
	out, err := uc.Login(ctx, usecase.LoginInput{Email: user.Email, Password: password})

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assert.Equal(t, user, out.User)
	userRepo.AssertExpectations(t)
	tokenRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// TestLogin_WrongPassword
// ---------------------------------------------------------------------------

func TestLogin_WrongPassword(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uuid.New()

	user := &domain.User{
		ID:           userID,
		Email:        "alice@example.com",
		PasswordHash: bcryptHash(t, "correct-password"),
		IsActive:     true,
	}

	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	auditRepo := new(mockAuditRepo)

	userRepo.On("FindByEmail", ctx, user.Email).Return(user, nil)

	uc := usecase.NewAuthUsecase(userRepo, tokenRepo, newJWTManager(), auditRepo)

	// Act
	out, err := uc.Login(ctx, usecase.LoginInput{Email: user.Email, Password: "wrong-password"})

	// Assert
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Nil(t, out)
	tokenRepo.AssertNotCalled(t, "StoreRefreshToken")
}

// ---------------------------------------------------------------------------
// TestLogin_UserNotFound – must not leak user-existence via error type
// ---------------------------------------------------------------------------

func TestLogin_UserNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()

	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	auditRepo := new(mockAuditRepo)

	userRepo.On("FindByEmail", ctx, "ghost@example.com").Return(nil, domain.ErrNotFound)

	uc := usecase.NewAuthUsecase(userRepo, tokenRepo, newJWTManager(), auditRepo)

	// Act
	out, err := uc.Login(ctx, usecase.LoginInput{Email: "ghost@example.com", Password: "any"})

	// Assert – caller receives ErrInvalidCredentials; ErrNotFound must not surface
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.NotErrorIs(t, err, domain.ErrNotFound, "user existence must not leak to the caller")
	assert.Nil(t, out)
}

// ---------------------------------------------------------------------------
// TestLogin_InactiveUser
// ---------------------------------------------------------------------------

func TestLogin_InactiveUser(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uuid.New()

	user := &domain.User{
		ID:           userID,
		Email:        "inactive@example.com",
		PasswordHash: bcryptHash(t, "password"),
		IsActive:     false,
	}

	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	auditRepo := new(mockAuditRepo)

	userRepo.On("FindByEmail", ctx, user.Email).Return(user, nil)

	uc := usecase.NewAuthUsecase(userRepo, tokenRepo, newJWTManager(), auditRepo)

	// Act
	out, err := uc.Login(ctx, usecase.LoginInput{Email: user.Email, Password: "password"})

	// Assert – inactive account must not reveal its status to the caller
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Nil(t, out)
	tokenRepo.AssertNotCalled(t, "StoreRefreshToken")
}

// ---------------------------------------------------------------------------
// TestRefreshToken_Success
// ---------------------------------------------------------------------------

func TestRefreshToken_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	jwtMgr := newJWTManager()

	// Issue a real refresh token so we have a valid signed string.
	rawRefresh, err := jwtMgr.GenerateRefreshToken(userID)
	assert.NoError(t, err)

	refreshClaims, err := jwtMgr.ExtractClaims(rawRefresh)
	assert.NoError(t, err)

	jti, err := uuid.Parse(refreshClaims.ID)
	assert.NoError(t, err)

	user := &domain.User{ID: userID, Email: "bob@example.com", IsActive: true}

	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	auditRepo := new(mockAuditRepo)

	tokenRepo.On("ValidateRefreshToken", ctx, userID, jti, mock.AnythingOfType("string")).Return(nil)
	tokenRepo.On("RevokeRefreshToken", ctx, userID, jti).Return(nil)
	tokenRepo.On(
		"StoreRefreshToken",
		ctx,
		userID,
		mock.AnythingOfType("uuid.UUID"),
		mock.AnythingOfType("string"),
		jwtMgr.RefreshTokenExpiry(),
	).Return(nil)
	userRepo.On("FindByID", ctx, userID).Return(user, nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).Return(nil)

	uc := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtMgr, auditRepo)

	// Act
	out, err := uc.RefreshToken(ctx, rawRefresh)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, out)
	assert.NotEmpty(t, out.AccessToken)
	assert.NotEmpty(t, out.RefreshToken)
	assert.NotEqual(t, rawRefresh, out.RefreshToken, "rotated refresh token must differ from the original")
	tokenRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// TestRefreshToken_Revoked
// ---------------------------------------------------------------------------

func TestRefreshToken_Revoked(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	jwtMgr := newJWTManager()

	rawRefresh, err := jwtMgr.GenerateRefreshToken(userID)
	assert.NoError(t, err)

	refreshClaims, err := jwtMgr.ExtractClaims(rawRefresh)
	assert.NoError(t, err)

	jti, err := uuid.Parse(refreshClaims.ID)
	assert.NoError(t, err)

	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	auditRepo := new(mockAuditRepo)

	// Token store reports the token has already been revoked.
	tokenRepo.On("ValidateRefreshToken", ctx, userID, jti, mock.AnythingOfType("string")).Return(domain.ErrTokenRevoked)

	uc := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtMgr, auditRepo)

	// Act
	out, err := uc.RefreshToken(ctx, rawRefresh)

	// Assert
	assert.ErrorIs(t, err, domain.ErrTokenRevoked)
	assert.Nil(t, out)
	tokenRepo.AssertNotCalled(t, "StoreRefreshToken")
}

// ---------------------------------------------------------------------------
// TestLogout_Success
// ---------------------------------------------------------------------------

func TestLogout_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	userID := uuid.New()
	jwtMgr := newJWTManager()

	rawRefresh, err := jwtMgr.GenerateRefreshToken(userID)
	assert.NoError(t, err)

	refreshClaims, err := jwtMgr.ExtractClaims(rawRefresh)
	assert.NoError(t, err)

	jti, err := uuid.Parse(refreshClaims.ID)
	assert.NoError(t, err)

	userRepo := new(mockUserRepo)
	tokenRepo := new(mockTokenRepo)
	auditRepo := new(mockAuditRepo)

	tokenRepo.On("RevokeRefreshToken", ctx, userID, jti).Return(nil)
	auditRepo.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).Return(nil)

	uc := usecase.NewAuthUsecase(userRepo, tokenRepo, jwtMgr, auditRepo)

	// Act
	err = uc.Logout(ctx, userID, rawRefresh)

	// Assert
	assert.NoError(t, err)
	tokenRepo.AssertCalled(t, "RevokeRefreshToken", ctx, userID, jti)
}
