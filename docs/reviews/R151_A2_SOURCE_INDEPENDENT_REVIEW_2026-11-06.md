# R151 A2 exact-source independent review

Status: BLOCKED
Date: 2026-11-06
Risk: R2
Reviewer: Astra Medium independent R2 REVIEWER, /root/r151_expired_review
Worker: Terra 6.1 IMPLEMENTATION_WORKER
Commit steward: Root

## Decision

BLOCKED: retain the worker's BLOCKED_WITH_REASON return. The corrected cleanup function and final-byte syntax have useful evidence, and this review supplies the permitted exact-source V06 static path review. However, V04's required direct actual-runner timeout/next-launch integration remains unproved. The exhausted allowance does not lower that requirement. No source acceptance or Windows effect admission follows.

Reviewed the entire runner/test diff against `bbc4e155917ae26c4ebe81b062d26375e63b1e1a`, current runner, unchanged helper/actor/PowerShell paths, candidate manifest, retained logs and worker return. The accepted A2 packet LF hash remains `57a76055fc62aa74ef864ae4972140f6940ccee7553abf61d310b4d1917d8d91`.

## Exact evidence identity

All seven current raw source hashes match `docs/reviews/probes/r151_evidence_capture_20261105/worker_a2_source_manifest.json`. In the table and subsequent line references, files are under that probe directory unless otherwise stated.

| Artifact | Raw SHA-256 |
| --- | --- |
| owner_runner.mjs | `0714964711391772de75f151ed97e7d27ba3f61c02f4056b3e535604b88ffcca` |
| worker_offline_tests.mjs | `57370533b859c86cce93142b26bf4ca897d74a2c60bb7827a0e91ab7a266b740` |
| diagnostic_helpers.mjs | `fe1f1bb92e63c5c3b8048949b9af4c957a340fcb5e105c45ff81e60c9e6b8df0` |
| actor_call.mjs | `e8809991c2d26a0ff8dc83665cc4ac93aa199221164ae3d4665a26f30bb7a8c1` |
| process_snapshot.ps1 | `50b99cdbb0f7fd621edc5fe493bacd65e0dd0b56d8f5722ba4cf42a54f81d939` |
| prestart_full_tree.ps1 | `8b1dec9c5f945f988b2a32a4804559c1ce44cf20225c902bd19985ed2bbd4db2` |
| worker_historical_red.ps1 | `8a4de27d8b19445216cb6a3ac71ef01fc6e04192509cd5f7b8e83ca209b7fa98` |
| worker_a2_source_manifest.json | `80c51918248391b333b5aff9ee023eb17b940eb430b832ad4a2689b897fee523` |
| worker_a2_offline_test.log | `11204b544f49a5721834edd0fb58251dc15a2aec6fc967779a057671c95a3c50` |
| worker_a2_syntax.log | `775cd3f75f05a80820b4b13693fb84cf7900fc15393d4c1abaf200020cfe229f` |

The test log contains 52 PASS lines and TOTAL 52 PASS; SKIPS 0. The worker return records command `node docs/reviews/probes/r151_evidence_capture_20261105/worker_offline_tests.mjs` and exit 0. The raw test log itself does not include an exit-code footer; exit 0 is worker-reported, not independently rerun here. The syntax log explicitly records Node --check exits 0 for runner/helper/tests, PowerShell ParseFile zero errors for the three probe scripts, and BATCH EXIT=0. Unchanged actor syntax is inherited; the changed runner and tests have final-byte syntax evidence.

Observed filesystem times: manifest 07:45:45, test log 07:46:01, syntax log 07:46:31 on Nov6 +07. The worker reports no source changes after the manifest/checks and stop by 07:46:40. Matching current hashes support retained candidate identity; filesystem times are not independent process supervision. Only runner/test among tracked probe sources differ from the committed baseline. Old logs/manifests and the historical red result (wrapped_count=1, match_count=0, exit1) remain retained.

## Consolidated finding and bounded improvement

**F01 — blocking verification gap, V04:** `owner_runner.mjs:13` exports `runReplacement(options)` as a one-line call to the unchanged `replaceOwnedChrome` helper. The new test at `worker_offline_tests.mjs:98` supplies its own close callback: it waits through custody and manually throws on timeout. That callback substitutes for production `closeHandle` at runner238-247, including its timed-out identity guard, original kill/record and waitExit connection. Its launch counter also substitutes for the complete production call at runner372, where debit/startProcess/profile assignment are wired. The private main body is never executed by the suite. Thus the new test proves wrapper/helper behavior given a test-authored timeout rejection, not that the actual runner supplies that rejection and prevents its next launch. This repeats the specific integration-proof gap identified before A2. A production change removing the timeout rejection from closeHandle would not invalidate this fixture's own rejection.

The packet expressly requires actual runner cleanup and launch orchestration under injected handles/custody/launch counters. Static inspection suggests the production close path currently throws on timeout, but static inference is not the admitted replacement for this V04 injected proof. A future bounded correction, if separately authorized, must expose and execute the production close/wait/replacement wiring with low-level effects injected, rather than reproducing the relevant decision in the fixture. No retry or further source change is granted here.

**F02 — remaining evidence limit, not a reproduced failure:** new tests exercise runCleanup but do not execute main's final terminal reduction and persistence at runner433-441. The custody-failure fixture supplies a successful event collector for cleanup persistence; it does not force failure of the cleanup-terminal record or final save. Static source now catches terminal record errors and final-save failure and downgrades stdout outcome. That is useful source evidence, not executed proof of final persisted-terminal behavior. Raw intermediate snapshots must not be mistaken for a final accepted terminal receipt. Preserve this limit in any later source/effect decision.

