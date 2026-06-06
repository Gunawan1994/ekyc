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

// CompanyHandler handles HTTP requests for company endpoints.
type CompanyHandler struct {
	companyUC usecase.CompanyUsecase
}

// NewCompanyHandler constructs a CompanyHandler.
func NewCompanyHandler(companyUC usecase.CompanyUsecase) *CompanyHandler {
	return &CompanyHandler{companyUC: companyUC}
}

// List returns a paginated list of companies.
//
// @Summary      List companies
// @Description  Returns a paginated, optionally filtered list of companies.
// @Tags         companies
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number (default 1)"
// @Param        page_size query     int     false  "Page size (default 10, max 100)"
// @Param        search    query     string  false  "Full-text search term"
// @Param        status    query     string  false  "Filter by status"
// @Param        sort_by   query     string  false  "Sort field"
// @Param        sort_dir  query     string  false  "Sort direction: asc or desc"
// @Success      200  {object}  response.Response{data=[]dto.CompanyResponse}
// @Failure      401  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /companies [get]
func (h *CompanyHandler) List(c echo.Context) error {
	params := pagination.ParseParams(c)

	companies, total, err := h.companyUC.List(c.Request().Context(), params)
	if err != nil {
		return response.InternalError(c, "failed to list companies")
	}

	meta := &response.Meta{
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
		Pages:    pagination.CalcPages(total, params.PageSize),
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    dto.ToCompanyResponses(companies),
		Meta:    meta,
	})
}

// Create registers a new company.
//
// @Summary      Create company
// @Description  Register a new company. The caller becomes the linked user.
// @Tags         companies
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateCompanyRequest  true  "Company data"
// @Success      201  {object}  response.Response{data=dto.CompanyResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /companies [post]
func (h *CompanyHandler) Create(c echo.Context) error {
	var req dto.CreateCompanyRequest
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

	company, err := h.companyUC.Create(c.Request().Context(), usecase.CreateCompanyInput{
		UserID:             claims.UserID,
		Name:               req.Name,
		LegalName:          req.Name,
		RegistrationNumber: req.RegistrationNumber,
		Address:            req.Address,
		Phone:              req.Phone,
		Email:              req.Email,
	}, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return response.Conflict(c, "company with this registration number already exists")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, "INVALID_INPUT", err.Error())
		}
		return response.InternalError(c, "failed to create company")
	}

	return c.JSON(http.StatusCreated, response.Response{
		Success: true,
		Data:    dto.ToCompanyResponse(company),
	})
}

// GetByID returns a single company by ID.
//
// @Summary      Get company
// @Description  Returns a single company identified by its UUID.
// @Tags         companies
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Company UUID"
// @Success      200  {object}  response.Response{data=dto.CompanyResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /companies/{id} [get]
func (h *CompanyHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	company, err := h.companyUC.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "company not found")
		}
		return response.InternalError(c, "failed to retrieve company")
	}

	return response.OK(c, dto.ToCompanyResponse(company))
}

// Update replaces the mutable fields of a company.
//
// @Summary      Update company
// @Description  Update a company's name, address, phone, and email.
// @Tags         companies
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                    true  "Company UUID"
// @Param        body  body      dto.UpdateCompanyRequest  true  "Updated company data"
// @Success      200  {object}  response.Response{data=dto.CompanyResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /companies/{id} [put]
func (h *CompanyHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.UpdateCompanyRequest
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

	company, err := h.companyUC.Update(c.Request().Context(), usecase.UpdateCompanyInput{
		ID:      id,
		Name:    req.Name,
		Address: req.Address,
		Phone:   req.Phone,
		Email:   req.Email,
	}, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "company not found")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, "INVALID_INPUT", err.Error())
		}
		return response.InternalError(c, "failed to update company")
	}

	return response.OK(c, dto.ToCompanyResponse(company))
}

// Delete soft-deletes a company.
//
// @Summary      Delete company
// @Description  Soft-delete a company by UUID.
// @Tags         companies
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Company UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /companies/{id} [delete]
func (h *CompanyHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	if err := h.companyUC.Delete(c.Request().Context(), id, claims.UserID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "company not found")
		}
		return response.InternalError(c, "failed to delete company")
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    map[string]string{"message": "company deleted successfully"},
	})
}
