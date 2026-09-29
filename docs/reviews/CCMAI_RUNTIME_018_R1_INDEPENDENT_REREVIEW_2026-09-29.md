# CCMAI-RUNTIME-018 R018-R1 — independent R2 re-review

**Date:** 2026-09-29 · **Reviewer:** Codex, independent of Claude's repair · **Repair diff:** `947bb0c..195c265` · **Disposition:** `REVIEW_PASS / FREEZE_OPEN`.

## Finding resolution

The repair changes only `backend/db/mysql.go` and `backend/db/demo_fixture_migration_test.go` among source/test files, within the R018-R1 work-order scope. The legacy backfill now uses `BINARY` for both channel type and external ID in each historical identity, and for the error status and `decrypt failed:` prefix in the guarded cleanup. The credential comparison remains binary; no schema, global collation, fresh import or runtime admission path changed.

Five new migration-matrix rows cover wrong-case Zalo type, wrong-case Zalo external ID, wrong-case Facebook identity, wrong-case `Error` status and wrong-case `Decrypt failed:` prefix. The first three stay unmarked; the last two are marked but retain their status/error. Canonical fixtures still mark and clear only the guarded old decrypt error; the test repeats `AutoMigrate` and checks the second pass is identical. Claude's evidence records M12–M15, each removing one `BINARY` check and causing the focused test to fail, then restoring the source.

## Independent checks

- Reviewed the exact repair diff against the prior [R018 finding](CCMAI_RUNTIME_018_INDEPENDENT_REVIEW_2026-09-29.md), SPEC and work-order addendum. No source path outside the allowed repair scope changed.
- On disposable MySQL, verified the operator semantics used by the repair: `BINARY 'Demo-Zalo-OA' = 'demo-zalo-oa'`, `BINARY 'Decrypt failed: x' LIKE 'decrypt failed:%'` and `BINARY 'Error' = 'error'` each returned `0`; the canonical type comparison returned `1`.
- Independently ran `powershell -ExecutionPolicy Bypass -File scripts/test-backend.ps1 -Packages './db' -Run 'DemoFixture|LegacyDemo'`: PASS (`backend/db` 1.146 s); the script removed its disposable MySQL and network. Claude's repair evidence also records a full backend run of 14 packages, `go build ./...`, `go vet ./...`, catalog and doctor PASS; this re-review independently reran only the focused DB tests plus catalog and doctor.

The blocking case-insensitive match/cleanup finding is resolved. R018 now passes independent REVIEW; FREEZE remains open and S1 remains IN_PROGRESS. No real provider/channel call, persistent Compose database, push, deployment or live CVF governance claim occurred. Older binaries still ignore the marker; mixed-version rollout remains unsupported. Zalo/legacy crash recovery and UX-016 F1 remain separate.
