package channels

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// All values below are synthetic. Nothing here reaches a network, credential store or database.

const (
	pfPage  = "page-proof-alpha"
	pfToken = "tok-CANARY-7f3a91c2"
	pfSHA   = "d869624cc15f55b39516a36f8937454e617dc3b3"
)

var (
	pfKey   = []byte("key-CANARY-0042aa77")
	pfSince = time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
)

func pfConvRow(id, typ, updated string) string {
	return fmt.Sprintf(`{"id":%q,"type":%q,"page_id":%q,"updated_at":%q,"inserted_at":"2026-09-01T00:00:00.000000","from":{"id":"u-%s","name":"Khach %s"}}`, id, typ, pfPage, updated, id, id)
}

func pfMsgRow(id, from, text, at, atts string) string {
	if atts == "" {
		atts = "[]"
	}
	return fmt.Sprintf(`{"id":%q,"type":"INBOX","page_id":%q,"original_message":%q,"from":{"id":%q,"name":"N"},"inserted_at":%q,"attachments":%s}`, id, pfPage, text, from, at, atts)
}

func pfList(field string, rows ...string) string {
	return `{"success":true,"` + field + `":[` + strings.Join(rows, ",") + `]}`
}

func pfOK(body string) ProofResponse { return ProofResponse{Status: 200, Body: body} }

// pfFixture is a complete multi-page provider transcript plus the independently authored inventory.
// Per run: 3 conversation pages + 3 + 2 + 2 message pages = 10 requests.
func pfFixture() (*ProofTranscript, *ProofInventory) {
	imgAtt := `[{"id":"a1","type":"image","url":"https://cdn.example.test/img/IMG-CANARY-9911.jpg"}]`
	stkAtt := `[{"id":"a2","type":"sticker","url":"https://cdn.example.test/s/stk.png"}]`
	r := map[string][]ProofResponse{
		"conv:": {pfOK(pfList("conversations",
			pfConvRow("c-aaa1", "INBOX", "2026-09-24T10:00:00.000000"),
			pfConvRow("c-bbb2", "INBOX", "2026-09-20T00:00:00.000000"), // equal to since: included
			pfConvRow("c-old3", "INBOX", "2026-09-01T00:00:00.000000"), // old: excluded
		))},
		"conv:c-old3": {pfOK(pfList("conversations",
			pfConvRow("c-aaa1", "INBOX", "2026-09-24T10:00:00.000000"), // duplicate
			pfConvRow("c-cmt4", "COMMENT", "2026-09-24T10:00:00.000000"),
			pfConvRow("c-ddd5", "INBOX", "2026-09-25T01:00:00+07:00"),
		))},
		"conv:c-ddd5": {pfOK(pfList("conversations"))},

		"msg:c-aaa1:0": {pfOK(pfList("messages",
			pfMsgRow("m-1", "u-c-aaa1", "xin chao SECRET-TEXT-1", "2026-09-24T09:00:00.000000", ""),
			pfMsgRow("m-2", pfPage, "da nhan", "2026-09-24T09:05:00.000000", imgAtt),
			pfMsgRow("m-old", "u-c-aaa1", "tin cu", "2026-09-10T09:00:00.000000", ""),
		))},
		"msg:c-aaa1:3": {pfOK(pfList("messages",
			pfMsgRow("m-1", "u-c-aaa1", "xin chao SECRET-TEXT-1", "2026-09-24T09:00:00.000000", ""), // duplicate
			pfMsgRow("m-3", "u-c-aaa1", "", "2026-09-24T09:10:00.000000", stkAtt),
		))},
		"msg:c-aaa1:5": {pfOK(pfList("messages"))},
		"msg:c-bbb2:0": {pfOK(pfList("messages", pfMsgRow("m-b1", "u-c-bbb2", "boundary", "2026-09-20T00:00:00.000000", "")))},
		"msg:c-bbb2:1": {pfOK(pfList("messages"))},
		"msg:c-ddd5:0": {pfOK(pfList("messages", pfMsgRow("m-d1", "u-c-ddd5", "zone", "2026-09-24T12:00:00.000000", "")))},
		"msg:c-ddd5:1": {pfOK(pfList("messages"))},
	}
	tp := &ProofTranscript{Routes: r, Repeat: true}
	inv := &ProofInventory{Provenance: "synthetic-r034-fixture", PageID: pfPage, Conversations: []ProofConversation{
		{ID: "c-aaa1", UpdatedAt: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), Messages: []ProofMessage{
			{ID: "m-1", SentAt: time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC), SenderType: "customer", ContentType: "text"},
			{ID: "m-2", SentAt: time.Date(2026, 9, 24, 9, 5, 0, 0, time.UTC), SenderType: "agent", ContentType: "attachment",
				Attachments: []ProofAttachment{{Type: "image", Name: "IMG-CANARY-9911.jpg"}}},
			{ID: "m-3", SentAt: time.Date(2026, 9, 24, 9, 10, 0, 0, time.UTC), SenderType: "customer", ContentType: "sticker",
				Attachments: []ProofAttachment{{Type: "sticker", Name: "stk.png"}}},
		}},
		{ID: "c-bbb2", UpdatedAt: pfSince, Messages: []ProofMessage{
			{ID: "m-b1", SentAt: pfSince, SenderType: "customer", ContentType: "text"},
		}},
		{ID: "c-ddd5", UpdatedAt: time.Date(2026, 9, 24, 18, 0, 0, 0, time.UTC), Messages: []ProofMessage{
			{ID: "m-d1", SentAt: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC), SenderType: "customer", ContentType: "text"},
		}},
	}}
	return tp, inv
}

func pfOpts(rt http.RoundTripper) PancakeProofOptions {
	tick := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	return PancakeProofOptions{
		SourceSHA: pfSHA, PageID: pfPage, Token: pfToken, PseudonymKey: pfKey, Since: pfSince, Transport: rt,
		MaxAttempts: 20, MaxDuration: time.Minute, RetryBackoff: time.Millisecond,
		Now: func() time.Time { tick = tick.Add(time.Millisecond); return tick },
	}
}

func pfRun(t *testing.T, tp *ProofTranscript, inv *ProofInventory, mut func(*PancakeProofOptions)) *ProofReceipt {
	t.Helper()
	o := pfOpts(tp)
	if mut != nil {
		mut(&o)
	}
	r, err := RunPancakeProof(context.Background(), o, inv)
	if err != nil {
		t.Fatalf("unexpected input rejection: %v", err)
	}
	return r
}

func pfRef(domain, v string) string {
	m := hmac.New(sha256.New, pfKey)
	m.Write([]byte("cvf-pancake-proof/v1\x00" + domain + "\x00" + v))
	return domain[:3] + "_" + hex.EncodeToString(m.Sum(nil))[:16]
}

func pfReasons(r *ProofReceipt) string { return strings.Join(r.Reasons, ",") }

// ---------------------------------------------------------------- PH-01 / PH-04

