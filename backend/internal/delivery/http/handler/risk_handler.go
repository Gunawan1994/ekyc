package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/dto"
	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/middleware"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
)

type RiskHandler struct {
	riskUC usecase.RiskUsecase
}

func NewRiskHandler(riskUC usecase.RiskUsecase) *RiskHandler {
	return &RiskHandler{riskUC: riskUC}
}

// GetKYCRisk returns the latest risk assessment for a KYC verification.
func (h *RiskHandler) GetKYCRisk(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	ra, err := h.riskUC.GetLatest(c.Request().Context(), "kyc", id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "risk assessment not found")
		}
		return response.InternalError(c, "failed to retrieve risk assessment")
	}

	return response.OK(c, dto.ToRiskAssessmentResponse(ra))
}

// GetKYBRisk returns the latest risk assessment for a KYB verification.
func (h *RiskHandler) GetKYBRisk(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	ra, err := h.riskUC.GetLatest(c.Request().Context(), "kyb", id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "risk assessment not found")
		}
		return response.InternalError(c, "failed to retrieve risk assessment")
	}

	return response.OK(c, dto.ToRiskAssessmentResponse(ra))
}

// OverrideKYCRisk allows a risk_analyst to manually set the risk level for a KYC.
func (h *RiskHandler) OverrideKYCRisk(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.ManualRiskOverrideRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	ra, err := h.riskUC.ManualOverride(c.Request().Context(), usecase.ManualRiskOverrideInput{
		EntityType: "kyc",
		EntityID:   id,
		RiskLevel:  domain.RiskLevel(req.RiskLevel),
		Notes:      req.Notes,
		AssessedBy: claims.UserID,
	})
	if err != nil {
		return response.InternalError(c, "failed to override risk assessment")
	}

	return c.JSON(http.StatusCreated, response.Response{Success: true, Data: dto.ToRiskAssessmentResponse(ra)})
}

// OverrideKYBRisk allows a risk_analyst to manually set the risk level for a KYB.
func (h *RiskHandler) OverrideKYBRisk(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.ManualRiskOverrideRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_BODY", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	ra, err := h.riskUC.ManualOverride(c.Request().Context(), usecase.ManualRiskOverrideInput{
		EntityType: "kyb",
		EntityID:   id,
		RiskLevel:  domain.RiskLevel(req.RiskLevel),
		Notes:      req.Notes,
		AssessedBy: claims.UserID,
	})
	if err != nil {
		return response.InternalError(c, "failed to override risk assessment")
	}

	return c.JSON(http.StatusCreated, response.Response{Success: true, Data: dto.ToRiskAssessmentResponse(ra)})
}

// ListKYCRiskHistory returns all risk assessments for a KYC verification.
func (h *RiskHandler) ListKYCRiskHistory(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	history, err := h.riskUC.ListHistory(c.Request().Context(), "kyc", id)
	if err != nil {
		return response.InternalError(c, "failed to retrieve risk history")
	}

	out := make([]dto.RiskAssessmentResponse, len(history))
	for i := range history {
		out[i] = dto.ToRiskAssessmentResponse(&history[i])
	}
	return response.OK(c, out)
}

// ListKYBRiskHistory returns all risk assessments for a KYB verification.
func (h *RiskHandler) ListKYBRiskHistory(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	history, err := h.riskUC.ListHistory(c.Request().Context(), "kyb", id)
	if err != nil {
		return response.InternalError(c, "failed to retrieve risk history")
	}

	out := make([]dto.RiskAssessmentResponse, len(history))
	for i := range history {
		out[i] = dto.ToRiskAssessmentResponse(&history[i])
	}
	return response.OK(c, out)
}
