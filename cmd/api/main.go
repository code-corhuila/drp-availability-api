package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/code-corhuila/drp-availability-api/internal/adapters/catalog"
	httpadapter "github.com/code-corhuila/drp-availability-api/internal/adapters/http"
	"github.com/code-corhuila/drp-availability-api/internal/adapters/memory"
	"github.com/code-corhuila/drp-availability-api/internal/adapters/occupancy"
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

	var spaces app.SpaceCatalog
	var occ app.OccupancyRepository
	store := "memory Corte2"
	if spaceURL := os.Getenv("SPACE_API_URL"); spaceURL != "" {
		spaces = catalog.HTTP{BaseURL: spaceURL}
		store = "catalog HTTP " + spaceURL
	} else {
		memSpaces, memOcc, err := memory.Corte2()
		if err != nil {
			log.Fatal(err)
		}
		spaces = memSpaces
		occ = memOcc
	}
	if reservationURL := os.Getenv("RESERVATION_API_URL"); reservationURL != "" {
		occ = occupancy.HTTP{BaseURL: reservationURL}
		store += "; occupancy HTTP " + reservationURL
	} else if occ == nil {
		_, memOcc, err := memory.Corte2()
		if err != nil {
			log.Fatal(err)
		}
		occ = memOcc
	}
	tokens := security.NewVerifier(security.HTTPJWKS{
		URL: jwksURL,
		Client: &http.Client{
			Timeout: 3 * time.Second,
		},
	})

	mux := httpadapter.NewMux(httpadapter.Deps{
		Service: service,
		Search:  app.Search{Spaces: spaces, Occupancy: occ},
		Tokens:  tokens,
	})

	log.Printf("drp-availability-api listening on %s (%s; JWKS %s)", addr, store, jwksURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
