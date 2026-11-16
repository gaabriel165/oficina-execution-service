package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	domainrepo "github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type DynamoPartRepository struct {
	table catalogTable
}

func NewDynamoPartRepository(client *dynamodb.Client, tableName string) *DynamoPartRepository {
	return &DynamoPartRepository{table: catalogTable{client: client, tableName: tableName}}
}

func (r *DynamoPartRepository) Create(ctx context.Context, part *entity.Part) error {
	return r.table.put(ctx, partItemFromDomain(part))
}

func (r *DynamoPartRepository) Update(ctx context.Context, part *entity.Part) error {
	return r.table.put(ctx, partItemFromDomain(part))
}

func (r *DynamoPartRepository) Delete(ctx context.Context, id string) error {
	return r.table.remove(ctx, id)
}

func (r *DynamoPartRepository) FindByID(ctx context.Context, id string) (*entity.Part, error) {
	item, found, err := r.table.find(ctx, id, catalogKindPart)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, domainrepo.ErrPartNotFound
	}
	return partFromItem(item), nil
}

func (r *DynamoPartRepository) FindAll(ctx context.Context) ([]*entity.Part, error) {
	items, err := r.table.list(ctx, catalogKindPart)
	if err != nil {
		return nil, err
	}

	parts := make([]*entity.Part, len(items))
	for index, item := range items {
		parts[index] = partFromItem(item)
	}
	return parts, nil
}

func partItemFromDomain(part *entity.Part) catalogItem {
	return catalogItem{
		ID:            part.ID(),
		Kind:          catalogKindPart,
		Name:          part.Name(),
		Description:   part.Description(),
		UnitPrice:     part.UnitPrice(),
		StockQuantity: part.StockQuantity(),
		CreatedAt:     part.CreatedAt(),
		UpdatedAt:     part.UpdatedAt(),
	}
}

func partFromItem(item catalogItem) *entity.Part {
	return entity.RestorePart(item.ID, item.Name, item.Description, item.UnitPrice, item.StockQuantity, item.CreatedAt, item.UpdatedAt)
}
