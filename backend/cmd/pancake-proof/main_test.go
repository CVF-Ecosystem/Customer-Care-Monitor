package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const goodSHA = "d869624cc15f55b39516a36f8937454e617dc3b3"

// TestMain lets tests execute the real main() (entrypoint, flag parsing, os.Exit) by re-running
// this test binary with PANCAKE_PROOF_AS_MAIN=1; no network or credential is involved.
func TestMain(m *testing.M) {
	if os.Getenv("PANCAKE_PROOF_AS_MAIN") == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

type result struct {
	out, err string
	code     int
}

func execMain(t *testing.T, dir string, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = dir
	cmd.Env = append([]string{"PANCAKE_PROOF_AS_MAIN=1", "SystemRoot=" + os.Getenv("SystemRoot")}, env...)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("could not run helper: %v", err)
		}
		code = ee.ExitCode()
	}
	return result{o.String(), e.String(), code}
}

var volatile = []*regexp.Regexp{
	regexp.MustCompile(`"(started_at|finished_at)": "[^"]+"`),
	regexp.MustCompile(`"(conversation_until|elapsed_ms)": \d+`),
}

func normalize(s string) string {
	for _, re := range volatile {
		s = re.ReplaceAllString(s, `"$1": "X"`)
	}
	return s
}

func base(extra ...string) []string {
	return append([]string{"-source-sha", goodSHA, "-key", "cli-test-key-0042"}, extra...)
}

func TestCLIPassReceiptIsLabelledSynthetic(t *testing.T) {
	r := execMain(t, t.TempDir(), nil, base()...)
	if r.code != 0 {
		t.Fatalf("offline PASS must exit 0, got %d stderr=%q", r.code, r.err)
	}
	for _, want := range []string{`"evidence_type": "SYNTHETIC_OFFLINE"`, `"live": false`, `"governance_claim": false`, `"disposition": "PASS"`, `"attempts_used": 12`, `"source_sha": "` + goodSHA + `"`} {
		if !strings.Contains(r.out, want) {
			t.Errorf("receipt missing %s\n%s", want, r.out)
		}
	}
	if !strings.Contains(r.err, "SYNTHETIC OFFLINE") || !strings.Contains(r.err, "not live-channel") {
		t.Errorf("stderr banner must label the output synthetic: %q", r.err)
	}
	for _, leak := range []string{"syn-conv-a", "syn-msg-a1", "synthetic hello", "photo-001", "synthetic-offline-token", "cli-test-key-0042", "synthetic-page-001", "cdn.example.invalid"} {
		if strings.Contains(r.out+r.err, leak) {
			t.Errorf("output leaks raw value %q", leak)
		}
	}
}

func TestCLINonPassExitCodes(t *testing.T) {
	for scenario, want := range map[string]string{
		"missing-conversation": `"disposition": "FAIL"`,
		"redirect":             `"disposition": "INCOMPLETE"`,
		"empty-inventory":      `"disposition": "INCOMPLETE"`,
	} {
		r := execMain(t, t.TempDir(), nil, base("-scenario", scenario)...)
		if r.code != 1 || !strings.Contains(r.out, want) {
			t.Errorf("%s: want exit 1 with %s, got exit %d\n%s", scenario, want, r.code, r.out)
		}
	}
	r := execMain(t, t.TempDir(), nil, base("-max-attempts", "11")...)
	if r.code != 1 || !strings.Contains(r.out, "budget_exhausted") || !strings.Contains(r.out, `"attempts_used": 11`) {
		t.Errorf("budget one short must be INCOMPLETE with 11 attempts: exit %d\n%s", r.code, r.out)
	}
	r = execMain(t, t.TempDir(), nil, base("-max-attempts", "12")...)
	if r.code != 0 {
		t.Errorf("exactly the needed 12 attempts must pass, exit %d", r.code)
	}
}

