package entity_test

import (
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newWorkOrder(t *testing.T) *entity.WorkOrder {
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
	workOrder := newWorkOrder(t)
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

func TestNewWorkOrder_ShouldShareIdentityWithServiceOrder(t *testing.T) {
	workOrder := newWorkOrder(t)

	assert.Equal(t, "order-1", workOrder.ID())
	assert.Equal(t, "order-1", workOrder.OrderID())
	assert.Equal(t, "saga-1", workOrder.SagaID())
	assert.Equal(t, "customer-1", workOrder.CustomerID())
	assert.Equal(t, "vehicle-1", workOrder.VehicleID())
	assert.Equal(t, "engine noise", workOrder.Notes())
	assert.Equal(t, valueobject.WorkOrderStatusAwaitingDiagnosis, workOrder.Status())
	assert.Empty(t, workOrder.Services())
	assert.Empty(t, workOrder.Parts())
	assert.False(t, workOrder.StockReserved())
	assert.Nil(t, workOrder.ExecutionStartedAt())
	assert.Nil(t, workOrder.FinishedAt())
	assert.False(t, workOrder.CreatedAt().IsZero())
	assert.False(t, workOrder.UpdatedAt().IsZero())
}

func TestNewWorkOrder_ShouldRequireServiceOrder(t *testing.T) {
	_, err := entity.NewWorkOrder("", "saga-1", "customer-1", "vehicle-1", "")

	assert.ErrorIs(t, err, entity.ErrWorkOrderOrderRequired)
}

func TestNewWorkOrder_ShouldRequireSaga(t *testing.T) {
	_, err := entity.NewWorkOrder("order-1", "", "customer-1", "vehicle-1", "")

	assert.ErrorIs(t, err, entity.ErrWorkOrderSagaRequired)
}

func TestWorkOrderStartDiagnosis_ShouldMoveToInDiagnosis(t *testing.T) {
	workOrder := newWorkOrder(t)

	err := workOrder.StartDiagnosis()

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusInDiagnosis, workOrder.Status())
}

func TestWorkOrderStartDiagnosis_ShouldFailWhenAlreadyStarted(t *testing.T) {
	workOrder := workOrderInDiagnosis(t)

	err := workOrder.StartDiagnosis()

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
}

func TestWorkOrderAddService_ShouldFreezeNameAndPrice(t *testing.T) {
	workOrder := workOrderInDiagnosis(t)
	service := oilChange(t)

	err := workOrder.AddService(service)

	require.NoError(t, err)
	require.Len(t, workOrder.Services(), 1)
	assert.Equal(t, service.ID(), workOrder.Services()[0].ServiceID())
	assert.Equal(t, "Troca de óleo", workOrder.Services()[0].Name())
	assert.Equal(t, 120.0, workOrder.Services()[0].LaborPrice())
}

func TestWorkOrderAddService_ShouldFailOutsideDiagnosis(t *testing.T) {
	workOrder := newWorkOrder(t)

	err := workOrder.AddService(oilChange(t))

	assert.ErrorIs(t, err, entity.ErrWorkOrderNotInDiagnosis)
}

func TestWorkOrderAddPart_ShouldFreezeNameAndPrice(t *testing.T) {
	workOrder := workOrderInDiagnosis(t)
	part := oilFilter(t)

	err := workOrder.AddPart(part, 2)

	require.NoError(t, err)
	require.Len(t, workOrder.Parts(), 1)
	assert.Equal(t, part.ID(), workOrder.Parts()[0].PartID())
	assert.Equal(t, "Filtro de óleo", workOrder.Parts()[0].Name())
	assert.Equal(t, 2, workOrder.Parts()[0].Quantity())
	assert.Equal(t, 29.9, workOrder.Parts()[0].UnitPrice())
}

func TestWorkOrderAddPart_ShouldSumQuantityOfRepeatedPart(t *testing.T) {
	workOrder := workOrderInDiagnosis(t)
	part := oilFilter(t)
	require.NoError(t, workOrder.AddPart(part, 2))

	err := workOrder.AddPart(part, 3)

	require.NoError(t, err)
	require.Len(t, workOrder.Parts(), 1)
	assert.Equal(t, 5, workOrder.Parts()[0].Quantity())
}

func TestWorkOrderAddPart_ShouldRejectNonPositiveQuantity(t *testing.T) {
	workOrder := workOrderInDiagnosis(t)

	err := workOrder.AddPart(oilFilter(t), 0)

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidQuantity)
}

