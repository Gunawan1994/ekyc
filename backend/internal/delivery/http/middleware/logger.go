package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// RequestLogger returns middleware that emits a structured zerolog entry for
// every handled request.  The log line includes:
//   - request_id  – value of the X-Request-ID response header (set by
//     Echo's built-in middleware.RequestID())
//   - method      – HTTP verb
//   - uri         – full request URI including query string
//   - status      – HTTP response status code
//   - latency_ms  – wall-clock duration in milliseconds
//   - remote_ip   – client IP as reported by Echo (honours X-Forwarded-For
//     when the proxy trust list is configured)
//
// Requests that complete with status >= 500 are logged at ERROR level;
// all others are logged at INFO level.
func RequestLogger(logger zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			// Echo writes the status via c.Response().Status after the handler
			// returns, so we read it here for accuracy.
			status := c.Response().Status
			latency := time.Since(start)
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)

			event := logger.Info()
			if status >= 500 {
				event = logger.Error()
			}

			event.
				Str("request_id", requestID).
				Str("method", c.Request().Method).
				Str("uri", c.Request().RequestURI).
				Int("status", status).
				Dur("latency_ms", latency).
				Str("remote_ip", c.RealIP()).
				Msg("request")

			return err
		}
	}
}
