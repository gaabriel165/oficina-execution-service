package workorder

import (
	"context"
	"errors"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type OpenWorkOrderInput struct {
	OrderID    string
	SagaID     string
	CustomerID string
	VehicleID  string
	Notes      string
}

type OpenWorkOrderUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewOpenWorkOrderUseCase(workOrderRepo repository.WorkOrderRepository) *OpenWorkOrderUseCase {
	return &OpenWorkOrderUseCase{workOrderRepo: workOrderRepo}
}

func (u *OpenWorkOrderUseCase) Execute(ctx context.Context, input OpenWorkOrderInput) (*entity.WorkOrder, error) {
	existing, err := u.workOrderRepo.FindByID(ctx, input.OrderID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, repository.ErrWorkOrderNotFound) {
		return nil, err
	}

	workOrder, err := entity.NewWorkOrder(input.OrderID, input.SagaID, input.CustomerID, input.VehicleID, input.Notes)
	if err != nil {
		return nil, err
	}

	if err := u.workOrderRepo.Create(ctx, workOrder); err != nil {
		return nil, err
	}
	return workOrder, nil
}
