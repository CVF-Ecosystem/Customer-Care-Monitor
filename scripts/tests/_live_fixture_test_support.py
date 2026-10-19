"""Temporary, synthetic roots for live-fixture admission unit tests."""

from contextlib import ExitStack, contextmanager
import json
from pathlib import Path
import subprocess
from unittest.mock import patch


def write_json(path: Path, value: dict) -> bytes:
    raw = json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(raw)
    return raw


def _authority_budget(module: object) -> dict:
    if Path(module.SEED_REL).name.endswith("100.json"):
        return {
            "physicalProviderRequestsTotal": 4,
            "workerProviderRequestsMax": 4,
            "reviewerProviderRequests": 0,
            "automaticRetries": 0,
            "redirects": 0,
            "fallbacks": 0,
            "model": module.MODEL,
            "region": module.REGION,
            "enableThinking": False,
            "endpointSha256": module.ENDPOINT_SHA,
            "requestedCompletionTokens": module.TOKENS,
            "ownerOutputCeiling": 768,
            "requestBytesMax": 6000,
            "responseBytesMax": module.MAX_RESPONSE,
            "requestTimeoutSeconds": 60,
            "campaignSecondsMax": 300,
            "freshOwnerGrantId": module.FRESH_OWNER_GRANT_ID,
            "historicalR098Calls": 2,
            "freshGrantCap": 4,
            "goInvocationsTotal": 4,
            "ownedDisposableMySQLFixturesTotal": 1,
        }

    budget = {
        "physicalProviderRequestsTotal": 2,
        "workerProviderRequests": 1,
        "reviewerProviderRequests": 1,
        "automaticRetries": 0,
        "redirects": 0,
        "fallbacks": 0,
        "model": module.MODEL,
        "region": module.REGION,
        "endpointSha256": module.ENDPOINT_SHA,
        "requestedCompletionTokens": module.TOKENS,
        "ownerOutputCeiling": 768,
        "requestBytesMax": 6000,
        "responseBytesMax": module.MAX_RESPONSE,
        "requestTimeoutSeconds": 60,
        "campaignSecondsMax": 300,
    }
    if hasattr(module, "PREDECESSOR_SHA"):
        budget.update({
            "parentPhysicalRequestsTotal": 2,
            "predecessorPhysicalRequestsKnown": 0,
            "predecessorReceiptSha256": module.PREDECESSOR_SHA,
        })
    if hasattr(module, "FRESH_OWNER_GRANT_ID"):
        budget.update({
            "freshOwnerGrantId": module.FRESH_OWNER_GRANT_ID,
            "historicalR096Calls": 2,
            "freshGrantCap": 2,
        })
    return budget


def fake_git(*, seed_bytes: bytes = b"", ignored: bool = True,
             tracked: bool = False, ancestors: bool = True):
    head = ("a" * 40).encode()

    def run(*args: str) -> subprocess.CompletedProcess:
        command = args[0] if args else ""
        if command == "show":
            return subprocess.CompletedProcess(args, 0, seed_bytes, b"")
        if command == "rev-parse":
            return subprocess.CompletedProcess(args, 0, head + b"\n", b"")
        if command == "merge-base":
            return subprocess.CompletedProcess(args, 0 if ancestors else 1, b"", b"")
        if command == "check-ignore":
            return subprocess.CompletedProcess(args, 0 if ignored else 1, b"", b"")
        if command == "ls-files":
            return subprocess.CompletedProcess(args, 0 if tracked else 1, b"", b"")
        return subprocess.CompletedProcess(args, 1, b"", b"")

    return run


@contextmanager
def isolated_authority(module: object, root: Path, role: str = "worker", *,
                       ignored: bool = True, tracked: bool = False):
    """Patch authority reads to a complete synthetic tree and fake Git history."""
    phase = "REVIEW" if role == "reviewer" else "BUILD"
    tranche_id = Path(module.SEED_REL).stem
    seed = {"executionBudget": _authority_budget(module)}
    seed_bytes = write_json(root / module.SEED_REL, seed)
    state = {
        "activeTranche": tranche_id,
        "activeHandoff": module.HANDOFF_REL,
        "currentMode": phase,
        "activePhase": phase,
        "activeRole": "REVIEWER" if role == "reviewer" else "IMPLEMENTATION_WORKER",
    }
    write_json(root / "CVF_SESSION/ACTIVE_SESSION_STATE.json", state)
    write_json(root / module.TRANCHE_REL, {
        "trancheId": tranche_id,
        "baseCommit": module.SEED_COMMIT,
        "phase": phase,
    })
    status = "COMPLETE_PENDING_REVIEW" if role == "reviewer" else "ACTIVE"
    order = root / module.ORDER_REL
    order.parent.mkdir(parents=True, exist_ok=True)
    order.write_text(f"Status: {status}\n", encoding="utf-8")
    handoff = root / module.HANDOFF_REL
    handoff.parent.mkdir(parents=True, exist_ok=True)
    handoff.write_text("synthetic handoff\n", encoding="utf-8")

    with patch.object(module, "ROOT", root), patch.object(
        module, "git", side_effect=fake_git(
            seed_bytes=seed_bytes, ignored=ignored, tracked=tracked)):
        yield {
            "root": root,
            "seed": seed,
            "state": state,
            "seed_path": root / module.SEED_REL,
            "state_path": root / "CVF_SESSION/ACTIVE_SESSION_STATE.json",
        }


