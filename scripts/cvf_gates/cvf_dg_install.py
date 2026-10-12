"""CVF-DG-INST-01: install, pin and verify the inherited gate profile (stdlib only)."""

from __future__ import annotations

import os
import re
from pathlib import Path

from cvf_dg_common import (
    PROFILE_FILE, PROFILE_ID, ControlResult, Finding, bundle_digest, canonical_json, content_sha256,
    file_content_sha256, load_json_strict,
)

CONTROL_ID = "CVF-DG-INST-01"
LOCK_SCHEMA = "cvf.downstreamGateLock@1.0.0"
CI_TEMPLATE = "downstream_pr_gates.yml.template"
OVERRIDE_ENV_VARS = ("CVF_DG_SKIP", "CVF_DOWNSTREAM_GATES_SKIP", "CVF_DG_DISABLE")
LEARNING_HOME = "docs/reviews/learnings"
LEARNING_TEMPLATES = {"README.md": "project_learning_home.template.md", "LEARNING_RECORD_TEMPLATE.md": "project_learning_record.template.md"}
LOCAL_ALLOWED_KEYS = {"schemaVersion", "additionalControls"}
LOCAL_CONTROL_KEYS = {"id", "phases", "command"}


def load_profile(lib_dir: Path) -> dict:
    return load_json_strict(lib_dir / PROFILE_FILE)  # type: ignore[return-value]


def _source_for(name: str, profile: dict, lib_dir: Path, core_root: Path | None) -> Path | None:
    mapped = profile.get("coreSources", {}).get(name)
    if mapped:
        return (core_root / mapped) if core_root else None
    return lib_dir / name


def _render_workflow(template: str, profile_sha: str, runner_sha: str, bundle_sha: str) -> str:
    return (template.replace("{{PROFILE_SHA256}}", profile_sha).replace("{{RUNNER_SHA256}}", runner_sha)
            .replace("{{BUNDLE_SHA256}}", bundle_sha))


def check_learning_home(project_root: Path) -> list[Finding]:
    """Structure only: editable project learning is never hashed as Core authority."""
    home = project_root / LEARNING_HOME
    problems = []
    for name in LEARNING_TEMPLATES:
        target = home / name
        if not target.resolve().is_relative_to(project_root.resolve()):
            problems.append(Finding("LEARNING_HOME_OUTSIDE_PROJECT", f"{LEARNING_HOME}/{name}", "learning home must stay inside the project"))
        elif not target.is_file() or not target.read_bytes().strip():
            problems.append(Finding("LEARNING_HOME_INCOMPLETE", f"{LEARNING_HOME}/{name}", "missing or empty learning discovery/template; deliberate migration required"))
    return problems


def install_learning_home(project_root: Path, lib_dir: Path) -> list[Finding]:
    """Prepare only missing scaffolds, never replace project-authored learning."""
    home = project_root / LEARNING_HOME
    for ancestor in (home, *home.parents):
        if ancestor == project_root.parent:
            break
        if ancestor.exists() and not ancestor.is_dir():
            return [Finding("LEARNING_HOME_INCOMPLETE", str(ancestor), "learning directory conflicts with existing file; owner repair required")]
    for name, source in LEARNING_TEMPLATES.items():
        target = home / name
        if not target.resolve().is_relative_to(project_root.resolve()):
            return [Finding("LEARNING_HOME_OUTSIDE_PROJECT", str(target), "refusing write outside project")]
        if target.exists() and (not target.is_file() or not target.read_bytes().strip()):
            return [Finding("LEARNING_HOME_INCOMPLETE", str(target), "existing empty/conflicting content preserved; owner repair required")]
        if not (lib_dir / source).is_file() or not (lib_dir / source).read_bytes().strip():
            return [Finding("LEARNING_TEMPLATE_MISSING", source, "Core learning template unavailable")]
    home.mkdir(parents=True, exist_ok=True)
    for name, source in LEARNING_TEMPLATES.items():
        target = home / name
        if not target.exists():
            target.write_bytes((lib_dir / source).read_bytes())
    return check_learning_home(project_root)


