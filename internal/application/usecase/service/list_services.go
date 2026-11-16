package service

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type ListServicesUseCase struct {
	serviceRepo repository.ServiceRepository
}

func NewListServicesUseCase(serviceRepo repository.ServiceRepository) *ListServicesUseCase {
	return &ListServicesUseCase{serviceRepo: serviceRepo}
}

func (u *ListServicesUseCase) Execute(ctx context.Context) ([]*entity.Service, error) {
	return u.serviceRepo.FindAll(ctx)
}
