"""R139 isolated native-tool-close receipt proof. No inherited test discovery.

Imports the inert legacy fixture helpers; the guarded main runs one explicit
native receipt method only. No DB, process, network or provider calls.
"""
import json
import sys
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path
from unittest.mock import patch

PROJECT_ROOT = Path(__file__).resolve().parents[5]
if not (PROJECT_ROOT / ".cvf/manifest.json").is_file():
    raise RuntimeError("PROJECT_IMPORT_ROOT_INVALID")
sys.path.insert(0, str(PROJECT_ROOT))
from scripts.tests import test_r139_execution_supervisor as legacy

module = legacy.module


class NativeToolCloseTests(unittest.TestCase):
    _native_fixture = legacy.R139SupervisorPureTests._native_fixture
    _page_target_fixture = legacy.R139SupervisorPureTests._page_target_fixture

    def _cua_fixture(self, root, close_mode="operator_fallback"):
        readiness = root / ".cvf/runtime/cua_readiness"
        runtime = root / ".cvf/runtime/cua_product"
        readiness.mkdir(parents=True)
        now = datetime.now(timezone.utc).replace(microsecond=0)
        stamp = lambda seconds: (now + timedelta(seconds=seconds)).strftime("%Y-%m-%dT%H:%M:%SZ")
        ident = {"nativeContractId": "r139-cua-page-target-v1", "trancheId": "CCMAI-RUNTIME-139",
                 "campaignNonce": "nonce-cua", "browserId": "2", "tabId": "tab-123",
                 "ownedScopeId": "owned-cua-tab", "producerAgentTaskId": "producer-task",
                 "observerAgentTaskId": "observer-task", "operator": "Owner in this conversation",
                 "closeMode": close_mode, "proofContract": "cua_page_target_v1",
                 "browserKind": "iab", "scopeOrigin": "caller_owned_explicit_cua_tab_target",
                 "ownedTabIds": ["tab-123"]}
        base, product = ".cvf/runtime/cua_readiness/", ".cvf/runtime/cua_product/"
        admission = {**ident, "browserTool": "supported_CUA_page_target", "browserSupported": True,
                     "fallbackCloseOperator": "Owner in this conversation",
                     "browserReadinessRoot": ".cvf/runtime/cua_readiness", "runtimeRoot": str(runtime),
                     "browserReceiptPath": base + "reservation.json",
                     "browserCloseReceiptPath": product + "close.json",
                     "actionEvidencePath": product + "action.json",
                     "finalInventoryPath": product + "inventory.json",
                     "rawCuaInventoryPath": product + "raw_inventory.json",
                     "browserDoneSignalPath": product + "browser_done.signal",
                     "grantDeadlineUTC": stamp(3600)}
        created = "https://example.com/#r139_nonce=nonce-cua&phase=created"
        controlled = "https://example.com/#r139_nonce=nonce-cua&phase=controlled"
        origin = {"cuaCallRef": "cua-call-1", "originalCuaReturnReference": "private/cua/call-1",
                  "originalCuaReturnSha256": "b" * 64,
                  "provenanceBasis": "caller_recorded_original_cua_return",
                  "rootOriginalReturnReviewRequired": True}
        def event(name, seconds, **extra):
            at = stamp(seconds)
            return {**ident, **origin, "event": name, "atUTC": at, "observedAtUTC": at, **extra}
        rows = {
            "baselineInventory": event("baseline_inventory", -60, tabIds=["protected-tab"], inventoryScope="metadata_only"),
            "creation": event("creation", -50, agentTaskId="producer-task", method="createBrowserTab",
                              createdTabId="tab-123", requestedBrowserId="2", requestedUrl=created,
                              callCompleted=True),
            "creationLedger": event("creation_ledger", -49, agentTaskId="producer-task", newTabIds=["tab-123"]),
            "control": event("control", -45, agentTaskId="producer-task", method="goto",
                             targetTabId="tab-123", requestedUrl=controlled, callCompleted=True),
            "capture": event("capture", -40, agentTaskId="producer-task", method="getScreenshot",
                             targetTabId="tab-123", captureSha256="c" * 64),
            "handoff": event("handoff", -30, agentTaskId="observer-task", method="getTab_then_listTabs",
                             requestedTabId="tab-123", observedTabId="tab-123",
                             metadataBrowserId="2", metadataTabId="tab-123",
                             observedUrl=controlled, observedNonce="nonce-cua",
                             getTabCallRef="cua-gettab", getTabReturnReference="private/cua/gettab",
                             getTabReturnSha256="e" * 64,
                             metadataCallRef="cua-listtabs", metadataReturnReference="private/cua/listtabs",
                             metadataReturnSha256="f" * 64),
        }
        prefix = {**ident, "status": "RESERVED_CUA_TAB_CONTROL_CAPTURE_HANDOFF",
                  "protocol": "https", "closeObserved": False, "rootOriginalReturnReviewed": True,
                  "reservedAtUTC": stamp(-25)}
        def write_row(name):
            path = readiness / (name + ".json")
            path.write_text(json.dumps(rows[name]), encoding="utf-8")
            suffix = "" if name in ("baselineInventory", "creationLedger") else "Evidence"
            prefix[name + suffix + "Path"] = base + name + ".json"
            prefix[name + suffix + "Sha256"] = module.digest(path)
        for name in rows:
            write_row(name)
        rows["creationLedger"]["baselineInventorySha256"] = prefix["baselineInventorySha256"]
        rows["creationLedger"]["creationEvidenceSha256"] = prefix["creationEvidenceSha256"]
        for field, links in (("creationLedger", {"baselineInventory": prefix["baselineInventorySha256"],
                                                 "creation": prefix["creationEvidenceSha256"]}),):
            rows[field].pop("cuaCallRef")
            rows[field].pop("originalCuaReturnReference")
            rows[field].pop("originalCuaReturnSha256")
            rows[field]["provenanceBasis"] = "derived_from_original_cua_returns"
            rows[field]["sourceEvidenceSha256"] = links
        write_row("creationLedger")
        rows["handoff"].pop("cuaCallRef")
        rows["handoff"].pop("originalCuaReturnReference")
        rows["handoff"].pop("originalCuaReturnSha256")
        rows["handoff"]["provenanceBasis"] = "derived_from_original_cua_returns"
        rows["handoff"]["sourceEvidenceSha256"] = {"getTab": "e" * 64, "listTabs": "f" * 64}
        write_row("handoff")
        receipt = readiness / "reservation.json"
        receipt.write_text(json.dumps(prefix), encoding="utf-8")
        admission["browserReceiptSha256"] = module.digest(receipt)
        return admission, prefix, rows, runtime, stamp, ident

    def test_cua_native_tool_close_contract(self):
        failures = (None, "failed_return", "unknown_return", "missing_return", "wrong_digest",
                    "wrong_request_digest", "wrong_request_link", "missing_request", "bad_request_origin",
                    "wrong_method", "foreign_target", "foreign_campaign", "wrong_close_actor",
                    "wrong_receipt_actor", "same_actor", "wrong_inventory_actor", "wrong_raw_actor",
                    "before_close", "late_close", "target_present", "baseline_drift", "post_close_action",
                    "owner_alias", "physical_time", "wrong_mode", "aggregate_before_return", "aggregate_after_observed")
        for bad in failures:
            with self.subTest(case=bad), tempfile.TemporaryDirectory() as name:
                root = Path(name)
                with patch.object(module, "ROOT", root):
                    admission, prefix, _, runtime, stamp, ident = self._cua_fixture(root, "native_tool")
                    runtime.mkdir()
                    origin = {"cuaCallRef": "cua-close-call", "originalCuaReturnReference": "private/cua/close-return",
                              "originalCuaReturnSha256": "d" * 64,
                              "provenanceBasis": "caller_recorded_original_cua_return",
                              "rootOriginalReturnReviewRequired": True}
                    def event(kind, seconds, **extra):
                        return {**ident, **origin, "event": kind, "atUTC": stamp(seconds),
                                "observedAtUTC": stamp(seconds), **extra}
                    def write(filename, value):
                        path = runtime / filename
                        path.write_text(json.dumps(value), encoding="utf-8")
                        return str(path.relative_to(root)).replace("\\", "/"), module.digest(path)
                    request = event("tool_close_request", -12, method="Tab.close", targetTabId=ident["tabId"],
                                    closeActorTaskId=ident["producerAgentTaskId"], status="REQUESTED",
                                    originalCuaRequestReference="private/cua/close-request",
                                    originalCuaRequestSha256="a" * 64)
                    for key in ("cuaCallRef", "originalCuaReturnReference", "originalCuaReturnSha256"):
                        request.pop(key)
                    request["provenanceBasis"] = "caller_recorded_original_cua_request"
                    if bad == "wrong_method": request["method"] = "Browser.close"
                    if bad == "bad_request_origin": request["originalCuaRequestSha256"] = "bad"
                    request_path, request_sha = write("request.json", request)
                    result = event("tool_close_return", -10, method="Tab.close", targetTabId=ident["tabId"],
                                   closeActorTaskId=ident["producerAgentTaskId"], status="SUCCEEDED",
                                   callCompleted=True, returnType="void", requestPath=request_path, requestSha256=request_sha)
                    if bad in ("failed_return", "unknown_return"): result["status"] = bad
                    if bad == "foreign_target": result["targetTabId"] = "foreign-tab"
                    if bad == "foreign_campaign": result["campaignNonce"] = "foreign"
                    if bad == "wrong_close_actor": result["closeActorTaskId"] = ident["observerAgentTaskId"]
                    if bad == "wrong_request_link": result["requestSha256"] = "f" * 64
                    return_path, return_sha = write("return.json", result)
                    row = {**origin, "tabId": ident["tabId"], "action": "tool_close", "atUTC": stamp(-10),
                           "observedAtUTC": stamp(-10), "closeActorTaskId": ident["producerAgentTaskId"],
                           "toolCloseReturnPath": return_path, "toolCloseReturnSha256": return_sha}
                    actions = [row]
                    if bad == "post_close_action": actions.append(dict(row, action="product_read", atUTC=stamp(-9)))
                    action = event("action_sequence", -10, actions=actions)
                    if bad == "aggregate_before_return": action.update(atUTC=stamp(-11), observedAtUTC=stamp(-11))
                    if bad == "aggregate_after_observed": action.update(atUTC=stamp(-4), observedAtUTC=stamp(-4))
                    for key in ("cuaCallRef", "originalCuaReturnReference", "originalCuaReturnSha256"): action.pop(key)
                    action["provenanceBasis"] = "derived_from_original_cua_returns"
                    action["sourceEvidenceSha256"] = {str(i): r["originalCuaReturnSha256"] for i, r in enumerate(actions)}
                    action_path, action_sha = write("action.json", action)
                    ids = ["protected-tab"]
                    if bad == "target_present": ids.append(ident["tabId"])
                    if bad == "baseline_drift": ids = []
                    inventory_seconds = -11 if bad == "before_close" else -5
                    raw = event("cua_inventory", inventory_seconds, tabIds=ids, ownedCount=0,
                                inventoryScope="metadata_only", inventoryActorTaskId=ident["observerAgentTaskId"])
                    raw["cuaCallRef"] = "cua-independent-final"
                    if bad == "wrong_raw_actor": raw["inventoryActorTaskId"] = ident["producerAgentTaskId"]
                    raw_path, raw_sha = write("raw_inventory.json", raw)
                    inv = event("final_inventory", inventory_seconds, tabIds=ids, ownedCount=0,
                                inventoryScope="metadata_only", inventoryActorTaskId=ident["observerAgentTaskId"],
                                rawCuaInventoryPath=raw_path, rawCuaInventorySha256=raw_sha)
                    if bad == "wrong_inventory_actor": inv["inventoryActorTaskId"] = ident["producerAgentTaskId"]
                    inv_path, inv_sha = write("inventory.json", inv)
                    receipt = {**ident, "status": "CLOSED_EXACT_OWNED_CUA_TAB", "closeObserved": True,
                               "ownedTabsFinal": 0, "closureTimeBasis": "cua_final_metadata_observation",
                               "rootOriginalReturnReviewed": True, "reservedAtUTC": prefix["reservedAtUTC"],
                               "closeActorTaskId": ident["producerAgentTaskId"], "toolCloseReturnedAtUTC": stamp(-10),
                               "closureObservedAtUTC": stamp(inventory_seconds), "actionAtUTC": action["atUTC"],
                               "toolCloseRequestPath": request_path, "toolCloseRequestSha256": request_sha,
                               "toolCloseReturnPath": return_path, "toolCloseReturnSha256": return_sha,
                               "actionEvidencePath": action_path, "actionEvidenceSha256": action_sha,
                               "finalInventoryPath": inv_path, "finalInventorySha256": inv_sha}
                    if bad == "wrong_receipt_actor": receipt["closeActorTaskId"] = ident["observerAgentTaskId"]
                    if bad == "owner_alias": receipt["ownerReportReferenceSha256"] = "a" * 64
                    if bad == "physical_time": receipt["closedAtUTC"] = stamp(-10)
                    if bad == "wrong_digest": receipt["toolCloseReturnSha256"] = "f" * 64
                    if bad == "wrong_request_digest": receipt["toolCloseRequestSha256"] = "f" * 64
                    if bad == "missing_return": (runtime / "return.json").unlink()
                    if bad == "missing_request": (runtime / "request.json").unlink()
                    if bad == "late_close": admission["grantDeadlineUTC"] = stamp(-11)
                    if bad == "same_actor": admission["observerAgentTaskId"] = admission["producerAgentTaskId"]
                    if bad == "wrong_mode": admission["closeMode"] = "api"
                    _, close_sha = write("close.json", receipt)
                    write("browser_done.signal", {"status": "TERMINAL_EVIDENCE_WRITTEN", "proofContract": ident["proofContract"],
                          "campaignNonce": ident["campaignNonce"], "browserId": ident["browserId"], "tabId": ident["tabId"],
                          "ownedScopeId": ident["ownedScopeId"], "atUTC": stamp(-4), "evidenceSha256": {
                              "browserCloseReceiptPath": close_sha, "actionEvidencePath": action_sha,
                              "finalInventoryPath": inv_sha, "rawCuaInventoryPath": raw_sha}})
                    if bad is None:
                        self.assertEqual(module.verify_browser_terminal_close(admission, runtime, runtime)["receipt"], receipt)
                    else:
                        with self.assertRaises(module.Hold): module.verify_browser_terminal_close(admission, runtime, runtime)
        # The added enum is deliberately unavailable to both preexisting non-CUA routes.
        for fixture, validator in ((self._native_fixture, module.native_identity),
                                   (self._page_target_fixture, module.page_target_identity)):
            with tempfile.TemporaryDirectory() as name:
                root = Path(name)
                with patch.object(module, "ROOT", root):
                    values = fixture(root)
                    admission, identity = values[0], values[-1]
                    admission["closeMode"] = identity["closeMode"] = "native_tool"
                    with self.assertRaises(module.Hold): validator(admission, identity, "NON_CUA_NATIVE_TOOL")


if __name__ == "__main__":
    suite = unittest.TestSuite([NativeToolCloseTests("test_cua_native_tool_close_contract")])
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    raise SystemExit(0 if result.wasSuccessful() else 1)
