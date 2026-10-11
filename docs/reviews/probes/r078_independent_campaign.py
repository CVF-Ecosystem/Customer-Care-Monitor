"""R078 bounded independent reviewer campaign for the committed scheduler startup change.

Offline synthetic MySQL only: cached images/modules, internal Docker network,
read-only source/cache mounts, no host ports, no provider/channel calls. The
runner executes one build and exactly three test commands; it never retries.
"""
import argparse
import base64
import difflib
import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import subprocess
import tarfile
import tempfile
import time
import uuid


EXPECTED_FILES = {
    "backend/engine/scheduler.go",
    "backend/engine/scheduler_startup_hardening_test.go",
}
LEGACY_BASELINE = "51d09cd4cb9282f22518ac32a4edbd3226d56e36"
FIRST_SOURCE_PACKET_COMMIT = "6219da376c63d806ce412bfbae2a738a0af529bc"
LEGACY_HELPER_START = "// cleanupStuckRuns marks any job_runs stuck in \"running\" status as failed."
LEGACY_HELPER_END = "\n// Stop gracefully shuts down the scheduler."
LEGACY_CALL = (
    "\t// Safety net: mark any stuck \"running\" jobs as failed on startup\n"
    "\tcleanupStuckRuns()\n"
)
SYNTHETIC_DB_PASSWORD = "r078_synthetic_fixture"  # cvf-allow-secret-fixture
TEST_DSN = (
    "ccma:" + SYNTHETIC_DB_PASSWORD
    + "@tcp({database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC"
)
LEGACY_CALL_LINE = re.compile(r"(?m)^[\t ]*cleanupStuckRuns\(\)[\t ]*$")
LEGACY_DEFINITION_LINE = re.compile(r"(?m)^func cleanupStuckRuns\(\)[\t ]*\{")
TEST_NAMES = [
    "TestSH01LiveOwnerStartPreservesRowAndPersistedBusyAdmission",
    "TestSH02StartupPreservesUnknownRunningAndTerminalRows",
    "TestSH03FreeJobAdmissionAndOwnerCancellationContrast",
    "TestSH04StartPreservesSyncLeaseRecoveryControls",
    "TestSH05UnknownOwnershipAndReservationFaultFailClosed",
    "TestSH06CronLoadQueryFaultStillStartsScheduler",
    "TestSH07SchedulerLoadsControlledCronAndStopsWithoutExternalEffects",
    "TestCronAndAfterSyncSkipAnOccupiedJob",
    "TestCronVersusAfterSyncAdmitExactlyOne",
    "TestSchedulerOwnersAreCancellableByRun",
    "TestSchedulerRecoversExpiredLeaseAndReadmitsAfterThrottle",
    "TestRecoveryReleasesOnlyExpiredAllowlistedR016Runs",
]
POSITIVE_PATTERN = "^(" + "|".join(TEST_NAMES) + ")$"
M01_PATTERN = (
    "^(TestSH02StartupPreservesUnknownRunningAndTerminalRows|"
    "TestSH04StartPreservesSyncLeaseRecoveryControls)$"
)
M01_DETECTOR = "TestSH02StartupPreservesUnknownRunningAndTerminalRows"
M01_HEALTHY_CONTROL = "TestSH04StartPreservesSyncLeaseRecoveryControls"
M01_ASSERTION = "SH02 startup attempted JobRun update target="


def sha(data):
    return hashlib.sha256(data).hexdigest()


def run(args, check=True, timeout=120):
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    if check and result.returncode:
        raise RuntimeError(
            f"command failed exit={result.returncode}: {args[:5]!r}; "
            f"stderr={result.stderr.decode('utf-8', 'replace')[-1000:]}"
        )
    return result


def manifest(directory):
    return sorted(
        [
            {
                "path": path.relative_to(directory).as_posix(),
                "bytes": path.stat().st_size,
                "sha256": sha(path.read_bytes()),
            }
            for path in directory.rglob("*")
            if path.is_file()
        ],
        key=lambda item: item["path"],
    )


def scrub_inspect(value):
    """Remove synthetic passwords/DSNs from Docker inspect output before publication."""
    cleaned = json.loads(json.dumps(value))
    config = cleaned.get("Config", {})
    env = config.get("Env")
    if isinstance(env, list):
        safe_env = []
        for entry in env:
            key, sep, val = entry.partition("=")
            if key in {"MYSQL_ROOT_PASSWORD", "MYSQL_PASSWORD", "MYSQL_PWD"}:
                val = "<SYNTHETIC_FIXTURE>"
            elif key == "TEST_DB_DSN":
                val = re.sub(r"(?<=:)[^:@/]+(?=@tcp\()", "<SYNTHETIC_FIXTURE>", val)
            safe_env.append(key + (sep + val if sep else ""))
        config["Env"] = safe_env
    return cleaned


def inspect(kind, name):
    result = run(["docker", kind, "inspect", name])
    decoded = json.loads(result.stdout)
    if not isinstance(decoded, list) or len(decoded) != 1:
        raise RuntimeError(f"unexpected exact-resource inspect response for {kind} {name}")
    return decoded[0]


def backend_git_binding(base_commit, source_commit, head_commit="HEAD"):
    changed = run(["git", "diff", "--name-only", base_commit, source_commit, "--", "backend"]).stdout.decode().splitlines()
    if set(changed) != EXPECTED_FILES or len(changed) != len(EXPECTED_FILES):
        raise RuntimeError("cumulative backend diff from base must contain exactly the two authorized paths")
    source_tree = run(["git", "rev-parse", source_commit + ":backend"]).stdout.decode().strip()
    head_tree = run(["git", "rev-parse", head_commit + ":backend"]).stdout.decode().strip()
    if source_tree != head_tree:
        raise RuntimeError("bound source backend tree differs from current HEAD backend tree")
    return {
        "baseCommit": base_commit,
        "sourceCommit": source_commit,
        "headCommit": head_commit,
        "cumulativeBackendChangedPaths": changed,
        "sourceBackendTree": source_tree,
        "headBackendTree": head_tree,
        "sourceBackendTreeEqualsHead": True,
    }


