package repository

import (
	"context"

	"github.com/gabrielcamargo/oficina-execution-service/internal/domain/entity"
)

type PartRepository interface {
	Create(ctx context.Context, part *entity.Part) error
	Update(ctx context.Context, part *entity.Part) error
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*entity.Part, error)
	FindAll(ctx context.Context) ([]*entity.Part, error)
}
