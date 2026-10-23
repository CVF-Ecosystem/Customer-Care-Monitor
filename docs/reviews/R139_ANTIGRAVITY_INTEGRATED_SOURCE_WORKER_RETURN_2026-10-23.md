# R139 Antigravity Integrated Source Worker Return — 2026-10-23

- **Tranche**: `CCMAI-RUNTIME-139`
- **Work Order**: `docs/work_orders/CCMAI_RUNTIME_139_INTEGRATED_EXECUTION_PROOF.md` (Section: "Antigravity integrated source dispatch — 2026-10-23")
- **Governing Authority**: `CVF_SESSION/authority/CCMAI-RUNTIME-139-ANTIGRAVITY-INTEGRATED-SOURCE-ADMISSION.json`
- **Dispatch HEAD**: `ea6b4fdf3e1fb7e6c03ae401b1bdd8f5312b2361`
- **Role**: `IMPLEMENTATION_WORKER` (Antigravity). Root holds `ORCHESTRATOR`, `REVIEWER`, and `COMMIT_STEWARD`.
- **Status**: `CHANGES_REQUIRED_SOURCE_CHECK_DEFECT`
- **Product B**: `BLOCKED_PENDING_OPERATIONAL_ADMISSION`
- **Evidence Root**: `docs/reviews/probes/r139_integrated_20261021/antigravity_integrated_source_20261023/`
- **Runtime Root**: `.cvf/runtime/r139_antigravity_integrated_source_20261023/`

---

## 1. Operating Context & Role Acknowledgment

In accordance with `AGENTS.md` First-Request Protocol and the mandatory governing admission:
- Antigravity acknowledges the role of `IMPLEMENTATION_WORKER` for the single admitted source stage of `CCMAI-RUNTIME-139`.
- Verified dispatch HEAD is exactly `ea6b4fdf3e1fb7e6c03ae401b1bdd8f5312b2361`.
- Verified `sourceBeforeSha256`, `protectedSourceSha256`, and tool hashes match admission bit-for-bit before authoring.
- No git commits, no edits to dispatcher authority, no edits to shared continuity/catalog/work-order files, no retired G0 rerun, and zero runtime/Docker/DB/fixture/browser/credential/provider invocations performed.

---

## 2. Summary of Implemented Changes

Exactly three authorized source paths were modified in this one authoring round:

1. **`scripts/r139_execution_supervisor.py`**:
   - Added native Antigravity branch to `verify_browser_receipt` alongside legacy IAB.
   - Enforced pre-fixture reservation contract (`status: "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE"`, `pageId`, `campaignNonce`, `browserSessionId`, `exclusiveReservation: True`, `controlObserved: True`, `captureObserved: True`, `closeObserved: False/absent`, `closeMode`, `operator`).
   - Disallowed early closure in pre-fixture reservation (`EARLY_CLOSE_FORBIDDEN`).
   - Disallowed terminal close receipt as pre-fixture reservation (`TERMINAL_CLOSE_RECEIPT_CANNOT_BE_PREFIXTURE_RESERVATION`).
   - Implemented `verify_browser_terminal_close(admission, runtime, external)` enforcing `CLOSED_EXACT_OWNED_PAGE`, `closeObserved: True`, `ownedTabsFinal: 0`, matching identities, UTC timestamp validation, and deadline checks.
   - Hooked `verify_browser_terminal_close` into `run()`'s `finally:` block so terminal close is checked on both success and ordinary error paths.

2. **`backend/api/handlers/intervention_execution_browser_test.go`**:
   - Extended `r139Admission` struct with `BrowserTool`, `BrowserSessionID`, `CampaignNonce`, `PageID`, `FallbackCloseOperator`, `BrowserCloseReceiptPath`, `BrowserCloseReceiptSHA256`.
   - Extracted `r139VerifyBrowserReservation` supporting both legacy IAB and native Antigravity pre-fixture reservation, called from `r139Admit`.
   - Extracted `r139VerifyBrowserTerminalClose` validating terminal close receipts.
   - Implemented `TestR139PureBrowserReservationAdmission(t *testing.T)` covering legacy IAB pass, native reservation pass, final close rejected as reservation, early close rejected, identity mismatches, protocol, closeMode, and admission missing close receipt path.
   - Implemented `TestR139PureBrowserTerminalClose(t *testing.T)` covering terminal close pass, missing/false closeObserved rejection, ownedTabsFinal != 0 rejection, wrong status rejection, identity mismatches, future timestamp rejection, and post-deadline rejection.

3. **`scripts/tests/test_r139_execution_supervisor.py`**:
   - Added `test_native_antigravity_reservation_admission_positive`.
   - Added `test_native_antigravity_early_close_rejected`.
   - Added `test_native_antigravity_final_close_cannot_be_prefixture_reservation`.
   - Added `test_native_antigravity_identity_mismatches_and_missing_close_path_rejected`.
   - Added `test_verify_browser_terminal_close_positive_and_rejections`.

---

## 3. Check Execution & Fail-Closed Stop Accounting

In accordance with admission preflight instructions:
> "Before checks write exclusive reservations with argv/tool/source pins and consume attempts on failure/timeout/unknown. Scratch paths only under ignored runtimeRoot; helper snippets may only orchestrate admitted checks/evidence, no persistent new source/controller. First failure stops dependent checks and preserves raw output; no automatic retry or source repair under this exhausted one-round cap. Report one consolidated defect/accounting set."

### 3.1 Focused Python Check (`workerFocusedPython`: 1/1 consumed)
- **Reservation**: `docs/reviews/probes/r139_integrated_20261021/antigravity_integrated_source_20261023/python_check_reservation.json`
- **Command**: `C:/Users/tiennm/AppData/Local/Programs/Python/Python313/python.exe -B scripts/tests/test_r139_execution_supervisor.py`
- **Cwd**: `D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor-AI`
- **Timing**: Started `2026-10-23T14:43:49.175683+00:00`, completed in `1.014s` (elapsed monotonic: `1.014s`)
- **Exit Code**: `1`
- **Result Summary**: 38 tests passed, 1 test failed (`test_verify_browser_terminal_close_positive_and_rejections`).
- **Raw Stderr Log**:
  ```text
  .................................F.....
  ======================================================================
  FAIL: test_verify_browser_terminal_close_positive_and_rejections (__main__.R139SupervisorPureTests.test_verify_browser_terminal_close_positive_and_rejections)
  ----------------------------------------------------------------------
  Traceback (most recent call last):
    File "D:\UNG DUNG AI\TOOL AI 2026\CVF-Workspace\Customer-Care-Monitor-AI\scripts\tests\test_r139_execution_supervisor.py", line 810, in test_verify_browser_terminal_close_positive_and_rejections
      with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_AFTER_DEADLINE"):
           ~~~~~~~~~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
  AssertionError: Hold not raised

  ----------------------------------------------------------------------
  Ran 39 tests in 0.696s

  FAILED (failures=1)
  ```