def campaign_exit_code(errors, summary_published):
    return 1 if errors or not summary_published else 0


def static_receipt_contract_check():
    """Exercise summary typing/hash rejection and the nonzero publication-failure exit."""
    if campaign_exit_code([], True) != 0:
        raise RuntimeError("validated summary must permit success")
    if campaign_exit_code([], False) == 0:
        raise RuntimeError("missing summary must return nonzero")
    if campaign_exit_code([{"type": "summary publication/validation"}], False) == 0:
        raise RuntimeError("summary publication failure must return nonzero")

    raw_path = Path("docs/reviews/probes/r078_independent_mutation_static.json")
    raw_bytes = raw_path.read_bytes()
    raw_ref = {"path": raw_path.as_posix(), "bytes": len(raw_bytes), "sha256": sha(raw_bytes)}
    summary = {
        "sourceCommit": FIRST_SOURCE_PACKET_COMMIT,
        "commands": [],
        "cleanup": {},
        "errors": [],
        "goInvocations": {"build": 0, "tests": 0},
        "result": "STATIC_ONLY",
        "probe": raw_ref,
    }
    with tempfile.TemporaryDirectory(prefix="ccmai-r078-receipt-contract-") as temp_dir:
        summary_path = Path(temp_dir) / "summary.json"
        encoded = (json.dumps(summary, indent=2) + "\n").encode()
        summary_path.write_bytes(encoded)
        expected = {"path": str(summary_path), "bytes": len(encoded), "sha256": sha(encoded)}
        accepted = validate_published_summary(summary_path, expected)
        if accepted["probe"] != raw_ref:
            raise RuntimeError("typed summary/raw reference was not preserved")

        bad_type = dict(summary)
        bad_type["commands"] = "invalid"
        bad_bytes = (json.dumps(bad_type, indent=2) + "\n").encode()
        summary_path.write_bytes(bad_bytes)
        try:
            validate_published_summary(
                summary_path,
                {"path": str(summary_path), "bytes": len(bad_bytes), "sha256": sha(bad_bytes)},
            )
        except RuntimeError as exc:
            if "invalid type" not in str(exc):
                raise
        else:
            raise RuntimeError("summary validator accepted a field with the wrong type")

        bad_ref_summary = dict(summary)
        bad_ref_summary["probe"] = dict(raw_ref, sha256="0" * 64)
        bad_ref_bytes = (json.dumps(bad_ref_summary, indent=2) + "\n").encode()
        summary_path.write_bytes(bad_ref_bytes)
        try:
            validate_published_summary(
                summary_path,
                {"path": str(summary_path), "bytes": len(bad_ref_bytes), "sha256": sha(bad_ref_bytes)},
            )
        except RuntimeError as exc:
            if "raw reference length/hash mismatch" not in str(exc):
                raise
        else:
            raise RuntimeError("summary validator accepted a raw reference with the wrong SHA256")
    return {
        "validatedSummaryExitCode": 0,
        "missingSummaryExitCode": 1,
        "publicationFailureExitCode": 1,
        "typedSummaryReadback": True,
        "rawReferenceSha256Readback": True,
        "wrongFieldTypeRejected": True,
        "wrongRawReferenceSha256Rejected": True,
        "goInvocations": 0,
        "dockerInvocations": 0,
    }


def exact_resource_exists(kind, name):
    result = run(["docker", kind, "inspect", name], check=False, timeout=30)
    if result.returncode == 0:
        return True
    text = (result.stdout + result.stderr).decode("utf-8", "replace").lower()
    if "no such" in text or "not found" in text:
        return False
    raise RuntimeError(f"cannot verify exact {kind} {name}: exit={result.returncode}")


def source_test_names(engine_dir):
    found = []
    for path in sorted(engine_dir.glob("*_test.go")):
        source = path.read_text(encoding="utf-8")
        found.extend(re.findall(r"^func (Test\w+)\(t \*testing\.T\)", source, re.M))
    if len(found) != len(set(found)):
        raise RuntimeError("duplicate top-level test function names in archived engine source")
    return sorted(found)


def event_summary(stdout, expected_names):
    events = [json.loads(line) for line in stdout.decode("utf-8").splitlines() if line.strip()]
    if any(not isinstance(event, dict) or "Action" not in event for event in events):
        raise RuntimeError("go test emitted an invalid JSON event")
    completed = [
        {"name": event["Test"], "action": event["Action"]}
        for event in events
        if event.get("Test") and event["Action"] in ("pass", "fail", "skip")
    ]
    top = [event for event in completed if "/" not in event["name"]]
    names = sorted(event["name"] for event in top)
    if names != sorted(expected_names):
        raise RuntimeError(f"top-level test inventory mismatch: observed={names!r}")
    output = "".join(event.get("Output", "") for event in events)
    return {
        "completed": completed,
        "topLevel": len(top),
        "pass": sum(event["action"] == "pass" for event in completed),
        "fail": sum(event["action"] == "fail" for event in completed),
        "skip": sum(event["action"] == "skip" for event in completed),
        "buildOrTimeout": bool(
            re.search(r"\[build failed\]|build-fail|undefined:|test timed out|panic:", output)
        ),
        "namedAssertion": M01_ASSERTION in output,
    }


def safe_extract_backend(archive, root):
    with tarfile.open(fileobj=io.BytesIO(archive), mode="r:") as tar:
        members = tar.getmembers()
        for member in members:
            name = PurePosixPath(member.name)
            if (
                name.is_absolute()
                or ".." in name.parts
                or member.issym()
                or member.islnk()
                or not (member.isfile() or member.isdir())
            ):
                raise RuntimeError("unsafe backend archive member: " + member.name)
        expected = sorted(
            [
                {
                    "path": member.name,
                    "bytes": member.size,
                    "sha256": sha(tar.extractfile(member).read()),
                }
                for member in members
                if member.isfile()
            ],
            key=lambda item: item["path"],
        )
        tar.extractall(root, members=members, filter="data")
    return expected


