"""R105 reviewed-source candidate with fail-closed formal runtime admission.

Importing has no external effects. The current R105 draft has no effective
seed/order/record/review receipt and cannot execute the supervised campaign.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import queue
import re
import secrets
import shutil
import socket
import subprocess
import threading
import time
from pathlib import Path
from urllib.parse import urlsplit

TARGET = "200df3e4a7bc2a326425922a5230508a4fd9b0fe"
BINARY_SHA256 = "4c7575e180dd198d995bfe09bdc45061740a987be8ec0765f02b203e1c251f31"
CONTAINER = "e1555e03d69360e1cd7729afd9e52856443f5172d76454653beec981a7d6477a"
IMAGE = "sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b"
NETWORK = "35c3959ac41517050defad05cbad7a08717b24c8879683e25baf3e2702625e98"
NAME = "ccmai-r101-review-final-20261015"
LABEL = {"ccmai.tranche": "CCMAI-RUNTIME-101"}
DB = "r101_fixture"
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
UUID = r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
READY = re.compile(
    r"^R101_BROWSER_READY (http://127\.0\.0\.1:[1-9][0-9]{0,4}) "
    rf"tenant=({UUID}) case=({UUID}) reader=({UUID})$"
)
OWNER = re.compile(rf"^R101_BROWSER_OWNER ({UUID}@example\.invalid)$")
EFFECTS = ("fixture_start", "backend_launch", "vite_launch")
ORDER = "docs/work_orders/CCMAI_RUNTIME_105_R101_UI_EVIDENCE.md"
SEED = "CVF_SESSION/authority/CCMAI-RUNTIME-105.json"
RECORD = "CVF_SESSION/tranches/CCMAI-RUNTIME-105.json"
STATE = "CVF_SESSION/ACTIVE_SESSION_STATE.json"
REVIEW = "docs/reviews/probes/r105_root/source_review_receipt.json"
FRONTEND_MANIFEST = "docs/reviews/probes/r105_dispatch_preparation/frontend_input_manifest.json"
HISTORICAL_MANIFEST = "docs/reviews/probes/r101_root/formatted_source_manifest.json"
RUNTIME = ".cvf/runtime/r105_claude"
TEMPLATE = "docs/reviews/probes/r105_root/r105_vite_config.mjs.template"
RUNNER = "docs/reviews/probes/r105_root/r105_ui_runner.py"
PASSWORD = "r101LocalOnly42"  # pinned synthetic fixture literal, never real credentials
CONTAINER_FORMAT = ('{"Id":{{json .Id}},"Name":{{json .Name}},"Image":{{json .Image}},'
                    '"Config":{"Labels":{{json .Config.Labels}}},'
                    '"State":{"Status":{{json .State.Status}}},'
                    '"Mounts":{{json .Mounts}},'
                    '"HostConfig":{"PortBindings":{{json .HostConfig.PortBindings}}},'
                    '"NetworkSettings":{"Networks":{{json .NetworkSettings.Networks}},"Ports":{{json .NetworkSettings.Ports}}}}')
NETWORK_FORMAT = ('{"Id":{{json .Id}},"Name":{{json .Name}},'
                  '"Labels":{{json .Labels}},"Containers":{{json .Containers}}}')


class Hold(ValueError):
    """A prerequisite is absent or inconsistent; no effect is admitted."""


def require(condition: bool, reason: str) -> None:
    if not condition:
        raise Hold(reason)


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def canonical_git_text(data: bytes) -> bytes:
    """Match Git's LF text blob for declared JSON without relaxing content."""
    require(b"\r" not in data.replace(b"\r\n", b""), "bare CR in governed text")
    data.decode("utf-8", errors="strict")
    return data.replace(b"\r\n", b"\n")


def inside(root: Path, relative: str) -> Path:
    require(isinstance(relative, str) and relative and "\\" not in relative, "unsafe relative path")
    candidate = Path(relative)
    require(not candidate.is_absolute() and all(p not in (".", "..") for p in candidate.parts), "unsafe relative path")
    result = (root / candidate).resolve()
    require(result.is_relative_to(root.resolve()) and result.is_file(), "missing or escaped input")
    return result


def validate_frontend(root: Path, manifest: dict) -> int:
    files = manifest.get("files")
    require(isinstance(files, list) and len(files) == 137, "frontend manifest count drift")
    seen: set[str] = set()
    for entry in files:
        path, digest, size = entry["path"], entry["sha256"], entry["bytes"]
        require(path.startswith("frontend/") and "/.env" not in path and path not in seen, "frontend path invalid")
        require(bool(HEX64.fullmatch(digest)) and isinstance(size, int) and size >= 0, "frontend pin invalid")
        seen.add(path)
        actual = inside(root, path)
        require(actual.stat().st_size == size and sha256(actual) == digest, f"frontend drift: {path}")
    return len(seen)


