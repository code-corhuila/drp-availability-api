# drp-availability-api

Availability read-model API (Go). **Hexagonal:** `internal/domain` has no web/DB imports. DDL in [`drp-availability-db`](https://github.com/code-corhuila/drp-availability-db).

This increment is **Occupancy** (`BLOCK | CONFIRMED`). Catalog SoT is space; CONFIRMED overlap SoT is reservation. HTTP search comes later.

```bash
go test ./...
```

Child of `develop` named `feat/…`. Promote with `cherry-pick -x`.