@contextmanager
def isolated_parent_manifest(module: object, root: Path, *,
                             ignored: bool = True, tracked: bool = False):
    """Create a fake parent ledger and historical receipt beneath a temp root."""
    parent_path = root / Path(module.PARENT).relative_to(module.ROOT)
    patches = ExitStack()
    patches.enter_context(patch.object(module, "ROOT", root))
    patches.enter_context(patch.object(module, "PARENT", parent_path))

    if hasattr(module, "HISTORICAL_REL"):
        history_path = root / module.HISTORICAL_REL
        history = [{"attempts": 1, "model": module.MODEL},
                   {"attempts": 1, "model": module.MODEL}]
        history_bytes = json.dumps(history, sort_keys=True, separators=(",", ":")).encode()
        history_path.parent.mkdir(parents=True, exist_ok=True)
        history_path.write_bytes(history_bytes)
        ledger = {
            "approvedBy": "Codex /root",
            "ownerCap": 4,
            "freshOwnerGrantId": module.FRESH_OWNER_GRANT_ID,
            "freshGrantCap": 4,
            "historicalR098Calls": 2,
            "seedCommit": module.SEED_COMMIT,
            "historicalReceipt": module.HISTORICAL_REL,
            "historicalReceiptSha256": module.digest(history_bytes),
            "provider": "Alibaba",
            "model": module.MODEL,
            "region": module.REGION,
            "endpointSha256": module.ENDPOINT_SHA,
            "automaticRetries": 0,
            "workerSlots": [1, 2, 3, 4],
        }
    else:
        source_hashes = {}
        for index in range(3):
            name = f"fixture_sources/source_{index}.txt"
            content = f"synthetic source {index}".encode()
            source_path = root / name
            source_path.parent.mkdir(parents=True, exist_ok=True)
            source_path.write_bytes(content)
            source_hashes[name] = module.digest(content)
        predecessor = {
            "actual": {
                "error_code": "SECRET_SCHEMA_MISMATCH",
                "wire": {"attempts": 0},
            },
            "sourceSha256": source_hashes,
        }
        predecessor_bytes = json.dumps(
            predecessor, sort_keys=True, separators=(",", ":")).encode()
        history_path = root / module.PREDECESSOR_REL
        history_path.parent.mkdir(parents=True, exist_ok=True)
        history_path.write_bytes(predecessor_bytes)
        predecessor_sha = module.digest(predecessor_bytes)
        patches.enter_context(patch.object(module, "PREDECESSOR_SHA", predecessor_sha))
        ledger = {
            "approvedBy": "Codex /root",
            "ownerCap": 2,
            "predecessorPhysicalCallsKnown": 0,
            "predecessorTranche": "CCMAI-RUNTIME-095",
            "predecessorReceipt": module.PREDECESSOR_REL,
            "predecessorReceiptSha256": predecessor_sha,
            "provider": "Alibaba",
            "model": module.MODEL,
            "region": module.REGION,
            "endpointSha256": module.ENDPOINT_SHA,
            "automaticRetries": 0,
            "roleSlots": {"worker": 1, "reviewer": 2},
        }
        if hasattr(module, "FRESH_OWNER_GRANT_ID"):
            ledger.update({
                "freshOwnerGrantId": module.FRESH_OWNER_GRANT_ID,
                "freshGrantCap": 2,
                "historicalR096Calls": 2,
                "seedCommit": module.SEED_COMMIT,
            })

    write_json(parent_path, ledger)
    patches.enter_context(patch.object(
        module, "git", side_effect=fake_git(ignored=ignored, tracked=tracked)))
    try:
        with patches:
            yield {
                "root": root,
                "ledger": ledger,
                "ledger_path": parent_path,
                "history_path": history_path,
                "history_bytes": history_bytes if hasattr(module, "HISTORICAL_REL") else predecessor_bytes,
            }
    finally:
        patches.close()
