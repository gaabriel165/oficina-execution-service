package workorder

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type AddServiceToWorkOrderInput struct {
	WorkOrderID string
	ServiceID   string
}

type AddServiceToWorkOrderUseCase struct {
	workOrderRepo repository.WorkOrderRepository
	serviceRepo   repository.ServiceRepository
}

func NewAddServiceToWorkOrderUseCase(
	workOrderRepo repository.WorkOrderRepository,
	serviceRepo repository.ServiceRepository,
) *AddServiceToWorkOrderUseCase {
	return &AddServiceToWorkOrderUseCase{workOrderRepo: workOrderRepo, serviceRepo: serviceRepo}
}

func (u *AddServiceToWorkOrderUseCase) Execute(ctx context.Context, input AddServiceToWorkOrderInput) (*entity.WorkOrder, error) {
	workOrder, err := u.workOrderRepo.FindByID(ctx, input.WorkOrderID)
	if err != nil {
		return nil, err
	}

	service, err := u.serviceRepo.FindByID(ctx, input.ServiceID)
	if err != nil {
		return nil, err
	}

	if err := workOrder.AddService(service); err != nil {
		return nil, err
	}

	if err := u.workOrderRepo.Update(ctx, workOrder); err != nil {
		return nil, err
	}
	return workOrder, nil
}
