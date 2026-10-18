# Desktop execution capability and minimal M3 transfer proposal

Status: DRAFT_PENDING_ORCHESTRATOR_REVIEW
Date: 2026-10-18
Risk: R2
Worker: Current Desktop investigation agent, acting only as transfer worker
Reviewer: Existing ORCHESTRATOR / independent REVIEWER; review NOT_PERFORMED here
Base: 9a8c917d6383fb797a5daac1340318b99a5ef4f2
Authority: Owner's read-only investigation and proposal request; no BUILD/runtime grant

## Capability result

SUPPORTED_SESSION_API_AVAILABLE; CAMPAIGN_LIFETIME_NOT_VERIFIED.

The tools exposed in this Desktop chat support `exec_command` returning a running
`session_id`, followed by `write_stdin` on that same ID returning output and the
actual command exit code. The prerequisite doctor provided a narrow observation:
exec returned session 49236 after 10.016 seconds; a subsequent write_stdin on
49236 returned exit_code 0 and doctor PASS 27/27. This was a workspace check,
not a supervisor/child shutdown experiment. No product process was started.

The official [Codex Prompting Guide](https://developers.openai.com/cookbook/examples/gpt-5/codex_prompting_guide)
also describes exec_command for long-lived terminal sessions and write_stdin for
feeding or polling an existing session. Neither that description nor the current
tool schema establishes retention through app restart, interrupted turns,
compaction, or the full 1200-second campaign. A tool session ID is not an OS
process handle held by this investigator. No alternative persistent-process API
was established in this investigation.

The existing R120 source holds subprocess.Popen objects and waits on its children
inside finally. Reuse that design only after separate admission; do not replace
it with a new supervisor. The outer exec must remain attached to the foreground
supervisor/wrapper until its real exit; a completed stop command is insufficient.
No new claim is made about what caused the old supervisor to disappear.

## Smallest proposed admission check — not executed or authorized

Before admitting any product runtime, ORCHESTRATOR may grant one harmless local
lifecycle check using the existing exec_command/write_stdin tools and installed
PowerShell. One parent, one short child, one cooperative file stop request,
one distinct new scratch directory under `.cvf/runtime/<new-id>/capability/`;
no listener, Docker, database, browser, provider, secret access or installation.
Wall cap 90 seconds, one invocation, zero retries. Use explicit finite child and
parent timeouts so loss of tool access cannot leave an unbounded dummy process.

The short child observes only its new stop sentinel, writes its own end marker
and exits. The parent retains its original process object, waits for that exit,
writes a receipt with PID/creation identity/start/end/observed child exit, then
exits with a predeclared nonzero discriminator (for example 17). Preserve raw
exec and write_stdin returns, stdout/stderr and hashes in a fresh proposal-owned
evidence directory. The investigator must receive exit_code 17 from the ORIGINAL
exec session, matching the parent's receipt and child's marker. A separate stop
command's zero exit is not acceptance. This is a disposable capability check,
not a supervisor replacement or proof of 1200-second retention.

Loss of session, mismatched exit, missing receipt, timeout or unsupported stop
mechanism ends the route: report HOLD, no product start, retry or stack change.
Even success only supports the observed short uninterrupted lifecycle; the
reviewer must explicitly accept its applicability before any longer campaign.

## Conditional product execution proposal — separate new authority required

Purpose: collect a NEW run's original supervisor/child terminal and final durable
accounting alongside a minimal read-only current-proposal/history observation.
It cannot recover, replace or close the missing ORIGINAL R120 records. Inherit
the accepted R120 UI observations and R117/R118 source evidence; no lifecycle
action, AI rerun, B07 repeat, offline experiment or new mobile-raster claim.

The old runner is not directly reusable: it pins r120 runtime/evidence paths,
resource identity and tranche, and the Go test validates the exact admission
path. ORCHESTRATOR must issue a new ID and bound any mechanical isolation copy
before BUILD: a new supervisor copy, matching Vite template, matching synthetic
Go fixture and focused supervisor tests. Limit changes to new identity/path/env/
symbol bindings and exact admission checks; retain lifecycle and product logic.
If that cannot be done mechanically, stop for a decision, not redesign. Old R120
files, admissions, archive, ledgers, resources and counters are immutable.

Suggested maximum NEW budgets for dispatcher consideration, never grants here:

- Source phase: worker one focused test invocation; independent reviewer one;
  offline Go compile one; no automatic repair/retry allowance.
- Runtime: one isolated synthetic fixture/network/volume, one setup, one backend,
  one Vite, one owned browser tab, one login and one logout (two auth POSTs), one
  cooperative stop, one exact owned fixture stop including failure cleanup.
  Provider calls, case mutations and service restarts: zero.
- Campaign total ceiling 1200 seconds including cleanup; retain the accepted
  180-second cleanup reserve. Target UI work within 180 seconds of READY, request
  stop immediately after logout/tab cleanup. No extension for missing evidence.
- Keep the original foreground exec session ID, poll it at bounded intervals
  through browser work and shutdown, then await its actual exit. Tool waits at
  most 30 seconds each; no end-of-turn handoff or intentional compaction while
  running. Record session availability at each poll. Polling is a procedure,
  not a guarantee against the UNKNOWN previous failure.

Before startup: independent source review, committed dispatcher seed, exact
source/binary/tool hashes and new resource admission; verify supported browser
and Docker capability then. Existing exited R120 resources stay preserved.
If prerequisites fail, do not start. If capability disappears after startup,
stop observation; use only the new order's preauthorized exact-ownership cleanup
budget, report HOLD, never restart or switch execution stacks.

Required return: original exec terminal exit code and raw output, wrapper exit
receipt written after actual foreground child return, terminal.json, each held
child's observed wait/exit and creation identity, backend final fingerprint and
effect counters validated against its NEW baseline, route counts, source pins,
listener absence and protected-inventory equality. Preserve raw bytes and hashes.
Reviewer decides disposition; worker does not accept its own run or freeze M3.

## Transfer boundary

Recommend HOLD for product runtime until the supported session path is admitted.
This proposal adds no authority seed, tranche, runtime budget, source repair,
commit, provider call or milestone acceptance. R120 stays CHANGES_REQUIRED /
FREEZE OPEN; disappearance cause UNKNOWN. Full M3 and other milestone exits stay
open. Facebook/Zalo OA remain parked; no real data, push, merge or deployment.

Current doctor PASS concerns static workspace checks only; manifest pin
e9ddcbcdcac2a9df520f5eb0761dbdb609b9deba differs from clean public Core
fa34ed49e56648acafe8b0388e48e0aa0523f007 (warning). Knowledge ingest was local.
No product tests or capability experiment were run. No independent review or
pre-commit gates are claimed for this uncommitted worker proposal.
