# CCMAI-IDENTITY-001: product identity and Go module

Status: BUILD. Risk: R2 (Go import path and release identity touch build/runtime surfaces).

## INTAKE / DESIGN / SPEC

Owner rejects the current README presentation and wants Customer Care Monitor AI presented as Blackbird081's own product. Public product copy should explain value, use, current limits, and setup without centering the source repository. MIT source copyright and permission notice remain in LICENSE. Keep historical provenance in existing audit records; do not erase license attribution.

The backend Go module lives under `backend/`, so its canonical module path is `github.com/CVF-Ecosystem/Customer-Care-Monitor-AI/backend`. Change the module directive and every in-repo Go import together. Update user-facing docs links and version lookup so the product does not send people to upstream release/docs pages. Prevent the inherited release workflow from publishing upstream Docker images until a product release pipeline is designed.

## Allowed files and evidence

- README, product-facing docs homepage/introduction/footer, relevant frontend help links, backend Go module/imports/version endpoint, inherited release workflow guard, this work order/review, and CVF continuity.
- Tests: `go test ./...`, `go build ./...` from `backend/`, frontend build if Vue help links change, VitePress build if docs change, static checks for stale module imports, CVF doctor/catalog.
- The local Windows Application Control policy may block transient `.test.exe` files. If so, use a Linux GitHub Actions job to complete the same test suite; record the local limit explicitly.
- No database schema or data migration, provider call, secrets, or CVF runtime governance claim.

REVIEW requires checking that old Go import strings are gone, all builds pass, MIT notices remain, and public README no longer reads as a CQA migration note. Independent review remains required before FREEZE.
