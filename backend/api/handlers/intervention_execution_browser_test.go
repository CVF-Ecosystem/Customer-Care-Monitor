package handlers_test

// R139 is opt-in. Ordinary `go test` compiles this file and performs no DB,
// browser, credential, or provider work. Runtime admission is checked before
// the first fixture connection; the source-only campaign cannot set R139_RUN.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type r139AuthorityRef struct {
	Path string `json:"path"`
	SHA256 string `json:"sha256"`
}

type r139Admission struct {
	NativeContractID          string            `json:"nativeContractId"`
	CloseMode                 string            `json:"closeMode"`
	Operator                  string            `json:"operator"`
	BrowserReadinessRoot      string            `json:"browserReadinessRoot"`
	ActionEvidencePath        string            `json:"actionEvidencePath"`
	FinalInventoryPath        string            `json:"finalInventoryPath"`
	RawNativeInventoryPath    string            `json:"rawNativeInventoryPath"`
	RawCuaInventoryPath       string            `json:"rawCuaInventoryPath"`
	nativeProjectRoot         string
	TrancheID                 string            `json:"trancheId"`
	Kind                      string            `json:"kind"`
	SourceCommit              string            `json:"sourceCommit"`
	FixtureName               string            `json:"fixtureName"`
	BrowserTool               string            `json:"browserTool"`
	OwnerCuaAmendmentRef      r139AuthorityRef  `json:"ownerCuaAmendmentRef"`
	InheritedOwnerEnvelopeRef r139AuthorityRef  `json:"inheritedOwnerEnvelopeRef"`
	ObservationCapsRef        r139AuthorityRef  `json:"observationCapsRef"`
	BrowserReceiptSHA256      string            `json:"browserReceiptSha256"`
	BrowserReceiptPath        string            `json:"browserReceiptPath"`
	BrowserCloseReceiptPath   string            `json:"browserCloseReceiptPath"`
	BrowserCloseReceiptSHA256 string            `json:"browserCloseReceiptSha256"`
	CampaignNonce             string            `json:"campaignNonce"`
	BrowserSessionID          string            `json:"browserSessionId"`
	OwnedScopeID              string            `json:"ownedScopeId"`
	ScopeOrigin               string            `json:"scopeOrigin"`
	OwnedPageIDs              []string          `json:"ownedPageIds"`
	OwnedTabIDs               []string          `json:"ownedTabIds"`
	BrowserID                 string            `json:"browserId"`
	BrowserKind               string            `json:"browserKind"`
	TabID                     string            `json:"tabId"`
	ProducerAgentTaskID       string            `json:"producerAgentTaskId"`
	ObserverAgentTaskID       string            `json:"observerAgentTaskId"`
	ProofContract             string            `json:"proofContract"`
	BrowserDoneSignalPath     string            `json:"browserDoneSignalPath"`
	PageID                    string            `json:"pageId"`
	FallbackCloseOperator     string            `json:"fallbackCloseOperator"`
	GoTestBinarySHA256        string            `json:"goTestBinarySha256"`
	SourceSHA256              map[string]string `json:"sourceSha256"`
	RuntimeRoot               string            `json:"runtimeRoot"`
	ExternalEvidenceDir       string            `json:"externalEvidenceDir"`
	BrowserSupported          bool              `json:"browserSupported"`
	OwnerApproved             bool              `json:"ownerApproved"`
	NativeExecutePOST         int               `json:"nativeExecutePOST"`
	BrowserExecutePOST        int               `json:"browserExecutePOST"`
	GrantCap                  int               `json:"grantCap"`
	ProviderCap               int               `json:"providerCap"`
	GrantDeadlineUTC          string            `json:"grantDeadlineUTC"`
	Provider                  string            `json:"provider"`
	Model                     string            `json:"model"`
	Endpoint                  string            `json:"endpoint"`
	CompletionTokens          int               `json:"completionTokens"`
	MaxRequestBytes           int64             `json:"maxRequestBytes"`
	Thinking                  bool              `json:"thinking"`
	SyntheticNamespace        string            `json:"syntheticNamespace"`
	IssuerRole                string            `json:"issuerRole"`
	ActorRole                 string            `json:"actorRole"`
	ConfigRef                 string            `json:"configRef"`
	PromptVersion             string            `json:"promptVersion"`
	PromptSHA256              string            `json:"promptSha256"`
	BackendTree               string            `json:"backendTree"`
	FrontendTree              string            `json:"frontendTree"`
}

func r139VerifyBrowserReservation(a r139Admission, browserRaw []byte) error {
	if a.BrowserTool == "antigravity_page_target" { return r139PageTargetReservation(a, browserRaw) }
	if a.BrowserTool == "supported_CUA_page_target" { return r139CuaReservation(a, browserRaw, false) }
	return r139StrictBrowserReservation(a, browserRaw)
}

func r139VerifyBrowserTerminalClose(a r139Admission, closeRaw []byte) error {
	if a.BrowserTool == "antigravity_page_target" { return r139PageTargetTerminal(a, closeRaw) }
	if a.BrowserTool == "supported_CUA_page_target" { return r139CuaTerminal(a, closeRaw) }
	return r139StrictBrowserTerminalClose(a, closeRaw)
}

func r139NativeRoot(a r139Admission, field string) (string, string, error) {
	project := a.nativeProjectRoot
	if project == "" {
		_, file, _, ok := runtime.Caller(0)
		if !ok { return "", "", errors.New("source root unavailable") }
		project = filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	}
	project, err := filepath.Abs(project)
	if err != nil { return "", "", err }
	var selected string
	switch field {
	case "browserReadinessRoot": selected = a.BrowserReadinessRoot
	case "runtimeRoot": selected = a.RuntimeRoot
	default: return "", "", errors.New("invalid root field")
	}
	if selected == "" || strings.Contains(selected, "\\") && field == "browserReadinessRoot" { return "", "", errors.New("native root missing or invalid") }
	var root string
	if field == "runtimeRoot" {
		if !filepath.IsAbs(selected) { return "", "", errors.New("runtimeRoot must be absolute") }
		root = filepath.Clean(selected)
	} else {
		if filepath.IsAbs(selected) || !strings.HasPrefix(selected, ".cvf/runtime/") { return "", "", errors.New("readiness root must be ignored project-relative") }
		for _, part := range strings.Split(selected, "/") { if part == ".." || part == "" { return "", "", errors.New("root traversal") } }
		root = filepath.Join(project, filepath.FromSlash(selected))
	}
	ignored := filepath.Join(project, ".cvf", "runtime")
	if rel, err := filepath.Rel(ignored, root); err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) { return "", "", errors.New("native root outside ignored area") }
	return project, root, nil
}

func r139NativePath(a r139Admission, field, selected string, root string, existing bool) (string, error) {
	if selected == "" || filepath.IsAbs(selected) || strings.Contains(selected, "\\") { return "", fmt.Errorf("%s invalid path", field) }
	for _, part := range strings.Split(selected, "/") { if part == ".." || part == "" { return "", fmt.Errorf("%s traversal", field) } }
	project, _, err := r139NativeRoot(a, "runtimeRoot")
	if err != nil { return "", err }
	path := filepath.Join(project, filepath.FromSlash(selected))
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) { return "", fmt.Errorf("%s outside selected root", field) }
	// Resolve every existing ancestor as well as an existing final file. A junction or symlink alias is rejected.
	ancestor := path
	for {
		if _, statErr := os.Lstat(ancestor); statErr == nil {
			resolved, resolveErr := filepath.EvalSymlinks(ancestor)
			if resolveErr != nil || !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(ancestor)) { return "", fmt.Errorf("%s reparse alias", field) }
		} else if !os.IsNotExist(statErr) { return "", statErr }
		if strings.EqualFold(ancestor, project) { break }
		parent := filepath.Dir(ancestor)
		if parent == ancestor { return "", errors.New("project ancestor unavailable") }
		ancestor = parent
	}
	if existing {
		info, statErr := os.Stat(path)
		if statErr != nil || !info.Mode().IsRegular() { return "", fmt.Errorf("%s missing file", field) }
	} else if _, statErr := os.Lstat(path); statErr == nil || !os.IsNotExist(statErr) { return "", fmt.Errorf("%s must be absent", field) }
	return path, nil
}

func r139NativeObject(raw []byte) (map[string]any, error) {
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil || value == nil { return nil, errors.New("native JSON object invalid") }
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF { return nil, errors.New("native JSON trailing content") }
	return value, nil
}

func r139NativeZeroInteger(value any) bool {
	number, ok := value.(json.Number)
	if !ok { return false }
	parsed, err := number.Int64()
	return err == nil && parsed == 0
}

func r139NativeText(value any) (string, error) {
	text, ok := value.(string)
	if !ok || len(text) == 0 || len(text) > 128 || strings.TrimSpace(text) != text { return "", errors.New("native text invalid") }
	for _, ch := range text { if ch < 32 || ch >= 127 { return "", errors.New("native text invalid") } }
	return text, nil
}

func r139NativeIdentity(a r139Admission, record map[string]any) error {
	if a.CloseMode != "api" && a.CloseMode != "operator" && a.CloseMode != "operator_fallback" { return errors.New("selected closeMode invalid") }
	op := a.Operator
	if a.CloseMode == "operator_fallback" && a.FallbackCloseOperator != op { return errors.New("fallback close operator mismatch") }
	expected := map[string]string{"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID, "campaignNonce": a.CampaignNonce, "pageId": a.PageID, "browserSessionId": a.BrowserSessionID, "operator": op, "closeMode": a.CloseMode}
	for field, want := range expected {
		if _, err := r139NativeText(want); err != nil { return fmt.Errorf("admission %s: %w", field, err) }
		got, err := r139NativeText(record[field])
		if err != nil || got != want { return fmt.Errorf("native %s mismatch", field) }
	}
	return nil
}

func r139NativeTime(value any) (time.Time, error) {
	text, ok := value.(string)
	if !ok || !regexp.MustCompile("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$").MatchString(text) { return time.Time{}, errors.New("native UTC time invalid") }
	at, err := time.Parse("2006-01-02T15:04:05Z", text)
	if err != nil || at.After(time.Now().UTC()) { return time.Time{}, errors.New("native UTC time future or invalid") }
	return at, nil
}

func r139NativeFile(a r139Admission, field string, rel any, sha any, root string) (map[string]any, error) {
	selected, err := r139NativeText(rel)
	if err != nil { return nil, err }
	digestText, ok := sha.(string)
	if !ok || !regexp.MustCompile("^[0-9a-f]{64}$").MatchString(digestText) { return nil, fmt.Errorf("%s SHA invalid", field) }
	path, err := r139NativePath(a, field, selected, root, true)
	if err != nil { return nil, err }
	raw, err := os.ReadFile(path)
	if err != nil { return nil, err }
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != digestText { return nil, fmt.Errorf("%s SHA mismatch", field) }
	return r139NativeObject(raw)
}

func r139StrictBrowserReservation(a r139Admission, browserRaw []byte) error {
	proof, err := r139NativeObject(browserRaw)
	if err != nil { return err }
	if !a.BrowserSupported { return errors.New("browser unsupported") }
	if a.BrowserTool == "supported_CUA_IAB" {
		if proof["status"] != "PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE" || proof["browserId"] != "iab" || proof["controlObserved"] != true || proof["captureObserved"] != true || proof["closeObserved"] != true || proof["protocol"] != "http" && proof["protocol"] != "https" || !r139NativeZeroInteger(proof["ownedTabsFinal"]) { return errors.New("IAB proof invalid") }
		return nil
	}
	if a.BrowserTool != "antigravity_native" { return errors.New("unsupported browser tool") }
	if proof["status"] != "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE" || proof["browserId"] != "antigravity_native" || proof["exclusiveReservation"] != true || proof["controlObserved"] != true || proof["captureObserved"] != true || proof["protocol"] != "http" && proof["protocol"] != "https" { return errors.New("native reservation incomplete") }
	if observed, exists := proof["closeObserved"]; exists && observed != false { return errors.New("early closeObserved forbidden") }
	if err := r139NativeIdentity(a, proof); err != nil { return err }
	project, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
	sealed, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, project)
	if err != nil || !reflect.DeepEqual(sealed, proof) { return errors.New("reservation file or digest mismatch") }
	if strings.HasPrefix(readiness+string(os.PathSeparator), product+string(os.PathSeparator)) || strings.HasPrefix(product+string(os.PathSeparator), readiness+string(os.PathSeparator)) { return errors.New("native roots overlap") }
	if info, err := os.Stat(readiness); err != nil || !info.IsDir() { return errors.New("readiness root missing") }
	if _, err := os.Stat(product); err == nil {
		if os.Getenv("R139_RUN") != "1" || os.Getenv("R139_RUNTIME_ROOT") != product { return errors.New("product runtime root preexisting outside admitted run") }
	} else if !os.IsNotExist(err) { return err }
	reserved, err := r139NativeTime(proof["reservedAtUTC"]); if err != nil { return err }
	previousEvidence := time.Time{}
	for _, event := range []string{"creation", "control", "capture"} {
		data, err := r139NativeFile(a, event, proof[event+"EvidencePath"], proof[event+"EvidenceSha256"], readiness)
		if err != nil { return err }
		if err := r139NativeIdentity(a, data); err != nil || data["event"] != event { return fmt.Errorf("%s original event invalid", event) }
		at, err := r139NativeTime(data["atUTC"])
		if err != nil || at.After(reserved) || !previousEvidence.IsZero() && at.Before(previousEvidence) { return fmt.Errorf("%s chronology invalid", event) }
		previousEvidence = at
	}
	for field, selected := range map[string]string{"browserCloseReceiptPath": a.BrowserCloseReceiptPath, "actionEvidencePath": a.ActionEvidencePath, "finalInventoryPath": a.FinalInventoryPath, "rawNativeInventoryPath": a.RawNativeInventoryPath} {
		if _, err := r139NativePath(a, field, selected, product, false); err != nil { return err }
	}
	return nil
}

func r139StrictBrowserTerminalClose(a r139Admission, closeRaw []byte) error {
	receipt, err := r139NativeObject(closeRaw); if err != nil { return err }
	if receipt["status"] != "CLOSED_EXACT_OWNED_PAGE" || receipt["browserId"] != "antigravity_native" || receipt["closeObserved"] != true || !r139NativeZeroInteger(receipt["ownedTabsFinal"]) { return errors.New("terminal close incomplete") }
	if err := r139NativeIdentity(a, receipt); err != nil { return err }
	project, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
	closePath, err := r139NativePath(a, "browserCloseReceipt", a.BrowserCloseReceiptPath, product, true)
	if err != nil { return err }
	storedClose, err := os.ReadFile(closePath)
	if err != nil || !bytes.Equal(storedClose, closeRaw) { return errors.New("terminal close file differs from validated bytes") }
	reservation, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, project)
	if err != nil { return err }
	if err := r139NativeIdentity(a, reservation); err != nil { return err }
	reservedEvidence := time.Time{}
	for _, event := range []string{"creation", "control", "capture"} {
		data, err := r139NativeFile(a, event, reservation[event+"EvidencePath"], reservation[event+"EvidenceSha256"], readiness)
		if err != nil { return err }
		if err := r139NativeIdentity(a, data); err != nil || data["event"] != event { return errors.New("original readiness evidence invalid") }
		at, err := r139NativeTime(data["atUTC"])
		if err != nil || !reservedEvidence.IsZero() && at.Before(reservedEvidence) { return errors.New("original readiness chronology invalid") }
		reservedEvidence = at
	}
	reserved, err := r139NativeTime(reservation["reservedAtUTC"]); if err != nil { return err }
	receiptReserved, err := r139NativeTime(receipt["reservedAtUTC"]); if err != nil { return err }
	if !receiptReserved.Equal(reserved) { return errors.New("reservation time mismatch") }
	if reservedEvidence.After(reserved) { return errors.New("readiness after reservation") }
	actionAt, err := r139NativeTime(receipt["actionAtUTC"]); if err != nil { return err }
	closed, err := r139NativeTime(receipt["closedAtUTC"]); if err != nil { return err }
	deadline, err := time.Parse("2006-01-02T15:04:05Z", a.GrantDeadlineUTC); if err != nil { return err }
	if reserved.After(actionAt) || actionAt.After(closed) || closed.After(deadline) { return errors.New("native chronology invalid") }
	if receipt["actionEvidencePath"] != a.ActionEvidencePath || receipt["finalInventoryPath"] != a.FinalInventoryPath { return errors.New("future evidence path mismatch") }
	action, err := r139NativeFile(a, "action", receipt["actionEvidencePath"], receipt["actionEvidenceSha256"], product)
	if err != nil { return err }
	if err := r139NativeIdentity(a, action); err != nil || action["event"] != "action_sequence" || action["atUTC"] != receipt["actionAtUTC"] { return errors.New("action evidence mismatch") }
	rows, ok := action["actions"].([]any)
	if !ok || len(rows) < 2 { return errors.New("action sequence missing") }
	previous := reserved
	for index, item := range rows {
		row, ok := item.(map[string]any)
		if !ok || row["pageId"] != a.PageID { return errors.New("action page invalid") }
		kind := row["action"]
		if kind != "control" && kind != "capture" && kind != "close" { return errors.New("action kind invalid") }
		at, err := r139NativeTime(row["atUTC"])
		if err != nil || at.Before(previous) || at.After(closed) { return errors.New("action sequence time invalid") }
		if kind == "close" && (index != len(rows)-1 || !at.Equal(closed)) || kind != "close" && index == len(rows)-1 { return errors.New("post-close action or missing close") }
		previous = at
	}
	inv, err := r139NativeFile(a, "finalInventory", receipt["finalInventoryPath"], receipt["finalInventorySha256"], product)
	if err != nil { return err }
	if err := r139NativeIdentity(a, inv); err != nil { return err }
	observed, err := r139NativeTime(inv["inventoryAtUTC"])
	if err != nil || observed.Before(closed) || observed.After(deadline) { return errors.New("inventory time invalid") }
	pages, ok := inv["pageIds"].([]any)
	if !ok || !r139NativeZeroInteger(inv["ownedCount"]) { return errors.New("inventory list or count invalid") }
	seen := map[string]bool{}
	for _, page := range pages {
		name, err := r139NativeText(page)
		if err != nil || seen[name] || name == a.PageID { return errors.New("inventory page invalid") }
		seen[name] = true
	}
	if inv["rawNativeInventoryPath"] != a.RawNativeInventoryPath { return errors.New("raw inventory path mismatch") }
	rawInventory, err := r139NativeFile(a, "rawNativeInventory", inv["rawNativeInventoryPath"], inv["rawNativeInventorySha256"], product)
	if err != nil { return err }
	if err := r139NativeIdentity(a, rawInventory); err != nil { return err }
	if rawInventory["event"] != "native_inventory" || rawInventory["atUTC"] != inv["inventoryAtUTC"] || !reflect.DeepEqual(rawInventory["pageIds"], inv["pageIds"]) || rawInventory["ownedCount"] != inv["ownedCount"] { return errors.New("raw native inventory mismatch") }
	return nil
}

func r139PageIdentity(a r139Admission, m map[string]any) error {
	if a.ProofContract != "native_page_target_v1" || a.ScopeOrigin != "caller_owned_explicit_page_target" ||
		a.BrowserSessionID != "" || a.CloseMode != "operator_fallback" || a.FallbackCloseOperator != a.Operator ||
		len(a.OwnedPageIDs) != 1 || a.OwnedPageIDs[0] != a.PageID { return errors.New("page-target admission identity invalid") }
	want := map[string]string{"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID,
		"campaignNonce": a.CampaignNonce, "pageId": a.PageID, "ownedScopeId": a.OwnedScopeID,
		"operator": a.Operator, "closeMode": a.CloseMode}
	for key, value := range want {
		if _, err := r139NativeText(value); err != nil || m[key] != value { return fmt.Errorf("page-target %s mismatch", key) }
	}
	if _, present := m["browserSessionId"]; present { return errors.New("native session alias forbidden") }
	if m["proofContract"] != a.ProofContract || m["scopeOrigin"] != a.ScopeOrigin { return errors.New("page-target contract mismatch") }
	ids, ok := m["ownedPageIds"].([]any)
	if !ok || len(ids) != 1 || ids[0] != a.PageID { return errors.New("owned target set invalid") }
	return nil
}

func r139PageOrigin(m map[string]any) (time.Time, error) {
	for _, key := range []string{"nativeTaskRunRef", "nativeToolCallRef", "rawOriginalToolReference"} {
		value, ok := m[key].(string)
		if !ok || len(value) < 1 || len(value) > 512 || strings.TrimSpace(value) != value { return time.Time{}, fmt.Errorf("original %s invalid", key) }
		for _, ch := range value { if ch < 32 || ch >= 127 { return time.Time{}, fmt.Errorf("original %s invalid", key) } }
	}
	value, ok := m["rawOriginalToolSha256"].(string)
	if !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(value) { return time.Time{}, errors.New("original tool hash invalid") }
	at, err := r139NativeTime(m["atUTC"])
	if err != nil { return time.Time{}, err }
	switch m["timeBasis"] {
	case "original_native_tool_return":
	case "normalized_from_original_native_tool_return":
		raw, ok := m["rawObservedAtUTC"].(string)
		if !ok { return time.Time{}, errors.New("original timestamp absent") }
		original, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil || original.Location() == nil || original.UTC().Truncate(time.Second) != at || original.Format("-07:00") != "+00:00" { return time.Time{}, errors.New("normalized timestamp unsupported") }
	default: return time.Time{}, errors.New("time basis invalid")
	}
	return at, nil
}

func r139PageEvent(a r139Admission, root string, prefix map[string]any, event string) (map[string]any, time.Time, error) {
	pathField, hashField := event+"EvidencePath", event+"EvidenceSha256"
	if event == "baselineInventory" || event == "creationLedger" { pathField, hashField = event+"Path", event+"Sha256" }
	m, err := r139NativeFile(a, event, prefix[pathField], prefix[hashField], root)
	if err != nil { return nil, time.Time{}, err }
	if err := r139PageIdentity(a, m); err != nil { return nil, time.Time{}, err }
	want := event
	if event == "baselineInventory" { want = "baseline_inventory" }
	if event == "creationLedger" { want = "creation_ledger" }
	if m["event"] != want { return nil, time.Time{}, errors.New("page-target event mismatch") }
	at, err := r139PageOrigin(m)
	return m, at, err
}

func r139PageIDs(value any) (map[string]bool, error) {
	items, ok := value.([]any)
	if !ok { return nil, errors.New("page ID list missing") }
	seen := map[string]bool{}
	for _, item := range items {
		text, err := r139NativeText(item)
		if err != nil || seen[text] { return nil, errors.New("page ID invalid or duplicate") }
		seen[text] = true
	}
	return seen, nil
}

func r139PageTargetReservation(a r139Admission, browserRaw []byte) error {
	return r139PageTargetReservationWithRuntime(a, browserRaw, false)
}

