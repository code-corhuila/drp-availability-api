package memory

import (
	"context"
	"sync"
	"time"

	"github.com/code-corhuila/drp-availability-api/internal/app"
	"github.com/code-corhuila/drp-availability-api/internal/domain"
)

const (
	NorteID  = "11111111-1111-1111-1111-111111111111"
	AulaID   = "22222222-2222-2222-2222-222222222222"
	CubID    = "33333333-3333-3333-3333-333333333333"
	OfficeID = "55555555-5555-5555-5555-555555555555"
)

var (
	SeedStart        = time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	SeedConfirmedEnd = time.Date(2026, 10, 10, 16, 0, 0, 0, time.UTC)
	SeedBlockEnd     = time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC)
)

type Catalog struct {
	mu   sync.RWMutex
	byID map[string]app.SpaceSnapshot
}

type Occupancy struct {
	mu   sync.RWMutex
	rows []domain.Occupancy
}

// Corte2 is an in-memory catalog + occupancy seed. DP-05 still holds: this
// slice does not HTTP-call space or reservation. Wire those later.
func Corte2() (*Catalog, *Occupancy, error) {
	now := time.Unix(0, 0).UTC()
	spaces := []app.SpaceSnapshot{
		{ID: NorteID, Name: "Sala Norte", Kind: "MEETING_ROOM", Capacity: 8, Active: true},
		{ID: AulaID, Name: "Aula Magna", Kind: "AUDITORIUM", Capacity: 80, Active: true},
		{ID: CubID, Name: "Cubículo 1", Kind: "WORKSTATION", Capacity: 1, Active: true},
		{ID: OfficeID, Name: "Oficina Cerrada", Kind: "PRIVATE_OFFICE", Capacity: 4, Active: false},
	}
	cat := &Catalog{byID: map[string]app.SpaceSnapshot{}}
	for _, s := range spaces {
		cat.byID[s.ID] = s
	}
	conf, err := domain.NewOccupancy("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", NorteID, "CONFIRMED", "reservation-seed", SeedStart, SeedConfirmedEnd, now)
	if err != nil {
		return nil, nil, err
	}
	block, err := domain.NewOccupancy("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", AulaID, "BLOCK", "block-seed", SeedStart, SeedBlockEnd, now)
	if err != nil {
		return nil, nil, err
	}
	return cat, &Occupancy{rows: []domain.Occupancy{conf, block}}, nil
}

func (c *Catalog) List(_ context.Context) ([]app.SpaceSnapshot, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]app.SpaceSnapshot, 0, len(c.byID))
	for _, s := range c.byID {
		out = append(out, s)
	}
	return out, nil
}

func (c *Catalog) FindByID(_ context.Context, id string) (app.SpaceSnapshot, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.byID[id]
	if !ok {
		return app.SpaceSnapshot{}, app.ErrNotFound
	}
	return s, nil
}

func (o *Occupancy) List(_ context.Context) ([]domain.Occupancy, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return append([]domain.Occupancy{}, o.rows...), nil
}
