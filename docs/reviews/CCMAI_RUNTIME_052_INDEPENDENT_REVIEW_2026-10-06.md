# R052 independent R051/R3 evidence review

Date: 2026-10-06. Codex `/root` independent REVIEWER, separate from worker `/root/r051_preparation`. **CHANGES_REQUIRED / REVIEW / FREEZE_OPEN**. Exact evidence17d60560cc28a3c7dce9351355cc46db8696d393, handbackd9e8e8f3c7cddd3690715e27ad05663eb4e7f479, before-edit acknowledgment423b3e2bd8bdc14648a8206522b0109b7f224fe3. This completes review of failed evidence, not closure or a worker retry.

## R052-R3-01 — Archive manifest comparator prevents campaign

Runner builds its expected204-file manifest from raw Git blobs, then compares archive-extracted bytes. On this Windows checkout the configured archive exports three files with CRLF while raw blobs use LF: backend/go.mod, backend/go.sum, backend/engine/testdata/s0_vietnamese_intervention_corpus.json. The comparator fails before cached checks, controlled throw, Docker creation or Go. Use exact archive-member bytes as the strict expected extraction manifest; keep raw-blob representation differences separately. Do not normalize runtime source as a substitute for exact archive identity, change Git configuration, or weaken membership/hash checks.

Independent Git archive has SHA256 b4d4daff222388d1f721294b0d2ab47899bb70943966bff1742b7c045054be38 and204 members, exactly matching the retained extracted manifest. Missing/extra files0; every extracted byte equals its archive member. Raw-blob manifest differs on precisely the three named paths, each solely LF/CRLF as independently checked. This is a newly demonstrated comparator implementation root cause, distinct from prior wrong-SHA/default-network/missing-restoration observations. It does not erase the repeated RP-04 history or authorize an automatic fourth repair.

## Evidence and limits

[Independent summary](probes/r052_independent_summary.json) verifies all evidence paths and committed artifact hashes against the later handback, exact archive/source identities and physical hashes of649 protected worktree files. Source145bd411 and production727d322, original seeds/tests/tooling/workflows and historical packets are unchanged. New root review performs no Go/provider/channel/runtime campaign.

Worker first invocation exit1,13.237s, campaigns0/Go0/resources0. FAILED_NOT_RUN is accurate; compile/vet, controlled throw, mutation and restored LP06 are NOT RUN. Publication gates assess only the failed packet and cannot settle RP-04. npm.ps1 launch and VitePress dead-link failures are retained with later publication success; reviewer does not recertify missing runtime proof. Existing RP-01/02/03 and42-top-level/98-event review remain inherited at145bd411. R050/R051 remain CHANGES_REQUIRED, residual R051-R2-01/R050-R1-03 open.

Fresh manifest/policy/canonical continuity/handoff/status/index rehydrated; doctor PASS WITH NOTE25/1, actual readonly core8a4119e11db00e774ed8e7cf7d9a8caa309e81d1 equals origin/main, manifest26c686cc mismatch warn-only; BOOTSTRAP_MIGRATION_PENDING nonblocking. Root REVIEWER -> ORCHESTRATOR / WORK_ORDER_AUTHOR -> SESSION_SYNC_STEWARD / planning COMMIT_STEWARD only for independent disposition and separate cost planning. No worker self-approval, FREEZE, external/provider/channel/persistent data/push/merge/deploy or live governance/hosted readiness claim. Accounts parked. A separate recorded cost disposition is required before correction.
