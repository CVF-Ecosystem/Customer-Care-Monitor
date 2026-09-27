# Project Session Memory

Memory class: POINTER_RECORD

This is the project continuity front door. It is CVF-governed project state,
not provider-specific memory and not a chat transcript.

## Startup Order

1. Read `.cvf/manifest.json` and `.cvf/policy.json`.
2. Read `CVF_SESSION/ACTIVE_SESSION_STATE.json`.
3. Read the active handoff named by that state file.
4. Read `IMPLEMENTATION_STATUS.json` and `docs/INDEX.md`.
5. State current mode, active handoff, next allowed move, parked checkpoint,
   and active role before material work.

## Mandatory Continuity Rehydration

Repeat the startup order before material work at every new or resumed
chat/session, after context loss or compaction, at the start of every new
tranche or work order, and whenever responsibility or the active handoff
changes. Read current files again; do not rely on chat history,
provider-local memory, or a declaration from a previous session.

Emit a fresh `CVF Agent Declaration` before the first material action. At a
tranche transition, also record the acknowledgment in the active handoff
before BUILD. If continuity surfaces disagree, stop and report
`BLOCKED_CONTINUITY_DRIFT`.

Active state: `CVF_SESSION/ACTIVE_SESSION_STATE.json`

Initial active handoff: `CVF_SESSION/handoffs/AGENT_HANDOFF_V1_2026-09-26.md`

Current mode: `REVIEW` for `CCMAI-IDENTITY-001`: owner-directed README and Go
module identity correction merged as PR #6. Evidence:
`docs/reviews/PRODUCT_IDENTITY_2026-09-27.md`. Authority:
`docs/work_orders/CCMAI_IDENTITY_001.md`.
The older `CCMAI-DOCS-001` and `CCMAI-CREDIT-001` remain in REVIEW for
independent content/provenance review and inherited usage-flow verification.

`CCMAI-ROADMAP-001` adds a source-grounded proposal for SoT-first local machine gates and AI cost control at `docs/roadmaps/AI_RUNTIME_GATES_AND_EVIDENCE_2026-09-27.md`. TypeSafe/Jev is a source of decision-design patterns only; there is no Jev API/SDK or intermediary-service plan. It remains in REVIEW; no product BUILD or real-provider runtime claim follows from the roadmap.

Owner amendment in REVIEW: `docs/decisions/SOT_FIRST_DATA_FILTERING_PATTERN_2026-09-27.md` proposes a reusable SoT-first filtering contract, with project-specific source adapters/rules and CVF phase, risk, evidence, and independent-review controls for application development. It is a proposal only, not a shared library or CVF SOT3 integration. Independent R2 review remains open.

Current owner-accepted direction (2026-09-27): finish the CSKH evidence-to-intervention workflow before reuse across projects or CVF uplift. Jev-inspired internal filtering complements downstream AI/LLM semantic analysis and response generation; no intermediary Jev service is planned. Quality/customer risk and Vietnamese context are hard acceptance conditions before cost savings. The revised roadmap keeps S0–S7 IDs but brings minimum S2/S3/S5 together before S4 optimization. Orthogonal decision states, owned/deadlined waiting, skipped-case sampling, rule preview and valid-result reuse are proposed; none is claimed implemented. Owner agreement to advisory critique does not close independent review. Direct autonomous replies to customers need a separate product specification and authority.

`CCMAI-DATABASE-001` changes the fresh-install MySQL schema default from `cqa` to `CCMA` and the application database user default to `ccma`; explicit environment overrides remain supported and no data migration is implied. `docs/specs/DATABASE_FILTER_PIPELINE_2026-09-27.md` places cheap SQL candidate selection before Go snapshot/policy/rule/admission and provider resolution. It absorbs projection, bounded batching/cache/spend/observability lessons from `Blackbird081/pg-jev` without adopting PostgreSQL, the pg-jev extension or TypeSafe API. BUILD validation is recorded at `docs/reviews/DATABASE_DEFAULT_AND_FILTER_DESIGN_2026-09-27.md`; the owner-reviewed database setup is complete while the machine gate is still proposal-only.

