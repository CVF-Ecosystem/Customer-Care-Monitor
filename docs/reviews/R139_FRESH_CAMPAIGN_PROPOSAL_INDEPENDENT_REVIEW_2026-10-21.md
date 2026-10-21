# R139 fresh-campaign combined proposal: independent documentary review

Status: DESIGN_READY_FOR_OWNER_REVIEW_ONLY
Risk: R2
Worker: Root ORCHESTRATOR / SPEC_AUTHOR (documentary proposal only)
Reviewer: Sol Medium, independent of Root as proposal author
Baseline: `bb7ded1b5b5b2fca410a50a4144b25520e3da32e`
Reviewed: `R139_FRESH_CAMPAIGN_COMBINED_PROPOSAL_2026-10-21.md`, `probes/r139_integrated_20261021/r5_fresh_campaign_proposal.json`, and `r5_fresh_campaign_source_proposal.patch`

This is a documentary design verdict only. It grants no source edit, compile, test, key read, fixture, Docker, browser, network, provider call, budget, runtime admission, or governance claim. R139 integrated execution remains HOLD; the old API slot is consumed as `MAY_HAVE_STARTED`, with actual response, usage, and billing unknown.

The proposal is bounded and reviewable as one conditional approval packet. The patch contains exactly seven literal changes in the three named source files: five fresh r3 identity/path bindings and two Python test literals. The three current source SHA-256 values match the JSON baseline. The corrected patch SHA-256 is `36571a2ff9dcd06f10014f71f86c611b5c95972b76c10f387b4d3c4415b3063a`, matches the JSON, and default `git apply --check` exits 0 against the baseline. The initial documentary patch failed the default apply check due to its line endings; Root corrected the patch bytes and pin before this verdict. No source was changed during this review.

The source/check stage has one offline compile, one selected six-test pure-Go IO invocation, one worker Python suite, and one independent Root Python suite, each with no retry. The runtime stage is dependent on source PASS, independent review, a new committed exact-pin admission, fresh resources, and a valid deadline. Its proposed limits include one integrated r3 campaign, native eight execute POSTs and one browser execute POST, three grants, four auth POSTs, one new physical synthetic provider call with no retry, and a separately proposed USD 1 ceiling. The prior USD 1 ceiling and consumed slot remain historical; the old bill is unknown. A new slot marker must consume the new 1/1 at `MAY_HAVE_STARTED`, including on uncertain transport. First failure or mismatch stops the dependent chain.

The packet now gates runtime on one proposed secret-free official public pricing/model-page read after source acceptance and future Owner grant, before key or fixture. A provider/model mismatch, unavailable tariff, or inability to bound this request within the new USD 1 ceiling yields HOLD without a key, fixture, or paid call. This is a proposed future network read, not a read performed in this review. The patch application contract uses verified committed LF git-blob bytes on Windows and requires default `git apply --check` without a whitespace-ignore workaround.

Qualifications for any Owner decision and later independent runtime review:

- A successful source/check round proves only the changed source and pure IO behavior. The prior MANUAL_HOLD, missing wire/terminal proof, and cleanup HOLD remain until a fresh observed runtime receipt resolves them. The finalizer may preserve partial facts on ordinary fatal paths, but forced termination can bypass Go `t.Cleanup`; missing terminal evidence remains HOLD.
- The supervisor accepts only the scoped `Up ...` or `Exited ...` elapsed Status forms with matching State. Unknown, restarting, dead, missing, or contradictory inventory formats must HOLD; they cannot be normalized into a cleanup PASS.
- One positive campaign cannot cover E04 crash, storage, concurrency, 5xx, or offline branches, or automatically close full M3, M1/M2, pilot, release, or hosted CI. G1/G2/cleanup may each PASS only on their stated observed receipts; a provider response or publication cannot be inferred from a slot marker or request journal alone.
- The runtime reviewer must be a person/agent independent of the source worker, Root's admission authorship and execution, and this documentary review where those responsibilities overlap. This verdict does not substitute for that later review.

No source, test, key, DB, Docker, browser, network, or provider operation was invoked for this review. Static reads, source hash checks, and `git apply --check` were the only checks. Owner approval, if granted, still requires fresh committed source and runtime authority with the stated caps and stop conditions before any effect.
