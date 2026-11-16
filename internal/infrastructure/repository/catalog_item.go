package repository

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	catalogKindPart    = "part"
	catalogKindService = "service"
)

type catalogItem struct {
	ID               string    `dynamodbav:"id"`
	Kind             string    `dynamodbav:"kind"`
	Name             string    `dynamodbav:"name"`
	Description      string    `dynamodbav:"description"`
	UnitPrice        float64   `dynamodbav:"unit_price"`
	StockQuantity    int       `dynamodbav:"stock_quantity"`
	LaborPrice       float64   `dynamodbav:"labor_price"`
	EstimatedMinutes int       `dynamodbav:"estimated_minutes"`
	CreatedAt        time.Time `dynamodbav:"created_at"`
	UpdatedAt        time.Time `dynamodbav:"updated_at"`
}

type catalogTable struct {
	client    *dynamodb.Client
	tableName string
}

func (t catalogTable) put(ctx context.Context, item catalogItem) error {
	marshalled, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	_, err = t.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(t.tableName), Item: marshalled})
	return err
}

func (t catalogTable) remove(ctx context.Context, id string) error {
	_, err := t.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(t.tableName), Key: keyOf(id)})
	return err
}

func (t catalogTable) find(ctx context.Context, id, kind string) (catalogItem, bool, error) {
	output, err := t.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(t.tableName),
		Key:            keyOf(id),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return catalogItem{}, false, err
	}
	if len(output.Item) == 0 {
		return catalogItem{}, false, nil
	}

	var item catalogItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return catalogItem{}, false, err
	}
	return item, item.Kind == kind, nil
}

func (t catalogTable) list(ctx context.Context, kind string) ([]catalogItem, error) {
	paginator := dynamodb.NewScanPaginator(t.client, &dynamodb.ScanInput{
		TableName:                 aws.String(t.tableName),
		FilterExpression:          aws.String("#kind = :kind"),
		ExpressionAttributeNames:  map[string]string{"#kind": "kind"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":kind": &types.AttributeValueMemberS{Value: kind}},
		ConsistentRead:            aws.Bool(true),
	})

	items := []catalogItem{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var pageItems []catalogItem
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &pageItems); err != nil {
			return nil, err
		}
		items = append(items, pageItems...)
	}
	return items, nil
}

func keyOf(id string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}
}
