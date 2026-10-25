# R139 boot-identity v2 source worker return

Status: WORKER_FOCUSED_PYTHON_PASS_PENDING_ROOT_INDEPENDENT_CHECK_AND_REVIEW  
Risk: R2  
Worker: `/root/r139_cua_recipe_worker` (Sol Medium REPAIR_WORKER)  
Reviewer: Root independent source REVIEWER and `/root/r139_cua_preclock_reviewer` independent Sol reviewer; final static review pending  
Tranche/order: existing CCMAI-RUNTIME-139, M3-05/M3-08; phase BUILD  
Effect boundary: one reserved focused pure Python check after static review; no compile, prototype/controller, OS query, Go, service, browser, fixture, key, provider or campaign reservation. Initial authoring pass 1/1 and consolidated repair pass 1/1; new worker Python checks 1/1, new Root Python checks 0/1; original eight checks remain spent.

## Changed set and byte custody

- New successor design: `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/reentry_activation_boot_identity_v2_design.py.txt` SHA-256 `89d6f20eb1b80e1e5f331876886542fce7046fe010e0d6982b1b6b8f56f29454`.
- Existing focused Python test: `scripts/tests/test_r139_execution_supervisor.py` SHA-256 `496a34143ba036b568e211f8e9c07926f20553c1df3d1c74d59fef6be39d0e6c`.
- Frozen original activation design remains SHA-256 `57ae20a2bb0054e38d5a3d919f5e2997780b56981a82f2f31924eddf623e5854` (`57ae20a2…`), unedited.
- Recovery admission `ab4b4d14…`, reviewed contract `4c900535…`, and reviewed proof `10937592…` are read-only source inputs. Old three grant refs and original failure receipts remain unchanged.

## Implementation and review points

Consolidated repair F01–F03 is incorporated in the full diffs below. F01 asserts the separate `preclockInvocation.additionalCap=1`, old attempt 1/exit1, exact recovery source/check budgets, runtime caps, committed 1+1 PASS receipts, accepted source review, and final technical review before the future query. The consumed discovery cap remains recorded without becoming invocation authority. F02 compares UTC integer ticks with exact `Fraction` monotonic differences; neither side is rounded or pre-subtracted as float. F03 adds malformed UTC/type/identity/event, exact and just-over timing, future 100 ns, and long-horizon/stale-output fixtures. The worker focused check has now passed; Root's independent check and final source/technical review remain open.

1. One pure helper parses an exact six-field observation and seven-field boot event with duplicate/unknown/missing-key rejection. It canonicalizes reviewed host/boot fields, hashes them, and returns `r139-hb2:` plus a derived digest only when both digests match the reviewed proof.
2. Raw seven-digit UTC values are converted to integer 100 ns ticks without discarding the seventh digit. Parent six-digit UTC and monotonic clocks enforce the reviewed two-second drift, five-second query, two-second CIM/event/current-baseline, and 120-second event-create bounds. The helper rejects boolean/nonfinite numeric values, bad UTC grammar, time reversal, stale output, future boot data, and placeholder host identity.
3. The successor preclock requires committed recovery admission, contract, capability receipt and proof refs. It issues the admitted exact one PowerShell argv via the existing child custody path before reservation, stores raw output only in ignored logs, captures parent UTC/monotonic endpoints, and validates the pure helper result. No second query/fallback is introduced.
4. Original freshGrant/inheritedOwnerEnvelope/observation cap checks, source-nine/tree/binary/tool pins, actors, protected resources, canonical templates, hooks, clean HEAD, reservation, signal, and original +1020/+1200 monitor remain inherited. The reservation `hostBootId` is the derived token and must equal the later Root-owned Runtime Effect Admission `rootPins.hostBootId` for the unchanged recorder design.
5. The test file is one of SOURCE_NINE. Root must issue a new accepted-build pin record with its changed SHA and new source commit while preserving all other source/binary/tool/tree pins. The new design needs a separate exact-byte technical pin. Root also owns the new recipe refs `bootRecoveryAdmission`, `bootIdentityContract`, and `bootProof`, exclusive new log/control paths, nonce/ACK rebinding, continuity and all commits.
6. Worker focused Python1/1 is consumed with the PASS receipt below; Root Python0/1 remains. Old eight attempts remain spent and the original preclock exit1 remains immutable. No Go rebuild/test is requested.

## Actual reserved worker check

Root created `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/boot_identity_v2_source_checks/worker_python_01_reservation.json` before launch, binding the recovery admission, pinned Python interpreter, exact `-B scripts/tests/test_r139_execution_supervisor.py` argv, source hashes, one worker slot, project CWD and 120-second timeout. The one original child PID 20756 returned from `communicate`, exit 0; 45 tests ran in 9.488 seconds and reported `OK`. The outer held process span was 11.369631 monotonic seconds. The result is `docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/boot_identity_v2_source_checks/worker_python_01_result.json` SHA-256 `c2d583c703996d8bc3158709aa27994dc622964cc91e02601b14b37545bb8bab` with exactly the agreed receipt fields. Private original stdout/stderr are under `.cvf/runtime/r139_boot_identity_recovery_20261025/worker_python_01/` with SHA-256 `b25ac0cc3753b14354f337ea451f96d0f23b9844004dfa5b90fb60f60733a044` and `fc2ba6659175b545fc984936cc2ee5fc147f47dcc51373c7e1cbef7ff2304684`; private original return SHA-256 `098f79996f4d7a66998c8a34c26c2cd077d2a084d08148ad5cb28cd77601d8c4`.

