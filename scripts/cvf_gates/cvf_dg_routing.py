"""CVF-DG-ROUTE-01: reviewer-local repair decision envelope before REWORK.

Mirrors the parent Review Cost standard section "Reviewer-Local Repair Versus
Worker Return Routing" (reviewerLocalRepairBoundary / reviewerLocalRepairBasis).
It validates the decision envelope only; it cannot judge whether a repair is
semantically small, and it never names a provider or commit role.
"""

from __future__ import annotations

import re
from pathlib import Path

from cvf_dg_common import ControlResult, Finding, read_text, scalar_fields

CONTROL_ID = "CVF-DG-ROUTE-01"


def _single(fields: dict[str, list[str]], name: str, rel: str, out: list[Finding]) -> str | None:
    values = fields[name]
    if len(values) > 1:
        out.append(Finding("DUPLICATE_FIELD", f"{rel}#{name}", f"{name} is declared {len(values)} times"))
        return None
    return values[0] if values else None


def check_routing(project_root: Path, packets: list[str], profile: dict) -> ControlResult:
    routing = profile["reviewerRouting"]
    assessment = routing["assessmentFields"]
    names = ["dispatchKind", "reviewerLocalRepairBoundary", "reviewerLocalRepairBasis", "focusedVerification"] + assessment
    out: list[Finding] = []
    checked: list[str] = []
    for rel in sorted(packets):
        path = project_root / rel
        if not path.is_file():
            out.append(Finding("PACKET_MISSING", rel, "candidate packet does not exist"))
            continue
        fields = scalar_fields(read_text(path), names)
        kind = _single(fields, "dispatchKind", rel, out)
        if kind is None:
            continue  # not a dispatch packet: nothing to route
        checked.append(rel)
        if kind == "INITIAL":
            continue
        if kind != "REWORK":
            out.append(Finding("DISPATCH_KIND_INVALID", f"{rel}#dispatchKind", "dispatchKind must be INITIAL or REWORK"))
            continue
        boundary = _single(fields, "reviewerLocalRepairBoundary", rel, out)
        basis = _single(fields, "reviewerLocalRepairBasis", rel, out)
        if boundary not in routing["allowedBoundaries"]:
            out.append(Finding("REWORK_BOUNDARY_INVALID", f"{rel}#reviewerLocalRepairBoundary", "REWORK needs one allowed reviewerLocalRepairBoundary token"))
        if not basis or len(basis) < routing["basisMinLength"] or re.search(routing["basisPlaceholderPattern"], basis):
            out.append(Finding("REWORK_BASIS_INVALID", f"{rel}#reviewerLocalRepairBasis", "REWORK needs a concrete non-placeholder reviewerLocalRepairBasis"))
        answers: dict[str, str] = {}
        for field in assessment:
            value = _single(fields, field, rel, out)
            if value not in ("YES", "NO"):
                out.append(Finding("ASSESSMENT_FIELD_INVALID", f"{rel}#{field}", f"{field} must be declared exactly once as YES or NO"))
            else:
                answers[field] = value
        if len(answers) != len(assessment):
            continue
        no_fields = [f for f in assessment if answers[f] == "NO"]
        if not no_fields:
            out.append(Finding("REWORK_UNJUSTIFIED_REVIEWER_LOCAL_AVAILABLE", rel,
                               "objective, design, paths, authority, effects, commit owner are unchanged and evidence is determined: reviewer-local repair applies before REWORK"))
            continue
        required_no = routing["boundaryRequiresNo"].get(boundary or "", [])
        if boundary in routing["allowedBoundaries"] and not set(required_no) & set(no_fields):
            out.append(Finding("REWORK_BOUNDARY_UNSUPPORTED", f"{rel}#reviewerLocalRepairBoundary",
                               f"boundary {boundary} needs one of {', '.join(required_no)} assessed NO"))
        if answers["evidenceDetermined"] == "YES":
            verification = _single(fields, "focusedVerification", rel, out)
            if not verification or len(verification) < 8:
                out.append(Finding("FOCUSED_VERIFICATION_MISSING", f"{rel}#focusedVerification", "determined evidence needs a focused verification description"))
    if out:
        return ControlResult(CONTROL_ID, "FAIL", out, checked_control_ids=[CONTROL_ID])
    if not checked:
        return ControlResult(CONTROL_ID, "NOT_APPLICABLE_WITH_REASON", [], reason="no dispatch packet declares dispatchKind in scope",
                             checked_control_ids=[CONTROL_ID])
    return ControlResult(CONTROL_ID, "PASS", [], checked_control_ids=[CONTROL_ID], notes=[f"checked: {', '.join(checked)}"])
