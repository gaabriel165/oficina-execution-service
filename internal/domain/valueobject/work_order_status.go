package valueobject

import "errors"

var ErrInvalidWorkOrderStatus = errors.New("invalid work order status")

type WorkOrderStatus string

const (
	WorkOrderStatusAwaitingDiagnosis WorkOrderStatus = "awaiting_diagnosis"
	WorkOrderStatusInDiagnosis       WorkOrderStatus = "in_diagnosis"
	WorkOrderStatusDiagnosed         WorkOrderStatus = "diagnosed"
	WorkOrderStatusInExecution       WorkOrderStatus = "in_execution"
	WorkOrderStatusFinished          WorkOrderStatus = "finished"
	WorkOrderStatusCancelled         WorkOrderStatus = "cancelled"
)

var allowedWorkOrderTransitions = map[WorkOrderStatus][]WorkOrderStatus{
	WorkOrderStatusAwaitingDiagnosis: {WorkOrderStatusInDiagnosis, WorkOrderStatusCancelled},
	WorkOrderStatusInDiagnosis:       {WorkOrderStatusDiagnosed, WorkOrderStatusCancelled},
	WorkOrderStatusDiagnosed:         {WorkOrderStatusInExecution, WorkOrderStatusCancelled},
	WorkOrderStatusInExecution:       {WorkOrderStatusFinished, WorkOrderStatusCancelled},
	WorkOrderStatusFinished:          {},
	WorkOrderStatusCancelled:         {},
}

func NewWorkOrderStatus(value string) (WorkOrderStatus, error) {
	status := WorkOrderStatus(value)
	if _, known := allowedWorkOrderTransitions[status]; !known {
		return "", ErrInvalidWorkOrderStatus
	}
	return status, nil
}

func (s WorkOrderStatus) CanTransitionTo(next WorkOrderStatus) bool {
	for _, status := range allowedWorkOrderTransitions[s] {
		if status == next {
			return true
		}
	}
	return false
}

func (s WorkOrderStatus) String() string {
	return string(s)
}
