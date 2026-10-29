# R139 Core reconciliation — read-only preparation

Status: PREPARATION_ONLY; Root independent review and committed maintenance authority precede effects. Worker: Terra6.1High CORE_MAINTENANCE_WORKER. No fetch, updater, install, SDK import, browser, provider, Core or shared-workspace write performed.

## Source and boundaries

Workspace `D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace`; project `D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor-AI`; Core `D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\.Controlled-Vibe-Framework-CVF`; backup root `D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\_cvf-core-backups`. Core was clean, HEAD b8ead4f0d1bf9224338a206b2abcd893ec1801b1, public origin `https://github.com/Blackbird081/Controlled-Vibe-Framework-CVF.git`, cached origin/main 15e39450853184eb65dbde73398c69c3171a6241. Cached ref is not a fresh remote observation.

The official updater is `Core\scripts\update_cvf_workspace_public_core.ps1`; pinned blob `5019107b93a0204f481a6dfaa8c062f4ad5af803`, cached target blob `f1ad15761fc39529b9943aef070b1bc99b7a6646`. Installer `Core\scripts\install_cvf_workspace_root_wrappers.ps1` blobs are `234e1ea7093a3569c96bed6e2b4d522c6a9221d2` and `17799b9ead34aaff96125fbd63160aa8eeb99469`. Their only cached differences add development responsibility text. Updater moves Core to timestamped `_cvf-core-backups\.Controlled-Vibe-Framework-CVF-<yyyyMMdd-HHmmss>`, clones public default branch, validates required kit, rewrites workspace rules, invokes the **new clone's** wrapper installer, and conditionally refreshes public-free/paid-user-safe rule packs. Catch preserves failed clone under the same backup root and restores Core only; it does not restore earlier shared writes. It can update all sibling manifests if `-UpdateProjectManifests` is supplied; do not supply it, `-AllowPendingCoreBackup`, or `-OverlaySourcePath`.

**Target race:** clone is not pinned and shared writes occur before it returns. Freshly fetch and verify before the transaction, but that alone cannot guarantee the cloned branch did not advance. Check clone HEAD, origin, clean state and kit immediately after; if target differs from the separately admitted full hash, stop and restore verified preimage before downstream adoption. Record all intermediate effects; do not silently adopt a changed target. Verify workspace, Core, backup, failed-clone and all restore paths resolve under the intended workspace, without reparse-point escape, and names do not collide before any move. Do not delete a failed clone or backup.

Active `CVF_RULE_PACKS\ACTIVE_RULE_PACK.json` says `operator-local`, so inspected updater's public-profile refresh condition is false. Recheck immediately before execution. If changed to public-free/paid-user-safe, stop for separately assessed recursive replacement effects. Avoid reading operator-local pack contents. Check the three named overlay files below before execution; the installer removes them when present. They are currently absent.

## Shared preimages and absence markers

All paths in this section are relative to the workspace absolute path above. Before execution, capture exact bytes, SHA-256, existence/type and ACL as applicable into a unique protected transaction backup outside Core, verify each copy, and record ABSENT explicitly. Restoration must copy bytes and verify hashes; remove a newly created target only if the recorded preimage was absent and its exact resolved path is approved. Do not recursively remove a computed directory. Preserve the original Core using the updater's own moved backup and verify its HEAD/worktree before any restoration. Independently review rollback before use.

| Existing shared target | Raw SHA-256 |
| --- | --- |
| `New-CVF-Governed-Project.ps1` | c650d21f8491000f9d2d0dcc29afa6220eaf14d5c0ff90be5ac3dcdf2c5c89d1 |
| `Run-CVF-NewProject-Enforcement.ps1` | 95632be706ef17ce3d93bd24efd071a129a5472431e638a062666e9117470f1f |
| `Update-CVF-Workspace.ps1` | ceb8b9ff50f31bcb8a19db28b15e01a58f18de1bcc522ebf52bfb339cb6a4d77 |
| `Update-CVF-Workspace-Public-Profile.ps1` | 8ae9de4a7fcf94bc8df6b3ab72d8b747db079df06d5a980e2f9c62d9ea5d7cee |
| `Test-CVF-Workspace.ps1` | 6d40d701d597a9d4d64dacf3e59b0c9855aafe0f84a486635d7400827a6c6c41 |
| `Repair-CVF-Workspace.ps1` | eecc44650f59e2923ae69f0e08af0fc3a6fcc13853cccda051157a85129cf39b |
| `Manage-CVF-Workspace.ps1` | 8eb098c6c6a8422e52f7beed9b32d1954b968db74e931fa79610b6e0ff2dd11c |
| `.agents\workflows\cvf-onboard.md` | d6a88de28c6585623fe8f74ae2db84849e743f984f17559914108f9da7192e80 |
| `.agents\workflows\pre-commit-check.md` | 2a749cac1286ff57601807b0176d52dc08a9b292894255e76a3fcbf4d2558039 |
| `WORKSPACE_RULES.md` | 996199450066797a9e0c1b10317c5b4503246404c3823ae06f7fe2c7f373d87c |
| `WORKSPACE_PROJECT_ENFORCEMENT_BASELINE.json` | a4ac3cb61e18b136d9cb6a39dd298774d56aeb37fbab3e6864c52fa0936f29fb |
| `CVF_WORKSPACE_USER_GUIDE.md` | 022feace321ebd4f0075d4eccd1e6a2485f2967c73b9e29275a2e1b19346f9f1 |
| `CVF_WORKSPACE_HUONG_DAN_SU_DUNG.md` | 375f3f1fd406e41a5e37edb485e3cbbbda92ecb18d2374f34f4422f8aaa5dbf4 |
| `CVF_WORKSPACE_CLASSIFICATION_GUIDE.md` | be1ffc5008684f8915c93305021866c790e750b716ea14084b20a350791ad360 |

