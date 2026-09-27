# Local demo-brand cleanup: CCMAI-DEMO-BRAND-001

**Operator/reviewer:** Codex (`IMPLEMENTATION_WORKER -> REVIEWER`, R1 synthetic content) · **Date:** 2026-09-27 · **Disposition:** source and current `localhost:8088` demo data PASS; no deployment/push.

## Source and storage trace

`ImportDemoData` in `backend/api/handlers/demo.go` generates the coffee-shop sample; the Results UI reads imported rows from MySQL, not from a frontend mock array. The current `localhost:8088` stack uses `ccma-db-1` with tenant `single-workspace` (`Blackbird`). The tenant has `is_demo_data=true`, two channels with `demo-*` external IDs, 2 jobs, 1,454 messages and 3 activity logs. The separate `ccma-uxreview-db` also initially had the same demo data, but its container was removed by another process before this cleanup; it was not recreated or modified.

Before the database edit, a read-only search of every char/varchar/text/JSON column found exact `SePay Coffee` matches in `activity_logs.detail` (2 rows), `channels.name` (2), `jobs.rules_content` (1), `messages.sender_name` (727) and `messages.content` (103). Related sample tokens occurred in message content: `SEPAY50` (12), `SePayCoffee_Guest` (20) and `sepay2024` (20).

## Change and verification

The demo generator now uses `Cà Phê Mẫu`; its sample voucher/Wi-Fi tokens use `CAFEMAU50`, `CafeMau_Guest` and `cafemau2024`. The channel-name example and demo-data documentation were updated. Legal and source-provenance attribution to SePay was left in place.

After confirming the demo flag and channel IDs, one transaction in `ccma-db-1` replaced only matching text for tenant `single-workspace`. Rows changed: channels 2, jobs 1, sender names 727, message content 135, activity details 2. Post-check: two identical channel IDs with names `Cà Phê Mẫu Facebook` and `Cà Phê Mẫu Zalo OA`; 2 channels, 2 jobs, 1,454 messages and 3 activity logs remain; demo flag remains true. A second scan of all text/JSON columns found zero `SePay` matches in this database. Source search found no old brand/tokens in `demo.go` or the updated channel example. `go build ./api/handlers` and `git diff --check` passed.

Changing current demo messages after analysis can correctly make prior snapshot-bound result status `changed_since_analysis`; no snapshot, saved verdict or evidence was rewritten to hide that. The running app image was not rebuilt: current imported data is clean after refresh, while a reset/reimport using the old running image could recreate the former wording until the changed source is deployed. No real channel, provider, customer data, CVF governance proof or app restart was involved.
