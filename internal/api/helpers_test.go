package api_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/configs"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret = "test-secret"
	roleOperator  = "operator"
	roleCustomer  = "customer"
)

type testEnvironment struct {
	server        *api.Server
	partRepo      *mocks.MockPartRepository
	serviceRepo   *mocks.MockServiceRepository
	workOrderRepo *mocks.MockWorkOrderRepository
}

func newTestEnvironment(t *testing.T) *testEnvironment {
	t.Helper()
	gin.SetMode(gin.TestMode)
	environment := &testEnvironment{
		partRepo:      mocks.NewMockPartRepository(t),
		serviceRepo:   mocks.NewMockServiceRepository(t),
		workOrderRepo: mocks.NewMockWorkOrderRepository(t),
	}
	environment.server = api.NewServer(
		&configs.Config{ServerPort: "0", ServiceName: "oficina-execution-service", JWTSecret: testJWTSecret},
		api.Dependencies{
			PartRepo:      environment.partRepo,
			ServiceRepo:   environment.serviceRepo,
			WorkOrderRepo: environment.workOrderRepo,
		},
	)
	return environment
}

func (e *testEnvironment) request(t *testing.T, method, path, role, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if role != "" {
		request.Header.Set("Authorization", "Bearer "+tokenFor(t, role))
	}
	recorder := httptest.NewRecorder()
	e.server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func (e *testEnvironment) asOperator(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.request(t, method, path, roleOperator, body)
}

func tokenFor(t *testing.T, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-1",
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

func assertStatus(t *testing.T, expected int, recorder *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, expected, recorder.Code, recorder.Body.String())
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

func awaitingWorkOrder(t *testing.T) *entity.WorkOrder {
	t.Helper()
	workOrder, err := entity.NewWorkOrder("order-1", "saga-1", "customer-1", "vehicle-1", "engine noise")
	require.NoError(t, err)
	return workOrder
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
		return len(outgoing) == 1 && outgoing[0].Message.Type == messageType
	})
}
