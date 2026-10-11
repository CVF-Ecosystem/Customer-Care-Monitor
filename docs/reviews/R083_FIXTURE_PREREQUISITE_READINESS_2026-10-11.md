# R083 disposable fixture prerequisite readiness

Status: source packet prepared for independent review. This record covers the local disposable-fixture tool and its portable unit tests only. It does not claim that the MySQL campaign ran, that product behavior passed, or that CVF governed an AI or agent decision.

## Authority and source binding

- Tranche: `CCMAI-RUNTIME-083` (R083), risk ceiling R2.
- Seed: `ea73a86e5ebb44d2c7ab6496d43c13d55b1610ca`.
- Activation: `eba77516d22f16df2a8bb0739a2cff24fcfb167a`.
- BUILD order snapshot: `c23d84e0048218e1d91d93e9b6557345e3ec2690`.
- The tool records the invoked source commit, physical hashes and Git blob IDs for its source packet, plus the separately pinned BUILD-order snapshot. At invocation it requires committed source bytes and a source commit descended from that snapshot.

## Bounded fixture behavior

The tool accepts one role (`worker` or `reviewer`) and creates a fresh random task namespace beneath that role's fixed evidence directory. It refuses TCP/SSH Docker endpoints, checks only selected local endpoint and engine metadata, uses the cached `mysql:8.0` image by local image ID, and never requests a pull. Its disposable container uses one internal network, no host ports or binds, and a bounded tmpfs for MySQL data. It creates one alphanumeric task schema and one task principal, verifies literal-safe schema grants, runs the bounded DDL/DML probe, drops the task objects, then removes and verifies only resources whose exact IDs and task labels match.

The tool records bounded command labels, exit codes, timeouts, elapsed time, and redacted-output digests. It does not serialize command arguments or generated credentials. Failure to verify a resource or cleanup produces a non-ready/failed-cleanup receipt and stops cleanup when ownership is uncertain. The tool contains no CVF approval-flow parser; orchestration approval is recorded outside the tool before campaign invocation.

## Portable source validation

Command: `python -m unittest scripts.tests.test_jobrun_fence_fixture`

Result: **PASS** — 10 tests ran in 0.048 seconds; process exit code `0` (`OK`). This was the single initial BUILD harness suite.

The suite exercises pure validation, grant parsing, output confinement, command redaction metadata, uncertain-create reconciliation, and cleanup ownership with in-memory fakes. It does not invoke Docker, connect to MySQL, inspect environment credentials, or claim a runtime fixture result.

## Later campaign invocation

After the source packet is committed and the root has recorded the separate pre-effect safety approval, run from the project root:

```powershell
python scripts/jobrun_fence_fixture.py --role worker
```

The worker receipt is written only under `docs/reviews/probes/r083_fixture/worker/<task-id>/manifest.json`. An independent campaign, if dispatched, uses `--role reviewer` and its own `reviewer/<task-id>/` subtree. Exit code `0` means the disposable campaign completed and verified its cleanup, leaving a receipt marked `READY_FOR_INDEPENDENT_REVIEW`; exit code `2` means the campaign is not ready or failed; exit code `3` means the receipt could not be written. No campaign is claimed by this readiness record.
