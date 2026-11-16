package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/database"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/messaging"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	elasticMQImage = "softwaremill/elasticmq-native:1.7.1"
	elasticMQPort  = "9324/tcp"
	dynamoDBImage  = "amazon/dynamodb-local:3.3.2"
	dynamoDBPort   = "8000/tcp"
	testRegion     = "us-east-1"
)

func startContainer(t *testing.T, image, port string, command ...string) string {
	t.Helper()
	ctx := context.Background()
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{port},
			Cmd:          command,
			WaitingFor:   wait.ForListeningPort(port),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := container.MappedPort(ctx, port)
	require.NoError(t, err)

	return fmt.Sprintf("http://%s:%s", host, mappedPort.Port())
}

func newTestDynamoDB(t *testing.T) (*dynamodb.Client, database.Tables) {
	t.Helper()
	ctx := context.Background()
	endpoint := startContainer(t, dynamoDBImage, dynamoDBPort, "-jar", "DynamoDBLocal.jar", "-sharedDb", "-inMemory")

	client, err := database.NewDynamoDBClient(ctx, testRegion, endpoint)
	require.NoError(t, err)

	tables := database.NewTables("test")
	require.NoError(t, database.EnsureTables(ctx, client, tables))
	return client, tables
}

func newTestQueueClient(t *testing.T, queueNames ...string) *sqs.Client {
	t.Helper()
	ctx := context.Background()
	endpoint := startContainer(t, elasticMQImage, elasticMQPort)

	client, err := messaging.NewSQSClient(ctx, testRegion, endpoint)
	require.NoError(t, err)

	for _, queueName := range queueNames {
		_, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(queueName)})
		require.NoError(t, err)
	}
	return client
}
