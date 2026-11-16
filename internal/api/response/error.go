package response

import (
	"errors"
	"net/http"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func Error(c *gin.Context, err error) {
	status := statusFromError(err)
	if status == http.StatusInternalServerError {
		c.JSON(status, ErrorResponse{Error: "internal server error"})
		return
	}
	c.JSON(status, ErrorResponse{Error: err.Error()})
}

func statusFromError(err error) int {
	notFound := []error{
		repository.ErrPartNotFound,
		repository.ErrServiceNotFound,
		repository.ErrWorkOrderNotFound,
	}
	for _, e := range notFound {
		if errors.Is(err, e) {
			return http.StatusNotFound
		}
	}

	unprocessable := []error{
		entity.ErrPartNameRequired,
		entity.ErrPartInvalidPrice,
		entity.ErrPartInvalidStock,
		entity.ErrPartInsufficientStock,
		entity.ErrServiceNameRequired,
		entity.ErrServiceInvalidLaborPrice,
		entity.ErrServiceInvalidEstimatedMin,
		entity.ErrWorkOrderInvalidTransition,
		entity.ErrWorkOrderNoItems,
		entity.ErrWorkOrderInvalidQuantity,
		entity.ErrWorkOrderNotInDiagnosis,
		entity.ErrWorkOrderReasonRequired,
		valueobject.ErrInvalidWorkOrderStatus,
	}
	for _, e := range unprocessable {
		if errors.Is(err, e) {
			return http.StatusUnprocessableEntity
		}
	}

	return http.StatusInternalServerError
}
