"""One offline R051 round-four evidence campaign under delegated R053 authority.

Run from project root after committing BUILD acknowledgment. No retries. Existing
source/tests/history remain protected; only an exact isolated archive is mutated.
"""
import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile
import time
import uuid

SOURCE = "145bd41109c1e3ac3fb1a85f261c61f668b2fe4d"
PRODUCTION = "727d3229338e9b29c749612677a08b7fd1c65428"
SOURCE_ACK = "c228931c35339f56cac0a934394e586082a3f7c1"
AUTHORITY = "a54cb73007f081fe4bbaa4baa11d28fed4a7d937"
ROUTE_SEED = "072812c7ec8e07025eaf90ae668744744f69f3bf"
DETECTOR = "TestLP06EagerInitializationDetectorNegativeAndPositive"
NEGATIVE = DETECTOR + "/negative_control:_no_work_succeeds_without_AI_settings/decryption"
EXPECTED_FAILURE = "negative control failed: expected success without AI settings"
PREFIX = "docs/reviews/probes/r051_r4_"


def sha(data):
    return hashlib.sha256(data).hexdigest()


def run(args, *, timeout=120, check=True):
    result = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                            timeout=timeout, check=False)
    if check and result.returncode:
        raise RuntimeError(f"Command failed ({result.returncode}): {args[0:4]!r}; "
                           + result.stderr.decode("utf-8", errors="replace"))
    return result


def text(result):
    return result.stdout.decode("utf-8").strip()


def inspect(kind, name):
    value = json.loads(text(run(["docker", kind, "inspect", name])))
    if len(value) != 1:
        raise RuntimeError("Unexpected inspect object count")
    return value[0]


def manifest(directory):
    return [{"path": p.relative_to(directory).as_posix(),
             "sha256": sha(p.read_bytes()), "bytes": p.stat().st_size}
            for p in sorted(directory.rglob("*"), key=lambda p: p.relative_to(directory).as_posix()) if p.is_file()]


def git_manifest(commit):
    records = run(["git", "ls-tree", "-r", "-z", commit, "backend"]).stdout.split(b"\0")
    result = []
    for record in records:
        if not record:
            continue
        meta, path = record.split(b"\t", 1)
        mode, kind, oid = meta.split()
        if kind != b"blob" or mode not in (b"100644", b"100755"):
            raise RuntimeError("Unexpected source tree member")
        data = run(["git", "cat-file", "blob", oid.decode()]).stdout
        result.append({"path": path.decode("utf-8"), "sha256": sha(data), "bytes": len(data)})
    return sorted(result, key=lambda x: x["path"])


