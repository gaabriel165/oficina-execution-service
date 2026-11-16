package database

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const partitionKey = "id"

type Tables struct {
	WorkOrders        string
	Catalog           string
	Outbox            string
	ProcessedMessages string
}

func NewTables(prefix string) Tables {
	return Tables{
		WorkOrders:        prefix + "-work-orders",
		Catalog:           prefix + "-catalog",
		Outbox:            prefix + "-outbox",
		ProcessedMessages: prefix + "-processed-messages",
	}
}

func (t Tables) Names() []string {
	return []string{t.WorkOrders, t.Catalog, t.Outbox, t.ProcessedMessages}
}

func NewDynamoDBClient(ctx context.Context, region, endpoint string) (*dynamodb.Client, error) {
	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(awsConfig, func(options *dynamodb.Options) {
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	}), nil
}

func EnsureTables(ctx context.Context, client *dynamodb.Client, tables Tables) error {
	for _, tableName := range tables.Names() {
		if err := ensureTable(ctx, client, tableName); err != nil {
			return err
		}
	}
	return nil
}

func Ping(ctx context.Context, client *dynamodb.Client, tables Tables) error {
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tables.WorkOrders)})
	return err
}

func ensureTable(ctx context.Context, client *dynamodb.Client, tableName string) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String(partitionKey), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(partitionKey), KeyType: types.KeyTypeHash},
		},
	})

	var alreadyExists *types.ResourceInUseException
	if errors.As(err, &alreadyExists) {
		return nil
	}
	return err
}
