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

// KYBHandler handles HTTP requests for KYB verification endpoints.
type KYBHandler struct {
	kybUC usecase.KYBUsecase
}

// NewKYBHandler constructs a KYBHandler.
func NewKYBHandler(kybUC usecase.KYBUsecase) *KYBHandler {
	return &KYBHandler{kybUC: kybUC}
}

// List returns a paginated list of KYB verifications.
//
// @Summary      List KYB verifications
// @Description  Returns a paginated, optionally filtered list of KYB verification records.
// @Tags         kyb
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number (default 1)"
// @Param        page_size query     int     false  "Page size (default 10, max 100)"
// @Param        search    query     string  false  "Full-text search term"
// @Param        status    query     string  false  "Filter by status (pending|approved|rejected)"
// @Param        sort_by   query     string  false  "Sort field"
// @Param        sort_dir  query     string  false  "Sort direction: asc or desc"
// @Success      200  {object}  response.Response{data=[]dto.KYBResponse}
// @Failure      401  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyb [get]
func (h *KYBHandler) List(c echo.Context) error {
	params := pagination.ParseParams(c)

	records, total, err := h.kybUC.List(c.Request().Context(), params)
	if err != nil {
		return response.InternalError(c, "failed to list KYB verifications")
	}

	meta := &response.Meta{
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
		Pages:    pagination.CalcPages(total, params.PageSize),
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    dto.ToKYBResponses(records),
		Meta:    meta,
	})
}

// GetByID returns a single KYB verification by ID.
//
// @Summary      Get KYB verification
// @Description  Returns a single KYB verification record identified by its UUID.
// @Tags         kyb
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "KYB verification UUID"
// @Success      200  {object}  response.Response{data=dto.KYBResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyb/{id} [get]
func (h *KYBHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	kyb, err := h.kybUC.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYB verification not found")
		}
		return response.InternalError(c, "failed to retrieve KYB verification")
	}

	return response.OK(c, dto.ToKYBResponse(kyb))
}

// Submit opens a new KYB verification for a company.
//
// @Summary      Submit KYB
// @Description  Open a new KYB verification in Pending status for the given company.
// @Tags         kyb
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.SubmitKYBRequest  true  "KYB submission data"
// @Success      201  {object}  response.Response{data=dto.KYBResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyb/submit [post]
func (h *KYBHandler) Submit(c echo.Context) error {
	var req dto.SubmitKYBRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return response.BadRequest(c, "VALIDATION_ERROR", err.Error())
	}

	companyID, err := uuid.Parse(req.CompanyID)
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "company_id must be a valid UUID")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyb, err := h.kybUC.Submit(c.Request().Context(), usecase.SubmitKYBInput{
		CompanyID:        companyID,
		SubmittedBy:      claims.UserID,
		Notes:            req.Notes,
		BusinessDocURL:   req.BusinessDocURL,
		TaxDocURL:        req.TaxDocURL,
		DirectorIDDocURL: req.DirectorIDDocURL,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "company not found")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, "INVALID_INPUT", err.Error())
		}
		return response.InternalError(c, "failed to submit KYB verification")
	}

	return c.JSON(http.StatusCreated, response.Response{
		Success: true,
		Data:    dto.ToKYBResponse(kyb),
	})
}

// Approve transitions a pending KYB verification to Approved.
//
// @Summary      Approve KYB
// @Description  Approve a pending KYB verification. Requires admin role.
// @Tags         kyb
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                true  "KYB verification UUID"
// @Param        body  body      dto.ReviewKYBRequest  false  "Review notes"
// @Success      200  {object}  response.Response{data=dto.KYBResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyb/{id}/approve [put]
func (h *KYBHandler) Approve(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.ReviewKYBRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyb, err := h.kybUC.Approve(c.Request().Context(), usecase.ReviewKYBInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYB verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYB verification is not in pending status")
		}
		return response.InternalError(c, "failed to approve KYB verification")
	}

	return response.OK(c, dto.ToKYBResponse(kyb))
}

// Reject transitions a pending or in-review KYB verification to Rejected.
//
// @Summary      Reject KYB
// @Description  Reject a pending or in-review KYB verification. Requires admin role.
// @Tags         kyb
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                true  "KYB verification UUID"
// @Param        body  body      dto.ReviewKYBRequest  false  "Review notes"
// @Success      200  {object}  response.Response{data=dto.KYBResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyb/{id}/reject [put]
func (h *KYBHandler) Reject(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.ReviewKYBRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return response.BadRequest(c, "INVALID_REQUEST", "invalid request body")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	kyb, err := h.kybUC.Reject(c.Request().Context(), usecase.ReviewKYBInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYB verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYB verification is not in a reviewable status")
		}
		return response.InternalError(c, "failed to reject KYB verification")
	}

	return response.OK(c, dto.ToKYBResponse(kyb))
}

// SetInReview transitions a pending KYB verification to in_review.
//
// @Summary      Set KYB In Review
// @Description  Move a pending KYB verification into in_review state. Requires admin role.
// @Tags         kyb
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "KYB verification UUID"
// @Param        body  body      dto.SetInReviewRequest  false "Optional reviewer notes"
// @Success      200  {object}  response.Response{data=dto.KYBResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyb/{id}/review [put]
func (h *KYBHandler) SetInReview(c echo.Context) error {
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

	kyb, err := h.kybUC.SetInReview(c.Request().Context(), usecase.SetInReviewKYBInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYB verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYB verification is not in pending status")
		}
		return response.InternalError(c, "failed to set KYB verification in review")
	}

	return response.OK(c, dto.ToKYBResponse(kyb))
}

// RequestAdditionalDocs requests additional documents for a KYB verification.
//
// @Summary      Request Additional Documents (KYB)
// @Description  Mark a pending or in-review KYB verification as requiring additional documents. Requires admin role.
// @Tags         kyb
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                  true  "KYB verification UUID"
// @Param        body  body      dto.RequestDocsRequest  true  "Notes describing what documents are needed"
// @Success      200  {object}  response.Response{data=dto.KYBResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /kyb/{id}/request-docs [post]
func (h *KYBHandler) RequestAdditionalDocs(c echo.Context) error {
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

	kyb, err := h.kybUC.RequestAdditionalDocs(c.Request().Context(), usecase.RequestAdditionalDocsKYBInput{
		ID:         id,
		ReviewerID: claims.UserID,
		Notes:      req.Notes,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYB verification not found")
		}
		if errors.Is(err, domain.ErrInvalidStatus) {
			return response.Conflict(c, "KYB verification is not in a reviewable status")
		}
		return response.InternalError(c, "failed to request additional documents for KYB verification")
	}

	return response.OK(c, dto.ToKYBResponse(kyb))
}

// Delete permanently removes a KYB verification record.
//
// @Summary      Delete KYB verification
// @Tags         kyb
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "KYB verification UUID"
// @Success      204
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /kyb/{id} [delete]
func (h *KYBHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	if err := h.kybUC.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "KYB verification not found")
		}
		return response.InternalError(c, "failed to delete KYB verification")
	}

	return c.NoContent(http.StatusNoContent)
}
