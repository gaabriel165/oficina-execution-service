package workorder

import (
	"context"
	"errors"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

const defaultCancellationReason = "cancelled by the service order saga"

type CancelWorkOrderInput struct {
	OrderID string
	SagaID  string
	Reason  string
}

type CancelWorkOrderUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewCancelWorkOrderUseCase(workOrderRepo repository.WorkOrderRepository) *CancelWorkOrderUseCase {
	return &CancelWorkOrderUseCase{workOrderRepo: workOrderRepo}
}

func (u *CancelWorkOrderUseCase) Execute(ctx context.Context, input CancelWorkOrderInput) error {
	cancelled, err := replyTo(messaging.TypeWorkOrderCancelled, input.SagaID, input.OrderID, nil)
	if err != nil {
		return err
	}

	workOrder, err := u.workOrderRepo.FindByID(ctx, input.OrderID)
	if errors.Is(err, repository.ErrWorkOrderNotFound) {
		return u.workOrderRepo.Enqueue(ctx, cancelled)
	}
	if err != nil {
		return err
	}

	if workOrder.IsCancelled() {
		return u.workOrderRepo.Enqueue(ctx, cancelled)
	}

	hadStockReserved := workOrder.StockReserved()
	if err := workOrder.Cancel(reasonOrDefault(input.Reason)); err != nil {
		return err
	}

	if !hadStockReserved {
		return u.workOrderRepo.Update(ctx, workOrder, cancelled)
	}

	workOrder.ReleaseStock()
	return u.workOrderRepo.UpdateReleasingStock(ctx, workOrder, cancelled)
}

func reasonOrDefault(reason string) string {
	if reason == "" {
		return defaultCancellationReason
	}
	return reason
}
