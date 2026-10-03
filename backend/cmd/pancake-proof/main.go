// Command pancake-proof runs the offline Pancake proof harness (CCMAI-RUNTIME-034) against a
// finite, deterministic, in-memory transcript. It has no live mode: it never reads credentials,
// configuration, .env files or the environment, never opens a network connection or database,
// and never downloads media. Its receipt is SYNTHETIC_OFFLINE evidence only.
//
// Usage:
//
//	pancake-proof -source-sha <40-hex> -key <nonempty> [-scenario pass|missing-conversation|redirect|empty-inventory]
//	              [-max-attempts N (1..50)] [-max-duration D (e.g. 30s, <=10m)]
//	              [-inventory PATH]
//
// -inventory loads a caller-authored synthetic expected inventory (schema
// pancake-proof-synthetic-inventory/1) from one explicit local file, only for the pass scenario.
// It replaces the expectations, never the transcript. Without it behavior is unchanged.
//
// Exit codes: 0 offline PASS; 1 FAIL or INCOMPLETE; 2 invalid usage/input or sanitation failure.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
)

const (
	fixturePage  = "synthetic-page-001"
	fixtureToken = "synthetic-offline-token-not-a-credential"
	banner       = "pancake-proof: SYNTHETIC OFFLINE receipt from an in-memory transcript; not live-channel, provider or AI-governance evidence"
)

var fixtureSince = time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// pathFlag records the explicit -inventory value and how often the option was given.
type pathFlag struct {
	value string
	count int
}

func (p *pathFlag) String() string { return "" }
func (p *pathFlag) Set(v string) error {
	p.value, p.count = v, p.count+1
	return nil
}