The outer orchestration shell raised an `AttributeError` while formatting its final summary after it had durably written the PASS result and original raw ledger. It did not affect the child exit or test evidence. No second child was started, no retry occurred, and no additional source check is authorized for this worker.

## Full static diff: successor versus frozen activation design

```diff
--- docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/reentry_activation_design.py.txt
+++ docs/reviews/probes/r139_integrated_20261021/native_operational_20261024/reentry_activation_boot_identity_v2_design.py.txt
@@ -30,6 +30,7 @@
 
 import hashlib, json, math, os, re, stat, subprocess, sys, time
 from datetime import datetime, timedelta, timezone
+from fractions import Fraction
 from pathlib import Path
 
 ROOT = Path(r"D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor-AI")
@@ -44,6 +45,10 @@
 RECEIPT = PROBE + "/reentry_activation_receipt.json"
 OBS_REF = {"path": "CVF_SESSION/authority/CCMAI-RUNTIME-139-CUA-OBSERVATION-CAPS-ADMISSION.json",
  "sha256": "fa7f2f344c75b6900f1fb65d74c9327e9e53d1957714226d880fa5a30eab9491"}
+BOOT_RECOVERY_REF = {"path": "CVF_SESSION/authority/CCMAI-RUNTIME-139-BOOT-IDENTITY-RECOVERY-ADMISSION.json",
+ "sha256": "ab4b4d148b52992d562b665baae6f96e9c812d987c4d197de0024272e317ffdb"}
+BOOT_CONTRACT_REF = {"path": PROBE + "/boot_identity_v2_contract_20261025.json",
+ "sha256": "4c90053543f1d21532fdb5c6c7487e04c7a4ee5b3369c6f24034cde285eb9e7c"}
 ALLOWED = set(CANON + CATALOG + (RECEIPT,))
 SOURCE_NINE = {"backend/api/handlers/intervention_execution_browser_test.go",
  "scripts/r139_execution_supervisor.py", "scripts/r139_vite_config.mjs.template",
@@ -62,6 +67,179 @@
 
 def hold(reason): raise RuntimeError(reason)
 def sha(data): return hashlib.sha256(data).hexdigest()
+def strict_json(raw):
+    """Parse one private observation, rejecting duplicate keys at every depth."""
+    def unique(pairs):
+        result = {}
+        for key, value in pairs:
+            if key in result: hold("BOOT_DUPLICATE_KEY")
+            result[key] = value
+        return result
+    try: return json.loads(raw.decode("utf-8", errors="strict"), object_pairs_hook=unique)
+    except (UnicodeError, ValueError, TypeError): hold("BOOT_JSON_INVALID")
+
+def utc_v2(value, parent=False):
+    if not isinstance(value, str): hold("BOOT_UTC_TYPE")
+    pattern = (r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}\+00:00" if parent else
+               r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{7}(?:Z|\+00:00)")
+    if not re.fullmatch(pattern, value, flags=re.ASCII): hold("BOOT_UTC_GRAMMAR")
+    try:
+        normalized = value.replace("Z", "+00:00")
+        if not parent:
+            prefix, tail = normalized.split(".", 1)
+            normalized = prefix + "." + tail[:6] + tail[7:]
+        parsed = datetime.fromisoformat(normalized)
+        if parsed.utcoffset() != timedelta(0): hold("BOOT_UTC_OFFSET")
+        # datetime retains microseconds only. Preserve the seventh raw digit
+        # explicitly so a 2.0000001 s boundary cannot round down to 2 s.
+        epoch = datetime(1970, 1, 1, tzinfo=timezone.utc)
+        delta = parsed - epoch
+        last_tick = 0 if parent else int(value.split(".", 1)[1][6])
+        return ((delta.days * 86400 + delta.seconds) * 10_000_000 +
+                delta.microseconds * 10 + last_tick)
+    except ValueError: hold("BOOT_UTC_INVALID")
+
+def numeric_v2(value):
+    if type(value) not in (int, float) or not math.isfinite(value): hold("BOOT_MONOTONIC_INVALID")
+    return value
+
+def validate_boot_identity_v2(proof, contract, actual_raw, parent_clock):
+    """Pure validation; no process, file, network, or campaign effect."""
+    if (not isinstance(proof, dict) or not isinstance(contract, dict) or
+        not isinstance(parent_clock, dict) or
+        contract.get("kind") != "R139_WINDOWS_BOOT_IDENTITY_V2_CONTRACT" or
+        contract.get("status") != "REVIEWED_ACCEPTED_SPEC" or
+        contract.get("ownerAdmissionRef") != BOOT_RECOVERY_REF or
+        proof.get("kind") != "ROOT_CURRENT_WINDOWS_BOOT_IDENTITY_PROOF_V2" or
+        proof.get("status") != "REVIEWED_CAPABILITY_BASELINE" or
+        proof.get("ownerAdmissionRef") != BOOT_RECOVERY_REF or
+        proof.get("contractRef") != BOOT_CONTRACT_REF or
+        proof.get("rootVerifiedReadOnlyOSMetadata") is not True): hold("BOOT_CONTRACT_OR_PROOF")
+    expected_bounds = {"maxAbsoluteCimBootVsEventStartSeconds": 2,
+      "minEventCreatedVsStartSeconds": 0, "maxEventCreatedVsStartSeconds": 120,
+      "maxAbsoluteCurrentVsBaselineCimBootSeconds": 2,
+      "maxAbsoluteParentUtcVsMonotonicElapsedDriftSeconds": 2,
+      "queryObservedUtcSlackSeconds": 2, "queryTimeoutSeconds": 5}
+    if contract.get("bounds") != expected_bounds: hold("BOOT_BOUNDS_DRIFT")
+    actual = strict_json(actual_raw)
+    top = {"schema", "observedUTC", "osComputerName", "computerSystemProductUuid", "lastBootUTC", "bootEvent"}
+    evkeys = {"logName", "providerName", "eventId", "recordId", "machineName", "timeCreatedUTC", "startTimeUTC"}
+    if not isinstance(actual, dict) or set(actual) != top or actual["schema"] != "r139.windowsBootIdentityObservation@1":
+        hold("BOOT_OBSERVATION_SHAPE")
+    event = actual["bootEvent"]
+    if not isinstance(event, dict) or set(event) != evkeys: hold("BOOT_EVENT_SHAPE")
+    name, uid, machine = actual["osComputerName"], actual["computerSystemProductUuid"], event["machineName"]
+    if any(not isinstance(v, str) or not v.isascii() for v in (name, uid, machine)):
+        hold("BOOT_HOST_TYPE")
+    name, uid, machine = name.strip().upper(), uid.strip().upper(), machine.strip().upper()
+    if (not re.fullmatch(r"[A-Z0-9][A-Z0-9._-]{0,254}", name, flags=re.ASCII) or
+        not re.fullmatch(r"[0-9A-F]{8}(?:-[0-9A-F]{4}){3}-[0-9A-F]{12}", uid, flags=re.ASCII) or
+        uid in ("00000000-0000-0000-0000-000000000000", "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF") or
+        machine != name): hold("BOOT_HOST_INVALID")
+    if (event["logName"] != "System" or event["providerName"] != "Microsoft-Windows-Kernel-General" or
+        type(event["eventId"]) is not int or event["eventId"] != 12 or
+        type(event["recordId"]) is not int or event["recordId"] <= 0):
+        hold("BOOT_EVENT_INVALID")
+    def digest(obj):
+        return sha(json.dumps(obj, ensure_ascii=True, sort_keys=True, separators=(",", ":")).encode("utf-8"))
+    host_sha = digest({"osComputerName": name, "computerSystemProductUuid": uid})
+    boot_sha = digest({"logName": event["logName"], "providerName": event["providerName"],
+      "eventId": event["eventId"], "recordId": event["recordId"], "machineName": machine,
+      "timeCreatedUTC": event["timeCreatedUTC"], "startTimeUTC": event["startTimeUTC"]})
+    stable = "r139-hb2:" + sha((host_sha + "|" + boot_sha).encode("ascii"))
+    if (host_sha != proof.get("hostIdentitySha256") or boot_sha != proof.get("bootSessionSha256") or
+        stable != proof.get("hostBootId")): hold("BOOT_SESSION_OR_HOST_DRIFT")
+    observed = utc_v2(actual["observedUTC"])
+    cim, started, created = (utc_v2(actual["lastBootUTC"]), utc_v2(event["startTimeUTC"]),
+                             utc_v2(event["timeCreatedUTC"]))
+    baseline_cim = utc_v2(proof.get("lastBootUTC"))
+    baseline_start = utc_v2(proof.get("bootEventStartUTC"))
+    baseline_created = utc_v2(proof.get("bootEventCreatedUTC"))
+    second = 10_000_000
+    if (started != baseline_start or created != baseline_created or
+        abs(cim-started) > 2*second or
+        not 0 <= created-started <= 120*second or
+        abs(cim-baseline_cim) > 2*second or
+        max(cim, started, created) > observed): hold("BOOT_TIMESTAMP_BOUND")
+    base = proof.get("baselineClock")
+    if base != contract.get("baselineClockFromOriginalIntent") or not isinstance(base, dict):
+        hold("BOOT_BASELINE_CLOCK")
+    if set(parent_clock) != {"startUTC", "startMonotonic", "endUTC", "endMonotonic"}:
+        hold("BOOT_PARENT_SHAPE")
+    bs, be = utc_v2(base.get("startUTC"), True), utc_v2(base.get("endUTC"), True)
+    ps, pe = utc_v2(parent_clock["startUTC"], True), utc_v2(parent_clock["endUTC"], True)
+    bm0, bm1 = numeric_v2(base.get("startMonotonic")), numeric_v2(base.get("endMonotonic"))
+    pm0, pm1 = numeric_v2(parent_clock["startMonotonic"]), numeric_v2(parent_clock["endMonotonic"])
+    if (base.get("monotonicSource") != "python.time.monotonic" or
+        base.get("pythonPath") != proof.get("pythonPath") or
+        not 0 <= be-bs <= 5*second or not 0 <= Fraction(bm1)-Fraction(bm0) <= 5 or
+        not 0 <= pe-ps <= 5*second or not 0 <= Fraction(pm1)-Fraction(pm0) <= 5 or
+        ps < bs or pm0 < bm0 or pe < ps or pm1 < pm0): hold("BOOT_CLOCK_REVERSED")
+    for utc_point, mono_point in ((ps, pm0), (pe, pm1)):
+        if abs(Fraction(utc_point-bs, second) - (Fraction(mono_point)-Fraction(bm0))) > 2:
+            hold("BOOT_CLOCK_DRIFT")
+    if not ps-2*second <= observed <= pe+2*second:
+        hold("BOOT_OBSERVED_OUTSIDE_PARENT")
+    return stable
+
+def reviewed_boot_recovery_checks(r, build, pins):
+    """Bind the new 1+1 check receipts and two independent reviews."""
+    v = build.get("bootRecoveryVerification")
+    if (not isinstance(v, dict) or v.get("ownerAdmissionRef") != BOOT_RECOVERY_REF or
+        v.get("contractRef") != BOOT_CONTRACT_REF or v.get("sourceChecksAccepted") is not True):
+        hold("BOOT_VERIFICATION_MISSING")
+    source_rel = PROBE + "/reentry_activation_boot_identity_v2_design.py.txt"
+    source_sha = sha(read(inside(source_rel, {source_rel})))
+    if git("show", "HEAD:" + source_rel) != read(inside(source_rel, {source_rel})):
+        hold("BOOT_DESIGN_NOT_COMMITTED")
+    test_rel = "scripts/tests/test_r139_execution_supervisor.py"
+    test_sha = pins["sourceSha256"].get(test_rel)
+    if not re.fullmatch(r"[0-9a-f]{64}", test_sha or ""):
+        hold("BOOT_TEST_PIN_MISSING")
+    required = {"kind", "actor", "status", "attempt", "cap", "originalWaitReturned",
+                "exitCode", "designSha256", "testSha256", "stdoutSha256", "stderrSha256",
+                "stdoutPath", "stderrPath", "startedAtUTC", "endedAtUTC"}
+    for actor in ("worker", "root"):
+        slot = v.get(actor + "FocusedPython")
+        if (not isinstance(slot, dict) or set(slot) != {"used", "cap", "outcome", "receiptRef"} or
+            type(slot["used"]) is not int or slot["used"] != 1 or
+            type(slot["cap"]) is not int or slot["cap"] != 1 or slot["outcome"] != "PASS"):
+            hold("BOOT_CHECK_SLOT")
+        ref = slot["receiptRef"]
+        if (not isinstance(ref, dict) or set(ref) != {"path", "sha256"} or
+            not isinstance(ref["path"], str) or
+            not ref["path"].startswith(PROBE + "/boot_identity_v2_source_checks/") or
+            not ref["path"].endswith(".json")):
+            hold("BOOT_CHECK_RECEIPT_REF")
+        receipt = committed(ref["path"], ref["sha256"])
+        if (set(receipt) != required or receipt.get("kind") != "R139_BOOT_IDENTITY_V2_FOCUSED_PYTHON_CHECK" or
+            receipt.get("actor") != actor or receipt.get("status") != "PASS" or
+            type(receipt.get("attempt")) is not int or receipt["attempt"] != 1 or
+            type(receipt.get("cap")) is not int or receipt["cap"] != 1 or
+            receipt.get("originalWaitReturned") is not True or
+            type(receipt.get("exitCode")) is not int or receipt["exitCode"] != 0 or
+            receipt.get("designSha256") != source_sha or receipt.get("testSha256") != test_sha):
+            hold("BOOT_CHECK_RECEIPT_CONTENT")
+        for key in ("stdoutSha256", "stderrSha256"):
+            if not isinstance(receipt[key], str) or not re.fullmatch(r"[0-9a-f]{64}", receipt[key]):
+                hold("BOOT_CHECK_OUTPUT_HASH")
+        for key in ("stdoutPath", "stderrPath"):
+            if (not isinstance(receipt[key], str) or
+                not receipt[key].startswith(".cvf/runtime/r139_") or
+                not receipt[key].endswith(".bin")):
+                hold("BOOT_CHECK_PRIVATE_PATH")
+        for key in ("startedAtUTC", "endedAtUTC"):
+            if not isinstance(receipt[key], str) or not receipt[key].endswith("+00:00"):
+                hold("BOOT_CHECK_TIME_SHAPE")
+    def review(ref, path):
+        if not isinstance(ref, dict) or set(ref) != {"path", "sha256"} or ref.get("path") != path:
+            hold("BOOT_REVIEW_REF")
+        body = committed(path, ref["sha256"], parse=False)
+        if b"Status: REVIEW_PASS" not in body[:320]: hold("BOOT_REVIEW_STATUS")
+    review(v.get("independentSourceReviewRef"),
+           "docs/reviews/R139_BOOT_IDENTITY_RECOVERY_REVIEW_2026-10-25.md")
+    review(r.get("bootRecoveryTechnicalReviewRef"),
+           "docs/reviews/R139_BOOT_IDENTITY_G0_TECHNICAL_REVIEW_2026-10-25.md")
 def no_reparse(path):
     p = Path(path)
     if not p.is_absolute(): hold("ABSOLUTE_PATH_REQUIRED")
@@ -439,17 +617,62 @@
         inherited.get("historicalLineage") != "UNKNOWN"): hold("FRESH_GRANT")
     if not git("merge-base", "--is-ancestor", build["sourceCommit"], "HEAD") == b"":
         hold("BUILD_COMMIT_ANCESTRY")
+    if r.get("bootRecoveryAdmission") != BOOT_RECOVERY_REF or r.get("bootIdentityContract") != BOOT_CONTRACT_REF:
+        hold("BOOT_RECOVERY_REFS")
+    recovery = committed(BOOT_RECOVERY_REF["path"], BOOT_RECOVERY_REF["sha256"])
+    contract = committed(BOOT_CONTRACT_REF["path"], BOOT_CONTRACT_REF["sha256"])
+    source_cap = {"authoringPasses": 1, "consolidatedRepairPasses": 1,
+                  "workerFocusedPython": 1, "rootFocusedPython": 1,
+                  "goCompile": 0, "goTests": 0}
+    accounting = {"oldSpent": 8, "newCap": 2, "totalLineageCap": 10,
+                  "oldFailedAttemptRetained": True}
+    runtime_cap = {"oneUnspentCampaign": 1, "wholeCampaignSeconds": 1200,
+      "cleanupCutoffSeconds": 1020, "serviceQueries": 5, "cuaMethods": 64,
+      "cuaStages": [8, 4, 48, 4], "tabMetadataInventories": 3,
+      "ownedTabs": 1, "g2StatusStarts": 16, "physicalProviderCalls": 1,
+      "maximumUSD": 1, "automaticRetry": 0}
+    preclock_cap = {"additionalCap": 1, "oldAttempts": 1,
+      "oldAttemptOutcome": "EXIT1_CURRENT_HOST_BOOT_DRIFT_BEFORE_RESERVATION",
+      "newAttemptConditionalOn": ["usable identity capability", "accepted explicit identity contract",
+        "independent R2 source and packet review", "two bounded focused check receipts",
+        "exact clean normal commit", "fresh exclusive paths and original Popen/wait/exit custody"],
+      "oldAttemptNeverResetOrRetried": True}
+    if (recovery.get("sourceBudget") != source_cap or recovery.get("checkAccounting") != accounting or
+        recovery.get("unchangedRuntimeCaps") != runtime_cap or
+        recovery.get("preclockInvocation") != preclock_cap or
+        recovery.get("oneExistingOrder") is not True or recovery.get("newTranches") != 0):
+        hold("BOOT_RECOVERY_EFFECT_BOUNDARY")
+    reviewed_boot_recovery_checks(r, build, pins)
     boot = committed(r["bootProof"]["path"], r["bootProof"]["sha256"])
-    if (boot.get("hostBootId") != r["bootProof"]["hostBootId"] or
+    if (r["bootProof"]["path"] != PROBE + "/boot_identity_v2_proof_20261025.json" or
+        boot.get("hostBootId") != r["bootProof"].get("hostBootId") or
         boot.get("pythonPath") != pins["tools"]["python"]["path"] or
-        boot.get("rootVerifiedReadOnlyOSMetadata") is not True): hold("BOOT_PROOF")
-    os_query = ("$x=Get-CimInstance Win32_OperatingSystem -ErrorAction Stop; "
-                "$h=$env:COMPUTERNAME; "
-                "$b=$x.LastBootUpTime.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffffffZ'); "
-                "Write-Output ($h+'|'+$b)")
-    actual_boot = child([pins["tools"]["powershell"]["path"], "-NoProfile", "-NonInteractive",
-                         "-Command", os_query], 5).decode("utf-8").strip()
-    if actual_boot != boot["hostBootId"]: hold("CURRENT_HOST_BOOT_DRIFT")
+        boot.get("capabilityReceiptRef") != {"path": PROBE + "/cua_boot_identity_capability_20261025.json",
+          "sha256": "284f47404e8f8e40c5ff1f42bebaf2276dc8c1390914e56b67521a476e90346e"}):
+        hold("BOOT_PROOF")
+    committed(boot["capabilityReceiptRef"]["path"], boot["capabilityReceiptRef"]["sha256"])
+    observation = recovery.get("metadataObservation", {})
+    argv = observation.get("argv")
+    if (recovery.get("kind") != "OWNER_APPROVED_R139_BOOT_IDENTITY_RECOVERY" or
+        recovery.get("ownerApproved") is not True or recovery.get("trancheId") != "CCMAI-RUNTIME-139" or
+        observation.get("cap") != 1 or observation.get("timeoutSeconds") != 5 or
+        observation.get("automaticRetry") != 0 or not isinstance(argv, list) or len(argv) != 5 or
+        argv[:4] != [pins["tools"]["powershell"]["path"], "-NoProfile", "-NonInteractive", "-Command"] or
+        observation.get("powershellPin") != pins["tools"]["powershell"]):
+        hold("BOOT_QUERY_ADMISSION")
+    # The admission's exact command is one additional argv element after -Command.
+    if (len(argv) != 5 or not isinstance(argv[4], str) or
+        sha(argv[4].encode("utf-8")) != observation.get("queryScriptSha256")):
+        hold("BOOT_QUERY_SCRIPT_PIN")
+    query_start_utc = datetime.now(timezone.utc).isoformat(timespec="microseconds")
+    query_start_mono = time.monotonic()
+    actual_raw = child(argv, 5)
+    query_end_mono = time.monotonic()
+    query_end_utc = datetime.now(timezone.utc).isoformat(timespec="microseconds")
+    actual_boot = validate_boot_identity_v2(boot, contract, actual_raw,
+      {"startUTC": query_start_utc, "startMonotonic": query_start_mono,
+       "endUTC": query_end_utc, "endMonotonic": query_end_mono})
+    if actual_boot != boot["hostBootId"]: hold("BOOT_DERIVED_ID_DRIFT")
     cua_tasks = []
     for who in ("root", "internal", "cuaProducer", "cuaObserver"):
         item = r["actorAcks"][who]; ack = committed(item["path"], item["sha256"])
```

