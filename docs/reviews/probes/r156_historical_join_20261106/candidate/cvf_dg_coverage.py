"""Truthful per-phase/per-control coverage: INSTALLED, INVOKED, PROVEN_HERMETIC.

INSTALLED  - identity-verified files from the lock (a YAML/file token alone is not execution).
INVOKED    - a receipt for this exact profile+runner identity records the control result.
PROVEN_HERMETIC - a Core golden-proof receipt bound to the same identity covers the control.
Hosted CI is never inferred: a pr-ci receipt only shows the command ran somewhere.
"""

from __future__ import annotations

from pathlib import Path

from cvf_dg_common import content_sha256, load_json_strict, read_text
from cvf_dg_install import load_profile, verify_install

RANK = {"NOT_INSTALLED": 0, "BLOCKED": 0, "INSTALLED": 1, "INVOKED": 2, "PROVEN_HERMETIC": 3}
RECEIPT_SCHEMA = "cvf.downstreamGateReceipt@1.0.0"
PROOF_SCHEMA = "cvf.downstreamGateProofReceipt@1.0.0"
PIN_KEYS = ("profileSha256", "runnerSha256", "bundleSha256")
CLAIM_BOUNDARY = ("INSTALLED = identity-verified files; INVOKED = receipt bound to the installed identity; "
                  "PROVEN_HERMETIC = Core golden proof bound to the installed identity. None of these implies a hosted CI run, "
                  "real-project adoption, public rollout or runtime AI enforcement.")


def _load_lock(project_root: Path, profile: dict) -> dict | None:
    path = project_root / profile["projectPaths"]["lock"]
    try:
        data = load_json_strict(path)
    except (OSError, ValueError):
        return None
    return data if isinstance(data, dict) else None


def _load_receipt(project_root: Path, profile: dict, phase: str, lock: dict) -> dict | None:
    path = project_root / profile["projectPaths"]["receiptDir"] / f"{phase}.json"
    try:
        data = load_json_strict(path)
    except (OSError, ValueError):
        return None
    if not isinstance(data, dict) or data.get("schemaVersion") != RECEIPT_SCHEMA or data.get("phase") != phase:
        return None
    if any(data.get(k) != lock.get(k) for k in PIN_KEYS):
        return None
    return data


def _workflow_installed(project_root: Path, profile: dict, lock: dict) -> bool:
    path = project_root / profile["projectPaths"]["ciWorkflow"]
    if not path.is_file():
        return False
    text = read_text(path)
    return ("cvf_downstream_gate_runner.py" in text and "--phase pr-ci" in text
            and "--base-from-env" in text and all(str(lock.get(k)) in text for k in PIN_KEYS))


def compute_coverage(project_root: Path, lib_dir: Path, proof_path: Path | None = None) -> dict:
    profile = load_profile(lib_dir)
    lock = _load_lock(project_root, profile)
    install = verify_install(project_root, lib_dir) if lock else None
    installed = bool(install and install.outcome == "PASS" and lock)
    block_reason = "" if installed else ("gate profile lock missing: MIGRATION_REQUIRED" if not lock else
                                          "install integrity failed: " + ", ".join(sorted({f.code for f in install.findings})))  # type: ignore[union-attr]
    proof = None
    if installed and proof_path is not None:
        try:
            candidate = load_json_strict(proof_path)
        except (OSError, ValueError):
            candidate = None
        if isinstance(candidate, dict) and candidate.get("schemaVersion") == PROOF_SCHEMA \
                and all(candidate.get(k) == lock.get(k) for k in PIN_KEYS):
            proof = candidate
    phases: dict[str, dict] = {}
    for phase in profile["phases"]:
        receipt = _load_receipt(project_root, profile, phase, lock) if installed and lock else None
        receipt_results = {r["controlId"]: r for r in (receipt or {}).get("results", []) if isinstance(r, dict) and "controlId" in r}
        row: dict[str, dict] = {}
        for control in profile["controls"]:
            cid = control["id"]
            if phase not in control["phases"]:
                row[cid] = {"state": "NOT_APPLICABLE_WITH_REASON", "reason": f"{cid} is not mapped to phase {phase} by the pinned profile"}
                continue
            if not installed:
                row[cid] = {"state": "NOT_INSTALLED" if not lock else "BLOCKED", "reason": block_reason}
                continue
            entry: dict = {"state": "INSTALLED", "reason": "identity-verified files"}
            if phase == "pr-ci" and not _workflow_installed(project_root, profile, lock):
                entry = {"state": "NOT_INSTALLED", "reason": "PR workflow is absent or does not carry the pinned runner invocation"}
            elif cid in receipt_results:
                result = receipt_results[cid]
                entry = {"state": "INVOKED", "reason": "receipt bound to installed identity", "result": result.get("outcome"),
                         "exitCode": receipt.get("exitCode"), "executionContext": receipt.get("executionContext")}  # type: ignore[union-attr]
            if proof and phase in proof.get("controls", {}).get(cid, {}).get("phases", []) and entry["state"] != "NOT_INSTALLED":
                entry = {**entry, "state": "PROVEN_HERMETIC", "proof": proof["controls"][cid].get("negativeCases", [])}
            row[cid] = entry
        phases[phase] = row
    summary: dict[str, int] = {}
    for row in phases.values():
        for entry in row.values():
            summary[entry["state"]] = summary.get(entry["state"], 0) + 1
    return {"profileId": profile["profileId"], "installed": installed, "lockPins": {k: lock.get(k) for k in PIN_KEYS} if lock else None,
            "hostedCiObserved": False, "phases": phases, "summary": dict(sorted(summary.items())), "claimBoundary": CLAIM_BOUNDARY}


def control_states(project_root: Path, lib_dir: Path) -> dict[str, dict]:
    """Strongest observed state per control (rank + name) for claim support."""
    coverage = compute_coverage(project_root, lib_dir)
    best: dict[str, dict] = {}
    for row in coverage["phases"].values():
        for cid, entry in row.items():
            rank = RANK.get(entry["state"], 0)
            if cid not in best or rank > best[cid]["rank"]:
                best[cid] = {"state": entry["state"], "rank": rank}
    return best


def format_table(coverage: dict) -> str:
    lines = [f"Downstream gate coverage ({coverage['profileId']}); hostedCiObserved={str(coverage['hostedCiObserved']).lower()}"]
    for phase, row in coverage["phases"].items():
        for cid, entry in row.items():
            lines.append(f"  {phase:<14} {cid:<16} {entry['state']:<28} {entry.get('reason', '')}")
    lines.append("  summary: " + ", ".join(f"{k}={v}" for k, v in coverage["summary"].items()))
    lines.append("  " + coverage["claimBoundary"])
    return "\n".join(lines)


def identity_of(path: Path) -> str:
    return content_sha256(path.read_bytes())
