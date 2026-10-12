#!/usr/bin/env python3
"""Portable downstream gate runner (cvf.downstreamGateProfile@1.1.0, stdlib only).

Core-owned: scripts/lib/downstream_governance/. A project copy lives in
scripts/cvf_gates/ and is identity-pinned by .cvf/gate-profile.lock.json.

Trust anchors (see the standard): INDEPENDENT_CORE - the doctor runs the Core copy of this
file with --trusted and compares every project byte to Core sources before any project module
is imported; CI_BUNDLE_PIN - the PR command carries profile, runner and bundle digests which this
file checks against the ACTUAL bytes of its own directory (stage 0, stdlib only) before importing
any sibling module. The CI pin lives in project-controlled YAML, so it is tamper evidence for
inconsistent edits, not an independent trust root. Fully replacing this entrypoint, or invoking
a project out of band, is not intercepted.
"""

from __future__ import annotations

import sys

sys.dont_write_bytecode = True

import argparse
import hashlib
import importlib
import json
import os
import subprocess
from datetime import datetime, timezone
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
EXIT_PASS, EXIT_GATE_FAIL, EXIT_BLOCKED = 0, 2, 3
PROFILE_FILE = "cvf_downstream_gate_profile.json"
RUNNER_FILE = "cvf_downstream_gate_runner.py"
CI_REQUIRED_PINS = ("expect_profile_sha256", "expect_runner_sha256", "expect_bundle_sha256")


def _lf_sha(data: bytes) -> str:
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def _stage0_bundle(directory: Path) -> tuple[str, dict[str, str]]:
    """Digest of every file in this directory; a stdlib-only twin of cvf_dg_common.bundle_digest (parity-tested)."""
    files = {p.relative_to(directory).as_posix(): _lf_sha(p.read_bytes()) for p in directory.rglob("*") if p.is_file()}
    files = dict(sorted(files.items()))  # ordinal order of the posix name: identical on Windows and Linux
    listing = "".join(f"{k}:{v}\n" for k, v in files.items())
    return hashlib.sha256(listing.encode("utf-8")).hexdigest(), files


def _stage0(args: argparse.Namespace) -> list[str]:
    """Verify the CI pins against actual bytes BEFORE any sibling module is imported (project copy only)."""
    problems: list[str] = []
    if args.phase == "pr-ci":
        problems += [f"PIN_REQUIRED @ --{name.replace('_', '-')}: the pr-ci phase must be pinned" for name in CI_REQUIRED_PINS if not getattr(args, name)]
    bundle, files = _stage0_bundle(HERE)
    for label, expected, actual in (("PROFILE_PIN_MISMATCH", args.expect_profile_sha256, files.get(PROFILE_FILE)),
                                    ("RUNNER_PIN_MISMATCH", args.expect_runner_sha256, files.get(RUNNER_FILE)),
                                    ("BUNDLE_PIN_MISMATCH", args.expect_bundle_sha256, bundle)):
        if expected and actual != expected:
            problems.append(f"{label} @ {HERE.name}: actual bytes differ from the expected pin (checked before importing project modules)")
    return problems


def _import_modules() -> dict:
    names = ("cvf_dg_applicability", "cvf_dg_common", "cvf_dg_continuity", "cvf_dg_coverage", "cvf_dg_install",
             "cvf_dg_intake", "cvf_dg_review", "cvf_dg_routing")
    return {name: importlib.import_module(name) for name in names}