Docker follow-up (2026-09-27): an isolated fresh MySQL 8 Compose volume created exact schema `CCMA` with the expected `utf8mb4` / `utf8mb4_unicode_ci`; no application schema `cqa` was created. The configured application user passed a disposable DDL/DML probe. All validation project resources and the temporary `.env` were removed. This does not validate migration of an existing database.

Owner-reviewed local setup (2026-09-27): this workspace has no CQA data to preserve or migrate. Git defaults are `Blackbird081 <nmtienctt@gmail.com>`. Compose project `ccma` now keeps the real development volumes, with database `CCMA`, application user `ccma`, generated secrets in ignored `.env`, authenticated healthcheck, and a running backend. AutoMigrate created 16 tables. Manual unique-index creation was made idempotent after restart exposed duplicate-index log errors. The first app start fetched public pricing metadata automatically; no provider credential or customer data was sent, and pricing sync is now opt-in/default-off with the final restart using the static table. `CCMAI-DATABASE-001` needs no separate database review; any future import of external data is a new scope.

`CCMAI-RUNTIME-001` begins roadmap execution with S0 and the first S1 reliability slice. Pancake is the pilot adapter for a complaint/waiting-customer intervention use case, while runtime contracts remain channel-neutral. Synthetic corpus `s0-vi-intervention-v1` covers 12 Vietnamese/context/risk cases; the real development DB baseline has no customer/job/AI rows. Sync item failures now yield observable `partial`, preserve `last_sync_at`, suppress after-sync jobs and appear as warnings in UI; aggregate sync no longer reports false success. BUILD evidence is `docs/reviews/RUNTIME_FOUNDATION_S0_S1_BUILD_2026-09-27.md`. Independent R2 review is open. No provider call or AI-governance runtime claim was made; remaining S1 precedes S2/S3/S5.

`CCMAI-RUNTIME-002` is ready for the owner-selected Claude assignee. Gate A requires Claude to independently review `CCMAI-RUNTIME-001`; only an accepted disposition opens Gate B. Gate B adds a shared single/batch snapshot builder, deterministic source digest, coverage and message-bound evidence refs under `docs/specs/RUNTIME_SNAPSHOT_EVIDENCE_S1_2026-09-27.md`. It does not authorize provider calls, customer data, channel sync, S2 machine gates, deployment or governance/cost-saving claims. Claude must acknowledge roles in the active handoff before BUILD, create local evidence and commit without push, then return Gate B to Codex for independent REVIEW.

`CCMAI-RUNTIME-002` Gate A: Claude independently reviewed `CCMAI-RUNTIME-001` with disposition `PASS_WITH_REPAIRS` (scheduler retry cadence for failed channels, list-view sync wording, reproducible corpus test), recorded at `docs/reviews/CCMAI_RUNTIME_001_INDEPENDENT_REVIEW_2026-09-27.md`; FREEZE is a CLOSER decision. Gate B BUILD adds shared single/batch snapshots (`ccma.snapshot.v1`, SHA-256 digest, typed coverage), message-bound `evidence_refs` validated before a single transaction, nullable `analysis_snapshot_id` with legacy/unverified status, and snapshot deletion alongside results. Evidence: `docs/reviews/RUNTIME_SNAPSHOT_EVIDENCE_S1_BUILD_2026-09-27.md`. Status REVIEW_PENDING for Codex; local commit only, not pushed. Test doubles prove parsing/persistence only; no provider, customer data or governance/cost claim.