def install(project_root: Path, lib_dir: Path, core_root: Path, core_commit: str,
            upgrade: bool = False) -> dict:
    """Idempotently install the pinned runner/profile/CI template; never overwrite silently."""
    profile = load_profile(lib_dir)
    paths = profile["projectPaths"]
    gates = project_root / paths["gatesDir"]
    lock_path = project_root / paths["lock"]
    sources: dict[str, Path] = {}
    for name in profile["installedFiles"]:
        source = _source_for(name, profile, lib_dir, core_root)
        if source is None or not source.is_file() or source.stat().st_size == 0:
            return {"status": "BLOCKED_CORE_SOURCE_MISSING", "detail": f"missing or empty core source for {name}"}
        sources[name] = source
    template_path = lib_dir / CI_TEMPLATE
    if not template_path.is_file():
        return {"status": "BLOCKED_CORE_SOURCE_MISSING", "detail": f"missing {CI_TEMPLATE}"}
    if lock_path.exists() and not upgrade:
        problems = verify_install(project_root, lib_dir, core_root).findings
        if problems:
            return {"status": "DRIFT_PRESERVED", "detail": "; ".join(f"{p.code}@{p.locator}" for p in problems[:6])}
        return {"status": "ALREADY_INSTALLED", "detail": "lock present and identity verified"}
    if not upgrade:
        collisions = [n for n in sources if (gates / n).exists()]
        collisions += [p.relative_to(gates).as_posix() for p in gates.rglob("*") if p.is_file() and p.relative_to(gates).as_posix() not in sources] if gates.is_dir() else []
        if collisions:
            return {"status": "BLOCKED_COLLISION", "detail": "project-owned files exist: " + ", ".join(sorted(collisions))}
    learning_problems = install_learning_home(project_root, lib_dir)
    if learning_problems:
        return {"status": "BLOCKED_LEARNING_HOME", "detail": "; ".join(f"{p.code}@{p.locator}" for p in learning_problems)}
    gates.mkdir(parents=True, exist_ok=True)
    files: dict[str, str] = {}
    for name, source in sources.items():
        data = source.read_bytes()
        (gates / name).write_bytes(data)
        files[name] = content_sha256(data)
    profile_sha = files[PROFILE_FILE]
    runner_sha = files["cvf_downstream_gate_runner.py"]
    bundle_sha, _ = bundle_digest(gates)
    workflow = project_root / paths["ciWorkflow"]
    rendered = _render_workflow(template_path.read_text(encoding="utf-8"), profile_sha, runner_sha, bundle_sha)
    if workflow.exists():
        disposition = "PROJECT_OWNED_PRESERVED"
    else:
        workflow.parent.mkdir(parents=True, exist_ok=True)
        workflow.write_bytes(rendered.replace("\r\n", "\n").encode("utf-8"))
        disposition = "CREATED"
    lock = {
        "schemaVersion": LOCK_SCHEMA, "profileId": PROFILE_ID, "profileVersion": profile["profileVersion"],
        "hashAlgorithm": "sha256-lf-normalized", "gatesDir": paths["gatesDir"], "coreCommit": core_commit,
        "files": files, "profileSha256": profile_sha, "runnerSha256": runner_sha, "bundleSha256": bundle_sha,
        "ciWorkflow": {"path": paths["ciWorkflow"], "disposition": disposition,
                       "renderedSha256": content_sha256(rendered.encode("utf-8"))},
    }
    lock_path.parent.mkdir(parents=True, exist_ok=True)
    lock_path.write_bytes(canonical_json(lock).encode("utf-8"))
    return {"status": "UPGRADED" if upgrade else "FRESH_INSTALLED", "detail": f"profile {profile_sha[:12]} runner {runner_sha[:12]} ci {disposition}"}


def _check_local_override(project_root: Path, profile: dict) -> list[Finding]:
    path = project_root / profile["projectPaths"]["localOverride"]
    if not path.exists():
        return []
    locator = profile["projectPaths"]["localOverride"]
    try:
        data = load_json_strict(path)
    except ValueError as exc:
        return [Finding("OVERRIDE_REFUSED", locator, f"unparseable local override: {exc}")]
    if not isinstance(data, dict):
        return [Finding("OVERRIDE_REFUSED", locator, "local override must be a JSON object")]
    findings = [Finding("OVERRIDE_REFUSED", f"{locator}#/{key}", "local profile may only add controls; it cannot disable, redefine or skip mandatory controls")
                for key in sorted(set(data) - LOCAL_ALLOWED_KEYS)]
    mandatory = {c["id"] for c in profile["controls"]}
    controls = data.get("additionalControls", [])
    if not isinstance(controls, list):
        return findings + [Finding("OVERRIDE_REFUSED", f"{locator}#/additionalControls", "must be a list")]
    for index, item in enumerate(controls):
        where = f"{locator}#/additionalControls/{index}"
        if not isinstance(item, dict) or set(item) - LOCAL_CONTROL_KEYS or not LOCAL_CONTROL_KEYS <= set(item):
            findings.append(Finding("OVERRIDE_REFUSED", where, "control needs exactly id, phases, command"))
            continue
        cid = item["id"]
        if cid in mandatory or not isinstance(cid, str) or not re.fullmatch(r"LOCAL-[A-Za-z0-9_-]+", cid):
            findings.append(Finding("OVERRIDE_REFUSED", where, "local control ids must match LOCAL-* and cannot collide with mandatory ids"))
        phases = item["phases"]
        if not isinstance(phases, list) or not phases or not set(phases) <= set(profile["phases"]):
            findings.append(Finding("OVERRIDE_REFUSED", where, "phases must be a non-empty subset of the profile phases"))
        command = item["command"]
        if not isinstance(command, list) or not command or not all(isinstance(c, str) and c for c in command):
            findings.append(Finding("OVERRIDE_REFUSED", where, "command must be a non-empty list of non-empty strings"))
    return findings


