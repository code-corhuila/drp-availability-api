package app

import (
	"context"

	"github.com/code-corhuila/drp-availability-api/internal/domain"
)

type SpaceSnapshot struct {
	ID       string
	Name     string
	Kind     string
	Capacity int
	Active   bool
}

type SpaceCatalog interface {
	List(ctx context.Context) ([]SpaceSnapshot, error)
	FindByID(ctx context.Context, id string) (SpaceSnapshot, error)
}

type OccupancyRepository interface {
	List(ctx context.Context) ([]domain.Occupancy, error)
}

type TokenVerifier interface {
	Parse(ctx context.Context, raw string) (subject string, err error)
}
