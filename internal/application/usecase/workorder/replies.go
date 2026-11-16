package workorder

import (
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/messaging"
)

func reply(messageType string, workOrder *entity.WorkOrder, payload any) (messaging.OutgoingMessage, error) {
	return replyTo(messageType, workOrder.SagaID(), workOrder.OrderID(), payload)
}

func replyTo(messageType, sagaID, orderID string, payload any) (messaging.OutgoingMessage, error) {
	message, err := messaging.NewMessage(messageType, sagaID, orderID, payload)
	if err != nil {
		return messaging.OutgoingMessage{}, err
	}
	return messaging.OutgoingMessage{Destination: messaging.DestinationSagaReplies, Message: message}, nil
}

func serviceLines(workOrder *entity.WorkOrder) []messaging.ServiceLine {
	lines := make([]messaging.ServiceLine, 0, len(workOrder.Services()))
	for _, service := range workOrder.Services() {
		lines = append(lines, messaging.ServiceLine{
			ServiceID:  service.ServiceID(),
			Name:       service.Name(),
			LaborPrice: service.LaborPrice(),
		})
	}
	return lines
}

func partLines(workOrder *entity.WorkOrder) []messaging.PartLine {
	lines := make([]messaging.PartLine, 0, len(workOrder.Parts()))
	for _, part := range workOrder.Parts() {
		lines = append(lines, messaging.PartLine{
			PartID:    part.PartID(),
			Name:      part.Name(),
			Quantity:  part.Quantity(),
			UnitPrice: part.UnitPrice(),
		})
	}
	return lines
}
