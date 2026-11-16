// @title           Tech Challenge — Oficina Mecânica Execution Service
// @version         1.0
// @description     Diagnosis and execution queue, catalog and stock of the mechanical workshop.
// @host            localhost:8082
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/configs"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/outbox"
	domainmessaging "github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/database"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/observability"
	infrarepo "github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/repository"
)

func main() {
	cfg, err := configs.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.ServiceName, cfg.Environment)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := observability.SetupTracing(ctx, observability.TracingConfig{
		ServiceName: cfg.ServiceName,
		Environment: cfg.Environment,
		Endpoint:    cfg.OtelExporterEndpoint,
		Headers:     cfg.OtelExporterHeaders,
	})
	if err != nil {
		logger.Error("failed to configure tracing", slog.String("error", err.Error()))
		os.Exit(1)
	}

	tables := database.NewTables(cfg.DynamoDBTablePrefix)
	dynamoClient, err := database.NewDynamoDBClient(ctx, cfg.AWSRegion, cfg.DynamoDBEndpoint)
	if err != nil {
		logger.Error("failed to configure database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.DynamoDBCreateTables {
		if err := database.EnsureTables(ctx, dynamoClient, tables); err != nil {
			logger.Error("failed to create tables", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	queueClient, err := messaging.NewSQSClient(ctx, cfg.AWSRegion, cfg.SQSEndpoint)
	if err != nil {
		logger.Error("failed to configure messaging", slog.String("error", err.Error()))
		os.Exit(1)
	}

	publisher := messaging.NewSQSPublisher(queueClient, map[domainmessaging.Destination]string{
		domainmessaging.DestinationSagaReplies: cfg.SagaRepliesQueue,
	})
	router := messaging.NewRouter()
	consumer := messaging.NewSQSConsumer(queueClient, cfg.ExecutionCommandsQueue, router.Handle,
		infrarepo.NewDynamoProcessedMessageRepository(dynamoClient, tables.ProcessedMessages))
	relay := outbox.NewRelay(infrarepo.NewDynamoOutboxRepository(dynamoClient, tables.Outbox), publisher)

	server := api.NewServer(cfg, func(ctx context.Context) error {
		return database.Ping(ctx, dynamoClient, tables)
	})

	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		consumer.Run(ctx)
	}()
	go func() {
		defer workers.Done()
		relay.Run(ctx)
	}()

	go func() {
		logger.Info("server.started",
			slog.String("event", "server.started"),
			slog.String("port", cfg.ServerPort),
			slog.Bool("tracing_enabled", cfg.OtelExporterEndpoint != ""),
		)
		if err := server.Run(); err != nil {
			logger.Error("server stopped unexpectedly", slog.String("error", err.Error()))
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shut down server gracefully", slog.String("error", err.Error()))
	}
	workers.Wait()
	if err := shutdownTracing(shutdownCtx); err != nil {
		logger.Error("failed to flush traces", slog.String("error", err.Error()))
	}
	logger.Info("server.stopped", slog.String("event", "server.stopped"))
}
