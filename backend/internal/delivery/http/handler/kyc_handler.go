package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/dto"
	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/middleware"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/pagination"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
)

// KYCHandler handles HTTP requests for KYC verification endpoints.
type KYCHandler struct {
	kycUC usecase.KYCUsecase
}

// NewKYCHandler constructs a KYCHandler.
func NewKYCHandler(kycUC usecase.KYCUsecase) *KYCHandler {
	return &KYCHandler{kycUC: kycUC}
}

// List returns a paginated list of KYC verifications.
//
// @Summary      List KYC verifications
// @Description  Returns a paginated, optionally filtered list of KYC verification records.
// @Tags         kyc
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number (default 1)"
// @Param        page_size query     int     false  "Page size (default 10, max 100)"
// @Param        search    query     string  false  "Full-text search term"
// @Param        status    query     string  false  "Filter by status (pending|approved|rejected)"
// @Param        sort_by   query     string  false  "Sort field"
// @Param        sort_dir  query     string  false  "Sort direction: asc or desc"
// @Success      200  {object}  response.Response{data=[]dto.KYCResponse}
// @Failure      401  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyc [get]
func (h *KYCHandler) List(c echo.Context) error {
	params := pagination.ParseParams(c)

	records, total, err := h.kycUC.List(c.Request().Context(), params)
	if err != nil {
		return response.InternalError(c, "failed to list KYC verifications")
	}

	meta := &response.Meta{
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
		Pages:    pagination.CalcPages(total, params.PageSize),
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    dto.ToKYCResponses(records),
		Meta:    meta,
	})
}

// GetByID returns a single KYC verification by ID.
//
// @Summary      Get KYC verification
// @Description  Returns a single KYC verification record identified by its UUID.
// @Tags         kyc
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "KYC verification UUID"
// @Success      200  {object}  response.Response{data=dto.KYCResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyc/{id} [get]
func (h *KYCHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	kyc, err := h.kycUC.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYC verification not found")
		}
		return response.InternalError(c, "failed to retrieve KYC verification")
	}

	return response.OK(c, dto.ToKYCResponse(kyc))
}

// Submit opens a new KYC verification for a customer.
//
// @Summary      Submit KYC
// @Description  Open a new KYC verification in Pending status for the given customer.
// @Tags         kyc
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.SubmitKYCRequest  true  "KYC submission data"
// @Success      201  {object}  response.Response{data=dto.KYCResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyc/submit [post]
func (h *KYCHandler) Submit(c echo.Context) error {
	var req dto.SubmitKYCRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "customer_id must be a valid UUID")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyc, err := h.kycUC.Submit(c.Request().Context(), usecase.SubmitKYCInput{
		CustomerID:    customerID,
		SubmittedBy:   claims.UserID,
		Notes:         req.Notes,
		IDDocumentURL: req.IDDocumentURL,
		SelfieURL:     req.SelfieURL,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "customer not found")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, "INVALID_INPUT", err.Error())
		}
		return response.InternalError(c, "failed to submit KYC verification")
	}

	return c.JSON(http.StatusCreated, response.Response{
		Success: true,
		Data:    dto.ToKYCResponse(kyc),
	})
}

// Approve transitions a pending KYC verification to Approved.
//
// @Summary      Approve KYC
// @Description  Approve a pending KYC verification. Requires admin role.
// @Tags         kyc
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                true  "KYC verification UUID"
// @Param        body  body      dto.ReviewKYCRequest  false  "Review notes"
// @Success      200  {object}  response.Response{data=dto.KYCResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyc/{id}/approve [put]
func (h *KYCHandler) Approve(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.ReviewKYCRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyc, err := h.kycUC.Approve(c.Request().Context(), usecase.ReviewKYCInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYC verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYC verification is not in pending status")
		}
		return response.InternalError(c, "failed to approve KYC verification")
	}

	return response.OK(c, dto.ToKYCResponse(kyc))
}

// Reject transitions a pending or in-review KYC verification to Rejected.
//
// @Summary      Reject KYC
// @Description  Reject a pending or in-review KYC verification. Requires admin role.
// @Tags         kyc
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                true  "KYC verification UUID"
// @Param        body  body      dto.ReviewKYCRequest  false  "Review notes"
// @Success      200  {object}  response.Response{data=dto.KYCResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyc/{id}/reject [put]
func (h *KYCHandler) Reject(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.ReviewKYCRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyc, err := h.kycUC.Reject(c.Request().Context(), usecase.ReviewKYCInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYC verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYC verification is not in a reviewable status")
		}
		return response.InternalError(c, "failed to reject KYC verification")
	}

	return response.OK(c, dto.ToKYCResponse(kyc))
}

// SetInReview transitions a pending KYC verification to in_review.
//
// @Summary      Set KYC In Review
// @Description  Move a pending KYC verification into in_review state. Requires admin role.
// @Tags         kyc
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "KYC verification UUID"
// @Param        body  body      dto.SetInReviewRequest  false "Optional reviewer notes"
// @Success      200  {object}  response.Response{data=dto.KYCResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyc/{id}/review [put]
func (h *KYCHandler) SetInReview(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.SetInReviewRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyc, err := h.kycUC.SetInReview(c.Request().Context(), usecase.SetInReviewKYCInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYC verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYC verification is not in pending status")
		}
		return response.InternalError(c, "failed to set KYC verification in review")
	}

	return response.OK(c, dto.ToKYCResponse(kyc))
}

// RequestAdditionalDocs requests additional documents for a KYC verification.
//
// @Summary      Request Additional Documents (KYC)
// @Description  Mark a pending or in-review KYC verification as requiring additional documents. Requires admin role.
// @Tags         kyc
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "KYC verification UUID"
// @Param        body  body      dto.RequestDocsRequest  true  "Notes describing what documents are needed"
// @Success      200  {object}  response.Response{data=dto.KYCResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyc/{id}/request-docs [post]
func (h *KYCHandler) RequestAdditionalDocs(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.RequestDocsRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyc, err := h.kycUC.RequestAdditionalDocs(c.Request().Context(), usecase.RequestAdditionalDocsKYCInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYC verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYC verification is not in a reviewable status")
		}
		return response.InternalError(c, "failed to request additional documents for KYC verification")
	}

	return response.OK(c, dto.ToKYCResponse(kyc))
}

// Delete permanently removes a KYC verification record.
//
// @Summary      Delete KYC verification
// @Tags         kyc
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "KYC verification UUID"
// @Success      204
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /kyc/{id} [delete]
func (h *KYCHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	if err := h.kycUC.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYC verification not found")
		}
		return response.InternalError(c, "failed to delete KYC verification")
	}

	return c.NoContent(http.StatusNoContent)
}
