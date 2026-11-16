package repository

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const processedMessageRetention = 30 * 24 * time.Hour

type processedMessageItem struct {
	ID        string    `dynamodbav:"id"`
	CreatedAt time.Time `dynamodbav:"created_at"`
	UpdatedAt time.Time `dynamodbav:"updated_at"`
	ExpiresAt int64     `dynamodbav:"expires_at"`
}

type DynamoProcessedMessageRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoProcessedMessageRepository(client *dynamodb.Client, tableName string) *DynamoProcessedMessageRepository {
	return &DynamoProcessedMessageRepository{client: client, tableName: tableName}
}

func (r *DynamoProcessedMessageRepository) Exists(ctx context.Context, messageID string) (bool, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(r.tableName),
		Key:            keyOf(messageID),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return false, err
	}
	return len(output.Item) > 0, nil
}

func (r *DynamoProcessedMessageRepository) Save(ctx context.Context, messageID string) error {
	now := time.Now().UTC()
	item, err := attributevalue.MarshalMap(processedMessageItem{
		ID:        messageID,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(processedMessageRetention).Unix(),
	})
	if err != nil {
		return err
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.tableName), Item: item})
	return err
}