def validate_historical_rows(rows: list[dict], blob_reader) -> int:
    require(len(rows) == 17, "historical manifest count drift")
    seen: set[str] = set()
    for row in rows:
        path, captured = row["path"], row["sha256"]
        require(path not in seen and not path.startswith("/") and ".." not in Path(path).parts, "historical path invalid")
        require(bool(HEX64.fullmatch(captured)), "historical digest invalid")
        seen.add(path)
        blob = blob_reader(TARGET, path)
        require(isinstance(blob, bytes), "historical blob unavailable")
        raw = hashlib.sha256(blob).hexdigest()
        crlf = hashlib.sha256(blob.replace(b"\r\n", b"\n").replace(b"\n", b"\r\n")).hexdigest()
        require(captured in (raw, crlf), f"historical source mismatch: {path}")
    return len(seen)


def validate_container(container: dict, network: dict, volume_users: list[str]) -> None:
    require(container.get("Id") == CONTAINER, "fixture ID mismatch")
    require(container.get("Name") == "/" + NAME, "fixture name mismatch")
    require(container.get("Image") == IMAGE, "fixture image mismatch")
    labels = container.get("Config", {}).get("Labels", {})
    require(all(labels.get(k) == v for k, v in LABEL.items()), "fixture labels mismatch")
    require(container.get("State", {}).get("Status") == "exited", "fixture must be stopped before start")
    mounts = container.get("Mounts")
    require(isinstance(mounts, list) and len(mounts) == 1, "fixture mounts mismatch")
    mount = mounts[0]
    require(mount.get("Type") == "volume" and mount.get("Name") == NAME and mount.get("Destination") == "/var/lib/mysql" and mount.get("RW") is True, "fixture volume mismatch")
    bindings = container.get("HostConfig", {}).get("PortBindings", {})
    require(bindings == {"3306/tcp": [{"HostIp": "127.0.0.1", "HostPort": ""}]}, "DB binding is not dynamic loopback")
    nets = container.get("NetworkSettings", {}).get("Networks", {})
    require(list(nets) == [NAME] and nets[NAME].get("NetworkID") == NETWORK, "fixture network mismatch")
    require(network.get("Id") == NETWORK and network.get("Name") == NAME and network.get("Containers") == {}, "network is not exclusive")
    require(volume_users == [CONTAINER], "fixture volume is shared")


def parse_db_port(container: dict) -> int:
    ports = container.get("NetworkSettings", {}).get("Ports", {}).get("3306/tcp")
    require(isinstance(ports, list) and len(ports) == 1, "DB port count mismatch")
    item = ports[0]
    require(item.get("HostIp") == "127.0.0.1", "DB port is not loopback")
    raw = item.get("HostPort")
    require(isinstance(raw, str) and raw.isdecimal(), "DB port missing")
    port = int(raw)
    require(1 <= port <= 65535, "DB port invalid")
    return port


def parse_ready(lines: list[str]) -> dict:
    ready = [READY.fullmatch(s.strip()) for s in lines]
    owner = [OWNER.fullmatch(s.strip()) for s in lines]
    ready = [x for x in ready if x]
    owner = [x for x in owner if x]
    require(len(ready) == 1 and len(owner) == 1, "fresh backend readiness absent or ambiguous")
    origin, tenant, case, reader = ready[0].groups()
    parsed = urlsplit(origin)
    require(parsed.hostname == "127.0.0.1" and parsed.port and parsed.path == "", "API origin invalid")
    return {"apiOrigin": origin, "tenant": tenant, "case": case, "reader": reader, "owner": owner[0].group(1)}


def protected_inventory(rows: list[dict]) -> list[dict]:
    """Safe, stable fields only. Never inspect or serialize Config.Env."""
    safe = []
    for row in rows:
        mounts = row.get("Mounts", [])
        safe.append({
            "id": row["Id"], "name": row["Name"], "image": row["Image"],
            "status": row["State"]["Status"],
            "mounts": sorted(({"type": m["Type"], "name": m.get("Name"), "destination": m["Destination"]} for m in mounts), key=lambda x: x["destination"]),
        })
    return sorted(safe, key=lambda x: x["id"])


