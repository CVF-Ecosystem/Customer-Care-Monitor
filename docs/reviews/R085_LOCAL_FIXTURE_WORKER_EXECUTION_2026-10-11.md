# R085 worker local MySQL fixture execution

Status: **FAILED / NOT_READY**. The one authorized worker campaign ran once and failed before MySQL readiness or SQL. The tool verified cleanup of its exact task-owned container and network. No replay, source repair, or reviewer campaign is authorized by this result.

## Process and preflight evidence

The exact command `python -B scripts/jobrun_fence_fixture.py --role worker` started once (PID `14440`) from the project root. The process returned exit code `2` after `3.922119` monotonic seconds. It wrote zero bytes to stdout and stderr; the raw captures are `docs/reviews/probes/r085_fixture/worker/launcher-de26d3b5dda944908754dc2afad2a5c3/stdout.raw.bin` and `.../stderr.raw.bin`. HEAD was `e03e39b05d4dc5ab14a21d7ca9e9366de53ec11d` both before and after the run. The launcher record reports one start attempt, one created fixture process, and zero automatic retries.

Before the campaign, `collect_source_identity` confirmed all four committed source/readiness/work-order blobs matched accepted source commit `67e76de80ffd1b091ea48b5a5bd4c1b1929fa634`; their current filtered blobs and physical hashes matched the pre-effect intake. The selected cached image metadata matched the approved full image ID and repository digest `sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b` (`linux/amd64`). File-level facts are in `docs/reviews/probes/r085_fixture/worker/launcher-de26d3b5dda944908754dc2afad2a5c3/preflight.json`.

Two preflight helper errors occurred before the fixture CLI process existed: an import registration error and a result-key error. A separate PowerShell expression error stopped my first sidecar write after the campaign had exited. These helper/post-processing issues did not launch another fixture campaign or alter cleanup. They are separately recorded in `docs/reviews/probes/r085_fixture/worker/launcher-de26d3b5dda944908754dc2afad2a5c3/launcher-errors.json`.

## Tool receipt and result

The tool generated the one new worker receipt at `docs/reviews/probes/r083_fixture/worker/39cd4dbacf3744bc8ea39f413b33e4d4/manifest.json`. Its legacy `CCMAI-RUNTIME-083` label remains unchanged; the R085 authority-to-receipt binding is `docs/reviews/probes/r085_fixture/worker/authority-binding-39cd4dbacf3744bc8ea39f413b33e4d4.json`.

The sanitized receipt reports `runtime.status=FAILED`, `runtime.failure.code=MOUNT_INVALID`, safe message `fixture container must have exactly one tmpfs mount`, and `overallStatus=NOT_READY`. The exact internal network and fresh container were both confirmed created. The tool then removed each and verified both absent: `cleanup.status=VERIFIED_CLEAN`, `absenceVerified=true`, `containerStatus=ABSENT_VERIFIED`, and `networkStatus=ABSENT_VERIFIED`. The recorded resource IDs are in the manifest and binding.

The failure occurred before readiness observations and before SQL. MySQL server version, settings, effective grants, denial checks, DDL/DML, row results, and schema-drop results are **NOT_OBSERVED**; the cached image tag does not establish a running server version. Existing `ccma-db-1` was not accessed. No existing database SQL, environment, container environment, or credentials were read.

## Disposition

The worker campaign did not succeed, so the root reviewer campaign remains NOT_RUN and not approved. No source, test, work-order, authority, tranche, or continuity files were changed by the worker. Root review and exact-ID absence checks remain pending; this report makes no fixture readiness or application/fence integration claim.