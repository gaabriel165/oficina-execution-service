package entity

type WorkOrderService struct {
	serviceID  string
	name       string
	laborPrice float64
}

func newWorkOrderService(service *Service) *WorkOrderService {
	return &WorkOrderService{serviceID: service.ID(), name: service.Name(), laborPrice: service.LaborPrice()}
}

func RestoreWorkOrderService(serviceID, name string, laborPrice float64) *WorkOrderService {
	return &WorkOrderService{serviceID: serviceID, name: name, laborPrice: laborPrice}
}

func (s *WorkOrderService) ServiceID() string   { return s.serviceID }
func (s *WorkOrderService) Name() string        { return s.name }
func (s *WorkOrderService) LaborPrice() float64 { return s.laborPrice }

type WorkOrderPart struct {
	partID    string
	name      string
	quantity  int
	unitPrice float64
}

func newWorkOrderPart(part *Part, quantity int) *WorkOrderPart {
	return &WorkOrderPart{partID: part.ID(), name: part.Name(), quantity: quantity, unitPrice: part.UnitPrice()}
}

func RestoreWorkOrderPart(partID, name string, quantity int, unitPrice float64) *WorkOrderPart {
	return &WorkOrderPart{partID: partID, name: name, quantity: quantity, unitPrice: unitPrice}
}

func (p *WorkOrderPart) PartID() string     { return p.partID }
func (p *WorkOrderPart) Name() string       { return p.name }
func (p *WorkOrderPart) Quantity() int      { return p.quantity }
func (p *WorkOrderPart) UnitPrice() float64 { return p.unitPrice }
