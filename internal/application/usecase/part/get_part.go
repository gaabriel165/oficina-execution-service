package part

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type GetPartUseCase struct {
	partRepo repository.PartRepository
}

func NewGetPartUseCase(partRepo repository.PartRepository) *GetPartUseCase {
	return &GetPartUseCase{partRepo: partRepo}
}

func (u *GetPartUseCase) Execute(ctx context.Context, id string) (*entity.Part, error) {
	return u.partRepo.FindByID(ctx, id)
}
