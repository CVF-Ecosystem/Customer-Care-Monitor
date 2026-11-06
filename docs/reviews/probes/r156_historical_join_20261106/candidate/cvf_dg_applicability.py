"""CVF-DG-APPL-01: strict work-order applicability via the pinned closeability checker.

The status grammar and closeability contract live in exactly one place: the
framework-owned check_gate_to_role_closeability.py, installed here by identity
pin. This module only loads that pinned copy and reports its decision; it never
re-implements the grammar, so an unknown or malformed status cannot be skipped.
"""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

from cvf_dg_common import ControlResult, Finding, read_text

CONTROL_ID = "CVF-DG-APPL-01"
CHECKER_FILE = "check_gate_to_role_closeability.py"


def load_checker(gates_dir: Path):
    spec = importlib.util.spec_from_file_location("cvf_dg_pinned_closeability", gates_dir / CHECKER_FILE)
    if spec is None or spec.loader is None:
        raise ImportError(f"cannot load {CHECKER_FILE}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    for name in ("classify_status", "check_work_order"):
        if not hasattr(module, name):
            raise ImportError(f"pinned checker lacks {name}")
    return module


def check_applicability(project_root: Path, packets: list[str], gates_dir: Path) -> ControlResult:
    try:
        checker = load_checker(gates_dir)
    except Exception as exc:  # fail closed: a missing or incompatible checker never passes
        return ControlResult(CONTROL_ID, "BLOCKED_INVALID_INPUT",
                             [Finding("CHECKER_UNAVAILABLE", f"{gates_dir.name}/{CHECKER_FILE}", f"{type(exc).__name__}: {exc}")])
    findings: list[Finding] = []
    checked: list[str] = []
    not_applicable: list[str] = []
    for rel in sorted(packets):
        path = project_root / rel
        if not path.is_file():
            findings.append(Finding("PACKET_MISSING", rel, "candidate packet does not exist"))
            continue
        text = read_text(path)
        decision = checker.classify_status(rel, text)
        if decision.outcome == "INVALID":
            findings.append(Finding(decision.code, rel, decision.reason))
        elif decision.outcome == "NOT_APPLICABLE":
            not_applicable.append(f"{rel}: {decision.reason}; unchecked={','.join(decision.unchecked_control_ids)}")
        else:
            checked.append(rel)
            for violation in checker.check_work_order(rel, text):
                findings.append(Finding(violation.code, rel, violation.message))
    control_ids = ["work_order_status_grammar", "closeability_contract"]
    if findings:
        return ControlResult(CONTROL_ID, "FAIL", findings, checked_control_ids=control_ids, notes=not_applicable)
    if not checked:
        reason = ("no candidate work-order packet in scope" if not not_applicable
                  else "every candidate is explicitly not active; closeability contract was not checked")
        return ControlResult(CONTROL_ID, "NOT_APPLICABLE_WITH_REASON", [], reason=reason,
                             checked_control_ids=["work_order_status_grammar"], notes=not_applicable)
    return ControlResult(CONTROL_ID, "PASS", [], checked_control_ids=control_ids,
                         notes=[f"checked: {', '.join(checked)}"] + not_applicable)
