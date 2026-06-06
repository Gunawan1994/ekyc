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

// CustomerHandler handles HTTP requests for customer endpoints.
type CustomerHandler struct {
	customerUC usecase.CustomerUsecase
}

// NewCustomerHandler constructs a CustomerHandler.
func NewCustomerHandler(customerUC usecase.CustomerUsecase) *CustomerHandler {
	return &CustomerHandler{customerUC: customerUC}
}

// List returns a paginated list of customers.
//
// @Summary      List customers
// @Description  Returns a paginated, optionally filtered list of customers.
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number (default 1)"
// @Param        page_size query     int     false  "Page size (default 10, max 100)"
// @Param        search    query     string  false  "Full-text search term"
// @Param        status    query     string  false  "Filter by status"
// @Param        sort_by   query     string  false  "Sort field"
// @Param        sort_dir  query     string  false  "Sort direction: asc or desc"
// @Success      200  {object}  response.Response{data=[]dto.CustomerResponse}
// @Failure      401  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /customers [get]
func (h *CustomerHandler) List(c echo.Context) error {
	params := pagination.ParseParams(c)

	customers, total, err := h.customerUC.List(c.Request().Context(), params)
	if err != nil {
		return response.InternalError(c, "failed to list customers")
	}

	meta := &response.Meta{
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
		Pages:    pagination.CalcPages(total, params.PageSize),
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    dto.ToCustomerResponses(customers),
		Meta:    meta,
	})
}

// Create registers a new customer.
//
// @Summary      Create customer
// @Description  Register a new customer record linked to a company.
// @Tags         customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateCustomerRequest  true  "Customer data"
// @Success      201  {object}  response.Response{data=dto.CustomerResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      409  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /customers [post]
func (h *CustomerHandler) Create(c echo.Context) error {
	var req dto.CreateCustomerRequest
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

	customer, err := h.customerUC.Create(c.Request().Context(), usecase.CreateCustomerInput{
		CompanyID: companyID,
		FullName:  req.FullName,
		IDNumber:  req.IDNumber,
		IDType:    domain.IDType(req.IDType),
		Phone:     req.Phone,
		Email:     req.Email,
		Address:   req.Address,
	}, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return response.Conflict(c, "customer with this ID number already exists")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, "INVALID_INPUT", err.Error())
		}
		return response.InternalError(c, "failed to create customer")
	}

	return c.JSON(http.StatusCreated, response.Response{
		Success: true,
		Data:    dto.ToCustomerResponse(customer),
	})
}

// GetByID returns a single customer by ID.
//
// @Summary      Get customer
// @Description  Returns a single customer identified by its UUID.
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Customer UUID"
// @Success      200  {object}  response.Response{data=dto.CustomerResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /customers/{id} [get]
func (h *CustomerHandler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	customer, err := h.customerUC.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "customer not found")
		}
		return response.InternalError(c, "failed to retrieve customer")
	}

	return response.OK(c, dto.ToCustomerResponse(customer))
}

// Update replaces the mutable fields of a customer.
//
// @Summary      Update customer
// @Description  Update a customer's full name, phone, email, and address.
// @Tags         customers
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string                     true  "Customer UUID"
// @Param        body  body      dto.UpdateCustomerRequest  true  "Updated customer data"
// @Success      200  {object}  response.Response{data=dto.CustomerResponse}
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /customers/{id} [put]
func (h *CustomerHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	var req dto.UpdateCustomerRequest
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

	customer, err := h.customerUC.Update(c.Request().Context(), usecase.UpdateCustomerInput{
		ID:       id,
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
	}, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "customer not found")
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.BadRequest(c, "INVALID_INPUT", err.Error())
		}
		return response.InternalError(c, "failed to update customer")
	}

	return response.OK(c, dto.ToCustomerResponse(customer))
}

// Delete soft-deletes a customer.
//
// @Summary      Delete customer
// @Description  Soft-delete a customer by UUID.
// @Tags         customers
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Customer UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /customers/{id} [delete]
func (h *CustomerHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return response.BadRequest(c, "INVALID_UUID", "id must be a valid UUID")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return response.Unauthorized(c, "unauthenticated")
	}

	if err := h.customerUC.Delete(c.Request().Context(), id, claims.UserID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.NotFound(c, "customer not found")
		}
		return response.InternalError(c, "failed to delete customer")
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    map[string]string{"message": "customer deleted successfully"},
	})
}