func TestWorkOrderAddPart_ShouldFailOutsideDiagnosis(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)

	err := workOrder.AddPart(oilFilter(t), 1)

	assert.ErrorIs(t, err, entity.ErrWorkOrderNotInDiagnosis)
}

func TestWorkOrderCompleteDiagnosis_ShouldRequireAtLeastOneItem(t *testing.T) {
	workOrder := workOrderInDiagnosis(t)

	err := workOrder.CompleteDiagnosis()

	assert.ErrorIs(t, err, entity.ErrWorkOrderNoItems)
	assert.Equal(t, valueobject.WorkOrderStatusInDiagnosis, workOrder.Status())
}

func TestWorkOrderCompleteDiagnosis_ShouldFailBeforeDiagnosisStarts(t *testing.T) {
	workOrder := newWorkOrder(t)

	err := workOrder.CompleteDiagnosis()

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
}

func TestWorkOrderStartExecution_ShouldReserveStockAndRecordStart(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)

	err := workOrder.StartExecution()

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusInExecution, workOrder.Status())
	assert.True(t, workOrder.StockReserved())
	assert.NotNil(t, workOrder.ExecutionStartedAt())
}

func TestWorkOrderStartExecution_ShouldFailBeforeDiagnosisIsComplete(t *testing.T) {
	workOrder := workOrderInDiagnosis(t)

	err := workOrder.StartExecution()

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
	assert.False(t, workOrder.StockReserved())
}

func TestWorkOrderFinish_ShouldRecordFinishTime(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)
	require.NoError(t, workOrder.StartExecution())

	err := workOrder.Finish()

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusFinished, workOrder.Status())
	assert.NotNil(t, workOrder.FinishedAt())
}

func TestWorkOrderFinish_ShouldFailWhenNotInExecution(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)

	err := workOrder.Finish()

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
	assert.Nil(t, workOrder.FinishedAt())
}

func TestWorkOrderCancel_ShouldStoreReason(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)

	err := workOrder.Cancel("budget rejected")

	require.NoError(t, err)
	assert.True(t, workOrder.IsCancelled())
	assert.Equal(t, "budget rejected", workOrder.CancellationReason())
}

func TestWorkOrderCancel_ShouldRequireReason(t *testing.T) {
	workOrder := newWorkOrder(t)

	err := workOrder.Cancel("")

	assert.ErrorIs(t, err, entity.ErrWorkOrderReasonRequired)
	assert.False(t, workOrder.IsCancelled())
}

func TestWorkOrderCancel_ShouldFailWhenFinished(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)
	require.NoError(t, workOrder.StartExecution())
	require.NoError(t, workOrder.Finish())

	err := workOrder.Cancel("too late")

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
	assert.Empty(t, workOrder.CancellationReason())
}

func TestWorkOrderReleaseStock_ShouldClearReservation(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)
	require.NoError(t, workOrder.StartExecution())

	workOrder.ReleaseStock()

	assert.False(t, workOrder.StockReserved())
}

func TestWorkOrderExecutionDuration_ShouldBeZeroWhileNotFinished(t *testing.T) {
	workOrder := diagnosedWorkOrder(t)
	require.NoError(t, workOrder.StartExecution())

	assert.Zero(t, workOrder.ExecutionDuration())
}

func TestWorkOrderExecutionDuration_ShouldMeasureFromStartToFinish(t *testing.T) {
	startedAt := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(90 * time.Minute)
	workOrder := entity.RestoreWorkOrder(entity.WorkOrderSnapshot{
		ID:                 "order-1",
		SagaID:             "saga-1",
		Status:             valueobject.WorkOrderStatusFinished,
		Services:           []*entity.WorkOrderService{entity.RestoreWorkOrderService("service-1", "Troca de óleo", 120)},
		Parts:              []*entity.WorkOrderPart{entity.RestoreWorkOrderPart("part-1", "Filtro de óleo", 2, 29.9)},
		ExecutionStartedAt: &startedAt,
		FinishedAt:         &finishedAt,
	})

	assert.Equal(t, 90*time.Minute, workOrder.ExecutionDuration())
	assert.Equal(t, "service-1", workOrder.Services()[0].ServiceID())
	assert.Equal(t, "part-1", workOrder.Parts()[0].PartID())
}
