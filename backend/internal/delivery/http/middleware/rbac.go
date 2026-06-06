package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
)

// RequireRole returns middleware that allows access only when the authenticated
// user's role matches one of the provided role strings.  It must be placed
// after JWTMiddleware so that the claims are already stored on the context.
//
// Returns 403 Forbidden when the role is absent or not in the allowed list.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	// Build a lookup set for O(1) membership checks.
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return response.Forbidden(c, "access denied")
			}

			if _, ok := allowed[claims.Role]; !ok {
				return response.Forbidden(c, "insufficient permissions")
			}

			return next(c)
		}
	}
}
