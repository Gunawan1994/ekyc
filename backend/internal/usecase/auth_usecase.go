package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	jwtpkg "github.com/monarchintiteknologi/ekyc-platform/internal/pkg/jwt"
)

// TokenRepository manages refresh-token state in an external store (Redis).
// Keys are scoped to refresh:{userID}:{jti} so that per-device revocation is
// possible without affecting unrelated sessions.
type TokenRepository interface {
	// StoreRefreshToken persists a SHA-256 hash of the raw refresh token.
	// hash is hex(sha256(rawToken)).
	StoreRefreshToken(ctx context.Context, userID, tokenID uuid.UUID, hash string, expiry time.Duration) error

	// ValidateRefreshToken returns nil only when the stored hash matches the
	// provided hash. It returns domain.ErrTokenRevoked when the key is absent.
	ValidateRefreshToken(ctx context.Context, userID, tokenID uuid.UUID, hash string) error

	// RevokeRefreshToken deletes a single session token.
	RevokeRefreshToken(ctx context.Context, userID, tokenID uuid.UUID) error

	// RevokeAllUserTokens deletes every refresh token belonging to userID.
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
}

// PasswordResetRepository manages password-reset tokens in an external store.
type PasswordResetRepository interface {
	// StorePasswordResetToken persists a one-time reset token for the given email.
	StorePasswordResetToken(ctx context.Context, email, token string, expiry time.Duration) error

	// GetPasswordResetToken retrieves the stored token. Returns ("", nil) when absent.
	GetPasswordResetToken(ctx context.Context, email string) (string, error)

	// DeletePasswordResetToken removes the token after successful use.
	DeletePasswordResetToken(ctx context.Context, email string) error
}

// LoginInput carries the credentials submitted by the caller.
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput is returned on successful authentication or token rotation.
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
}

// AuthUsecase defines the authentication contract.
type AuthUsecase interface {
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginOutput, error)
	Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	ForgotPassword(ctx context.Context, email string) (resetToken string, err error)
	ResetPassword(ctx context.Context, email, token, newPassword string) error
}

type authUsecase struct {
	userRepo      domain.UserRepository
	tokenRepo     TokenRepository
	passwordReset PasswordResetRepository
	jwtManager    *jwtpkg.Manager
	auditRepo     domain.AuditRepository
}

// NewAuthUsecase constructs an AuthUsecase with all required dependencies.
func NewAuthUsecase(
	userRepo domain.UserRepository,
	tokenRepo TokenRepository,
	passwordReset PasswordResetRepository,
	jwtManager *jwtpkg.Manager,
	auditRepo domain.AuditRepository,
) AuthUsecase {
	return &authUsecase{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		passwordReset: passwordReset,
		jwtManager:    jwtManager,
		auditRepo:     auditRepo,
	}
}

// Login authenticates a user by email and password.
//
// Security invariants:
//   - Only domain.ErrInvalidCredentials is returned to the caller regardless
//     of which specific check fails (user not found, inactive, wrong password).
//     This prevents user enumeration.
//   - When the user record is not found a dummy bcrypt comparison is executed
//     so that the response time profile matches the happy path.
//   - Password comparison uses bcrypt.CompareHashAndPassword which is
//     intrinsically constant-time at the algorithm level.
func (a *authUsecase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	user, err := a.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		// Run a dummy comparison to keep the timing profile consistent with the
		// case where the user exists but provides a wrong password.
		_ = runDummyBcrypt()
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsActive {
		_ = runDummyBcrypt()
		return nil, domain.ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	out, err := a.issueTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	a.writeAuditLog(ctx, user.ID, user.Email, "login")

	return out, nil
}

// RefreshToken validates an existing refresh token, revokes it, and issues a
// fresh access+refresh pair (token rotation). The old refresh token is
// invalidated before the new pair is issued so that a replayed stolen token
// cannot be used indefinitely.
func (a *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*LoginOutput, error) {
	claims, err := a.jwtManager.ExtractClaims(refreshToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	tokenHash := hashToken(refreshToken)

	if err = a.tokenRepo.ValidateRefreshToken(ctx, claims.UserID, jti, tokenHash); err != nil {
		return nil, err
	}

	// Revoke before issuing so a concurrent re-use of the same token fails.
	if err = a.tokenRepo.RevokeRefreshToken(ctx, claims.UserID, jti); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	user, err := a.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	out, err := a.issueTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	a.writeAuditLog(ctx, user.ID, user.Email, "token_refresh")

	return out, nil
}

// Logout revokes the specific refresh token identified by the provided token
// string. The short-lived access token is not revoked; its natural expiry is
// the defence against misuse after logout.
func (a *authUsecase) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	claims, err := a.jwtManager.ExtractClaims(refreshToken)
	if err != nil {
		// Accept parse failures on logout — token may be expired or malformed.
		return nil
	}

	// Constant-time comparison guards against presenting another user's token.
	if subtle.ConstantTimeCompare(claims.UserID[:], userID[:]) != 1 {
		return domain.ErrUnauthorized
	}

	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return domain.ErrTokenInvalid
	}

	if err = a.tokenRepo.RevokeRefreshToken(ctx, userID, jti); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	a.writeAuditLog(ctx, userID, "", "logout")

	return nil
}

