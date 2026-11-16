package part

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/repository"
)

type ListPartsUseCase struct {
	partRepo repository.PartRepository
}

func NewListPartsUseCase(partRepo repository.PartRepository) *ListPartsUseCase {
	return &ListPartsUseCase{partRepo: partRepo}
}

func (u *ListPartsUseCase) Execute(ctx context.Context) ([]*entity.Part, error) {
	return u.partRepo.FindAll(ctx)
}
