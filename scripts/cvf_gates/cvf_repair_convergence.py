"""Repair convergence extension of milestone admission; declared evidence, not causal truth.

Uses the existing authority registry. Enrollment is Local-owned; unreported failures
and commands outside the gate are not intercepted. No effect permission is created.
"""
from __future__ import annotations

import json
import subprocess
from pathlib import Path

REGISTRY = "docs/milestone-coverage-authority.json"
SCHEMA = "cvf.milestoneAuthority@1.1.0"
LEGACY = "cvf.milestoneAuthority@1.0.0"


def parse(raw, api):
    def constant(_):
        api.need(False, "COV_REPAIR_NONFINITE")
    return json.loads(raw.decode("utf-8-sig"), object_pairs_hook=api.pairs,
                      parse_constant=constant)


def pinned(root, ref, api):
    api.evidence(root, [ref], True)
    return parse(api.read_file(root, ref["path"]), api)


def minutes(value, api):
    api.need(value == "UNKNOWN" or (type(value) is int and value >= 0),
             "COV_REPAIR_COST_INVALID")


def controls(registry, api):
    api.need(isinstance(registry, dict) and isinstance(registry.get("milestones"), dict),
             "COV_AUTHORITY_REGISTRY")
    if registry.get("schemaVersion") == LEGACY:
        api.shape(registry, ("schemaVersion", "milestones"))
        return {}
    api.need(registry.get("schemaVersion") == SCHEMA, "COV_AUTHORITY_REGISTRY")
    api.shape(registry, ("schemaVersion", "milestones", "repairControls"))
    rows = registry["repairControls"]
    api.need(isinstance(rows, dict) and len(rows) <= 32
             and set(rows) <= set(registry["milestones"]), "COV_REPAIR_ENROLLMENT_INVALID")
    return rows