func r139PageTargetReservationWithRuntime(a r139Admission, browserRaw []byte, allowRuntime bool) error {
	proof, err := r139NativeObject(browserRaw); if err != nil { return err }
	if !a.BrowserSupported || proof["status"] != "RESERVED_NATIVE_PAGE_TARGET_CONTROL_CAPTURE_HANDOFF" ||
		proof["browserId"] != "antigravity_native" || proof["protocol"] != "https" || proof["closeObserved"] != false {
		return errors.New("page-target prefix incomplete")
	}
	if err := r139PageIdentity(a, proof); err != nil { return err }
	_, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
	if strings.EqualFold(readiness, product) || strings.HasPrefix(strings.ToLower(readiness), strings.ToLower(product)+string(os.PathSeparator)) || strings.HasPrefix(strings.ToLower(product), strings.ToLower(readiness)+string(os.PathSeparator)) { return errors.New("page-target roots overlap") }
	stored, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, readiness)
	if err != nil || !reflect.DeepEqual(stored, proof) { return errors.New("page-target receipt bytes/hash differ") }
	if _, err := os.Stat(readiness); err != nil { return errors.New("readiness root absent") }
	if info, statErr := os.Stat(product); statErr == nil && !allowRuntime {
		if !info.IsDir() || os.Getenv("R139_RUN") != "1" || os.Getenv("R139_RUNTIME_ROOT") != product ||
			!regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(os.Getenv("R139_RUNTIME_ADMISSION_SHA256")) ||
			a.SourceCommit == "" || a.SourceCommit != os.Getenv("R139_SOURCE_COMMIT") {
			return errors.New("product root preexisting outside admitted supervisor")
		}
		intentRaw, readErr := os.ReadFile(filepath.Join(product, "run_intent.json"))
		if readErr != nil { return errors.New("supervisor run intent missing") }
		var intent struct {
			AdmissionSHA256 string `json:"admissionSha256"`
			SourceCommit string `json:"sourceCommit"`
			StartedAt float64 `json:"startedAt"`
			DeadlineSeconds int `json:"deadlineSeconds"`
		}
		if json.Unmarshal(intentRaw, &intent) != nil || intent.AdmissionSHA256 != os.Getenv("R139_RUNTIME_ADMISSION_SHA256") ||
			intent.SourceCommit != a.SourceCommit || intent.DeadlineSeconds != 1200 ||
			intent.StartedAt < float64(time.Now().Unix()-1200) || intent.StartedAt > float64(time.Now().Unix()+1) {
			return errors.New("supervisor run intent mismatch")
		}
	} else if statErr != nil && !os.IsNotExist(statErr) { return statErr }
	sealed, err := r139NativeTime(proof["reservedAtUTC"]); if err != nil { return err }
	baseline, baseTime, err := r139PageEvent(a, readiness, proof, "baselineInventory"); if err != nil { return err }
	baseIDs, err := r139PageIDs(baseline["pageIds"]); if err != nil || baseIDs[a.PageID] || baseline["inventoryScope"] != "metadata_only" || baseTime.After(sealed) { return errors.New("target preexisting or baseline scope invalid") }
	previous := baseTime
	firstRun := ""
	creationAt := baseTime
	created := "https://example.com/#r139_nonce=" + a.CampaignNonce + "&phase=created"
	controlled := "https://example.com/#r139_nonce=" + a.CampaignNonce + "&phase=controlled"
	for _, event := range []string{"creation", "control", "capture", "handoff"} {
		data, at, err := r139PageEvent(a, readiness, proof, event)
		if err != nil || at.Before(previous) || at.After(sealed) { return fmt.Errorf("%s page-target chronology/evidence invalid", event) }
		previous = at
		switch event {
		case "creation":
			firstRun, _ = data["nativeTaskRunRef"].(string)
			creationAt = at
			if data["pageIdToReplace"] != "new_page" || data["observedPageId"] != a.PageID || data["observedUrl"] != created { return errors.New("creation target invalid") }
		case "control":
			if data["nativeTaskRunRef"] != firstRun || data["pageIdToReplace"] != a.PageID || data["observedUrl"] != controlled { return errors.New("same-page control invalid") }
		case "capture":
			hash, ok := data["captureSha256"].(string)
			if data["nativeTaskRunRef"] != firstRun || data["observedPageId"] != a.PageID || data["observedUrl"] != controlled || !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(hash) { return errors.New("capture invalid") }
		case "handoff":
			if data["nativeTaskRunRef"] == firstRun || data["observedPageId"] != a.PageID || data["observedUrl"] != controlled || data["observedNonce"] != a.CampaignNonce { return errors.New("cross-task handoff unproved") }
		}
	}
	ledger, at, err := r139PageEvent(a, readiness, proof, "creationLedger")
	if err != nil || at.Before(creationAt) || at.After(sealed) || ledger["baselineInventorySha256"] != proof["baselineInventorySha256"] || ledger["creationEvidenceSha256"] != proof["creationEvidenceSha256"] || ledger["nativeTaskRunRef"] != firstRun { return errors.New("creation ledger invalid") }
	newIDs, ok := ledger["newPageIds"].([]any)
	if !ok || len(newIDs) != 1 || newIDs[0] != a.PageID { return errors.New("multiple/new target invalid") }
	for field, value := range map[string]string{"browserCloseReceiptPath": a.BrowserCloseReceiptPath,
		"actionEvidencePath": a.ActionEvidencePath, "finalInventoryPath": a.FinalInventoryPath,
		"rawNativeInventoryPath": a.RawNativeInventoryPath, "browserDoneSignalPath": a.BrowserDoneSignalPath} {
		if _, err := r139NativePath(a, field, value, product, allowRuntime); err != nil { return err }
	}
	return nil
}

func r139PageTargetTerminal(a r139Admission, closeRaw []byte) error {
	receipt, err := r139NativeObject(closeRaw); if err != nil { return err }
	if err := r139PageIdentity(a, receipt); err != nil { return err }
	if receipt["status"] != "CLOSED_EXACT_OWNED_PAGE_TARGET" || receipt["browserId"] != "antigravity_native" || receipt["closeObserved"] != true || !r139NativeZeroInteger(receipt["ownedTabsFinal"]) || receipt["closureTimeBasis"] != "native_final_metadata_observation" {
		return errors.New("page-target close status invalid")
	}
	if _, exists := receipt["actualCloseActionAtUTC"]; exists { return errors.New("invented physical close time") }
	if _, exists := receipt["closedAtUTC"]; exists { return errors.New("legacy physical close time") }
	_, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
	closePath, err := r139NativePath(a, "browserCloseReceipt", a.BrowserCloseReceiptPath, product, true)
	if err != nil { return err }
	stored, err := os.ReadFile(closePath)
	if err != nil || !bytes.Equal(stored, closeRaw) { return errors.New("close receipt bytes differ") }
	prefix, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, readiness)
	if err != nil || prefix["reservedAtUTC"] != receipt["reservedAtUTC"] { return errors.New("page-target prefix mismatch") }
	prefixRaw, err := json.Marshal(prefix); if err != nil { return err }
	if err := r139PageTargetReservationWithRuntime(a, prefixRaw, true); err != nil { return err }
	reserved, err := r139NativeTime(prefix["reservedAtUTC"]); if err != nil { return err }
	reported, err := r139NativeTime(receipt["operatorReportReceivedAtUTC"]); if err != nil { return err }
	observed, err := r139NativeTime(receipt["closureObservedAtUTC"]); if err != nil { return err }
	deadline, err := time.Parse("2006-01-02T15:04:05Z", a.GrantDeadlineUTC)
	if err != nil || reserved.After(reported) || reported.After(observed) || observed.After(deadline) { return errors.New("page-target close chronology") }
	ownerHash, ok := receipt["ownerReportReferenceSha256"].(string)
	if !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(ownerHash) || receipt["operator"] != "Owner in this conversation" { return errors.New("owner report invalid") }
	if receipt["actionEvidencePath"] != a.ActionEvidencePath || receipt["finalInventoryPath"] != a.FinalInventoryPath { return errors.New("terminal path mismatch") }
	action, err := r139NativeFile(a, "action", receipt["actionEvidencePath"], receipt["actionEvidenceSha256"], product)
	if err != nil { return err }
	if err := r139PageIdentity(a, action); err != nil || action["event"] != "action_sequence" || action["atUTC"] != receipt["actionAtUTC"] { return errors.New("action identity invalid") }
	if _, err := r139PageOrigin(action); err != nil { return err }
	rows, ok := action["actions"].([]any)
	if !ok || len(rows) < 2 { return errors.New("action rows missing") }
	previous := reserved
	for index, value := range rows {
		row, ok := value.(map[string]any)
		if !ok || row["pageId"] != a.PageID { return errors.New("action target invalid") }
		kind := row["action"]
		if kind != "control" && kind != "capture" && kind != "handoff" && kind != "product_read" && kind != "product_execute" && kind != "owner_close_report" { return errors.New("action kind invalid") }
		at, err := r139NativeTime(row["atUTC"])
		if err != nil || at.Before(previous) || at.After(reported) { return errors.New("action time invalid") }
		previous = at
		if kind == "owner_close_report" {
			if index != len(rows)-1 || !at.Equal(reported) || row["reportSource"] != "owner_in_conversation" || row["ownerReportReferenceSha256"] != ownerHash { return errors.New("report not terminal") }
		} else {
			hash, ok := row["rawOriginalToolSha256"].(string)
			if index == len(rows)-1 || !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(hash) { return errors.New("action original hash absent") }
			if _, err := r139NativeText(row["rawOriginalToolReference"]); err != nil { return err }
		}
	}
	if rows[len(rows)-1].(map[string]any)["action"] != "owner_close_report" { return errors.New("post-close action") }
	inv, err := r139NativeFile(a, "finalInventory", receipt["finalInventoryPath"], receipt["finalInventorySha256"], product)
	if err != nil { return err }
	if err := r139PageIdentity(a, inv); err != nil || inv["event"] != "final_inventory" || inv["atUTC"] != receipt["closureObservedAtUTC"] || inv["rawNativeInventoryPath"] != a.RawNativeInventoryPath || inv["inventoryScope"] != "metadata_only" || !r139NativeZeroInteger(inv["ownedCount"]) { return errors.New("final inventory invalid") }
	if _, err := r139PageOrigin(inv); err != nil { return err }
	finalIDs, err := r139PageIDs(inv["pageIds"]); if err != nil || finalIDs[a.PageID] { return errors.New("owned target remains") }
	baseline, _, err := r139PageEvent(a, readiness, prefix, "baselineInventory"); if err != nil { return err }
	baselineIDs, err := r139PageIDs(baseline["pageIds"]); if err != nil || !reflect.DeepEqual(finalIDs, baselineIDs) { return errors.New("protected baseline drift") }
	raw, err := r139NativeFile(a, "rawNativeInventory", inv["rawNativeInventoryPath"], inv["rawNativeInventorySha256"], product)
	if err != nil { return err }
	if err := r139PageIdentity(a, raw); err != nil || raw["event"] != "native_inventory" || raw["atUTC"] != inv["atUTC"] || !reflect.DeepEqual(raw["pageIds"], inv["pageIds"]) || raw["inventoryScope"] != "metadata_only" || !r139NativeZeroInteger(raw["ownedCount"]) || raw["nativeTaskRunRef"] == baseline["nativeTaskRunRef"] { return errors.New("raw native inventory invalid") }
	if _, err := r139PageOrigin(raw); err != nil { return err }
	markerPath, err := r139NativePath(a, "browserDoneSignalPath", a.BrowserDoneSignalPath, product, true)
	if err != nil { return err }
	markerRaw, err := os.ReadFile(markerPath); if err != nil { return err }
	marker, err := r139NativeObject(markerRaw); if err != nil { return err }
	if marker["status"] != "TERMINAL_EVIDENCE_WRITTEN" || marker["proofContract"] != a.ProofContract || marker["campaignNonce"] != a.CampaignNonce || marker["pageId"] != a.PageID || marker["ownedScopeId"] != a.OwnedScopeID { return errors.New("browser done marker identity invalid") }
	markerAt, err := r139NativeTime(marker["atUTC"])
	if err != nil || markerAt.Before(observed) || markerAt.After(deadline) { return errors.New("browser done chronology invalid") }
	marked, ok := marker["evidenceSha256"].(map[string]any)
	if !ok || len(marked) != 4 || marked["browserCloseReceiptPath"] != func() string { sum := sha256.Sum256(closeRaw); return hex.EncodeToString(sum[:]) }() || marked["actionEvidencePath"] != receipt["actionEvidenceSha256"] || marked["finalInventoryPath"] != receipt["finalInventorySha256"] || marked["rawNativeInventoryPath"] != inv["rawNativeInventorySha256"] { return errors.New("browser done evidence hashes invalid") }
	return nil
}

// CUA evidence is caller-recorded from original CUA returns. It is deliberately
// distinct from the native browser-subagent proof contract above.
func r139CuaIdentity(a r139Admission, m map[string]any) error {
	if a.BrowserTool != "supported_CUA_page_target" || a.ProofContract != "cua_page_target_v1" ||
		a.BrowserKind != "iab" || a.ScopeOrigin != "caller_owned_explicit_cua_tab_target" ||
		(a.CloseMode != "native_tool" && (a.CloseMode != "operator_fallback" || a.FallbackCloseOperator != a.Operator)) ||
		a.ProducerAgentTaskID == a.ObserverAgentTaskID || len(a.OwnedTabIDs) != 1 ||
		a.OwnedTabIDs[0] != a.TabID || a.PageID != "" || a.BrowserSessionID != "" || len(a.OwnedPageIDs) != 0 {
		return errors.New("CUA admission identity invalid")
	}
	for _, key := range []string{"pageId", "browserSessionId", "nativeTaskRunRef", "nativeToolCallRef", "ownedPageIds", "rawOriginalToolReference", "rawOriginalToolSha256"} {
		if _, ok := m[key]; ok { return fmt.Errorf("native alias %s forbidden in CUA evidence", key) }
	}
	want := map[string]string{"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID,
		"campaignNonce": a.CampaignNonce, "browserId": a.BrowserID, "tabId": a.TabID,
		"ownedScopeId": a.OwnedScopeID, "producerAgentTaskId": a.ProducerAgentTaskID,
		"observerAgentTaskId": a.ObserverAgentTaskID, "operator": a.Operator, "closeMode": a.CloseMode}
	for key, value := range want {
		if _, err := r139NativeText(value); err != nil || m[key] != value { return fmt.Errorf("CUA %s mismatch", key) }
	}
	if m["proofContract"] != "cua_page_target_v1" || m["browserKind"] != "iab" || m["scopeOrigin"] != a.ScopeOrigin { return errors.New("CUA contract mismatch") }
	ids, ok := m["ownedTabIds"].([]any)
	if !ok || len(ids) != 1 || ids[0] != a.TabID { return errors.New("CUA owned tab set invalid") }
	return nil
}

func r139CuaOrigin(m map[string]any) (time.Time, error) {
	for _, key := range []string{"cuaCallRef", "originalCuaReturnReference"} {
		if err := r139CuaReference(m[key]); err != nil { return time.Time{}, fmt.Errorf("CUA %s invalid", key) }
	}
	sha, ok := m["originalCuaReturnSha256"].(string)
	if !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(sha) ||
		m["provenanceBasis"] != "caller_recorded_original_cua_return" || m["rootOriginalReturnReviewRequired"] != true ||
		m["observedAtUTC"] != m["atUTC"] { return time.Time{}, errors.New("CUA original return provenance invalid") }
	return r139NativeTime(m["atUTC"])
}

func r139CuaReference(value any) error {
	text, ok := value.(string)
	if !ok || len(text) < 1 || len(text) > 512 || strings.TrimSpace(text) != text { return errors.New("CUA reference length/trim invalid") }
	for _, ch := range text { if ch < 32 || ch >= 127 { return errors.New("CUA reference not ASCII") } }
	return nil
}

func TestR139PureCuaReferenceBounds(t *testing.T) {
	if err := r139CuaReference(strings.Repeat("r", 512)); err != nil { t.Fatalf("valid long CUA reference rejected: %v", err) }
	for _, bad := range []any{strings.Repeat("r", 513), " leading", "bad\nref", false} {
		if err := r139CuaReference(bad); err == nil { t.Fatal("malformed CUA reference accepted") }
	}
}

func r139CuaDerived(m map[string]any) (time.Time, error) {
	if m["provenanceBasis"] != "derived_from_original_cua_returns" || m["rootOriginalReturnReviewRequired"] != true || m["observedAtUTC"] != m["atUTC"] { return time.Time{}, errors.New("CUA derived provenance invalid") }
	for _, key := range []string{"cuaCallRef", "originalCuaReturnReference", "originalCuaReturnSha256"} {
		if _, exists := m[key]; exists { return time.Time{}, errors.New("CUA aggregate cannot claim one original return") }
	}
	links, ok := m["sourceEvidenceSha256"].(map[string]any)
	if !ok || len(links) == 0 { return time.Time{}, errors.New("CUA aggregate source links absent") }
	for key, raw := range links {
		if _, err := r139NativeText(key); err != nil { return time.Time{}, err }
		sha, ok := raw.(string)
		if !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(sha) { return time.Time{}, errors.New("CUA aggregate source link hash invalid") }
	}
	return r139NativeTime(m["atUTC"])
}

func r139CuaEvent(a r139Admission, root string, proof map[string]any, field, event string) (map[string]any, time.Time, error) {
	suffix := ""
	if field != "baselineInventory" && field != "creationLedger" { suffix = "Evidence" }
	data, err := r139NativeFile(a, field, proof[field+suffix+"Path"], proof[field+suffix+"Sha256"], root)
	if err != nil { return nil, time.Time{}, err }
	if err := r139CuaIdentity(a, data); err != nil || data["event"] != event { return nil, time.Time{}, errors.New("CUA event identity invalid") }
	var at time.Time
	if event == "creation_ledger" || event == "action_sequence" || event == "handoff" { at, err = r139CuaDerived(data) } else { at, err = r139CuaOrigin(data) }
	return data, at, err
}

func r139CuaReservation(a r139Admission, browserRaw []byte, allowRuntime bool) error {
	proof, err := r139NativeObject(browserRaw); if err != nil { return err }
	if !a.BrowserSupported || proof["status"] != "RESERVED_CUA_TAB_CONTROL_CAPTURE_HANDOFF" ||
		proof["protocol"] != "https" || proof["closeObserved"] != false || proof["rootOriginalReturnReviewed"] != true { return errors.New("CUA prefix incomplete") }
	if err := r139CuaIdentity(a, proof); err != nil { return err }
	_, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
	if strings.EqualFold(readiness, product) || strings.HasPrefix(strings.ToLower(readiness), strings.ToLower(product)+string(os.PathSeparator)) || strings.HasPrefix(strings.ToLower(product), strings.ToLower(readiness)+string(os.PathSeparator)) { return errors.New("CUA roots overlap") }
	stored, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, readiness)
	if err != nil || !reflect.DeepEqual(stored, proof) { return errors.New("CUA prefix bytes/hash mismatch") }
	if _, err := os.Stat(readiness); err != nil { return errors.New("CUA readiness absent") }
	if info, statErr := os.Stat(product); statErr == nil && !allowRuntime {
		if !info.IsDir() || os.Getenv("R139_RUN") != "1" || os.Getenv("R139_RUNTIME_ROOT") != product ||
			!regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(os.Getenv("R139_RUNTIME_ADMISSION_SHA256")) ||
			a.SourceCommit == "" || a.SourceCommit != os.Getenv("R139_SOURCE_COMMIT") { return errors.New("CUA product root preexisting") }
		intentRaw, readErr := os.ReadFile(filepath.Join(product, "run_intent.json"))
		if readErr != nil { return errors.New("CUA supervisor intent missing") }
		var intent struct { AdmissionSHA256 string `json:"admissionSha256"`; SourceCommit string `json:"sourceCommit"`; StartedAt float64 `json:"startedAt"`; DeadlineSeconds int `json:"deadlineSeconds"` }
		if json.Unmarshal(intentRaw, &intent) != nil || intent.AdmissionSHA256 != os.Getenv("R139_RUNTIME_ADMISSION_SHA256") ||
			intent.SourceCommit != a.SourceCommit || intent.DeadlineSeconds != 1200 ||
			intent.StartedAt < float64(time.Now().Unix()-1200) || intent.StartedAt > float64(time.Now().Unix()+1) { return errors.New("CUA supervisor intent mismatch") }
	} else if statErr != nil && !os.IsNotExist(statErr) { return statErr }
	sealed, err := r139NativeTime(proof["reservedAtUTC"]); if err != nil { return err }
	baseline, baseTime, err := r139CuaEvent(a, readiness, proof, "baselineInventory", "baseline_inventory")
	if err != nil || baseline["inventoryScope"] != "metadata_only" || baseTime.After(sealed) { return errors.New("CUA baseline invalid") }
	baseIDs, err := r139PageIDs(baseline["tabIds"]); if err != nil || baseIDs[a.TabID] { return errors.New("CUA tab preexisting") }
	previous := baseTime
	creationAt := baseTime
	created := "https://example.com/#r139_nonce=" + a.CampaignNonce + "&phase=created"
	controlled := "https://example.com/#r139_nonce=" + a.CampaignNonce + "&phase=controlled"
	for _, event := range []string{"creation", "control", "capture", "handoff"} {
		row, at, err := r139CuaEvent(a, readiness, proof, event, event)
		if err != nil || at.Before(previous) || at.After(sealed) { return fmt.Errorf("CUA %s chronology/evidence invalid", event) }
		previous = at
		switch event {
		case "creation":
			creationAt = at
			if _, invented := row["observedUrl"]; invented { return errors.New("CUA creation URL not returned") }
			if row["agentTaskId"] != a.ProducerAgentTaskID || row["createdTabId"] != a.TabID || row["requestedBrowserId"] != a.BrowserID || row["requestedUrl"] != created || row["callCompleted"] != true || row["method"] != "createBrowserTab" { return errors.New("CUA creation invalid") }
		case "control":
			if _, invented := row["observedUrl"]; invented { return errors.New("CUA void goto URL not observed") }
			if row["agentTaskId"] != a.ProducerAgentTaskID || row["targetTabId"] != a.TabID || row["requestedUrl"] != controlled || row["callCompleted"] != true || row["method"] != "goto" { return errors.New("CUA control invalid") }
		case "capture":
			hash, ok := row["captureSha256"].(string)
			if _, invented := row["observedUrl"]; invented { return errors.New("CUA screenshot URL not returned") }
			if row["agentTaskId"] != a.ProducerAgentTaskID || row["targetTabId"] != a.TabID || row["method"] != "getScreenshot" || !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(hash) { return errors.New("CUA capture invalid") }
		case "handoff":
			if row["agentTaskId"] != a.ObserverAgentTaskID || row["method"] != "getTab_then_listTabs" || row["requestedTabId"] != a.TabID || row["observedTabId"] != a.TabID || row["metadataBrowserId"] != a.BrowserID || row["metadataTabId"] != a.TabID || row["observedUrl"] != controlled || row["observedNonce"] != a.CampaignNonce { return errors.New("CUA cross-agent handoff invalid") }
			links := map[string]any{"getTab": row["getTabReturnSha256"], "listTabs": row["metadataReturnSha256"]}
			if !reflect.DeepEqual(row["sourceEvidenceSha256"], links) { return errors.New("CUA handoff source links invalid") }
			for _, key := range []string{"getTabReturnReference", "metadataReturnReference", "getTabCallRef", "metadataCallRef"} {
				if err := r139CuaReference(row[key]); err != nil { return err }
			}
		}
	}
	ledger, at, err := r139CuaEvent(a, readiness, proof, "creationLedger", "creation_ledger")
	if err != nil || at.Before(creationAt) || at.After(sealed) || ledger["baselineInventorySha256"] != proof["baselineInventorySha256"] || ledger["creationEvidenceSha256"] != proof["creationEvidenceSha256"] || ledger["agentTaskId"] != a.ProducerAgentTaskID || !reflect.DeepEqual(ledger["sourceEvidenceSha256"], map[string]any{"baselineInventory": proof["baselineInventorySha256"], "creation": proof["creationEvidenceSha256"]}) { return errors.New("CUA creation ledger invalid") }
	ids, ok := ledger["newTabIds"].([]any)
	if !ok || len(ids) != 1 || ids[0] != a.TabID { return errors.New("CUA new tab set invalid") }
	for field, value := range map[string]string{"browserCloseReceiptPath": a.BrowserCloseReceiptPath,
		"actionEvidencePath": a.ActionEvidencePath, "finalInventoryPath": a.FinalInventoryPath,
		"rawCuaInventoryPath": a.RawCuaInventoryPath, "browserDoneSignalPath": a.BrowserDoneSignalPath} {
		if _, err := r139NativePath(a, field, value, product, allowRuntime); err != nil { return err }
	}
	return nil
}

