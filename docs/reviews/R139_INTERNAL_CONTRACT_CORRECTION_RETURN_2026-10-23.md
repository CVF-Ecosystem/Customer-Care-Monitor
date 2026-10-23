# R139 internal native contract correction return

Status: COMPLETE_PENDING_REVIEW
Risk: R2
Worker: Internal Sol Medium `/root/r139_internal_contract_repair`
Reviewer: Root independent source REVIEWER
Role: Internal REPAIR_WORKER `/root/r139_internal_contract_repair`
Independent reviewer and commit steward: Root
Dispatch HEAD: `8f6063b12cf36c6391e29d43b0fd684c3a9c61e4`
Finished: 2026-10-23 17:02:51 UTC / 2026-10-24 00:02:51 Asia/Saigon

## Before-source acknowledgment

I rehydrated the current R139 REVIEW continuity, active handoff, implementation status, documentation index, work-order C1–C4 section, and both immutable authority records. The committed actor-only supplement is SHA256 `5b5d1dc4af4c669255867d3334c75a145587ac923f35cefb4bc043a1654d3a9b`; its parent correction admission is SHA256 `4a7e2d30b13c783dda1248e148908e11c6ebbab418e61280d1cc3fd0fc26317a`. The tracked tree was clean at the exact dispatch HEAD. Starting source SHA256 values match the parent admission: Python supervisor `830d12f5d36e849daf9558163ecf16ae5e74a5e3926a28fa164cadc5b0f7d023`, Python tests `d882c5f90356a87fd1da491ba137ed290bcc5cfb40e40082366ee1dba775223f`, and Go test file `19f923bc0b261d7e6279de302d55bf79e990abb1a96a5c16a2ae144788d68cf1`.

I accept the REPAIR_WORKER role for one existing same-scope manual C1–C4 correction. I will edit only the three admitted source paths and this return, execute zero source tests/checks or runtime effects, and make no commit. Root remains independent reviewer and holds the existing Python1 → compile1 → pureGo1 check custody. Product B and future runtime/provider effects remain blocked.

## Correction and return

One existing manual correction round was used. I changed only the three admitted source files and this return. The full three-file diff follows below. No source test, parser, compiler, binary, browser, runtime, process, Docker, database, network, provider, or API invocation was run by this worker; no source check budget was consumed. No commit, authority, continuity, work-order, catalog, previous report, Core, product source, or protected file was changed by this worker. Root retains static acceptance and the original unused Python1 → compile1 → audited pureGo1 chain.

| File | Starting SHA256 | Return SHA256 |
| --- | --- | --- |
| `scripts/r139_execution_supervisor.py` | `830d12f5d36e849daf9558163ecf16ae5e74a5e3926a28fa164cadc5b0f7d023` | `2328365350cf356cfe475a95937d777d0a6b66545e717efd7a1e7db82ba730e7` |
| `scripts/tests/test_r139_execution_supervisor.py` | `d882c5f90356a87fd1da491ba137ed290bcc5cfb40e40082366ee1dba775223f` | `f3457f8ece30e699effbe88adb46439b6e5b2320815057394b14e8df75187771` |
| `backend/api/handlers/intervention_execution_browser_test.go` | `19f923bc0b261d7e6279de302d55bf79e990abb1a96a5c16a2ae144788d68cf1` | `13109be744b8f94d8822879c70ab53ddabd8a220f71a3051b35f8eda5452d7bb` |

C1: Native records now bind the same bounded `nativeContractId`, `trancheId`, `campaignNonce`, `pageId`, `browserSessionId`, `operator`, and selected `closeMode` as the admission. `operator_fallback` additionally requires `fallbackCloseOperator == operator`; application `actorRole` is never a close operator. Reservation permits absent or exact false `closeObserved`; terminal requires exact true. IAB remains supported with exact integer zero tab count. Go uses `json.Decoder.UseNumber` and `Int64` to reject `0.0`, `0e0`, booleans, and null as counts, matching Python's exact integer test.

C2: A committed tracked readiness receipt is pinned by `browserReceiptPath` and `browserReceiptSha256`. Its original `creationEvidencePath/Sha256`, `controlEvidencePath/Sha256`, and `captureEvidencePath/Sha256` each name readable JSON under a pre-existing ignored project-relative `browserReadinessRoot`. The admission's `runtimeRoot` stays the original exact absolute product root and is absent before Python launch. It is distinct from readiness. The admission also selects future `browserCloseReceiptPath`, `actionEvidencePath`, `finalInventoryPath`, and `rawNativeInventoryPath` under that product root; these are absent before launch and read only at terminal. Canonical project-root path resolution rejects absolute/traversal/reparse escapes for project-relative evidence. Every required file, SHA, JSON object, and raw read failure rejects. The future close receipt has no fabricated prelaunch admission SHA; Python seals its actual SHA in the terminal validation result after creation.

C3: The reservation binds `reservedAtUTC`; original creation, control, and capture evidence each carries the seven identities plus `event` and `atUTC` in chronological order before reservation. The close receipt carries matching reservation time, `actionAtUTC`, `closedAtUTC`, and the selected action/inventory descriptors. Action evidence has `event: action_sequence`, an ordered `actions` list for the exact page ending with its sole close event; any recorded use after close rejects. Canonical final inventory has the seven identities, `inventoryAtUTC`, exact `pageIds` list, integer `ownedCount: 0`, and a descriptor for original raw native inventory JSON. Raw inventory must match the canonical identities, time, list, and count. The required chronology is reservation ≤ action ≤ close ≤ inventory ≤ admitted deadline, with observations ≤ current UTC and no future tolerance. This validates the captured stream and its hashes; Root still must inspect original native/Owner provenance before operational acceptance.

C4: Both exact Go pure test names remain, with complete temp-file positive fixtures and missing, wrong-type, identity, count, evidence, root, inventory, and post-close negatives. Python tests use complete phase-bound fixtures and invoke the actual `module.run` in success, ordinary error, missing-close, and failed-close scenarios. `ExitStack` replaces a nested context-manager block; every external boundary is patched, fake original child handles are asserted, and the real terminal-close validator reads synthetic temp evidence while owned cleanup remains observable. These are software contract checks only, not live browser or AI governance proof.

The intended future admission shape is concrete: the seven identities above; `browserReadinessRoot: ".cvf/runtime/<campaign-readiness>"`; existing exact absolute product `runtimeRoot`; the pinned readiness receipt path/hash; and four selected future product-root paths. The reservation contains the three original evidence path/hash pairs. At terminal, close/action/inventory/raw files appear at the selected paths, are read and checked, then the actual close digest is sealed in the Python terminal result. Root should reject a runtime admission that omits any of these required fields.

No source or runtime acceptance is claimed by this return. Product B remains `BLOCKED_PENDING_OPERATIONAL_ADMISSION`; G1–G3/E05/E06/full M3 are open. The prior G0/provider/protected/old-tab qualifications and parked Facebook/Zalo OA checkpoint remain. Future campaign/API1/USD1 and exact-page Owner close are conditional and non-executable here.

## Full admitted source diff

~~~~diff
diff --git a/backend/api/handlers/intervention_execution_browser_test.go b/backend/api/handlers/intervention_execution_browser_test.go
index 5bc968cf..d98e736b 100644
--- a/backend/api/handlers/intervention_execution_browser_test.go
+++ b/backend/api/handlers/intervention_execution_browser_test.go
@@ -37,6 +37,14 @@ import (
 )
 
 type r139Admission struct {
+	NativeContractID          string            `json:"nativeContractId"`
+	CloseMode                 string            `json:"closeMode"`
+	Operator                  string            `json:"operator"`
+	BrowserReadinessRoot      string            `json:"browserReadinessRoot"`
+	ActionEvidencePath        string            `json:"actionEvidencePath"`
+	FinalInventoryPath        string            `json:"finalInventoryPath"`
+	RawNativeInventoryPath    string            `json:"rawNativeInventoryPath"`
+	nativeProjectRoot         string
 	TrancheID                 string            `json:"trancheId"`
 	Kind                      string            `json:"kind"`
 	SourceCommit              string            `json:"sourceCommit"`
@@ -78,375 +86,231 @@ type r139Admission struct {
 }
 
 func r139VerifyBrowserReservation(a r139Admission, browserRaw []byte) error {
-	if !a.BrowserSupported {
-		return errors.New("browserSupported is false")
-	}
-	var proof map[string]any
-	if err := json.Unmarshal(browserRaw, &proof); err != nil {
-		return fmt.Errorf("invalid json in browser proof: %w", err)
-	}
-
-	tool := a.BrowserTool
-	if tool == "supported_CUA_IAB" {
-		status, _ := proof["status"].(string)
-		browserID, _ := proof["browserId"].(string)
-		ctrl, _ := proof["controlObserved"].(bool)
-		capt, _ := proof["captureObserved"].(bool)
-		closed, _ := proof["closeObserved"].(bool)
-		proto, _ := proof["protocol"].(string)
-
-		ownedTabsVal, ok := proof["ownedTabsFinal"]
-		if !ok {
-			return errors.New("ownedTabsFinal missing in legacy IAB proof")
-		}
-		ownedTabs, isFloat := ownedTabsVal.(float64)
-		if !isFloat || ownedTabs != 0 {
-			return errors.New("ownedTabsFinal must be numeric 0 in legacy IAB proof")
-		}
-
-		if status != "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" ||
-			browserID != "iab" ||
-			!ctrl || !capt || !closed ||
-			(proto != "http" && proto != "https") {
-			return errors.New("legacy IAB browser proof content incomplete")
-		}
-		return nil
-	}
-
-	if tool == "antigravity_native" {
-		status, _ := proof["status"].(string)
-		if status == "CLOSED_EXACT_OWNED_PAGE" {
-			return errors.New("final-close evidence rejected as prefixture reservation")
-		}
-		if status != "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE" {
-			return fmt.Errorf("unexpected reservation status: %s", status)
-		}
-
-		browserID, _ := proof["browserId"].(string)
-		if browserID != "antigravity_native" {
-			return fmt.Errorf("unexpected browserId: %s", browserID)
-		}
-
-		excl, _ := proof["exclusiveReservation"].(bool)
-		if !excl {
-			return errors.New("exclusiveReservation must be true")
-		}
-
-		ctrl, _ := proof["controlObserved"].(bool)
-		if !ctrl {
-			return errors.New("controlObserved must be true")
-		}
-
-		capt, _ := proof["captureObserved"].(bool)
-		if !capt {
-			return errors.New("captureObserved must be true")
-		}
-
-		if closedVal, exists := proof["closeObserved"]; exists {
-			closed, isBool := closedVal.(bool)
-			if !isBool || closed {
-				return errors.New("early closeObserved is forbidden in reservation")
-			}
-		}
-
-		proto, _ := proof["protocol"].(string)
-		if proto != "http" && proto != "https" {
-			return fmt.Errorf("unsupported protocol: %s", proto)
-		}
-
-		closeMode, _ := proof["closeMode"].(string)
-		if closeMode != "api" && closeMode != "operator" && closeMode != "operator_fallback" {
-			return fmt.Errorf("invalid closeMode: %s", closeMode)
-		}
-
-		if strings.TrimSpace(a.TrancheID) == "" {
-			return errors.New("trancheId required in admission")
-		}
-		trancheID, _ := proof["trancheId"].(string)
-		if trancheID != a.TrancheID {
-			return fmt.Errorf("trancheId mismatch: expected %s, got %s", a.TrancheID, trancheID)
-		}
-
-		if strings.TrimSpace(a.PageID) == "" {
-			return errors.New("pageId required in admission")
-		}
-		pageID, _ := proof["pageId"].(string)
-		if pageID != a.PageID {
-			return fmt.Errorf("pageId mismatch: expected %s, got %s", a.PageID, pageID)
-		}
-
-		if strings.TrimSpace(a.CampaignNonce) == "" {
-			return errors.New("campaignNonce required in admission")
-		}
-		nonce, _ := proof["campaignNonce"].(string)
-		if nonce != a.CampaignNonce {
-			return fmt.Errorf("campaignNonce mismatch: expected %s, got %s", a.CampaignNonce, nonce)
-		}
-
-		if strings.TrimSpace(a.BrowserSessionID) == "" {
-			return errors.New("browserSessionId required in admission")
-		}
-		sessionID, _ := proof["browserSessionId"].(string)
-		if sessionID != a.BrowserSessionID {
-			return fmt.Errorf("browserSessionId mismatch: expected %s, got %s", a.BrowserSessionID, sessionID)
-		}
-
-		expectedOp := a.FallbackCloseOperator
-		if expectedOp == "" {
-			expectedOp = a.ActorRole
-		}
-		if strings.TrimSpace(expectedOp) == "" {
-			return errors.New("operator required in admission")
-		}
-		op, _ := proof["operator"].(string)
-		if op != expectedOp {
-			return fmt.Errorf("operator mismatch: expected %s, got %s", expectedOp, op)
-		}
-
-		if strings.TrimSpace(a.BrowserCloseReceiptPath) == "" {
-			return errors.New("browserCloseReceiptPath is required in admission")
-		}
-		if strings.HasPrefix(a.BrowserCloseReceiptPath, "docs/") {
-			return errors.New("browserCloseReceiptPath must be in ignored runtime path, not tracked docs/")
-		}
-
-		if reservedStr, ok := proof["reservedAtUTC"].(string); ok && reservedStr != "" {
-			if !strings.HasSuffix(reservedStr, "Z") {
-				return errors.New("reservedAtUTC must end in Z")
-			}
-			reservedAt, rErr := time.Parse(time.RFC3339, reservedStr)
-			if rErr != nil {
-				reservedAt, rErr = time.Parse(time.RFC3339Nano, reservedStr)
-			}
-			if rErr != nil {
-				return fmt.Errorf("invalid reservedAtUTC: %w", rErr)
-			}
-			if reservedAt.After(time.Now().UTC().Add(time.Minute)) {
-				return errors.New("reservedAtUTC cannot be in the future")
-			}
-		}
-
-		evRel, _ := proof["actionEvidencePath"].(string)
-		if evRel == "" {
-			evRel, _ = proof["evidencePath"].(string)
-		}
-		if evRel != "" {
-			if strings.HasPrefix(evRel, "docs/") {
-				return errors.New("actionEvidencePath must not be in tracked docs/")
-			}
-			evSha, _ := proof["actionEvidenceSha256"].(string)
-			if evSha == "" {
-				evSha, _ = proof["evidenceSha256"].(string)
-			}
-			if len(evSha) != 64 {
-				return errors.New("actionEvidenceSha256 must be 64-hex string")
-			}
-			if data, readErr := os.ReadFile(evRel); readErr == nil {
-				sum := sha256.Sum256(data)
-				if hex.EncodeToString(sum[:]) != evSha {
-					return errors.New("action evidence sha mismatch")
-				}
-			}
-		}
-
-		return nil
-	}
-
-	return fmt.Errorf("unsupported browserTool: %s", tool)
+	return r139StrictBrowserReservation(a, browserRaw)
 }
 
 func r139VerifyBrowserTerminalClose(a r139Admission, closeRaw []byte) error {
-	var receipt map[string]any
-	if err := json.Unmarshal(closeRaw, &receipt); err != nil {
-		return fmt.Errorf("invalid json in terminal close receipt: %w", err)
-	}
-
-	status, _ := receipt["status"].(string)
-	if status != "CLOSED_EXACT_OWNED_PAGE" {
-		return fmt.Errorf("status must be CLOSED_EXACT_OWNED_PAGE, got %s", status)
-	}
-
-	browserID, _ := receipt["browserId"].(string)
-	if browserID != "antigravity_native" {
-		return fmt.Errorf("browserId must be antigravity_native, got %s", browserID)
-	}
-
-	closeObservedVal, exists := receipt["closeObserved"]
-	if !exists {
-		return errors.New("closeObserved is required in terminal close receipt")
-	}
-	closeObserved, isBool := closeObservedVal.(bool)
-	if !isBool || !closeObserved {
-		return errors.New("closeObserved must be exact boolean true")
-	}
-
-	ownedTabsVal, ok := receipt["ownedTabsFinal"]
-	if !ok {
-		return errors.New("ownedTabsFinal is required in terminal close receipt")
-	}
-	ownedTabs, isFloat := ownedTabsVal.(float64)
-	if !isFloat || ownedTabs != 0 {
-		return fmt.Errorf("ownedTabsFinal must be numeric 0, got %v", ownedTabsVal)
-	}
-
-	if strings.TrimSpace(a.TrancheID) == "" {
-		return errors.New("trancheId required in admission")
-	}
-	if trancheID, _ := receipt["trancheId"].(string); trancheID != a.TrancheID {
-		return fmt.Errorf("trancheId mismatch: expected %s, got %s", a.TrancheID, trancheID)
-	}
-
-	if strings.TrimSpace(a.PageID) == "" {
-		return errors.New("pageId required in admission")
-	}
-	if pageID, _ := receipt["pageId"].(string); pageID != a.PageID {
-		return fmt.Errorf("pageId mismatch: expected %s, got %s", a.PageID, pageID)
-	}
-
-	if strings.TrimSpace(a.CampaignNonce) == "" {
-		return errors.New("campaignNonce required in admission")
-	}
-	if nonce, _ := receipt["campaignNonce"].(string); nonce != a.CampaignNonce {
-		return fmt.Errorf("campaignNonce mismatch: expected %s, got %s", a.CampaignNonce, nonce)
-	}
-
-	if strings.TrimSpace(a.BrowserSessionID) == "" {
-		return errors.New("browserSessionId required in admission")
-	}
-	if sessionID, _ := receipt["browserSessionId"].(string); sessionID != a.BrowserSessionID {
-		return fmt.Errorf("browserSessionId mismatch: expected %s, got %s", a.BrowserSessionID, sessionID)
-	}
+	return r139StrictBrowserTerminalClose(a, closeRaw)
+}
 
-	closeMode, _ := receipt["closeMode"].(string)
-	if closeMode != "api" && closeMode != "operator" && closeMode != "operator_fallback" {
-		return fmt.Errorf("invalid closeMode: %s", closeMode)
-	}
+func r139NativeRoot(a r139Admission, field string) (string, string, error) {
+	project := a.nativeProjectRoot
+	if project == "" {
+		_, file, _, ok := runtime.Caller(0)
+		if !ok { return "", "", errors.New("source root unavailable") }
+		project = filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
+	}
+	project, err := filepath.Abs(project)
+	if err != nil { return "", "", err }
+	var selected string
+	switch field {
+	case "browserReadinessRoot": selected = a.BrowserReadinessRoot
+	case "runtimeRoot": selected = a.RuntimeRoot
+	default: return "", "", errors.New("invalid root field")
+	}
+	if selected == "" || strings.Contains(selected, "\\") && field == "browserReadinessRoot" { return "", "", errors.New("native root missing or invalid") }
+	var root string
+	if field == "runtimeRoot" {
+		if !filepath.IsAbs(selected) { return "", "", errors.New("runtimeRoot must be absolute") }
+		root = filepath.Clean(selected)
+	} else {
+		if filepath.IsAbs(selected) || !strings.HasPrefix(selected, ".cvf/runtime/") { return "", "", errors.New("readiness root must be ignored project-relative") }
+		for _, part := range strings.Split(selected, "/") { if part == ".." || part == "" { return "", "", errors.New("root traversal") } }
+		root = filepath.Join(project, filepath.FromSlash(selected))
+	}
+	ignored := filepath.Join(project, ".cvf", "runtime")
+	if rel, err := filepath.Rel(ignored, root); err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) { return "", "", errors.New("native root outside ignored area") }
+	return project, root, nil
+}
 
