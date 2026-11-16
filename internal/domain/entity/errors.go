package entity

import "errors"

var (
	ErrPartNameRequired      = errors.New("part name is required")
	ErrPartInvalidPrice      = errors.New("part unit price must be greater than zero")
	ErrPartInvalidStock      = errors.New("part stock quantity cannot be negative")
	ErrPartInsufficientStock = errors.New("insufficient stock quantity")

	ErrServiceNameRequired        = errors.New("service name is required")
	ErrServiceInvalidLaborPrice   = errors.New("service labor price must be greater than zero")
	ErrServiceInvalidEstimatedMin = errors.New("service estimated minutes must be greater than zero")

	ErrWorkOrderOrderRequired     = errors.New("work order service order id is required")
	ErrWorkOrderSagaRequired      = errors.New("work order saga id is required")
	ErrWorkOrderInvalidTransition = errors.New("work order status transition not allowed")
	ErrWorkOrderNoItems           = errors.New("work order must have at least one service or part")
	ErrWorkOrderInvalidQuantity   = errors.New("work order part quantity must be greater than zero")
	ErrWorkOrderNotInDiagnosis    = errors.New("work order items can only change during diagnosis")
	ErrWorkOrderReasonRequired    = errors.New("work order failure reason is required")
)
