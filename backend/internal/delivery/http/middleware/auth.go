package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	jwtpkg "github.com/monarchintiteknologi/ekyc-platform/internal/pkg/jwt"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
)

const userClaimsKey = "user_claims"

// JWTMiddleware validates the Bearer token in the Authorization header.
// On success it stores the parsed *jwtpkg.Claims under the userClaimsKey
// context key and calls the next handler.  On failure it returns 401.
func JWTMiddleware(jwtManager *jwtpkg.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return response.Unauthorized(c, "missing or invalid authorization header")
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := jwtManager.ValidateToken(tokenStr)
			if err != nil {
				return response.Unauthorized(c, "invalid or expired token")
			}

			c.Set(userClaimsKey, claims)
			return next(c)
		}
	}
}

// GetClaims retrieves the *jwtpkg.Claims stored by JWTMiddleware.
// Returns nil when called outside of a JWT-protected route.
func GetClaims(c echo.Context) *jwtpkg.Claims {
	claims, _ := c.Get(userClaimsKey).(*jwtpkg.Claims)
	return claims
}
