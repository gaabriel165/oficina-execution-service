package integration_test

import (
	"context"
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/application/outbox"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	infrarepo "github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingPublisher struct {
	published []messaging.Message
}

func (p *capturingPublisher) Publish(_ context.Context, _ messaging.Destination, message messaging.Message) error {
	p.published = append(p.published, message)
	return nil
}

func (p *capturingPublisher) types() []string {
	types := make([]string, len(p.published))
	for index, message := range p.published {
		types[index] = message.Type
	}
	return types
}

type executionFixture struct {
	ctx           context.Context
	parts         *infrarepo.DynamoPartRepository
	services      *infrarepo.DynamoServiceRepository
	workOrders    *infrarepo.DynamoWorkOrderRepository
	relay         *outbox.Relay
	publisher     *capturingPublisher
	orderID       string
	sagaID        string
	part          *entity.Part
	service       *entity.Service
	partQuantity  int
	initialStock  int
	workOrderRepo repository.WorkOrderRepository
}

func newExecutionFixture(t *testing.T, initialStock, partQuantity int) *executionFixture {
	t.Helper()
	client, tables := newTestDynamoDB(t)
	ctx := context.Background()

	fixture := &executionFixture{
		ctx:          ctx,
		parts:        infrarepo.NewDynamoPartRepository(client, tables.Catalog),
		services:     infrarepo.NewDynamoServiceRepository(client, tables.Catalog),
		workOrders:   infrarepo.NewDynamoWorkOrderRepository(client, tables.WorkOrders, tables.Catalog, tables.Outbox),
		publisher:    &capturingPublisher{},
		orderID:      uuid.NewString(),
		sagaID:       uuid.NewString(),
		partQuantity: partQuantity,
		initialStock: initialStock,
	}
	fixture.workOrderRepo = fixture.workOrders
	fixture.relay = outbox.NewRelay(infrarepo.NewDynamoOutboxRepository(client, tables.Outbox), fixture.publisher)

	part, err := entity.NewPart("Oil filter", "Standard filter", 29.90, initialStock)
	require.NoError(t, err)
	require.NoError(t, fixture.parts.Create(ctx, part))
	fixture.part = part

	service, err := entity.NewService("Oil change", "Full synthetic", 120, 45)
	require.NoError(t, err)
	require.NoError(t, fixture.services.Create(ctx, service))
	fixture.service = service

	return fixture
}

func (f *executionFixture) diagnose(t *testing.T) {
	t.Helper()
	_, err := workorder.NewOpenWorkOrderUseCase(f.workOrderRepo).Execute(f.ctx, workorder.OpenWorkOrderInput{
		OrderID: f.orderID, SagaID: f.sagaID, CustomerID: uuid.NewString(), VehicleID: uuid.NewString(), Notes: "engine noise",
	})
	require.NoError(t, err)

	_, err = workorder.NewStartDiagnosisUseCase(f.workOrderRepo).Execute(f.ctx, f.orderID)
	require.NoError(t, err)

	_, err = workorder.NewAddServiceToWorkOrderUseCase(f.workOrderRepo, f.services).Execute(f.ctx, workorder.AddServiceToWorkOrderInput{
		WorkOrderID: f.orderID, ServiceID: f.service.ID(),
	})
	require.NoError(t, err)

	_, err = workorder.NewAddPartToWorkOrderUseCase(f.workOrderRepo, f.parts).Execute(f.ctx, workorder.AddPartToWorkOrderInput{
		WorkOrderID: f.orderID, PartID: f.part.ID(), Quantity: f.partQuantity,
	})
	require.NoError(t, err)

	_, err = workorder.NewCompleteDiagnosisUseCase(f.workOrderRepo).Execute(f.ctx, f.orderID)
	require.NoError(t, err)
}

func (f *executionFixture) relayAll(t *testing.T) {
	t.Helper()
	_, err := f.relay.RelayOnce(f.ctx)
	require.NoError(t, err)
}

func (f *executionFixture) stock(t *testing.T) int {
	t.Helper()
	part, err := f.parts.FindByID(f.ctx, f.part.ID())
	require.NoError(t, err)
	return part.StockQuantity()
}

func (f *executionFixture) status(t *testing.T) valueobject.WorkOrderStatus {
	t.Helper()
	workOrder, err := f.workOrders.FindByID(f.ctx, f.orderID)
	require.NoError(t, err)
	return workOrder.Status()
}

func TestWorkOrderFlow_ShouldReserveStockAndFinishExecution(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	fixture.diagnose(t)

	require.NoError(t, workorder.NewStartExecutionUseCase(fixture.workOrderRepo).Execute(fixture.ctx, fixture.orderID))
	_, err := workorder.NewFinishExecutionUseCase(fixture.workOrderRepo).Execute(fixture.ctx, fixture.orderID)
	require.NoError(t, err)
	fixture.relayAll(t)

	assert.Equal(t, 8, fixture.stock(t))
	assert.Equal(t, valueobject.WorkOrderStatusFinished, fixture.status(t))
	assert.ElementsMatch(t, []string{
		messaging.TypeDiagnosisStarted,
		messaging.TypeDiagnosisCompleted,
		messaging.TypeExecutionStarted,
		messaging.TypeExecutionFinished,
	}, fixture.publisher.types())
}

func TestWorkOrderFlow_ShouldCarryDiagnosedItemsInTheReply(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	fixture.diagnose(t)
	fixture.relayAll(t)

	var completed messaging.DiagnosisCompletedPayload
	for _, message := range fixture.publisher.published {
		if message.Type == messaging.TypeDiagnosisCompleted {
			require.NoError(t, message.DecodePayload(&completed))
			assert.Equal(t, fixture.sagaID, message.SagaID)
			assert.Equal(t, fixture.orderID, message.OrderID)
		}
	}

	require.Len(t, completed.Services, 1)
	require.Len(t, completed.Parts, 1)
	assert.Equal(t, 120.0, completed.Services[0].LaborPrice)
	assert.Equal(t, 2, completed.Parts[0].Quantity)
	assert.Equal(t, 29.90, completed.Parts[0].UnitPrice)
}

func TestWorkOrderFlow_ShouldRejectExecutionWhenStockIsInsufficient(t *testing.T) {
	fixture := newExecutionFixture(t, 1, 2)
	fixture.diagnose(t)

	require.NoError(t, workorder.NewStartExecutionUseCase(fixture.workOrderRepo).Execute(fixture.ctx, fixture.orderID))
	fixture.relayAll(t)

	assert.Equal(t, 1, fixture.stock(t))
	assert.Equal(t, valueobject.WorkOrderStatusDiagnosed, fixture.status(t))
	assert.Contains(t, fixture.publisher.types(), messaging.TypeExecutionRejected)
	assert.NotContains(t, fixture.publisher.types(), messaging.TypeExecutionStarted)
}

func TestWorkOrderFlow_ShouldNotReserveStockTwiceWhenCommandIsRedelivered(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	fixture.diagnose(t)
	startExecution := workorder.NewStartExecutionUseCase(fixture.workOrderRepo)

	require.NoError(t, startExecution.Execute(fixture.ctx, fixture.orderID))
	require.NoError(t, startExecution.Execute(fixture.ctx, fixture.orderID))

	assert.Equal(t, 8, fixture.stock(t))
}

func TestWorkOrderFlow_ShouldReturnStockWhenCancelledAfterReservation(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	fixture.diagnose(t)
	require.NoError(t, workorder.NewStartExecutionUseCase(fixture.workOrderRepo).Execute(fixture.ctx, fixture.orderID))

	err := workorder.NewCancelWorkOrderUseCase(fixture.workOrderRepo).Execute(fixture.ctx, workorder.CancelWorkOrderInput{
		OrderID: fixture.orderID, SagaID: fixture.sagaID, Reason: "payment refunded",
	})
	require.NoError(t, err)
	fixture.relayAll(t)

	assert.Equal(t, 10, fixture.stock(t))
	assert.Equal(t, valueobject.WorkOrderStatusCancelled, fixture.status(t))
	assert.Contains(t, fixture.publisher.types(), messaging.TypeWorkOrderCancelled)
}

func TestWorkOrderFlow_ShouldCancelWithoutTouchingStockBeforeReservation(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	fixture.diagnose(t)

	err := workorder.NewCancelWorkOrderUseCase(fixture.workOrderRepo).Execute(fixture.ctx, workorder.CancelWorkOrderInput{
		OrderID: fixture.orderID, SagaID: fixture.sagaID, Reason: "budget rejected",
	})
	require.NoError(t, err)

	assert.Equal(t, 10, fixture.stock(t))
	assert.Equal(t, valueobject.WorkOrderStatusCancelled, fixture.status(t))
}

func TestWorkOrderFlow_ShouldAnswerCancellationOfUnknownWorkOrder(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)

	err := workorder.NewCancelWorkOrderUseCase(fixture.workOrderRepo).Execute(fixture.ctx, workorder.CancelWorkOrderInput{
		OrderID: fixture.orderID, SagaID: fixture.sagaID,
	})
	require.NoError(t, err)
	fixture.relayAll(t)

	assert.Equal(t, []string{messaging.TypeWorkOrderCancelled}, fixture.publisher.types())
}