func r139CuaNativeCloseFields(m map[string]any) error {
	for _, key := range []string{"actualCloseActionAtUTC", "closedAtUTC", "operatorReportReceivedAtUTC", "ownerReportReferenceSha256", "reportSource"} {
		if _, exists := m[key]; exists { return errors.New("CUA tool close cannot claim physical time or owner report") }
	}
	return nil
}

func r139CuaNativeToolTerminal(a r139Admission, closeRaw []byte) error {
	receipt, err := r139NativeObject(closeRaw); if err != nil { return err }
	if err := r139CuaIdentity(a, receipt); err != nil { return err }
	if err := r139CuaNativeCloseFields(receipt); err != nil { return err }
	if receipt["status"] != "CLOSED_EXACT_OWNED_CUA_TAB" || receipt["closeObserved"] != true ||
		!r139NativeZeroInteger(receipt["ownedTabsFinal"]) || receipt["rootOriginalReturnReviewed"] != true ||
		receipt["closureTimeBasis"] != "cua_final_metadata_observation" || receipt["closeActorTaskId"] != a.ProducerAgentTaskID { return errors.New("CUA tool close status invalid") }
	_, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
	closePath, err := r139NativePath(a, "browserCloseReceiptPath", a.BrowserCloseReceiptPath, product, true); if err != nil { return err }
	stored, err := os.ReadFile(closePath); if err != nil || !bytes.Equal(stored, closeRaw) { return errors.New("CUA tool close bytes differ") }
	prefix, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, readiness); if err != nil { return err }
	prefixRaw, err := json.Marshal(prefix); if err != nil { return err }
	if err := r139CuaReservation(a, prefixRaw, true); err != nil { return err }
	if receipt["reservedAtUTC"] != prefix["reservedAtUTC"] { return errors.New("CUA tool prefix mismatch") }
	reserved, err := r139NativeTime(prefix["reservedAtUTC"]); if err != nil { return err }
	returned, err := r139NativeTime(receipt["toolCloseReturnedAtUTC"]); if err != nil { return err }
	observed, err := r139NativeTime(receipt["closureObservedAtUTC"]); if err != nil { return err }
	deadline, err := time.Parse("2006-01-02T15:04:05Z", a.GrantDeadlineUTC); if err != nil { return err }
	request, err := r139NativeFile(a, "toolCloseRequest", receipt["toolCloseRequestPath"], receipt["toolCloseRequestSha256"], product); if err != nil { return err }
	if err := r139CuaIdentity(a, request); err != nil { return err }
	if request["event"] != "tool_close_request" || request["provenanceBasis"] != "caller_recorded_original_cua_request" ||
		request["rootOriginalReturnReviewRequired"] != true || request["atUTC"] != request["observedAtUTC"] { return errors.New("CUA tool request provenance invalid") }
	if err := r139CuaReference(request["originalCuaRequestReference"]); err != nil { return err }
	requestOrigin, ok := request["originalCuaRequestSha256"].(string)
	if !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(requestOrigin) { return errors.New("CUA request origin hash invalid") }
	result, err := r139NativeFile(a, "toolCloseReturn", receipt["toolCloseReturnPath"], receipt["toolCloseReturnSha256"], product); if err != nil { return err }
	if err := r139CuaIdentity(a, result); err != nil { return err }
	if _, err := r139CuaOrigin(result); err != nil { return err }
	for _, m := range []map[string]any{request, result} {
		if err := r139CuaNativeCloseFields(m); err != nil { return err }
		if m["method"] != "Tab.close" || m["targetTabId"] != a.TabID || m["closeActorTaskId"] != a.ProducerAgentTaskID { return errors.New("CUA close target or actor invalid") }
	}
	requested, err := r139NativeTime(request["atUTC"]); if err != nil { return err }
	if requested.Before(reserved) || returned.Before(requested) || observed.Before(returned) || observed.After(deadline) ||
		request["status"] != "REQUESTED" || result["event"] != "tool_close_return" || result["status"] != "SUCCEEDED" ||
		result["callCompleted"] != true || result["returnType"] != "void" || result["atUTC"] != receipt["toolCloseReturnedAtUTC"] ||
		result["requestPath"] != receipt["toolCloseRequestPath"] || result["requestSha256"] != receipt["toolCloseRequestSha256"] { return errors.New("CUA close method return invalid") }
	if receipt["actionEvidencePath"] != a.ActionEvidencePath || receipt["finalInventoryPath"] != a.FinalInventoryPath { return errors.New("CUA terminal path mismatch") }
	action, err := r139NativeFile(a, "action", receipt["actionEvidencePath"], receipt["actionEvidenceSha256"], product); if err != nil { return err }
	if err := r139CuaIdentity(a, action); err != nil { return err }
	if err := r139CuaNativeCloseFields(action); err != nil { return err }
	if _, err := r139CuaDerived(action); err != nil { return err }
	if action["event"] != "action_sequence" || action["atUTC"] != receipt["actionAtUTC"] { return errors.New("CUA action identity invalid") }
	aggregated, err := r139NativeTime(action["atUTC"]); if err != nil || aggregated.Before(returned) || aggregated.After(observed) { return errors.New("CUA tool action aggregate chronology invalid") }
	rows, ok := action["actions"].([]any); if !ok || len(rows) == 0 { return errors.New("CUA actions absent") }
	links := map[string]any{}; previous := reserved
	for index, value := range rows {
		row, ok := value.(map[string]any); if !ok { return errors.New("CUA action row invalid") }
		if err := r139CuaNativeCloseFields(row); err != nil { return err }
		at, err := r139CuaOrigin(row); if err != nil { return err }
		if row["tabId"] != a.TabID || at.Before(previous) || at.After(returned) { return errors.New("CUA action chronology invalid") }; previous = at
		links[fmt.Sprint(index)] = row["originalCuaReturnSha256"]
		if index == len(rows)-1 {
			if row["action"] != "tool_close" || !at.Equal(returned) || row["closeActorTaskId"] != a.ProducerAgentTaskID ||
				row["toolCloseReturnPath"] != receipt["toolCloseReturnPath"] || row["toolCloseReturnSha256"] != receipt["toolCloseReturnSha256"] { return errors.New("CUA close not terminal") }
			for _, key := range []string{"cuaCallRef", "originalCuaReturnReference", "originalCuaReturnSha256"} { if row[key] != result[key] { return errors.New("CUA close row return mismatch") } }
		} else {
			kind := row["action"]
			if at.After(requested) || (kind != "control" && kind != "capture" && kind != "handoff" && kind != "product_read" && kind != "product_execute") { return errors.New("CUA post-close action") }
		}
	}
	if !reflect.DeepEqual(action["sourceEvidenceSha256"], links) { return errors.New("CUA action source linkage invalid") }
	inv, err := r139NativeFile(a, "finalInventory", receipt["finalInventoryPath"], receipt["finalInventorySha256"], product); if err != nil { return err }
	raw, err := r139NativeFile(a, "rawCuaInventory", inv["rawCuaInventoryPath"], inv["rawCuaInventorySha256"], product); if err != nil { return err }
	for _, m := range []map[string]any{inv, raw} {
		if err := r139CuaIdentity(a, m); err != nil { return err }
		if err := r139CuaNativeCloseFields(m); err != nil { return err }
		if _, err := r139CuaOrigin(m); err != nil { return err }
		if m["inventoryActorTaskId"] != a.ObserverAgentTaskID || m["atUTC"] != receipt["closureObservedAtUTC"] ||
			m["inventoryScope"] != "metadata_only" || !r139NativeZeroInteger(m["ownedCount"]) { return errors.New("CUA observing actor or inventory invalid") }
	}
	if inv["event"] != "final_inventory" || raw["event"] != "cua_inventory" || inv["rawCuaInventoryPath"] != a.RawCuaInventoryPath || !reflect.DeepEqual(raw["tabIds"], inv["tabIds"]) { return errors.New("CUA raw inventory mismatch") }
	finalIDs, err := r139PageIDs(inv["tabIds"]); if err != nil || finalIDs[a.TabID] { return errors.New("CUA target still present") }
	baseline, _, err := r139CuaEvent(a, readiness, prefix, "baselineInventory", "baseline_inventory"); if err != nil { return err }
	baseIDs, err := r139PageIDs(baseline["tabIds"]); if err != nil || !reflect.DeepEqual(finalIDs, baseIDs) || raw["cuaCallRef"] == baseline["cuaCallRef"] { return errors.New("CUA protected baseline drift") }
	markerPath, err := r139NativePath(a, "browserDoneSignalPath", a.BrowserDoneSignalPath, product, true); if err != nil { return err }
	markerRaw, err := os.ReadFile(markerPath); if err != nil { return err }
	marker, err := r139NativeObject(markerRaw); if err != nil { return err }
	if marker["status"] != "TERMINAL_EVIDENCE_WRITTEN" { return errors.New("CUA done status invalid") }
	for key, value := range map[string]any{"proofContract": a.ProofContract, "campaignNonce": a.CampaignNonce, "browserId": a.BrowserID, "tabId": a.TabID, "ownedScopeId": a.OwnedScopeID} { if marker[key] != value { return errors.New("CUA done identity invalid") } }
	markerAt, err := r139NativeTime(marker["atUTC"]); if err != nil || markerAt.Before(observed) || markerAt.After(deadline) { return errors.New("CUA done chronology invalid") }
	sum := sha256.Sum256(closeRaw)
	if !reflect.DeepEqual(marker["evidenceSha256"], map[string]any{"browserCloseReceiptPath": hex.EncodeToString(sum[:]), "actionEvidencePath": receipt["actionEvidenceSha256"], "finalInventoryPath": receipt["finalInventorySha256"], "rawCuaInventoryPath": inv["rawCuaInventorySha256"]}) { return errors.New("CUA done evidence mismatch") }
	return nil
}

func r139CuaTerminal(a r139Admission, closeRaw []byte) error {
	if a.CloseMode == "native_tool" { return r139CuaNativeToolTerminal(a, closeRaw) }
	receipt, err := r139NativeObject(closeRaw); if err != nil { return err }
	if err := r139CuaIdentity(a, receipt); err != nil { return err }
	if receipt["status"] != "CLOSED_EXACT_OWNED_CUA_TAB" || receipt["closeObserved"] != true ||
		!r139NativeZeroInteger(receipt["ownedTabsFinal"]) || receipt["closureTimeBasis"] != "cua_final_metadata_observation" ||
		receipt["rootOriginalReturnReviewed"] != true { return errors.New("CUA close status invalid") }
	for _, key := range []string{"actualCloseActionAtUTC", "closedAtUTC"} {
		if _, exists := receipt[key]; exists { return errors.New("invented CUA close time") }
	}
	_, readiness, err := r139NativeRoot(a, "browserReadinessRoot"); if err != nil { return err }
	_, product, err := r139NativeRoot(a, "runtimeRoot"); if err != nil { return err }
	closePath, err := r139NativePath(a, "browserCloseReceipt", a.BrowserCloseReceiptPath, product, true)
	if err != nil { return err }
	stored, err := os.ReadFile(closePath)
	if err != nil || !bytes.Equal(stored, closeRaw) { return errors.New("CUA close receipt bytes differ") }
	prefix, err := r139NativeFile(a, "browserReceipt", a.BrowserReceiptPath, a.BrowserReceiptSHA256, readiness)
	if err != nil || prefix["reservedAtUTC"] != receipt["reservedAtUTC"] { return errors.New("CUA prefix mismatch") }
	prefixRaw, err := json.Marshal(prefix); if err != nil { return err }
	if err := r139CuaReservation(a, prefixRaw, true); err != nil { return err }
	reserved, err := r139NativeTime(prefix["reservedAtUTC"]); if err != nil { return err }
	reported, err := r139NativeTime(receipt["operatorReportReceivedAtUTC"]); if err != nil { return err }
	observed, err := r139NativeTime(receipt["closureObservedAtUTC"]); if err != nil { return err }
	deadline, err := time.Parse("2006-01-02T15:04:05Z", a.GrantDeadlineUTC)
	if err != nil || reserved.After(reported) || reported.After(observed) || observed.After(deadline) { return errors.New("CUA close chronology") }
	ownerHash, ok := receipt["ownerReportReferenceSha256"].(string)
	if !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(ownerHash) || receipt["operator"] != "Owner in this conversation" { return errors.New("CUA owner report invalid") }
	if receipt["actionEvidencePath"] != a.ActionEvidencePath || receipt["finalInventoryPath"] != a.FinalInventoryPath { return errors.New("CUA terminal path mismatch") }
	action, err := r139NativeFile(a, "action", receipt["actionEvidencePath"], receipt["actionEvidenceSha256"], product)
	if err != nil { return err }
	if err := r139CuaIdentity(a, action); err != nil || action["event"] != "action_sequence" || action["atUTC"] != receipt["actionAtUTC"] { return errors.New("CUA action identity invalid") }
	if _, err := r139CuaDerived(action); err != nil { return err }
	rows, ok := action["actions"].([]any)
	if !ok || len(rows) < 1 { return errors.New("CUA action rows absent") }
	links, ok := action["sourceEvidenceSha256"].(map[string]any)
	if !ok { return errors.New("CUA action source links absent") }
	wantLinks := map[string]any{}
	previous := reserved
	for index, value := range rows {
		row, ok := value.(map[string]any)
		if !ok || row["tabId"] != a.TabID { return errors.New("CUA action tab invalid") }
		kind := row["action"]
		if kind != "control" && kind != "capture" && kind != "handoff" && kind != "product_read" && kind != "product_execute" && kind != "owner_close_report" { return errors.New("CUA action kind invalid") }
		at, err := r139NativeTime(row["atUTC"])
		if err != nil || at.Before(previous) || at.After(reported) { return errors.New("CUA action chronology invalid") }
		previous = at
		if kind == "owner_close_report" {
			if index != len(rows)-1 || !at.Equal(reported) || row["reportSource"] != "owner_in_conversation" || row["ownerReportReferenceSha256"] != ownerHash { return errors.New("CUA report not terminal") }
		} else {
			sha, ok := row["originalCuaReturnSha256"].(string)
			if index == len(rows)-1 || !ok || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(sha) || row["provenanceBasis"] != "caller_recorded_original_cua_return" { return errors.New("CUA action original invalid") }
			if err := r139CuaReference(row["originalCuaReturnReference"]); err != nil { return err }
			if err := r139CuaReference(row["cuaCallRef"]); err != nil { return err }
			wantLinks[fmt.Sprint(index)] = sha
		}
	}
	if rows[len(rows)-1].(map[string]any)["action"] != "owner_close_report" { return errors.New("CUA post-close action") }
	if len(rows) == 1 { wantLinks["prefix"] = a.BrowserReceiptSHA256 }
	if !reflect.DeepEqual(links, wantLinks) { return errors.New("CUA action source linkage invalid") }
	inv, err := r139NativeFile(a, "finalInventory", receipt["finalInventoryPath"], receipt["finalInventorySha256"], product)
	if err != nil { return err }
	if err := r139CuaIdentity(a, inv); err != nil || inv["event"] != "final_inventory" || inv["atUTC"] != receipt["closureObservedAtUTC"] || inv["rawCuaInventoryPath"] != a.RawCuaInventoryPath || inv["inventoryScope"] != "metadata_only" || !r139NativeZeroInteger(inv["ownedCount"]) { return errors.New("CUA final inventory invalid") }
	if _, err := r139CuaOrigin(inv); err != nil { return err }
	finalIDs, err := r139PageIDs(inv["tabIds"]); if err != nil || finalIDs[a.TabID] { return errors.New("CUA target still present") }
	baseline, _, err := r139CuaEvent(a, readiness, prefix, "baselineInventory", "baseline_inventory"); if err != nil { return err }
	baseIDs, err := r139PageIDs(baseline["tabIds"]); if err != nil || !reflect.DeepEqual(finalIDs, baseIDs) { return errors.New("CUA protected baseline drift") }
	raw, err := r139NativeFile(a, "rawCuaInventory", inv["rawCuaInventoryPath"], inv["rawCuaInventorySha256"], product)
	if err != nil { return err }
	if err := r139CuaIdentity(a, raw); err != nil || raw["event"] != "cua_inventory" || raw["atUTC"] != inv["atUTC"] || !reflect.DeepEqual(raw["tabIds"], inv["tabIds"]) || raw["inventoryScope"] != "metadata_only" || !r139NativeZeroInteger(raw["ownedCount"]) || raw["cuaCallRef"] == baseline["cuaCallRef"] { return errors.New("CUA raw inventory invalid") }
	if _, err := r139CuaOrigin(raw); err != nil { return err }
	markerPath, err := r139NativePath(a, "browserDoneSignalPath", a.BrowserDoneSignalPath, product, true); if err != nil { return err }
	markerRaw, err := os.ReadFile(markerPath); if err != nil { return err }
	marker, err := r139NativeObject(markerRaw); if err != nil { return err }
	if marker["status"] != "TERMINAL_EVIDENCE_WRITTEN" || marker["proofContract"] != "cua_page_target_v1" || marker["campaignNonce"] != a.CampaignNonce || marker["browserId"] != a.BrowserID || marker["tabId"] != a.TabID || marker["ownedScopeId"] != a.OwnedScopeID { return errors.New("CUA done identity invalid") }
	markerAt, err := r139NativeTime(marker["atUTC"])
	if err != nil || markerAt.Before(observed) || markerAt.After(deadline) { return errors.New("CUA done chronology invalid") }
	marked, ok := marker["evidenceSha256"].(map[string]any)
	if !ok || len(marked) != 4 || marked["browserCloseReceiptPath"] != func() string { sum := sha256.Sum256(closeRaw); return hex.EncodeToString(sum[:]) }() || marked["actionEvidencePath"] != receipt["actionEvidenceSha256"] || marked["finalInventoryPath"] != receipt["finalInventorySha256"] || marked["rawCuaInventoryPath"] != inv["rawCuaInventorySha256"] { return errors.New("CUA done hashes invalid") }
	return nil
}

func r139BrowserRoute(method, path string) (auth, execute, read bool) {
	auth = method == http.MethodPost && (path == "/api/v1/auth/login" || path == "/api/v1/auth/logout" || path == "/api/v1/auth/refresh")
	execute = method == http.MethodPost && regexp.MustCompile(`^/api/v1/tenants/[^/]+/intervention-cases/[^/]+/rerun-requests/[^/]+/execute$`).MatchString(path)
	read = method == http.MethodGet || method == http.MethodHead
	return
}

func TestR139PureRouteAdmission(t *testing.T) {
	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/refresh"} {
		auth, execute, read := r139BrowserRoute(http.MethodPost, path)
		if !auth || execute || read { t.Fatalf("R139 auth route %q", path) }
	}
	for _, path := range []string{"/api/v1/auth/refresh/extra", "/api/v1/auth/reset", "/api/v1/tenants/t/intervention-cases/c/rerun-requests/r/execute/extra"} {
		auth, execute, read := r139BrowserRoute(http.MethodPost, path)
		if auth || execute || read { t.Fatalf("R139 mutation route %q", path) }
	}
	auth, execute, read := r139BrowserRoute(http.MethodPost, "/api/v1/tenants/t/intervention-cases/c/rerun-requests/r/execute")
	if auth || !execute || read { t.Fatal("R139 execute route") }
}

