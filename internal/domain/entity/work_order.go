package entity

import (
	"time"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
)

type WorkOrder struct {
	id                 string
	sagaID             string
	customerID         string
	vehicleID          string
	notes              string
	status             valueobject.WorkOrderStatus
	services           []*WorkOrderService
	parts              []*WorkOrderPart
	stockReserved      bool
	cancellationReason string
	executionStartedAt *time.Time
	finishedAt         *time.Time
	createdAt          time.Time
	updatedAt          time.Time
}

type WorkOrderSnapshot struct {
	ID                 string
	SagaID             string
	CustomerID         string
	VehicleID          string
	Notes              string
	Status             valueobject.WorkOrderStatus
	Services           []*WorkOrderService
	Parts              []*WorkOrderPart
	StockReserved      bool
	CancellationReason string
	ExecutionStartedAt *time.Time
	FinishedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewWorkOrder(orderID, sagaID, customerID, vehicleID, notes string) (*WorkOrder, error) {
	if orderID == "" {
		return nil, ErrWorkOrderOrderRequired
	}
	if sagaID == "" {
		return nil, ErrWorkOrderSagaRequired
	}

	now := time.Now()
	return &WorkOrder{
		id:         orderID,
		sagaID:     sagaID,
		customerID: customerID,
		vehicleID:  vehicleID,
		notes:      notes,
		status:     valueobject.WorkOrderStatusAwaitingDiagnosis,
		services:   []*WorkOrderService{},
		parts:      []*WorkOrderPart{},
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

func RestoreWorkOrder(snapshot WorkOrderSnapshot) *WorkOrder {
	return &WorkOrder{
		id:                 snapshot.ID,
		sagaID:             snapshot.SagaID,
		customerID:         snapshot.CustomerID,
		vehicleID:          snapshot.VehicleID,
		notes:              snapshot.Notes,
		status:             snapshot.Status,
		services:           snapshot.Services,
		parts:              snapshot.Parts,
		stockReserved:      snapshot.StockReserved,
		cancellationReason: snapshot.CancellationReason,
		executionStartedAt: snapshot.ExecutionStartedAt,
		finishedAt:         snapshot.FinishedAt,
		createdAt:          snapshot.CreatedAt,
		updatedAt:          snapshot.UpdatedAt,
	}
}

func (w *WorkOrder) ID() string                          { return w.id }
func (w *WorkOrder) OrderID() string                     { return w.id }
func (w *WorkOrder) SagaID() string                      { return w.sagaID }
func (w *WorkOrder) CustomerID() string                  { return w.customerID }
func (w *WorkOrder) VehicleID() string                   { return w.vehicleID }
func (w *WorkOrder) Notes() string                       { return w.notes }
func (w *WorkOrder) Status() valueobject.WorkOrderStatus { return w.status }
func (w *WorkOrder) Services() []*WorkOrderService       { return w.services }
func (w *WorkOrder) Parts() []*WorkOrderPart             { return w.parts }
func (w *WorkOrder) StockReserved() bool                 { return w.stockReserved }
func (w *WorkOrder) CancellationReason() string          { return w.cancellationReason }
func (w *WorkOrder) ExecutionStartedAt() *time.Time      { return w.executionStartedAt }
func (w *WorkOrder) FinishedAt() *time.Time              { return w.finishedAt }
func (w *WorkOrder) CreatedAt() time.Time                { return w.createdAt }
func (w *WorkOrder) UpdatedAt() time.Time                { return w.updatedAt }

func (w *WorkOrder) IsCancelled() bool {
	return w.status == valueobject.WorkOrderStatusCancelled
}

func (w *WorkOrder) StartDiagnosis() error {
	return w.transitionTo(valueobject.WorkOrderStatusInDiagnosis)
}

func (w *WorkOrder) AddService(service *Service) error {
	if w.status != valueobject.WorkOrderStatusInDiagnosis {
		return ErrWorkOrderNotInDiagnosis
	}
	w.services = append(w.services, newWorkOrderService(service))
	w.updatedAt = time.Now()
	return nil
}

func (w *WorkOrder) AddPart(part *Part, quantity int) error {
	if w.status != valueobject.WorkOrderStatusInDiagnosis {
		return ErrWorkOrderNotInDiagnosis
	}
	if quantity <= 0 {
		return ErrWorkOrderInvalidQuantity
	}

	w.updatedAt = time.Now()
	for _, existing := range w.parts {
		if existing.partID == part.ID() {
			existing.quantity += quantity
			return nil
		}
	}
	w.parts = append(w.parts, newWorkOrderPart(part, quantity))
	return nil
}

func (w *WorkOrder) CompleteDiagnosis() error {
	if w.status == valueobject.WorkOrderStatusInDiagnosis && len(w.services) == 0 && len(w.parts) == 0 {
		return ErrWorkOrderNoItems
	}
	return w.transitionTo(valueobject.WorkOrderStatusDiagnosed)
}

func (w *WorkOrder) StartExecution() error {
	if err := w.transitionTo(valueobject.WorkOrderStatusInExecution); err != nil {
		return err
	}
	now := time.Now()
	w.executionStartedAt = &now
	w.stockReserved = true
	return nil
}

func (w *WorkOrder) Finish() error {
	if err := w.transitionTo(valueobject.WorkOrderStatusFinished); err != nil {
		return err
	}
	now := time.Now()
	w.finishedAt = &now
	return nil
}

func (w *WorkOrder) Cancel(reason string) error {
	if reason == "" {
		return ErrWorkOrderReasonRequired
	}
	if err := w.transitionTo(valueobject.WorkOrderStatusCancelled); err != nil {
		return err
	}
	w.cancellationReason = reason
	return nil
}

func (w *WorkOrder) ReleaseStock() {
	w.stockReserved = false
	w.updatedAt = time.Now()
}

func (w *WorkOrder) ExecutionDuration() time.Duration {
	if w.executionStartedAt == nil || w.finishedAt == nil {
		return 0
	}
	return w.finishedAt.Sub(*w.executionStartedAt)
}

func (w *WorkOrder) transitionTo(next valueobject.WorkOrderStatus) error {
	if !w.status.CanTransitionTo(next) {
		return ErrWorkOrderInvalidTransition
	}
	w.status = next
	w.updatedAt = time.Now()
	return nil
}
