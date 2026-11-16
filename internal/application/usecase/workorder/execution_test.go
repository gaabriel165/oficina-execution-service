package workorder_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartExecution_ShouldReserveStockAndReplyToSaga(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := diagnosedWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("UpdateReservingStock", ctx, existing, replyOfType(messaging.TypeExecutionStarted)).Return(nil)

	err := workorder.NewStartExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusInExecution, existing.Status())
	assert.True(t, existing.StockReserved())
}

func TestStartExecution_ShouldRejectWhenStockIsInsufficient(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := diagnosedWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("UpdateReservingStock", ctx, existing, replyOfType(messaging.TypeExecutionStarted)).Return(entity.ErrPartInsufficientStock)
	workOrderRepo.On("Enqueue", ctx, replyOfType(messaging.TypeExecutionRejected)).Return(nil)

	err := workorder.NewStartExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.NoError(t, err)
}

func TestStartExecution_ShouldRejectWhenDiagnosisIsNotComplete(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(workOrderInDiagnosis(t), nil)
	workOrderRepo.On("Enqueue", ctx, replyOfType(messaging.TypeExecutionRejected)).Return(nil)

	err := workorder.NewStartExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.NoError(t, err)
}

func TestStartExecution_ShouldIgnoreRepeatedCommand(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(workOrderInExecution(t), nil)

	err := workorder.NewStartExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.NoError(t, err)
}

func TestStartExecution_ShouldReturnErrorWhenNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	err := workorder.NewStartExecutionUseCase(workOrderRepo).Execute(ctx, "missing")

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestStartExecution_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := diagnosedWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("UpdateReservingStock", ctx, existing, replyOfType(messaging.TypeExecutionStarted)).Return(errDatabase)

	err := workorder.NewStartExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.ErrorIs(t, err, errDatabase)
}

func TestFinishExecution_ShouldFinishAndReplyToSaga(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInExecution(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeExecutionFinished)).Return(nil)

	result, err := workorder.NewFinishExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusFinished, result.Status())
	assert.NotNil(t, result.FinishedAt())
}

func TestFinishExecution_ShouldReturnErrorWhenNotInExecution(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(diagnosedWorkOrder(t), nil)

	_, err := workorder.NewFinishExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
}

func TestFinishExecution_ShouldReturnErrorWhenNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewFinishExecutionUseCase(workOrderRepo).Execute(ctx, "missing")

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestFinishExecution_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInExecution(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeExecutionFinished)).Return(errDatabase)

	_, err := workorder.NewFinishExecutionUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.ErrorIs(t, err, errDatabase)
}

var cancelInput = workorder.CancelWorkOrderInput{OrderID: "order-1", SagaID: "saga-1", Reason: "payment failed"}

func TestCancelWorkOrder_ShouldCancelWithoutTouchingStockBeforeExecution(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := diagnosedWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeWorkOrderCancelled)).Return(nil)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, cancelInput)

	require.NoError(t, err)
	assert.True(t, existing.IsCancelled())
	assert.Equal(t, "payment failed", existing.CancellationReason())
}

func TestCancelWorkOrder_ShouldReleaseReservedStock(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInExecution(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("UpdateReleasingStock", ctx, existing, replyOfType(messaging.TypeWorkOrderCancelled)).Return(nil)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, cancelInput)

	require.NoError(t, err)
	assert.True(t, existing.IsCancelled())
	assert.False(t, existing.StockReserved())
}

func TestCancelWorkOrder_ShouldUseDefaultReasonWhenNoneIsGiven(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := awaitingWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeWorkOrderCancelled)).Return(nil)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, workorder.CancelWorkOrderInput{OrderID: "order-1", SagaID: "saga-1"})

	require.NoError(t, err)
	assert.Equal(t, "cancelled by the service order saga", existing.CancellationReason())
}

func TestCancelWorkOrder_ShouldOnlyReplyWhenWorkOrderDoesNotExist(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(nil, repository.ErrWorkOrderNotFound)
	workOrderRepo.On("Enqueue", ctx, replyOfType(messaging.TypeWorkOrderCancelled)).Return(nil)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, cancelInput)

	assert.NoError(t, err)
}

func TestCancelWorkOrder_ShouldOnlyReplyWhenAlreadyCancelled(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := awaitingWorkOrder(t)
	require.NoError(t, existing.Cancel("diagnosis failed"))
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Enqueue", ctx, replyOfType(messaging.TypeWorkOrderCancelled)).Return(nil)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, cancelInput)

	require.NoError(t, err)
	assert.Equal(t, "diagnosis failed", existing.CancellationReason())
}

func TestCancelWorkOrder_ShouldReturnErrorWhenAlreadyFinished(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInExecution(t)
	require.NoError(t, existing.Finish())
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, cancelInput)

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
}

func TestCancelWorkOrder_ShouldReturnErrorWhenLookupFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(nil, errDatabase)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, cancelInput)

	assert.ErrorIs(t, err, errDatabase)
}

func TestCancelWorkOrder_ShouldReturnErrorWhenSagaIsMissing(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)

	err := workorder.NewCancelWorkOrderUseCase(workOrderRepo).Execute(ctx, workorder.CancelWorkOrderInput{OrderID: "order-1"})

	assert.ErrorIs(t, err, messaging.ErrMessageSagaRequired)
}
