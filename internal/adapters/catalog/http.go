package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/code-corhuila/drp-availability-api/internal/app"
)

type ctxKey struct{}

func WithBearer(ctx context.Context, authorization string) context.Context {
	return context.WithValue(ctx, ctxKey{}, strings.TrimSpace(authorization))
}

func Bearer(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

// HTTP reads the space catalog over HTTP. Never opens schema space.
type HTTP struct {
	BaseURL string
	Client  *http.Client
}

type spaceDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Capacity  int    `json:"capacity"`
	Available bool   `json:"available"`
}

type listEnvelope struct {
	Data []spaceDTO `json:"data"`
}

func (h HTTP) client() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return &http.Client{Timeout: 3 * time.Second}
}

func (h HTTP) List(ctx context.Context) ([]app.SpaceSnapshot, error) {
	if h.BaseURL == "" {
		return nil, app.ErrNotFound
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(h.BaseURL, "/")+"/api/v1/spaces?limit=100", nil)
	if err != nil {
		return nil, err
	}
	if tok := Bearer(ctx); tok != "" {
		req.Header.Set("Authorization", tok)
	}
	res, err := h.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("space catalog HTTP %d", res.StatusCode)
	}
	var body listEnvelope
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	out := make([]app.SpaceSnapshot, 0, len(body.Data))
	for _, s := range body.Data {
		out = append(out, app.SpaceSnapshot{ID: s.ID, Name: s.Name, Kind: s.Kind, Capacity: s.Capacity, Active: s.Available})
	}
	return out, nil
}

func (h HTTP) FindByID(ctx context.Context, id string) (app.SpaceSnapshot, error) {
	if h.BaseURL == "" {
		return app.SpaceSnapshot{}, app.ErrNotFound
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(h.BaseURL, "/")+"/api/v1/spaces/"+id, nil)
	if err != nil {
		return app.SpaceSnapshot{}, err
	}
	if tok := Bearer(ctx); tok != "" {
		req.Header.Set("Authorization", tok)
	}
	res, err := h.client().Do(req)
	if err != nil {
		return app.SpaceSnapshot{}, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return app.SpaceSnapshot{}, app.ErrNotFound
	}
	if res.StatusCode >= 400 {
		return app.SpaceSnapshot{}, fmt.Errorf("space catalog HTTP %d", res.StatusCode)
	}
	var s spaceDTO
	if err := json.NewDecoder(res.Body).Decode(&s); err != nil {
		return app.SpaceSnapshot{}, err
	}
	return app.SpaceSnapshot{ID: s.ID, Name: s.Name, Kind: s.Kind, Capacity: s.Capacity, Active: s.Available}, nil
}