def reserve(ledger: Path, effect: str) -> dict:
    """Append-only reservation; existence burns a slot even after a crash."""
    require(effect in EFFECTS, "unknown effect")
    require(ledger.is_dir(), "ledger directory missing")
    path = ledger / (effect + ".json")
    payload = {"effect": effect, "state": "RESERVED", "replayAllowed": False}
    try:
        with path.open("x", encoding="utf-8") as stream:
            json.dump(payload, stream, sort_keys=True)
            stream.flush()
            os.fsync(stream.fileno())
    except FileExistsError as exc:
        raise Hold(f"effect slot already consumed: {effect}") from exc
    return payload


def read_json(root: Path, relative: str) -> dict:
    return json.loads(inside(root, relative).read_text(encoding="utf-8"))


def git_bytes(root: Path, *args: str) -> bytes:
    completed = subprocess.run(["git", *args], cwd=root, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=15, check=False)
    require(completed.returncode == 0, "Git source-history check failed")
    return completed.stdout


def active_status_line(order: str) -> bool:
    """One unfenced, exact status line; inline/backtick and duplicates fail."""
    statuses = []
    fence = None
    for line in order.splitlines():
        stripped = line.strip()
        marker = re.match(r"^(`{3,}|~{3,})", stripped)
        if marker:
            if fence is None:
                fence = marker.group(1)[0]
            elif marker.group(1)[0] == fence:
                fence = None
            continue
        if fence is None and "Status:" in line:
            statuses.append(line)
    return statuses == ["Status: ACTIVE"]


def check_admission(root: Path, *, for_start: bool = True, git=git_bytes) -> dict:
    """Formal schema is prospective. A DRAFT candidate cannot satisfy it."""
    seed_bytes = inside(root, SEED).read_bytes()
    seed = json.loads(seed_bytes)
    record = read_json(root, RECORD)
    state = read_json(root, STATE)
    order = inside(root, ORDER).read_text(encoding="utf-8")
    review_bytes = inside(root, REVIEW).read_bytes()
    review = json.loads(review_bytes)
    pins = seed.get("runtimePins", {})
    require(seed.get("authorityKind") == "DISPATCHER_OWNED_HISTORICAL_R101_UI_EVIDENCE" and seed.get("riskCeiling") == "R2", "formal R2 seed missing")
    require(seed.get("trancheId") == "CCMAI-RUNTIME-105", "seed tranche mismatch")
    require(record.get("riskCeiling") == "R2" and record.get("trancheId") == "CCMAI-RUNTIME-105", "record tranche mismatch")
    require(record.get("roles", {}).get("implementationWorker") != record.get("roles", {}).get("reviewer"), "R2 reviewer is worker")
    require(record.get("roles") == seed.get("roles"), "seed/record roles drift")
    require(record.get("allowedPaths") == seed.get("allowedPaths"), "seed/record paths drift")
    require(record.get("prohibitedEffects") == seed.get("prohibitedEffects"), "seed/record effects drift")
    budget = seed.get("executionBudget", {})
    require(all(budget.get(k) == 1 for k in ("existingFixtureStarts", "historicalBackendLaunches", "viteLaunches")), "runtime effect budget must be exactly one each")
    require(budget.get("runtimeRetries") == 0 and budget.get("wholeCampaignSeconds") == 2100 and budget.get("backendSeconds") == 1800, "runtime duration/retry budget mismatch")
    require(all(budget.get(k) == 0 for k in ("providerCalls", "mockCalls", "goCompiles", "testBuildCommands", "newFixtures")), "prohibited effect budget mismatch")
    paths_allowed = seed.get("allowedPaths", [])
    require(all(p in paths_allowed for p in ("docs/reviews/probes/r105_root/**", "docs/reviews/probes/r105_claude/**", ".cvf/runtime/r105_claude/**")), "formal path families incomplete")
    require(record.get("workOrder") == ORDER and record.get("workOrderStatusGrammar") == "cvf.workOrderStatusGrammar@1.0.0", "record work-order binding mismatch")
    if for_start:
        require(active_status_line(order) and record.get("workOrderStatus") == "ACTIVE", "formal work order inactive")
    base = record.get("baseCommit", "")
    require(bool(HEX40.fullmatch(base)), "invalid base commit")
    require(git(root, "merge-base", "--is-ancestor", base, "HEAD") == b"", "base not ancestor")
    first = git(root, "log", "--diff-filter=A", "--format=%H", "--reverse", "--", SEED).decode().splitlines()
    require(len(first) >= 1 and bool(HEX40.fullmatch(first[0])), "seed first commit absent")
    require(canonical_git_text(git(root, "show", f"{first[0]}:{SEED}")) == canonical_git_text(seed_bytes), "seed edited after first commit")
    require(canonical_git_text(git(root, "show", f"{base}:{SEED}")) == canonical_git_text(seed_bytes), "base does not contain immutable seed")
    require(canonical_git_text(git(root, "show", f"{base}:{REVIEW}")) == canonical_git_text(review_bytes), "source review not committed at base")
    require(review.get("disposition") == "SOURCE_REVIEW_PASS_RUNTIME_NOT_RUN", "independent source review missing")
    require(review.get("reviewer") == seed.get("roles", {}).get("reviewer"), "source reviewer mismatch")
    require(review.get("reviewer") != seed.get("roles", {}).get("implementationWorker"), "self-review")
    paths = (RUNNER, TEMPLATE, FRONTEND_MANIFEST, HISTORICAL_MANIFEST, REVIEW)
    require(set(pins) == set(paths), "runtime pins must name exact five paths")
    for path in paths:
        digest = pins[path]
        require(isinstance(digest, str) and bool(HEX64.fullmatch(digest)) and digest == sha256(inside(root, path)), f"admission pin mismatch: {path}")
    require(review.get("sourceCandidateAuthor") == "/root/r105_runner_candidate", "candidate author mismatch")
    require(review.get("runtimePins") == {p: pins[p] for p in paths if p != REVIEW}, "reviewed source hash mismatch")
    require(seed.get("historicalTargetCommit") == TARGET and seed.get("binarySha256") == BINARY_SHA256, "historical target mismatch")
    if for_start:
        require(sha256(inside(root, ".cvf/runtime/r101_worker/r101_handlers.test.exe")) == BINARY_SHA256, "binary bytes mismatch")
    require(seed.get("nodeExe") == "C:/Program Files/nodejs/node.exe" and seed.get("nodeSha256") == "3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237", "Node authority mismatch")
    require(sha256(Path(seed["nodeExe"])) == seed["nodeSha256"], "Node bytes mismatch")
    docker_path = Path(seed.get("dockerExe", ""))
    require(docker_path.is_absolute() and docker_path.is_file() and sha256(docker_path) == seed.get("dockerSha256"), "Docker executable pin mismatch")
    require(seed.get("dockerHost") == "npipe:////./pipe/dockerDesktopLinuxEngine", "Docker host must be exact local engine pipe")
    if for_start:
        require(state.get("activeTranche") == "CCMAI-RUNTIME-105" and state.get("activePhase") == "BUILD" and state.get("currentMode") == "BUILD", "active continuity mismatch")
        require(record.get("phase") == "BUILD" and record.get("status") == "BUILD", "record not BUILD")
    return {"seedSha256": hashlib.sha256(canonical_git_text(seed_bytes)).hexdigest(), "seedHashRepresentation": "UTF8_LF", "baseCommit": base, "pins": pins, "dockerExe": str(docker_path), "dockerHost": seed["dockerHost"], "nodeExe": seed["nodeExe"]}


