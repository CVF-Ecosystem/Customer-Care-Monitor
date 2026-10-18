#!/usr/bin/env python3
"""Root-only, one-use owner-authorized R117 restart and final stop.

This imports the admitted R117 supervisor for its source, ownership, Docker,
port, inventory, and secret-safe receipt controls. It never creates resources,
runs Go tests, or reads provider credentials.
"""
from __future__ import annotations

import argparse
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import time


ROOT = Path(__file__).resolve().parents[4]
SUPERVISOR = ROOT / "scripts/r117_waitdata_fixture.py"
spec = importlib.util.spec_from_file_location("r117_waitdata_fixture", SUPERVISOR)
m = importlib.util.module_from_spec(spec)
spec.loader.exec_module(m)

SUPPLEMENT = ROOT / "docs/reviews/probes/r117_root/owner_restart_authority_18.json"
SUPPLEMENT_SHA = "652149d549bdef060fc32a2ec73cd732c061cf3a98ef0bb12cc70eb3f60c13e2"
ORIGINAL_PIN = ROOT / "docs/reviews/probes/r117_root/fixture_pin.json"
ORIGINAL_PIN_SHA = "41b73398a692c153b7bcf889d7691f22d88078d6381fe314ef434beb4709fc93"
CONTAINER = "033e0151bb4805cce1ce6dc347ab79ab8275e016c8ace371cfda158297a2c352"
NETWORK = "fe2271a8b533ec1ed3d3d09f1b561afcf6491c9353b246555ad3fd3518e5bab0"
OLD_BRIDGE = "0d9f7d2dba52be2e98f8d20ade28956797e6950fecb9e6fea798e6d497d1d271"
NEW_BRIDGE = "ee9a01565a47c6d046857ae6534a688143034693f7b379faeacd6a94ab97cbd2"
ROOT_EVIDENCE = ROOT / "docs/reviews/probes/r117_root"


def authority():
    own_path = Path(__file__).resolve()
    own_relative = own_path.relative_to(ROOT).as_posix()
    own_committed = subprocess.run(["git", "show", "HEAD:" + own_relative], cwd=ROOT,
                                   capture_output=True, timeout=15)
    if own_committed.returncode or own_committed.stdout.replace(b"\r\n", b"\n") != own_path.read_bytes().replace(b"\r\n", b"\n"):
        raise RuntimeError("OWNER_HELPER_NOT_COMMITTED")
    if m.sha(SUPPLEMENT) != SUPPLEMENT_SHA:
        raise RuntimeError("OWNER_SUPPLEMENT_CHANGED")
    value = m.committed(SUPPLEMENT)
    expected = {
        "schemaVersion": 1, "authorityKind": "EXPLICIT_HUMAN_POST_SHUTDOWN_CONTINUATION",
        "trancheId": "CCMAI-RUNTIME-117", "parentSeedUnchanged": True,
        "containerId": CONTAINER, "volumeName": m.NAME, "networkId": NETWORK,
        "image": m.IMAGE, "providerGrantId": "CCMAI-RUNTIME-117-OWNER-NEW1-20261017",
        "additionalBudget": {"ownedFixtureRestarts": 1, "ownedFixtureFinalStops": 1,
                             "nativeReplays": 0, "additionalProviderCalls": 0},
    }
    if any(value.get(key) != item for key, item in expected.items()):
        raise RuntimeError("OWNER_SUPPLEMENT_MISMATCH")
    replacement = value.get("acceptedDefaultBridgeReplacement")
    if not isinstance(replacement, dict) or any(replacement.get(k) != v for k, v in {
        "name": "bridge", "oldId": OLD_BRIDGE, "newId": NEW_BRIDGE,
        "evidence": "docs/reviews/probes/r117_root/protected_resource_drift_18.json",
    }.items()):
        raise RuntimeError("BRIDGE_EXCEPTION_MISMATCH")
    m.verify_source()
    return value


