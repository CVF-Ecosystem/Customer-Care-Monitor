package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// validInventoryJSON is written by hand as an external synthetic expectation. It is not produced
// by marshalling buildFixture's inventory, so it stays independent of the built-in transcript.
const validInventoryJSON = `{
  "schema_version": "pancake-proof-synthetic-inventory/1",
  "evidence_type": "SYNTHETIC_OFFLINE",
  "provenance": "synthetic-file-v1",
  "page_id": "synthetic-page-001",
  "conversations": [
    {"id": "syn-conv-a", "updated_at": "2026-09-24T10:00:00Z", "messages": [
      {"id": "syn-msg-a1", "sent_at": "2026-09-24T09:00:00Z", "sender_type": "customer", "content_type": "text", "attachments": []},
      {"id": "syn-msg-a2", "sent_at": "2026-09-24T09:05:00Z", "sender_type": "agent", "content_type": "attachment",
       "attachments": [{"type": "image", "name": "photo-001.jpg"}]}
    ]},
    {"id": "syn-conv-b", "updated_at": "2026-09-20T00:00:00Z", "messages": [
      {"id": "syn-msg-b1", "sent_at": "2026-09-20T00:00:00Z", "sender_type": "customer", "content_type": "text", "attachments": []}
    ]}
  ]
}`

const inventoryRejected = "pancake-proof: inventory rejected\n"

var digestRe = regexp.MustCompile(`"digest": "([0-9a-f]{64})"`)