func TestProofPH01TwoRunsExactSequence(t *testing.T) {
	tp, inv := pfFixture()
	r := pfRun(t, tp, inv, nil)
	if r.Disposition != ProofPass || len(r.Runs) != 2 {
		t.Fatalf("want PASS with two runs, got %s %v", r.Disposition, r.Reasons)
	}
	if tp.Calls() != 20 || r.Budget.AttemptsUsed != 20 {
		t.Fatalf("want exactly 20 physical requests, transport=%d receipt=%d", tp.Calls(), r.Budget.AttemptsUsed)
	}
	type step struct {
		kind, conv, cursor string
		count              int
	}
	want := []step{
		{"conversations", "", "", 0},
		{"conversations", "", pfRef("cursor", "c-old3"), 0},
		{"conversations", "", pfRef("cursor", "c-ddd5"), 0},
		{"messages", pfRef("conversation", "c-aaa1"), "", 0},
		{"messages", pfRef("conversation", "c-aaa1"), "", 3},
		{"messages", pfRef("conversation", "c-aaa1"), "", 5},
		{"messages", pfRef("conversation", "c-bbb2"), "", 0},
		{"messages", pfRef("conversation", "c-bbb2"), "", 1},
		{"messages", pfRef("conversation", "c-ddd5"), "", 0},
		{"messages", pfRef("conversation", "c-ddd5"), "", 1},
	}
	for ri, run := range r.Runs {
		if run.Disposition != ProofPass || len(run.Requests) != len(want) {
			t.Fatalf("run %d: disposition %s with %d requests", ri+1, run.Disposition, len(run.Requests))
		}
		for i, rec := range run.Requests {
			w := want[i]
			if rec.Kind != w.kind || rec.ConversationRef != w.conv || rec.CursorRef != w.cursor || rec.CurrentCount != w.count || rec.Seq != ri*10+i+1 {
				t.Fatalf("run %d request %d mismatch: %+v want %+v", ri+1, i, rec, w)
			}
		}
		if !run.ConversationTerm || run.MessageTerminals != 3 || run.MessageFetches != 3 {
			t.Fatalf("terminal accounting wrong: %+v", run)
		}
		c, m := run.ConversationRows, run.MessageRows
		if c.PhysicalRows != 6 || c.DuplicateRow != 1 || c.OldRows != 1 || c.NonInboxRows != 1 || c.RawEligible != 3 || c.Mapped != 3 {
			t.Fatalf("conversation accounting %+v", c)
		}
		if m.PhysicalRows != 7 || m.DuplicateRow != 1 || m.OldRows != 1 || m.RawEligible != 5 || m.Mapped != 5 {
			t.Fatalf("message accounting %+v", m)
		}
		if run.ConversationUntil <= 0 || !run.UntilStable {
			t.Fatalf("actual until not recorded: %+v", run)
		}
	}
}

func TestProofPH04ReconciliationFailures(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(inv *ProofInventory, tp *ProofTranscript)
		check func(t *testing.T, r *ProofReceipt, tp *ProofTranscript)
	}{
		{"missing-conversation", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations = append(inv.Conversations, ProofConversation{ID: "c-zzz9", UpdatedAt: pfSince, Messages: []ProofMessage{{ID: "m-z", SentAt: pfSince, SenderType: "customer", ContentType: "text"}}})
		}, func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
			if got := r.Runs[0].Conversations.Missing; len(got) != 1 || got[0] != pfRef("conversation", "c-zzz9") {
				t.Fatalf("missing set %v", got)
			}
		}},
		{"missing-message", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations[1].Messages = append(inv.Conversations[1].Messages, ProofMessage{ID: "m-gone", SentAt: pfSince, SenderType: "customer", ContentType: "text"})
		}, func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
			if got := r.Runs[0].Messages.Missing; len(got) != 1 || got[0] != pfRef("message", "c-bbb2\x00m-gone") {
				t.Fatalf("missing messages %v", got)
			}
		}},
		{"extra-conversation-not-fetched", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations = inv.Conversations[:2] // c-ddd5 is observed but not expected
		}, func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
			if got := r.Runs[0].Conversations.Extra; len(got) != 1 || got[0] != pfRef("conversation", "c-ddd5") {
				t.Fatalf("extra set %v", got)
			}
			for _, rec := range r.Runs[0].Requests {
				if rec.ConversationRef == pfRef("conversation", "c-ddd5") {
					t.Fatalf("messages of an unexpected conversation must not be fetched: %+v", rec)
				}
			}
		}},
		{"extra-message", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations[0].Messages = inv.Conversations[0].Messages[:2]
		}, func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
			if got := r.Runs[0].Messages.Extra; len(got) != 1 || got[0] != pfRef("message", "c-aaa1\x00m-3") {
				t.Fatalf("extra messages %v", got)
			}
		}},
		{"wrong-sender", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations[0].Messages[1].SenderType = "customer"
		},
			func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
				pfWantMismatch(t, r, "c-aaa1\x00m-2", "sender_type")
			}},
		{"wrong-content-type", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations[0].Messages[2].ContentType = "attachment"
		},
			func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
				pfWantMismatch(t, r, "c-aaa1\x00m-3", "content_type")
			}},
		{"wrong-attachment", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations[0].Messages[1].Attachments[0].Type = "video"
		}, func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
			pfWantMismatch(t, r, "c-aaa1\x00m-2", "attachments")
		}},
		{"zoneless-timestamp-disagreement", func(inv *ProofInventory, _ *ProofTranscript) {
			// Provider really meant +07:00; the adapter reads zone-less text as UTC, so the instant disagrees.
			inv.Conversations[2].Messages[0].SentAt = time.Date(2026, 9, 24, 5, 0, 0, 0, time.UTC)
		}, func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
			pfWantMismatch(t, r, "c-ddd5\x00m-d1", "sent_at")
		}},
		{"conversation-instant", func(inv *ProofInventory, _ *ProofTranscript) {
			inv.Conversations[0].UpdatedAt = inv.Conversations[0].UpdatedAt.Add(7 * time.Hour)
		}, func(t *testing.T, r *ProofReceipt, _ *ProofTranscript) {
			if got := r.Runs[0].Conversations.Mismapped; len(got) != 1 || got[0].Fields[0] != "last_message_at" {
				t.Fatalf("conversation instant mismatch %v", got)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tp, inv := pfFixture()
			c.edit(inv, tp)
			r := pfRun(t, tp, inv, nil)
			if r.Disposition != ProofFail {
				t.Fatalf("want FAIL, got %s %v", r.Disposition, r.Reasons)
			}
			if !strings.Contains(pfReasons(r), "reconciliation_mismatch") {
				t.Fatalf("reason missing: %v", r.Reasons)
			}
			c.check(t, r, tp)
		})
	}
}

func pfWantMismatch(t *testing.T, r *ProofReceipt, key, field string) {
	t.Helper()
	ref := pfRef("message", key)
	for _, m := range r.Runs[0].Messages.Mismapped {
		if m.Ref == ref {
			for _, f := range m.Fields {
				if f == field {
					return
				}
			}
		}
	}
	t.Fatalf("expected mismatch %s on %s, got %+v", field, ref, r.Runs[0].Messages.Mismapped)
}

func TestProofPH04EmptyInventoryCannotPass(t *testing.T) {
	tp := &ProofTranscript{Routes: map[string][]ProofResponse{"conv:": {pfOK(pfList("conversations"))}}, Repeat: true}
	inv := &ProofInventory{Provenance: "synthetic-empty", PageID: pfPage}
	r := pfRun(t, tp, inv, nil)
	if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "empty_inventory") {
		t.Fatalf("empty inventory must be INCOMPLETE, got %s %v", r.Disposition, r.Reasons)
	}
}

func TestProofPH04InventoryWithoutMessagesCannotPass(t *testing.T) {
	tp, inv := pfFixture()
	tp.Routes = map[string][]ProofResponse{
		"conv:":        {pfOK(pfList("conversations", pfConvRow("c-aaa1", "INBOX", "2026-09-24T10:00:00.000000")))},
		"conv:c-aaa1":  {pfOK(pfList("conversations"))},
		"msg:c-aaa1:0": {pfOK(pfList("messages"))},
	}
	inv.Conversations = []ProofConversation{{ID: "c-aaa1", UpdatedAt: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)}}
	r := pfRun(t, tp, inv, nil)
	if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "empty_message_inventory") || r.Runs[0].Conversations.failing() {
		t.Fatalf("no expected messages must be INCOMPLETE, got %s %v", r.Disposition, r.Reasons)
	}
}

