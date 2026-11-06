#!/usr/bin/env python3
"""Fail closed on uncloseable declared gate-to-role responsibility graphs.

This checker validates packet topology, not implementation design. It applies
to changed active work orders and to changed self-declared worker returns or
completion reviews. Historical unchanged artifacts are not migrated.
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
CONTRACT_HEADING = "## Gate-To-Role Closeability Contract"
RECHECK_HEADING = "## Return-Time Closeability Recheck"
ACTIVE_STATUSES = {"DISPATCH_READY", "READY", "ACTIVE", "APPROVED_FOR_EXECUTION"}
# Prospective applicability grammar (cvf.workOrderStatusGrammar@1.0.0). A
# candidate work order declares exactly one line `Status: TOKEN`, optionally
# followed by one parenthetical annotation `Status: TOKEN (issue #12)`. Any
# other shape is malformed and fails closed instead of being skipped.
STATUS_GRAMMAR_ID = "cvf.workOrderStatusGrammar@1.0.0"
STATUS_DECLARATION = re.compile(
    r"(?im)^[ \t]*(?:\*\*)?status(?:\*\*)?[ \t]*:(?:\*\*)?(?P<value>[^\n]*)$"
)
STATUS_VALUE = re.compile(r"^(?P<token>[A-Z][A-Z0-9_]*)(?:[ \t]+\((?P<note>[^()\n]+)\))?$")
FENCED_BLOCK = re.compile(r"(?ms)^[ \t]*(```|~~~).*?^[ \t]*\1[ \t]*$")
# Dispatch-ready variants are checked (fail-closed bias); terminal, held and
# reviewer-disposition families are explicitly not applicable. Anything else
# is unknown and rejected rather than silently skipped.
ACTIVE_VARIANT_PATTERNS = (re.compile(r"^READY_FOR_(?:RE)?DISPATCH$"), re.compile(r"DISPATCH_READY"))
NOT_APPLICABLE_EXACT = {
    "DRAFT", "COMPLETE_PENDING_REVIEW", "BLOCKED_WITH_REASON", "PUBLIC_SYNC_EXPORTED",
    "IMPLEMENTATION_REVIEW_READY", "SUPERSEDED", "CANCELLED", "ARCHIVED", "PARKED",
}
NOT_APPLICABLE_PREFIXES = (
    "CLOSED", "HOLD", "REVIEWER_ACCEPTED", "REVIEWED_", "DISPATCHED", "BACKLOG_",
    "PROPOSED_", "IMPLEMENTED_", "SUPERSEDED_", "CANCELLED_", "DRAFT_", "BLOCKED_",
)
CONTROL_STATUS_GRAMMAR = "work_order_status_grammar"
CONTROL_CONTRACT = "closeability_contract"
REQUIRED_GATE_IDS = {
    "authorization_review", "pre_dispatch_gate", "dispatch_continuity", "focused_checker_tests",
    "adif_integrity", "pre_implementation_autorun", "worker_return_fast",
    "reviewer_fast", "pre_commit", "terminal_completion_review",
    "committed_range_closure", "continuity",
}
PHASE_ORDER = {
    "PRE_DISPATCH": 0, "IMPLEMENTATION": 1, "WORKER_RETURN": 2,
    "REVIEW": 3, "PRE_MATERIAL_COMMIT": 4, "POST_MATERIAL": 5,
    "CONTINUITY_COMMIT": 6, "POST_MATERIAL_CLOSURE": 7,
}
TOPOLOGY_POLICIES = {
    "BOUNDED_PATH_FAMILIES": "COVERED_BY_BOUNDED_PATH_FAMILY",
    "EXACT_PATHS_WITH_NO_FORESEEABLE_SPLIT": "NOT_REQUIRED_UNDER_SIZE_BUDGET",
}
COMMIT_PHASES = {
    "DISPATCH_COMMIT", "DISPATCH_CONTINUITY_COMMIT", "MATERIAL_COMMIT", "CORRECTIVE_MATERIAL_COMMIT",
    "CONTINUITY_COMMIT",
}


@dataclass(frozen=True)
class Violation:
    path: str
    code: str
    message: str


@dataclass(frozen=True)
class StatusDecision:
    """Applicability outcome: CHECK, NOT_APPLICABLE (reason + control IDs) or INVALID."""

    outcome: str
    code: str
    token: str
    reason: str
    checked_control_ids: tuple[str, ...]
    unchecked_control_ids: tuple[str, ...] = ()


def _git(root: Path, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args], cwd=root, text=True, encoding="utf-8",
        errors="replace", stdout=subprocess.PIPE, stderr=subprocess.PIPE,
    )


def changed_paths(base: str, head: str, root: Path = REPO_ROOT) -> tuple[str, ...]:
    paths: set[str] = set()
    if base and head:
        result = _git(root, "diff", "--name-only", f"{base}..{head}")
        if result.returncode:
            raise RuntimeError(result.stderr.strip() or "git diff failed")
        paths.update(result.stdout.splitlines())
    for args in (("diff", "--name-only"), ("diff", "--cached", "--name-only"),
                 ("ls-files", "--others", "--exclude-standard")):
        result = _git(root, *args)
        if result.returncode == 0:
            paths.update(result.stdout.splitlines())
    return tuple(sorted(p.replace("\\", "/") for p in paths if p.strip()))


def _section(text: str, heading: str) -> str:
    match = re.search(
        rf"^{re.escape(heading)}\s*$([\s\S]*?)(?=^##\s+|\Z)", text,
        re.MULTILINE,
    )
    return match.group(1).strip() if match else ""


def _scalar(section: str, field: str) -> str:
    match = re.search(rf"(?m)^\s*{re.escape(field)}:\s*(\S.*?)\s*$", section)
    return match.group(1).strip().strip("`") if match else ""


def _trace_table_field(text: str, field: str) -> str:
    section = _section(text, "## Agent Operation Trace Block")
    for table in _tables(section):
        for row in table[2:]:
            if len(row) >= 2 and row[0] == field:
                return row[1].strip()
    return ""


def _tables(section: str) -> list[list[list[str]]]:
    tables: list[list[list[str]]] = []
    current: list[list[str]] = []
    for line in section.splitlines() + [""]:
        if line.strip().startswith("|"):
            current.append([cell.strip().strip("`") for cell in line.strip().strip("|").split("|")])
        elif current:
            if len(current) >= 2:
                tables.append(current)
            current = []
    return tables


def _graph_rows(section: str) -> tuple[list[dict[str, str]], list[str]]:
    issues: list[str] = []
    expected = ["gateId", "mustPassBy", "repairOwner", "repairPhase",
                "mutationSurface", "topology", "commitOwner", "commitPhase",
                "dependsOn"]
    for table in _tables(section):
        if table[0] != expected:
            continue
        rows: list[dict[str, str]] = []
        for cells in table[2:]:
            if len(cells) != len(expected):
                issues.append("graph row has the wrong column count")
                continue
            rows.append(dict(zip(expected, cells)))
        return rows, issues
    return [], ["exact gate-to-role graph table is missing"]


def _cycle(ids: set[str], dependencies: dict[str, list[str]]) -> bool:
    visiting: set[str] = set()
    visited: set[str] = set()

    def visit(node: str) -> bool:
        if node in visiting:
            return True
        if node in visited:
            return False
        visiting.add(node)
        if any(visit(dep) for dep in dependencies.get(node, []) if dep in ids):
            return True
        visiting.remove(node)
        visited.add(node)
        return False

    return any(visit(node) for node in ids)


def is_work_order_candidate(path: str, text: str) -> bool:
    name = path.replace("\\", "/").rsplit("/", 1)[-1]
    return bool(
        name.startswith("CVF_AGENT_WORK_ORDER_")
        or re.search(r"(?mi)^docType:\s*work_order\s*$", text)
        or CONTRACT_HEADING in text
        or STATUS_DECLARATION.search(FENCED_BLOCK.sub("", text))
    )


def classify_status(path: str, text: str) -> StatusDecision:
    """Strictly classify a work-order status; never silently skip a candidate."""
    grammar = (CONTROL_STATUS_GRAMMAR,)
    both = (CONTROL_STATUS_GRAMMAR, CONTROL_CONTRACT)
    declarations = [m.group("value").strip() for m in STATUS_DECLARATION.finditer(FENCED_BLOCK.sub("", text))]

    def invalid(code: str, reason: str, token: str = "") -> StatusDecision:
        return StatusDecision("INVALID", code, token, f"{reason} (grammar {STATUS_GRAMMAR_ID})", grammar, (CONTROL_CONTRACT,))

    if not declarations:
        if is_work_order_candidate(path, text):
            return invalid("status_missing", "work-order candidate declares no `Status: TOKEN` line")
        return StatusDecision(
            "NOT_APPLICABLE", "not_work_order", "",
            "not a work-order candidate: no Status declaration, work-order name, docType or contract heading",
            grammar, (CONTROL_CONTRACT,),
        )
    if len(declarations) > 1:
        if len(set(declarations)) == 1:
            return invalid("status_duplicate", f"Status declared {len(declarations)} times")
        return invalid("status_contradictory", "conflicting Status declarations: " + " / ".join(declarations))
    value = declarations[0]
    if not value:
        return invalid("status_blank", "Status value is blank")
    parsed = STATUS_VALUE.match(value)
    if not parsed:
        return invalid(
            "status_malformed",
            f"Status value {value!r} is not `TOKEN` or `TOKEN (annotation)`",
        )
    token = parsed.group("token")
    if token in ACTIVE_STATUSES or any(p.search(token) for p in ACTIVE_VARIANT_PATTERNS):
        return StatusDecision("CHECK", "active", token, f"active status {token}", both)
    if token in NOT_APPLICABLE_EXACT or token.startswith(NOT_APPLICABLE_PREFIXES):
        return StatusDecision(
            "NOT_APPLICABLE", "status_not_applicable", token,
            f"status {token} is explicitly not an active dispatch state; closeability contract was NOT checked",
            grammar, (CONTROL_CONTRACT,),
        )
    return invalid("status_unknown", f"Status token {token} is not in the known active or not-applicable sets", token)


def check_work_order(path: str, text: str) -> list[Violation]:
    violations: list[Violation] = []
    decision = classify_status(path, text)
    if decision.outcome == "INVALID":
        return [Violation(path, decision.code, decision.reason)]
    if decision.outcome != "CHECK":
        return violations
    section = _section(text, CONTRACT_HEADING)
    if not section:
        return [Violation(path, "contract_missing", f"missing `{CONTRACT_HEADING}`")]
    expected_scalars = {
        "closeabilityContractVersion": "cvf.gate-role-closeability@1.0.0",
        "closeabilityDisposition": "CLOSEABLE",
        "returnTimeRecheck": "REQUIRED_BEFORE_REPAIR",
    }
    for field, expected in expected_scalars.items():
        actual = _scalar(section, field)
        if actual != expected:
            violations.append(Violation(path, "contract_scalar_invalid", f"{field} must be {expected}"))
    policy = _scalar(section, "implementationTopologyPolicy")
    split = _scalar(section, "foreseeableFileSplitDisposition")
    if policy not in TOPOLOGY_POLICIES or TOPOLOGY_POLICIES.get(policy) != split:
        violations.append(Violation(path, "topology_policy_invalid", "topology policy and foreseeable split disposition are inconsistent"))

    rows, row_issues = _graph_rows(section)
    violations.extend(Violation(path, "graph_invalid", issue) for issue in row_issues)
    ids = {row["gateId"] for row in rows if row.get("gateId")}
    missing = sorted(REQUIRED_GATE_IDS - ids)
    if missing:
        violations.append(Violation(path, "mandatory_gates_missing", "missing mandatory gate ids: " + ", ".join(missing)))
    if len(ids) != len(rows):
        violations.append(Violation(path, "gate_id_duplicate", "gateId values must be non-empty and unique"))
    dependencies: dict[str, list[str]] = {}
    for row in rows:
        gate = row["gateId"]
        deadline = PHASE_ORDER.get(row["mustPassBy"])
        repair_phase = PHASE_ORDER.get(row["repairPhase"])
        if deadline is None or repair_phase is None:
            violations.append(Violation(path, "phase_invalid", f"{gate} uses an unknown phase"))
        elif repair_phase > deadline:
            violations.append(Violation(path, "late_repair_owner", f"{gate} can be repaired only after its pass deadline"))
        for field in ("repairOwner", "mutationSurface", "topology", "commitOwner"):
            if not row[field] or row[field].upper() in {"NONE", "N/A", "TBD"}:
                violations.append(Violation(path, "owner_surface_missing", f"{gate} has no usable {field}"))
        if row["commitPhase"] not in COMMIT_PHASES:
            violations.append(Violation(path, "commit_phase_invalid", f"{gate} has unknown commitPhase"))
        deps = [] if row["dependsOn"].upper() == "NONE" else [d.strip() for d in re.split(r"[,;]", row["dependsOn"]) if d.strip()]
        dependencies[gate] = deps
        unknown = sorted(set(deps) - ids)
        if unknown:
            violations.append(Violation(path, "dependency_unknown", f"{gate} depends on unknown gate(s): " + ", ".join(unknown)))

    dispatch_continuity = next(
        (row for row in rows if row.get("gateId") == "dispatch_continuity"),
        None,
    )
    if dispatch_continuity is not None:
        exact_fields = {
            "mustPassBy": "IMPLEMENTATION",
            "repairOwner": "session-sync-steward",
            "repairPhase": "IMPLEMENTATION",
            "topology": "EXACT_PATHS",
            "commitOwner": "session-sync-steward",
            "commitPhase": "DISPATCH_CONTINUITY_COMMIT",
        }
        for field, expected in exact_fields.items():
            if dispatch_continuity.get(field) != expected:
                violations.append(
                    Violation(
                        path,
                        "dispatch_continuity_invalid",
                        f"dispatch_continuity {field} must be {expected}",
                    )
                )
        surface = dispatch_continuity.get("mutationSurface", "")
        if not re.search(r"AGENT_HANDOFF[^\s;`]*\.md.*material-SHA marker", surface):
            violations.append(
                Violation(
                    path,
                    "dispatch_continuity_invalid",
                    "dispatch_continuity mutationSurface must name an exact AGENT_HANDOFF material-SHA marker",
                )
            )
        if dependencies.get("dispatch_continuity") != ["pre_dispatch_gate"]:
            violations.append(
                Violation(
                    path,
                    "dispatch_continuity_invalid",
                    "dispatch_continuity must depend directly on pre_dispatch_gate",
                )
            )

    if "focused_checker_tests" in dependencies and "dispatch_continuity" not in dependencies["focused_checker_tests"]:
        violations.append(
            Violation(
                path,
                "dispatch_continuity_bypassed",
                "focused_checker_tests must depend on dispatch_continuity before implementation proof proceeds",
            )
        )

    continuity = next((row for row in rows if row.get("gateId") == "continuity"), None)
    if continuity is not None:
        exact_fields = {
            "mustPassBy": "CONTINUITY_COMMIT",
            "repairOwner": "session-sync-steward",
            "repairPhase": "CONTINUITY_COMMIT",
            "commitOwner": "session-sync-steward",
            "commitPhase": "CONTINUITY_COMMIT",
        }
        for field, expected in exact_fields.items():
            if continuity.get(field) != expected:
                violations.append(Violation(path, "terminal_continuity_invalid", f"continuity {field} must be {expected}"))
        if dependencies.get("continuity") != ["terminal_completion_review"]:
            violations.append(Violation(path, "terminal_continuity_invalid", "continuity must depend directly on terminal_completion_review"))

    if "committed_range_closure" in dependencies and dependencies["committed_range_closure"] != ["continuity"]:
        violations.append(
            Violation(
                path,
                "post_material_continuity_bypassed",
                "committed_range_closure must depend directly on continuity so GC-020 can record the material SHA before clean split-range closure",
            )
        )
    if _cycle(ids, dependencies):
        violations.append(Violation(path, "dependency_cycle", "gate dependency graph contains a cycle"))
    return violations


def check_recheck(path: str, text: str) -> list[Violation]:
    applicable = (
        re.search(r"(?mi)^Self-declared worker-return artifact:\s*yes\s*$", text)
        or re.search(r"(?mi)^docType:\s*completion_review\s*$", text)
    )
    if not applicable:
        return []
    section = _section(text, RECHECK_HEADING)
    if not section:
        return [Violation(path, "return_recheck_missing", f"missing `{RECHECK_HEADING}`")]
    disposition = _scalar(section, "closeabilityDisposition")
    blockers = _scalar(section, "outsideAuthorityBlockers")
    route = _scalar(section, "nextRepairRoute")
    redispatch = _scalar(section, "workerRedispatchAllowed")
    status = _scalar(text, "Status")
    manifest_delta = _trace_table_field(text, "Manifest delta")
    issues: list[Violation] = []
    if disposition not in {"CLOSEABLE", "UNCLOSEABLE_PACKET_CONTRADICTION"}:
        issues.append(Violation(path, "return_disposition_invalid", "closeabilityDisposition is invalid"))
    if disposition == "CLOSEABLE" and blockers != "NONE":
        issues.append(Violation(path, "closeable_has_blockers", "CLOSEABLE requires outsideAuthorityBlockers: NONE"))
    if disposition == "UNCLOSEABLE_PACKET_CONTRADICTION":
        if status != "BLOCKED_WITH_REASON":
            issues.append(
                Violation(
                    path,
                    "uncloseable_status_mismatch",
                    "UNCLOSEABLE_PACKET_CONTRADICTION requires top-level Status: BLOCKED_WITH_REASON",
                )
            )
        if blockers in {"", "NONE"}:
            issues.append(Violation(path, "contradiction_without_blocker", "packet contradiction requires a named blocker"))
        if redispatch != "NO":
            issues.append(Violation(path, "contradictory_redispatch", "worker redispatch must be NO for an uncloseable packet"))
        if route not in {"CONSOLIDATED_ORCHESTRATOR_AMENDMENT", "OPERATOR_ESCALATION", "REVIEWER_LOCAL_REPAIR"}:
            issues.append(Violation(path, "repair_route_invalid", "uncloseable packet requires one controlled repair route"))
    if status == "COMPLETE_PENDING_REVIEW" and (disposition != "CLOSEABLE" or blockers != "NONE"):
        issues.append(
            Violation(
                path,
                "complete_status_not_closeable",
                "COMPLETE_PENDING_REVIEW requires closeabilityDisposition: CLOSEABLE and outsideAuthorityBlockers: NONE",
            )
        )
    if status == "COMPLETE_PENDING_REVIEW" and re.search(
        r"(?i)(?:outside\s+(?:the\s+)?manifest|unauthori[sz]ed|partial[_ -]?match|mismatch)",
        manifest_delta,
    ):
        issues.append(
            Violation(
                path,
                "complete_status_manifest_delta",
                "COMPLETE_PENDING_REVIEW cannot declare an out-of-manifest or unauthorized Manifest delta",
            )
        )
    if status == "COMPLETE_PENDING_REVIEW" and "run_worker_return_fast_gate.py" in text:
        limitation_sections = "\n".join(
            match.group(0)
            for match in re.finditer(
                r"(?ims)^###\s+Known Machine-Gate Limitation[^\n]*run_worker_return_fast_gate\.py[\s\S]*?(?=^#{2,3}\s+|\Z)",
                text,
            )
        )
        if limitation_sections and re.search(r"(?i)\b(?:fail(?:ed|ure)?|cannot execute|did not pass)\b", limitation_sections):
            issues.append(
                Violation(
                    path,
                    "complete_status_required_gate_failed",
                    "COMPLETE_PENDING_REVIEW cannot disclose a failed required worker-return gate",
                )
            )
    if redispatch not in {"YES", "NO"}:
        issues.append(Violation(path, "redispatch_value_invalid", "workerRedispatchAllowed must be YES or NO"))
    return issues


def evaluate_detailed(
    base: str, head: str, root: Path = REPO_ROOT
) -> tuple[list[Violation], list[tuple[str, StatusDecision]]]:
    violations: list[Violation] = []
    not_applicable: list[tuple[str, StatusDecision]] = []
    for path in changed_paths(base, head, root):
        full = root / path
        if not full.is_file() or not path.endswith(".md"):
            continue
        text = full.read_text(encoding="utf-8", errors="replace")
        if path.startswith("docs/work_orders/"):
            violations.extend(check_work_order(path, text))
            decision = classify_status(path, text)
            if decision.outcome == "NOT_APPLICABLE":
                not_applicable.append((path, decision))
        if path.startswith("docs/reviews/"):
            violations.extend(check_recheck(path, text))
    return sorted(violations, key=lambda item: (item.path, item.code, item.message)), not_applicable


def evaluate(base: str, head: str, root: Path = REPO_ROOT) -> list[Violation]:
    return evaluate_detailed(base, head, root)[0]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", default="HEAD")
    parser.add_argument("--head", default="HEAD")
    parser.add_argument(
        "--repo-root",
        type=Path,
        default=REPO_ROOT,
        help="Git worktree to evaluate (defaults to the checker repository).",
    )
    parser.add_argument("--enforce", action="store_true")
    args = parser.parse_args()
    repo_root = args.repo_root.resolve()
    if not (repo_root / ".git").exists():
        parser.error(f"--repo-root is not a Git worktree: {repo_root}")
    violations, not_applicable = evaluate_detailed(args.base, args.head, repo_root)
    print("=== CVF Gate-To-Role Closeability Guard ===")
    print(f"Violations: {len(violations)}")
    for item in violations:
        print(f"- {item.path} [{item.code}]: {item.message}")
    for path, decision in not_applicable:
        print(
            f"NOT_APPLICABLE_WITH_REASON {path}: {decision.reason}; "
            f"checkedControlIds={','.join(decision.checked_control_ids)}; "
            f"uncheckedControlIds={','.join(decision.unchecked_control_ids)}"
        )
    if violations and args.enforce:
        print("VIOLATION - repair responsibility topology before dispatch or redispatch.")
        return 2
    print("COMPLIANT" if not violations else "ADVISORY")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
