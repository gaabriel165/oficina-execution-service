package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/configs"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api/consumer"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api/handler"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api/middleware"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/part"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/service"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type ReadinessCheck func(ctx context.Context) error

type Dependencies struct {
	PartRepo      repository.PartRepository
	ServiceRepo   repository.ServiceRepository
	WorkOrderRepo repository.WorkOrderRepository
	Readiness     ReadinessCheck
}

type Server struct {
	router            *gin.Engine
	config            *configs.Config
	dependencies      Dependencies
	workOrderConsumer *consumer.WorkOrderConsumer
	httpServer        *http.Server
}

func NewServer(cfg *configs.Config, dependencies Dependencies) *Server {
	router := gin.New()
	router.Use(
		gin.CustomRecovery(recoverWithLog),
		otelgin.Middleware(cfg.ServiceName),
		middleware.RequestID(),
		middleware.RequestLogger(),
	)

	server := &Server{
		router:       router,
		config:       cfg,
		dependencies: dependencies,
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%s", cfg.ServerPort),
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	server.registerRoutes()

	return server
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) Run() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func recoverWithLog(c *gin.Context, recovered any) {
	slog.ErrorContext(c.Request.Context(), "http.panic",
		slog.String("event", "http.panic"),
		slog.Any("panic", recovered),
	)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func (s *Server) RegisterMessageHandlers(registrar consumer.Registrar) {
	s.workOrderConsumer.RegisterHandlers(registrar)
}

func (s *Server) registerRoutes() {
	partRepo := s.dependencies.PartRepo
	serviceRepo := s.dependencies.ServiceRepo
	workOrderRepo := s.dependencies.WorkOrderRepo

	partHandler := handler.NewPartHandler(
		part.NewCreatePartUseCase(partRepo),
		part.NewUpdatePartUseCase(partRepo),
		part.NewDeletePartUseCase(partRepo),
		part.NewGetPartUseCase(partRepo),
		part.NewListPartsUseCase(partRepo),
		part.NewUpdateStockUseCase(partRepo),
	)

	serviceHandler := handler.NewServiceHandler(
		service.NewCreateServiceUseCase(serviceRepo),
		service.NewUpdateServiceUseCase(serviceRepo),
		service.NewDeleteServiceUseCase(serviceRepo),
		service.NewGetServiceUseCase(serviceRepo),
		service.NewListServicesUseCase(serviceRepo),
	)

	workOrderHandler := handler.NewWorkOrderHandler(
		workorder.NewStartDiagnosisUseCase(workOrderRepo),
		workorder.NewAddServiceToWorkOrderUseCase(workOrderRepo, serviceRepo),
		workorder.NewAddPartToWorkOrderUseCase(workOrderRepo, partRepo),
		workorder.NewCompleteDiagnosisUseCase(workOrderRepo),
		workorder.NewFailDiagnosisUseCase(workOrderRepo),
		workorder.NewFinishExecutionUseCase(workOrderRepo),
		workorder.NewGetWorkOrderUseCase(workOrderRepo),
		workorder.NewListWorkOrdersUseCase(workOrderRepo),
		workorder.NewGetExecutionMetricsUseCase(workOrderRepo),
	)

	s.workOrderConsumer = consumer.NewWorkOrderConsumer(
		workorder.NewOpenWorkOrderUseCase(workOrderRepo),
		workorder.NewStartExecutionUseCase(workOrderRepo),
		workorder.NewCancelWorkOrderUseCase(workOrderRepo),
	)

	s.router.GET("/health", s.health)
	s.router.GET("/ready", s.ready)

	v1 := s.router.Group("/api/v1")
	operators := v1.Group("", middleware.Auth(s.config.JWTSecret), middleware.RequireRole(middleware.RoleOperator))

	partHandler.RegisterRoutes(operators)
	serviceHandler.RegisterRoutes(operators)
	workOrderHandler.RegisterRoutes(operators)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) ready(c *gin.Context) {
	if err := s.dependencies.Readiness(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "database unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