func TestProofPH04ExpectedButEmptyResultIsFail(t *testing.T) {
	tp, inv := pfFixture()
	tp.Routes = map[string][]ProofResponse{"conv:": {pfOK(pfList("conversations"))}}
	r := pfRun(t, tp, inv, nil)
	if r.Disposition != ProofFail || len(r.Runs[0].Conversations.Missing) != 3 {
		t.Fatalf("expected FAIL with 3 missing, got %s %+v", r.Disposition, r.Runs[0].Conversations)
	}
}

// ---------------------------------------------------------------- PH-02

func pfReq(t *testing.T, method, raw string) *http.Request {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Request{Method: method, URL: u, Header: http.Header{}, Host: u.Host}
}

func TestProofPH02AdmissionDeniesBeforeTransport(t *testing.T) {
	tp, inv := pfFixture()
	o := pfOpts(tp)
	tr := &proofTransport{inner: tp, opts: &o, approved: map[string]bool{}, now: o.Now, start: o.Now()}
	for _, c := range inv.Conversations {
		tr.approved[c.ID] = true
	}
	tr.beginRun(1)
	good := "https://pages.fm/api/public_api/v2/pages/" + pfPage + "/conversations?type=INBOX&order_by=updated_at&until=1790000000&since=" + fmt.Sprint(pfSince.Unix()) + "&page_access_token=" + pfToken
	goodMsg := "https://pages.fm/api/public_api/v1/pages/" + pfPage + "/conversations/c-aaa1/messages?page_access_token=" + pfToken
	denied := map[string]*http.Request{
		"post-method":        pfReq(t, "POST", good),
		"http-scheme":        pfReq(t, "GET", strings.Replace(good, "https://", "http://", 1)),
		"explicit-port":      pfReq(t, "GET", strings.Replace(good, "pages.fm", "pages.fm:443", 1)),
		"alternate-host":     pfReq(t, "GET", strings.Replace(good, "pages.fm", "api.pages.fm", 1)),
		"host-suffix":        pfReq(t, "GET", strings.Replace(good, "pages.fm", "pages.fm.attacker.test", 1)),
		"userinfo":           pfReq(t, "GET", strings.Replace(good, "https://", "https://user:pw@", 1)),
		"fragment":           pfReq(t, "GET", good+"#frag"),
		"health-endpoint":    pfReq(t, "GET", "https://pages.fm/api/public_api/v1/pages/"+pfPage+"/tags?page_access_token="+pfToken),
		"alternate-version":  pfReq(t, "GET", strings.Replace(good, "/v2/", "/v3/", 1)),
		"other-page":         pfReq(t, "GET", strings.Replace(good, pfPage, "page-other", 1)),
		"path-traversal":     pfReq(t, "GET", "https://pages.fm/api/public_api/v1/pages/"+pfPage+"/conversations/c-aaa1/../../tags/messages?page_access_token="+pfToken),
		"encoded-separator":  pfReq(t, "GET", "https://pages.fm/api/public_api/v1/pages/"+pfPage+"/conversations/c-aaa1%2F..%2Fx/messages?page_access_token="+pfToken),
		"unapproved-conv":    pfReq(t, "GET", strings.Replace(goodMsg, "c-aaa1", "c-unlisted", 1)),
		"media-destination":  pfReq(t, "GET", "https://cdn.example.test/img/IMG-CANARY-9911.jpg"),
		"extra-query-key":    pfReq(t, "GET", good+"&debug=1"),
		"duplicate-query":    pfReq(t, "GET", good+"&until=1"),
		"wrong-token":        pfReq(t, "GET", strings.Replace(good, pfToken, "tok-other", 1)),
		"missing-token":      pfReq(t, "GET", strings.Replace(good, "&page_access_token="+pfToken, "", 1)),
		"cursor-unobserved":  pfReq(t, "GET", good+"&last_conversation_id=c-never-seen"),
		"since-mismatch":     pfReq(t, "GET", strings.Replace(good, "since="+fmt.Sprint(pfSince.Unix()), "since=1", 1)),
		"non-numeric-until":  pfReq(t, "GET", strings.Replace(good, "until=1790000000", "until=abc", 1)),
		"messages-bad-count": pfReq(t, "GET", goodMsg+"&current_count=-1"),
		"messages-extra-key": pfReq(t, "GET", goodMsg+"&until=1"),
	}
	for name, req := range denied {
		before := tp.Calls()
		resp, err := tr.RoundTrip(req)
		if err == nil || resp != nil || !errors.Is(err, errProofDenied) {
			t.Errorf("%s: want denial, got resp=%v err=%v", name, resp, err)
		}
		if tp.Calls() != before {
			t.Errorf("%s: denied request reached the underlying transport", name)
		}
	}
	if len(tr.denied) != len(denied) || tr.attempts != 0 {
		t.Fatalf("denials recorded %d want %d, attempts %d want 0", len(tr.denied), len(denied), tr.attempts)
	}
	// Positive controls: the table is not simply denying everything.
	for name, raw := range map[string]string{"conversations": good, "messages": goodMsg} {
		resp, err := tr.RoundTrip(pfReq(t, "GET", raw))
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("%s positive control rejected: %v", name, err)
		}
	}
	if tp.Calls() != 2 {
		t.Fatalf("positive controls should reach transport exactly twice, got %d", tp.Calls())
	}
}

func TestProofPH02RedirectsNeverFollowed(t *testing.T) {
	target := "https://pages.fm/api/public_api/v2/pages/" + pfPage + "/conversations?type=INBOX&order_by=updated_at&until=1790000000&page_access_token=" + pfToken + "&since=" + fmt.Sprint(pfSince.Unix())
	for name, loc := range map[string]string{"same-host": target, "other-host": "https://evil.example.test/x", "relative": "/api/public_api/v2/pages/x"} {
		t.Run(name, func(t *testing.T) {
			tp, inv := pfFixture()
			tp.Routes["conv:"] = []ProofResponse{{Status: 302, Location: loc}}
			r := pfRun(t, tp, inv, nil)
			if tp.Calls() != 1 {
				t.Fatalf("a redirect hop reached the transport: %d calls", tp.Calls())
			}
			if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "redirect_rejected") {
				t.Fatalf("want INCOMPLETE redirect_rejected, got %s %v", r.Disposition, r.Reasons)
			}
			if r.Runs[1].Disposition != proofNotRun {
				t.Fatalf("run 2 must not execute after incomplete run 1")
			}
		})
	}
}

// ---------------------------------------------------------------- PH-03