def render_config(template: str, frontend: Path, api_origin: str, nonce: str) -> str:
    parsed = urlsplit(api_origin)
    require(parsed.scheme == "http" and parsed.hostname == "127.0.0.1" and parsed.port and not parsed.path and not parsed.query, "API origin invalid")
    require(bool(re.fullmatch(r"[0-9a-f]{32}", nonce)), "health nonce invalid")
    require(all(template.count("'" + marker + "'") == 1 for marker in ("__R105_FRONTEND_ROOT__", "__R105_API_ORIGIN__", "__R105_NONCE__")), "config markers invalid")
    return (template.replace("'__R105_FRONTEND_ROOT__'", json.dumps(str(frontend.resolve()).replace("\\", "/")))
            .replace("'__R105_API_ORIGIN__'", json.dumps(api_origin))
            .replace("'__R105_NONCE__'", json.dumps(nonce)))


def minimal_env() -> dict[str, str]:
    allowed = ("SystemRoot", "WINDIR", "TEMP", "TMP")
    return {key: os.environ[key] for key in allowed if key in os.environ}


def port_free(port: int) -> bool:
    with socket.socket() as sock:
        try:
            sock.bind(("127.0.0.1", port))
        except OSError:
            return False
    return True


def write_once(path: Path, payload: dict) -> None:
    with path.open("x", encoding="utf-8") as stream:
        json.dump(payload, stream, sort_keys=True)
        stream.flush()
        os.fsync(stream.fileno())


