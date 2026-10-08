package main

import (
	"log"
	"net/http"
	"os"
	"time"

	httpadapter "github.com/code-corhuila/drp-availability-api/internal/adapters/http"
	"github.com/code-corhuila/drp-availability-api/internal/adapters/memory"
	"github.com/code-corhuila/drp-availability-api/internal/adapters/security"
	"github.com/code-corhuila/drp-availability-api/internal/app"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8086"
	}
	service := os.Getenv("SERVICE_NAME")
	if service == "" {
		service = "availability-service"
	}
	jwksURL := os.Getenv("IDENTITY_JWKS_URL")
	if jwksURL == "" {
		jwksURL = "http://identity-api:8081/api/v1/auth/jwks"
	}

	cat, occ, err := memory.Corte2()
	if err != nil {
		log.Fatal(err)
	}
	tokens := security.NewVerifier(security.HTTPJWKS{
		URL: jwksURL,
		Client: &http.Client{
			Timeout: 3 * time.Second,
		},
	})

	mux := httpadapter.NewMux(httpadapter.Deps{
		Service: service,
		Search:  app.Search{Spaces: cat, Occupancy: occ},
		Tokens:  tokens,
	})

	log.Printf("drp-availability-api listening on %s (in-memory occupancy; DP-05 still; JWKS %s)", addr, jwksURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
