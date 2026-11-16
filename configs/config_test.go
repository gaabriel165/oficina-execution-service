package configs_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ShouldUseDefaultsWhenEnvironmentIsEmpty(t *testing.T) {
	t.Setenv("SERVER_PORT", "")
	t.Setenv("EXECUTION_COMMANDS_QUEUE", "")
	t.Setenv("DYNAMODB_TABLE_PREFIX", "")
	t.Setenv("DYNAMODB_CREATE_TABLES", "")

	cfg, err := configs.Load()

	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "oficina-execution-commands", cfg.ExecutionCommandsQueue)
	assert.Equal(t, "oficina-execution", cfg.DynamoDBTablePrefix)
	assert.False(t, cfg.DynamoDBCreateTables)
}

func TestLoad_ShouldReadValuesFromEnvironment(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DYNAMODB_ENDPOINT", "http://localhost:8000")
	t.Setenv("DYNAMODB_CREATE_TABLES", "true")

	cfg, err := configs.Load()

	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "http://localhost:8000", cfg.DynamoDBEndpoint)
	assert.True(t, cfg.DynamoDBCreateTables)
}

func TestLoad_ShouldIgnoreInvalidBoolean(t *testing.T) {
	t.Setenv("DYNAMODB_CREATE_TABLES", "maybe")

	cfg, err := configs.Load()

	require.NoError(t, err)
	assert.False(t, cfg.DynamoDBCreateTables)
}