// issueTokenPair generates a new access+refresh pair, stores the refresh token
// hash in Redis, and returns a LoginOutput.
func (a *authUsecase) issueTokenPair(ctx context.Context, user *domain.User) (*LoginOutput, error) {
	roleName := ""
	if user.Role != nil {
		roleName = string(user.Role.Name)
	}

	accessToken, err := a.jwtManager.GenerateAccessToken(user.ID, user.Email, roleName)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := a.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Parse generated refresh token to extract JTI for Redis keying.
	refreshClaims, err := a.jwtManager.ExtractClaims(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("extract refresh claims: %w", err)
	}

	jti, err := uuid.Parse(refreshClaims.ID)
	if err != nil {
		return nil, fmt.Errorf("parse refresh jti: %w", err)
	}

	if err = a.tokenRepo.StoreRefreshToken(
		ctx,
		user.ID,
		jti,
		hashToken(refreshToken),
		a.jwtManager.RefreshTokenExpiry(),
	); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// hashToken returns a lowercase hex SHA-256 digest of the raw token string.
// The hash is stored in Redis instead of the raw token to limit damage if the
// store is compromised.
func hashToken(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

// runDummyBcrypt performs a bcrypt comparison to normalise the response time
// when the user record is not found, preventing timing-based enumeration.
func runDummyBcrypt() error {
	// Pre-computed bcrypt hash of "dummy" at cost 10.
	const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	return bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte("dummy"))
}

// writeAuditLog writes an audit entry in a best-effort manner; failures are
// silently discarded so that a logging outage never blocks the auth flow.
func (a *authUsecase) writeAuditLog(ctx context.Context, userID uuid.UUID, email, action string) {
	entry := &domain.AuditLog{
		ID:         uuid.New(),
		ActorID:    userID,
		ActorEmail: email,
		Action:     action,
		EntityType: "user",
		EntityID:   userID,
		CreatedAt:  time.Now(),
	}
	_ = a.auditRepo.Create(ctx, entry)
}

// GetCurrentUser fetches the authenticated user by ID.
func (a *authUsecase) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := a.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

// ForgotPassword generates a one-time password-reset token and stores it in
// Redis with a 1-hour expiry.
//
// Security invariants:
//   - The response is identical whether or not the email is registered. This
//     prevents user enumeration via the forgot-password endpoint.
//   - The raw token (32 random bytes hex-encoded) is returned to the caller so
//     that it can be delivered out-of-band (e.g. email). In production the token
//     would be sent via email; for this implementation it is returned directly.
func (a *authUsecase) ForgotPassword(ctx context.Context, email string) (string, error) {
	// Always generate a token regardless of whether the user exists so that the
	// response time is constant and the caller cannot enumerate accounts.
	token, err := generateResetToken()
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}

	// Only store the token when the user actually exists. The caller still
	// receives a token string either way (no enumeration).
	if _, lookupErr := a.userRepo.FindByEmail(ctx, email); lookupErr == nil {
		if storeErr := a.passwordReset.StorePasswordResetToken(ctx, email, token, time.Hour); storeErr != nil {
			return "", fmt.Errorf("store reset token: %w", storeErr)
		}
		a.writeAuditLog(ctx, uuid.Nil, email, "forgot_password")
	}

	return token, nil
}

// ResetPassword validates the one-time token for email, then hashes and stores
// the new password. The token is consumed on success so it cannot be replayed.
func (a *authUsecase) ResetPassword(ctx context.Context, email, token, newPassword string) error {
	stored, err := a.passwordReset.GetPasswordResetToken(ctx, email)
	if err != nil {
		return fmt.Errorf("get reset token: %w", err)
	}

	// Treat an absent/expired token as invalid credentials.
	if stored == "" || subtle.ConstantTimeCompare([]byte(stored), []byte(token)) != 1 {
		return domain.ErrInvalidCredentials
	}

	user, err := a.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return domain.ErrInvalidCredentials
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err = a.userRepo.UpdatePassword(ctx, user.ID, string(hashed)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Consume the token so it cannot be reused.
	if err = a.passwordReset.DeletePasswordResetToken(ctx, email); err != nil {
		// Best-effort: log but do not fail the reset.
		_ = err
	}

	a.writeAuditLog(ctx, user.ID, email, "reset_password")

	return nil
}

// generateResetToken returns a cryptographically random 32-byte hex string.
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// mapJWTError translates jwt package sentinel errors to domain errors.
func mapJWTError(err error) error {
	if errors.Is(err, jwtpkg.ErrTokenExpired) {
		return domain.ErrTokenExpired
	}
	return domain.ErrTokenInvalid
}
