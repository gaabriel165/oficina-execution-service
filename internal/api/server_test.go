package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielcamargo/oficina-execution-service/configs"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newTestServer(readiness api.ReadinessCheck) *api.Server {
	gin.SetMode(gin.TestMode)
	return api.NewServer(&configs.Config{ServerPort: "0", ServiceName: "oficina-execution-service"}, readiness)
}

func perform(server *api.Server, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func TestHealth_ShouldReturnOK(t *testing.T) {
	recorder := perform(newTestServer(nil), "/health")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
}

func TestReady_ShouldReturnOKWhenDatabaseAnswers(t *testing.T) {
	recorder := perform(newTestServer(func(context.Context) error { return nil }), "/ready")

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"status":"ready"}`, recorder.Body.String())
}

func TestReady_ShouldReturnUnavailableWhenDatabaseFails(t *testing.T) {
	recorder := perform(newTestServer(func(context.Context) error { return errors.New("unreachable") }), "/ready")

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestServer_ShouldAnswerWithRequestID(t *testing.T) {
	recorder := perform(newTestServer(nil), "/health")

	assert.NotEmpty(t, recorder.Header().Get("X-Request-ID"))
}

func TestServer_ShouldRecoverFromPanicWithInternalError(t *testing.T) {
	recorder := perform(newTestServer(nil), "/ready")

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}
