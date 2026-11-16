package handler

import (
	"log/slog"
	"net/http"

	"github.com/gabrielcamargo/oficina-execution-service/internal/api/dto"
	"github.com/gabrielcamargo/oficina-execution-service/internal/api/response"
	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/infrastructure/observability"
	"github.com/gin-gonic/gin"
)

const (
	eventWorkOrderStatusChanged = "work_order.status_changed"
	eventWorkOrderFailed        = "work_order.failed"
)

type WorkOrderHandler struct {
	startDiagnosisUC    *workorder.StartDiagnosisUseCase
	addServiceUC        *workorder.AddServiceToWorkOrderUseCase
	addPartUC           *workorder.AddPartToWorkOrderUseCase
	completeDiagnosisUC *workorder.CompleteDiagnosisUseCase
	failDiagnosisUC     *workorder.FailDiagnosisUseCase
	finishExecutionUC   *workorder.FinishExecutionUseCase
	getUC               *workorder.GetWorkOrderUseCase
	listUC              *workorder.ListWorkOrdersUseCase
	metricsUC           *workorder.GetExecutionMetricsUseCase
}

func NewWorkOrderHandler(
	startDiagnosisUC *workorder.StartDiagnosisUseCase,
	addServiceUC *workorder.AddServiceToWorkOrderUseCase,
	addPartUC *workorder.AddPartToWorkOrderUseCase,
	completeDiagnosisUC *workorder.CompleteDiagnosisUseCase,
	failDiagnosisUC *workorder.FailDiagnosisUseCase,
	finishExecutionUC *workorder.FinishExecutionUseCase,
	getUC *workorder.GetWorkOrderUseCase,
	listUC *workorder.ListWorkOrdersUseCase,
	metricsUC *workorder.GetExecutionMetricsUseCase,
) *WorkOrderHandler {
	return &WorkOrderHandler{
		startDiagnosisUC:    startDiagnosisUC,
		addServiceUC:        addServiceUC,
		addPartUC:           addPartUC,
		completeDiagnosisUC: completeDiagnosisUC,
		failDiagnosisUC:     failDiagnosisUC,
		finishExecutionUC:   finishExecutionUC,
		getUC:               getUC,
		listUC:              listUC,
		metricsUC:           metricsUC,
	}
}

func (h *WorkOrderHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/work-orders/metrics", h.GetMetrics)
	router.GET("/work-orders", h.List)
	router.GET("/work-orders/:id", h.Get)
	router.PATCH("/work-orders/:id/start-diagnosis", h.StartDiagnosis)
	router.POST("/work-orders/:id/services", h.AddService)
	router.POST("/work-orders/:id/parts", h.AddPart)
	router.PATCH("/work-orders/:id/complete-diagnosis", h.CompleteDiagnosis)
	router.PATCH("/work-orders/:id/fail-diagnosis", h.FailDiagnosis)
	router.PATCH("/work-orders/:id/finish-execution", h.FinishExecution)
}

// GetMetrics godoc
// @Summary      Average execution time, overall and per service
// @Tags         work-orders
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  dto.ExecutionMetricsResponse
// @Router       /work-orders/metrics [get]
func (h *WorkOrderHandler) GetMetrics(c *gin.Context) {
	metrics, err := h.metricsUC.Execute(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToExecutionMetricsResponse(metrics))
}

// List godoc
// @Summary      List work orders, oldest first, optionally filtered by status (the diagnosis and execution queues)
// @Tags         work-orders
// @Security     BearerAuth
// @Produce      json
// @Param        status  query     string  false  "awaiting_diagnosis, in_diagnosis, diagnosed, in_execution, finished or cancelled"
// @Success      200     {array}   dto.WorkOrderResponse
// @Failure      422     {object}  response.ErrorResponse
// @Router       /work-orders [get]
func (h *WorkOrderHandler) List(c *gin.Context) {
	workOrders, err := h.listUC.Execute(c.Request.Context(), c.Query("status"))
	if err != nil {
		response.Error(c, err)
		return
	}

	responses := make([]dto.WorkOrderResponse, 0, len(workOrders))
	for _, workOrder := range workOrders {
		responses = append(responses, dto.ToWorkOrderResponse(workOrder))
	}
	c.JSON(http.StatusOK, responses)
}

// Get godoc
// @Summary      Get a work order by the service order ID
// @Tags         work-orders
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Service order ID"
// @Success      200  {object}  dto.WorkOrderResponse
// @Failure      404  {object}  response.ErrorResponse
// @Router       /work-orders/{id} [get]
func (h *WorkOrderHandler) Get(c *gin.Context) {
	workOrder, err := h.getUC.Execute(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToWorkOrderResponse(workOrder))
}