def inspect_control(root, milestone, row, api):
    api.shape(row, ("history", "affectedCriteria", "audit"))
    affected = api.unique(row["affectedCriteria"])
    api.need(bool(affected), "COV_REPAIR_CRITERIA_REQUIRED")
    history = row["history"]
    api.need(isinstance(history, list) and 1 <= len(history) <= 32,
             "COV_REPAIR_HISTORY_REQUIRED")
    api.need(len({x.get("path") for x in history if isinstance(x, dict)}) == len(history),
             "COV_REPAIR_HISTORY_DUPLICATE")
    spent, unknown, failed, ever_failed = 0, False, set(), False
    for ref in history:
        attempt = pinned(root, ref, api)
        api.shape(attempt, ("schemaVersion", "milestoneId", "criterionIds", "trancheId",
                            "activity", "result", "proofState", "elapsedMinutes", "evidence"))
        api.need(attempt["schemaVersion"] == "cvf.repairAttempt@1.0.0"
                 and attempt["milestoneId"] == milestone, "COV_REPAIR_OUTCOME_MISMATCH")
        ids = api.unique(attempt["criterionIds"])
        api.need(bool(ids) and ids <= affected, "COV_REPAIR_CRITERIA_MISMATCH")
        api.text(attempt["trancheId"])
        api.need(attempt["activity"] in ("IMPLEMENT", "REPAIR")
                 and attempt["result"] in ("MET", "UNMET"), "COV_REPAIR_ATTEMPT_INVALID")
        api.need(attempt["proofState"] in ("UNPROVED", "STATIC_FIX_VERIFIED", "FULL_FLOW_VERIFIED"),
                 "COV_REPAIR_PROOF_STATE")
        if attempt["result"] == "MET":
            api.need(attempt["proofState"] == "FULL_FLOW_VERIFIED", "COV_REPAIR_STATIC_NOT_RESOLUTION")
            failed.difference_update(ids)
        api.evidence(root, attempt["evidence"], True)
        minutes(attempt["elapsedMinutes"], api)
        if attempt["elapsedMinutes"] == "UNKNOWN":
            unknown = True
        else:
            spent += attempt["elapsedMinutes"]
        if attempt["activity"] == "REPAIR" and attempt["result"] == "UNMET":
            failed.update(ids)
            ever_failed = True
    api.need(ever_failed, "COV_REPAIR_FAILED_ATTEMPT_REQUIRED")
    if not failed:
        api.need(row["audit"] is None, "COV_REPAIR_RESOLVED_AUDIT_MUST_CLEAR")
        return set(), None
    if row["audit"] is None:
        return failed, None
    audit = pinned(root, row["audit"], api)
    api.shape(audit, ("schemaVersion", "milestoneId", "history", "affectedCriteria",
                      "failureBoundary", "observedEvidence", "causeStatus", "causeOrHypothesis",
                      "remainingChain", "discriminator", "correctiveAction", "owner",
                      "cumulativeMinutes", "additionalCeilingMinutes", "stopCondition",
                      "reopenCondition", "disposition", "admittedAction", "review"))
    api.need(audit["schemaVersion"] == "cvf.repairChainAssessment@1.0.0"
             and audit["milestoneId"] == milestone and audit["history"] == history
             and api.unique(audit["affectedCriteria"]) == affected,
             "COV_REPAIR_AUDIT_HISTORY_MISMATCH")
    for key in ("failureBoundary", "causeOrHypothesis", "correctiveAction", "owner",
                "stopCondition", "reopenCondition"):
        api.text(audit[key])
    api.evidence(root, audit["observedEvidence"], True)
    api.need(audit["causeStatus"] in ("PROVEN", "HYPOTHESIS", "UNKNOWN"),
             "COV_REPAIR_CAUSE_STATUS")
    api.need(isinstance(audit["remainingChain"], list) and 1 <= len(audit["remainingChain"]) <= 32,
             "COV_REPAIR_CHAIN_REQUIRED")
    for item in audit["remainingChain"]:
        api.text(item)
    minutes(audit["cumulativeMinutes"], api)
    api.need((unknown and audit["cumulativeMinutes"] == "UNKNOWN") or
             (not unknown and type(audit["cumulativeMinutes"]) is int
              and audit["cumulativeMinutes"] >= spent), "COV_REPAIR_COST_RESET")
    api.need(audit["disposition"] in ("REPAIR_ADMITTED", "DIAGNOSTIC_ONLY", "BLOCKED"),
             "COV_REPAIR_DISPOSITION")
    review = audit["review"]
    api.shape(review, ("reviewer", "decision", "evidence"))
    api.text(review["reviewer"])
    api.need(review["decision"] == "ACCEPTED", "COV_REPAIR_REVIEW_REQUIRED")
    api.evidence(root, review["evidence"], True)
    if audit["disposition"] == "BLOCKED":
        api.need(type(audit["additionalCeilingMinutes"]) is int and audit["additionalCeilingMinutes"] == 0
                 and audit["discriminator"] is None and audit["admittedAction"] is None,
                 "COV_REPAIR_BLOCKED_HAS_ACTION")
        return failed, audit
    discriminator = audit["discriminator"]
    api.shape(discriminator, ("question", "expected", "budgetMinutes"))
    api.text(discriminator["question"]); api.text(discriminator["expected"])
    api.positive(discriminator["budgetMinutes"])
    api.positive(audit["additionalCeilingMinutes"])
    api.need(discriminator["budgetMinutes"] <= audit["additionalCeilingMinutes"],
             "COV_REPAIR_BUDGET_EXCEEDED")
    action = audit["admittedAction"]
    api.shape(action, ("kind", "selected", "owner", "output", "packet"))
    api.need(action["kind"] == ("IMPLEMENT" if audit["disposition"] == "REPAIR_ADMITTED" else "VERIFY")
             and bool(api.unique(action["selected"])) and set(action["selected"]) <= affected,
             "COV_REPAIR_ACTION_MISMATCH")
    api.text(action["owner"]); api.text(action["output"])
    api.evidence(root, [action["packet"]], True)
    return failed, audit