func TestProofPH03SharedBudget(t *testing.T) {
	t.Run("exactly-N-with-terminals-passes", func(t *testing.T) {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxAttempts = 20 })
		if r.Disposition != ProofPass || tp.Calls() != 20 {
			t.Fatalf("got %s with %d calls", r.Disposition, tp.Calls())
		}
	})
	t.Run("request-N-plus-1-never-reaches-transport", func(t *testing.T) {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxAttempts = 19 })
		if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "budget_exhausted") {
			t.Fatalf("want budget INCOMPLETE, got %s %v", r.Disposition, r.Reasons)
		}
		if tp.Calls() != 19 || r.Budget.AttemptsUsed != 19 {
			t.Fatalf("N+1 leaked: transport=%d receipt=%d", tp.Calls(), r.Budget.AttemptsUsed)
		}
	})
	t.Run("budget-ends-mid-traversal-before-terminal", func(t *testing.T) {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxAttempts = 9 }) // last explicit-empty page of run 1 does not fit
		if r.Disposition != ProofIncomplete || r.Runs[0].Disposition != ProofIncomplete || tp.Calls() != 9 {
			t.Fatalf("truncated traversal earned %s/%s with %d calls", r.Disposition, r.Runs[0].Disposition, tp.Calls())
		}
	})
	t.Run("second-run-consumes-same-counter", func(t *testing.T) {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxAttempts = 10 })
		if r.Runs[0].Disposition != ProofPass || r.Runs[1].Disposition != ProofIncomplete || r.Disposition != ProofIncomplete {
			t.Fatalf("run dispositions %s/%s overall %s", r.Runs[0].Disposition, r.Runs[1].Disposition, r.Disposition)
		}
		if tp.Calls() != 10 || len(r.Runs[1].Requests) != 0 {
			t.Fatalf("second run reached transport: calls=%d requests=%d", tp.Calls(), len(r.Runs[1].Requests))
		}
	})
	t.Run("429-retries-count", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["conv:"] = []ProofResponse{{Status: 429, Body: `{}`}, tp.Routes["conv:"][0]}
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxAttempts = 20 })
		if r.Disposition != ProofIncomplete || tp.Calls() != 20 {
			t.Fatalf("retry must consume budget: %s calls=%d", r.Disposition, tp.Calls())
		}
		tp, inv = pfFixture()
		tp.Routes["conv:"] = []ProofResponse{{Status: 429, Body: `{}`}, tp.Routes["conv:"][0]}
		r = pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxAttempts = 21 })
		if r.Disposition != ProofPass || tp.Calls() != 21 || r.Runs[0].Requests[0].Outcome != "http_429" {
			t.Fatalf("21 attempts including the retry should pass: %s calls=%d", r.Disposition, tp.Calls())
		}
	})
	t.Run("option-ceilings", func(t *testing.T) {
		for name, mut := range map[string]func(*PancakeProofOptions){
			"zero-attempts": func(o *PancakeProofOptions) { o.MaxAttempts = 0 },
			"over-50":       func(o *PancakeProofOptions) { o.MaxAttempts = 51 },
			"zero-duration": func(o *PancakeProofOptions) { o.MaxDuration = 0 },
			"over-10min":    func(o *PancakeProofOptions) { o.MaxDuration = 10*time.Minute + time.Nanosecond },
		} {
			tp, inv := pfFixture()
			o := pfOpts(tp)
			mut(&o)
			if _, err := RunPancakeProof(context.Background(), o, inv); !errors.Is(err, ErrPancakeProofInput) || tp.Calls() != 0 {
				t.Errorf("%s: want input rejection with zero calls, got %v (%d)", name, err, tp.Calls())
			}
		}
	})
}

func TestProofPH03DeadlineAndCancellation(t *testing.T) {
	t.Run("in-flight-deadline", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["conv:"] = []ProofResponse{{Block: true}}
		begin := time.Now()
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxDuration = 60 * time.Millisecond })
		if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "deadline_or_cancelled") {
			t.Fatalf("want deadline INCOMPLETE, got %s %v", r.Disposition, r.Reasons)
		}
		if time.Since(begin) > time.Second {
			t.Fatalf("deadline not enforced promptly: %v", time.Since(begin))
		}
	})
	t.Run("elapsed-before-request", func(t *testing.T) {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) {
			base := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
			calls := 0
			o.MaxDuration = time.Second
			o.Now = func() time.Time { calls++; return base.Add(time.Duration(calls) * 400 * time.Millisecond) }
		})
		if r.Disposition != ProofIncomplete || tp.Calls() >= 10 {
			t.Fatalf("elapsed time must stop requests: %s calls=%d", r.Disposition, tp.Calls())
		}
		if !strings.Contains(pfReasons(r), "deadline_or_cancelled") {
			t.Fatalf("reason %v", r.Reasons)
		}
	})
	t.Run("cancelled-context", func(t *testing.T) {
		tp, inv := pfFixture()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r, err := RunPancakeProof(ctx, pfOpts(tp), inv)
		if err != nil || r.Disposition != ProofIncomplete || tp.Calls() != 0 {
			t.Fatalf("cancelled run: %v %v calls=%d", err, r, tp.Calls())
		}
	})
	t.Run("cancel-mid-run", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["msg:c-aaa1:0"] = []ProofResponse{{Block: true}}
		ctx, cancel := context.WithCancel(context.Background())
		go func() { time.Sleep(40 * time.Millisecond); cancel() }()
		r, err := RunPancakeProof(ctx, pfOpts(tp), inv)
		if err != nil || r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "deadline_or_cancelled") {
			t.Fatalf("cancel mid-run: %v %v", err, r.Reasons)
		}
	})
	t.Run("oversize-response-never-passes", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["conv:"] = []ProofResponse{pfOK(`{"conversations":[],"pad":"` + strings.Repeat("x", ProofMaxResponseBytes) + `"}`)}
		r := pfRun(t, tp, inv, nil)
		if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "response_oversize") {
			t.Fatalf("oversize: %s %v", r.Disposition, r.Reasons)
		}
	})
}

// ---------------------------------------------------------------- PH-05

func TestProofPH05ReceiptSchemaAndDigest(t *testing.T) {
	tp, inv := pfFixture()
	r := pfRun(t, tp, inv, nil)
	if r.SchemaVersion != ProofSchemaVersion || r.EvidenceType != "SYNTHETIC_OFFLINE" || r.Live || r.Governance || r.SourceSHA != pfSHA {
		t.Fatalf("receipt header wrong: %+v", r)
	}
	if r.Options.MaxAttempts != 20 || r.Options.ResponseCeiling != 8<<20 || r.Options.Runs != 2 || r.Inventory.ConversationCount != 3 || r.Inventory.MessageCount != 5 {
		t.Fatalf("options/inventory summary wrong: %+v %+v", r.Options, r.Inventory)
	}
	if !strings.HasSuffix(r.StartedAt, "Z") || !strings.HasSuffix(r.Runs[1].FinishedAt, "Z") {
		t.Fatalf("times must be UTC: %s %s", r.StartedAt, r.Runs[1].FinishedAt)
	}
	// identical inventory + key -> identical identifiers across runs and across invocations
	for i := range r.Runs[0].Requests {
		a, b := r.Runs[0].Requests[i], r.Runs[1].Requests[i]
		if a.ConversationRef != b.ConversationRef || a.CursorRef != b.CursorRef {
			t.Fatalf("identifiers drift between runs at %d", i)
		}
	}
	tp2, inv2 := pfFixture()
	r2 := pfRun(t, tp2, inv2, nil)
	if r.Inventory.Digest != r2.Inventory.Digest || r.Inventory.PageRef != r2.Inventory.PageRef || r.Inventory.PageRef != pfRef("page", pfPage) {
		t.Fatalf("digest/page ref unstable")
	}
	// a different expectation changes the digest
	inv2.Conversations[0].Messages[0].SentAt = inv2.Conversations[0].Messages[0].SentAt.Add(time.Second)
	tp3, _ := pfFixture()
	if r3 := pfRun(t, tp3, inv2, nil); r3.Inventory.Digest == r.Inventory.Digest {
		t.Fatalf("changing the inventory did not change its digest")
	}
	// a different pseudonym key changes identifiers
	tp4, inv4 := pfFixture()
	r4 := pfRun(t, tp4, inv4, func(o *PancakeProofOptions) { o.PseudonymKey = []byte("another-key-value") })
	if r4.Runs[0].Requests[3].ConversationRef == r.Runs[0].Requests[3].ConversationRef {
		t.Fatalf("pseudonyms must depend on the key")
	}
}

func TestProofPH05DeterministicSerialization(t *testing.T) {
	var out [2][]byte
	for i := range out {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, nil)
		for j := range r.Runs {
			r.Runs[j].ConversationUntil = 0 // real wall-clock chosen by the unchanged adapter
		}
		b, err := MarshalProofReceipt(r, ProofSensitiveValues(pfOpts(tp), inv))
		if err != nil {
			t.Fatal(err)
		}
		out[i] = b
	}
	if string(out[0]) != string(out[1]) {
		t.Fatalf("receipt serialization is not deterministic")
	}
}