class ShellAdapter:
    """Bounded external operations, only constructed after formal admission."""

    def __init__(self, root: Path, admission: dict, output: Path):
        self.root, self.output = root, output
        self.docker, self.node = admission["dockerExe"], admission["nodeExe"]
        require(admission.get("dockerHost") == "npipe:////./pipe/dockerDesktopLinuxEngine", "remote Docker host rejected")
        config = output / "docker-cli-config"
        self.docker_config = config
        self.docker_config_ready = False
        self.docker_prefix = [self.docker, "--host", admission["dockerHost"], "--config", str(config)]
        self.call_number = 0

    def command(self, args: list[str], timeout: int = 20, *, record_output: bool = True) -> str:
        if not self.docker_config_ready:
            self.docker_config.mkdir(exist_ok=False)
            self.docker_config_ready = True
        self.call_number += 1
        redact = lambda s: s.replace(PASSWORD, "[SYNTHETIC_DSN_REDACTED]")
        def printable(value):
            if isinstance(value, bytes):
                return value.decode("utf-8", errors="replace")
            return value or ""
        try:
            result = subprocess.run(args, cwd=self.root, env=minimal_env(), capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout, check=False)
            out, err, exit_code, failure = result.stdout, result.stderr, result.returncode, None
        except (subprocess.TimeoutExpired, OSError) as exc:
            out, err, exit_code, failure = printable(getattr(exc, "stdout", None)), printable(getattr(exc, "stderr", None)), None, type(exc).__name__
        safe_out, safe_err = redact(out), redact(err)
        write_once(self.output / f"command_{self.call_number:03}.json", {
            "argv": [redact(a) for a in args], "timeoutSeconds": timeout,
            "exitCode": exit_code, "failureType": failure,
            "stdout": safe_out[:65536] if record_output else "[SAFE_FIELDS_EXTRACTED_IN_MEMORY]",
            "stdoutSha256": hashlib.sha256(safe_out.encode()).hexdigest(),
            "stdoutBytes": len(safe_out.encode()), "stdoutTruncated": len(safe_out) > 65536,
            "stderr": safe_err[:65536], "stderrBytes": len(safe_err.encode()), "stderrTruncated": len(safe_err) > 65536,
        })
        require(failure is None, f"bounded command {failure}")
        require(result.returncode == 0, "bounded command failed")
        return result.stdout

    def docker_json(self, kind: str, identity: str) -> dict:
        # Docker evaluates the template and emits only named safe fields.
        # Config.Env is never returned to this process.
        format_string = CONTAINER_FORMAT if kind == "container" else NETWORK_FORMAT
        require(kind in ("container", "network"), "unsupported inspect kind")
        lines = self.command([*self.docker_prefix, kind, "inspect", "--format", format_string, identity]).splitlines()
        require(len(lines) == 1, "Docker inspect count mismatch")
        return json.loads(lines[0])

    def fixture(self) -> tuple[dict, dict, list[str]]:
        container = self.docker_json("container", CONTAINER)
        network = self.docker_json("network", NETWORK)
        users = self.command([*self.docker_prefix, "ps", "-aq", "--no-trunc", "--filter", f"volume={NAME}"]).splitlines()
        return container, network, users

    def protected(self) -> list[dict]:
        ids = self.command([*self.docker_prefix, "ps", "-aq", "--no-trunc"]).splitlines()
        ids = sorted(x for x in ids if x != CONTAINER)
        if not ids:
            return []
        rows = [json.loads(line) for line in self.command([*self.docker_prefix, "container", "inspect", "--format", CONTAINER_FORMAT, *ids]).splitlines()]
        require(len(rows) == len(ids), "protected inspect count mismatch")
        return protected_inventory(rows)

    def start_fixture(self) -> None:
        require(self.command([*self.docker_prefix, "start", CONTAINER]).strip() == CONTAINER, "fixture start identity mismatch")

    def db_ready(self, deadline: float, clock=time.monotonic, pause=time.sleep) -> int:
        while clock() < deadline:
            c = self.docker_json("container", CONTAINER)
            require(c.get("Id") == CONTAINER and c.get("State", {}).get("Status") == "running", "fixture did not remain owned/running")
            try:
                port = parse_db_port(c)
                out = self.command([*self.docker_prefix, "exec", "-e", "MYSQL_PWD=" + PASSWORD, CONTAINER, "mysql", "--protocol=TCP", "-h127.0.0.1", "-uroot", "--batch", "--skip-column-names", "-e", "SELECT 1; SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='r101_fixture'"], timeout=10)
                require(out.strip().splitlines() == ["1", DB], "fixture schema/SELECT1 mismatch")
                return port
            except (Hold, subprocess.TimeoutExpired):
                pause(min(2, max(0, deadline - clock())))
        raise Hold("fixture DB not ready within 90 seconds")

    def launch(self, args: list[str], env: dict[str, str]):
        flags = getattr(subprocess, "CREATE_NO_WINDOW", 0)
        self.call_number += 1
        path = self.output / f"launch_{self.call_number:03}"
        safe_args = [a.replace(PASSWORD, "[SYNTHETIC_DSN_REDACTED]") for a in args]
        try:
            executable_digest = sha256(Path(args[0]))
        except OSError:
            executable_digest = None
        write_once(path.with_suffix(".intent.json"), {"argv": safe_args, "executableSha256": executable_digest, "environmentNames": sorted(env), "attemptedAtUnix": time.time()})
        try:
            process = subprocess.Popen(args, cwd=self.root, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding="utf-8", errors="replace", bufsize=1, creationflags=flags)
        except OSError as exc:
            write_once(path.with_suffix(".result.json"), {"pid": None, "failureType": type(exc).__name__, "exitCode": None})
            raise Hold("process launch failed after reservation") from exc
        try:
            write_once(path.with_suffix(".result.json"), {"pid": process.pid, "failureType": None, "exitCode": None})
        except Exception as receipt_exc:
            # The caller has not received this Popen handle yet. Stop this exact
            # child here so a failed receipt cannot orphan a live process.
            try:
                stop_owned(process)
            except Exception as cleanup_exc:
                raise Hold(f"launch receipt failed for owned PID {process.pid}; direct-handle stop failed: {type(cleanup_exc).__name__}") from receipt_exc
            raise Hold(f"launch receipt failed for owned PID {process.pid}; direct-handle stop completed") from receipt_exc
        return process

    def stop_fixture(self) -> None:
        c = self.docker_json("container", CONTAINER)
        require(c.get("Id") == CONTAINER and c.get("Name") == "/" + NAME and c.get("Image") == IMAGE, "fixture identity changed before stop")
        if c.get("State", {}).get("Status") == "running":
            self.command([*self.docker_prefix, "stop", "--time", "10", CONTAINER], timeout=20)

    def vite_ready(self, process, nonce: str, deadline: float, clock=time.monotonic, pause=time.sleep) -> None:
        while clock() < deadline:
            require(process.poll() is None, "owned Vite process exited before readiness")
            try:
                with socket.create_connection(("127.0.0.1", 3011), timeout=2) as connection:
                    connection.sendall(b"GET /__r105_health HTTP/1.1\r\nHost: 127.0.0.1:3011\r\nConnection: close\r\n\r\n")
                    response = b""
                    while len(response) < 4096:
                        chunk = connection.recv(4096 - len(response))
                        if not chunk:
                            break
                        response += chunk
                require(response.startswith(b"HTTP/1.1 200 ") and response.split(b"\r\n\r\n", 1)[-1] == nonce.encode(), "Vite owned health marker mismatch")
                require(process.poll() is None, "owned Vite process exited after readiness")
                return
            except (OSError, Hold):
                pause(min(0.5, max(0, deadline - clock())))
        raise Hold("Vite HTTP readiness missing within 30 seconds")