func r139NativeTestWrite(t *testing.T, path string, value any) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { t.Fatal(err) }
	raw, err := json.Marshal(value)
	if err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, raw, 0644); err != nil { t.Fatal(err) }
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func TestR139PureCuaPageTargetV1(t *testing.T) {
	root := t.TempDir()
	base := ".cvf/runtime/cua_readiness/"
	product := ".cvf/runtime/cua_product/"
	readiness := filepath.Join(root, filepath.FromSlash(base))
	runtime := filepath.Join(root, filepath.FromSlash(product))
	if err := os.MkdirAll(readiness, 0755); err != nil { t.Fatal(err) }
	now := time.Now().UTC().Truncate(time.Second)
	stamp := func(offset int) string { return now.Add(time.Duration(offset) * time.Second).Format("2006-01-02T15:04:05Z") }
	a := r139Admission{NativeContractID: "r139-cua-page-target-v1", TrancheID: "CCMAI-RUNTIME-139",
		CampaignNonce: "nonce-cua", BrowserID: "2", BrowserKind: "iab", TabID: "tab-123",
		OwnedScopeID: "owned-cua-tab", OwnedTabIDs: []string{"tab-123"},
		ProducerAgentTaskID: "producer-task", ObserverAgentTaskID: "observer-task",
		Operator: "Owner in this conversation", FallbackCloseOperator: "Owner in this conversation",
		CloseMode: "operator_fallback", ScopeOrigin: "caller_owned_explicit_cua_tab_target",
		ProofContract: "cua_page_target_v1", BrowserTool: "supported_CUA_page_target", BrowserSupported: true,
		BrowserReadinessRoot: base[:len(base)-1], RuntimeRoot: runtime, nativeProjectRoot: root,
		BrowserReceiptPath: base + "reservation.json", BrowserCloseReceiptPath: product + "close.json",
		ActionEvidencePath: product + "action.json", FinalInventoryPath: product + "inventory.json",
		RawCuaInventoryPath: product + "raw_inventory.json", BrowserDoneSignalPath: product + "browser_done.signal",
		GrantDeadlineUTC: stamp(3600)}
	identity := func() map[string]any { return map[string]any{
		"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID, "campaignNonce": a.CampaignNonce,
		"browserId": a.BrowserID, "tabId": a.TabID, "ownedScopeId": a.OwnedScopeID,
		"producerAgentTaskId": a.ProducerAgentTaskID, "observerAgentTaskId": a.ObserverAgentTaskID,
		"operator": a.Operator, "closeMode": a.CloseMode, "proofContract": a.ProofContract,
		"browserKind": a.BrowserKind, "scopeOrigin": a.ScopeOrigin, "ownedTabIds": []string{a.TabID}} }
	event := func(name string, offset int) map[string]any {
		m := identity(); m["event"] = name; m["atUTC"] = stamp(offset); m["observedAtUTC"] = m["atUTC"]
		m["cuaCallRef"] = "cua-call-1"; m["originalCuaReturnReference"] = "private/cua/call-1"
		m["originalCuaReturnSha256"] = strings.Repeat("b", 64)
		m["provenanceBasis"] = "caller_recorded_original_cua_return"; m["rootOriginalReturnReviewRequired"] = true
		return m
	}
	created := "https://example.com/#r139_nonce=nonce-cua&phase=created"
	controlled := "https://example.com/#r139_nonce=nonce-cua&phase=controlled"
	rows := map[string]map[string]any{
		"baselineInventory": event("baseline_inventory", -60), "creation": event("creation", -50),
		"creationLedger": event("creation_ledger", -49), "control": event("control", -45),
		"capture": event("capture", -40), "handoff": event("handoff", -30),
	}
	rows["baselineInventory"]["tabIds"] = []string{"protected-tab"}; rows["baselineInventory"]["inventoryScope"] = "metadata_only"
	rows["creation"]["agentTaskId"] = a.ProducerAgentTaskID; rows["creation"]["method"] = "createBrowserTab"
	rows["creation"]["createdTabId"] = a.TabID; rows["creation"]["requestedBrowserId"] = a.BrowserID
	rows["creation"]["requestedUrl"] = created; rows["creation"]["callCompleted"] = true
	rows["creationLedger"]["agentTaskId"] = a.ProducerAgentTaskID; rows["creationLedger"]["newTabIds"] = []string{a.TabID}
	rows["control"]["agentTaskId"] = a.ProducerAgentTaskID; rows["control"]["method"] = "goto"
	rows["control"]["targetTabId"] = a.TabID; rows["control"]["requestedUrl"] = controlled; rows["control"]["callCompleted"] = true
	rows["capture"]["agentTaskId"] = a.ProducerAgentTaskID; rows["capture"]["method"] = "getScreenshot"
	rows["capture"]["targetTabId"] = a.TabID
	rows["capture"]["captureSha256"] = strings.Repeat("c", 64)
	rows["handoff"]["agentTaskId"] = a.ObserverAgentTaskID; rows["handoff"]["method"] = "getTab_then_listTabs"
	rows["handoff"]["requestedTabId"] = a.TabID; rows["handoff"]["observedTabId"] = a.TabID
	rows["handoff"]["metadataBrowserId"] = a.BrowserID; rows["handoff"]["metadataTabId"] = a.TabID
	rows["handoff"]["observedUrl"] = controlled; rows["handoff"]["observedNonce"] = a.CampaignNonce
	rows["handoff"]["getTabCallRef"] = "cua-gettab"; rows["handoff"]["getTabReturnReference"] = "private/cua/gettab"
	rows["handoff"]["getTabReturnSha256"] = strings.Repeat("e", 64)
	rows["handoff"]["metadataCallRef"] = "cua-listtabs"; rows["handoff"]["metadataReturnReference"] = "private/cua/listtabs"
	rows["handoff"]["metadataReturnSha256"] = strings.Repeat("f", 64)
	proof := identity(); proof["status"] = "RESERVED_CUA_TAB_CONTROL_CAPTURE_HANDOFF"
	proof["protocol"] = "https"; proof["closeObserved"] = false; proof["rootOriginalReturnReviewed"] = true
	proof["reservedAtUTC"] = stamp(-25)
	writeEvent := func(name string) {
		suffix := "Evidence"; if name == "baselineInventory" || name == "creationLedger" { suffix = "" }
		proof[name+suffix+"Path"] = base + name + ".json"
		proof[name+suffix+"Sha256"] = r139NativeTestWrite(t, filepath.Join(readiness, name+".json"), rows[name])
	}
	for name := range rows { writeEvent(name) }
	rows["creationLedger"]["baselineInventorySha256"] = proof["baselineInventorySha256"]
	rows["creationLedger"]["creationEvidenceSha256"] = proof["creationEvidenceSha256"]
	delete(rows["creationLedger"], "cuaCallRef")
	delete(rows["creationLedger"], "originalCuaReturnReference")
	delete(rows["creationLedger"], "originalCuaReturnSha256")
	rows["creationLedger"]["provenanceBasis"] = "derived_from_original_cua_returns"
	rows["creationLedger"]["sourceEvidenceSha256"] = map[string]any{"baselineInventory": proof["baselineInventorySha256"], "creation": proof["creationEvidenceSha256"]}
	writeEvent("creationLedger")
	delete(rows["handoff"], "cuaCallRef")
	delete(rows["handoff"], "originalCuaReturnReference")
	delete(rows["handoff"], "originalCuaReturnSha256")
	rows["handoff"]["provenanceBasis"] = "derived_from_original_cua_returns"
	rows["handoff"]["sourceEvidenceSha256"] = map[string]any{"getTab": strings.Repeat("e", 64), "listTabs": strings.Repeat("f", 64)}
	writeEvent("handoff")
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(readiness, "reservation.json"), proof)
	encode := func(value any) []byte { raw, err := json.Marshal(value); if err != nil { t.Fatal(err) }; return raw }
	if err := r139VerifyBrowserReservation(a, encode(proof)); err != nil { t.Fatalf("CUA prefix rejected: %v", err) }
	bad := rows["handoff"]["agentTaskId"]
	rows["handoff"]["agentTaskId"] = a.ProducerAgentTaskID
	writeEvent("handoff")
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(readiness, "reservation.json"), proof)
	if err := r139VerifyBrowserReservation(a, encode(proof)); err == nil { t.Fatal("same-agent CUA handoff accepted") }
	rows["handoff"]["agentTaskId"] = bad; writeEvent("handoff")
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(readiness, "reservation.json"), proof)
	if err := r139VerifyBrowserReservation(a, encode(proof)); err != nil { t.Fatalf("restored CUA prefix rejected: %v", err) }
	badAdmission := a; badAdmission.PageID = a.TabID
	if err := r139VerifyBrowserReservation(badAdmission, encode(proof)); err == nil { t.Fatal("fabricated native page alias accepted") }
	if err := os.MkdirAll(runtime, 0755); err != nil { t.Fatal(err) }
	raw := event("cua_inventory", -5); raw["tabIds"] = []string{"protected-tab"}; raw["ownedCount"] = 0; raw["inventoryScope"] = "metadata_only"; raw["cuaCallRef"] = "cua-final-call"
	rawSHA := r139NativeTestWrite(t, filepath.Join(runtime, "raw_inventory.json"), raw)
	actions := event("action_sequence", -20)
	actions["actions"] = []map[string]any{{"tabId": a.TabID, "action": "control", "atUTC": stamp(-20),
		"cuaCallRef": "cua-control", "originalCuaReturnReference": "private/cua/control",
		"originalCuaReturnSha256": strings.Repeat("e", 64), "provenanceBasis": "caller_recorded_original_cua_return"},
		{"tabId": a.TabID, "action": "owner_close_report", "atUTC": stamp(-8),
		"reportSource": "owner_in_conversation", "ownerReportReferenceSha256": strings.Repeat("a", 64)}}
	delete(actions, "cuaCallRef")
	delete(actions, "originalCuaReturnReference")
	delete(actions, "originalCuaReturnSha256")
	actions["provenanceBasis"] = "derived_from_original_cua_returns"
	actions["sourceEvidenceSha256"] = map[string]any{"0": strings.Repeat("e", 64)}
	actionSHA := r139NativeTestWrite(t, filepath.Join(runtime, "action.json"), actions)
	inv := event("final_inventory", -5); inv["tabIds"] = []string{"protected-tab"}; inv["ownedCount"] = 0
	inv["inventoryScope"] = "metadata_only"; inv["rawCuaInventoryPath"] = a.RawCuaInventoryPath
	inv["rawCuaInventorySha256"] = rawSHA
	invSHA := r139NativeTestWrite(t, filepath.Join(runtime, "inventory.json"), inv)
	close := identity(); close["status"] = "CLOSED_EXACT_OWNED_CUA_TAB"; close["closeObserved"] = true
	close["ownedTabsFinal"] = 0; close["closureTimeBasis"] = "cua_final_metadata_observation"
	close["rootOriginalReturnReviewed"] = true; close["reservedAtUTC"] = proof["reservedAtUTC"]
	close["actionAtUTC"] = stamp(-20); close["operatorReportReceivedAtUTC"] = stamp(-8)
	close["closureObservedAtUTC"] = stamp(-5); close["ownerReportReferenceSha256"] = strings.Repeat("a", 64)
	close["actionEvidencePath"] = a.ActionEvidencePath; close["actionEvidenceSha256"] = actionSHA
	close["finalInventoryPath"] = a.FinalInventoryPath; close["finalInventorySha256"] = invSHA
	closeSHA := r139NativeTestWrite(t, filepath.Join(runtime, "close.json"), close)
	marker := map[string]any{"status": "TERMINAL_EVIDENCE_WRITTEN", "proofContract": a.ProofContract,
		"campaignNonce": a.CampaignNonce, "browserId": a.BrowserID, "tabId": a.TabID,
		"ownedScopeId": a.OwnedScopeID, "atUTC": stamp(-4),
		"evidenceSha256": map[string]any{"browserCloseReceiptPath": closeSHA, "actionEvidencePath": actionSHA,
			"finalInventoryPath": invSHA, "rawCuaInventoryPath": rawSHA}}
	r139NativeTestWrite(t, filepath.Join(runtime, "browser_done.signal"), marker)
	if err := r139VerifyBrowserTerminalClose(a, encode(close)); err != nil { t.Fatalf("CUA terminal rejected: %v", err) }
	actions["actions"] = []map[string]any{actions["actions"].([]map[string]any)[1]}
	actions["atUTC"] = stamp(-8); actions["observedAtUTC"] = stamp(-8)
	actions["sourceEvidenceSha256"] = map[string]any{"prefix": a.BrowserReceiptSHA256}
	close["actionAtUTC"] = stamp(-8)
	close["actionEvidenceSha256"] = r139NativeTestWrite(t, filepath.Join(runtime, "action.json"), actions)
	closeSHA = r139NativeTestWrite(t, filepath.Join(runtime, "close.json"), close)
	marker["evidenceSha256"] = map[string]any{"browserCloseReceiptPath": closeSHA,
		"actionEvidencePath": close["actionEvidenceSha256"], "finalInventoryPath": invSHA,
		"rawCuaInventoryPath": rawSHA}
	r139NativeTestWrite(t, filepath.Join(runtime, "browser_done.signal"), marker)
	if err := r139VerifyBrowserTerminalClose(a, encode(close)); err != nil { t.Fatalf("CUA owner-only terminal rejected: %v", err) }
	inv["tabIds"] = []string{"protected-tab", a.TabID}
	close["finalInventorySha256"] = r139NativeTestWrite(t, filepath.Join(runtime, "inventory.json"), inv)
	r139NativeTestWrite(t, filepath.Join(runtime, "close.json"), close)
	if err := r139VerifyBrowserTerminalClose(a, encode(close)); err == nil { t.Fatal("target still present accepted") }
}

func r139CuaNativeTestFixture(t *testing.T) (r139Admission, map[string]any, func(string, int) map[string]any, func(int) string, string) {
	t.Helper()
	root := t.TempDir()
	base := ".cvf/runtime/cua_readiness/"
	product := ".cvf/runtime/cua_product/"
	readiness := filepath.Join(root, filepath.FromSlash(base))
	runtime := filepath.Join(root, filepath.FromSlash(product))
	if err := os.MkdirAll(readiness, 0755); err != nil { t.Fatal(err) }
	now := time.Now().UTC().Truncate(time.Second)
	stamp := func(offset int) string { return now.Add(time.Duration(offset) * time.Second).Format("2006-01-02T15:04:05Z") }
	a := r139Admission{NativeContractID: "r139-cua-page-target-v1", TrancheID: "CCMAI-RUNTIME-139",
		CampaignNonce: "nonce-cua", BrowserID: "2", BrowserKind: "iab", TabID: "tab-123",
		OwnedScopeID: "owned-cua-tab", OwnedTabIDs: []string{"tab-123"},
		ProducerAgentTaskID: "producer-task", ObserverAgentTaskID: "observer-task",
		Operator: "Owner in this conversation", FallbackCloseOperator: "Owner in this conversation",
		CloseMode: "native_tool", ScopeOrigin: "caller_owned_explicit_cua_tab_target",
		ProofContract: "cua_page_target_v1", BrowserTool: "supported_CUA_page_target", BrowserSupported: true,
		BrowserReadinessRoot: base[:len(base)-1], RuntimeRoot: runtime, nativeProjectRoot: root,
		BrowserReceiptPath: base + "reservation.json", BrowserCloseReceiptPath: product + "close.json",
		ActionEvidencePath: product + "action.json", FinalInventoryPath: product + "inventory.json",
		RawCuaInventoryPath: product + "raw_inventory.json", BrowserDoneSignalPath: product + "browser_done.signal",
		GrantDeadlineUTC: stamp(3600)}
	identity := func() map[string]any { return map[string]any{
		"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID, "campaignNonce": a.CampaignNonce,
		"browserId": a.BrowserID, "tabId": a.TabID, "ownedScopeId": a.OwnedScopeID,
		"producerAgentTaskId": a.ProducerAgentTaskID, "observerAgentTaskId": a.ObserverAgentTaskID,
		"operator": a.Operator, "closeMode": a.CloseMode, "proofContract": a.ProofContract,
		"browserKind": a.BrowserKind, "scopeOrigin": a.ScopeOrigin, "ownedTabIds": []string{a.TabID}} }
	event := func(name string, offset int) map[string]any {
		m := identity(); m["event"] = name; m["atUTC"] = stamp(offset); m["observedAtUTC"] = m["atUTC"]
		m["cuaCallRef"] = "cua-call-1"; m["originalCuaReturnReference"] = "private/cua/call-1"
		m["originalCuaReturnSha256"] = strings.Repeat("b", 64)
		m["provenanceBasis"] = "caller_recorded_original_cua_return"; m["rootOriginalReturnReviewRequired"] = true
		return m
	}
	created := "https://example.com/#r139_nonce=nonce-cua&phase=created"
	controlled := "https://example.com/#r139_nonce=nonce-cua&phase=controlled"
	rows := map[string]map[string]any{
		"baselineInventory": event("baseline_inventory", -60), "creation": event("creation", -50),
		"creationLedger": event("creation_ledger", -49), "control": event("control", -45),
		"capture": event("capture", -40), "handoff": event("handoff", -30),
	}
	rows["baselineInventory"]["tabIds"] = []string{"protected-tab"}; rows["baselineInventory"]["inventoryScope"] = "metadata_only"
	rows["creation"]["agentTaskId"] = a.ProducerAgentTaskID; rows["creation"]["method"] = "createBrowserTab"
	rows["creation"]["createdTabId"] = a.TabID; rows["creation"]["requestedBrowserId"] = a.BrowserID
	rows["creation"]["requestedUrl"] = created; rows["creation"]["callCompleted"] = true
	rows["creationLedger"]["agentTaskId"] = a.ProducerAgentTaskID; rows["creationLedger"]["newTabIds"] = []string{a.TabID}
	rows["control"]["agentTaskId"] = a.ProducerAgentTaskID; rows["control"]["method"] = "goto"
	rows["control"]["targetTabId"] = a.TabID; rows["control"]["requestedUrl"] = controlled; rows["control"]["callCompleted"] = true
	rows["capture"]["agentTaskId"] = a.ProducerAgentTaskID; rows["capture"]["method"] = "getScreenshot"
	rows["capture"]["targetTabId"] = a.TabID
	rows["capture"]["captureSha256"] = strings.Repeat("c", 64)
	rows["handoff"]["agentTaskId"] = a.ObserverAgentTaskID; rows["handoff"]["method"] = "getTab_then_listTabs"
	rows["handoff"]["requestedTabId"] = a.TabID; rows["handoff"]["observedTabId"] = a.TabID
	rows["handoff"]["metadataBrowserId"] = a.BrowserID; rows["handoff"]["metadataTabId"] = a.TabID
	rows["handoff"]["observedUrl"] = controlled; rows["handoff"]["observedNonce"] = a.CampaignNonce
	rows["handoff"]["getTabCallRef"] = "cua-gettab"; rows["handoff"]["getTabReturnReference"] = "private/cua/gettab"
	rows["handoff"]["getTabReturnSha256"] = strings.Repeat("e", 64)
	rows["handoff"]["metadataCallRef"] = "cua-listtabs"; rows["handoff"]["metadataReturnReference"] = "private/cua/listtabs"
	rows["handoff"]["metadataReturnSha256"] = strings.Repeat("f", 64)
	proof := identity(); proof["status"] = "RESERVED_CUA_TAB_CONTROL_CAPTURE_HANDOFF"
	proof["protocol"] = "https"; proof["closeObserved"] = false; proof["rootOriginalReturnReviewed"] = true
	proof["reservedAtUTC"] = stamp(-25)
	writeEvent := func(name string) {
		suffix := "Evidence"; if name == "baselineInventory" || name == "creationLedger" { suffix = "" }
		proof[name+suffix+"Path"] = base + name + ".json"
		proof[name+suffix+"Sha256"] = r139NativeTestWrite(t, filepath.Join(readiness, name+".json"), rows[name])
	}
	for name := range rows { writeEvent(name) }
	rows["creationLedger"]["baselineInventorySha256"] = proof["baselineInventorySha256"]
	rows["creationLedger"]["creationEvidenceSha256"] = proof["creationEvidenceSha256"]
	delete(rows["creationLedger"], "cuaCallRef")
	delete(rows["creationLedger"], "originalCuaReturnReference")
	delete(rows["creationLedger"], "originalCuaReturnSha256")
	rows["creationLedger"]["provenanceBasis"] = "derived_from_original_cua_returns"
	rows["creationLedger"]["sourceEvidenceSha256"] = map[string]any{"baselineInventory": proof["baselineInventorySha256"], "creation": proof["creationEvidenceSha256"]}
	writeEvent("creationLedger")
	delete(rows["handoff"], "cuaCallRef")
	delete(rows["handoff"], "originalCuaReturnReference")
	delete(rows["handoff"], "originalCuaReturnSha256")
	rows["handoff"]["provenanceBasis"] = "derived_from_original_cua_returns"
	rows["handoff"]["sourceEvidenceSha256"] = map[string]any{"getTab": strings.Repeat("e", 64), "listTabs": strings.Repeat("f", 64)}
	writeEvent("handoff")
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(readiness, "reservation.json"), proof)
	return a, proof, event, stamp, runtime
}

func TestR139PureCuaNativeToolClose(t *testing.T) {
	for _, bad := range []string{"", "failed_return", "unknown_return", "missing_return", "wrong_digest", "wrong_request_digest", "wrong_request_link", "missing_request", "bad_request_origin", "wrong_method", "foreign_target", "foreign_campaign", "wrong_close_actor", "wrong_receipt_actor", "same_actor", "wrong_inventory_actor", "wrong_raw_actor", "before_close", "late_close", "target_present", "baseline_drift", "post_close_action", "owner_alias", "physical_time", "wrong_mode", "aggregate_before_return", "aggregate_after_observed"} {
		t.Run("case_"+bad, func(t *testing.T) {
			a, prefix, event, stamp, runtime := r139CuaNativeTestFixture(t)
			if err := os.MkdirAll(runtime, 0755); err != nil { t.Fatal(err) }
			write := func(filename string, m map[string]any) (string, string) { return ".cvf/runtime/cua_product/"+filename, r139NativeTestWrite(t, filepath.Join(runtime, filename), m) }
			request := event("tool_close_request", -12)
			for _, key := range []string{"cuaCallRef", "originalCuaReturnReference", "originalCuaReturnSha256"} { delete(request, key) }
			request["provenanceBasis"] = "caller_recorded_original_cua_request"
			request["originalCuaRequestReference"] = "private/cua/close-request"; request["originalCuaRequestSha256"] = strings.Repeat("a", 64)
			request["method"] = "Tab.close"; request["targetTabId"] = a.TabID; request["closeActorTaskId"] = a.ProducerAgentTaskID; request["status"] = "REQUESTED"
			if bad == "wrong_method" { request["method"] = "Browser.close" }; if bad == "bad_request_origin" { request["originalCuaRequestSha256"] = "bad" }
			requestPath, requestSHA := write("request.json", request)
			result := event("tool_close_return", -10)
			result["method"] = "Tab.close"; result["targetTabId"] = a.TabID; result["closeActorTaskId"] = a.ProducerAgentTaskID
			result["status"] = "SUCCEEDED"; result["callCompleted"] = true; result["returnType"] = "void"
			result["requestPath"] = requestPath; result["requestSha256"] = requestSHA
			if bad == "failed_return" || bad == "unknown_return" { result["status"] = bad }
			if bad == "foreign_target" { result["targetTabId"] = "foreign-tab" }; if bad == "foreign_campaign" { result["campaignNonce"] = "foreign" }
			if bad == "wrong_close_actor" { result["closeActorTaskId"] = a.ObserverAgentTaskID }; if bad == "wrong_request_link" { result["requestSha256"] = strings.Repeat("f", 64) }
			returnPath, returnSHA := write("return.json", result)
			row := event("tool_close", -10); row["action"] = "tool_close"; row["closeActorTaskId"] = a.ProducerAgentTaskID
			row["toolCloseReturnPath"] = returnPath; row["toolCloseReturnSha256"] = returnSHA
			rows := []map[string]any{row}
			if bad == "post_close_action" { extra := event("product_read", -9); extra["action"] = "product_read"; rows = append(rows, extra) }
			action := event("action_sequence", -10); action["actions"] = rows
			if bad == "aggregate_before_return" { action["atUTC"] = stamp(-11); action["observedAtUTC"] = stamp(-11) }
			if bad == "aggregate_after_observed" { action["atUTC"] = stamp(-4); action["observedAtUTC"] = stamp(-4) }
			for _, key := range []string{"cuaCallRef", "originalCuaReturnReference", "originalCuaReturnSha256"} { delete(action, key) }
			action["provenanceBasis"] = "derived_from_original_cua_returns"; links := map[string]any{}
			for i, r := range rows { links[fmt.Sprint(i)] = r["originalCuaReturnSha256"] }; action["sourceEvidenceSha256"] = links
			actionPath, actionSHA := write("action.json", action)
			ids := []string{"protected-tab"}; if bad == "target_present" { ids = append(ids, a.TabID) }; if bad == "baseline_drift" { ids = []string{} }
			inventoryTime := -5; if bad == "before_close" { inventoryTime = -11 }
			raw := event("cua_inventory", inventoryTime); raw["tabIds"] = ids; raw["ownedCount"] = 0; raw["inventoryScope"] = "metadata_only"
			raw["inventoryActorTaskId"] = a.ObserverAgentTaskID; raw["cuaCallRef"] = "cua-independent-final"
			if bad == "wrong_raw_actor" { raw["inventoryActorTaskId"] = a.ProducerAgentTaskID }
			rawPath, rawSHA := write("raw_inventory.json", raw)
			inv := event("final_inventory", inventoryTime); inv["tabIds"] = ids; inv["ownedCount"] = 0; inv["inventoryScope"] = "metadata_only"
			inv["inventoryActorTaskId"] = a.ObserverAgentTaskID; inv["rawCuaInventoryPath"] = rawPath; inv["rawCuaInventorySha256"] = rawSHA
			if bad == "wrong_inventory_actor" { inv["inventoryActorTaskId"] = a.ProducerAgentTaskID }
			invPath, invSHA := write("inventory.json", inv)
			close := event("close_receipt", -5); close["status"] = "CLOSED_EXACT_OWNED_CUA_TAB"; close["closeObserved"] = true
			close["ownedTabsFinal"] = 0; close["closureTimeBasis"] = "cua_final_metadata_observation"; close["rootOriginalReturnReviewed"] = true
			close["reservedAtUTC"] = prefix["reservedAtUTC"]; close["closeActorTaskId"] = a.ProducerAgentTaskID; close["toolCloseReturnedAtUTC"] = stamp(-10)
			close["closureObservedAtUTC"] = stamp(inventoryTime); close["actionAtUTC"] = action["atUTC"]
			close["toolCloseRequestPath"] = requestPath; close["toolCloseRequestSha256"] = requestSHA
			close["toolCloseReturnPath"] = returnPath; close["toolCloseReturnSha256"] = returnSHA
			close["actionEvidencePath"] = actionPath; close["actionEvidenceSha256"] = actionSHA; close["finalInventoryPath"] = invPath; close["finalInventorySha256"] = invSHA
			if bad == "wrong_receipt_actor" { close["closeActorTaskId"] = a.ObserverAgentTaskID }; if bad == "owner_alias" { close["ownerReportReferenceSha256"] = strings.Repeat("a", 64) }
			if bad == "physical_time" { close["closedAtUTC"] = stamp(-10) }; if bad == "wrong_digest" { close["toolCloseReturnSha256"] = strings.Repeat("f", 64) }
			if bad == "wrong_request_digest" { close["toolCloseRequestSha256"] = strings.Repeat("f", 64) }
			if bad == "missing_return" { if err := os.Remove(filepath.Join(runtime, "return.json")); err != nil { t.Fatal(err) } }
			if bad == "missing_request" { if err := os.Remove(filepath.Join(runtime, "request.json")); err != nil { t.Fatal(err) } }
			if bad == "late_close" { a.GrantDeadlineUTC = stamp(-11) }; if bad == "same_actor" { a.ObserverAgentTaskID = a.ProducerAgentTaskID }; if bad == "wrong_mode" { a.CloseMode = "api" }
			_, closeSHA := write("close.json", close)
			write("browser_done.signal", map[string]any{"status": "TERMINAL_EVIDENCE_WRITTEN", "proofContract": a.ProofContract,
				"campaignNonce": a.CampaignNonce, "browserId": a.BrowserID, "tabId": a.TabID, "ownedScopeId": a.OwnedScopeID, "atUTC": stamp(-4),
				"evidenceSha256": map[string]any{"browserCloseReceiptPath": closeSHA, "actionEvidencePath": actionSHA, "finalInventoryPath": invSHA, "rawCuaInventoryPath": rawSHA}})
			body, err := json.Marshal(close); if err != nil { t.Fatal(err) }
			err = r139VerifyBrowserTerminalClose(a, body)
			if bad == "" && err != nil { t.Fatalf("valid native close rejected: %v", err) }; if bad != "" && err == nil { t.Fatal("invalid native close accepted") }
		})
	}
	t.Run("non_cua_modes_unchanged", func(t *testing.T) {
		a, _, _ := r139NativeTestFixture(t); m := r139NativeTestIdentity(a)
		if err := r139NativeIdentity(a, m); err != nil { t.Fatal(err) }
		a.CloseMode = "native_tool"; m["closeMode"] = a.CloseMode
		if err := r139NativeIdentity(a, m); err == nil { t.Fatal("native_tool accepted for generic native path") }
		a.ProofContract = "native_page_target_v1"; a.ScopeOrigin = "caller_owned_explicit_page_target"; a.BrowserSessionID = ""
		a.OwnedScopeID = "owned-page"; a.OwnedPageIDs = []string{a.PageID}; a.Operator = a.FallbackCloseOperator; a.CloseMode = "operator_fallback"
		m = map[string]any{"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID, "campaignNonce": a.CampaignNonce,
			"pageId": a.PageID, "ownedScopeId": a.OwnedScopeID, "ownedPageIds": []any{a.PageID}, "operator": a.Operator,
			"closeMode": a.CloseMode, "proofContract": a.ProofContract, "scopeOrigin": a.ScopeOrigin}
		if err := r139PageIdentity(a, m); err != nil { t.Fatal(err) }
		a.CloseMode = "native_tool"; m["closeMode"] = a.CloseMode
		if err := r139PageIdentity(a, m); err == nil { t.Fatal("native_tool accepted for non-CUA page target") }
	})
}

