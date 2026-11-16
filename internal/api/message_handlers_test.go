package api_test

import (
	"context"
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type handlerRegistry map[string]messaging.Handler

func (r handlerRegistry) Register(messageType string, handler messaging.Handler) {
	r[messageType] = handler
}

func registeredHandlers(environment *testEnvironment) handlerRegistry {
	registry := handlerRegistry{}
	environment.server.RegisterMessageHandlers(registry)
	return registry
}

func command(t *testing.T, messageType string, payload any) messaging.Message {
	t.Helper()
	message, err := messaging.NewMessage(messageType, "saga-1", "order-1", payload)
	require.NoError(t, err)
	return message
}

func TestMessageHandlers_ShouldRegisterEveryExecutionCommand(t *testing.T) {
	registry := registeredHandlers(newTestEnvironment(t))

	assert.Len(t, registry, 3)
	assert.Contains(t, registry, messaging.TypeRequestDiagnosis)
	assert.Contains(t, registry, messaging.TypeStartExecution)
	assert.Contains(t, registry, messaging.TypeCancelWorkOrder)
}

func TestRequestDiagnosisCommand_ShouldOpenWorkOrderForServiceOrder(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(nil, repository.ErrWorkOrderNotFound)
	environment.workOrderRepo.On("Create", mock.Anything, mock.MatchedBy(func(workOrder *entity.WorkOrder) bool {
		return workOrder.ID() == "order-1" && workOrder.SagaID() == "saga-1" && workOrder.VehicleID() == "vehicle-1"
	})).Return(nil)
	message := command(t, messaging.TypeRequestDiagnosis, messaging.RequestDiagnosisPayload{
		CustomerID: "customer-1", VehicleID: "vehicle-1", Notes: "engine noise",
	})

	err := registeredHandlers(environment)[messaging.TypeRequestDiagnosis](context.Background(), message)

	assert.NoError(t, err)
}

func TestRequestDiagnosisCommand_ShouldRejectInvalidPayload(t *testing.T) {
	environment := newTestEnvironment(t)
	message := command(t, messaging.TypeRequestDiagnosis, nil)
	message.Payload = []byte(`{"customer_id":1}`)

	err := registeredHandlers(environment)[messaging.TypeRequestDiagnosis](context.Background(), message)

	assert.ErrorIs(t, err, messaging.ErrMessagePayloadInvalid)
}

func TestStartExecutionCommand_ShouldReserveStock(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := diagnosedWorkOrder(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.workOrderRepo.On("UpdateReservingStock", mock.Anything, workOrder, replyOfType(messaging.TypeExecutionStarted)).Return(nil)

	err := registeredHandlers(environment)[messaging.TypeStartExecution](context.Background(), command(t, messaging.TypeStartExecution, nil))

	require.NoError(t, err)
	assert.True(t, workOrder.StockReserved())
}

func TestCancelWorkOrderCommand_ShouldCancelWithGivenReason(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := diagnosedWorkOrder(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.workOrderRepo.On("Update", mock.Anything, workOrder, replyOfType(messaging.TypeWorkOrderCancelled)).Return(nil)
	message := command(t, messaging.TypeCancelWorkOrder, messaging.ReasonPayload{Reason: "budget rejected"})

	err := registeredHandlers(environment)[messaging.TypeCancelWorkOrder](context.Background(), message)

	require.NoError(t, err)
	assert.Equal(t, "budget rejected", workOrder.CancellationReason())
}

func TestCancelWorkOrderCommand_ShouldRejectInvalidPayload(t *testing.T) {
	environment := newTestEnvironment(t)
	message := command(t, messaging.TypeCancelWorkOrder, nil)
	message.Payload = []byte(`{"reason":1}`)

	err := registeredHandlers(environment)[messaging.TypeCancelWorkOrder](context.Background(), message)

	assert.ErrorIs(t, err, messaging.ErrMessagePayloadInvalid)
}