// realDir returns a temp directory without symlink ancestors (some hosts place temp roots
// behind links, which the loader deliberately rejects).
func realDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func writeInventory(t *testing.T, body []byte) string {
	t.Helper()
	p := filepath.Join(realDir(t), "synthetic-inventory.json")
	if err := os.WriteFile(p, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// mutate replaces exactly one occurrence so a mutation can never silently be a no-op.
func mutate(t *testing.T, old, repl string) string {
	t.Helper()
	if n := strings.Count(validInventoryJSON, old); n != 1 {
		t.Fatalf("mutation target %q matched %d times", old, n)
	}
	return strings.Replace(validInventoryJSON, old, repl, 1)
}

func runInventory(t *testing.T, body string, extra ...string) result {
	t.Helper()
	return execMain(t, t.TempDir(), nil, base(append([]string{"-inventory", writeInventory(t, []byte(body))}, extra...)...)...)
}

func digestOf(t *testing.T, r result) string {
	t.Helper()
	m := digestRe.FindStringSubmatch(r.out)
	if m == nil {
		t.Fatalf("receipt has no inventory digest: %q", r.out)
	}
	return m[1]
}

// OI-01
func TestInventoryValidFilePassesThroughMountedMain(t *testing.T) {
	first := runInventory(t, validInventoryJSON)
	second := runInventory(t, validInventoryJSON)
	if first.code != 0 {
		t.Fatalf("a valid synthetic inventory must PASS offline, got exit %d stderr=%q\n%s", first.code, first.err, first.out)
	}
	for _, want := range []string{`"disposition": "PASS"`, `"attempts_used": 12`, `"provenance": "synthetic-file-v1"`,
		`"evidence_type": "SYNTHETIC_OFFLINE"`, `"live": false`, `"governance_claim": false`, `"conversation_count": 2`, `"message_count": 3`} {
		if !strings.Contains(first.out, want) {
			t.Errorf("receipt missing %s\n%s", want, first.out)
		}
	}
	if !strings.Contains(first.err, "SYNTHETIC OFFLINE") {
		t.Errorf("banner missing: %q", first.err)
	}
	if second.code != first.code || normalize(second.out) != normalize(first.out) {
		t.Errorf("two runs of the same file must agree")
	}
	// The same expectations under the built-in provenance reproduce the built-in receipt: the file
	// replaces only the expected inventory, nothing else about the proof run.
	builtin := execMain(t, t.TempDir(), nil, base()...)
	same := runInventory(t, strings.Replace(validInventoryJSON, "synthetic-file-v1", "synthetic-cli-fixture-v1", 1))
	if normalize(same.out) != normalize(builtin.out) || same.code != builtin.code {
		t.Errorf("equivalent file expectations must give the built-in receipt:\n%s\n%s", same.out, builtin.out)
	}
	// Explicit -scenario pass is accepted as well.
	if r := runInventory(t, validInventoryJSON, "-scenario", "pass"); r.code != 0 {
		t.Errorf("explicit pass scenario with -inventory must work, got %d", r.code)
	}
}

type invalidCase struct {
	name string
	body []byte
	code string
}

func invalidCases(t *testing.T) []invalidCase {
	m := func(name, old, repl, code string) invalidCase {
		return invalidCase{name, []byte(mutate(t, old, repl)), code}
	}
	cases := []invalidCase{
		m("schema-version", `inventory/1"`, `inventory/2"`, "schema_version"),
		m("schema-missing-prefix", `"pancake-proof-synthetic-inventory/1"`, `"other"`, "schema_version"),
		m("evidence-live", `"SYNTHETIC_OFFLINE"`, `"LIVE"`, "evidence_type"),
		m("provenance-not-synthetic", `"synthetic-file-v1"`, `"live-file-v1"`, "provenance"),
		m("provenance-unsafe-chars", `"synthetic-file-v1"`, `"synthetic file v1"`, "provenance"),
		m("provenance-too-long", `"synthetic-file-v1"`, `"synthetic-`+strings.Repeat("x", 60)+`"`, "provenance"),
		m("page-id", `"synthetic-page-001"`, `"synthetic-page-002"`, "page_id"),
		m("missing-root-field", `"page_id": "synthetic-page-001",`, ``, "missing_field"),
		m("missing-conversation-field", `"id": "syn-conv-b", "updated_at": "2026-09-20T00:00:00Z", `, `"id": "syn-conv-b", `, "missing_field"),
		m("missing-message-field", `"sender_type": "agent", `, ``, "missing_field"),
		m("missing-attachment-field", `{"type": "image", "name": "photo-001.jpg"}`, `{"type": "image"}`, "missing_field"),
		m("unknown-root-field", `"provenance":`, `"live": true, "provenance":`, "unknown_field"),
		m("unknown-root-token-field", `"provenance":`, `"page_access_token": "CANARY-INV-0001", "provenance":`, "unknown_field"),
		m("unknown-conversation-field", `"id": "syn-conv-b",`, `"raw_capture": "x", "id": "syn-conv-b",`, "unknown_field"),
		m("unknown-message-field", `"id": "syn-msg-b1",`, `"text": "hello", "id": "syn-msg-b1",`, "unknown_field"),
		m("unknown-attachment-field", `{"type": "image", "name": "photo-001.jpg"}`, `{"type": "image", "name": "photo-001.jpg", "url": "https://x.invalid/a"}`, "unknown_field"),
		m("case-alias-root-key", `"page_id":`, `"Page_ID":`, "missing_field"),
		m("case-alias-nested-key", `"sender_type": "agent"`, `"Sender_Type": "agent"`, "missing_field"),
		m("duplicate-root-key-same-value", `"provenance": "synthetic-file-v1",`, `"provenance": "synthetic-file-v1", "provenance": "synthetic-file-v1",`, "duplicate_key"),
		m("duplicate-message-key", `"id": "syn-msg-b1",`, `"id": "syn-msg-b1", "id": "syn-msg-b1",`, "duplicate_key"),
		m("duplicate-attachment-key", `{"type": "image", "name": "photo-001.jpg"}`, `{"type": "image", "type": "image", "name": "photo-001.jpg"}`, "duplicate_key"),
		m("null-required-string", `"provenance": "synthetic-file-v1"`, `"provenance": null`, "type"),
		m("null-required-array", `"attachments": []},
      {"id": "syn-msg-a2"`, `"attachments": null},
      {"id": "syn-msg-a2"`, "type"),
		m("number-for-string", `"sender_type": "agent"`, `"sender_type": 7`, "type"),
		m("string-for-array", `"attachments": []},
      {"id": "syn-msg-a2"`, `"attachments": "none"},
      {"id": "syn-msg-a2"`, "type"),
		m("object-for-array", `"attachments": []}
    ]}
  ]`, `"attachments": {}}
    ]}
  ]`, "type"),
		{"trailing-garbage", []byte(validInventoryJSON + " x"), "trailing_data"},
		{"trailing-second-document", []byte(validInventoryJSON + "\n" + validInventoryJSON), "trailing_data"},
		{"truncated-document", []byte(validInventoryJSON[:len(validInventoryJSON)-20]), "syntax"},
		{"empty-file", []byte(""), "syntax"},
		{"not-an-object", []byte(`[]`), "type"},
		{"invalid-utf8", bytes.Replace([]byte(validInventoryJSON), []byte("photo-001.jpg"), []byte("photo-\xff\xfe.jpg"), 1), "utf8"},
		m("replacement-char-name", `"photo-001.jpg"`, `"photo-�.jpg"`, "attachment_name"),
		m("conversation-id-without-prefix", `"id": "syn-conv-a"`, `"id": "conv-a"`, "conversation_id"),
		m("conversation-id-separator", `"id": "syn-conv-a"`, `"id": "syn-conv/a"`, "conversation_id"),
		m("conversation-id-dot-escape", `"id": "syn-conv-a"`, `"id": "syn-%2e%2e"`, "conversation_id"),
		m("conversation-id-too-long", `"id": "syn-conv-a"`, `"id": "syn-`+strings.Repeat("a", 125)+`"`, "conversation_id"),
		m("conversation-id-duplicate", `"id": "syn-conv-b"`, `"id": "syn-conv-a"`, "conversation_id"),
		m("message-id-without-prefix", `"id": "syn-msg-a1"`, `"id": "msg-a1"`, "message_id"),
		m("message-id-duplicate", `"id": "syn-msg-a2"`, `"id": "syn-msg-a1"`, "message_id"),
		m("time-without-zone", `"2026-09-24T10:00:00Z"`, `"2026-09-24T10:00:00"`, "time"),
		m("time-date-only", `"2026-09-24T10:00:00Z"`, `"2026-09-24"`, "time"),
		m("time-zero-instant", `"2026-09-20T00:00:00Z", "sender_type"`, `"0001-01-01T00:00:00Z", "sender_type"`, "time"),
		m("sender-type", `"agent"`, `"robot"`, "sender_type"),
		m("content-type", `"content_type": "attachment"`, `"content_type": "video"`, "content_type"),
		m("attachment-type", `"type": "image"`, `"type": "video"`, "attachment_type"),
		m("attachment-name-empty", `"photo-001.jpg"`, `""`, "attachment_name"),
		m("attachment-name-control", `"photo-001.jpg"`, `"photo\u0007.jpg"`, "attachment_name"),
		m("attachment-name-too-long", `"photo-001.jpg"`, `"`+strings.Repeat("n", 257)+`"`, "attachment_name"),
	}
	return cases
}

// OI-02 and OI-05: every malformed file is rejected for its specific reason, before any receipt.
func TestInventoryStrictSchemaRejectsEveryMalformedInput(t *testing.T) {
	for _, c := range invalidCases(t) {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseInventory(bytes.NewReader(c.body)); err == nil || inventoryCode(err) != c.code {
				t.Fatalf("want rejection code %q, got %v (%q)", c.code, err, inventoryCode(err))
			}
			r := execMain(t, t.TempDir(), nil, base("-inventory", writeInventory(t, c.body))...)
			if r.code != 2 || r.out != "" || r.err != inventoryRejected {
				t.Fatalf("CLI must exit 2 with empty stdout and the fixed message, got %d %q %q", r.code, r.out, r.err)
			}
		})
	}
}