def restore_legacy_mutation(source_bytes, baseline_bytes):
    """Return the exact old helper/call mutation and its one-file diff metadata."""
    source = source_bytes.decode("utf-8")
    baseline = baseline_bytes.decode("utf-8")
    helper_start = baseline.find(LEGACY_HELPER_START)
    helper_end = baseline.find(LEGACY_HELPER_END, helper_start)
    if helper_start < 0 or helper_end < 0:
        raise RuntimeError("legacy helper boundaries are missing from the fixed baseline")
    helper = baseline[helper_start:helper_end] + "\n"
    baseline_call_matches = LEGACY_CALL_LINE.findall(baseline)
    baseline_definition_matches = LEGACY_DEFINITION_LINE.findall(baseline)
    if len(baseline_call_matches) != 1 or len(baseline_definition_matches) != 1:
        raise RuntimeError("baseline must contain exactly one standalone call and definition")
    if baseline.count(LEGACY_CALL) != 1 or helper.count("func cleanupStuckRuns()") != 1:
        raise RuntimeError("baseline exact call/helper block match count must both be one")
    if LEGACY_CALL_LINE.findall(source) or LEGACY_DEFINITION_LINE.findall(source):
        raise RuntimeError("hardened source unexpectedly contains a standalone legacy call/helper")
    start_marker = "\ts.scheduler.Start()"
    stop_marker = "// Stop gracefully shuts down the scheduler."
    if source.count(start_marker) != 1 or source.count(stop_marker) != 1:
        raise RuntimeError("legacy mutation insertion markers must each be unique")
    mutant = source.replace(start_marker, LEGACY_CALL + start_marker, 1)
    mutant = mutant.replace(stop_marker, helper + stop_marker, 1)
    call_matches = LEGACY_CALL_LINE.findall(mutant)
    definition_matches = LEGACY_DEFINITION_LINE.findall(mutant)
    if len(call_matches) != 1 or len(definition_matches) != 1:
        raise RuntimeError("mutant must contain exactly one standalone call and definition")
    mutant_bytes = mutant.encode("utf-8")
    diff = "".join(difflib.unified_diff(
        source.splitlines(True), mutant.splitlines(True),
        fromfile="a/backend/engine/scheduler.go", tofile="b/backend/engine/scheduler.go",
    )).encode()
    metadata = {
        "legacyBaselineCommit": LEGACY_BASELINE,
        "baselineSchedulerSha256": sha(baseline_bytes),
        "helperSourceSha256": sha(helper.encode()),
        "baselineCallExactMatchCount": len(baseline_call_matches),
        "baselineDefinitionExactMatchCount": len(baseline_definition_matches),
        "mutantCallExactMatchCount": len(call_matches),
        "mutantDefinitionExactMatchCount": len(definition_matches),
        "appliedInDisposableArchiveOnly": True,
        "originalSha256": sha(source_bytes),
        "mutantSha256": sha(mutant_bytes),
        "diffSha256": sha(diff),
        "diffBase64": base64.b64encode(diff).decode(),
    }
    return mutant_bytes, metadata


def static_mutation_check():
    """Pure Python/git preflight; it does not invoke Go, Docker, or edit source."""
    source_path = Path("backend/engine/scheduler.go")
    source_bytes = source_path.read_bytes()
    baseline_bytes = run(["git", "show", LEGACY_BASELINE + ":backend/engine/scheduler.go"]).stdout
    mutant, metadata = restore_legacy_mutation(source_bytes, baseline_bytes)
    diff = base64.b64decode(metadata["diffBase64"])
    receipt = {
        "result": "PASS",
        "scope": "pure_disposable_M01_mutation_construction_only",
        "goInvocations": 0,
        "dockerInvocations": 0,
        "targetPath": "backend/engine/scheduler.go",
        "candidateSourceSha256": sha(source_bytes),
        "mutantSha256": sha(mutant),
        "diffBytes": len(diff),
        **metadata,
    }
    target = Path("docs/reviews/probes/r078_independent_mutation_static.json")
    if target.exists():
        raise RuntimeError("refusing to overwrite static mutation receipt: " + target.as_posix())
    with target.open("xb") as stream:
        stream.write((json.dumps(receipt, indent=2) + "\n").encode())
    check = json.loads(target.read_text(encoding="utf-8"))
    if not isinstance(check, dict) or check.get("result") != "PASS" or check.get("goInvocations") != 0 or check.get("dockerInvocations") != 0:
        raise RuntimeError("static mutation receipt failed typed readback")
    if sha(target.read_bytes()) != sha((json.dumps(receipt, indent=2) + "\n").encode()):
        raise RuntimeError("static mutation receipt byte/hash readback mismatch")
    return receipt


def static_backend_binding_check():
    """Check immutable first-source tree identity while repair1 is still uncommitted."""
    base = "0579a792c473e53eb11c6efa007ab9d539370dec"
    binding = backend_git_binding(base, FIRST_SOURCE_PACKET_COMMIT)
    dirty = run(["git", "status", "--short", "--untracked-files=all", "--", "backend"]).stdout.decode().splitlines()
    dirty_paths = [line[3:] for line in dirty if len(line) >= 4]
    allowed_repair = {"backend/engine/scheduler_startup_hardening_test.go"}
    if set(dirty_paths) - allowed_repair:
        raise RuntimeError("unexpected backend worktree paths during repair1 static check")
    receipt = {
        "result": "PASS",
        "scope": "immutable_first_source_packet_binding_during_repair1",
        "goInvocations": 0,
        "dockerInvocations": 0,
        **binding,
        "backendWorktreeClean": not dirty,
        "expectedPendingRepairPaths": dirty_paths,
        "finalRuntimeCleanWorktreeRequired": True,
        "receiptContract": static_receipt_contract_check(),
    }
    target = Path("docs/reviews/probes/r078_independent_backend_binding_static.json")
    if target.exists():
        raise RuntimeError("refusing to overwrite backend binding receipt: " + target.as_posix())
    encoded = (json.dumps(receipt, indent=2) + "\n").encode()
    with target.open("xb") as stream:
        stream.write(encoded)
    readback = target.read_bytes()
    check = json.loads(readback)
    if (check.get("sourceBackendTree") != check.get("headBackendTree")
            or check.get("result") != "PASS"
            or check.get("goInvocations") != 0
            or check.get("dockerInvocations") != 0
            or sha(readback) != sha(encoded)):
        raise RuntimeError("backend binding receipt failed typed readback")
    return receipt


