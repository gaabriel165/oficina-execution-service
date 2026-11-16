package repository

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	domainrepo "github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type DynamoServiceRepository struct {
	table catalogTable
}

func NewDynamoServiceRepository(client *dynamodb.Client, tableName string) *DynamoServiceRepository {
	return &DynamoServiceRepository{table: catalogTable{client: client, tableName: tableName}}
}

func (r *DynamoServiceRepository) Create(ctx context.Context, service *entity.Service) error {
	return r.table.put(ctx, serviceItemFromDomain(service))
}

func (r *DynamoServiceRepository) Update(ctx context.Context, service *entity.Service) error {
	return r.table.put(ctx, serviceItemFromDomain(service))
}

func (r *DynamoServiceRepository) Delete(ctx context.Context, id string) error {
	return r.table.remove(ctx, id)
}

func (r *DynamoServiceRepository) FindByID(ctx context.Context, id string) (*entity.Service, error) {
	item, found, err := r.table.find(ctx, id, catalogKindService)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, domainrepo.ErrServiceNotFound
	}
	return serviceFromItem(item), nil
}

func (r *DynamoServiceRepository) FindAll(ctx context.Context) ([]*entity.Service, error) {
	items, err := r.table.list(ctx, catalogKindService)
	if err != nil {
		return nil, err
	}

	services := make([]*entity.Service, len(items))
	for index, item := range items {
		services[index] = serviceFromItem(item)
	}
	return services, nil
}

func serviceItemFromDomain(service *entity.Service) catalogItem {
	return catalogItem{
		ID:               service.ID(),
		Kind:             catalogKindService,
		Name:             service.Name(),
		Description:      service.Description(),
		LaborPrice:       service.LaborPrice(),
		EstimatedMinutes: service.EstimatedMinutes(),
		CreatedAt:        service.CreatedAt(),
		UpdatedAt:        service.UpdatedAt(),
	}
}

func serviceFromItem(item catalogItem) *entity.Service {
	return entity.RestoreService(item.ID, item.Name, item.Description, item.LaborPrice, item.EstimatedMinutes, item.CreatedAt, item.UpdatedAt)
}