def collect_lines(stream_source, output: Path, events: queue.Queue) -> threading.Thread:
    def reader():
        digest = hashlib.sha256()
        total = retained = 0
        limit = 1024 * 1024
        problem = None
        try:
            with output.open("xb") as stream:
                for line in stream_source:
                    safe = line.replace(PASSWORD, "[SYNTHETIC_DSN_REDACTED]")
                    encoded = safe.encode("utf-8")
                    digest.update(encoded)
                    total += len(encoded)
                    prefix = encoded[:max(0, limit - retained)]
                    keep = prefix.decode("utf-8", errors="ignore").encode("utf-8")
                    if keep:
                        stream.write(keep)
                        stream.flush()
                        os.fsync(stream.fileno())
                        retained += len(keep)
                    try:
                        events.put_nowait(safe.rstrip("\r\n"))
                    except queue.Full:
                        pass
        except Exception as exc:
            problem = type(exc).__name__
            thread.error = problem
        try:
            write_once(output.with_suffix(output.suffix + ".meta.json"), {"fullRedactedSha256": digest.hexdigest(), "fullBytes": total, "retainedBytes": retained, "retainedSha256": sha256(output) if output.is_file() else None, "truncated": total > retained, "readerError": problem})
        except Exception as exc:
            thread.error = type(exc).__name__
    thread = threading.Thread(target=reader, daemon=True)
    thread.error = None
    thread.start()
    return thread


def stop_owned(process) -> None:
    """Only the Popen handle made by this process; no PID lookup or tree kill."""
    if process is None:
        return
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=8)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait(timeout=5)


def await_ready(process, events: queue.Queue, deadline: float, clock=time.monotonic) -> dict:
    seen = []
    while clock() < deadline:
        require(process.poll() is None, "backend exited before readiness")
        try:
            seen.append(events.get(timeout=min(0.25, max(0.01, deadline - clock()))))
        except queue.Empty:
            continue
        if any(READY.fullmatch(s) for s in seen) and any(OWNER.fullmatch(s) for s in seen):
            return parse_ready(seen)
    raise Hold("backend readiness missing within 60 seconds")


