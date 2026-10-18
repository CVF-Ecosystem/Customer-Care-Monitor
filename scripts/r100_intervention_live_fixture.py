#!/usr/bin/env python3
"""R100 source-bound, one-fixture application admission. No implicit live mode.

The actual network request is emitted only by the hash-pinned Go test binary.
This supervisor never prints a provider credential or launches a shell.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import socket
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parent.parent
RUNTIME = ROOT / ".cvf/runtime/r100_live"
APPROVAL = RUNTIME / "root_approval.json"
OWNERSHIP = RUNTIME / "ownership.json"
FIXTURE = RUNTIME / "fixture.json"
CAMPAIGN = RUNTIME / "campaign.json"
BINARY = RUNTIME / "intervention_r100.test.exe"
SECRET_REL = Path("CVF_SESSION/LOCAL_PROVIDER_SECRETS.json")
SOURCE_REL = (
    "backend/api/handlers/intervention_live_r100_test.go",
    "scripts/r100_intervention_live_fixture.py",
    "scripts/tests/test_r100_intervention_live_fixture.py",
)
FROZEN_IDENTITY = ROOT / "docs/reviews/probes/r100_root/protected_baseline.json"
MODEL = "qwen3.7-flash"
REGION = "ap-southeast-1"
ENDPOINT_SHA = "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f"
TOKENS = 758
MAX_RESPONSE = 1 << 20
CONTAINER = "ccmai-r100-live-20261015"
DB_PASSWORD = "r100-synthetic-only"
NETWORK = "ccmai-r100-isolated-20261015"
SEED_COMMIT = "b33d1fe83be35aa0e20d18590a5462e7f0ecd6fd"
ACTIVATION_COMMIT = "72af95d99d31ad8af4f2daac775841f7ba964841"
SEED_REL = "CVF_SESSION/authority/CCMAI-RUNTIME-100.json"
TRANCHE_REL = "CVF_SESSION/tranches/CCMAI-RUNTIME-100.json"
ORDER_REL = "docs/work_orders/CCMAI_RUNTIME_100_M2_INTERVENTION_LIVE.md"
HANDOFF_REL = "CVF_SESSION/handoffs/AGENT_HANDOFF_M2_LIVE_R100_2026-10-15.md"
CONTAINER_PIN = RUNTIME / "container_pin.json"
PARENT = ROOT / ".cvf/runtime/owner_alibaba_r100_new4_20261015/owner_cap.json"
FRESH_OWNER_GRANT_ID = "CCMAI-RUNTIME-100-OWNER-NEW4-20261015"
TLS_DEPENDENCY_REL = "backend/api/handlers/intervention_live_r096_test.go"
TLS_DEPENDENCY_SHA = "3d2340131829f7afcb54d81be7d903c29fc8b36f160d065a05cd338a5df50dad"
HISTORICAL_REL = "docs/reviews/probes/r098_root/live_outcome_summary.json"


class AdmissionError(Exception):
    def __init__(self, code: str):
        super().__init__(code)
        self.code = code


def digest(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def strict_json(raw: bytes) -> dict[str, Any]:
    def unique(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
        output: dict[str, Any] = {}
        for key, value in pairs:
            if key in output:
                raise ValueError("duplicate")
            output[key] = value
        return output

    value = json.loads(raw, object_pairs_hook=unique,
                       parse_constant=lambda _: (_ for _ in ()).throw(ValueError("number")))
    if not isinstance(value, dict):
        raise ValueError("object")
    return value


def read_object(path: Path, code: str, max_bytes: int = 65536) -> dict[str, Any]:
    try:
        with path.open("rb") as stream:
            raw = stream.read(max_bytes + 1)
        if len(raw) > max_bytes:
            raise ValueError("oversize")
        return strict_json(raw)
    except (OSError, UnicodeError, ValueError, json.JSONDecodeError):
        raise AdmissionError(code) from None


def exclusive(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    raw = json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    try:
        fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY | getattr(os, "O_BINARY", 0), 0o600)
    except FileExistsError:
        raise AdmissionError("MARKER_ALREADY_EXISTS") from None
    except OSError:
        raise AdmissionError("MARKER_CREATE_FAILED") from None
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
    except OSError:
        raise AdmissionError("MARKER_WRITE_FAILED_CONSUMED") from None


def git(*args: str) -> subprocess.CompletedProcess[bytes]:
    return subprocess.run(["git", *args], cwd=ROOT, capture_output=True, check=False)


def assert_ignored_untracked(relative: Path) -> None:
    name = relative.as_posix()
    ignored = git("check-ignore", "--quiet", "--", name)
    tracked = git("ls-files", "--error-unmatch", "--", name)
    if ignored.returncode != 0 or tracked.returncode != 1:
        raise AdmissionError("SECRET_OR_APPROVAL_PATH_NOT_IGNORED_UNTRACKED")


def source_hashes() -> dict[str, str]:
    return {name: digest((ROOT / name).read_bytes()) for name in SOURCE_REL}


def frozen_hashes() -> dict[str, str]:
    identity = read_object(FROZEN_IDENTITY, "FROZEN_IDENTITY_MISSING")
    protected = identity.get("protectedFiles")
    prior = read_object(ROOT / "docs/reviews/probes/r094_root/final_source_identity.json", "PRIOR_IDENTITY_MISSING")
    keys = set(prior.get("source", {})) | {"backend/ai/chat_once_response.go"}
    source = {key: protected.get(key) for key in keys} if isinstance(protected, dict) else None
    if not isinstance(source, dict) or len(source) != 13:
        raise AdmissionError("FROZEN_IDENTITY_INVALID")
    for name, expected in source.items():
        if not isinstance(name, str) or not isinstance(expected, str) or digest((ROOT / name).read_bytes()) != expected:
            raise AdmissionError("FROZEN_SOURCE_CHANGED")
    return source


def historical_protection() -> None:
    baseline = read_object(FROZEN_IDENTITY, "PROTECTED_BASELINE_MISSING", 256 << 10)
    for group in ("protectedFiles", "oldConsumedMarkers"):
        rows = baseline.get(group)
        if not isinstance(rows, dict) or not rows:
            raise AdmissionError("PROTECTED_BASELINE_INVALID")
        for name, expected in rows.items():
            if (not isinstance(name, str) or not isinstance(expected, str) or
                ".." in Path(name).parts or not (ROOT / name).is_file() or
                digest((ROOT / name).read_bytes()) != expected):
                raise AdmissionError("PROTECTED_BASELINE_CHANGED")


def parent_manifest() -> str:
    assert_ignored_untracked(Path(".cvf/runtime/owner_alibaba_r100_new4_20261015/owner_cap.json"))
    parent = read_object(PARENT, "PARENT_LEDGER_UNAVAILABLE")
    expected = {"approvedBy": "Codex /root", "ownerCap": 4,
                "freshOwnerGrantId": FRESH_OWNER_GRANT_ID,
                "freshGrantCap": 4, "historicalR098Calls": 2,
                "seedCommit": SEED_COMMIT,
                "historicalReceipt": HISTORICAL_REL,
                "provider": "Alibaba", "model": MODEL, "region": REGION,
                "endpointSha256": ENDPOINT_SHA, "automaticRetries": 0,
                "workerSlots": [1, 2, 3, 4]}
    if any(parent.get(key) != value for key, value in expected.items()):
        raise AdmissionError("PARENT_LEDGER_MISMATCH")
    predecessor_bytes = (ROOT / HISTORICAL_REL).read_bytes()
    if digest(predecessor_bytes) != parent.get("historicalReceiptSha256"):
        raise AdmissionError("HISTORICAL_RECEIPT_CHANGED")
    predecessor = json.loads(predecessor_bytes)
    if not isinstance(predecessor, list) or len(predecessor) != 2 or any(
        not isinstance(item, dict) or item.get("attempts") != 1 or item.get("model") != MODEL for item in predecessor):
        raise AdmissionError("HISTORICAL_EFFECT_UNKNOWN")
    return digest(PARENT.read_bytes())


def canonical_authority(role: str) -> None:
    seed = read_object(ROOT / SEED_REL, "SEED_UNAVAILABLE")
    committed = git("show", SEED_COMMIT + ":" + SEED_REL)
    # Windows checkout may expand the committed LF bytes to CRLF. Compare the
    # normalized content; all non-newline bytes must still match the seed.
    local_seed = (ROOT / SEED_REL).read_bytes().replace(b"\r\n", b"\n")
    if committed.returncode != 0 or local_seed != committed.stdout:
        raise AdmissionError("IMMUTABLE_SEED_MISMATCH")
    budget = seed.get("executionBudget", {})
    required = {"physicalProviderRequestsTotal": 4, "workerProviderRequestsMax": 4,
                "reviewerProviderRequests": 0, "automaticRetries": 0, "redirects": 0,
                "fallbacks": 0, "model": MODEL, "region": REGION,
                "enableThinking": False,
                "endpointSha256": ENDPOINT_SHA, "requestedCompletionTokens": TOKENS,
                "ownerOutputCeiling": 768, "requestBytesMax": 6000,
                "responseBytesMax": MAX_RESPONSE, "requestTimeoutSeconds": 60,
                "campaignSecondsMax": 300,
                "freshOwnerGrantId": FRESH_OWNER_GRANT_ID,
                "historicalR098Calls": 2, "freshGrantCap": 4,
                "goInvocationsTotal": 4, "ownedDisposableMySQLFixturesTotal": 1}
    if any(budget.get(key) != value for key, value in required.items()):
        raise AdmissionError("SEED_BUDGET_MISMATCH")
    state = read_object(ROOT / "CVF_SESSION/ACTIVE_SESSION_STATE.json", "STATE_UNAVAILABLE", 256 << 10)
    tranche = read_object(ROOT / TRANCHE_REL, "TRANCHE_UNAVAILABLE")
    phase = "BUILD"
    if (state.get("activeTranche") != "CCMAI-RUNTIME-100" or
        state.get("activeHandoff") != HANDOFF_REL or state.get("currentMode") != phase or
        state.get("activePhase") != phase or tranche.get("trancheId") != "CCMAI-RUNTIME-100" or
        tranche.get("baseCommit") != SEED_COMMIT or tranche.get("phase") != phase):
        raise AdmissionError("ACTIVE_CONTINUITY_MISMATCH")
    if role == "worker" and not any(tag in str(state.get("activeRole")) for tag in ("IMPLEMENTATION_WORKER", "REPAIR_WORKER")):
        raise AdmissionError("WORKER_ROLE_NOT_ACTIVE")
    order = (ROOT / ORDER_REL).read_text(encoding="utf-8")
    statuses = re.findall(r"(?m)^Status:\s*(.*?)\s*$", order)
    if statuses != ["ACTIVE"] or not (ROOT / HANDOFF_REL).is_file():
        raise AdmissionError("ORDER_OR_HANDOFF_MISMATCH")
    head = git("rev-parse", "HEAD")
    if head.returncode != 0 or not re.fullmatch(rb"[0-9a-f]{40}\s*", head.stdout):
        raise AdmissionError("GIT_HEAD_UNAVAILABLE")
    for ancestor in (SEED_COMMIT, ACTIVATION_COMMIT):
        if git("merge-base", "--is-ancestor", ancestor, head.stdout.decode().strip()).returncode != 0:
            raise AdmissionError("PINNED_COMMIT_NOT_ANCESTOR")


def admission(role: str = "worker") -> dict[str, Any]:
    canonical_authority(role)
    historical_protection()
    assert_ignored_untracked(Path(".cvf/runtime/r100_live/root_approval.json"))
    approval = read_object(APPROVAL, "ROOT_APPROVAL_MISSING")
    expected = {
        "approvedBy": "Codex /root", "model": MODEL, "region": REGION,
        "endpointSha256": ENDPOINT_SHA, "requestedTokens": TOKENS,
        "physicalRequestsTotal": 4, "workerRequestsMax": 4, "reviewerRequests": 0,
        "automaticRetries": 0,
        "freshOwnerGrantId": FRESH_OWNER_GRANT_ID,
        "tlsDependencySha256": TLS_DEPENDENCY_SHA,
    }
    if any(approval.get(key) != value for key, value in expected.items()):
        raise AdmissionError("ROOT_APPROVAL_METADATA_MISMATCH")
    if approval.get("sourceSha256") != source_hashes() or approval.get("frozenSourceSha256") != frozen_hashes():
        raise AdmissionError("ROOT_APPROVAL_SOURCE_MISMATCH")
    if digest((ROOT / TLS_DEPENDENCY_REL).read_bytes()) != TLS_DEPENDENCY_SHA:
        raise AdmissionError("TLS_DEPENDENCY_CHANGED")
    if approval.get("parentManifestSha256") != parent_manifest():
        raise AdmissionError("ROOT_APPROVAL_PARENT_MISMATCH")
    if not BINARY.is_file() or approval.get("binarySha256") != digest(BINARY.read_bytes()):
        raise AdmissionError("ROOT_APPROVAL_BINARY_MISMATCH")
    if approval.get("activationCommit") != ACTIVATION_COMMIT:
        raise AdmissionError("ACTIVATION_COMMIT_CHANGED")
    return approval


def endpoint_metadata(value: Any) -> str:
    if not isinstance(value, str) or value != value.strip() or not value:
        raise AdmissionError("ENDPOINT_INVALID")
    normalized = value[:-1] if value.endswith("/") else value
    if digest(normalized.encode()) != ENDPOINT_SHA:
        raise AdmissionError("ENDPOINT_HASH_MISMATCH")
    parsed = urlsplit(normalized)
    if (parsed.scheme != "https" or not parsed.hostname or
        not parsed.hostname.endswith("." + REGION + ".maas.aliyuncs.com") or
        parsed.path != "/compatible-mode/v1" or parsed.query or parsed.fragment or
        parsed.username or parsed.password or parsed.port not in (None, 443)):
        raise AdmissionError("ENDPOINT_SHAPE_MISMATCH")
    return normalized


def parse_provider_config(value: dict[str, Any]) -> tuple[str, str, bool]:
    """Accept only the current inert provider container, never executable hints."""
    providers = value.get("providers")
    if not isinstance(providers, dict):
        raise AdmissionError("SECRET_SCHEMA_MISMATCH")
    selected = providers.get("alibaba_maas")
    if not isinstance(selected, dict):
        raise AdmissionError("SECRET_SCHEMA_MISMATCH")
    key = selected.get("api_key")
    if not isinstance(key, str) or key != key.strip() or not 16 <= len(key) <= 512 or any(ord(ch) < 33 or ord(ch) > 126 for ch in key):
        raise AdmissionError("KEY_INVALID")
    endpoint = endpoint_metadata(selected.get("base_url"))
    presets = selected.get("models")
    mentioned = selected.get("recommended_model") == MODEL or (
        isinstance(presets, list) and any(
            item == MODEL if isinstance(item, str) else
            isinstance(item, dict) and item.get("id") == MODEL for item in presets))
    return key, endpoint, mentioned


def load_secret_after_admission() -> tuple[str, str, bool]:
    assert_ignored_untracked(SECRET_REL)
    return parse_provider_config(read_object(ROOT / SECRET_REL, "SECRET_UNAVAILABLE"))


def logical_secret(value: str, key: str) -> bool:
    """Bounded scan through nested JSON text and unicode-escaped key spellings."""
    pending = [value]
    seen: set[str] = set()
    scanned = 0
    while pending:
        candidate = pending.pop()
        if candidate in seen:
            continue
        seen.add(candidate)
        scanned += len(candidate.encode("utf-8", errors="replace"))
        if scanned > 4 * MAX_RESPONSE or len(seen) > 128:
            return True
        if key and key in candidate:
            return True
        if re.search(r"(?i)\bsk-[A-Za-z0-9_-]{16,}\b|\bBearer\s+[A-Za-z0-9._~+/-]{16,}", candidate):
            return True
        unescaped = re.sub(r"\\u([0-9a-fA-F]{4})", lambda match: chr(int(match.group(1), 16)), candidate)
        if unescaped != candidate:
            pending.append(unescaped)
        if not candidate.lstrip().startswith(("{", "[", '"')):
            continue
        try:
            def unique(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
                output: dict[str, Any] = {}
                for name, item in pairs:
                    if name in output:
                        raise AdmissionError("NESTED_DUPLICATE_KEY_UNSAFE")
                    output[name] = item
                return output
            nested = json.loads(candidate, object_pairs_hook=unique)
        except AdmissionError:
            return True
        except RecursionError:
            return True
        except (ValueError, json.JSONDecodeError):
            continue
        stack = [nested]
        while stack:
            item = stack.pop()
            if isinstance(item, str):
                pending.append(item)
            elif isinstance(item, list):
                stack.extend(item)
            elif isinstance(item, dict):
                stack.extend(item.keys())
                stack.extend(item.values())
            if len(stack) > 10000:
                return True
    return False


def sanitize(value: Any, key: str, depth: int = 0) -> Any:
    if depth > 16:
        raise AdmissionError("SANITIZE_DEPTH_EXCEEDED")
    if isinstance(value, str):
        if len(value) > 2 * MAX_RESPONSE:
            raise AdmissionError("SANITIZE_STRING_TOO_LARGE")
        if logical_secret(value, key):
            return "[REDACTED]"
        encoded = json.dumps(key, ensure_ascii=True)[1:-1] if key else ""
        for needle in (key, encoded):
            if needle:
                value = value.replace(needle, "[REDACTED]")
        value = re.sub(r"(?i)\bsk-[A-Za-z0-9_-]{16,}\b", "[REDACTED]", value)
        value = re.sub(r"(?i)\bBearer\s+[A-Za-z0-9._~+/-]{16,}", "[REDACTED]", value)
        return value
    if isinstance(value, list):
        if len(value) > 10000:
            raise AdmissionError("SANITIZE_COLLECTION_TOO_LARGE")
        return [sanitize(item, key, depth + 1) for item in value]
    if isinstance(value, dict):
        if len(value) > 10000:
            raise AdmissionError("SANITIZE_COLLECTION_TOO_LARGE")
        return {str(sanitize(name, key, depth + 1)): sanitize(item, key, depth + 1)
                for name, item in value.items()}
    return value


def safe_serialized(value: dict[str, Any], key: str) -> bytes:
    clean = sanitize(value, key)
    raw = json.dumps(clean, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    if len(raw) > 4 * MAX_RESPONSE:
        raise AdmissionError("RECEIPT_TOO_LARGE")
    escaped = json.dumps(key, ensure_ascii=True)[1:-1].encode() if key else b""
    if key and (key.encode() in raw or (escaped and escaped in raw)):
        raise AdmissionError("RECEIPT_SECRET_GUARD")
    return raw


def binary_call(data: dict[str, Any], key: str = "") -> dict[str, Any]:
    allowed_env = ("SYSTEMROOT", "WINDIR", "PATH", "TEMP", "TMP")
    env = {name: os.environ[name] for name in allowed_env if name in os.environ}
    env["R100_EXPLICIT_BINARY"] = "1"
    # The key exists only in the child's stdin bytes, never env or argv.
    command = [str(BINARY), "-test.run=^TestR100Application$", "-test.v"]
    try:
        completed = subprocess.run(command, cwd=ROOT, env=env, input=json.dumps(data).encode(),
                                   capture_output=True, timeout=315 if data.get("mode") == "live" else 75, check=False)
    except subprocess.TimeoutExpired:
        return {"error_code": "BINARY_TIMEOUT_UNKNOWN", "binary_exit": None, "wire": {"attempts": "UNKNOWN"}}
    if len(completed.stdout) > 4 * MAX_RESPONSE or len(completed.stderr) > MAX_RESPONSE:
        return {"error_code": "BINARY_OUTPUT_TOO_LARGE", "binary_exit": completed.returncode,
                "wire": {"attempts": "UNKNOWN"}}
    stdout = completed.stdout.decode("utf-8", errors="replace")
    stderr = completed.stderr.decode("utf-8", errors="replace")
    lines = [line.removeprefix("R100_RESULT=") for line in stdout.splitlines() if line.startswith("R100_RESULT=")]
    if len(lines) != 1:
        return {"error_code": "BINARY_RESULT_MISSING", "binary_exit": completed.returncode,
                "stdout": sanitize(stdout[:2000], key), "stderr": sanitize(stderr[:2000], key)}
    try:
        result = strict_json(lines[0].encode())
    except (ValueError, json.JSONDecodeError, RecursionError):
        return {"error_code": "BINARY_RESULT_INVALID", "binary_exit": completed.returncode,
                "stdout": sanitize(stdout[:2000], key), "stderr": sanitize(stderr[:2000], key)}
    result["binary_exit"] = completed.returncode
    result["stderr"] = sanitize(stderr[:2000], key)
    return sanitize(result, key)


def utc() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


def docker(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["docker", *args], cwd=ROOT, capture_output=True, text=True, check=False)


def docker_object(*args: str) -> dict[str, Any]:
    inspected = docker(*args)
    if inspected.returncode != 0:
        raise AdmissionError("DOCKER_INSPECT_FAILED")
    try:
        value = json.loads(inspected.stdout)
        if not isinstance(value, list) or len(value) != 1 or not isinstance(value[0], dict):
            raise ValueError("shape")
        return value[0]
    except (ValueError, TypeError, json.JSONDecodeError):
        raise AdmissionError("DOCKER_INSPECT_INVALID") from None


def protected_ids() -> dict[str, str]:
    protected = {}
    for name in ("ccma-app-1", "ccma-nginx-1", "ccma-db-1"):
        row = docker_object("inspect", name)
        if row.get("Name") != "/" + name or not row.get("State", {}).get("Running"):
            raise AdmissionError("PROTECTED_CONTAINER_STATE_CHANGED")
        protected[name] = row["Id"]
    return protected


def ownership(require_running: bool = True) -> dict[str, Any]:
    intent = read_object(OWNERSHIP, "FIXTURE_OWNERSHIP_MISSING")
    pin = read_object(CONTAINER_PIN, "FIXTURE_PIN_MISSING")
    ident = pin.get("containerId")
    if not isinstance(ident, str) or not re.fullmatch(r"[0-9a-f]{64}", ident):
        raise AdmissionError("FIXTURE_ID_INVALID")
    if protected_ids() != intent.get("protectedIds"):
        raise AdmissionError("PROTECTED_CONTAINER_IDS_CHANGED")
    metadata = docker_object("inspect", ident)
    labels = metadata.get("Config", {}).get("Labels", {})
    network = docker_object("network", "inspect", pin.get("networkId", ""))
    image = docker_object("image", "inspect", "mysql:8.0")
    mounts = sorted(mount.get("Name") for mount in metadata.get("Mounts", [])
                    if mount.get("Type") == "volume")
    binding = (metadata.get("NetworkSettings", {}).get("Ports") or {}).get("3306/tcp")
    attached = (metadata.get("NetworkSettings", {}).get("Networks") or {}).get(NETWORK, {})
    configured = metadata.get("HostConfig", {}).get("PortBindings", {}).get("3306/tcp")
    running = bool(metadata.get("State", {}).get("Running"))
    configured_exact = (isinstance(configured, list) and len(configured) == 1 and
                        configured[0].get("HostIp") == "127.0.0.1" and
                        configured[0].get("HostPort") == str(intent.get("port")))
    active_exact = (isinstance(binding, list) and len(binding) == 1 and
                    binding[0].get("HostIp") == "127.0.0.1" and
                    binding[0].get("HostPort") == str(intent.get("port")) and
                    attached.get("NetworkID") == pin.get("networkId"))
    if (metadata.get("Id") != ident or metadata.get("Name") != "/" + CONTAINER or
        labels.get("ccmai.tranche") != "R100" or labels.get("ccmai.owner") != "m2-worker" or
        metadata.get("Image") != pin.get("imageId") or image.get("Id") != pin.get("imageId") or
        network.get("Id") != pin.get("networkId") or network.get("Name") != NETWORK or
        network.get("Labels", {}).get("ccmai.owner") != "m2-worker" or
        metadata.get("HostConfig", {}).get("NetworkMode") != NETWORK or
        not configured_exact or
        (running and not active_exact) or
        mounts != pin.get("volumeIds") or len(mounts) != 1 or
        len(metadata.get("Mounts", [])) != len(mounts)):
        raise AdmissionError("FIXTURE_OWNERSHIP_MISMATCH")
    if require_running and not running:
        raise AdmissionError("FIXTURE_NOT_RUNNING")
    return {**intent, **pin}


def setup_fixture() -> None:
    admission("worker")
    if any((PARENT.parent / f"slot-{slot}.effect").exists() for slot in range(1, 5)):
        raise AdmissionError("OWNER_PHYSICAL_LEDGER_ALREADY_CONSUMED")
    if OWNERSHIP.exists() or CONTAINER_PIN.exists() or FIXTURE.exists() or CAMPAIGN.exists():
        raise AdmissionError("FIXTURE_ALREADY_USED")
    if docker("inspect", CONTAINER).returncode == 0 or docker("network", "inspect", NETWORK).returncode == 0:
        raise AdmissionError("FIXTURE_NAME_ALREADY_PRESENT")
    protected = protected_ids()
    image = docker_object("image", "inspect", "mysql:8.0")
    image_id = image.get("Id")
    if not isinstance(image_id, str) or not image_id.startswith("sha256:"):
        raise AdmissionError("CACHED_IMAGE_UNAVAILABLE")
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
    # Exclusive intent predates every Docker effect. A failure cannot silently
    # create a second fixture or reset a role/campaign ledger.
    exclusive(OWNERSHIP, {"name": CONTAINER, "network": NETWORK, "port": port,
                          "imageId": image_id, "protectedIds": protected, "createdAt": utc()})
    created_network = docker("network", "create", "--label", "ccmai.tranche=R100",
                             "--label", "ccmai.owner=m2-worker", NETWORK)
    network_id = created_network.stdout.strip()
    if created_network.returncode != 0 or not re.fullmatch(r"[0-9a-f]{64}", network_id):
        raise AdmissionError("FIXTURE_NETWORK_CREATE_FAILED")
    created = docker("run", "-d", "--pull", "never", "--name", CONTAINER,
                     "--label", "ccmai.tranche=R100", "--label", "ccmai.owner=m2-worker",
                     "-e", "MYSQL_ROOT_PASSWORD=" + DB_PASSWORD,
                     "-e", "MYSQL_ROOT_HOST=%",
                     "-e", "MYSQL_DATABASE=r100_fixture", "--network", NETWORK,
                     "-p", f"127.0.0.1:{port}:3306", "mysql:8.0")
    ident = created.stdout.strip()
    if created.returncode != 0 or not re.fullmatch(r"[0-9a-f]{64}", ident):
        raise AdmissionError("FIXTURE_CREATE_FAILED")
    metadata = docker_object("inspect", ident)
    volumes = sorted(mount.get("Name") for mount in metadata.get("Mounts", [])
                     if mount.get("Type") == "volume")
    if len(volumes) != 1 or any(not isinstance(item, str) for item in volumes):
        raise AdmissionError("FIXTURE_MOUNT_UNEXPECTED")
    exclusive(CONTAINER_PIN, {"containerId": ident, "networkId": network_id,
                              "imageId": image_id, "volumeIds": volumes})
    ownership()
    for _ in range(60):
        ready = docker("exec", "-e", "MYSQL_PWD=" + DB_PASSWORD, ident,
                       "mysql", "-h", "127.0.0.1", "-u", "root", "-N", "-B",
                       "-e", "SELECT 1 FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='r100_fixture'")
        if ready.returncode == 0 and ready.stdout.strip() == "1":
            break
        time.sleep(1)
    else:
        raise AdmissionError("FIXTURE_NOT_READY")
    dsn = f"root:{DB_PASSWORD}@tcp(127.0.0.1:{port})/r100_fixture?parseTime=true&charset=utf8mb4"
    seeded = binary_call({"mode": "seed", "role": "seed", "dsn": dsn})
    if seeded.get("error_code") != "" or not isinstance(seeded.get("fixture"), dict):
        raise AdmissionError("FIXTURE_SEED_FAILED")
    exclusive(FIXTURE, {"schemaVersion": 1, "containerId": ident, "port": port,
                        "fixture": seeded["fixture"], "seedBinarySha256": digest(BINARY.read_bytes())})
    print("R100_FIXTURE_SEEDED")


def validate_fixture_fields(fields: Any) -> dict[str, Any]:
    scalar = {"tenant", "user", "job", "channel", "wait_conversation", "deny_job", "deny_conversation"}
    if not isinstance(fields, dict) or set(fields) != scalar | {"conversations"}:
        raise AdmissionError("FIXTURE_IDS_INVALID")
    def valid_uuid(value: Any) -> bool:
        return isinstance(value, str) and re.fullmatch(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}", value) is not None
    conversations = fields["conversations"]
    if (any(not valid_uuid(fields[name]) for name in scalar) or
        not isinstance(conversations, list) or len(conversations) != 4 or
        any(not valid_uuid(value) for value in conversations) or
        len(set(conversations)) != 4 or
        len(set(conversations) | {fields[name] for name in scalar}) != 11):
        raise AdmissionError("FIXTURE_IDS_INVALID")
    return fields


def fixture() -> tuple[dict[str, Any], str]:
    owner = ownership()
    value = read_object(FIXTURE, "FIXTURE_MANIFEST_MISSING")
    if value.get("containerId") != owner["containerId"] or value.get("port") != owner.get("port") or value.get("seedBinarySha256") != digest(BINARY.read_bytes()):
        raise AdmissionError("FIXTURE_MANIFEST_MISMATCH")
    fields = validate_fixture_fields(value.get("fixture"))
    dsn = f"root:{DB_PASSWORD}@tcp(127.0.0.1:{owner['port']})/r100_fixture?parseTime=true&charset=utf8mb4"
    return fields, dsn


def claim(role: str) -> None:
    if role != "worker":
        raise AdmissionError("ROLE_INVALID")
    if any((PARENT.parent / f"slot-{slot}.effect").exists() for slot in range(1, 5)):
        raise AdmissionError("OWNER_PHYSICAL_LEDGER_ALREADY_CONSUMED")
    exclusive(CAMPAIGN, {"startedAt": utc(), "startedEpoch": time.time(), "maxSeconds": 300})
    exclusive(RUNTIME / "worker.claim", {"role": "worker", "claimedAt": utc(), "state": "CONSUMED_BEFORE_HTTP"})


def live(role: str) -> None:
    approval = admission(role)
    fields, dsn = fixture()
    claim(role)
    key = ""
    preset_mentions_model = False
    result: dict[str, Any] = {"error_code": "SECRET_NOT_READ", "wire": {"attempts": 0}}
    try:
        key, endpoint, preset_mentions_model = load_secret_after_admission()
        exclusive(RUNTIME / (role + ".dispatch"), {"role": role, "binarySha256": approval["binarySha256"], "at": utc()})
        result = binary_call({"mode": "live", "role": role, "dsn": dsn, "fixture": fields,
                              "endpoint": endpoint, "api_key": key}, key)
    except AdmissionError as error:
        result = {"error_code": error.code, "wire": {"attempts": 0}}
    parent_slots = {f"slot{slot}": (PARENT.parent / f"slot-{slot}.effect").is_file()
                    for slot in range(1, 5)}
    receipt = {"schemaVersion": 1, "role": role, "sourceSha256": approval["sourceSha256"],
               "binarySha256": approval["binarySha256"], "model": MODEL,
               "requestedTokens": TOKENS, "endpointSha256": ENDPOINT_SHA,
               "parentManifestSha256": approval["parentManifestSha256"],
               "parentPhysicalSlots": parent_slots,
               "presetMentionsPinnedModel": preset_mentions_model,
               "modelAvailabilityBeforeResponse": "UNKNOWN",
               "actual": result, "billing": "UNKNOWN", "recordedAt": utc()}
    path = RUNTIME / (role + "_receipt.json")
    serialized = safe_serialized(receipt, key)
    exclusive(path, strict_json(serialized))
    print("R100_" + role.upper() + "_RECEIPT_RECORDED")


def authorize_continuation(slot: int) -> None:
    """Root invokes this after independently reading an actual semantic failure."""
    if slot not in (1, 2, 3):
        raise AdmissionError("CONTINUATION_SLOT_INVALID")
    approval = admission("worker")
    fields, _ = fixture()
    owner = ownership()
    if protected_ids() != owner["protectedIds"]:
        raise AdmissionError("PROTECTED_CONTAINER_IDS_CHANGED")
    campaign = read_object(CAMPAIGN, "CAMPAIGN_NOT_STARTED")
    started = campaign.get("startedEpoch")
    if isinstance(started, bool) or not isinstance(started, (int, float)) or campaign.get("maxSeconds") != 300 or not 0 <= time.time() - started < 300:
        raise AdmissionError("CAMPAIGN_EXPIRED")
    for index in range(1, slot + 1):
        if not (PARENT.parent / f"slot-{index}.effect").is_file():
            raise AdmissionError("PHYSICAL_LEDGER_GAP")
    if any((PARENT.parent / f"slot-{index}.effect").exists() for index in range(slot + 1, 5)):
        raise AdmissionError("PHYSICAL_LEDGER_ADVANCED")
    receipt_path = RUNTIME / f"attempt-{slot}.json"
    if not receipt_path.is_file() or receipt_path.stat().st_size > 4 * MAX_RESPONSE:
        raise AdmissionError("ATTEMPT_RECEIPT_UNAVAILABLE")
    raw = receipt_path.read_bytes()
    receipt_sha = digest(raw)
    if (RUNTIME / f"attempt-{slot}.ready").read_text(encoding="ascii") != receipt_sha:
        raise AdmissionError("ATTEMPT_RECEIPT_NOT_READY")
    receipt = strict_json(raw)
    conversations = fields["conversations"]
    if receipt.get("fixture") != fields or receipt.get("db_read_ok") is not True or receipt.get("positive_status") != 200 or receipt.get("detail_status") != 200 or receipt.get("wait_status") != 200 or receipt.get("deny_status") != 403:
        raise AdmissionError("ATTEMPT_CONTEXT_MISMATCH")
    wire = receipt.get("wire")
    attempt = receipt.get("attempt")
    case = receipt.get("case")
    snapshot = receipt.get("snapshot")
    messages = receipt.get("messages")
    receipts = receipt.get("receipts")
    request = strict_json(wire.get("request", "").encode()) if isinstance(wire, dict) else {}
    body = strict_json(wire.get("response", "").encode()) if isinstance(wire, dict) else {}
    usage = body.get("usage")
    if (set(request) != {"model", "messages", "max_completion_tokens", "enable_thinking"} or
        request.get("model") != MODEL or request.get("max_completion_tokens") != TOKENS or request.get("enable_thinking") is not False or
        not isinstance(request.get("messages"), list) or len(request["messages"]) != 2 or
        [item.get("role") for item in request["messages"] if isinstance(item, dict)] != ["system", "user"]):
        raise AdmissionError("REQUEST_SHAPE_MISMATCH")
    if (receipt.get("slot") != slot or receipt.get("decision") != "SEMANTIC_INVALID" or
        receipt.get("budget_cap") != 4 or receipt.get("budget_consumed") != slot or
        not isinstance(wire, dict) or wire.get("attempts") != 1 or wire.get("status") != 200 or wire.get("transport_code") not in (None, "") or
        body.get("model") != MODEL or not isinstance(usage, dict) or
        not isinstance(attempt, dict) or attempt.get("usage_state") != "KNOWN" or attempt.get("state") != "RESPONSE_RECEIVED" or
        attempt.get("input_usage_presence") != "known" or attempt.get("output_usage_presence") != "known" or
        not isinstance(usage.get("prompt_tokens"), int) or isinstance(usage.get("prompt_tokens"), bool) or
        not isinstance(usage.get("completion_tokens"), int) or isinstance(usage.get("completion_tokens"), bool) or
        not 0 <= usage["completion_tokens"] <= 768 or usage["prompt_tokens"] < 0 or
        attempt.get("input_tokens") != usage["prompt_tokens"] or attempt.get("output_tokens") != usage["completion_tokens"] or
        attempt.get("response_model") != MODEL or not isinstance(case, dict) or case.get("execution") != "ERROR" or case.get("reason_code") != "OUTPUT_INVALID" or
        receipt.get("proposals") not in (None, [])):
        raise AdmissionError("SEMANTIC_CONTINUATION_NOT_PROVEN")
    if (case.get("tenant_id") != fields["tenant"] or case.get("job_id") != fields["job"] or case.get("conversation_id") != conversations[slot - 1] or
        not isinstance(case.get("run_id"), str) or not case["run_id"] or
        attempt.get("tenant_id") != fields["tenant"] or attempt.get("job_id") != fields["job"] or attempt.get("run_id") != case["run_id"] or
        not isinstance(snapshot, dict) or snapshot.get("id") != case.get("snapshot_id") or snapshot.get("tenant_id") != fields["tenant"] or
        snapshot.get("job_run_id") != case["run_id"] or snapshot.get("conversation_id") != conversations[slot - 1] or
        snapshot.get("digest") != case.get("snapshot_digest") or not isinstance(snapshot.get("manifest"), str) or
        digest(snapshot["manifest"].encode()) != snapshot["digest"] or
        not isinstance(messages, list) or len(messages) != 1 or not isinstance(messages[0], dict) or
        messages[0].get("tenant_id") != fields["tenant"] or messages[0].get("conversation_id") != conversations[slot - 1] or messages[0].get("sender_type") != "customer" or
        not isinstance(receipts, list) or len(receipts) != 1 or not isinstance(receipts[0], dict) or
        receipts[0].get("case_id") != case.get("id") or receipts[0].get("run_id") != case["run_id"] or receipts[0].get("code") != "OUTPUT_INVALID" or receipts[0].get("usage_state") != "KNOWN"):
        raise AdmissionError("ATTEMPT_IDENTITY_MISMATCH")
    token = {"approvedBy": "Codex /root", "priorSlot": slot, "nextSlot": slot + 1,
             "nextConversation": conversations[slot], "receiptSha256": receipt_sha,
             "approvalSha256": digest(APPROVAL.read_bytes()),
             "parentManifestSha256": approval["parentManifestSha256"],
             "expiresEpoch": started + 300}
    exclusive(RUNTIME / f"continuation-{slot}.json", token)
    print(f"R100_ROOT_CONTINUATION_{slot}_TO_{slot + 1}_RECORDED")


def cleanup() -> None:
    owner = ownership(require_running=False)
    ident = owner["containerId"]
    stopped = docker("rm", "-f", ident)
    if stopped.returncode != 0:
        raise AdmissionError("FIXTURE_CONTAINER_REMOVE_FAILED")
    for volume in owner["volumeIds"]:
        if docker("volume", "rm", volume).returncode != 0:
            raise AdmissionError("FIXTURE_VOLUME_REMOVE_FAILED")
    if docker("network", "rm", owner["networkId"]).returncode != 0:
        raise AdmissionError("FIXTURE_NETWORK_REMOVE_FAILED")
    if protected_ids() != owner["protectedIds"]:
        raise AdmissionError("PROTECTED_CONTAINER_IDS_CHANGED_AFTER_CLEANUP")
    print("R100_OWNED_FIXTURE_REMOVED_MARKERS_RETAINED")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("command", choices=("offline", "precheck", "setup", "live", "continue", "cleanup"))
    parser.add_argument("--role", choices=("worker",))
    parser.add_argument("--slot", type=int)
    args = parser.parse_args()
    try:
        if args.command == "offline":
            tls_dependency = digest((ROOT / TLS_DEPENDENCY_REL).read_bytes())
            if tls_dependency != TLS_DEPENDENCY_SHA:
                raise AdmissionError("TLS_DEPENDENCY_CHANGED")
            print(json.dumps({"sourceSha256": source_hashes(), "frozenSourceSha256": frozen_hashes(),
                              "tlsDependencySha256": tls_dependency}, sort_keys=True))
        elif args.command == "precheck":
            admission("worker")
            _, _, mentioned = load_secret_after_admission()
            print(json.dumps({"schemaAccepted": True, "keyShapeValid": True,
                              "endpointPinValid": True, "requestedModel": MODEL,
                              "presetMentionsPinnedModel": mentioned,
                              "modelAvailability": "UNKNOWN"}, sort_keys=True))
        elif args.command == "setup":
            setup_fixture()
        elif args.command == "live":
            if args.role != "worker":
                raise AdmissionError("ROLE_REQUIRED")
            live(args.role)
        elif args.command == "continue":
            if args.slot is None:
                raise AdmissionError("SLOT_REQUIRED")
            authorize_continuation(args.slot)
        else:
            cleanup()
        return 0
    except AdmissionError as error:
        print(error.code, file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