func TestProofPH05InvalidSetupRejectedBeforeWork(t *testing.T) {
	mutations := map[string]func(*PancakeProofOptions, *ProofInventory){
		"missing-transport":    func(o *PancakeProofOptions, _ *ProofInventory) { o.Transport = nil },
		"short-sha":            func(o *PancakeProofOptions, _ *ProofInventory) { o.SourceSHA = "d869624" },
		"uppercase-sha":        func(o *PancakeProofOptions, _ *ProofInventory) { o.SourceSHA = strings.ToUpper(pfSHA) },
		"missing-sha":          func(o *PancakeProofOptions, _ *ProofInventory) { o.SourceSHA = "" },
		"missing-key":          func(o *PancakeProofOptions, _ *ProofInventory) { o.PseudonymKey = nil },
		"missing-token":        func(o *PancakeProofOptions, _ *ProofInventory) { o.Token = "" },
		"missing-since":        func(o *PancakeProofOptions, _ *ProofInventory) { o.Since = time.Time{} },
		"missing-provenance":   func(_ *PancakeProofOptions, i *ProofInventory) { i.Provenance = "" },
		"free-text-provenance": func(_ *PancakeProofOptions, i *ProofInventory) { i.Provenance = "has spaces and /paths" },
		"nil-inventory":        func(_ *PancakeProofOptions, i *ProofInventory) { *i = ProofInventory{} },
		"page-mismatch":        func(_ *PancakeProofOptions, i *ProofInventory) { i.PageID = "other" },
		"duplicate-conv":       func(_ *PancakeProofOptions, i *ProofInventory) { i.Conversations[1].ID = i.Conversations[0].ID },
		"zero-message-time":    func(_ *PancakeProofOptions, i *ProofInventory) { i.Conversations[0].Messages[0].SentAt = time.Time{} },
		"negative-backoff":     func(o *PancakeProofOptions, _ *ProofInventory) { o.RetryBackoff = -1 },
	}
	for name, mut := range mutations {
		tp, inv := pfFixture()
		o := pfOpts(tp)
		mut(&o, inv)
		r, err := RunPancakeProof(context.Background(), o, inv)
		if r != nil || !errors.Is(err, ErrPancakeProofInput) {
			t.Errorf("%s: want ErrPancakeProofInput, got %v %v", name, r, err)
		}
		if tp.Calls() != 0 {
			t.Errorf("%s: transport used before validation", name)
		}
		if err != nil && (strings.Contains(err.Error(), pfToken) || strings.Contains(err.Error(), string(pfKey))) {
			t.Errorf("%s: error leaks a secret: %v", name, err)
		}
	}
}

// ---------------------------------------------------------------- PH-06

func TestProofPH06PseudonymIsDomainSeparatedHMAC(t *testing.T) {
	if got := proofPseudonym(pfKey, "conversation", "c-aaa1"); got != pfRef("conversation", "c-aaa1") || strings.Contains(got, "c-aaa1") {
		t.Fatalf("pseudonym %q is not the expected HMAC form", got)
	}
	if proofPseudonym(pfKey, "conversation", "x") == proofPseudonym(pfKey, "message", "x") {
		t.Fatalf("domains must separate identical raw values")
	}
	if proofPseudonym(pfKey, "conversation", "x") == proofPseudonym([]byte("other"), "conversation", "x") {
		t.Fatalf("key must change the pseudonym")
	}
}

func pfAssertClean(t *testing.T, label, text string) {
	t.Helper()
	for _, canary := range []string{pfToken, string(pfKey), "SECRET-TEXT-1", "IMG-CANARY-9911", "c-aaa1", "c-bbb2", "c-ddd5", "c-old3", "m-old", "m-1", pfPage, "CANARY-ERR-5521", "CANARY-BODY-7788", "CANARY-PANIC-3344", "cdn.example.test", "last_conversation_id", "page_access_token"} {
		if strings.Contains(text, canary) {
			t.Errorf("%s leaks %q", label, canary)
		}
	}
}

func TestProofPH06CanariesNeverAppear(t *testing.T) {
	scenarios := map[string]func(tp *ProofTranscript){
		"clean-pass": func(*ProofTranscript) {},
		"malformed-body-with-canary": func(tp *ProofTranscript) {
			tp.Routes["conv:c-old3"] = []ProofResponse{pfOK(`{"conversations": [ CANARY-BODY-7788 `)}
		},
		"success-false-body": func(tp *ProofTranscript) {
			tp.Routes["conv:"] = []ProofResponse{pfOK(`{"success":false,"error_code":9,"message":"CANARY-BODY-7788 c-aaa1"}`)}
		},
		"transport-error-with-canary": func(tp *ProofTranscript) {
			tp.Routes["msg:c-bbb2:0"] = []ProofResponse{{Err: errors.New("dial https://pages.fm/?page_access_token=" + pfToken + " CANARY-ERR-5521")}}
		},
		"http-500-body": func(tp *ProofTranscript) {
			tp.Routes["msg:c-aaa1:3"] = []ProofResponse{{Status: 500, Body: "CANARY-BODY-7788 SECRET-TEXT-1"}}
		},
		"redirect-to-canary-host": func(tp *ProofTranscript) {
			tp.Routes["conv:"] = []ProofResponse{{Status: 301, Location: "https://evil.test/?t=" + pfToken}}
		},
	}
	for name, edit := range scenarios {
		t.Run(name, func(t *testing.T) {
			tp, inv := pfFixture()
			edit(tp)
			o := pfOpts(tp)
			r, err := RunPancakeProof(context.Background(), o, inv)
			if err != nil {
				t.Fatalf("unexpected rejection %v", err)
			}
			b, merr := MarshalProofReceipt(r, ProofSensitiveValues(o, inv))
			if merr != nil {
				t.Fatalf("receipt failed its own sanitation scan: %v", merr)
			}
			pfAssertClean(t, "receipt", string(b))
			if name != "clean-pass" && r.Disposition == ProofPass {
				t.Fatalf("a failing transcript must not PASS")
			}
		})
	}
	t.Run("transport-panic", func(t *testing.T) {
		tp, inv := pfFixture()
		o := pfOpts(panicTransport{})
		r, err := RunPancakeProof(context.Background(), o, inv)
		if err != nil || r.Disposition != ProofIncomplete {
			t.Fatalf("panic must be recovered as INCOMPLETE: %v %v", err, r)
		}
		b, _ := MarshalProofReceipt(r, nil)
		pfAssertClean(t, "panic receipt", string(b))
		_ = tp
	})
	t.Run("input-error-text", func(t *testing.T) {
		tp, inv := pfFixture()
		o := pfOpts(tp)
		o.SourceSHA = "bad-" + pfToken
		_, err := RunPancakeProof(context.Background(), o, inv)
		if err == nil {
			t.Fatal("expected rejection")
		}
		pfAssertClean(t, "input error", err.Error())
	})
	t.Run("marshal-fails-closed", func(t *testing.T) {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, nil)
		r.Reasons = append(r.Reasons, "leaked:"+pfToken)
		if _, err := MarshalProofReceipt(r, ProofSensitiveValues(pfOpts(tp), inv)); !errors.Is(err, ErrPancakeProofUnsafe) {
			t.Fatalf("a receipt containing a raw secret must be refused, got %v", err)
		}
		r.Reasons = r.Reasons[:len(r.Reasons)-1]
		r.Live = true
		if _, err := MarshalProofReceipt(r, nil); !errors.Is(err, ErrPancakeProofUnsafe) {
			t.Fatalf("a receipt claiming live must be refused")
		}
		r.Live, r.EvidenceType = false, "LIVE"
		if _, err := MarshalProofReceipt(r, nil); !errors.Is(err, ErrPancakeProofUnsafe) {
			t.Fatalf("a non-synthetic evidence type must be refused")
		}
	})
}

type panicTransport struct{}

func (panicTransport) RoundTrip(*http.Request) (*http.Response, error) { panic("CANARY-PANIC-3344") }

// ---------------------------------------------------------------- PH-07 (library part)