def validate(root, data, registry, api):
    enrolled = controls(registry, api)
    milestone = data["milestone"]["id"]
    if milestone not in enrolled:
        return
    affected, audit = inspect_control(root, milestone, enrolled[milestone], api)
    api.need(affected <= {r["id"] for r in data["rows"]}, "COV_REPAIR_CRITERIA_MISMATCH")
    api.need(not affected or data["conclusions"]["product"] != "COMPLETE",
             "COV_REPAIR_UNRESOLVED_PRODUCT")
    if data["action"]["kind"] not in ("IMPLEMENT", "VERIFY") or not affected:
        return
    selected = set(data["action"]["selected"])
    if not selected & affected:
        for row in data["rows"]:
            if row["id"] in affected:
                api.need("independence" in row, "COV_REPAIR_INDEPENDENCE_REQUIRED")
        return
    if audit is None and data["action"]["kind"] == "VERIFY":
        return  # Existing finite intake may gather assessment evidence; no experiment grant.
    api.need(audit is not None, "COV_REPAIR_AUDIT_REQUIRED")
    api.need(audit["disposition"] != "BLOCKED", "COV_REPAIR_SUCCESSOR_HELD")
    admitted = audit["admittedAction"]
    api.need(selected == set(admitted["selected"]) and all(
        data["action"][k] == admitted[k] for k in ("kind", "owner", "output")), "COV_REPAIR_ACTION_MISMATCH")
    api.need(api.packet_path == admitted["packet"]["path"] and api.packet_digest == admitted["packet"]["sha256"],
             "COV_REPAIR_PACKET_MISMATCH")
    api.need(data["action"]["budgetMinutes"] <= audit["additionalCeilingMinutes"],
             "COV_REPAIR_BUDGET_EXCEEDED")
    if data["action"]["kind"] == "VERIFY":
        api.need(data["action"]["question"] == audit["discriminator"]["question"]
                 and data["action"]["budgetMinutes"] <= audit["discriminator"]["budgetMinutes"],
                 "COV_REPAIR_DIAGNOSTIC_MISMATCH")


def transition(root: Path, base: str | None, api):
    """Compare with resolved Git ancestry, including registry-only changes/deletion."""
    def git(*args):
        try:
            return subprocess.run(["git", "-C", str(root), *args], capture_output=True,
                                  text=True, encoding="utf-8", errors="replace", timeout=10)
        except subprocess.TimeoutExpired:
            api.need(False, "COV_REPAIR_BASE_TIMEOUT")
    current = parse(api.read_file(root, REGISTRY), api) if (root / REGISTRY).is_file() else None
    current_rows = controls(current, api) if (root / REGISTRY).is_file() else {}
    top = git("rev-parse", "--show-toplevel")
    if top.returncode or Path(top.stdout.strip()).resolve() != root.resolve():
        api.need(not current_rows, "COV_REPAIR_BASE_REQUIRED")
        return
    ref = base or "HEAD"
    resolved = git("rev-parse", "--verify", ref + "^{commit}")
    if resolved.returncode:
        api.need(not base and not current_rows, "COV_REPAIR_BASE_REQUIRED")
        return
    sha = resolved.stdout.strip()
    api.need(git("merge-base", "--is-ancestor", sha, "HEAD").returncode == 0,
             "COV_REPAIR_BASE_NOT_ANCESTOR")
    exists = git("ls-tree", "--name-only", sha, "--", REGISTRY)
    api.need(exists.returncode == 0, "COV_REPAIR_BASE_REQUIRED")
    previous = None
    if exists.stdout.strip():
        old = git("show", sha + ":" + REGISTRY)
        api.need(old.returncode == 0, "COV_REPAIR_BASE_REQUIRED")
        previous = parse(old.stdout.encode("utf-8"), api)
    for milestone, before in controls(previous, api).items() if exists.stdout.strip() else ():
        api.need(milestone in current_rows and current["milestones"].get(milestone)
                 == previous["milestones"][milestone], "COV_REPAIR_ENROLLMENT_ERASED")
        after = current_rows[milestone]
        api.need(after.get("history", [])[:len(before["history"])] == before["history"]
                 and set(before["affectedCriteria"]) <= set(after.get("affectedCriteria", [])),
                 "COV_REPAIR_HISTORY_RESET")
    for milestone, row in current_rows.items():
        inspect_control(root, milestone, row, api)
    if base and base != "HEAD":
        transition(root, None, api)  # Preserve HEAD enrollment as well as the selected ancestor.
