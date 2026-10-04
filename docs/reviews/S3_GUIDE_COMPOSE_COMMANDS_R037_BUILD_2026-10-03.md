# R037 BUILD record — S3 guide Compose commands

Status: BUILT, REVIEW_PENDING (not accepted). Worker: Claude documentation IMPLEMENTATION_WORKER. Risk R1, documentation only. [Order](../work_orders/CCMAI_RUNTIME_037.md), [SPEC](../specs/S3_GUIDE_COMPOSE_COMMANDS_R037_2026-10-03.md).

## 1. Baseline and seed

- baseCommit `964f406a855c4813c64fc0655eeeba484129f507` (also the commit that introduced the seed); HEAD at BUILD start `3a9d012` (dispatch).
- Seed `CVF_SESSION/authority/CCMAI-RUNTIME-037.json` unchanged (not edited, not in the changed set). Tranche record fields (risk, roles, allowed paths, prohibited effects) unchanged.
- Role acknowledgment, declaration and phase transition were recorded in the active handoff and BUILD continuity was synchronized and passed the default preflight (7/7) **before** the first guide edit.

## 2. Changed set

Guide (the only product-adjacent file): `docs/guide/s3-storage.md`, 5 lines changed (5 insertions, 5 deletions), CRLF preserved.

Governed documentation: this record, SPEC status/current-truth text, work order status, tranche record, active state, session memory, active handoff, `IMPLEMENTATION_STATUS.json`, artifact registry, generated `docs/INDEX.md`.

Not changed: backend, frontend, `docker-compose.yml`, `Dockerfile`, dependencies, workflows, gate tooling, parent CVF, the seed.

## 3. Exact guide diff

| # | Before | After |
|---|---|---|
| 1 | `docker exec cqa-app /app/cqa-server migrate-files` | `docker compose exec app /app/cqa-server migrate-files` |
| 2 | `docker exec cqa-app /app/cqa-server migrate-files -apply` | `docker compose exec app /app/cqa-server migrate-files -apply` |
| 3 | `docker exec -it cqa-app /app/cqa-server migrate-files -apply -delete-local` | `docker compose exec app /app/cqa-server migrate-files -apply -delete-local` |
| 4 | `docker exec cqa-app /app/cqa-server migrate-files -down -apply` | `docker compose exec app /app/cqa-server migrate-files -down -apply` |
| 5 (sentence) | `Chạy trên máy chủ. Xem trước, không đụng gì:` | Chạy trên máy chủ, trong thư mục chứa docker-compose.yml, khi dịch vụ app đang chạy. Xem trước, không đụng gì: (the two names are code-formatted in the guide) |

Each replacement was applied with an exact-one-match assertion before writing. `grep -c cqa-app docs/guide/s3-storage.md` = 0.

## 4. SG matrix

| ID | Result | Source pointer |
| --- | --- | --- |
| SG-01 | PASS: four examples match the SPEC list exactly; no `cqa-app` remains; only four prefix lines and one prerequisite sentence changed (diff above, 5+/5−). | `git diff docs/guide/s3-storage.md` |
| SG-02 | PASS by source comparison: Compose service is `app` with no `container_name` (`docker-compose.yml` lines 1 and 19); Dockerfile copies the binary to `/app/cqa-server` (line 27, `ENTRYPOINT` line 34); `frontend/src/views/Settings.vue:274` already uses `docker compose exec app /app/cqa-server migrate-files -down -apply` (asserted by `settings-users-auth.spec.ts:125`). `docker compose exec --help` (local help only, Docker 29.8.1) shows `Usage: docker compose exec [OPTIONS] SERVICE COMMAND [ARGS...]` and that exec allocates a TTY by default (`-T` disables), so dropping `-it` keeps the interactive XOA confirmation. migrate-files arguments, order and flags unchanged. | files above, local help |
| SG-03 | PASS: caution banner, links, direction table, XOA confirmation text, deletion workflow and every other guide line unchanged (diff shows only the five lines). Docs build, catalog, diff check and gates below pass. No full-guide, operational migration/storage safety, universal S3 compatibility or governance claim. | diff and checks below |

## 5. Commands and results

Project root, cached dependencies only, no downloads.

| Check | Result |
| --- | --- |
| Workspace doctor (`check_cvf_workspace_agent_enforcement.ps1`) | 25/25 PASS before BUILD; rerun before commit recorded in the handoff |
| Knowledge ingest | complete; generated `knowledge/_index.json` removed (not committed) |
| Default preflight after BUILD sync, before the guide edit | 7/7 PASS |
| `npm --prefix docs run docs:build` | exit 0, built in 29.05 s; inherited warnings only (large chunk size, `env` highlighter language falls back to `txt`) |
| `scripts/manage_cvf_downstream_catalog.ps1 -Write`, then `-Check` | regenerated; `-Check` PASS |
| `git diff --check` | exit 0 |
| `python -B -m unittest discover -s scripts/tests -p "test_cvf_downstream_gate*.py"` | 46 tests OK (38.5 s) |
| Pre-commit default / PR-range / exact changed-set preflights | recorded in the active handoff |

No new unit, mutation or product tests were added (not required for this order).

## 6. Failed attempts and observations

- A first `docker compose exec --help` combined with a recursive `grep -r` over the whole repository exceeded the 120 s shell timeout and was moved to background (the grep walked `node_modules`). It changed nothing; the help text and source pointers were then re-obtained with a plain help call and the ripgrep-based search tool. No other failure occurred.
- The printed guide commands were **not** executed, not even the dry-run, because it can read the database and S3.
- Out of scope and unchanged: `IMPLEMENTATION_STATUS.json` still lists the stale-guide limitation line in `knownLimitations`; its disposition belongs to the reviewer/orchestrator after acceptance.

## 7. NOT RUN

Migration (`migrate-files`, dry-run or apply), Docker daemon operations and any `exec` against a container, S3 or bucket access, database access, `.env`/credential/config reads, provider/channel/network calls, product tests, backend/frontend builds, GitHub Actions, push, merge, deployment. No governance, hosted-readiness or FREEZE claim.

## 8. BUILD identity and hand-back

Exact BUILD commit: `cf91801168a9d41fb491bd2614f709b201b2556c` (parent `3a9d012`; seed `964f406a855c4813c64fc0655eeeba484129f507` unchanged). It is recorded in the tranche record `buildCommit` and the active handoff by a follow-up documentation commit that changes no guide text (a commit cannot contain its own SHA). Pre-commit validation (doctor, preflights, gate tests) is in the active handoff. Independent Codex REVIEW is next; no self-approval, push, merge, deployment or FREEZE.
