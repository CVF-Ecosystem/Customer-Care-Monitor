"""Source-bound milestone coverage. Stdlib-only, shared by Core and portable gates.

Checks declared set/identity/phase joins, not semantic truth or authenticated roles.
"""
from __future__ import annotations

import hashlib
import json
import re
from pathlib import Path

SCHEMA = "cvf.milestoneCoverage@1.0.0"
TAG = "cvf-milestone-coverage-json"
AUTHORITY_REGISTRY = "docs/milestone-coverage-authority.json"
MAX_BYTES = 1024 * 1024
ID = re.compile(r"[A-Z][A-Z0-9_-]*[0-9][A-Z0-9_-]*\Z")
SHA = re.compile(r"[0-9a-f]{64}\Z")
STATES = {"NOT_STARTED", "IMPLEMENTED_UNPROVED", "VERIFIED", "BLOCKED", "UNKNOWN"}
PLACEHOLDERS = {"", "TODO", "TBD", "N/A", "NONE", "UNKNOWN", "LATER"}


class Invalid(ValueError):
    pass


def need(condition, code):
    if not condition:
        raise Invalid(code)


def shape(value, keys, optional=()):
    need(isinstance(value, dict) and set(keys) <= set(value)
         and set(value) <= set(keys) | set(optional), "COV_OBJECT_SHAPE")


def text(value):
    need(isinstance(value, str) and value.strip().upper() not in PLACEHOLDERS, "COV_TEXT_REQUIRED")


def positive(value):
    need(type(value) is int and value > 0, "COV_FINITE_BUDGET_REQUIRED")


def unique(values):
    need(isinstance(values, list) and len(values) <= 500
         and all(isinstance(x, str) and ID.fullmatch(x) for x in values), "COV_ID_LIST")
    need(len(values) == len(set(values)), "COV_DUPLICATE_ID")
    return set(values)


def digest(data: bytes) -> str:
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def read_file(root: Path, relative: str) -> bytes:
    need(isinstance(relative, str) and relative and "\\" not in relative
         and ":" not in relative and not relative.startswith("/")
         and all(x not in ("", ".", "..") for x in relative.split("/")), "COV_UNSAFE_PATH")
    target = root / relative
    need(target.resolve().is_relative_to(root.resolve()), "COV_UNSAFE_PATH")
    need(target.is_file() and target.stat().st_size <= MAX_BYTES, "COV_FILE_MISSING_OR_OVERSIZE")
    data = target.read_bytes()
    need(len(data) <= MAX_BYTES, "COV_FILE_MISSING_OR_OVERSIZE")
    return data


def unfenced(value: str) -> str:
    """Retain line positions while removing Markdown fenced examples."""
    fence = None
    lines = []
    for line in value.splitlines(keepends=True):
        m = re.match(r"^\s{0,3}(`{3,}|~{3,})(.*)$", line)
        if fence is None and m:
            fence = m[1]
            lines.append("\n")
        elif fence is not None:
            if m and m[1][0] == fence[0] and len(m[1]) >= len(fence) and not m[2].strip():
                fence = None
            lines.append("\n")
        else:
            lines.append(line)
    return "".join(lines)


def section(value: str, heading: str) -> str:
    need(isinstance(heading, str) and re.fullmatch(r"#{1,6} .+", heading), "COV_HEADING_INVALID")
    raw = value.replace("\r\n", "\n").splitlines(keepends=True)
    visible = unfenced(value.replace("\r\n", "\n")).splitlines(keepends=True)
    starts = [i for i, line in enumerate(visible) if line.rstrip("\n") == heading]
    need(len(starts) == 1, "COV_HEADING_MISSING_OR_AMBIGUOUS")
    start = starts[0]
    level = len(heading.split(" ", 1)[0])
    end = len(raw)
    for i in range(start + 1, len(visible)):
        m = re.match(r"^(#{1,6}) ", visible[i])
        if m and len(m[1]) <= level:
            end = i
            break
    return "".join(raw[start:end])


def criterion_ids(value: str) -> set[str]:
    tables, current = [], []
    for line in unfenced(value).splitlines() + [""]:
        if line.strip().startswith("|"):
            current.append(line.strip())
        elif current:
            tables.append(current)
            current = []
    need(len(tables) == 1 and len(tables[0]) >= 3, "COV_AUTHORITY_TABLE_REQUIRED")
    table = tables[0]
    need(all(re.fullmatch(r":?-{3,}:?", c.strip()) for c in table[1].strip("|").split("|")),
         "COV_AUTHORITY_TABLE_SEPARATOR")
    ids = []
    for line in table[2:]:
        cell = line.strip("|").split("|", 1)[0].strip()
        first = cell.split()[0] if cell else ""
        need(bool(ID.fullmatch(first)), "COV_AUTHORITY_ID_INVALID")
        suffix = cell[len(first):].strip()
        if suffix.startswith("/"):
            need(all(ID.fullmatch(x.strip()) for x in suffix[1:].split(",")), "COV_AUTHORITY_ID_AMBIGUOUS")
        else:
            need(not any(ID.fullmatch(x) for x in suffix.split()), "COV_AUTHORITY_ID_AMBIGUOUS")
        ids.append(first)
    return unique(ids)