### 3.2 Root Cause Analysis of the Single Test Failure
In `scripts/r139_execution_supervisor.py`, lines 212-218:
```python
    deadline_str = admission.get("grantDeadlineUTC")
    if deadline_str and deadline_str.endswith("Z"):
        try:
            deadline_dt = datetime.fromisoformat(deadline_str[:-1] + "+00:00")
            require(closed_dt <= deadline_dt, "BROWSER_CLOSE_AFTER_DEADLINE")
        except Exception:
            pass
```
- **Defect**: The `require(closed_dt <= deadline_dt, "BROWSER_CLOSE_AFTER_DEADLINE")` assertion was placed inside the `try: ... except Exception: pass` block intended for catching timestamp parsing errors. Because `Hold` inherits from `RuntimeError` which inherits from `Exception`, the raised `Hold("BROWSER_CLOSE_AFTER_DEADLINE")` was swallowed by `except Exception: pass`, resulting in `AssertionError: Hold not raised`.
- **Proposed Exact Repair** (for Root review and next authorized repair round):
  ```python
    deadline_str = admission.get("grantDeadlineUTC")
    if deadline_str and deadline_str.endswith("Z"):
        try:
            deadline_dt = datetime.fromisoformat(deadline_str[:-1] + "+00:00")
        except (ValueError, TypeError):
            deadline_dt = None
        if deadline_dt is not None:
            require(closed_dt <= deadline_dt, "BROWSER_CLOSE_AFTER_DEADLINE")
  ```

### 3.3 Offline Go Compile Check (`workerOfflineGoCompile`: 0/1 consumed, HELD)
- Under the fail-closed rule ("First failure stops dependent checks and preserves raw output; no automatic retry or source repair under this exhausted one-round cap"), Go offline compile was **NOT executed**.
- Recorded status in `docs/reviews/probes/r139_integrated_20261021/antigravity_integrated_source_20261023/worker_compile_held.json`.
- `handlers.test.exe` binary was not compiled (`binarySha256: null`).
- No unauthorized retry was attempted.

### 3.4 Shared Pure Go Check Slot
- Unexecuted, custodied by Root for independent review as specified in the admission (`sharedPureGoSlotCustodiedByRoot: 1`).

---

## 4. File Accounting and SHA-256 Hashes

| Path | Pre-Edit SHA-256 (`sourceBeforeSha256`) | Post-Edit SHA-256 | Status |
| --- | --- | --- | --- |
| `scripts/r139_execution_supervisor.py` | `2382b07efe1aa235045ee0ae98e8c7b9ed26378a8233c911b254acbed6ed5053` | `85b6615383a31da65b55335768c9f1997d1f3a2228ea25218203547c116df8d7` | Modified (authorized) |
| `scripts/tests/test_r139_execution_supervisor.py` | `ed1d50de8c297130b6101e2de3f3def8376dfd8ddd3fe80dfd0dd24485d5b9ec` | `676aa7ee9e00aaaec9a5dde65f229d484f4aa972a72300d089c4d534e746a2b6` | Modified (authorized) |
| `backend/api/handlers/intervention_execution_browser_test.go` | `fe3f5441c5fe1875ad6a81972fef30b02a857337d9950193f302d6bcf326d6dd` | `183729dedc8490b7d1a3c9783a6aaa4f3d36cbcb66a5bc62f821ea078590e242` | Modified (authorized) |

### Protected Files Verification (Unchanged)
| Path | Pinned SHA-256 (`protectedSourceSha256`) | Current SHA-256 | Match |
| --- | --- | --- | --- |
| `scripts/r139_vite_config.mjs.template` | `82a1d9ae8576fcd46718945d0a5cb44eb963a2a14d436a16e2bcdfb72b26bee9` | `82a1d9ae8576fcd46718945d0a5cb44eb963a2a14d436a16e2bcdfb72b26bee9` | YES |
| `backend/go.mod` | `15fe056859327cce63780c0390c82f93a7fa3c421adaffc9c289cd5961e24716` | `15fe056859327cce63780c0390c82f93a7fa3c421adaffc9c289cd5961e24716` | YES |
| `backend/go.sum` | `a7a5ba1e28e08702c104401b436cb139db1cea922fd07fdec27f99fa8308d67b` | `a7a5ba1e28e08702c104401b436cb139db1cea922fd07fdec27f99fa8308d67b` | YES |

### Toolchain Verification
| Tool | Path | Pinned SHA-256 | Observed SHA-256 | Match |
| --- | --- | --- | --- | --- |
| Python | `C:/Users/tiennm/AppData/Local/Programs/Python/Python313/python.exe` | `a2a4eff8d0b0c845284c607d50a3b5b966ac5a3121736a2e38e165bd6644d9fe` | `a2a4eff8d0b0c845284c607d50a3b5b966ac5a3121736a2e38e165bd6644d9fe` | YES |
| Go | `C:/Users/tiennm/AppData/Local/Programs/Go-1.26.7/go/bin/go.exe` | `5463fe58fa999d74420f00ee1b36d31c3da90a57ff204159f159859567ad61fc` | `5463fe58fa999d74420f00ee1b36d31c3da90a57ff204159f159859567ad61fc` | YES |

---

## 5. Check Budget Accounting

| Budget Item | Admitted | Used | Remaining / Disposition |
| --- | --- | --- | --- |
| `workerAuthoringRounds` | 1 | 1 | 0 (exhausted) |
| `workerFocusedPython` | 1 | 1 | 0 (exit 1: 38 passed, 1 failed) |
| `workerOfflineGoCompile` | 1 | 0 | 1 (held due to prior check failure) |
| `sharedPureGoSlotCustodiedByRoot` | 1 | 0 | 1 (custodied by Root for independent review) |
| `rootFocusedPython` | 1 | 0 | 1 (Root custody) |
| `automaticRetry` | 0 | 0 | 0 (strictly prohibited and respected) |
| `sameRootCauseRepairRounds` | 0 | 0 | 0 (exhausted for this authoring round) |
| `runtimeEffects` (Docker/DB/fixture/browser/keys/provider) | 0 | 0 | 0 (NOT_RUN) |

---

## 6. Complete Source Diffs

