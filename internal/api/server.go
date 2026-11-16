package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/configs"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api/middleware"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type ReadinessCheck func(ctx context.Context) error

type Server struct {
	router     *gin.Engine
	config     *configs.Config
	readiness  ReadinessCheck
	httpServer *http.Server
}

func NewServer(cfg *configs.Config, readiness ReadinessCheck) *Server {
	router := gin.New()
	router.Use(
		gin.CustomRecovery(recoverWithLog),
		otelgin.Middleware(cfg.ServiceName),
		middleware.RequestID(),
		middleware.RequestLogger(),
	)

	server := &Server{
		router:    router,
		config:    cfg,
		readiness: readiness,
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

func (s *Server) registerRoutes() {
	s.router.GET("/health", s.health)
	s.router.GET("/ready", s.ready)
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) ready(c *gin.Context) {
	if err := s.readiness(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "database unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