func TestInventoryValidFileParsesToCallerExpectations(t *testing.T) {
	inv, err := parseInventory(strings.NewReader(validInventoryJSON))
	if err != nil {
		t.Fatalf("valid file rejected: %v", err)
	}
	if inv.Provenance != "synthetic-file-v1" || inv.PageID != fixturePage || len(inv.Conversations) != 2 {
		t.Fatalf("header not mapped: %+v", inv)
	}
	a := inv.Conversations[0]
	if a.ID != "syn-conv-a" || len(a.Messages) != 2 || a.Messages[1].SenderType != "agent" || a.Messages[1].ContentType != "attachment" ||
		len(a.Messages[1].Attachments) != 1 || a.Messages[1].Attachments[0].Name != "photo-001.jpg" || a.Messages[1].SentAt.Location().String() != "UTC" {
		t.Fatalf("conversation not mapped: %+v", a)
	}
}

type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { return len(p), nil }

func paddedTo(size int) []byte {
	b := []byte(validInventoryJSON)
	return append(b, bytes.Repeat([]byte(" "), size-len(b))...)
}

func generated(convs, msgsPerConv, atts int) string {
	var sb strings.Builder
	sb.WriteString(`{"schema_version":"pancake-proof-synthetic-inventory/1","evidence_type":"SYNTHETIC_OFFLINE","provenance":"synthetic-gen","page_id":"synthetic-page-001","conversations":[`)
	for c := 0; c < convs; c++ {
		if c > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"id":"syn-c%d","updated_at":"2026-09-24T10:00:00Z","messages":[`, c)
		for m := 0; m < msgsPerConv; m++ {
			if m > 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, `{"id":"syn-m%d","sent_at":"2026-09-24T09:00:00Z","sender_type":"customer","content_type":"text","attachments":[`, m)
			for a := 0; a < atts; a++ {
				if a > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(`{"type":"image","name":"p.jpg"}`)
			}
			sb.WriteString(`]}`)
		}
		sb.WriteString(`]}`)
	}
	sb.WriteString(`]}`)
	return sb.String()
}

// OI-03
func TestInventoryReadAndResourceLimits(t *testing.T) {
	// The reader is never consumed beyond the ceiling plus one byte, even for an endless source.
	cr := &countingReader{r: zeroReader{}}
	if _, err := parseInventory(cr); inventoryCode(err) != "too_large" || cr.n != maxInventoryBytes+1 {
		t.Fatalf("endless input: code %q after reading %d bytes (want too_large after %d)", inventoryCode(err), cr.n, maxInventoryBytes+1)
	}
	// Boundary: exactly the ceiling is accepted, one byte more is rejected (not truncated).
	at := &countingReader{r: bytes.NewReader(paddedTo(maxInventoryBytes))}
	if _, err := parseInventory(at); err != nil || at.n != maxInventoryBytes {
		t.Fatalf("a file of exactly %d bytes must parse (err=%v, read=%d)", maxInventoryBytes, err, at.n)
	}
	over := &countingReader{r: bytes.NewReader(paddedTo(maxInventoryBytes + 1))}
	if _, err := parseInventory(over); inventoryCode(err) != "too_large" || over.n != maxInventoryBytes+1 {
		t.Fatalf("one byte over must be rejected as too_large after at most %d bytes, got %q read=%d", maxInventoryBytes+1, inventoryCode(err), over.n)
	}
	if r := execMain(t, t.TempDir(), nil, base("-inventory", writeInventory(t, paddedTo(maxInventoryBytes)))...); r.code != 0 {
		t.Errorf("boundary-size file through mounted main: exit %d %q", r.code, r.err)
	}
	if r := execMain(t, t.TempDir(), nil, base("-inventory", writeInventory(t, paddedTo(maxInventoryBytes+1)))...); r.code != 2 || r.out != "" || r.err != inventoryRejected {
		t.Errorf("oversized file through mounted main: %d %q %q", r.code, r.out, r.err)
	}

	for _, c := range []struct {
		name            string
		convs, msgs, at int
		code            string
	}{
		{"100-conversations-ok", 100, 10, 0, ""},
		{"101-conversations", 101, 1, 0, "too_many_conversations"},
		{"1000-messages-ok", 10, 100, 0, ""},
		{"1001-messages", 11, 91, 0, "too_many_messages"},
		{"16-attachments-ok", 1, 1, 16, ""},
		{"17-attachments", 1, 1, 17, "too_many_attachments"},
	} {
		_, err := parseInventory(strings.NewReader(generated(c.convs, c.msgs, c.at)))
		if c.code == "" && err != nil || c.code != "" && inventoryCode(err) != c.code {
			t.Errorf("%s: want %q got %v (%q)", c.name, c.code, err, inventoryCode(err))
		}
	}
	if r := execMain(t, t.TempDir(), nil, base("-inventory", writeInventory(t, []byte(generated(101, 1, 0))))...); r.code != 2 || r.out != "" {
		t.Errorf("101 conversations through mounted main: %d %q", r.code, r.out)
	}

	nest := func(depth int) string {
		return strings.Replace(validInventoryJSON, `"conversations":`, `"x": `+strings.Repeat("[", depth)+strings.Repeat("]", depth)+`, "conversations":`, 1)
	}
	if _, err := parseInventory(strings.NewReader(nest(64))); inventoryCode(err) != "depth" {
		t.Errorf("65 total levels must hit the depth ceiling, got %q", inventoryCode(err))
	}
	if _, err := parseInventory(strings.NewReader(nest(63))); inventoryCode(err) != "unknown_field" {
		t.Errorf("64 total levels pass the depth ceiling and fail only on the unknown key, got %q", inventoryCode(err))
	}
}

// OI-03 file admission and option handling.
func TestInventoryFileAdmissionAndOptionConflicts(t *testing.T) {
	dir := realDir(t)
	missing := filepath.Join(dir, "missing.json")
	for name, p := range map[string]string{
		"missing": missing, "directory": dir, "empty": "", "stdin-dash": "-", "unc": `\\server\share\inv.json`,
		"double-slash": "//server/share/inv.json", "file-uri": "file:///tmp/inv.json", "http-uri": "http://127.0.0.1/inv.json", "nul": "a\x00b",
	} {
		if _, err := loadInventoryFile(p); err == nil || (inventoryCode(err) != "path" && inventoryCode(err) != "not_regular_file") {
			t.Errorf("%s: must be rejected at admission, got %v", name, err)
		}
	}
	target := writeInventory(t, []byte(validInventoryJSON))

	// Option conflicts are rejected before the file is touched: the path does not exist, yet the
	// message names the scenario conflict instead of an inventory failure.
	for _, sc := range []string{"missing-conversation", "redirect", "empty-inventory", "live"} {
		r := execMain(t, t.TempDir(), nil, base("-scenario", sc, "-inventory", missing)...)
		if r.code != 2 || r.out != "" || !strings.Contains(r.err, "supported only with the pass scenario") {
			t.Errorf("%s: want the scenario conflict before the file is opened, got %d %q", sc, r.code, r.err)
		}
	}
	for name, args := range map[string][]string{
		"empty-path":    base("-inventory", ""),
		"twice":         base("-inventory", target, "-inventory", target),
		"missing-value": base("-inventory"),
	} {
		if r := execMain(t, t.TempDir(), nil, args...); r.code != 2 || r.out != "" {
			t.Errorf("%s: want exit 2 and empty stdout, got %d %q", name, r.code, r.out)
		}
	}
}

// R041-R1-01: every UNC/device/namespace spelling, with either separator in any mix, is rejected
// by syntax alone. The filesystem seams fail the test if anything touches the filesystem, so no
// real share or server is ever contacted, even when the guard is broken.
func TestInventoryUNCSpellingsRejectedBeforeAnyFilesystemOperation(t *testing.T) {
	origLstat, origOpen := lstatFn, openFn
	t.Cleanup(func() { lstatFn, openFn = origLstat, origOpen })
	lstatFn = func(name string) (os.FileInfo, error) {
		t.Errorf("filesystem touched (lstat) for %q", name)
		return nil, os.ErrNotExist
	}
	openFn = func(name string) (*os.File, error) {
		t.Errorf("filesystem touched (open) for %q", name)
		return nil, os.ErrNotExist
	}
	rejected := []string{
		`\\server\share\inv.json`, `//server/share/inv.json`,
		`\/server/share/inv.json`, `/\server\share\inv.json`, // mixed leading separators
		`\\?\UNC\server\share\inv.json`, `//?/UNC/server/share/inv.json`, `\/?/UNC/server/share/inv.json`, `/\?\UNC\server\share\inv.json`,
		`\\.\pipe\inventory`, `\\?\C:\data\inv.json`,
		`\??\UNC\server\share\inv.json`, `/??/UNC/server/share/inv.json`, `\??/UNC\server/share\inv.json`,
		"", "-", "a\x00b", "file:///tmp/inv.json", "http://127.0.0.1/inv.json",
	}
	for _, p := range rejected {
		if !syntacticReject(p) {
			t.Errorf("syntacticReject(%q) = false", p)
		}
		if _, err := loadInventoryFile(p); inventoryCode(err) != "path" {
			t.Errorf("loadInventoryFile(%q): want rejection code path, got %q", p, inventoryCode(err))
		}
	}
	for _, p := range []string{`C:\data\inv.json`, "rel/inv.json", "./inv.json", "/tmp/inv.json", `dir\inv.json`, `..\inv.json`, "a/b/../inv.json"} {
		if syntacticReject(p) {
			t.Errorf("ordinary local spelling %q must not be rejected syntactically", p)
		}
	}
}