def verify_install(project_root: Path, lib_dir: Path, core_root: Path | None = None,
                   trusted: bool = False, expect_profile_sha: str | None = None,
                   expect_runner_sha: str | None = None, expect_bundle_sha: str | None = None) -> ControlResult:
    """Verify the project's pinned copies. `trusted` compares against the Core source files."""
    profile = load_profile(lib_dir)
    paths = profile["projectPaths"]
    findings: list[Finding] = []
    for var in OVERRIDE_ENV_VARS:
        if os.environ.get(var):
            findings.append(Finding("OVERRIDE_REFUSED", f"env:{var}", "environment override of a mandatory gate is refused"))
    lock_rel = paths["lock"]
    lock_path = project_root / lock_rel
    if not lock_path.is_file():
        findings.append(Finding("LOCK_MISSING", lock_rel, "gate profile lock is missing; profile is not installed"))
        return ControlResult(CONTROL_ID, "FAIL", findings)
    try:
        lock = load_json_strict(lock_path)
    except ValueError as exc:
        return ControlResult(CONTROL_ID, "FAIL", findings + [Finding("LOCK_INVALID", lock_rel, str(exc))])
    if not isinstance(lock, dict) or lock.get("schemaVersion") != LOCK_SCHEMA or lock.get("profileId") != PROFILE_ID \
            or not isinstance(lock.get("files"), dict):
        return ControlResult(CONTROL_ID, "FAIL", findings + [Finding("LOCK_INVALID", lock_rel, "lock schema or profile pin mismatch")])
    gates = project_root / paths["gatesDir"]
    expected_set = set(profile["installedFiles"])
    if set(lock["files"]) != expected_set:
        findings.append(Finding("FILE_SET_MISMATCH", lock_rel + "#/files", "locked file set differs from the profile installedFiles"))
    for name in sorted(expected_set):
        locator = f"{paths['gatesDir']}/{name}"
        target = gates / name
        if not target.is_file():
            findings.append(Finding("FILE_MISSING", locator, "mandatory inherited file is missing"))
            continue
        if target.stat().st_size == 0:
            findings.append(Finding("FILE_EMPTY", locator, "mandatory inherited file is empty"))
            continue
        actual = file_content_sha256(target)
        if actual != lock["files"].get(name):
            findings.append(Finding("FILE_TAMPERED", locator, "content identity differs from the lock pin"))
        if trusted:
            source = _source_for(name, profile, lib_dir, core_root)
            if source is None or not source.is_file():
                findings.append(Finding("TRUSTED_SOURCE_MISSING", locator, "trusted Core source is unavailable"))
            elif file_content_sha256(source) != actual:
                findings.append(Finding("TRUSTED_MISMATCH", locator, "project copy differs from the trusted Core source"))
    actual_bundle, listing = bundle_digest(gates) if gates.is_dir() else ("", {})
    for extra in sorted(set(listing) - expected_set):
        findings.append(Finding("UNEXPECTED_FILE", f"{paths['gatesDir']}/{extra}", "file is not part of the pinned inherited set (a shadow module or cache would be imported or trusted)"))
    if actual_bundle and lock.get("bundleSha256") != actual_bundle:
        findings.append(Finding("BUNDLE_MISMATCH", lock_rel + "#/bundleSha256", "actual bytes of the inherited directory differ from the locked bundle digest"))
    # Expected pins are compared with the ACTUAL bytes on disk, never with editable lock metadata.
    for label, expected, actual in (("PROFILE_PIN_MISMATCH", expect_profile_sha, listing.get(PROFILE_FILE)),
                                    ("RUNNER_PIN_MISMATCH", expect_runner_sha, listing.get("cvf_downstream_gate_runner.py")),
                                    ("BUNDLE_PIN_MISMATCH", expect_bundle_sha, actual_bundle)):
        if expected and actual != expected:
            findings.append(Finding(label, paths["gatesDir"], "actual inherited bytes differ from the expected CI pin"))
    ci = lock.get("ciWorkflow", {})
    if isinstance(ci, dict) and ci.get("disposition") == "CREATED":
        workflow = project_root / ci.get("path", "")
        if not workflow.is_file() or content_sha256(workflow.read_bytes()) != ci.get("renderedSha256"):
            findings.append(Finding("CI_WORKFLOW_DRIFT", str(ci.get("path")), "generated PR workflow is missing or edited from its pinned rendering"))
    findings.extend(_check_local_override(project_root, profile))
    findings.extend(check_learning_home(project_root))
    return ControlResult(CONTROL_ID, "FAIL" if findings else "PASS", findings)
