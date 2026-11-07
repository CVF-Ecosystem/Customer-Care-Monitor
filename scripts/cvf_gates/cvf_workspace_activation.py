"""Read-only activation admission; Local reviews semantic authority and invocation truth."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
from pathlib import Path

from cvf_dg_common import (ControlResult, Finding, bundle_digest, content_sha256,
                           load_json_strict, safe_relative)

CONTROL_ID = "CVF-DG-ACT-01"
RECEIPT = ".cvf/workspace-activation.json"
FAMILY = "scripts/lib/downstream_governance"
SUPPLEMENTAL_ID = "SUPPLEMENTAL_ASSESSMENT"


def raw_sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def expected_inventory(profile: dict) -> dict[str, str]:
    return {c["id"]: c["module"] for c in profile["controls"]
            if c.get("mandatory") is True and "pre-dispatch" in c.get("phases", [])
            and c.get("id") != CONTROL_ID}


def _nonempty(value: object) -> bool:
    return isinstance(value, str) and bool(value.strip())


def _path(root: Path, relative: object) -> Path:
    target = safe_relative(root, relative) if isinstance(relative, str) else None
    if not target or not target.is_file() or target.is_symlink():
        raise ValueError("missing or unsafe source/evidence file")
    return target


def _file(root: Path, entry: object) -> bool:
    try:
        return (isinstance(entry, dict) and set(entry) == {"path", "sha256"}
                and raw_sha(_path(root, entry["path"])) == entry["sha256"])
    except (OSError, ValueError, TypeError):
        return False


def _review(root: Path, ref: object, action: dict, purpose: str, control: str) -> bool:
    if not _file(root, ref):
        return False
    try:
        value = load_json_strict(_path(root, ref["path"]))
        return (isinstance(value, dict) and value.get("action") == action
                and value.get("purpose") == purpose and value.get("controlId") == control
                and _nonempty(value.get("reviewer")) and _nonempty(value.get("authority"))
                and value.get("decision") == "ACCEPTED")
    except (OSError, ValueError, TypeError):
        return False


def _reviews(root: Path, refs: object, action: dict, purpose: str, control: str) -> bool:
    return isinstance(refs, list) and any(_review(root, ref, action, purpose, control) for ref in refs)


def action_binding(root: Path, packet: str) -> dict:
    target = _path(root, packet)
    blocks = re.findall(r"(?ms)^" + chr(96) * 3 + r"cvf-milestone-coverage-json\s*(\{.*?\})\s*^" + chr(96) * 3,
                        target.read_text(encoding="utf-8-sig"))
    if len(blocks) != 1:
        raise ValueError("action requires one milestone coverage block")
    coverage = json.loads(blocks[0])
    if not isinstance(coverage, dict) or not isinstance(coverage.get("action"), dict):
        raise ValueError("action coverage must be an object")
    selected = coverage["action"].get("selected")
    if not isinstance(selected, list) or not selected or not all(_nonempty(x) for x in selected) or len(set(selected)) != len(selected):
        raise ValueError("action criteria missing or duplicated")
    return {"packet": packet, "sha256": raw_sha(target), "criteria": sorted(selected)}


def _git(core: Path, *args: str) -> str:
    return subprocess.run(["git", "-C", str(core), *args], check=True,
                          capture_output=True, text=True, timeout=20).stdout.strip()


def _core(root: Path, manifest: dict) -> Path:
    pin, relative = manifest.get("cvfCoreCommit"), manifest.get("cvfCoreRelativePath")
    if not isinstance(pin, str) or not re.fullmatch(r"[0-9a-f]{40}", pin) or not _nonempty(relative):
        raise ValueError("ACT_CORE_INTEGRITY: full Core pin and path required")
    core = (root / relative).resolve()
    if (Path(_git(core, "rev-parse", "--show-toplevel")).resolve() != core
            or _git(core, "rev-parse", "HEAD") != pin or _git(core, "status", "--porcelain")
            or not _nonempty(manifest.get("cvfCoreRepository"))
            or _git(core, "remote", "get-url", "origin") != manifest["cvfCoreRepository"]):
        raise ValueError("ACT_CORE_INTEGRITY: Core HEAD, cleanliness or origin differs")
    return core


def _delivery(root: Path, source_policy: str, core_only: bool) -> tuple[dict, Path | None, set[Path]]:
    if source_policy == "CORE_ONLY" or core_only:
        return {"mode": "CORE_ONLY", "pack": None}, None, set()
    workspace = root.parent
    active_path = _path(workspace, "CVF_RULE_PACKS/ACTIVE_RULE_PACK.json")
    active = load_json_strict(active_path)
    if not isinstance(active, dict):
        raise ValueError("supplemental active manifest must be an object")
    pack_path = _path(workspace, active.get("activeProfilePath", "").replace("\\", "/"))
    pack = load_json_strict(pack_path)
    if not isinstance(pack, dict):
        raise ValueError("supplemental pack manifest must be an object")
    if (active.get("activeProfile") != pack.get("profileName")
            or not _nonempty(active.get("sourceCommit")) or active["sourceCommit"] != pack.get("sourceCommit")):
        raise ValueError("supplemental pack identity mismatch")
    artifacts = pack.get("artifacts")
    if (not isinstance(artifacts, list) or type(pack.get("artifactCount")) is not int
            or len(artifacts) != pack["artifactCount"] or len(artifacts) != active.get("artifactCount")):
        raise ValueError("supplemental pack artifact inventory mismatch")
    verified = set()
    for item in artifacts:
        if not isinstance(item, dict) or not isinstance(item.get("targetPath"), str):
            raise ValueError("invalid supplemental artifact")
        target = safe_relative(workspace, item["targetPath"].replace("\\", "/"))
        expected = item.get("sourceSha256")
        if not target or target.is_symlink():
            raise ValueError("unsafe supplemental artifact")
        if target.is_file():
            if target in verified or not isinstance(expected, str) or raw_sha(target) != expected.lower():
                raise ValueError("supplemental artifact mismatch")
            verified.add(target)
        elif not target.is_dir() or expected is not None:
            raise ValueError("supplemental artifact hash missing")
    source = pack_path.parent / "source"
    return {"mode": "CORE_AND_SUPPLEMENTAL", "activeSha256": raw_sha(active_path),
            "manifestSha256": raw_sha(pack_path)}, source, verified


def _source_bundle(root: Path, source: Path, profile: dict, lock: dict, verified: set[Path] | None) -> None:
    source_profile = _path(source, f"{FAMILY}/cvf_downstream_gate_profile.json")
    if load_json_strict(source_profile) != profile or content_sha256(source_profile.read_bytes()) != lock.get("profileSha256"):
        raise ValueError("ACT_SOURCE_INVENTORY: admitted bundle differs from actual source profile")
    names = profile.get("installedFiles")
    if not isinstance(names, list) or not names or set(names) != set(lock.get("files", {})):
        raise ValueError("ACT_SOURCE_INVENTORY: source/installed inventory differs")
    digest, files = bundle_digest(root / profile["projectPaths"]["gatesDir"])
    if digest != lock.get("bundleSha256") or files != lock["files"]:
        raise ValueError("ACT_SOURCE_INVENTORY: installed bundle changed")
    for name in names:
        target = _path(source, profile.get("coreSources", {}).get(name, f"{FAMILY}/{name}"))
        if verified is not None and target not in verified:
            raise ValueError("ACT_SOURCE_INVENTORY: source input not individually verified")
        if content_sha256(target.read_bytes()) != files[name]:
            raise ValueError("ACT_SOURCE_INVENTORY: installed module differs from actual source")
    if files.get("cvf_downstream_gate_runner.py") != lock.get("runnerSha256") or lock.get("profileId") != profile.get("profileId"):
        raise ValueError("ACT_SOURCE_INVENTORY: runner/profile lock differs")


def comparison(root: Path, profile: dict, core_only: bool = False, trusted_core: Path | None = None) -> dict:
    """CORE_ONLY never reads a pack. Permitted supplemental mode verifies its bytes."""
    manifest = load_json_strict(root / ".cvf/manifest.json")
    policy = load_json_strict(root / ".cvf/policy.json")
    lock = load_json_strict(root / profile["projectPaths"]["lock"])
    if not all(isinstance(value, dict) for value in (manifest, policy, lock)):
        raise ValueError("manifest, policy and lock must be objects")
    source_policy = policy.get("workspaceActivationSourcePolicy")
    executable = policy.get("workspaceActivationExecutableSource", "PINNED_CORE")
    if source_policy not in ("CORE_ONLY", "CORE_AND_SUPPLEMENTAL") or executable not in ("PINNED_CORE", "OPERATOR_LOCAL_PACK"):
        raise ValueError("project must explicitly declare valid source policy")
    core = _core(root, manifest)
    delivery, supplemental_source, verified = _delivery(root, source_policy, core_only)
    route, source = "PINNED_CORE", core
    if executable == "OPERATOR_LOCAL_PACK" and not (core_only or source_policy == "CORE_ONLY"):
        ref = policy.get("workspaceActivationUpgrade")
        if not _file(root, ref):
            raise ValueError("ACT_UPGRADE_UNPROVED: project policy must pin reviewed upgrade authority")
        upgrade = load_json_strict(_path(root, ref["path"]))
        if not isinstance(upgrade, dict):
            raise ValueError("ACT_UPGRADE_UNPROVED: upgrade record must be an object")
        expected = {"projectCoreCommit": manifest["cvfCoreCommit"], "deliveryManifestSha256": delivery.get("manifestSha256"),
                    "profileSha256": lock.get("profileSha256"), "runnerSha256": lock.get("runnerSha256"), "bundleSha256": lock.get("bundleSha256")}
        if (upgrade.get("schemaVersion") != "cvf.workspaceActivationUpgrade@1.0.0"
                or upgrade.get("expected") != expected or upgrade.get("decision") != "ACCEPTED"
                or not _nonempty(upgrade.get("reviewer")) or not _nonempty(upgrade.get("authority"))
                or not _file(root, upgrade.get("reviewEvidence"))):
            raise ValueError("ACT_UPGRADE_UNPROVED: reviewed upgrade pins/evidence differ")
        route, source = "AUTHORIZED_SUPPLEMENTAL", supplemental_source
    if trusted_core is not None and source != trusted_core.resolve():
        raise ValueError("ACT_CORE_PATH_MISMATCH: execution source differs from independently trusted source")
    _source_bundle(root, source, profile, lock, verified if route == "AUTHORIZED_SUPPLEMENTAL" else None)
    delivery_hash = hashlib.sha256(json.dumps(delivery, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    identity = {"projectRootSha256": hashlib.sha256(str(root.resolve()).encode()).hexdigest(),
                "manifestSha256": raw_sha(root / ".cvf/manifest.json"), "policySha256": raw_sha(root / ".cvf/policy.json"),
                "lockSha256": raw_sha(root / profile["projectPaths"]["lock"]), "coreCommit": manifest["cvfCoreCommit"],
                "profileId": profile["profileId"], "profileSha256": lock["profileSha256"],
                "deliveryFingerprint": delivery_hash, "sourceRoute": route}
    return {"identity": identity, "sourcePolicy": source_policy, "delivery": delivery,
            "expectedControls": expected_inventory(profile),
            "supplementalRequirement": {"id": SUPPLEMENTAL_ID, "mode": "NOT_INSPECTED" if delivery["mode"] == "CORE_ONLY" else "INSPECTED",
                                        "sourceSha256": delivery.get("manifestSha256")},
            "note": "Pinned declarations and verified bytes only; Local reviews authority and invocation truth."}


def check_activation(root: Path, profile: dict, packets: list[str], phase: str, trusted_core: Path | None = None) -> ControlResult:
    findings: list[Finding] = []
    def fail(code: str, detail: str) -> None:
        findings.append(Finding(code, RECEIPT, detail))
    try:
        receipt = load_json_strict(root / RECEIPT)
        if not isinstance(receipt, dict) or receipt.get("schemaVersion") != "cvf.workspaceActivation@1.0.0":
            raise ValueError("invalid activation receipt schema")
        observed = comparison(root, profile, trusted_core=trusted_core)
        if receipt.get("identity") != observed["identity"]:
            fail("ACT_IDENTITY_STALE", "project, policy, source or installed bundle binding differs")
        if receipt.get("sourcePolicy") != observed["sourcePolicy"]:
            fail("ACT_POLICY_CONFLICT", "receipt source policy differs from project policy")
        if not _nonempty(receipt.get("reviewer")) or not _nonempty(receipt.get("authority")):
            fail("ACT_REVIEW_MISSING", "reviewer and authority required")
        bindings = receipt.get("actions")
        if not isinstance(bindings, list) or not bindings:
            raise ValueError("action bindings required")
        by_packet = {}
        for entry in bindings:
            if not isinstance(entry, dict) or entry != action_binding(root, entry.get("packet")) or entry["packet"] in by_packet:
                raise ValueError("ACT_ACTION_STALE: exact action bytes/criteria differ or duplicated")
            by_packet[entry["packet"]] = entry
        selected = packets if phase == "pre-dispatch" else list(by_packet)
        if not selected or any(packet not in by_packet for packet in selected):
            raise ValueError("ACT_ACTION_MISSING: current action bindings required")
        supplemental = receipt.get("supplemental")
        if not isinstance(supplemental, list) or len(supplemental) != 1 or not isinstance(supplemental[0], dict):
            raise ValueError("ACT_SUPPLEMENTAL_UNASSESSED: mandatory owned supplemental assessment missing")
        item = supplemental[0]
        if (any(item.get(key) != value for key, value in observed["supplementalRequirement"].items())
                or not _nonempty(item.get("owner")) or not isinstance(item.get("actions"), list) or not item["actions"]):
            raise ValueError("ACT_SUPPLEMENTAL_INVALID: independently derived assessment identity differs")
        held = item.get("disposition") == "HELD"
        if (observed["delivery"]["mode"] == "CORE_ONLY" and not held) or item.get("disposition") not in ("HELD", "ASSESSED"):
            raise ValueError("ACT_SUPPLEMENTAL_INVALID: uninspected coverage remains held")
        for packet in selected:
            if held and packet in item["actions"]:
                fail("ACT_SUPPLEMENTAL_HELD", "selected action depends on held coverage")
            elif held and not _reviews(root, item.get("independenceEvidence"), by_packet[packet], "INDEPENDENCE", SUPPLEMENTAL_ID):
                fail("ACT_INDEPENDENCE_UNPROVED", "unknown supplemental coverage requires reviewed action independence")
            elif not held and not _reviews(root, item.get("reviewEvidence"), by_packet[packet], "SUPPLEMENTAL_ASSESSMENT", SUPPLEMENTAL_ID):
                fail("ACT_SUPPLEMENTAL_UNASSESSED", "inspected manifest needs reviewed action-specific assessment")
        inventory, rows = expected_inventory(profile), receipt.get("controls")
        if not isinstance(rows, list):
            raise ValueError("controls must be an array")
        by_id = {r.get("id"): r for r in rows if isinstance(r, dict) and isinstance(r.get("id"), str)}
        if len(by_id) != len(rows) or set(by_id) != set(inventory):
            fail("ACT_INVENTORY_MISMATCH", "every independently verified source control required once")
        lock = load_json_strict(root / profile["projectPaths"]["lock"])
        for cid, module in inventory.items():
            row = by_id.get(cid)
            if not isinstance(row, dict):
                continue
            if (not all(_nonempty(row.get(k)) for k in ("owner", "authority", "reason"))
                    or not isinstance(row.get("actions"), list) or not row["actions"]
                    or not isinstance(row.get("criteria"), list) or not row["criteria"] or not _file(root, row.get("evidence"))):
                fail("ACT_CONTROL_INCOMPLETE", cid)
                continue
            for packet in selected:
                binding = by_packet[packet]
                if packet not in row["actions"]:
                    if not _reviews(root, row.get("independenceEvidence"), binding, "INDEPENDENCE", cid):
                        fail("ACT_INDEPENDENCE_UNPROVED", cid)
                    continue
                if binding not in row.get("actionBindings", []) or not set(binding["criteria"]) <= set(row["criteria"]):
                    fail("ACT_CRITERIA_UNASSESSED", cid)
                disposition = row.get("disposition")
                if disposition == "APPLIED":
                    source = root / profile["projectPaths"]["gatesDir"] / (module + ".py")
                    if (not source.is_file() or content_sha256(source.read_bytes()) != lock["files"].get(module + ".py")
                            or not _file(root, row.get("configuredInvocation"))
                            or not _reviews(root, row.get("invocationEvidence"), binding, "INVOCATION", cid)):
                        fail("ACT_APPLIED_UNPROVED", cid)
                    else:
                        invocation = _path(root, row["configuredInvocation"]["path"]).read_text(encoding="utf-8-sig")
                        if "cvf_downstream_gate_runner.py" not in invocation or "pre-dispatch" not in invocation:
                            fail("ACT_INVOCATION_UNBOUND", cid)
                elif disposition == "NOT_APPLICABLE_WITH_EVIDENCE":
                    pass
                elif disposition in ("HELD", "EXPLICIT_UPDATE_REQUIRED"):
                    fail("ACT_CONTROL_HELD", cid)
                else:
                    fail("ACT_DISPOSITION_INVALID", cid)
    except (OSError, ValueError, TypeError, KeyError, subprocess.SubprocessError) as exc:
        detail = str(exc)
        code = detail.split(":", 1)[0] if re.match(r"^ACT_[A-Z_]+:", detail) else "ACT_INVALID_INPUT"
        fail(code, detail)
    return ControlResult(CONTROL_ID, "FAIL" if findings else "PASS", findings,
                         notes=["Mechanical evidence bindings; Local reviews semantic authority and invocation truth."])


def main() -> int:
    parser = argparse.ArgumentParser(description="Read-only project activation comparison")
    parser.add_argument("--project", required=True)
    parser.add_argument("--core-only", action="store_true", help="Do not inspect supplemental sources")
    args = parser.parse_args()
    try:
        profile = load_json_strict(Path(__file__).with_name("cvf_downstream_gate_profile.json"))
        root = Path(args.project).resolve()
        result = comparison(root, profile, args.core_only)
        if args.core_only and result["sourcePolicy"] != "CORE_ONLY":
            result["status"] = "REVIEW_REQUIRED_SUPPLEMENTAL_NOT_INSPECTED"
        else:
            checked = check_activation(root, profile, [], "bootstrap")
            result["status"] = "CURRENT_ASSESSED" if checked.ok else "REVIEW_REQUIRED"
            result["findings"] = [f.as_dict() for f in checked.findings]
        print(json.dumps(result, indent=2))
        return 0 if result["status"] == "CURRENT_ASSESSED" else 2
    except (OSError, ValueError, TypeError, KeyError, subprocess.SubprocessError) as exc:
        print(json.dumps({"status": "BLOCKED_INVALID_INPUT", "error": str(exc)}))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