func TestWorkOrderRepository_ShouldRejectDuplicateCreation(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	workOrder, err := entity.NewWorkOrder(fixture.orderID, fixture.sagaID, uuid.NewString(), uuid.NewString(), "")
	require.NoError(t, err)
	require.NoError(t, fixture.workOrders.Create(fixture.ctx, workOrder))

	err = fixture.workOrders.Create(fixture.ctx, workOrder)

	assert.Error(t, err)
}

func TestWorkOrderRepository_ShouldListByStatus(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	fixture.diagnose(t)

	diagnosed, err := fixture.workOrders.FindByStatus(fixture.ctx, valueobject.WorkOrderStatusDiagnosed)
	require.NoError(t, err)
	finished, err := fixture.workOrders.FindByStatus(fixture.ctx, valueobject.WorkOrderStatusFinished)
	require.NoError(t, err)
	all, err := fixture.workOrders.FindAll(fixture.ctx)
	require.NoError(t, err)

	assert.Len(t, diagnosed, 1)
	assert.Empty(t, finished)
	assert.Len(t, all, 1)
}

func TestCatalogRepositories_ShouldKeepPartsAndServicesApart(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)

	parts, err := fixture.parts.FindAll(fixture.ctx)
	require.NoError(t, err)
	services, err := fixture.services.FindAll(fixture.ctx)
	require.NoError(t, err)
	_, partAsService := fixture.services.FindByID(fixture.ctx, fixture.part.ID())
	_, missingPart := fixture.parts.FindByID(fixture.ctx, uuid.NewString())

	assert.Len(t, parts, 1)
	assert.Len(t, services, 1)
	assert.ErrorIs(t, partAsService, repository.ErrServiceNotFound)
	assert.ErrorIs(t, missingPart, repository.ErrPartNotFound)
}

