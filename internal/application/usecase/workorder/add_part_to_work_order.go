package workorder

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type AddPartToWorkOrderInput struct {
	WorkOrderID string
	PartID      string
	Quantity    int
}

type AddPartToWorkOrderUseCase struct {
	workOrderRepo repository.WorkOrderRepository
	partRepo      repository.PartRepository
}

func NewAddPartToWorkOrderUseCase(
	workOrderRepo repository.WorkOrderRepository,
	partRepo repository.PartRepository,
) *AddPartToWorkOrderUseCase {
	return &AddPartToWorkOrderUseCase{workOrderRepo: workOrderRepo, partRepo: partRepo}
}

func (u *AddPartToWorkOrderUseCase) Execute(ctx context.Context, input AddPartToWorkOrderInput) (*entity.WorkOrder, error) {
	workOrder, err := u.workOrderRepo.FindByID(ctx, input.WorkOrderID)
	if err != nil {
		return nil, err
	}

	part, err := u.partRepo.FindByID(ctx, input.PartID)
	if err != nil {
		return nil, err
	}

	if err := workOrder.AddPart(part, input.Quantity); err != nil {
		return nil, err
	}

	if err := u.workOrderRepo.Update(ctx, workOrder); err != nil {
		return nil, err
	}
	return workOrder, nil
}