func r139NativeTestIdentity(a r139Admission) map[string]any {
	return map[string]any{
		"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID,
		"campaignNonce": a.CampaignNonce, "pageId": a.PageID,
		"browserSessionId": a.BrowserSessionID, "operator": a.FallbackCloseOperator,
		"closeMode": a.CloseMode,
	}
}

func r139NativeTestFixture(t *testing.T) (r139Admission, map[string]any, func(int) string) {
	t.Helper()
	project := t.TempDir()
	readiness := filepath.Join(project, ".cvf", "runtime", "readiness")
	product := filepath.Join(project, ".cvf", "runtime", "product")
	if err := os.MkdirAll(readiness, 0755); err != nil { t.Fatal(err) }
	now := time.Now().UTC().Truncate(time.Second)
	stamp := func(offset int) string { return now.Add(time.Duration(offset) * time.Second).Format(time.RFC3339) }
	a := r139Admission{
		NativeContractID: "r139-native-v1", TrancheID: "CCMAI-RUNTIME-139",
		CampaignNonce: "nonce-456", PageID: "page-123",
		BrowserSessionID: "session-789", Operator: "Owner", FallbackCloseOperator: "Owner",
		CloseMode: "operator_fallback", BrowserSupported: true,
		BrowserTool: "antigravity_native", BrowserReadinessRoot: ".cvf/runtime/readiness",
		RuntimeRoot: product, BrowserReceiptPath: ".cvf/runtime/readiness/reservation.json",
		BrowserCloseReceiptPath: ".cvf/runtime/product/close.json",
		ActionEvidencePath: ".cvf/runtime/product/action.json",
		FinalInventoryPath: ".cvf/runtime/product/inventory.json",
		RawNativeInventoryPath: ".cvf/runtime/product/raw_inventory.json",
		GrantDeadlineUTC: stamp(3600), nativeProjectRoot: project,
	}
	reservation := r139NativeTestIdentity(a)
	reservation["status"] = "RESERVED_NATIVE_PAGE_CONTROL_CAPTURE"
	reservation["browserId"] = "antigravity_native"
	reservation["exclusiveReservation"] = true
	reservation["controlObserved"] = true
	reservation["captureObserved"] = true
	reservation["closeObserved"] = false
	reservation["protocol"] = "http"
	reservation["reservedAtUTC"] = stamp(-15)
	for index, event := range []string{"creation", "control", "capture"} {
		evidence := r139NativeTestIdentity(a)
		evidence["event"] = event
		evidence["atUTC"] = stamp(-30 + index*5)
		rel := ".cvf/runtime/readiness/" + event + ".json"
		reservation[event+"EvidencePath"] = rel
		reservation[event+"EvidenceSha256"] = r139NativeTestWrite(t, filepath.Join(readiness, event+".json"), evidence)
	}
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(readiness, "reservation.json"), reservation)
	return a, reservation, stamp
}

func r139NativeTestTerminal(t *testing.T, a r139Admission, reservation map[string]any, stamp func(int) string) map[string]any {
	t.Helper()
	product := a.RuntimeRoot
	raw := r139NativeTestIdentity(a)
	raw["event"] = "native_inventory"
	raw["atUTC"] = stamp(-4)
	raw["pageIds"] = []string{}
	raw["ownedCount"] = 0
	rawSHA := r139NativeTestWrite(t, filepath.Join(product, "raw_inventory.json"), raw)
	action := r139NativeTestIdentity(a)
	action["event"] = "action_sequence"
	action["atUTC"] = stamp(-8)
	action["actions"] = []map[string]any{
		{"pageId": a.PageID, "action": "control", "atUTC": stamp(-10)},
		{"pageId": a.PageID, "action": "close", "atUTC": stamp(-5)},
	}
	actionSHA := r139NativeTestWrite(t, filepath.Join(product, "action.json"), action)
	inv := r139NativeTestIdentity(a)
	inv["inventoryAtUTC"] = stamp(-4)
	inv["pageIds"] = []string{}
	inv["ownedCount"] = 0
	inv["rawNativeInventoryPath"] = a.RawNativeInventoryPath
	inv["rawNativeInventorySha256"] = rawSHA
	invSHA := r139NativeTestWrite(t, filepath.Join(product, "inventory.json"), inv)
	close := r139NativeTestIdentity(a)
	close["status"] = "CLOSED_EXACT_OWNED_PAGE"
	close["browserId"] = "antigravity_native"
	close["closeObserved"] = true
	close["ownedTabsFinal"] = 0
	close["reservedAtUTC"] = reservation["reservedAtUTC"]
	close["actionAtUTC"] = stamp(-8)
	close["closedAtUTC"] = stamp(-5)
	close["actionEvidencePath"] = a.ActionEvidencePath
	close["actionEvidenceSha256"] = actionSHA
	close["finalInventoryPath"] = a.FinalInventoryPath
	close["finalInventorySha256"] = invSHA
	r139NativeTestWrite(t, filepath.Join(product, "close.json"), close)
	return close
}

func TestR139PureBrowserReservationAdmission(t *testing.T) {
	iab := r139Admission{BrowserSupported: true, BrowserTool: "supported_CUA_IAB"}
	validIAB := []byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0,\"protocol\":\"http\"}")
	if err := r139VerifyBrowserReservation(iab, validIAB); err != nil { t.Fatalf("legacy IAB parity: %v", err) }
	for _, raw := range [][]byte{
		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":false,\"protocol\":\"http\"}"),
		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0.5,\"protocol\":\"http\"}"),
		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0.0,\"protocol\":\"http\"}"),
		[]byte("{\"status\":\"PASS_SUPPORTED_CUA_IAB_CONTROL_CAPTURE_CLOSE\",\"browserId\":\"iab\",\"controlObserved\":true,\"captureObserved\":true,\"closeObserved\":true,\"ownedTabsFinal\":0e0,\"protocol\":\"http\"}"),
	} {
		if err := r139VerifyBrowserReservation(iab, raw); err == nil { t.Fatal("IAB noninteger count accepted") }
	}
	a, reservation, _ := r139NativeTestFixture(t)
	raw, _ := json.Marshal(reservation)
	if err := r139VerifyBrowserReservation(a, raw); err != nil { t.Fatalf("complete reservation rejected: %v", err) }
	tracked := a
	tracked.BrowserReceiptPath = "docs/reviews/probes/reservation.json"
	tracked.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(tracked.BrowserReceiptPath)), reservation)
	if err := r139VerifyBrowserReservation(tracked, raw); err != nil { t.Fatalf("tracked readiness receipt rejected: %v", err) }
	for name, changed := range map[string]any{
		"missing_contract": nil, "wrong_mode": "api", "wrong_type_time": false,
	} {
		copy := map[string]any{}
		for key, value := range reservation { copy[key] = value }
		switch name {
		case "missing_contract": copy["nativeContractId"] = changed
		case "wrong_mode": copy["closeMode"] = changed
		case "wrong_type_time": copy["reservedAtUTC"] = changed
		}
		bad, _ := json.Marshal(copy)
		modified := a
		modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), copy)
		if err := r139VerifyBrowserReservation(modified, bad); err == nil { t.Fatalf("%s accepted", name) }
	}
	for _, field := range []string{"creationEvidencePath", "controlEvidenceSha256", "captureEvidencePath"} {
		copy := map[string]any{}
		for key, value := range reservation { copy[key] = value }
		delete(copy, field)
		bad, _ := json.Marshal(copy)
		modified := a
		modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), copy)
		if err := r139VerifyBrowserReservation(modified, bad); err == nil { t.Fatalf("missing %s accepted", field) }
	}
	for _, field := range []string{"trancheId", "campaignNonce", "pageId", "browserSessionId", "operator"} {
		copy := map[string]any{}
		for key, value := range reservation { copy[key] = value }
		copy[field] = "wrong"
		bad, _ := json.Marshal(copy)
		modified := a
		modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), copy)
		if err := r139VerifyBrowserReservation(modified, bad); err == nil { t.Fatalf("wrong %s accepted", field) }
	}
	r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), reservation)
	badRoot := a
	badRoot.BrowserReadinessRoot = ".cvf/runtime/../escape"
	if err := r139VerifyBrowserReservation(badRoot, raw); err == nil { t.Fatal("root traversal accepted") }
}

func TestR139PureBrowserTerminalClose(t *testing.T) {
	a, reservation, stamp := r139NativeTestFixture(t)
	a.BrowserReceiptPath = "docs/reviews/probes/reservation.json"
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(a.nativeProjectRoot, filepath.FromSlash(a.BrowserReceiptPath)), reservation)
	rawReservation, _ := json.Marshal(reservation)
	if err := r139VerifyBrowserReservation(a, rawReservation); err != nil { t.Fatal(err) }
	close := r139NativeTestTerminal(t, a, reservation, stamp)
	raw, _ := json.Marshal(close)
	if err := r139VerifyBrowserTerminalClose(a, raw); err != nil { t.Fatalf("complete terminal rejected: %v", err) }
	for _, spelling := range []string{"0.0", "0e0"} {
		fractional := bytes.Replace(raw, []byte("\"ownedTabsFinal\":0"), []byte("\"ownedTabsFinal\":"+spelling), 1)
		if bytes.Equal(fractional, raw) { t.Fatal("terminal numeric fixture replacement missing") }
		if err := os.WriteFile(filepath.Join(a.RuntimeRoot, "close.json"), fractional, 0644); err != nil { t.Fatal(err) }
		if err := r139VerifyBrowserTerminalClose(a, fractional); err == nil { t.Fatalf("terminal count %s accepted", spelling) }
	}
	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), close)
	for name, changed := range map[string]any{
		"missing_action_time": nil, "future_close": stamp(60),
		"wrong_inventory_sha": strings.Repeat("x", 64), "wrong_mode": "api",
		"false_count": false,
	} {
		copy := map[string]any{}
		for key, value := range close { copy[key] = value }
		switch name {
		case "missing_action_time": copy["actionAtUTC"] = changed
		case "future_close": copy["closedAtUTC"] = changed
		case "wrong_inventory_sha": copy["finalInventorySha256"] = changed
		case "wrong_mode": copy["closeMode"] = changed
		case "false_count": copy["ownedTabsFinal"] = changed
		}
		bad, _ := json.Marshal(copy)
		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), copy)
		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("%s accepted", name) }
	}
	for _, field := range []string{"trancheId", "campaignNonce", "pageId", "browserSessionId", "operator", "nativeContractId"} {
		copy := map[string]any{}
		for key, value := range close { copy[key] = value }
		copy[field] = "wrong"
		bad, _ := json.Marshal(copy)
		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), copy)
		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("wrong terminal %s accepted", field) }
	}
	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), close)
	inventoryPath := filepath.Join(a.RuntimeRoot, "inventory.json")
	inventory, err := r139NativeObject(func() []byte { raw, readErr := os.ReadFile(inventoryPath); if readErr != nil { t.Fatal(readErr) }; return raw }())
	if err != nil { t.Fatal(err) }
	for _, spelling := range []string{"0.0", "0e0"} {
		original, readErr := os.ReadFile(inventoryPath); if readErr != nil { t.Fatal(readErr) }
		fractional := bytes.Replace(original, []byte("\"ownedCount\":0"), []byte("\"ownedCount\":"+spelling), 1)
		if bytes.Equal(fractional, original) { t.Fatal("inventory numeric fixture replacement missing") }
		if err := os.WriteFile(inventoryPath, fractional, 0644); err != nil { t.Fatal(err) }
		sum := sha256.Sum256(fractional)
		mutated := map[string]any{}
		for key, value := range close { mutated[key] = value }
		mutated["finalInventorySha256"] = hex.EncodeToString(sum[:])
		bad, _ := json.Marshal(mutated)
		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), mutated)
		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("inventory count %s accepted", spelling) }
		if err := os.WriteFile(inventoryPath, original, 0644); err != nil { t.Fatal(err) }
	}
	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), close)
	for name, changed := range map[string]any{"missing_pages": nil, "page_open": []string{a.PageID}, "wrong_count": false, "missing_time": nil} {
		copy := map[string]any{}
		for key, value := range inventory { copy[key] = value }
		switch name {
		case "missing_pages", "page_open": copy["pageIds"] = changed
		case "wrong_count": copy["ownedCount"] = changed
		case "missing_time": copy["inventoryAtUTC"] = changed
		}
		sha := r139NativeTestWrite(t, inventoryPath, copy)
		mutated := map[string]any{}
		for key, value := range close { mutated[key] = value }
		mutated["finalInventorySha256"] = sha
		bad, _ := json.Marshal(mutated)
		r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), mutated)
		if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatalf("%s inventory accepted", name) }
	}
	r139NativeTestWrite(t, inventoryPath, inventory)
	actionPath := filepath.Join(a.RuntimeRoot, "action.json")
	actionRaw, err := os.ReadFile(actionPath); if err != nil { t.Fatal(err) }
	action, err := r139NativeObject(actionRaw); if err != nil { t.Fatal(err) }
	action["actions"] = append(action["actions"].([]any), map[string]any{"pageId": a.PageID, "action": "control", "atUTC": stamp(-4)})
	sha := r139NativeTestWrite(t, actionPath, action)
	mutated := map[string]any{}
	for key, value := range close { mutated[key] = value }
	mutated["actionEvidenceSha256"] = sha
	bad, _ := json.Marshal(mutated)
	r139NativeTestWrite(t, filepath.Join(a.RuntimeRoot, "close.json"), mutated)
	if err := r139VerifyBrowserTerminalClose(a, bad); err == nil { t.Fatal("post-close action accepted") }
}

func TestR139PurePageTargetV1(t *testing.T) {
	root := t.TempDir()
	ready := filepath.Join(root, ".cvf", "runtime", "readiness")
	product := filepath.Join(root, ".cvf", "runtime", "product")
	if err := os.MkdirAll(ready, 0755); err != nil { t.Fatal(err) }
	now := time.Now().UTC().Truncate(time.Second)
	stamp := func(delta int) string { return now.Add(time.Duration(delta) * time.Second).Format("2006-01-02T15:04:05Z") }
	a := r139Admission{NativeContractID: "r139-page-target-v1", TrancheID: "CCMAI-RUNTIME-139",
		CampaignNonce: "nonce-456", PageID: "page-123", OwnedScopeID: "owned-scope-123",
		ScopeOrigin: "caller_owned_explicit_page_target", OwnedPageIDs: []string{"page-123"},
		ProofContract: "native_page_target_v1", Operator: "Owner in this conversation",
		FallbackCloseOperator: "Owner in this conversation", CloseMode: "operator_fallback",
		BrowserTool: "antigravity_page_target", BrowserSupported: true,
		BrowserReadinessRoot: ".cvf/runtime/readiness", RuntimeRoot: product,
		BrowserReceiptPath: ".cvf/runtime/readiness/reservation.json",
		BrowserCloseReceiptPath: ".cvf/runtime/product/close.json",
		ActionEvidencePath: ".cvf/runtime/product/action.json",
		FinalInventoryPath: ".cvf/runtime/product/inventory.json",
		RawNativeInventoryPath: ".cvf/runtime/product/raw_inventory.json",
		BrowserDoneSignalPath: ".cvf/runtime/product/browser_done.signal",
		GrantDeadlineUTC: stamp(3600), nativeProjectRoot: root}
	identity := map[string]any{"nativeContractId": a.NativeContractID, "trancheId": a.TrancheID,
		"campaignNonce": a.CampaignNonce, "pageId": a.PageID, "ownedScopeId": a.OwnedScopeID,
		"operator": a.Operator, "closeMode": a.CloseMode, "proofContract": a.ProofContract,
		"scopeOrigin": a.ScopeOrigin, "ownedPageIds": []string{a.PageID}}
	clone := func(source map[string]any) map[string]any { copied := map[string]any{}; for k, v := range source { copied[k] = v }; return copied }
	origin := map[string]any{"nativeToolCallRef": "step-10", "rawOriginalToolReference": "private/native/step-10",
		"rawOriginalToolSha256": strings.Repeat("b", 64), "timeBasis": "original_native_tool_return"}
	createURL := "https://example.com/#r139_nonce=nonce-456&phase=created"
	controlURL := "https://example.com/#r139_nonce=nonce-456&phase=controlled"
	prefix := clone(identity)
	for k, v := range map[string]any{"status": "RESERVED_NATIVE_PAGE_TARGET_CONTROL_CAPTURE_HANDOFF",
		"browserId": "antigravity_native", "protocol": "https", "closeObserved": false,
		"reservedAtUTC": stamp(-25)} { prefix[k] = v }
	events := map[string]map[string]any{}
	for name, offset := range map[string]int{"baselineInventory": -60, "creation": -50, "creationLedger": -50,
		"control": -45, "capture": -40, "handoff": -30} {
		row := clone(identity)
		for k, v := range origin { row[k] = v }
		row["nativeTaskRunRef"] = "run-g0"; row["atUTC"] = stamp(offset)
		row["event"] = name
		if name == "baselineInventory" { row["event"] = "baseline_inventory"; row["pageIds"] = []string{"host-protected"}; row["inventoryScope"] = "metadata_only" }
		if name == "creationLedger" { row["event"] = "creation_ledger"; row["newPageIds"] = []string{a.PageID} }
		if name == "creation" { row["pageIdToReplace"] = "new_page"; row["observedPageId"] = a.PageID; row["observedUrl"] = createURL }
		if name == "control" { row["pageIdToReplace"] = a.PageID; row["observedUrl"] = controlURL }
		if name == "capture" { row["observedPageId"] = a.PageID; row["observedUrl"] = controlURL; row["captureSha256"] = strings.Repeat("c", 64) }
		if name == "handoff" { row["nativeTaskRunRef"] = "run-handoff"; row["observedPageId"] = a.PageID; row["observedUrl"] = controlURL; row["observedNonce"] = a.CampaignNonce }
		events[name] = row
		pathKey, hashKey := name+"EvidencePath", name+"EvidenceSha256"
		if name == "baselineInventory" || name == "creationLedger" { pathKey, hashKey = name+"Path", name+"Sha256" }
		prefix[pathKey] = ".cvf/runtime/readiness/" + name + ".json"
		prefix[hashKey] = r139NativeTestWrite(t, filepath.Join(ready, name+".json"), row)
	}
	events["creationLedger"]["baselineInventorySha256"] = prefix["baselineInventorySha256"]
	events["creationLedger"]["creationEvidenceSha256"] = prefix["creationEvidenceSha256"]
	prefix["creationLedgerSha256"] = r139NativeTestWrite(t, filepath.Join(ready, "creationLedger.json"), events["creationLedger"])
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(ready, "reservation.json"), prefix)
	prefixRaw, _ := json.Marshal(prefix)
	if err := r139VerifyBrowserReservation(a, prefixRaw); err != nil { t.Fatalf("page target prefix: %v", err) }
	bad := clone(prefix); bad["reservedAtUTC"] = stamp(-55)
	badRaw, _ := json.Marshal(bad); modified := a
	modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(ready, "reservation.json"), bad)
	if err := r139VerifyBrowserReservation(modified, badRaw); err == nil { t.Fatal("early native seal accepted") }
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(ready, "reservation.json"), prefix)
	badHandoff := clone(events["handoff"]); badHandoff["nativeTaskRunRef"] = "run-g0"
	bad = clone(prefix); bad["handoffEvidenceSha256"] = r139NativeTestWrite(t, filepath.Join(ready, "handoff.json"), badHandoff)
	modified = a; modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(ready, "reservation.json"), bad)
	badRaw, _ = json.Marshal(bad)
	if err := r139VerifyBrowserReservation(modified, badRaw); err == nil { t.Fatal("same-task handoff accepted") }
	prefix["handoffEvidenceSha256"] = r139NativeTestWrite(t, filepath.Join(ready, "handoff.json"), events["handoff"])
	badLedger := clone(events["creationLedger"]); badLedger["atUTC"] = stamp(-51)
	bad = clone(prefix); bad["creationLedgerSha256"] = r139NativeTestWrite(t, filepath.Join(ready, "creationLedger.json"), badLedger)
	modified = a; modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(ready, "reservation.json"), bad)
	badRaw, _ = json.Marshal(bad)
	if err := r139VerifyBrowserReservation(modified, badRaw); err == nil { t.Fatal("ledger before creation accepted") }
	prefix["creationLedgerSha256"] = r139NativeTestWrite(t, filepath.Join(ready, "creationLedger.json"), events["creationLedger"])
	badCapture := clone(events["capture"]); badCapture["observedUrl"] = "https://example.com/#wrong"
	bad = clone(prefix); bad["captureEvidenceSha256"] = r139NativeTestWrite(t, filepath.Join(ready, "capture.json"), badCapture)
	modified = a; modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(ready, "reservation.json"), bad)
	badRaw, _ = json.Marshal(bad)
	if err := r139VerifyBrowserReservation(modified, badRaw); err == nil { t.Fatal("capture on wrong URL accepted") }
	prefix["captureEvidenceSha256"] = r139NativeTestWrite(t, filepath.Join(ready, "capture.json"), events["capture"])
	a.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(ready, "reservation.json"), prefix)
	prefixRaw, _ = json.Marshal(prefix)
	outside := filepath.Join(root, ".cvf", "runtime", "outside")
	if err := os.MkdirAll(outside, 0755); err != nil { t.Fatal(err) }
	modified = a; modified.BrowserReceiptPath = ".cvf/runtime/outside/reservation.json"
	modified.BrowserReceiptSHA256 = r139NativeTestWrite(t, filepath.Join(outside, "reservation.json"), prefix)
	if err := r139VerifyBrowserReservation(modified, prefixRaw); err == nil { t.Fatal("valid-hash prefix outside readiness accepted") }
	if err := os.MkdirAll(product, 0755); err != nil { t.Fatal(err) }
	if err := r139VerifyBrowserReservation(a, prefixRaw); err == nil { t.Fatal("preexisting product root accepted") }
	a.SourceCommit = strings.Repeat("a", 40)
	t.Setenv("R139_RUN", "1")
	t.Setenv("R139_RUNTIME_ROOT", product)
	t.Setenv("R139_SOURCE_COMMIT", a.SourceCommit)
	t.Setenv("R139_RUNTIME_ADMISSION_SHA256", strings.Repeat("b", 64))
	intent := map[string]any{"admissionSha256": strings.Repeat("c", 64), "sourceCommit": a.SourceCommit,
		"startedAt": float64(time.Now().Unix()), "deadlineSeconds": 1200}
	r139NativeTestWrite(t, filepath.Join(product, "run_intent.json"), intent)
	if err := r139VerifyBrowserReservation(a, prefixRaw); err == nil { t.Fatal("wrong supervisor intent hash accepted") }
	intent["admissionSha256"] = strings.Repeat("b", 64)
	r139NativeTestWrite(t, filepath.Join(product, "run_intent.json"), intent)
	if err := r139VerifyBrowserReservation(a, prefixRaw); err != nil { t.Fatalf("actual supervisor-owned product root rejected: %v", err) }
	terminalOrigin := clone(origin); terminalOrigin["nativeTaskRunRef"] = "run-final"
	raw := clone(identity)
	for k, v := range terminalOrigin { raw[k] = v }
	raw["event"] = "native_inventory"; raw["atUTC"] = stamp(-5); raw["pageIds"] = []string{"host-protected"}; raw["ownedCount"] = 0; raw["inventoryScope"] = "metadata_only"
	rawSHA := r139NativeTestWrite(t, filepath.Join(product, "raw_inventory.json"), raw)
	actions := clone(identity)
	for k, v := range terminalOrigin { actions[k] = v }
	actions["event"] = "action_sequence"; actions["atUTC"] = stamp(-20)
	actions["actions"] = []map[string]any{{"pageId": a.PageID, "action": "control", "atUTC": stamp(-20), "rawOriginalToolReference": "private/control", "rawOriginalToolSha256": strings.Repeat("e", 64)},
		{"pageId": a.PageID, "action": "owner_close_report", "atUTC": stamp(-8), "reportSource": "owner_in_conversation", "ownerReportReferenceSha256": strings.Repeat("a", 64)}}
	actionSHA := r139NativeTestWrite(t, filepath.Join(product, "action.json"), actions)
	inv := clone(identity)
	for k, v := range terminalOrigin { inv[k] = v }
	inv["event"] = "final_inventory"; inv["atUTC"] = stamp(-5); inv["pageIds"] = []string{"host-protected"}; inv["ownedCount"] = 0; inv["inventoryScope"] = "metadata_only"
	inv["rawNativeInventoryPath"] = a.RawNativeInventoryPath; inv["rawNativeInventorySha256"] = rawSHA
	invSHA := r139NativeTestWrite(t, filepath.Join(product, "inventory.json"), inv)
	close := clone(identity)
	for k, v := range map[string]any{"status": "CLOSED_EXACT_OWNED_PAGE_TARGET", "browserId": "antigravity_native",
		"closeObserved": true, "ownedTabsFinal": 0, "closureTimeBasis": "native_final_metadata_observation",
		"reservedAtUTC": prefix["reservedAtUTC"], "actionAtUTC": stamp(-20),
		"operatorReportReceivedAtUTC": stamp(-8), "closureObservedAtUTC": stamp(-5),
		"ownerReportReferenceSha256": strings.Repeat("a", 64), "actionEvidencePath": a.ActionEvidencePath,
		"actionEvidenceSha256": actionSHA, "finalInventoryPath": a.FinalInventoryPath,
		"finalInventorySha256": invSHA} { close[k] = v }
	closeSHA := r139NativeTestWrite(t, filepath.Join(product, "close.json"), close)
	marker := map[string]any{"status": "TERMINAL_EVIDENCE_WRITTEN", "proofContract": a.ProofContract,
		"campaignNonce": a.CampaignNonce, "pageId": a.PageID, "ownedScopeId": a.OwnedScopeID,
		"atUTC": stamp(-4), "evidenceSha256": map[string]any{"browserCloseReceiptPath": closeSHA,
			"actionEvidencePath": actionSHA, "finalInventoryPath": invSHA, "rawNativeInventoryPath": rawSHA}}
	r139NativeTestWrite(t, filepath.Join(product, "browser_done.signal"), marker)
	closeRaw, _ := json.Marshal(close)
	if err := r139VerifyBrowserTerminalClose(a, closeRaw); err != nil { t.Fatalf("page-target terminal: %v", err) }
	marker["ownedScopeId"] = "wrong"
	r139NativeTestWrite(t, filepath.Join(product, "browser_done.signal"), marker)
	if err := r139VerifyBrowserTerminalClose(a, closeRaw); err == nil { t.Fatal("wrong done marker accepted") }
}