-	expectedOp := a.FallbackCloseOperator
-	if expectedOp == "" {
-		expectedOp = a.ActorRole
-	}
-	if strings.TrimSpace(expectedOp) == "" {
-		return errors.New("operator required in admission")
-	}
-	if op, _ := receipt["operator"].(string); op != expectedOp {
-		return fmt.Errorf("operator mismatch: expected %s, got %s", expectedOp, op)
-	}
+func r139NativePath(a r139Admission, field, selected string, root string, existing bool) (string, error) {
+	if selected == "" || filepath.IsAbs(selected) || strings.Contains(selected, "\\") { return "", fmt.Errorf("%s invalid path", field) }
+	for _, part := range strings.Split(selected, "/") { if part == ".." || part == "" { return "", fmt.Errorf("%s traversal", field) } }
+	project, _, err := r139NativeRoot(a, "runtimeRoot")
+	if err != nil { return "", err }
+	path := filepath.Join(project, filepath.FromSlash(selected))
+	rel, err := filepath.Rel(root, path)
+	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) { return "", fmt.Errorf("%s outside selected root", field) }
+	// Resolve every existing ancestor as well as an existing final file. A junction or symlink alias is rejected.
+	ancestor := path
+	for {
+		if _, statErr := os.Lstat(ancestor); statErr == nil {
+			resolved, resolveErr := filepath.EvalSymlinks(ancestor)
+			if resolveErr != nil || !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(ancestor)) { return "", fmt.Errorf("%s reparse alias", field) }
+		} else if !os.IsNotExist(statErr) { return "", statErr }
+		if strings.EqualFold(ancestor, project) { break }
+		parent := filepath.Dir(ancestor)
+		if parent == ancestor { return "", errors.New("project ancestor unavailable") }
+		ancestor = parent
+	}
+	if existing {
+		info, statErr := os.Stat(path)
+		if statErr != nil || !info.Mode().IsRegular() { return "", fmt.Errorf("%s missing file", field) }
+	} else if _, statErr := os.Lstat(path); statErr == nil || !os.IsNotExist(statErr) { return "", fmt.Errorf("%s must be absent", field) }
+	return path, nil
+}
 
-	closedAtStr, _ := receipt["closedAtUTC"].(string)
-	if strings.TrimSpace(closedAtStr) == "" || !strings.HasSuffix(closedAtStr, "Z") {
-		return errors.New("closedAtUTC must be non-empty ISO 8601 ending in Z")
-	}
+func r139NativeObject(raw []byte) (map[string]any, error) {
+	var value map[string]any
+	decoder := json.NewDecoder(bytes.NewReader(raw))
+	decoder.UseNumber()
+	if err := decoder.Decode(&value); err != nil || value == nil { return nil, errors.New("native JSON object invalid") }
+	var extra any
+	if err := decoder.Decode(&extra); err != io.EOF { return nil, errors.New("native JSON trailing content") }
+	return value, nil
+}
 
-	closedAt, err := time.Parse(time.RFC3339Nano, closedAtStr)
-	if err != nil {
-		closedAt, err = time.Parse(time.RFC3339, closedAtStr)
-	}
-	if err != nil {
-		return fmt.Errorf("invalid closedAtUTC: %w", err)
-	}
+func r139NativeZeroInteger(value any) bool {
+	number, ok := value.(json.Number)
+	if !ok { return false }
+	parsed, err := number.Int64()
+	return err == nil && parsed == 0
+}
 
-	now := time.Now().UTC()
-	if closedAt.After(now.Add(time.Minute)) {
-		return errors.New("closedAtUTC cannot be in the future")
-	}
+func r139NativeText(value any) (string, error) {
+	text, ok := value.(string)
+	if !ok || len(text) == 0 || len(text) > 128 || strings.TrimSpace(text) != text { return "", errors.New("native text invalid") }
+	for _, ch := range text { if ch < 32 || ch >= 127 { return "", errors.New("native text invalid") } }
+	return text, nil
+}
 