func TestCLIInvalidInputRejectedWithoutReceipt(t *testing.T) {
	canary := "CANARY-CLI-7741"
	cases := map[string][]string{
		"no-args":            {},
		"bad-sha":            {"-source-sha", "abc", "-key", "k"},
		"missing-sha":        {"-key", "k"},
		"missing-key":        {"-source-sha", goodSHA},
		"attempts-zero":      base("-max-attempts", "0"),
		"attempts-over-50":   base("-max-attempts", "51"),
		"duration-over-10m":  base("-max-duration", "11m"),
		"duration-zero":      base("-max-duration", "0s"),
		"unknown-scenario":   base("-scenario", "live"),
		"positional":         base("extra"),
		"flag-live":          base("-live"),
		"flag-token":         base("-token", canary),
		"flag-token-eq":      base("-token=" + canary),
		"flag-page-token":    base("-page-access-token", canary),
		"flag-config":        base("-config", canary),
		"flag-env-file":      base("-env-file", canary),
		"flag-db":            base("-db", canary),
		"flag-base-url":      base("-base-url", "https://"+canary+".test"),
		"flag-transport":     base("-transport", "network"),
		"flag-download":      base("-download-media"),
		"flag-out":           base("-out", filepath.Join(os.TempDir(), canary)),
		"malformed-duration": base("-max-duration", canary),
	}
	for name, args := range cases {
		dir := t.TempDir()
		r := execMain(t, dir, nil, args...)
		if r.code != 2 || r.out != "" {
			t.Errorf("%s: want exit 2 and empty stdout, got %d %q", name, r.code, r.out)
		}
		if strings.Contains(r.err, canary) {
			t.Errorf("%s: stderr echoes caller input", name)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Errorf("%s: created files in the working directory", name)
		}
	}
	if _, err := os.Stat(filepath.Join(os.TempDir(), canary)); err == nil {
		t.Errorf("an -out style flag created a file")
	}
}

func TestCLIPoisonedCredentialAndConfigEnvironmentIsIgnored(t *testing.T) {
	clean := execMain(t, t.TempDir(), nil, base()...)

	dir := t.TempDir()
	poison := "CANARY-POISON-9902"
	files := map[string]string{
		".env":        "PANCAKE_PAGE_ACCESS_TOKEN=" + poison + "\nDATABASE_URL=mysql://" + poison + "@127.0.0.1:1/x\n",
		"config.json": `{"page_access_token":"` + poison + `","base_url":"https://` + poison + `.invalid"}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{
		"PANCAKE_PAGE_ACCESS_TOKEN=" + poison, "PANCAKE_PAGE_ID=" + poison, "DATABASE_URL=mysql://" + poison + "@127.0.0.1:1/x",
		"DB_HOST=127.0.0.1", "HTTPS_PROXY=http://127.0.0.1:1", "HTTP_PROXY=http://127.0.0.1:1", "ALL_PROXY=http://127.0.0.1:1",
		"SSL_CERT_FILE=" + filepath.Join(dir, "missing.pem"), "ANTHROPIC_API_KEY=" + poison, "CVF_PROVIDER_KEY=" + poison,
		"JWT_SECRET=" + poison, "GODEBUG=http2debug=2",
	}
	poisoned := execMain(t, dir, env, base()...)

	if poisoned.code != clean.code || normalize(poisoned.out) != normalize(clean.out) || poisoned.err != clean.err {
		t.Fatalf("environment changed behavior:\nclean:    %d %q\npoisoned: %d %q", clean.code, normalize(clean.out), poisoned.code, normalize(poisoned.out))
	}
	if strings.Contains(poisoned.out+poisoned.err, poison) {
		t.Fatalf("poisoned value surfaced")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != len(files) {
		t.Fatalf("working directory changed: %d entries", len(entries))
	}
	for name, body := range files {
		if got, _ := os.ReadFile(filepath.Join(dir, name)); string(got) != body {
			t.Fatalf("%s was modified", name)
		}
	}
}

func TestCLIEntrypointAndRunAgree(t *testing.T) {
	var o, e bytes.Buffer
	code := run(base(), &o, &e)
	sub := execMain(t, t.TempDir(), nil, base()...)
	if code != sub.code || normalize(o.String()) != normalize(sub.out) {
		t.Fatalf("run() and mounted main() differ: %d vs %d", code, sub.code)
	}
}

func TestCLIFixtureIsFiniteAndIndependent(t *testing.T) {
	tp, inv, ok := buildFixture("pass")
	if !ok || tp.Ceiling <= 0 || tp.Ceiling > 500 {
		t.Fatalf("fixture must carry a hard ceiling")
	}
	if len(inv.Conversations) != 2 {
		t.Fatalf("inventory expectation must be written independently of observed output")
	}
	if _, _, ok := buildFixture("nope"); ok {
		t.Fatalf("unknown scenarios must be rejected")
	}
}
