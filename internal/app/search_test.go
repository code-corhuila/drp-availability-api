package app

import (
	"testing"
	"time"

	"github.com/code-corhuila/drp-availability-api/internal/domain"
)

func TestClassifyOrder(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	start := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	period, _ := domain.NewDateTimeRange(start, end)
	conf, _ := domain.NewOccupancy("o1", "space-1", "CONFIRMED", "r1", start, end, now)
	block, _ := domain.NewOccupancy("o2", "space-1", "BLOCK", "b1", start, end, now)
	sp := SpaceSnapshot{ID: "space-1", Name: "Sala", Kind: "MEETING_ROOM", Capacity: 8, Active: false}

	got := Classify(sp, []domain.Occupancy{conf, block}, period)
	if got.Reason != ReasonConfirmedOverlap || got.Available {
		t.Fatalf("confirmed first %+v", got)
	}
	got = Classify(sp, []domain.Occupancy{block}, period)
	if got.Reason != ReasonBlocked {
		t.Fatalf("block before inactive %+v", got)
	}
	got = Classify(sp, nil, period)
	if got.Reason != ReasonInactive {
		t.Fatalf("inactive %+v", got)
	}
	sp.Active = true
	got = Classify(sp, nil, period)
	if !got.Available || got.Reason != ReasonOK {
		t.Fatalf("ok %+v", got)
	}
}
