package dto

import (
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/internal/application/usecase/workorder"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
)

type AddWorkOrderServiceRequest struct {
	ServiceID string `json:"service_id" binding:"required"`
}

type AddWorkOrderPartRequest struct {
	PartID   string `json:"part_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
}

type FailDiagnosisRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type WorkOrderServiceResponse struct {
	ServiceID  string  `json:"service_id"`
	Name       string  `json:"name"`
	LaborPrice float64 `json:"labor_price"`
}

type WorkOrderPartResponse struct {
	PartID    string  `json:"part_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type WorkOrderResponse struct {
	ID                 string                     `json:"id"`
	OrderID            string                     `json:"order_id"`
	SagaID             string                     `json:"saga_id"`
	CustomerID         string                     `json:"customer_id"`
	VehicleID          string                     `json:"vehicle_id"`
	Notes              string                     `json:"notes"`
	Status             string                     `json:"status"`
	Services           []WorkOrderServiceResponse `json:"services"`
	Parts              []WorkOrderPartResponse    `json:"parts"`
	StockReserved      bool                       `json:"stock_reserved"`
	CancellationReason string                     `json:"cancellation_reason,omitempty"`
	ExecutionStartedAt *time.Time                 `json:"execution_started_at,omitempty"`
	FinishedAt         *time.Time                 `json:"finished_at,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
}

type ServiceMetricResponse struct {
	ServiceName     string  `json:"service_name"`
	AvgMinutes      float64 `json:"avg_minutes"`
	CompletedOrders int     `json:"completed_orders"`
}

type ExecutionMetricsResponse struct {
	OverallAvgMinutes float64                 `json:"overall_avg_minutes"`
	CompletedOrders   int                     `json:"completed_orders"`
	ByService         []ServiceMetricResponse `json:"by_service"`
}

func ToWorkOrderResponse(workOrder *entity.WorkOrder) WorkOrderResponse {
	services := make([]WorkOrderServiceResponse, 0, len(workOrder.Services()))
	for _, service := range workOrder.Services() {
		services = append(services, WorkOrderServiceResponse{
			ServiceID:  service.ServiceID(),
			Name:       service.Name(),
			LaborPrice: service.LaborPrice(),
		})
	}

	parts := make([]WorkOrderPartResponse, 0, len(workOrder.Parts()))
	for _, part := range workOrder.Parts() {
		parts = append(parts, WorkOrderPartResponse{
			PartID:    part.PartID(),
			Name:      part.Name(),
			Quantity:  part.Quantity(),
			UnitPrice: part.UnitPrice(),
		})
	}

	return WorkOrderResponse{
		ID:                 workOrder.ID(),
		OrderID:            workOrder.OrderID(),
		SagaID:             workOrder.SagaID(),
		CustomerID:         workOrder.CustomerID(),
		VehicleID:          workOrder.VehicleID(),
		Notes:              workOrder.Notes(),
		Status:             workOrder.Status().String(),
		Services:           services,
		Parts:              parts,
		StockReserved:      workOrder.StockReserved(),
		CancellationReason: workOrder.CancellationReason(),
		ExecutionStartedAt: workOrder.ExecutionStartedAt(),
		FinishedAt:         workOrder.FinishedAt(),
		CreatedAt:          workOrder.CreatedAt(),
		UpdatedAt:          workOrder.UpdatedAt(),
	}
}

func ToExecutionMetricsResponse(metrics *workorder.ExecutionMetrics) ExecutionMetricsResponse {
	byService := make([]ServiceMetricResponse, 0, len(metrics.ByService))
	for _, metric := range metrics.ByService {
		byService = append(byService, ServiceMetricResponse{
			ServiceName:     metric.ServiceName,
			AvgMinutes:      metric.AvgMinutes,
			CompletedOrders: metric.CompletedOrders,
		})
	}
	return ExecutionMetricsResponse{
		OverallAvgMinutes: metrics.OverallAvgMinutes,
		CompletedOrders:   metrics.CompletedOrders,
		ByService:         byService,
	}
}