def verify_historical(root: Path) -> int:
    rows = read_json(root, HISTORICAL_MANIFEST)["files"]
    return validate_historical_rows(rows, lambda commit, path: git_bytes(root, "show", f"{commit}:{path}"))


def run_campaign(root: Path, admission: dict, adapter, *, clock=time.monotonic, pause=time.sleep) -> None:
    """One supervised campaign. Adapter injection keeps tests offline."""
    require(Path.cwd().resolve() == root.resolve(), "project CWD mismatch")
    output = root / RUNTIME
    require(output.parent.resolve().is_relative_to(root.resolve()) and not output.parent.is_symlink(), "runtime parent escapes project")
    require(not output.exists(), "R105 runtime output already exists; no replay")
    output.mkdir(parents=True, exist_ok=False)
    write_once(output / "admission.json", {"seedSha256": admission["seedSha256"], "baseCommit": admission["baseCommit"], "startedAtUnix": time.time(), "runnerSha256": admission["pins"][RUNNER]})
    backend = vite = None
    readers = []
    started = False
    baseline = adapter.protected()
    write_once(output / "protected_before.json", {"rows": baseline})
    begun = clock()
    try:
        c, n, users = adapter.fixture()
        validate_container(c, n, users)
        reserve(output, "fixture_start")
        started = True  # reserve before the call; conservative cleanup after uncertain result
        adapter.start_fixture()
        port = adapter.db_ready(min(begun + 90, begun + 2100), clock, pause)
        require(port_free(3011), "Vite port 3011 occupied")
        reserve(output, "backend_launch")
        env = minimal_env()
        env["R101_BROWSER_FIXTURE"] = "1"
        env["R101_MYSQL_DSN"] = f"root:{PASSWORD}@tcp(127.0.0.1:{port})/{DB}?parseTime=true&charset=utf8mb4"
        binary = str(root / ".cvf/runtime/r101_worker/r101_handlers.test.exe")
        backend_begun = clock()
        backend = adapter.launch([binary, "-test.run", "^TestR101BrowserFixture$", "-test.v", "-test.timeout=35m"], env)
        write_once(output / "backend_process.json", {"pid": backend.pid, "startedAtUnix": time.time(), "executable": binary, "executableSha256": BINARY_SHA256, "argv": [binary, "-test.run", "^TestR101BrowserFixture$", "-test.v", "-test.timeout=35m"]})
        events = queue.Queue(maxsize=100)
        readers.append(collect_lines(backend.stdout, output / "backend.stdout.log", events))
        readers.append(collect_lines(backend.stderr, output / "backend.stderr.log", queue.Queue(maxsize=1)))
        ready = await_ready(backend, events, min(clock() + 60, begun + 2100), clock)
        write_once(output / "ready.json", ready)
        health_nonce = secrets.token_hex(16)
        rendered = render_config((root / TEMPLATE).read_text(encoding="utf-8"), root / "frontend", ready["apiOrigin"], health_nonce)
        config = output / "vite.config.mjs"
        with config.open("x", encoding="utf-8") as stream:
            stream.write(rendered)
            stream.flush()
            os.fsync(stream.fileno())
        write_once(output / "vite_config_receipt.json", {"sha256": sha256(config), "apiOrigin": ready["apiOrigin"]})
        require(port_free(3011), "Vite port 3011 occupied at launch")
        reserve(output, "vite_launch")
        vite_cli = root / "frontend/node_modules/vite/bin/vite.js"
        require(vite_cli.is_file(), "cached Vite CLI absent")
        vite = adapter.launch([admission["nodeExe"], str(vite_cli), "--config", str(config), "--host", "127.0.0.1", "--port", "3011", "--strictPort"], minimal_env())
        write_once(output / "vite_process.json", {"pid": vite.pid, "startedAtUnix": time.time(), "executable": admission["nodeExe"], "executableSha256": sha256(Path(admission["nodeExe"])), "argv": [admission["nodeExe"], str(vite_cli), "--config", str(config), "--host", "127.0.0.1", "--port", "3011", "--strictPort"]})
        readers.append(collect_lines(vite.stdout, output / "vite.stdout.log", queue.Queue(maxsize=1)))
        readers.append(collect_lines(vite.stderr, output / "vite.stderr.log", queue.Queue(maxsize=1)))
        adapter.vite_ready(vite, health_nonce, min(clock() + 30, begun + 2040), clock, pause)
        write_once(output / "running.json", {"backendPid": backend.pid, "vitePid": vite.pid, "deadlineSeconds": 2100})
        while clock() < min(begun + 2040, backend_begun + 1740):
            require(backend.poll() is None and vite.poll() is None, "owned server exited during campaign")
            if (output / "stop.request").exists():
                break
            pause(0.5)
        else:
            write_once(output / "deadline.json", {"reason": "campaign or backend lifetime limit reached"})
    finally:
        cleanup_errors = []
        for process in (vite, backend):
            try:
                stop_owned(process)
            except Exception as exc:
                cleanup_errors.append(type(exc).__name__)
        for reader in readers:
            reader.join(timeout=5)
            if reader.is_alive():
                cleanup_errors.append("process log reader remained active")
            elif reader.error:
                cleanup_errors.append("process log reader failed: " + reader.error)
        if started:
            try:
                adapter.stop_fixture()
            except Exception as exc:
                cleanup_errors.append(type(exc).__name__)
        try:
            after = adapter.protected()
            write_once(output / "protected_after.json", {"rows": after, "matchesBefore": after == baseline})
            if after != baseline:
                cleanup_errors.append("protected inventory changed")
        except Exception as exc:
            cleanup_errors.append("protected comparison: " + type(exc).__name__)
        log_hashes = {}
        for path in output.glob("*.log"):
            meta_path = path.with_suffix(path.suffix + ".meta.json")
            try:
                meta = json.loads(meta_path.read_text(encoding="utf-8"))
                digest = sha256(path)
                require(meta.get("readerError") is None and meta.get("retainedBytes") == path.stat().st_size and digest == meta.get("retainedSha256"), "log metadata mismatch")
                log_hashes[path.name] = digest
            except Exception as exc:
                cleanup_errors.append("log verification: " + type(exc).__name__)
        write_once(output / "cleanup.json", {"ownedProcessStopAttempted": True, "fixtureStopAttempted": started, "backendExitCode": backend.poll() if backend else None, "viteExitCode": vite.poll() if vite else None, "logSha256": log_hashes, "errors": cleanup_errors})
        require(not cleanup_errors, "cleanup incomplete: " + ", ".join(cleanup_errors))


