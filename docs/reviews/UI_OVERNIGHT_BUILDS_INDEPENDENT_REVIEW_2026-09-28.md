# Independent REVIEW — overnight UI BUILDs (2026-09-28)

**Reviewer:** Codex (`REVIEWER`, independent of Claude's BUILD). **Risk:** R2. **Disposition:** UX-000 `CHANGES_REQUIRED`; UX-002 `REVIEW_PASS`; UX-001a `REVIEW_PASS` for its bounded change, with its UX-000 dependency still open. No FREEZE, push, deployment, live provider call, or claim of CVF runtime governance.

## Scope and checks

Reviewed commits `3dce3e7` through `a4c14be`, the three SPECs/work orders/BUILD records, changed source and tests, UX-000 A1/A2, representative desktop/mobile light/dark captures, and the current API/scheduler paths. Independently ran `npm test -- --run` (7 files, 86 passed), `npm run build` (pass), `git diff 44dceec..HEAD --check` (pass), and `scripts/test-backend.ps1 -Run TestImportDemoData` (handler test pass on disposable MySQL; container/network removed). The workspace doctor passed 25/25 at review intake. UI mock API tests assert structure only; no provider result was used as governance proof.

## UX-000 — CHANGES_REQUIRED (finding UX000-R1)

The shared tokens, font, components, status/confidence/sync wording, self-hosted assets, screenshot A1 capture fix and A2 external DNS block match the bounded SPEC. The synthetic preview and app captures have no reported JS errors. The captured dark-mode app exposes one regression caused by the new global palette: `DefaultLayout.vue` lines 37, 76 and 86 force `text-white` inside avatars filled with the light dark-theme `secondary`/`primary` colors. The white initials have roughly 2.2:1 contrast on dark-theme primary `#9AA6F0`, below the SPEC's existing-view legibility requirement (T2) and the 4.5:1 text target. The UX-000 BUILD record already names this as F2, but deferring it would leave the shared theme in a failing state for every later screen.

**Repair contract UX000-R1:** In `frontend/src/layouts/DefaultLayout.vue`, use the theme's `on-secondary` for the tenant avatar and `on-primary` for both user avatars, or an equivalent tested token-driven foreground. Keep theme persistence and the existing avatar behavior. Amend the UX-000 work-order path allowance for these three color classes before BUILD. Add a focused light/dark contrast assertion for both avatar fill/foreground token pairs and recapture the affected dark layout (desktop and mobile). Rerun frontend build/tests, diff check, catalog and doctor. Return a local repair/evidence commit as `REVIEW_PENDING` for independent re-review; no self-PASS or FREEZE. The recorded F1 onboarding banner remains legible and belongs to the later screen tranche.

## UX-002 — REVIEW_PASS

The local-only note appears immediately before the populated Results list in table and card modes; the table has a visible “Nguồn” header. Loading/empty states do not claim a row source status. The template-only diff stays inside its work order. Its independent UI-structure tests and captures support the claim. No repair finding. This review does not close the tranche at FREEZE.

## UX-001a — REVIEW_PASS, dependency open

Dashboard relative time cannot be negative, the card label truthfully describes the unchanged `issues` value, classification cards count “nhãn”, and Job Detail groups by one Vietnam calendar day. The new demo writer puts synthetic conversations on Vietnam business hours before their demo runs; the focused database test passed independently. No API or running database changed. UX-001a uses UX-000 `format.ts`, so its review pass does not authorize UX-010 BUILD until UX-000-R1 passes. Existing demo rows retain old timestamps until a separately authorized re-import.

## Orchestrator decisions for the next work orders

1. **UX-02 dashboard API:** Preserve the existing `issues` field and its present meaning for clients. Keep the truthful “Kết quả đánh giá” card until a separate R2 backend/API tranche adds a read-only `qc_violation_count` field counting `result_type = 'qc_violation'` in the same tenant/time window. Specify zero/error behavior and tests against mixed QC/classification rows; then the Dashboard screen tranche can display “Vấn đề” from the new field. Do not silently redefine `issues`.
2. **UX-06 demo sync:** Route to a separate R2 runtime work order. Give imported demo channels an explicit, channel-level marker and have scheduled sync skip only marked demo channels, before due-check/decrypt or status mutation. Do not skip all channels of a tenant with `is_demo_data`; a real channel may coexist there. Cover marked/unmarked channels, tenant isolation, manual-sync behavior and no outbound adapter call in tests. The current failed decrypt occurred before outbound traffic; UX-000 A2 now blocks name-based external traffic in the disposable screenshot app.
3. **UX-04 Job Detail:** Accept the UX-010 SPEC's latest-run default, with a specific-prior-run/“Mọi lần chạy” selector and explicit list-count scope. Metric cards stay captioned as latest-run facts; card drill-down selects that run before filtering. The current API returns `job_run_id` on results, so frontend list filtering is feasible. The existing export endpoint always exports all runs: label its button “Xuất mọi lần chạy”; a scoped export needs a separate contract. Keep the fallback in the SPEC as a historical option, not the selected design.

**Next governed move:** Claude repairs UX000-R1 and returns `REVIEW_PENDING`; Codex re-reviews it. UX-010 work order/BUILD wait for that PASS. Runtime and dashboard API follow-ups require their own SPEC/work orders. R001–R009 stay REVIEW PASS / FREEZE open; S1 remains IN_PROGRESS.
