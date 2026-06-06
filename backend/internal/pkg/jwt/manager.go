package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Sentinel errors returned by ValidateToken and ExtractClaims.
var (
	ErrTokenExpired       = errors.New("jwt: token expired")
	ErrTokenInvalid       = errors.New("jwt: token invalid or malformed")
	ErrTokenSigningMethod = errors.New("jwt: unexpected signing method")
	ErrSecretEmpty        = errors.New("jwt: secret must not be empty")
)

// Claims carries the standard JWT registered claims plus application-specific
// fields that are embedded in every token.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwtv5.RegisteredClaims
}

// Manager issues and validates HMAC-SHA256 signed JWTs.
type Manager struct {
	secret             []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewManager constructs a Manager.
// secret must not be empty; accessExpiry and refreshExpiry must be positive.
func NewManager(secret string, accessExpiry, refreshExpiry time.Duration) *Manager {
	if secret == "" {
		panic(ErrSecretEmpty)
	}
	return &Manager{
		secret:             []byte(secret),
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
	}
}

// GenerateAccessToken creates a signed JWT access token with the configured
// expiry (typically 15 minutes). The token embeds userID, email, and role.
func (m *Manager) GenerateAccessToken(userID uuid.UUID, email, role string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(m.accessTokenExpiry)),
		},
	}

	return m.sign(claims)
}

// GenerateRefreshToken creates a signed JWT refresh token with the configured
// expiry (typically 7 days). Only userID is embedded to minimise payload size.
func (m *Manager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(m.refreshTokenExpiry)),
		},
	}

	return m.sign(claims)
}

// ValidateToken parses tokenStr, verifies the signature, and checks all
// registered claims including expiry. It returns the embedded Claims on
// success, or a typed sentinel error on failure.
func (m *Manager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(
		tokenStr,
		&Claims{},
		m.keyFunc,
		jwtv5.WithExpirationRequired(),
	)
	if err != nil {
		return nil, m.mapError(err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ExtractClaims parses tokenStr and returns the embedded Claims without
// enforcing expiry. This is intended for refresh-token flows where a
// caller needs to inspect a legitimately expired access token before
// issuing a new one. The signature is still verified.
func (m *Manager) ExtractClaims(tokenStr string) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(
		tokenStr,
		&Claims{},
		m.keyFunc,
		// Allow expired tokens so the caller can decide what to do.
		jwtv5.WithoutClaimsValidation(),
	)
	if err != nil {
		// WithoutClaimsValidation still surfaces signature errors.
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// AccessTokenExpiry returns the configured access token lifetime.
func (m *Manager) AccessTokenExpiry() time.Duration {
	return m.accessTokenExpiry
}

// RefreshTokenExpiry returns the configured refresh token lifetime.
func (m *Manager) RefreshTokenExpiry() time.Duration {
	return m.refreshTokenExpiry
}

// sign serialises claims into a signed JWT string.
func (m *Manager) sign(claims *Claims) (string, error) {
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign token: %w", err)
	}
	return signed, nil
}

// keyFunc is the jwtv5.Keyfunc that returns the HMAC secret after asserting
// that the token header advertises an HMAC signing algorithm.
func (m *Manager) keyFunc(t *jwtv5.Token) (interface{}, error) {
	if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("%w: %v", ErrTokenSigningMethod, t.Header["alg"])
	}
	return m.secret, nil
}

// mapError translates jwtv5 errors into package-level sentinel errors.
func (m *Manager) mapError(err error) error {
	if errors.Is(err, jwtv5.ErrTokenExpired) {
		return ErrTokenExpired
	}
	if errors.Is(err, ErrTokenSigningMethod) {
		return ErrTokenSigningMethod
	}
	return ErrTokenInvalid
}
