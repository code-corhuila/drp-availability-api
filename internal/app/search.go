package app

import (
	"context"
	"sort"

	"github.com/code-corhuila/drp-availability-api/internal/domain"
)

const (
	ReasonOK               = "OK"
	ReasonConfirmedOverlap = "CONFIRMED_OVERLAP"
	ReasonBlocked          = "BLOCKED"
	ReasonInactive         = "INACTIVE"
)

type ListFilter struct {
	Period      domain.DateTimeRange
	Kind        *string
	MinCapacity *int
	Page        int
	Limit       int
}

type SpacePage struct {
	Items      []SpaceSnapshot
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type Availability struct {
	SpaceID   string
	Available bool
	Reason    string
}

type Search struct {
	Spaces    SpaceCatalog
	Occupancy OccupancyRepository
}

func (s Search) ListAvailable(ctx context.Context, f ListFilter) (SpacePage, error) {
	if f.Page < 1 || f.Limit < 1 || f.Limit > 100 {
		return SpacePage{}, ErrValidation
	}
	spaces, rows, err := s.load(ctx)
	if err != nil {
		return SpacePage{}, err
	}
	filtered := make([]SpaceSnapshot, 0)
	for _, sp := range spaces {
		if f.Kind != nil && sp.Kind != *f.Kind {
			continue
		}
		if f.MinCapacity != nil && sp.Capacity < *f.MinCapacity {
			continue
		}
		got := Classify(sp, rows, f.Period)
		if !got.Available {
			continue
		}
		filtered = append(filtered, sp)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Name != filtered[j].Name {
			return filtered[i].Name < filtered[j].Name
		}
		return filtered[i].ID < filtered[j].ID
	})
	total := len(filtered)
	totalPages := 0
	if total > 0 {
		totalPages = (total + f.Limit - 1) / f.Limit
	}
	start := (f.Page - 1) * f.Limit
	if start >= total {
		return SpacePage{Items: []SpaceSnapshot{}, Page: f.Page, Limit: f.Limit, Total: total, TotalPages: totalPages}, nil
	}
	end := start + f.Limit
	if end > total {
		end = total
	}
	return SpacePage{Items: filtered[start:end], Page: f.Page, Limit: f.Limit, Total: total, TotalPages: totalPages}, nil
}

func (s Search) One(ctx context.Context, id string, period domain.DateTimeRange) (Availability, error) {
	sp, err := s.Spaces.FindByID(ctx, id)
	if err != nil {
		return Availability{}, err
	}
	rows, err := s.Occupancy.List(ctx)
	if err != nil {
		return Availability{}, err
	}
	return Classify(sp, rows, period), nil
}

func (s Search) load(ctx context.Context) ([]SpaceSnapshot, []domain.Occupancy, error) {
	spaces, err := s.Spaces.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.Occupancy.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	return spaces, rows, nil
}

// Classify applies E-10 reason order: CONFIRMED_OVERLAP, BLOCKED, INACTIVE.
func Classify(sp SpaceSnapshot, rows []domain.Occupancy, period domain.DateTimeRange) Availability {
	if domain.OccupiedBy(rows, sp.ID, period, domain.SourceConfirmed) {
		return Availability{SpaceID: sp.ID, Available: false, Reason: ReasonConfirmedOverlap}
	}
	if domain.OccupiedBy(rows, sp.ID, period, domain.SourceBlock) {
		return Availability{SpaceID: sp.ID, Available: false, Reason: ReasonBlocked}
	}
	if !sp.Active {
		return Availability{SpaceID: sp.ID, Available: false, Reason: ReasonInactive}
	}
	return Availability{SpaceID: sp.ID, Available: true, Reason: ReasonOK}
}
