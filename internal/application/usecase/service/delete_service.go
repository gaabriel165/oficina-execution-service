package service

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type DeleteServiceUseCase struct {
	serviceRepo repository.ServiceRepository
}

func NewDeleteServiceUseCase(serviceRepo repository.ServiceRepository) *DeleteServiceUseCase {
	return &DeleteServiceUseCase{serviceRepo: serviceRepo}
}

func (u *DeleteServiceUseCase) Execute(ctx context.Context, id string) error {
	if _, err := u.serviceRepo.FindByID(ctx, id); err != nil {
		return err
	}
	return u.serviceRepo.Delete(ctx, id)
}
