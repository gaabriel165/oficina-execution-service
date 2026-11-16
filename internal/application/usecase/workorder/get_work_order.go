package workorder

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type GetWorkOrderUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewGetWorkOrderUseCase(workOrderRepo repository.WorkOrderRepository) *GetWorkOrderUseCase {
	return &GetWorkOrderUseCase{workOrderRepo: workOrderRepo}
}

func (u *GetWorkOrderUseCase) Execute(ctx context.Context, workOrderID string) (*entity.WorkOrder, error) {
	return u.workOrderRepo.FindByID(ctx, workOrderID)
}
