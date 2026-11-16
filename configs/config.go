package configs

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort             string
	Environment            string
	ServiceName            string
	JWTSecret              string
	AWSRegion              string
	SQSEndpoint            string
	ExecutionCommandsQueue string
	SagaRepliesQueue       string
	DynamoDBEndpoint       string
	DynamoDBTablePrefix    string
	DynamoDBCreateTables   bool
	OtelExporterEndpoint   string
	OtelExporterHeaders    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		ServerPort:             getEnv("SERVER_PORT", "8080"),
		Environment:            getEnv("APP_ENV", "local"),
		ServiceName:            getEnv("OTEL_SERVICE_NAME", "oficina-execution-service"),
		JWTSecret:              getEnv("JWT_SECRET", ""),
		AWSRegion:              getEnv("AWS_REGION", "us-east-1"),
		SQSEndpoint:            getEnv("SQS_ENDPOINT", ""),
		ExecutionCommandsQueue: getEnv("EXECUTION_COMMANDS_QUEUE", "oficina-execution-commands"),
		SagaRepliesQueue:       getEnv("SAGA_REPLIES_QUEUE", "oficina-saga-replies"),
		DynamoDBEndpoint:       getEnv("DYNAMODB_ENDPOINT", ""),
		DynamoDBTablePrefix:    getEnv("DYNAMODB_TABLE_PREFIX", "oficina-execution"),
		DynamoDBCreateTables:   getBool("DYNAMODB_CREATE_TABLES", false),
		OtelExporterEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OtelExporterHeaders:    getEnv("OTEL_EXPORTER_OTLP_HEADERS", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