func TestCatalogRepositories_ShouldUpdateAndDelete(t *testing.T) {
	fixture := newExecutionFixture(t, 10, 2)
	require.NoError(t, fixture.part.Update("Premium oil filter", "Long life", 49.90))
	require.NoError(t, fixture.parts.Update(fixture.ctx, fixture.part))
	require.NoError(t, fixture.service.Update("Oil change", "Mineral", 90, 30))
	require.NoError(t, fixture.services.Update(fixture.ctx, fixture.service))

	updatedPart, err := fixture.parts.FindByID(fixture.ctx, fixture.part.ID())
	require.NoError(t, err)
	updatedService, err := fixture.services.FindByID(fixture.ctx, fixture.service.ID())
	require.NoError(t, err)
	require.NoError(t, fixture.parts.Delete(fixture.ctx, fixture.part.ID()))
	require.NoError(t, fixture.services.Delete(fixture.ctx, fixture.service.ID()))
	_, deletedPart := fixture.parts.FindByID(fixture.ctx, fixture.part.ID())
	_, deletedService := fixture.services.FindByID(fixture.ctx, fixture.service.ID())

	assert.Equal(t, "Premium oil filter", updatedPart.Name())
	assert.Equal(t, 90.0, updatedService.LaborPrice())
	assert.ErrorIs(t, deletedPart, repository.ErrPartNotFound)
	assert.ErrorIs(t, deletedService, repository.ErrServiceNotFound)
}
