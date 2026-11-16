package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/internal/application/outbox"
	domain "github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/database"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/messaging"
	infrarepo "github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCommandsQueue = "oficina-execution-commands"
	testRepliesQueue  = "oficina-saga-replies"
)

type collectingHandler struct {
	received chan domain.Message
}

func (h *collectingHandler) handle(_ context.Context, message domain.Message) error {
	h.received <- message
	return nil
}

func TestEnsureTables_ShouldBeIdempotent(t *testing.T) {
	client, tables := newTestDynamoDB(t)
	ctx := context.Background()

	require.NoError(t, database.EnsureTables(ctx, client, tables))

	assert.NoError(t, database.Ping(ctx, client, tables))
}

func TestProcessedMessages_ShouldRememberSavedMessage(t *testing.T) {
	client, tables := newTestDynamoDB(t)
	processed := infrarepo.NewDynamoProcessedMessageRepository(client, tables.ProcessedMessages)
	messageID := uuid.NewString()
	ctx := context.Background()

	existsBefore, err := processed.Exists(ctx, messageID)
	require.NoError(t, err)
	require.NoError(t, processed.Save(ctx, messageID))
	require.NoError(t, processed.Save(ctx, messageID))
	existsAfter, err := processed.Exists(ctx, messageID)
	require.NoError(t, err)

	assert.False(t, existsBefore)
	assert.True(t, existsAfter)
}

func TestConsumer_ShouldReceiveCommandPublishedToQueue(t *testing.T) {
	queueClient := newTestQueueClient(t, testCommandsQueue)
	dynamoClient, tables := newTestDynamoDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := messaging.NewSQSPublisher(queueClient, map[domain.Destination]string{
		domain.DestinationExecutionCommands: testCommandsQueue,
	})
	command, err := domain.NewMessage(domain.TypeCancelWorkOrder, uuid.NewString(), uuid.NewString(), domain.ReasonPayload{Reason: "budget rejected"})
	require.NoError(t, err)
	require.NoError(t, publisher.Publish(ctx, domain.DestinationExecutionCommands, command))

	handler := &collectingHandler{received: make(chan domain.Message, 1)}
	consumer := messaging.NewSQSConsumer(queueClient, testCommandsQueue, handler.handle,
		infrarepo.NewDynamoProcessedMessageRepository(dynamoClient, tables.ProcessedMessages))
	go consumer.Run(ctx)

	select {
	case received := <-handler.received:
		var payload domain.ReasonPayload
		require.NoError(t, received.DecodePayload(&payload))
		assert.Equal(t, command.ID, received.ID)
		assert.Equal(t, "budget rejected", payload.Reason)
	case <-time.After(20 * time.Second):
		t.Fatal("command was not consumed")
	}
}

func TestOutboxRelay_ShouldDoNothingWhenOutboxIsEmpty(t *testing.T) {
	queueClient := newTestQueueClient(t, testRepliesQueue)
	dynamoClient, tables := newTestDynamoDB(t)
	publisher := messaging.NewSQSPublisher(queueClient, map[domain.Destination]string{
		domain.DestinationSagaReplies: testRepliesQueue,
	})

	relayed, err := outbox.NewRelay(infrarepo.NewDynamoOutboxRepository(dynamoClient, tables.Outbox), publisher).RelayOnce(context.Background())

	require.NoError(t, err)
	assert.Zero(t, relayed)
}
