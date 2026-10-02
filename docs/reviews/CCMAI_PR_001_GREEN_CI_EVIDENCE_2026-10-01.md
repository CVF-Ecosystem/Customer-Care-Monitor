# PR #1 — reviewed hosted rerun after GOV1-CI1

Date: 2026-10-01. Reviewer: Codex. [Draft PR #1](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/pull/1) head `3e0b37ea729c75fe1a319dda61ee384b71706e7c`, still open, draft and unmerged when checked. GitHub API reported four workflow runs with this exact `head_sha`, each `status=completed` and `conclusion=success`. The previous red head `19fdc4e` remains a historical observation in `CCMAI_PR_001_HOSTED_CI_REVIEW_2026-10-01.md`.

| Workflow/run | Checked job and log evidence | Conclusion |
|---|---|---|
| [CVF Governance](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/36798737665) | Ubuntu: PyYAML install, 46/46 gate tests, base fetch, then PR-range preflight `PASS (6/6 executed gates passed; 1 skipped)`; catalog is honestly `SKIP` in Ubuntu. Windows catalog `-Check` job passed. The earlier generated-`.pyc` failure did not recur. | `success` |
| [Backend Go / F08](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/36798737618) | Disposable MySQL readiness passed, full `go test ./... -json` and build passed. The result gate printed PASS for all five package-qualified DB sentinels: db, api/handlers, engine, cli, storagecfg; it printed `GATE PASSED: all required DB sentinels ran and passed; no DB-unavailable skips.` Cleanup passed. | `success` |
| [Frontend](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/36798737608) | npm cache hit from `frontend/package-lock.json`, 18 test files and 186 tests passed, build passed. | `success` |
| [Build and Deploy Docs](https://github.com/CVF-Ecosystem/Customer-Care-Monitor/actions/runs/36798737598) | Docs build passed; deploy job was skipped for pull_request as configured. | `success` (build) |

The backend job log shows the `add-mask` invocation, loopback `127.0.0.1:3306:3306` bind and three later `TEST_DB_DSN: ccma:***` entries. A bounded search of that job log found no unmasked `TEST_DB_DSN=ccma:<value>` line. This supports the observed runner behavior, not a universal no-secret-leak guarantee. No real AI provider call was made or claimed; repository CI proof does not prove CVF governance of AI runtime behavior.

**Disposition:** GOV1-CI1 hosted gate and F08's requested GitHub Actions DB-test proof are verified at this SHA. GOV-001 and R019/F08 remain `REVIEW_PASS / FREEZE_OPEN`; no merge, deployment or FREEZE follows. R020 F01-A may leave its parked dependency checkpoint after this evidence is synchronized; MCP F01-B and F02–F07 remain open. The PR stays draft for the owner's later disposition. This evidence file is a local reviewer record after the proven remote head; a later push would create a new head requiring its own CI check.
