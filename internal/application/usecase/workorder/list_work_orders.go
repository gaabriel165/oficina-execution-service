package workorder

import (
	"context"
	"sort"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
)

type ListWorkOrdersUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewListWorkOrdersUseCase(workOrderRepo repository.WorkOrderRepository) *ListWorkOrdersUseCase {
	return &ListWorkOrdersUseCase{workOrderRepo: workOrderRepo}
}

func (u *ListWorkOrdersUseCase) Execute(ctx context.Context, statusFilter string) ([]*entity.WorkOrder, error) {
	workOrders, err := u.find(ctx, statusFilter)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(workOrders, func(first, second int) bool {
		return workOrders[first].CreatedAt().Before(workOrders[second].CreatedAt())
	})
	return workOrders, nil
}

func (u *ListWorkOrdersUseCase) find(ctx context.Context, statusFilter string) ([]*entity.WorkOrder, error) {
	if statusFilter == "" {
		return u.workOrderRepo.FindAll(ctx)
	}

	status, err := valueobject.NewWorkOrderStatus(statusFilter)
	if err != nil {
		return nil, err
	}
	return u.workOrderRepo.FindByStatus(ctx, status)
}
