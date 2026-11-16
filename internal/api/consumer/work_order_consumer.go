package consumer

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
)

type Registrar interface {
	Register(messageType string, handler messaging.Handler)
}

type WorkOrderConsumer struct {
	openUC           *workorder.OpenWorkOrderUseCase
	startExecutionUC *workorder.StartExecutionUseCase
	cancelUC         *workorder.CancelWorkOrderUseCase
}

func NewWorkOrderConsumer(
	openUC *workorder.OpenWorkOrderUseCase,
	startExecutionUC *workorder.StartExecutionUseCase,
	cancelUC *workorder.CancelWorkOrderUseCase,
) *WorkOrderConsumer {
	return &WorkOrderConsumer{openUC: openUC, startExecutionUC: startExecutionUC, cancelUC: cancelUC}
}

func (c *WorkOrderConsumer) RegisterHandlers(registrar Registrar) {
	registrar.Register(messaging.TypeRequestDiagnosis, c.RequestDiagnosis)
	registrar.Register(messaging.TypeStartExecution, c.StartExecution)
	registrar.Register(messaging.TypeCancelWorkOrder, c.CancelWorkOrder)
}

func (c *WorkOrderConsumer) RequestDiagnosis(ctx context.Context, message messaging.Message) error {
	var payload messaging.RequestDiagnosisPayload
	if err := message.DecodePayload(&payload); err != nil {
		return err
	}

	_, err := c.openUC.Execute(ctx, workorder.OpenWorkOrderInput{
		OrderID:    message.OrderID,
		SagaID:     message.SagaID,
		CustomerID: payload.CustomerID,
		VehicleID:  payload.VehicleID,
		Notes:      payload.Notes,
	})
	return err
}

func (c *WorkOrderConsumer) StartExecution(ctx context.Context, message messaging.Message) error {
	return c.startExecutionUC.Execute(ctx, message.OrderID)
}

func (c *WorkOrderConsumer) CancelWorkOrder(ctx context.Context, message messaging.Message) error {
	var payload messaging.ReasonPayload
	if err := message.DecodePayload(&payload); err != nil {
		return err
	}

	return c.cancelUC.Execute(ctx, workorder.CancelWorkOrderInput{
		OrderID: message.OrderID,
		SagaID:  message.SagaID,
		Reason:  payload.Reason,
	})
}
