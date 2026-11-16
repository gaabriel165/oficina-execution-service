package workorder_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	ctx         = context.Background()
	errDatabase = errors.New("database unavailable")
)

func awaitingWorkOrder(t *testing.T) *entity.WorkOrder {
	t.Helper()
	workOrder, err := entity.NewWorkOrder("order-1", "saga-1", "customer-1", "vehicle-1", "engine noise")
	require.NoError(t, err)
	return workOrder
}

func oilChange(t *testing.T) *entity.Service {
	t.Helper()
	service, err := entity.NewService("Troca de óleo", "Troca completa", 120, 45)
	require.NoError(t, err)
	return service
}

func oilFilter(t *testing.T) *entity.Part {
	t.Helper()
	part, err := entity.NewPart("Filtro de óleo", "Filtro padrão", 29.9, 10)
	require.NoError(t, err)
	return part
}

func workOrderInDiagnosis(t *testing.T) *entity.WorkOrder {
	t.Helper()
	workOrder := awaitingWorkOrder(t)
	require.NoError(t, workOrder.StartDiagnosis())
	return workOrder
}

func diagnosedWorkOrder(t *testing.T) *entity.WorkOrder {
	t.Helper()
	workOrder := workOrderInDiagnosis(t)
	require.NoError(t, workOrder.AddService(oilChange(t)))
	require.NoError(t, workOrder.AddPart(oilFilter(t), 2))
	require.NoError(t, workOrder.CompleteDiagnosis())
	return workOrder
}

func workOrderInExecution(t *testing.T) *entity.WorkOrder {
	t.Helper()
	workOrder := diagnosedWorkOrder(t)
	require.NoError(t, workOrder.StartExecution())
	return workOrder
}

func replyOfType(messageType string) any {
	return mock.MatchedBy(func(outgoing []messaging.OutgoingMessage) bool {
		return len(outgoing) == 1 &&
			outgoing[0].Destination == messaging.DestinationSagaReplies &&
			outgoing[0].Message.Type == messageType &&
			outgoing[0].Message.SagaID == "saga-1" &&
			outgoing[0].Message.OrderID == "order-1"
	})
}
