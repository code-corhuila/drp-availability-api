package httpadapter

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/code-corhuila/drp-availability-api/internal/adapters/memory"
	"github.com/code-corhuila/drp-availability-api/internal/adapters/security"
	"github.com/code-corhuila/drp-availability-api/internal/app"
	"github.com/golang-jwt/jwt/v5"
)

const testKID = "spacehub-identity-2026"

type testIssuer struct {
	priv *rsa.PrivateKey
}

func availMux(t *testing.T) (http.Handler, testIssuer) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := security.PublicJWKS(&priv.PublicKey, testKID)
	ver := security.NewVerifier(security.StaticJWKS(jwks))
	cat, occ, err := memory.Corte2()
	if err != nil {
		t.Fatal(err)
	}
	return NewMux(Deps{
		Service: "availability-service",
		Search:  app.Search{Spaces: cat, Occupancy: occ},
		Tokens:  ver,
	}), testIssuer{priv: priv}
}

func (i testIssuer) bearer(t *testing.T) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "44444444-4444-4444-4444-444444444444",
		"iat": time.Now().UTC().Unix(),
		"exp": time.Now().UTC().Add(time.Hour).Unix(),
	})
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(i.priv)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + signed
}

func TestListRequiresToken(t *testing.T) {
	mux, _ := availMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/v1/spaces/available?startAt=2026-10-10T14:00:00Z&endAt=2026-10-10T15:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body errorBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "UNAUTHORIZED" {
		t.Fatalf("body %+v", body)
	}
}

func TestListEnvelope(t *testing.T) {
	mux, keys := availMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/available?startAt=2026-10-10T14:00:00Z&endAt=2026-10-10T15:00:00Z", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var env listEnvelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if env.Data == nil || env.Meta.Page != 1 || env.Meta.Total != 1 {
		t.Fatalf("envelope %+v", env)
	}
	if env.Data[0].ID != memory.CubID || env.Data[0].Kind != "WORKSTATION" {
		t.Fatalf("only free cubicle %+v", env.Data)
	}
}

func TestGetMalformedUUID(t *testing.T) {
	mux, keys := availMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/not-a-uuid/availability?startAt=2026-10-10T14:00:00Z&endAt=2026-10-10T15:00:00Z", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "VALIDATION_ERROR" {
		t.Fatalf("body %+v", body)
	}
}

func TestGetMissingSpace(t *testing.T) {
	mux, keys := availMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/99999999-9999-9999-9999-999999999999/availability?startAt=2026-10-10T14:00:00Z&endAt=2026-10-10T15:00:00Z", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestReasonsAndBadRange(t *testing.T) {
	mux, keys := availMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	assertReason := func(id, want string) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/"+id+"/availability?startAt=2026-10-10T14:00:00Z&endAt=2026-10-10T15:00:00Z", nil)
		req.Header.Set("Authorization", keys.bearer(t))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s status %d", id, res.StatusCode)
		}
		var dto availabilityDTO
		if err := json.NewDecoder(res.Body).Decode(&dto); err != nil {
			t.Fatal(err)
		}
		if dto.Reason != want {
			t.Fatalf("%s reason %s want %s", id, dto.Reason, want)
		}
	}
	assertReason(memory.NorteID, "CONFIRMED_OVERLAP")
	assertReason(memory.AulaID, "BLOCKED")
	assertReason(memory.OfficeID, "INACTIVE")
	assertReason(memory.CubID, "OK")

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/spaces/available?startAt=2026-10-10T15:00:00Z&endAt=2026-10-10T14:00:00Z", nil)
	req.Header.Set("Authorization", keys.bearer(t))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("inverted %d", res.StatusCode)
	}
}

func TestHealth(t *testing.T) {
	mux, _ := availMux(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var body healthBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" || body.Service != "availability-service" {
		t.Fatalf("body %+v", body)
	}
}
