package workorder

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type CompleteDiagnosisUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewCompleteDiagnosisUseCase(workOrderRepo repository.WorkOrderRepository) *CompleteDiagnosisUseCase {
	return &CompleteDiagnosisUseCase{workOrderRepo: workOrderRepo}
}

func (u *CompleteDiagnosisUseCase) Execute(ctx context.Context, workOrderID string) (*entity.WorkOrder, error) {
	workOrder, err := u.workOrderRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, err
	}

	if err := workOrder.CompleteDiagnosis(); err != nil {
		return nil, err
	}

	completed, err := reply(messaging.TypeDiagnosisCompleted, workOrder, messaging.DiagnosisCompletedPayload{
		WorkOrderID: workOrder.ID(),
		Services:    serviceLines(workOrder),
		Parts:       partLines(workOrder),
	})
	if err != nil {
		return nil, err
	}

	if err := u.workOrderRepo.Update(ctx, workOrder, completed); err != nil {
		return nil, err
	}
	return workOrder, nil
}
