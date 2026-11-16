package part

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type DeletePartUseCase struct {
	partRepo repository.PartRepository
}

func NewDeletePartUseCase(partRepo repository.PartRepository) *DeletePartUseCase {
	return &DeletePartUseCase{partRepo: partRepo}
}

func (u *DeletePartUseCase) Execute(ctx context.Context, id string) error {
	if _, err := u.partRepo.FindByID(ctx, id); err != nil {
		return err
	}
	return u.partRepo.Delete(ctx, id)
}
