package httpadapter

import (
	"net/http"

	"github.com/code-corhuila/drp-availability-api/internal/app"
)

type Deps struct {
	Service string
	Search  app.Search
	Tokens  app.TokenVerifier
}

func NewMux(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", HealthHandler(d.Service))
	if d.Tokens != nil {
		auth := RequireBearer(d.Tokens)
		mux.Handle("GET /api/v1/spaces/available", auth(ListAvailable(d.Search)))
		mux.Handle("GET /api/v1/spaces/{spaceId}/availability", auth(GetAvailability(d.Search)))
	}
	mux.Handle("/", NotFoundHandler())
	return WithCorrelation(stripSpoofedIdentity(mux))
}
