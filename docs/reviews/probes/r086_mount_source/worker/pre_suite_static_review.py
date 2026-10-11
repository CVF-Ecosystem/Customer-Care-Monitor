"""Static-only R086 worker check; does not invoke tests or native resources."""

from __future__ import annotations

import ast
import hashlib
import json
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[5]
BASELINE_PATH = ROOT / "docs/reviews/probes/r086_mount_source/worker/pre_edit_baseline.json"
SOURCE_PATH = ROOT / "scripts/jobrun_fence_fixture.py"
TEST_PATH = ROOT / "scripts/tests/test_jobrun_fence_fixture.py"
RELEASE = "750d983a388e2ebee43434799f43b6de6d67a4fa"
PROTECTED_SOURCE_NAMES = {
    "TRANCHE_ID",
    "BUILD_ORDER_SNAPSHOT_COMMIT",
    "WORK_ORDER_PATH",
    "SOURCE_PATHS",
}


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def method_bodies(tree: ast.Module) -> dict[tuple[str, str], str]:
    result: dict[tuple[str, str], str] = {}
    for node in tree.body:
        if not isinstance(node, ast.ClassDef):
            continue
        for item in node.body:
            if isinstance(item, ast.FunctionDef) and item.name.startswith("test_"):
                body_module = ast.Module(body=item.body, type_ignores=[])
                result[(node.name, item.name)] = sha256(ast.dump(body_module, include_attributes=False).encode())
    return result


def assigned_constants(tree: ast.Module) -> dict[str, str]:
    values: dict[str, str] = {}
    for node in tree.body:
        if isinstance(node, ast.Assign) and len(node.targets) == 1 and isinstance(node.targets[0], ast.Name):
            values[node.targets[0].id] = ast.dump(node.value, include_attributes=False)
        elif isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name) and node.value is not None:
            values[node.target.id] = ast.dump(node.value, include_attributes=False)
    return values


def main() -> int:
    baseline = json.loads(BASELINE_PATH.read_text(encoding="utf-8"))
    test_bytes = TEST_PATH.read_bytes()
    current_test = ast.parse(test_bytes.decode("utf-8"))
    current_methods = method_bodies(current_test)
    baseline_methods = {
        (entry["class"], entry["name"]): entry["bodyAstSha256"]
        for entry in baseline["originalTestMethodBodies"]
    }
    unchanged = {
        f"{class_name}.{name}": current_methods.get((class_name, name)) == expected
        for (class_name, name), expected in baseline_methods.items()
    }

    source_bytes = SOURCE_PATH.read_bytes()
    current_source = ast.parse(source_bytes.decode("utf-8"))
    release_source_text = subprocess.run(
        ["git", "show", f"{RELEASE}:scripts/jobrun_fence_fixture.py"],
        cwd=ROOT,
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    ).stdout
    release_constants = assigned_constants(ast.parse(release_source_text))
    current_constants = assigned_constants(current_source)
    protected_constant_matches = {
        name: current_constants.get(name) == release_constants.get(name)
        for name in sorted(PROTECTED_SOURCE_NAMES)
    }

    diff_check = subprocess.run(
        ["git", "diff", "--check", "--", "scripts/jobrun_fence_fixture.py", "scripts/tests/test_jobrun_fence_fixture.py"],
        cwd=ROOT,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="utf-8",
    )
    new_methods = sorted(
        f"{class_name}.{name}"
        for class_name, name in set(current_methods) - set(baseline_methods)
    )
    report = {
        "trancheId": "CCMAI-RUNTIME-086",
        "phase": "BUILD_PRE_SUITE_STATIC_ONLY",
        "releaseCommit": RELEASE,
        "testMethodCount": len(current_methods),
        "expectedTestMethodCount": 27,
        "originalMethodCount": len(baseline_methods),
        "originalMethodsUnchanged": all(unchanged.values()) and len(unchanged) == len(baseline_methods),
        "originalMethodBodyChecks": unchanged,
        "newTestMethods": new_methods,
        "protectedSourceConstantsMatchRelease": all(protected_constant_matches.values()),
        "protectedSourceConstantChecks": protected_constant_matches,
        "gitDiffCheckExitCode": diff_check.returncode,
        "gitDiffCheckStdout": diff_check.stdout,
        "gitDiffCheckStderr": diff_check.stderr,
        "sourcePhysicalSha256BeforeSuite": sha256(source_bytes),
        "testPhysicalSha256BeforeSuite": sha256(test_bytes),
        "testInvocations": 0,
        "nativeResourceEffects": 0,
    }
    output_path = ROOT / "docs/reviews/probes/r086_mount_source/worker/pre_suite_static_review.json"
    output_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({key: report[key] for key in (
        "testMethodCount",
        "expectedTestMethodCount",
        "originalMethodCount",
        "originalMethodsUnchanged",
        "newTestMethods",
        "protectedSourceConstantsMatchRelease",
        "gitDiffCheckExitCode",
        "testInvocations",
        "nativeResourceEffects",
    )}, indent=2))
    if len(current_methods) != 27 or not report["originalMethodsUnchanged"]:
        return 1
    if len(new_methods) != 4 or not report["protectedSourceConstantsMatchRelease"]:
        return 1
    if diff_check.returncode != 0:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