func r139CuaGrantShape(a r139Admission, amendment, inherited, observation map[string]any) error {
	wantAmendment := r139AuthorityRef{Path: "CVF_SESSION/authority/CCMAI-RUNTIME-139-INTERNAL-CUA-SOURCE-ADMISSION.json", SHA256: "0ee21ae60ea9f3248b36af58ed003892f3e4c1f088c687f8683f9ae4f37ddd9e"}
	wantInherited := r139AuthorityRef{Path: "CVF_SESSION/authority/CCMAI-RUNTIME-139-SOURCE-AND-CONDITIONAL-REENTRY-ADMISSION.json", SHA256: "0ef818bfa4a60b9bd1b01335996989ca4c00d63f11141d9074dd3c26befb5b4f"}
	wantObservation := r139AuthorityRef{Path: "CVF_SESSION/authority/CCMAI-RUNTIME-139-CUA-OBSERVATION-CAPS-ADMISSION.json", SHA256: "fa7f2f344c75b6900f1fb65d74c9327e9e53d1957714226d880fa5a30eab9491"}
	if a.OwnerCuaAmendmentRef != wantAmendment || a.InheritedOwnerEnvelopeRef != wantInherited ||
		a.ObservationCapsRef != wantObservation { return errors.New("CUA owner grant refs invalid") }
	amendParent := map[string]any{"path": wantInherited.Path, "sha256": wantInherited.SHA256}
	if amendment["ownerApproved"] != true || amendment["sourceGranted"] != true ||
		!reflect.DeepEqual(amendment["previousOwnerEnvelope"], amendParent) ||
		inherited["ownerApproved"] != true || inherited["freshCampaignCap"] != float64(1) ||
		inherited["providerPhysicalCap"] != float64(1) { return errors.New("CUA owner lineage invalid") }
	contract, ok := amendment["browserContractProposed"].(map[string]any)
	if !ok || contract["proofContract"] != "cua_page_target_v1" { return errors.New("CUA owner contract invalid") }
	obsParent := map[string]any{"path": wantAmendment.Path, "sha256": wantAmendment.SHA256}
	if observation["ownerApproved"] != true || observation["kind"] != "OWNER_APPROVED_R139_CUA_OBSERVATION_CAP_AMENDMENT" ||
		!reflect.DeepEqual(observation["parentCuaAmendment"], obsParent) ||
		!reflect.DeepEqual(observation["inheritedOwnerEnvelope"], amendParent) { return errors.New("CUA observation authority absent") }
	caps, ok := observation["proposedReadonlyChanges"].(map[string]any)
	if !ok { return errors.New("CUA observation caps absent") }
	for key, expected := range map[string]float64{"globalMetadataEnumerationMaximum": 3,
		"baselineMetadataEnumeration": 1, "handoffMetadataEnumeration": 1, "finalMetadataEnumeration": 1,
		"g2StatusReadStartsMaximum": 16, "automaticPostSendStatusPollStartsMaximum": 12,
		"automaticPollSpacingSecondsMinimum": 5, "automaticPollLifetimeSecondsMaximum": 60,
		"eventDrivenStatusReadsMaximum": 4} {
		if caps[key] != expected { return fmt.Errorf("CUA observation %s mismatch", key) }
	}
	categories := []any{"initial_writer_load", "publication_version_watch_reload", "one_deliberate_writer_reload", "reader_load"}
	if !reflect.DeepEqual(caps["eventDrivenCategories"], categories) ||
		caps["spacingAndLifetimeScope"] != "automatic post-send polling only; event-driven loads recorded separately" ||
		caps["noExtraGlobalInventory"] != true || caps["noUnrelatedTabContentCaptureOrControl"] != true ||
		caps["rawMetadataPrivateEvidenceIdsOrDigestsOnly"] != true { return errors.New("CUA observation privacy scope mismatch") }
	return nil
}

func TestR139PureCuaGrantChainShape(t *testing.T) {
	a := r139Admission{OwnerCuaAmendmentRef: r139AuthorityRef{Path: "CVF_SESSION/authority/CCMAI-RUNTIME-139-INTERNAL-CUA-SOURCE-ADMISSION.json", SHA256: "0ee21ae60ea9f3248b36af58ed003892f3e4c1f088c687f8683f9ae4f37ddd9e"},
		InheritedOwnerEnvelopeRef: r139AuthorityRef{Path: "CVF_SESSION/authority/CCMAI-RUNTIME-139-SOURCE-AND-CONDITIONAL-REENTRY-ADMISSION.json", SHA256: "0ef818bfa4a60b9bd1b01335996989ca4c00d63f11141d9074dd3c26befb5b4f"},
		ObservationCapsRef: r139AuthorityRef{Path: "CVF_SESSION/authority/CCMAI-RUNTIME-139-CUA-OBSERVATION-CAPS-ADMISSION.json", SHA256: "fa7f2f344c75b6900f1fb65d74c9327e9e53d1957714226d880fa5a30eab9491"}}
	oldRef := map[string]any{"path": a.InheritedOwnerEnvelopeRef.Path, "sha256": a.InheritedOwnerEnvelopeRef.SHA256}
	newRef := map[string]any{"path": a.OwnerCuaAmendmentRef.Path, "sha256": a.OwnerCuaAmendmentRef.SHA256}
	amendment := map[string]any{"ownerApproved": true, "sourceGranted": true, "previousOwnerEnvelope": oldRef,
		"browserContractProposed": map[string]any{"proofContract": "cua_page_target_v1"}}
	inherited := map[string]any{"ownerApproved": true, "freshCampaignCap": float64(1), "providerPhysicalCap": float64(1)}
	observation := map[string]any{"ownerApproved": true, "kind": "OWNER_APPROVED_R139_CUA_OBSERVATION_CAP_AMENDMENT",
		"parentCuaAmendment": newRef, "inheritedOwnerEnvelope": oldRef,
		"proposedReadonlyChanges": map[string]any{"globalMetadataEnumerationMaximum": float64(3),
			"baselineMetadataEnumeration": float64(1), "handoffMetadataEnumeration": float64(1),
			"finalMetadataEnumeration": float64(1), "g2StatusReadStartsMaximum": float64(16),
			"automaticPostSendStatusPollStartsMaximum": float64(12),
			"automaticPollSpacingSecondsMinimum": float64(5),
			"automaticPollLifetimeSecondsMaximum": float64(60), "eventDrivenStatusReadsMaximum": float64(4),
			"eventDrivenCategories": []any{"initial_writer_load", "publication_version_watch_reload", "one_deliberate_writer_reload", "reader_load"},
			"spacingAndLifetimeScope": "automatic post-send polling only; event-driven loads recorded separately",
			"noExtraGlobalInventory": true, "noUnrelatedTabContentCaptureOrControl": true,
			"rawMetadataPrivateEvidenceIdsOrDigestsOnly": true}}
	if err := r139CuaGrantShape(a, amendment, inherited, observation); err != nil { t.Fatalf("valid CUA grant chain rejected: %v", err) }
	observation["ownerApproved"] = false
	if err := r139CuaGrantShape(a, amendment, inherited, observation); err == nil { t.Fatal("unapproved CUA observation record accepted") }
	observation["ownerApproved"] = true
	observation["parentCuaAmendment"] = oldRef
	if err := r139CuaGrantShape(a, amendment, inherited, observation); err == nil { t.Fatal("mismatched CUA parent accepted") }
	observation["parentCuaAmendment"] = newRef
	caps := observation["proposedReadonlyChanges"].(map[string]any)
	caps["g2StatusReadStartsMaximum"] = float64(12)
	if err := r139CuaGrantShape(a, amendment, inherited, observation); err == nil { t.Fatal("old status12 cap accepted") }
	caps["g2StatusReadStartsMaximum"] = float64(16)
	caps["handoffMetadataEnumeration"] = float64(0)
	if err := r139CuaGrantShape(a, amendment, inherited, observation); err == nil { t.Fatal("missing handoff metadata cap accepted") }
	caps["handoffMetadataEnumeration"] = float64(1)
	a.ObservationCapsRef.SHA256 = strings.Repeat("c", 64)
	if err := r139CuaGrantShape(a, amendment, inherited, observation); err == nil { t.Fatal("wrong CUA observation SHA accepted") }
	a.ObservationCapsRef.SHA256 = "fa7f2f344c75b6900f1fb65d74c9327e9e53d1957714226d880fa5a30eab9491"
	a.OwnerCuaAmendmentRef.SHA256 = strings.Repeat("d", 64)
	if err := r139CuaGrantShape(a, amendment, inherited, observation); err == nil { t.Fatal("wrong CUA grant SHA accepted") }
}

