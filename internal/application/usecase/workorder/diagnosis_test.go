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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStartDiagnosis_ShouldStartAndReplyToSaga(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := awaitingWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeDiagnosisStarted)).Return(nil)

	result, err := workorder.NewStartDiagnosisUseCase(workOrderRepo).Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusInDiagnosis, result.Status())
}

func TestStartDiagnosis_ShouldReturnErrorWhenNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewStartDiagnosisUseCase(workOrderRepo).Execute(ctx, "missing")

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestStartDiagnosis_ShouldReturnErrorWhenAlreadyStarted(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(workOrderInDiagnosis(t), nil)

	_, err := workorder.NewStartDiagnosisUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
}

func TestStartDiagnosis_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := awaitingWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeDiagnosisStarted)).Return(errDatabase)

	_, err := workorder.NewStartDiagnosisUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.ErrorIs(t, err, errDatabase)
}

func TestAddServiceToWorkOrder_ShouldAddCatalogService(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	serviceRepo := mocks.NewMockServiceRepository(t)
	existing := workOrderInDiagnosis(t)
	service := oilChange(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	serviceRepo.On("FindByID", ctx, service.ID()).Return(service, nil)
	workOrderRepo.On("Update", ctx, existing).Return(nil)

	result, err := workorder.NewAddServiceToWorkOrderUseCase(workOrderRepo, serviceRepo).Execute(ctx, workorder.AddServiceToWorkOrderInput{
		WorkOrderID: "order-1", ServiceID: service.ID(),
	})

	require.NoError(t, err)
	assert.Len(t, result.Services(), 1)
}

func TestAddServiceToWorkOrder_ShouldReturnErrorWhenWorkOrderNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	serviceRepo := mocks.NewMockServiceRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewAddServiceToWorkOrderUseCase(workOrderRepo, serviceRepo).Execute(ctx, workorder.AddServiceToWorkOrderInput{WorkOrderID: "missing"})

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestAddServiceToWorkOrder_ShouldReturnErrorWhenServiceNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	serviceRepo := mocks.NewMockServiceRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(workOrderInDiagnosis(t), nil)
	serviceRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrServiceNotFound)

	_, err := workorder.NewAddServiceToWorkOrderUseCase(workOrderRepo, serviceRepo).Execute(ctx, workorder.AddServiceToWorkOrderInput{
		WorkOrderID: "order-1", ServiceID: "missing",
	})

	assert.ErrorIs(t, err, repository.ErrServiceNotFound)
}

func TestAddServiceToWorkOrder_ShouldReturnErrorOutsideDiagnosis(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	serviceRepo := mocks.NewMockServiceRepository(t)
	service := oilChange(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(awaitingWorkOrder(t), nil)
	serviceRepo.On("FindByID", ctx, service.ID()).Return(service, nil)

	_, err := workorder.NewAddServiceToWorkOrderUseCase(workOrderRepo, serviceRepo).Execute(ctx, workorder.AddServiceToWorkOrderInput{
		WorkOrderID: "order-1", ServiceID: service.ID(),
	})

	assert.ErrorIs(t, err, entity.ErrWorkOrderNotInDiagnosis)
}

func TestAddServiceToWorkOrder_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	serviceRepo := mocks.NewMockServiceRepository(t)
	existing := workOrderInDiagnosis(t)
	service := oilChange(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	serviceRepo.On("FindByID", ctx, service.ID()).Return(service, nil)
	workOrderRepo.On("Update", ctx, existing).Return(errDatabase)

	_, err := workorder.NewAddServiceToWorkOrderUseCase(workOrderRepo, serviceRepo).Execute(ctx, workorder.AddServiceToWorkOrderInput{
		WorkOrderID: "order-1", ServiceID: service.ID(),
	})

	assert.ErrorIs(t, err, errDatabase)
}

func TestAddPartToWorkOrder_ShouldAddCatalogPart(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	partRepo := mocks.NewMockPartRepository(t)
	existing := workOrderInDiagnosis(t)
	part := oilFilter(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	partRepo.On("FindByID", ctx, part.ID()).Return(part, nil)
	workOrderRepo.On("Update", ctx, existing).Return(nil)

	result, err := workorder.NewAddPartToWorkOrderUseCase(workOrderRepo, partRepo).Execute(ctx, workorder.AddPartToWorkOrderInput{
		WorkOrderID: "order-1", PartID: part.ID(), Quantity: 2,
	})

	require.NoError(t, err)
	require.Len(t, result.Parts(), 1)
	assert.Equal(t, 2, result.Parts()[0].Quantity())
}

func TestAddPartToWorkOrder_ShouldReturnErrorWhenWorkOrderNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	partRepo := mocks.NewMockPartRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewAddPartToWorkOrderUseCase(workOrderRepo, partRepo).Execute(ctx, workorder.AddPartToWorkOrderInput{WorkOrderID: "missing"})

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestAddPartToWorkOrder_ShouldReturnErrorWhenPartNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	partRepo := mocks.NewMockPartRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(workOrderInDiagnosis(t), nil)
	partRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrPartNotFound)

	_, err := workorder.NewAddPartToWorkOrderUseCase(workOrderRepo, partRepo).Execute(ctx, workorder.AddPartToWorkOrderInput{
		WorkOrderID: "order-1", PartID: "missing", Quantity: 1,
	})

	assert.ErrorIs(t, err, repository.ErrPartNotFound)
}