```diff
diff --git a/backend/api/handlers/intervention_execution_browser_test.go b/backend/api/handlers/intervention_execution_browser_test.go
index fe3f544..183729d 100644
--- a/backend/api/handlers/intervention_execution_browser_test.go
+++ b/backend/api/handlers/intervention_execution_browser_test.go
@@ -39,33 +39,261 @@ import (
 type r139Admission struct {
-	TrancheID            string            `json:"trancheId"`
-	Kind                 string            `json:"kind"`
-	SourceCommit         string            `json:"sourceCommit"`
-	FixtureName          string            `json:"fixtureName"`
-	BrowserReceiptSHA256 string            `json:"browserReceiptSha256"`
-	BrowserReceiptPath   string            `json:"browserReceiptPath"`
-	GoTestBinarySHA256   string            `json:"goTestBinarySha256"`
-	SourceSHA256         map[string]string `json:"sourceSha256"`
-	RuntimeRoot          string            `json:"runtimeRoot"`
-	ExternalEvidenceDir  string            `json:"externalEvidenceDir"`
-	BrowserSupported     bool              `json:"browserSupported"`
-	OwnerApproved        bool              `json:"ownerApproved"`
-	NativeExecutePOST    int               `json:"nativeExecutePOST"`
-	BrowserExecutePOST   int               `json:"browserExecutePOST"`
-	GrantCap             int               `json:"grantCap"`
-	ProviderCap          int               `json:"providerCap"`
-	GrantDeadlineUTC     string            `json:"grantDeadlineUTC"`
-	Provider             string            `json:"provider"`
-	Model                string            `json:"model"`
-	Endpoint             string            `json:"endpoint"`
-	CompletionTokens     int               `json:"completionTokens"`
-	MaxRequestBytes      int64             `json:"maxRequestBytes"`
-	Thinking             bool              `json:"thinking"`
-	SyntheticNamespace   string            `json:"syntheticNamespace"`
-	IssuerRole           string            `json:"issuerRole"`
-	ActorRole            string            `json:"actorRole"`
-	ConfigRef            string            `json:"configRef"`
-	PromptVersion        string            `json:"promptVersion"`
-	PromptSHA256         string            `json:"promptSha256"`
-	BackendTree          string            `json:"backendTree"`
-	FrontendTree         string            `json:"frontendTree"`
+	TrancheID                 string            `json:"trancheId"`
+	Kind                      string            `json:"kind"`
+	SourceCommit              string            `json:"sourceCommit"`
+	FixtureName               string            `json:"fixtureName"`
+	BrowserTool               string            `json:"browserTool"`
+	BrowserReceiptSHA256      string            `json:"browserReceiptSha256"`
+	BrowserReceiptPath        string            `json:"browserReceiptPath"`
+	BrowserCloseReceiptPath   string            `json:"browserCloseReceiptPath"`
+	BrowserCloseReceiptSHA256 string            `json:"browserCloseReceiptSha256"`
+	CampaignNonce             string            `json:"campaignNonce"`
+	BrowserSessionID          string            `json:"browserSessionId"`
+	PageID                    string            `json:"pageId"`
+	FallbackCloseOperator     string            `json:"fallbackCloseOperator"`
+	GoTestBinarySHA256        string            `json:"goTestBinarySha256"`
+	SourceSHA256              map[string]string `json:"sourceSha256"`
+	RuntimeRoot               string            `json:"runtimeRoot"`
+	ExternalEvidenceDir       string            `json:"externalEvidenceDir"`
+	BrowserSupported          bool              `json:"browserSupported"`
+	OwnerApproved             bool              `json:"ownerApproved"`
+	NativeExecutePOST         int               `json:"nativeExecutePOST"`
+	BrowserExecutePOST        int               `json:"browserExecutePOST"`
+	GrantCap                  int               `json:"grantCap"`
+	ProviderCap               int               `json:"providerCap"`
+	GrantDeadlineUTC          string            `json:"grantDeadlineUTC"`
+	Provider                  string            `json:"provider"`
+	Model                     string            `json:"model"`
+	Endpoint                  string            `json:"endpoint"`
+	CompletionTokens          int               `json:"completionTokens"`
+	MaxRequestBytes           int64             `json:"maxRequestBytes"`
+	Thinking                  bool              `json:"thinking"`
+	SyntheticNamespace        string            `json:"syntheticNamespace"`
+	IssuerRole                string            `json:"issuerRole"`
+	ActorRole                 string            `json:"actorRole"`
+	ConfigRef                 string            `json:"configRef"`
+	PromptVersion             string            `json:"promptVersion"`
+	PromptSHA256              string            `json:"promptSha256"`
+	BackendTree               string            `json:"backendTree"`
+	FrontendTree              string            `json:"frontendTree"`
+}
+
+func r139VerifyBrowserReservation(a r139Admission, browserRaw []byte) error {
+	if !a.BrowserSupported {
+		return errors.New("browserSupported is false")
+	}
+	var proof map[string]any
+	if err := json.Unmarshal(browserRaw, &proof); err != nil {
+		return fmt.Errorf("invalid json in browser proof: %w", err)
+	}
+
+	tool := a.BrowserTool
+	if tool == "" || tool == "supported_CUA_IAB" {
+		status, _ := proof["status"].(string)
+		browserID, _ := proof["browserId"].(string)
+		ctrl, _ := proof["controlObserved"].(bool)
+		capt, _ := proof["captureObserved"].(bool)
+		closed, _ := proof["closeObserved"].(bool)
+		ownedTabs, _ := proof["ownedTabsFinal"].(float64)
+		proto, _ := proof["protocol"].(string)
+
+		if status != "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" ||
+			browserID != "iab" ||
+			!ctrl || !capt || !closed ||
+			ownedTabs != 0 ||
+			(proto != "http" && proto != "https") {
+			return errors.New("legacy IAB browser proof content incomplete")
+		}
+		return nil
+	}
+
+	if tool == "antigravity_native" {
+		status, _ := proof["status"].(string)
+		if status == "CLOSED_EXACT_OWNED_PAGE" {
+			return errors.New("final-close evidence rejected as prefixture reservation")
+		}
+		if status != "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE" {
+			return fmt.Errorf("unexpected reservation status: %s", status)
+		}
+
+		browserID, _ := proof["browserId"].(string)
+		if browserID != "antigravity_native" {
+			return fmt.Errorf("unexpected browserId: %s", browserID)
+		}
+
+		excl, _ := proof["exclusiveReservation"].(bool)
+		if !excl {
+			return errors.New("exclusiveReservation must be true")
+		}
+
+		ctrl, _ := proof["controlObserved"].(bool)
+		if !ctrl {
+			return errors.New("controlObserved must be true")
+		}
+
+		capt, _ := proof["captureObserved"].(bool)
+		if !capt {
+			return errors.New("captureObserved must be true")
+		}
+
+		if closed, ok := proof["closeObserved"].(bool); ok && closed {
+			return errors.New("early closeObserved is forbidden in reservation")
+		}
+
+		proto, _ := proof["protocol"].(string)
+		if proto != "http" && proto != "https" {
+			return fmt.Errorf("unsupported protocol: %s", proto)
+		}
+
+		closeMode, _ := proof["closeMode"].(string)
+		if closeMode != "api" && closeMode != "operator" && closeMode != "operator_fallback" {
+			return fmt.Errorf("invalid closeMode: %s", closeMode)
+		}
+
+		pageID, _ := proof["pageId"].(string)
+		if strings.TrimSpace(pageID) == "" {
+			return errors.New("pageId is required in reservation")
+		}
+		if a.PageID != "" && pageID != a.PageID {
+			return fmt.Errorf("pageId mismatch: expected %s, got %s", a.PageID, pageID)
+		}
+
+		nonce, _ := proof["campaignNonce"].(string)
+		if strings.TrimSpace(nonce) == "" {
+			return errors.New("campaignNonce is required in reservation")
+		}
+		if a.CampaignNonce != "" && nonce != a.CampaignNonce {
+			return fmt.Errorf("campaignNonce mismatch: expected %s, got %s", a.CampaignNonce, nonce)
+		}
+
+		sessionID, _ := proof["browserSessionId"].(string)
+		if strings.TrimSpace(sessionID) == "" {
+			return errors.New("browserSessionId is required in reservation")
+		}
+		if a.BrowserSessionID != "" && sessionID != a.BrowserSessionID {
+			return fmt.Errorf("browserSessionId mismatch: expected %s, got %s", a.BrowserSessionID, sessionID)
+		}
+
+		trancheID, _ := proof["trancheId"].(string)
+		if trancheID != "" && a.TrancheID != "" && trancheID != a.TrancheID {
+			return fmt.Errorf("trancheId mismatch: expected %s, got %s", a.TrancheID, trancheID)
+		}
+
+		op, _ := proof["operator"].(string)
+		if strings.TrimSpace(op) == "" {
+			return errors.New("operator is required in reservation")
+		}
+		expectedOp := a.FallbackCloseOperator
+		if a.FallbackCloseOperator == "" {
+			expectedOp = a.ActorRole
+		}
+		if a.FallbackCloseOperator != "" && op != a.FallbackCloseOperator {
+			return fmt.Errorf("operator mismatch: expected %s, got %s", expectedOp, op)
+		}
+
+		if strings.TrimSpace(a.BrowserCloseReceiptPath) == "" {
+			return errors.New("browserCloseReceiptPath is required in admission")
+		}
+
+		return nil
+	}
+
+	return fmt.Errorf("unsupported browserTool: %s", tool)
+}
+
+func r139VerifyBrowserTerminalClose(a r139Admission, closeRaw []byte) error {
+	var receipt map[string]any
+	if err := json.Unmarshal(closeRaw, &receipt); err != nil {
+		return fmt.Errorf("invalid json in terminal close receipt: %w", err)
+	}
+
+	status, _ := receipt["status"].(string)
+	if status != "CLOSED_EXACT_OWNED_PAGE" {
+		return fmt.Errorf("status must be CLOSED_EXACT_OWNED_PAGE, got %s", status)
+	}
+
+	browserID, _ := receipt["browserId"].(string)
+	if browserID != "antigravity_native" {
+		return fmt.Errorf("browserId must be antigravity_native, got %s", browserID)
+	}
+
+	closeObserved, _ := receipt["closeObserved"].(bool)
+	if !closeObserved {
+		return errors.New("closeObserved must be true")
+	}
+
+	ownedTabs, ok := receipt["ownedTabsFinal"].(float64)
+	if !ok || ownedTabs != 0 {
+		return fmt.Errorf("ownedTabsFinal must be 0, got %v", receipt["ownedTabsFinal"])
+	}
+
+	pageID, _ := receipt["pageId"].(string)
+	if strings.TrimSpace(pageID) == "" {
+		return errors.New("pageId is required in terminal close receipt")
+	}
+	if a.PageID != "" && pageID != a.PageID {
+		return fmt.Errorf("pageId mismatch: expected %s, got %s", a.PageID, pageID)
+	}
+
+	nonce, _ := receipt["campaignNonce"].(string)
+	if strings.TrimSpace(nonce) == "" {
+		return errors.New("campaignNonce is required in terminal close receipt")
+	}
+	if a.CampaignNonce != "" && nonce != a.CampaignNonce {
+		return fmt.Errorf("campaignNonce mismatch: expected %s, got %s", a.CampaignNonce, nonce)
+	}
+
+	sessionID, _ := receipt["browserSessionId"].(string)
+	if strings.TrimSpace(sessionID) == "" {
+		return errors.New("browserSessionId is required in terminal close receipt")
+	}
+	if a.BrowserSessionID != "" && sessionID != a.BrowserSessionID {
+		return fmt.Errorf("browserSessionId mismatch: expected %s, got %s", a.BrowserSessionID, sessionID)
+	}
+
+	closeMode, _ := receipt["closeMode"].(string)
+	if closeMode != "api" && closeMode != "operator" && closeMode != "operator_fallback" {
+		return fmt.Errorf("invalid closeMode: %s", closeMode)
+	}
+
+	op, _ := receipt["operator"].(string)
+	if strings.TrimSpace(op) == "" {
+		return errors.New("operator is required in terminal close receipt")
+	}
+	expectedOp := a.FallbackCloseOperator
+	if a.FallbackCloseOperator == "" {
+		expectedOp = a.ActorRole
+	}
+	if a.FallbackCloseOperator != "" && op != a.FallbackCloseOperator {
+		return fmt.Errorf("operator mismatch: expected %s, got %s", expectedOp, op)
+	}
+
+	closedAtStr, _ := receipt["closedAtUTC"].(string)
+	if strings.TrimSpace(closedAtStr) == "" || !strings.HasSuffix(closedAtStr, "Z") {
+		return errors.New("closedAtUTC must be non-empty ISO 8601 ending in Z")
+	}
+
+	closedAt, err := time.Parse(time.RFC3339Nano, closedAtStr)
+	if err != nil {
+		closedAt, err = time.Parse(time.RFC3339, closedAtStr)
+	}
+	if err != nil {
+		return fmt.Errorf("invalid closedAtUTC: %w", err)
+	}
+
+	now := time.Now().UTC()
+	if closedAt.After(now.Add(time.Minute)) {
+		return errors.New("closedAtUTC cannot be in the future")
+	}
+
+	if a.GrantDeadlineUTC != "" {
+		deadline, dErr := time.Parse(time.RFC3339Nano, a.GrantDeadlineUTC)
+		if dErr != nil {
+			deadline, dErr = time.Parse(time.RFC3339, a.GrantDeadlineUTC)
+		}
+		if dErr == nil && closedAt.After(deadline) {
+			return errors.New("closedAtUTC cannot be after grant deadline")
+		}
+	}
+
+	return nil
 }
@@ -89,2 +317,255 @@ func TestR139PureRouteAdmission(t *testing.T) {
 }
+
+func TestR139PureBrowserReservationAdmission(t *testing.T) {
+	iabAdmission := r139Admission{
+		BrowserSupported: true,
+		BrowserTool:      "supported_CUA_IAB",
+	}
+	validIABRaw := []byte(`{
+		"status": "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE",
+		"browserId": "iab",
+		"controlObserved": true,
+		"captureObserved": true,
+		"closeObserved": true,
+		"ownedTabsFinal": 0,
+		"protocol": "http"
+	}`)
+	if err := r139VerifyBrowserReservation(iabAdmission, validIABRaw); err != nil {
+		t.Fatalf("expected legacy IAB pass, got: %v", err)
+	}
+
+	nativeAdmission := r139Admission{
+		TrancheID:               "CCMAI-RUNTIME-139",
+		BrowserSupported:        true,
+		BrowserTool:             "antigravity_native",
+		PageID:                  "page-test-123",
+		CampaignNonce:           "nonce-abc-456",
+		BrowserSessionID:        "session-xyz-789",
+		FallbackCloseOperator:   "Owner",
+		BrowserCloseReceiptPath: "docs/reviews/probes/close.json",
+	}
+	validNativeRaw := []byte(`{
+		"status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
+		"browserId": "antigravity_native",
+		"exclusiveReservation": true,
+		"controlObserved": true,
+		"captureObserved": true,
+		"closeObserved": false,
+		"protocol": "http",
+		"closeMode": "operator_fallback",
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"trancheId": "CCMAI-RUNTIME-139",
+		"operator": "Owner"
+	}`)
+	if err := r139VerifyBrowserReservation(nativeAdmission, validNativeRaw); err != nil {
+		t.Fatalf("expected native reservation pass, got: %v", err)
+	}
+
+	finalCloseAsReservation := []byte(`{
+		"status": "CLOSED_EXACT_OWNED_PAGE",
+		"browserId": "antigravity_native",
+		"exclusiveReservation": true,
+		"controlObserved": true,
+		"captureObserved": true,
+		"closeObserved": true,
+		"ownedTabsFinal": 0,
+		"protocol": "http",
+		"closeMode": "operator_fallback",
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"operator": "Owner"
+	}`)
+	if err := r139VerifyBrowserReservation(nativeAdmission, finalCloseAsReservation); err == nil {
+		t.Fatal("expected rejection of final-close receipt as prefixture reservation")
+	}
+
+	earlyCloseRaw := []byte(`{
+		"status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
+		"browserId": "antigravity_native",
+		"exclusiveReservation": true,
+		"controlObserved": true,
+		"captureObserved": true,
+		"closeObserved": true,
+		"protocol": "http",
+		"closeMode": "operator_fallback",
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"operator": "Owner"
+	}`)
+	if err := r139VerifyBrowserReservation(nativeAdmission, earlyCloseRaw); err == nil {
+		t.Fatal("expected rejection of early closeObserved=true in reservation")
+	}
+
+	for _, tc := range []struct {
+		name string
+		raw  []byte
+	}{
+		{"wrong-pageId", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"wrong-page","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","operator":"Owner"}`)},
+		{"wrong-nonce", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"wrong-nonce","browserSessionId":"session-xyz-789","operator":"Owner"}`)},
+		{"wrong-session", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"wrong-session","operator":"Owner"}`)},
+		{"wrong-operator", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","operator":"wrong-operator"}`)},
+		{"unsupported-protocol", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"ftp","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","operator":"Owner"}`)},
+		{"invalid-closeMode", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"invalid_mode","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","operator":"Owner"}`)},
+		{"missing-exclusive", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":false,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","operator":"Owner"}`)},
+	} {
+		if err := r139VerifyBrowserReservation(nativeAdmission, tc.raw); err == nil {
+			t.Fatalf("expected error for %s, got nil", tc.name)
+		}
+	}
+
+	missingClosePathAdmission := nativeAdmission
+	missingClosePathAdmission.BrowserCloseReceiptPath = ""
+	if err := r139VerifyBrowserReservation(missingClosePathAdmission, validNativeRaw); err == nil {
+		t.Fatal("expected rejection when admission lacks browserCloseReceiptPath")
+	}
+
+	unsupportedAdmission := nativeAdmission
+	unsupportedAdmission.BrowserSupported = false
+	if err := r139VerifyBrowserReservation(unsupportedAdmission, validNativeRaw); err == nil {
+		t.Fatal("expected rejection when browserSupported is false")
+	}
+
+	unknownToolAdmission := nativeAdmission
+	unknownToolAdmission.BrowserTool = "unknown_tool"
+	if err := r139VerifyBrowserReservation(unknownToolAdmission, validNativeRaw); err == nil {
+		t.Fatal("expected rejection for unsupported browserTool")
+	}
+}
+
+func TestR139PureBrowserTerminalClose(t *testing.T) {
+	deadline := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
+	nativeAdmission := r139Admission{
+		TrancheID:             "CCMAI-RUNTIME-139",
+		BrowserSupported:      true,
+		BrowserTool:           "antigravity_native",
+		PageID:                "page-test-123",
+		CampaignNonce:         "nonce-abc-456",
+		BrowserSessionID:      "session-xyz-789",
+		FallbackCloseOperator: "Owner",
+		GrantDeadlineUTC:      deadline,
+	}
+
+	validCloseRaw := []byte(`{
+		"status": "CLOSED_EXACT_OWNED_PAGE",
+		"browserId": "antigravity_native",
+		"closeObserved": true,
+		"ownedTabsFinal": 0,
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"closeMode": "operator_fallback",
+		"operator": "Owner",
+		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `"
+	}`)
+
+	if err := r139VerifyBrowserTerminalClose(nativeAdmission, validCloseRaw); err != nil {
+		t.Fatalf("expected terminal close pass, got: %v", err)
+	}
+
+	notClosedRaw := []byte(`{
+		"status": "CLOSED_EXACT_OWNED_PAGE",
+		"browserId": "antigravity_native",
+		"closeObserved": false,
+		"ownedTabsFinal": 0,
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"closeMode": "operator_fallback",
+		"operator": "Owner",
+		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `"
+	}`)
+	if err := r139VerifyBrowserTerminalClose(nativeAdmission, notClosedRaw); err == nil {
+		t.Fatal("expected rejection when closeObserved is false")
+	}
+
+	tabsRemainingRaw := []byte(`{
+		"status": "CLOSED_EXACT_OWNED_PAGE",
+		"browserId": "antigravity_native",
+		"closeObserved": true,
+		"ownedTabsFinal": 1,
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"closeMode": "operator_fallback",
+		"operator": "Owner",
+		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `"
+	}`)
+	if err := r139VerifyBrowserTerminalClose(nativeAdmission, tabsRemainingRaw); err == nil {
+		t.Fatal("expected rejection when ownedTabsFinal != 0")
+	}
+
+	wrongStatusRaw := []byte(`{
+		"status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
+		"browserId": "antigravity_native",
+		"closeObserved": true,
+		"ownedTabsFinal": 0,
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"closeMode": "operator_fallback",
+		"operator": "Owner",
+		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `"
+	}`)
+	if err := r139VerifyBrowserTerminalClose(nativeAdmission, wrongStatusRaw); err == nil {
+		t.Fatal("expected rejection for non-closed status")
+	}
+
+	for _, tc := range []struct {
+		name string
+		raw  []byte
+	}{
+		{"wrong-pageId", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"pageId":"wrong","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
+		{"wrong-nonce", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"pageId":"page-test-123","campaignNonce":"wrong","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
+		{"wrong-session", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"wrong","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
+		{"wrong-operator", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"wrong","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
+		{"invalid-closeMode", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"bad","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
+	} {
+		if err := r139VerifyBrowserTerminalClose(nativeAdmission, tc.raw); err == nil {
+			t.Fatalf("expected error for %s, got nil", tc.name)
+		}
+	}
+
+	futureTime := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
+	futureRaw := []byte(`{
+		"status": "CLOSED_EXACT_OWNED_PAGE",
+		"browserId": "antigravity_native",
+		"closeObserved": true,
+		"ownedTabsFinal": 0,
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"closeMode": "operator_fallback",
+		"operator": "Owner",
+		"closedAtUTC": "` + futureTime + `"
+	}`)
+	if err := r139VerifyBrowserTerminalClose(nativeAdmission, futureRaw); err == nil {
+		t.Fatal("expected rejection for future timestamp")
+	}
+
+	pastDeadlineAdmission := nativeAdmission
+	pastDeadlineAdmission.GrantDeadlineUTC = time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
+	if err := r139VerifyBrowserTerminalClose(pastDeadlineAdmission, validCloseRaw); err == nil {
+		t.Fatal("expected rejection when closedAtUTC is after grantDeadlineUTC")
+	}
+
+	badTimeRaw := []byte(`{
+		"status": "CLOSED_EXACT_OWNED_PAGE",
+		"browserId": "antigravity_native",
+		"closeObserved": true,
+		"ownedTabsFinal": 0,
+		"pageId": "page-test-123",
+		"campaignNonce": "nonce-abc-456",
+		"browserSessionId": "session-xyz-789",
+		"closeMode": "operator_fallback",
+		"operator": "Owner",
+		"closedAtUTC": "invalid-time"
+	}`)
+	if err := r139VerifyBrowserTerminalClose(nativeAdmission, badTimeRaw); err == nil {
+		t.Fatal("expected rejection for invalid closedAtUTC")
+	}
+}
@@ -203,4 +684,3 @@ func r139Admit(t *testing.T) (r139Admission, string) {
-	var browserProof map[string]any
-	if json.Unmarshal(browserRaw, &browserProof) != nil || !a.BrowserSupported || browserProof["status"] != "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" || browserProof["browserId"] != "iab" || browserProof["controlObserved"] != true || browserProof["captureObserved"] != true || browserProof["closeObserved"] != true || browserProof["ownedTabsFinal"] != float64(0) || (browserProof["protocol"] != "http" && browserProof["protocol"] != "https") {
-		t.Fatal("R139 supported browser proof absent")
+	if err := r139VerifyBrowserReservation(a, browserRaw); err != nil {
+		t.Fatalf("R139 supported browser proof absent: %v", err)
 	}
diff --git a/scripts/r139_execution_supervisor.py b/scripts/r139_execution_supervisor.py
index 2382b07..85b6615 100644
--- a/scripts/r139_execution_supervisor.py
+++ b/scripts/r139_execution_supervisor.py
@@ -128,8 +128,101 @@ def committed(relative: str) -> Path:
 def verify_browser_receipt(value: dict) -> None:
-    require(value.get("browserTool") == "supported_CUA_IAB" and value.get("browserSupported") is True, "BROWSER_CAPABILITY_UNPROVEN")
-    browser = project_path(value.get("browserReceiptPath", ""))
+    tool = value.get("browserTool")
+    require(tool in ("supported_CUA_IAB", "antigravity_native") and value.get("browserSupported") is True, "BROWSER_CAPABILITY_UNPROVEN")
+    browser_rel = value.get("browserReceiptPath", "")
+    require(isinstance(browser_rel, str) and browser_rel.strip(), "BROWSER_RECEIPT_PATH_MISSING")
+    browser = project_path(browser_rel)
     expected = value.get("browserReceiptSha256", "")
     require(isinstance(expected, str) and HEX64.fullmatch(expected) and digest(browser) == expected, "BROWSER_RECEIPT_MISMATCH")
     proof = read_json(browser)
-    require(proof.get("status") == "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" and proof.get("browserId") == "iab" and proof.get("controlObserved") is True and proof.get("captureObserved") is True and proof.get("closeObserved") is True and proof.get("ownedTabsFinal") == 0 and proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+    if tool == "supported_CUA_IAB":
+        require(proof.get("status") == "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" and proof.get("browserId") == "iab" and proof.get("controlObserved") is True and proof.get("captureObserved") is True and proof.get("closeObserved") is True and proof.get("ownedTabsFinal") == 0 and proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+    elif tool == "antigravity_native":
+        require(proof.get("status") != "CLOSED_EXACT_OWNED_PAGE", "TERMINAL_CLOSE_RECEIPT_CANNOT_BE_PREFIXTURE_RESERVATION")
+        require(proof.get("status") == "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE", "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        require(proof.get("browserId") == "antigravity_native", "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        require(proof.get("exclusiveReservation") is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        require(proof.get("controlObserved") is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        require(proof.get("captureObserved") is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        require(proof.get("closeObserved") is not True, "EARLY_CLOSE_FORBIDDEN")
+        require(proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        require(proof.get("closeMode") in ("api", "operator", "operator_fallback"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        page_id = proof.get("pageId")
+        require(isinstance(page_id, str) and page_id.strip(), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        if value.get("pageId"):
+            require(page_id == value["pageId"], "BROWSER_PROOF_IDENTITY_MISMATCH")
+        nonce = proof.get("campaignNonce")
+        require(isinstance(nonce, str) and nonce.strip(), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        if value.get("campaignNonce"):
+            require(nonce == value["campaignNonce"], "BROWSER_PROOF_IDENTITY_MISMATCH")
+        session_id = proof.get("browserSessionId")
+        require(isinstance(session_id, str) and session_id.strip(), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        if value.get("browserSessionId"):
+            require(session_id == value["browserSessionId"], "BROWSER_PROOF_IDENTITY_MISMATCH")
+        tranche_id = proof.get("trancheId")
+        if tranche_id:
+            require(tranche_id == value.get("trancheId", "CCMAI-RUNTIME-139"), "BROWSER_PROOF_IDENTITY_MISMATCH")
+        expected_op = value.get("fallbackCloseOperator") or value.get("operator")
+        op = proof.get("operator")
+        require(isinstance(op, str) and op.strip(), "BROWSER_PROOF_CONTENT_INCOMPLETE")
+        if expected_op:
+            require(op == expected_op, "BROWSER_PROOF_IDENTITY_MISMATCH")
+        close_path_rel = value.get("browserCloseReceiptPath")
+        require(isinstance(close_path_rel, str) and close_path_rel.strip(), "BROWSER_CLOSE_RECEIPT_PATH_MISSING")
+    else:
+        raise Hold("BROWSER_TOOL_UNSUPPORTED")
+
+
+def verify_browser_terminal_close(admission: dict, runtime: Path, external: Path) -> dict:
+    close_rel = admission.get("browserCloseReceiptPath")
+    require(isinstance(close_rel, str) and close_rel.strip(), "BROWSER_CLOSE_RECEIPT_PATH_MISSING")
+    close_path = project_path(close_rel)
+    expected_sha = admission.get("browserCloseReceiptSha256")
+    if expected_sha:
+        require(isinstance(expected_sha, str) and HEX64.fullmatch(expected_sha) and digest(close_path) == expected_sha, "BROWSER_CLOSE_RECEIPT_SHA_MISMATCH")
+    receipt = read_json(close_path)
+    require(receipt.get("status") == "CLOSED_EXACT_OWNED_PAGE", "BROWSER_CLOSE_STATUS_INVALID")
+    require(receipt.get("browserId") == "antigravity_native", "BROWSER_CLOSE_BROWSER_ID_INVALID")
+    require(receipt.get("closeObserved") is True, "BROWSER_CLOSE_NOT_OBSERVED")
+    require(receipt.get("ownedTabsFinal") == 0, "BROWSER_CLOSE_OWNED_TABS_REMAINING")
+    page_id = receipt.get("pageId")
+    require(isinstance(page_id, str) and page_id.strip(), "BROWSER_CLOSE_PAGE_ID_MISSING")
+    if admission.get("pageId"):
+        require(page_id == admission["pageId"], "BROWSER_CLOSE_PAGE_ID_MISMATCH")
+    nonce = receipt.get("campaignNonce")
+    require(isinstance(nonce, str) and nonce.strip(), "BROWSER_CLOSE_NONCE_MISSING")
+    if admission.get("campaignNonce"):
+        require(nonce == admission["campaignNonce"], "BROWSER_CLOSE_NONCE_MISMATCH")
+    session_id = receipt.get("browserSessionId")
+    require(isinstance(session_id, str) and session_id.strip(), "BROWSER_CLOSE_SESSION_MISSING")
+    if admission.get("browserSessionId"):
+        require(session_id == admission["browserSessionId"], "BROWSER_CLOSE_SESSION_MISMATCH")
+    close_mode = receipt.get("closeMode")
+    require(close_mode in ("api", "operator", "operator_fallback"), "BROWSER_CLOSE_MODE_INVALID")
+    expected_op = admission.get("fallbackCloseOperator") or admission.get("operator")
+    operator = receipt.get("operator")
+    require(isinstance(operator, str) and operator.strip(), "BROWSER_CLOSE_OPERATOR_MISSING")
+    if expected_op:
+        require(operator == expected_op, "BROWSER_CLOSE_OPERATOR_MISMATCH")
+    closed_at_str = receipt.get("closedAtUTC")
+    require(isinstance(closed_at_str, str) and closed_at_str.endswith("Z"), "BROWSER_CLOSE_TIMESTAMP_INVALID")
+    try:
+        closed_dt = datetime.fromisoformat(closed_at_str[:-1] + "+00:00")
+    except Exception:
+        raise Hold("BROWSER_CLOSE_TIMESTAMP_INVALID")
+    now_utc = datetime.now(timezone.utc)
+    require(closed_dt <= now_utc + timedelta(seconds=60), "BROWSER_CLOSE_TIMESTAMP_IN_FUTURE")
+    deadline_str = admission.get("grantDeadlineUTC")
+    if deadline_str and deadline_str.endswith("Z"):
+        try:
+            deadline_dt = datetime.fromisoformat(deadline_str[:-1] + "+00:00")
+            require(closed_dt <= deadline_dt, "BROWSER_CLOSE_AFTER_DEADLINE")
+        except Exception:
+            pass
+    evidence_rel = receipt.get("evidencePath") or receipt.get("actionEvidencePath")
+    if evidence_rel:
+        ev_path = project_path(evidence_rel)
+        ev_sha = receipt.get("evidenceSha256")
+        if ev_sha:
+            require(digest(ev_path) == ev_sha, "BROWSER_CLOSE_EVIDENCE_SHA_MISMATCH")
+    return receipt
@@ -1084,2 +1177,7 @@ def run(relative: str) -> None:
             errors.append("SOURCE_RECHECK_FAILED:" + type(exc).__name__)
+        if admission.get("browserTool") == "antigravity_native":
+            try:
+                outcome["browserTerminalClose"] = verify_browser_terminal_close(admission, runtime, external)
+            except Exception as exc:
+                errors.append("BROWSER_TERMINAL_CLOSE_FAILED:" + type(exc).__name__)
         errors.extend("CLEANUP_BEST_EFFORT_ERROR:" + issue for issue in CLEANUP_ERRORS)
diff --git a/scripts/tests/test_r139_execution_supervisor.py b/scripts/tests/test_r139_execution_supervisor.py
index ed1d50d..676aa7e 100644
--- a/scripts/tests/test_r139_execution_supervisor.py
+++ b/scripts/tests/test_r139_execution_supervisor.py
@@ -608,2 +608,206 @@ class R139SupervisorPureTests(unittest.TestCase):
                 module._validate_route_rows(rows, "TEST", {"executePOST": 1, "authPOST": 3})
+
+    def test_native_antigravity_reservation_admission_positive(self):
+        with tempfile.TemporaryDirectory() as name:
+            receipt = Path(name) / "reservation.json"
+            receipt.write_text(json.dumps({
+                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
+                "browserId": "antigravity_native",
+                "exclusiveReservation": True,
+                "controlObserved": True,
+                "captureObserved": True,
+                "closeObserved": False,
+                "protocol": "http",
+                "closeMode": "operator_fallback",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "trancheId": "CCMAI-RUNTIME-139",
+                "operator": "Owner"
+            }), encoding="utf-8")
+            admission = {
+                "browserTool": "antigravity_native",
+                "browserSupported": True,
+                "browserReceiptPath": "reservation.json",
+                "browserReceiptSha256": module.digest(receipt),
+                "browserCloseReceiptPath": "close.json",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "fallbackCloseOperator": "Owner"
+            }
+            with patch.object(module, "project_path", return_value=receipt):
+                module.verify_browser_receipt(admission)
+
+    def test_native_antigravity_early_close_rejected(self):
+        with tempfile.TemporaryDirectory() as name:
+            receipt = Path(name) / "reservation.json"
+            receipt.write_text(json.dumps({
+                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
+                "browserId": "antigravity_native",
+                "exclusiveReservation": True,
+                "controlObserved": True,
+                "captureObserved": True,
+                "closeObserved": True,
+                "protocol": "http",
+                "closeMode": "operator_fallback",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "operator": "Owner"
+            }), encoding="utf-8")
+            admission = {
+                "browserTool": "antigravity_native",
+                "browserSupported": True,
+                "browserReceiptPath": "reservation.json",
+                "browserReceiptSha256": module.digest(receipt),
+                "browserCloseReceiptPath": "close.json",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "fallbackCloseOperator": "Owner"
+            }
+            with patch.object(module, "project_path", return_value=receipt):
+                with self.assertRaisesRegex(module.Hold, "EARLY_CLOSE_FORBIDDEN"):
+                    module.verify_browser_receipt(admission)
+
+    def test_native_antigravity_final_close_cannot_be_prefixture_reservation(self):
+        with tempfile.TemporaryDirectory() as name:
+            receipt = Path(name) / "reservation.json"
+            receipt.write_text(json.dumps({
+                "status": "CLOSED_EXACT_OWNED_PAGE",
+                "browserId": "antigravity_native",
+                "exclusiveReservation": True,
+                "controlObserved": True,
+                "captureObserved": True,
+                "closeObserved": True,
+                "ownedTabsFinal": 0,
+                "protocol": "http",
+                "closeMode": "operator_fallback",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "operator": "Owner"
+            }), encoding="utf-8")
+            admission = {
+                "browserTool": "antigravity_native",
+                "browserSupported": True,
+                "browserReceiptPath": "reservation.json",
+                "browserReceiptSha256": module.digest(receipt),
+                "browserCloseReceiptPath": "close.json",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "fallbackCloseOperator": "Owner"
+            }
+            with patch.object(module, "project_path", return_value=receipt):
+                with self.assertRaisesRegex(module.Hold, "TERMINAL_CLOSE_RECEIPT_CANNOT_BE_PREFIXTURE_RESERVATION"):
+                    module.verify_browser_receipt(admission)
+
+    def test_native_antigravity_identity_mismatches_and_missing_close_path_rejected(self):
+        with tempfile.TemporaryDirectory() as name:
+            receipt = Path(name) / "reservation.json"
+            base_data = {
+                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
+                "browserId": "antigravity_native",
+                "exclusiveReservation": True,
+                "controlObserved": True,
+                "captureObserved": True,
+                "closeObserved": False,
+                "protocol": "http",
+                "closeMode": "operator_fallback",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "operator": "Owner"
+            }
+            receipt.write_text(json.dumps(base_data), encoding="utf-8")
+            admission = {
+                "browserTool": "antigravity_native",
+                "browserSupported": True,
+                "browserReceiptPath": "reservation.json",
+                "browserReceiptSha256": module.digest(receipt),
+                "browserCloseReceiptPath": "close.json",
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "fallbackCloseOperator": "Owner"
+            }
+            with patch.object(module, "project_path", return_value=receipt):
+                bad_adm = dict(admission, browserCloseReceiptPath="")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_RECEIPT_PATH_MISSING"):
+                    module.verify_browser_receipt(bad_adm)
+                bad_adm = dict(admission, pageId="other-page")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
+                    module.verify_browser_receipt(bad_adm)
+                bad_adm = dict(admission, campaignNonce="other-nonce")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
+                    module.verify_browser_receipt(bad_adm)
+                bad_adm = dict(admission, browserSessionId="other-session")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
+                    module.verify_browser_receipt(bad_adm)
+                bad_adm = dict(admission, fallbackCloseOperator="other-operator")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
+                    module.verify_browser_receipt(bad_adm)
+
+    def test_verify_browser_terminal_close_positive_and_rejections(self):
+        deadline = (datetime.now(timezone.utc) + timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ")
+        admission = {
+            "browserTool": "antigravity_native",
+            "browserCloseReceiptPath": "close.json",
+            "pageId": "page-123",
+            "campaignNonce": "nonce-456",
+            "browserSessionId": "session-789",
+            "fallbackCloseOperator": "Owner",
+            "grantDeadlineUTC": deadline
+        }
+        with tempfile.TemporaryDirectory() as name:
+            close_file = Path(name) / "close.json"
+            valid_close = {
+                "status": "CLOSED_EXACT_OWNED_PAGE",
+                "browserId": "antigravity_native",
+                "closeObserved": True,
+                "ownedTabsFinal": 0,
+                "pageId": "page-123",
+                "campaignNonce": "nonce-456",
+                "browserSessionId": "session-789",
+                "closeMode": "operator_fallback",
+                "operator": "Owner",
+                "closedAtUTC": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
+            }
+            close_file.write_text(json.dumps(valid_close), encoding="utf-8")
+            with patch.object(module, "project_path", return_value=close_file):
+                result = module.verify_browser_terminal_close(admission, Path(name), Path(name))
+                self.assertEqual(result["status"], "CLOSED_EXACT_OWNED_PAGE")
+
+                bad = dict(valid_close, closeObserved=False)
+                close_file.write_text(json.dumps(bad), encoding="utf-8")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_NOT_OBSERVED"):
+                    module.verify_browser_terminal_close(admission, Path(name), Path(name))
+
+                bad = dict(valid_close, ownedTabsFinal=1)
+                close_file.write_text(json.dumps(bad), encoding="utf-8")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OWNED_TABS_REMAINING"):
+                    module.verify_browser_terminal_close(admission, Path(name), Path(name))
+
+                bad = dict(valid_close, status="RESERVED_NATIVE_PAGE_CONTROL_CAPTURE")
+                close_file.write_text(json.dumps(bad), encoding="utf-8")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_STATUS_INVALID"):
+                    module.verify_browser_terminal_close(admission, Path(name), Path(name))
+
+                bad = dict(valid_close, pageId="wrong")
+                close_file.write_text(json.dumps(bad), encoding="utf-8")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_PAGE_ID_MISMATCH"):
+                    module.verify_browser_terminal_close(admission, Path(name), Path(name))
+
+                future_ts = (datetime.now(timezone.utc) + timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
+                bad = dict(valid_close, closedAtUTC=future_ts)
+                close_file.write_text(json.dumps(bad), encoding="utf-8")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_TIMESTAMP_IN_FUTURE"):
+                    module.verify_browser_terminal_close(admission, Path(name), Path(name))
+
+                past_deadline_adm = dict(admission, grantDeadlineUTC=(datetime.now(timezone.utc) - timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ"))
+                close_file.write_text(json.dumps(valid_close), encoding="utf-8")
+                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_AFTER_DEADLINE"):
+                    module.verify_browser_terminal_close(past_deadline_adm, Path(name), Path(name))
```

---

## 7. Claim Boundary and Mandatory Governance Adherence

- Under CVF First-Request Protocol and downstream mandatory governance rules (`AGENTS.md` and `.cvf/policy.json`):
  - No claims of runtime AI execution, component-level passing, or CVF governance behavior are made here.
  - Mock mode was not used to assert any AI governance behavior.
  - Zero API keys, credentials, or provider secrets were staged or accessed in this source stage.
  - The working tree remains uncommitted with changes strictly confined to the three authorized files.
  - Product B remains strictly `BLOCKED_PENDING_OPERATIONAL_ADMISSION`.
  - The completed return is returned to Root for independent review and subsequent repair admission / check authorization under the approved envelope.
