package workorder_test

import (
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finishedWorkOrder(id string, createdAt time.Time, minutes int, serviceNames ...string) *entity.WorkOrder {
	finishedAt := createdAt.Add(time.Duration(minutes) * time.Minute)
	services := make([]*entity.WorkOrderService, 0, len(serviceNames))
	for _, name := range serviceNames {
		services = append(services, entity.RestoreWorkOrderService("service-"+name, name, 100))
	}
	return entity.RestoreWorkOrder(entity.WorkOrderSnapshot{
		ID:                 id,
		SagaID:             "saga-" + id,
		Status:             valueobject.WorkOrderStatusFinished,
		Services:           services,
		ExecutionStartedAt: &createdAt,
		FinishedAt:         &finishedAt,
		CreatedAt:          createdAt,
	})
}

func TestGetWorkOrder_ShouldReturnWorkOrder(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(awaitingWorkOrder(t), nil)

	result, err := workorder.NewGetWorkOrderUseCase(workOrderRepo).Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.Equal(t, "order-1", result.ID())
}

func TestGetWorkOrder_ShouldReturnErrorWhenNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewGetWorkOrderUseCase(workOrderRepo).Execute(ctx, "missing")

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestListWorkOrders_ShouldReturnOldestFirst(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	morning := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	newer := finishedWorkOrder("order-2", morning.Add(time.Hour), 30, "Alinhamento")
	older := finishedWorkOrder("order-1", morning, 30, "Alinhamento")
	workOrderRepo.On("FindAll", ctx).Return([]*entity.WorkOrder{newer, older}, nil)

	result, err := workorder.NewListWorkOrdersUseCase(workOrderRepo).Execute(ctx, "")

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "order-1", result[0].ID())
	assert.Equal(t, "order-2", result[1].ID())
}

func TestListWorkOrders_ShouldFilterByStatus(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByStatus", ctx, valueobject.WorkOrderStatusAwaitingDiagnosis).Return([]*entity.WorkOrder{awaitingWorkOrder(t)}, nil)

	result, err := workorder.NewListWorkOrdersUseCase(workOrderRepo).Execute(ctx, "awaiting_diagnosis")

	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListWorkOrders_ShouldRejectUnknownStatus(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)

	_, err := workorder.NewListWorkOrdersUseCase(workOrderRepo).Execute(ctx, "unknown")

	assert.ErrorIs(t, err, valueobject.ErrInvalidWorkOrderStatus)
}

func TestListWorkOrders_ShouldReturnErrorWhenLookupFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindAll", ctx).Return(nil, errDatabase)

	_, err := workorder.NewListWorkOrdersUseCase(workOrderRepo).Execute(ctx, "")

	assert.ErrorIs(t, err, errDatabase)
}

func TestGetExecutionMetrics_ShouldAverageDurationOverallAndByService(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	morning := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	workOrderRepo.On("FindByStatus", ctx, valueobject.WorkOrderStatusFinished).Return([]*entity.WorkOrder{
		finishedWorkOrder("order-1", morning, 60, "Troca de óleo", "Alinhamento"),
		finishedWorkOrder("order-2", morning, 120, "Troca de óleo", "Troca de óleo"),
	}, nil)

	metrics, err := workorder.NewGetExecutionMetricsUseCase(workOrderRepo).Execute(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, metrics.CompletedOrders)
	assert.Equal(t, 90.0, metrics.OverallAvgMinutes)
	assert.Equal(t, []workorder.ServiceMetric{
		{ServiceName: "Alinhamento", AvgMinutes: 60, CompletedOrders: 1},
		{ServiceName: "Troca de óleo", AvgMinutes: 90, CompletedOrders: 2},
	}, metrics.ByService)
}

func TestGetExecutionMetrics_ShouldReturnZeroedMetricsWithoutFinishedOrders(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByStatus", ctx, valueobject.WorkOrderStatusFinished).Return([]*entity.WorkOrder{}, nil)

	metrics, err := workorder.NewGetExecutionMetricsUseCase(workOrderRepo).Execute(ctx)

	require.NoError(t, err)
	assert.Zero(t, metrics.CompletedOrders)
	assert.Zero(t, metrics.OverallAvgMinutes)
	assert.Empty(t, metrics.ByService)
}

func TestGetExecutionMetrics_ShouldReturnErrorWhenLookupFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByStatus", ctx, valueobject.WorkOrderStatusFinished).Return(nil, errDatabase)

	_, err := workorder.NewGetExecutionMetricsUseCase(workOrderRepo).Execute(ctx)

	assert.ErrorIs(t, err, errDatabase)
}