func r139Admit(t *testing.T) (r139Admission, string) {
	t.Helper()
	if os.Getenv("R139_RUN") != "1" {
		t.Skip("NOT_RUN: R139 runtime admission absent")
	}
	path := os.Getenv("R139_RUNTIME_ADMISSION")
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		t.Fatal("R139 admission path invalid")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	pathRoot, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(pathRoot, "..") || !regexp.MustCompile(`^CVF_SESSION[/\\]authority[/\\]CCMAI-RUNTIME-139-[A-Za-z0-9_-]+-ADMISSION\.json$`).MatchString(pathRoot) {
		t.Fatal("R139 admission outside authority")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		t.Fatal("R139 admission symlink")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("R139 admission missing")
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != os.Getenv("R139_RUNTIME_ADMISSION_SHA256") {
		t.Fatal("R139 admission hash mismatch")
	}
	var a r139Admission
	if json.Unmarshal(data, &a) != nil || a.TrancheID != "CCMAI-RUNTIME-139" || a.Kind != "OWNER_APPROVED_RUNTIME_EFFECT_ADMISSION" || !a.OwnerApproved || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(a.SourceCommit) || a.SourceCommit != os.Getenv("R139_SOURCE_COMMIT") || a.NativeExecutePOST != 8 || a.BrowserExecutePOST != 1 || a.GrantCap != 3 || a.ProviderCap != 1 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(a.BrowserReceiptSHA256) || a.GrantDeadlineUTC == "" || os.Getenv("R139_GRANT_DEADLINE_UTC") != a.GrantDeadlineUTC || a.Provider != "alibaba_maas" || a.Model != "qwen3.7-flash" || a.Endpoint == "" || os.Getenv("R139_ALIBABA_ENDPOINT") != a.Endpoint || a.CompletionTokens != 768 || a.MaxRequestBytes != 6000 || a.Thinking || !regexp.MustCompile(`^ccma_r139_[a-z0-9_-]{1,64}$`).MatchString(a.SyntheticNamespace) || a.IssuerRole != "owner" || a.ActorRole != "writer" || a.ConfigRef != "r139-owned-config" || a.PromptVersion != "r139-synthetic-v1" {
		t.Fatal("R139 effect authority incomplete")
	}
	if a.BrowserTool == "supported_CUA_page_target" {
		loadCommitted := func(ref r139AuthorityRef) map[string]any {
			if !regexp.MustCompile(`^CVF_SESSION/authority/CCMAI-RUNTIME-139-[A-Za-z0-9_-]+-ADMISSION\.json$`).MatchString(ref.Path) ||
				!regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(ref.SHA256) { t.Fatal("CUA authority path/hash invalid") }
			body, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref.Path)))
			sum := sha256.Sum256(body)
			if readErr != nil || hex.EncodeToString(sum[:]) != ref.SHA256 { t.Fatal("CUA authority file hash mismatch") }
			show := exec.Command("git", "show", "HEAD:"+ref.Path); show.Dir = root
			committed, showErr := show.Output()
			if showErr != nil || !bytes.Equal(bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n")), bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))) { t.Fatal("CUA authority not committed") }
			var parsed map[string]any
			if json.Unmarshal(body, &parsed) != nil { t.Fatal("CUA authority JSON invalid") }
			return parsed
		}
		amendment := loadCommitted(a.OwnerCuaAmendmentRef)
		inherited := loadCommitted(a.InheritedOwnerEnvelopeRef)
		observation := loadCommitted(a.ObservationCapsRef)
		if err := r139CuaGrantShape(a, amendment, inherited, observation); err != nil { t.Fatal(err) }
	}
	endpointDigest := sha256.Sum256([]byte(a.Endpoint))
	if hex.EncodeToString(endpointDigest[:]) != "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f" {
		t.Fatal("R139 admitted endpoint mismatch")
	}
	promptDigest := sha256.Sum256([]byte(r139PromptText))
	if a.PromptSHA256 != hex.EncodeToString(promptDigest[:]) {
		t.Fatal("R139 admitted prompt mismatch")
	}
	deadline, deadlineErr := time.Parse(time.RFC3339, a.GrantDeadlineUTC)
	now := time.Now().UTC()
	if deadlineErr != nil || !strings.HasSuffix(a.GrantDeadlineUTC, "Z") || deadline.Format(time.RFC3339) != a.GrantDeadlineUTC || deadline.Before(now.Add(time.Minute)) || deadline.After(now.Add(24*time.Hour)) {
		t.Fatal("R139 owner grant deadline invalid")
	}
	show := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(pathRoot))
	show.Dir = root
	committed, err := show.Output()
	if err != nil || !bytes.Equal(bytes.ReplaceAll(committed, []byte("\r\n"), []byte("\n")), bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))) {
		t.Fatal("R139 runtime admission uncommitted")
	}
	ancestor := exec.Command("git", "merge-base", "--is-ancestor", a.SourceCommit, "HEAD")
	ancestor.Dir = root
	if err := ancestor.Run(); err != nil {
		t.Fatal("R139 source commit not ancestor")
	}
	dirty := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
	dirty.Dir = root
	dirtyRaw, dirtyErr := dirty.Output()
	if dirtyErr != nil || len(bytes.TrimSpace(dirtyRaw)) != 0 {
		t.Fatal("R139 source worktree dirty")
	}
	required := []string{"backend/api/handlers/intervention_execution_browser_test.go", "scripts/r139_execution_supervisor.py", "scripts/tests/test_r139_execution_supervisor.py", "scripts/r139_vite_config.mjs.template", "backend/go.mod", "backend/go.sum", "frontend/package.json", "frontend/package-lock.json", "frontend/vite.config.ts"}
	for _, relative := range required {
		if a.SourceSHA256[relative] == "" {
			t.Fatal("R139 required source pin absent")
		}
	}
	for relative, expected := range a.SourceSHA256 {
		if filepath.IsAbs(relative) || strings.Contains(relative, "\\") || strings.HasPrefix(relative, "../") || strings.Contains(relative, "/../") {
			t.Fatal("R139 source pin path invalid")
		}
		raw, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		sum := sha256.Sum256(raw)
		if readErr != nil || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(expected) || hex.EncodeToString(sum[:]) != expected {
			t.Fatal("R139 source pin mismatch")
		}
	}
	for _, tree := range []struct{ name, expected string }{{"backend", a.BackendTree}, {"frontend", a.FrontendTree}} {
		original := exec.Command("git", "rev-parse", a.SourceCommit+":"+tree.name)
		original.Dir = root
		current := exec.Command("git", "rev-parse", "HEAD:"+tree.name)
		current.Dir = root
		originalRaw, originalErr := original.Output()
		currentRaw, currentErr := current.Output()
		if originalErr != nil || currentErr != nil || !bytes.Equal(originalRaw, currentRaw) || strings.TrimSpace(string(currentRaw)) != tree.expected {
			t.Fatal("R139 product source tree changed")
		}
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	selfRaw, err := os.ReadFile(self)
	selfSum := sha256.Sum256(selfRaw)
	if err != nil || hex.EncodeToString(selfSum[:]) != a.GoTestBinarySHA256 {
		t.Fatal("R139 binary pin mismatch")
	}
	browserPath := filepath.Join(root, filepath.FromSlash(a.BrowserReceiptPath))
	browserResolved, err := filepath.EvalSymlinks(browserPath)
	browserRaw, readErr := os.ReadFile(browserPath)
	browserSum := sha256.Sum256(browserRaw)
	if err != nil || readErr != nil || browserResolved != browserPath || hex.EncodeToString(browserSum[:]) != a.BrowserReceiptSHA256 {
		t.Fatal("R139 browser proof receipt mismatch")
	}
	if err := r139VerifyBrowserReservation(a, browserRaw); err != nil {
		t.Fatalf("R139 supported browser proof absent: %v", err)
	}
	if !filepath.IsAbs(a.RuntimeRoot) || filepath.Clean(a.RuntimeRoot) != a.RuntimeRoot || os.Getenv("R139_RUNTIME_ROOT") != a.RuntimeRoot {
		t.Fatal("R139 runtime evidence root mismatch")
	}
	runtimeRel, err := filepath.Rel(filepath.Join(root, ".cvf", "runtime"), a.RuntimeRoot)
	if err != nil || runtimeRel == "." || strings.HasPrefix(runtimeRel, "..") {
		t.Fatal("R139 runtime root outside ignored area")
	}
	expectedEvidence := filepath.Join(filepath.Dir(root), ".ccmai_local_evidence", "r139_execution_proof_20261023", "fixture_r4")
	if a.ExternalEvidenceDir != expectedEvidence || os.Getenv("R139_EVIDENCE_DIR") != expectedEvidence {
		t.Fatal("R139 external evidence location mismatch")
	}
	evidenceResolved, err := filepath.EvalSymlinks(expectedEvidence)
	if err != nil || evidenceResolved != expectedEvidence {
		t.Fatal("R139 external evidence symlink")
	}
	entries, err := os.ReadDir(expectedEvidence)
	if err != nil || len(entries) != 1 || entries[0].Name() != "fixture_identity.json" {
		t.Fatal("R139 external evidence is not fresh identity-only")
	}
	if engine.WaitDataExecutor != nil {
		t.Fatal("R139 inherited executor")
	}
	if os.Getenv("R101_MYSQL_DSN") != "" || os.Getenv("R102_MYSQL_DSN") != "" {
		t.Fatal("R139 inherited DSN")
	}
	identityPath := os.Getenv("R139_FIXTURE_IDENTITY")
	if identityPath != filepath.Join(expectedEvidence, "fixture_identity.json") {
		t.Fatal("R139 identity not external")
	}
	identityResolved, err := filepath.EvalSymlinks(identityPath)
	if err != nil || identityResolved != identityPath || !filepath.IsAbs(identityPath) {
		t.Fatal("R139 identity path invalid")
	}
	identityRaw, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatal("R139 fixture identity unavailable")
	}
	var identity struct {
		TrancheID string `json:"trancheId"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Database  string `json:"database"`
		Name      string `json:"name"`
		Nonce     string `json:"nonce"`
	}
	if json.Unmarshal(identityRaw, &identity) != nil || identity.TrancheID != a.TrancheID || identity.Name != a.FixtureName || identity.Host != "127.0.0.1" || identity.Port < 1024 || identity.Port > 65535 || !regexp.MustCompile(`^ccma_r139_[a-z0-9_]{1,32}$`).MatchString(identity.Database) || !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(identity.Nonce) {
		t.Fatal("R139 fixture identity mismatch")
	}
	dsn := os.Getenv("R139_MYSQL_DSN")
	parsed, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || parsed == nil || parsed.Net != "tcp" || parsed.Addr != fmt.Sprintf("127.0.0.1:%d", identity.Port) || parsed.DBName != identity.Database {
		t.Fatal("R139 DSN identity mismatch")
	}
	// Supervisor independently proves the committed admission and Docker owner.
	// This test repeats the exact nonce table check before compatibility helpers.
	probe, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("R139 nonce probe open")
	}
	defer probe.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var count int
	var nonce string
	if err := probe.QueryRowContext(ctx, "SELECT COUNT(*), MIN(nonce) FROM ccma_r139_fixture_owner").Scan(&count, &nonce); err != nil || count != 1 || nonce != identity.Nonce {
		t.Fatal("R139 nonce table does not prove ownership")
	}
	return a, dsn
}

type r139Trap struct{ calls atomic.Int32 }

func (p *r139Trap) AnalyzeChatOnce(context.Context, string, string, int64) (ai.AIResponse, error) {
	p.calls.Add(1)
	return ai.AIResponse{}, errors.New("r139_native_call_trap_no_ai_output")
}

const r139PromptText = "Analyze only fabricated Vietnamese complaint; return the required JSON."

func r139Service(transport engine.InterventionTransport, admission r139Admission) *engine.InterventionService {
	return &engine.InterventionService{Transport: transport, Provider: admission.Provider, Model: admission.Model, ConfigRef: admission.ConfigRef, PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: admission.PromptVersion, Prompt: r139PromptText, MaxRequestBytes: admission.MaxRequestBytes, Timeout: 60 * time.Second}
}

func r139Request(t *testing.T, f *r102Fixture) models.InterventionRerunRequest {
	t.Helper()
	f.source("Tôi khiếu nại và chờ đơn hàng", "text")
	if rec := f.send(http.MethodPost, f.tenant, f.writer, "/data-revisions", f.revisionBody(1)); rec.Code != 201 {
		t.Fatalf("R139 revision: %d", rec.Code)
	}
	var revision models.InterventionDataRevision
	r102DB(t, db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Take(&revision).Error)
	if rec := f.send(http.MethodPost, f.tenant, f.writer, "/rerun-requests", f.requestBody(2, revision, "r139-"+pkg.NewUUID())); rec.Code != 201 {
		t.Fatalf("R139 request: %d", rec.Code)
	}
	var request models.InterventionRerunRequest
	r102DB(t, db.DB.Where("tenant_id = ? AND case_id = ?", f.tenant, f.caseID).Take(&request).Error)
	return request
}

func r139SendRaw(f *r102Fixture, actor, tenant, caseID, requestID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+tenant+"/intervention-cases/"+caseID+"/rerun-requests/"+requestID+"/execute", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.token[actor])
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func r139JSONNew(t *testing.T, path string, value any) {
	t.Helper()
	if err := r139WriteNew(path, value); err != nil {
		t.Fatal(err)
	}
}

func r139WriteNew(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(raw, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// A final receipt reports only observations already made. Incomplete tests
// keep a separate HOLD receipt; they cannot produce the completed terminal.
type r139FinalEvidence struct {
	Phase                 string `json:"phase"`
	Disposition           string `json:"disposition"`
	Failed                bool   `json:"failed"`
	PhysicalSlot          string `json:"physicalSlot"`
	RequestCaptured       bool   `json:"requestCaptured"`
	ResponseCaptured      bool   `json:"responseCaptured"`
	ProviderHTTPStatus    int    `json:"providerHTTPStatus"`
	ExecutePOSTObserved   int32  `json:"executePOSTObserved"`
	AuthPOSTObserved      int32  `json:"authPOSTObserved"`
	NativeExecutePOST     int    `json:"nativeExecutePOST"`
	NativeNegativeReads   int    `json:"nativeNegativeReads"`
	NativePositiveReads   int    `json:"nativePositiveReads"`
	GrantIssuances        int    `json:"grantIssuances"`
	DurableFingerprint    string `json:"durableFingerprint,omitempty"`
	FingerprintError      string `json:"fingerprintError,omitempty"`
	ExecutionQueryPresent bool   `json:"executionQueryPresent"`
	CaseQueryPresent      bool   `json:"caseQueryPresent"`
	ExecutionState        string `json:"executionState,omitempty"`
	ExecutionTerminal     string `json:"executionTerminal,omitempty"`
	CaseWorkState         string `json:"caseWorkState,omitempty"`
}

func r139WriteFinalEvidence(path string, evidence r139FinalEvidence) error {
	if evidence.Phase == "" || evidence.Disposition == "" || evidence.PhysicalSlot == "" {
		return errors.New("r139 final evidence incomplete")
	}
	return r139WriteNew(path, evidence)
}

func TestR139PureFinalEvidenceSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "final.json")
	evidence := r139FinalEvidence{Phase: "VALIDATED_PUBLICATION", Disposition: "PASS", PhysicalSlot: "MAY_HAVE_STARTED", RequestCaptured: true, ResponseCaptured: true, ExecutionState: "PUBLISHED"}
	if err := r139WriteFinalEvidence(path, evidence); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved r139FinalEvidence
	if json.Unmarshal(raw, &saved) != nil || saved != evidence {
		t.Fatal("R139 pure final success evidence mismatch")
	}
}

func TestR139PureFinalEvidenceFatalGoexit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "final.json")
	done := make(chan error, 1)
	go func() {
		defer close(done)
		defer func() {
			done <- r139WriteFinalEvidence(path, r139FinalEvidence{Phase: "BROWSER_STOPPED_TERMINAL_OBSERVATION", Disposition: "HOLD", Failed: true, PhysicalSlot: "MAY_HAVE_STARTED", RequestCaptured: true, ResponseCaptured: false, ExecutionState: "MANUAL_HOLD"})
		}()
		runtime.Goexit()
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved r139FinalEvidence
	if json.Unmarshal(raw, &saved) != nil || saved.Disposition != "HOLD" || !saved.Failed || saved.ExecutionState != "MANUAL_HOLD" || saved.ResponseCaptured {
		t.Fatal("R139 pure fatal evidence mismatch")
	}
}

func TestR139PureFinalEvidenceIOFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "final.json")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r139WriteFinalEvidence(path, r139FinalEvidence{Phase: "ANY", Disposition: "HOLD", PhysicalSlot: "NOT_RESERVED"}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("R139 pure final IO failure missing: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "original" {
		t.Fatal("R139 pure final IO failure changed original")
	}
}

type r139PureRoundTrip func(*http.Request) (*http.Response, error)

func (fn r139PureRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestR139PureJournalRequestIOFailureBlocksNext(t *testing.T) {
	dir := t.TempDir()
	requestPath := filepath.Join(dir, "provider_request_captured_private.json")
	if err := os.WriteFile(requestPath, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	journal := r139JournalTransport{evidenceDir: dir, key: "synthetic-secret", next: r139PureRoundTrip(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("should not execute")
	})}
	req := httptest.NewRequest(http.MethodPost, "https://example.invalid/chat/completions", strings.NewReader(`{"synthetic":true}`))
	if response, err := journal.RoundTrip(req); err == nil || response != nil || called {
		t.Fatal("R139 pure journal IO failure allowed next effect")
	}
}

func TestR139PureJournalCapturedResponseSurvivesGoexit(t *testing.T) {
	dir := t.TempDir()
	done := make(chan error, 1)
	go func() {
		defer close(done)
		journal := r139JournalTransport{evidenceDir: dir, key: "synthetic-secret", next: r139PureRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader(`{"actual":"captured"}`))}, nil
		})}
		req := httptest.NewRequest(http.MethodPost, "https://example.invalid/chat/completions", strings.NewReader(`{"synthetic":true}`))
		response, err := journal.RoundTrip(req)
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		done <- err
		runtime.Goexit()
	}()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "provider_response_captured_private.json"))
	var saved struct {
		HTTPStatus int    `json:"httpStatus"`
		Response   string `json:"response"`
	}
	if err != nil || json.Unmarshal(raw, &saved) != nil || saved.HTTPStatus != 202 || saved.Response != `{"actual":"captured"}` {
		t.Fatal("R139 pure journal lost captured response after fatal exit")
	}
}

func TestR139PureJournalUnsafeResponseHasNoInventedReceipt(t *testing.T) {
	dir := t.TempDir()
	journal := r139JournalTransport{evidenceDir: dir, key: "synthetic-secret", next: r139PureRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("not-json"))}, nil
	})}
	req := httptest.NewRequest(http.MethodPost, "https://example.invalid/chat/completions", strings.NewReader(`{"synthetic":true}`))
	if response, err := journal.RoundTrip(req); err == nil || response != nil {
		t.Fatal("R139 pure journal accepted unsafe response")
	}
	if _, err := os.Stat(filepath.Join(dir, "provider_response_captured_private.json")); !os.IsNotExist(err) {
		t.Fatal("R139 pure journal invented unsafe response")
	}
}

// The inner transport is invoked only after a durable, secret-free request
// journal. A response is journaled only when its actual bytes are readable
// and safe. The historical one-slot transport still owns slot reservation.
type r139JournalTransport struct {
	next        http.RoundTripper
	evidenceDir string
	key         string
}

func (j r139JournalTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(io.LimitReader(req.Body, 6001))
	if err != nil || len(raw) > 6000 || !r116SafeJSON(raw, j.key) {
		return nil, errors.New("r139 journal request invalid")
	}
	h := sha256.Sum256(raw)
	if err := r139WriteNew(filepath.Join(j.evidenceDir, "provider_request_captured_private.json"), map[string]any{"provider": "alibaba_maas", "model": "qwen3.7-flash", "request": string(raw), "requestSha256": hex.EncodeToString(h[:]), "state": "MAY_HAVE_STARTED"}); err != nil {
		return nil, fmt.Errorf("r139 request journal: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	response, err := j.next.RoundTrip(req)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, err
	}
	if response == nil || response.Body == nil {
		return nil, errors.New("r139 response missing")
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	_ = response.Body.Close()
	if readErr != nil || !r116SafeJSON(body, j.key) {
		return nil, errors.New("r139 response unreadable or unsafe")
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	responseHash := sha256.Sum256(body)
	if err := r139WriteNew(filepath.Join(j.evidenceDir, "provider_response_captured_private.json"), map[string]any{"provider": "alibaba_maas", "model": "qwen3.7-flash", "httpStatus": response.StatusCode, "response": string(body), "responseSha256": hex.EncodeToString(responseHash[:])}); err != nil {
		return nil, fmt.Errorf("r139 response journal: %w", err)
	}
	return response, nil
}

func r139Append(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(append(raw, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

type r139ResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *r139ResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func r139Grant(t *testing.T, f *r102Fixture, request models.InterventionRerunRequest, service *engine.InterventionService, deadlineUTC string) models.InterventionExecutionGrant {
	t.Helper()
	deadline, err := time.Parse(time.RFC3339, deadlineUTC)
	if err != nil || !strings.HasSuffix(deadlineUTC, "Z") || deadline.Format(time.RFC3339) != deadlineUTC {
		t.Fatal("R139 owner grant deadline invalid")
	}
	r102Must(t, &models.InterventionBudget{ID: pkg.NewUUID(), TenantID: f.tenant, JobID: f.job, Cap: 1, Consumed: 0, CreatedAt: time.Now().UTC()})
	grant, replayed, err := engine.IssueSyntheticInterventionGrant(context.Background(), f.tenant, f.owner, f.writer, f.caseID, request.ID, "R139 admitted synthetic fixture", deadline.UTC(), service)
	if err != nil || replayed || grant == nil {
		t.Fatalf("R139 server issuer: %v", err)
	}
	return *grant
}

// This uses the existing synthetic M3 seed without its old browser entrypoint.
// The WAIT_DATA case is projected into the R102 request helpers; M3's separate
// historical OPEN/proposal seed is presentation data and never AI proof.
// Unlike r102StartWithDSN, this fixture does not delete the final tenant graph.
func r139Start(t *testing.T, dsn string) *r102Fixture {
	t.Helper()
	t.Setenv("R101_MYSQL_DSN", dsn)
	m := m3FixtureStart(t)
	var wait models.InterventionCase
	r102DB(t, db.DB.Where("id = ?", m.waitID).Take(&wait).Error)
	if wait.SnapshotID == nil || wait.WorkState != "WAITING_ON_DATA" {
		t.Fatal("R139 WAIT_DATA seed missing")
	}
	f := &r102Fixture{t: t, tenant: m.tenant, other: m.otherTenant, owner: m.owner, writer: m.member, noJobs: m.assignee, reader: m.reader, conv: wait.ConversationID, job: wait.JobID, caseID: wait.ID, snap: *wait.SnapshotID, run: wait.RunID, originalDigest: wait.SnapshotDigest, token: m.token, router: m.router}
	r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{"intervention_cases":"rw","jobs":"w"}`).Error)
	r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.noJobs).Update("permissions", `{"intervention_cases":"rw"}`).Error)
	r102Must(t, &models.UserTenant{UserID: f.owner, TenantID: f.other, Role: "owner", Permissions: "{}"})
	return f
}

type r139BrandedTransport struct{ inner engine.InterventionTransport }

func (p r139BrandedTransport) AnalyzeChatOnce(ctx context.Context, prompt, transcript string, maxBytes int64) (ai.AIResponse, error) {
	response, err := p.inner.AnalyzeChatOnce(ctx, prompt, transcript, maxBytes)
	if response.Provider != "" {
		response.Provider = "alibaba_maas"
	}
	return response, err
}

func r139Count(t *testing.T, model any, clause string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Model(model).Where(clause, args...).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func r139FingerprintRaw(tenant string) (string, error) {
	tables := []string{"intervention_cases", "intervention_data_revisions", "intervention_rerun_requests", "intervention_execution_grants", "intervention_executions", "intervention_attempts", "intervention_budgets", "intervention_receipts", "intervention_proposals", "intervention_human_events", "job_runs", "analysis_snapshots", "messages", "user_tenants"}
	all := map[string][]map[string]any{}
	for _, table := range tables {
		var rows []map[string]any
		order := "id ASC"
		if table == "user_tenants" {
			order = "user_id ASC"
		}
		if err := db.DB.Table(table).Where("tenant_id = ?", tenant).Order(order).Find(&rows).Error; err != nil {
			return "", fmt.Errorf("fingerprint %s: %w", table, err)
		}
		all[table] = rows
	}
	raw, err := json.Marshal(all)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}

func r139Fingerprint(t *testing.T, tenant string) string {
	t.Helper()
	fingerprint, err := r139FingerprintRaw(tenant)
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint
}

func r139ReadNoWrite(t *testing.T, f *r102Fixture, actor, tenant, path string, want int, handle string, permitHandle bool, expectedBody ...string) map[string]any {
	t.Helper()
	before := r139Fingerprint(t, f.tenant)
	rec := f.send(http.MethodGet, tenant, actor, path, nil)
	after := r139Fingerprint(t, f.tenant)
	if before != after || rec.Code != want || !strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("R139 read status=%d want=%d durableEqual=%v", rec.Code, want, before == after)
	}
	if permitHandle {
		if handle == "" || !bytes.Contains(rec.Body.Bytes(), []byte(handle)) {
			t.Fatal("R139 current writer handle absent")
		}
	} else if bytes.Contains(rec.Body.Bytes(), []byte(`"grant_handle"`)) || (handle != "" && bytes.Contains(rec.Body.Bytes(), []byte(handle))) {
		t.Fatal("R139 denied/read-only handle leaked")
	}
	for _, expected := range expectedBody {
		if !bytes.Contains(rec.Body.Bytes(), []byte(expected)) {
			t.Fatalf("R139 read response missing expected field %q", expected)
		}
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("R139 read response JSON invalid: %v", err)
	}
	h := sha256.Sum256(rec.Body.Bytes())
	return map[string]any{"status": rec.Code, "durableBefore": before, "durableAfter": after, "bodySha256": hex.EncodeToString(h[:]), "state": response["state"], "error": response["error"], "executionPresent": response["execution_present"], "phase": response["phase"], "publishedCaseVersion": response["published_case_version"], "proposalAvailable": response["proposal_available"], "grantHandlePresent": bytes.Contains(rec.Body.Bytes(), []byte(`"grant_handle"`)), "deniedHandleAbsent": !bytes.Contains(rec.Body.Bytes(), []byte(`"grant_handle"`)) && (handle == "" || !bytes.Contains(rec.Body.Bytes(), []byte(handle))), "noStore": strings.Contains(rec.Header().Get("Cache-Control"), "no-store")}
}

func r139PostDeniedNoWrite(t *testing.T, f *r102Fixture, actor, tenant, caseID, requestID, body, name string, want int, expectedError string) map[string]any {
	t.Helper()
	before := r139Fingerprint(t, f.tenant)
	rec := r139SendRaw(f, actor, tenant, caseID, requestID, body)
	after := r139Fingerprint(t, f.tenant)
	var response map[string]any
	if json.Unmarshal(rec.Body.Bytes(), &response) != nil || len(response) != 1 || response["error"] != expectedError || before != after || rec.Code != want {
		t.Fatalf("R139 rejected POST %s status=%d want=%d durableEqual=%v", name, rec.Code, want, before == after)
	}
	h := sha256.Sum256(rec.Body.Bytes())
	return map[string]any{"slot": 0, "name": name, "status": rec.Code, "error": expectedError, "durableBefore": before, "durableAfter": after, "bodySha256": hex.EncodeToString(h[:])}
}

