# R096 pre-effect worker source return

Status: SOURCE_READY_PENDING_INDEPENDENT_ADMISSION
Risk: R2
Worker: /root/m2_worker
Reviewer: Codex /root
Role: /root/m2_worker IMPLEMENTATION_WORKER
Date: 2026-10-14

The R096 harness source and compiled test binary are ready for independent source admission. No R096 fixture was created, no provider credential was read, and no provider request was made. The R095 worker pre-provider failure and consumed markers remain intact; its recorded physical attempt count is zero. R096 shares the original owner ceiling of two new physical Alibaba calls across both tranches. The R094 product source remains frozen. This return makes no runtime governance or M2 completion claim.

The supervisor accepts the current `providers.alibaba_maas` credential container, checks its pinned endpoint and region, and takes the requested model only from immutable owner approval. Model presets are observation metadata, including dictionary `id` entries; they cannot substitute or reject the owner model. The signed Go binary validates the parent manifest and known-zero R095 receipt before stdin or effects. Its recorder consumes one permanent parent slot before each physical RoundTrip, forces fresh HTTP/1.1 transport, and captures a bounded partial response with a read error. Parent manifest and old R095 source/receipts/markers are unchanged. The source still requires Root signature and the metadata-only precheck before setup or live use.

Worker Go/test-binary invocations: **1/4 used**. The sole invocation, from `backend/`, was `GOPROXY=off GOSUMDB=off go test -c -o ../.cvf/runtime/r096_live/intervention_r096.test.exe ./api/handlers`; exit 0, empty compiler output. The compiled binary SHA-256 is `dec0ba5c83474b23e0748a03b142616eabb34112af36968581ba16e24f063b5e`. Remaining worker invocations are one seed, one live launch, and one same-scope repair reserve; Root's independent allocation is 3/3. These are process budgets, not provider calls.

Procedural deviation: that one compile command actually ran with the terminal working directory set to `backend/`, contrary to the project-root terminal rule in `AGENTS.md`. It made no source or external effect. Future Go commands will run from the project root with `go -C backend ...`; this compile is not repeated merely to alter its command form.

Final pending-review source SHA-256: Go `37073914e995819415153a9e3f87f362e7a43c74990d2fd03de117f21c9aaeaf`; Python supervisor `5b1dec9ff3ade01cb9ca22e88e557e8b59876488ba76e98fc77c698be4c5969d`; Python tests `1fd61db390928044e69157e6f96e34d2e4f8b267266f659e3670050618cd6b23`. The supervisor's `offline` command checked all twelve frozen R094 source hashes. Root must sign these exact source and binary hashes plus the parent manifest hash before credential precheck or setup.

Offline results: `python -B -m unittest scripts.tests.test_r096_intervention_live_fixture -v` **14/14 PASS**, covering actual current authority, known-zero parent receipt, no-approval binary rejection, current and old credential schemas, model preset observation, inactive phase, sanitizer and duplicate/escaped-key handling, exclusive claims, and usage breach; `python -B scripts/r096_intervention_live_fixture.py offline` PASS; `git diff --check` exited 0 with a line-ending warning for the Root-owned R095 tranche record. `gofmt` completed. No test executed against MySQL or a provider in this return.

Next governed move: Root independently checks source and the exact binary, signs admission, then runs the read-only metadata precheck. The worker may create the one owned fixture and seed only on subsequent Root dispatch. A live worker request remains separately gated; no retry, reset, tuning, or budget increase follows a failure.
