package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	domain "github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/messaging"
)

type outboxItem struct {
	ID           string            `dynamodbav:"id"`
	Destination  string            `dynamodbav:"destination"`
	MessageType  string            `dynamodbav:"message_type"`
	SagaID       string            `dynamodbav:"saga_id"`
	OrderID      string            `dynamodbav:"order_id"`
	Payload      string            `dynamodbav:"payload"`
	TraceContext map[string]string `dynamodbav:"trace_context"`
	OccurredAt   time.Time         `dynamodbav:"occurred_at"`
	CreatedAt    time.Time         `dynamodbav:"created_at"`
	UpdatedAt    time.Time         `dynamodbav:"updated_at"`
}

type DynamoOutboxRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoOutboxRepository(client *dynamodb.Client, tableName string) *DynamoOutboxRepository {
	return &DynamoOutboxRepository{client: client, tableName: tableName}
}

func (r *DynamoOutboxRepository) RelayPending(ctx context.Context, limit int, relay domain.RelayFunc) (int, error) {
	output, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:      aws.String(r.tableName),
		Limit:          aws.Int32(int32(limit)),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return 0, err
	}

	var pending []outboxItem
	if err := attributevalue.UnmarshalListOfMaps(output.Items, &pending); err != nil {
		return 0, err
	}

	relayed := 0
	for _, item := range pending {
		if err := relay(messaging.ExtractTraceContext(ctx, item.TraceContext), outgoingFromItem(item)); err != nil {
			return relayed, err
		}
		if err := r.remove(ctx, item.ID); err != nil {
			return relayed, err
		}
		relayed++
	}
	return relayed, nil
}

func (r *DynamoOutboxRepository) remove(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key:       map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}},
	})
	return err
}

func outboxWrites(ctx context.Context, tableName string, outgoing []domain.OutgoingMessage) ([]types.TransactWriteItem, error) {
	traceContext := messaging.InjectTraceContext(ctx)
	now := time.Now().UTC()

	writes := make([]types.TransactWriteItem, 0, len(outgoing))
	for _, item := range outgoing {
		marshalled, err := attributevalue.MarshalMap(outboxItem{
			ID:           item.Message.ID,
			Destination:  string(item.Destination),
			MessageType:  item.Message.Type,
			SagaID:       item.Message.SagaID,
			OrderID:      item.Message.OrderID,
			Payload:      string(item.Message.Payload),
			TraceContext: traceContext,
			OccurredAt:   item.Message.OccurredAt,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if err != nil {
			return nil, err
		}
		writes = append(writes, types.TransactWriteItem{
			Put: &types.Put{TableName: aws.String(tableName), Item: marshalled},
		})
	}
	return writes, nil
}

func outgoingFromItem(item outboxItem) domain.OutgoingMessage {
	return domain.OutgoingMessage{
		Destination: domain.Destination(item.Destination),
		Message: domain.Message{
			ID:         item.ID,
			Type:       item.MessageType,
			SagaID:     item.SagaID,
			OrderID:    item.OrderID,
			OccurredAt: item.OccurredAt,
			Payload:    json.RawMessage(item.Payload),
		},
	}
}