// StartDiagnosis godoc
// @Summary      Take the work order from the diagnosis queue and start the diagnosis
// @Tags         work-orders
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Service order ID"
// @Success      200  {object}  dto.WorkOrderResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      422  {object}  response.ErrorResponse
// @Router       /work-orders/{id}/start-diagnosis [patch]
func (h *WorkOrderHandler) StartDiagnosis(c *gin.Context) {
	workOrder, err := h.startDiagnosisUC.Execute(c.Request.Context(), c.Param("id"))
	respondWorkOrderTransition(c, "start_diagnosis", workOrder, err)
}

// AddService godoc
// @Summary      Add a catalog service to the work order during diagnosis
// @Tags         work-orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                          true  "Service order ID"
// @Param        body  body      dto.AddWorkOrderServiceRequest  true  "Service to add"
// @Success      200   {object}  dto.WorkOrderResponse
// @Failure      400   {object}  response.ErrorResponse
// @Failure      404   {object}  response.ErrorResponse
// @Failure      422   {object}  response.ErrorResponse
// @Router       /work-orders/{id}/services [post]
func (h *WorkOrderHandler) AddService(c *gin.Context) {
	var req dto.AddWorkOrderServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workOrder, err := h.addServiceUC.Execute(c.Request.Context(), workorder.AddServiceToWorkOrderInput{
		WorkOrderID: c.Param("id"),
		ServiceID:   req.ServiceID,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToWorkOrderResponse(workOrder))
}

// AddPart godoc
// @Summary      Add a catalog part to the work order during diagnosis
// @Tags         work-orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                       true  "Service order ID"
// @Param        body  body      dto.AddWorkOrderPartRequest  true  "Part and quantity"
// @Success      200   {object}  dto.WorkOrderResponse
// @Failure      400   {object}  response.ErrorResponse
// @Failure      404   {object}  response.ErrorResponse
// @Failure      422   {object}  response.ErrorResponse
// @Router       /work-orders/{id}/parts [post]
func (h *WorkOrderHandler) AddPart(c *gin.Context) {
	var req dto.AddWorkOrderPartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workOrder, err := h.addPartUC.Execute(c.Request.Context(), workorder.AddPartToWorkOrderInput{
		WorkOrderID: c.Param("id"),
		PartID:      req.PartID,
		Quantity:    req.Quantity,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToWorkOrderResponse(workOrder))
}

// CompleteDiagnosis godoc
// @Summary      Complete the diagnosis and hand the items over for budgeting
// @Tags         work-orders
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Service order ID"
// @Success      200  {object}  dto.WorkOrderResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      422  {object}  response.ErrorResponse
// @Router       /work-orders/{id}/complete-diagnosis [patch]
func (h *WorkOrderHandler) CompleteDiagnosis(c *gin.Context) {
	workOrder, err := h.completeDiagnosisUC.Execute(c.Request.Context(), c.Param("id"))
	respondWorkOrderTransition(c, "complete_diagnosis", workOrder, err)
}

// FailDiagnosis godoc
// @Summary      Report that the diagnosis could not be done, cancelling the service order
// @Tags         work-orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                    true  "Service order ID"
// @Param        body  body      dto.FailDiagnosisRequest  true  "Failure reason"
// @Success      200   {object}  dto.WorkOrderResponse
// @Failure      400   {object}  response.ErrorResponse
// @Failure      404   {object}  response.ErrorResponse
// @Failure      422   {object}  response.ErrorResponse
// @Router       /work-orders/{id}/fail-diagnosis [patch]
func (h *WorkOrderHandler) FailDiagnosis(c *gin.Context) {
	var req dto.FailDiagnosisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workOrder, err := h.failDiagnosisUC.Execute(c.Request.Context(), workorder.FailDiagnosisInput{
		WorkOrderID: c.Param("id"),
		Reason:      req.Reason,
	})
	respondWorkOrderTransition(c, "fail_diagnosis", workOrder, err)
}

// FinishExecution godoc
// @Summary      Finish the repair and notify the service order
// @Tags         work-orders
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Service order ID"
// @Success      200  {object}  dto.WorkOrderResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      422  {object}  response.ErrorResponse
// @Router       /work-orders/{id}/finish-execution [patch]
func (h *WorkOrderHandler) FinishExecution(c *gin.Context) {
	workOrder, err := h.finishExecutionUC.Execute(c.Request.Context(), c.Param("id"))
	respondWorkOrderTransition(c, "finish_execution", workOrder, err)
}

func respondWorkOrderTransition(c *gin.Context, action string, workOrder *entity.WorkOrder, err error) {
	if err != nil {
		observability.Failure(c.Request.Context(), eventWorkOrderFailed, err,
			slog.String("action", action),
			slog.String("order_id", c.Param("id")),
		)
		response.Error(c, err)
		return
	}

	observability.Event(c.Request.Context(), eventWorkOrderStatusChanged,
		slog.String("action", action),
		slog.String("order_id", workOrder.OrderID()),
		slog.String("saga_id", workOrder.SagaID()),
		slog.String("status", workOrder.Status().String()),
	)
	c.JSON(http.StatusOK, dto.ToWorkOrderResponse(workOrder))
}