func TestR139IntegratedExecutionProof(t *testing.T) {
	a, dsn := r139Admit(t)
	evidenceDir := os.Getenv("R139_EVIDENCE_DIR")
	final := r139FinalEvidence{Phase: "ADMITTED_BEFORE_NATIVE", Disposition: "HOLD", PhysicalSlot: "NOT_RESERVED"}
	r139JSONNew(t, filepath.Join(evidenceDir, "go_evidence_started.json"), final)
	var executeCount, authCount atomic.Int32
	var wire *r116WireTransport
	nativeNegativeReadCount, nativePositiveReadCount := 0, 0
	nativeGrantIssuances, nativeExecutePostCount := 0, 0
	t.Cleanup(func() {
		final.Failed = t.Failed()
		final.ExecutePOSTObserved = executeCount.Load()
		final.AuthPOSTObserved = authCount.Load()
		final.NativeExecutePOST = nativeExecutePostCount
		final.NativeNegativeReads = nativeNegativeReadCount
		final.NativePositiveReads = nativePositiveReadCount
		final.GrantIssuances = nativeGrantIssuances
		if _, err := os.Stat(filepath.Join(os.Getenv("R139_EFFECT_DIR"), "slot-1.effect")); err == nil {
			final.PhysicalSlot = "MAY_HAVE_STARTED"
		} else if !os.IsNotExist(err) {
			final.PhysicalSlot = "UNKNOWN"
		}
		if _, err := os.Stat(filepath.Join(evidenceDir, "provider_request_captured_private.json")); err == nil {
			final.RequestCaptured = true
		}
		if _, err := os.Stat(filepath.Join(evidenceDir, "provider_response_captured_private.json")); err == nil {
			final.ResponseCaptured = true
		}
		if wire != nil {
			wire.mu.Lock()
			final.ProviderHTTPStatus = wire.status
			wire.mu.Unlock()
		}
		if final.Failed {
			final.Disposition = "HOLD"
		}
		if err := r139WriteFinalEvidence(filepath.Join(evidenceDir, "go_evidence_final.json"), final); err != nil {
			t.Errorf("R139 final evidence write: %v", err)
		}
	})
	if a.FixtureName == "" {
		t.Fatal("R139 fixture identity absent")
	}
	trap := &r139Trap{}
	issueGrant := func(t *testing.T, f *r102Fixture, request models.InterventionRerunRequest, service *engine.InterventionService) models.InterventionExecutionGrant {
		grant := r139Grant(t, f, request, service, a.GrantDeadlineUTC)
		nativeGrantIssuances++
		return grant
	}
	deniedPost := func(t *testing.T, f *r102Fixture, actor, tenant, caseID, requestID, body, name string, want int, expectedError string) map[string]any {
		nativeExecutePostCount++
		return r139PostDeniedNoWrite(t, f, actor, tenant, caseID, requestID, body, name, want, expectedError)
	}
	// The first native group is intentionally software-only: transport returns
	// an error with no AI output, and every reserved rejection must precede it.
	// Positive publication requires the separately admitted one-call transport.
	if !t.Run("native-negative", func(t *testing.T) {
		f := r139Start(t, dsn)
		request := r139Request(t, f)
		engine.WaitDataExecutor = r139Service(trap, a)
		defer func() { engine.WaitDataExecutor = nil }()
		readinessPath := "/rerun-requests/" + request.ID + "/execution-readiness"
		readLedger := []map[string]any{}
		read := func(name string, want int, handle, expected string) {
			row := r139ReadNoWrite(t, f, f.writer, f.tenant, readinessPath, want, handle, false, expected)
			row["name"] = name
			readLedger = append(readLedger, row)
		}
		read("grant-absent", 200, "", `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		base := fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, pkg.NewUUID())
		cases := []struct {
			name, body                       string
			actor, tenant, caseID, requestID string
			want                             int
			errorText                        string
		}{
			{"malformed-json", `{"expected_case_version":`, f.writer, f.tenant, f.caseID, request.ID, 400, "invalid_request"},
			{"unknown-field", `{"expected_case_version":3,"grant_id":"x","extra":1}`, f.writer, f.tenant, f.caseID, request.ID, 400, "invalid_request"},
			{"missing-version", `{"grant_id":"x"}`, f.writer, f.tenant, f.caseID, request.ID, 400, "invalid_request"},
			{"stale-version", strings.Replace(base, `"expected_case_version":3`, `"expected_case_version":2`, 1), f.writer, f.tenant, f.caseID, request.ID, 409, "case_conflict"},
		}
		ledger := make([]map[string]any, 0, len(cases))
		for _, c := range cases {
			row := deniedPost(t, f, c.actor, c.tenant, c.caseID, c.requestID, c.body, c.name, c.want, c.errorText)
			row["slot"] = len(ledger) + 1
			ledger = append(ledger, row)
		}
		if trap.calls.Load() != 0 {
			t.Fatal("R139 native call trap reached")
		}
		// One sacrificial grant is bound to a different actor for slot five,
		// then restored and expired for slot six. Neither can call transport.
		grant := issueGrant(t, f, request, engine.WaitDataExecutor)
		// Every read below compares complete tenant rows before and after the
		// GET. Mutation of each prerequisite is test setup, then restored.
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("state", "CONSUMED").Error)
		read("consumed-grant", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("state", "ISSUED").Error)
		engine.WaitDataExecutor.ConfigRef = "r139-invalid-config"
		read("config-mismatch", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		engine.WaitDataExecutor.ConfigRef = "r139-owned-config"
		r102DB(t, db.DB.Model(&models.InterventionBudget{}).Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Update("cap", 0).Error)
		read("budget-exhausted", 200, grant.ID, `"state":"STALE_OR_INELIGIBLE"`)
		r102DB(t, db.DB.Model(&models.InterventionBudget{}).Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Update("cap", 1).Error)
		r102DB(t, db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("version", 4).Error)
		read("case-version-changed", 200, grant.ID, `"state":"STALE_OR_INELIGIBLE"`)
		r102DB(t, db.DB.Model(&models.InterventionCase{}).Where("id = ?", f.caseID).Update("version", 3).Error)
		r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{"intervention_cases":"r","jobs":"w"}`).Error)
		read("writer-permission-revoked", 403, grant.ID, `"error":"permission_denied"`)
		r102DB(t, db.DB.Model(&models.UserTenant{}).Where("tenant_id = ? AND user_id = ?", f.tenant, f.writer).Update("permissions", `{"intervention_cases":"rw","jobs":"w"}`).Error)
		var message models.Message
		r102DB(t, db.DB.Where("tenant_id = ? AND conversation_id = ?", f.tenant, f.conv).Take(&message).Error)
		originalContent := message.Content
		r102DB(t, db.DB.Model(&models.Message{}).Where("id = ?", message.ID).Update("content", "Đổi nội dung tổng hợp").Error)
		read("source-changed", 200, grant.ID, `"state":"STALE_OR_INELIGIBLE"`)
		r102DB(t, db.DB.Model(&models.Message{}).Where("id = ?", message.ID).Update("content", originalContent).Error)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("actor_id", f.reader).Error)
		read("foreign-actor-grant", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		foreignActor := deniedPost(t, f, f.writer, f.tenant, f.caseID, request.ID, fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID), "foreign-actor-grant", 409, "case_conflict")
		foreignActor["slot"] = 5
		ledger = append(ledger, foreignActor)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("actor_id", f.writer).Error)
		r102DB(t, db.DB.Model(&models.InterventionExecutionGrant{}).Where("id = ?", grant.ID).Update("deadline", time.Now().UTC().Add(-time.Minute)).Error)
		read("expired-grant", 200, grant.ID, `"state":"AWAITING_OPERATOR_AUTHORITY"`)
		expired := deniedPost(t, f, f.writer, f.tenant, f.caseID, request.ID, fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID), "expired-grant", 409, "case_conflict")
		expired["slot"] = 6
		ledger = append(ledger, expired)
		foreignCase := deniedPost(t, f, f.owner, f.other, f.caseID, request.ID, fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID), "foreign-case", 404, "not_found")
		foreignCase["slot"] = 7
		ledger = append(ledger, foreignCase)
		if trap.calls.Load() != 0 {
			t.Fatal("R139 native call trap reached")
		}
		r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_rejections.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "slotsUsed": len(ledger), "nativeExecutePOST": nativeExecutePostCount, "grantIssuances": nativeGrantIssuances, "cases": ledger, "trapCalls": trap.calls.Load()})
		nativeNegativeReadCount = len(readLedger)
		r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_reads_negative.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": readLedger})
	}) {
		t.Fatal("R139 G1 native rejection stage failed; G2 forbidden")
	}
	if !t.Run("native-precall-denial", func(t *testing.T) {
		f := r139Start(t, dsn)
		request := r139Request(t, f)
		service := r139Service(trap, a)
		engine.WaitDataExecutor = service
		defer func() { engine.WaitDataExecutor = nil }()
		grant := issueGrant(t, f, request, service)
		callbackName := "r139:after_execution_create"
		fired := false
		if err := db.DB.Callback().Create().After("gorm:create").Register(callbackName, func(tx *gorm.DB) {
			if row, ok := tx.Statement.Dest.(*models.InterventionExecution); ok && row.RequestID == request.ID {
				service.PromptVersion = "r139-changed-after-claim"
				fired = true
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := db.DB.Callback().Create().Remove(callbackName); err != nil {
				t.Error(err)
			}
		}()
		body := fmt.Sprintf(`{"expected_case_version":3,"grant_id":%q}`, grant.ID)
		before := r139Fingerprint(t, f.tenant)
		nativeExecutePostCount++
		rec := r139SendRaw(f, f.writer, f.tenant, f.caseID, request.ID, body)
		if rec.Code != 202 || !fired {
			t.Fatalf("R139 pre-call race claim=%d callback=%v", rec.Code, fired)
		}
		var accepted map[string]any
		if json.Unmarshal(rec.Body.Bytes(), &accepted) != nil {
			t.Fatal("R139 pre-call accepted response JSON invalid")
		}
		acceptedExecution, _ := accepted["execution"].(map[string]any)
		acceptedID, _ := acceptedExecution["id"].(string)
		acceptedState, _ := acceptedExecution["state"].(string)
		if acceptedID == "" || acceptedState != "CLAIMED" {
			t.Fatalf("R139 pre-call accepted response identity/state invalid: %q %q", acceptedID, acceptedState)
		}
		var execution models.InterventionExecution
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, request.ID).Take(&execution).Error == nil && execution.TerminalCode != "" {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if execution.TerminalCode != "DENIED" || trap.calls.Load() != 0 || r139Count(t, &models.InterventionAttempt{}, "execution_id = ?", execution.ID) != 0 {
			t.Fatalf("R139 pre-call denial=%q calls=%d", execution.TerminalCode, trap.calls.Load())
		}
		var budget models.InterventionBudget
		r102DB(t, db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error)
		if budget.Consumed != 0 {
			t.Fatal("R139 denied pre-call debited budget")
		}
		after := r139Fingerprint(t, f.tenant)
		if before == after {
			t.Fatal("R139 accepted pre-call denial did not record its terminal state")
		}
		responseHash := sha256.Sum256(rec.Body.Bytes())
		r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_precall_denied.json"), map[string]any{"slot": 8, "status": rec.Code, "acceptedState": acceptedState, "acceptedExecutionId": acceptedID, "terminal": execution.TerminalCode, "executionId": execution.ID, "durableBefore": before, "durableAfter": after, "bodySha256": hex.EncodeToString(responseHash[:]), "trapCalls": trap.calls.Load(), "attempts": 0, "budgetConsumed": budget.Consumed, "nativeExecutePOSTBeforeBrowser": nativeExecutePostCount, "grantIssuancesBeforeBrowser": nativeGrantIssuances, "kind": "SOFTWARE_ONLY_NO_AI_OUTPUT"})
	}) {
		t.Fatal("R139 G1 pre-call stage failed; G2 forbidden")
	}
	if trap.calls.Load() != 0 {
		t.Fatal("R139 native call trap reached")
	}
	final.Phase = "NATIVE_SOFTWARE_MATRIX_COMPLETE"
	// The positive case is a new tenant/request/grant. Only its browser POST
	// reaches the real one-call wire wrapper. Its server is held in this process.
	f := r139Start(t, dsn)
	request := r139Request(t, f)
	var originalSnapshot models.AnalysisSnapshot
	r102DB(t, db.DB.Where("id = ?", f.snap).Take(&originalSnapshot).Error)
	key := os.Getenv("R139_ALIBABA_KEY")
	endpoint := os.Getenv("R139_ALIBABA_ENDPOINT")
	endpointDigest := sha256.Sum256([]byte(endpoint))
	effectDir := filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "effect_slots")
	if len(key) < 16 || hex.EncodeToString(endpointDigest[:]) != "b22a15b6e8aa4113c2816acae6935a84d052a2d7b81db734ccdc4f5be81b4f8f" || os.Getenv("R139_EFFECT_DIR") != effectDir {
		t.Fatal("R139 admitted provider material missing")
	}
	if err := os.Mkdir(effectDir, 0700); err != nil {
		t.Fatal("R139 effect slot directory not fresh")
	}
	isolation, err := r096IsolatedTransport()
	if err != nil {
		t.Fatal(err)
	}
	defer isolation.CloseIdleConnections()
	wire = &r116WireTransport{next: r139JournalTransport{next: isolation, evidenceDir: evidenceDir, key: key}, key: key, endpoint: endpoint, effectDir: os.Getenv("R139_EFFECT_DIR")}
	oldTransport := http.DefaultTransport
	http.DefaultTransport = wire
	defer func() { http.DefaultTransport = oldTransport }()
	provider, err := ai.NewOpenAIProvider(key, a.Model, a.CompletionTokens, endpoint).WithAlibabaQwen37OnceThinkingDisabled()
	if err != nil {
		t.Fatal(err)
	}
	service := r139Service(r139BrandedTransport{inner: provider}, a)
	engine.WaitDataExecutor = service
	defer func() { engine.WaitDataExecutor = nil }()
	grant := issueGrant(t, f, request, service)
	readinessPath := "/rerun-requests/" + request.ID + "/execution-readiness"
	statusPath := "/rerun-requests/" + request.ID + "/execution-status"
	positiveReads := []map[string]any{}
	addRead := func(name string, row map[string]any) { row["name"] = name; positiveReads = append(positiveReads, row) }
	for _, denied := range []struct{ actor, name string }{{f.reader, "readiness-denied-reader"}, {f.noJobs, "readiness-denied-nojobs"}} {
		addRead(denied.name, r139ReadNoWrite(t, f, denied.actor, f.tenant, readinessPath, 403, grant.ID, false, `"error":"permission_denied"`))
	}
	addRead("readiness-foreign-case", r139ReadNoWrite(t, f, f.owner, f.other, readinessPath, 404, grant.ID, false, `"error":"not_found"`))
	addRead("writer-READY", r139ReadNoWrite(t, f, f.writer, f.tenant, readinessPath, 200, grant.ID, true, `"state":"READY"`))
	status := r139ReadNoWrite(t, f, f.reader, f.tenant, statusPath, 200, grant.ID, false, `"execution_present":false`, `"phase":"NOT_STARTED"`)
	addRead("reader-status-no-row", status)
	engine.WaitDataExecutor = nil
	disabledStatus := r139ReadNoWrite(t, f, f.reader, f.tenant, statusPath, 200, grant.ID, false, `"execution_present":false`, `"phase":"NOT_STARTED"`)
	addRead("process-disabled-status", disabledStatus)
	addRead("process-disabled-readiness", r139ReadNoWrite(t, f, f.writer, f.tenant, readinessPath, 200, grant.ID, false, `"state":"PROCESS_DISABLED"`))
	engine.WaitDataExecutor = service
	beforeFault := r139Fingerprint(t, f.tenant)
	callbackName := "r139:unreadable_status"
	if err := db.DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "intervention_executions" {
			tx.AddError(errors.New("r139_controlled_storage_unreadable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	unreadable := f.send(http.MethodGet, f.tenant, f.reader, statusPath, nil)
	unreadableReadiness := f.send(http.MethodGet, f.tenant, f.writer, readinessPath, nil)
	if err := db.DB.Callback().Query().Remove(callbackName); err != nil {
		t.Fatal(err)
	}
	afterFault := r139Fingerprint(t, f.tenant)
	if unreadable.Code != 503 || unreadableReadiness.Code != 503 || beforeFault != afterFault || bytes.Contains(unreadable.Body.Bytes(), []byte(grant.ID)) || bytes.Contains(unreadableReadiness.Body.Bytes(), []byte(grant.ID)) || !strings.Contains(unreadable.Header().Get("Cache-Control"), "no-store") || !strings.Contains(unreadableReadiness.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("R139 unreadable status=%d readiness=%d durableEqual=%v", unreadable.Code, unreadableReadiness.Code, beforeFault == afterFault)
	}
	statusFaultHash := sha256.Sum256(unreadable.Body.Bytes())
	readinessFaultHash := sha256.Sum256(unreadableReadiness.Body.Bytes())
	addRead("status-unreadable-503", map[string]any{"status": unreadable.Code, "error": "status_unavailable", "durableBefore": beforeFault, "durableAfter": afterFault, "bodySha256": hex.EncodeToString(statusFaultHash[:]), "grantHandlePresent": false, "deniedHandleAbsent": !bytes.Contains(unreadable.Body.Bytes(), []byte(grant.ID)), "noStore": strings.Contains(unreadable.Header().Get("Cache-Control"), "no-store")})
	addRead("readiness-unreadable-503", map[string]any{"status": unreadableReadiness.Code, "error": "readiness_unavailable", "durableBefore": beforeFault, "durableAfter": afterFault, "bodySha256": hex.EncodeToString(readinessFaultHash[:]), "grantHandlePresent": false, "deniedHandleAbsent": !bytes.Contains(unreadableReadiness.Body.Bytes(), []byte(grant.ID)), "noStore": strings.Contains(unreadableReadiness.Header().Get("Cache-Control"), "no-store")})
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_reads_positive_pre_browser.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": positiveReads})
	var routeSequence atomic.Int64
	var routeLock sync.Mutex
	routePath := filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "go_routes.jsonl")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := &r139ResponseWriter{ResponseWriter: w, status: 200}
		path := r.URL.Path
		id := routeSequence.Add(1)
		auth, execute, allowed := r139BrowserRoute(r.Method, path)
		if auth {
			allowed = authCount.Add(1) <= 4
		}
		if execute {
			allowed = executeCount.Add(1) <= 1
		}
		routeLock.Lock()
		logErr := r139Append(routePath, map[string]any{"phase": "attempt", "id": id, "method": r.Method, "path": path, "allowed": allowed})
		routeLock.Unlock()
		if logErr != nil {
			http.Error(wrapped, "r139_route_ledger_unavailable", 503)
			return
		}
		defer func() {
			routeLock.Lock()
			defer routeLock.Unlock()
			if err := r139Append(routePath, map[string]any{"phase": "response", "id": id, "method": r.Method, "path": path, "status": wrapped.status}); err != nil {
				t.Errorf("R139 route response ledger: %v", err)
			}
		}()
		if !allowed {
			http.Error(wrapped, "r139_route_not_allowed", 405)
			return
		}
		if execute {
			body, err := io.ReadAll(io.LimitReader(r.Body, 513))
			if err != nil || len(body) > 512 {
				http.Error(wrapped, "r139_body_invalid", http.StatusBadRequest)
				return
			}
			var fields map[string]json.RawMessage
			if json.Unmarshal(body, &fields) != nil || len(fields) != 2 || fields["grant_id"] == nil || fields["expected_case_version"] == nil {
				http.Error(wrapped, "r139_body_invalid", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		f.router.ServeHTTP(wrapped, r)
	}))
	defer server.Close()
	if !strings.HasPrefix(server.URL, "http://127.0.0.1:") {
		t.Fatal("R139 browser origin not loopback")
	}
	manifest := map[string]any{"tranche": "CCMAI-RUNTIME-139", "apiOrigin": server.URL, "tenant": f.tenant, "writerEmail": f.writer + "@example.invalid", "readerEmail": f.reader + "@example.invalid", "syntheticPassword": "R101SyntheticOnly!123", "caseId": f.caseID, "requestId": request.ID, "expectedCaseVersion": 3, "browserExecuteCap": 1}
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "browser_manifest_private.json"), manifest)
	final.Phase = "BROWSER_READY_AWAITING_STOP"
	fmt.Printf("R139_BROWSER_READY %s\n", server.URL)
	stop := os.Getenv("R139_STOP_FILE")
	if !filepath.IsAbs(stop) {
		t.Fatal("R139 stop file invalid")
	}
	end := time.Now().Add(20 * time.Minute)
	for time.Now().Before(end) {
		if _, err := os.Stat(stop); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if _, err := os.Stat(stop); err != nil {
		t.Error("R139 supervisor stop deadline")
	}
	final.Phase = "BROWSER_STOPPED_TERMINAL_OBSERVATION"
	wire.mu.Lock()
	capturedCalls, capturedRequest, capturedResponse, capturedStatus := wire.attempts, wire.request, wire.response, wire.status
	wire.mu.Unlock()
	if capturedCalls > 0 && capturedRequest != "" && capturedResponse != "" {
		r139JSONNew(t, filepath.Join(evidenceDir, "provider_wire_private.json"), map[string]any{"provider": "alibaba_maas", "model": "qwen3.7-flash", "httpStatus": capturedStatus, "request": capturedRequest, "response": capturedResponse, "physicalCalls": capturedCalls})
	}
	var row models.InterventionExecution
	rowErr := db.DB.Where("tenant_id = ? AND request_id = ?", f.tenant, request.ID).Take(&row).Error
	var storedRequest models.InterventionRerunRequest
	requestErr := db.DB.Where("id = ?", request.ID).Take(&storedRequest).Error
	var current models.InterventionCase
	caseErr := db.DB.Where("id = ?", f.caseID).Take(&current).Error
	final.ExecutionQueryPresent = rowErr == nil
	final.CaseQueryPresent = caseErr == nil
	if rowErr == nil {
		final.ExecutionState = row.State
		final.ExecutionTerminal = row.TerminalCode
	}
	if caseErr == nil {
		final.CaseWorkState = current.WorkState
	}
	var snapshotAfter models.AnalysisSnapshot
	snapshotErr := db.DB.Where("id = ?", f.snap).Take(&snapshotAfter).Error
	var run models.JobRun
	runErr := db.DB.Where("id = ?", row.RunID).Take(&run).Error
	var newSnapshot models.AnalysisSnapshot
	newSnapshotErr := db.DB.Where("id = ?", row.SnapshotID).Take(&newSnapshot).Error
	var proposal models.InterventionProposal
	proposalErr := db.DB.Where("tenant_id = ? AND case_id = ? AND run_id = ?", f.tenant, f.caseID, row.RunID).Take(&proposal).Error
	var event models.InterventionHumanEvent
	eventErr := db.DB.Where("tenant_id = ? AND case_id = ? AND rerun_request_id = ? AND action = ?", f.tenant, f.caseID, request.ID, "EXECUTE_RERUN").Take(&event).Error
	var receipt models.InterventionReceipt
	receiptErr := db.DB.Where("tenant_id = ? AND run_id = ?", f.tenant, row.RunID).Take(&receipt).Error
	var attempt models.InterventionAttempt
	attemptErr := db.DB.Where("tenant_id = ? AND execution_id = ?", f.tenant, row.ID).Take(&attempt).Error
	var budget models.InterventionBudget
	budgetErr := db.DB.Where("tenant_id = ? AND job_id = ?", f.tenant, f.job).Take(&budget).Error
	var after map[string]any
	jsonErr := json.Unmarshal([]byte(event.After), &after)
	originalRequestUnchanged := requestErr == nil && reflect.DeepEqual(storedRequest, request)
	originalSnapshotUnchanged := snapshotErr == nil && reflect.DeepEqual(snapshotAfter, originalSnapshot)
	lineage := rowErr == nil && runErr == nil && newSnapshotErr == nil && proposalErr == nil && eventErr == nil && receiptErr == nil && attemptErr == nil && budgetErr == nil && jsonErr == nil && caseErr == nil && row.RequestID == request.ID && row.CaseID == f.caseID && row.GrantID == grant.ID && row.RunID == run.ID && row.SnapshotID == newSnapshot.ID && row.SnapshotDigest == newSnapshot.Digest && proposal.RunID == row.RunID && proposal.SnapshotID == row.SnapshotID && event.ProposalID != nil && *event.ProposalID == proposal.ID && event.RerunRequestID != nil && *event.RerunRequestID == request.ID && event.CaseVersion == proposal.Revision && current.ActiveProposalID != nil && *current.ActiveProposalID == proposal.ID && current.Version == proposal.Revision && run.Status == "success" && run.FinishedAt != nil && receipt.RunID == run.ID && attempt.RunID == run.ID && attempt.ExecutionID != nil && *attempt.ExecutionID == row.ID && budget.Consumed == 1 && after["execution_id"] == row.ID && after["request_id"] == request.ID && after["run_id"] == run.ID && after["snapshot_id"] == newSnapshot.ID
	// Export the observed state before any publication assertion can terminate
	// this test. A failed expectation leaves the separate partial HOLD receipt.
	fingerprint, fingerprintErr := r139FingerprintRaw(f.tenant)
	final.DurableFingerprint = fingerprint
	if fingerprintErr != nil {
		final.FingerprintError = fingerprintErr.Error()
	}
	readerStatus := r139ReadNoWrite(t, f, f.reader, f.tenant, statusPath, 200, grant.ID, false, `"phase":"PUBLISHED"`, `"proposal_available":true`, fmt.Sprintf(`"published_case_version":%d`, proposal.Revision))
	writerStatus := r139ReadNoWrite(t, f, f.writer, f.tenant, statusPath, 200, grant.ID, false, `"phase":"PUBLISHED"`)
	history := r139ReadNoWrite(t, f, f.reader, f.tenant, "/rerun-requests?limit=10", 200, grant.ID, false, request.ID)
	detail := r139ReadNoWrite(t, f, f.reader, f.tenant, "", 200, grant.ID, false, proposal.ID)
	addRead("reader-published-status", readerStatus)
	addRead("writer-published-status", writerStatus)
	addRead("reader-history", history)
	addRead("reader-open-detail", detail)
	nativePositiveReadCount = len(positiveReads)
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "native_reads_positive.json"), map[string]any{"kind": "SOFTWARE_ONLY_NO_AI_OUTPUT", "reads": positiveReads})
	statusPublished := readerStatus["status"] == 200 && writerStatus["status"] == 200
	historyVisible := history["status"] == 200 && detail["status"] == 200
	wire.mu.Lock()
	calls, requestWire, responseWire, providerStatus := wire.attempts, wire.request, wire.response, wire.status
	wire.mu.Unlock()
	terminal := map[string]any{"nativeExecutePOST": nativeExecutePostCount, "nativeNegativeReadCount": nativeNegativeReadCount, "nativePositiveReadCount": nativePositiveReadCount, "grantIssuances": nativeGrantIssuances, "executePOST": executeCount.Load(), "authPOST": authCount.Load(), "providerPhysicalCalls": calls, "provider": service.Provider, "model": service.Model, "configRef": service.ConfigRef, "promptVersion": service.PromptVersion, "promptSha256": a.PromptSHA256, "completionTokens": a.CompletionTokens, "maxRequestBytes": a.MaxRequestBytes, "thinkingDisabled": true, "syntheticNamespace": a.SyntheticNamespace, "issuerRole": a.IssuerRole, "actorRole": a.ActorRole, "grantDeadlineUTC": a.GrantDeadlineUTC, "grantId": grant.ID, "requestSha256": func() string { h := sha256.Sum256([]byte(requestWire)); return hex.EncodeToString(h[:]) }(), "responseSha256": func() string { h := sha256.Sum256([]byte(responseWire)); return hex.EncodeToString(h[:]) }(), "providerHTTPStatus": providerStatus, "originalRequestUnchanged": originalRequestUnchanged, "originalSnapshotUnchanged": originalSnapshotUnchanged, "lineageVerified": lineage, "readerWriterPublishedStatus": statusPublished, "readerHistoryAndOpenDetail": historyVisible, "caseWorkState": current.WorkState, "executionPresent": rowErr == nil, "executionTerminal": row.TerminalCode, "executionState": row.State, "caseId": f.caseID, "requestId": request.ID, "storedRequestId": storedRequest.ID, "storedRequestCaseId": storedRequest.CaseID, "executionId": row.ID, "executionRequestId": row.RequestID, "executionCaseId": row.CaseID, "executionGrantId": row.GrantID, "executionRunId": row.RunID, "executionSnapshotId": row.SnapshotID, "jobRunId": run.ID, "snapshotId": newSnapshot.ID, "snapshotDigest": newSnapshot.Digest, "proposalId": proposal.ID, "proposalRunId": proposal.RunID, "proposalSnapshotId": proposal.SnapshotID, "proposalRevision": proposal.Revision, "eventId": event.ID, "eventRequestId": event.RerunRequestID, "eventProposalId": event.ProposalID, "eventCaseVersion": event.CaseVersion, "attemptId": attempt.ID, "attemptRunId": attempt.RunID, "attemptExecutionId": attempt.ExecutionID, "receiptId": receipt.ID, "receiptRunId": receipt.RunID, "budgetConsumed": budget.Consumed, "usageState": attempt.UsageState, "inputUsagePresence": attempt.InputUsagePresence, "outputUsagePresence": attempt.OutputUsagePresence, "inputTokens": attempt.InputTokens, "outputTokens": attempt.OutputTokens, "responseModel": attempt.ResponseModel, "durableFingerprint": fingerprint, "fingerprintError": fingerprintErr != nil}
	if executeCount.Load() != 1 || calls != 1 || responseWire == "" || requestWire == "" || providerStatus != 200 || rowErr != nil || row.State != "PUBLISHED" || current.WorkState != "OPEN" || !originalRequestUnchanged || !originalSnapshotUnchanged || !lineage || !statusPublished || !historyVisible || fingerprintErr != nil {
		t.Fatal("R139 positive publication HOLD")
	}
	r139JSONNew(t, filepath.Join(os.Getenv("R139_EVIDENCE_DIR"), "go_terminal.json"), terminal)
	final.Phase = "VALIDATED_PUBLICATION"
	final.Disposition = "PASS"
}