def evidence(root: Path, refs, required=False):
    need(isinstance(refs, list) and len(refs) <= 100, "COV_EVIDENCE_LIST")
    need(not required or bool(refs), "COV_EVIDENCE_REQUIRED")
    for ref in refs:
        shape(ref, ("path", "sha256"))
        need(isinstance(ref["sha256"], str) and bool(SHA.fullmatch(ref["sha256"])), "COV_HASH_INVALID")
        need(digest(read_file(root, ref["path"])) == ref["sha256"], "COV_EVIDENCE_STALE")


def validate(root: Path, data) -> list[str]:
    try:
        _validate(root, data)
        return []
    except (Invalid, OSError, UnicodeError, ValueError, TypeError, KeyError) as exc:
        return [str(exc) if isinstance(exc, Invalid) else "COV_INVALID_INPUT:" + type(exc).__name__]


def _validate(root: Path, data):
    shape(data, ("schemaVersion", "authority", "milestone", "phase", "action", "rows", "supportingWork", "conclusions"))
    need(data["schemaVersion"] == SCHEMA, "COV_SCHEMA")
    milestone = data["milestone"]
    shape(milestone, ("id", "consumer", "outcome", "checkpoint"))
    for key in ("id", "consumer", "outcome"):
        text(milestone[key])
    shape(milestone["checkpoint"], ("trigger", "owner", "decision"))
    for value in milestone["checkpoint"].values():
        text(value)
    phase, action = data["phase"], data["action"]
    need(phase in ("PLAN", "DISPATCH", "REVIEW"), "COV_PHASE")
    shape(action, ("kind", "selected", "owner", "output", "budgetMinutes", "stop", "next"), ("question", "sourceBoundary"))
    kind = action["kind"]
    need(kind in ("PLAN", "IMPLEMENT", "VERIFY", "REVIEW"), "COV_ACTION_KIND")
    need((phase == "PLAN" and kind in ("PLAN", "VERIFY")) or
         (phase == "DISPATCH" and kind in ("IMPLEMENT", "VERIFY")) or
         (phase == "REVIEW" and kind == "REVIEW"), "COV_PHASE_ACTION_MISMATCH")
    for key in ("owner", "output", "stop", "next"):
        text(action[key])
    positive(action["budgetMinutes"])
    selected = unique(action["selected"])
    if kind == "VERIFY":
        text(action.get("question"))
        text(action.get("sourceBoundary"))
    authority = data["authority"]
    if authority is None:
        need(kind == "VERIFY" and not selected and data["rows"] == [], "COV_UNKNOWN_AUTHORITY_IMPLEMENTATION")
        expected = set()
    else:
        shape(authority, ("path", "heading", "sha256"))
        need(isinstance(authority["path"], str) and authority["path"].startswith("docs/"), "COV_AUTHORITY_LOCATION")
        registry = json.loads(read_file(root, AUTHORITY_REGISTRY).decode("utf-8-sig"), object_pairs_hook=_pairs)
        shape(registry, ("schemaVersion", "milestones"))
        need(registry["schemaVersion"] == "cvf.milestoneAuthority@1.0.0" and isinstance(registry["milestones"], dict), "COV_AUTHORITY_REGISTRY")
        binding = registry["milestones"].get(milestone["id"])
        need(binding == {k: authority[k] for k in ("path", "heading")}, "COV_MILESTONE_AUTHORITY_MISMATCH")
        source = section(read_file(root, authority["path"]).decode("utf-8-sig"), authority["heading"])
        need(digest(source.encode("utf-8")) == authority["sha256"], "COV_AUTHORITY_STALE")
        expected = criterion_ids(source)
        need(bool(expected) and (bool(selected) or kind in ("PLAN", "REVIEW")), "COV_SELECTION_EMPTY")
    need(selected <= expected, "COV_SELECTION_OUTSIDE_AUTHORITY")
    rows = data["rows"]
    need(isinstance(rows, list) and len(rows) <= 500, "COV_ROWS")
    actual = []
    for row in rows:
        shape(row, ("id", "state", "gap", "owner", "destination", "evidence"), ("independence",))
        actual.append(row["id"])
        need(row["state"] in STATES, "COV_STATE")
        evidence(root, row["evidence"], row["state"] == "VERIFIED")
        for key in ("gap", "owner", "destination"):
            need(isinstance(row[key], str), "COV_ROW_TEXT_TYPE")
            if row["state"] != "VERIFIED":
                text(row[key])
        if "independence" in row:
            decision = row["independence"]
            shape(decision, ("reviewer", "reason", "evidence"))
            text(decision["reviewer"]); text(decision["reason"])
            evidence(root, decision["evidence"], True)
        if kind == "IMPLEMENT" and row["state"] == "UNKNOWN":
            need(row["id"] not in selected and "independence" in row, "COV_UNKNOWN_IMPLEMENTATION")
    need(unique(actual) == expected, "COV_CRITERION_SET_MISMATCH")
    supporting = data["supportingWork"]
    need(isinstance(supporting, list) and len(supporting) <= 500, "COV_SUPPORTING_LIST")
    for item in supporting:
        shape(item, ("criterionIds", "reason", "budgetMinutes", "stop", "returnToProduct"))
        ids = unique(item["criterionIds"])
        need(bool(ids) and ids <= expected, "COV_SUPPORTING_CRITERIA")
        for key in ("reason", "stop", "returnToProduct"):
            text(item[key])
        positive(item["budgetMinutes"])
    conclusion = data["conclusions"]
    shape(conclusion, ("tranche", "trancheSummary", "trancheEvidence", "product", "productSummary", "acceptance", "owner"))
    for key in ("tranche", "product"):
        need(conclusion[key] in ("PENDING", "COMPLETE", "BLOCKED"), "COV_CONCLUSION")
        need(phase == "REVIEW" or conclusion[key] == "PENDING", "COV_PREMATURE_TERMINAL")
    for key in ("trancheSummary", "productSummary", "owner"):
        text(conclusion[key])
    evidence(root, conclusion["trancheEvidence"], conclusion["tranche"] == "COMPLETE")
    evidence(root, conclusion["acceptance"], conclusion["product"] == "COMPLETE")
    if conclusion["product"] == "COMPLETE":
        need(bool(expected) and all(r["state"] == "VERIFIED" for r in rows), "COV_PRODUCT_INCOMPLETE")


