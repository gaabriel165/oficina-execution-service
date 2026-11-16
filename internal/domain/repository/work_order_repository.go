package repository

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/valueobject"
)

type WorkOrderRepository interface {
	Create(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error
	Update(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error
	UpdateReservingStock(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error
	UpdateReleasingStock(ctx context.Context, workOrder *entity.WorkOrder, outgoing ...messaging.OutgoingMessage) error
	Enqueue(ctx context.Context, outgoing ...messaging.OutgoingMessage) error
	FindByID(ctx context.Context, id string) (*entity.WorkOrder, error)
	FindByStatus(ctx context.Context, status valueobject.WorkOrderStatus) ([]*entity.WorkOrder, error)
	FindAll(ctx context.Context) ([]*entity.WorkOrder, error)
}
