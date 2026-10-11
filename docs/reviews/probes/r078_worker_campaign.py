"""R078 bounded worker campaign for the committed scheduler startup change.

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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True, help="immutable 40-hex source-packet commit")
    parser.add_argument("--plan", default="docs/reviews/probes/r078_worker_plan.json")
    args = parser.parse_args()
    if not re.fullmatch(r"[0-9a-f]{40}", args.source):
        raise RuntimeError("invalid source commit; bind plan only after root commits the source packet")

    plan_path = Path(args.plan)
    plan = json.loads(plan_path.read_text(encoding="utf-8"))
    if plan.get("sourceCommit") != args.source:
        raise RuntimeError("runtime source differs from the committed worker plan")
    if plan.get("legacyBaselineCommit") != LEGACY_BASELINE:
        raise RuntimeError("legacy mutation baseline differs from the reviewed plan")
    if plan.get("workerGoCallBudget") != {"build": 1, "tests": 3, "retries": 0}:
        raise RuntimeError("worker Go invocation budget mismatch")

    token = uuid.uuid4().hex[:10]
    raw_backup = Path(tempfile.mkdtemp(prefix="ccmai-r078-worker-raw-backup-"))
    source_root = Path(tempfile.mkdtemp(prefix="ccmai-r078-worker-source-"))
    net = "ccmai-r078-worker-net-" + token
    database = "ccmai-r078-worker-db-" + token
    build_cache = "ccmai-r078-worker-build-cache-" + token
    db_volume = "ccmai-r078-worker-dbdata-" + token
    prefix = Path("docs/reviews/probes/r078_worker_")
    cache = Path(os.environ["USERPROFILE"]) / "go/pkg/mod"
    receipt = {
        "sourceCommit": args.source,
        "worker": "Codex /root/r078_worker",
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
        name = "ccmai-r078-worker-" + label + "-" + token
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
        changed = run(["git", "diff-tree", "--no-commit-id", "--name-only", "-r", args.source]).stdout.decode().splitlines()
        if set(changed) != EXPECTED_FILES or len(changed) != len(EXPECTED_FILES):
            raise RuntimeError("source packet commit must contain exactly the two authorized backend files")
        dirty = run(["git", "status", "--short", "--", "backend"]).stdout.decode().strip()
        if dirty:
            raise RuntimeError("backend worktree must match the committed source packet before runtime")

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
        baseline = run(["git", "show", LEGACY_BASELINE + ":backend/engine/scheduler.go"]).stdout.decode("utf-8")
        helper_start = baseline.find(LEGACY_HELPER_START)
        helper_end = baseline.find(LEGACY_HELPER_END, helper_start)
        if helper_start < 0 or helper_end < 0:
            raise RuntimeError("legacy helper boundaries are missing from fixed baseline")
        helper = baseline[helper_start:helper_end] + "\n"
        scheduler_path = source_root / "backend/engine/scheduler.go"
        original_scheduler = scheduler_path.read_bytes()
        original_text = original_scheduler.decode("utf-8")
        if original_text.count("cleanupStuckRuns()") != 0 or original_text.count("func cleanupStuckRuns()") != 0:
            raise RuntimeError("hardened source unexpectedly contains the legacy startup sweep")
        if baseline.count(LEGACY_CALL) != 1 or helper.count("func cleanupStuckRuns()") != 1:
            raise RuntimeError("baseline legacy helper/call match counts must both be one")
        if original_text.count("\ts.scheduler.Start()") != 1:
            raise RuntimeError("current Scheduler.Start insertion point is not unique")
        mutant_text = original_text.replace("\ts.scheduler.Start()", LEGACY_CALL + "\ts.scheduler.Start()", 1)
        mutant_text = mutant_text.replace("// Stop gracefully shuts down the scheduler.", helper + "// Stop gracefully shuts down the scheduler.", 1)
        if mutant_text.count("cleanupStuckRuns()") != 1 or mutant_text.count("func cleanupStuckRuns()") != 1:
            raise RuntimeError("M01 isolated source did not restore exactly one call and helper")
        mutant_bytes = mutant_text.encode("utf-8")
        diff = "".join(difflib.unified_diff(
            original_text.splitlines(True), mutant_text.splitlines(True),
            fromfile="a/backend/engine/scheduler.go", tofile="b/backend/engine/scheduler.go",
        )).encode()
        mutation = {"original": original_scheduler, "mutant": mutant_bytes}
        receipt["M01Mutation"] = {
            "baselineCommit": LEGACY_BASELINE,
            "baselineSchedulerSha256": sha(baseline.encode()),
            "targetPath": "backend/engine/scheduler.go",
            "helperSourceSha256": sha(helper.encode()),
            "legacyCallExactMatchCount": 1,
            "helperExactMatchCount": 1,
            "appliedInDisposableArchiveOnly": True,
            "originalSha256": sha(original_scheduler),
            "mutantSha256": sha(mutant_bytes),
            "diffSha256": sha(diff),
            "diffBase64": base64.b64encode(diff).decode(),
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
            if source_root.exists():
                import shutil
                shutil.rmtree(source_root)
            receipt["cleanup"]["sourceSnapshotRemoved"] = not source_root.exists()
            if not receipt["cleanup"]["sourceSnapshotRemoved"]:
                raise RuntimeError("owned temporary source snapshot remains")
        except Exception as exc:
            receipt["errors"].append({"type": "source snapshot cleanup", "message": str(exc)})
        receipt["seconds"] = round(time.monotonic() - began, 3)
        if receipt["errors"]:
            receipt["result"] = "CHANGES_REQUIRED_NO_RETRY"
        receipt["rawBackupDirectory"] = str(raw_backup)
        try:
            summary_ref = save("summary.json", (json.dumps(receipt, indent=2) + "\n").encode())
            receipt["summaryReference"] = summary_ref
            print(json.dumps({"result": receipt["result"], "goInvocations": receipt["goInvocations"],
                              "seconds": receipt["seconds"], "errors": receipt["errors"]}), flush=True)
        except Exception as exc:
            print(json.dumps({"result": "CHANGES_REQUIRED_NO_RETRY", "summaryWriteError": str(exc),
                              "rawBackupDirectory": str(raw_backup)}), flush=True)
    return 1 if receipt["errors"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
