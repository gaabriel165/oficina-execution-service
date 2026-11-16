package api_test

import (
	"net/http"
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCatalog_ShouldRequireAuthentication(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.request(t, http.MethodGet, "/api/v1/parts", "", "")

	assertStatus(t, http.StatusUnauthorized, recorder)
}

func TestCatalog_ShouldForbidCustomers(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.request(t, http.MethodGet, "/api/v1/services", roleCustomer, "")

	assertStatus(t, http.StatusForbidden, recorder)
}

func TestCreatePart_ShouldReturnCreatedPart(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.partRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Part")).Return(nil)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/parts", `{"name":"Filtro de óleo","unit_price":29.9,"stock_quantity":10}`)

	assertStatus(t, http.StatusCreated, recorder)
	assert.Contains(t, recorder.Body.String(), `"name":"Filtro de óleo"`)
	assert.Contains(t, recorder.Body.String(), `"stock_quantity":10`)
}

func TestCreatePart_ShouldReturnBadRequestWithoutPrice(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/parts", `{"name":"Filtro de óleo"}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestListParts_ShouldReturnParts(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.partRepo.On("FindAll", mock.Anything).Return([]*entity.Part{oilFilter(t)}, nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/parts", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"unit_price":29.9`)
}

func TestGetPart_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.partRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrPartNotFound)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/parts/missing", "")

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestGetPart_ShouldReturnPart(t *testing.T) {
	environment := newTestEnvironment(t)
	part := oilFilter(t)
	environment.partRepo.On("FindByID", mock.Anything, part.ID()).Return(part, nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/parts/"+part.ID(), "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"id":"`+part.ID()+`"`)
}

func TestUpdatePart_ShouldReturnUpdatedPart(t *testing.T) {
	environment := newTestEnvironment(t)
	part := oilFilter(t)
	environment.partRepo.On("FindByID", mock.Anything, part.ID()).Return(part, nil)
	environment.partRepo.On("Update", mock.Anything, part).Return(nil)

	recorder := environment.asOperator(t, http.MethodPut, "/api/v1/parts/"+part.ID(), `{"name":"Filtro premium","unit_price":49.9}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"name":"Filtro premium"`)
}

func TestUpdatePart_ShouldReturnBadRequestForInvalidBody(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPut, "/api/v1/parts/part-1", `{"name":""}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestUpdatePart_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.partRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrPartNotFound)

	recorder := environment.asOperator(t, http.MethodPut, "/api/v1/parts/missing", `{"name":"Filtro premium","unit_price":49.9}`)

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestDeletePart_ShouldReturnNoContent(t *testing.T) {
	environment := newTestEnvironment(t)
	part := oilFilter(t)
	environment.partRepo.On("FindByID", mock.Anything, part.ID()).Return(part, nil).Maybe()
	environment.partRepo.On("Delete", mock.Anything, part.ID()).Return(nil)

	recorder := environment.asOperator(t, http.MethodDelete, "/api/v1/parts/"+part.ID(), "")

	assertStatus(t, http.StatusNoContent, recorder)
}

func TestUpdatePartStock_ShouldAddQuantityToStock(t *testing.T) {
	environment := newTestEnvironment(t)
	part := oilFilter(t)
	environment.partRepo.On("FindByID", mock.Anything, part.ID()).Return(part, nil)
	environment.partRepo.On("Update", mock.Anything, part).Return(nil)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/parts/"+part.ID()+"/stock", `{"quantity":5}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"stock_quantity":15`)
}

func TestUpdatePartStock_ShouldReturnBadRequestWithoutQuantity(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/parts/part-1/stock", `{}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestUpdatePartStock_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.partRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrPartNotFound)

	recorder := environment.asOperator(t, http.MethodPatch, "/api/v1/parts/missing/stock", `{"quantity":5}`)

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestCreateService_ShouldReturnCreatedService(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.serviceRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Service")).Return(nil)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/services", `{"name":"Troca de óleo","labor_price":120,"estimated_minutes":45}`)

	assertStatus(t, http.StatusCreated, recorder)
	assert.Contains(t, recorder.Body.String(), `"labor_price":120`)
}

func TestCreateService_ShouldReturnBadRequestWithoutEstimate(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPost, "/api/v1/services", `{"name":"Troca de óleo","labor_price":120}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestListServices_ShouldReturnServices(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.serviceRepo.On("FindAll", mock.Anything).Return([]*entity.Service{oilChange(t)}, nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/services", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"name":"Troca de óleo"`)
}

func TestGetService_ShouldReturnService(t *testing.T) {
	environment := newTestEnvironment(t)
	service := oilChange(t)
	environment.serviceRepo.On("FindByID", mock.Anything, service.ID()).Return(service, nil)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/services/"+service.ID(), "")

	assertStatus(t, http.StatusOK, recorder)
}

func TestGetService_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.serviceRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrServiceNotFound)

	recorder := environment.asOperator(t, http.MethodGet, "/api/v1/services/missing", "")

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestUpdateService_ShouldReturnUpdatedService(t *testing.T) {
	environment := newTestEnvironment(t)
	service := oilChange(t)
	environment.serviceRepo.On("FindByID", mock.Anything, service.ID()).Return(service, nil)
	environment.serviceRepo.On("Update", mock.Anything, service).Return(nil)

	recorder := environment.asOperator(t, http.MethodPut, "/api/v1/services/"+service.ID(), `{"name":"Troca de óleo sintético","labor_price":150,"estimated_minutes":50}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"labor_price":150`)
}

func TestUpdateService_ShouldReturnBadRequestForInvalidBody(t *testing.T) {
	environment := newTestEnvironment(t)

	recorder := environment.asOperator(t, http.MethodPut, "/api/v1/services/service-1", `{"name":"Troca"}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestUpdateService_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.serviceRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrServiceNotFound)

	recorder := environment.asOperator(t, http.MethodPut, "/api/v1/services/missing", `{"name":"Troca","labor_price":150,"estimated_minutes":50}`)

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestDeleteService_ShouldReturnNoContent(t *testing.T) {
	environment := newTestEnvironment(t)
	service := oilChange(t)
	environment.serviceRepo.On("FindByID", mock.Anything, service.ID()).Return(service, nil).Maybe()
	environment.serviceRepo.On("Delete", mock.Anything, service.ID()).Return(nil)

	recorder := environment.asOperator(t, http.MethodDelete, "/api/v1/services/"+service.ID(), "")

	assertStatus(t, http.StatusNoContent, recorder)
}

func TestDeleteService_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t)
	environment.serviceRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrServiceNotFound).Maybe()
	environment.serviceRepo.On("Delete", mock.Anything, "missing").Return(repository.ErrServiceNotFound).Maybe()

	recorder := environment.asOperator(t, http.MethodDelete, "/api/v1/services/missing", "")

	assertStatus(t, http.StatusNotFound, recorder)
}
