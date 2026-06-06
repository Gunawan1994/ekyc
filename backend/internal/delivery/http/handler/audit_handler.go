package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/dto"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/pagination"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
)

// AuditHandler handles HTTP requests for audit log endpoints.
type AuditHandler struct {
	auditUC usecase.AuditUsecase
}

// NewAuditHandler constructs an AuditHandler.
func NewAuditHandler(auditUC usecase.AuditUsecase) *AuditHandler {
	return &AuditHandler{auditUC: auditUC}
}

// List returns a paginated list of audit log entries.
//
// @Summary      List audit logs
// @Description  Returns a paginated, optionally filtered list of audit log entries. Admin only.
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number (default 1)"
// @Param        page_size query     int     false  "Page size (default 10, max 100)"
// @Param        actor_id  query     string  false  "Filter by actor UUID"
// @Param        action    query     string  false  "Filter by action (create, update, delete, submit, review)"
// @Success      200  {object}  response.Response{data=[]dto.AuditLogResponse}
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /audit-logs [get]
func (h *AuditHandler) List(c echo.Context) error {
	base := pagination.ParseParams(c)

	params := domain.AuditListParams{
		ListParams: base,
		ActorID:    c.QueryParam("actor_id"),
		Action:     c.QueryParam("action"),
	}

	logs, total, err := h.auditUC.ListAuditLogs(c.Request().Context(), params)
	if err != nil {
		return response.InternalError(c, "failed to list audit logs")
	}

	meta := &response.Meta{
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
		Pages:    pagination.CalcPages(total, params.PageSize),
	}

	return c.JSON(http.StatusOK, response.Response{
		Success: true,
		Data:    dto.ToAuditLogResponses(logs),
		Meta:    meta,
	})
}
