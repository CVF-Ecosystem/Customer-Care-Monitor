"""R060 worker exact committed-source campaign; four bounded Go calls.

No runtime governance claim. Synthetic local MySQL, cached/offline images and
modules, read-only binds, internal network, no host ports. Never retries.
"""
import argparse
import base64
import difflib
import hashlib
import io
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile
import time
import uuid


def sha(data):
    return hashlib.sha256(data).hexdigest()


def command(args, check=True, timeout=120):
    p = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    if check and p.returncode:
        raise RuntimeError(f"command failed: {args[:4]!r}, exit={p.returncode}")
    return p


def inspect(kind, name):
    return json.loads(command(["docker", kind, "inspect", name]).stdout)[0]


def manifest(directory):
    return sorted([{"path": p.relative_to(directory).as_posix(), "bytes": p.stat().st_size,
                    "sha256": sha(p.read_bytes())} for p in directory.rglob("*") if p.is_file()],
                  key=lambda item: item["path"])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True)
    args = parser.parse_args()
    prefix = Path("docs/reviews/probes/r060_worker_")
    token = uuid.uuid4().hex[:10]
    root = Path(tempfile.mkdtemp(prefix="ccmai-r060-worker-"))
    net, database = "ccmai-r060-worker-net-" + token, "ccmai-r060-worker-db-" + token
    # cvf-allow-secret-fixture: disposable fixture only, never a real credential.
    password = "r060_synthetic_fixture"
    cache = Path(os.environ["USERPROFILE"]) / "go/pkg/mod"
    receipt = {"sourceCommit": args.source, "worker": "Codex /root/r059_worker", "campaigns": 0,
               "goInvocations": 0, "automaticRetries": 0, "commands": [], "errors": [],
               "sourceDirectory": str(root), "scope": "application_observation_only",
               "cleanup": {"containers": [], "volumes": [], "network": None}}
    containers, volumes = [], set()
    network_created = False
    cache_created = False
    build_cache = "ccmai-r060-worker-cache-" + token
    original = target = expected_manifest = None
    mutations = []
    began = time.monotonic()

    def save(label, data):
        path = Path(str(prefix) + label)
        path.write_bytes(data)
        return {"path": path.as_posix(), "bytes": len(data), "sha256": sha(data)}

    def verify_source(label):
        observed = manifest(root)
        equal = observed == expected_manifest
        receipt.setdefault("sourceChecks", []).append({"stage": label, "equal": equal})
        if not equal:
            raise RuntimeError("archive member restoration mismatch: " + label)

    def expected_names(pattern):
        names = []
        for file in (root / "backend/engine").glob("*_test.go"):
            names += re.findall(r"^func (Test\w+)\(t \*testing\.T\)", file.read_text(encoding="utf-8"), re.M)
        selected = sorted(name for name in names if re.search(pattern, name))
        if not selected or len(selected) != len(set(selected)):
            raise RuntimeError("empty or duplicate source-derived top-level detector names")
        return selected

    def go(label, pattern, network, mutant=False, detector=None, assertion=None):
        if receipt["goInvocations"] >= 4:
            raise RuntimeError("worker cost ceiling")
        names = expected_names(pattern)
        name = "ccmai-r060-worker-" + label + "-" + token
        verify_source("before " + label)
        create = ["docker", "create", "--pull=never", "--name", name, "--network", network,
                  "--mount", f"type=bind,source={root / 'backend'},target=/src,readonly",
                  "--mount", f"type=bind,source={cache},target=/go/pkg/mod,readonly",
                  "--mount", f"type=volume,source={build_cache},target=/root/.cache/go-build", "-w", "/src",
                  "-e", "GOFLAGS=-mod=readonly", "-e", "GOPROXY=off", "-e", "GOSUMDB=off",
                  "-e", "GOTOOLCHAIN=local", "-e", "CGO_ENABLED=0"]
        if network != "none":
            create += ["-e", f"TEST_DB_DSN=ccma:{password}@tcp({database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC"]
        create += ["golang:1.26-alpine", "go", "test", "-json", "-count=1", "-timeout=" + ("480s" if label == "positive" else "180s"), "-run", pattern, "./engine"]
        command(create)
        containers.append(name)
        observed = inspect("container", name)
        volumes.update(m["Name"] for m in observed["Mounts"] if m["Type"] == "volume")
        if observed["HostConfig"]["NetworkMode"] != network or observed["HostConfig"]["PortBindings"]:
            raise RuntimeError("unexpected network or host ports")
        if any(m["RW"] for m in observed["Mounts"] if m["Type"] == "bind"):
            raise RuntimeError("writable source/cache bind")
        record = {"label": label, "command": [a.replace(password, "<SYNTHETIC_FIXTURE>") for a in create],
                  "expectedTopLevel": names, "observed": {"image": observed["Image"],
                  "network": network, "ports": observed["HostConfig"]["PortBindings"], "mounts": observed["Mounts"]}}
        receipt["commands"].append(record)
        receipt["goInvocations"] += 1
        start = time.monotonic()
        try:
            result = command(["docker", "start", "--attach", name], check=False, timeout=600 if label == "positive" else 300)
        except subprocess.TimeoutExpired as exc:
            record["timeoutSeconds"] = 600 if label == "positive" else 300
            record["partialStdout"] = save(label + "_timeout_stdout.log", exc.stdout or b"")
            record["partialStderr"] = save(label + "_timeout_stderr.log", exc.stderr or b"")
            current = inspect("container", name)["State"]
            record["timeoutContainerState"] = current
            snapshot = command(["docker", "logs", name], check=False)
            record["containerStdoutAtTimeout"] = save(label + "_container_timeout_stdout.log", snapshot.stdout)
            record["containerStderrAtTimeout"] = save(label + "_container_timeout_stderr.log", snapshot.stderr)
            raise
        record.update({"seconds": round(time.monotonic() - start, 3), "attachExit": result.returncode,
                       "stdout": save(label + ".jsonl", result.stdout), "stderr": save(label + "_stderr.log", result.stderr)})
        state = inspect("container", name)["State"]
        record["state"] = state
        if state["Running"] or state["OOMKilled"] or state["Error"] or result.returncode != state["ExitCode"]:
            raise RuntimeError("container/attach execution mismatch")
        events = [json.loads(line) for line in result.stdout.decode("utf-8").splitlines() if line.strip()]
        if any(not isinstance(e, dict) or "Action" not in e for e in events):
            raise RuntimeError("invalid JSON event")
        completed = [{"name": e["Test"], "action": e["Action"]} for e in events
                     if e.get("Test") and e["Action"] in ("pass", "fail", "skip")]
        top = [e for e in completed if "/" not in e["name"]]
        if sorted(e["name"] for e in top) != names:
            raise RuntimeError("missing/duplicate/unexpected completed top-level detector")
        output = "".join(e.get("Output", "") for e in events)
        record["events"] = {"completed": completed, "topLevel": len(top),
                            **{action: sum(e["action"] == action for e in completed) for action in ("pass", "fail", "skip")},
                            "buildOrTimeout": bool(re.search(r"\[build failed\]|build-fail|undefined:|test timed out|panic:", output)),
                            "namedAssertion": bool(assertion and assertion in output)}
        stats = record["events"]
        if mutant:
            if result.returncode == 0 or stats["skip"] or stats["buildOrTimeout"] or not stats["namedAssertion"]:
                raise RuntimeError("mutation not killed by intended behavioral stored-receipt assertion")
            if not any(e["name"] == detector and e["action"] == "fail" for e in top):
                raise RuntimeError("stored-receipt detector did not fail")
        elif result.returncode or stats["fail"] or stats["skip"] or stats["buildOrTimeout"]:
            raise RuntimeError("positive control failed")
        print(json.dumps({"command": label, "exit": result.returncode, "topLevel": len(top),
                          "pass": stats["pass"], "fail": stats["fail"], "seconds": record["seconds"]}), flush=True)

    try:
        if not re.fullmatch(r"[0-9a-f]{40}", args.source):
            raise RuntimeError("invalid source identity")
        command(["git", "merge-base", "--is-ancestor", args.source, "HEAD"])
        if not cache.is_dir():
            raise RuntimeError("cached modules absent")
        receipt["images"] = [{"tag": tag, "id": inspect("image", tag)["Id"]}
                             for tag in ("golang:1.26-alpine", "mysql:8.0")]
        archive = command(["git", "archive", "--format=tar", args.source, "backend"]).stdout
        receipt["archive"] = {"command": ["git", "archive", "--format=tar", args.source, "backend"],
                              "bytes": len(archive), "sha256": sha(archive)}
        with tarfile.open(fileobj=io.BytesIO(archive), mode="r:") as tar:
            members = tar.getmembers()
            if any(m.issym() or m.islnk() or m.name.startswith("/") or ".." in Path(m.name).parts for m in members):
                raise RuntimeError("unsafe archive member")
            expected_manifest = sorted([{"path": m.name, "bytes": m.size,
                "sha256": sha(tar.extractfile(m).read())} for m in members if m.isfile()], key=lambda item: item["path"])
            tar.extractall(root, filter="data")
        receipt["manifest"] = save("manifest.json", (json.dumps(expected_manifest, indent=2) + "\n").encode())
        verify_source("before mutation")
        mutation_path = "backend/engine/analyzer.go"
        target = root / mutation_path
        original = target.read_bytes()
        specs = [
            ("M01", b'initialSummary := observedSummary(map[string]interface{}{',
             b'for _, candidate := range prepared { execution.begin("SINGLE", []models.Conversation{candidate.Conv}) }\n\tinitialSummary := observedSummary(map[string]interface{}{',
             "TestEXNoCallConstructionAndPreparation", "logical invocation began without provider method"),
            ("M02", b'finalizeOrdinaryRun(&run, job, runStatus, run.ErrorMessage, string(summaryJSON), finishedAt, checkpoint)',
             b'finalizeOrdinaryRun(&run, job, runStatus, run.ErrorMessage, func() string { var envelope map[string]json.RawMessage; _ = json.Unmarshal([]byte(summaryJSON), &envelope); delete(envelope, "source_execution"); encoded, _ := json.Marshal(envelope); return string(encoded) }(), finishedAt, checkpoint)',
             "TestEXStoredTerminalSingleSuccess", "stored terminal execution receipt missing")]
        for label, needle, replacement, detector, assertion in specs:
            if original.count(needle) != 1:
                raise RuntimeError("mutation match count must be one: " + label)
            mutant = original.replace(needle, replacement, 1)
            diff = "".join(difflib.unified_diff(original.decode().splitlines(True), mutant.decode().splitlines(True),
                fromfile="a/" + mutation_path, tofile="b/" + mutation_path)).encode()
            mutations.append((label, mutant, detector, assertion))
            receipt.setdefault("mutations", []).append({"label": label, "targetPath": mutation_path,
                "appliedInIsolatedCopyOnly": True, "matchCount": 1, "detector": detector, "assertion": assertion,
                "originalSha256": sha(original), "mutantSha256": sha(mutant), "diffSha256": sha(diff),
                "diffBase64": base64.b64encode(diff).decode()})
        receipt["campaigns"] = 1
        command(["docker", "volume", "create", build_cache])
        cache_created = True
        volumes.add(build_cache)
        receipt["buildCache"] = {"name": build_cache, "freshDisposable": True, "rwGoOnly": True}
        command(["docker", "network", "create", "--internal", net])
        network_created = True
        receipt["network"] = inspect("network", net)
        if not receipt["network"]["Internal"]:
            raise RuntimeError("network must be internal")
        command(["docker", "run", "--pull=never", "-d", "--name", database, "--network", net,
                 "-e", "MYSQL_ROOT_PASSWORD=" + password, "-e", "MYSQL_DATABASE=CCMA", "-e", "MYSQL_USER=ccma",
                 "-e", "MYSQL_PASSWORD=" + password, "mysql:8.0", "--log-bin-trust-function-creators=1", "--max-connections=1000"])
        containers.append(database)
        db = inspect("container", database)
        receipt["database"] = {"image": db["Image"], "mounts": db["Mounts"], "ports": db["HostConfig"]["PortBindings"]}
        volumes.update(m["Name"] for m in db["Mounts"] if m["Type"] == "volume")
        if db["HostConfig"]["NetworkMode"] != net or db["HostConfig"]["PortBindings"]:
            raise RuntimeError("database isolation invalid")
        ready = time.monotonic()
        while time.monotonic() - ready < 90:
            ping = command(["docker", "exec", "-e", "MYSQL_PWD=" + password, database,
                            "mysqladmin", "ping", "-uccma", "--silent"], check=False, timeout=10)
            if ping.returncode == 0:
                break
            time.sleep(2)
        else:
            raise RuntimeError("database readiness timeout")
        pattern = "^Test(EX|PreparationReceipt|SP|LP|Snapshot|EvidenceRefs|SingleAndBatchShareSnapshot$|ExplicitModeMatrix$|FullModeVietnamDatesKeepFullSnapshot$|OnlyOrdinaryRunWritesCheckpoint$|OrdinaryRunFailuresKeepCheckpoint$|AdmissionRefusesASecondOwnerForEveryEntryPointAndMode$|CancelTargetsTheExactRunAndHoldsOwnershipUntilExit$|EveryTerminalPathHonorsAnAcceptedCancel$|ProviderPanicIsCleanedUpWithoutLeakingTheValue$|FL03SuccessfulTerminalWrite$|FL03ExhaustedRetriesReturnsBoundedFailure$|FL06SuccessLifecyclePreservation$)"
        go("positive", pattern, net)
        baseline_manifest = expected_manifest
        for index, (label, mutant, detector, assertion) in enumerate(mutations):
            try:
                target.write_bytes(mutant)
                expected_manifest = [dict(entry, bytes=len(mutant), sha256=sha(mutant)) if entry["path"] == mutation_path else entry for entry in baseline_manifest]
                receipt["mutations"][index]["appliedManifest"] = save(label + "_manifest.json", (json.dumps(expected_manifest, indent=2) + "\n").encode())
                verify_source("applied " + label)
                go(label, "^" + detector + "$", net, mutant=True, detector=detector, assertion=assertion)
                receipt["mutations"][index]["disposition"] = "KILLED_BY_NAMED_SEMANTIC_ASSERTION"
            finally:
                target.write_bytes(original)
                expected_manifest = baseline_manifest
                verify_source("restored " + label)
                receipt["mutations"][index]["restoredByteEqual"] = target.read_bytes() == original
        go("restored", "^TestEX", net)
        if receipt["goInvocations"] != 4:
            raise RuntimeError("exact four-call campaign not completed")
        receipt["result"] = "PASS"
    except Exception as exc:
        receipt["errors"].append({"type": type(exc).__name__, "message": str(exc)})
        receipt["result"] = "CHANGES_REQUIRED_NO_RETRY"
    finally:
        for label, mutant, detector, assertion in mutations:
            for item in receipt.get("mutations", []):
                if item["label"] == label: item.setdefault("disposition", "NOT_RUN_OR_INCONCLUSIVE")
        if original is not None:
            expected_manifest = baseline_manifest if "baseline_manifest" in locals() else expected_manifest
            try:
                target.write_bytes(original)
                verify_source("final")
            except Exception as exc:
                receipt["errors"].append({"type": "restoration", "message": str(exc)})
        for name in reversed(containers):
            item = {"name": name, "removeExit": None}
            receipt["cleanup"]["containers"].append(item)
            try:
                item["removeExit"] = command(["docker", "rm", "-fv", name], check=False, timeout=30).returncode
            except Exception as exc:
                receipt["errors"].append({"type": "container cleanup", "message": str(exc)})
        if network_created:
            receipt["cleanup"]["network"] = {"name": net, "removeExit": None}
            try:
                receipt["cleanup"]["network"]["removeExit"] = command(["docker", "network", "rm", net], check=False, timeout=30).returncode
            except Exception as exc:
                receipt["errors"].append({"type": "network cleanup", "message": str(exc)})
        if cache_created:
            try:
                removed = command(["docker", "volume", "rm", build_cache], check=False, timeout=30)
                receipt["cleanup"]["buildCacheRemoveExit"] = removed.returncode
                if removed.returncode:
                    raise RuntimeError("build cache removal failed")
            except Exception as exc:
                receipt["errors"].append({"type": "build cache cleanup", "message": str(exc)})
        try:
            inventories = {}
            for kind, query in (("containers", ["docker", "ps", "-a", "--format", "{{.Names}}"]),
                                ("network", ["docker", "network", "ls", "--format", "{{.Name}}"]),
                                ("volumes", ["docker", "volume", "ls", "--format", "{{.Name}}"] )):
                p = command(query)
                inventories[kind] = p.stdout.decode().splitlines()
            for item in receipt["cleanup"]["containers"]:
                item["absent"] = item["name"] not in inventories["containers"]
            if network_created:
                receipt["cleanup"]["network"]["absent"] = net not in inventories["network"]
            receipt["cleanup"]["volumes"] = [{"name": name, "absent": name not in inventories["volumes"]} for name in sorted(volumes)]
            items = receipt["cleanup"]["containers"] + receipt["cleanup"]["volumes"]
            if network_created:
                items += [receipt["cleanup"]["network"]]
            if any(not item["absent"] or item.get("removeExit", 0) for item in items):
                raise RuntimeError("cleanup not proved")
            receipt["cleanup"]["successfulInventoryExits"] = [0, 0, 0]
        except Exception as exc:
            receipt["errors"].append({"type": "cleanup", "message": str(exc)})
        receipt["seconds"] = round(time.monotonic() - began, 3)
        if receipt["errors"]:
            receipt["result"] = "CHANGES_REQUIRED_NO_RETRY"
        save("summary.json", (json.dumps(receipt, indent=2) + "\n").encode())
        print(json.dumps({"result": receipt["result"], "goInvocations": receipt["goInvocations"],
                          "seconds": receipt["seconds"], "errors": receipt["errors"]}), flush=True)
    return 1 if receipt["errors"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
