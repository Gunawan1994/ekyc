package middleware

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
)

// errorEntry pairs a domain sentinel error with the HTTP response it maps to.
type errorEntry struct {
	target     error
	httpStatus int
	code       string
	message    string
}

// errorMap is consulted in order; the first matching entry wins.
// errors.Is is used so wrapped errors are matched correctly.
var errorMap = []errorEntry{
	{domain.ErrNotFound, http.StatusNotFound, "NOT_FOUND", "resource not found"},
	{domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN", "access denied"},
	{domain.ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized"},
	{domain.ErrAlreadyExists, http.StatusConflict, "CONFLICT", "resource already exists"},
	{domain.ErrInvalidInput, http.StatusBadRequest, "INVALID_INPUT", "invalid input"},
	{domain.ErrInvalidStatus, http.StatusUnprocessableEntity, "INVALID_STATUS", "invalid status transition"},
	{domain.ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials"},
	{domain.ErrUserInactive, http.StatusForbidden, "USER_INACTIVE", "user account is inactive"},
	{domain.ErrTokenExpired, http.StatusUnauthorized, "TOKEN_EXPIRED", "token has expired"},
	{domain.ErrTokenInvalid, http.StatusUnauthorized, "TOKEN_INVALID", "token is invalid"},
	{domain.ErrTokenRevoked, http.StatusUnauthorized, "TOKEN_REVOKED", "token has been revoked"},
}

// CustomErrorHandler is an echo.HTTPErrorHandler that maps domain errors and
// echo.HTTPError values to structured JSON responses using the project response
// envelope.  Register it via e.HTTPErrorHandler = middleware.CustomErrorHandler.
//
// Mapping rules:
//   - domain sentinel errors → specific 4xx/5xx codes (see errorMap above)
//   - *echo.HTTPError        → use its Code and Message fields
//   - everything else        → 500 Internal Server Error
//
// The handler is a no-op when the response has already been committed.
func CustomErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	// Walk the domain error map first so wrapped domain errors are caught.
	for _, entry := range errorMap {
		if errors.Is(err, entry.target) {
			_ = writeErrorResponse(c, entry.httpStatus, entry.code, entry.message)
			return
		}
	}

	// Fall back to echo.HTTPError for framework-generated errors (e.g. 404 from
	// the router, 405 Method Not Allowed, or manually returned echo.NewHTTPError).
	var he *echo.HTTPError
	if errors.As(err, &he) {
		msg, ok := he.Message.(string)
		if !ok {
			msg = http.StatusText(he.Code)
		}
		_ = writeErrorResponse(c, he.Code, httpStatusToCode(he.Code), msg)
		return
	}

	// Anything else is an unexpected server error.
	_ = writeErrorResponse(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an internal error occurred")
}

// writeErrorResponse serialises the error envelope.
func writeErrorResponse(c echo.Context, status int, code, message string) error {
	return c.JSON(status, response.Response{
		Success: false,
		Error: &response.ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// httpStatusToCode converts a numeric HTTP status to an UPPER_SNAKE_CASE code
// used in the response envelope when mapping echo.HTTPError values.
func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusMethodNotAllowed:
		return "METHOD_NOT_ALLOWED"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusUnprocessableEntity:
		return "UNPROCESSABLE_ENTITY"
	case http.StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	default:
		return "INTERNAL_SERVER_ERROR"
	}
}
