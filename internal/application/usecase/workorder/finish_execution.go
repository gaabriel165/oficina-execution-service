package workorder

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type FinishExecutionUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewFinishExecutionUseCase(workOrderRepo repository.WorkOrderRepository) *FinishExecutionUseCase {
	return &FinishExecutionUseCase{workOrderRepo: workOrderRepo}
}

func (u *FinishExecutionUseCase) Execute(ctx context.Context, workOrderID string) (*entity.WorkOrder, error) {
	workOrder, err := u.workOrderRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, err
	}

	if err := workOrder.Finish(); err != nil {
		return nil, err
	}

	finished, err := reply(messaging.TypeExecutionFinished, workOrder, messaging.WorkOrderPayload{WorkOrderID: workOrder.ID()})
	if err != nil {
		return nil, err
	}

	if err := u.workOrderRepo.Update(ctx, workOrder, finished); err != nil {
		return nil, err
	}
	return workOrder, nil
}
