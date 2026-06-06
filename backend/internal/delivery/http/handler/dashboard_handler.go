package handler

import (
	"github.com/labstack/echo/v4"

	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/dto"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/response"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
)

// DashboardHandler handles HTTP requests for dashboard endpoints.
type DashboardHandler struct {
	dashboardUC usecase.DashboardUsecase
}

// NewDashboardHandler constructs a DashboardHandler.
func NewDashboardHandler(dashboardUC usecase.DashboardUsecase) *DashboardHandler {
	return &DashboardHandler{dashboardUC: dashboardUC}
}

// GetStats returns aggregated platform statistics.
//
// @Summary      Get dashboard stats
// @Description  Returns platform-wide aggregated counters for customers, companies, KYC, and KYB verifications. Results are cached for 1 minute. Requires admin role.
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response{data=dto.DashboardStatsResponse}
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /dashboard/stats [get]
func (h *DashboardHandler) GetStats(c echo.Context) error {
	stats, err := h.dashboardUC.GetStats(c.Request().Context())
	if err != nil {
		return response.InternalError(c, "failed to retrieve dashboard statistics")
	}

	return response.OK(c, dto.ToDashboardStatsResponse(stats))
}
