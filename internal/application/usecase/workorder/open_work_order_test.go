package workorder_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var openInput = workorder.OpenWorkOrderInput{
	OrderID: "order-1", SagaID: "saga-1", CustomerID: "customer-1", VehicleID: "vehicle-1", Notes: "engine noise",
}

func TestOpenWorkOrder_ShouldCreateWorkOrderAwaitingDiagnosis(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(nil, repository.ErrWorkOrderNotFound)
	workOrderRepo.On("Create", ctx, mock.AnythingOfType("*entity.WorkOrder")).Return(nil)

	result, err := workorder.NewOpenWorkOrderUseCase(workOrderRepo).Execute(ctx, openInput)

	require.NoError(t, err)
	assert.Equal(t, "order-1", result.ID())
	assert.Equal(t, valueobject.WorkOrderStatusAwaitingDiagnosis, result.Status())
}

func TestOpenWorkOrder_ShouldReturnExistingWhenCommandIsRepeated(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInDiagnosis(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)

	result, err := workorder.NewOpenWorkOrderUseCase(workOrderRepo).Execute(ctx, openInput)

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusInDiagnosis, result.Status())
}

func TestOpenWorkOrder_ShouldReturnErrorWhenLookupFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(nil, errDatabase)

	_, err := workorder.NewOpenWorkOrderUseCase(workOrderRepo).Execute(ctx, openInput)

	assert.ErrorIs(t, err, errDatabase)
}

func TestOpenWorkOrder_ShouldReturnErrorWhenSagaIsMissing(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewOpenWorkOrderUseCase(workOrderRepo).Execute(ctx, workorder.OpenWorkOrderInput{OrderID: "order-1"})

	assert.ErrorIs(t, err, entity.ErrWorkOrderSagaRequired)
}

func TestOpenWorkOrder_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(nil, repository.ErrWorkOrderNotFound)
	workOrderRepo.On("Create", ctx, mock.AnythingOfType("*entity.WorkOrder")).Return(errDatabase)

	_, err := workorder.NewOpenWorkOrderUseCase(workOrderRepo).Execute(ctx, openInput)

	assert.ErrorIs(t, err, errDatabase)
}
