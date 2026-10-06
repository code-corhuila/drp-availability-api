package domain

import (
	"time"
)

type DateTimeRange struct {
	Start time.Time
	End   time.Time
}

func NewDateTimeRange(start, end time.Time) (DateTimeRange, error) {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return DateTimeRange{}, ErrInvalidInput
	}
	return DateTimeRange{Start: start.UTC(), End: end.UTC()}, nil
}

func (r DateTimeRange) Overlaps(other DateTimeRange) bool {
	return r.Start.Before(other.End) && other.Start.Before(r.End)
}

type SourceKind string

const (
	SourceBlock     SourceKind = "BLOCK"
	SourceConfirmed SourceKind = "CONFIRMED"
)

func ParseSourceKind(v string) (SourceKind, error) {
	switch SourceKind(v) {
	case SourceBlock, SourceConfirmed:
		return SourceKind(v), nil
	default:
		return "", ErrInvalidInput
	}
}

// Occupancy is a read-model projection. Catalog SoT is space; CONFIRMED
// overlap SoT is reservation. spaceId is a reference, not a cross-domain FK.
type Occupancy struct {
	ID         string
	SpaceID    string
	Period     DateTimeRange
	SourceKind SourceKind
	SourceID   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

func NewOccupancy(id, spaceID, sourceKind, sourceID string, start, end, now time.Time) (Occupancy, error) {
	k, err := ParseSourceKind(sourceKind)
	if err != nil {
		return Occupancy{}, err
	}
	p, err := NewDateTimeRange(start, end)
	if err != nil {
		return Occupancy{}, err
	}
	if id == "" || spaceID == "" {
		return Occupancy{}, ErrInvalidInput
	}
	return Occupancy{
		ID: id, SpaceID: spaceID, Period: p, SourceKind: k, SourceID: sourceID,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (o Occupancy) Active() bool { return o.DeletedAt == nil }

func Occupied(rows []Occupancy, spaceID string, period DateTimeRange) bool {
	for _, r := range rows {
		if r.Active() && r.SpaceID == spaceID && r.Period.Overlaps(period) {
			return true
		}
	}
	return false
}