def original_pin():
    if m.sha(ORIGINAL_PIN) != ORIGINAL_PIN_SHA or m.sha(m.RUN / "fixture.json") != ORIGINAL_PIN_SHA:
        raise RuntimeError("ORIGINAL_FIXTURE_PIN_CHANGED")
    pin = m.committed(ORIGINAL_PIN)
    if (pin.get("containerId") != CONTAINER or pin.get("networkId") != NETWORK
            or pin.get("name") != m.NAME or pin.get("port") != 59002
            or pin.get("volumeIdentity", {}).get("Name") != m.NAME
            or pin.get("networkIdentity", {}).get("Id") != NETWORK):
        raise RuntimeError("ORIGINAL_FIXTURE_IDENTITY_MISMATCH")
    return pin


def bridge_only_drift(old, current):
    if old.get("volumes") != current.get("volumes"):
        raise RuntimeError("PROTECTED_VOLUME_DRIFT")
    old_networks, new_networks = old.get("networks"), current.get("networks")
    if not isinstance(old_networks, list) or not isinstance(new_networks, list):
        raise RuntimeError("PROTECTED_NETWORK_INVALID")
    old_bridge = [row for row in old_networks if row.get("Name") == "bridge"]
    new_bridge = [row for row in new_networks if row.get("Name") == "bridge"]
    if (len(old_bridge) != 1 or len(new_bridge) != 1
            or old_bridge[0].get("ID") != OLD_BRIDGE or new_bridge[0].get("ID") != NEW_BRIDGE):
        raise RuntimeError("DEFAULT_BRIDGE_REPLACEMENT_MISMATCH")
    if (sorted((row for row in old_networks if row.get("Name") != "bridge"), key=lambda row: row["ID"])
            != sorted((row for row in new_networks if row.get("Name") != "bridge"), key=lambda row: row["ID"])):
        raise RuntimeError("OTHER_PROTECTED_NETWORK_DRIFT")
    for field in ("ID", "CreatedAt"):
        old_bridge[0] = {k: v for k, v in old_bridge[0].items() if k != field}
        new_bridge[0] = {k: v for k, v in new_bridge[0].items() if k != field}
    if old_bridge[0] != new_bridge[0]:
        raise RuntimeError("DEFAULT_BRIDGE_FIELDS_DRIFT")


def exact_pin(pin):
    if (pin.get("containerId") != CONTAINER or pin.get("networkId") != NETWORK
            or pin.get("name") != m.NAME or pin.get("volumeIdentity", {}).get("Name") != m.NAME):
        raise RuntimeError("OWNED_FIXTURE_PIN_MISMATCH")


