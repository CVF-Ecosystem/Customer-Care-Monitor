# R139 official owned-stdio capability — prerequisite and source review

Status: BLOCKED_CORE_FRESHNESS_SOURCE_REJECTED
Risk: R2
Worker: Sol Medium candidate author; replacement Terra6.1High /root/sdk_source_worker stopped before edits/checks
Reviewer: Root, independent source/readiness and result reviewer
Independent admission reviewer: Terra6.1High /root/sdk_admission_review
Prospective authority: `CVF_SESSION/authority/CCMAI-RUNTIME-139-OFFICIAL-CLI-ADMISSION.json`, held draft, not execution authority

## Decision and evidence

Doctor rerun returned FAIL27/28: only BEHIND_PUBLIC_REMOTE, b8ead4f0d1bf9224338a206b2abcd893ec1801b1 versus locally recorded origin/main15e39450853184eb65dbde73398c69c3171a6241. Public origin, kit, clean Core, manifest and installed controls passed. This is a cached tracking-ref comparison, not a new remote freshness observation. Exact prior Owner approval in ROADMAP-DOCS-ADMISSION limits the exception to documentary scope; COMMIT-MAINTENANCE admits dependent static repairs only. Full browser-connection handling does not silently replace the approved Core pin or expand a scoped exception. Root accepts independent Terra HOLD; no runtime release is written. The replacement source worker conservatively stopped before editing or syntax checks; Root preserves that dissent rather than claiming completed authoring.

Installed package1.9.0 exports official Client/StdioClientTransport used by its own daemon. That native SDK remains a plausible managed isolated route, not tested capability. The earlier CLI draft was rejected before execution because shared-daemon stop/implicit restart conflict with one-start bounds. No CLI, SDK import/connect, Chrome launch, browser call, provider call, install or configuration edit was made in this checkpoint. Browser cumulative16 remains.

Sol left one candidate at `docs/reviews/probes/r139_integrated_20261021/connection_recovery_20261029/native_cli/run_owned_stdio.mjs`, raw SHA256528e97b748bd4ac8f902c254398e90c0d1c4c66a4dd77cd918ad30a05f5a673c,10251bytes. It is retained unchanged as rejected worker source, not an executable release. Root's bounded read and replacement worker source inspection identify: project-root traversal goes one level too far; importing with process argv containing --run can execute; SDK request timeout does not bound transport startup; post-close server-child observation misses reparented owned Chrome PID. Profile isolation also needs strict temporary-directory ownership rather than name regex. These are consolidated source findings, not an exhaustive implementation review. Actual installed source does support structured pages and pageId on scoped tools, so those are not defects. Replacement and Root syntax invocations0; inherited Sol syntax count UNKNOWN without original receipt. No rerun is spent to manufacture history.

## Concrete next scope for shared-Core maintainer

The project AGENTS First-Request Protocol step2 requires stale Core reconciliation with the official updater; its workspace isolation rule otherwise makes sibling Core read-only for downstream work. This boundary must be explicitly handled before adapter BUILD or runtime. Proposed next action is Core-maintainer reconciliation, not another Antigravity browser retry:

1. Preserve current clean Core b8ead4f0 and this project's committed documentary checkpoint. Fetch and verify public origin/main; current15e39450 is only a candidate until that verification. Stop if the public target changes materially or pending Core edits appear.
2. Review the official `scripts/update_cvf_workspace_public_core.ps1` transaction from the workspace owner context. It backs up/moves the shared Core, clones public Core, writes workspace rules and installs root wrappers. Do not run blanket `-UpdateProjectManifests`: it can affect sibling projects and does not repair this project's portable/profile joins by itself. Resolve/check all absolute move targets under the intended workspace before any move. No private overlay.
   Shared effects also require review/backup: wrapper installation removes its named prior overlay artifacts; the updater conditionally synchronizes public-free/paid-user-safe packs, which recursively replaces the selected pack root and rewrites its manifest/ACTIVE_RULE_PACK plus possibly missing root templates. Current read-only ACTIVE_RULE_PACK.json says operator-local, so that profile-sync branch is not triggered by the inspected pinned updater now. Recheck the chosen replacement script and actual profile before execution. Core catch restores the Core directory, not a proven transaction rollback for earlier rules/wrapper/profile writes. Capture exact wrapper/overlay target paths and back up all approved affected shared files before running; stop if an unapproved target appears.
3. Explicitly admit this project's new full Core pin and matching gate installation through the public installer, with reviewed immutable maintenance scope/hashes. Do not hand-edit installed modules or lock and do not silently rewrite old R139 execution authority. Reconcile affected template/manifest/CI/catalog joins, run doctor and static gates; preserve other projects' pins and report shared-Core effects to their owners.
4. Re-enter the same R139 source repair/review only after valid prerequisites. Product execution e9ddcbcd/profile1.1 mismatch remains a separate HOLD; capability admission cannot release product scenarios.

