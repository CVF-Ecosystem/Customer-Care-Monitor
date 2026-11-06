"""CVF-DG-ROLE-01 reviewer independence and CVF-DG-CLAIM-01 claim support.

Review artifacts declare `Risk:`, `Worker:` and `Reviewer:` lines; gate claims use
`Gate claim: <CONTROL_ID>=<STATE>`. Declarations are collected first (unfenced, any
label case) and only then parsed, so a malformed, mis-cased or duplicated declaration
is refused instead of disappearing. A claim is supported only by the coverage evidence of
this project (install lock, invocation receipts, proof receipt); prose never upgrades a state.
"""

from __future__ import annotations

import re
from pathlib import Path

from cvf_dg_common import ControlResult, Finding, declarations, normalize_value, read_text, strip_fences
from cvf_dg_coverage import RANK, control_states

ROLE_ID = "CVF-DG-ROLE-01"
CLAIM_ID = "CVF-DG-CLAIM-01"
NEGATION = re.compile(r"(?i)\b(?:not|never|no claim|cannot|must not|without|does not)\b")
CLAIM_VALUE = re.compile(r"^(?P<control>[A-Z][A-Z0-9-]*[A-Z0-9])=(?P<state>[A-Z_]+)(?:[ \t]+\((?P<reason>[^()]+)\))?$")


def _identity(value: str) -> str:
    return re.sub(r"\s+", " ", re.sub(r"\(.*?\)", "", value)).strip().casefold()


def _one(rel: str, decls: list[tuple[str, str]], label: str, out: list[Finding]) -> str | None:
    """The single, correctly spelled, non-empty value of a declaration, or None after recording why."""
    if len(decls) != 1:
        if decls:
            out.append(Finding("DUPLICATE_FIELD", f"{rel}#{label}", f"{label} is declared {len(decls)} times"))
        return None
    spelled, raw = decls[0]
    if spelled != label:
        out.append(Finding("LABEL_CASE", f"{rel}#{label}", f"declaration label {spelled!r} must be spelled exactly {label!r}"))
        return None
    value = normalize_value(raw)
    if not value:
        out.append(Finding("EMPTY_FIELD", f"{rel}#{label}", f"{label} is declared but blank"))
        return None
    return value


def check_roles(project_root: Path, reviews: list[str], profile: dict) -> ControlResult:
    roles = profile["reviewRoles"]
    independent, valid = set(roles["independentRiskLevels"]), roles["validRiskLevels"]
    out: list[Finding] = []
    graded: list[str] = []
    skipped: list[str] = []
    for rel in sorted(reviews):
        path = project_root / rel
        if not path.is_file():
            out.append(Finding("PACKET_MISSING", rel, "candidate review does not exist"))
            continue
        text = read_text(path)
        risks, workers, reviewers = declarations(text, "Risk"), declarations(text, "Worker"), declarations(text, "Reviewer")
        if not (risks or workers or reviewers):
            skipped.append(f"{rel}: no Risk, Worker or Reviewer declaration (not a role-bearing review)")
            continue
        if not risks:
            out.append(Finding("RISK_MISSING", f"{rel}#Risk", "a review that names a Worker or Reviewer must declare `Risk: R0|R1|R2|R3`; independence cannot be graded without it"))
            continue
        risk = _one(rel, risks, "Risk", out)
        worker = _one(rel, workers, "Worker", out) if workers else None
        reviewer = _one(rel, reviewers, "Reviewer", out) if reviewers else None
        if risk is None:
            continue
        if risk not in valid:
            out.append(Finding("RISK_INVALID", f"{rel}#Risk", f"Risk {risk!r} is not one of {valid} (exact upper-case token required)"))
            continue
        graded.append(rel)
        if risk in independent:
            for name, value in (("Worker", worker), ("Reviewer", reviewer)):
                if value is None and not (workers if name == "Worker" else reviewers):
                    out.append(Finding("ROLE_FIELD_INVALID", f"{rel}#{name}", f"{risk} review needs exactly one non-empty {name} line"))
            if worker and reviewer and _identity(worker) == _identity(reviewer):
                out.append(Finding("SELF_REVIEW", f"{rel}#Reviewer", f"{risk} review names the same agent as worker and reviewer; an independent reviewer is required"))
    if out:
        return ControlResult(ROLE_ID, "FAIL", out, checked_control_ids=[ROLE_ID], notes=skipped)
    if not graded:
        return ControlResult(ROLE_ID, "NOT_APPLICABLE_WITH_REASON", [], reason="no role-bearing review artifact in scope",
                             checked_control_ids=[ROLE_ID], notes=skipped)
    return ControlResult(ROLE_ID, "PASS", [], checked_control_ids=[ROLE_ID], notes=[f"checked: {', '.join(graded)}"] + skipped)


