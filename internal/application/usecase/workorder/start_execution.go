package workorder

import (
	"context"
	"errors"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
)

type StartExecutionUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewStartExecutionUseCase(workOrderRepo repository.WorkOrderRepository) *StartExecutionUseCase {
	return &StartExecutionUseCase{workOrderRepo: workOrderRepo}
}

func (u *StartExecutionUseCase) Execute(ctx context.Context, workOrderID string) error {
	workOrder, err := u.workOrderRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return err
	}

	if hasExecutionStarted(workOrder.Status()) {
		return nil
	}

	if err := workOrder.StartExecution(); err != nil {
		return u.reject(ctx, workOrder, err)
	}

	started, err := reply(messaging.TypeExecutionStarted, workOrder, messaging.WorkOrderPayload{WorkOrderID: workOrder.ID()})
	if err != nil {
		return err
	}

	err = u.workOrderRepo.UpdateReservingStock(ctx, workOrder, started)
	if errors.Is(err, entity.ErrPartInsufficientStock) {
		return u.reject(ctx, workOrder, err)
	}
	return err
}

func (u *StartExecutionUseCase) reject(ctx context.Context, workOrder *entity.WorkOrder, cause error) error {
	rejected, err := reply(messaging.TypeExecutionRejected, workOrder, messaging.ReasonPayload{Reason: cause.Error()})
	if err != nil {
		return err
	}
	return u.workOrderRepo.Enqueue(ctx, rejected)
}

func hasExecutionStarted(status valueobject.WorkOrderStatus) bool {
	return status == valueobject.WorkOrderStatusInExecution || status == valueobject.WorkOrderStatusFinished
}
