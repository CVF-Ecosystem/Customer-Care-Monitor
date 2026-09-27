# Work order CCMAI-DEMO-BRAND-001 — remove inherited demo brand

**State:** `REVIEW_PASS / DEPLOY_OPEN` for source and current `localhost:8088` demo data · **Risk:** R1 (local synthetic demo content) · **Authority:** owner's request to remove all visible “SePay Coffee” from mock data · **Implementer/reviewer:** Codex, with an explicit role transition and [evidence](../reviews/CCMAI_DEMO_BRAND_001_LOCAL_CLEANUP_2026-09-27.md). This work is separate from `CCMAI-RUNTIME-008`, which remains assigned to Claude.

## Contract and scope

Replace the inherited coffee-shop brand in the demo generator and in the two known local demo databases with neutral “Cà Phê Mẫu”. Include brand-derived sample Wi-Fi and voucher tokens. Preserve legal/source attribution to SePay in LICENSE, provenance records and CVF handoffs. Do not delete demo rows, reset a tenant, change user/workspace identity, invoke adapters/providers or alter analysis results/snapshots.

Allowed source paths: `backend/api/handlers/demo.go`, the inherited example in `docs/usage/channels.md`, and a storage-location note in `docs/admin/demo-data.md`. Allowed local DB targets: `ccma-db-1` (the `localhost:8088` stack) and `ccma-uxreview-db`, each only after verifying tenant `single-workspace` has `settings.is_demo_data=true` and channels use `demo-*` external IDs. The UX-review database was removed externally before mutation, so only `ccma-db-1` was updated. Update only matching text fields in `channels.name`, `jobs.rules_content`, `messages.sender_name`, `messages.content`, and `activity_logs.detail`, within a transaction per DB. No other container/database may be changed. No application restart/deployment is authorized by this work order.

## Evidence and exit

Record pre/post matching counts for the exact phrase and related brand tokens, affected-row counts, and the fact that snapshot-bound results can become stale after message-text edits. Check that the two DBs retain the same row counts, demo flag and channel IDs. Check source search for the phrase, Go build or focused test, catalog and workspace doctor. Keep R008 state/handoff pointer intact and exclude unrelated `.gitignore`, `docs/references/` and generated knowledge index from the commit.

After source and DB checks, record review evidence, synchronize this work in the handoff and memory, and create one local commit without push. If a target is not confirmed synthetic demo data, stop before its update and report the boundary.
