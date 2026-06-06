package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/dto"
	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/middleware"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
)

// AuthHandler handles HTTP requests for authentication endpoints.
type AuthHandler struct {
	authUC usecase.AuthUsecase
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authUC usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

// Login authenticates a user and returns an access/refresh token pair.
//
// @Summary      Login
// @Description  Authenticate with email and password. Returns JWT access and refresh tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "Login credentials"
// @Success      200   {object}  response.Response{data=dto.AuthResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      500   {object}  response.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	out, err := h.authUC.Login(c.Request().Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return response.Unauthorized(c, "invalid email or password")
		}
		return response.InternalError(c, "login failed")
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    dto.ToAuthResponse(out),
	})
}

// Refresh rotates a refresh token and returns a new access/refresh token pair.
//
// @Summary      Refresh token
// @Description  Exchange a valid refresh token for a new access/refresh token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RefreshTokenRequest  true  "Refresh token"
// @Success      200   {object}  response.Response{data=dto.AuthResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      500   {object}  response.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req dto.RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	out, err := h.authUC.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTokenExpired):
			return response.Unauthorized(c, "refresh token expired")
		case errors.Is(err, domain.ErrTokenInvalid):
			return response.Unauthorized(c, "invalid refresh token")
		case errors.Is(err, domain.ErrTokenRevoked):
			return response.Unauthorized(c, "refresh token has been revoked")
		case errors.Is(err, domain.ErrUserInactive):
			return response.Unauthorized(c, "user account is inactive")
		default:
			return response.InternalError(c, "token refresh failed")
		}
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    dto.ToAuthResponse(out),
	})
}

// Me returns the profile of the currently authenticated user.
//
// @Summary      Get current user
// @Description  Returns the profile of the authenticated user based on the Bearer token.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=dto.UserResponse}
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	user, err := h.authUC.GetCurrentUser(c.Request().Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "user not found")
		}
		return response.InternalError(c, "failed to fetch user")
	}

	roleName := ""
	if user.Role != nil {
		roleName = string(user.Role.Name)
	}
	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data: dto.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      roleName,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Logout revokes the caller's refresh token.
//
// @Summary      Logout
// @Description  Revoke the provided refresh token. The Authorization header must carry a valid Bearer access token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.LogoutRequest  true  "Refresh token to revoke"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      500   {object}  response.Response
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	var req dto.LogoutRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	if err := h.authUC.Logout(c.Request().Context(), claims.UserID, req.RefreshToken); err != nil {
		if errors.Is(err, domain.ErrTokenInvalid) || errors.Is(err, domain.ErrUnauthorized) {
			return response.Unauthorized(c, "invalid token")
		}
		return response.InternalError(c, "logout failed")
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    map[string]string{"message": "logged out successfully"},
	})
}

// ForgotPassword generates a one-time password-reset token for the given email.
// The response is identical whether or not the email is registered to prevent
// user enumeration. In production the token would be sent via email; here it is
// returned directly in the response.
//
// @Summary      Forgot password
// @Description  Request a password-reset token for the given email address.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ForgotPasswordRequest  true  "Email address"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      500   {object}  response.Response
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req dto.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	resetToken, err := h.authUC.ForgotPassword(c.Request().Context(), req.Email)
	if err != nil {
		return response.InternalError(c, "failed to process request")
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    map[string]string{"reset_token": resetToken},
	})
}

// ResetPassword validates a one-time token and sets a new password.
//
// @Summary      Reset password
// @Description  Reset the account password using a valid one-time token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ResetPasswordRequest  true  "Reset credentials"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      500   {object}  response.Response
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req dto.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	if err := h.authUC.ResetPassword(c.Request().Context(), req.Email, req.Token, req.NewPassword); err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return response.Unauthorized(c, "invalid or expired reset token")
		}
		return response.InternalError(c, "failed to reset password")
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    map[string]string{"message": "password reset successfully"},
	})
}