func TestProofPH07NoDefaultTransportAndNoWork(t *testing.T) {
	_, inv := pfFixture()
	o := pfOpts(nil)
	o.Transport = nil
	if _, err := RunPancakeProof(context.Background(), o, inv); !errors.Is(err, ErrPancakeProofInput) || !strings.Contains(err.Error(), "transport_required") {
		t.Fatalf("missing transport must be rejected up front: %v", err)
	}
	// The adapter built by the harness uses only the guarded transport and refuses redirects.
	tp, _ := pfFixture()
	oo := pfOpts(tp)
	ad := newProofAdapter(&oo, tp)
	if ad.client.Transport != tp || ad.client.CheckRedirect == nil {
		t.Fatalf("proof adapter must use the injected transport and a redirect refusal")
	}
	if err := ad.client.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirects must not be followed")
	}
}

// ---------------------------------------------------------------- PH-08

func TestProofPH08FailureSemantics(t *testing.T) {
	cases := map[string]struct {
		edit   func(tp *ProofTranscript)
		reason string
	}{
		"late-malformed-page": {func(tp *ProofTranscript) { tp.Routes["conv:c-old3"] = []ProofResponse{pfOK(`{"conversations":{}}`)} }, "conversation_coverage_incomplete"},
		"non-2xx":             {func(tp *ProofTranscript) { tp.Routes["msg:c-bbb2:0"] = []ProofResponse{{Status: 503, Body: "x"}} }, "adapter_error"},
		"repeated-cursor": {func(tp *ProofTranscript) {
			tp.Routes["conv:c-old3"] = []ProofResponse{pfOK(pfList("conversations", pfConvRow("c-old3", "INBOX", "2026-09-24T10:00:00.000000")))}
		}, "conversation_coverage_incomplete"},
		"message-no-progress": {func(tp *ProofTranscript) {
			tp.Routes["msg:c-aaa1:3"] = []ProofResponse{pfOK(pfList("messages", pfMsgRow("m-1", "u-c-aaa1", "t", "2026-09-24T09:00:00.000000", "")))}
		}, "message_coverage_incomplete"},
		"conversation-body-has-no-array": {func(tp *ProofTranscript) { tp.Routes["conv:"] = []ProofResponse{pfOK(`{"success":true}`)} }, "conversation_coverage_incomplete"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			tp, inv := pfFixture()
			c.edit(tp)
			r := pfRun(t, tp, inv, nil)
			if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), c.reason) {
				t.Fatalf("want INCOMPLETE %s, got %s %v", c.reason, r.Disposition, r.Reasons)
			}
			if r.Runs[1].Disposition != proofNotRun {
				t.Fatalf("run 2 must be NOT_RUN after an incomplete run 1, got %s", r.Runs[1].Disposition)
			}
		})
	}
	t.Run("second-run-failure-does-not-retain-first-run-pass", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["conv:"] = []ProofResponse{tp.Routes["conv:"][0], {Status: 500, Body: "x"}}
		r := pfRun(t, tp, inv, nil)
		if r.Runs[0].Disposition != ProofPass || r.Runs[1].Disposition != ProofIncomplete || r.Disposition != ProofIncomplete {
			t.Fatalf("got %s / %s overall %s", r.Runs[0].Disposition, r.Runs[1].Disposition, r.Disposition)
		}
		if !strings.Contains(pfReasons(r), "run2:") || strings.Contains(pfReasons(r), "run1:") {
			t.Fatalf("original run attribution lost: %v", r.Reasons)
		}
	})
	t.Run("second-run-mismatch-is-fail", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["msg:c-bbb2:0"] = []ProofResponse{tp.Routes["msg:c-bbb2:0"][0], pfOK(pfList("messages", pfMsgRow("m-b1", "u-c-bbb2", "boundary", "2026-09-20T00:00:01.000000", "")))}
		r := pfRun(t, tp, inv, nil)
		if r.Runs[0].Disposition != ProofPass || r.Runs[1].Disposition != ProofFail || r.Disposition != ProofFail {
			t.Fatalf("got %s / %s overall %s", r.Runs[0].Disposition, r.Runs[1].Disposition, r.Disposition)
		}
	})
	t.Run("until-is-stable-inside-a-run", func(t *testing.T) {
		// The unchanged adapter pins until per traversal; the receipt records that observation.
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, nil)
		if !r.Runs[0].UntilStable || !r.Runs[1].UntilStable {
			t.Fatalf("until must be stable inside a run")
		}
	})
	t.Run("fixture-stays-finite-without-guards", func(t *testing.T) {
		// The transcript itself caps total calls, so a mutated guard cannot spin forever.
		tp, inv := pfFixture()
		tp.Ceiling = 40
		r := pfRun(t, tp, inv, nil)
		if r.Disposition != ProofPass || tp.Calls() > 40 {
			t.Fatalf("fixture must complete well inside its ceiling: %s %d", r.Disposition, tp.Calls())
		}
	})
}

