"""R055 worker exact-source application campaign; one campaign, eight Go calls.

Mechanics adapted from root read-only independent harness prepared before execution.

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
    parser.add_argument("--acknowledgment", required=True)
    parser.add_argument("--detector", required=True)
    parser.add_argument("--assertion", required=True)
    args = parser.parse_args()
    prefix = Path("docs/reviews/probes/r055_worker_")
    token = uuid.uuid4().hex[:10]
    root = Path(tempfile.mkdtemp(prefix="ccmai-r055-worker-"))
    net, database = "ccmai-r055-worker-net-" + token, "ccmai-r055-worker-db-" + token
    # cvf-allow-secret-fixture: disposable fixture only, never a real credential.
    password = "r055_synthetic_fixture"
    cache = Path(os.environ["USERPROFILE"]) / "go/pkg/mod"
    receipt = {"sourceCommit": args.source, "worker": "Codex /root/r051_preparation", "independentReviewer": "Codex /root", "acknowledgmentCommit": args.acknowledgment, "authoritySeedCommit": "8d9137d8fd4c23d08744abc30b0b9975c68da626", "activationCommit": "f0bc1518d1246f5ffa4090b7a801563c2b4d18b6", "runnerSha256": sha(Path(__file__).read_bytes()), "artifactCommit": "DEFERRED_TO_LATER_HANDBACK", "campaigns": 0,
               "goInvocations": 0, "automaticRetries": 0, "commands": [], "errors": [],
               "sourceDirectory": str(root), "scope": "application_observation_only",
               "cleanup": {"containers": [], "volumes": [], "network": None}}
    containers, volumes = [], set()
    network_created = False
    original = target = expected_manifest = None
    if Path(str(prefix)+"summary.json").exists():raise RuntimeError("Existing worker receipt; retry forbidden")
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

    def go(label, pattern, network, mutant=False, operation=None):
        if receipt["goInvocations"] >= 8:
            raise RuntimeError("review cost ceiling")
        names = expected_names(pattern) if pattern else []
        name = "ccmai-r055-worker-" + label + "-" + token
        create = ["docker", "create", "--pull=never", "--name", name, "--network", network,
                  "--mount", f"type=bind,source={root / 'backend'},target=/src,readonly",
                  "--mount", f"type=bind,source={cache},target=/go/pkg/mod,readonly", "-w", "/src",
                  "-e", "GOFLAGS=-mod=readonly", "-e", "GOPROXY=off", "-e", "GOSUMDB=off",
                  "-e", "GOTOOLCHAIN=local", "-e", "CGO_ENABLED=0"]
        if network != "none":
            create += ["-e", f"TEST_DB_DSN=ccma:{password}@tcp({database}:3306)/CCMA?charset=utf8mb4&parseTime=True&loc=UTC"]
        create += ["golang:1.26-alpine", "go"] + (operation if operation else ["test", "-json", "-count=1", "-timeout=180s", "-run", pattern, "./engine"])
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
        result = command(["docker", "start", "--attach", name], check=False, timeout=240)
        record.update({"seconds": round(time.monotonic() - start, 3), "attachExit": result.returncode,
                       "stdout": save(label + ".jsonl", result.stdout), "stderr": save(label + "_stderr.log", result.stderr)})
        state = inspect("container", name)["State"]
        record["state"] = state
        if state["Running"] or state["OOMKilled"] or state["Error"] or result.returncode != state["ExitCode"]:
            raise RuntimeError("container/attach execution mismatch")
        if operation:
            if result.returncode: raise RuntimeError(label+" failed")
            print(json.dumps({"command":label,"exit":result.returncode,"seconds":record["seconds"]}),flush=True)
            return
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
                            "namedAssertion": args.assertion in output}
        stats = record["events"]
        if mutant:
            if result.returncode == 0 or stats["skip"] or stats["buildOrTimeout"] or not stats["namedAssertion"]:
                raise RuntimeError("mutation not killed by intended behavioral stored-receipt assertion")
            if not any(e["name"] == args.detector and e["action"] == "fail" for e in top):
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
        receipt["moduleCache"]={"path":str(cache),"exists":True,"mount":"read_only","discovery":"USERPROFILE path and inherited receipt; no host Go invocation"}
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
        target = root / "backend/engine/source_preparation_receipt.go"
        original = target.read_bytes()
        needle = b'values["source_preparation"] = receipt'
        if original.count(needle) != 1:
            raise RuntimeError("mutation match count must be one")
        mutant = original.replace(needle, b'_ = receipt // independent applied receipt erasure', 1)
        diff = "".join(difflib.unified_diff(original.decode().splitlines(True), mutant.decode().splitlines(True),
                       fromfile="a/backend/engine/source_preparation_receipt.go", tofile="b/backend/engine/source_preparation_receipt.go")).encode()
        receipt["mutation"] = {"targetPath":"backend/engine/source_preparation_receipt.go", "appliedInIsolatedCopyOnly":True,"matchCount": 1, "originalSha256": sha(original), "mutantSha256": sha(mutant),
                               "diffSha256": sha(diff), "diffBase64": base64.b64encode(diff).decode()}
        try:
            try:
                target.write_bytes(mutant)
                raise RuntimeError("R055_CONTROLLED_RESTORE")
            finally:
                target.write_bytes(original)
        except RuntimeError as exc:
            if str(exc) != "R055_CONTROLLED_RESTORE":
                raise
            receipt["controlledThrow"] = {"caughtOnlyExpected": True, "restored": target.read_bytes() == original, "restoredSha256":sha(target.read_bytes()), "expectedError":"R055_CONTROLLED_RESTORE"}
        verify_source("after controlled throw")
        receipt["campaigns"] = 1
        go("compile", None, "none", operation=["test","-c","-o","/dev/null","./engine"])
        go("vet", None, "none", operation=["vet","./engine"])
        go("pure", "^TestPreparationReceipt", "none")
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
        receipt["database"] = {"name":database, "image": db["Image"], "mounts": db["Mounts"], "networkMode":db["HostConfig"]["NetworkMode"], "ports": db["HostConfig"]["PortBindings"]}
        volumes.update(m["Name"] for m in db["Mounts"] if m["Type"] == "volume")
        if db["HostConfig"]["NetworkMode"] != net or db["HostConfig"]["PortBindings"]:
            raise RuntimeError("database isolation invalid")
        ready = time.monotonic()
        polls=[]
        while time.monotonic() - ready < 90:
            ping = command(["docker", "exec", "-e", "MYSQL_PWD=" + password, database,
                            "mysql", "-uccma", "-N", "-e", "SELECT @@log_bin_trust_function_creators", "CCMA"], check=False, timeout=10)
            polls.append({"exit":ping.returncode,"stdout":ping.stdout.decode().strip()})
            if ping.returncode == 0 and ping.stdout.decode().strip()=="1":
                break
            time.sleep(2)
        else:
            raise RuntimeError("database readiness timeout")
        receipt["readiness"]={"seconds":round(time.monotonic()-ready,3),"maxSeconds":90,"polls":polls,"ready":True}
        def file_pattern(files):
            names=[]
            for filename in files:
                names+=re.findall(r"^func (Test\w+)\(t \*testing\.T\)",(root/"backend/engine"/filename).read_text(),re.M)
            if not names or len(names)!=len(set(names)):raise RuntimeError("empty/duplicate regression file test names")
            return "^("+"|".join(sorted(names))+")$"
        go("integration", "^TestSP", net)
        go("source-regressions",file_pattern(["analyzer_provider_initialization_test.go","analyzer_incremental_test.go","analyzer_modes_test.go","analyzer_modes_acceptance_test.go","snapshot_test.go","snapshot_db_test.go"]),net)
        ownership_files=sorted(p.name for p in (root/"backend/engine").glob("analyzer_f06*_test.go"))+["job_run_ownership_test.go","analyzer_finalizer_logging_test.go"]
        go("ownership-regressions",file_pattern(ownership_files),net)
        try:
            target.write_bytes(mutant)
            go("mutant", "^" + args.detector + "$", net, mutant=True)
        finally:
            target.write_bytes(original)
            verify_source("after actual mutation")
        go("restored", "^Test(SP|PreparationReceipt)", net)
        if receipt["goInvocations"]!=8:raise RuntimeError("exact worker8 sequence not completed")
        receipt["result"] = "PASS_PENDING_INDEPENDENT_REVIEW"
    except Exception as exc:
        receipt["errors"].append({"type": type(exc).__name__, "message": str(exc)})
        receipt["result"] = "CHANGES_REQUIRED_NO_RETRY"
    finally:
        if original is not None:
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
        try:
            inventories = {}
            for kind, query in (("containers", ["docker", "ps", "-a", "--format", "{{.Names}}"]),
                                ("network", ["docker", "network", "ls", "--format", "{{.Name}}"]),
                                ("volumes", ["docker", "volume", "ls", "--format", "{{.Name}}"] )):
                p = command(query)
                inventories[kind] = p.stdout.decode().splitlines()
                receipt["cleanup"].setdefault("inventories",[]).append({"command":query,"exit":p.returncode,"names":inventories[kind]})
            for item in receipt["cleanup"]["containers"]:
                item["absent"] = item["name"] not in inventories["containers"]
            if network_created:
                receipt["cleanup"]["network"]["absent"] = net not in inventories["network"]
            receipt["cleanup"]["volumes"] = [{"name": name, "absent": name not in inventories["volumes"],"removalMechanism":"docker rm -fv of captured database container"} for name in sorted(volumes)]
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