def event_summary(log, expected):
    events = []
    for line in log.decode("utf-8").splitlines():
        if not line.strip():
            continue
        value = json.loads(line)  # malformed lines fail closed, never discarded
        if not isinstance(value, dict) or "Action" not in value:
            raise RuntimeError("Invalid go test JSON event")
        events.append(value)
    completed = [{"name": e["Test"], "action": e["Action"]} for e in events
                 if e.get("Test") and e["Action"] in ("pass", "fail", "skip")]
    names = [e["name"] for e in completed]
    if set(names) != set(expected) or len(names) != len(expected):
        raise RuntimeError("Missing, duplicate or unexpected completed detector names: " + repr(names))
    if any(e["action"] == "skip" for e in completed):
        raise RuntimeError("Skipped detector is not evidence")
    output = "".join(e.get("Output", "") for e in events)
    build_or_timeout = bool(re.search(r"build-fail|\[build failed\]|undefined:|test timed out|panic:", output))
    return {"completed": completed, "count": len(completed),
            "pass": sum(e["action"] == "pass" for e in completed),
            "fail": sum(e["action"] == "fail" for e in completed),
            "skip": sum(e["action"] == "skip" for e in completed),
            "buildOrTimeout": build_or_timeout,
            "namedAssertionPresent": EXPECTED_FAILURE in output}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--acknowledgment", required=True)
    parser.add_argument("--mode", choices=("prepare", "runtime"), required=True)
    args = parser.parse_args()
    root = Path.cwd().resolve()
    target = root / (PREFIX + ("preparation_receipt.json" if args.mode == "prepare" else "worker_receipt.json"))
    if target.exists():
        raise RuntimeError("Existing campaign receipt: automatic retry forbidden")
    began = time.monotonic()
    runner_hash = sha(Path(__file__).read_bytes())
    token = uuid.uuid4().hex[:10]
    task = Path(tempfile.mkdtemp(prefix="ccmai-r051-r4-"))
    source_root = task / "source"
    source_root.mkdir()
    network = "ccmai-r051-r4-net-" + token
    database = "ccmai-r051-r4-db-" + token
    # Synthetic disposable fixture; never reads project .env or real config.
    fixture_password = uuid.uuid4().hex
    receipt = {"schemaVersion": "1.0", "trancheId": "CCMAI-RUNTIME-053",
               "inheritedTranche": "CCMAI-RUNTIME-051", "repairRound": 4,
               "campaigns": 0, "goInvocations": 0, "automaticRetries": 0,
               "startedUtc": dt.datetime.now(dt.timezone.utc).isoformat(),
               "artifactCommit": "DEFERRED_TO_LATER_COMMITTED_HANDBACK",
               "sourceCommit": SOURCE, "inheritedProductionCommit": PRODUCTION,
               "sourceAcknowledgmentCommit": SOURCE_ACK, "originalAuthorityCommit": AUTHORITY,
               "routeAuthorityCommit": ROUTE_SEED, "acknowledgmentCommit": args.acknowledgment,
               "taskRoot": str(task), "commands": [], "errors": [],
               "mode": args.mode, "runnerSha256": runner_hash,
               "invocation": {"command": ["python", "-B", "docs/reviews/probes/r051_r4_campaign.py", "--acknowledgment", args.acknowledgment, "--mode", args.mode]},
               "aggregateBudget": {"priorCampaigns": 0, "priorGoInvocations": 0, "maxCampaigns": 1, "maxGoInvocations": 4},
               "inherited": {"acceptedR2Review": "42 top-level / 98 PASS; INHERITED_NOT_RERUN",
                             "RP01_RP02_RP03": "ACCEPTED_INHERITED",
                             "productionBuildVet": "INHERITED_BYTE_IDENTICAL_PRODUCTION"},
               "notRun": ["full engine", "full backend", "frontend", "race", "real provider",
                          "channel", "CVF runtime governance", "hosted readiness"],
               "cleanup": {"containers": [], "network": {}, "volumes": []}}
    created_containers = []
    captured_volumes = []
    created_network = False
    analyzer = None
    original = None

    def save(name, data):
        p = root / (PREFIX + name)
        p.write_bytes(data)
        return {"path": p.relative_to(root).as_posix(), "sha256": sha(data), "bytes": len(data)}

    def verify_manifest(label, expected):
        actual = manifest(source_root)
        match = actual == expected
        receipt.setdefault("manifestChecks", []).append({"name": label, "files": len(actual),
                                                         "exactMembershipAndBytes": match})
        if not match:
            raise RuntimeError("Archive source manifest mismatch: " + label)

    def apply_mutation():
        data = original
        function_start = data.index(b"func (a *Analyzer) executeReserved(")
        scoped = data[function_start:]
        pattern = rb"(\tvar provider ai\.AIProvider\r?\n)(\r?\n\t// Select the conversations and prepare their snapshots\.)"
        matches = list(re.finditer(pattern, scoped))
        receipt.setdefault("mutation", {})["expectedMatchCount"] = 1
        receipt["mutation"]["actualMatchCount"] = len(matches)
        if len(matches) != 1:
            raise RuntimeError("Scoped eager insertion match count is not one")
        newline = b"\r\n" if b"\r\n" in data else b"\n"
        insertion = newline.join([
            b"\tif injectedProvider != nil {", b"\t\tprovider = injectedProvider",
            b"\t} else if a.providerOverride != nil {", b"\t\tprovider = a.providerOverride",
            b"\t} else if a.providerResolver != nil {", b"\t\tvar provErr error",
            b"\t\tprovider, provErr = a.providerResolver(job)", b"\t\tif provErr != nil {",
            '\t\t\treturn a.failOwnedRun(owner, &run, job, "Không khởi tạo được AI provider; kiểm tra cấu hình AI trong Cài đặt.", earlyProviderUnavailable)'.encode(),
            b"\t\t}", b"\t} else {", b"\t\tvar provErr error",
            b"\t\tprovider, provErr = a.getProvider(job)", b"\t\tif provErr != nil {",
            '\t\t\treturn a.failOwnedRun(owner, &run, job, "Không khởi tạo được AI provider; kiểm tra cấu hình AI trong Cài đặt.", earlyProviderUnavailable)'.encode(),
            b"\t\t}", b"\t}", b""])
        position = function_start + matches[0].end(1)
        mutated = data[:position] + insertion + data[position:]
        analyzer.write_bytes(mutated)
        if mutated == data:
            raise RuntimeError("Mutation did not change bytes")
        receipt["mutation"].update(name="M_EAGER_INIT_ORDERING", target="backend/engine/analyzer.go",
                                   baselineSha256=sha(data), mutantSha256=sha(mutated))
        import difflib
        diff = "".join(difflib.unified_diff(data.decode().splitlines(True), mutated.decode().splitlines(True),
                                          fromfile="baseline/backend/engine/analyzer.go",
                                          tofile="mutant/backend/engine/analyzer.go"))
        receipt["mutation"]["appliedDiff"] = save("applied_mutation.diff", diff.encode())

    def restore(label):
        analyzer.write_bytes(original)
        equality = analyzer.read_bytes() == original
        receipt.setdefault("restoration", []).append({"name": label, "sha256": sha(analyzer.read_bytes()),
                                                     "byteEquality": equality})
        if not equality:
            raise RuntimeError("Failed byte restoration")

    def go_container(label, arguments, network_mode, mod_cache):
        if receipt["goInvocations"] >= 4:
            raise RuntimeError("Go invocation cost ceiling reached")
        name = "ccmai-r051-r4-" + label + "-" + token
        command = ["docker", "create", "--pull=never", "--name", name, "--network", network_mode,
                   "--mount", f"type=bind,source={source_root / 'backend'},target=/src,readonly",
                   "--mount", f"type=bind,source={mod_cache},target=/go/pkg/mod,readonly", "-w", "/src",
                   "-e", "GOFLAGS=-mod=readonly", "-e", "GOPROXY=off", "-e", "GOSUMDB=off",
                   "-e", "GOTOOLCHAIN=local", "-e", "CGO_ENABLED=0"]
        if network_mode != "none":
            command += ["-e", f"TEST_DB_DSN=ccma:{fixture_password}@tcp({database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC"]
        command += ["golang:1.26-alpine", "go"] + arguments
        run(command)
        created_containers.append(name)
        observed = inspect("container", name)
        settings = {"name": name, "imageId": observed["Image"],
                    "networkMode": observed["HostConfig"]["NetworkMode"],
                    "portBindings": observed["HostConfig"]["PortBindings"],
                    "networkSettings": observed["NetworkSettings"]["Networks"],
                    "mounts": [{k: m.get(k) for k in ("Type", "Source", "Destination", "RW", "Name")}
                               for m in observed["Mounts"]]}
        if settings["networkMode"] != network_mode or settings["portBindings"]:
            raise RuntimeError("Unexpected command network/host port settings")
        if any(m["RW"] for m in settings["mounts"] if m["Type"] == "bind"):
            raise RuntimeError("Source/module bind is not readonly")
        captured_volumes.extend(m["Name"] for m in settings["mounts"] if m["Type"] == "volume")
        record = {"name": label, "command": [a.replace(fixture_password, "<SYNTHETIC_FIXTURE_PASSWORD>") for a in command],
                  "executionCommand": ["docker", "start", "--attach", name], "observed": settings}
        receipt["commands"].append(record)
        receipt["goInvocations"] += 1
        started = time.monotonic()
        try:
            result = run(["docker", "start", "--attach", name], timeout=180, check=False)
            record["attachExit"] = result.returncode
            record["stdout"] = save(label + ".jsonl" if "control" in label else label + ".log", result.stdout)
            record["stderr"] = save(label + "_stderr.log", result.stderr)
            state = inspect("container", name)["State"]
            record["state"] = state
            record["exit"] = state["ExitCode"]
            if state["Running"] or state["Error"] or state["OOMKilled"]:
                raise RuntimeError("Go container state/harness failure")
            if result.returncode != state["ExitCode"]:
                raise RuntimeError("Attach/container exit disagreement")
            return record, result.stdout
        finally:
            record["seconds"] = round(time.monotonic() - started, 3)

    try:
        for label, commit in [("source", SOURCE), ("production", PRODUCTION), ("sourceAck", SOURCE_ACK),
                              ("authority", AUTHORITY), ("routeSeed", ROUTE_SEED), ("r3Ack", args.acknowledgment)]:
            actual = text(run(["git", "rev-parse", "--verify", commit + "^{commit}"]))
            if actual != commit:
                raise RuntimeError("Commit resolution changed")
            run(["git", "merge-base", "--is-ancestor", commit, "HEAD"])
        receipt["identitiesVerified"] = True
        receipt["campaignParentHead"] = text(run(["git", "rev-parse", "HEAD"]))
        raw_manifest = git_manifest(SOURCE)
        receipt["rawBlobManifestDiagnostic"] = save("raw_blob_manifest.json", (json.dumps(raw_manifest, indent=2) + "\n").encode())
        archive = task / "source.tar"
        run(["git", "archive", "--format=tar", "--output=" + str(archive), SOURCE, "backend"])
        receipt["archiveSha256"] = sha(archive.read_bytes())
        if receipt["archiveSha256"] != "b4d4daff222388d1f721294b0d2ab47899bb70943966bff1742b7c045054be38":
            raise RuntimeError("Fixed source archive identity mismatch")
        with tarfile.open(archive) as bundle:
            expected = []
            for member in bundle.getmembers():
                resolved = (source_root / member.name).resolve()
                if not resolved.is_relative_to(source_root.resolve()) or member.issym() or member.islnk():
                    raise RuntimeError("Unsafe archive member")
                if member.isfile():
                    payload = bundle.extractfile(member).read()
                    expected.append({"path": member.name, "sha256": sha(payload), "bytes": len(payload)})
            expected.sort(key=lambda row: row["path"])
            if len(expected) != 204 or len({row["path"] for row in expected}) != 204:
                raise RuntimeError("Archive member count/uniqueness mismatch")
            receipt["sourceManifest"] = save("source_manifest.json", (json.dumps(expected, indent=2) + "\n").encode())
            receipt["sourceFileCount"] = len(expected)
            receipt["manifestBasis"] = "EXACT_GIT_ARCHIVE_MEMBER_BYTES_NOT_RAW_BLOBS"
            raw = {row["path"]: row for row in raw_manifest}
            receipt["rawBlobDiagnostic"] = {"notUsedForAcceptance": True, "differentFiles": [
                {"path": row["path"], "rawBlob": raw[row["path"]], "archiveMember": row}
                for row in expected if row != raw[row["path"]]]}
            bundle.extractall(source_root, filter="data")
        verify_manifest("before mutation", expected)
        mod_cache = Path(text(run(["go", "env", "GOMODCACHE"])))
        if not mod_cache.is_dir():
            raise RuntimeError("Offline module cache missing")
        receipt["moduleCache"] = {"path": str(mod_cache), "exists": True, "readOnlyMount": True,
                                  "staticReachability": "Not assumed; budgeted isolated compile fails closed"}
        receipt["cachedImages"] = []
        for image in ("mysql:8.0", "golang:1.26-alpine"):
            ob = inspect("image", image)
            receipt["cachedImages"].append({"tag": image, "id": ob["Id"]})
        analyzer = source_root / "backend/engine/analyzer.go"
        original = analyzer.read_bytes()
        import ast
        ast.parse(Path(__file__).read_text(encoding="utf-8"))
        receipt["syntaxPreparation"] = "Python AST PASS; no bytecode or runtime proof"
        maintained_test = (source_root / "backend/engine/analyzer_provider_initialization_test.go").read_text(encoding="utf-8")
        expected_leaf_literals = [
            "negative control: no work succeeds without AI settings/decryption",
            "positive control: eligible work strictly requires provider initialization (missing key)",
            "positive control: eligible work strictly requires provider initialization (corrupt key)"]
        detector_start = maintained_test.index("func " + DETECTOR + "(")
        detector_end = maintained_test.find("\nfunc ", detector_start + 1)
        detector_source = maintained_test[detector_start:detector_end if detector_end >= 0 else None]
        actual_literals = re.findall(r't\.Run\("([^"\n]+)"', detector_source)
        if actual_literals != expected_leaf_literals:
            raise RuntimeError("Maintained LP06 detector names mismatch")
        receipt["detectorPreparation"] = {"expectedLeafLiterals": expected_leaf_literals,
                                           "actualLeafLiterals": actual_literals, "exactMatch": True}
        if args.mode == "prepare":
            try:
                apply_mutation()
                receipt["mutationConstructionPreparation"] = {"changedBytes": analyzer.read_bytes() != original,
                    "exactOneMatch": receipt["mutation"]["actualMatchCount"] == 1,
                    "runtimeProof": "NOT_RUN"}
            finally:
                restore("pure offline construction preparation finally")
            verify_manifest("after preparation restoration", expected)
            receipt["preparationResult"] = "PASS_NOT_RUNTIME_PROOF"
            receipt["campaignResult"] = "NOT_RUN_PREPARATION_ONLY"
            return 0
        preparation_path = root / (PREFIX + "preparation_receipt.json")
        preparation = json.loads(preparation_path.read_text(encoding="utf-8"))
        if preparation.get("preparationResult") != "PASS_NOT_RUNTIME_PROOF" or preparation.get("errors"):
            raise RuntimeError("Successful preparation receipt required")
        if preparation["runnerSha256"] != runner_hash or preparation["acknowledgmentCommit"] != args.acknowledgment:
            raise RuntimeError("Prepared runner/acknowledgment identity changed")
        if preparation["sourceManifest"]["sha256"] != receipt["sourceManifest"]["sha256"]:
            raise RuntimeError("Prepared source manifest identity changed")
        receipt["preparationReceipt"] = {"path": preparation_path.relative_to(root).as_posix(),
                                         "sha256": sha(preparation_path.read_bytes()), "result": preparation["preparationResult"]}
        try:
            try:
                apply_mutation()
                raise RuntimeError("R051_R4_EXPECTED_CONTROLLED_RESTORATION_THROW")
            finally:
                restore("controlled throw finally")
        except RuntimeError as exc:
            if str(exc) != "R051_R4_EXPECTED_CONTROLLED_RESTORATION_THROW":
                raise
            receipt["controlledThrow"] = {"expectedError": str(exc), "caughtOnlyExpected": True,
                                          "restoredBeforeCampaign": analyzer.read_bytes() == original}
        verify_manifest("after controlled throw", expected)
        receipt["campaigns"] = 1
        run(["docker", "network", "create", "--internal", network])
        created_network = True
        net_observed = inspect("network", network)
        receipt["network"] = {"name": network, "id": net_observed["Id"], "internal": net_observed["Internal"]}
        if not net_observed["Internal"]:
            raise RuntimeError("Campaign network is not internal")
        db_command = ["docker", "run", "--pull=never", "-d", "--name", database, "--network", network,
                      "-e", "MYSQL_ROOT_PASSWORD=" + fixture_password, "-e", "MYSQL_DATABASE=CCMA",
                      "-e", "MYSQL_USER=ccma", "-e", "MYSQL_PASSWORD=" + fixture_password,
                      "mysql:8.0", "--log-bin-trust-function-creators=1", "--max-connections=1000"]
        run(db_command)
        created_containers.append(database)
        db = inspect("container", database)
        receipt["database"] = {"name": database, "command": [a.replace(fixture_password, "<SYNTHETIC_FIXTURE_PASSWORD>") for a in db_command],
                               "imageId": db["Image"], "networkMode": db["HostConfig"]["NetworkMode"],
                               "portBindings": db["HostConfig"]["PortBindings"], "mounts": db["Mounts"]}
        captured_volumes.extend(m["Name"] for m in db["Mounts"] if m["Type"] == "volume")
        if db["HostConfig"]["NetworkMode"] != network or db["HostConfig"]["PortBindings"]:
            raise RuntimeError("Database network/ports out of boundary")
        if not captured_volumes:
            raise RuntimeError("Anonymous MySQL volume not captured")
        ready_started = time.monotonic()
        polls = []
        while time.monotonic() - ready_started < 90:
            probe = run(["docker", "exec", "-e", "MYSQL_PWD=" + fixture_password, database,
                         "mysql", "-uccma", "-N", "-e", "SELECT @@log_bin_trust_function_creators", "CCMA"],
                        timeout=5, check=False)
            polls.append({"exit": probe.returncode, "readyValue": text(probe)})
            if probe.returncode == 0 and text(probe) == "1":
                break
            time.sleep(1)
        else:
            raise RuntimeError("Disposable DB readiness timed out")
        receipt["readiness"] = {"seconds": round(time.monotonic() - ready_started, 3), "polls": polls,
                                "maxSeconds": 90, "ready": True}
        for label, arguments in [("compile", ["test", "-c", "-o", "/dev/null", "./engine"]),
                                 ("vet", ["vet", "./engine"])]:
            record, _ = go_container(label, arguments, "none", mod_cache)
            if record["exit"] != 0:
                raise RuntimeError(label + " failed; no retry")
        try:
            apply_mutation()
            record, log = go_container("mutant-control", ["test", "./engine", "-json", "-count=1", "-p", "1",
                                                           "-timeout", "120s", "-run", "^" + DETECTOR + "$/^negative_control:"], network, mod_cache)
            summary = event_summary(log, [DETECTOR, NEGATIVE])
            record["testSummary"] = summary
            killed = record["exit"] == 1 and summary["fail"] == 2 and summary["namedAssertionPresent"] and not summary["buildOrTimeout"]
            receipt["mutation"]["killedBehaviorally"] = killed
            if not killed:
                raise RuntimeError("Mutation not killed by named no-work assertion")
        finally:
            restore("actual mutation finally")
        verify_manifest("after actual mutation restoration", expected)
        record, log = go_container("restored-control", ["test", "./engine", "-json", "-count=1", "-p", "1",
                                                        "-timeout", "120s", "-run", "^" + DETECTOR + "$"], network, mod_cache)
        # Exact maintained leaf names are read from accepted source, never supplied by an overlay.
        names = [DETECTOR, NEGATIVE,
                 DETECTOR + "/positive_control:_eligible_work_strictly_requires_provider_initialization_(missing_key)",
                 DETECTOR + "/positive_control:_eligible_work_strictly_requires_provider_initialization_(corrupt_key)"]
        summary = event_summary(log, names)
        record["testSummary"] = summary
        if record["exit"] != 0 or summary["pass"] != 4 or summary["fail"] or summary["buildOrTimeout"]:
            raise RuntimeError("Restored full LP06 suite did not pass")
        receipt["campaignResult"] = "PASS_PENDING_INDEPENDENT_REVIEW"
    except Exception as exc:
        receipt["errors"].append({"type": type(exc).__name__, "message": str(exc)})
        receipt["campaignResult"] = "CHANGES_REQUIRED_NO_RETRY"
    finally:
        if analyzer is not None and original is not None:
            try:
                restore("outer finally safety")
            except Exception as exc:
                receipt["errors"].append({"type": "restoration", "message": str(exc)})
        receipt["cleanup"]["capturedVolumeNamesBeforeTeardown"] = sorted(set(captured_volumes))
        for name in reversed(created_containers):
            result = run(["docker", "rm", "-f", "-v", name], check=False)
            receipt["cleanup"]["containers"].append({"name": name, "removeExit": result.returncode})
        if created_network:
            result = run(["docker", "network", "rm", network], check=False)
            receipt["cleanup"]["network"] = {"name": network, "removeExit": result.returncode}
        try:
            container_inventory = text(run(["docker", "ps", "-a", "--format", "{{.Names}}"])).splitlines()
            network_inventory = text(run(["docker", "network", "ls", "--format", "{{.Name}}"])).splitlines()
            volume_inventory = text(run(["docker", "volume", "ls", "--format", "{{.Name}}"])).splitlines()
            for item in receipt["cleanup"]["containers"]:
                item["verifiedAbsent"] = item["name"] not in container_inventory
                if item["removeExit"] or not item["verifiedAbsent"]:
                    raise RuntimeError("Container teardown not proved")
            if created_network:
                item = receipt["cleanup"]["network"]
                item["verifiedAbsent"] = network not in network_inventory
                if item["removeExit"] or not item["verifiedAbsent"]:
                    raise RuntimeError("Network teardown not proved")
            receipt["cleanup"]["volumes"] = [{"name": n, "verifiedAbsent": n not in volume_inventory}
                                              for n in sorted(set(captured_volumes))]
            if any(not item["verifiedAbsent"] for item in receipt["cleanup"]["volumes"]):
                raise RuntimeError("Anonymous volume teardown not proved")
            receipt["cleanup"]["successfulInventoryCommands"] = [
                {"command": ["docker", "ps", "-a", "--format", "{{.Names}}"], "exit": 0},
                {"command": ["docker", "network", "ls", "--format", "{{.Name}}"], "exit": 0},
                {"command": ["docker", "volume", "ls", "--format", "{{.Name}}"], "exit": 0}]
        except Exception as exc:
            receipt["errors"].append({"type": "cleanup", "message": str(exc)})
        if receipt["errors"]:
            receipt["campaignResult"] = "CHANGES_REQUIRED_NO_RETRY"
        receipt["totalSeconds"] = round(time.monotonic() - began, 3)
        receipt["invocation"]["exit"] = 1 if receipt["errors"] else 0
        if args.mode == "prepare" and receipt["errors"]:
            receipt["preparationResult"] = "FAILED_NO_RETRY"
        receipt["publication"] = {"status": "PENDING_FINAL_MARKDOWN_AND_GATES"}
        target.write_text(json.dumps(receipt, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(json.dumps({"receipt": str(target), "result": receipt.get("campaignResult"),
                          "goInvocations": receipt["goInvocations"], "seconds": receipt["totalSeconds"],
                          "errors": receipt["errors"]}, ensure_ascii=False))
    return 1 if receipt["errors"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
