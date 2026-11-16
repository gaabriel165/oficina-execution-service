package response

import (
	"net/http"

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

func statusFromError(_ error) int {
	return http.StatusInternalServerError
}