func TestProofTranscriptIsFinite(t *testing.T) {
	tp := &ProofTranscript{Routes: map[string][]ProofResponse{"conv:": {pfOK("{}")}}, Repeat: true, Ceiling: 3}
	for i := 0; i < 3; i++ {
		if _, err := tp.RoundTrip(pfReq(t, "GET", "https://pages.fm/api/public_api/v2/pages/p/conversations")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tp.RoundTrip(pfReq(t, "GET", "https://pages.fm/api/public_api/v2/pages/p/conversations")); err == nil {
		t.Fatalf("transcript must stop at its hard ceiling")
	}
	tp2 := &ProofTranscript{Routes: map[string][]ProofResponse{}}
	if _, err := tp2.RoundTrip(pfReq(t, "GET", "https://pages.fm/api/public_api/v2/pages/p/conversations")); err == nil {
		t.Fatalf("unscripted requests must fail")
	}
}

// ---------------------------------------------------------------- R034-R1 regression detectors

// pfBodyErrTransport wraps a transport so selected responses deliver their body and then fail the read.
type pfBodyErrTransport struct {
	inner   http.RoundTripper
	match   func(*http.Request) bool
	partial bool // deliver only half of the body before the error
}

type pfErrBody struct {
	data []byte
	off  int
	err  error
}

func (b *pfErrBody) Read(dst []byte) (int, error) {
	if b.off >= len(b.data) {
		return 0, b.err
	}
	n := copy(dst, b.data[b.off:])
	b.off += n
	if b.off >= len(b.data) {
		return n, b.err // the error arrives together with the final chunk
	}
	return n, nil
}
func (b *pfErrBody) Close() error { return nil }

func (p pfBodyErrTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := p.inner.RoundTrip(req)
	if err != nil || !p.match(req) {
		return res, err
	}
	data, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if p.partial {
		data = data[:len(data)/2]
	}
	res.Body = &pfErrBody{data: data, err: errors.New("read failed CANARY-ERR-5521 " + pfToken)}
	return res, nil
}

func TestProofR1BodyReadErrorNeverPasses(t *testing.T) {
	cases := map[string]pfBodyErrTransport{
		"complete-json-then-error-every-body": {match: func(*http.Request) bool { return true }},
		"partial-json-then-error-every-body":  {match: func(*http.Request) bool { return true }, partial: true},
		"late-terminal-read-fails": {match: func(r *http.Request) bool {
			return strings.HasSuffix(r.URL.Path, "/messages") && r.URL.Query().Get("current_count") == "1"
		}},
		"conversation-terminal-read-fails": {match: func(r *http.Request) bool {
			return !strings.HasSuffix(r.URL.Path, "/messages") && r.URL.Query().Get("last_conversation_id") == "c-ddd5"
		}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			tp, inv := pfFixture()
			c.inner = tp
			o := pfOpts(c)
			r, err := RunPancakeProof(context.Background(), o, inv)
			if err != nil {
				t.Fatal(err)
			}
			if r.Disposition != ProofIncomplete || !strings.Contains(pfReasons(r), "run1:response_read_error") {
				t.Fatalf("read error must be INCOMPLETE response_read_error, got %s %v", r.Disposition, r.Reasons)
			}
			if r.Runs[0].Disposition != ProofIncomplete || r.Runs[1].Disposition != proofNotRun {
				t.Fatalf("run attribution wrong: %s/%s", r.Runs[0].Disposition, r.Runs[1].Disposition)
			}
			last := r.Runs[0].Requests[len(r.Runs[0].Requests)-1]
			if last.Outcome != "body_read_error" || last.EmptyTerminal || last.Rows != 0 {
				t.Fatalf("failed read must not yield rows or a terminal: %+v", last)
			}
			if tp.Calls() != r.Budget.AttemptsUsed {
				t.Fatalf("calls %d vs receipt %d", tp.Calls(), r.Budget.AttemptsUsed)
			}
			b, merr := MarshalProofReceipt(r, ProofSensitiveValues(o, inv))
			if merr != nil {
				t.Fatalf("receipt must serialize: %v", merr)
			}
			pfAssertClean(t, "read-error receipt", string(b))
		})
	}
}

func TestProofR1ReceiptValidationAndSemanticScan(t *testing.T) {
	// semantic scan: JSON escaping must not hide a value, in string values or in keys
	for _, canary := range []string{"CANARY-<RAW>&VALUE", "CANARY-QUOTE-\"-VALUE", "CANARY-SLASH-\\-VALUE", "CANARY-LINE-\n-VALUE", "CANARY-TAB-\t-VALUE"} {
		asValue, _ := json.Marshal(map[string]interface{}{"k": []string{"x", "pre " + canary + " post"}})
		asKey, _ := json.Marshal(map[string]string{canary: "v"})
		for label, doc := range map[string][]byte{"value": asValue, "key": asKey} {
			if strings.Contains(string(doc), canary) {
				t.Fatalf("setup: %q must be escaped in the encoded form", canary)
			}
			if !proofContainsSensitive(doc, []string{canary}) {
				t.Errorf("%s: escaped canary %q evaded the scan", label, canary)
			}
			if proofContainsSensitive(doc, []string{"CANARY-UNRELATED-9999"}) {
				t.Errorf("%s: unrelated value produced a false hit", label)
			}
		}
	}
	if !proofContainsSensitive([]byte("{not json"), []string{"abcdefgh"}) {
		t.Errorf("undecodable documents must fail closed")
	}

	// the scan layer on a structurally valid receipt: a sensitive value that is an allowed label
	tp, inv := pfFixture()
	good := pfRun(t, tp, inv, nil)
	if _, err := MarshalProofReceipt(good, []string{good.Inventory.Provenance}); !errors.Is(err, ErrPancakeProofUnsafe) {
		t.Errorf("scan layer must reject a decoded sensitive value in a structurally valid receipt: %v", err)
	}
	if _, err := MarshalProofReceipt(good, nil); err != nil {
		t.Fatalf("control receipt must serialize: %v", err)
	}

	// unsupported values are refused even without any sensitive list
	edits := map[string]func(*ProofReceipt){
		"unknown-schema":       func(r *ProofReceipt) { r.SchemaVersion = "unknown-schema" },
		"unknown-disposition":  func(r *ProofReceipt) { r.Disposition = "UNKNOWN" },
		"not-run-overall":      func(r *ProofReceipt) { r.Disposition = proofNotRun },
		"invalid-source-sha":   func(r *ProofReceipt) { r.SourceSHA = "not-a-source-sha" },
		"uppercase-source-sha": func(r *ProofReceipt) { r.SourceSHA = strings.ToUpper(r.SourceSHA) },
		"bad-start-time":       func(r *ProofReceipt) { r.StartedAt = "yesterday" },
		"free-text-reason":     func(r *ProofReceipt) { r.Reasons = append(r.Reasons, "free text \"with quotes\"") },
		"canary-reason":        func(r *ProofReceipt) { r.Reasons = append(r.Reasons, "leaked:"+pfToken) },
		"free-text-provenance": func(r *ProofReceipt) { r.Inventory.Provenance = "has spaces" },
		"raw-page-ref":         func(r *ProofReceipt) { r.Inventory.PageRef = pfPage },
		"bad-digest":           func(r *ProofReceipt) { r.Inventory.Digest = "abc" },
		"raw-conversation-ref": func(r *ProofReceipt) { r.Runs[0].Requests[3].ConversationRef = "c-aaa1" },
		"raw-cursor-ref":       func(r *ProofReceipt) { r.Runs[0].Requests[1].CursorRef = "c-old3" },
		"free-text-outcome":    func(r *ProofReceipt) { r.Runs[0].Requests[0].Outcome = "dial tcp: " + pfToken },
		"unknown-template":     func(r *ProofReceipt) { r.Runs[0].Requests[0].Template = "/api/public_api/v2/pages/" + pfPage },
		"unknown-kind":         func(r *ProofReceipt) { r.Runs[0].Requests[0].Kind = "media" },
		"run-index-mismatch":   func(r *ProofReceipt) { r.Runs[1].Index = 7 },
		"missing-run":          func(r *ProofReceipt) { r.Runs = r.Runs[:1] },
		"unknown-run-state":    func(r *ProofReceipt) { r.Runs[0].Disposition = "MAYBE" },
		"raw-missing-ref":      func(r *ProofReceipt) { r.Runs[0].Conversations.Missing = []string{"c-zzz9"} },
		"unknown-mismatch-name": func(r *ProofReceipt) {
			r.Runs[0].Messages.Mismapped = []ProofMismatch{{Ref: pfRef("message", "x"), Fields: []string{"body_text"}}}
		},
		"denial-free-text":   func(r *ProofReceipt) { r.Denials = []ProofDenial{{Run: 1, Reason: "path " + pfToken}} },
		"live-flag":          func(r *ProofReceipt) { r.Live = true },
		"governance-flag":    func(r *ProofReceipt) { r.Governance = true },
		"non-synthetic-type": func(r *ProofReceipt) { r.EvidenceType = "LIVE" },
	}
	for name, edit := range edits {
		tp, inv := pfFixture()
		r := pfRun(t, tp, inv, nil)
		edit(r)
		b, err := MarshalProofReceipt(r, nil)
		if !errors.Is(err, ErrPancakeProofUnsafe) || b != nil {
			t.Errorf("%s: unsupported receipt accepted (err=%v)", name, err)
		}
		if err != nil && strings.Contains(err.Error(), pfToken) {
			t.Errorf("%s: rejection echoes a value", name)
		}
	}
	if _, err := MarshalProofReceipt(nil, nil); !errors.Is(err, ErrPancakeProofUnsafe) {
		t.Errorf("nil receipt must be refused")
	}
}

type pfCountingTransport struct{ n int }

func (c *pfCountingTransport) RoundTrip(q *http.Request) (*http.Response, error) {
	c.n++
	return &http.Response{StatusCode: 200, Status: "200", Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"conversations":[],"messages":[]}`)), Request: q}, nil
}

func TestProofR1ApprovedUnsafeIdentifiersDenied(t *testing.T) {
	for _, id := range []string{"c-aaa1/../x", "..", ".", "a\\..\\x", "a%2F..%2Fx", "a%252e%252e", "a?b", "a#b", "a b", "a\x00b", "a;b"} {
		t.Run(fmt.Sprintf("conversation-%q", id), func(t *testing.T) {
			rt := &pfCountingTransport{}
			o := pfOpts(rt)
			tr := &proofTransport{inner: rt, opts: &o, approved: map[string]bool{id: true}, now: o.Now, start: o.Now()}
			tr.beginRun(1)
			tr.setConv(id)
			raw := "https://pages.fm/api/public_api/v1/pages/" + url.PathEscape(o.PageID) + "/conversations/" + url.PathEscape(id) + "/messages?page_access_token=" + url.QueryEscape(o.Token)
			u, err := url.Parse(raw)
			if err != nil {
				t.Skip("not a parseable URL")
			}
			if _, err := tr.RoundTrip(&http.Request{Method: "GET", URL: u, Header: http.Header{}}); !errors.Is(err, errProofDenied) || rt.n != 0 {
				t.Fatalf("approved unsafe id reached the transport: err=%v calls=%d", err, rt.n)
			}
		})
	}
	for _, page := range []string{"..", "a/b", "a\\b", "a%2e", "a?b"} {
		rt := &pfCountingTransport{}
		o := pfOpts(rt)
		o.PageID = page
		tr := &proofTransport{inner: rt, opts: &o, approved: map[string]bool{}, now: o.Now, start: o.Now()}
		tr.beginRun(1)
		raw := "https://pages.fm/api/public_api/v2/pages/" + url.PathEscape(page) + "/conversations?type=INBOX&order_by=updated_at&since=" + fmt.Sprint(o.Since.Unix()) + "&until=1790000000&page_access_token=" + url.QueryEscape(o.Token)
		u, _ := url.Parse(raw)
		if _, err := tr.RoundTrip(&http.Request{Method: "GET", URL: u, Header: http.Header{}}); !errors.Is(err, errProofDenied) || rt.n != 0 {
			t.Errorf("unsafe page %q reached the transport: err=%v calls=%d", page, err, rt.n)
		}
	}
	// setup validation rejects the same identifiers before any request
	for name, mut := range map[string]func(*PancakeProofOptions, *ProofInventory){
		"page-dotdot":  func(o *PancakeProofOptions, i *ProofInventory) { o.PageID, i.PageID = "..", ".." },
		"conv-slash":   func(_ *PancakeProofOptions, i *ProofInventory) { i.Conversations[0].ID = "c-aaa1/../x" },
		"conv-dotdot":  func(_ *PancakeProofOptions, i *ProofInventory) { i.Conversations[0].ID = ".." },
		"conv-percent": func(_ *PancakeProofOptions, i *ProofInventory) { i.Conversations[0].ID = "a%2e%2e" },
	} {
		tp, inv := pfFixture()
		o := pfOpts(tp)
		mut(&o, inv)
		if _, err := RunPancakeProof(context.Background(), o, inv); !errors.Is(err, ErrPancakeProofInput) || !strings.Contains(err.Error(), "unsafe_identifier") || tp.Calls() != 0 {
			t.Errorf("%s: want unsafe_identifier rejection with zero calls, got %v (%d)", name, err, tp.Calls())
		}
	}
	// positive control: ordinary approved identifiers are admitted
	tp, inv := pfFixture()
	if r := pfRun(t, tp, inv, nil); r.Disposition != ProofPass {
		t.Fatalf("ordinary identifiers must still pass: %s", r.Disposition)
	}
}

func TestProofR1ObserverFollowsEachContractOrder(t *testing.T) {
	t.Run("conversation-old-first-then-eligible", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["conv:"] = []ProofResponse{pfOK(pfList("conversations",
			pfConvRow("c-aaa1", "INBOX", "2026-09-24T10:00:00.000000"),
			pfConvRow("c-bbb2", "INBOX", "2026-09-01T00:00:00.000000"), // old first occurrence
			pfConvRow("c-old3", "INBOX", "2026-09-01T00:00:00.000000")))}
		tp.Routes["conv:c-old3"] = []ProofResponse{pfOK(pfList("conversations",
			pfConvRow("c-aaa1", "INBOX", "2026-09-24T10:00:00.000000"),
			pfConvRow("c-cmt4", "COMMENT", "2026-09-24T10:00:00.000000"),
			pfConvRow("c-bbb2", "INBOX", "2026-09-20T00:00:00.000000"), // later, boundary-equal, eligible
			pfConvRow("c-ddd5", "INBOX", "2026-09-25T01:00:00+07:00")))}
		tp.Routes["conv:c-ddd5"] = []ProofResponse{pfOK(pfList("conversations"))}
		r := pfRun(t, tp, inv, nil)
		if r.Disposition != ProofPass {
			t.Fatalf("unchanged adapter and inventory must PASS: %s %v", r.Disposition, r.Reasons)
		}
		for _, run := range r.Runs {
			c := run.ConversationRows
			if c.PhysicalRows != 7 || c.OldRows != 2 || c.DuplicateRow != 1 || c.NonInboxRows != 1 || c.RawEligible != 3 || c.Mapped != 3 {
				t.Fatalf("conversation accounting follows the adapter order: %+v", c)
			}
		}
	})
	t.Run("message-dedupe-before-since", func(t *testing.T) {
		tp, inv := pfFixture()
		tp.Routes["msg:c-bbb2:0"] = []ProofResponse{pfOK(pfList("messages",
			pfMsgRow("m-b1", "u-c-bbb2", "boundary", "2026-09-20T00:00:00.000000", ""),
			pfMsgRow("m-x", "u-c-bbb2", "old", "2026-09-10T00:00:00.000000", "")))}
		tp.Routes["msg:c-bbb2:2"] = []ProofResponse{pfOK(pfList("messages",
			pfMsgRow("m-x", "u-c-bbb2", "later same id", "2026-09-25T00:00:00.000000", ""), // dropped as a duplicate by the adapter
			pfMsgRow("m-y", "u-c-bbb2", "new", "2026-09-25T00:00:00.000000", "")))}
		tp.Routes["msg:c-bbb2:4"] = []ProofResponse{pfOK(pfList("messages"))}
		inv.Conversations[1].Messages = append(inv.Conversations[1].Messages,
			ProofMessage{ID: "m-y", SentAt: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), SenderType: "customer", ContentType: "text"})
		r := pfRun(t, tp, inv, func(o *PancakeProofOptions) { o.MaxAttempts = 22 }) // one extra message page per run
		if r.Disposition != ProofPass {
			t.Fatalf("message contract must PASS: %s %v", r.Disposition, r.Reasons)
		}
		for _, run := range r.Runs {
			m := run.MessageRows
			// c-aaa1: 5 physical, dup 1, old 1; c-bbb2: 4 physical, dup 1, old 1; c-ddd5: 1 physical
			if m.PhysicalRows != 10 || m.DuplicateRow != 2 || m.OldRows != 2 || m.RawEligible != 6 || m.Mapped != 6 {
				t.Fatalf("message accounting follows the adapter order: %+v", m)
			}
		}
	})
}

func TestProofR1UntilStabilityEvaluatedBeforeDisposition(t *testing.T) {
	newRun := func(perturb bool) ProofRunReport {
		tp, inv := pfFixture()
		opts := pfOpts(tp)
		tr := &proofTransport{inner: tp, opts: &opts, approved: map[string]bool{}, now: opts.Now, start: opts.Now()}
		for _, c := range inv.Conversations {
			tr.approved[c.ID] = true
		}
		if perturb {
			// Synthetic perturbation of the harness observation state (not an adapter change):
			// an unlike until value is recorded right after the run begins.
			first := true
			opts.Now = func() time.Time {
				if first {
					first = false
					tr.untils = append(tr.untils, "1")
				}
				return time.Now()
			}
			tr.now, tr.start = time.Now, time.Now()
		}
		return runProofOnce(context.Background(), 1, newProofAdapter(&opts, tr), tr, &opts, inv, 5)
	}
	if r := newRun(false); !r.UntilStable || r.Disposition != ProofPass {
		t.Fatalf("real unchanged adapter must pin until: stable=%v disp=%s", r.UntilStable, r.Disposition)
	}
	r := newRun(true)
	if r.UntilStable {
		t.Fatal("setup: perturbation must make the observation unstable")
	}
	if r.Disposition != ProofFail || !strings.Contains(strings.Join(r.Reasons, ","), "until_unstable") {
		t.Fatalf("until_stable=false must be FAIL until_unstable, got %s %v", r.Disposition, r.Reasons)
	}
}
