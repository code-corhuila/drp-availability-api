package occupancy

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/code-corhuila/drp-availability-api/internal/adapters/catalog"
	"github.com/code-corhuila/drp-availability-api/internal/domain"
)

// HTTP reads CONFIRMED bookings from reservation-api. USER tokens only see
// the caller's rows (E-11); walking hop until an internal list exists.
type HTTP struct {
	BaseURL string
	Client  *http.Client
}

type reservationDTO struct {
	ID      string `json:"id"`
	SpaceID string `json:"spaceId"`
	StartAt string `json:"startAt"`
	EndAt   string `json:"endAt"`
	State   string `json:"state"`
}

type listEnvelope struct {
	Data []reservationDTO `json:"data"`
}

func (h HTTP) List(ctx context.Context) ([]domain.Occupancy, error) {
	if h.BaseURL == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(h.BaseURL, "/")+"/api/v1/reservations?state=CONFIRMED&limit=100", nil)
	if err != nil {
		return nil, err
	}
	if tok := catalog.Bearer(ctx); tok != "" {
		req.Header.Set("Authorization", tok)
	}
	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, nil
	}
	var body listEnvelope
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, nil
	}
	now := time.Now().UTC()
	out := make([]domain.Occupancy, 0, len(body.Data))
	for _, row := range body.Data {
		start, err1 := time.Parse(time.RFC3339, row.StartAt)
		end, err2 := time.Parse(time.RFC3339, row.EndAt)
		if err1 != nil || err2 != nil {
			continue
		}
		id := row.ID
		if id == "" {
			id = row.SpaceID
		}
		got, oerr := domain.NewOccupancy(id, row.SpaceID, string(domain.SourceConfirmed), id, start, end, now)
		if oerr != nil {
			continue
		}
		out = append(out, got)
	}
	return out, nil
}