def collect_raw_references(value):
    refs = []
    if isinstance(value, dict):
        if {"path", "bytes", "sha256"}.issubset(value):
            refs.append(value)
        for nested in value.values():
            refs.extend(collect_raw_references(nested))
    elif isinstance(value, list):
        for nested in value:
            refs.extend(collect_raw_references(nested))
    return refs


def validate_published_summary(path, expected_ref):
    data = path.read_bytes()
    if len(data) != expected_ref["bytes"] or sha(data) != expected_ref["sha256"]:
        raise RuntimeError("published summary length/hash mismatch")
    parsed = json.loads(data)
    if not isinstance(parsed, dict):
        raise RuntimeError("published summary root is not a JSON object")
    required = {"sourceCommit": str, "commands": list, "cleanup": dict, "errors": list,
                "goInvocations": dict, "result": str}
    for key, expected_type in required.items():
        if not isinstance(parsed.get(key), expected_type):
            raise RuntimeError("published summary field has invalid type: " + key)
    for ref in collect_raw_references(parsed):
        if not isinstance(ref.get("path"), str) or type(ref.get("bytes")) is not int:
            raise RuntimeError("published raw reference has invalid path/bytes type")
        if not re.fullmatch(r"[0-9a-f]{64}", str(ref.get("sha256", ""))):
            raise RuntimeError("published raw reference has invalid SHA256")
        raw_path = PurePosixPath(ref["path"])
        if raw_path.is_absolute() or ".." in raw_path.parts or not ref["path"].startswith("docs/reviews/probes/r078_independent_"):
            raise RuntimeError("published raw reference escapes the R078 worker artifact prefix")
        raw = Path(*raw_path.parts).read_bytes()
        if len(raw) != ref["bytes"] or sha(raw) != ref["sha256"]:
            raise RuntimeError("published raw reference length/hash mismatch: " + ref["path"])
    return parsed


