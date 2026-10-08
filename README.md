# drp-availability-api

Availability read-model API (Go). **Hexagonal:** `internal/domain` has no web/DB imports. DDL in [`drp-availability-db`](https://github.com/code-corhuila/drp-availability-db). Engine: [`drp-infra-postgres`](https://github.com/code-corhuila/drp-infra-postgres). This repo must not define a database container.

Contract: `drp-docs` `07-api/contracts/openapi/availability-service.yaml` and `07-api/api-contract.md` (E-09, E-10). RS256 is validated **here** against identity JWKS.

## This increment

`GET /health`, `GET /api/v1/spaces/available` (E-09), `GET /api/v1/spaces/{spaceId}/availability` (E-10). Occupancy and a catalog snapshot are **in-memory**.

**DP-05 still holds:** this slice does **not** HTTP-call `drp-space-api` or `drp-reservation-api`. Corte 3 wires those reads (cache invalidated by events; never a cross-schema `SELECT`). `PAYMENT_PENDING` is not stored here (DEC-001).

```bash
go test ./...
go run ./cmd/api
curl http://localhost:8086/health
```

Child of `develop` named `feat/…`. Promote with `cherry-pick -x`.