def request_stop(root: Path, admission: dict) -> None:
    """Cooperative only; supervisor retains its own Popen handles."""
    output = root / RUNTIME
    original = read_json(root, RUNTIME + "/admission.json")
    require(original.get("seedSha256") == admission["seedSha256"] and original.get("runnerSha256") == admission["pins"][RUNNER], "original admission mismatch")
    if (output / "stop.request").is_file():
        return
    require((output / "running.json").is_file() and not (output / "cleanup.json").exists(), "no live supervised campaign")
    write_once(output / "stop.request", {"requestedAtUnix": time.time()})


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("stage", choices=("inspect", "start", "status", "stop"))
    parser.add_argument("--project-root", type=Path, required=True)
    args = parser.parse_args(argv)
    root = args.project_root.resolve()
    try:
        require((root / ".cvf" / "manifest.json").is_file(), "project root invalid")
        count = None
        if args.stage in ("inspect", "start"):
            frontend = read_json(root, FRONTEND_MANIFEST)
            count = validate_frontend(root, frontend)
            binary = root / ".cvf/runtime/r101_worker/r101_handlers.test.exe"
            require(binary.is_file() and sha256(binary) == BINARY_SHA256, "historical binary mismatch")
        if args.stage != "inspect":
            admission = check_admission(root, for_start=args.stage == "start")
            if args.stage == "start":
                verify_historical(root)
                adapter = ShellAdapter(root, admission, root / RUNTIME)
                run_campaign(root, admission, adapter)
                print(json.dumps({"stage": "start", "result": "CAMPAIGN_ENDED"}))
                return 0
            if args.stage == "stop":
                request_stop(root, admission)
                print(json.dumps({"stage": "stop", "result": "COOPERATIVE_REQUEST_WRITTEN"}))
                return 0
            original = read_json(root, RUNTIME + "/admission.json")
            require(original.get("seedSha256") == admission["seedSha256"], "original admission mismatch")
            print(json.dumps({"stage": "status", "runningReceipt": (root / RUNTIME / "running.json").is_file(), "cleanupReceipt": (root / RUNTIME / "cleanup.json").is_file()}))
            return 0
        print(json.dumps({"stage": "OFFLINE_INSPECT", "frontendFiles": count, "binarySha256": BINARY_SHA256, "runtime": "HOLD"}, sort_keys=True))
        return 0
    except (Hold, OSError, KeyError, TypeError, json.JSONDecodeError) as exc:
        print(json.dumps({"stage": args.stage, "result": "HOLD", "reason": str(exc)}, sort_keys=True))
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
