package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	domainrepo "github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
)

const conditionalCheckFailedCode = "ConditionalCheckFailed"

type workOrderServiceItem struct {
	ServiceID  string  `dynamodbav:"service_id"`
	Name       string  `dynamodbav:"name"`
	LaborPrice float64 `dynamodbav:"labor_price"`
}

type workOrderPartItem struct {
	PartID    string  `dynamodbav:"part_id"`
	Name      string  `dynamodbav:"name"`
	Quantity  int     `dynamodbav:"quantity"`
	UnitPrice float64 `dynamodbav:"unit_price"`
}

type workOrderItem struct {
	ID                 string                 `dynamodbav:"id"`
	SagaID             string                 `dynamodbav:"saga_id"`
	CustomerID         string                 `dynamodbav:"customer_id"`
	VehicleID          string                 `dynamodbav:"vehicle_id"`
	Notes              string                 `dynamodbav:"notes"`
	Status             string                 `dynamodbav:"status"`
	Services           []workOrderServiceItem `dynamodbav:"services"`
	Parts              []workOrderPartItem    `dynamodbav:"parts"`
	StockReserved      bool                   `dynamodbav:"stock_reserved"`
	CancellationReason string                 `dynamodbav:"cancellation_reason"`
	ExecutionStartedAt *time.Time             `dynamodbav:"execution_started_at"`
	FinishedAt         *time.Time             `dynamodbav:"finished_at"`
	CreatedAt          time.Time              `dynamodbav:"created_at"`
	UpdatedAt          time.Time              `dynamodbav:"updated_at"`
}

type DynamoWorkOrderRepository struct {
	client       *dynamodb.Client
	tableName    string
	catalogTable string
	outboxTable  string
}

func NewDynamoWorkOrderRepository(client *dynamodb.Client, tableName, catalogTable, outboxTable string) *DynamoWorkOrderRepository {
	return &DynamoWorkOrderRepository{client: client, tableName: tableName, catalogTable: catalogTable, outboxTable: outboxTable}
}

func (r *DynamoWorkOrderRepository) Create(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error {
	put, err := r.putWorkOrder(workOrder)
	if err != nil {
		return err
	}
	put.Put.ConditionExpression = aws.String("attribute_not_exists(id)")
	return r.write(ctx, []types.TransactWriteItem{put}, outgoing)
}

func (r *DynamoWorkOrderRepository) Update(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error {
	put, err := r.putWorkOrder(workOrder)
	if err != nil {
		return err
	}
	return r.write(ctx, []types.TransactWriteItem{put}, outgoing)
}

func (r *DynamoWorkOrderRepository) UpdateReservingStock(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error {
	put, err := r.putWorkOrder(workOrder)
	if err != nil {
		return err
	}

	writes := []types.TransactWriteItem{put}
	for _, part := range workOrder.Parts() {
		writes = append(writes, r.stockMovement(part, "stock_quantity - :quantity", "attribute_exists(id) AND stock_quantity >= :quantity"))
	}

	err = r.write(ctx, writes, outgoing)
	if isConditionalCheckFailure(err) {
		return entity.ErrPartInsufficientStock
	}
	return err
}

func (r *DynamoWorkOrderRepository) UpdateReleasingStock(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error {
	put, err := r.putWorkOrder(workOrder)
	if err != nil {
		return err
	}

	writes := []types.TransactWriteItem{put}
	for _, part := range workOrder.Parts() {
		writes = append(writes, r.stockMovement(part, "stock_quantity + :quantity", "attribute_exists(id)"))
	}
	return r.write(ctx, writes, outgoing)
}

func (r *DynamoWorkOrderRepository) Enqueue(ctx context.Context, outgoing ...messaging.OutgoingMessage) error {
	return r.write(ctx, nil, outgoing)
}

func (r *DynamoWorkOrderRepository) FindByID(ctx context.Context, id string) (*entity.WorkOrder, error) {
	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(r.tableName),
		Key:            keyOf(id),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if len(output.Item) == 0 {
		return nil, domainrepo.ErrWorkOrderNotFound
	}

	var item workOrderItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return nil, err
	}
	return workOrderFromItem(item), nil
}

func (r *DynamoWorkOrderRepository) FindByStatus(ctx context.Context, status valueobject.WorkOrderStatus) ([]*entity.WorkOrder, error) {
	return r.scan(ctx, &dynamodb.ScanInput{
		TableName:                 aws.String(r.tableName),
		FilterExpression:          aws.String("#status = :status"),
		ExpressionAttributeNames:  map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":status": &types.AttributeValueMemberS{Value: status.String()}},
		ConsistentRead:            aws.Bool(true),
	})
}

func (r *DynamoWorkOrderRepository) FindAll(ctx context.Context) ([]*entity.WorkOrder, error) {
	return r.scan(ctx, &dynamodb.ScanInput{TableName: aws.String(r.tableName), ConsistentRead: aws.Bool(true)})
}

func (r *DynamoWorkOrderRepository) scan(ctx context.Context, input *dynamodb.ScanInput) ([]*entity.WorkOrder, error) {
	paginator := dynamodb.NewScanPaginator(r.client, input)

	workOrders := []*entity.WorkOrder{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var items []workOrderItem
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &items); err != nil {
			return nil, err
		}
		for _, item := range items {
			workOrders = append(workOrders, workOrderFromItem(item))
		}
	}
	return workOrders, nil
}

func (r *DynamoWorkOrderRepository) write(ctx context.Context, writes []types.TransactWriteItem, outgoing []messaging.OutgoingMessage) error {
	outbox, err := outboxWrites(ctx, r.outboxTable, outgoing)
	if err != nil {
		return err
	}

	writes = append(writes, outbox...)
	if len(writes) == 0 {
		return nil
	}

	_, err = r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: writes})
	return err
}