def check_claims(project_root: Path, docs: list[str], profile: dict, lib_dir: Path) -> ControlResult:
    states = control_states(project_root, lib_dir)
    known = {c["id"] for c in profile["controls"]}
    allowed = set(profile["claimStates"])
    phrases = [p.lower() for p in profile["broadClaimPhrases"]]
    out: list[Finding] = []
    seen_claims = 0
    for rel in sorted(docs):
        path = project_root / rel
        if not path.is_file():
            out.append(Finding("PACKET_MISSING", rel, "candidate document does not exist"))
            continue
        text = read_text(path)
        claimed: set[str] = set()
        for spelled, raw in declarations(text, "Gate claim"):
            seen_claims += 1
            where = f"{rel}#Gate claim"
            parsed = CLAIM_VALUE.fullmatch(normalize_value(raw))
            if spelled != "Gate claim" or not parsed:
                out.append(Finding("CLAIM_MALFORMED", where, f"claim {spelled}: {raw!r} is not exactly `Gate claim: <CONTROL_ID>=<STATE>` with an optional `(reason)`"))
                continue
            control, state, reason = parsed.group("control"), parsed.group("state"), parsed.group("reason")
            where = f"{rel}#Gate claim:{control}"
            if control in claimed:
                out.append(Finding("CLAIM_DUPLICATE", where, f"{control} is claimed more than once in this document"))
            claimed.add(control)
            if control not in known:
                out.append(Finding("CLAIM_CONTROL_UNKNOWN", where, f"{control} is not a control of the pinned profile"))
            elif state not in allowed:
                out.append(Finding("CLAIM_STATE_UNKNOWN", where, f"state {state} is not one of {sorted(allowed)}"))
            elif state == "NOT_APPLICABLE_WITH_REASON" and not (reason and reason.strip()):
                out.append(Finding("CLAIM_REASON_MISSING", where, "NOT_APPLICABLE_WITH_REASON needs a parenthesised reason"))
            elif state in RANK and RANK[state] > states.get(control, {"rank": 0})["rank"]:
                actual = states.get(control, {"state": "NOT_INSTALLED"})["state"]
                out.append(Finding("UNSUPPORTED_CLAIM", where, f"claims {state} but project evidence supports at most {actual}"))
        body = strip_fences(text)
        for number, line in enumerate(body.splitlines(), 1):
            if any(p in line.lower() for p in phrases) and not NEGATION.search(line):
                out.append(Finding("BROAD_CLAIM_PHRASE", f"{rel}:{number}", "blanket enforcement claim; state INSTALLED/INVOKED/PROVEN_HERMETIC per control instead"))
    if out:
        return ControlResult(CLAIM_ID, "FAIL", out, checked_control_ids=[CLAIM_ID])
    return ControlResult(CLAIM_ID, "PASS", [], checked_control_ids=[CLAIM_ID],
                         notes=[f"{seen_claims} explicit claim line(s) and {len(docs)} document(s) scanned"])
