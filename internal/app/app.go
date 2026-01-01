package app

import (
	"context"
	"net/http"

	"github.com/creafly/logger"
	sharedmw "github.com/creafly/middleware"
	intmiddleware "github.com/creafly/storage/internal/middleware"
	"github.com/creafly/tracing"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/xlab/closer"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type App struct {
	serviceProvider *serviceProvider
	httpServer      *http.Server
}

func NewApp() *App {
	app := &App{}
	app.initApp()
	return app
}

func (a *App) initApp() {
	a.initEnv()
	a.initLogger()
	a.initCloser()
	a.initServiceProvider()
}

func (a *App) initEnv() {
	if err := godotenv.Load(); err != nil {
		logger.Info().Msg("No .env file found, using environment variables")
	}
}

func (a *App) initLogger() {
	logger.InitFromEnv("storage")
}

func (a *App) initCloser() {
	closer.Bind(func() {
		logger.Info().Msg("Application cleanup completed")
	})
}

func (a *App) initServiceProvider() {
	a.serviceProvider = newServiceProvider()
}

func (a *App) StartMigrator(migrateUp, migrateDown bool) {
	cfg := a.serviceProvider.GetConfig()
	migrator := a.serviceProvider.GetMigrator()

	if migrateUp {
		if err := migrator.Up(); err != nil {
			logger.Fatal().Err(err).Msg("Failed to run migrations")
		}
		logger.Info().Msg("Migrations applied successfully")
		closer.Close()
		return
	}

	if migrateDown {
		if err := migrator.Down(); err != nil {
			logger.Fatal().Err(err).Msg("Failed to rollback migrations")
		}
		logger.Info().Msg("Migration rolled back successfully")
		closer.Close()
		return
	}

	if cfg.Database.AutoMigrate {
		if err := migrator.Up(); err != nil {
			logger.Fatal().Err(err).Msg("Failed to run auto migrations")
		}
	}
}

func (a *App) StartApp(ctx context.Context) {
	a.startTracing(ctx)
	a.ensureMinioBucket(ctx)
	a.startHTTPServer()
}

func (a *App) startTracing(ctx context.Context) {
	cfg := a.serviceProvider.GetConfig()

	tracingShutdown, err := tracing.Init(tracing.Config{
		ServiceName:    cfg.Tracing.ServiceName,
		ServiceVersion: cfg.Tracing.ServiceVersion,
		Environment:    cfg.Tracing.Environment,
		OTLPEndpoint:   cfg.Tracing.OTLPEndpoint,
		Enabled:        cfg.Tracing.Enabled,
	})
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize tracing")
	}

	closer.Bind(func() {
		if err := tracingShutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("Error shutting down tracer provider")
		}
		logger.Info().Msg("Tracing shutdown completed")
	})
}

func (a *App) ensureMinioBucket(ctx context.Context) {
	a.serviceProvider.EnsureMinioBucket(ctx)
}

func (a *App) startHTTPServer() {
	server := a.getHttpServer()

	go func() {
		logger.Info().Str("addr", server.Addr).Msg("Starting Storage Service")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()
}

func (a *App) getHttpServer() *http.Server {
	if a.httpServer == nil {
		cfg := a.serviceProvider.GetConfig()
		gin.SetMode(cfg.Server.GinMode)

		router := gin.New()
		a.initHttpMiddleware(router)
		a.initHttpRouting(router)

		addr := cfg.Server.Host + ":" + cfg.Server.Port
		a.httpServer = &http.Server{
			Addr:    addr,
			Handler: router,
		}

		closer.Bind(func() {
			if err := a.httpServer.Shutdown(context.Background()); err != nil {
				logger.Error().Err(err).Msg("Server forced to shutdown")
			}
			logger.Info().Msg("HTTP server shutdown completed")
		})
	}
	return a.httpServer
}

func (a *App) initHttpMiddleware(r *gin.Engine) {
	cfg := a.serviceProvider.GetConfig()

	r.Use(gin.Recovery())
	r.Use(sharedmw.RequestID())
	r.Use(sharedmw.SecurityHeaders())
	r.Use(sharedmw.HSTS(cfg.Server.GinMode == "release"))
	r.Use(sharedmw.ContentTypeValidation())
	r.Use(sharedmw.RateLimit(sharedmw.RateLimitConfig{
		Enabled:           cfg.RateLimit.Enabled,
		RequestsPerSecond: cfg.RateLimit.RequestsPerSecond,
		BurstSize:         cfg.RateLimit.BurstSize,
	}))
	r.Use(otelgin.Middleware("storage"))
	r.Use(sharedmw.Logging())
	r.Use(intmiddleware.PrometheusMiddleware())
	r.Use(sharedmw.CORS(sharedmw.CORSConfig{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}))
	r.Use(sharedmw.Locale())
	r.Use(sharedmw.Compression())
}

func (a *App) initHttpRouting(r *gin.Engine) {
	healthHandler := a.serviceProvider.GetHealthHandler()
	fileHandler := a.serviceProvider.GetFileHandler()
	folderHandler := a.serviceProvider.GetFolderHandler()
	identityClient := a.serviceProvider.GetIdentityClient()

	r.GET("/health", healthHandler.Health)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api/v1")

	auth := api.Group("")
	auth.Use(intmiddleware.AuthMiddleware(identityClient))
	auth.Use(intmiddleware.TenantValidatorMiddleware(identityClient))
	{
		auth.POST("/files", fileHandler.Upload)
		auth.GET("/files", fileHandler.List)
		auth.GET("/files/usage", fileHandler.GetUsage)
		auth.GET("/files/:id", fileHandler.GetByID)
		auth.DELETE("/files/:id", fileHandler.Delete)
		auth.DELETE("/files/batch", fileHandler.BatchDelete)
		auth.GET("/files/:id/url", fileHandler.GetPresignedURL)

		auth.POST("/folders", folderHandler.Create)
		auth.GET("/folders", folderHandler.List)
		auth.GET("/folders/:id", folderHandler.GetByID)
		auth.PUT("/folders/:id", folderHandler.Update)
		auth.PUT("/folders/:id/move", folderHandler.Move)
		auth.DELETE("/folders/:id", folderHandler.Delete)
		auth.GET("/folders/:id/breadcrumb", folderHandler.GetBreadcrumb)
	}
}