func run(args []string, stdout, stderr io.Writer) (code int) {
	defer func() {
		if recover() != nil { // panic text may carry arbitrary values: report a fixed message only
			fmt.Fprintln(stderr, "pancake-proof: internal error")
			code = 2
		}
	}()

	fs := flag.NewFlagSet("pancake-proof", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // never echo option text back
	scenario := fs.String("scenario", "pass", "pass|missing-conversation|redirect|empty-inventory")
	sha := fs.String("source-sha", "", "40-hex source revision")
	key := fs.String("key", "", "nonempty pseudonymization key")
	attempts := fs.Int("max-attempts", 20, "aggregate request cap (1..50)")
	dur := fs.Duration("max-duration", time.Minute, "aggregate duration cap (<=10m)")
	var inventoryPath pathFlag
	fs.Var(&inventoryPath, "inventory", "explicit local synthetic expected-inventory JSON file (pass scenario only)")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || inventoryPath.count > 1 {
		fmt.Fprintln(stderr, "pancake-proof: invalid usage; offline-only tool with options -scenario -source-sha -key -max-attempts -max-duration -inventory")
		return 2
	}
	if inventoryPath.count == 1 && *scenario != "pass" { // rejected before the file is touched
		fmt.Fprintln(stderr, "pancake-proof: -inventory is supported only with the pass scenario")
		return 2
	}

	transcript, inv, ok := buildFixture(*scenario)
	if !ok {
		fmt.Fprintln(stderr, "pancake-proof: unknown scenario")
		return 2
	}
	if inventoryPath.count == 1 {
		loaded, err := loadInventoryFile(inventoryPath.value)
		if err != nil {
			fmt.Fprintln(stderr, "pancake-proof: inventory rejected")
			return 2
		}
		inv = loaded // expectations only; the transcript is never derived from the file
	}
	opts := channels.PancakeProofOptions{
		SourceSHA: *sha, PageID: fixturePage, Token: fixtureToken, PseudonymKey: []byte(*key), Since: fixtureSince,
		Transport: transcript, MaxAttempts: *attempts, MaxDuration: *dur,
	}
	receipt, err := channels.RunPancakeProof(context.Background(), opts, inv)
	if err != nil {
		fmt.Fprintln(stderr, "pancake-proof: input rejected")
		return 2
	}
	out, err := channels.MarshalProofReceipt(receipt, channels.ProofSensitiveValues(opts, inv))
	if err != nil {
		fmt.Fprintln(stderr, "pancake-proof: receipt failed sanitation")
		return 2
	}
	fmt.Fprintln(stderr, banner)
	stdout.Write(out)
	if receipt.Disposition == channels.ProofPass {
		return 0
	}
	return 1
}

func row(id, typ, updated string) string {
	return fmt.Sprintf(`{"id":%q,"type":%q,"updated_at":%q,"from":{"id":"cust-%s","name":"Synthetic %s"}}`, id, typ, updated, id, id)
}

func msg(id, from, text, at, atts string) string {
	if atts == "" {
		atts = "[]"
	}
	return fmt.Sprintf(`{"id":%q,"page_id":%q,"original_message":%q,"from":{"id":%q,"name":"S"},"inserted_at":%q,"attachments":%s}`, id, fixturePage, text, from, at, atts)
}

func list(field string, rows ...string) channels.ProofResponse {
	return channels.ProofResponse{Status: 200, Body: `{"success":true,"` + field + `":[` + strings.Join(rows, ",") + `]}`}
}

// buildFixture returns the finite transcript and its independently written expectation.
func buildFixture(scenario string) (*channels.ProofTranscript, *channels.ProofInventory, bool) {
	att := `[{"id":"a1","type":"image","url":"https://cdn.example.invalid/synthetic/photo-001.jpg"}]`
	routes := map[string][]channels.ProofResponse{
		"conv:": {list("conversations",
			row("syn-conv-a", "INBOX", "2026-09-24T10:00:00.000000"),
			row("syn-conv-b", "INBOX", "2026-09-20T00:00:00.000000"),
			row("syn-conv-old", "INBOX", "2026-09-01T00:00:00.000000"))},
		"conv:syn-conv-old": {list("conversations")},
		"msg:syn-conv-a:0": {list("messages",
			msg("syn-msg-a1", "cust-syn-conv-a", "synthetic hello", "2026-09-24T09:00:00.000000", ""),
			msg("syn-msg-a2", fixturePage, "synthetic reply", "2026-09-24T09:05:00.000000", att))},
		"msg:syn-conv-a:2": {list("messages")},
		"msg:syn-conv-b:0": {list("messages", msg("syn-msg-b1", "cust-syn-conv-b", "synthetic boundary", "2026-09-20T00:00:00.000000", ""))},
		"msg:syn-conv-b:1": {list("messages")},
	}
	inv := &channels.ProofInventory{Provenance: "synthetic-cli-fixture-v1", PageID: fixturePage, Conversations: []channels.ProofConversation{
		{ID: "syn-conv-a", UpdatedAt: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), Messages: []channels.ProofMessage{
			{ID: "syn-msg-a1", SentAt: time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC), SenderType: "customer", ContentType: "text"},
			{ID: "syn-msg-a2", SentAt: time.Date(2026, 9, 24, 9, 5, 0, 0, time.UTC), SenderType: "agent", ContentType: "attachment",
				Attachments: []channels.ProofAttachment{{Type: "image", Name: "photo-001.jpg"}}},
		}},
		{ID: "syn-conv-b", UpdatedAt: fixtureSince, Messages: []channels.ProofMessage{
			{ID: "syn-msg-b1", SentAt: fixtureSince, SenderType: "customer", ContentType: "text"},
		}},
	}}
	switch scenario {
	case "pass":
	case "missing-conversation":
		inv.Conversations = append(inv.Conversations, channels.ProofConversation{ID: "syn-conv-missing", UpdatedAt: fixtureSince,
			Messages: []channels.ProofMessage{{ID: "syn-msg-m1", SentAt: fixtureSince, SenderType: "customer", ContentType: "text"}}})
	case "redirect":
		routes["conv:"] = []channels.ProofResponse{{Status: 302, Location: "https://pages.fm/elsewhere"}}
	case "empty-inventory":
		routes = map[string][]channels.ProofResponse{"conv:": {list("conversations")}}
		inv.Conversations = nil
	default:
		return nil, nil, false
	}
	return &channels.ProofTranscript{Routes: routes, Repeat: true, Ceiling: 200}, inv, true
}