// A relative path that resolves into a UNC location (for example a working directory on a share)
// is re-filtered after resolution, still before any filesystem operation.
func TestInventoryResolvedUNCPathRejectedBeforeAnyFilesystemOperation(t *testing.T) {
	origLstat, origOpen, origAbs := lstatFn, openFn, absFn
	t.Cleanup(func() { lstatFn, openFn, absFn = origLstat, origOpen, origAbs })
	lstatFn = func(name string) (os.FileInfo, error) {
		t.Errorf("filesystem touched (lstat) for %q", name)
		return nil, os.ErrNotExist
	}
	openFn = func(name string) (*os.File, error) {
		t.Errorf("filesystem touched (open) for %q", name)
		return nil, os.ErrNotExist
	}
	for _, resolved := range []string{`\\server\share\dir\inv.json`, `\/server/share/dir/inv.json`, `//server/share/inv.json`} {
		absFn = func(string) (string, error) { return resolved, nil }
		if _, err := loadInventoryFile("inv.json"); inventoryCode(err) != "path" {
			t.Errorf("a path resolving to %q must be rejected before I/O, got %q", resolved, inventoryCode(err))
		}
	}
}

// A non-regular final element (here a directory) is rejected during admission, before the file
// is opened: opening a pipe or device could block or have side effects.
func TestInventoryNonRegularFileRejectedBeforeOpen(t *testing.T) {
	origOpen := openFn
	t.Cleanup(func() { openFn = origOpen })
	openFn = func(name string) (*os.File, error) {
		t.Errorf("open called for non-regular %q", name)
		return nil, os.ErrNotExist
	}
	if _, err := loadInventoryFile(realDir(t)); inventoryCode(err) != "not_regular_file" {
		t.Errorf("a directory must be rejected as not_regular_file, got %q", inventoryCode(err))
	}
}

