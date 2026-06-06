package http

import (
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/handler"
	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/middleware"
	jwtpkg "github.com/monarchintiteknologi/ekyc-platform/internal/pkg/jwt"
	echoswagger "github.com/swaggo/echo-swagger"

	_ "github.com/monarchintiteknologi/ekyc-platform/docs"
)

// Handlers aggregates all HTTP handler dependencies injected at startup.
type Handlers struct {
	Auth      *handler.AuthHandler
	Customer  *handler.CustomerHandler
	Company   *handler.CompanyHandler
	KYC       *handler.KYCHandler
	KYB       *handler.KYBHandler
	Risk      *handler.RiskHandler
	Dashboard *handler.DashboardHandler
	Audit     *handler.AuditHandler
	Upload    *handler.UploadHandler
}

// SetupRoutes registers all application routes on the given Echo instance.
// Global middleware (RequestID, CORS, Recover) is applied first, followed by
// the Swagger UI endpoint, and then the versioned API tree.
func SetupRoutes(e *echo.Echo, h *Handlers, jwtMgr *jwtpkg.Manager) {
	// Global middleware: applied to every request regardless of path.
	e.Use(echomiddleware.RequestID())
	e.Use(echomiddleware.CORS())
	e.Use(echomiddleware.Recover())

	// Swagger UI — publicly accessible at /swagger/*
	e.GET("/swagger/*", echoswagger.WrapHandler)

	// Versioned API root
	api := e.Group("/api/v1")

	registerAuthRoutes(api, h.Auth, jwtMgr)
	registerProtectedRoutes(api, h, jwtMgr)
}

// registerAuthRoutes mounts authentication endpoints under /auth.
// /logout and /me require a valid JWT; the other endpoints are public.
func registerAuthRoutes(api *echo.Group, h *handler.AuthHandler, jwtMgr *jwtpkg.Manager) {
	auth := api.Group("/auth")
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.GET("/me", h.Me, middleware.JWTMiddleware(jwtMgr))
	auth.POST("/logout", h.Logout, middleware.JWTMiddleware(jwtMgr))
	auth.POST("/forgot-password", h.ForgotPassword)
	auth.POST("/reset-password", h.ResetPassword)
}

// registerProtectedRoutes mounts all JWT-protected resource routes.
func registerProtectedRoutes(api *echo.Group, h *Handlers, jwtMgr *jwtpkg.Manager) {
	protected := api.Group("", middleware.JWTMiddleware(jwtMgr))

	// Dashboard — admin, super_admin, risk_analyst, compliance_officer
	protected.GET("/dashboard/stats", h.Dashboard.GetStats,
		middleware.RequireRole("admin", "super_admin", "risk_analyst", "compliance_officer"))

	registerCustomerRoutes(protected, h.Customer)
	registerCompanyRoutes(protected, h.Company)
	registerKYCRoutes(protected, h.KYC)
	registerKYBRoutes(protected, h.KYB)
	registerRiskRoutes(protected, h.Risk)
	registerAuditRoutes(protected, h.Audit)
	registerUploadRoutes(api, protected, h.Upload)
}

// registerCustomerRoutes mounts CRUD endpoints under /customers.
func registerCustomerRoutes(g *echo.Group, h *handler.CustomerHandler) {
	customers := g.Group("/customers")
	customers.GET("", h.List)
	customers.POST("", h.Create)
	customers.GET("/:id", h.GetByID)
	customers.PUT("/:id", h.Update)
	customers.DELETE("/:id", h.Delete)
}

// registerCompanyRoutes mounts CRUD endpoints under /companies.
func registerCompanyRoutes(g *echo.Group, h *handler.CompanyHandler) {
	companies := g.Group("/companies")
	companies.GET("", h.List)
	companies.POST("", h.Create)
	companies.GET("/:id", h.GetByID)
	companies.PUT("/:id", h.Update)
	companies.DELETE("/:id", h.Delete)
}

// registerKYCRoutes mounts KYC submission and review endpoints under /kyc.
// Approve, reject, set-in-review, and request-docs actions require
// admin, super_admin, compliance_officer, or reviewer.
func registerKYCRoutes(g *echo.Group, h *handler.KYCHandler) {
	kyc := g.Group("/kyc")
	kyc.GET("", h.List)
	kyc.POST("/submit", h.Submit)
	kyc.GET("/:id", h.GetByID)
	kyc.PUT("/:id/approve", h.Approve,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyc.PUT("/:id/reject", h.Reject,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyc.PUT("/:id/review", h.SetInReview,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyc.POST("/:id/request-docs", h.RequestAdditionalDocs,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyc.DELETE("/:id", h.Delete,
		middleware.RequireRole("admin", "super_admin"))
}

// registerKYBRoutes mounts KYB submission and review endpoints under /kyb.
// Approve, reject, set-in-review, and request-docs actions require
// admin, super_admin, compliance_officer, or reviewer.
func registerKYBRoutes(g *echo.Group, h *handler.KYBHandler) {
	kyb := g.Group("/kyb")
	kyb.GET("", h.List)
	kyb.POST("/submit", h.Submit)
	kyb.GET("/:id", h.GetByID)
	kyb.PUT("/:id/approve", h.Approve,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyb.PUT("/:id/reject", h.Reject,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyb.PUT("/:id/review", h.SetInReview,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyb.POST("/:id/request-docs", h.RequestAdditionalDocs,
		middleware.RequireRole("admin", "super_admin", "compliance_officer", "reviewer"))
	kyb.DELETE("/:id", h.Delete,
		middleware.RequireRole("admin", "super_admin"))
}

// registerRiskRoutes mounts risk assessment endpoints under /kyc/:id/risk and /kyb/:id/risk.
// GET is accessible to all authenticated users; POST (manual override) is risk_analyst only.
func registerRiskRoutes(g *echo.Group, h *handler.RiskHandler) {
	g.GET("/kyc/:id/risk", h.GetKYCRisk)
	g.GET("/kyb/:id/risk", h.GetKYBRisk)
	g.POST("/kyc/:id/risk", h.OverrideKYCRisk,
		middleware.RequireRole("admin", "super_admin", "risk_analyst"))
	g.POST("/kyb/:id/risk", h.OverrideKYBRisk,
		middleware.RequireRole("admin", "super_admin", "risk_analyst"))
	g.GET("/kyc/:id/risk/history", h.ListKYCRiskHistory)
	g.GET("/kyb/:id/risk/history", h.ListKYBRiskHistory)
}

// registerAuditRoutes mounts read-only audit log endpoints under /audit-logs.
// Access is restricted to admin and super_admin roles.
func registerAuditRoutes(g *echo.Group, h *handler.AuditHandler) {
	g.GET("/audit-logs", h.List,
		middleware.RequireRole("admin", "super_admin"))
}

// registerUploadRoutes mounts file upload and static file serving.
// Upload requires JWT; serving uploaded files is public (URLs are unguessable UUIDs).
func registerUploadRoutes(api, protected *echo.Group, h *handler.UploadHandler) {
	protected.POST("/upload", h.Upload)
	api.GET("/uploads/:filename", h.ServeFile)
}
