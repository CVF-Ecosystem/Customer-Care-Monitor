#!/usr/bin/env python3
"""Finite, opt-in R139 execution fixture supervisor.

Import, inspect, and --help are inert. `run` requires a separate, committed,
owner-approved runtime/effect admission. The R139 source-resume admission is
explicitly insufficient. This script never interprets a synthetic AI result
as real governance evidence.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timedelta, timezone
import errno
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import socket
import subprocess
import sys
import time
import urllib.request
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parent.parent
IMAGE = "sha256:7dcddc01f13bab2f15cde676d44d01f61fc9f99fe7785e86196dfc07d358ae2b"
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
READY = re.compile(r"R139_BROWSER_READY (http://127\.0\.0\.1:[0-9]{2,5})\Z")
EXPECTED_NATIVE = 8
EXPECTED_BROWSER = 1
EXPECTED_GRANTS = 3
MAX_SECONDS = 20 * 60
MIN_CLEANUP_SECONDS = 180
MYSQL_STARTUP_WINDOW_SECONDS = 60
MYSQL_MAX_PROBES = 45
MYSQL_PROBE_SPACING_SECONDS = 1
MYSQL_PROBE_TIMEOUT_SECONDS = 8
WINDOWS_LISTENER_QUERY_TIMEOUT_SECONDS = 5
GRANT_MIN_SECONDS = 60
GRANT_MAX_SECONDS = 24 * 60 * 60
R139_REPLACEMENT_FIXTURE_NAME = "ccmai-r139-r4-20261023"
R139_REPLACEMENT_DATABASE = "ccma_r139_r4_20261023"
R139_REPLACEMENT_RUNTIME_ROOT = Path(".cvf/runtime/r139_integrated_execution_r4_20261023")
PROMPT_TEXT = "Analyze only fabricated Vietnamese complaint; return the required JSON."
PROMPT_SHA256 = hashlib.sha256(PROMPT_TEXT.encode("utf-8")).hexdigest()
ACTIVE_HARD_DEADLINE: float | None = None
ACTIVE_CLEANUP_DEADLINE: float | None = None
CLEANUP_ERRORS: list[str] = []
COMMAND_SEQUENCE = 0
MINIMAL_ENVIRONMENT_KEYS = {"SYSTEMROOT", "WINDIR", "TEMP", "TMP", "PATH", "USERPROFILE", "LOCALAPPDATA"}


class Hold(RuntimeError):
    pass


def require(ok: bool, code: str) -> None:
    if not ok:
        raise Hold(code)


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read_json(path: Path) -> dict:
    value = json.loads(path.read_text(encoding="utf-8"))
    require(isinstance(value, dict), "JSON_OBJECT_REQUIRED")
    return value


def write_new(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("x", encoding="utf-8") as stream:
        json.dump(value, stream, sort_keys=True, ensure_ascii=True, indent=2)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())


def append(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as stream:
        stream.write(json.dumps(value, sort_keys=True, ensure_ascii=True) + "\n")
        stream.flush()
        os.fsync(stream.fileno())


def minimal_environment() -> dict[str, str]:
    return {key: value for key, value in os.environ.items() if key.upper() in MINIMAL_ENVIRONMENT_KEYS}


def parse_grant_deadline_utc(value: object, now: datetime | None = None) -> datetime:
    require(isinstance(value, str) and value.endswith("Z"), "GRANT_DEADLINE_UTC_INVALID")
    try:
        deadline = datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    except ValueError as exc:
        raise Hold("GRANT_DEADLINE_UTC_INVALID") from exc
    require(deadline.strftime("%Y-%m-%dT%H:%M:%SZ") == value, "GRANT_DEADLINE_UTC_INVALID")
    current = now or datetime.now(timezone.utc)
    require(current.tzinfo is not None and current.utcoffset() == timedelta(0), "GRANT_DEADLINE_REFERENCE_NOT_UTC")
    require(current + timedelta(seconds=GRANT_MIN_SECONDS) <= deadline <= current + timedelta(seconds=GRANT_MAX_SECONDS), "GRANT_DEADLINE_OUT_OF_RANGE")
    return deadline


def project_path(relative: str) -> Path:
    require(isinstance(relative, str) and relative and not Path(relative).is_absolute(), "PATH_INVALID")
    require("\\" not in relative and ".." not in Path(relative).parts, "PATH_INVALID")
    path = (ROOT / relative).resolve()
    require(path.is_relative_to(ROOT.resolve()) and path.is_file(), "PATH_MISSING_OR_OUTSIDE_PROJECT")
    return path


def replacement_external_evidence_dir() -> Path:
    return ROOT.parent / ".ccmai_local_evidence" / "r139_execution_proof_20261023" / "fixture_r4"


def committed(relative: str) -> Path:
    path = project_path(relative)
    result = subprocess.run(["git", "show", "HEAD:" + relative], cwd=ROOT, capture_output=True, timeout=15)
    require(result.returncode == 0 and result.stdout.replace(b"\r\n", b"\n") == path.read_bytes().replace(b"\r\n", b"\n"), "ADMISSION_NOT_COMMITTED")
    return path


NATIVE_ID_FIELDS = ("nativeContractId", "trancheId", "campaignNonce", "pageId", "browserSessionId", "operator", "closeMode")
NATIVE_EVENT_FIELDS = ("creation", "control", "capture")
PAGE_TARGET_ID_FIELDS = ("nativeContractId", "trancheId", "campaignNonce", "pageId", "ownedScopeId", "operator", "closeMode")
PAGE_TARGET_EVENTS = ("creation", "control", "capture", "handoff")
CUA_ID_FIELDS = ("nativeContractId", "trancheId", "campaignNonce", "browserId", "tabId",
                 "ownedScopeId", "producerAgentTaskId", "observerAgentTaskId", "operator", "closeMode")
CUA_EVENTS = ("creation", "control", "capture", "handoff")


def native_text(value: object, code: str) -> str:
    require(type(value) is str and 1 <= len(value) <= 128 and value == value.strip() and
            all(32 <= ord(character) < 127 for character in value), code)
    return value


def native_path_text(value: object, code: str) -> str:
    require(type(value) is str and 1 <= len(value) <= 512 and value == value.strip() and
            all(32 <= ord(character) < 127 for character in value), code)
    return value


def native_time(value: object, code: str) -> datetime:
    require(type(value) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", value) is not None, code)
    try:
        parsed = datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    except ValueError as exc:
        raise Hold(code) from exc
    require(parsed <= datetime.now(timezone.utc), code)
    return parsed


def native_root(value: dict, field: str) -> Path:
    rel = native_path_text(value.get(field), "NATIVE_ROOT_INVALID")
    if field == "runtimeRoot":
        require(Path(rel).is_absolute(), "NATIVE_ROOT_INVALID")
        candidate = Path(rel)
    else:
        require(not Path(rel).is_absolute() and "\\" not in rel and ".." not in Path(rel).parts and
                rel.startswith(".cvf/runtime/"), "NATIVE_ROOT_INVALID")
        candidate = ROOT / rel
    root = candidate.resolve()
    require(candidate == root and root.is_relative_to((ROOT / ".cvf/runtime").resolve()), "NATIVE_ROOT_INVALID")
    return root


def native_file(rel: object, sha: object, root: Path, code: str) -> tuple[Path, dict]:
    relative = native_path_text(rel, code)
    require(not Path(relative).is_absolute() and "\\" not in relative and ".." not in Path(relative).parts,
            code)
    require(type(sha) is str and HEX64.fullmatch(sha) is not None, code)
    candidate = ROOT / relative
    path = candidate.resolve()
    require(candidate == path and path.is_relative_to(root) and path != root and path.is_file(), code)
    try:
        require(digest(path) == sha, code)
        return path, read_json(path)
    except (OSError, ValueError) as exc:
        raise Hold(code) from exc


def native_identity(admission: dict, record: dict, code: str) -> None:
    expected = {field: admission.get(field) for field in NATIVE_ID_FIELDS}
    require(expected["closeMode"] in ("api", "operator", "operator_fallback"), code)
    if expected["closeMode"] == "operator_fallback":
        require(admission.get("fallbackCloseOperator") == expected["operator"], code)
    for field in NATIVE_ID_FIELDS:
        native_text(expected[field], code)
        require(type(record.get(field)) is str and record[field] == expected[field], code)


def native_event(admission: dict, path: object, sha: object, root: Path, event: str) -> dict:
    _, data = native_file(path, sha, root, "NATIVE_EVIDENCE_INVALID")
    native_identity(admission, data, "NATIVE_EVIDENCE_IDENTITY_MISMATCH")
    require(data.get("event") == event, "NATIVE_EVIDENCE_EVENT_INVALID")
    native_time(data.get("atUTC"), "NATIVE_EVIDENCE_TIME_INVALID")
    return data


def page_target_identity(admission: dict, record: dict, code: str) -> None:
    require(admission.get("proofContract") == "native_page_target_v1", code)
    require(admission.get("scopeOrigin") == "caller_owned_explicit_page_target" and
            type(admission.get("ownedPageIds")) is list and
            admission["ownedPageIds"] == [admission.get("pageId")], code)
    require(admission.get("closeMode") == "operator_fallback" and
            admission.get("fallbackCloseOperator") == admission.get("operator"), code)
    require("browserSessionId" not in admission and "browserSessionId" not in record, code)
    for field in PAGE_TARGET_ID_FIELDS:
        expected = native_text(admission.get(field), code)
        require(record.get(field) == expected and type(record.get(field)) is str, code)
    require(record.get("proofContract") == "native_page_target_v1" and
            record.get("scopeOrigin") == "caller_owned_explicit_page_target" and
            type(record.get("ownedPageIds")) is list and
            record["ownedPageIds"] == [admission["pageId"]], code)


def page_target_origin(record: dict, code: str) -> None:
    for field in ("nativeTaskRunRef", "nativeToolCallRef", "rawOriginalToolReference"):
        native_path_text(record.get(field), code)
    require(type(record.get("rawOriginalToolSha256")) is str and
            HEX64.fullmatch(record["rawOriginalToolSha256"]) is not None, code)
    basis = record.get("timeBasis")
    at = native_time(record.get("atUTC"), code)
    require(basis in ("original_native_tool_return", "normalized_from_original_native_tool_return"), code)
    if basis == "normalized_from_original_native_tool_return":
        raw = record.get("rawObservedAtUTC")
        require(type(raw) is str, code)
        try:
            original = datetime.fromisoformat(raw.replace("Z", "+00:00"))
        except ValueError as exc:
            raise Hold(code) from exc
        require(original.tzinfo is not None and original.utcoffset() == timedelta(0) and
                original.replace(microsecond=0) == at, code)


def page_target_ids(value: object, code: str) -> list[str]:
    require(type(value) is list and all(type(item) is str and native_text(item, code) for item in value), code)
    require(len(value) == len(set(value)), code)
    return value


def page_target_file(admission: dict, path: object, sha: object, root: Path,
                     event: str) -> dict:
    _, data = native_file(path, sha, root, "PAGE_TARGET_EVIDENCE_INVALID")
    page_target_identity(admission, data, "PAGE_TARGET_IDENTITY_INVALID")
    require(data.get("event") == event, "PAGE_TARGET_EVENT_INVALID")
    page_target_origin(data, "PAGE_TARGET_ORIGIN_INVALID")
    return data


def verify_page_target_receipt(admission: dict, proof: dict, *, require_runtime_absent: bool) -> None:
    readiness, runtime = native_roots(admission)
    require(proof.get("status") == "RESERVED_NATIVE_PAGE_TARGET_CONTROL_CAPTURE_HANDOFF" and
            proof.get("browserId") == "antigravity_native" and
            proof.get("protocol") == "https" and proof.get("closeObserved") is False,
            "PAGE_TARGET_PREFIX_INVALID")
    page_target_identity(admission, proof, "PAGE_TARGET_PREFIX_IDENTITY_INVALID")
    if require_runtime_absent:
        require(not runtime.exists(), "PRODUCT_RUNTIME_ROOT_PREEXISTING")
    sealed = native_time(proof.get("reservedAtUTC"), "PAGE_TARGET_SEAL_INVALID")
    nonce = admission["campaignNonce"]
    creation_url = f"https://example.com/#r139_nonce={nonce}&phase=created"
    control_url = f"https://example.com/#r139_nonce={nonce}&phase=controlled"
    baseline = page_target_file(admission, proof.get("baselineInventoryPath"),
                                proof.get("baselineInventorySha256"), readiness, "baseline_inventory")
    baseline_ids = page_target_ids(baseline.get("pageIds"), "PAGE_TARGET_BASELINE_IDS_INVALID")
    require(admission["pageId"] not in baseline_ids, "PAGE_TARGET_PREEXISTING")
    previous = native_time(baseline["atUTC"], "PAGE_TARGET_BASELINE_TIME_INVALID")
    require(previous <= sealed and baseline.get("inventoryScope") == "metadata_only", "PAGE_TARGET_BASELINE_SCOPE_INVALID")
    first_run = None
    creation_at = None
    for event in PAGE_TARGET_EVENTS:
        data = page_target_file(admission, proof.get(event + "EvidencePath"),
                                proof.get(event + "EvidenceSha256"), readiness, event)
        at = native_time(data["atUTC"], "PAGE_TARGET_EVENT_TIME_INVALID")
        require(previous <= at <= sealed, "PAGE_TARGET_CHRONOLOGY_INVALID")
        previous = at
        if event == "creation":
            first_run = data["nativeTaskRunRef"]
            creation_at = at
            require(data.get("pageIdToReplace") == "new_page" and
                    data.get("observedPageId") == admission["pageId"] and
                    data.get("observedUrl") == creation_url, "PAGE_TARGET_CREATION_INVALID")
        elif event == "control":
            require(data["nativeTaskRunRef"] == first_run and
                    data.get("pageIdToReplace") == admission["pageId"] and
                    data.get("observedUrl") == control_url, "PAGE_TARGET_CONTROL_INVALID")
        elif event == "capture":
            require(data["nativeTaskRunRef"] == first_run and
                    data.get("observedPageId") == admission["pageId"] and
                    data.get("observedUrl") == control_url and
                    type(data.get("captureSha256")) is str and
                    HEX64.fullmatch(data["captureSha256"]) is not None,
                    "PAGE_TARGET_CAPTURE_INVALID")
        else:
            require(data["nativeTaskRunRef"] != first_run and
                    data.get("observedPageId") == admission["pageId"] and
                    data.get("observedUrl") == control_url and
                    data.get("observedNonce") == nonce,
                    "PAGE_TARGET_CROSS_TASK_HANDOFF_INVALID")
    ledger = page_target_file(admission, proof.get("creationLedgerPath"),
                              proof.get("creationLedgerSha256"), readiness, "creation_ledger")
    require(ledger.get("newPageIds") == [admission["pageId"]] and
            ledger.get("baselineInventorySha256") == proof.get("baselineInventorySha256") and
            ledger.get("creationEvidenceSha256") == proof.get("creationEvidenceSha256") and
            ledger.get("nativeTaskRunRef") == first_run and
            creation_at is not None and
            creation_at <= native_time(ledger["atUTC"], "PAGE_TARGET_LEDGER_TIME_INVALID") <= sealed,
            "PAGE_TARGET_CREATION_LEDGER_INVALID")
    for field in ("browserCloseReceiptPath", "actionEvidencePath", "finalInventoryPath",
                  "rawNativeInventoryPath", "browserDoneSignalPath"):
        native_future_path(admission, field, runtime, absent=require_runtime_absent)


def cua_identity(admission: dict, record: dict, code: str) -> None:
    require(admission.get("proofContract") == "cua_page_target_v1" and
            admission.get("browserTool") == "supported_CUA_page_target" and
            admission.get("browserKind") == "iab" and
            admission.get("scopeOrigin") == "caller_owned_explicit_cua_tab_target" and
            admission.get("ownedTabIds") == [admission.get("tabId")] and
            admission.get("closeMode") == "operator_fallback" and
            admission.get("fallbackCloseOperator") == admission.get("operator"), code)
    require(admission.get("producerAgentTaskId") != admission.get("observerAgentTaskId"), code)
    for forbidden in ("pageId", "browserSessionId", "nativeTaskRunRef", "nativeToolCallRef",
                      "ownedPageIds", "rawOriginalToolReference", "rawOriginalToolSha256"):
        require(forbidden not in admission and forbidden not in record, code)
    for field in CUA_ID_FIELDS:
        expected = native_text(admission.get(field), code)
        require(type(record.get(field)) is str and record[field] == expected, code)
    require(record.get("proofContract") == "cua_page_target_v1" and
            record.get("browserKind") == "iab" and
            record.get("scopeOrigin") == "caller_owned_explicit_cua_tab_target" and
            record.get("ownedTabIds") == [admission["tabId"]], code)


def cua_origin(record: dict, code: str) -> datetime:
    for field in ("cuaCallRef", "originalCuaReturnReference"):
        native_path_text(record.get(field), code)
    require(type(record.get("originalCuaReturnSha256")) is str and
            HEX64.fullmatch(record["originalCuaReturnSha256"]) is not None and
            record.get("provenanceBasis") == "caller_recorded_original_cua_return" and
            record.get("rootOriginalReturnReviewRequired") is True, code)
    at = native_time(record.get("atUTC"), code)
    require(record.get("observedAtUTC") == record.get("atUTC"), code)
    return at


def cua_derived(record: dict, code: str) -> datetime:
    require(record.get("provenanceBasis") == "derived_from_original_cua_returns" and
            record.get("rootOriginalReturnReviewRequired") is True and
            "originalCuaReturnReference" not in record and
            "originalCuaReturnSha256" not in record and "cuaCallRef" not in record, code)
    links = record.get("sourceEvidenceSha256")
    require(type(links) is dict and bool(links) and
            all(type(key) is str and native_text(key, code) and type(value) is str and
                HEX64.fullmatch(value) is not None for key, value in links.items()), code)
    at = native_time(record.get("atUTC"), code)
    require(record.get("observedAtUTC") == record.get("atUTC"), code)
    return at


def cua_file(admission: dict, path: object, sha: object, root: Path, event: str) -> dict:
    _, record = native_file(path, sha, root, "CUA_EVIDENCE_INVALID")
    cua_identity(admission, record, "CUA_IDENTITY_INVALID")
    require(record.get("event") == event, "CUA_EVENT_INVALID")
    if event in ("creation_ledger", "action_sequence", "handoff"):
        cua_derived(record, "CUA_DERIVED_ORIGIN_INVALID")
    else:
        cua_origin(record, "CUA_ORIGIN_INVALID")
    return record


def verify_cua_page_target_receipt(admission: dict, proof: dict, *, require_runtime_absent: bool) -> None:
    readiness, runtime = native_roots(admission)
    cua_identity(admission, proof, "CUA_PREFIX_IDENTITY_INVALID")
    require(proof.get("status") == "RESERVED_CUA_TAB_CONTROL_CAPTURE_HANDOFF" and
            proof.get("protocol") == "https" and proof.get("closeObserved") is False and
            proof.get("rootOriginalReturnReviewed") is True, "CUA_PREFIX_INVALID")
    if require_runtime_absent:
        require(not runtime.exists(), "PRODUCT_RUNTIME_ROOT_PREEXISTING")
    sealed = native_time(proof.get("reservedAtUTC"), "CUA_SEAL_INVALID")
    nonce = admission["campaignNonce"]
    created = f"https://example.com/#r139_nonce={nonce}&phase=created"
    controlled = f"https://example.com/#r139_nonce={nonce}&phase=controlled"
    baseline = cua_file(admission, proof.get("baselineInventoryPath"),
                        proof.get("baselineInventorySha256"), readiness, "baseline_inventory")
    baseline_ids = page_target_ids(baseline.get("tabIds"), "CUA_BASELINE_IDS_INVALID")
    require(admission["tabId"] not in baseline_ids and
            baseline.get("inventoryScope") == "metadata_only", "CUA_BASELINE_INVALID")
    previous = native_time(baseline["atUTC"], "CUA_BASELINE_TIME_INVALID")
    producer = admission["producerAgentTaskId"]
    observer = admission["observerAgentTaskId"]
    creation_at = None
    for event in CUA_EVENTS:
        row = cua_file(admission, proof.get(event + "EvidencePath"),
                       proof.get(event + "EvidenceSha256"), readiness, event)
        at = native_time(row["atUTC"], "CUA_EVENT_TIME_INVALID")
        require(previous <= at <= sealed, "CUA_CHRONOLOGY_INVALID")
        previous = at
        if event == "creation":
            creation_at = at
            require(row.get("agentTaskId") == producer and
                    row.get("createdTabId") == admission["tabId"] and
                    row.get("requestedBrowserId") == admission["browserId"] and
                    row.get("requestedUrl") == created and
                    row.get("callCompleted") is True and
                    "observedUrl" not in row and
                    row.get("method") == "createBrowserTab", "CUA_CREATION_INVALID")
        elif event == "control":
            require(row.get("agentTaskId") == producer and
                    row.get("targetTabId") == admission["tabId"] and
                    row.get("requestedUrl") == controlled and
                    row.get("callCompleted") is True and
                    "observedUrl" not in row and
                    row.get("method") == "goto", "CUA_CONTROL_INVALID")
        elif event == "capture":
            require(row.get("agentTaskId") == producer and
                    row.get("targetTabId") == admission["tabId"] and
                    "observedUrl" not in row and
                    row.get("method") == "getScreenshot" and
                    type(row.get("captureSha256")) is str and
                    HEX64.fullmatch(row["captureSha256"]) is not None, "CUA_CAPTURE_INVALID")
        else:
            require(row.get("agentTaskId") == observer and
                    row.get("method") == "getTab_then_listTabs" and
                    row.get("requestedTabId") == admission["tabId"] and
                    row.get("observedTabId") == admission["tabId"] and
                    row.get("metadataBrowserId") == admission["browserId"] and
                    row.get("metadataTabId") == admission["tabId"] and
                    row.get("observedUrl") == controlled and
                    row.get("observedNonce") == nonce,
                    "CUA_CROSS_AGENT_HANDOFF_INVALID")
            require(row.get("sourceEvidenceSha256") == {
                "getTab": row.get("getTabReturnSha256"),
                "listTabs": row.get("metadataReturnSha256")} and
                all(type(row.get(field)) is str and HEX64.fullmatch(row[field]) is not None
                    for field in ("getTabReturnSha256", "metadataReturnSha256")) and
                all(native_path_text(row.get(field), "CUA_HANDOFF_ORIGIN_INVALID")
                    for field in ("getTabReturnReference", "metadataReturnReference",
                                  "getTabCallRef", "metadataCallRef")),
                "CUA_HANDOFF_ORIGIN_INVALID")
    ledger = cua_file(admission, proof.get("creationLedgerPath"),
                      proof.get("creationLedgerSha256"), readiness, "creation_ledger")
    require(ledger.get("newTabIds") == [admission["tabId"]] and
            ledger.get("baselineInventorySha256") == proof.get("baselineInventorySha256") and
            ledger.get("creationEvidenceSha256") == proof.get("creationEvidenceSha256") and
            ledger.get("sourceEvidenceSha256") == {
                "baselineInventory": proof.get("baselineInventorySha256"),
                "creation": proof.get("creationEvidenceSha256")} and
            ledger.get("agentTaskId") == producer and creation_at is not None and
            creation_at <= native_time(ledger["atUTC"], "CUA_LEDGER_TIME_INVALID") <= sealed,
            "CUA_CREATION_LEDGER_INVALID")
    for field in ("browserCloseReceiptPath", "actionEvidencePath", "finalInventoryPath",
                  "rawCuaInventoryPath", "browserDoneSignalPath"):
        native_future_path(admission, field, runtime, absent=require_runtime_absent)


def native_roots(admission: dict) -> tuple[Path, Path]:
    readiness = native_root(admission, "browserReadinessRoot")
    runtime = native_root(admission, "runtimeRoot")
    require(readiness != runtime and not readiness.is_relative_to(runtime) and
            not runtime.is_relative_to(readiness), "NATIVE_ROOTS_OVERLAP")
    require(readiness.is_dir(), "READINESS_ROOT_MISSING")
    return readiness, runtime


def native_future_path(admission: dict, field: str, root: Path, *, absent: bool) -> Path:
    rel = native_path_text(admission.get(field), "FUTURE_EVIDENCE_PATH_INVALID")
    require(not Path(rel).is_absolute() and "\\" not in rel and ".." not in Path(rel).parts,
            "FUTURE_EVIDENCE_PATH_INVALID")
    candidate = ROOT / rel
    resolved = candidate.resolve()
    require(candidate == resolved and resolved.is_relative_to(root) and resolved != root and
            (not absent or not resolved.exists()), "FUTURE_EVIDENCE_PATH_INVALID")
    return resolved


def verify_browser_receipt(value: dict, *, require_runtime_absent: bool = True) -> None:
    tool = value.get("browserTool")
    require(tool in ("supported_CUA_IAB", "supported_CUA_page_target", "antigravity_native", "antigravity_page_target") and value.get("browserSupported") is True, "BROWSER_CAPABILITY_UNPROVEN")
    browser_rel = value.get("browserReceiptPath", "")
    require(isinstance(browser_rel, str) and browser_rel.strip(), "BROWSER_RECEIPT_PATH_MISSING")
    browser = project_path(browser_rel)
    expected = value.get("browserReceiptSha256", "")
    require(isinstance(expected, str) and HEX64.fullmatch(expected) and digest(browser) == expected, "BROWSER_RECEIPT_MISMATCH")
    proof = read_json(browser)
    if tool == "supported_CUA_IAB":
        require(proof.get("status") == "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" and proof.get("browserId") == "iab", "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(type(proof.get("controlObserved")) is bool and proof["controlObserved"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(type(proof.get("captureObserved")) is bool and proof["captureObserved"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(type(proof.get("closeObserved")) is bool and proof["closeObserved"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(type(proof.get("ownedTabsFinal")) is int and not isinstance(proof.get("ownedTabsFinal"), bool) and proof["ownedTabsFinal"] == 0, "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
    elif tool == "antigravity_native":
        readiness, runtime = native_roots(value)
        _, proof = native_file(browser_rel, expected, ROOT.resolve(), "RESERVATION_RECEIPT_INVALID")
        require(proof.get("status") != "CLOSED_EXACT_OWNED_PAGE", "TERMINAL_CLOSE_RECEIPT_CANNOT_BE_PREFIXTURE_RESERVATION")
        require(proof.get("status") == "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE", "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(proof.get("browserId") == "antigravity_native", "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(type(proof.get("exclusiveReservation")) is bool and proof["exclusiveReservation"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(type(proof.get("controlObserved")) is bool and proof["controlObserved"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require(type(proof.get("captureObserved")) is bool and proof["captureObserved"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
        require("closeObserved" not in proof or proof["closeObserved"] is False, "EARLY_CLOSE_FORBIDDEN")
        require(proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
        native_identity(value, proof, "BROWSER_PROOF_IDENTITY_MISMATCH")
        if require_runtime_absent:
            require(not runtime.exists(), "PRODUCT_RUNTIME_ROOT_PREEXISTING")
        reserved = native_time(proof.get("reservedAtUTC"), "RESERVATION_TIMESTAMP_INVALID")
        previous_evidence = None
        for event in NATIVE_EVENT_FIELDS:
            data = native_event(value, proof.get(event + "EvidencePath"),
                                proof.get(event + "EvidenceSha256"), readiness, event)
            at = native_time(data["atUTC"], "NATIVE_EVIDENCE_TIME_INVALID")
            require(at <= reserved and (previous_evidence is None or previous_evidence <= at),
                    "RESERVATION_EVIDENCE_CHRONOLOGY_INVALID")
            previous_evidence = at
        for field in ("browserCloseReceiptPath", "actionEvidencePath", "finalInventoryPath", "rawNativeInventoryPath"):
            native_future_path(value, field, runtime, absent=require_runtime_absent)
    elif tool == "supported_CUA_page_target":
        readiness, _ = native_roots(value)
        _, proof = native_file(browser_rel, expected, ROOT.resolve(), "CUA_PREFIX_FILE_INVALID")
        require((ROOT / browser_rel).resolve().is_relative_to(readiness), "CUA_PREFIX_ROOT_INVALID")
        verify_cua_page_target_receipt(value, proof, require_runtime_absent=require_runtime_absent)
    elif tool == "antigravity_page_target":
        readiness, _ = native_roots(value)
        _, proof = native_file(browser_rel, expected, ROOT.resolve(), "PAGE_TARGET_PREFIX_FILE_INVALID")
        require((ROOT / browser_rel).resolve().is_relative_to(readiness), "PAGE_TARGET_PREFIX_ROOT_INVALID")
        verify_page_target_receipt(value, proof, require_runtime_absent=require_runtime_absent)
    else:
        raise Hold("BROWSER_TOOL_UNSUPPORTED")


def verify_browser_terminal_close(admission: dict, runtime: Path, external: Path) -> dict:
    if admission.get("browserTool") == "supported_CUA_page_target":
        return verify_cua_page_target_terminal(admission, runtime)
    if admission.get("browserTool") == "antigravity_page_target":
        return verify_page_target_terminal(admission, runtime)
    verify_browser_receipt(admission, require_runtime_absent=False)
    readiness, product_root = native_roots(admission)
    require(runtime.resolve() == product_root, "PRODUCT_RUNTIME_ROOT_MISMATCH")
    _, reservation = native_file(admission.get("browserReceiptPath"), admission.get("browserReceiptSha256"),
                                  ROOT.resolve(), "RESERVATION_RECEIPT_INVALID")
    native_identity(admission, reservation, "RESERVATION_IDENTITY_MISMATCH")
    close_path = native_future_path(admission, "browserCloseReceiptPath", product_root, absent=False)
    require(close_path.is_file(), "BROWSER_CLOSE_RECEIPT_INVALID")
    try:
        receipt = read_json(close_path)
        close_sha = digest(close_path)
    except (OSError, ValueError) as exc:
        raise Hold("BROWSER_CLOSE_RECEIPT_INVALID") from exc
    native_identity(admission, receipt, "BROWSER_CLOSE_IDENTITY_MISMATCH")
    require(receipt.get("status") == "CLOSED_EXACT_OWNED_PAGE", "BROWSER_CLOSE_STATUS_INVALID")
    require(receipt.get("browserId") == "antigravity_native", "BROWSER_CLOSE_BROWSER_ID_INVALID")
    require(receipt.get("closeObserved") is True, "BROWSER_CLOSE_NOT_OBSERVED")
    owned_tabs = receipt.get("ownedTabsFinal")
    require(type(owned_tabs) is int and owned_tabs == 0, "BROWSER_CLOSE_OWNED_TABS_REMAINING")
    reserved = native_time(reservation.get("reservedAtUTC"), "RESERVATION_TIMESTAMP_INVALID")
    require(receipt.get("reservedAtUTC") == reservation.get("reservedAtUTC"), "RESERVATION_TIME_MISMATCH")
    action_at = native_time(receipt.get("actionAtUTC"), "ACTION_TIMESTAMP_INVALID")
    closed = native_time(receipt.get("closedAtUTC"), "BROWSER_CLOSE_TIMESTAMP_INVALID")
    try:
        deadline_text = admission.get("grantDeadlineUTC")
        require(type(deadline_text) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", deadline_text), "GRANT_DEADLINE_INVALID")
        deadline = datetime.strptime(deadline_text, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    except ValueError as exc:
        raise Hold("GRANT_DEADLINE_INVALID") from exc
    require(reserved <= action_at <= closed <= deadline, "BROWSER_CHRONOLOGY_INVALID")
    require(receipt.get("actionEvidencePath") == admission.get("actionEvidencePath") and
            receipt.get("finalInventoryPath") == admission.get("finalInventoryPath"), "FUTURE_EVIDENCE_PATH_MISMATCH")
    native_future_path(admission, "actionEvidencePath", product_root, absent=False)
    native_future_path(admission, "finalInventoryPath", product_root, absent=False)
    native_future_path(admission, "rawNativeInventoryPath", product_root, absent=False)
    action = native_event(admission, receipt.get("actionEvidencePath"), receipt.get("actionEvidenceSha256"),
                          product_root, "action_sequence")
    actions = action.get("actions")
    require(type(actions) is list and len(actions) >= 2, "ACTION_SEQUENCE_INVALID")
    previous = reserved
    close_index = None
    for index, row in enumerate(actions):
        require(type(row) is dict and type(row.get("pageId")) is str and row["pageId"] == admission["pageId"] and
                row.get("action") in ("control", "capture", "close"), "ACTION_SEQUENCE_INVALID")
        at = native_time(row.get("atUTC"), "ACTION_SEQUENCE_TIME_INVALID")
        require(previous <= at <= closed, "ACTION_SEQUENCE_CHRONOLOGY_INVALID")
        previous = at
        if row["action"] == "close":
            require(close_index is None and at == closed, "ACTION_SEQUENCE_CLOSE_INVALID")
            close_index = index
    require(close_index == len(actions) - 1 and native_time(action["atUTC"], "ACTION_TIMESTAMP_INVALID") == action_at,
            "POST_CLOSE_ACTION_OR_ACTION_TIME_MISMATCH")
    _, inventory = native_file(receipt.get("finalInventoryPath"), receipt.get("finalInventorySha256"),
                               product_root, "FINAL_INVENTORY_INVALID")
    native_identity(admission, inventory, "FINAL_INVENTORY_IDENTITY_MISMATCH")
    observed = native_time(inventory.get("inventoryAtUTC"), "INVENTORY_TIMESTAMP_INVALID")
    require(closed <= observed <= deadline, "INVENTORY_CHRONOLOGY_INVALID")
    pages = inventory.get("pageIds")
    require(type(pages) is list and all(type(page) is str and native_text(page, "PAGE_ID_INVALID") for page in pages) and
            len(pages) == len(set(pages)) and admission["pageId"] not in pages, "FINAL_PAGE_LIST_INVALID")
    require(type(inventory.get("ownedCount")) is int and inventory["ownedCount"] == 0,
            "FINAL_OWNED_COUNT_INVALID")
    require(inventory.get("rawNativeInventoryPath") == admission.get("rawNativeInventoryPath"),
            "RAW_NATIVE_INVENTORY_PATH_MISMATCH")
    _, raw_inventory = native_file(inventory.get("rawNativeInventoryPath"), inventory.get("rawNativeInventorySha256"),
                                   product_root, "RAW_NATIVE_INVENTORY_INVALID")
    native_identity(admission, raw_inventory, "RAW_NATIVE_INVENTORY_IDENTITY_MISMATCH")
    require(raw_inventory.get("event") == "native_inventory" and
            raw_inventory.get("atUTC") == inventory.get("inventoryAtUTC") and
            raw_inventory.get("pageIds") == pages and
            type(raw_inventory.get("ownedCount")) is int and raw_inventory["ownedCount"] == inventory["ownedCount"],
            "RAW_NATIVE_INVENTORY_MISMATCH")
    return {"receipt": receipt, "closeReceiptSha256": close_sha,
            "actionEvidenceSha256": receipt["actionEvidenceSha256"],
            "finalInventorySha256": receipt["finalInventorySha256"]}


def verify_page_target_terminal(admission: dict, runtime: Path) -> dict:
    verify_browser_receipt(admission, require_runtime_absent=False)
    readiness, product_root = native_roots(admission)
    require(runtime.resolve() == product_root, "PAGE_TARGET_RUNTIME_ROOT_MISMATCH")
    _, prefix = native_file(admission.get("browserReceiptPath"),
                            admission.get("browserReceiptSha256"), ROOT.resolve(),
                            "PAGE_TARGET_PREFIX_INVALID")
    close_path = native_future_path(admission, "browserCloseReceiptPath", product_root, absent=False)
    require(close_path.is_file(), "PAGE_TARGET_CLOSE_FILE_MISSING")
    receipt = read_json(close_path)
    close_sha = digest(close_path)
    page_target_identity(admission, receipt, "PAGE_TARGET_CLOSE_IDENTITY_INVALID")
    require(receipt.get("status") == "CLOSED_EXACT_OWNED_PAGE_TARGET" and
            receipt.get("browserId") == "antigravity_native" and
            receipt.get("closeObserved") is True and
            type(receipt.get("ownedTabsFinal")) is int and receipt["ownedTabsFinal"] == 0 and
            "actualCloseActionAtUTC" not in receipt and "closedAtUTC" not in receipt and
            receipt.get("closureTimeBasis") == "native_final_metadata_observation" and
            receipt.get("reservedAtUTC") == prefix.get("reservedAtUTC"),
            "PAGE_TARGET_CLOSE_STATUS_INVALID")
    report_at = native_time(receipt.get("operatorReportReceivedAtUTC"), "PAGE_TARGET_REPORT_TIME_INVALID")
    observed_at = native_time(receipt.get("closureObservedAtUTC"), "PAGE_TARGET_OBSERVATION_TIME_INVALID")
    reserved = native_time(prefix.get("reservedAtUTC"), "PAGE_TARGET_SEAL_INVALID")
    deadline_text = admission.get("grantDeadlineUTC")
    require(type(deadline_text) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", deadline_text),
            "PAGE_TARGET_DEADLINE_INVALID")
    deadline = datetime.strptime(deadline_text, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    require(reserved <= report_at <= observed_at <= deadline, "PAGE_TARGET_CLOSE_CHRONOLOGY_INVALID")
    require(type(receipt.get("ownerReportReferenceSha256")) is str and
            HEX64.fullmatch(receipt["ownerReportReferenceSha256"]) is not None and
            receipt.get("operator") == "Owner in this conversation", "PAGE_TARGET_OWNER_REPORT_INVALID")
    require(receipt.get("actionEvidencePath") == admission.get("actionEvidencePath") and
            receipt.get("finalInventoryPath") == admission.get("finalInventoryPath"),
            "PAGE_TARGET_FUTURE_PATH_MISMATCH")
    action = page_target_file(admission, receipt.get("actionEvidencePath"),
                              receipt.get("actionEvidenceSha256"), product_root, "action_sequence")
    rows = action.get("actions")
    require(type(rows) is list and len(rows) >= 2 and
            action.get("atUTC") == receipt.get("actionAtUTC"), "PAGE_TARGET_ACTIONS_MISSING")
    previous = reserved
    for index, row in enumerate(rows):
        require(type(row) is dict and row.get("pageId") == admission["pageId"] and
                row.get("action") in ("control", "capture", "handoff", "product_read",
                                      "product_execute", "owner_close_report"),
                "PAGE_TARGET_ACTION_INVALID")
        at = native_time(row.get("atUTC"), "PAGE_TARGET_ACTION_TIME_INVALID")
        require(previous <= at <= report_at, "PAGE_TARGET_ACTION_CHRONOLOGY_INVALID")
        previous = at
        if row["action"] == "owner_close_report":
            require(index == len(rows) - 1 and at == report_at and
                    row.get("reportSource") == "owner_in_conversation" and
                    row.get("ownerReportReferenceSha256") == receipt["ownerReportReferenceSha256"],
                    "PAGE_TARGET_OWNER_REPORT_ACTION_INVALID")
        else:
            require(index < len(rows) - 1 and
                    type(row.get("rawOriginalToolSha256")) is str and
                    HEX64.fullmatch(row["rawOriginalToolSha256"]) is not None and
                    native_path_text(row.get("rawOriginalToolReference"), "PAGE_TARGET_ACTION_ORIGIN_INVALID"),
                    "PAGE_TARGET_ACTION_ORIGIN_INVALID")
    require(rows[-1]["action"] == "owner_close_report", "PAGE_TARGET_POST_CLOSE_ACTION")
    inventory = page_target_file(admission, receipt.get("finalInventoryPath"),
                                 receipt.get("finalInventorySha256"), product_root, "final_inventory")
    require(inventory.get("atUTC") == receipt.get("closureObservedAtUTC") and
            inventory.get("rawNativeInventoryPath") == admission.get("rawNativeInventoryPath") and
            inventory.get("inventoryScope") == "metadata_only" and
            type(inventory.get("ownedCount")) is int and inventory["ownedCount"] == 0,
            "PAGE_TARGET_FINAL_INVENTORY_INVALID")
    final_ids = page_target_ids(inventory.get("pageIds"), "PAGE_TARGET_FINAL_IDS_INVALID")
    require(admission["pageId"] not in final_ids, "PAGE_TARGET_STILL_PRESENT")
    baseline = page_target_file(admission, prefix.get("baselineInventoryPath"),
                                prefix.get("baselineInventorySha256"), readiness, "baseline_inventory")
    baseline_ids = page_target_ids(baseline.get("pageIds"), "PAGE_TARGET_BASELINE_IDS_INVALID")
    require(set(final_ids) == set(baseline_ids), "PAGE_TARGET_PROTECTED_DRIFT")
    raw = page_target_file(admission, inventory.get("rawNativeInventoryPath"),
                           inventory.get("rawNativeInventorySha256"), product_root, "native_inventory")
    require(raw.get("atUTC") == inventory["atUTC"] and raw.get("pageIds") == final_ids and
            raw.get("inventoryScope") == "metadata_only" and
            type(raw.get("ownedCount")) is int and raw["ownedCount"] == 0 and
            raw.get("nativeTaskRunRef") != baseline.get("nativeTaskRunRef"),
            "PAGE_TARGET_RAW_INVENTORY_MISMATCH")
    marker_path = native_future_path(admission, "browserDoneSignalPath", product_root, absent=False)
    require(marker_path.is_file(), "PAGE_TARGET_DONE_MARKER_MISSING")
    marker = read_json(marker_path)
    marker_at = native_time(marker.get("atUTC"), "PAGE_TARGET_DONE_TIME_INVALID")
    require(marker.get("status") == "TERMINAL_EVIDENCE_WRITTEN" and
            marker.get("proofContract") == "native_page_target_v1" and
            marker.get("campaignNonce") == admission["campaignNonce"] and
            marker.get("pageId") == admission["pageId"] and
            marker.get("ownedScopeId") == admission["ownedScopeId"] and
            observed_at <= marker_at <= deadline and
            type(marker.get("evidenceSha256")) is dict and
            marker["evidenceSha256"] == {
                "browserCloseReceiptPath": close_sha,
                "actionEvidencePath": receipt["actionEvidenceSha256"],
                "finalInventoryPath": receipt["finalInventorySha256"],
                "rawNativeInventoryPath": inventory["rawNativeInventorySha256"],
            }, "PAGE_TARGET_DONE_MARKER_INVALID")
    return {"receipt": receipt, "closeReceiptSha256": close_sha,
            "actionEvidenceSha256": receipt["actionEvidenceSha256"],
            "finalInventorySha256": receipt["finalInventorySha256"]}


def verify_cua_page_target_terminal(admission: dict, runtime: Path) -> dict:
    verify_browser_receipt(admission, require_runtime_absent=False)
    readiness, product_root = native_roots(admission)
    require(runtime.resolve() == product_root, "CUA_RUNTIME_ROOT_MISMATCH")
    _, prefix = native_file(admission.get("browserReceiptPath"),
                            admission.get("browserReceiptSha256"), ROOT.resolve(),
                            "CUA_PREFIX_INVALID")
    close_path = native_future_path(admission, "browserCloseReceiptPath", product_root, absent=False)
    require(close_path.is_file(), "CUA_CLOSE_FILE_MISSING")
    receipt = read_json(close_path)
    close_sha = digest(close_path)
    cua_identity(admission, receipt, "CUA_CLOSE_IDENTITY_INVALID")
    require(receipt.get("status") == "CLOSED_EXACT_OWNED_CUA_TAB" and
            receipt.get("closeObserved") is True and
            receipt.get("ownedTabsFinal") == 0 and type(receipt.get("ownedTabsFinal")) is int and
            "actualCloseActionAtUTC" not in receipt and "closedAtUTC" not in receipt and
            receipt.get("closureTimeBasis") == "cua_final_metadata_observation" and
            receipt.get("reservedAtUTC") == prefix.get("reservedAtUTC") and
            receipt.get("rootOriginalReturnReviewed") is True, "CUA_CLOSE_STATUS_INVALID")
    report_at = native_time(receipt.get("operatorReportReceivedAtUTC"), "CUA_REPORT_TIME_INVALID")
    observed_at = native_time(receipt.get("closureObservedAtUTC"), "CUA_OBSERVATION_TIME_INVALID")
    reserved = native_time(prefix.get("reservedAtUTC"), "CUA_SEAL_INVALID")
    deadline_text = admission.get("grantDeadlineUTC")
    require(type(deadline_text) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", deadline_text),
            "CUA_DEADLINE_INVALID")
    deadline = datetime.strptime(deadline_text, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
    require(reserved <= report_at <= observed_at <= deadline and
            type(receipt.get("ownerReportReferenceSha256")) is str and
            HEX64.fullmatch(receipt["ownerReportReferenceSha256"]) is not None and
            receipt.get("operator") == "Owner in this conversation", "CUA_CLOSE_CHRONOLOGY_INVALID")
    require(receipt.get("actionEvidencePath") == admission.get("actionEvidencePath") and
            receipt.get("finalInventoryPath") == admission.get("finalInventoryPath"),
            "CUA_FUTURE_PATH_MISMATCH")
    action = cua_file(admission, receipt.get("actionEvidencePath"),
                      receipt.get("actionEvidenceSha256"), product_root, "action_sequence")
    rows = action.get("actions")
    require(type(rows) is list and len(rows) >= 1 and
            action.get("atUTC") == receipt.get("actionAtUTC"), "CUA_ACTIONS_MISSING")
    require(action.get("sourceEvidenceSha256") == {
        str(index): row.get("originalCuaReturnSha256") for index, row in enumerate(rows)
        if type(row) is dict and row.get("action") != "owner_close_report"
    } or (len(rows) == 1 and rows[0].get("action") == "owner_close_report" and
          action.get("sourceEvidenceSha256") == {"prefix": admission["browserReceiptSha256"]}),
        "CUA_ACTION_SOURCE_LINKS_INVALID")
    previous = reserved
    for index, row in enumerate(rows):
        require(type(row) is dict and row.get("tabId") == admission["tabId"] and
                row.get("action") in ("control", "capture", "handoff", "product_read",
                                      "product_execute", "owner_close_report"), "CUA_ACTION_INVALID")
        at = native_time(row.get("atUTC"), "CUA_ACTION_TIME_INVALID")
        require(previous <= at <= report_at, "CUA_ACTION_CHRONOLOGY_INVALID")
        previous = at
        if row["action"] == "owner_close_report":
            require(index == len(rows) - 1 and at == report_at and
                    row.get("reportSource") == "owner_in_conversation" and
                    row.get("ownerReportReferenceSha256") == receipt["ownerReportReferenceSha256"],
                    "CUA_OWNER_REPORT_ACTION_INVALID")
        else:
            require(index < len(rows) - 1 and
                    type(row.get("originalCuaReturnSha256")) is str and
                    HEX64.fullmatch(row["originalCuaReturnSha256"]) is not None and
                    native_path_text(row.get("originalCuaReturnReference"), "CUA_ACTION_ORIGIN_INVALID") and
                    native_path_text(row.get("cuaCallRef"), "CUA_ACTION_ORIGIN_INVALID") and
                    row.get("provenanceBasis") == "caller_recorded_original_cua_return",
                    "CUA_ACTION_ORIGIN_INVALID")
    require(rows[-1]["action"] == "owner_close_report", "CUA_POST_CLOSE_ACTION")
    inventory = cua_file(admission, receipt.get("finalInventoryPath"),
                         receipt.get("finalInventorySha256"), product_root, "final_inventory")
    require(inventory.get("atUTC") == receipt.get("closureObservedAtUTC") and
            inventory.get("rawCuaInventoryPath") == admission.get("rawCuaInventoryPath") and
            inventory.get("inventoryScope") == "metadata_only" and
            type(inventory.get("ownedCount")) is int and inventory["ownedCount"] == 0,
            "CUA_FINAL_INVENTORY_INVALID")
    final_ids = page_target_ids(inventory.get("tabIds"), "CUA_FINAL_IDS_INVALID")
    require(admission["tabId"] not in final_ids, "CUA_TAB_STILL_PRESENT")
    baseline = cua_file(admission, prefix.get("baselineInventoryPath"),
                        prefix.get("baselineInventorySha256"), readiness, "baseline_inventory")
    baseline_ids = page_target_ids(baseline.get("tabIds"), "CUA_BASELINE_IDS_INVALID")
    require(set(final_ids) == set(baseline_ids), "CUA_PROTECTED_DRIFT")
    raw = cua_file(admission, inventory.get("rawCuaInventoryPath"),
                   inventory.get("rawCuaInventorySha256"), product_root, "cua_inventory")
    require(raw.get("atUTC") == inventory["atUTC"] and raw.get("tabIds") == final_ids and
            raw.get("inventoryScope") == "metadata_only" and
            type(raw.get("ownedCount")) is int and raw["ownedCount"] == 0 and
            raw.get("cuaCallRef") != baseline.get("cuaCallRef"),
            "CUA_RAW_INVENTORY_MISMATCH")
    marker_path = native_future_path(admission, "browserDoneSignalPath", product_root, absent=False)
    require(marker_path.is_file(), "CUA_DONE_MARKER_MISSING")
    marker = read_json(marker_path)
    marker_at = native_time(marker.get("atUTC"), "CUA_DONE_TIME_INVALID")
    require(marker.get("status") == "TERMINAL_EVIDENCE_WRITTEN" and
            marker.get("proofContract") == "cua_page_target_v1" and
            marker.get("campaignNonce") == admission["campaignNonce"] and
            marker.get("browserId") == admission["browserId"] and
            marker.get("tabId") == admission["tabId"] and
            marker.get("ownedScopeId") == admission["ownedScopeId"] and
            observed_at <= marker_at <= deadline and
            marker.get("evidenceSha256") == {
                "browserCloseReceiptPath": close_sha,
                "actionEvidencePath": receipt["actionEvidenceSha256"],
                "finalInventoryPath": receipt["finalInventorySha256"],
                "rawCuaInventoryPath": inventory["rawCuaInventorySha256"],
            }, "CUA_DONE_MARKER_INVALID")
    return {"receipt": receipt, "closeReceiptSha256": close_sha,
            "actionEvidenceSha256": receipt["actionEvidenceSha256"],
            "finalInventorySha256": receipt["finalInventorySha256"]}


SOURCE_PIN_REQUIRED = {"backend/api/handlers/intervention_execution_browser_test.go", "scripts/r139_execution_supervisor.py", "scripts/r139_vite_config.mjs.template", "scripts/tests/test_r139_execution_supervisor.py", "backend/go.mod", "backend/go.sum", "frontend/package.json", "frontend/package-lock.json", "frontend/vite.config.ts"}
CUA_AMENDMENT_REF = {"path": "CVF_SESSION/authority/CCMAI-RUNTIME-139-INTERNAL-CUA-SOURCE-ADMISSION.json",
                     "sha256": "0ee21ae60ea9f3248b36af58ed003892f3e4c1f088c687f8683f9ae4f37ddd9e"}
CUA_INHERITED_REF = {"path": "CVF_SESSION/authority/CCMAI-RUNTIME-139-SOURCE-AND-CONDITIONAL-REENTRY-ADMISSION.json",
                     "sha256": "0ef818bfa4a60b9bd1b01335996989ca4c00d63f11141d9074dd3c26befb5b4f"}
CUA_OBSERVATION_REF = {"path": "CVF_SESSION/authority/CCMAI-RUNTIME-139-CUA-OBSERVATION-CAPS-ADMISSION.json",
                       "sha256": "fa7f2f344c75b6900f1fb65d74c9327e9e53d1957714226d880fa5a30eab9491"}


def verify_cua_grant_chain(value: dict) -> None:
    require(value.get("ownerCuaAmendmentRef") == CUA_AMENDMENT_REF and
            value.get("inheritedOwnerEnvelopeRef") == CUA_INHERITED_REF,
            "CUA_OWNER_GRANT_REFS_INVALID")
    amendment = read_json(committed(CUA_AMENDMENT_REF["path"]))
    inherited = read_json(committed(CUA_INHERITED_REF["path"]))
    require(digest(project_path(CUA_AMENDMENT_REF["path"])) == CUA_AMENDMENT_REF["sha256"] and
            digest(project_path(CUA_INHERITED_REF["path"])) == CUA_INHERITED_REF["sha256"] and
            amendment.get("ownerApproved") is True and amendment.get("sourceGranted") is True and
            amendment.get("previousOwnerEnvelope") == CUA_INHERITED_REF and
            amendment.get("browserContractProposed", {}).get("proofContract") == "cua_page_target_v1" and
            inherited.get("ownerApproved") is True and inherited.get("freshCampaignCap") == 1 and
            inherited.get("providerPhysicalCap") == 1, "CUA_OWNER_GRANT_CHAIN_INVALID")
    obs_ref = value.get("observationCapsRef")
    require(obs_ref == CUA_OBSERVATION_REF, "CUA_OBSERVATION_AUTHORITY_MISSING")
    obs = read_json(committed(obs_ref["path"]))
    caps = obs.get("proposedReadonlyChanges", {})
    require(digest(project_path(obs_ref["path"])) == obs_ref["sha256"] and
            obs.get("ownerApproved") is True and
            obs.get("kind") == "OWNER_APPROVED_R139_CUA_OBSERVATION_CAP_AMENDMENT" and
            obs.get("parentCuaAmendment") == CUA_AMENDMENT_REF and
            obs.get("inheritedOwnerEnvelope") == CUA_INHERITED_REF and
            caps.get("globalMetadataEnumerationMaximum") == 3 and
            caps.get("baselineMetadataEnumeration") == 1 and
            caps.get("handoffMetadataEnumeration") == 1 and
            caps.get("finalMetadataEnumeration") == 1 and
            caps.get("g2StatusReadStartsMaximum") == 16 and
            caps.get("automaticPostSendStatusPollStartsMaximum") == 12 and
            caps.get("automaticPollSpacingSecondsMinimum") == 5 and
            caps.get("automaticPollLifetimeSecondsMaximum") == 60 and
            caps.get("eventDrivenStatusReadsMaximum") == 4 and
            caps.get("eventDrivenCategories") == ["initial_writer_load", "publication_version_watch_reload",
                                                   "one_deliberate_writer_reload", "reader_load"] and
            caps.get("spacingAndLifetimeScope") == "automatic post-send polling only; event-driven loads recorded separately" and
            caps.get("noExtraGlobalInventory") is True and
            caps.get("noUnrelatedTabContentCaptureOrControl") is True and
            caps.get("rawMetadataPrivateEvidenceIdsOrDigestsOnly") is True,
            "CUA_OBSERVATION_AUTHORITY_NOT_APPROVED")


def verify_source_pins(pins: object) -> None:
    require(isinstance(pins, dict) and SOURCE_PIN_REQUIRED.issubset(pins), "SOURCE_PIN_SET_INCOMPLETE")
    for relative_path, expected in pins.items():
        require(isinstance(expected, str) and HEX64.fullmatch(expected) and digest(project_path(relative_path)) == expected, "SOURCE_PIN_MISMATCH")


def fresh_runtime_admission(relative: str, *, expect_fresh_external: bool = True, validate_live_deadline: bool = True) -> tuple[dict, Path]:
    # Only a new immutable R139 effect admission can enable `run`.
    require(bool(re.fullmatch(r"CVF_SESSION/authority/CCMAI-RUNTIME-139-[A-Za-z0-9_-]+-ADMISSION\.json", relative)), "RUNTIME_ADMISSION_PATH_INVALID")
    path = committed(relative)
    value = read_json(path)
    require(value.get("trancheId") == "CCMAI-RUNTIME-139" and value.get("kind") == "OWNER_APPROVED_RUNTIME_EFFECT_ADMISSION", "RUNTIME_ADMISSION_KIND_INVALID")
    require(value.get("ownerApproved") is True and value.get("sourceOnly") is not True, "OWNER_EFFECT_APPROVAL_MISSING")
    require(value.get("parentAuthorityCommit") == "642f5bd026dfe31e6dda6441dd25704eb3ececa3", "PARENT_SEED_MISMATCH")
    require(value.get("sourceResumeAdmissionCommit") == "6fc951a84c54cd31f20f2404d6dc8957d97bc339", "SOURCE_RESUME_BINDING_MISMATCH")
    if value.get("browserTool") == "supported_CUA_page_target":
        verify_cua_grant_chain(value)
    source = value.get("sourceCommit")
    require(isinstance(source, str) and HEX40.fullmatch(source), "SOURCE_COMMIT_INVALID")
    ancestor = subprocess.run(["git", "merge-base", "--is-ancestor", source, "HEAD"], cwd=ROOT, capture_output=True, timeout=10)
    require(ancestor.returncode == 0, "SOURCE_COMMIT_NOT_ANCESTOR")
    dirty = subprocess.run(["git", "status", "--porcelain", "--untracked-files=all"], cwd=ROOT, capture_output=True, timeout=10)
    require(dirty.returncode == 0 and not dirty.stdout.strip(), "SOURCE_WORKTREE_DIRTY")
    require(value.get("nativeExecutePOST") == EXPECTED_NATIVE and value.get("browserExecutePOST") == EXPECTED_BROWSER and value.get("grantCap") == EXPECTED_GRANTS and value.get("providerCap") == 1 and value.get("automaticRetry") == 0, "EFFECT_CAPS_INVALID")
    verify_browser_receipt(value, require_runtime_absent=expect_fresh_external)
    if expect_fresh_external and value.get("browserTool") in ("antigravity_native", "antigravity_page_target", "supported_CUA_page_target"):
        close_path_rel = value.get("browserCloseReceiptPath")
        if close_path_rel:
            close_p = ROOT / close_path_rel
            require(not close_p.exists(), "FUTURE_CLOSE_RECEIPT_PREEXISTING")
    require(value.get("image") == IMAGE, "IMAGE_MISMATCH")
    require(value.get("fixtureName") == R139_REPLACEMENT_FIXTURE_NAME, "FIXTURE_NAME_INVALID")
    require(value.get("database") == R139_REPLACEMENT_DATABASE, "DATABASE_NAME_INVALID")
    require(value.get("dockerHost") == "npipe:////./pipe/dockerDesktopLinuxEngine", "DOCKER_HOST_INVALID")
    require(value.get("deadlineSeconds") == MAX_SECONDS, "LIFECYCLE_LIMIT_INVALID")
    verify_source_pins(value.get("sourceSha256"))
    binary = project_path(value.get("goTestBinaryPath", ""))
    require(digest(binary) == value.get("goTestBinarySha256"), "GO_BINARY_PIN_MISMATCH")
    for field in ("dockerBinary", "nodeBinary", "powershellBinary"):
        binary_path = Path(value.get(field, ""))
        require(binary_path.is_absolute() and binary_path.is_file() and digest(binary_path) == value.get(field + "Sha256"), field.upper() + "_PIN_MISMATCH")
    runtime_root_value = value.get("runtimeRoot")
    require(isinstance(runtime_root_value, str) and runtime_root_value, "RUNTIME_ROOT_INVALID")
    runtime_root = Path(runtime_root_value)
    expected_runtime_root = (ROOT / R139_REPLACEMENT_RUNTIME_ROOT).resolve()
    require(runtime_root.is_absolute() and runtime_root == expected_runtime_root and runtime_root.resolve() == expected_runtime_root, "RUNTIME_ROOT_INVALID")
    require(runtime_root.resolve().is_relative_to((ROOT / ".cvf/runtime").resolve()), "RUNTIME_ROOT_OUTSIDE_IGNORED_AREA")
    external = replacement_external_evidence_dir()
    require(value.get("externalEvidenceDir") == str(external), "EXTERNAL_EVIDENCE_PATH_INVALID")
    if expect_fresh_external:
        require(not external.exists(), "EXTERNAL_EVIDENCE_NOT_FRESH")
    else:
        require(external.is_dir() and external.resolve() == external, "EXTERNAL_EVIDENCE_CHANGED")
    require(value.get("keyCustodian") and value.get("budgetAuthorized") is True and value.get("syntheticIdentityApproved") is True, "PROVIDER_PREREQUISITE_MISSING")
    require(value.get("provider") == "alibaba_maas" and value.get("model") == "qwen3.7-flash" and value.get("completionTokens") == 768 and value.get("maxRequestBytes") == 6000 and value.get("thinking") is False, "PROVIDER_LIMITS_INVALID")
    require(isinstance(value.get("grantDeadlineUTC"), str), "GRANT_DEADLINE_UTC_MISSING")
    if validate_live_deadline:
        parse_grant_deadline_utc(value.get("grantDeadlineUTC"))
    budget = value.get("approvedBudget")
    require(isinstance(budget, dict) and isinstance(budget.get("amount"), (int, float)) and budget["amount"] > 0 and isinstance(budget.get("currency"), str) and len(budget["currency"]) == 3 and value.get("ownerApprovalReference"), "OWNER_BUDGET_APPROVAL_MISSING")
    namespace = value.get("syntheticNamespace")
    require(isinstance(namespace, str) and re.fullmatch(r"ccma_r139_[a-z0-9_-]{1,64}", namespace) is not None and value.get("issuerRole") == "owner" and value.get("actorRole") == "writer" and value.get("configRef") == "r139-owned-config" and value.get("promptVersion") == "r139-synthetic-v1" and value.get("promptSha256") == PROMPT_SHA256 and value.get("grantDeadlineUTC"), "SYNTHETIC_ISSUER_BINDING_MISSING")
    endpoint = value.get("endpoint", "")
    require(isinstance(endpoint, str) and hashlib.sha256(endpoint.encode("utf-8")).hexdigest() == "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f", "PROVIDER_ENDPOINT_MISMATCH")
    require(isinstance(value.get("vitePort"), int) and 1024 <= value["vitePort"] <= 65535, "VITE_PORT_INVALID")
    vite_cli = project_path("frontend/node_modules/vite/bin/vite.js")
    require(digest(vite_cli) == value.get("viteCliSha256"), "VITE_CLI_PIN_MISMATCH")
    key_path = Path(value.get("keyPath", ""))
    require(key_path.is_absolute() and key_path.is_file() and not key_path.resolve().is_relative_to(ROOT.resolve()) and key_path.stat().st_size <= 8192, "KEY_CUSTODY_PATH_INVALID")
    for tree in ("backend", "frontend"):
        original = subprocess.run(["git", "rev-parse", source + ":" + tree], cwd=ROOT, capture_output=True, text=True, timeout=10)
        current = subprocess.run(["git", "rev-parse", "HEAD:" + tree], cwd=ROOT, capture_output=True, text=True, timeout=10)
        require(original.returncode == 0 and current.returncode == 0 and original.stdout == current.stdout and original.stdout.strip() == value.get(tree + "Tree"), "PRODUCT_TREE_CHANGED")
    return value, path


def docker(admission: dict, runtime: Path, *args: str, timeout: int = 20) -> str:
    global COMMAND_SEQUENCE
    cleanup = ACTIVE_CLEANUP_DEADLINE is not None
    deadline = ACTIVE_CLEANUP_DEADLINE if cleanup else (
        None if ACTIVE_HARD_DEADLINE is None else ACTIVE_HARD_DEADLINE - MIN_CLEANUP_SECONDS
    )
    if deadline is not None:
        remaining = deadline - time.monotonic()
        require(remaining > 0.05, "CLEANUP_DEADLINE_EXPIRED" if cleanup else "GLOBAL_LIFECYCLE_DEADLINE")
        timeout = min(timeout, max(0.05, remaining))
    binary = Path(admission["dockerBinary"])
    require(digest(binary) == admission["dockerBinarySha256"], "DOCKER_BINARY_CHANGED")
    config = runtime / "docker_config"
    if not config.exists():
        try:
            config.mkdir(parents=True, exist_ok=True)
            if not (config / "config.json").exists():
                write_new(config / "config.json", {})
        except Exception as exc:
            if not cleanup:
                raise
            CLEANUP_ERRORS.append("docker_config:" + type(exc).__name__)
    argv = [str(binary), "--host", admission["dockerHost"], "--config", str(config), *args]
    COMMAND_SEQUENCE += 1
    number = COMMAND_SEQUENCE
    command_dir = runtime / ("cleanup_commands" if cleanup else "commands")
    try:
        write_new(command_dir / f"{number:04d}.json", {"argv": argv, "at": time.time(), "timeout": timeout, "cleanup": cleanup})
    except Exception as exc:
        if not cleanup:
            raise
        CLEANUP_ERRORS.append("journal_intent:" + type(exc).__name__)
    try:
        result = subprocess.run(argv, cwd=ROOT, env=minimal_environment(), capture_output=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        if cleanup:
            CLEANUP_ERRORS.append("docker_timeout")
            raise Hold("DOCKER_CLEANUP_TIMEOUT") from None
        append(runtime / "errors.jsonl", {"command": number, "code": "DOCKER_TIMEOUT_NO_RETRY"})
        raise Hold("DOCKER_TIMEOUT_NO_RETRY") from None
    try:
        (command_dir / f"{number:04d}.stdout").write_bytes(result.stdout)
        (command_dir / f"{number:04d}.stderr").write_bytes(result.stderr)
        write_new(command_dir / f"{number:04d}.exit.json", {"code": result.returncode})
    except Exception as exc:
        if not cleanup:
            raise
        CLEANUP_ERRORS.append("journal_result:" + type(exc).__name__)
    if cleanup and result.returncode != 0:
        CLEANUP_ERRORS.append("docker_exit:" + str(result.returncode))
    require(result.returncode == 0, "DOCKER_COMMAND_FAILED")
    return result.stdout.decode("utf-8", errors="strict").strip()


def inventory(admission: dict, runtime: Path) -> dict:
    def rows(*args: str) -> list[dict]:
        raw = docker(admission, runtime, *args)
        return sorted((json.loads(line) for line in raw.splitlines() if line), key=lambda row: json.dumps(row, sort_keys=True))
    observed = {"containers": rows("ps", "-a", "--no-trunc", "--format", "{{json .}}"), "networks": rows("network", "ls", "--no-trunc", "--format", "{{json .}}"), "volumes": rows("volume", "ls", "--format", "{{json .}}")}
    # Inspect only explicitly selected safe fields. Never request Config.Env,
    # commands, labels, mounts or health probe logs from protected containers.
    projection = ('{"Id":{{json .Id}},"Name":{{json .Name}},"Image":{{json .Image}},"Created":{{json .Created}},'
                  '"RestartCount":{{json .RestartCount}},"State":{"Status":{{json .State.Status}},"Running":{{json .State.Running}},'
                  '"Paused":{{json .State.Paused}},"Restarting":{{json .State.Restarting}},"OOMKilled":{{json .State.OOMKilled}},'
                  '"Dead":{{json .State.Dead}},"Pid":{{json .State.Pid}},"ExitCode":{{json .State.ExitCode}},'
                  '"StartedAt":{{json .State.StartedAt}},"FinishedAt":{{json .State.FinishedAt}}},'
                  '"Health":{{if .State.Health}}{"Status":{{json .State.Health.Status}},"FailingStreak":{{json .State.Health.FailingStreak}}}{{else}}null{{end}},'
                  '"NetworkIdentity":{{json .NetworkSettings.Networks}}}')
    ids = [row.get("ID") for row in observed["containers"]]
    require(all(isinstance(item, str) and HEX64.fullmatch(item) for item in ids) and len(set(ids)) == len(ids), "PROTECTED_INSPECT_ID_INVALID")
    indexed = {}
    for offset in range(0, len(ids), 16):
        chunk = ids[offset:offset + 16]
        raw = docker(admission, runtime, "inspect", "--format", projection, *chunk)
        for line in raw.splitlines():
            item = json.loads(line)
            require(isinstance(item, dict) and item.get("Id") in chunk and item["Id"] not in indexed, "PROTECTED_INSPECT_ID_MISMATCH")
            indexed[item["Id"]] = item
    require(set(indexed) == set(ids), "PROTECTED_INSPECT_SET_INVALID")
    for row in observed["containers"]:
        row["_typed"] = indexed[row["ID"]]
    for row in observed["networks"]:
        require(isinstance(row.get("ID"), str) and row["ID"] and isinstance(row.get("Name"), str) and row["Name"], "PROTECTED_NETWORK_ID_INVALID")
        row["_typed"] = {field: row.get(field) for field in ("Name", "Driver", "Scope")}
        row["_typed"]["Id"] = row["ID"]
    for row in observed["volumes"]:
        require(isinstance(row.get("Name"), str) and row["Name"], "PROTECTED_VOLUME_ID_INVALID")
        row["_typed"] = {field: row.get(field) for field in ("Name", "Driver", "Scope")}
    observed["capturedMonotonic"] = time.monotonic()
    observed["capturedAtUTC"] = datetime.now(timezone.utc).isoformat()
    return observed


def excluded_owned(observed: dict, owner: dict) -> dict:
    return {"containers": [row for row in observed["containers"] if row.get("ID") != owner.get("containerId")], "networks": [row for row in observed["networks"] if row.get("ID") != owner.get("networkId")], "volumes": [row for row in observed["volumes"] if row.get("Name") != owner.get("volumeName")], "capturedMonotonic": observed["capturedMonotonic"], "capturedAtUTC": observed["capturedAtUTC"]}


_STATUS_ELAPSED = r"[1-9][0-9]* (?:second|minute|hour|day|week|month|year)s?"
_UP_STATUS = re.compile(rf"^Up (?P<elapsed>{_STATUS_ELAPSED})(?P<health> \((?:healthy|unhealthy|health: starting)\))?$")
_EXITED_STATUS = re.compile(rf"^Exited \((?P<exitcode>[0-9]+)\) (?P<elapsed>{_STATUS_ELAPSED}) ago$")
_AUTH_POST_PATHS = {"/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/refresh"}
_EXECUTE_POST_PATH = re.compile(r"^/api/v1/tenants/[^/]+/intervention-cases/[^/]+/rerun-requests/[^/]+/execute$")
_ELAPSED_VALUE = re.compile(r"^(?P<count>[1-9][0-9]*) (?P<unit>second|minute|hour|day|week|month|year)s?(?: ago)?$")
_UNIT_SECONDS = {"second": 1, "minute": 60, "hour": 3600, "day": 86400, "week": 604800, "month": 2592000, "year": 31536000}


def _anchored_elapsed(display: str, anchor: str, capture: datetime) -> None:
    match = _ELAPSED_VALUE.fullmatch(display)
    require(match is not None, "PROTECTED_ELAPSED_UNKNOWN_FORMAT")
    try:
        anchored = datetime.fromisoformat(anchor.replace("Z", "+00:00"))
    except ValueError as exc:
        raise Hold("PROTECTED_TIME_ANCHOR_INVALID") from exc
    require(anchored.tzinfo is not None, "PROTECTED_TIME_ANCHOR_INVALID")
    seconds = (capture - anchored).total_seconds()
    unit = _UNIT_SECONDS[match.group("unit")]
    count = int(match.group("count"))
    # Docker rounds display units; tolerate one display bucket, never a reset.
    require(max(0, (count - 1) * unit) <= seconds <= (count + 2) * unit, "PROTECTED_ELAPSED_ANCHOR_MISMATCH")


def _route_kind(method: object, path: object) -> str:
    require(isinstance(method, str) and isinstance(path, str), "ROUTE_IDENTITY_INVALID")
    if method == "POST" and path in _AUTH_POST_PATHS:
        return "auth"
    if method == "POST" and _EXECUTE_POST_PATH.fullmatch(path):
        return "execute"
    if method in ("GET", "HEAD") and (path == "/api" or path.startswith("/api/")):
        return "read"
    raise Hold("UNLISTED_ROUTE_ATTEMPTED")


def _validate_route_rows(rows: list, label: str, terminal: dict) -> dict:
    require(all(isinstance(row, dict) for row in rows), label + "_ROUTE_ROW_INVALID")
    attempts = [row for row in rows if row.get("phase") == "attempt"]
    responses = [row for row in rows if row.get("phase") == "response"]
    require(len(attempts) + len(responses) == len(rows) and len(attempts) == len(responses), label + "_ROUTE_RESPONSE_GAP")
    ids = [row.get("id") for row in attempts]
    response_ids = [row.get("id") for row in responses]
    require(all(type(i) is int and i > 0 for i in ids + response_ids) and len(ids) == len(set(ids)) and len(response_ids) == len(set(response_ids)) and set(ids) == set(response_ids), label + "_ROUTE_RESPONSE_GAP")
    by_id = {row["id"]: row for row in attempts}
    require(all(row.get("method") == by_id[row["id"]].get("method") and row.get("path") == by_id[row["id"]].get("path") and type(row.get("status")) is int and 100 <= row["status"] <= 599 for row in responses), label + "_ROUTE_RESPONSE_IDENTITY_MISMATCH")
    require(all(row.get("allowed") is True for row in attempts), label + "_BLOCKED_ROUTE_ATTEMPTED")
    require(all(row["status"] != 405 for row in responses), label + "_ROUTE_DENIAL_OBSERVED")
    kinds = [_route_kind(row.get("method"), row.get("path")) for row in attempts]
    auth = kinds.count("auth")
    execute = kinds.count("execute")
    require(execute == terminal.get("executePOST") == 1 and auth == terminal.get("authPOST") and auth <= 4, label + "_EFFECT_ROUTE_MISMATCH")
    return {"attempts": len(attempts), "responses": len(responses), "executePOST": execute, "authPOST": auth,
            "authRoutes": [row["path"] for row, kind in zip(attempts, kinds) if kind == "auth"]}


def canonicalize_container_status(status: str) -> str:
    require(isinstance(status, str), "PROTECTED_STATUS_UNKNOWN_FORMAT")
    up = _UP_STATUS.fullmatch(status)
    if up:
        return "Up <elapsed>" + (up.group("health") or "")
    exited = _EXITED_STATUS.fullmatch(status)
    if exited:
        return f"Exited ({exited.group('exitcode')}) <elapsed> ago"
    raise Hold("PROTECTED_STATUS_UNKNOWN_FORMAT")


def canonicalize_protected_inventory(value: dict) -> dict:
    require(isinstance(value, dict) and set(value) == {"containers", "networks", "volumes", "capturedMonotonic", "capturedAtUTC"} and type(value["capturedMonotonic"]) in (float, int) and value["capturedMonotonic"] >= 0, "PROTECTED_INVENTORY_INVALID")
    try:
        captured = datetime.fromisoformat(value["capturedAtUTC"].replace("Z", "+00:00"))
    except (ValueError, AttributeError, TypeError) as exc:
        raise Hold("PROTECTED_CAPTURE_TIME_INVALID") from exc
    require(captured.tzinfo is not None and captured.utcoffset() == timedelta(0), "PROTECTED_CAPTURE_TIME_INVALID")
    normalized: dict[str, list[dict]] = {}
    for collection in ("containers", "networks", "volumes"):
        rows = value[collection]
        require(isinstance(rows, list) and all(isinstance(row, dict) for row in rows), "PROTECTED_INVENTORY_INVALID")
        copied_rows = []
        for row in rows:
            copied = dict(row)
            networks = copied.get("Networks")
            if collection == "containers" and isinstance(networks, str) and "," in networks:
                copied["Networks"] = ",".join(sorted(member.strip() for member in networks.split(",")))
            if collection == "containers":
                typed = copied.get("_typed")
                require(isinstance(typed, dict) and typed.get("Id") == copied.get("ID") and isinstance(typed.get("Name"), str) and isinstance(typed.get("Image"), str) and isinstance(typed.get("Created"), str) and type(typed.get("RestartCount")) is int and typed["RestartCount"] >= 0, "PROTECTED_CONTAINER_ANCHOR_INVALID")
                state = typed.get("State")
                require(isinstance(state, dict) and all(field in state for field in ("Status", "Running", "Paused", "Restarting", "OOMKilled", "Dead", "Pid", "ExitCode", "StartedAt", "FinishedAt")) and isinstance(state["StartedAt"], str) and isinstance(state["FinishedAt"], str) and all(type(state[field]) is bool for field in ("Running", "Paused", "Restarting", "OOMKilled", "Dead")) and type(state["Pid"]) is int and state["Pid"] >= 0 and type(state["ExitCode"]) is int and isinstance(typed.get("NetworkIdentity"), dict), "PROTECTED_STATE_ANCHOR_INVALID")
                for field in ("StartedAt", "FinishedAt"):
                    try:
                        require(datetime.fromisoformat(state[field].replace("Z", "+00:00")).tzinfo is not None, "PROTECTED_TIME_ANCHOR_INVALID")
                    except ValueError as exc:
                        raise Hold("PROTECTED_TIME_ANCHOR_INVALID") from exc
                health = typed.get("Health")
                require(health is None or (isinstance(health, dict) and isinstance(health.get("Status"), str) and type(health.get("FailingStreak")) is int and health["FailingStreak"] >= 0), "PROTECTED_HEALTH_ANCHOR_INVALID")
                original_status = copied.get("Status")
                copied["Status"] = canonicalize_container_status(original_status)
                expected_state = "running" if original_status.startswith("Up ") else "exited"
                require(copied.get("State") == expected_state and state["Status"] == expected_state and state["Running"] is (expected_state == "running"), "PROTECTED_STATUS_STATE_MISMATCH")
                status_health = _UP_STATUS.fullmatch(original_status).group("health") if expected_state == "running" else None
                if status_health:
                    display_health = status_health.strip(" ()").removeprefix("health: ")
                    require(isinstance(health, dict) and display_health == health.get("Status"), "PROTECTED_HEALTH_MISMATCH")
                running_for = copied.get("RunningFor")
                require(isinstance(running_for, str), "PROTECTED_RUNNING_FOR_UNKNOWN_FORMAT")
                status_elapsed = _UP_STATUS.fullmatch(original_status).group("elapsed") if expected_state == "running" else _EXITED_STATUS.fullmatch(original_status).group("elapsed")
                _anchored_elapsed(status_elapsed, state["StartedAt"] if expected_state == "running" else state["FinishedAt"], captured)
                _anchored_elapsed(running_for, typed["Created"], captured)
                copied["RunningFor"] = "<elapsed>"
            else:
                typed = copied.get("_typed")
                identity = "Id" if collection == "networks" else "Name"
                listing = "ID" if collection == "networks" else "Name"
                require(isinstance(typed, dict) and typed.get(identity) == copied.get(listing) and isinstance(typed.get("Name"), str) and typed["Name"] and isinstance(typed.get("Driver"), str) and typed["Driver"] and isinstance(typed.get("Scope"), str) and typed["Scope"], "PROTECTED_RESOURCE_ANCHOR_INVALID")
            copied_rows.append(copied)
        normalized[collection] = sorted(copied_rows, key=lambda row: json.dumps(row, sort_keys=True))
    return normalized


def verify_owned_resources(admission: dict, runtime: Path, owner: dict) -> dict:
    name, nonce = owner["fixtureName"], owner["nonce"]
    found = {}
    if owner.get("networkId"):
        rows = json.loads(docker(admission, runtime, "network", "inspect", owner["networkId"]))
        require(len(rows) == 1 and rows[0].get("Id") == owner["networkId"] and rows[0].get("Name") == name and rows[0].get("Labels", {}).get("ccmai.owner") == nonce and rows[0].get("Labels", {}).get("ccmai.tranche") == "CCMAI-RUNTIME-139", "NETWORK_OWNERSHIP_UNPROVEN")
        found["networkId"] = owner["networkId"]
    if owner.get("volumeName"):
        rows = json.loads(docker(admission, runtime, "volume", "inspect", owner["volumeName"]))
        require(len(rows) == 1 and rows[0].get("Name") == name and rows[0].get("Labels", {}).get("ccmai.owner") == nonce and rows[0].get("Labels", {}).get("ccmai.tranche") == "CCMAI-RUNTIME-139", "VOLUME_OWNERSHIP_UNPROVEN")
        found["volumeName"] = owner["volumeName"]
    return found


def owned_container(admission: dict, runtime: Path, owner: dict) -> dict:
    raw = docker(admission, runtime, "inspect", owner["containerId"])
    rows = json.loads(raw)
    require(isinstance(rows, list) and len(rows) == 1, "OWNED_CONTAINER_INSPECT_INVALID")
    row = rows[0]
    labels = row.get("Config", {}).get("Labels", {})
    require(row.get("Id") == owner["containerId"] and row.get("Name") == "/" + admission["fixtureName"] and labels.get("ccmai.tranche") == "CCMAI-RUNTIME-139" and labels.get("ccmai.owner") == owner["nonce"], "CONTAINER_OWNERSHIP_UNPROVEN")
    require(row.get("Image") == IMAGE, "CONTAINER_IMAGE_CHANGED")
    mounts = row.get("Mounts", [])
    require(any(m.get("Type") == "volume" and m.get("Name") == owner.get("volumeName") and m.get("Destination") == "/var/lib/mysql" for m in mounts), "CONTAINER_MOUNT_MISMATCH")
    require(row.get("HostConfig", {}).get("NetworkMode") == owner.get("fixtureName"), "CONTAINER_NETWORK_MISMATCH")
    return row


def recover_partial_owner(admission: dict, runtime: Path, owner: dict) -> dict:
    # The named resources are inspected, never inferred solely from names.
    if not owner.get("nonce"):
        return owner
    name = owner["fixtureName"]
    for kind, key, id_field in (("network", "networkId", "Id"), ("volume", "volumeName", "Name"), ("container", "containerId", "Id")):
        if owner.get(key):
            continue
        command = ("inspect", name) if kind == "container" else (kind, "inspect", name)
        try:
            rows = json.loads(docker(admission, runtime, *command))
        except Exception as exc:
            if ACTIVE_CLEANUP_DEADLINE is not None:
                CLEANUP_ERRORS.append("owner_recovery:" + type(exc).__name__)
            continue
        if not isinstance(rows, list) or len(rows) != 1:
            continue
        row = rows[0]
        labels = row.get("Config", {}).get("Labels", {}) if kind == "container" else row.get("Labels", {})
        if labels.get("ccmai.owner") == owner["nonce"] and labels.get("ccmai.tranche") == "CCMAI-RUNTIME-139" and (row.get("Name") in (name, "/" + name)):
            owner[key] = row.get(id_field)
    return owner


def wait_mysql_tcp_ready(admission: dict, runtime: Path, owner: dict) -> dict:
    started = time.monotonic()
    deadline = started + MYSQL_STARTUP_WINDOW_SECONDS
    if ACTIVE_HARD_DEADLINE is not None:
        deadline = min(deadline, ACTIVE_HARD_DEADLINE - MIN_CLEANUP_SECONDS)
    require(deadline - started > 0.05, "MYSQL_READY_WINDOW_UNAVAILABLE")
    probes = 0
    for _ in range(MYSQL_MAX_PROBES):
        now = time.monotonic()
        if now >= deadline:
            break
        timeout = min(MYSQL_PROBE_TIMEOUT_SECONDS, deadline - now)
        if timeout <= 0.05:
            break
        probes += 1
        try:
            result = docker(
                admission,
                runtime,
                "exec",
                "-e",
                "MYSQL_PWD=" + owner["fixturePassword"],
                owner["containerId"],
                "mysql",
                "--protocol=tcp",
                "--host=127.0.0.1",
                "--port=3306",
                "--connect-timeout=3",
                "-uroot",
                "-D",
                admission["database"],
                "-N",
                "-B",
                "-e",
                "SELECT 1",
                timeout=timeout,
            )
        except Hold as exc:
            if str(exc) == "DOCKER_TIMEOUT_NO_RETRY":
                raise Hold("MYSQL_TCP_READY_PROBE_TIMEOUT_NO_RETRY") from None
            if str(exc) != "DOCKER_COMMAND_FAILED":
                raise
        else:
            require(result == "1", "MYSQL_TCP_READY_RESULT_INVALID")
            require(time.monotonic() <= deadline, "MYSQL_TCP_READY_WINDOW_EXPIRED")
            return {
                "status": "PASS_TCP_SELECT_1",
                "database": admission["database"],
                "host": "127.0.0.1",
                "port": 3306,
                "probeCount": probes,
                "elapsedSeconds": round(time.monotonic() - started, 3),
            }
        remaining = deadline - time.monotonic()
        if probes >= MYSQL_MAX_PROBES or remaining <= 0:
            break
        time.sleep(min(MYSQL_PROBE_SPACING_SECONDS, remaining))
    raise Hold("MYSQL_TCP_READY_NOT_PROVEN")


def start_fixture(admission: dict, runtime: Path, external: Path, owner: dict, *, before: dict | None = None) -> dict:
    before = inventory(admission, runtime) if before is None else before
    canonicalize_protected_inventory(before)
    name = admission["fixtureName"]
    require(not any(row.get("Names") == name for row in before["containers"]), "CONTAINER_NAME_EXISTS")
    require(not any(row.get("Name") == name for row in before["networks"] + before["volumes"]), "RESOURCE_NAME_EXISTS")
    nonce = secrets.token_hex(16)
    owner.update({"nonce": nonce, "protectedBefore": before, "fixtureName": name, "database": admission["database"], "fixturePassword": "r139-synthetic-" + nonce})
    write_new(runtime / "fixture_intent.json", owner)
    labels = ["--label", "ccmai.tranche=CCMAI-RUNTIME-139", "--label", "ccmai.owner=" + nonce]
    owner["networkId"] = docker(admission, runtime, "network", "create", "--driver", "bridge", *labels, name)
    require(bool(HEX64.fullmatch(owner["networkId"])), "NETWORK_ID_INVALID")
    write_new(runtime / "network_created.json", owner)
    require(docker(admission, runtime, "volume", "create", *labels, name) == name, "VOLUME_NAME_MISMATCH")
    owner["volumeName"] = name
    write_new(runtime / "volume_created.json", owner)
    owner["containerId"] = docker(admission, runtime, "create", "--pull", "never", "--name", name, *labels, "--network", name, "--mount", "type=volume,src=" + name + ",dst=/var/lib/mysql", "-p", "127.0.0.1::3306", "-e", "MYSQL_ROOT_PASSWORD=" + owner["fixturePassword"], "-e", "MYSQL_DATABASE=" + admission["database"], IMAGE)
    require(bool(HEX64.fullmatch(owner["containerId"])), "CONTAINER_ID_INVALID")
    write_new(runtime / "container_created.json", owner)
    verify_owned_resources(admission, runtime, owner)
    owned_container(admission, runtime, owner)
    docker(admission, runtime, "start", owner["containerId"], timeout=30)
    row = owned_container(admission, runtime, owner)
    ports = row.get("NetworkSettings", {}).get("Ports", {}).get("3306/tcp") or []
    require(len(ports) == 1 and ports[0].get("HostIp") == "127.0.0.1" and str(ports[0].get("HostPort", "")).isdigit(), "OWNED_LOOPBACK_PORT_INVALID")
    owner["port"] = int(ports[0]["HostPort"])
    require(1024 <= owner["port"] <= 65535, "OWNED_PORT_INVALID")
    readiness = wait_mysql_tcp_ready(admission, runtime, owner)
    write_new(runtime / "mysql_ready.json", readiness)
    sql = "CREATE TABLE ccma_r139_fixture_owner (nonce CHAR(32) NOT NULL); INSERT INTO ccma_r139_fixture_owner (nonce) VALUES ('" + nonce + "');"
    docker(admission, runtime, "exec", "-e", "MYSQL_PWD=" + owner["fixturePassword"], owner["containerId"], "mysql", "--protocol=tcp", "--host=127.0.0.1", "--port=3306", "--connect-timeout=3", "-uroot", "-D", admission["database"], "-e", sql, timeout=8)
    write_new(external / "fixture_identity.json", {"trancheId": "CCMAI-RUNTIME-139", "host": "127.0.0.1", "port": owner["port"], "database": admission["database"], "name": name, "nonce": nonce})
    write_new(runtime / "fixture_started.json", owner)
    return owner


def launch(admission: dict, runtime: Path, name: str, argv: list[str], env: dict, children: dict) -> subprocess.Popen:
    cwd = ROOT / "backend/api/handlers" if name == "go" else ROOT
    write_new(runtime / f"{name}_intent.json", {"argv": argv, "cwd": str(cwd), "at": time.time()})
    stream = (runtime / f"{name}.log").open("xb")
    try:
        process = subprocess.Popen(argv, cwd=cwd, env=env, stdout=stream, stderr=subprocess.STDOUT, creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
        children[name] = process  # publish ownership before fallible stream close/receipts
    finally:
        stream.close()
    try:
        identity = process_creation_identity(process)
        write_new(runtime / f"{name}_process.json", {"pid": process.pid, "creationIdentity": identity, "at": time.time()})
    except Exception:
        try:
            stop_child(runtime, name, process)
        finally:
            raise
    return process


def process_creation_identity(process: subprocess.Popen) -> str:
    if os.name == "nt":
        import ctypes
        from ctypes import wintypes
        kernel = ctypes.WinDLL("kernel32", use_last_error=True)
        creation = wintypes.FILETIME()
        exit_time = wintypes.FILETIME()
        get_times = kernel.GetProcessTimes
        get_times.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.FILETIME), ctypes.POINTER(wintypes.FILETIME), ctypes.POINTER(wintypes.FILETIME), ctypes.POINTER(wintypes.FILETIME)]
        get_times.restype = wintypes.BOOL
        require(bool(get_times(wintypes.HANDLE(process._handle), ctypes.byref(creation), ctypes.byref(exit_time), ctypes.byref(wintypes.FILETIME()), ctypes.byref(wintypes.FILETIME()))), "PROCESS_CREATION_ID_UNAVAILABLE")
        return f"windows-filetime-{(creation.dwHighDateTime << 32) | creation.dwLowDateTime:016x}"
    stat = Path(f"/proc/{process.pid}/stat")
    require(stat.is_file(), "PROCESS_CREATION_ID_UNAVAILABLE")
    fields = stat.read_text(encoding="ascii").rsplit(")", 1)[-1].split()
    require(len(fields) > 19, "PROCESS_CREATION_ID_UNAVAILABLE")
    return f"proc-start-ticks-{fields[19]}"


def windows_listener_absent(port: int, admission: dict, runtime: Path, *, connect_code: int) -> dict:
    binary = Path(admission.get("powershellBinary", ""))
    expected_hash = admission.get("powershellBinarySha256")
    require(binary.is_absolute() and binary.is_file(), "POWERSHELL_BINARY_INVALID")
    require(isinstance(expected_hash, str) and HEX64.fullmatch(expected_hash) and digest(binary) == expected_hash, "POWERSHELL_BINARY_CHANGED")
    deadline = ACTIVE_CLEANUP_DEADLINE if ACTIVE_CLEANUP_DEADLINE is not None else ACTIVE_HARD_DEADLINE
    remaining = WINDOWS_LISTENER_QUERY_TIMEOUT_SECONDS if deadline is None else deadline - time.monotonic()
    require(remaining > 0.05, "WINDOWS_LISTENER_QUERY_DEADLINE_EXPIRED")
    timeout = min(WINDOWS_LISTENER_QUERY_TIMEOUT_SECONDS, remaining)
    script = (
        "$ErrorActionPreference='Stop';try{"
        "$found=@(Get-NetTCPConnection -State Listen -ErrorAction Stop | "
        "Where-Object {$_.LocalPort -eq " + str(port) + "} | "
        "ForEach-Object {[pscustomobject]@{port=[int]$_.LocalPort;state=[string]$_.State}});"
        "$receipt=[pscustomobject]@{queryOk=$true;port=" + str(port) + ";listeners=$found};"
        "$receipt | ConvertTo-Json -Compress -Depth 3;exit 0"
        "}catch{[Console]::Error.WriteLine('listener_inventory_query_failed');exit 2}"
    )
    argv = [str(binary), "-NoProfile", "-NonInteractive", "-Command", script]
    intent = runtime / f"listener_query_{port}_intent.json"
    result_path = runtime / f"listener_query_{port}_result.json"
    write_new(intent, {"argv": argv, "port": port, "timeoutSeconds": timeout, "powershellBinarySha256": expected_hash})
    try:
        result = subprocess.run(
            argv,
            cwd=ROOT,
            env=minimal_environment(),
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            timeout=timeout,
            creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0),
        )
    except subprocess.TimeoutExpired as exc:
        stdout = exc.stdout.decode("utf-8", errors="replace") if isinstance(exc.stdout, bytes) else (exc.stdout or "")
        stderr = exc.stderr.decode("utf-8", errors="replace") if isinstance(exc.stderr, bytes) else (exc.stderr or "")
        write_new(result_path, {"status": "TIMEOUT_NO_RETRY", "argv": argv, "port": port, "timeoutSeconds": timeout, "stdout": stdout, "stderr": stderr})
        raise Hold("WINDOWS_LISTENER_QUERY_TIMEOUT_NO_RETRY") from None
    except OSError as exc:
        write_new(result_path, {"status": "LAUNCH_FAILED", "argv": argv, "port": port, "timeoutSeconds": timeout, "errorType": type(exc).__name__})
        raise Hold("WINDOWS_LISTENER_QUERY_FAILED") from None

    raw_stdout = result.stdout.decode("utf-8", errors="replace") if isinstance(result.stdout, bytes) else (result.stdout or "")
    raw_stderr = result.stderr.decode("utf-8", errors="replace") if isinstance(result.stderr, bytes) else (result.stderr or "")
    payload = None
    parse_error = None
    try:
        payload = json.loads(raw_stdout)
    except (TypeError, json.JSONDecodeError):
        parse_error = "JSON_INVALID"
    valid_shape = (
        isinstance(payload, dict)
        and set(payload) == {"queryOk", "port", "listeners"}
        and payload.get("queryOk") is True
        and type(payload.get("port")) is int
        and payload.get("port") == port
        and isinstance(payload.get("listeners"), list)
    )
    status = "PASS_ZERO_LISTENERS" if result.returncode == 0 and not raw_stderr and valid_shape and payload["listeners"] == [] else "HOLD"
    write_new(
        result_path,
        {
            "status": status,
            "argv": argv,
            "port": port,
            "timeoutSeconds": timeout,
            "returnCode": result.returncode,
            "stdout": raw_stdout,
            "stderr": raw_stderr,
            "parseError": parse_error,
            "parsed": payload,
        },
    )
    require(result.returncode == 0, "WINDOWS_LISTENER_QUERY_FAILED")
    require(not raw_stderr, "WINDOWS_LISTENER_QUERY_STDERR")
    require(valid_shape, "WINDOWS_LISTENER_QUERY_INVALID")
    require(payload["listeners"] == [], "WINDOWS_LISTENER_PRESENT")
    return {
        "host": "127.0.0.1",
        "port": port,
        "connectEx": connect_code,
        "listenerAbsent": True,
        "evidenceSource": "pinned_powershell_tcp_listener_inventory",
        "powershellBinarySha256": expected_hash,
        "queryExitCode": result.returncode,
        "queryMatchedPort": payload["port"],
        "queryListenerCount": 0,
        "queryReceiptPath": str(result_path),
        "queryReceiptSha256": digest(result_path),
    }


def listener_absent(port: int | None, admission: dict | None = None, runtime: Path | None = None) -> dict:
    require(isinstance(port, int) and 1024 <= port <= 65535, "LISTENER_PORT_UNKNOWN")
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client:
        client.settimeout(2)
        code = client.connect_ex(("127.0.0.1", port))
    if code in (errno.ECONNREFUSED, 10061):
        return {"host": "127.0.0.1", "port": port, "connectEx": code, "listenerAbsent": True, "evidenceSource": "tcp_connect_refused"}
    require(code == 10035, "LISTENER_ABSENCE_UNPROVEN")
    require(isinstance(admission, dict) and isinstance(runtime, Path), "WINDOWS_LISTENER_QUERY_CONTEXT_MISSING")
    return windows_listener_absent(port, admission, runtime, connect_code=code)


def stop_child(runtime: Path, name: str, process: subprocess.Popen | None) -> dict:
    if process is None:
        return {"name": name, "started": False}
    if process.poll() is None:
        try:
            append(runtime / "effects.jsonl", {"kind": "terminate", "name": name, "pid": process.pid})
        except OSError:
            pass  # process termination must proceed even when the disk fails
        process.terminate()
        try:
            remaining = None if ACTIVE_CLEANUP_DEADLINE is None else max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
            if remaining is not None and remaining <= 0:
                raise subprocess.TimeoutExpired(process.args or name, 0)
            process.wait(timeout=12 if remaining is None else min(12, remaining))
        except subprocess.TimeoutExpired:
            try:
                append(runtime / "effects.jsonl", {"kind": "kill", "name": name, "pid": process.pid})
            except OSError:
                pass
            process.kill()
            remaining = None if ACTIVE_CLEANUP_DEADLINE is None else max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
            if remaining is not None and remaining <= 0:
                raise subprocess.TimeoutExpired(process.args or name, 0)
            process.wait(timeout=8 if remaining is None else min(8, remaining))
    else:
        remaining = None if ACTIVE_CLEANUP_DEADLINE is None else max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
        if remaining is not None and remaining <= 0:
            raise Hold("CHILD_WAIT_CLEANUP_DEADLINE")
        process.wait(timeout=1 if remaining is None else min(1, remaining))
    return {"name": name, "pid": process.pid, "exitCode": process.returncode, "waitObserved": True}


def stop_fixture(admission: dict, runtime: Path, owner: dict) -> dict:
    ownership_error: Exception | None = None
    try:
        verify_owned_resources(admission, runtime, owner)
    except Exception as exc:
        ownership_error = exc
    # Stop only the exact container whose label, image, mount and network are
    # independently inspected. A volume/network inspection failure cannot
    # leave a verified live container running.
    before = owned_container(admission, runtime, owner)
    if before.get("State", {}).get("Running") is True:
        docker(admission, runtime, "stop", "--time", "10", owner["containerId"], timeout=30)
    after = owned_container(admission, runtime, owner)
    state = after.get("State", {})
    require(state.get("Running") is False and state.get("Pid") == 0, "CONTAINER_NOT_TERMINAL")
    require(after.get("NetworkSettings", {}).get("Ports", {}).get("3306/tcp") in (None, []), "CONTAINER_PORT_STILL_PUBLISHED")
    protected_before_raw = owner["protectedBefore"]
    protected_after_raw = excluded_owned(inventory(admission, runtime), owner)
    require(protected_after_raw["capturedMonotonic"] >= protected_before_raw["capturedMonotonic"], "PROTECTED_CAPTURE_TIME_REVERSED")
    wall_elapsed = (datetime.fromisoformat(protected_after_raw["capturedAtUTC"]) - datetime.fromisoformat(protected_before_raw["capturedAtUTC"])).total_seconds()
    monotonic_elapsed = protected_after_raw["capturedMonotonic"] - protected_before_raw["capturedMonotonic"]
    require(wall_elapsed >= 0 and abs(wall_elapsed - monotonic_elapsed) <= 5, "PROTECTED_CAPTURE_CLOCK_DRIFT")
    protected_before = canonicalize_protected_inventory(protected_before_raw)
    protected_after = canonicalize_protected_inventory(protected_after_raw)
    comparison = {
        "kind": "PROTECTED_INVENTORY_RAW_AND_NETWORKS_ORDER_COMPARISON",
        "rawBefore": protected_before_raw,
        "rawAfter": protected_after_raw,
        "canonicalBefore": protected_before,
        "canonicalAfter": protected_after,
        "canonicalEqual": protected_before == protected_after,
    }
    comparison_path = runtime / "protected_inventory_comparison.json"
    write_new(comparison_path, comparison)
    require(comparison["canonicalEqual"], "PROTECTED_INVENTORY_CHANGED")
    if ownership_error is not None:
        raise Hold("RELATED_RESOURCE_OWNERSHIP_UNPROVEN") from ownership_error
    return {
        "containerId": owner["containerId"],
        "running": False,
        "pid": 0,
        "protectedEqual": True,
        "protectedComparisonPath": str(comparison_path),
        "protectedComparisonSha256": digest(comparison_path),
    }


def wait_go_ready(process: subprocess.Popen, log: Path, deadline: float) -> str:
    while time.monotonic() < deadline:
        require(process.poll() is None, "GO_CHILD_EXITED_BEFORE_READY")
        if log.exists():
            for line in log.read_text(encoding="utf-8", errors="replace").splitlines():
                match = READY.fullmatch(line)
                if match:
                    return match.group(1)
        time.sleep(0.2)
    raise Hold("GO_CHILD_READY_TIMEOUT")


def _validate_read_rows(rows: object, expected_names: set[str], *, ready_name: str | None = None) -> list[dict]:
    require(isinstance(rows, list) and all(isinstance(row, dict) for row in rows), "NATIVE_READ_ROWS_INVALID")
    names = [row.get("name") for row in rows]
    require(len(names) == len(set(names)) and set(names) == expected_names, "NATIVE_READ_MATRIX_INCOMPLETE")
    for row in rows:
        require(isinstance(row.get("status"), int) and isinstance(row.get("bodySha256"), str) and HEX64.fullmatch(row["bodySha256"]), "NATIVE_READ_RECEIPT_INVALID")
        before, after = row.get("durableBefore"), row.get("durableAfter")
        require(isinstance(before, str) and HEX64.fullmatch(before) and before == after, "NATIVE_READ_MUTATED_DURABLE_STATE")
        is_ready = row.get("name") == ready_name
        require(row.get("noStore") is True and row.get("deniedHandleAbsent") is (not is_ready), "NATIVE_READ_CACHE_OR_HANDLE_GUARD_FAILED")
        require(row.get("grantHandlePresent") is is_ready, "NATIVE_READ_GRANT_HANDLE_MISMATCH")
    return rows


def validate_native_evidence(runtime: Path, *, require_published: bool = False) -> dict:
    negative = read_json(runtime / "native_rejections.json")
    race = read_json(runtime / "native_precall_denied.json")
    cases = negative.get("cases")
    require(isinstance(cases, list) and all(isinstance(row, dict) for row in cases), "NATIVE_SCENARIO_SET_INCOMPLETE")
    names = [row.get("name") for row in cases]
    require(names == ["malformed-json", "unknown-field", "missing-version", "stale-version", "foreign-actor-grant", "expired-grant", "foreign-case"], "NATIVE_SCENARIO_SET_INCOMPLETE")
    require(negative.get("slotsUsed") == 7 and [row.get("slot") for row in cases] == list(range(1, 8)), "NATIVE_SLOT_SEQUENCE_INVALID")
    require([row.get("status") for row in cases] == [400, 400, 400, 409, 409, 409, 404], "NATIVE_REJECTION_STATUS_INVALID")
    expected_errors = ["invalid_request", "invalid_request", "invalid_request", "case_conflict", "case_conflict", "case_conflict", "not_found"]
    require([row.get("error") for row in cases] == expected_errors, "NATIVE_REJECTION_REASON_INVALID")
    for row in cases:
        require(isinstance(row.get("bodySha256"), str) and HEX64.fullmatch(row["bodySha256"]) and isinstance(row.get("durableBefore"), str) and HEX64.fullmatch(row["durableBefore"]) and row.get("durableBefore") == row.get("durableAfter"), "NATIVE_REJECTION_NO_WRITE_UNPROVEN")
    require(negative.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT" and negative.get("trapCalls") == 0 and negative.get("nativeExecutePOST") == 7 and negative.get("grantIssuances") == 1 and race.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT", "NATIVE_AI_CLAIM_INVALID")
    require(race.get("slot") == 8 and race.get("status") == 202 and race.get("acceptedState") == "CLAIMED" and race.get("terminal") == "DENIED" and race.get("trapCalls") == 0 and race.get("attempts") == 0 and race.get("budgetConsumed") == 0 and race.get("nativeExecutePOSTBeforeBrowser") == 8 and race.get("grantIssuancesBeforeBrowser") == 2 and isinstance(race.get("executionId"), str) and race.get("executionId") and race.get("acceptedExecutionId") == race.get("executionId") and isinstance(race.get("bodySha256"), str) and HEX64.fullmatch(race["bodySha256"]) and isinstance(race.get("durableBefore"), str) and HEX64.fullmatch(race["durableBefore"]) and isinstance(race.get("durableAfter"), str) and HEX64.fullmatch(race["durableAfter"]) and race["durableBefore"] != race["durableAfter"], "NATIVE_PRECALL_DENIAL_INCOMPLETE")
    negative_reads = read_json(runtime / "native_reads_negative.json")
    pre_positive_reads = read_json(runtime / "native_reads_positive_pre_browser.json")
    positive_reads = read_json(runtime / "native_reads_positive.json") if require_published else pre_positive_reads
    require(negative_reads.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT" and pre_positive_reads.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT" and positive_reads.get("kind") == "SOFTWARE_ONLY_NO_AI_OUTPUT", "NATIVE_READ_EVIDENCE_KIND_INVALID")
    expected_negative = {"grant-absent", "consumed-grant", "config-mismatch", "budget-exhausted", "case-version-changed", "writer-permission-revoked", "source-changed", "foreign-actor-grant", "expired-grant"}
    neg_rows = _validate_read_rows(negative_reads.get("reads"), expected_negative)
    states = {"grant-absent": "AWAITING_OPERATOR_AUTHORITY", "consumed-grant": "AWAITING_OPERATOR_AUTHORITY", "config-mismatch": "AWAITING_OPERATOR_AUTHORITY", "budget-exhausted": "STALE_OR_INELIGIBLE", "case-version-changed": "STALE_OR_INELIGIBLE", "source-changed": "STALE_OR_INELIGIBLE", "foreign-actor-grant": "AWAITING_OPERATOR_AUTHORITY", "expired-grant": "AWAITING_OPERATOR_AUTHORITY"}
    for row in neg_rows:
        name = row["name"]
        if name == "writer-permission-revoked":
            require(row.get("status") == 403 and row.get("error") == "permission_denied", "NATIVE_PERMISSION_READ_DENIAL_INVALID")
        else:
            require(row.get("status") == 200 and row.get("state") == states[name], "NATIVE_READINESS_STATE_INVALID")
    pre_positive_source_rows = pre_positive_reads.get("reads")
    require(isinstance(pre_positive_source_rows, list) and all(isinstance(row, dict) for row in pre_positive_source_rows), "NATIVE_READ_ROWS_INVALID")
    positive_source_rows = positive_reads.get("reads")
    require(isinstance(positive_source_rows, list) and all(isinstance(row, dict) for row in positive_source_rows), "NATIVE_READ_ROWS_INVALID")
    denied_names = {"readiness-denied-reader", "readiness-denied-nojobs"}
    base_positive = {"readiness-foreign-case", "writer-READY", "reader-status-no-row", "process-disabled-status", "process-disabled-readiness", "status-unreadable-503", "readiness-unreadable-503", *denied_names}
    published_positive = {"reader-published-status", "writer-published-status", "reader-history", "reader-open-detail"}
    expected_positive = base_positive | (published_positive if require_published else set())
    require(sum(row.get("name") in denied_names for row in positive_source_rows) == 2, "NATIVE_PERMISSION_READ_MATRIX_INCOMPLETE")
    pos_rows = _validate_read_rows(positive_reads.get("reads"), expected_positive, ready_name="writer-READY")
    if require_published:
        pre_names = [row.get("name") for row in pre_positive_source_rows]
        require(set(pre_names) == base_positive and len(pre_names) == len(set(pre_names)), "NATIVE_PRE_BROWSER_READ_MATRIX_INCOMPLETE")
        full_by_name = {row["name"]: row for row in pos_rows}
        pre_by_name = {row["name"]: row for row in pre_positive_source_rows}
        require(all(full_by_name[name] == row for name, row in pre_by_name.items()), "NATIVE_PRE_BROWSER_READ_RECEIPT_CHANGED")
    by_name = {row["name"]: row for row in pos_rows}
    require(all(by_name[name].get("status") == 403 and by_name[name].get("error") == "permission_denied" for name in denied_names), "NATIVE_PERMISSION_READ_DENIAL_INVALID")
    require(by_name["readiness-foreign-case"].get("status") == 404 and by_name["readiness-foreign-case"].get("error") == "not_found", "NATIVE_FOREIGN_READ_INVALID")
    require(by_name["writer-READY"].get("status") == 200 and by_name["writer-READY"].get("state") == "READY", "NATIVE_READY_READ_INVALID")
    for name in ("reader-status-no-row", "process-disabled-status"):
        require(by_name[name].get("status") == 200 and by_name[name].get("executionPresent") is False and by_name[name].get("phase") == "NOT_STARTED", "NATIVE_EMPTY_STATUS_READ_INVALID")
    require(by_name["process-disabled-readiness"].get("status") == 200 and by_name["process-disabled-readiness"].get("state") == "PROCESS_DISABLED", "NATIVE_DISABLED_READ_INVALID")
    for name, error in (("status-unreadable-503", "status_unavailable"), ("readiness-unreadable-503", "readiness_unavailable")):
        require(by_name[name].get("status") == 503 and by_name[name].get("error") == error, "NATIVE_UNREADABLE_READ_INVALID")
    if require_published:
        require(by_name["reader-published-status"].get("status") == 200 and by_name["reader-published-status"].get("phase") == "PUBLISHED" and by_name["reader-published-status"].get("proposalAvailable") is True and isinstance(by_name["reader-published-status"].get("publishedCaseVersion"), int), "NATIVE_PUBLISHED_READ_INVALID")
        require(by_name["writer-published-status"].get("status") == 200 and by_name["writer-published-status"].get("phase") == "PUBLISHED", "NATIVE_WRITER_PUBLISHED_READ_INVALID")
        require(by_name["reader-history"].get("status") == 200 and by_name["reader-open-detail"].get("status") == 200, "NATIVE_HISTORY_DETAIL_READ_INVALID")
    return {"nativeExecutePOST": race["nativeExecutePOSTBeforeBrowser"], "rejectionSlots": len(cases), "grantIssuancesBeforeBrowser": race["grantIssuancesBeforeBrowser"], "negativeReadCount": len(neg_rows), "positiveReadCount": len(pos_rows), "transportCalls": 0, "kind": "SOFTWARE_ONLY_NO_AI_OUTPUT"}


def validate_vite_ledger(runtime: Path, terminal: dict) -> dict:
    path = runtime / "vite_routes.jsonl"
    require(path.is_file(), "VITE_ROUTE_LEDGER_MISSING")
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line]
    return _validate_route_rows(rows, "VITE", terminal)


def validate_go_ledger(external: Path, terminal: dict) -> dict:
    path = external / "go_routes.jsonl"
    require(path.is_file(), "GO_ROUTE_LEDGER_MISSING")
    rows = [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line]
    return _validate_route_rows(rows, "GO", terminal)


def validate_go_terminal(external: Path, terminal: dict, admission: dict | None = None, native_validation: dict | None = None) -> dict:
    require(terminal.get("nativeExecutePOST") == 8 and terminal.get("grantIssuances") == 3 and terminal.get("executePOST") == 1 and terminal.get("providerPhysicalCalls") == 1, "GO_EFFECT_ACCOUNTING_INCOMPLETE")
    require(terminal.get("providerHTTPStatus") == 200 and terminal.get("budgetConsumed") == 1 and terminal.get("executionState") == "PUBLISHED" and terminal.get("caseWorkState") == "OPEN", "GO_PUBLICATION_STATE_INCOMPLETE")
    require(all(terminal.get(field) is True for field in ("originalRequestUnchanged", "originalSnapshotUnchanged", "lineageVerified", "readerWriterPublishedStatus", "readerHistoryAndOpenDetail", "executionPresent")), "GO_LINEAGE_OR_READ_INCOMPLETE")
    require(terminal.get("fingerprintError") is False and isinstance(terminal.get("durableFingerprint"), str) and HEX64.fullmatch(terminal["durableFingerprint"]), "GO_DURABLE_FINGERPRINT_INVALID")
    lineage_pairs = (("storedRequestId", "requestId"), ("executionRequestId", "requestId"), ("executionCaseId", "caseId"), ("executionGrantId", "grantId"), ("executionRunId", "jobRunId"), ("attemptRunId", "jobRunId"), ("receiptRunId", "jobRunId"), ("proposalRunId", "jobRunId"), ("executionSnapshotId", "snapshotId"), ("proposalSnapshotId", "snapshotId"), ("eventRequestId", "requestId"), ("eventProposalId", "proposalId"), ("attemptExecutionId", "executionId"))
    for field, target in lineage_pairs:
        require(isinstance(terminal.get(field), str) and terminal.get(field) and terminal[field] == terminal.get(target), "GO_LINEAGE_ID_MISMATCH")
    for field in ("caseId", "requestId", "grantId", "executionId", "jobRunId", "snapshotId", "proposalId", "eventId", "attemptId", "receiptId"):
        require(isinstance(terminal.get(field), str) and terminal[field], "GO_LINEAGE_ID_MISSING")
    require(terminal.get("storedRequestCaseId") == terminal.get("caseId") and terminal.get("proposalRevision") == terminal.get("eventCaseVersion"), "GO_LINEAGE_REVISION_MISMATCH")
    require(isinstance(terminal.get("snapshotDigest"), str) and HEX64.fullmatch(terminal["snapshotDigest"]), "GO_SNAPSHOT_DIGEST_INVALID")
    if admission is not None:
        admitted = {"provider": admission.get("provider"), "model": admission.get("model"), "configRef": admission.get("configRef"), "promptVersion": admission.get("promptVersion"), "promptSha256": admission.get("promptSha256"), "syntheticNamespace": admission.get("syntheticNamespace"), "issuerRole": admission.get("issuerRole"), "actorRole": admission.get("actorRole"), "grantDeadlineUTC": admission.get("grantDeadlineUTC"), "completionTokens": admission.get("completionTokens"), "maxRequestBytes": admission.get("maxRequestBytes"), "thinkingDisabled": admission.get("thinking") is False}
        require(all(terminal.get(field) == expected for field, expected in admitted.items()), "GO_TERMINAL_ADMISSION_MISMATCH")
        require(terminal.get("responseModel") == admission.get("model"), "GO_RESPONSE_MODEL_MISMATCH")
    if native_validation is not None:
        require(terminal.get("nativeExecutePOST") == native_validation.get("nativeExecutePOST") == 8 and terminal.get("nativeNegativeReadCount") == native_validation.get("negativeReadCount") and terminal.get("nativePositiveReadCount") == native_validation.get("positiveReadCount") and terminal.get("grantIssuances") == native_validation.get("grantIssuancesBeforeBrowser") + 1 == 3 and native_validation.get("transportCalls") == 0 and native_validation.get("rejectionSlots") == 7, "GO_NATIVE_MATRIX_COUNTER_MISMATCH")
    wire = read_json(external / "provider_wire_private.json")
    require(wire.get("provider") == "alibaba_maas" and wire.get("model") == "qwen3.7-flash" and wire.get("physicalCalls") == 1 and wire.get("httpStatus") == 200, "GO_PROVIDER_WIRE_INCOMPLETE")
    for field in ("request", "response"):
        raw = wire.get(field)
        require(isinstance(raw, str) and raw and hashlib.sha256(raw.encode("utf-8")).hexdigest() == terminal.get(field + "Sha256"), "GO_PROVIDER_WIRE_HASH_MISMATCH")
    return validate_go_ledger(external, terminal)


def run(relative: str) -> None:
    global ACTIVE_HARD_DEADLINE, ACTIVE_CLEANUP_DEADLINE
    admission, path = fresh_runtime_admission(relative)
    started = time.monotonic()
    hard_deadline = started + MAX_SECONDS
    ACTIVE_HARD_DEADLINE = hard_deadline
    ACTIVE_CLEANUP_DEADLINE = None
    CLEANUP_ERRORS.clear()
    runtime = Path(admission["runtimeRoot"]).resolve()
    external = Path(admission["externalEvidenceDir"])
    require(not runtime.exists(), "R139_RUNTIME_ALREADY_EXISTS")
    require(not external.exists(), "R139_EXTERNAL_EVIDENCE_ALREADY_EXISTS")
    runtime.mkdir(parents=True, exist_ok=False)
    external.mkdir(parents=True, exist_ok=False)
    write_new(runtime / "run_intent.json", {"admissionSha256": digest(path), "sourceCommit": admission["sourceCommit"], "startedAt": time.time(), "deadlineSeconds": MAX_SECONDS})
    append(runtime / "progress.jsonl", {"phase": "admitted", "monotonic": time.monotonic(), "atUTC": datetime.now(timezone.utc).isoformat()})
    owner: dict = {}
    children: dict[str, subprocess.Popen] = {}
    go = vite = None
    api_port: int | None = None
    outcome: dict = {"result": "HOLD", "terminal": False}
    native_validation: dict | None = None
    try:
        image_rows = json.loads(docker(admission, runtime, "image", "inspect", IMAGE))
        require(isinstance(image_rows, list) and len(image_rows) == 1 and image_rows[0].get("Id") == IMAGE, "PINNED_IMAGE_NOT_CACHED")
        protected_baseline = inventory(admission, runtime)
        canonicalize_protected_inventory(protected_baseline)
        write_new(runtime / "protected_inventory_baseline.json", protected_baseline)
        append(runtime / "progress.jsonl", {"phase": "protected_baseline_validated", "monotonic": time.monotonic()})
        key_file = Path(admission["keyPath"])
        key = key_file.read_text(encoding="utf-8").strip()
        require(len(key) >= 16 and "\n" not in key, "KEY_INVALID")
        require(time.monotonic() < hard_deadline - 180, "PREFLIGHT_DEADLINE")
        start_fixture(admission, runtime, external, owner, before=protected_baseline)
        append(runtime / "progress.jsonl", {"phase": "fixture_started", "monotonic": time.monotonic(), "atUTC": datetime.now(timezone.utc).isoformat()})
        minimal = minimal_environment()
        env = dict(minimal)
        env.update({"R139_RUN": "1", "R139_RUNTIME_ADMISSION": str(path), "R139_RUNTIME_ADMISSION_SHA256": digest(path), "R139_SOURCE_COMMIT": admission["sourceCommit"], "R139_FIXTURE_IDENTITY": str(external / "fixture_identity.json"), "R139_MYSQL_DSN": f'root:{owner["fixturePassword"]}@tcp(127.0.0.1:{owner["port"]})/{admission["database"]}?parseTime=true', "R139_RUNTIME_ROOT": str(runtime), "R139_EVIDENCE_DIR": str(external), "R139_EFFECT_DIR": str(external / "effect_slots"), "R139_ALIBABA_KEY": key, "R139_ALIBABA_ENDPOINT": admission["endpoint"], "R139_GRANT_DEADLINE_UTC": admission["grantDeadlineUTC"], "R139_STOP_FILE": str(runtime / "stop_requested")})
        go = launch(admission, runtime, "go", [str(project_path(admission["goTestBinaryPath"])), "-test.run", "^TestR139IntegratedExecutionProof$", "-test.v"], env, children)
        origin = wait_go_ready(go, runtime / "go.log", min(time.monotonic() + 90, hard_deadline - 150))
        api_port = urlsplit(origin).port
        native_validation = validate_native_evidence(external)
        write_new(runtime / "native_evidence_validation.json", native_validation)
        append(runtime / "progress.jsonl", {"phase": "native_validated", "monotonic": time.monotonic(), "atUTC": datetime.now(timezone.utc).isoformat()})
        template = project_path("scripts/r139_vite_config.mjs.template").read_text(encoding="utf-8")
        for token, replacement in {"__R139_FRONTEND_ROOT__": str(ROOT / "frontend"), "__R139_RUNTIME__": str(runtime), "__R139_API_ORIGIN__": origin, "__R139_NONCE__": secrets.token_hex(16)}.items():
            template = template.replace(token, json.dumps(replacement))
        require("__R139_" not in template and "strictPort: true" in template, "VITE_TEMPLATE_INVALID")
        config = runtime / "vite_config.mjs"
        config.write_text(template, encoding="utf-8")
        node = Path(admission["nodeBinary"])
        vite_cli = project_path("frontend/node_modules/vite/bin/vite.js")
        vite = launch(admission, runtime, "vite", [str(node), str(vite_cli), "--config", str(config), "--host", "127.0.0.1", "--port", str(admission["vitePort"]), "--strictPort"], minimal, children)
        health_deadline = min(time.monotonic() + 45, hard_deadline - 130)
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        while time.monotonic() < health_deadline:
            require(vite.poll() is None, "VITE_EXITED_BEFORE_READY")
            try:
                with opener.open(f'http://127.0.0.1:{admission["vitePort"]}/__r139_health', timeout=2) as response:
                    if response.read().decode("utf-8") == json.loads(template.split("const nonce = ", 1)[1].split("\n", 1)[0]):
                        break
            except (OSError, ValueError):
                time.sleep(0.2)
        else:
            raise Hold("VITE_HEALTH_TIMEOUT")
        write_new(runtime / "browser_ready.json", {"apiOrigin": origin, "uiOrigin": f'http://127.0.0.1:{admission["vitePort"]}', "sourceCommit": admission["sourceCommit"], "admissionSha256": digest(path)})
        print(json.dumps({"result": "BROWSER_READY", "apiOrigin": origin, "uiOrigin": f'http://127.0.0.1:{admission["vitePort"]}'}, sort_keys=True), flush=True)
        append(runtime / "progress.jsonl", {"phase": "browser_ready", "monotonic": time.monotonic(), "atUTC": datetime.now(timezone.utc).isoformat()})
        deadline = hard_deadline - min(120, MIN_CLEANUP_SECONDS)
        next_progress = time.monotonic() + 10
        while time.monotonic() < deadline:
            require(go.poll() is None and vite.poll() is None, "CHILD_EXIT_DURING_CAMPAIGN")
            if (runtime / "stop_requested").is_file():
                break
            if time.monotonic() >= next_progress:
                append(runtime / "progress.jsonl", {"phase": "children_held", "monotonic": time.monotonic(), "goPid": go.pid, "vitePid": vite.pid})
                next_progress = time.monotonic() + 10
            time.sleep(0.25)
        else:
            raise Hold("CAMPAIGN_DEADLINE")
        outcome["result"] = "STOP_REQUESTED"
    except Exception as exc:
        outcome["error"] = type(exc).__name__ + ":" + str(exc)
    finally:
        cleanup_started = time.monotonic()
        try:
            append(runtime / "progress.jsonl", {"phase": "cleanup_entered", "monotonic": cleanup_started, "atUTC": datetime.now(timezone.utc).isoformat(), "reason": outcome.get("error", outcome["result"])})
        except Exception as exc:
            outcome["progressError"] = type(exc).__name__
        cleanup_wall_started = time.time()
        ACTIVE_CLEANUP_DEADLINE = cleanup_started + MIN_CLEANUP_SECONDS
        errors: list[str] = []
        outcome["cleanup"] = {"startedAt": cleanup_wall_started, "deadlineSeconds": MIN_CLEANUP_SECONDS, "campaignDeadlineExpiredAtStart": cleanup_started >= hard_deadline - min(120, MIN_CLEANUP_SECONDS)}
        try:
            write_new(runtime / "stop_requested", {"reason": "supervisor-finally", "at": time.time()})
        except FileExistsError:
            pass
        except Exception as exc:
            errors.append("STOP_SIGNAL_FAILED:" + type(exc).__name__)
        if children.get("go") is not None:
            try:
                remaining = max(0.0, ACTIVE_CLEANUP_DEADLINE - time.monotonic())
                require(remaining > 0, "CLEANUP_DEADLINE_EXPIRED")
                children["go"].wait(timeout=min(30, remaining))
            except subprocess.TimeoutExpired:
                errors.append("GO_FINAL_WAIT_TIMEOUT")
            except Exception as exc:
                errors.append("GO_FINAL_WAIT_FAILED:" + type(exc).__name__)
        for name in ("vite", "go"):
            try:
                outcome[name] = stop_child(runtime, name, children.get(name))
            except Exception as exc:
                errors.append(name.upper() + "_STOP_FAILED:" + type(exc).__name__)
        if outcome.get("go", {}).get("exitCode") != 0 or not outcome.get("go", {}).get("waitObserved"):
            errors.append("ORIGINAL_GO_EXIT_NOT_ZERO_OR_UNOBSERVED")
        if children.get("vite") is not None and not outcome.get("vite", {}).get("waitObserved"):
            errors.append("VITE_EXIT_UNOBSERVED")
        try:
            outcome["apiListener"] = listener_absent(api_port, admission, runtime)
        except Exception as exc:
            errors.append("API_LISTENER_CLOSE_UNPROVEN:" + type(exc).__name__)
        if children.get("vite") is not None:
            try:
                outcome["viteListener"] = listener_absent(admission["vitePort"], admission, runtime)
            except Exception as exc:
                errors.append("VITE_LISTENER_CLOSE_UNPROVEN:" + type(exc).__name__)
        if children.get("go") is None or not (external / "go_terminal.json").is_file():
            errors.append("GO_FINAL_DURABLE_EVIDENCE_MISSING")
        else:
            try:
                outcome["goTerminal"] = read_json(external / "go_terminal.json")
                require(native_validation is not None or (runtime / "native_evidence_validation.json").is_file(), "GO_NATIVE_VALIDATION_MISSING")
                final_native_validation = validate_native_evidence(external, require_published=True)
                write_new(runtime / "native_evidence_terminal_validation.json", final_native_validation)
                outcome["nativeEvidenceFinal"] = final_native_validation
                outcome["goRouteAccounting"] = validate_go_terminal(external, outcome["goTerminal"], admission, final_native_validation)
                outcome["viteRouteAccounting"] = validate_vite_ledger(runtime, outcome["goTerminal"])
            except Exception as exc:
                errors.append("TERMINAL_OR_ROUTE_VALIDATION_FAILED:" + type(exc).__name__)
        try:
            recover_partial_owner(admission, runtime, owner)
        except Exception as exc:
            errors.append("PARTIAL_OWNER_RECOVERY_FAILED:" + type(exc).__name__)
        if owner.get("containerId"):
            try:
                outcome["fixture"] = stop_fixture(admission, runtime, owner)
            except Exception as exc:
                errors.append("FIXTURE_TERMINAL_FAILED:" + type(exc).__name__)
        elif owner.get("networkId") or owner.get("volumeName"):
            try:
                outcome["partialOwnership"] = verify_owned_resources(admission, runtime, owner)
            except Exception as exc:
                errors.append("PARTIAL_RESOURCE_OWNERSHIP_UNPROVEN:" + type(exc).__name__)
            errors.append("PARTIAL_RESOURCE_PRESERVED_HOLD")
        if owner.get("port"):
            try:
                outcome["dbListener"] = listener_absent(owner["port"], admission, runtime)
            except Exception as exc:
                errors.append("DB_LISTENER_CLOSE_UNPROVEN:" + type(exc).__name__)
        try:
            fresh_runtime_admission(relative, expect_fresh_external=False, validate_live_deadline=False)
        except Exception as exc:
            errors.append("SOURCE_RECHECK_FAILED:" + type(exc).__name__)
        if admission.get("browserTool") in ("antigravity_native", "supported_CUA_page_target"):
            try:
                outcome["browserTerminalClose"] = verify_browser_terminal_close(admission, runtime, external)
            except Exception as exc:
                errors.append("BROWSER_TERMINAL_CLOSE_FAILED:" + type(exc).__name__)
        errors.extend("CLEANUP_BEST_EFFORT_ERROR:" + issue for issue in CLEANUP_ERRORS)
        cleanup_finished = time.monotonic()
        outcome["cleanup"].update({"finishedAt": time.time(), "elapsedSeconds": round(cleanup_finished - cleanup_started, 3), "completedBeforeDeadline": cleanup_finished <= ACTIVE_CLEANUP_DEADLINE})
        if cleanup_finished > ACTIVE_CLEANUP_DEADLINE:
            errors.append("CLEANUP_DEADLINE_OVERRUN")
        if cleanup_finished > hard_deadline:
            errors.append("LIFECYCLE_DEADLINE_OVERRUN")
        outcome["terminalErrors"] = errors
        outcome["terminal"] = True
        if errors:
            outcome["result"] = "HOLD"
        try:
            write_new(runtime / "terminal.json", outcome)
        except Exception as exc:
            errors.append("TERMINAL_RECEIPT_WRITE_FAILED:" + type(exc).__name__)
            outcome["result"] = "HOLD"
        print(json.dumps(outcome, sort_keys=True), flush=True)
    require(outcome["result"] == "STOP_REQUESTED" and not outcome["terminalErrors"], "R139_TERMINAL_HOLD")


def inspect() -> None:
    print(json.dumps({"tranche": "CCMAI-RUNTIME-139", "sourceOnly": True, "runtimeEffectsAdmittedByThisSource": 0, "browserDiagnosticRemaining": 0}, sort_keys=True))


def main() -> None:
    parser = argparse.ArgumentParser(description="R139 admitted finite runtime supervisor")
    parser.add_argument("command", choices=["inspect", "run"])
    parser.add_argument("--admission", help="Committed fresh R139 runtime/effect admission path")
    args = parser.parse_args()
    if args.command == "inspect":
        inspect()
    else:
        require(bool(args.admission), "RUNTIME_ADMISSION_REQUIRED")
        run(args.admission)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print(type(error).__name__ + ":" + str(error), file=sys.stderr)
        raise SystemExit(1)