func (r *DynamoWorkOrderRepository) putWorkOrder(workOrder *entity.WorkOrder) (types.TransactWriteItem, error) {
	marshalled, err := attributevalue.MarshalMap(workOrderItemFromDomain(workOrder))
	if err != nil {
		return types.TransactWriteItem{}, err
	}
	return types.TransactWriteItem{Put: &types.Put{TableName: aws.String(r.tableName), Item: marshalled}}, nil
}

func (r *DynamoWorkOrderRepository) stockMovement(part *entity.WorkOrderPart, movement, condition string) types.TransactWriteItem {
	return types.TransactWriteItem{Update: &types.Update{
		TableName:           aws.String(r.catalogTable),
		Key:                 keyOf(part.PartID()),
		UpdateExpression:    aws.String("SET stock_quantity = " + movement + ", updated_at = :now"),
		ConditionExpression: aws.String(condition),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":quantity": &types.AttributeValueMemberN{Value: strconv.Itoa(part.Quantity())},
			":now":      &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339Nano)},
		},
	}}
}

func isConditionalCheckFailure(err error) bool {
	var cancelled *types.TransactionCanceledException
	if !errors.As(err, &cancelled) {
		return false
	}
	for _, reason := range cancelled.CancellationReasons {
		if aws.ToString(reason.Code) == conditionalCheckFailedCode {
			return true
		}
	}
	return false
}

func workOrderItemFromDomain(workOrder *entity.WorkOrder) workOrderItem {
	services := make([]workOrderServiceItem, 0, len(workOrder.Services()))
	for _, service := range workOrder.Services() {
		services = append(services, workOrderServiceItem{ServiceID: service.ServiceID(), Name: service.Name(), LaborPrice: service.LaborPrice()})
	}

	parts := make([]workOrderPartItem, 0, len(workOrder.Parts()))
	for _, part := range workOrder.Parts() {
		parts = append(parts, workOrderPartItem{PartID: part.PartID(), Name: part.Name(), Quantity: part.Quantity(), UnitPrice: part.UnitPrice()})
	}

	return workOrderItem{
		ID:                 workOrder.ID(),
		SagaID:             workOrder.SagaID(),
		CustomerID:         workOrder.CustomerID(),
		VehicleID:          workOrder.VehicleID(),
		Notes:              workOrder.Notes(),
		Status:             workOrder.Status().String(),
		Services:           services,
		Parts:              parts,
		StockReserved:      workOrder.StockReserved(),
		CancellationReason: workOrder.CancellationReason(),
		ExecutionStartedAt: workOrder.ExecutionStartedAt(),
		FinishedAt:         workOrder.FinishedAt(),
		CreatedAt:          workOrder.CreatedAt(),
		UpdatedAt:          workOrder.UpdatedAt(),
	}
}

func workOrderFromItem(item workOrderItem) *entity.WorkOrder {
	services := make([]*entity.WorkOrderService, 0, len(item.Services))
	for _, service := range item.Services {
		services = append(services, entity.RestoreWorkOrderService(service.ServiceID, service.Name, service.LaborPrice))
	}

	parts := make([]*entity.WorkOrderPart, 0, len(item.Parts))
	for _, part := range item.Parts {
		parts = append(parts, entity.RestoreWorkOrderPart(part.PartID, part.Name, part.Quantity, part.UnitPrice))
	}

	return entity.RestoreWorkOrder(entity.WorkOrderSnapshot{
		ID:                 item.ID,
		SagaID:             item.SagaID,
		CustomerID:         item.CustomerID,
		VehicleID:          item.VehicleID,
		Notes:              item.Notes,
		Status:             valueobject.WorkOrderStatus(item.Status),
		Services:           services,
		Parts:              parts,
		StockReserved:      item.StockReserved,
		CancellationReason: item.CancellationReason,
		ExecutionStartedAt: item.ExecutionStartedAt,
		FinishedAt:         item.FinishedAt,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	})
}