func TestAddPartToWorkOrder_ShouldReturnErrorWhenQuantityIsInvalid(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	partRepo := mocks.NewMockPartRepository(t)
	part := oilFilter(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(workOrderInDiagnosis(t), nil)
	partRepo.On("FindByID", ctx, part.ID()).Return(part, nil)

	_, err := workorder.NewAddPartToWorkOrderUseCase(workOrderRepo, partRepo).Execute(ctx, workorder.AddPartToWorkOrderInput{
		WorkOrderID: "order-1", PartID: part.ID(),
	})

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidQuantity)
}

func TestAddPartToWorkOrder_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	partRepo := mocks.NewMockPartRepository(t)
	existing := workOrderInDiagnosis(t)
	part := oilFilter(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	partRepo.On("FindByID", ctx, part.ID()).Return(part, nil)
	workOrderRepo.On("Update", ctx, existing).Return(errDatabase)

	_, err := workorder.NewAddPartToWorkOrderUseCase(workOrderRepo, partRepo).Execute(ctx, workorder.AddPartToWorkOrderInput{
		WorkOrderID: "order-1", PartID: part.ID(), Quantity: 1,
	})

	assert.ErrorIs(t, err, errDatabase)
}

func completedDiagnosisWithLines(t *testing.T) any {
	return mock.MatchedBy(func(outgoing []messaging.OutgoingMessage) bool {
		if len(outgoing) != 1 || outgoing[0].Message.Type != messaging.TypeDiagnosisCompleted {
			return false
		}
		var payload messaging.DiagnosisCompletedPayload
		if err := outgoing[0].Message.DecodePayload(&payload); err != nil {
			return false
		}
		return payload.WorkOrderID == "order-1" &&
			len(payload.Services) == 1 && payload.Services[0].LaborPrice == 120 &&
			len(payload.Parts) == 1 && payload.Parts[0].Quantity == 2 && payload.Parts[0].UnitPrice == 29.9
	})
}

func TestCompleteDiagnosis_ShouldReplyWithFrozenServicesAndParts(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInDiagnosis(t)
	require.NoError(t, existing.AddService(oilChange(t)))
	require.NoError(t, existing.AddPart(oilFilter(t), 2))
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, completedDiagnosisWithLines(t)).Return(nil)

	result, err := workorder.NewCompleteDiagnosisUseCase(workOrderRepo).Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.Equal(t, valueobject.WorkOrderStatusDiagnosed, result.Status())
}

func TestCompleteDiagnosis_ShouldReturnErrorWithoutItems(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(workOrderInDiagnosis(t), nil)

	_, err := workorder.NewCompleteDiagnosisUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.ErrorIs(t, err, entity.ErrWorkOrderNoItems)
}

func TestCompleteDiagnosis_ShouldReturnErrorWhenNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewCompleteDiagnosisUseCase(workOrderRepo).Execute(ctx, "missing")

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestCompleteDiagnosis_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInDiagnosis(t)
	require.NoError(t, existing.AddService(oilChange(t)))
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeDiagnosisCompleted)).Return(errDatabase)

	_, err := workorder.NewCompleteDiagnosisUseCase(workOrderRepo).Execute(ctx, "order-1")

	assert.ErrorIs(t, err, errDatabase)
}

func TestFailDiagnosis_ShouldCancelAndReplyToSaga(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := workOrderInDiagnosis(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeDiagnosisFailed)).Return(nil)

	result, err := workorder.NewFailDiagnosisUseCase(workOrderRepo).Execute(ctx, workorder.FailDiagnosisInput{
		WorkOrderID: "order-1", Reason: "vehicle not delivered",
	})

	require.NoError(t, err)
	assert.True(t, result.IsCancelled())
	assert.Equal(t, "vehicle not delivered", result.CancellationReason())
}

func TestFailDiagnosis_ShouldReturnErrorAfterDiagnosisIsComplete(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(diagnosedWorkOrder(t), nil)

	_, err := workorder.NewFailDiagnosisUseCase(workOrderRepo).Execute(ctx, workorder.FailDiagnosisInput{
		WorkOrderID: "order-1", Reason: "too late",
	})

	assert.ErrorIs(t, err, entity.ErrWorkOrderInvalidTransition)
}

func TestFailDiagnosis_ShouldRequireReason(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(awaitingWorkOrder(t), nil)

	_, err := workorder.NewFailDiagnosisUseCase(workOrderRepo).Execute(ctx, workorder.FailDiagnosisInput{WorkOrderID: "order-1"})

	assert.ErrorIs(t, err, entity.ErrWorkOrderReasonRequired)
}

func TestFailDiagnosis_ShouldReturnErrorWhenNotFound(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	workOrderRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	_, err := workorder.NewFailDiagnosisUseCase(workOrderRepo).Execute(ctx, workorder.FailDiagnosisInput{WorkOrderID: "missing", Reason: "x"})

	assert.ErrorIs(t, err, repository.ErrWorkOrderNotFound)
}

func TestFailDiagnosis_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	workOrderRepo := mocks.NewMockWorkOrderRepository(t)
	existing := awaitingWorkOrder(t)
	workOrderRepo.On("FindByID", ctx, "order-1").Return(existing, nil)
	workOrderRepo.On("Update", ctx, existing, replyOfType(messaging.TypeDiagnosisFailed)).Return(errDatabase)

	_, err := workorder.NewFailDiagnosisUseCase(workOrderRepo).Execute(ctx, workorder.FailDiagnosisInput{WorkOrderID: "order-1", Reason: "x"})

	assert.ErrorIs(t, err, errDatabase)
}
