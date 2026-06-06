package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Response is the standard API response envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo carries a machine-readable code and a human-readable message.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta carries pagination metadata for list responses.
type Meta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	Pages    int   `json:"pages"`
}

func jsonResp(c echo.Context, status int, r Response) error {
	return c.JSON(status, r)
}

// OK returns HTTP 200 with the given data payload.
func OK(c echo.Context, data interface{}) error {
	return jsonResp(c, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// OKWithMeta returns HTTP 200 with data and pagination metadata.
func OKWithMeta(c echo.Context, data interface{}, meta *Meta) error {
	return jsonResp(c, http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created returns HTTP 201 with the created resource.
func Created(c echo.Context, data interface{}) error {
	return jsonResp(c, http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// BadRequest returns HTTP 400 with an error code and message.
func BadRequest(c echo.Context, code, message string) error {
	return jsonResp(c, http.StatusBadRequest, Response{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message},
	})
}

// Unauthorized returns HTTP 401.
func Unauthorized(c echo.Context, message string) error {
	return jsonResp(c, http.StatusUnauthorized, Response{
		Success: false,
		Error:   &ErrorInfo{Code: "UNAUTHORIZED", Message: message},
	})
}

// Forbidden returns HTTP 403.
func Forbidden(c echo.Context, message string) error {
	return jsonResp(c, http.StatusForbidden, Response{
		Success: false,
		Error:   &ErrorInfo{Code: "FORBIDDEN", Message: message},
	})
}

// NotFound returns HTTP 404.
func NotFound(c echo.Context, message string) error {
	return jsonResp(c, http.StatusNotFound, Response{
		Success: false,
		Error:   &ErrorInfo{Code: "NOT_FOUND", Message: message},
	})
}

// InternalError returns HTTP 500.
func InternalError(c echo.Context, message string) error {
	return jsonResp(c, http.StatusInternalServerError, Response{
		Success: false,
		Error:   &ErrorInfo{Code: "INTERNAL_SERVER_ERROR", Message: message},
	})
}

// Conflict returns HTTP 409.
func Conflict(c echo.Context, message string) error {
	return jsonResp(c, http.StatusConflict, Response{
		Success: false,
		Error:   &ErrorInfo{Code: "CONFLICT", Message: message},
	})
}
