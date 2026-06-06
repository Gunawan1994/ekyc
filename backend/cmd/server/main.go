// Package main is the entrypoint for the eKYC platform API server.
//
// Startup order:
//  1. Load configuration from env / config file.
//  2. Initialise zerolog logger.
//  3. Open PostgreSQL connection pool (pgxpool) and wrap it with sqlx.
//  4. Open Redis client.
//  5. Build JWT manager.
//  6. Wire all repositories (postgres + redis).
//  7. Wire all usecases.
//  8. Wire all HTTP handlers.
//  9. Run database seeders.
//  10. Build Echo instance with middleware, routes, health check, and Swagger UI.
//  11. Start HTTP server with graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	echoSwagger "github.com/swaggo/echo-swagger"

	"github.com/monarchintiteknologi/ekyc-platform/internal/config"
	httpdelivery "github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http"
	"github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/handler"
	httpmiddleware "github.com/monarchintiteknologi/ekyc-platform/internal/delivery/http/middleware"
	jwtpkg "github.com/monarchintiteknologi/ekyc-platform/internal/pkg/jwt"
	"github.com/monarchintiteknologi/ekyc-platform/internal/pkg/logger"
	validatorpkg "github.com/monarchintiteknologi/ekyc-platform/internal/pkg/validator"
	postgresrepo "github.com/monarchintiteknologi/ekyc-platform/internal/repository/postgres"
	redisrepo "github.com/monarchintiteknologi/ekyc-platform/internal/repository/redis"
	"github.com/monarchintiteknologi/ekyc-platform/internal/usecase"
	"github.com/monarchintiteknologi/ekyc-platform/seeders"

	// Swagger-generated docs — side-effect import to register route metadata.
	_ "github.com/monarchintiteknologi/ekyc-platform/docs"
)

// @title           eKYC Platform API
// @version         1.0
// @description     REST API for the eKYC & eKYB verification platform.
// @termsOfService  http://swagger.io/terms/

// @contact.name   PT Sun Energy
// @contact.email  support@sunenergy.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// -----------------------------------------------------------------------
	// 1. Configuration
	// -----------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// 2. Logger
	// -----------------------------------------------------------------------
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info().
		Str("app", cfg.App.Name).
		Str("env", cfg.App.Env).
		Msg("starting server")

	// -----------------------------------------------------------------------
	// 3. PostgreSQL connection pool (pgxpool) + sqlx wrapper
	// -----------------------------------------------------------------------
	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer poolCancel()

	pool, err := pgxpool.New(poolCtx, cfg.DB.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("create pgxpool")
	}
	defer pool.Close()

	if err = pool.Ping(poolCtx); err != nil {
		log.Fatal().Err(err).Msg("ping postgres")
	}
	log.Info().Msg("connected to postgres")

	// Wrap the pgxpool via stdlib so that sqlx-based helpers work alongside pgx.
	db := sqlx.NewDb(stdlib.OpenDBFromPool(pool), "pgx")

	// -----------------------------------------------------------------------
	// 4. Redis client
	// -----------------------------------------------------------------------
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	if err = redisClient.Ping(redisCtx).Err(); err != nil {
		log.Fatal().Err(err).Msg("ping redis")
	}
	defer redisClient.Close()
	log.Info().Msg("connected to redis")

	// -----------------------------------------------------------------------
	// 5. JWT manager
	// -----------------------------------------------------------------------
	jwtManager := jwtpkg.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)

	// -----------------------------------------------------------------------
	// 6. Repositories
	// -----------------------------------------------------------------------
	userRepo := postgresrepo.NewUserRepository(db)
	companyRepo := postgresrepo.NewCompanyRepository(db)
	customerRepo := postgresrepo.NewCustomerRepository(db)
	kycRepo := postgresrepo.NewKYCRepository(db)
	kybRepo := postgresrepo.NewKYBRepository(db)
	auditRepo := postgresrepo.NewAuditRepository(db)
	riskRepo := postgresrepo.NewRiskRepository(db)

	tokenRepo := redisrepo.NewTokenRepository(redisClient)
	cacheRepo := redisrepo.NewCacheRepository(redisClient)

	// -----------------------------------------------------------------------
	// 7. Usecases
	// -----------------------------------------------------------------------
	authUC := usecase.NewAuthUsecase(userRepo, tokenRepo, cacheRepo, jwtManager, auditRepo)
	customerUC := usecase.NewCustomerUsecase(customerRepo, auditRepo, cacheRepo)
	companyUC := usecase.NewCompanyUsecase(companyRepo, auditRepo, cacheRepo)
	riskUC := usecase.NewRiskUsecase(riskRepo)
	kycUC := usecase.NewKYCUsecase(kycRepo, customerRepo, auditRepo, riskUC)
	kybUC := usecase.NewKYBUsecase(kybRepo, companyRepo, auditRepo, riskUC)
	dashboardUC := usecase.NewDashboardUsecase(customerRepo, companyRepo, kycRepo, kybRepo, cacheRepo)
	auditUC := usecase.NewAuditUsecase(auditRepo)

	// -----------------------------------------------------------------------
	// 8. HTTP handlers
	// -----------------------------------------------------------------------
	handlers := &httpdelivery.Handlers{
		Auth:      handler.NewAuthHandler(authUC),
		Customer:  handler.NewCustomerHandler(customerUC),
		Company:   handler.NewCompanyHandler(companyUC),
		KYC:       handler.NewKYCHandler(kycUC),
		KYB:       handler.NewKYBHandler(kybUC),
		Risk:      handler.NewRiskHandler(riskUC),
		Dashboard: handler.NewDashboardHandler(dashboardUC),
		Audit:     handler.NewAuditHandler(auditUC),
		Upload:    handler.NewUploadHandler(),
	}

	// -----------------------------------------------------------------------
	// 9. Seeders
	// -----------------------------------------------------------------------
	seeder := seeders.NewSeeder(db)
	if err = seeder.Run(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("run seeders")
	}

	// -----------------------------------------------------------------------
	// 10. Echo instance
	// -----------------------------------------------------------------------
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Custom validator (go-playground/validator under the hood).
	e.Validator = validatorpkg.New()

	// Custom error handler that maps domain errors to HTTP status codes.
	e.HTTPErrorHandler = httpmiddleware.CustomErrorHandler

	// Global middleware.
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			event := log.Info()
			if v.Error != nil {
				event = log.Error().Err(v.Error)
			}
			event.
				Str("id", v.RequestID).
				Str("method", v.Method).
				Str("uri", v.URI).
				Int("status", v.Status).
				Dur("latency", v.Latency).
				Msg("request")
			return nil
		},
	}))

	// Application routes (also registers RequestID and CORS middleware).
	httpdelivery.SetupRoutes(e, handlers, jwtManager)

	// Health check.
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Swagger UI.
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// -----------------------------------------------------------------------
	// 11. Start server with graceful shutdown
	// -----------------------------------------------------------------------
	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	log.Info().Str("addr", addr).Msg("server listening")

	serverErr := make(chan error, 1)
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatal().Err(err).Msg("server error")
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
		os.Exit(1)
	}

	log.Info().Msg("server stopped")
}