## Full static diff: focused Python fixture

```diff
--- HEAD:scripts/tests/test_r139_execution_supervisor.py
+++ scripts/tests/test_r139_execution_supervisor.py
@@ -33,6 +33,157 @@
 
 
 class R139SupervisorPureTests(unittest.TestCase):
+    def test_boot_identity_v2_pure_contract_no_process(self):
+        design_root = SOURCE.parents[1] / "docs/reviews/probes/r139_integrated_20261021/native_operational_20261024"
+        def no_process(*args, **kwargs):
+            raise AssertionError("boot identity pure helper attempted a subprocess")
+        with patch.object(subprocess, "Popen", side_effect=no_process), patch.object(
+            subprocess, "run", side_effect=no_process
+        ), patch.object(subprocess, "call", side_effect=no_process), patch.object(
+            subprocess, "check_call", side_effect=no_process
+        ), patch.object(subprocess, "check_output", side_effect=no_process):
+            design = runpy.run_path(str(design_root / "reentry_activation_boot_identity_v2_design.py.txt"),
+                                    run_name="r139_inert_boot_identity_v2_definitions")
+            validate = design["validate_boot_identity_v2"]
+            contract = json.loads((design_root / "boot_identity_v2_contract_20261025.json").read_text(encoding="utf-8"))
+            base = contract["baselineClockFromOriginalIntent"]
+            event = {"logName": "System", "providerName": "Microsoft-Windows-Kernel-General",
+                     "eventId": 12, "recordId": 123456, "machineName": "TEST-HOST",
+                     "timeCreatedUTC": "2026-10-23T05:48:28.0990423Z",
+                     "startTimeUTC": "2026-10-23T05:48:27.5000000Z"}
+            observation = {"schema": "r139.windowsBootIdentityObservation@1",
+                           "observedUTC": "2026-10-25T08:40:24.9665051Z",
+                           "osComputerName": " test-host ",
+                           "computerSystemProductUuid": "12345678-1234-1234-1234-123456789ABC",
+                           "lastBootUTC": "2026-10-23T05:48:27.5037950Z",
+                           "bootEvent": event}
+            canonical = lambda value: json.dumps(value, ensure_ascii=True, sort_keys=True,
+                                                 separators=(",", ":")).encode("utf-8")
+            host_sha = hashlib.sha256(canonical({"osComputerName": "TEST-HOST",
+                "computerSystemProductUuid": observation["computerSystemProductUuid"]})).hexdigest()
+            boot_sha = hashlib.sha256(canonical(event)).hexdigest()
+            stable = "r139-hb2:" + hashlib.sha256((host_sha + "|" + boot_sha).encode("ascii")).hexdigest()
+            proof = {"kind": "ROOT_CURRENT_WINDOWS_BOOT_IDENTITY_PROOF_V2",
+                     "status": "REVIEWED_CAPABILITY_BASELINE", "rootVerifiedReadOnlyOSMetadata": True,
+                     "ownerAdmissionRef": design["BOOT_RECOVERY_REF"],
+                     "contractRef": design["BOOT_CONTRACT_REF"], "hostIdentitySha256": host_sha,
+                     "bootSessionSha256": boot_sha, "hostBootId": stable,
+                     "lastBootUTC": observation["lastBootUTC"],
+                     "bootEventStartUTC": event["startTimeUTC"],
+                     "bootEventCreatedUTC": event["timeCreatedUTC"],
+                     "baselineClock": base, "pythonPath": base["pythonPath"]}
+            parent_start = datetime.fromisoformat(base["startUTC"]) + timedelta(seconds=2)
+            parent = {"startUTC": parent_start.isoformat(timespec="microseconds"),
+                      "startMonotonic": base["startMonotonic"] + 2,
+                      "endUTC": (parent_start + timedelta(seconds=0.5)).isoformat(timespec="microseconds"),
+                      "endMonotonic": base["startMonotonic"] + 2.5}
+            raw = lambda value: canonical(value)
+            self.assertEqual(validate(proof, contract, raw(observation), parent), stable)
+            # The observed 2.743 ms raw CIM variation qualifies only with the
+            # same host and boot-session digests and independent clock bounds.
+            jitter = dict(observation, lastBootUTC="2026-10-23T05:48:27.5010520Z")
+            self.assertEqual(validate(proof, contract, raw(jitter), parent), stable)
+            # Inclusive policy edges are deliberate: 2 s from event start,
+            # 2 s from baseline CIM, and 120 s event creation delay.
+            edge_event = dict(event, timeCreatedUTC="2026-10-23T05:50:27.5000000Z")
+            edge_observation = dict(observation, bootEvent=edge_event,
+                                    lastBootUTC="2026-10-23T05:48:29.5000000Z")
+            edge_boot_sha = hashlib.sha256(canonical(edge_event)).hexdigest()
+            edge_proof = dict(proof, bootSessionSha256=edge_boot_sha,
+                              hostBootId="r139-hb2:" + hashlib.sha256((host_sha + "|" + edge_boot_sha).encode("ascii")).hexdigest(),
+                              bootEventCreatedUTC=edge_event["timeCreatedUTC"])
+            self.assertEqual(validate(edge_proof, contract, raw(edge_observation), parent),
+                             edge_proof["hostBootId"])
+            failures = [
+                (raw(dict(observation, osComputerName="OTHER-HOST")), parent),
+                (raw(dict(observation, bootEvent=dict(event, recordId=123457))), parent),
+                (raw(dict(observation, lastBootUTC="2026-10-23T05:48:29.5000001Z")), parent),
+                (raw(dict(observation, lastBootUTC="2026-10-23T05:48:29.5037951Z")), parent),
+                (raw(dict(observation, observedUTC="2026-10-25T08:40:21.0000000Z")), parent),
+                (raw(dict(observation, lastBootUTC="2026-10-26T05:48:27.5037950Z")), parent),
+                (raw(dict(observation, bootEvent=dict(event, timeCreatedUTC="2026-10-23T05:50:27.5000001Z"))), parent),
+                (raw(dict(observation, bootEvent=dict(event, eventId=True))), parent),
+                (raw(dict(observation, bootEvent=dict(event, recordId=True))), parent),
+                (raw(dict(observation, computerSystemProductUuid="00000000-0000-0000-0000-000000000000")), parent),
+                (raw({k: v for k, v in observation.items() if k != "osComputerName"}), parent),
+                (raw(dict(observation, unknown=True)), parent),
+                (raw(observation).replace(b'"observedUTC":', b'"observedUTC":"2026-10-25T08:40:24.0000000Z","observedUTC":', 1), parent),
+                (raw(observation).replace(b'"eventId":', b'"eventId":12,"eventId":', 1), parent),
+                (raw(dict(observation, bootEvent=dict(event, unknown=True))), parent),
+                (raw(observation), dict(parent, startMonotonic=True)),
+                (raw(observation), dict(parent, endUTC=(parent_start-timedelta(seconds=1)).isoformat(timespec="microseconds"))),
+                (raw(observation), dict(parent, startMonotonic=base["startMonotonic"]+9)),
+            ]
+            for index, (bad_raw, bad_parent) in enumerate(failures):
+                with self.subTest(case=index), self.assertRaises(RuntimeError):
+                    validate(proof, contract, bad_raw, bad_parent)
+            def rebound(candidate):
+                host = {"osComputerName": candidate["osComputerName"].strip().upper(),
+                        "computerSystemProductUuid": candidate["computerSystemProductUuid"].strip().upper()}
+                hs = hashlib.sha256(canonical(host)).hexdigest()
+                bs = hashlib.sha256(canonical(candidate["bootEvent"])).hexdigest()
+                return dict(proof, hostIdentitySha256=hs, bootSessionSha256=bs,
+                    hostBootId="r139-hb2:" + hashlib.sha256((hs+"|"+bs).encode("ascii")).hexdigest(),
+                    lastBootUTC=candidate["lastBootUTC"],
+                    bootEventStartUTC=candidate["bootEvent"]["startTimeUTC"],
+                    bootEventCreatedUTC=candidate["bootEvent"]["timeCreatedUTC"])
+            adversarial = [
+                (dict(observation, observedUTC="2026-10-25T08:40:24.966505Z"), "BOOT_UTC_GRAMMAR"),
+                (dict(observation, observedUTC="2026-10-25T08:40:24.96650511Z"), "BOOT_UTC_GRAMMAR"),
+                (dict(observation, observedUTC="2026-10-25T08:40:24.9665051+01:00"), "BOOT_UTC_GRAMMAR"),
+                (dict(observation, observedUTC="2026-02-30T08:40:24.9665051Z"), "BOOT_UTC_INVALID"),
+                (dict(observation, computerSystemProductUuid="FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF"), "BOOT_HOST_INVALID"),
+                (dict(observation, bootEvent=dict(event, logName="Other")), "BOOT_EVENT_INVALID"),
+                (dict(observation, bootEvent=dict(event, providerName="Other")), "BOOT_EVENT_INVALID"),
+                (dict(observation, bootEvent=dict(event, machineName="OTHER-HOST")), "BOOT_HOST_INVALID"),
+                (dict(observation, bootEvent=dict(event, timeCreatedUTC="2026-10-23T05:48:27.4999999Z")), "BOOT_TIMESTAMP_BOUND"),
+            ]
+            for index, (candidate, branch) in enumerate(adversarial):
+                with self.subTest(adversarial=index), self.assertRaisesRegex(RuntimeError, branch):
+                    validate(rebound(candidate), contract, raw(candidate), parent)
+            for numeric in (float("nan"), float("inf"), float("-inf"), True):
+                with self.subTest(monotonic=numeric), self.assertRaisesRegex(RuntimeError, "BOOT_MONOTONIC_INVALID"):
+                    validate(proof, contract, raw(observation), dict(parent, endMonotonic=numeric))
+            exact_drift = dict(parent, startMonotonic=base["startMonotonic"]+4.0,
+                               endMonotonic=base["startMonotonic"]+4.5)
+            self.assertEqual(validate(proof, contract, raw(observation), exact_drift), stable)
+            just_over = dict(exact_drift, startMonotonic=base["startMonotonic"]+4.00000004,
+                             endMonotonic=base["startMonotonic"]+4.50000004)
+            with self.assertRaisesRegex(RuntimeError, "BOOT_CLOCK_DRIFT"):
+                validate(proof, contract, raw(observation), just_over)
+            end_only_over = dict(exact_drift, endMonotonic=base["startMonotonic"]+4.50000004)
+            with self.assertRaisesRegex(RuntimeError, "BOOT_CLOCK_DRIFT"):
+                validate(proof, contract, raw(observation), end_only_over)
+            long_start = datetime.fromisoformat(base["startUTC"]) + timedelta(seconds=600)
+            long_parent = {"startUTC": long_start.isoformat(timespec="microseconds"),
+                           "startMonotonic": base["startMonotonic"]+600,
+                           "endUTC": (long_start+timedelta(seconds=0.5)).isoformat(timespec="microseconds"),
+                           "endMonotonic": base["startMonotonic"]+600.5}
+            long_observed = dict(observation, observedUTC=(long_start+timedelta(seconds=0.3)).strftime("%Y-%m-%dT%H:%M:%S.%f")+"0Z")
+            self.assertEqual(validate(proof, contract, raw(long_observed), long_parent), stable)
+            with self.assertRaisesRegex(RuntimeError, "BOOT_OBSERVED_OUTSIDE_PARENT"):
+                validate(proof, contract, raw(observation), long_parent)
+            edge_over = dict(edge_observation, bootEvent=dict(edge_event,
+                              timeCreatedUTC="2026-10-23T05:50:27.5000001Z"))
+            edge_over_sha = hashlib.sha256(canonical(edge_over["bootEvent"])).hexdigest()
+            edge_over_proof = dict(edge_proof, bootSessionSha256=edge_over_sha,
+                hostBootId="r139-hb2:" + hashlib.sha256((host_sha + "|" + edge_over_sha).encode("ascii")).hexdigest(),
+                bootEventCreatedUTC=edge_over["bootEvent"]["timeCreatedUTC"])
+            with self.assertRaises(RuntimeError):
+                validate(edge_over_proof, contract, raw(edge_over), parent)
+            future_event = dict(event, startTimeUTC="2026-10-25T08:40:24.9665052Z",
+                                timeCreatedUTC="2026-10-25T08:40:24.9665052Z")
+            future_observation = dict(observation, bootEvent=future_event,
+                                      lastBootUTC=future_event["startTimeUTC"])
+            future_boot_sha = hashlib.sha256(canonical(future_event)).hexdigest()
+            future_proof = dict(proof, bootSessionSha256=future_boot_sha,
+                hostBootId="r139-hb2:" + hashlib.sha256((host_sha + "|" + future_boot_sha).encode("ascii")).hexdigest(),
+                lastBootUTC=future_observation["lastBootUTC"],
+                bootEventStartUTC=future_event["startTimeUTC"],
+                bootEventCreatedUTC=future_event["timeCreatedUTC"])
+            with self.assertRaises(RuntimeError):
+                validate(future_proof, contract, raw(future_observation), parent)
+
     def test_cua_identity_and_origin_no_io(self):
         identity = {"nativeContractId": "r139-cua", "trancheId": "CCMAI-RUNTIME-139",
                     "campaignNonce": "nonce-cua", "browserId": "2", "tabId": "tab-123",
```

## Root independent completion acknowledgment

Root independently reconciled worker PID20756/original exit0/45 tests and Root PID2416/original exit0/45 tests with their private raw outputs and public exact-shape receipts. Independent peer source review is frozen at `docs/reviews/R139_BOOT_IDENTITY_RECOVERY_REVIEW_2026-10-25.md`, SHA55a84ad99d6c1aae05fa117ae4f841f0e2bcc10ad9145b6a3cb9b1ad3294fff8. Total source checks10/10 consumed; old failures and worker post-write orchestration exception retained, no rerun. Source89d6/test496a unchanged. Offline component acceptance only; material commit binding pending and G0/runtime/key/provider not released.
