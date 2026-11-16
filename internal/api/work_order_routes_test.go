package api_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListWorkOrders_ShouldReturnWorkOrders(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindAll", mock.Anything).Return([]*entity.WorkOrder{awaitingWorkOrder(t)}, nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"awaiting_diagnosis"`)
}

func TestListWorkOrders_ShouldFilterByStatus(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByStatus", mock.Anything, valueobject.WorkOrderStatusInExecution).Return([]*entity.WorkOrder{workOrderInExecution(t)}, nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders?status=in_execution", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"in_execution"`)
}

func TestListWorkOrders_ShouldReturnUnprocessableForUnknownStatus(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders?status=unknown", "")

	assertStatus(t, http.StatusUnprocessableEntity, recorder)
}

func TestGetWorkOrder_ShouldReturnWorkOrderWithItems(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(diagnosedWorkOrder(t), nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders/order-1", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"name":"Troca de óleo"`)
	assert.Contains(t, recorder.Body.String(), `"quantity":2`)
}

func TestGetWorkOrder_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrWorkOrderNotFound)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders/missing", "")

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestGetWorkOrder_ShouldHideInternalErrorDetails(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(nil, errors.New("connection refused"))

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders/order-1", "")

	assertStatus(t, http.StatusInternalServerError, recorder)
	assert.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
}

func TestGetWorkOrderMetrics_ShouldReturnAverages(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByStatus", mock.Anything, valueobject.WorkOrderStatusFinished).Return([]*entity.WorkOrder{}, nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders/metrics", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"completed_orders":0`)
}

func TestGetWorkOrderMetrics_ShouldReturnInternalErrorWhenLookupFails(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByStatus", mock.Anything, valueobject.WorkOrderStatusFinished).Return(nil, errors.New("connection refused"))

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/work-orders/metrics", "")

	assertStatus(t, http.StatusInternalServerError, recorder)
}

func TestStartDiagnosis_ShouldReturnWorkOrderInDiagnosis(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := awaitingWorkOrder(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.workOrderRepo.On("Update", mock.Anything, workOrder, replyOfType(messaging.TypeDiagnosisStarted)).Return(nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/start-diagnosis", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"in_diagnosis"`)
}

func TestStartDiagnosis_ShouldReturnUnprocessableWhenAlreadyStarted(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrderInDiagnosis(t), nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/start-diagnosis", "")

	assertStatus(t, http.StatusUnprocessableEntity, recorder)
}

func TestAddWorkOrderService_ShouldReturnWorkOrderWithService(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := workOrderInDiagnosis(t)
	service := oilChange(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.serviceRepo.On("FindByID", mock.Anything, service.ID()).Return(service, nil)
	environment.workOrderRepo.On("Update", mock.Anything, workOrder).Return(nil)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/work-orders/order-1/services", `{"service_id":"`+service.ID()+`"}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"service_id":"`+service.ID()+`"`)
}

func TestAddWorkOrderService_ShouldReturnBadRequestWithoutServiceID(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/work-orders/order-1/services", `{}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestAddWorkOrderService_ShouldReturnNotFoundForUnknownService(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrderInDiagnosis(t), nil)
	environment.serviceRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrServiceNotFound)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/work-orders/order-1/services", `{"service_id":"missing"}`)

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestAddWorkOrderPart_ShouldReturnWorkOrderWithPart(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := workOrderInDiagnosis(t)
	part := oilFilter(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.partRepo.On("FindByID", mock.Anything, part.ID()).Return(part, nil)
	environment.workOrderRepo.On("Update", mock.Anything, workOrder).Return(nil)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/work-orders/order-1/parts", `{"part_id":"`+part.ID()+`","quantity":2}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"quantity":2`)
}

func TestAddWorkOrderPart_ShouldReturnBadRequestForZeroQuantity(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/work-orders/order-1/parts", `{"part_id":"part-1","quantity":0}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestAddWorkOrderPart_ShouldReturnUnprocessableOutsideDiagnosis(t *testing.T) {
	environment := newTestEnvironment(t)
	part := oilFilter(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(awaitingWorkOrder(t), nil)
	environment.partRepo.On("FindByID", mock.Anything, part.ID()).Return(part, nil)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/work-orders/order-1/parts", `{"part_id":"`+part.ID()+`","quantity":1}`)

	assertStatus(t, http.StatusUnprocessableEntity, recorder)
}

func TestCompleteDiagnosis_ShouldReturnDiagnosedWorkOrder(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := workOrderInDiagnosis(t)
	assert.NoError(t, workOrder.AddService(oilChange(t)))
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.workOrderRepo.On("Update", mock.Anything, workOrder, replyOfType(messaging.TypeDiagnosisCompleted)).Return(nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/complete-diagnosis", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"diagnosed"`)
}

func TestCompleteDiagnosis_ShouldReturnUnprocessableWithoutItems(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrderInDiagnosis(t), nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/complete-diagnosis", "")

	assertStatus(t, http.StatusUnprocessableEntity, recorder)
}

func TestFailDiagnosis_ShouldReturnCancelledWorkOrder(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := workOrderInDiagnosis(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.workOrderRepo.On("Update", mock.Anything, workOrder, replyOfType(messaging.TypeDiagnosisFailed)).Return(nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/fail-diagnosis", `{"reason":"vehicle not delivered"}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"cancelled"`)
}

func TestFailDiagnosis_ShouldReturnBadRequestWithoutReason(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/fail-diagnosis", `{}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestFailDiagnosis_ShouldReturnUnprocessableAfterDiagnosis(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(diagnosedWorkOrder(t), nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/fail-diagnosis", `{"reason":"too late"}`)

	assertStatus(t, http.StatusUnprocessableEntity, recorder)
}

func TestFinishExecution_ShouldReturnFinishedWorkOrder(t *testing.T) {
	environment := newTestEnvironment(t)
	workOrder := workOrderInExecution(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(workOrder, nil)
	environment.workOrderRepo.On("Update", mock.Anything, workOrder, replyOfType(messaging.TypeExecutionFinished)).Return(nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/finish-execution", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"finished"`)
}

func TestFinishExecution_ShouldReturnUnprocessableWhenNotInExecution(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.workOrderRepo.On("FindByID", mock.Anything, "order-1").Return(diagnosedWorkOrder(t), nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/work-orders/order-1/finish-execution", "")

	assertStatus(t, http.StatusUnprocessableEntity, recorder)
}
