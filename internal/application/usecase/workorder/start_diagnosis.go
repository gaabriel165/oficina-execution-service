package workorder

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type StartDiagnosisUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewStartDiagnosisUseCase(workOrderRepo repository.WorkOrderRepository) *StartDiagnosisUseCase {
	return &StartDiagnosisUseCase{workOrderRepo: workOrderRepo}
}

func (u *StartDiagnosisUseCase) Execute(ctx context.Context, workOrderID string) (*entity.WorkOrder, error) {
	workOrder, err := u.workOrderRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, err
	}

	if err := workOrder.StartDiagnosis(); err != nil {
		return nil, err
	}

	started, err := reply(messaging.TypeDiagnosisStarted, workOrder, messaging.WorkOrderPayload{WorkOrderID: workOrder.ID()})
	if err != nil {
		return nil, err
	}

	if err := u.workOrderRepo.Update(ctx, workOrder, started); err != nil {
		return nil, err
	}
	return workOrder, nil
}