def raw_archive(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb") as stream:
        stream.write(data)


def restart():
    authority()
    pin = original_pin()
    if (m.RUN / "owner_restart_18.effect").exists() or (m.RUN / "owner_restart_18_counter.json").exists():
        raise RuntimeError("OWNER_RESTART_ALREADY_RESERVED")
    before = m.inspect_fixture(pin, check_protected=False, require_port=False)
    if before["State"] != "exited":
        raise RuntimeError("OWNER_RESTART_REQUIRES_EXITED")
    containers, resources = m.inventory(), m.resource_inventory()
    if containers != pin["protectedBaseline"]:
        raise RuntimeError("PROTECTED_CONTAINER_DRIFT")
    bridge_only_drift(pin["protectedResources"], resources)
    baseline = {"at": m.utc(), "containerId": CONTAINER, "protectedBaseline": containers,
                "protectedResources": resources, "supplementSha256": SUPPLEMENT_SHA,
                "sourceAdmissionSha256": m.sha(m.SOURCE_ADMISSION), "originalPinSha256": ORIGINAL_PIN_SHA}
    m.exclusive(m.RUN / "owner_operational_baseline_18.json", baseline)
    m.exclusive(ROOT_EVIDENCE / "owner_operational_baseline_18.json", baseline)
    raw_archive(ROOT_EVIDENCE / "fixture_local_before_owner_restart_18.json",
                (m.RUN / "fixture.json").read_bytes())
    m.exclusive(m.RUN / "owner_restart_18.effect", {"at": m.utc(), "containerId": CONTAINER,
                "supplementSha256": SUPPLEMENT_SHA, "maxRestarts": 1})
    m.exclusive(m.RUN / "owner_restart_18_counter.json", {"ownedFixtureRestartsUsed": 1,
                "ownedFixtureFinalStopsUsed": 0, "providerCallsObservedByHelper": "NOT_OBSERVED",
                "additionalProviderAuthority": 0,
                "parentFixtureStartsUntouched": True, "supplementSha256": SUPPLEMENT_SHA})
    m.docker("start", CONTAINER, timeout=30)
    ports = json.loads(m.docker("inspect", "--format", "{{json .NetworkSettings.Ports}}", CONTAINER).stdout)
    m.exclusive(ROOT_EVIDENCE / "owner_restart_port_observation_18.json",
                {"at": m.utc(), "containerId": CONTAINER, "ports": ports})
    port = m.published_port(ports)
    live_pin = {**pin, "port": port, "protectedBaseline": containers,
                "protectedResources": resources, "ownerRestartSupplementSha256": SUPPLEMENT_SHA,
                "operationalBaselineSha256": m.sha(m.RUN / "owner_operational_baseline_18.json")}
    m.inspect_fixture(live_pin)
    deadline = time.monotonic() + 60
    ready = False
    while time.monotonic() < deadline:
        result = m.docker("exec", "-e", "MYSQL_PWD=" + m.DB_PASSWORD, CONTAINER, "mysql", "-h", "127.0.0.1",
                          "-u", "root", "-N", "-B", "-e",
                          "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='r116_fixture'",
                          checked=False)
        if result.returncode == 0 and result.stdout.strip() == b"r116_fixture":
            ready = True
            break
        time.sleep(1)
    if not ready:
        raise RuntimeError("OWNER_RESTART_MYSQL_NOT_READY_NO_RETRY")
    m.inspect_fixture(live_pin)
    pending = m.RUN / "owner_fixture_18.pending.json"
    m.exclusive(pending, live_pin)
    os.replace(pending, m.RUN / "fixture.json")
    m.exclusive(ROOT_EVIDENCE / "owner_restart_ready_18.json",
                {"at": m.utc(), "containerId": CONTAINER, "port": port, "state": "running",
                 "mysqlSchema": "r116_fixture", "operationalBaselineSha256": live_pin["operationalBaselineSha256"],
                 "localFixtureSha256": m.sha(m.RUN / "fixture.json"), "originalPinSha256": ORIGINAL_PIN_SHA,
                 "supplementSha256": SUPPLEMENT_SHA, "sourceAdmissionSha256": m.sha(m.SOURCE_ADMISSION),
                 "nativeReplay": False, "providerCall": False})
    print("R117_OWNER_RESTART_READY")


def cleanup():
    authority()
    if not (m.RUN / "owner_restart_18.effect").exists():
        raise RuntimeError("OWNER_RESTART_NOT_RESERVED")
    counter = m.load(m.RUN / "owner_restart_18_counter.json")
    if counter.get("ownedFixtureRestartsUsed") != 1 or counter.get("ownedFixtureFinalStopsUsed") != 0:
        raise RuntimeError("OWNER_SUPPLEMENT_COUNTER_MISMATCH")
    if m.sha(ORIGINAL_PIN) != ORIGINAL_PIN_SHA or m.sha(ROOT_EVIDENCE / "fixture_local_before_owner_restart_18.json") != ORIGINAL_PIN_SHA:
        raise RuntimeError("ORIGINAL_EVIDENCE_CHANGED")
    baseline_path = m.RUN / "owner_operational_baseline_18.json"
    baseline_sha = m.sha(baseline_path)
    if m.sha(ROOT_EVIDENCE / "owner_operational_baseline_18.json") != baseline_sha:
        raise RuntimeError("OPERATIONAL_BASELINE_CHANGED")
    baseline = m.load(m.RUN / "owner_operational_baseline_18.json")
    if (baseline.get("containerId") != CONTAINER or baseline.get("supplementSha256") != SUPPLEMENT_SHA
            or baseline.get("originalPinSha256") != ORIGINAL_PIN_SHA):
        raise RuntimeError("OPERATIONAL_BASELINE_MISMATCH")
    local_pin_path = m.RUN / "fixture.json"
    local_pin_sha = m.sha(local_pin_path)
    local_pin = m.load(local_pin_path)
    exact_pin(local_pin)
    rebound = local_pin_sha != ORIGINAL_PIN_SHA
    if rebound:
        if (local_pin.get("ownerRestartSupplementSha256") != SUPPLEMENT_SHA
                or local_pin.get("operationalBaselineSha256") != baseline_sha):
            raise RuntimeError("OPERATIONAL_PIN_MISMATCH")
    elif m.sha(ROOT_EVIDENCE / "fixture_local_before_owner_restart_18.json") != local_pin_sha:
        raise RuntimeError("ORIGINAL_LOCAL_PIN_CHANGED")
    # Cleanup is independent of published-port and MySQL readiness. If start
    # succeeded but either failed, the original local pin is still in place.
    pin = {**local_pin, "protectedBaseline": baseline["protectedBaseline"],
           "protectedResources": baseline["protectedResources"]}
    pin.pop("port", None)
    before = m.inspect_fixture(pin, check_protected=False, require_port=False)
    if before["State"] != "running":
        raise RuntimeError("OWNER_FINAL_STOP_REQUIRES_RUNNING")
    m.exclusive(m.RUN / "owner_final_stop_18.effect", {"at": m.utc(), "containerId": CONTAINER,
                "supplementSha256": SUPPLEMENT_SHA, "maxStops": 1})
    m.exclusive(m.RUN / "owner_final_stop_18_counter.json", {"ownedFixtureRestartsUsed": 1,
                "ownedFixtureFinalStopsUsed": 1, "providerCallsObservedByHelper": "NOT_OBSERVED",
                "additionalProviderAuthority": 0,
                "parentFixtureStartsUntouched": True, "supplementSha256": SUPPLEMENT_SHA})
    stopped = m.docker("stop", "-t", "10", CONTAINER, timeout=30)
    after = m.inspect_fixture(pin, check_protected=False, require_port=False)
    containers_equal = m.inventory() == baseline["protectedBaseline"]
    resources_equal = m.resource_inventory() == baseline["protectedResources"]
    m.exclusive(ROOT_EVIDENCE / "owner_cleanup_receipt_18.json",
                {"at": m.utc(), "containerId": CONTAINER, "before": before, "after": after,
                 "stopExitCode": stopped.returncode, "protectedBaselineEquals": containers_equal,
                 "protectedResourcesEqual": resources_equal, "containerPreservedStopped": True,
                 "volumeAndNetworkPreservedOwned": True,
                 "operationalBaselineSha256": baseline_sha,
                 "supplementSha256": SUPPLEMENT_SHA,
                 "originalCleanupReceiptPreserved": True,
                 "localPinReboundBeforeStop": rebound,
                 "readyReceiptPresent": (ROOT_EVIDENCE / "owner_restart_ready_18.json").exists(),
                 "providerCallsObservedByHelper": "NOT_OBSERVED"})
    if after["State"] != "exited" or not containers_equal or not resources_equal:
        raise RuntimeError("OWNER_FINAL_STOP_REVIEW_HOLD")
    print("R117_OWNER_FINAL_STOPPED_PRESERVED")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=("restart", "cleanup"))
    choice = parser.parse_args().action
    try:
        {"restart": restart, "cleanup": cleanup}[choice]()
    except Exception as error:
        code = str(error) if isinstance(error, RuntimeError) else type(error).__name__
        print("R117_OWNER_HOLD " + code)
        raise SystemExit(1) from None