def _pairs(pairs):
    out = {}
    for key, value in pairs:
        need(key not in out, "COV_DUPLICATE_JSON_KEY")
        out[key] = value
    return out


def required_artifact(relative: str, value: str) -> bool:
    if "/archive/" in relative or not relative.endswith(".md"):
        return False
    if relative.startswith("docs/roadmaps/"):
        return True
    if not relative.startswith("docs/work_orders/"):
        return False
    status = re.findall(r"^Status:\s*(\S+)", unfenced(value), re.M)
    # Historical/draft packets are not execution grants; malformed status remains required.
    return len(status) != 1 or not status[0].startswith(("DRAFT", "HOLD", "CLOSED", "COMPLETE", "BLOCKED", "REJECTED", "SUPERSEDED", "ARCHIVED"))


def validate_text(root: Path, relative: str, value: str, required=None) -> list[str]:
    if len(value.encode("utf-8")) > MAX_BYTES:
        return ["COV_FILE_MISSING_OR_OVERSIZE"]
    if required is None:
        required = required_artifact(relative, value)
    # Parse fence boundaries explicitly so examples nested in another fence do not enroll.
    bodies, fence, collecting, body = [], None, False, []
    for line in value.splitlines():
        m = re.match(r"^\s{0,3}(`{3,}|~{3,})(.*)$", line)
        if fence is None and m:
            fence, collecting, body = m[1], m[2].strip() == TAG, []
        elif fence is not None and m and m[1][0] == fence[0] and len(m[1]) >= len(fence) and not m[2].strip():
            if collecting:
                bodies.append("\n".join(body))
            fence, collecting = None, False
        elif collecting:
            body.append(line)
    if collecting:
        return ["COV_UNCLOSED_BLOCK"]
    if not bodies and not required:
        return []
    if len(bodies) != 1:
        return ["COV_BLOCK_REQUIRED"]
    try:
        need(len(bodies[0].encode()) <= MAX_BYTES, "COV_BLOCK_OVERSIZE")
        data = json.loads(bodies[0], object_pairs_hook=_pairs,
                          parse_constant=lambda _: (_ for _ in ()).throw(Invalid("COV_JSON_CONSTANT")))
        issues = validate(root, data)
        if not issues and relative.startswith("docs/work_orders/") and required:
            need(data["phase"] == "DISPATCH", "COV_EXECUTABLE_PHASE_REQUIRED")
        statuses = re.findall(r"^Status:\s*(\S+)", unfenced(value), re.M)
        terminal = any(s.startswith(("CLOSED", "COMPLETE", "ACCEPTED", "BLOCKED", "REJECTED")) for s in statuses)
        if not issues and relative.startswith(("docs/roadmaps/", "docs/reviews/")) and terminal:
            need(data["phase"] == "REVIEW", "COV_TERMINAL_REVIEW_REQUIRED")
            need(data["conclusions"]["tranche"] != "PENDING" and data["conclusions"]["product"] != "PENDING", "COV_TERMINAL_CONCLUSIONS_REQUIRED")
        return issues
    except (Invalid, ValueError, TypeError, RecursionError) as exc:
        return [str(exc) if isinstance(exc, Invalid) else "COV_INVALID_JSON"]


def validate_file(root: Path, relative: str) -> list[str]:
    try:
        return validate_text(root, relative, read_file(root, relative).decode("utf-8-sig"))
    except (Invalid, OSError, UnicodeError) as exc:
        return [str(exc) if isinstance(exc, Invalid) else "COV_INVALID_INPUT:" + type(exc).__name__]