def _head(root: Path) -> str | None:
    if not (root / ".git").exists():
        return None
    result = subprocess.run(["git", "rev-parse", "HEAD"], cwd=root, text=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
    return result.stdout.strip() or None if result.returncode == 0 else None


def _range_result(M: dict, candidates, require: bool):
    common = M["cvf_dg_common"]
    notes = [candidates.summary()] + [f"deleted candidate not checked: {p}" for p in candidates.deleted]
    if candidates.findings:
        return common.ControlResult("CVF-DG-RANGE-01", "FAIL", candidates.findings, notes=notes)
    return common.ControlResult("CVF-DG-RANGE-01", "PASS", [], notes=notes)


def _local_controls(M: dict, root: Path, profile: dict, phase: str) -> list:
    common = M["cvf_dg_common"]
    path = root / profile["projectPaths"]["localOverride"]
    if not path.is_file():
        return []
    try:
        data = common.load_json_strict(path)
    except ValueError:
        return []
    results = []
    for item in data.get("additionalControls", []) if isinstance(data, dict) else []:
        if not isinstance(item, dict) or phase not in item.get("phases", []):
            continue
        cid = str(item.get("id"))
        try:
            code = subprocess.run(item["command"], cwd=root, timeout=300, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True).returncode
        except (OSError, subprocess.SubprocessError) as exc:
            results.append(common.ControlResult(cid, "FAIL", [common.Finding("LOCAL_CONTROL_ERROR", cid, str(exc))]))
            continue
        findings = [] if code == 0 else [common.Finding("LOCAL_CONTROL_FAILED", cid, f"exit {code}")]
        results.append(common.ControlResult(cid, "PASS" if code == 0 else "FAIL", findings, notes=["local non-mandatory control"]))
    return results


def run_phase(args: argparse.Namespace) -> int:
    root = Path(args.project_root).resolve()
    trusted = bool(args.trusted)
    if not trusted:
        problems = _stage0(args)
        if problems:
            print("[FAIL] CVF-DG-INST-01 stage-0 pin verification")
            for line in problems:
                print("       " + line)
            print(f"RESULT: FAIL phase={args.phase} stage0=REFUSED")
            return EXIT_GATE_FAIL
    try:
        M = _import_modules()
    except Exception as exc:  # a missing, empty or broken inherited module is a blocked gate, never a pass
        print(f"RESULT: BLOCKED_INVALID_INPUT inherited module unusable: {type(exc).__name__}: {exc}")
        return EXIT_BLOCKED
    common, install_m = M["cvf_dg_common"], M["cvf_dg_install"]
    profile = install_m.load_profile(HERE)
    phase = args.phase
    if phase not in profile["phases"]:
        print(f"RESULT: BLOCKED_INVALID_INPUT unknown phase {phase}")
        return EXIT_BLOCKED
    core_root = HERE.parents[2] if trusted else None
    checker_dir = (core_root / "governance" / "compat") if trusted and core_root else HERE
    base = args.base
    if args.base_from_env:
        base = os.environ.get(args.base_from_env, "").strip() or base
    require_range = phase in profile["candidateRange"]["requiredPhases"]
    candidates = common.resolve_candidates(root, base, args.head, require_range)
    if args.base_from_env and not base:
        candidates.findings.append(common.Finding("RANGE_BASE_MISSING", f"env:{args.base_from_env}", "the named base environment variable is unset or empty"))
    ok_range = not candidates.findings

    def packets(prefixes: tuple[str, ...]) -> list[str]:
        return sorted({p for prefix in prefixes for p in common.discover_packets(root, args.packet or [], candidates, prefix)}) if ok_range else []

    results = []
    for control in profile["controls"]:
        if phase not in control["phases"]:
            continue
        cid = control["id"]
        if cid == "CVF-DG-INST-01":
            results.append(install_m.verify_install(root, HERE, core_root, trusted, args.expect_profile_sha256, args.expect_runner_sha256, args.expect_bundle_sha256))
        elif cid == "CVF-DG-RANGE-01":
            results.append(_range_result(M, candidates, require_range))
        elif cid == "CVF-DG-CONT-01":
            results.append(M["cvf_dg_continuity"].check_continuity(root, profile, checker_dir))
        elif cid == "CVF-DG-APPL-01":
            results.append(M["cvf_dg_applicability"].check_applicability(root, packets(("docs/work_orders/",)), checker_dir))
        elif cid == "CVF-DG-ROUTE-01":
            results.append(M["cvf_dg_routing"].check_routing(root, packets(("docs/work_orders/", "docs/reviews/")), profile))
        elif cid == "CVF-DG-ROLE-01":
            results.append(M["cvf_dg_review"].check_roles(root, packets(("docs/reviews/",)), profile))
        elif cid == "CVF-DG-CLAIM-01":
            results.append(M["cvf_dg_review"].check_claims(root, packets(("docs/reviews/", "docs/work_orders/")), profile, HERE))
    results.extend(_local_controls(M, root, profile, phase))
    failed = [r for r in results if not r.ok]
    exit_code = EXIT_PASS if not failed else EXIT_GATE_FAIL
    try:
        lock = common.load_json_strict(root / profile["projectPaths"]["lock"])
    except (OSError, ValueError):
        lock = {}
    lock = lock if isinstance(lock, dict) else {}
    receipt = {
        "schemaVersion": M["cvf_dg_coverage"].RECEIPT_SCHEMA, "profileId": profile["profileId"], "phase": phase,
        "profileSha256": lock.get("profileSha256"), "runnerSha256": lock.get("runnerSha256"), "bundleSha256": lock.get("bundleSha256"),
        "head": _head(root), "executionContext": "trusted-core" if trusted else "project-copy",
        "range": candidates.summary(), "exitCode": exit_code, "invokedAtUtc": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "results": [{"controlId": r.control_id, "outcome": r.outcome, "ok": r.ok, "findingCount": len(r.findings)} for r in results],
    }
    if not args.no_receipt:
        receipt_dir = Path(args.receipt_dir) if args.receipt_dir else root / profile["projectPaths"]["receiptDir"]
        try:
            receipt_dir.mkdir(parents=True, exist_ok=True)
            (receipt_dir / f"{phase}.json").write_bytes(common.canonical_json(receipt).encode("utf-8"))
        except OSError as exc:
            print(f"WARN receipt not written: {exc}")
    if args.json:
        print(json.dumps({"phase": phase, "exitCode": exit_code, "range": candidates.summary(), "results": [r.as_dict() for r in results]}, indent=2, sort_keys=True))
    else:
        for r in results:
            print(f"[{'PASS' if r.ok else 'FAIL'}] {r.control_id} {r.outcome}" + (f" - {r.reason}" if r.reason else ""))
            for f in sorted(r.findings, key=lambda x: (x.locator, x.code)):
                print(f"       {f.code} @ {f.locator}: {f.message}")
            for note in r.notes:
                print(f"       note: {note}")
        print(f"RESULT: {'PASS' if exit_code == EXIT_PASS else 'FAIL'} phase={phase} mandatoryControls={len(results)} failed={len(failed)}")
    return exit_code


def _parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    run = sub.add_parser("run")
    run.add_argument("--phase", required=True)
    run.add_argument("--project-root", default=".")
    run.add_argument("--packet", action="append")
    run.add_argument("--base")
    run.add_argument("--base-from-env", help="name of an environment variable holding the PR/push base commit")
    run.add_argument("--head")
    run.add_argument("--trusted", action="store_true", help="run from the Core copy and compare the project copy to Core sources")
    run.add_argument("--expect-profile-sha256")
    run.add_argument("--expect-runner-sha256")
    run.add_argument("--expect-bundle-sha256")
    run.add_argument("--receipt-dir")
    run.add_argument("--no-receipt", action="store_true")
    run.add_argument("--json", action="store_true")
    cov = sub.add_parser("coverage")
    cov.add_argument("--project-root", default=".")
    cov.add_argument("--proof-receipt")
    cov.add_argument("--json", action="store_true")
    intake = sub.add_parser("intake")
    intake.add_argument("--project-root", default=".")
    intake.add_argument("--finding", required=True)
    check = sub.add_parser("validate-intake")
    check.add_argument("--record", required=True)
    inst = sub.add_parser("install")
    inst.add_argument("--project-root", required=True)
    inst.add_argument("--core-root")
    inst.add_argument("--core-commit", default="UNKNOWN")
    inst.add_argument("--upgrade", action="store_true")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = _parser().parse_args(argv)
    try:
        if args.command == "run":
            return run_phase(args)
        M = _import_modules()
        profile = M["cvf_dg_install"].load_profile(HERE)
        if args.command == "coverage":
            data = M["cvf_dg_coverage"].compute_coverage(Path(args.project_root).resolve(), HERE, Path(args.proof_receipt) if args.proof_receipt else None)
            print(json.dumps(data, indent=2, sort_keys=True) if args.json else M["cvf_dg_coverage"].format_table(data))
            return EXIT_PASS
        if args.command == "intake":
            target, problems = M["cvf_dg_intake"].build_intake(Path(args.finding), Path(args.project_root).resolve(), profile)
            for p in problems:
                print(f"{p.code} @ {p.locator}: {p.message}")
            print(f"INTAKE: {'WRITTEN ' + str(target) if target else 'REFUSED'} (generation is not parent acceptance)")
            return EXIT_PASS if target else EXIT_GATE_FAIL
        if args.command == "validate-intake":
            problems = M["cvf_dg_intake"].validate_record(M["cvf_dg_common"].load_json_strict(Path(args.record)), profile)
            for p in problems:
                print(f"{p.code} @ {p.locator}: {p.message}")
            print(f"INTAKE_RECORD: {'VALID' if not problems else 'INVALID'}")
            return EXIT_PASS if not problems else EXIT_GATE_FAIL
        core_root = Path(args.core_root).resolve() if args.core_root else HERE.parents[2]
        outcome = M["cvf_dg_install"].install(Path(args.project_root).resolve(), HERE, core_root, args.core_commit, args.upgrade)
        print(f"GATE_PROFILE_STATUS: {outcome['status']} {outcome['detail']}")
        return EXIT_PASS if outcome["status"] in ("FRESH_INSTALLED", "ALREADY_INSTALLED", "UPGRADED") else EXIT_GATE_FAIL
    except Exception as exc:  # fail closed: any internal error is a blocked gate, never a pass
        print(f"RESULT: BLOCKED_INVALID_INPUT {type(exc).__name__}: {exc}")
        return EXIT_BLOCKED


if __name__ == "__main__":
    raise SystemExit(main())