// mkJunction creates a Windows directory junction (no privilege needed); elsewhere, or when the
// host refuses, the calling subtest is skipped so the machine-readable result shows the omission.
func mkJunction(t *testing.T, link, target string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("directory junctions exist only on Windows")
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Skipf("cannot create a junction on this host: %v %s", err, out)
	}
}

func mkSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable on this host (platform/privilege): %v", err)
	}
}

// R041-R1-02 and R041-R1-03: links are rejected at the final element and at every ancestor,
// through the loader and through the mounted CLI. Subtests that need a capability the host lacks
// (symlinks without privilege, junctions off Windows) are real t.Skip subtests, so a skipped
// symlink probe is visible in the test results and never counted as exercised coverage.
func TestInventoryLinkAdmission(t *testing.T) {
	root := realDir(t)
	realSub := filepath.Join(root, "real")
	if err := os.Mkdir(realSub, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(realSub, "inventory.json")
	if err := os.WriteFile(file, []byte(validInventoryJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	wantRejected := func(t *testing.T, path, code string) {
		t.Helper()
		if _, err := loadInventoryFile(path); inventoryCode(err) != code {
			t.Fatalf("loader: want %q, got %v (%q)", code, err, inventoryCode(err))
		}
		r := execMain(t, t.TempDir(), nil, base("-inventory", path)...)
		if r.code != 2 || r.out != "" || r.err != inventoryRejected {
			t.Fatalf("mounted CLI: want exit 2, empty stdout and the fixed message, got %d %q %q", r.code, r.out, r.err)
		}
	}

	t.Run("real-path-control", func(t *testing.T) {
		if _, err := loadInventoryFile(file); err != nil {
			t.Fatalf("a plain file in a plain directory must be admitted: %v", err)
		}
		if r := execMain(t, t.TempDir(), nil, base("-inventory", file)...); r.code != 0 {
			t.Fatalf("mounted CLI control: exit %d %q", r.code, r.err)
		}
	})
	t.Run("relative-paths-resolve-lexically", func(t *testing.T) {
		for _, rel := range []string{"real/inventory.json", filepath.Join("real", "..", "real", "inventory.json")} {
			if r := execMain(t, root, nil, base("-inventory", rel)...); r.code != 0 {
				t.Errorf("relative %q: exit %d %q", rel, r.code, r.err)
			}
		}
	})
	t.Run("symlink-final-file", func(t *testing.T) {
		link := filepath.Join(root, "link.json")
		mkSymlink(t, file, link)
		wantRejected(t, link, "link_component")
	})
	t.Run("symlink-ancestor-directory", func(t *testing.T) {
		link := filepath.Join(root, "dirlink")
		mkSymlink(t, realSub, link)
		wantRejected(t, filepath.Join(link, "inventory.json"), "link_component")
	})
	t.Run("junction-ancestor-directory", func(t *testing.T) {
		junction := filepath.Join(root, "junction")
		mkJunction(t, junction, realSub)
		through := filepath.Join(junction, "inventory.json")
		wantRejected(t, through, "link_component")
		// Relative spelling reaches the same junction from the working directory.
		r := execMain(t, root, nil, base("-inventory", "junction/inventory.json")...)
		if r.code != 2 || r.out != "" || r.err != inventoryRejected {
			t.Fatalf("relative path through a junction: want exit 2, empty stdout and the fixed message, got %d %q %q", r.code, r.out, r.err)
		}
		// The very same file by its real path is still accepted: the rejection is the junction.
		if r := execMain(t, t.TempDir(), nil, base("-inventory", file)...); r.code != 0 {
			t.Fatalf("real path to the same file: exit %d", r.code)
		}
	})
	t.Run("junction-nested-in-the-middle", func(t *testing.T) {
		outer := filepath.Join(root, "outer")
		if err := os.Mkdir(outer, 0o700); err != nil {
			t.Fatal(err)
		}
		mkJunction(t, filepath.Join(outer, "hop"), realSub)
		wantRejected(t, filepath.Join(outer, "hop", "inventory.json"), "link_component")
	})
	t.Run("junction-as-final-element", func(t *testing.T) {
		junction := filepath.Join(root, "junction-final")
		mkJunction(t, junction, realSub)
		wantRejected(t, junction, "link_component")
	})
}

// OI-04: the file is the expectation. Any divergence from what the transcript yields is an
// ordinary FAIL/INCOMPLETE, never a regenerated transcript or an ignored file.
func TestInventoryExpectationsStayIndependentOfObservations(t *testing.T) {
	baseRun := runInventory(t, validInventoryJSON)
	baseDigest := digestOf(t, baseRun)

	variants := map[string]string{
		"extra-expected-conversation": strings.Replace(validInventoryJSON, `"conversations": [`, `"conversations": [
    {"id": "syn-conv-z", "updated_at": "2026-09-24T11:00:00Z", "messages": [{"id": "syn-msg-z1", "sent_at": "2026-09-24T11:00:00Z", "sender_type": "customer", "content_type": "text", "attachments": []}]},`, 1),
		"conversation-removed": mutate(t, `,
    {"id": "syn-conv-b", "updated_at": "2026-09-20T00:00:00Z", "messages": [
      {"id": "syn-msg-b1", "sent_at": "2026-09-20T00:00:00Z", "sender_type": "customer", "content_type": "text", "attachments": []}
    ]}`, ``),
		"message-removed":    mutate(t, `{"id": "syn-msg-a1", "sent_at": "2026-09-24T09:00:00Z", "sender_type": "customer", "content_type": "text", "attachments": []},`, ``),
		"sender-differs":     mutate(t, `"sender_type": "agent"`, `"sender_type": "customer"`),
		"content-differs":    mutate(t, `"content_type": "attachment"`, `"content_type": "text"`),
		"timestamp-differs":  mutate(t, `"2026-09-24T09:05:00Z"`, `"2026-09-24T09:05:01Z"`),
		"attachment-differs": mutate(t, `"photo-001.jpg"`, `"photo-002.jpg"`),
		"updated-differs":    mutate(t, `"2026-09-24T10:00:00Z"`, `"2026-09-24T10:00:01Z"`),
		"message-added": mutate(t, `{"id": "syn-msg-b1",`, `{"id": "syn-msg-b0", "sent_at": "2026-09-19T00:00:00Z", "sender_type": "customer", "content_type": "text", "attachments": []},
      {"id": "syn-msg-b1",`),
	}
	digests := map[string]string{baseDigest: "base"}
	for name, body := range variants {
		r := runInventory(t, body)
		if r.code != 1 || !(strings.Contains(r.out, `"disposition": "FAIL"`) || strings.Contains(r.out, `"disposition": "INCOMPLETE"`)) || strings.Contains(r.out, `"disposition": "PASS"`) {
			t.Errorf("%s: a divergent expectation must not PASS, got exit %d\n%s", name, r.code, r.out)
		}
		d := digestOf(t, r)
		if prev, dup := digests[d]; dup {
			t.Errorf("%s: digest equals %s; the caller digest must change with the expected data", name, prev)
		}
		digests[d] = name
	}

	empty := runInventory(t, `{"schema_version":"pancake-proof-synthetic-inventory/1","evidence_type":"SYNTHETIC_OFFLINE","provenance":"synthetic-empty","page_id":"synthetic-page-001","conversations":[]}`)
	// Against the unchanged transcript, which does yield rows, an empty expectation is a FAIL with
	// observed rows reported as extra. (INCOMPLETE/empty_inventory needs an empty observation too,
	// which is the built-in empty-inventory scenario and cannot be combined with -inventory.)
	if empty.code != 1 || !strings.Contains(empty.out, `"disposition": "FAIL"`) || !strings.Contains(empty.out, "reconciliation_mismatch") ||
		strings.Contains(empty.out, `"disposition": "PASS"`) || !strings.Contains(empty.out, `"attempts_used": 4`) {
		t.Errorf("an empty expectation must never PASS against observed rows, got %d\n%s", empty.code, empty.out)
	}

	// Equivalent instants (different offsets) and reordered conversations/messages are the same
	// expectation: same digest, same PASS.
	reordered := `{"schema_version":"pancake-proof-synthetic-inventory/1","evidence_type":"SYNTHETIC_OFFLINE","provenance":"synthetic-file-v1","page_id":"synthetic-page-001","conversations":[
    {"id":"syn-conv-b","updated_at":"2026-09-20T00:00:00Z","messages":[{"id":"syn-msg-b1","sent_at":"2026-09-20T00:00:00Z","sender_type":"customer","content_type":"text","attachments":[]}]},
    {"id":"syn-conv-a","updated_at":"2026-09-24T10:00:00Z","messages":[
      {"id":"syn-msg-a2","sent_at":"2026-09-24T09:05:00Z","sender_type":"agent","content_type":"attachment","attachments":[{"type":"image","name":"photo-001.jpg"}]},
      {"id":"syn-msg-a1","sent_at":"2026-09-24T09:00:00Z","sender_type":"customer","content_type":"text","attachments":[]}]}]}`
	if r := runInventory(t, reordered); r.code != 0 || digestOf(t, r) != baseDigest {
		t.Errorf("reordered rows must give the same digest and PASS (exit %d)", r.code)
	}
	offsets := strings.NewReplacer(`"2026-09-24T10:00:00Z"`, `"2026-09-24T17:00:00+07:00"`, `"2026-09-24T09:00:00Z"`, `"2026-09-24T04:00:00-05:00"`)
	if r := runInventory(t, offsets.Replace(validInventoryJSON)); r.code != 0 || digestOf(t, r) != baseDigest {
		t.Errorf("equivalent offsets must compare as the same instants (exit %d)", r.code)
	}
}

// OI-05
func TestInventoryInputNeverAppearsInOutput(t *testing.T) {
	canaries := []string{"CANARY-CONV-5521", "CANARY-MSG-5521", "CANARY-NAME-5521", "CANARY-KEY-5521", "CANARY-VALUE-5521", "CANARY-DIR-5521"}
	dir := filepath.Join(t.TempDir(), "CANARY-DIR-5521")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "CANARY-FILE-5521.json")

	// A well-formed file whose rows the transcript cannot produce: FAIL receipt, no raw values.
	body := strings.Replace(validInventoryJSON, `"conversations": [`, `"conversations": [
    {"id": "syn-CANARY-CONV-5521", "updated_at": "2026-09-24T11:00:00Z", "messages": [{"id": "syn-CANARY-MSG-5521", "sent_at": "2026-09-24T11:00:00Z", "sender_type": "customer", "content_type": "attachment", "attachments": [{"type": "image", "name": "CANARY-NAME-5521.jpg"}]}]},`, 1)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	r := execMain(t, t.TempDir(), nil, base("-inventory", path)...)
	if r.code != 1 || !strings.Contains(r.out, `"disposition": "FAIL"`) {
		t.Fatalf("canary expectations must FAIL, got %d\n%s", r.code, r.out)
	}
	for _, leak := range append(canaries, "CANARY-FILE-5521", dir, path) {
		if strings.Contains(r.out+r.err, leak) {
			t.Errorf("receipt/stderr leaks %q", leak)
		}
	}

	// Rejected inputs: fixed stderr only, no key/value/path/OS or parser text.
	bad := map[string]string{
		"unknown-key":   strings.Replace(validInventoryJSON, `"provenance":`, `"CANARY-KEY-5521": "CANARY-VALUE-5521", "provenance":`, 1),
		"bad-id-value":  strings.Replace(validInventoryJSON, `"syn-conv-a"`, `"CANARY-CONV-5521/../x"`, 1),
		"syntax-error":  `{"CANARY-VALUE-5521" ` + validInventoryJSON,
		"duplicate-key": strings.Replace(validInventoryJSON, `"page_id": "synthetic-page-001",`, `"page_id": "synthetic-page-001", "page_id": "CANARY-VALUE-5521",`, 1),
	}
	for name, b := range bad {
		if err := os.WriteFile(path, []byte(b), 0o600); err != nil {
			t.Fatal(err)
		}
		r := execMain(t, t.TempDir(), nil, base("-inventory", path)...)
		if r.code != 2 || r.out != "" || r.err != inventoryRejected {
			t.Errorf("%s: want exit 2, empty stdout and the fixed message, got %d %q %q", name, r.code, r.out, r.err)
		}
	}
	for _, missing := range []string{filepath.Join(dir, "CANARY-MISSING-5521.json"), dir} {
		r := execMain(t, t.TempDir(), nil, base("-inventory", missing)...)
		if r.code != 2 || r.out != "" || r.err != inventoryRejected {
			t.Errorf("unavailable file: want the fixed message only, got %d %q %q", r.code, r.out, r.err)
		}
	}
	if err := os.WriteFile(path, []byte(validInventoryJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != validInventoryJSON {
		t.Errorf("the input file must never be modified")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("the CLI must not create files next to the input: %d entries", len(entries))
	}

	// Poisoned credential/config environment still has no effect with an explicit inventory.
	poison := "CANARY-POISON-5521"
	env := []string{"PANCAKE_PAGE_ACCESS_TOKEN=" + poison, "DATABASE_URL=mysql://" + poison + "@127.0.0.1:1/x", "HTTPS_PROXY=http://127.0.0.1:1", "ANTHROPIC_API_KEY=" + poison}
	clean := execMain(t, t.TempDir(), nil, base("-inventory", path)...)
	poisoned := execMain(t, t.TempDir(), env, base("-inventory", path)...)
	if poisoned.code != clean.code || normalize(poisoned.out) != normalize(clean.out) || poisoned.err != clean.err || strings.Contains(poisoned.out+poisoned.err, poison) {
		t.Errorf("environment changed behavior with -inventory")
	}

	// Unsupported live/token/config/network options remain rejected next to -inventory.
	for _, extra := range [][]string{{"-live"}, {"-token", poison}, {"-config", poison}, {"-base-url", "https://x.invalid"}, {"-db", poison}} {
		r := execMain(t, t.TempDir(), nil, base(append([]string{"-inventory", path}, extra...)...)...)
		if r.code != 2 || r.out != "" || strings.Contains(r.err, poison) {
			t.Errorf("%v must stay rejected, got %d %q %q", extra, r.code, r.out, r.err)
		}
	}
}
