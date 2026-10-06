package domain

import (
	"testing"
	"time"
)

func TestOccupancyOverlap(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	end := start.Add(2 * time.Hour)
	row, err := NewOccupancy("o1", "space-1", "CONFIRMED", "res-1", start, end, now)
	if err != nil {
		t.Fatal(err)
	}
	hit, _ := NewDateTimeRange(start.Add(time.Minute), end)
	if !Occupied([]Occupancy{row}, "space-1", hit) {
		t.Fatal("expected occupied")
	}
	miss, _ := NewDateTimeRange(end, end.Add(time.Hour))
	if Occupied([]Occupancy{row}, "space-1", miss) {
		t.Fatal("expected free")
	}
	if Occupied([]Occupancy{row}, "space-2", hit) {
		t.Fatal("other space")
	}
}

func TestOccupancyRejects(t *testing.T) {
	now := time.Now().UTC()
	if _, err := NewOccupancy("o", "s", "PENDING", "x", now, now.Add(time.Hour), now); err == nil {
		t.Fatal("bad source")
	}
}