def remove_owned_source_snapshot(source_root, temp_parent, token):
    """Remove only the exact runner-created child directly inside the resolved TEMP parent."""
    expected_prefix = "ccmai-r078-independent-source-" + token + "-"
    if not source_root.exists():
        return {"removed": False, "alreadyAbsent": True, "validatedParent": str(temp_parent)}
    if source_root.is_symlink():
        raise RuntimeError("owned source snapshot path became a symlink")
    resolved_parent = temp_parent.resolve(strict=True)
    resolved = source_root.resolve(strict=True)
    if resolved.parent != resolved_parent or resolved.name != source_root.name or not resolved.name.startswith(expected_prefix):
        raise RuntimeError("refuse recursive removal outside the exact owned TEMP child")
    shutil.rmtree(resolved)
    if resolved.exists():
        raise RuntimeError("owned source snapshot remains after recursive removal")
    return {"removed": True, "alreadyAbsent": False, "validatedParent": str(resolved_parent),
            "ownedPrefix": expected_prefix}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", help="immutable 40-hex repaired source commit")
    parser.add_argument("--plan", default="docs/reviews/probes/r078_independent_plan.json")
    parser.add_argument("--check-mutation", action="store_true",
                        help="validate the exact old-helper mutation with Python/git only; no Go or Docker")
    parser.add_argument("--check-binding", action="store_true",
                        help="validate immutable source binding and receipt contracts with Python/git only")
    args = parser.parse_args()
    if args.check_mutation:
        result = static_mutation_check()
        print(json.dumps({"result": result["result"], "goInvocations": 0, "dockerInvocations": 0,
                          "candidateSourceSha256": result["candidateSourceSha256"],
                          "mutantSha256": result["mutantSha256"], "diffSha256": result["diffSha256"]}), flush=True)
        return 0
    if args.check_binding:
        result = static_backend_binding_check()
        print(json.dumps({"result": result["result"], "goInvocations": 0, "dockerInvocations": 0,
                          "sourceBackendTreeEqualsHead": result["sourceBackendTreeEqualsHead"],
                          "cumulativeBackendChangedPaths": result["cumulativeBackendChangedPaths"],
                          "receiptContract": result["receiptContract"]}), flush=True)
        return 0
    if not args.source:
        parser.error("--source is required for the runtime campaign")
    if not re.fullmatch(r"[0-9a-f]{40}", args.source):
        raise RuntimeError("invalid source commit; bind plan only after root commits the source packet")

    plan_path = Path(args.plan)
    plan = json.loads(plan_path.read_text(encoding="utf-8"))
    if plan.get("sourceCommit") != args.source:
        raise RuntimeError("runtime source differs from the committed worker plan")
    if plan.get("legacyBaselineCommit") != LEGACY_BASELINE:
        raise RuntimeError("legacy mutation baseline differs from the reviewed plan")
    if plan.get("firstSourcePacketCommit") != FIRST_SOURCE_PACKET_COMMIT:
        raise RuntimeError("plan does not preserve the immutable first source packet identity")
    if plan.get("baseCommit") != "0579a792c473e53eb11c6efa007ab9d539370dec":
        raise RuntimeError("plan base differs from immutable R078 seed")
    if plan.get("reviewerGoCallBudget") != {"build": 1, "tests": 3, "retries": 0}:
        raise RuntimeError("worker Go invocation budget mismatch")

    token = uuid.uuid4().hex[:10]
    raw_backup = Path(tempfile.mkdtemp(prefix="ccmai-r078-independent-raw-backup-"))
    temp_parent = Path(tempfile.gettempdir()).resolve(strict=True)
    source_prefix = "ccmai-r078-independent-source-" + token + "-"
    source_root = Path(tempfile.mkdtemp(prefix=source_prefix, dir=temp_parent)).resolve(strict=True)
    net = "ccmai-r078-independent-net-" + token
    database = "ccmai-r078-independent-db-" + token
    build_cache = "ccmai-r078-independent-build-cache-" + token
    db_volume = "ccmai-r078-independent-dbdata-" + token
    prefix = Path("docs/reviews/probes/r078_independent_")
    cache = Path(os.environ["USERPROFILE"]) / "go/pkg/mod"
    receipt = {
        "sourceCommit": args.source,
        "reviewer": "Codex /root",
        "scope": "scheduler_startup_hardening_synthetic_local_only",
        "campaigns": 0,
        "goInvocations": {"build": 0, "tests": 0},
        "automaticRetries": 0,
        "commands": [],
        "errors": [],
        "sourceDirectory": str(source_root),
        "cleanup": {"containers": [], "volumes": [], "network": None},
        "infrastructureCommands": [],
        "evidenceLimit": "No CVF governance, provider, deployment, or production claim.",
    }
    containers = []
    volumes = []
    network_created = False
    mutation = None
    original_scheduler = None
    expected_manifest = None
    archive = None
    summary_published = False
    began = time.monotonic()

    def save(label, data):
        target = Path(str(prefix) + label)
        if target.exists():
            raise RuntimeError("refusing to overwrite R078 raw evidence: " + target.as_posix())
        with target.open("xb") as stream:
            stream.write(data)
        backup = raw_backup / target.name
        backup.write_bytes(data)
        if target.read_bytes() != data or backup.read_bytes() != data:
            raise RuntimeError("raw publication byte check failed")
        return {"path": target.as_posix(), "bytes": len(data), "sha256": sha(data)}

    def verify_source(label):
        observed = manifest(source_root)
        equal = observed == expected_manifest
        receipt.setdefault("sourceChecks", []).append({"stage": label, "equal": equal})
        if not equal:
            raise RuntimeError("archive source differs from expected manifest at " + label)

    def call_go(label, go_command, network_name, expected_names=None, mutation_case=False):
        is_build = label == "build"
        limit = 1 if is_build else 3
        count = receipt["goInvocations"]["build" if is_build else "tests"]
        if count >= limit:
            raise RuntimeError("worker Go invocation ceiling reached")
        verify_source("before " + label)
        name = "ccmai-r078-independent-" + label + "-" + token
        command_args = [
            "docker", "create", "--pull=never", "--name", name,
            "--network", network_name,
            "--mount", f"type=bind,source={source_root / 'backend'},target=/src,readonly",
            "--mount", f"type=bind,source={cache},target=/go/pkg/mod,readonly",
            "--mount", f"type=volume,source={build_cache},target=/root/.cache/go-build",
            "-w", "/src",
            "-e", "GOFLAGS=-mod=readonly",
            "-e", "GOPROXY=off",
            "-e", "GOSUMDB=off",
            "-e", "GOTOOLCHAIN=local",
            "-e", "CGO_ENABLED=0",
        ]
        if not is_build:
            command_args += ["-e", "TEST_DB_DSN=" + TEST_DSN.format(database=database)]
        command_args += ["golang:1.26-alpine"] + go_command
        created = run(command_args)
        containers.append(name)
        before = inspect("container", name)
        if before["HostConfig"].get("NetworkMode") != network_name:
            raise RuntimeError(label + " container network differs from plan")
        if before["HostConfig"].get("PortBindings"):
            raise RuntimeError(label + " container unexpectedly publishes host ports")
        bind_mounts = [mount for mount in before.get("Mounts", []) if mount["Type"] == "bind"]
        if any(mount.get("RW") for mount in bind_mounts):
            raise RuntimeError(label + " container has a writable source/cache bind")
        observed_sources = {mount.get("Destination") for mount in bind_mounts}
        if observed_sources != {"/src", "/go/pkg/mod"}:
            raise RuntimeError(label + " container bind mount set differs from plan")
        before_ref = save(label + "_container_before.json", (json.dumps(scrub_inspect(before), indent=2) + "\n").encode())
        record = {
            "label": label,
            "container": name,
            "command": [value.replace(SYNTHETIC_DB_PASSWORD, "<SYNTHETIC_FIXTURE>") for value in command_args],
            "imageId": before["Image"],
            "network": network_name,
            "ports": before["HostConfig"].get("PortBindings"),
            "mounts": before.get("Mounts", []),
            "rawBeforeInspect": before_ref,
        }
        if expected_names is not None:
            record["expectedTopLevelTests"] = expected_names
        receipt["commands"].append(record)
        receipt["goInvocations"]["build" if is_build else "tests"] += 1

        start = time.monotonic()
        try:
            result = run(["docker", "start", "--attach", name], check=False, timeout=360 if label in ("positive", "restored") else 300)
        except subprocess.TimeoutExpired as exc:
            record["timeoutSeconds"] = 360 if label in ("positive", "restored") else 300
            record["partialStdout"] = save(label + "_timeout_stdout.log", exc.stdout or b"")
            record["partialStderr"] = save(label + "_timeout_stderr.log", exc.stderr or b"")
            stopped = run(["docker", "stop", "--time", "10", name], check=False, timeout=30)
            record["timeoutStopExit"] = stopped.returncode
            logs = run(["docker", "logs", name], check=False)
            record["timeoutLogsStdout"] = save(label + "_timeout_container_stdout.log", logs.stdout)
            record["timeoutLogsStderr"] = save(label + "_timeout_container_stderr.log", logs.stderr)
            record["timeoutContainerState"] = inspect("container", name)["State"]
            raise
        record.update({
            "seconds": round(time.monotonic() - start, 3),
            "attachExit": result.returncode,
            "stdout": save(label + (".log" if is_build else ".jsonl"), result.stdout),
            "stderr": save(label + "_stderr.log", result.stderr),
        })
        after = inspect("container", name)
        record["rawAfterInspect"] = save(label + "_container_after.json", (json.dumps(scrub_inspect(after), indent=2) + "\n").encode())
        state = after["State"]
        record["state"] = state
        if state["Running"] or state["OOMKilled"] or state["Error"] or result.returncode != state["ExitCode"]:
            raise RuntimeError(label + " container/attach exit state mismatch")

        if is_build:
            if result.returncode:
                raise RuntimeError("backend compile failed; no tests run")
            record["build"] = {"command": go_command, "passed": True}
            print(json.dumps({"command": label, "exit": result.returncode, "seconds": record["seconds"]}), flush=True)
            return

        stats = event_summary(result.stdout, expected_names)
        record["events"] = stats
        if mutation_case:
            if result.returncode == 0 or stats["skip"] or stats["buildOrTimeout"] or not stats["namedAssertion"]:
                raise RuntimeError("M01 did not fail at the named attempted-write assertion")
            if not any(item["name"] == M01_DETECTOR and item["action"] == "fail" for item in stats["completed"]):
                raise RuntimeError("M01 named SH02 detector did not fail")
            if not any(item["name"] == M01_HEALTHY_CONTROL and item["action"] == "pass" for item in stats["completed"]):
                raise RuntimeError("M01 SH04 healthy lease control did not pass")
            record["mutationDisposition"] = "KILLED_BY_NAMED_SH02_ATTEMPTED_WRITE_ASSERTION"
            record["healthyControl"] = {"name": M01_HEALTHY_CONTROL, "passed": True}
        elif result.returncode or stats["fail"] or stats["skip"] or stats["buildOrTimeout"]:
            raise RuntimeError(label + " focused positive test command failed or skipped")
        print(json.dumps({"command": label, "exit": result.returncode, "topLevel": stats["topLevel"],
                          "pass": stats["pass"], "fail": stats["fail"], "skip": stats["skip"],
                          "seconds": record["seconds"]}), flush=True)

    try:
        if plan.get("positiveTopLevelTests") != TEST_NAMES:
            raise RuntimeError("plan positive test inventory differs from runner contract")
        if plan.get("positiveRegex") != POSITIVE_PATTERN or plan.get("M01Regex") != M01_PATTERN:
            raise RuntimeError("plan exact Go test selectors differ from runner contract")
        if plan.get("M01NamedAssertion") != M01_ASSERTION:
            raise RuntimeError("plan M01 assertion differs from runner contract")
        if plan.get("expectedChangedPaths") != sorted(EXPECTED_FILES):
            raise RuntimeError("plan changed-path scope differs from runner contract")
        cache = cache.resolve()
        if not cache.is_dir():
            raise RuntimeError("offline Go module cache is absent")
        if run(["git", "merge-base", "--is-ancestor", args.source, "HEAD"], check=False).returncode:
            raise RuntimeError("source commit is not an ancestor of current HEAD")
        if run(["git", "merge-base", "--is-ancestor", FIRST_SOURCE_PACKET_COMMIT, args.source], check=False).returncode:
            raise RuntimeError("repaired source does not descend from the preserved first source packet")
        changed = run(["git", "diff", "--name-only", plan["baseCommit"], args.source, "--", "backend"]).stdout.decode().splitlines()
        if set(changed) != EXPECTED_FILES or len(changed) != len(EXPECTED_FILES):
            raise RuntimeError("cumulative backend diff from base must contain exactly the two authorized paths")
        source_tree = run(["git", "rev-parse", args.source + ":backend"]).stdout.decode().strip()
        head_tree = run(["git", "rev-parse", "HEAD:backend"]).stdout.decode().strip()
        if source_tree != head_tree:
            raise RuntimeError("bound source backend tree differs from current HEAD backend tree")
        dirty = run(["git", "status", "--short", "--untracked-files=all", "--", "backend"]).stdout.decode().strip()
        if dirty:
            raise RuntimeError("backend worktree must be clean and match the bound source backend tree")
        receipt["sourceBinding"] = {
            "firstSourcePacketCommit": FIRST_SOURCE_PACKET_COMMIT,
            "boundSourceCommit": args.source,
            "baseCommit": plan["baseCommit"],
            "cumulativeBackendChangedPaths": changed,
            "sourceBackendTree": source_tree,
            "headBackendTree": head_tree,
            "backendWorktreeClean": True,
        }

        root_image = inspect("image", "golang:1.26-alpine")
        mysql_image = inspect("image", "mysql:8.0")
        receipt["images"] = [
            {"tag": "golang:1.26-alpine", "id": root_image["Id"]},
            {"tag": "mysql:8.0", "id": mysql_image["Id"]},
        ]
        archive_cmd = ["git", "archive", "--format=tar", args.source, "backend"]
        archive = run(archive_cmd).stdout
        receipt["archive"] = {"command": archive_cmd, "bytes": len(archive), "sha256": sha(archive)}
        expected_manifest = safe_extract_backend(archive, source_root)
        receipt["manifest"] = save("manifest.json", (json.dumps(expected_manifest, indent=2) + "\n").encode())
        verify_source("initial extraction")
        derived = source_test_names(source_root / "backend/engine")
        selected = sorted(name for name in derived if name in set(TEST_NAMES))
        if selected != sorted(TEST_NAMES):
            raise RuntimeError("archived source does not contain the complete expected test inventory")
        receipt["plannedInventory"] = [
            {"label": "positive", "pattern": POSITIVE_PATTERN, "topLevel": TEST_NAMES},
            {"label": "M01", "pattern": M01_PATTERN,
             "topLevel": [M01_DETECTOR, M01_HEALTHY_CONTROL]},
            {"label": "restored", "pattern": POSITIVE_PATTERN, "topLevel": TEST_NAMES},
        ]
        actual_binding = {
            "sourceCommit": args.source,
            "archive": receipt["archive"],
            "manifestSha256": sha(json.dumps(expected_manifest, sort_keys=True).encode()),
            "inventory": receipt["plannedInventory"],
        }
        if plan.get("archiveSha256") != receipt["archive"]["sha256"]:
            raise RuntimeError("committed plan archive hash differs from exact source packet")
        if plan.get("physicalSourceHashes") != {
            path: sha((source_root / path).read_bytes()) for path in sorted(EXPECTED_FILES)
        }:
            raise RuntimeError("committed plan source hashes differ from archived bytes")
        receipt["preRuntimePlan"] = {
            "path": plan_path.as_posix(),
            "sha256": sha(plan_path.read_bytes()),
            "committedBeforeRuntime": True,
            "archiveBinding": actual_binding,
        }
        receipt["runtimePlan"] = save("runtime_plan.json", (json.dumps(actual_binding, indent=2) + "\n").encode())
        baseline_bytes = run(["git", "show", LEGACY_BASELINE + ":backend/engine/scheduler.go"]).stdout
        scheduler_path = source_root / "backend/engine/scheduler.go"
        original_scheduler = scheduler_path.read_bytes()
        mutant_bytes, mutation_metadata = restore_legacy_mutation(original_scheduler, baseline_bytes)
        diff = base64.b64decode(mutation_metadata["diffBase64"])
        mutation = {"original": original_scheduler, "mutant": mutant_bytes}
        receipt["M01Mutation"] = {
            **mutation_metadata,
            "targetPath": "backend/engine/scheduler.go",
            "appliedInDisposableArchiveOnly": True,
            "diffFile": save("M01_diff.patch", diff),
        }

        run(["docker", "volume", "create", build_cache])
        volumes.append(build_cache)
        receipt["infrastructureCommands"].append({
            "label": "create-disposable-go-build-cache-volume",
            "command": ["docker", "volume", "create", build_cache],
            "ownedResource": build_cache,
        })
        run(["docker", "volume", "create", db_volume])
        volumes.append(db_volume)
        receipt["infrastructureCommands"].append({
            "label": "create-disposable-mysql-data-volume",
            "command": ["docker", "volume", "create", db_volume],
            "ownedResource": db_volume,
        })
        run(["docker", "network", "create", "--internal", net])
        network_created = True
        receipt["infrastructureCommands"].append({
            "label": "create-internal-network",
            "command": ["docker", "network", "create", "--internal", net],
            "ownedResource": net,
        })
        network = inspect("network", net)
        if not network.get("Internal"):
            raise RuntimeError("synthetic database network is not Docker-internal")
        receipt["network"] = {"name": net, "internal": True, "driver": network.get("Driver")}
        containers.append(database)
        database_command = [
            "docker", "run", "--pull=never", "-d", "--name", database,
            "--network", net,
            "--mount", f"type=volume,source={db_volume},target=/var/lib/mysql",
            "-e", "MYSQL_ROOT_PASSWORD=" + SYNTHETIC_DB_PASSWORD,
            "-e", "MYSQL_DATABASE=CCMA",
            "-e", "MYSQL_USER=ccma",
            "-e", "MYSQL_PASSWORD=" + SYNTHETIC_DB_PASSWORD,
            "mysql:8.0", "--log-bin-trust-function-creators=1", "--max-connections=1000",
        ]
        run(database_command)
        receipt["infrastructureCommands"].append({
            "label": "start-synthetic-mysql",
            "command": [value.replace(SYNTHETIC_DB_PASSWORD, "<SYNTHETIC_FIXTURE>") for value in database_command],
            "ownedResource": database,
        })
        db_inspect = inspect("container", database)
        if db_inspect["HostConfig"].get("NetworkMode") != net or db_inspect["HostConfig"].get("PortBindings"):
            raise RuntimeError("synthetic MySQL container network/port isolation failed")
        db_mounts = db_inspect.get("Mounts", [])
        if not any(mount.get("Name") == db_volume and mount.get("Destination") == "/var/lib/mysql" for mount in db_mounts):
            raise RuntimeError("synthetic MySQL container is not using its named disposable volume")
        receipt["database"] = {
            "container": database,
            "imageId": db_inspect["Image"],
            "network": net,
            "ports": db_inspect["HostConfig"].get("PortBindings"),
            "mounts": db_mounts,
            "environmentScrubbed": True,
        }
        ready_start = time.monotonic()
        while time.monotonic() - ready_start < 90:
            ping = run([
                "docker", "exec", "-e", "MYSQL_PWD=" + SYNTHETIC_DB_PASSWORD,
                database, "mysqladmin", "ping", "-uccma", "--silent",
            ], check=False, timeout=10)
            if ping.returncode == 0:
                break
            time.sleep(2)
        else:
            raise RuntimeError("synthetic MySQL readiness timeout")
        receipt["databaseReadySeconds"] = round(time.monotonic() - ready_start, 3)
        receipt["databaseReadyProbe"] = {
            "command": ["docker", "exec", "-e", "MYSQL_PWD=<SYNTHETIC_FIXTURE>", database,
                        "mysqladmin", "ping", "-uccma", "--silent"],
            "boundedSeconds": 90,
            "result": "ready",
        }

        receipt["campaigns"] = 1
        call_go("build", ["go", "build", "-o", "/tmp/r078-backend", "."], "none")
        test_command = ["go", "test", "-json", "-count=1", "-timeout=240s", "-run"]
        call_go("positive", test_command + [POSITIVE_PATTERN, "./engine"], net, TEST_NAMES)

        scheduler_path.write_bytes(mutation["mutant"])
        expected_manifest = [
            dict(item, bytes=len(mutation["mutant"]), sha256=sha(mutation["mutant"]))
            if item["path"] == "backend/engine/scheduler.go" else item
            for item in expected_manifest
        ]
        receipt["M01Mutation"]["appliedManifest"] = save("M01_manifest.json", (json.dumps(expected_manifest, indent=2) + "\n").encode())
        verify_source("M01 mutation applied")
        call_go("M01", test_command + [M01_PATTERN, "./engine"], net,
                [M01_DETECTOR, M01_HEALTHY_CONTROL], mutation_case=True)
        scheduler_path.write_bytes(mutation["original"])
        expected_manifest = json.loads((Path(str(prefix) + "manifest.json")).read_text(encoding="utf-8"))
        receipt["M01Mutation"]["restoredExactBytes"] = scheduler_path.read_bytes() == mutation["original"]
        verify_source("M01 exact restoration")
        if not receipt["M01Mutation"]["restoredExactBytes"]:
            raise RuntimeError("M01 source file did not restore to exact archived bytes")
        call_go("restored", test_command + [POSITIVE_PATTERN, "./engine"], net, TEST_NAMES)
        if receipt["goInvocations"] != {"build": 1, "tests": 3}:
            raise RuntimeError("campaign did not use exactly one build and three test invocations")
        receipt["result"] = "PASS"
    except Exception as exc:
        receipt["errors"].append({"type": type(exc).__name__, "message": str(exc)})
        receipt["result"] = "CHANGES_REQUIRED_NO_RETRY"
    finally:
        if mutation is not None:
            try:
                target = source_root / "backend/engine/scheduler.go"
                target.write_bytes(mutation["original"])
                if expected_manifest is not None:
                    expected_manifest = json.loads((Path(str(prefix) + "manifest.json")).read_text(encoding="utf-8"))
                    verify_source("final guaranteed restoration")
                receipt.setdefault("M01Mutation", {})["finalRestoredSha256"] = sha(target.read_bytes())
            except Exception as exc:
                receipt["errors"].append({"type": "restoration", "message": str(exc)})
        completed = {record["label"] for record in receipt["commands"]}
        receipt["notRun"] = [label for label in ("build", "positive", "M01", "restored") if label not in completed]
        for name in reversed(containers):
            item = {"name": name, "removeExit": None}
            receipt["cleanup"]["containers"].append(item)
            try:
                removed = run(["docker", "rm", "-fv", name], check=False, timeout=30)
                item["removeExit"] = removed.returncode
                if removed.returncode and exact_resource_exists("container", name):
                    raise RuntimeError("exact named container remains after cleanup")
            except Exception as exc:
                receipt["errors"].append({"type": "container cleanup", "message": str(exc)})
        for volume in reversed(volumes):
            item = {"name": volume, "removeExit": None}
            receipt["cleanup"]["volumes"].append(item)
            try:
                removed = run(["docker", "volume", "rm", volume], check=False, timeout=30)
                item["removeExit"] = removed.returncode
                if removed.returncode and exact_resource_exists("volume", volume):
                    raise RuntimeError("exact named volume remains after cleanup")
            except Exception as exc:
                receipt["errors"].append({"type": "volume cleanup", "message": str(exc)})
        if network_created:
            receipt["cleanup"]["network"] = {"name": net, "removeExit": None}
            try:
                removed = run(["docker", "network", "rm", net], check=False, timeout=30)
                receipt["cleanup"]["network"]["removeExit"] = removed.returncode
                if removed.returncode and exact_resource_exists("network", net):
                    raise RuntimeError("exact named network remains after cleanup")
            except Exception as exc:
                receipt["errors"].append({"type": "network cleanup", "message": str(exc)})
        try:
            for item in receipt["cleanup"]["containers"]:
                item["absent"] = not exact_resource_exists("container", item["name"])
            for item in receipt["cleanup"]["volumes"]:
                item["absent"] = not exact_resource_exists("volume", item["name"])
            if receipt["cleanup"]["network"]:
                item = receipt["cleanup"]["network"]
                item["absent"] = not exact_resource_exists("network", item["name"])
            all_items = receipt["cleanup"]["containers"] + receipt["cleanup"]["volumes"]
            if receipt["cleanup"]["network"]:
                all_items.append(receipt["cleanup"]["network"])
            if any(not item["absent"] for item in all_items):
                raise RuntimeError("owned R078 cleanup resource remains")
            receipt["cleanup"]["ownedResourcesOnly"] = True
            receipt["cleanup"]["verifiedByExactName"] = True
        except Exception as exc:
            receipt["errors"].append({"type": "cleanup verification", "message": str(exc)})
        try:
            receipt["cleanup"]["sourceSnapshot"] = remove_owned_source_snapshot(source_root, temp_parent, token)
        except Exception as exc:
            receipt["errors"].append({"type": "source snapshot cleanup", "message": str(exc)})
        receipt["seconds"] = round(time.monotonic() - began, 3)
        if receipt["errors"]:
            receipt["result"] = "CHANGES_REQUIRED_NO_RETRY"
        receipt["rawBackupDirectory"] = str(raw_backup)
        try:
            summary_ref = save("summary.json", (json.dumps(receipt, indent=2) + "\n").encode())
            published_summary = validate_published_summary(Path(summary_ref["path"]), summary_ref)
            if published_summary["sourceCommit"] != args.source:
                raise RuntimeError("published summary source identity mismatch")
            summary_published = True
            print(json.dumps({"result": receipt["result"], "goInvocations": receipt["goInvocations"],
                              "summaryValidated": True, "seconds": receipt["seconds"],
                              "errors": receipt["errors"]}), flush=True)
        except Exception as exc:
            receipt["errors"].append({"type": "summary publication/validation", "message": str(exc)})
            receipt["result"] = "CHANGES_REQUIRED_NO_RETRY"
            failure = {
                "result": receipt["result"],
                "summaryPublishedAndValidated": summary_published,
                "summaryAttemptPath": str(prefix) + "summary.json",
                "errorType": type(exc).__name__,
                "error": str(exc),
                "sourceCommit": args.source,
                "rawBackupDirectory": str(raw_backup),
            }
            failure_path = Path(str(prefix) + "summary_failure.json")
            try:
                with failure_path.open("xb") as stream:
                    failure_bytes = (json.dumps(failure, indent=2) + "\n").encode()
                    stream.write(failure_bytes)
                backup_path = raw_backup / failure_path.name
                backup_path.write_bytes(failure_bytes)
                if backup_path.read_bytes() != failure_bytes or failure_path.read_bytes() != failure_bytes:
                    raise RuntimeError("summary failure record byte verification failed")
            except Exception as backup_exc:
                receipt["errors"].append({"type": "summary failure record", "message": str(backup_exc)})
            print(json.dumps({"result": "CHANGES_REQUIRED_NO_RETRY", "summaryWriteError": str(exc),
                              "summaryPublishedAndValidated": summary_published,
                              "rawBackupDirectory": str(raw_backup)}), flush=True)
    return campaign_exit_code(receipt["errors"], summary_published)


if __name__ == "__main__":
    raise SystemExit(main())