Read-only cached diff is seven public commits,1456files/23935insertions/7204deletions. Gate subtree changes include milestone coverage validator and its contract; this is broader than a browser configuration fix. No updater, fetch, move, Core write, pin/profile change, release or deployment was performed. Approval must cover the shared workspace Core transaction and project adoption, not merely repeat browser permission.

## Documentary verification

Terra /root/sdk_admission_review independently returned PASS DOCUMENTARY CHECKPOINT after the consolidated shared-effect disclosure; runtime remains HOLD. Root local legacy preflight7/7 and pinned profile pre-commit mandatoryControls7/failed0 passed with both explicit review packets; COV and ROLE were checked. APPL closeability is NOT_APPLICABLE_WITH_REASON because the work order is BLOCKED_WITH_REASON; it was not checked. Mandatory gate-unit discovery was subsequently rerun:103tests/109.543s, OK. Cross-shell catalog parity2 is inherited because catalog source is unchanged. No hosted CI success is claimed. Received raw-file hashes were directly checked; raw original EOF blank lines are retained as a disclosed style exception. Final staged gate checks and postcommit range results remain necessary.

## Two conclusions

Selected supporting slice: BLOCKED. Return custody reviewed, native route identified, candidate rejected and prerequisite bounded; no browser capability established.

Product: full M3/FREEZE OPEN. All eight approved criteria retain their previous states, owners and destinations; M3-05/M3-08 blocked. Existing source27/controllers4/campaign1/service5 and all UNKNOWNs remain. Facebook/Zalo OA parked. Next owner is the workspace/Core maintainer under an explicit reconciliation scope; Root resumes independent source/runtime review afterwards.

## M3 coverage reconciliation

