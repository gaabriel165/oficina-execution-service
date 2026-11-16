package service

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type GetServiceUseCase struct {
	serviceRepo repository.ServiceRepository
}

func NewGetServiceUseCase(serviceRepo repository.ServiceRepository) *GetServiceUseCase {
	return &GetServiceUseCase{serviceRepo: serviceRepo}
}

func (u *GetServiceUseCase) Execute(ctx context.Context, id string) (*entity.Service, error) {
	return u.serviceRepo.FindByID(ctx, id)
}
