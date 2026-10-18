#!/usr/bin/env python3
"""Root-only R117 fixture supervisor. No implicit provider or installed activation.

cvf-allow-secret-fixture: DB_PASSWORD is a disposable synthetic fixture value.
The actual Alibaba key is read only after committed live admission and is passed
to one owned test child through its private environment, never argv or receipts.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone

ROOT = Path(__file__).resolve().parent.parent
RUN = ROOT / ".cvf/runtime/r117_execution"
EVIDENCE = ROOT / "docs/reviews/probes/r117_root"
DOCKER = Path("C:/Users/tiennm/AppData/Local/Programs/DockerDesktop/resources/bin/docker.exe")
DOCKER_SHA = "6c699720bb9523dedb9eb5d22a49951ea9210dbdb1314a3554029eee9be76896"
IMAGE = "sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b"
NAME = "ccmai-r117-root-20261017"
DB_PASSWORD = "r117-synthetic-only"
LABEL = "ccmai.tranche=CCMAI-RUNTIME-117"
SOURCE_ADMISSION = EVIDENCE / "source_binary_admission.json"
RUNTIME_ADMISSION = EVIDENCE / "runtime_admission.json"
LIVE_ADMISSION = EVIDENCE / "live_admission.json"
ENDPOINT_SHA = "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f"


def utc():
    return datetime.now(timezone.utc).isoformat()


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def load(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def exclusive(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("x", encoding="utf-8") as stream:
        json.dump(value, stream, ensure_ascii=False, indent=2)
        stream.write("\n")


def committed(path):
    rel = path.relative_to(ROOT).as_posix()
    result = subprocess.run(["git", "show", "HEAD:" + rel], cwd=ROOT,
                            capture_output=True, timeout=15)
    if result.returncode or result.stdout.replace(b"\r\n", b"\n") != path.read_bytes().replace(b"\r\n", b"\n"):
        raise RuntimeError("ADMISSION_NOT_COMMITTED")
    return load(path)


def minimal_env():
    return {k: v for k, v in os.environ.items()
            if k.upper() in {"SYSTEMROOT", "WINDIR", "TEMP", "TMP", "PATH", "USERPROFILE", "LOCALAPPDATA"}}


def docker(*args, timeout=15, checked=True):
    if sha(DOCKER) != DOCKER_SHA:
        raise RuntimeError("DOCKER_BINARY_CHANGED")
    config = RUN / "docker-cli-config"
    config.mkdir(parents=True, exist_ok=True)
    if not (config / "config.json").exists():
        exclusive(config / "config.json", {})
    argv = [str(DOCKER), "--host", "npipe:////./pipe/dockerDesktopLinuxEngine", "--config", str(config), *args]
    commands = EVIDENCE / "docker_commands"
    commands.mkdir(parents=True, exist_ok=True)
    # Exclusive directory is a permanent command reservation, including timeout.
    index = len(list(commands.iterdir())) + 1
    directory = commands / f"{index:04d}"
    directory.mkdir(exist_ok=False)
    exclusive(directory / "command.json", {"argv": argv, "startedAt": utc(), "timeoutSeconds": timeout})
    try:
        result = subprocess.run(argv, cwd=ROOT, env=minimal_env(), capture_output=True, timeout=timeout)
    except subprocess.TimeoutExpired as error:
        (directory / "stdout.txt").write_bytes(error.stdout or b"")
        (directory / "stderr.txt").write_bytes(error.stderr or b"")
        exclusive(directory / "exit.json", {"finishedAt": utc(), "exitCode": None, "timeout": True})
        raise RuntimeError("DOCKER_TIMEOUT_STOP_NO_RESTART") from None
    (directory / "stdout.txt").write_bytes(result.stdout)
    (directory / "stderr.txt").write_bytes(result.stderr)
    exclusive(directory / "exit.json", {"finishedAt": utc(), "exitCode": result.returncode, "timeout": False})
    if checked and result.returncode:
        raise RuntimeError("DOCKER_COMMAND_FAILED")
    return result


def inventory():
    rows = [json.loads(line) for line in docker("ps", "-a", "--no-trunc", "--format", "{{json .}}").stdout.splitlines()]
    return sorted([{k: row.get(k) for k in ("ID", "Image", "Names", "State", "Networks", "Mounts", "Ports")}
                   for row in rows if row.get("Names") != NAME], key=lambda row: row["ID"])


def resource_inventory():
    volumes = docker("volume", "ls", "--format", "{{json .}}").stdout.splitlines()
    networks = docker("network", "ls", "--no-trunc", "--format", "{{json .}}").stdout.splitlines()
    return {"volumes": sorted([json.loads(line) for line in volumes if json.loads(line).get("Name") != NAME], key=lambda row: row["Name"]),
            "networks": sorted([json.loads(line) for line in networks if json.loads(line).get("Name") != NAME], key=lambda row: row["ID"])}


def safe_relative(path, prefix):
    if not isinstance(path, str) or "\\" in path or not path.startswith(prefix) or Path(path).is_absolute() or ".." in Path(path).parts:
        raise RuntimeError("ADMISSION_PATH_INVALID")
    if not (ROOT / path).resolve().is_relative_to(ROOT.resolve()):
        raise RuntimeError("ADMISSION_PATH_OUTSIDE_PROJECT")
    return ROOT / path


def verify_source():
    value = committed(SOURCE_ADMISSION)
    if value.get("schemaVersion") != 1 or value.get("trancheId") != "CCMAI-RUNTIME-117" or value.get("reviewDisposition") != "ROOT_ACCEPTED_FOR_ISOLATED_PROOF":
        raise RuntimeError("SOURCE_NOT_ADMITTED")
    required = {"backend/engine/intervention_waitdata_executor.go", "backend/api/handlers/intervention_waitdata_executor_r116_test.go", "backend/db/models/intervention_waitdata.go", "backend/db/models/intervention.go", "backend/db/mysql.go", "backend/api/router.go", "backend/engine/intervention_m3.go", "backend/engine/job_run_ownership.go", "backend/api/handlers/channels.go", "backend/api/handlers/demo.go", "backend/api/handlers/jobs.go", "backend/go.mod", "backend/go.sum", "scripts/r117_waitdata_fixture.py"}
    if not isinstance(value.get("sourceSha256"), dict) or not required.issubset(value["sourceSha256"]) or not isinstance(value.get("binarySha256"), dict) or not value["binarySha256"]:
        raise RuntimeError("ADMISSION_PINS_INCOMPLETE")
    for path, digest in value["sourceSha256"].items():
        target = safe_relative(path, "backend/" if path.startswith("backend/") else "scripts/r117_waitdata_fixture.py")
        if not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest) or sha(target) != digest:
            raise RuntimeError("SOURCE_CHANGED")
        blob = subprocess.run(["git", "show", "HEAD:" + path], cwd=ROOT, capture_output=True, timeout=15)
        if blob.returncode or blob.stdout.replace(b"\r\n", b"\n") != target.read_bytes().replace(b"\r\n", b"\n"):
            raise RuntimeError("SOURCE_NOT_COMMITTED")
    pinned_go = {path for path in value["sourceSha256"] if path.startswith("backend/") and path.endswith(".go")}
    actual_go = {path.relative_to(ROOT).as_posix() for path in (ROOT / "backend").rglob("*.go") if path.is_file()}
    if pinned_go != actual_go:
        raise RuntimeError("BACKEND_SOURCE_SET_CHANGED")
    for path, digest in value["binarySha256"].items():
        if not isinstance(digest, str) or not re.fullmatch(r"[0-9a-f]{64}", digest) or sha(safe_relative(path, ".cvf/runtime/r117_execution/bin/")) != digest:
            raise RuntimeError("BINARY_CHANGED")
    if value.get("sourceVerifiedOnlyLoopbackR116DSN") is not True or value.get("osNetworkConfinement") != "NOT_CLAIMED_SOURCE_BOUND_CONNECTOR":
        raise RuntimeError("CONNECTOR_NOT_REVIEWED")
    return value


def verify_resources(pin):
    volume = json.loads(docker("volume", "inspect", "--format", "{{json .}}", NAME).stdout)
    network = json.loads(docker("network", "inspect", "--format", "{{json .}}", pin["networkId"]).stdout)
    expected = {"ccmai.tranche": "CCMAI-RUNTIME-117", "ccmai.owner": pin["ownerNonce"]}
    if volume["Name"] != NAME or volume.get("Driver") != "local" or volume.get("Labels") != expected or volume.get("Options") not in (None, {}):
        raise RuntimeError("VOLUME_OWNERSHIP_CHANGED")
    if "volumeIdentity" in pin and {k: volume.get(k) for k in ("Name", "CreatedAt", "Mountpoint", "Driver", "Labels", "Options")} != pin["volumeIdentity"]:
        raise RuntimeError("VOLUME_REPLACED")
    options = network.get("Options")
    if (network["Id"] != pin["networkId"] or network["Name"] != NAME or network.get("Labels") != expected
            or network.get("Internal") is not False or network.get("Driver") != "bridge"
            or not isinstance(options, dict)
            or options.get("com.docker.network.bridge.gateway_mode_ipv4") != "nat"
            or options.get("com.docker.network.bridge.host_binding_ipv4") != "127.0.0.1"):
        raise RuntimeError("NETWORK_OWNERSHIP_CHANGED")
    return {"containers": network.get("Containers", {}), "volume": {k: volume.get(k) for k in ("Name", "CreatedAt", "Mountpoint", "Driver", "Labels", "Options")},
            "network": {k: network.get(k) for k in ("Id", "Name", "Created", "Driver", "Internal", "Labels", "Options")}}


def verify_network_only(pin):
    value = json.loads(docker("network", "inspect", "--format", "{{json .}}", pin["networkId"]).stdout)
    options = value.get("Options")
    if (value["Id"] != pin["networkId"] or value["Name"] != NAME or value.get("Internal") is not False
            or value.get("Driver") != "bridge"
            or value.get("Labels") != {"ccmai.tranche": "CCMAI-RUNTIME-117", "ccmai.owner": pin["ownerNonce"]}
            or not isinstance(options, dict)
            or options.get("com.docker.network.bridge.gateway_mode_ipv4") != "nat"
            or options.get("com.docker.network.bridge.host_binding_ipv4") != "127.0.0.1"):
        raise RuntimeError("PARTIAL_NETWORK_OWNERSHIP_CHANGED")
    return {k: value.get(k) for k in ("Id", "Name", "Created", "Driver", "Internal", "Labels", "Options", "Containers")}


def published_port(ports):
    """Accept exactly one complete localhost MySQL publication."""
    if not isinstance(ports, dict) or set(ports) != {"3306/tcp"}:
        raise RuntimeError("FIXTURE_PORT_INVALID")
    bindings = ports["3306/tcp"]
    if not isinstance(bindings, list) or len(bindings) != 1 or not isinstance(bindings[0], dict):
        raise RuntimeError("FIXTURE_PORT_INVALID")
    binding = bindings[0]
    if set(binding) != {"HostIp", "HostPort"} or binding["HostIp"] != "127.0.0.1":
        raise RuntimeError("FIXTURE_PORT_INVALID")
    value = binding["HostPort"]
    if (not isinstance(value, str) or len(value) > 5 or not re.fullmatch(r"[0-9]+", value)
            or not 1024 <= int(value) <= 65535):
        raise RuntimeError("FIXTURE_PORT_INVALID")
    return int(value)


def inspect_fixture(pin, check_protected=True, require_port=True):
    ident = pin["containerId"]
    if not re.fullmatch(r"[0-9a-f]{64}", ident):
        raise RuntimeError("INVALID_CONTAINER_ID")
    # Never expose Config.Env or unrestricted inspect JSON.
    fmt = '{"Id":{{json .Id}},"Image":{{json .Image}},"Name":{{json .Name}},"State":{{json .State.Status}},"Labels":{{json .Config.Labels}},"Mounts":{{json .Mounts}},"Networks":{{json .NetworkSettings.Networks}},"Ports":{{json .NetworkSettings.Ports}}}'
    value = json.loads(docker("inspect", "--format", fmt, ident).stdout)
    if value["Id"] != ident or value["Image"] != IMAGE or value["Name"] != "/" + NAME or value["Labels"].get("ccmai.tranche") != "CCMAI-RUNTIME-117" or value["Labels"].get("ccmai.owner") != pin["ownerNonce"]:
        raise RuntimeError("FIXTURE_IDENTITY_CHANGED")
    mounts = value["Mounts"]
    if len(mounts) != 1 or mounts[0].get("Type") != "volume" or mounts[0].get("Name") != NAME or mounts[0].get("Destination") != "/var/lib/mysql":
        raise RuntimeError("FIXTURE_MOUNT_CHANGED")
    networks = value["Networks"]
    if set(networks) != {NAME} or networks[NAME]["NetworkID"] != pin["networkId"]:
        raise RuntimeError("FIXTURE_NETWORK_CHANGED")
    users = docker("ps", "-aq", "--no-trunc", "--filter", "volume=" + NAME).stdout.decode().split()
    network_users = docker("ps", "-aq", "--no-trunc", "--filter", "network=" + NAME).stdout.decode().split()
    if users != [ident] or network_users != [ident]:
        raise RuntimeError("FIXTURE_NOT_EXCLUSIVE")
    if require_port:
        if value["State"] != "running" or published_port(value.get("Ports")) != pin.get("port"):
            raise RuntimeError("FIXTURE_PORT_CHANGED")
    resources = verify_resources(pin)
    if check_protected and (inventory() != pin["protectedBaseline"] or resource_inventory() != pin["protectedResources"]):
        raise RuntimeError("PROTECTED_INVENTORY_CHANGED")
    value["ownedResourceIdentity"] = resources
    return value


def setup():
    verify_source()
    before = inventory()
    protected_resources = resource_inventory()
    for args in [("container", "ls", "-aq", "--filter", "name=^/" + NAME + "$"),
                 ("volume", "ls", "-q", "--filter", "name=^" + NAME + "$"),
                 ("network", "ls", "-q", "--filter", "name=^" + NAME + "$")]:
        if docker(*args).stdout.strip():
            raise RuntimeError("FIXTURE_NAME_ALREADY_EXISTS")
    if docker("image", "inspect", "--format", "{{.Id}}", IMAGE).stdout.decode().strip() != IMAGE:
        raise RuntimeError("IMAGE_NOT_CACHED")
    nonce = str(uuid.uuid4())
    owner = {"at": utc(), "name": NAME, "ownerNonce": nonce, "protectedBaseline": before, "protectedResources": protected_resources}
    exclusive(RUN / "fixture-start.effect", {**owner, "maxStarts": 1})
    network = docker("network", "create", "--driver", "bridge",
                     "--opt", "com.docker.network.bridge.gateway_mode_ipv4=nat",
                     "--opt", "com.docker.network.bridge.host_binding_ipv4=127.0.0.1",
                     "--label", LABEL, "--label", "ccmai.owner=" + nonce, NAME).stdout.decode().strip()
    owner["networkId"] = network
    exclusive(RUN / "network-created.json", owner)
    docker("volume", "create", "--label", LABEL, "--label", "ccmai.owner=" + nonce, NAME)
    identities = verify_resources(owner)
    owner["volumeIdentity"] = identities["volume"]
    owner["networkIdentity"] = identities["network"]
    exclusive(RUN / "volume-created.json", owner)
    exclusive(RUN / "container-create.effect", owner)
    created = docker("create", "--pull=never", "--name", NAME, "--label", LABEL, "--label", "ccmai.owner=" + nonce,
                     "--network", NAME, "-p", "127.0.0.1::3306", "-v", NAME + ":/var/lib/mysql",
                     "-e", "MYSQL_ROOT_PASSWORD=" + DB_PASSWORD, "-e", "MYSQL_DATABASE=r116_fixture",
                     IMAGE, "--skip-name-resolve").stdout.decode().strip()
    owner["containerId"] = created
    exclusive(RUN / "container-created.json", owner)
    docker("start", created, timeout=30)
    fmt = '{{json .NetworkSettings.Ports}}'
    ports = json.loads(docker("inspect", "--format", fmt, created).stdout)
    exclusive(RUN / "port-observation.json", {"at": utc(), "containerId": created, "ports": ports})
    owner["port"] = published_port(ports)
    exclusive(RUN / "fixture.json", owner)
    inspect_fixture(owner)
    deadline = time.monotonic() + 60
    while time.monotonic() < deadline:
        ready = docker("exec", "-e", "MYSQL_PWD=" + DB_PASSWORD, created, "mysql", "-h", "127.0.0.1",
                       "-u", "root", "-N", "-B", "-e", "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='r116_fixture'",
                       checked=False)
        if ready.returncode == 0 and ready.stdout.strip() == b"r116_fixture":
            exclusive(EVIDENCE / "fixture_ready.json", {**owner, "imageId": IMAGE, "readyAt": utc(), "state": "running", "exclusive": True})
            print("R117_FIXTURE_READY")
            return
        time.sleep(1)
    raise RuntimeError("FIXTURE_NOT_READY_STOP_NO_RESTART")


def run_test(mode):
    source = verify_source()
    admission = committed(RUNTIME_ADMISSION)
    pin = load(RUN / "fixture.json")
    if admission.get("schemaVersion") != 1 or admission.get("trancheId") != "CCMAI-RUNTIME-117" or admission.get("fixtureSha256") != sha(RUN / "fixture.json") or admission.get("sourceAdmissionSha256") != sha(SOURCE_ADMISSION):
        raise RuntimeError("RUNTIME_ADMISSION_CHANGED")
    if inspect_fixture(pin)["State"] != "running":
        raise RuntimeError("FIXTURE_NOT_RUNNING")
    binary_relative = admission["reviewerTestBinary"] if mode == "reviewer" else admission["testBinary"]
    binary = safe_relative(binary_relative, ".cvf/runtime/r117_execution/bin/")
    if binary_relative not in source["binarySha256"]:
        raise RuntimeError("BINARY_NOT_ADMITTED")
    if mode not in admission["testModes"]:
        raise RuntimeError("TEST_MODE_NOT_ADMITTED")
    pattern = admission["testModes"][mode]
    allowed_patterns = {"native": "^TestR116WaitDataNative[A-Za-z0-9_]+$", "reviewer": "^TestR116WaitDataNative[A-Za-z0-9_]+$", "live": "^TestR116WaitDataRealAlibaba$"}
    if pattern != allowed_patterns[mode]:
        raise RuntimeError("TEST_SELECTION_INVALID")
    expected_binary = source["binarySha256"][binary_relative]
    if sha(binary) != expected_binary:
        raise RuntimeError("BINARY_CHANGED")
    exclusive(RUN / (mode + ".effect"), {"at": utc(), "mode": mode, "pattern": pattern, "binarySha256": expected_binary})
    env = minimal_env()
    env.update(CCMAI_R116_NATIVE="1", CCMAI_R116_MYSQL_DSN=f"root:{DB_PASSWORD}@tcp(127.0.0.1:{pin['port']})/r116_fixture?parseTime=true&charset=utf8mb4&timeout=5s&readTimeout=15s&writeTimeout=15s")
    key = ""
    if mode == "live":
        live = committed(LIVE_ADMISSION)
        expected_live = {"schemaVersion": 1, "trancheId": "CCMAI-RUNTIME-117", "grantId": "CCMAI-RUNTIME-117-OWNER-NEW1-20261017", "provider": "alibaba_maas", "model": "qwen3.7-flash", "endpointSha256": ENDPOINT_SHA, "maxCompletionTokens": 768, "maxRequestBytes": 6000, "enableThinking": False, "physicalCallCap": 1, "stopFirstValid": True, "stopAnyUncertainty": True, "runtimeAdmissionSha256": sha(RUNTIME_ADMISSION), "sourceAdmissionSha256": sha(SOURCE_ADMISSION), "testBinarySha256": expected_binary}
        if any(live.get(k) != v for k, v in expected_live.items()) or not live.get("configRef") or not live.get("promptVersion") or not live.get("promptSha256"):
            raise RuntimeError("LIVE_ADMISSION_INVALID")
        deadline = datetime.fromisoformat(live["expiresAtUTC"])
        if deadline.tzinfo is None or not datetime.now(timezone.utc) < deadline:
            raise RuntimeError("LIVE_ADMISSION_EXPIRED")
        secret = ROOT / "CVF_SESSION/LOCAL_PROVIDER_SECRETS.json"
        rel = secret.relative_to(ROOT).as_posix()
        if subprocess.run(["git", "check-ignore", "-q", rel], cwd=ROOT).returncode != 0 or subprocess.run(["git", "ls-files", "--error-unmatch", rel], cwd=ROOT, capture_output=True).returncode == 0:
            raise RuntimeError("SECRET_NOT_IGNORED_UNTRACKED")
        provider = json.loads(secret.read_text(encoding="utf-8-sig"))["providers"]["alibaba_maas"]
        key, endpoint = provider["api_key"], provider["base_url"].rstrip("/")
        if not isinstance(key, str) or not 16 <= len(key) <= 512 or key != key.strip() or hashlib.sha256(endpoint.encode()).hexdigest() != ENDPOINT_SHA:
            raise RuntimeError("SECRET_OR_ENDPOINT_INVALID")
        (RUN / "physical_calls").mkdir(exist_ok=False)
        (EVIDENCE / "live").mkdir(exist_ok=False)
        env.update(CCMAI_R116_LIVE="1", CCMAI_R116_PRESERVE="1", CCMAI_R116_OWNER_GRANT="CCMAI-RUNTIME-117-OWNER-NEW1-20261017",
                   CCMAI_R116_ADMISSION_SHA=sha(LIVE_ADMISSION), CCMAI_R116_ALIBABA_KEY=key, CCMAI_R116_ALIBABA_ENDPOINT=endpoint,
                   CCMAI_R116_EVIDENCE_DIR=str(EVIDENCE / "live"), CCMAI_R116_EFFECT_DIR=str(RUN / "physical_calls"))
    argv = [str(binary), "-test.run=" + pattern, "-test.v", "-test.count=1", "-test.timeout=180s"]
    started = utc()
    child = subprocess.Popen(argv, cwd=ROOT / "backend", env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    timed_out = False
    try:
        stdout, stderr = child.communicate(timeout=210)
    except subprocess.TimeoutExpired:
        timed_out = True
        child.kill()
        stdout, stderr = child.communicate(timeout=15)
    unsafe_files = []
    if key:
        directory_live = EVIDENCE / "live"
        for path in directory_live.rglob("*") if directory_live.exists() else []:
            metadata = path.lstat()
            if stat.S_ISDIR(metadata.st_mode) and not path.is_symlink():
                continue
            parent_safe = path.parent.resolve().is_relative_to(directory_live.resolve())
            invalid = path.is_symlink() or not stat.S_ISREG(metadata.st_mode) or metadata.st_nlink != 1 or not path.resolve().is_relative_to(directory_live.resolve())
            unsafe = invalid or metadata.st_size > 8 * 1024 * 1024 or secret_unsafe(path.read_bytes(), key)
            if unsafe:
                unsafe_files.append(path.relative_to(ROOT).as_posix())
                if parent_safe:
                    # Replace the directory entry, never write through a link
                    # or a possibly shared inode supplied by the child.
                    replacement = path.with_name(path.name + ".r117-redacted-" + uuid.uuid4().hex)
                    with replacement.open("xb") as stream:
                        stream.write(b"REDACTED_SECRET_GUARD\n")
                    os.replace(replacement, path)
    if key and (any(secret_unsafe(item, key) for item in (stdout, stderr)) or unsafe_files):
        stdout, stderr = b"REDACTED_SECRET_GUARD\n", b""
        failure = "SECRET_GUARD"
    else:
        failure = "TIMEOUT" if timed_out else ""
    directory = EVIDENCE / (mode + "_terminal")
    directory.mkdir(parents=True, exist_ok=False)
    (directory / "stdout.txt").write_bytes(stdout)
    (directory / "stderr.txt").write_bytes(stderr)
    exclusive(directory / "receipt.json", {"argv": argv, "cwd": "backend", "startedAt": started,
              "finishedAt": utc(), "exitCode": child.returncode, "pid": child.pid, "failure": failure,
              "sourceAdmissionSha256": sha(SOURCE_ADMISSION), "runtimeAdmissionSha256": sha(RUNTIME_ADMISSION),
              "binarySha256": expected_binary, "binaryAfterSha256": sha(binary), "stdoutSha256": sha(directory / "stdout.txt"), "stderrSha256": sha(directory / "stderr.txt"),
              "testSkipObserved": b"--- SKIP:" in stdout, "providerMode": mode == "live"})
    verify_source()
    if mode == "live":
        physical = sorted((RUN / "physical_calls").glob("*.effect"))
        artifacts = {p.relative_to(ROOT).as_posix(): sha(p) for p in (EVIDENCE / "live").rglob("*") if p.is_file()}
        exclusive(directory / "live_artifact_binding.json", {"physicalMarkerCount": len(physical), "physicalMarkers": {p.name: sha(p) for p in physical}, "artifacts": artifacts, "unsafeFilesRedacted": unsafe_files, "requiresIndependentProviderUsageAndDBCounterReview": True})
        if len(physical) != 1 or not artifacts:
            raise RuntimeError("LIVE_EVIDENCE_INCOMPLETE")
    after = inspect_fixture(pin)
    exclusive(directory / "post_inventory.json", after)
    print("R117_" + mode.upper() + "_TERMINAL exit=" + str(child.returncode) + " failure=" + failure)
    if child.returncode != 0 or failure or b"--- SKIP:" in stdout:
        raise RuntimeError("TEST_FAILED_OR_SKIPPED_STOP")


def secret_unsafe(raw, key):
    # Check raw bytes and repeatedly decoded JSON unicode escapes; never print
    # candidate text. This also handles JSON strings containing nested JSON.
    if len(raw) > 8 * 1024 * 1024:
        return True
    text = raw.decode("utf-8", errors="replace")
    for _ in range(16):
        if key in text:
            return True
        decoded = re.sub(r"\\u([0-9a-fA-F]{4})", lambda match: chr(int(match.group(1), 16)), text)
        decoded = decoded.replace("\\\\", "\\")
        if decoded == text:
            return False
        text = decoded
    return True


def cleanup():
    # A persisted exact owned container ID is required; never delete by prefix.
    if (RUN / "fixture.json").exists():
        pin = load(RUN / "fixture.json")
    elif (RUN / "container-created.json").exists():
        pin = load(RUN / "container-created.json")
    else:
        owner = load(RUN / "network-created.json") if (RUN / "network-created.json").exists() else None
        if (RUN / "fixture-start.effect").exists() and owner is None:
            raise RuntimeError("PARTIAL_NETWORK_CREATION_UNRESOLVED_MANUAL_HOLD")
        if owner and not (RUN / "volume-created.json").exists():
            raise RuntimeError("PARTIAL_VOLUME_CREATION_UNRESOLVED_MANUAL_HOLD")
        if (RUN / "container-create.effect").exists():
            raise RuntimeError("PARTIAL_CONTAINER_CREATION_UNRESOLVED_MANUAL_HOLD")
        resources = verify_resources(load(RUN / "volume-created.json")) if (RUN / "volume-created.json").exists() else verify_network_only(owner) if owner else None
        exclusive(EVIDENCE / "cleanup_receipt.json", {"at": utc(), "containerCreated": False,
                  "networkOwnedPreserved": owner, "volumeCreated": (RUN / "volume-created.json").exists(),
                  "protectedBaselineEquals": inventory() == owner["protectedBaseline"] if owner else None,
                  "protectedResourcesEqual": resource_inventory() == owner["protectedResources"] if owner else None,
                  "resourceIdentity": resources, "fixtureNotStarted": True, "partialSetup": True})
        receipt = load(EVIDENCE / "cleanup_receipt.json")
        if owner and (receipt["protectedBaselineEquals"] is not True or receipt["protectedResourcesEqual"] is not True or (resources and (resources.get("containers") or resources.get("Containers")))):
            raise RuntimeError("PARTIAL_CLEANUP_INVENTORY_HOLD")
        print("R117_PARTIAL_SETUP_NO_CONTAINER")
        return
    before = inspect_fixture(pin, check_protected=False, require_port=False)
    exclusive(RUN / "cleanup.effect", {"at": utc(), "containerId": pin["containerId"]})
    stopped = docker("stop", "-t", "10", pin["containerId"], timeout=30)
    after = inspect_fixture(pin, check_protected=False, require_port=False)
    if after["State"] != "exited":
        raise RuntimeError("CLEANUP_NOT_EXITED")
    exclusive(EVIDENCE / "cleanup_receipt.json", {"at": utc(), "before": before, "after": after,
              "stopExitCode": stopped.returncode, "protectedBaselineEquals": inventory() == pin["protectedBaseline"], "protectedResourcesEqual": resource_inventory() == pin["protectedResources"],
              "containerPreservedStopped": True, "volumeAndNetworkPreservedOwned": True,
              "originalTerminalReceipts": sorted(p.name for p in EVIDENCE.glob("*_terminal"))})
    receipt = load(EVIDENCE / "cleanup_receipt.json")
    if receipt["protectedBaselineEquals"] is not True or receipt["protectedResourcesEqual"] is not True:
        raise RuntimeError("CLEANUP_PROTECTED_INVENTORY_HOLD")
    print("R117_CLEANUP_STOPPED_PRESERVED")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["setup", "native", "reviewer", "live", "cleanup"])
    action = parser.parse_args().action
    {"setup": setup, "cleanup": cleanup}.get(action, lambda: run_test(action))()


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        # Class only: underlying exceptions may contain child env/SQL/provider text.
        code = str(error) if isinstance(error, RuntimeError) and re.fullmatch(r"[A-Z_]+", str(error)) else type(error).__name__
        RUN.mkdir(parents=True, exist_ok=True)
        hold = RUN / ("hold-" + str(time.time_ns()) + ".json")
        exclusive(hold, {"at": utc(), "code": code, "automaticRetry": False})
        print("R117_HOLD " + code)
        sys.exit(1)
