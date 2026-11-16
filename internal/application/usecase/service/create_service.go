package service

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type CreateServiceInput struct {
	Name             string
	Description      string
	LaborPrice       float64
	EstimatedMinutes int
}

type CreateServiceUseCase struct {
	serviceRepo repository.ServiceRepository
}

func NewCreateServiceUseCase(serviceRepo repository.ServiceRepository) *CreateServiceUseCase {
	return &CreateServiceUseCase{serviceRepo: serviceRepo}
}

func (u *CreateServiceUseCase) Execute(ctx context.Context, input CreateServiceInput) (*entity.Service, error) {
	svc, err := entity.NewService(input.Name, input.Description, input.LaborPrice, input.EstimatedMinutes)
	if err != nil {
		return nil, err
	}

	if err := u.serviceRepo.Create(ctx, svc); err != nil {
		return nil, err
	}

	return svc, nil
}
