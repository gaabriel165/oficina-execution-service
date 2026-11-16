package workorder

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
)

type FailDiagnosisInput struct {
	WorkOrderID string
	Reason      string
}

type FailDiagnosisUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewFailDiagnosisUseCase(workOrderRepo repository.WorkOrderRepository) *FailDiagnosisUseCase {
	return &FailDiagnosisUseCase{workOrderRepo: workOrderRepo}
}

func (u *FailDiagnosisUseCase) Execute(ctx context.Context, input FailDiagnosisInput) (*entity.WorkOrder, error) {
	workOrder, err := u.workOrderRepo.FindByID(ctx, input.WorkOrderID)
	if err != nil {
		return nil, err
	}

	if !isBeforeDiagnosisCompletion(workOrder.Status()) {
		return nil, entity.ErrWorkOrderInvalidTransition
	}

	if err := workOrder.Cancel(input.Reason); err != nil {
		return nil, err
	}

	failed, err := reply(messaging.TypeDiagnosisFailed, workOrder, messaging.ReasonPayload{Reason: input.Reason})
	if err != nil {
		return nil, err
	}

	if err := u.workOrderRepo.Update(ctx, workOrder, failed); err != nil {
		return nil, err
	}
	return workOrder, nil
}

func isBeforeDiagnosisCompletion(status valueobject.WorkOrderStatus) bool {
	return status == valueobject.WorkOrderStatusAwaitingDiagnosis || status == valueobject.WorkOrderStatusInDiagnosis
}
