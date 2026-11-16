package workorder

import (
	"context"
	"sort"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
)

type ServiceMetric struct {
	ServiceName     string
	AvgMinutes      float64
	CompletedOrders int
}

type ExecutionMetrics struct {
	OverallAvgMinutes float64
	CompletedOrders   int
	ByService         []ServiceMetric
}

type GetExecutionMetricsUseCase struct {
	workOrderRepo repository.WorkOrderRepository
}

func NewGetExecutionMetricsUseCase(workOrderRepo repository.WorkOrderRepository) *GetExecutionMetricsUseCase {
	return &GetExecutionMetricsUseCase{workOrderRepo: workOrderRepo}
}

func (u *GetExecutionMetricsUseCase) Execute(ctx context.Context) (*ExecutionMetrics, error) {
	finished, err := u.workOrderRepo.FindByStatus(ctx, valueobject.WorkOrderStatusFinished)
	if err != nil {
		return nil, err
	}

	metrics := &ExecutionMetrics{CompletedOrders: len(finished), ByService: []ServiceMetric{}}
	if len(finished) == 0 {
		return metrics, nil
	}

	totalMinutes := 0.0
	minutesByService := map[string]float64{}
	ordersByService := map[string]int{}
	for _, workOrder := range finished {
		minutes := workOrder.ExecutionDuration().Minutes()
		totalMinutes += minutes
		for _, serviceName := range distinctServiceNames(workOrder) {
			minutesByService[serviceName] += minutes
			ordersByService[serviceName]++
		}
	}

	metrics.OverallAvgMinutes = totalMinutes / float64(len(finished))
	for serviceName, completed := range ordersByService {
		metrics.ByService = append(metrics.ByService, ServiceMetric{
			ServiceName:     serviceName,
			AvgMinutes:      minutesByService[serviceName] / float64(completed),
			CompletedOrders: completed,
		})
	}
	sort.Slice(metrics.ByService, func(first, second int) bool {
		return metrics.ByService[first].ServiceName < metrics.ByService[second].ServiceName
	})
	return metrics, nil
}

func distinctServiceNames(workOrder *entity.WorkOrder) []string {
	seen := map[string]bool{}
	names := []string{}
	for _, service := range workOrder.Services() {
		if seen[service.Name()] {
			continue
		}
		seen[service.Name()] = true
		names = append(names, service.Name())
	}
	return names
}