The original cleanup-bypass finding is materially improved: runCleanup21-45 separately guards daemon stop/wait/kill, Chrome close, terminal inventory and persistence. The new custody-failure fixture executes this actual function with throwing custody persistence and observes one later Chrome-close attempt and one snapshot, retained error and UNKNOWN terminal. The unknown-descendant fixture also executes actual runCleanup and rejects terminal acceptance. This is not a wholly parallel cleanup model. The failure is the remaining production replacement integration, not denial of this demonstrated improvement.

## V01-V07 assessment and exact-source V06 review

| Obligation | Evidence and disposition |
| --- | --- |
| V01 | Retained historical red plus current actual PowerShell fixture zero/one/two and invalid-profile cases. Bounded fixture evidence retained; no new historical rerun. |
| V02 | Current exact argument fixtures cover prefix/unrelated text/quotes/case/separate argument/unknown command. Source unchanged and manifest-matched. Bounded fixture evidence retained. |
| V03 | Current ledger fixtures cover absent/false root, foreign profile, unknown inventory/identity, descendants/orphan retention and PID reuse. Bounded fixture evidence retained. |
| V04 | PARTIAL/BLOCKED: actual cleanup/custody-failure and unknown-terminal fixtures improve evidence; production replacement close/launch integration is not executed, as F01 details. |
| V05 | Actual unchanged HTTP helper parser and injected negativeProbe transport fixtures retained, with malformed/oversized/late/reset/timeout classifications. No TCP/Chrome result inferred. |
| V06 | Source-path preservation reviewed independently below on the final hashes, as the packet permits. This discharges the worker's request for static review of these preserved controls at source scope, not V04 or live A1/A2. |
| V07 | Current seven-file identity, final syntax, command/skip results and history retained. Exit/timing/effect provenance is qualified above. Overall completion remains blocked by V04. |

V06 static conclusions bind the runner/actor/PowerShell raw hashes in the table:

- **Positive control:** runner276-302 debits handshake, uses the supplied endpoint, sends Browser.getVersion on open, correlates reply id1 and settles once under a timer. Callers353-354,367-368,384-385 require opened plus result before continuation. This code is unchanged except its enclosure in main; token presence was not used as the proof.
- **A3:** runner341-350 invokes the fixed prestart script, records it, maps nonzero to NO_START and enters first Chrome launch only on success. Unchanged prestart_full_tree.ps1 pins paths/manifest, rejects reparses, hashes the tree, and at49 requires the explicitly inherited 347-file/README-missing/package-hash condition;62 returns exit0 or2. This preserves the admitted exception and is not a 348/348 equality claim.
- **A1-c:** runner347-349 captures one port for both profile argument sets;372 requests owned replacement,373-374 requires a changed endpoint;377 probes old endpoint1 with Chrome2 ownership checks;379-385 classifies refusal/failure and requires the new endpoint2 positive control before daemon start. Static sequencing is retained; the replacement timeout obligation remains F01.
- **A2:** runner404-423 requires a selected about:blank integer page id and full initial inventory, executes missing-pageId snapshot, compares subsequent inventory, requires the expected validation error class, then permits the valid-pageId control. Unchanged actor_call.mjs9-20 constructs missing `{}` versus valid `{pageId}`;27 makes one sendCommand with no ensure/start fallback. These are source contracts only, not observed native rejection behavior.
- **Import/effects:** all previous executable runner setup is inside main; line445 calls it only at direct CLI entry. The tests import actual exported seams. Admission/source-manifest/registry/gate/committed-path guards remain inside main before runtime setup. Fixture PowerShell uses supplied rows. No newly executed native actor or network path is present in the fixture source. Zero prohibited effects remains worker-reported with this source corroboration, not an OS-wide independent absence proof.

## Cost, continuity and next move

This A2 allocation consumed its sole focused invocation and sole syntax batch. Cumulative focused invocations are five; total evidenced syntax batches are two, comprising one historical plus one new A2 batch. The unused old A1 syntax allowance remains expired. Old Nov5 stop, earlier active minutes UNKNOWN and all prior attempts remain. The reported A2 interval from release07:42:44 to stop by07:46:40 is under four elapsed minutes, before conservative08:02:44 and wall08:12:44. Time remaining does not replenish spent test/source-pass authority.

Root may retain the blocked result, reconcile canonical state and causal assessment, and decide a separately authorized bounded next move. Do not mark source accepted, dispatch Windows effects, or silently relax direct V04 proof. This is incomplete implementation/verification; no completed product repair failure or automatic budget reset is inferred.

CVF Agent Declaration: Customer-Care-Monitor-AI; Core `../.Controlled-Vibe-Framework-CVF` at `c4c8767ac8861854f40d466ba338ffd11eb9558e`; independent REVIEWER, R2 ceiling, live governance evidence required YES. Bootstrap/state/memory/handoff/implementation truth/index were rehydrated; they still show the timed A2 BUILD/ACK while the worker has returned blocked. Root owns the ensuing continuity transition. No gates were rerun; Root-reported worker-return/preflight passes are static checks and do not resolve F01.

Only this review artifact was written. No tests, syntax checks, source edits, runtime effects or commits were performed. R139/M3 remain blocked; R150 INCONCLUSIVE/A2 UNRUN/cleanup UNKNOWN and Facebook/Zalo OA parking remain. No source-only acceptance, product/runtime governance, hosted CI, deployment or release claim follows.

## Public Export Disposition

DEFERRED_PRIVATE_ONLY