```cvf-milestone-coverage-json
{
  "schemaVersion": "cvf.milestoneCoverage@1.0.0",
  "authority": {
    "path": "docs/reviews/planning/CCMA_PRODUCT_DELIVERY_ROADMAP_2026-10-13.md",
    "heading": "## 7. M3 — Người duyệt, giao việc và kết quả xử lý",
    "sha256": "712817cd25936efd4fb13e0dd456ebe4ddceb5fccfd1cacb83042838724065a4"
  },
  "milestone": {
    "id": "M3",
    "consumer": "Người duyệt và người xử lý ca trên một installation Pancake",
    "outcome": "Hoàn tất review, giao việc, kết quả và lịch sử qua browser/backend theo tiêu chí M3 đã duyệt.",
    "checkpoint": {
      "trigger": "Thay đổi trạng thái kết nối Chrome, cạn budget hoặc thay đổi nguồn, evidence hay phụ thuộc trọng yếu.",
      "owner": "Root điều phối; Product Owner quyết định phạm vi và acceptance nghiệp vụ.",
      "decision": "Đối chiếu evidence và caps; tiếp tục hoặc sửa trong authority, trình Owner nếu đổi phạm vi, HOLD hoặc dừng; đưa kết quả về R139/G3."
    }
  },
  "phase": "REVIEW",
  "action": {
    "kind": "REVIEW",
    "selected": [
      "M3-05",
      "M3-08"
    ],
    "owner": "Root independent reviewer",
    "output": "Held native capability admission, rejected source and concrete Core reconciliation boundary",
    "budgetMinutes": 20,
    "stop": "Doctor freshness failure outside scoped exception; no source repair or runtime effects",
    "next": "Workspace/Core maintainer explicit reconciliation scope, then same R139 source repair/review"
  },
  "rows": [
    {
      "id": "M3-01",
      "state": "IMPLEMENTED_UNPROVED",
      "gap": "Quy tắc ưu tiên và applicability current-source còn mở.",
      "owner": "Product Owner / Root",
      "destination": "M1-01 quyết định trigger/quy tắc chờ và ngoại lệ; Owner xử lý quy tắc queue còn mở, Root đối chiếu source applicability trước G3.",
      "evidence": [
        {
          "path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md",
          "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"
        }
      ]
    },
    {
      "id": "M3-02",
      "state": "IMPLEMENTED_UNPROVED",
      "gap": "Quyền, reason và binding current-source chưa có acceptance đầy đủ.",
      "owner": "Root / Product Owner",
      "destination": "Root kiểm binding và trace; Owner chấp nhận process tại G3.",
      "evidence": [
        {
          "path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md",
          "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"
        },
        {
          "path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md",
          "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"
        }
      ]
    },
    {
      "id": "M3-03",
      "state": "IMPLEMENTED_UNPROVED",
      "gap": "Applicability lineage và negative hiện hành còn cần đối chiếu.",
      "owner": "Root / Product Owner",
      "destination": "Root kiểm source; G1 stale/competing negative nếu source thay đổi hoặc lộ gap; Owner chấp nhận effect.",
      "evidence": [
        {
          "path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md",
          "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"
        },
        {
          "path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md",
          "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"
        }
      ]
    },
    {
      "id": "M3-04",
      "state": "IMPLEMENTED_UNPROVED",
      "gap": "Role, SLA và demo current-source chưa hoàn tất.",
      "owner": "Product Owner / Root",
      "destination": "M1-02/04 quyết định role/SLA; G3 demo assignment, outcome và audit.",
      "evidence": [
        {
          "path": "docs/reviews/R131_M3_ACCEPTANCE_REVIEW_2026-10-19.md",
          "sha256": "ec78f8db455589e9bad8a35d6f757ebb0068de5e175853e841a3fd0188e4c31b"
        }
      ]
    },
    {
      "id": "M3-05",
      "state": "BLOCKED",
      "gap": "E01–E06 chưa có runtime proof; Chrome connection và effect admission đang HOLD.",
      "owner": "Root / Product Owner",
      "destination": "Cùng R139 G0–G3 sau thay đổi kết nối được duyệt và admission phù hợp; Owner chấp nhận user consequence.",
      "evidence": [
        {
          "path": "docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md",
          "sha256": "63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"
        },
        {
          "path": "docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md",
          "sha256": "0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"
        }
      ]
    },
    {
      "id": "M3-06",
      "state": "IMPLEMENTED_UNPROVED",
      "gap": "Browser lifetime và visible hold chưa được chứng minh current-source.",
      "owner": "Root / Product Owner",
      "destination": "Cùng R139 E04/G2 lifetime proof; Owner chấp nhận visible hold tại G3.",
      "evidence": [
        {
          "path": "docs/reviews/R137_INDEPENDENT_SOURCE_REVIEW_2026-10-21.md",
          "sha256": "63cc0bc309a235c7d36384b7b78afa48acafea8267622d93d191780633071783"
        }
      ]
    },
    {
      "id": "M3-07",
      "state": "UNKNOWN",
      "gap": "Phạm vi R101 hiện không gửi notification; applicability với source và scope hiện hành còn cần kiểm tra.",
      "owner": "Root",
      "destination": "G3 đối chiếu applicability của exclusion đã có; nếu scope đổi, áp dụng change control trước khi xét contract notification.",
      "evidence": [
        {
          "path": "docs/reviews/R131_M3_ACCEPTANCE_MATRIX_2026-10-19.md",
          "sha256": "7dabfc6a2a8bbf42a4447063e961a428849b0a020ae4c02589a7fb323510f9ac"
        }
      ]
    },
    {
      "id": "M3-08",
      "state": "BLOCKED",
      "gap": "Current-source browser/backend demo và disposition viewport/offline còn mở; Chrome connection HOLD.",
      "owner": "Root / Product Owner",
      "destination": "Cùng R139 G2 proof và G3; Owner chấp nhận qualification viewport/raster/offline.",
      "evidence": [
        {
          "path": "docs/reviews/R139_GEMINI_NATIVE_BROWSER_CHECK_ROOT_REVIEW_2026-10-28.md",
          "sha256": "0f434528ae218886eca1c8c68efb68d3cab164cbad57b2a2b51af79f14135d47"
        },
        {
          "path": "docs/reviews/R135_M3_ACCEPTANCE_DELTA_2026-10-20.md",
          "sha256": "cfe89ee2ddf42dcbf80edfb316ff5a05f5f364907bfb8636d9e69bfc99bd4a00"
        }
      ]
    }
  ],
  "supportingWork": [
    {
      "criterionIds": [
        "M3-05",
        "M3-08"
      ],
      "reason": "Resolve the evidenced browser prerequisite with installed native capability without blind retries; bound the actual Core dependency.",
      "budgetMinutes": 20,
      "stop": "Core prerequisite or rejected source prevents execution; no retries or budget reset.",
      "returnToProduct": "Same R139 G0-G3 after Core prerequisite and independent capability/source admission."
    }
  ],
  "conclusions": {
    "tranche": "BLOCKED",
    "trancheSummary": "Native SDK candidate rejected unexecuted; Core freshness prerequisite not admitted for effects.",
    "trancheEvidence": [],
    "product": "BLOCKED",
    "productSummary": "M3-05/M3-08, full M3/FREEZE remain open; exact prior coverage and remainder retained.",
    "acceptance": [],
    "owner": "Root review kỹ thuật; Product Owner chấp nhận nghiệp vụ."
  }
}
```
