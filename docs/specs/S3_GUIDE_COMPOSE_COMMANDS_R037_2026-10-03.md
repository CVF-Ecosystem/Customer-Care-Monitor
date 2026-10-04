# R037 — S3 guide uses current Compose service

Status: REVIEW_PASS / FREEZE_OPEN (documentation contract).

Date: 2026-10-03. Risk ceiling R1 (documentation only). Baseline93102f26117a28cc7146fdfe2b4bc1fea82e10f7; immutable seed 964f406a855c4813c64fc0655eeeba484129f507. [Order](../work_orders/CCMAI_RUNTIME_037.md).

## Intake and design

Implementation status records the stale cqa-app S3-guide command. At dispatch, exactly four examples in `docs/guide/s3-storage.md` used docker exec with that old container name. Static source: `docker-compose.yml` has service `app`, no fixed container_name; root Dockerfile installs `/app/cqa-server`; Settings S3_DOWN_COMMAND already uses docker compose exec app. Installed `docker compose exec --help` confirms service syntax and default TTY. Source/help observations only, no container/S3 execution.

Replace only the four executable examples with:

```bash
docker compose exec app /app/cqa-server migrate-files
docker compose exec app /app/cqa-server migrate-files -apply
docker compose exec app /app/cqa-server migrate-files -apply -delete-local
docker compose exec app /app/cqa-server migrate-files -down -apply
```

Replace the introductory migration sentence with a short prerequisite: run from the directory containing `docker-compose.yml`, with service `app` already running. Do not add start/restart commands. Compose exec allocates TTY by default, so the old docker exec `-it` prefix is removed from the deletion example. Do not add -T, background execution or confirmation bypass. Preserve migrate-files arguments/order, tenant explanation, dry-run default, direction table, XOA confirmation and separate deletion workflow.

## Acceptance

| ID | Static/documentation evidence |
| --- | --- |
| SG-01 | Four examples exactly match the list, no executable cqa-app remains in this guide. Only four prefix lines and one prerequisite sentence change. |
| SG-02 | Compare service/binary with Compose/Dockerfile, existing Settings constant and installed local exec help. Arguments and interactive deletion intent unchanged; no migration/container/S3 execution. |
| SG-03 | Inherited caution banner, links and all other guide text unchanged. Docs/catalog/gates/diff pass. No full-guide, operational migration/storage safety, universal S3 compatibility or governance validation claim. |

No new unit/mutation/product tests for this small documentation correction. Mandatory repository gate tests still run. Product/Compose/Dockerfile/files/credentials/DB/network/runtime changes excluded. Missing cached docs prerequisites -> BUILD_BLOCKED, no downloads.

## Current truth

*Historical (dispatch-time): DISPATCH_READY / NOT_BUILT, guide stale, no worker started.* **Current:** Claude BUILD corrected exactly the four command examples and the prerequisite sentence in `docs/guide/s3-storage.md` (evidence: [BUILD record](../reviews/S3_GUIDE_COMPOSE_COMMANDS_R037_BUILD_2026-10-03.md)), by source comparison only; Independently REVIEW_PASS / FREEZE_OPEN for exact BUILD cf91801168a9d41fb491bd2614f709b201b2556c; [review](../reviews/CCMAI_RUNTIME_037_INDEPENDENT_REVIEW_2026-10-03.md). Other inherited S3 assertions not evaluated. Prior R036/R035/R034 acceptance, source-specific NOT RUN/observation limits and R033 local-message FREEZE unchanged. Actual MCP execution and live Pancake inputs/authority remain separate. No new FREEZE/provider/governance/hosted claim.