`CCMAI-RUNTIME-002` Gate B independent review by Codex is `CHANGES_REQUIRED`: core digest/evidence validation and MySQL transaction tests pass, but channel delete/prune can leave dangling or orphan evidence records, attachment replacement is not fingerprinted by the digest, and the new DB tests retain a hardcoded CQA fallback DSN. Authority and repair acceptance are in `docs/reviews/CCMAI_RUNTIME_002_GATE_B_INDEPENDENT_REVIEW_2026-09-27.md`. Claude may repair within the same R2 work order and local-commit boundary; Gate B remains in REVIEW and S2 remains closed.

`CCMAI-RUNTIME-002` Gate B repair round 1 (Claude, `REPAIR_WORKER`, same-scope, no new authorization): `DeleteChannel` now deletes `JobResult` and `AnalysisSnapshot` inside one transaction with error checks (previously `JobResult` was never deleted there); `ApplyPrunePlan` deletes orphaned snapshots per stale run but keeps any snapshot a surviving result still cites; `classifyAttachments` adds a deterministic attachment fingerprint to the digest manifest so a same-count/same-coverage attachment swap changes the digest; the new DB test's hardcoded CQA fallback DSN is removed in favor of a skip. Three new regression tests added and passing; all 5 pre-existing Gate B DB tests unaffected; full `go test ./...` (13 packages) and workspace doctor (25/25) pass. Evidence: `docs/reviews/RUNTIME_SNAPSHOT_EVIDENCE_S1_REPAIR_2026-09-27.md`. Status remains REVIEW_PENDING for Codex re-review; local commit only, not pushed; no provider call, customer data or governance/cost claim.

Codex re-review of repair commit `f924a6b` remains `CHANGES_REQUIRED_ROUND_2`: prune cleanup and CQA DSN removal pass, while channel cascade still reads IDs outside its transaction/deletes files before commit and attachment fingerprint has a reproducible delimiter collision. Authority and exact repair acceptance: `docs/reviews/CCMAI_RUNTIME_002_GATE_B_REREVIEW_2026-09-27.md`. Gate B remains `REVIEW_PENDING`; Claude may make a same-scope local repair commit and S2 remains closed.

`CCMAI-RUNTIME-002` Gate B repair round 2 (Claude, `REPAIR_WORKER`, same-scope, no new authorization): `DeleteChannel` now reads conversation IDs with a locking (`FOR UPDATE`) read inside the transaction on the same predicate the conversation delete uses, closing the insert-during-cascade race via InnoDB gap locks, and checks that read's error; attachment-file cleanup moved to after commit and is log-only on failure. `classifyAttachments` now hashes `json.Marshal` of the typed attachment slice instead of a hand-joined delimited string, closing a reproduced delimiter collision, and hashes untrimmed raw bytes for invalid JSON. New regression tests: a MySQL-trigger-forced mid-cascade failure test proving full rollback, and three new attachment-fingerprint cases (collision pair now differs, whitespace/key-order invariance holds, invalid-JSON whitespace change still moves the digest). All 13 packages pass; workspace doctor 25/25. Evidence: `docs/reviews/RUNTIME_SNAPSHOT_EVIDENCE_S1_REPAIR_ROUND2_2026-09-27.md`. Status remains REVIEW_PENDING for Codex re-review; local commit only, not pushed; no provider call, customer data or governance/cost claim.

## Local Provider Registry

`CVF_SESSION/LOCAL_PROVIDER_SECRETS.json` — **gitignored, machine-local** — chứa API keys của các AI provider thực, dùng khi work order yêu cầu `liveGovernanceEvidenceRequired: true`. Mọi agent cần credential thực đều đọc file này. KHÔNG commit file này.

Provider hiện có (ghi nhận 2026-09-27):
- **alibaba_maas** — Alibaba Cloud MaaS workspace `ws-remplsp27g5oicq1`, OpenAI-compatible. Model khuyến nghị: `deepseek-v4.1-flash` (quota 1M, hết hạn 2026-12-12). Xem catalog đầy đủ tại `docs/references/ALIBABA_MAAS_PROVIDER.md`.

Provider-local files may assist execution but are not project source authority.