`Get-CVF-Workspace-OverlayProfiles.ps1`, `Update-CVF-Workspace-Overlay.ps1`, and `CVF_WORKSPACE_OVERLAY_STATUS.json` were ABSENT. The wrapper creates baseline only when absent; backup its existing bytes regardless. Also preserve `CVF_RULE_PACKS\ACTIVE_RULE_PACK.json` as an observation/preimage without changing it, and preserve project `.cvf\manifest.json`, `.cvf\gate-profile.lock.json`, `scripts\cvf_gates\` installed files, `.github\workflows\cvf-downstream-gates.yml`, and learning templates before project adoption. Record absent files as absent and verify directory/file types. No private overlays or secrets are needed.

## Sibling baseline

All sibling manifest paths are `<workspace>\<name>\.cvf\manifest.json`, all with `cvfCoreRelativePath` `../.Controlled-Vibe-Framework-CVF`. Thirteen siblings pin `19386f64e6bc36d1dcdbadca6ff97253feefb1bf`: Bao_gia_dich_vu_container_pkd_edit, Bao_gia_phu_thu_Dau DO, Controlled-Vibe-Framework-CVF-P3, Controlled-Vibe-Framework-CVF-P3-CP2, Dang_ky_dich_vu_container, Khai-bao-Cang-vu 07.2026, Mini_Game, QD209, shift-operations-workspace, shift-operations-workspace-recovery-20260926, tdr_processor-container-2026, Trading-Tools, XD_App. CVF-Operations-Workspace pins `27137db4d9aa2aea931ddd2507185d5c24943080`; this project pins b8ead4f0d1bf9224338a206b2abcd893ec1801b1. Snapshot each sibling manifest's raw SHA-256 and recheck unchanged after; do not update sibling manifests. Shared Core replacement can make sibling pins stale and must be reported to their owners; it does not imply sibling validation or adoption.

## Project profile adoption and validation plan

At cached target, `cvf.downstreamGateProfile@1.2.0` and status grammar remain unchanged. The cached gate-subtree diff changes only `cvf_milestone_coverage.py`: exact empty bootstrap README placeholders for roadmaps/work orders are exempted; any added content still requires admission. The associated machine contract also changes. Inspect full target diff and verify target source/commit after fetch; do not assume cached results prove a later target.

The official source parser accepts `install --project-root <project> --core-root <Core> --core-commit <full-head> --upgrade`. `--upgrade` writes each inherited gate file and lock, prepares missing learning scaffolds, and may rewrite CI workflow only if its current bytes match both old lock and old generated template. Otherwise it preserves workflow and may return `UPGRADED_CI_RECONCILIATION_REQUIRED`, which is not success and requires explicit project-owner reconciliation of pins. No hand edits to gate modules or lock. The exact post-review candidate from project root is:

```powershell
python -B "../.Controlled-Vibe-Framework-CVF/scripts/lib/downstream_governance/cvf_downstream_gate_runner.py" install --project-root . --core-root "../.Controlled-Vibe-Framework-CVF" --core-commit <verified-full-head> --upgrade
```

This is a proposal, not authorization or an executed command. After approved backup and independently reviewed authority, the updater candidate is `powershell -NoProfile -ExecutionPolicy Bypass -File "../.Controlled-Vibe-Framework-CVF/scripts/update_cvf_workspace_public_core.ps1" -WorkspaceRoot "D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace"`; preserve evidence, check actual head and shared outputs, update only this project's manifest under the approved maintenance contract, install the matching profile, reconcile CI/catalog/continuity as needed, and run doctor plus applicable gates with explicit packets. Any failed check remains open, and rollback must cover shared preimages as well as Core. Root owns authority, independent review and commit.

R139 product execution seed e9ddcbcd/profile1.1 stays immutable and HOLD. Browser cumulative16, source27, controllers4, campaign1, service5 and existing UNKNOWNs are not reset. No product, runtime, FREEZE or governance proof is claimed.
