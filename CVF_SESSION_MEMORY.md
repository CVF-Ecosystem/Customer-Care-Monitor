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

Provider-local files may assist execution but are not project source authority.