-	if strings.TrimSpace(a.GrantDeadlineUTC) == "" || !strings.HasSuffix(a.GrantDeadlineUTC, "Z") {
-		return errors.New("grantDeadlineUTC must be non-empty ISO 8601 ending in Z")
-	}
-	deadline, dErr := time.Parse(time.RFC3339Nano, a.GrantDeadlineUTC)
-	if dErr != nil {
-		deadline, dErr = time.Parse(time.RFC3339, a.GrantDeadlineUTC)
-	}
-	if dErr != nil {
-		return fmt.Errorf("invalid grantDeadlineUTC: %w", dErr)
-	}
-	if closedAt.After(deadline) {
-		return errors.New("closedAtUTC cannot be after grant deadline")
+func r139NativeIdentity(a r139Admission, record map[string]any) error {
+	if a.CloseMode != "api" && a.CloseMode != "operator" && a.CloseMode != "operator_fallback" { return errors.New("selected closeMode invalid") }
+	op := a.Operator
+	if a.CloseMode == "operator_fallback" && a.FallbackCloseOperator != op { return errors.New("fallback close operator mismatch") }
+	expected := map[string]string{"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID, "campaignNonce": a.CampaignNonce, "pageId": a.PageID, "browserSessionId": a.BrowserSessionID, "operator": op, "closeMode": a.CloseMode}
+	for field, want := range expected {
+		if _, err := r139NativeText(want); err != nil { return fmt.Errorf("admission %s: %w", field, err) }
+		got, err := r139NativeText(record[field])
+		if err != nil || got != want { return fmt.Errorf("native %s mismatch", field) }
 	}
+	return nil
+}
 
-	if reservedStr, ok := receipt["reservedAtUTC"].(string); ok && reservedStr != "" {
-		if !strings.HasSuffix(reservedStr, "Z") {
-			return errors.New("reservedAtUTC must end in Z")
-		}
-		reservedAt, rErr := time.Parse(time.RFC3339, reservedStr)
-		if rErr != nil {
-			reservedAt, rErr = time.Parse(time.RFC3339Nano, reservedStr)
-		}
-		if rErr != nil {
-			return fmt.Errorf("invalid reservedAtUTC: %w", rErr)
-		}
-		if reservedAt.After(closedAt) {
-			return errors.New("closedAtUTC cannot be before reservedAtUTC")
-		}
-	}
+func r139NativeTime(value any) (time.Time, error) {
+	text, ok := value.(string)
+	if !ok || !regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$").MatchString(text) { return time.Time{}, errors.New("native UTC time invalid") }
+	at, err := time.Parse("2006-01-02T15:04:05Z", text)
+	if err != nil || at.After(time.Now().UTC()) { return time.Time{}, errors.New("native UTC time future or invalid") }
+	return at, nil
+}
 
-	actionRel, _ := receipt["actionEvidencePath"].(string)
-	if actionRel == "" {
-		actionRel, _ = receipt["evidencePath"].(string)
-	}
-	if actionRel != "" {
-		if strings.HasPrefix(actionRel, "docs/") {
-			return errors.New("actionEvidencePath must not be in tracked docs/")
-		}
-		actionSha, _ := receipt["actionEvidenceSha256"].(string)
-		if actionSha == "" {
-			actionSha, _ = receipt["evidenceSha256"].(string)
-		}
-		if len(actionSha) != 64 {
-			return errors.New("actionEvidenceSha256 must be 64-hex string")
-		}
-		if data, readErr := os.ReadFile(actionRel); readErr == nil {
-			sum := sha256.Sum256(data)
-			if hex.EncodeToString(sum[:]) != actionSha {
-				return errors.New("action evidence sha mismatch")
-			}
-		}
-	}
+func r139NativeFile(a r139Admission, field string, rel any, sha any, root string) (map[string]any, error) {
+	selected, err := r139NativeText(rel)
+	if err != nil { return nil, err }
+	digestText, ok := sha.(string)
+	if !ok || !regexp.MustCompile("^[0-9a-f]{64}$").MatchString(digestText) { return nil, fmt.Errorf("%s SHA invalid", field) }
+	path, err := r139NativePath(a, field, selected, root, true)
+	if err != nil { return nil, err }
+	raw, err := os.ReadFile(path)
+	if err != nil { return nil, err }
+	sum := sha256.Sum256(raw)
+	if hex.EncodeToString(sum[:]) != digestText { return nil, fmt.Errorf("%s SHA mismatch", field) }
+	return r139NativeObject(raw)
+}
 
-	invRel, _ := receipt["finalInventoryPath"].(string)
-	if invRel == "" {
-		invRel, _ = receipt["inventoryPath"].(string)
+func r139StrictBrowserReservation(a r139Admission, browserRaw []byte) error {
+	proof, err := r139NativeObject(browserRaw)
+	if err != nil { return err }
+	if !a.BrowserSupported { return errors.New("browser unsupported") }
+	if a.BrowserTool == "supported_CUA_IAB" {
+		if proof["status"] != "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" || proof["browserId"] != "iab" || proof["controlObserved"] != true || proof["captureObserved"] != true || proof["closeObserved"] != true || proof["protocol"] != "http" && proof["protocol"] != "https" || !r139NativeZeroInteger(proof["ownedTabsFinal"]) { return errors.New("IAB proof invalid") }
+		return nil
 	}
-	if invRel != "" {
-		if strings.HasPrefix(invRel, "docs/") {
-			return errors.New("finalInventoryPath must not be in tracked docs/")
-		}
-		invSha, _ := receipt["finalInventorySha256"].(string)
-		if invSha == "" {
-			invSha, _ = receipt["inventorySha256"].(string)
-		}
-		if len(invSha) != 64 {
-			return errors.New("finalInventorySha256 must be 64-hex string")
-		}
-		if data, readErr := os.ReadFile(invRel); readErr == nil {
-			sum := sha256.Sum256(data)
-			if hex.EncodeToString(sum[:]) != invSha {
-				return errors.New("final inventory sha mismatch")
-			}
-			var invData map[string]any
-			if jsonErr := json.Unmarshal(data, &invData); jsonErr == nil {
-				if openPages, ok := invData["openPages"].([]any); ok {
-					for _, p := range openPages {
-						if pStr, ok := p.(string); ok && pStr == a.PageID {
-							return errors.New("page still present in final inventory openPages")
-						}
-					}
-				}
-				if openTabs, ok := invData["openTabs"].([]any); ok {
-					for _, p := range openTabs {
-						if pStr, ok := p.(string); ok && pStr == a.PageID {
-							return errors.New("page still present in final inventory openTabs")
-						}
-					}
-				}
-				if ownedTabs, ok := invData["ownedTabsFinal"].(float64); ok && ownedTabs != 0 {
-					return errors.New("ownedTabsFinal in inventory must be 0")
-				}
-			}
-		}
+	if a.BrowserTool != "antigravity_native" { return errors.New("unsupported browser tool") }
+	if proof["status"] != "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE" || proof["browserId"] != "antigravity_native" || proof["exclusiveReservation"] != true || proof["controlObserved"] != true || proof["captureObserved"] != true || proof["protocol"] != "http" && proof["protocol"] != "https" { return errors.New("native reservation incomplete") }
+	if observed, exists := proof["closeObserved"]; exists && observed != false { return errors.New("early closeObserved forbidden") }
+	if err := r139NativeIdentity(a, proof); err != nil { return err }
+	_, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
+	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
+	sealed, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, readiness)
+	if err != nil || !reflect.DeepEqual(sealed, proof) { return errors.New("reservation file or digest mismatch") }
+	if strings.HasPrefix(readiness+string(os.PathSeparator), product+string(os.PathSeparator)) || strings.HasPrefix(product+string(os.PathSeparator), readiness+string(os.PathSeparator)) { return errors.New("native roots overlap") }
+	if info, err := os.Stat(readiness); err != nil || !info.IsDir() { return errors.New("readiness root missing") }
+	if _, err := os.Stat(product); err == nil {
+		if os.Getenv("R139_RUN") != "1" || os.Getenv("R139_RUNTIME_ROOT") != product { return errors.New("product runtime root preexisting outside admitted run") }
+	} else if !os.IsNotExist(err) { return err }
+	reserved, err := r139NativeTime(proof["reservedAtUTC"]); if err != nil { return err }
+	previousEvidence := time.Time{}
+	for _, event := range []string{"creation", "control", "capture"} {
+		data, err := r139NativeFile(a, event, proof[event+"EvidencePath"], proof[event+"EvidenceSha256"], readiness)
+		if err != nil { return err }
+		if err := r139NativeIdentity(a, data); err != nil || data["event"] != event { return fmt.Errorf("%s original event invalid", event) }
+		at, err := r139NativeTime(data["atUTC"])
+		if err != nil || at.After(reserved) || !previousEvidence.IsZero() && at.Before(previousEvidence) { return fmt.Errorf("%s chronology invalid", event) }
+		previousEvidence = at
+	}
+	for field, selected := range map[string]string{"browserCloseReceiptPath": a.BrowserCloseReceiptPath, "actionEvidencePath": a.ActionEvidencePath, "finalInventoryPath": a.FinalInventoryPath, "rawNativeInventoryPath": a.RawNativeInventoryPath} {
+		if _, err := r139NativePath(a, field, selected, product, false); err != nil { return err }
 	}
+	return nil
+}
 
+func r139StrictBrowserTerminalClose(a r139Admission, closeRaw []byte) error {
+	receipt, err := r139NativeObject(closeRaw); if err != nil { return err }
+	if receipt["status"] != "CLOSED_EXACT_OWNED_PAGE" || receipt["browserId"] != "antigravity_native" || receipt["closeObserved"] != true || !r139NativeZeroInteger(receipt["ownedTabsFinal"]) { return errors.New("terminal close incomplete") }
+	if err := r139NativeIdentity(a, receipt); err != nil { return err }
+	_, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
+	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
+	closePath, err := r139NativePath(a, "browserCloseReceipt", a.BrowserCloseReceiptPath, product, true)
+	if err != nil { return err }
+	storedClose, err := os.ReadFile(closePath)
+	if err != nil || !bytes.Equal(storedClose, closeRaw) { return errors.New("terminal close file differs from validated bytes") }
+	reservation, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, readiness)
+	if err != nil { return err }
+	if err := r139NativeIdentity(a, reservation); err != nil { return err }
+	reservedEvidence := time.Time{}
+	for _, event := range []string{"creation", "control", "capture"} {
+		data, err := r139NativeFile(a, event, reservation[event+"EvidencePath"], reservation[event+"EvidenceSha256"], readiness)
+		if err != nil { return err }
+		if err := r139NativeIdentity(a, data); err != nil || data["event"] != event { return errors.New("original readiness evidence invalid") }
+		at, err := r139NativeTime(data["atUTC"])
+		if err != nil || !reservedEvidence.IsZero() && at.Before(reservedEvidence) { return errors.New("original readiness chronology invalid") }
+		reservedEvidence = at
+	}
+	reserved, err := r139NativeTime(reservation["reservedAtUTC"]); if err != nil { return err }
+	receiptReserved, err := r139NativeTime(receipt["reservedAtUTC"]); if err != nil { return err }
+	if !receiptReserved.Equal(reserved) { return errors.New("reservation time mismatch") }
+	if reservedEvidence.After(reserved) { return errors.New("readiness after reservation") }
+	actionAt, err := r139NativeTime(receipt["actionAtUTC"]); if err != nil { return err }
+	closed, err := r139NativeTime(receipt["closedAtUTC"]); if err != nil { return err }
+	deadline, err := time.Parse("2006-01-02T15:04:05Z", a.GrantDeadlineUTC); if err != nil { return err }
+	if reserved.After(actionAt) || actionAt.After(closed) || closed.After(deadline) { return errors.New("native chronology invalid") }
+	if receipt["actionEvidencePath"] != a.ActionEvidencePath || receipt["finalInventoryPath"] != a.FinalInventoryPath { return errors.New("future evidence path mismatch") }
+	action, err := r139NativeFile(a, "action", receipt["actionEvidencePath"], receipt["actionEvidenceSha256"], product)
+	if err != nil { return err }
+	if err := r139NativeIdentity(a, action); err != nil || action["event"] != "action_sequence" || action["atUTC"] != receipt["actionAtUTC"] { return errors.New("action evidence mismatch") }
+	rows, ok := action["actions"].([]any)
+	if !ok || len(rows) < 2 { return errors.New("action sequence missing") }
+	previous := reserved
+	for index, item := range rows {
+		row, ok := item.(map[string]any)
+		if !ok || row["pageId"] != a.PageID { return errors.New("action page invalid") }
+		kind := row["action"]
+		if kind != "control" && kind != "capture" && kind != "close" { return errors.New("action kind invalid") }
+		at, err := r139NativeTime(row["atUTC"])
+		if err != nil || at.Before(previous) || at.After(closed) { return errors.New("action sequence time invalid") }
+		if kind == "close" && (index != len(rows)-1 || !at.Equal(closed)) || kind != "close" && index == len(rows)-1 { return errors.New("post-close action or missing close") }
+		previous = at
+	}
+	inv, err := r139NativeFile(a, "finalInventory", receipt["finalInventoryPath"], receipt["finalInventorySha256"], product)
+	if err != nil { return err }
+	if err := r139NativeIdentity(a, inv); err != nil { return err }
+	observed, err := r139NativeTime(inv["inventoryAtUTC"])
+	if err != nil || observed.Before(closed) || observed.After(deadline) { return errors.New("inventory time invalid") }
+	pages, ok := inv["pageIds"].([]any)
+	if !ok || !r139NativeZeroInteger(inv["ownedCount"]) { return errors.New("inventory list or count invalid") }
+	seen := map[string]bool{}
+	for _, page := range pages {
+		name, err := r139NativeText(page)
+		if err != nil || seen[name] || name == a.PageID { return errors.New("inventory page invalid") }
+		seen[name] = true
+	}
+	if inv["rawNativeInventoryPath"] != a.RawNativeInventoryPath { return errors.New("raw inventory path mismatch") }
+	rawInventory, err := r139NativeFile(a, "rawNativeInventory", inv["rawNativeInventoryPath"], inv["rawNativeInventorySha256"], product)
+	if err != nil { return err }
+	if err := r139NativeIdentity(a, rawInventory); err != nil { return err }
+	if rawInventory["event"] != "native_inventory" || rawInventory["atUTC"] != inv["inventoryAtUTC"] || !reflect.DeepEqual(rawInventory["pageIds"], inv["pageIds"]) || rawInventory["ownedCount"] != inv["ownedCount"] { return errors.New("raw native inventory mismatch") }
 	return nil
 }
 
@@ -470,350 +334,248 @@ func TestR139PureRouteAdmission(t *testing.T) {
 	if auth || !execute || read { t.Fatal("R139 execute route") }
 }
 
-func TestR139PureBrowserReservationAdmission(t *testing.T) {
-	iabAdmission := r139Admission{
-		BrowserSupported: true,
-		BrowserTool:      "supported_CUA_IAB",
-	}
-	validIABRaw := []byte(`{
-		"status": "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE",
-		"browserId": "iab",
-		"controlObserved": true,
-		"captureObserved": true,
-		"closeObserved": true,
-		"ownedTabsFinal": 0,
-		"protocol": "http"
-	}`)
-	if err := r139VerifyBrowserReservation(iabAdmission, validIABRaw); err != nil {
-		t.Fatalf("expected legacy IAB pass, got: %v", err)
-	}
-
-	badIABRaw := []byte(`{
-		"status": "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE",
-		"browserId": "iab",
-		"controlObserved": true,
-		"captureObserved": true,
-		"closeObserved": true,
-		"ownedTabsFinal": 1,
-		"protocol": "http"
-	}`)
-	if err := r139VerifyBrowserReservation(iabAdmission, badIABRaw); err == nil {
-		t.Fatal("expected legacy IAB rejection for ownedTabsFinal != 0")
-	}
-
-	emptyToolAdmission := r139Admission{
-		BrowserSupported: true,
-		BrowserTool:      "",
-	}
-	if err := r139VerifyBrowserReservation(emptyToolAdmission, validIABRaw); err == nil {
-		t.Fatal("expected rejection when browserTool is empty (no empty string fallback)")
-	}
+func r139NativeTestWrite(t *testing.T, path string, value any) string {
+	t.Helper()
+	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { t.Fatal(err) }
+	raw, err := json.Marshal(value)
+	if err != nil { t.Fatal(err) }
+	if err := os.WriteFile(path, raw, 0644); err != nil { t.Fatal(err) }
+	sum := sha256.Sum256(raw)
+	return hex.EncodeToString(sum[:])
+}
 
-	nativeAdmission := r139Admission{
-		TrancheID:               "CCMAI-RUNTIME-139",
-		BrowserSupported:        true,
-		BrowserTool:             "antigravity_native",
-		PageID:                  "page-test-123",
-		CampaignNonce:           "nonce-abc-456",
-		BrowserSessionID:        "session-xyz-789",
-		FallbackCloseOperator:   "Owner",
-		BrowserCloseReceiptPath: ".cvf/runtime/test/close.json",
-	}
-	validNativeRaw := []byte(`{
-		"status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
-		"browserId": "antigravity_native",
-		"exclusiveReservation": true,
-		"controlObserved": true,
-		"captureObserved": true,
-		"closeObserved": false,
-		"protocol": "http",
-		"closeMode": "operator_fallback",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"trancheId": "CCMAI-RUNTIME-139",
-		"operator": "Owner",
-		"reservedAtUTC": "2026-10-23T12:00:00Z"
-	}`)
-	if err := r139VerifyBrowserReservation(nativeAdmission, validNativeRaw); err != nil {
-		t.Fatalf("expected native reservation pass, got: %v", err)
+func r139NativeTestIdentity(a r139Admission) map[string]any {
+	return map[string]any{
+		"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID,
+		"campaignNonce": a.CampaignNonce, "pageId": a.PageID,
+		"browserSessionId": a.BrowserSessionID, "operator": a.FallbackCloseOperator,
+		"closeMode": a.CloseMode,
 	}
+}
 
-	finalCloseAsReservation := []byte(`{
-		"status": "CLOSED_EXACT_OWNED_PAGE",
-		"browserId": "antigravity_native",
-		"exclusiveReservation": true,
-		"controlObserved": true,
-		"captureObserved": true,
-		"closeObserved": true,
-		"ownedTabsFinal": 0,
-		"protocol": "http",
-		"closeMode": "operator_fallback",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"operator": "Owner"
-	}`)
-	if err := r139VerifyBrowserReservation(nativeAdmission, finalCloseAsReservation); err == nil {
-		t.Fatal("expected rejection of final-close receipt as prefixture reservation")
-	}
+func r139NativeTestFixture(t *testing.T) (r139Admission, map[string]any, func(int) string) {
+	t.Helper()
+	project := t.TempDir()
+	readiness := filepath.Join(project, ".cvf", "runtime", "readiness")
+	product := filepath.Join(project, ".cvf", "runtime", "product")
+	if err := os.MkdirAll(readiness, 0755); err != nil { t.Fatal(err) }
+	now := time.Now().UTC().Truncate(time.Second)
+	stamp := func(offset int) string { return now.Add(time.Duration(offset) * time.Second).Format(time.RFC3339) }
+	a := r139Admission{
+		NativeContractID: "r139-native-v1", TrancheID: "CCMAI-RUNTIME-139",
+		CampaignNonce: "nonce-456", PageID: "page-123",
+		BrowserSessionID: "session-789", Operator: "Owner", FallbackCloseOperator: "Owner",
+		CloseMode: "operator_fallback", BrowserSupported: true,
+		BrowserTool: "antigravity_native", BrowserReadinessRoot: ".cvf/runtime/readiness",
+		RuntimeRoot: product, BrowserReceiptPath: ".cvf/runtime/readiness/reservation.json",
+		BrowserCloseReceiptPath: ".cvf/runtime/product/close.json",
+		ActionEvidencePath: ".cvf/runtime/product/action.json",
+		FinalInventoryPath: ".cvf/runtime/product/inventory.json",
+		RawNativeInventoryPath: ".cvf/runtime/product/raw_inventory.json",
+		GrantDeadlineUTC: stamp(3600), nativeProjectRoot: project,
+	}
+	reservation := r139NativeTestIdentity(a)
+	reservation["status"] = "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE"
+	reservation["browserId"] = "antigravity_native"
+	reservation["exclusiveReservation"] = true
+	reservation["controlObserved"] = true
+	reservation["captureObserved"] = true
+	reservation["closeObserved"] = false
+	reservation["protocol"] = "http"
+	reservation["reservedAtUTC"] = stamp(-15)
+	for index, event := range []string{"creation", "control", "capture"} {
+		evidence := r139NativeTestIdentity(a)
+		evidence["event"] = event
+		evidence["atUTC"] = stamp(-30 + index*5)
+		rel := ".cvf/runtime/readiness/" + event + ".json"
+		reservation[event+"EvidencePath"] = rel
+		reservation[event+"EvidenceSha256"] = r139NativeTestWrite(t, filepath.Join(readiness, event+".json"), evidence)
+	}
+	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(readiness, "reservation.json"), reservation)
+	return a, reservation, stamp
+}
 
-	earlyCloseRaw := []byte(`{
-		"status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
-		"browserId": "antigravity_native",
-		"exclusiveReservation": true,
-		"controlObserved": true,
-		"captureObserved": true,
-		"closeObserved": true,
-		"protocol": "http",
-		"closeMode": "operator_fallback",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"operator": "Owner"
-	}`)
-	if err := r139VerifyBrowserReservation(nativeAdmission, earlyCloseRaw); err == nil {
-		t.Fatal("expected rejection of early closeObserved=true in reservation")
-	}
+func r139NativeTestTerminal(t *testing.T, a r139Admission, reservation map[string]any, stamp func(int) string) map[string]any {
+	t.Helper()
+	product := a.RuntimeRoot
+	raw := r139NativeTestIdentity(a)
+	raw["event"] = "native_inventory"
+	raw["atUTC"] = stamp(-4)
+	raw["pageIds"] = []string{}
+	raw["ownedCount"] = 0
+	rawSHA := r139NativeTestWrite(t, filepath.Join(product, "raw_inventory.json"), raw)
+	action := r139NativeTestIdentity(a)
+	action["event"] = "action_sequence"
+	action["atUTC"] = stamp(-8)
+	action["actions"] = []map[string]any{
+		{"pageId": a.PageID, "action": "control", "atUTC": stamp(-10)},
+		{"pageId": a.PageID, "action": "close", "atUTC": stamp(-5)},
+	}
+	actionSHA := r139NativeTestWrite(t, filepath.Join(product, "action.json"), action)
+	inv := r139NativeTestIdentity(a)
+	inv["inventoryAtUTC"] = stamp(-4)
+	inv["pageIds"] = []string{}
+	inv["ownedCount"] = 0
+	inv["rawNativeInventoryPath"] = a.RawNativeInventoryPath
+	inv["rawNativeInventorySha256"] = rawSHA
+	invSHA := r139NativeTestWrite(t, filepath.Join(product, "inventory.json"), inv)
+	close := r139NativeTestIdentity(a)
+	close["status"] = "CLOSED_EXACT_OWNED_PAGE"
+	close["browserId"] = "antigravity_native"
+	close["closeObserved"] = true
+	close["ownedTabsFinal"] = 0
+	close["reservedAtUTC"] = reservation["reservedAtUTC"]
+	close["actionAtUTC"] = stamp(-8)
+	close["closedAtUTC"] = stamp(-5)
+	close["actionEvidencePath"] = a.ActionEvidencePath
+	close["actionEvidenceSha256"] = actionSHA
+	close["finalInventoryPath"] = a.FinalInventoryPath
+	close["finalInventorySha256"] = invSHA
+	r139NativeTestWrite(t, filepath.Join(product, "close.json"), close)
+	return close
+}
 
-	for _, tc := range []struct {
-		name string
-		raw  []byte
-	}{
-		{"wrong-trancheId", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","trancheId":"wrong-tranche","operator":"Owner"}`)},
-		{"wrong-pageId", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"wrong-page","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","trancheId":"CCMAI-RUNTIME-139","operator":"Owner"}`)},
-		{"wrong-nonce", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"wrong-nonce","browserSessionId":"session-xyz-789","trancheId":"CCMAI-RUNTIME-139","operator":"Owner"}`)},
-		{"wrong-session", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"wrong-session","trancheId":"CCMAI-RUNTIME-139","operator":"Owner"}`)},
-		{"wrong-operator", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","trancheId":"CCMAI-RUNTIME-139","operator":"wrong-operator"}`)},
-		{"unsupported-protocol", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"ftp","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","trancheId":"CCMAI-RUNTIME-139","operator":"Owner"}`)},
-		{"invalid-closeMode", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"invalid_mode","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","trancheId":"CCMAI-RUNTIME-139","operator":"Owner"}`)},
-		{"missing-exclusive", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":false,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","trancheId":"CCMAI-RUNTIME-139","operator":"Owner"}`)},
-		{"future-reservation-time", []byte(`{"status":"RESERVED_NATIVE_PAGE_CONTROL_CAPTURE","browserId":"antigravity_native","exclusiveReservation":true,"controlObserved":true,"captureObserved":true,"protocol":"http","closeMode":"operator_fallback","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","trancheId":"CCMAI-RUNTIME-139","operator":"Owner","reservedAtUTC":"2099-01-01T00:00:00Z"}`)},
+func TestR139PureBrowserReservationAdmission(t *testing.T) {
+	iab := r139Admission{BrowserSupported: true, BrowserTool: "supported_CUA_IAB"}
+	validIAB := []byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0,\"protocol\":\"http\"}")
+	if err := r139VerifyBrowserReservation(iab, validIAB); err != nil { t.Fatalf("legacy IAB parity: %v", err) }
+	for _, raw := range [][]byte{
+		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":false,\"protocol\":\"http\"}"),
+		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0.5,\"protocol\":\"http\"}"),
+		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0.0,\"protocol\":\"http\"}"),
+		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0e0,\"protocol\":\"http\"}"),
 	} {
-		if err := r139VerifyBrowserReservation(nativeAdmission, tc.raw); err == nil {
-			t.Fatalf("expected error for %s, got nil", tc.name)
-		}
-	}
-
-	missingClosePathAdmission := nativeAdmission
-	missingClosePathAdmission.BrowserCloseReceiptPath = ""
-	if err := r139VerifyBrowserReservation(missingClosePathAdmission, validNativeRaw); err == nil {
-		t.Fatal("expected rejection when admission lacks browserCloseReceiptPath")
-	}
-
-	docsClosePathAdmission := nativeAdmission
-	docsClosePathAdmission.BrowserCloseReceiptPath = "docs/reviews/probes/close.json"
-	if err := r139VerifyBrowserReservation(docsClosePathAdmission, validNativeRaw); err == nil {
-		t.Fatal("expected rejection when browserCloseReceiptPath is under tracked docs/")
-	}
-
-	unsupportedAdmission := nativeAdmission
-	unsupportedAdmission.BrowserSupported = false
-	if err := r139VerifyBrowserReservation(unsupportedAdmission, validNativeRaw); err == nil {
-		t.Fatal("expected rejection when browserSupported is false")
-	}
-
-	unknownToolAdmission := nativeAdmission
-	unknownToolAdmission.BrowserTool = "unknown_tool"
-	if err := r139VerifyBrowserReservation(unknownToolAdmission, validNativeRaw); err == nil {
-		t.Fatal("expected rejection for unsupported browserTool")
+		if err := r139VerifyBrowserReservation(iab, raw); err == nil { t.Fatal("IAB noninteger count accepted") }
 	}
+	a, reservation, _ := r139NativeTestFixture(t)
+	raw, _ := json.Marshal(reservation)
+	if err := r139VerifyBrowserReservation(a, raw); err != nil { t.Fatalf("complete reservation rejected: %v", err) }
+	for name, changed := range map[string]any{
+		"missing_contract": nil, "wrong_mode": "api", "wrong_type_time": false,
+	} {
+		copy := map[string]any{}
+		for key, value := range reservation { copy[key] = value }
+		switch name {
+		case "missing_contract": copy["nativeContractId"] = changed
+		case "wrong_mode": copy["closeMode"] = changed
+		case "wrong_type_time": copy["reservedAtUTC"] = changed
+		}
+		bad, _ := json.Marshal(copy)
+		modified := a
+		modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), copy)
+		if err := r139VerifyBrowserReservation(modified, bad); err == nil { t.Fatalf("%s accepted", name) }
+	}
+	for _, field := range []string{"creationEvidencePath", "controlEvidenceSha256", "captureEvidencePath"} {
+		copy := map[string]any{}
+		for key, value := range reservation { copy[key] = value }
+		delete(copy, field)
+		bad, _ := json.Marshal(copy)
+		modified := a
+		modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), copy)
+		if err := r139VerifyBrowserReservation(modified, bad); err == nil { t.Fatalf("missing %s accepted", field) }
+	}
+	for _, field := range []string{"trancheId", "campaignNonce", "pageId", "browserSessionId", "operator"} {
+		copy := map[string]any{}
+		for key, value := range reservation { copy[key] = value }
+		copy[field] = "wrong"
+		bad, _ := json.Marshal(copy)
+		modified := a
+		modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), copy)
+		if err := r139VerifyBrowserReservation(modified, bad); err == nil { t.Fatalf("wrong %s accepted", field) }
+	}
+	r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), reservation)
+	badRoot := a
+	badRoot.BrowserReadinessRoot = ".cvf/runtime/../escape"
+	if err := r139VerifyBrowserReservation(badRoot, raw); err == nil { t.Fatal("root traversal accepted") }
 }
 
 func TestR139PureBrowserTerminalClose(t *testing.T) {
-	tempDir := t.TempDir()
-	actionFile := filepath.Join(tempDir, "action.json")
-	actionContent := []byte(`{"action":"click","pageId":"page-test-123"}`)
-	if err := os.WriteFile(actionFile, actionContent, 0644); err != nil {
-		t.Fatal(err)
-	}
-	actionShaBytes := sha256.Sum256(actionContent)
-	actionSha := hex.EncodeToString(actionShaBytes[:])
-
-	invFile := filepath.Join(tempDir, "inventory.json")
-	invContent := []byte(`{"openPages":[],"openTabs":[],"ownedTabsFinal":0}`)
-	if err := os.WriteFile(invFile, invContent, 0644); err != nil {
-		t.Fatal(err)
-	}
-	invShaBytes := sha256.Sum256(invContent)
-	invSha := hex.EncodeToString(invShaBytes[:])
-
-	deadline := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
-	nativeAdmission := r139Admission{
-		TrancheID:             "CCMAI-RUNTIME-139",
-		BrowserSupported:      true,
-		BrowserTool:           "antigravity_native",
-		PageID:                "page-test-123",
-		CampaignNonce:         "nonce-abc-456",
-		BrowserSessionID:      "session-xyz-789",
-		FallbackCloseOperator: "Owner",
-		GrantDeadlineUTC:      deadline,
-	}
-
-	validCloseRaw := []byte(`{
-		"status": "CLOSED_EXACT_OWNED_PAGE",
-		"browserId": "antigravity_native",
-		"closeObserved": true,
-		"ownedTabsFinal": 0,
-		"trancheId": "CCMAI-RUNTIME-139",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"closeMode": "operator_fallback",
-		"operator": "Owner",
-		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `",
-		"actionEvidencePath": ` + fmt.Sprintf("%q", filepath.ToSlash(actionFile)) + `,
-		"actionEvidenceSha256": "` + actionSha + `",
-		"finalInventoryPath": ` + fmt.Sprintf("%q", filepath.ToSlash(invFile)) + `,
-		"finalInventorySha256": "` + invSha + `"
-	}`)
-
-	if err := r139VerifyBrowserTerminalClose(nativeAdmission, validCloseRaw); err != nil {
-		t.Fatalf("expected terminal close pass, got: %v", err)
-	}
-
-	notClosedRaw := []byte(`{
-		"status": "CLOSED_EXACT_OWNED_PAGE",
-		"browserId": "antigravity_native",
-		"closeObserved": false,
-		"ownedTabsFinal": 0,
-		"trancheId": "CCMAI-RUNTIME-139",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"closeMode": "operator_fallback",
-		"operator": "Owner",
-		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `"
-	}`)
-	if err := r139VerifyBrowserTerminalClose(nativeAdmission, notClosedRaw); err == nil {
-		t.Fatal("expected rejection when closeObserved is false")
-	}
-
-	tabsRemainingRaw := []byte(`{
-		"status": "CLOSED_EXACT_OWNED_PAGE",
-		"browserId": "antigravity_native",
-		"closeObserved": true,
-		"ownedTabsFinal": 1,
-		"trancheId": "CCMAI-RUNTIME-139",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"closeMode": "operator_fallback",
-		"operator": "Owner",
-		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `"
-	}`)
-	if err := r139VerifyBrowserTerminalClose(nativeAdmission, tabsRemainingRaw); err == nil {
-		t.Fatal("expected rejection when ownedTabsFinal != 0")
-	}
-
-	wrongStatusRaw := []byte(`{
-		"status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
-		"browserId": "antigravity_native",
-		"closeObserved": true,
-		"ownedTabsFinal": 0,
-		"trancheId": "CCMAI-RUNTIME-139",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"closeMode": "operator_fallback",
-		"operator": "Owner",
-		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `"
-	}`)
-	if err := r139VerifyBrowserTerminalClose(nativeAdmission, wrongStatusRaw); err == nil {
-		t.Fatal("expected rejection for non-closed status")
-	}
-
-	for _, tc := range []struct {
-		name string
-		raw  []byte
-	}{
-		{"wrong-trancheId", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"wrong-tranche","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
-		{"wrong-pageId", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"CCMAI-RUNTIME-139","pageId":"wrong","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
-		{"wrong-nonce", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"CCMAI-RUNTIME-139","pageId":"page-test-123","campaignNonce":"wrong","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
-		{"wrong-session", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"CCMAI-RUNTIME-139","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"wrong","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
-		{"wrong-operator", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"CCMAI-RUNTIME-139","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"wrong","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
-		{"invalid-closeMode", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"CCMAI-RUNTIME-139","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"bad","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)},
-		{"wrong-action-sha", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"CCMAI-RUNTIME-139","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `","actionEvidencePath":` + fmt.Sprintf("%q", filepath.ToSlash(actionFile)) + `,"actionEvidenceSha256":"0000000000000000000000000000000000000000000000000000000000000000"}`)},
-		{"docs-action-path", []byte(`{"status":"CLOSED_EXACT_OWNED_PAGE","browserId":"antigravity_native","closeObserved":true,"ownedTabsFinal":0,"trancheId":"CCMAI-RUNTIME-139","pageId":"page-test-123","campaignNonce":"nonce-abc-456","browserSessionId":"session-xyz-789","closeMode":"operator_fallback","operator":"Owner","closedAtUTC":"` + time.Now().UTC().Format(time.RFC3339) + `","actionEvidencePath":"docs/action.json","actionEvidenceSha256":"` + actionSha + `"}`)},
+	a, reservation, stamp := r139NativeTestFixture(t)
+	rawReservation, _ := json.Marshal(reservation)
+	if err := r139VerifyBrowserReservation(a, rawReservation); err != nil { t.Fatal(err) }
+	close := r139NativeTestTerminal(t, a, reservation, stamp)
+	raw, _ := json.Marshal(close)
+	if err := r139VerifyBrowserTerminalClose(a, raw); err != nil { t.Fatalf("complete terminal rejected: %v", err) }
+	for _, spelling := range []string{"0.0", "0e0"} {
+		fractional := bytes.Replace(raw, []byte("\"ownedTabsFinal\":0"), []byte("\"ownedTabsFinal\":"+spelling), 1)
+		if bytes.Equal(fractional, raw) { t.Fatal("terminal numeric fixture replacement missing") }
+		if err := os.WriteFile(filepath.Join(a.RuntimeRoot, "close.json"), fractional, 0644); err != nil { t.Fatal(err) }
+		if err := r139VerifyBrowserTerminalClose(a, fractional); err == nil { t.Fatalf("terminal count %s accepted", spelling) }
+	}
+	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), close)
+	for name, changed := range map[string]any{
+		"missing_action_time": nil, "future_close": stamp(60),
+		"wrong_inventory_sha": strings.Repeat("x", 64), "wrong_mode": "api",
+		"false_count": false,
 	} {
-		if err := r139VerifyBrowserTerminalClose(nativeAdmission, tc.raw); err == nil {
-			t.Fatalf("expected error for %s, got nil", tc.name)
-		}
-	}
-
-	futureTime := time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
-	futureRaw := []byte(`{
-		"status": "CLOSED_EXACT_OWNED_PAGE",
-		"browserId": "antigravity_native",
-		"closeObserved": true,
-		"ownedTabsFinal": 0,
-		"trancheId": "CCMAI-RUNTIME-139",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"closeMode": "operator_fallback",
-		"operator": "Owner",
-		"closedAtUTC": "` + futureTime + `"
-	}`)
-	if err := r139VerifyBrowserTerminalClose(nativeAdmission, futureRaw); err == nil {
-		t.Fatal("expected rejection for future timestamp")
-	}
-
-	pastDeadlineAdmission := nativeAdmission
-	pastDeadlineAdmission.GrantDeadlineUTC = time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
-	if err := r139VerifyBrowserTerminalClose(pastDeadlineAdmission, validCloseRaw); err == nil {
-		t.Fatal("expected rejection when closedAtUTC is after grantDeadlineUTC")
-	}
-
-	missingDeadlineAdmission := nativeAdmission
-	missingDeadlineAdmission.GrantDeadlineUTC = ""
-	if err := r139VerifyBrowserTerminalClose(missingDeadlineAdmission, validCloseRaw); err == nil {
-		t.Fatal("expected rejection when grantDeadlineUTC is missing in admission")
-	}
-
-	badTimeRaw := []byte(`{
-		"status": "CLOSED_EXACT_OWNED_PAGE",
-		"browserId": "antigravity_native",
-		"closeObserved": true,
-		"ownedTabsFinal": 0,
-		"trancheId": "CCMAI-RUNTIME-139",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"closeMode": "operator_fallback",
-		"operator": "Owner",
-		"closedAtUTC": "invalid-time"
-	}`)
-	if err := r139VerifyBrowserTerminalClose(nativeAdmission, badTimeRaw); err == nil {
-		t.Fatal("expected rejection for invalid closedAtUTC")
-	}
-
-	dirtyInvFile := filepath.Join(tempDir, "dirty_inv.json")
-	dirtyInvContent := []byte(`{"openPages":["page-test-123"],"openTabs":[],"ownedTabsFinal":0}`)
-	if err := os.WriteFile(dirtyInvFile, dirtyInvContent, 0644); err != nil {
-		t.Fatal(err)
-	}
-	dirtyInvShaBytes := sha256.Sum256(dirtyInvContent)
-	dirtyInvSha := hex.EncodeToString(dirtyInvShaBytes[:])
-
-	dirtyInvRaw := []byte(`{
-		"status": "CLOSED_EXACT_OWNED_PAGE",
-		"browserId": "antigravity_native",
-		"closeObserved": true,
-		"ownedTabsFinal": 0,
-		"trancheId": "CCMAI-RUNTIME-139",
-		"pageId": "page-test-123",
-		"campaignNonce": "nonce-abc-456",
-		"browserSessionId": "session-xyz-789",
-		"closeMode": "operator_fallback",
-		"operator": "Owner",
-		"closedAtUTC": "` + time.Now().UTC().Format(time.RFC3339) + `",
-		"finalInventoryPath": ` + fmt.Sprintf("%q", filepath.ToSlash(dirtyInvFile)) + `,
-		"finalInventorySha256": "` + dirtyInvSha + `"
-	}`)
-	if err := r139VerifyBrowserTerminalClose(nativeAdmission, dirtyInvRaw); err == nil {
-		t.Fatal("expected rejection when target page is still present in inventory")
-	}
+		copy := map[string]any{}
+		for key, value := range close { copy[key] = value }
+		switch name {
+		case "missing_action_time": copy["actionAtUTC"] = changed
+		case "future_close": copy["closedAtUTC"] = changed
+		case "wrong_inventory_sha": copy["finalInventorySha256"] = changed
+		case "wrong_mode": copy["closeMode"] = changed
+		case "false_count": copy["ownedTabsFinal"] = changed
+		}
+		bad, _ := json.Marshal(copy)
+		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), copy)
+		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("%s accepted", name) }
+	}
+	for _, field := range []string{"trancheId", "campaignNonce", "pageId", "browserSessionId", "operator", "nativeContractId"} {
+		copy := map[string]any{}
+		for key, value := range close { copy[key] = value }
+		copy[field] = "wrong"
+		bad, _ := json.Marshal(copy)
+		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), copy)
+		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("wrong terminal %s accepted", field) }
+	}
+	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), close)
+	inventoryPath := filepath.Join(a.RuntimeRoot, "inventory.json")
+	inventory, err := r139NativeObject(func() []byte { raw, readErr := os.ReadFile(inventoryPath); if readErr != nil { t.Fatal(readErr) }; return raw }())
+	if err != nil { t.Fatal(err) }
+	for _, spelling := range []string{"0.0", "0e0"} {
+		original, readErr := os.ReadFile(inventoryPath); if readErr != nil { t.Fatal(readErr) }
+		fractional := bytes.Replace(original, []byte("\"ownedCount\":0"), []byte("\"ownedCount\":"+spelling), 1)
+		if bytes.Equal(fractional, original) { t.Fatal("inventory numeric fixture replacement missing") }
+		if err := os.WriteFile(inventoryPath, fractional, 0644); err != nil { t.Fatal(err) }
+		sum := sha256.Sum256(fractional)
+		mutated := map[string]any{}
+		for key, value := range close { mutated[key] = value }
+		mutated["finalInventorySha256"] = hex.EncodeToString(sum[:])
+		bad, _ := json.Marshal(mutated)
+		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), mutated)
+		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("inventory count %s accepted", spelling) }
+		if err := os.WriteFile(inventoryPath, original, 0644); err != nil { t.Fatal(err) }
+	}
+	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), close)
+	for name, changed := range map[string]any{"missing_pages": nil, "page_open": []string{a.PageID}, "wrong_count": false, "missing_time": nil} {
+		copy := map[string]any{}
+		for key, value := range inventory { copy[key] = value }
+		switch name {
+		case "missing_pages", "page_open": copy["pageIds"] = changed
+		case "wrong_count": copy["ownedCount"] = changed
+		case "missing_time": copy["inventoryAtUTC"] = changed
+		}
+		sha := r139NativeTestWrite(t, inventoryPath, copy)
+		mutated := map[string]any{}
+		for key, value := range close { mutated[key] = value }
+		mutated["finalInventorySha256"] = sha
+		bad, _ := json.Marshal(mutated)
+		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), mutated)
+		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("%s inventory accepted", name) }
+	}
+	r139NativeTestWrite(t, inventoryPath, inventory)
+	actionPath := filepath.Join(a.RuntimeRoot, "action.json")
+	actionRaw, err := os.ReadFile(actionPath); if err != nil { t.Fatal(err) }
+	action, err := r139NativeObject(actionRaw); if err != nil { t.Fatal(err) }
+	action["actions"] = append(action["actions"].([]any), map[string]any{"pageId": a.PageID, "action": "control", "atUTC": stamp(-4)})
+	sha := r139NativeTestWrite(t, actionPath, action)
+	mutated := map[string]any{}
+	for key, value := range close { mutated[key] = value }
+	mutated["actionEvidenceSha256"] = sha
+	bad, _ := json.Marshal(mutated)
+	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), mutated)
+	if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatal("post-close action accepted") }
 }
 
 func r139Admit(t *testing.T) (r139Admission, string) {
diff --git a/scripts/r139_execution_supervisor.py b/scripts/r139_execution_supervisor.py
index a9b680de..3d84ca0b 100644
--- a/scripts/r139_execution_supervisor.py
+++ b/scripts/r139_execution_supervisor.py
@@ -125,7 +125,100 @@ def committed(relative: str) -> Path:
     return path
 
 
-def verify_browser_receipt(value: dict) -> None:
+NATIVE_ID_FIELDS = ("nativeContractId", "trancheId", "campaignNonce", "pageId", "browserSessionId", "operator", "closeMode")
+NATIVE_EVENT_FIELDS = ("creation", "control", "capture")
+
+
+def native_text(value: object, code: str) -> str:
+    require(type(value) is str and 1 <= len(value) <= 128 and value == value.strip() and
+            all(32 <= ord(character) < 127 for character in value), code)
+    return value
+
+
+def native_path_text(value: object, code: str) -> str:
+    require(type(value) is str and 1 <= len(value) <= 512 and value == value.strip() and
+            all(32 <= ord(character) < 127 for character in value), code)
+    return value
+
+
+def native_time(value: object, code: str) -> datetime:
+    require(type(value) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", value) is not None, code)
+    try:
+        parsed = datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
+    except ValueError as exc:
+        raise Hold(code) from exc
+    require(parsed <= datetime.now(timezone.utc), code)
+    return parsed
+
+
+def native_root(value: dict, field: str) -> Path:
+    rel = native_path_text(value.get(field), "NATIVE_ROOT_INVALID")
+    if field == "runtimeRoot":
+        require(Path(rel).is_absolute(), "NATIVE_ROOT_INVALID")
+        candidate = Path(rel)
+    else:
+        require(not Path(rel).is_absolute() and "\\" not in rel and ".." not in Path(rel).parts and
+                rel.startswith(".cvf/runtime/"), "NATIVE_ROOT_INVALID")
+        candidate = ROOT / rel
+    root = candidate.resolve()
+    require(candidate == root and root.is_relative_to((ROOT / ".cvf/runtime").resolve()), "NATIVE_ROOT_INVALID")
+    return root
+
+
+def native_file(rel: object, sha: object, root: Path, code: str) -> tuple[Path, dict]:
+    relative = native_path_text(rel, code)
+    require(not Path(relative).is_absolute() and "\\" not in relative and ".." not in Path(relative).parts,
+            code)
+    require(type(sha) is str and HEX64.fullmatch(sha) is not None, code)
+    candidate = ROOT / relative
+    path = candidate.resolve()
+    require(candidate == path and path.is_relative_to(root) and path != root and path.is_file(), code)
+    try:
+        require(digest(path) == sha, code)
+        return path, read_json(path)
+    except (OSError, ValueError) as exc:
+        raise Hold(code) from exc
+
+
+def native_identity(admission: dict, record: dict, code: str) -> None:
+    expected = {field: admission.get(field) for field in NATIVE_ID_FIELDS}
+    require(expected["closeMode"] in ("api", "operator", "operator_fallback"), code)
+    if expected["closeMode"] == "operator_fallback":
+        require(admission.get("fallbackCloseOperator") == expected["operator"], code)
+    for field in NATIVE_ID_FIELDS:
+        native_text(expected[field], code)
+        require(type(record.get(field)) is str and record[field] == expected[field], code)
+
+
+def native_event(admission: dict, path: object, sha: object, root: Path, event: str) -> dict:
+    _, data = native_file(path, sha, root, "NATIVE_EVIDENCE_INVALID")
+    native_identity(admission, data, "NATIVE_EVIDENCE_IDENTITY_MISMATCH")
+    require(data.get("event") == event, "NATIVE_EVIDENCE_EVENT_INVALID")
+    native_time(data.get("atUTC"), "NATIVE_EVIDENCE_TIME_INVALID")
+    return data
+
+
+def native_roots(admission: dict) -> tuple[Path, Path]:
+    readiness = native_root(admission, "browserReadinessRoot")
+    runtime = native_root(admission, "runtimeRoot")
+    require(readiness != runtime and not readiness.is_relative_to(runtime) and
+            not runtime.is_relative_to(readiness), "NATIVE_ROOTS_OVERLAP")
+    require(readiness.is_dir(), "READINESS_ROOT_MISSING")
+    return readiness, runtime
+
+
+def native_future_path(admission: dict, field: str, root: Path, *, absent: bool) -> Path:
+    rel = native_path_text(admission.get(field), "FUTURE_EVIDENCE_PATH_INVALID")
+    require(not Path(rel).is_absolute() and "\\" not in rel and ".." not in Path(rel).parts,
+            "FUTURE_EVIDENCE_PATH_INVALID")
+    candidate = ROOT / rel
+    resolved = candidate.resolve()
+    require(candidate == resolved and resolved.is_relative_to(root) and resolved != root and
+            (not absent or not resolved.exists()), "FUTURE_EVIDENCE_PATH_INVALID")
+    return resolved
+
+
+def verify_browser_receipt(value: dict, *, require_runtime_absent: bool = True) -> None:
     tool = value.get("browserTool")
     require(tool in ("supported_CUA_IAB", "antigravity_native") and value.get("browserSupported") is True, "BROWSER_CAPABILITY_UNPROVEN")
     browser_rel = value.get("browserReceiptPath", "")
@@ -142,159 +235,111 @@ def verify_browser_receipt(value: dict) -> None:
         require(type(proof.get("ownedTabsFinal")) is int and not isinstance(proof.get("ownedTabsFinal"), bool) and proof["ownedTabsFinal"] == 0, "BROWSER_PROOF_CONTENT_INCOMPLETE")
         require(proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
     elif tool == "antigravity_native":
+        readiness, runtime = native_roots(value)
+        require(browser.is_relative_to(readiness), "RESERVATION_PATH_OUTSIDE_READINESS_ROOT")
+        _, proof = native_file(browser_rel, expected, readiness, "RESERVATION_RECEIPT_INVALID")
         require(proof.get("status") != "CLOSED_EXACT_OWNED_PAGE", "TERMINAL_CLOSE_RECEIPT_CANNOT_BE_PREFIXTURE_RESERVATION")
         require(proof.get("status") == "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE", "BROWSER_PROOF_CONTENT_INCOMPLETE")
         require(proof.get("browserId") == "antigravity_native", "BROWSER_PROOF_CONTENT_INCOMPLETE")
         require(type(proof.get("exclusiveReservation")) is bool and proof["exclusiveReservation"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
         require(type(proof.get("controlObserved")) is bool and proof["controlObserved"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
         require(type(proof.get("captureObserved")) is bool and proof["captureObserved"] is True, "BROWSER_PROOF_CONTENT_INCOMPLETE")
-        close_obs = proof.get("closeObserved")
-        require(close_obs is None or (type(close_obs) is bool and close_obs is False), "EARLY_CLOSE_FORBIDDEN")
+        require("closeObserved" not in proof or proof["closeObserved"] is False, "EARLY_CLOSE_FORBIDDEN")
         require(proof.get("protocol") in ("http", "https"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
-        require(proof.get("closeMode") in ("api", "operator", "operator_fallback"), "BROWSER_PROOF_CONTENT_INCOMPLETE")
-
-        tranche_id = value.get("trancheId")
-        require(isinstance(tranche_id, str) and tranche_id.strip(), "BROWSER_PROOF_IDENTITY_MISMATCH")
-        require(proof.get("trancheId") == tranche_id, "BROWSER_PROOF_IDENTITY_MISMATCH")
-
-        page_id = value.get("pageId")
-        require(isinstance(page_id, str) and page_id.strip(), "BROWSER_PROOF_IDENTITY_MISMATCH")
-        require(proof.get("pageId") == page_id, "BROWSER_PROOF_IDENTITY_MISMATCH")
-
-        nonce = value.get("campaignNonce")
-        require(isinstance(nonce, str) and nonce.strip(), "BROWSER_PROOF_IDENTITY_MISMATCH")
-        require(proof.get("campaignNonce") == nonce, "BROWSER_PROOF_IDENTITY_MISMATCH")
-
-        session_id = value.get("browserSessionId")
-        require(isinstance(session_id, str) and session_id.strip(), "BROWSER_PROOF_IDENTITY_MISMATCH")
-        require(proof.get("browserSessionId") == session_id, "BROWSER_PROOF_IDENTITY_MISMATCH")
-
-        expected_op = value.get("fallbackCloseOperator") or value.get("operator")
-        require(isinstance(expected_op, str) and expected_op.strip(), "BROWSER_PROOF_IDENTITY_MISMATCH")
-        require(proof.get("operator") == expected_op, "BROWSER_PROOF_IDENTITY_MISMATCH")
-
-        close_path_rel = value.get("browserCloseReceiptPath")
-        require(isinstance(close_path_rel, str) and close_path_rel.strip(), "BROWSER_CLOSE_RECEIPT_PATH_MISSING")
-        require(not close_path_rel.startswith("docs/"), "BROWSER_CLOSE_RECEIPT_PATH_MUST_NOT_BE_TRACKED_DOCS")
-
-        reserved_str = proof.get("reservedAtUTC")
-        if reserved_str is not None:
-            require(isinstance(reserved_str, str) and reserved_str.endswith("Z"), "RESERVATION_TIMESTAMP_INVALID")
-            try:
-                reserved_dt = datetime.fromisoformat(reserved_str[:-1] + "+00:00")
-            except Exception:
-                raise Hold("RESERVATION_TIMESTAMP_INVALID")
-            now_utc = datetime.now(timezone.utc)
-            require(reserved_dt <= now_utc + timedelta(seconds=60), "RESERVATION_TIMESTAMP_IN_FUTURE")
-
-        ev_rel = proof.get("actionEvidencePath") or proof.get("evidencePath") or proof.get("controlEvidencePath")
-        if ev_rel is not None:
-            require(isinstance(ev_rel, str) and ev_rel.strip(), "RESERVATION_EVIDENCE_PATH_INVALID")
-            require(not ev_rel.startswith("docs/"), "EVIDENCE_PATH_MUST_NOT_BE_TRACKED_DOCS")
-            ev_path = project_path(ev_rel)
-            ev_sha = proof.get("actionEvidenceSha256") or proof.get("evidenceSha256") or proof.get("controlEvidenceSha256")
-            require(isinstance(ev_sha, str) and HEX64.fullmatch(ev_sha) and digest(ev_path) == ev_sha, "RESERVATION_EVIDENCE_SHA_MISMATCH")
+        native_identity(value, proof, "BROWSER_PROOF_IDENTITY_MISMATCH")
+        if require_runtime_absent:
+            require(not runtime.exists(), "PRODUCT_RUNTIME_ROOT_PREEXISTING")
+        reserved = native_time(proof.get("reservedAtUTC"), "RESERVATION_TIMESTAMP_INVALID")
+        previous_evidence = None
+        for event in NATIVE_EVENT_FIELDS:
+            data = native_event(value, proof.get(event + "EvidencePath"),
+                                proof.get(event + "EvidenceSha256"), readiness, event)
+            at = native_time(data["atUTC"], "NATIVE_EVIDENCE_TIME_INVALID")
+            require(at <= reserved and (previous_evidence is None or previous_evidence <= at),
+                    "RESERVATION_EVIDENCE_CHRONOLOGY_INVALID")
+            previous_evidence = at
+        for field in ("browserCloseReceiptPath", "actionEvidencePath", "finalInventoryPath", "rawNativeInventoryPath"):
+            native_future_path(value, field, runtime, absent=require_runtime_absent)
     else:
         raise Hold("BROWSER_TOOL_UNSUPPORTED")
 
 
 def verify_browser_terminal_close(admission: dict, runtime: Path, external: Path) -> dict:
-    close_rel = admission.get("browserCloseReceiptPath")
-    require(isinstance(close_rel, str) and close_rel.strip(), "BROWSER_CLOSE_RECEIPT_PATH_MISSING")
-    require(not close_rel.startswith("docs/"), "BROWSER_CLOSE_RECEIPT_PATH_MUST_NOT_BE_TRACKED_DOCS")
-    close_path = project_path(close_rel)
-    expected_sha = admission.get("browserCloseReceiptSha256")
-    if expected_sha:
-        require(isinstance(expected_sha, str) and HEX64.fullmatch(expected_sha) and digest(close_path) == expected_sha, "BROWSER_CLOSE_RECEIPT_SHA_MISMATCH")
-    receipt = read_json(close_path)
+    verify_browser_receipt(admission, require_runtime_absent=False)
+    readiness, product_root = native_roots(admission)
+    require(runtime.resolve() == product_root, "PRODUCT_RUNTIME_ROOT_MISMATCH")
+    _, reservation = native_file(admission.get("browserReceiptPath"), admission.get("browserReceiptSha256"),
+                                  readiness, "RESERVATION_RECEIPT_INVALID")
+    native_identity(admission, reservation, "RESERVATION_IDENTITY_MISMATCH")
+    close_path = native_future_path(admission, "browserCloseReceiptPath", product_root, absent=False)
+    require(close_path.is_file(), "BROWSER_CLOSE_RECEIPT_INVALID")
+    try:
+        receipt = read_json(close_path)
+        close_sha = digest(close_path)
+    except (OSError, ValueError) as exc:
+        raise Hold("BROWSER_CLOSE_RECEIPT_INVALID") from exc
+    native_identity(admission, receipt, "BROWSER_CLOSE_IDENTITY_MISMATCH")
     require(receipt.get("status") == "CLOSED_EXACT_OWNED_PAGE", "BROWSER_CLOSE_STATUS_INVALID")
     require(receipt.get("browserId") == "antigravity_native", "BROWSER_CLOSE_BROWSER_ID_INVALID")
-    close_obs = receipt.get("closeObserved")
-    require(type(close_obs) is bool and close_obs is True, "BROWSER_CLOSE_NOT_OBSERVED")
+    require(receipt.get("closeObserved") is True, "BROWSER_CLOSE_NOT_OBSERVED")
     owned_tabs = receipt.get("ownedTabsFinal")
-    require(type(owned_tabs) is int and not isinstance(owned_tabs, bool) and owned_tabs == 0, "BROWSER_CLOSE_OWNED_TABS_REMAINING")
-
-    tranche_id = admission.get("trancheId")
-    require(isinstance(tranche_id, str) and tranche_id.strip(), "BROWSER_CLOSE_IDENTITY_MISSING")
-    require(receipt.get("trancheId") == tranche_id, "BROWSER_CLOSE_TRANCHE_ID_MISMATCH")
-
-    page_id = admission.get("pageId")
-    require(isinstance(page_id, str) and page_id.strip(), "BROWSER_CLOSE_PAGE_ID_MISSING")
-    require(receipt.get("pageId") == page_id, "BROWSER_CLOSE_PAGE_ID_MISMATCH")
-
-    nonce = admission.get("campaignNonce")
-    require(isinstance(nonce, str) and nonce.strip(), "BROWSER_CLOSE_NONCE_MISSING")
-    require(receipt.get("campaignNonce") == nonce, "BROWSER_CLOSE_NONCE_MISMATCH")
-
-    session_id = admission.get("browserSessionId")
-    require(isinstance(session_id, str) and session_id.strip(), "BROWSER_CLOSE_SESSION_MISSING")
-    require(receipt.get("browserSessionId") == session_id, "BROWSER_CLOSE_SESSION_MISMATCH")
-
-    close_mode = receipt.get("closeMode")
-    require(close_mode in ("api", "operator", "operator_fallback"), "BROWSER_CLOSE_MODE_INVALID")
-    expected_op = admission.get("fallbackCloseOperator") or admission.get("operator")
-    operator = receipt.get("operator")
-    require(isinstance(operator, str) and operator.strip(), "BROWSER_CLOSE_OPERATOR_MISSING")
-    require(operator == expected_op, "BROWSER_CLOSE_OPERATOR_MISMATCH")
-
-    closed_at_str = receipt.get("closedAtUTC")
-    require(isinstance(closed_at_str, str) and closed_at_str.endswith("Z"), "BROWSER_CLOSE_TIMESTAMP_INVALID")
+    require(type(owned_tabs) is int and owned_tabs == 0, "BROWSER_CLOSE_OWNED_TABS_REMAINING")
+    reserved = native_time(reservation.get("reservedAtUTC"), "RESERVATION_TIMESTAMP_INVALID")
+    require(receipt.get("reservedAtUTC") == reservation.get("reservedAtUTC"), "RESERVATION_TIME_MISMATCH")
+    action_at = native_time(receipt.get("actionAtUTC"), "ACTION_TIMESTAMP_INVALID")
+    closed = native_time(receipt.get("closedAtUTC"), "BROWSER_CLOSE_TIMESTAMP_INVALID")
     try:
-        closed_dt = datetime.fromisoformat(closed_at_str[:-1] + "+00:00")
-    except Exception:
-        raise Hold("BROWSER_CLOSE_TIMESTAMP_INVALID")
-    now_utc = datetime.now(timezone.utc)
-    require(closed_dt <= now_utc + timedelta(seconds=60), "BROWSER_CLOSE_TIMESTAMP_IN_FUTURE")
-
-    deadline_str = admission.get("grantDeadlineUTC")
-    require(isinstance(deadline_str, str) and deadline_str.endswith("Z"), "GRANT_DEADLINE_INVALID")
-    try:
-        deadline_dt = datetime.fromisoformat(deadline_str[:-1] + "+00:00")
-    except (ValueError, TypeError):
-        raise Hold("GRANT_DEADLINE_INVALID")
-    require(closed_dt <= deadline_dt, "BROWSER_CLOSE_AFTER_DEADLINE")
-
-    reserved_str = receipt.get("reservedAtUTC")
-    if reserved_str:
-        require(isinstance(reserved_str, str) and reserved_str.endswith("Z"), "RESERVATION_TIMESTAMP_INVALID")
-        try:
-            reserved_dt = datetime.fromisoformat(reserved_str[:-1] + "+00:00")
-        except Exception:
-            raise Hold("RESERVATION_TIMESTAMP_INVALID")
-        require(reserved_dt <= closed_dt, "BROWSER_CLOSE_BEFORE_RESERVATION")
-
-    action_rel = receipt.get("actionEvidencePath") or receipt.get("evidencePath")
-    if action_rel:
-        require(isinstance(action_rel, str) and action_rel.strip(), "BROWSER_CLOSE_ACTION_EVIDENCE_PATH_INVALID")
-        require(not action_rel.startswith("docs/"), "EVIDENCE_PATH_MUST_NOT_BE_TRACKED_DOCS")
-        action_path = project_path(action_rel)
-        action_sha = receipt.get("actionEvidenceSha256") or receipt.get("evidenceSha256")
-        require(isinstance(action_sha, str) and HEX64.fullmatch(action_sha), "BROWSER_CLOSE_ACTION_EVIDENCE_SHA_MISSING")
-        require(digest(action_path) == action_sha, "BROWSER_CLOSE_ACTION_EVIDENCE_SHA_MISMATCH")
-
-    inv_rel = receipt.get("finalInventoryPath") or receipt.get("inventoryPath")
-    if inv_rel:
-        require(isinstance(inv_rel, str) and inv_rel.strip(), "BROWSER_CLOSE_FINAL_INVENTORY_PATH_INVALID")
-        require(not inv_rel.startswith("docs/"), "INVENTORY_PATH_MUST_NOT_BE_TRACKED_DOCS")
-        inv_path = project_path(inv_rel)
-        inv_sha = receipt.get("finalInventorySha256") or receipt.get("inventorySha256")
-        require(isinstance(inv_sha, str) and HEX64.fullmatch(inv_sha), "BROWSER_CLOSE_FINAL_INVENTORY_SHA_MISSING")
-        require(digest(inv_path) == inv_sha, "BROWSER_CLOSE_FINAL_INVENTORY_SHA_MISMATCH")
-        inv_data = read_json(inv_path)
-        open_pages = inv_data.get("openPages", [])
-        open_tabs = inv_data.get("openTabs", [])
-        require(page_id not in open_pages and page_id not in open_tabs, "PAGE_STILL_OPEN_IN_FINAL_INVENTORY")
-        if "ownedTabsFinal" in inv_data:
-            require(inv_data["ownedTabsFinal"] == 0, "BROWSER_CLOSE_OWNED_TABS_REMAINING")
-        if "inventoryAtUTC" in inv_data:
-            inv_ts = inv_data["inventoryAtUTC"]
-            require(isinstance(inv_ts, str) and inv_ts.endswith("Z"), "INVENTORY_TIMESTAMP_INVALID")
-            try:
-                inv_dt = datetime.fromisoformat(inv_ts[:-1] + "+00:00")
-            except Exception:
-                raise Hold("INVENTORY_TIMESTAMP_INVALID")
-            require(closed_dt <= inv_dt <= deadline_dt, "BROWSER_INVENTORY_CHRONOLOGY_INVALID")
-    return receipt
+        deadline_text = admission.get("grantDeadlineUTC")
+        require(type(deadline_text) is str and re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ", deadline_text), "GRANT_DEADLINE_INVALID")
+        deadline = datetime.strptime(deadline_text, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
+    except ValueError as exc:
+        raise Hold("GRANT_DEADLINE_INVALID") from exc
+    require(reserved <= action_at <= closed <= deadline, "BROWSER_CHRONOLOGY_INVALID")
+    require(receipt.get("actionEvidencePath") == admission.get("actionEvidencePath") and
+            receipt.get("finalInventoryPath") == admission.get("finalInventoryPath"), "FUTURE_EVIDENCE_PATH_MISMATCH")
+    native_future_path(admission, "actionEvidencePath", product_root, absent=False)
+    native_future_path(admission, "finalInventoryPath", product_root, absent=False)
+    native_future_path(admission, "rawNativeInventoryPath", product_root, absent=False)
+    action = native_event(admission, receipt.get("actionEvidencePath"), receipt.get("actionEvidenceSha256"),
+                          product_root, "action_sequence")
+    actions = action.get("actions")
+    require(type(actions) is list and len(actions) >= 2, "ACTION_SEQUENCE_INVALID")
+    previous = reserved
+    close_index = None
+    for index, row in enumerate(actions):
+        require(type(row) is dict and type(row.get("pageId")) is str and row["pageId"] == admission["pageId"] and
+                row.get("action") in ("control", "capture", "close"), "ACTION_SEQUENCE_INVALID")
+        at = native_time(row.get("atUTC"), "ACTION_SEQUENCE_TIME_INVALID")
+        require(previous <= at <= closed, "ACTION_SEQUENCE_CHRONOLOGY_INVALID")
+        previous = at
+        if row["action"] == "close":
+            require(close_index is None and at == closed, "ACTION_SEQUENCE_CLOSE_INVALID")
+            close_index = index
+    require(close_index == len(actions) - 1 and native_time(action["atUTC"], "ACTION_TIMESTAMP_INVALID") == action_at,
+            "POST_CLOSE_ACTION_OR_ACTION_TIME_MISMATCH")
+    _, inventory = native_file(receipt.get("finalInventoryPath"), receipt.get("finalInventorySha256"),
+                               product_root, "FINAL_INVENTORY_INVALID")
+    native_identity(admission, inventory, "FINAL_INVENTORY_IDENTITY_MISMATCH")
+    observed = native_time(inventory.get("inventoryAtUTC"), "INVENTORY_TIMESTAMP_INVALID")
+    require(closed <= observed <= deadline, "INVENTORY_CHRONOLOGY_INVALID")
+    pages = inventory.get("pageIds")
+    require(type(pages) is list and all(type(page) is str and native_text(page, "PAGE_ID_INVALID") for page in pages) and
+            len(pages) == len(set(pages)) and admission["pageId"] not in pages, "FINAL_PAGE_LIST_INVALID")
+    require(type(inventory.get("ownedCount")) is int and inventory["ownedCount"] == 0,
+            "FINAL_OWNED_COUNT_INVALID")
+    require(inventory.get("rawNativeInventoryPath") == admission.get("rawNativeInventoryPath"),
+            "RAW_NATIVE_INVENTORY_PATH_MISMATCH")
+    _, raw_inventory = native_file(inventory.get("rawNativeInventoryPath"), inventory.get("rawNativeInventorySha256"),
+                                   product_root, "RAW_NATIVE_INVENTORY_INVALID")
+    native_identity(admission, raw_inventory, "RAW_NATIVE_INVENTORY_IDENTITY_MISMATCH")
+    require(raw_inventory.get("event") == "native_inventory" and
+            raw_inventory.get("atUTC") == inventory.get("inventoryAtUTC") and
+            raw_inventory.get("pageIds") == pages and
+            type(raw_inventory.get("ownedCount")) is int and raw_inventory["ownedCount"] == inventory["ownedCount"],
+            "RAW_NATIVE_INVENTORY_MISMATCH")
+    return {"receipt": receipt, "closeReceiptSha256": close_sha,
+            "actionEvidenceSha256": receipt["actionEvidenceSha256"],
+            "finalInventorySha256": receipt["finalInventorySha256"]}
 
 
 SOURCE_PIN_REQUIRED = {"backend/api/handlers/intervention_execution_browser_test.go", "scripts/r139_execution_supervisor.py", "scripts/r139_vite_config.mjs.template", "scripts/tests/test_r139_execution_supervisor.py", "backend/go.mod", "backend/go.sum", "frontend/package.json", "frontend/package-lock.json", "frontend/vite.config.ts"}
@@ -322,7 +367,7 @@ def fresh_runtime_admission(relative: str, *, expect_fresh_external: bool = True
     dirty = subprocess.run(["git", "status", "--porcelain", "--untracked-files=all"], cwd=ROOT, capture_output=True, timeout=10)
     require(dirty.returncode == 0 and not dirty.stdout.strip(), "SOURCE_WORKTREE_DIRTY")
     require(value.get("nativeExecutePOST") == EXPECTED_NATIVE and value.get("browserExecutePOST") == EXPECTED_BROWSER and value.get("grantCap") == EXPECTED_GRANTS and value.get("providerCap") == 1 and value.get("automaticRetry") == 0, "EFFECT_CAPS_INVALID")
-    verify_browser_receipt(value)
+    verify_browser_receipt(value, require_runtime_absent=expect_fresh_external)
     if expect_fresh_external and value.get("browserTool") == "antigravity_native":
         close_path_rel = value.get("browserCloseReceiptPath")
         if close_path_rel:
diff --git a/scripts/tests/test_r139_execution_supervisor.py b/scripts/tests/test_r139_execution_supervisor.py
index 57727036..ff4bfe24 100644
--- a/scripts/tests/test_r139_execution_supervisor.py
+++ b/scripts/tests/test_r139_execution_supervisor.py
@@ -2,6 +2,7 @@
 from __future__ import annotations
 
 import importlib.util
+from contextlib import ExitStack
 import json
 from pathlib import Path
 import subprocess
@@ -606,497 +607,240 @@ class R139SupervisorPureTests(unittest.TestCase):
             with self.assertRaisesRegex(module.Hold, "EFFECT_ROUTE_MISMATCH"):
                 module._validate_route_rows(rows, "TEST", {"executePOST": 1, "authPOST": 3})
 
-    def test_native_antigravity_reservation_admission_positive(self):
-        with tempfile.TemporaryDirectory() as name:
-            temp_dir = Path(name)
-            evidence = temp_dir / "evidence.json"
-            evidence.write_text(json.dumps({"action": "reserve", "captured": True}), encoding="utf-8")
-            evidence_sha = module.digest(evidence)
-
-            receipt = temp_dir / "reservation.json"
-            now_iso = (datetime.now(timezone.utc) - timedelta(seconds=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
-            receipt.write_text(json.dumps({
-                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
-                "browserId": "antigravity_native",
-                "exclusiveReservation": True,
-                "controlObserved": True,
-                "captureObserved": True,
-                "closeObserved": False,
-                "protocol": "http",
+    def _native_fixture(self, root):
+        with patch.object(module, "ROOT", root):
+            readiness = root / ".cvf/runtime/readiness"
+            runtime = root / ".cvf/runtime/product"
+            readiness.mkdir(parents=True)
+            now = datetime.now(timezone.utc).replace(microsecond=0)
+            stamp = lambda seconds: (now + timedelta(seconds=seconds)).strftime("%Y-%m-%dT%H:%M:%SZ")
+            identity = {
+                "nativeContractId": "r139-native-v1", "trancheId": "CCMAI-RUNTIME-139",
+                "campaignNonce": "nonce-456", "pageId": "page-123",
+                "browserSessionId": "session-789", "operator": "Owner",
                 "closeMode": "operator_fallback",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "operator": "Owner",
-                "reservedAtUTC": now_iso,
-                "actionEvidencePath": ".cvf/runtime/evidence.json",
-                "actionEvidenceSha256": evidence_sha,
-            }), encoding="utf-8")
-            admission = {
-                "browserTool": "antigravity_native",
-                "browserSupported": True,
-                "browserReceiptPath": "reservation.json",
-                "browserReceiptSha256": module.digest(receipt),
-                "browserCloseReceiptPath": ".cvf/runtime/close.json",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "fallbackCloseOperator": "Owner",
             }
-            def fake_project_path(rel):
-                if rel == "reservation.json":
-                    return receipt
-                if rel == ".cvf/runtime/evidence.json":
-                    return evidence
-                return temp_dir / rel
-
-            with patch.object(module, "project_path", side_effect=fake_project_path):
-                module.verify_browser_receipt(admission)
-
-    def test_native_antigravity_early_close_rejected(self):
-        with tempfile.TemporaryDirectory() as name:
-            temp_dir = Path(name)
-            receipt = temp_dir / "reservation.json"
-            receipt.write_text(json.dumps({
-                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
-                "browserId": "antigravity_native",
-                "exclusiveReservation": True,
-                "controlObserved": True,
-                "captureObserved": True,
-                "closeObserved": True,
-                "protocol": "http",
-                "closeMode": "operator_fallback",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "operator": "Owner",
-            }), encoding="utf-8")
             admission = {
-                "browserTool": "antigravity_native",
-                "browserSupported": True,
-                "browserReceiptPath": "reservation.json",
-                "browserReceiptSha256": module.digest(receipt),
-                "browserCloseReceiptPath": ".cvf/runtime/close.json",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
+                **identity, "browserTool": "antigravity_native", "browserSupported": True,
                 "fallbackCloseOperator": "Owner",
+                "browserReadinessRoot": ".cvf/runtime/readiness",
+                "runtimeRoot": str(runtime),
+                "browserReceiptPath": ".cvf/runtime/readiness/reservation.json",
+                "browserCloseReceiptPath": ".cvf/runtime/product/close.json",
+                "actionEvidencePath": ".cvf/runtime/product/action.json",
+                "finalInventoryPath": ".cvf/runtime/product/inventory.json",
+                "rawNativeInventoryPath": ".cvf/runtime/product/raw_inventory.json",
+                "grantDeadlineUTC": stamp(3600),
             }
-            with patch.object(module, "project_path", return_value=receipt):
-                with self.assertRaisesRegex(module.Hold, "EARLY_CLOSE_FORBIDDEN"):
-                    module.verify_browser_receipt(admission)
-
-                receipt.write_text(json.dumps({
-                    "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
-                    "browserId": "antigravity_native",
-                    "exclusiveReservation": True,
-                    "controlObserved": True,
-                    "captureObserved": True,
-                    "closeObserved": "true",
-                    "protocol": "http",
-                    "closeMode": "operator_fallback",
-                    "pageId": "page-123",
-                    "campaignNonce": "nonce-456",
-                    "browserSessionId": "session-789",
-                    "trancheId": "CCMAI-RUNTIME-139",
-                    "operator": "Owner",
-                }), encoding="utf-8")
-                admission["browserReceiptSha256"] = module.digest(receipt)
-                with self.assertRaisesRegex(module.Hold, "EARLY_CLOSE_FORBIDDEN"):
-                    module.verify_browser_receipt(admission)
-
-    def test_native_antigravity_final_close_cannot_be_prefixture_reservation(self):
-        with tempfile.TemporaryDirectory() as name:
-            receipt = Path(name) / "reservation.json"
-            receipt.write_text(json.dumps({
-                "status": "CLOSED_EXACT_OWNED_PAGE",
-                "browserId": "antigravity_native",
-                "exclusiveReservation": True,
-                "controlObserved": True,
-                "captureObserved": True,
-                "closeObserved": True,
-                "ownedTabsFinal": 0,
-                "protocol": "http",
-                "closeMode": "operator_fallback",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "operator": "Owner",
+            reservation = {
+                **identity, "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
+                "browserId": "antigravity_native", "exclusiveReservation": True,
+                "controlObserved": True, "captureObserved": True,
+                "closeObserved": False, "protocol": "http", "reservedAtUTC": stamp(-15),
+            }
+            for event, seconds in (("creation", -30), ("control", -25), ("capture", -20)):
+                path = readiness / (event + ".json")
+                path.write_text(json.dumps({**identity, "event": event, "atUTC": stamp(seconds)}), encoding="utf-8")
+                reservation[event + "EvidencePath"] = ".cvf/runtime/readiness/" + event + ".json"
+                reservation[event + "EvidenceSha256"] = module.digest(path)
+            receipt_path = readiness / "reservation.json"
+            receipt_path.write_text(json.dumps(reservation), encoding="utf-8")
+            admission["browserReceiptSha256"] = module.digest(receipt_path)
+            return admission, reservation, runtime, stamp, identity
+
+    def _native_terminal(self, root, admission, reservation, runtime, stamp, identity):
+        with patch.object(module, "ROOT", root):
+            runtime.mkdir(exist_ok=True)
+            raw = runtime / "raw_inventory.json"
+            raw.write_text(json.dumps({**identity, "event": "native_inventory", "atUTC": stamp(-4), "pageIds": [], "ownedCount": 0}), encoding="utf-8")
+            actions = [
+                {"pageId": admission["pageId"], "action": "control", "atUTC": stamp(-10)},
+                {"pageId": admission["pageId"], "action": "close", "atUTC": stamp(-5)},
+            ]
+            action = runtime / "action.json"
+            action.write_text(json.dumps({**identity, "event": "action_sequence", "atUTC": stamp(-8), "actions": actions}), encoding="utf-8")
+            inventory = runtime / "inventory.json"
+            inventory.write_text(json.dumps({
+                **identity, "inventoryAtUTC": stamp(-4), "pageIds": [], "ownedCount": 0,
+                "rawNativeInventoryPath": admission["rawNativeInventoryPath"],
+                "rawNativeInventorySha256": module.digest(raw),
             }), encoding="utf-8")
-            admission = {
-                "browserTool": "antigravity_native",
-                "browserSupported": True,
-                "browserReceiptPath": "reservation.json",
-                "browserReceiptSha256": module.digest(receipt),
-                "browserCloseReceiptPath": ".cvf/runtime/close.json",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "fallbackCloseOperator": "Owner",
+            close = {
+                **identity, "status": "CLOSED_EXACT_OWNED_PAGE", "browserId": "antigravity_native",
+                "closeObserved": True, "ownedTabsFinal": 0, "reservedAtUTC": reservation["reservedAtUTC"],
+                "actionAtUTC": stamp(-8), "closedAtUTC": stamp(-5),
+                "actionEvidencePath": admission["actionEvidencePath"],
+                "actionEvidenceSha256": module.digest(action),
+                "finalInventoryPath": admission["finalInventoryPath"],
+                "finalInventorySha256": module.digest(inventory),
             }
-            with patch.object(module, "project_path", return_value=receipt):
-                with self.assertRaisesRegex(module.Hold, "TERMINAL_CLOSE_RECEIPT_CANNOT_BE_PREFIXTURE_RESERVATION"):
-                    module.verify_browser_receipt(admission)
+            (runtime / "close.json").write_text(json.dumps(close), encoding="utf-8")
+            return close
 
-    def test_native_antigravity_identity_mismatches_and_missing_close_path_rejected(self):
+    def test_native_complete_reservation_and_terminal(self):
         with tempfile.TemporaryDirectory() as name:
-            temp_dir = Path(name)
-            evidence = temp_dir / "evidence.json"
-            evidence.write_text(json.dumps({"action": "reserve"}), encoding="utf-8")
-            evidence_sha = module.digest(evidence)
-
-            receipt = temp_dir / "reservation.json"
-            now_iso = (datetime.now(timezone.utc) - timedelta(seconds=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
-            base_data = {
-                "status": "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE",
-                "browserId": "antigravity_native",
-                "exclusiveReservation": True,
-                "controlObserved": True,
-                "captureObserved": True,
-                "closeObserved": False,
-                "protocol": "http",
-                "closeMode": "operator_fallback",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "operator": "Owner",
-                "reservedAtUTC": now_iso,
-                "actionEvidencePath": ".cvf/runtime/evidence.json",
-                "actionEvidenceSha256": evidence_sha,
-            }
-            receipt.write_text(json.dumps(base_data), encoding="utf-8")
-            admission = {
-                "browserTool": "antigravity_native",
-                "browserSupported": True,
-                "browserReceiptPath": "reservation.json",
-                "browserReceiptSha256": module.digest(receipt),
-                "browserCloseReceiptPath": ".cvf/runtime/close.json",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "fallbackCloseOperator": "Owner",
-            }
-            def fake_project_path(rel):
-                if rel == "reservation.json":
-                    return receipt
-                if rel == ".cvf/runtime/evidence.json":
-                    return evidence
-                return temp_dir / rel
-
-            with patch.object(module, "project_path", side_effect=fake_project_path):
-                bad_adm = dict(admission, browserCloseReceiptPath="")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_RECEIPT_PATH_MISSING"):
-                    module.verify_browser_receipt(bad_adm)
-                bad_adm = dict(admission, browserCloseReceiptPath="docs/reviews/close.json")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_RECEIPT_PATH_MUST_NOT_BE_TRACKED_DOCS"):
-                    module.verify_browser_receipt(bad_adm)
-                bad_adm = dict(admission, trancheId="")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
-                    module.verify_browser_receipt(bad_adm)
-                bad_adm = dict(admission, trancheId="OTHER-TRANCHE")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
-                    module.verify_browser_receipt(bad_adm)
-                bad_adm = dict(admission, pageId="other-page")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
-                    module.verify_browser_receipt(bad_adm)
-                bad_adm = dict(admission, campaignNonce="other-nonce")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
-                    module.verify_browser_receipt(bad_adm)
-                bad_adm = dict(admission, browserSessionId="other-session")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
-                    module.verify_browser_receipt(bad_adm)
-                bad_adm = dict(admission, fallbackCloseOperator="other-operator")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_PROOF_IDENTITY_MISMATCH"):
-                    module.verify_browser_receipt(bad_adm)
-
-                bad_receipt_data = dict(base_data, reservedAtUTC="invalid-time")
-                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
-                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
-                with self.assertRaisesRegex(module.Hold, "RESERVATION_TIMESTAMP_INVALID"):
-                    module.verify_browser_receipt(admission_with_bad_receipt)
-
-                future_iso = (datetime.now(timezone.utc) + timedelta(minutes=5)).strftime("%Y-%m-%dT%H:%M:%SZ")
-                bad_receipt_data = dict(base_data, reservedAtUTC=future_iso)
-                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
-                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
-                with self.assertRaisesRegex(module.Hold, "RESERVATION_TIMESTAMP_IN_FUTURE"):
-                    module.verify_browser_receipt(admission_with_bad_receipt)
-
-                bad_receipt_data = dict(base_data, actionEvidencePath="docs/reviews/evidence.json")
-                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
-                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
-                with self.assertRaisesRegex(module.Hold, "EVIDENCE_PATH_MUST_NOT_BE_TRACKED_DOCS"):
-                    module.verify_browser_receipt(admission_with_bad_receipt)
-
-                bad_receipt_data = dict(base_data, actionEvidenceSha256="0" * 64)
-                receipt.write_text(json.dumps(bad_receipt_data), encoding="utf-8")
-                admission_with_bad_receipt = dict(admission, browserReceiptSha256=module.digest(receipt))
-                with self.assertRaisesRegex(module.Hold, "RESERVATION_EVIDENCE_SHA_MISMATCH"):
-                    module.verify_browser_receipt(admission_with_bad_receipt)
-
-    def test_verify_browser_terminal_close_positive_and_rejections(self):
-        now = datetime.now(timezone.utc)
-        reserved_time = (now - timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
-        closed_time = (now - timedelta(minutes=5)).strftime("%Y-%m-%dT%H:%M:%SZ")
-        inv_time = (now - timedelta(minutes=2)).strftime("%Y-%m-%dT%H:%M:%SZ")
-        deadline = (now + timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ")
+            root = Path(name)
+            with patch.object(module, "ROOT", root):
+                admission, reservation, runtime, stamp, identity = self._native_fixture(root)
+                module.verify_browser_receipt(admission)
+                close = self._native_terminal(root, admission, reservation, runtime, stamp, identity)
+                result = module.verify_browser_terminal_close(admission, runtime, runtime)
+                self.assertEqual(result["receipt"], close)
+                self.assertEqual(result["closeReceiptSha256"], module.digest(runtime / "close.json"))
 
-        admission = {
-            "browserTool": "antigravity_native",
-            "browserCloseReceiptPath": ".cvf/runtime/close.json",
-            "trancheId": "CCMAI-RUNTIME-139",
-            "pageId": "page-123",
-            "campaignNonce": "nonce-456",
-            "browserSessionId": "session-789",
-            "fallbackCloseOperator": "Owner",
-            "grantDeadlineUTC": deadline,
-        }
+    def test_native_missing_wrong_type_evidence_root_and_chronology_hold(self):
         with tempfile.TemporaryDirectory() as name:
-            temp_dir = Path(name)
-            action_file = temp_dir / "action.json"
-            action_file.write_text(json.dumps({"action": "navigate_and_click"}), encoding="utf-8")
-            action_sha = module.digest(action_file)
-
-            inv_file = temp_dir / "inventory.json"
-            inv_data = {
-                "openPages": ["other-page"],
-                "openTabs": ["other-tab"],
-                "ownedTabsFinal": 0,
-                "inventoryAtUTC": inv_time,
-            }
-            inv_file.write_text(json.dumps(inv_data), encoding="utf-8")
-            inv_sha = module.digest(inv_file)
-
-            close_file = temp_dir / "close.json"
-            valid_close = {
-                "status": "CLOSED_EXACT_OWNED_PAGE",
-                "browserId": "antigravity_native",
-                "closeObserved": True,
-                "ownedTabsFinal": 0,
-                "trancheId": "CCMAI-RUNTIME-139",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "closeMode": "operator_fallback",
-                "operator": "Owner",
-                "reservedAtUTC": reserved_time,
-                "closedAtUTC": closed_time,
-                "actionEvidencePath": ".cvf/runtime/action.json",
-                "actionEvidenceSha256": action_sha,
-                "finalInventoryPath": ".cvf/runtime/inventory.json",
-                "finalInventorySha256": inv_sha,
-            }
-            close_file.write_text(json.dumps(valid_close), encoding="utf-8")
-
-            def fake_project_path(rel):
-                if rel == ".cvf/runtime/close.json":
-                    return close_file
-                if rel == ".cvf/runtime/action.json":
-                    return action_file
-                if rel == ".cvf/runtime/inventory.json":
-                    return inv_file
-                return temp_dir / rel
-
-            with patch.object(module, "project_path", side_effect=fake_project_path):
-                # Positive check
-                result = module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-                self.assertEqual(result["status"], "CLOSED_EXACT_OWNED_PAGE")
-
-                # F01: Deadline swallowed Hold bug check & malformed deadline
-                past_deadline_adm = dict(admission, grantDeadlineUTC=(now - timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ"))
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_AFTER_DEADLINE"):
-                    module.verify_browser_terminal_close(past_deadline_adm, temp_dir, temp_dir)
-
-                bad_deadline_adm = dict(admission, grantDeadlineUTC="malformed-deadlineZ")
-                with self.assertRaisesRegex(module.Hold, "GRANT_DEADLINE_INVALID"):
-                    module.verify_browser_terminal_close(bad_deadline_adm, temp_dir, temp_dir)
-
-                bad_deadline_adm2 = dict(admission, grantDeadlineUTC="")
-                with self.assertRaisesRegex(module.Hold, "GRANT_DEADLINE_INVALID"):
-                    module.verify_browser_terminal_close(bad_deadline_adm2, temp_dir, temp_dir)
-
-                # F02: Identity checks in admission & receipt
-                bad_adm = dict(admission, trancheId="")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_IDENTITY_MISSING"):
-                    module.verify_browser_terminal_close(bad_adm, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, trancheId="OTHER-TRANCHE")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_TRANCHE_ID_MISMATCH"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, pageId="wrong-page")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_PAGE_ID_MISMATCH"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, campaignNonce="wrong-nonce")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_NONCE_MISMATCH"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, browserSessionId="wrong-session")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_SESSION_MISMATCH"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, operator="wrong-op")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OPERATOR_MISMATCH"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                # Restore valid close file for remaining tests
-                close_file.write_text(json.dumps(valid_close), encoding="utf-8")
-
-                # F03 & F04: Evidence & final inventory verification
-                bad_close = dict(valid_close, actionEvidenceSha256="0" * 64)
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_ACTION_EVIDENCE_SHA_MISMATCH"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, finalInventorySha256="0" * 64)
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_FINAL_INVENTORY_SHA_MISMATCH"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                # Target still open in final inventory
-                bad_inv = dict(inv_data, openPages=["page-123"])
-                inv_file.write_text(json.dumps(bad_inv), encoding="utf-8")
-                bad_close = dict(valid_close, finalInventorySha256=module.digest(inv_file))
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "PAGE_STILL_OPEN_IN_FINAL_INVENTORY"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_inv = dict(inv_data, openTabs=["page-123"])
-                inv_file.write_text(json.dumps(bad_inv), encoding="utf-8")
-                bad_close = dict(valid_close, finalInventorySha256=module.digest(inv_file))
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "PAGE_STILL_OPEN_IN_FINAL_INVENTORY"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                # Restore valid inventory
-                inv_file.write_text(json.dumps(inv_data), encoding="utf-8")
-                close_file.write_text(json.dumps(valid_close), encoding="utf-8")
-
-                # F05: Chronology checks
-                bad_close = dict(valid_close, reservedAtUTC=(now - timedelta(minutes=1)).strftime("%Y-%m-%dT%H:%M:%SZ"))
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_BEFORE_RESERVATION"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                future_close = (now + timedelta(minutes=10)).strftime("%Y-%m-%dT%H:%M:%SZ")
-                bad_close = dict(valid_close, closedAtUTC=future_close)
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_TIMESTAMP_IN_FUTURE"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_inv = dict(inv_data, inventoryAtUTC=(now - timedelta(minutes=8)).strftime("%Y-%m-%dT%H:%M:%SZ"))
-                inv_file.write_text(json.dumps(bad_inv), encoding="utf-8")
-                bad_close = dict(valid_close, finalInventorySha256=module.digest(inv_file))
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_INVENTORY_CHRONOLOGY_INVALID"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                # Restore valid inventory & close
-                inv_file.write_text(json.dumps(inv_data), encoding="utf-8")
-                close_file.write_text(json.dumps(valid_close), encoding="utf-8")
-
-                # F06: Type strictness (booleans & integers)
-                bad_close = dict(valid_close, closeObserved=False)
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_NOT_OBSERVED"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, closeObserved="true")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_NOT_OBSERVED"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, ownedTabsFinal=1)
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OWNED_TABS_REMAINING"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, ownedTabsFinal=False)  # boolean False is 0 in python int comparison
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OWNED_TABS_REMAINING"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, ownedTabsFinal="0")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_OWNED_TABS_REMAINING"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                # F07: Rejection of docs/ path for close receipt and evidence
-                bad_adm = dict(admission, browserCloseReceiptPath="docs/reviews/close.json")
-                close_file.write_text(json.dumps(valid_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "BROWSER_CLOSE_RECEIPT_PATH_MUST_NOT_BE_TRACKED_DOCS"):
-                    module.verify_browser_terminal_close(bad_adm, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, actionEvidencePath="docs/reviews/action.json")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "EVIDENCE_PATH_MUST_NOT_BE_TRACKED_DOCS"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
-
-                bad_close = dict(valid_close, finalInventoryPath="docs/reviews/inv.json")
-                close_file.write_text(json.dumps(bad_close), encoding="utf-8")
-                with self.assertRaisesRegex(module.Hold, "INVENTORY_PATH_MUST_NOT_BE_TRACKED_DOCS"):
-                    module.verify_browser_terminal_close(admission, temp_dir, temp_dir)
+            root = Path(name)
+            with patch.object(module, "ROOT", root):
+                admission, reservation, runtime, stamp, identity = self._native_fixture(root)
+                receipt = root / admission["browserReceiptPath"]
+                for field, bad in (("nativeContractId", None), ("closeMode", "api"), ("reservedAtUTC", None),
+                                   ("creationEvidencePath", None), ("controlEvidenceSha256", "x" * 64),
+                                   ("closeObserved", None)):
+                    changed = dict(reservation, **{field: bad})
+                    receipt.write_text(json.dumps(changed), encoding="utf-8")
+                    attempt = dict(admission, browserReceiptSha256=module.digest(receipt))
+                    with self.subTest(field=field), self.assertRaises(module.Hold):
+                        module.verify_browser_receipt(attempt)
+                receipt.write_text(json.dumps(reservation), encoding="utf-8")
+                bad_root = dict(admission, browserReadinessRoot=".cvf/runtime/../elsewhere")
+                with self.assertRaises(module.Hold):
+                    module.verify_browser_receipt(bad_root)
+                close = self._native_terminal(root, admission, reservation, runtime, stamp, identity)
+                close_path = runtime / "close.json"
+                for field, bad in (("actionAtUTC", None), ("closedAtUTC", stamp(60)),
+                                   ("finalInventorySha256", "x" * 64), ("ownedTabsFinal", False),
+                                   ("closeMode", "api"), ("finalInventoryPath", "docs/fake.json")):
+                    close_path.write_text(json.dumps(dict(close, **{field: bad})), encoding="utf-8")
+                    with self.subTest(field=field), self.assertRaises(module.Hold):
+                        module.verify_browser_terminal_close(admission, runtime, runtime)
+                close_path.write_text(json.dumps(close), encoding="utf-8")
+                inventory_path = runtime / "inventory.json"
+                inventory = json.loads(inventory_path.read_text(encoding="utf-8"))
+                for field, bad in (("pageIds", None), ("pageIds", ["page-123"]),
+                                   ("ownedCount", False), ("inventoryAtUTC", None)):
+                    inventory_path.write_text(json.dumps(dict(inventory, **{field: bad})), encoding="utf-8")
+                    close_path.write_text(json.dumps(dict(close, finalInventorySha256=module.digest(inventory_path))), encoding="utf-8")
+                    with self.subTest(field=field), self.assertRaises(module.Hold):
+                        module.verify_browser_terminal_close(admission, runtime, runtime)
+                inventory_path.write_text(json.dumps(inventory), encoding="utf-8")
+                action_path = runtime / "action.json"
+                action = json.loads(action_path.read_text(encoding="utf-8"))
+                action["actions"].append({"pageId": "page-123", "action": "control", "atUTC": stamp(-4)})
+                action_path.write_text(json.dumps(action), encoding="utf-8")
+                close_path.write_text(json.dumps(dict(close, actionEvidenceSha256=module.digest(action_path),
+                                                      finalInventorySha256=module.digest(inventory_path))), encoding="utf-8")
+                with self.assertRaises(module.Hold):
+                    module.verify_browser_terminal_close(admission, runtime, runtime)
 
     def test_supervisor_terminal_close_enforcement_on_success_and_error_paths(self):
-        # F08: Test supervisor terminal evaluation and cleanup preservation on success and error paths
-        with tempfile.TemporaryDirectory() as name:
-            temp_dir = Path(name)
-            runtime = temp_dir / "runtime"
-            external = temp_dir / "external"
-            runtime.mkdir()
-            external.mkdir()
-
-            admission = {
-                "browserTool": "antigravity_native",
-                "browserCloseReceiptPath": ".cvf/runtime/close.json",
-                "trancheId": "CCMAI-RUNTIME-139",
-                "pageId": "page-123",
-                "campaignNonce": "nonce-456",
-                "browserSessionId": "session-789",
-                "fallbackCloseOperator": "Owner",
-                "grantDeadlineUTC": (datetime.now(timezone.utc) + timedelta(hours=1)).strftime("%Y-%m-%dT%H:%M:%SZ"),
-            }
-
-            # Success path: terminal close evaluation records receipt and no error is recorded
-            outcome_success: dict = {"result": "STOP_REQUESTED", "terminalErrors": []}
-            mock_close_receipt = {"status": "CLOSED_EXACT_OWNED_PAGE", "closeObserved": True, "ownedTabsFinal": 0}
-            with patch.object(module, "verify_browser_terminal_close", return_value=mock_close_receipt):
-                if admission.get("browserTool") == "antigravity_native":
-                    outcome_success["browserTerminalClose"] = module.verify_browser_terminal_close(admission, runtime, external)
-                self.assertEqual(outcome_success["browserTerminalClose"]["status"], "CLOSED_EXACT_OWNED_PAGE")
-                self.assertEqual(len(outcome_success["terminalErrors"]), 0)
-
-            # Error path: terminal close evaluation failure records error and preserves cleanup
-            outcome_error: dict = {"result": "STOP_REQUESTED", "terminalErrors": []}
-            cleanup_called = []
-            def fake_cleanup():
-                cleanup_called.append(True)
-
-            with patch.object(module, "verify_browser_terminal_close", side_effect=module.Hold("BROWSER_CLOSE_AFTER_DEADLINE")):
-                try:
-                    if admission.get("browserTool") == "antigravity_native":
-                        outcome_error["browserTerminalClose"] = module.verify_browser_terminal_close(admission, runtime, external)
-                except Exception as exc:
-                    outcome_error["terminalErrors"].append("BROWSER_TERMINAL_CLOSE_FAILED:" + type(exc).__name__)
-                finally:
-                    fake_cleanup()
+        class Child:
+            pid = 1001
+            def poll(self): return None
+            def wait(self, timeout=None): return 0
 
-                self.assertIn("BROWSER_TERMINAL_CLOSE_FAILED:Hold", outcome_error["terminalErrors"])
-                self.assertTrue(cleanup_called and cleanup_called[0], "Cleanup must execute even when terminal close validation fails")
+        for scenario in ("success", "ordinary_error", "missing_close", "failed_close"):
+            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as name:
+                root = Path(name)
+                admission, reservation, runtime, stamp, identity = self._native_fixture(root)
+                external = root / "external"
+                admission_file = root / "admission.json"
+                admission_file.write_text("{}", encoding="utf-8")
+                key_file = root / "key.txt"
+                key_file.write_text("synthetic-key-value", encoding="utf-8")
+                template_file = root / "vite_template"
+                template_file.write_text('const nonce = "ok"\nstrictPort: true\n', encoding="utf-8")
+                admission.update({
+                    "externalEvidenceDir": str(external), "sourceCommit": "a" * 40,
+                    "keyPath": str(key_file), "vitePort": 4001, "database": "synthetic",
+                    "goTestBinaryPath": "fake-go-binary", "nodeBinary": str(root / "fake-node"),
+                    "endpoint": "https://example.invalid/synthetic",
+                })
+                original_children = {}
+                stopped = []
+                fixture = []
+                def fake_fixture(a, r, e, owner, **kwargs):
+                    owner.update({"containerId": "owned-id", "port": 3308, "fixturePassword": "fake"})
+                def fake_launch(a, r, kind, argv, env, registry):
+                    child = Child()
+                    registry[kind] = child
+                    original_children[kind] = child
+                    if kind == "go":
+                        (external / "go_terminal.json").write_text("{}", encoding="utf-8")
+                    return child
+                def fake_sleep(_):
+                    (runtime / "stop_requested").touch(exist_ok=True)
+                class Reply:
+                    def __enter__(self): return self
+                    def __exit__(self, *args): return False
+                    def read(self): return b"ok"
+                class Opener:
+                    def open(self, *args, **kwargs): return Reply()
+                def fake_stop_child(r, kind, child):
+                    stopped.append((kind, child))
+                    return {"exitCode": 0, "waitObserved": True}
+                def fake_stop_fixture(a, r, owner):
+                    self.assertIs(a, admission)
+                    self.assertEqual(r, runtime)
+                    self.assertEqual(owner["containerId"], "owned-id")
+                    fixture.append(True)
+                    if scenario != "missing_close":
+                        self._native_terminal(root, admission, reservation, runtime, stamp, identity)
+                        if scenario == "failed_close":
+                            (runtime / "close.json").write_text('{"status":"CLOSED_EXACT_OWNED_PAGE"}', encoding="utf-8")
+                    return {"stopped": True}
+                original_read = Path.read_text
+                original_project_path = module.project_path
+                def fake_project_path(rel):
+                    if rel in ("fake-go-binary", "scripts/r139_vite_config.mjs.template",
+                               "frontend/node_modules/vite/bin/vite.js"):
+                        return template_file
+                    return original_project_path(rel)
+                def read_text(path, *args, **kwargs):
+                    if path == key_file: return "synthetic-key-value"
+                    return original_read(path, *args, **kwargs)
+                with ExitStack() as stack:
+                    stack.enter_context(patch.object(module, "ROOT", root))
+                    stack.enter_context(patch.object(module, "fresh_runtime_admission", return_value=(admission, admission_file)))
+                    stack.enter_context(patch.object(module, "docker", return_value=json.dumps([{"Id": module.IMAGE}])))
+                    stack.enter_context(patch.object(module, "inventory", return_value={"containers": [], "networks": [], "volumes": []}))
+                    stack.enter_context(patch.object(module, "canonicalize_protected_inventory", return_value={}))
+                    stack.enter_context(patch.object(module, "start_fixture", side_effect=fake_fixture))
+                    stack.enter_context(patch.object(module, "launch", side_effect=fake_launch))
+                    stack.enter_context(patch.object(module, "wait_go_ready", side_effect=(module.Hold("ORDINARY_ERROR") if scenario == "ordinary_error" else "http://127.0.0.1:4000")))
+                    stack.enter_context(patch.object(module, "validate_native_evidence", return_value={}))
+                    stack.enter_context(patch.object(module, "project_path", side_effect=fake_project_path))
+                    stack.enter_context(patch.object(module, "minimal_environment", return_value={}))
+                    stack.enter_context(patch.object(module.urllib.request, "build_opener", return_value=Opener()))
+                    stack.enter_context(patch.object(module.time, "sleep", side_effect=fake_sleep))
+                    stack.enter_context(patch.object(module, "stop_child", side_effect=fake_stop_child))
+                    stack.enter_context(patch.object(module, "listener_absent", return_value={"absent": True}))
+                    stack.enter_context(patch.object(module, "validate_go_terminal", return_value={}))
+                    stack.enter_context(patch.object(module, "validate_vite_ledger", return_value={}))
+                    stack.enter_context(patch.object(module, "recover_partial_owner", return_value={}))
+                    stack.enter_context(patch.object(module, "stop_fixture", side_effect=fake_stop_fixture))
+                    close_check = stack.enter_context(patch.object(module, "verify_browser_terminal_close",
+                                                                    wraps=module.verify_browser_terminal_close))
+                    stack.enter_context(patch.object(Path, "read_text", read_text))
+                    if scenario == "success":
+                        module.run("admission.json")
+                    else:
+                        with self.assertRaisesRegex(module.Hold, "R139_TERMINAL_HOLD"):
+                            module.run("admission.json")
+                terminal = json.loads((runtime / "terminal.json").read_text(encoding="utf-8"))
+                close_check.assert_called_once_with(admission, runtime, external)
+                self.assertEqual([kind for kind, _ in stopped], ["vite", "go"])
+                self.assertIs(stopped[-1][1], original_children["go"])
+                if scenario != "ordinary_error":
+                    self.assertIs(stopped[0][1], original_children["vite"])
+                self.assertTrue(fixture, "production owned fixture cleanup must run")
+                if scenario == "success":
+                    self.assertEqual(terminal["result"], "STOP_REQUESTED")
+                    self.assertEqual(terminal["browserTerminalClose"]["receipt"]["status"], "CLOSED_EXACT_OWNED_PAGE")
+                else:
+                    self.assertEqual(terminal["result"], "HOLD")
+                if scenario in ("missing_close", "failed_close"):
+                    self.assertTrue(any("BROWSER_TERMINAL_CLOSE_FAILED" in item for item in terminal["terminalErrors"]))
 
 
 if __name__ == "__main__":

~~~~
