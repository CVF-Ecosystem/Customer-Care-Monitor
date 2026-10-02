package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// CCMAI-RUNTIME-032 (F02-F): Zalo message traversal walks absolute row offsets to an explicit empty
// page, validates every physical row before deduplication, maps rows with the inherited rules,
// stays inside a finite page budget and never follows a redirect. A synthetic RoundTripper stands
// in for openapi.zalo.me / oauth.zaloapp.com and records every request. The probes assert the
// application contract only; they say nothing about live Zalo ordering, retention or snapshot
// stability, and they are not AI-governance evidence.

var errZMTransport = errors.New("tcp reset " + zlMarker + " https://openapi.zalo.me/v2.0/oa/conversation?access_token=" + zlAccess)

type zmReq struct {
	method, host, path, token, data string
	offset, count                   int
	userID                          string
}

// zmTransport records every request and lets the handler answer by request index (0-based).
type zmTransport struct {
	mu      sync.Mutex
	reqs    []zmReq
	handler func(n int, q zmReq, r *http.Request) (*http.Response, error)
}

func (s *zmTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	q := zmReq{method: r.Method, host: r.URL.Host, path: r.URL.Path, token: r.Header.Get("access_token"), data: r.URL.Query().Get("data"), offset: -1}
	if q.data != "" {
		var p struct {
			Offset, Count int
			UserID        string `json:"user_id"`
		}
		_ = json.Unmarshal([]byte(q.data), &p)
		q.offset, q.count, q.userID = p.Offset, p.Count, p.UserID
	}
	s.mu.Lock()
	n := len(s.reqs)
	s.reqs = append(s.reqs, q)
	s.mu.Unlock()
	resp, err := s.handler(n, q, r)
	if resp != nil {
		resp.Request = r
	}
	return resp, err
}

func (s *zmTransport) requests() []zmReq {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]zmReq(nil), s.reqs...)
}

func (s *zmTransport) offsets() []int {
	var out []int
	for _, q := range s.requests() {
		if q.host == "openapi.zalo.me" {
			out = append(out, q.offset)
		}
	}
	return out
}

func zmAdapter(rt http.RoundTripper) *ZaloOAAdapter {
	a := NewZaloOAAdapter(ZaloOACredentials{AppID: "app-r032", AppSecret: zlSecret, AccessToken: zlAccess, RefreshToken: zlRefresh, OAId: "oa-r032"})
	a.client = &http.Client{Transport: rt}
	return a
}

func zmReply(code int, body string) (*http.Response, error) {
	return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func zmMsg(id string, src int, ms int64, extra string) string {
	return fmt.Sprintf(`{"message_id":%q,"src":%d,"time":%d,"message":"t-%s"%s}`, id, src, ms, id, extra)
}

func zmPage(nested bool, rows ...string) string {
	arr := "[" + strings.Join(rows, ",") + "]"
	if nested {
		return `{"error":0,"message":"Success","data":{"total":999,"data":` + arr + `}}`
	}
	return `{"error":0,"message":"Success","data":` + arr + `}`
}

const zmEmpty = `{"error":0,"message":"Success","data":[]}`

// zmScript serves pages back to back: page i is answered at the offset equal to the number of
// physical rows of pages before it, then an explicit empty page. Other offsets fail the request.
func zmScript(pages ...[]string) *zmTransport {
	byOffset := map[int]string{}
	off := 0
	for i, rows := range pages {
		byOffset[off] = zmPage(i%2 == 1, rows...)
		off += len(rows)
	}
	byOffset[off] = zmEmpty
	return &zmTransport{handler: func(n int, q zmReq, r *http.Request) (*http.Response, error) {
		body, ok := byOffset[q.offset]
		if !ok {
			return nil, fmt.Errorf("unscripted offset %d", q.offset)
		}
		return zmReply(200, body)
	}}
}

func zmIDs(msgs []SyncedMessage) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = m.ExternalID
	}
	return out
}

func zmFetch(t *testing.T, rt http.RoundTripper, conv string) ([]SyncedMessage, error) {
	t.Helper()
	captureZaloLog(t)
	return zmAdapter(rt).FetchMessages(context.Background(), conv, time.Time{})
}

// F02F-01: more than 130 messages across pages, including a short nonterminal page, both data
// shapes, an old history and a short final page before the explicit empty page. since never
// filters; a nonzero since returns the same history as zero.
func TestZaloMessagesTraverseWholeHistoryToExplicitEmpty(t *testing.T) {
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	type item struct {
		id string
		ms int64
	}
	var all []item
	for i := 0; i < 143; i++ {
		all = append(all, item{fmt.Sprintf("m%03d", i), base + int64((i*7919)%1000)}) // out of order
	}
	all = append(all, item{"9007199254740993123", base + 5}, item{"18446744073709551617", base + 6}) // beyond float64 precision
	all = append(all, item{"ancient", 1})                                                            // far older than any since
	sizes := []int{10, 10, 10, 10, 10, 7, 10, 10, 10, 10, 10, 10, 10, 10, 9}                         // short nonterminal page of 7, short final page
	total := 0
	for _, n := range sizes {
		total += n
	}
	if total != len(all) {
		t.Fatalf("fixture sizes %d != items %d", total, len(all))
	}
	var pages [][]string
	i := 0
	for _, n := range sizes {
		var rows []string
		for _, it := range all[i : i+n] {
			rows = append(rows, zmMsg(it.id, int(it.ms%2), it.ms, ""))
		}
		pages = append(pages, rows)
		i += n
	}
	// Expected: stable ascending by time, ties in first-seen order.
	exp := append([]item(nil), all...)
	sort.SliceStable(exp, func(a, b int) bool { return exp[a].ms < exp[b].ms })
	var want []string
	for _, it := range exp {
		want = append(want, it.id)
	}
	var wantOffsets []int
	off := 0
	for _, n := range sizes {
		wantOffsets = append(wantOffsets, off)
		off += n
	}
	wantOffsets = append(wantOffsets, off) // the explicit empty page

	for _, since := range []time.Time{{}, time.UnixMilli(base + 500)} {
		s := zmScript(pages...)
		captureZaloLog(t)
		msgs, err := zmAdapter(s).FetchMessages(context.Background(), "u-1", since)
		if err != nil {
			t.Fatalf("since %v: %v", since, err)
		}
		if got := zmIDs(msgs); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("since %v: %d messages, want %d in stable chronological order\n got %v\nwant %v", since, len(got), len(want), got, want)
		}
		if got := s.offsets(); fmt.Sprint(got) != fmt.Sprint(wantOffsets) {
			t.Fatalf("offsets %v, want physical-row offsets %v", got, wantOffsets)
		}
		for _, q := range s.requests() {
			wantData := fmt.Sprintf(`{"count":10,"offset":%d,"user_id":"u-1"}`, q.offset)
			if q.method != "GET" || q.host != "openapi.zalo.me" || q.path != "/v2.0/oa/conversation" || q.token != zlAccess || q.count != 10 || q.userID != "u-1" || q.data != wantData {
				t.Fatalf("request %+v, want GET conversation with count 10, string user_id, access token (data %s)", q, wantData)
			}
		}
	}
}

// F02F-01: a short page is not terminal; a duplicate-only page that differs from every earlier
// page continues; offsets advance by physical rows including the duplicate.
func TestZaloMessagesShortAndDuplicateOnlyPagesContinue(t *testing.T) {
	a, b, c, d := zmMsg("a", 1, 1000, ""), zmMsg("b", 1, 2000, ""), zmMsg("c", 0, 3000, ""), zmMsg("d", 1, 4000, "")
	s := zmScript([]string{a, b, c}, []string{a}, []string{d}) // page 2 repeats A only (distinct from page 1)
	msgs, err := zmFetch(t, s, "u-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(zmIDs(msgs), ","); got != "a,b,c,d" {
		t.Fatalf("ids %s, want a,b,c,d", got)
	}
	if got := fmt.Sprint(s.offsets()); got != "[0 3 4 5]" {
		t.Fatalf("offsets %s, want [0 3 4 5]", got)
	}
}

// Both explicit terminal shapes return an empty history without error.
func TestZaloMessagesExplicitEmptyTerminalShapes(t *testing.T) {
	for _, body := range []string{`{"error":0,"data":[]}`, `{"error":0,"data":{"data":[]}}`, `{"error":0,"message":"ok","data":{"total":0,"data":[]}}`} {
		s := &zmTransport{handler: func(int, zmReq, *http.Request) (*http.Response, error) { return zmReply(200, body) }}
		msgs, err := zmFetch(t, s, "u-1")
		if err != nil || len(msgs) != 0 || len(s.requests()) != 1 {
			t.Fatalf("%s: msgs %d err %v requests %d", body, len(msgs), err, len(s.requests()))
		}
	}
}

type zmInvalidCase struct{ name, body, want string }

func zmInvalidCases() []zmInvalidCase {
	ok := func(extra string) string {
		return fmt.Sprintf(`{"message_id":"m1","src":1,"time":1790000000000%s}`, extra)
	}
	row := func(r string) string { return `{"error":0,"data":[` + r + `]}` }
	var cases []zmInvalidCase
	add := func(name, body, want string) { cases = append(cases, zmInvalidCase{name, body, want}) }

	add("not an object array", `[]`, "not a JSON object")
	add("null body", `null`, "not a JSON object")
	add("string body", `"x"`, "not a JSON object")
	add("empty body", ``, "not a JSON object")
	add("trailing data", `{"error":0,"data":[]}{"x":1}`, "not a JSON object")
	add("error missing", `{"data":[]}`, "no successful error code")
	add("error quoted", `{"error":"0","data":[]}`, "malformed error code")
	add("error null", `{"error":null,"data":[]}`, "malformed error code")
	add("error fractional", `{"error":0.5,"data":[]}`, "malformed error code")
	add("error provider code", `{"error":1,"message":"provider says no","data":[]}`, "zalo api error 1")
	add("data missing", `{"error":0}`, "no data array")
	add("data null", `{"error":0,"data":null}`, "no data array")
	add("data string", `{"error":0,"data":"x"}`, "no data array")
	add("data number", `{"error":0,"data":3}`, "no data array")
	add("nested data missing", `{"error":0,"data":{"total":1}}`, "no data array")
	add("nested data null", `{"error":0,"data":{"data":null}}`, "no data array")
	add("nested data object", `{"error":0,"data":{"data":{}}}`, "no data array")
	for _, r := range []string{`1`, `null`, `"x"`, `[]`} {
		add("row "+r, row(r), "malformed row")
	}
	for _, v := range []string{``, `"message_id":null,`, `"message_id":123,`, `"message_id":12345678901234567890,`, `"message_id":{},`, `"message_id":"",`, `"message_id":"   ",`} {
		add("message_id "+v, row(`{`+v+`"src":1,"time":1790000000000}`), "no message id")
	}
	for _, v := range []string{``, `"src":null,`, `"src":2,`, `"src":-1,`, `"src":"1",`, `"src":1.5,`, `"src":true,`} {
		add("src "+v, row(`{"message_id":"m1",`+v+`"time":1790000000000}`), "invalid src")
	}
	for _, v := range []string{``, `"time":null`, `"time":0`, `"time":-1`, `"time":"1790000000000"`, `"time":1.5`, `"time":1e3`, `"time":99999999999999999999`, `"time":253402300800000`, `"time":true`} {
		sep := ""
		if v != "" {
			sep = ","
		}
		add("time "+v, row(`{"message_id":"m1","src":1`+sep+v+`}`), "invalid time")
	}
	for _, f := range []string{`"message":5`, `"message":null`, `"type":1`, `"type":null`, `"from_display_name":9`, `"url":5`, `"thumb":[]`, `"url":null`} {
		add("optional "+f, row(ok(","+f)), "mistyped field")
	}
	add("links string", row(ok(`,"links":"x"`)), "malformed links field")
	add("links object", row(ok(`,"links":{}`)), "malformed links field")
	add("links null", row(ok(`,"links":null`)), "malformed links field")
	add("link number", row(ok(`,"links":[1]`)), "malformed link")
	add("link null", row(ok(`,"links":[null]`)), "malformed link")
	add("second link not object", row(ok(`,"links":[{"url":"https://example.invalid/a"},7]`)), "malformed link")
	add("link url number", row(ok(`,"links":[{"url":5}]`)), "mistyped link field")
	add("link name number", row(ok(`,"links":[{"name":5}]`)), "mistyped link field")
	// An invalid duplicate must be validated before deduplication. The valid first row and a valid
	// peer keep every competing guard (conflict, repeated page, non-progress) out of the way, so only
	// row validation can produce this exact message.
	add("invalid duplicate src", row(zmMsg("m1", 1, 1000, "")+","+zmMsg("peer", 1, 2000, "")+","+`{"message_id":"m1","src":2,"time":1000}`), "invalid src")
	add("invalid duplicate field type", row(zmMsg("m1", 1, 1000, "")+","+zmMsg("peer", 1, 2000, "")+","+`{"message_id":"m1","src":1,"time":1000,"url":5}`), "mistyped field")
	add("invalid duplicate id type", row(zmMsg("m1", 1, 1000, "")+","+zmMsg("peer", 1, 2000, "")+","+`{"message_id":1,"src":1,"time":1000}`), "no message id")
	return cases
}

// F02F-02: every malformed envelope or row is incomplete coverage on its first page, with the
// message-specific error, a safe displayed text and no further request.
func TestZaloMessagesRejectMalformedSuccess(t *testing.T) {
	for _, c := range zmInvalidCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			s := &zmTransport{handler: func(int, zmReq, *http.Request) (*http.Response, error) { return zmReply(200, c.body) }}
			msgs, err := zmFetch(t, s, "u-1")
			if !errors.Is(err, ErrZaloMessageCoverageIncomplete) {
				t.Fatalf("want ErrZaloMessageCoverageIncomplete, got %v (msgs %d)", err, len(msgs))
			}
			if errors.Is(err, ErrZaloCoverageIncomplete) {
				t.Fatalf("message failure must not be the conversation sentinel: %v", err)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not name %q", err.Error(), c.want)
			}
			if len(msgs) != 0 {
				t.Fatalf("a failed first page returned %d rows", len(msgs))
			}
			if n := len(s.requests()); n != 1 {
				t.Fatalf("%d requests after an invalid page, want 1", n)
			}
			assertZaloSafe(t, c.name, err)
		})
	}
}

// F02F-02 controls: boundary values and optional-field absence are valid, so the table above
// rejects malformed success rather than everything.
func TestZaloMessagesAcceptBoundaryAndMinimalRows(t *testing.T) {
	rows := []string{
		`{"message_id":"min","src":1,"time":1}`,
		`{"message_id":"max","src":0,"time":253402300799999}`,
		`{"message_id":"9007199254740993123","src":1,"time":5,"links":[]}`,
		`{"message_id":"loc","src":1,"time":6,"type":"location","message":"x"}`,
		`{"message_id":"emp","src":1,"time":7,"type":""}`,
		`{"message_id":" padded ","src":1,"time":8}`,
	}
	msgs, err := zmFetch(t, zmScript(rows), "u-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(zmIDs(msgs), "|"); got != "min|9007199254740993123|loc|emp| padded |max" {
		t.Fatalf("ids %q", got)
	}
	byID := map[string]SyncedMessage{}
	for _, m := range msgs {
		byID[m.ExternalID] = m
	}
	if byID["emp"].ContentType != "text" || byID["min"].ContentType != "text" || byID["loc"].ContentType != "location" {
		t.Fatalf("content types: %q %q %q", byID["emp"].ContentType, byID["min"].ContentType, byID["loc"].ContentType)
	}
	if byID["max"].SentAt.UnixMilli() != 253402300799999 || byID["min"].SentAt.UnixMilli() != 1 {
		t.Fatalf("times not exact: %v %v", byID["max"].SentAt, byID["min"].SentAt)
	}
}

// F02F-02: a later page failing after valid rows returns the diagnostic slice with the error.
func TestZaloMessagesLatePageFailureKeepsDiagnosticRowsOnly(t *testing.T) {
	s := zmScript([]string{zmMsg("a", 1, 1000, ""), zmMsg("b", 1, 2000, "")})
	good := s.handler
	s.handler = func(n int, q zmReq, r *http.Request) (*http.Response, error) {
		if q.offset == 2 {
			return zmReply(200, `{"error":0,"data":[{"message_id":"bad","src":9,"time":3000}]}`)
		}
		return good(n, q, r)
	}
	msgs, err := zmFetch(t, s, "u-1")
	if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !strings.Contains(err.Error(), "page 2") || !strings.Contains(err.Error(), "invalid src") {
		t.Fatalf("err %v", err)
	}
	if got := strings.Join(zmIDs(msgs), ","); got != "a,b" {
		t.Fatalf("diagnostic rows %q, want a,b", got)
	}
}

// F02F-03: complete mapped fields for each message kind, inherited precedence and raw data.
func TestZaloMessagesMapEveryKind(t *testing.T) {
	rows := []string{
		`{"message_id":"t1","src":1,"time":1000,"type":"text","message":"hi","from_display_name":"Khach"}`,
		`{"message_id":"p1","src":0,"time":2000,"type":"photo","message":"","url":"https://example.invalid/p.jpg","from_display_name":"Someone"}`,
		`{"message_id":"g1","src":1,"time":3000,"type":"gif","thumb":"https://example.invalid/g.gif","from_display_name":"K2"}`,
		`{"message_id":"v1","src":1,"time":4000,"type":"voice","url":"https://example.invalid/v.aac"}`,
		`{"message_id":"f1","src":1,"time":5000,"type":"file","url":"https://example.invalid/top","thumb":"https://example.invalid/th","links":[{"url":"https://example.invalid/file.pdf","name":"bao.pdf"},{"url":"https://example.invalid/ignored","name":"ignored"}]}`,
		`{"message_id":"f2","src":1,"time":6000,"type":"file","links":[{"url":"https://example.invalid/f2"}]}`,
		`{"message_id":"s1","src":1,"time":7000,"type":"sticker","url":"https://example.invalid/s1","thumb":"https://example.invalid/s1t"}`,
		`{"message_id":"l1","src":1,"time":8000,"type":"location"}`,
		`{"message_id":"d1","src":1,"time":9000,"message":"no type"}`,
	}
	msgs, err := zmFetch(t, zmScript(rows[:4], rows[4:]), "u-1")
	if err != nil || len(msgs) != len(rows) {
		t.Fatalf("msgs %d err %v", len(msgs), err)
	}
	type wantMsg struct {
		sender, name, content, ctype string
		atts                         []Attachment
	}
	want := map[string]wantMsg{
		"t1": {"customer", "Khach", "hi", "text", nil},
		"p1": {"agent", "OA", "", "photo", []Attachment{{Type: "photo", URL: "https://example.invalid/p.jpg", Name: "photo-p1"}}},
		"g1": {"customer", "K2", "", "gif", []Attachment{{Type: "gif", URL: "https://example.invalid/g.gif", Name: "gif-g1"}}},
		"v1": {"customer", "", "", "voice", []Attachment{{Type: "voice", URL: "https://example.invalid/v.aac", Name: "voice-v1"}}},
		"f1": {"customer", "", "", "file", []Attachment{{Type: "file", URL: "https://example.invalid/file.pdf", Name: "bao.pdf"}}},
		"f2": {"customer", "", "", "file", []Attachment{{Type: "file", URL: "https://example.invalid/f2", Name: "file-f2"}}},
		"s1": {"customer", "", "", "sticker", []Attachment{{Type: "sticker", URL: "https://example.invalid/s1", Name: "sticker-s1"}}},
		"l1": {"customer", "", "", "location", nil},
		"d1": {"customer", "", "no type", "text", nil},
	}
	for _, m := range msgs {
		w, ok := want[m.ExternalID]
		if !ok {
			t.Fatalf("unexpected message %q", m.ExternalID)
		}
		if m.SenderType != w.sender || m.SenderName != w.name || m.Content != w.content || m.ContentType != w.ctype {
			t.Fatalf("%s mapped %+v, want %+v", m.ExternalID, m, w)
		}
		if fmt.Sprint(m.Attachments) != fmt.Sprint(w.atts) {
			t.Fatalf("%s attachments %v, want %v", m.ExternalID, m.Attachments, w.atts)
		}
		if m.RawData["message_id"] != m.ExternalID {
			t.Fatalf("%s raw data lost: %v", m.ExternalID, m.RawData)
		}
		if n, ok := m.RawData["time"].(json.Number); !ok || n.String() != fmt.Sprint(m.SentAt.UnixMilli()) {
			t.Fatalf("%s raw time %v (%T) does not match SentAt %d", m.ExternalID, m.RawData["time"], m.RawData["time"], m.SentAt.UnixMilli())
		}
	}
}

// F02F-03: out-of-order times and ties sort stably across pages; exact duplicates collapse to the
// first mapping; conflicting duplicates are incomplete coverage.
func TestZaloMessagesOrderDeduplicateAndConflict(t *testing.T) {
	b, a, c := zmMsg("b", 1, 2000, ""), zmMsg("a", 1, 1000, ""), zmMsg("c", 1, 2000, "")
	d := zmMsg("d", 1, 2000, "")
	msgs, err := zmFetch(t, zmScript([]string{b, a}, []string{c, b}, []string{d}), "u-1")
	if err != nil {
		t.Fatal(err)
	}
	// a first (earliest); then the 2000 ties in first-seen order b, c, d; the repeated b is dropped.
	if got := strings.Join(zmIDs(msgs), ","); got != "a,b,c,d" {
		t.Fatalf("ids %s, want a,b,c,d", got)
	}
	conflict := zmMsg("a", 0, 1000, "")
	cmsgs, err := zmFetch(t, zmScript([]string{a, b}, []string{conflict, c}), "u-1")
	if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("conflicting duplicate: %v", err)
	}
	if got := strings.Join(zmIDs(cmsgs), ","); got != "a,b" {
		t.Fatalf("diagnostic rows %q", got)
	}
}

// F02F-04: a repeated canonical page fails, even across envelope text changes and a cycle A-B-A.
func TestZaloMessagesRepeatedPageFails(t *testing.T) {
	pageA := []string{zmMsg("a", 1, 1000, ""), zmMsg("b", 1, 2000, "")}
	pageB := []string{zmMsg("c", 1, 3000, "")}
	// Each fixture keeps serving valid recurring pages until an outer ceiling (a synthetic
	// transport failure at request 7, never a slice index), so a missing repeat guard reaches the
	// ceiling and fails the named assertions below instead of panicking or hanging.
	const ceiling = 6
	cases := []struct {
		name         string
		body         func(n int) string
		wantRequests int
	}{
		{"cycle A-B-A", func(n int) string {
			if n%2 == 0 {
				return zmPage(false, pageA...)
			}
			return zmPage(false, pageB...)
		}, 3},
		{"same rows with changed envelope text", func(n int) string {
			if n == 0 {
				return `{"error":0,"message":"first","data":[` + strings.Join(pageA, ",") + `]}`
			}
			return `{"error":0,"message":"second","extra":42,"data":{"total":7,"data":[` + strings.Join(pageA, ",") + `]}}`
		}, 2},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			s := &zmTransport{handler: func(n int, q zmReq, r *http.Request) (*http.Response, error) {
				if n >= ceiling {
					return nil, errors.New("synthetic ceiling: the repeat guard did not stop the traversal")
				}
				return zmReply(200, c.body(n))
			}}
			_, err := zmFetch(t, s, "u-1")
			if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !strings.Contains(err.Error(), "repeats an earlier page") {
				t.Fatalf("want the repeated-page failure, got %v", err)
			}
			if n := len(s.requests()); n != c.wantRequests {
				t.Fatalf("%d requests, want %d: the traversal must stop at the first repeated page", n, c.wantRequests)
			}
		})
	}
}

// F02F-04: the finite budget counts the terminal page. Pages carry distinct IDs and are never
// repeated, so only the budget (not the cycle guard) can end the traversal; the server ceiling is
// finite so the original unbounded loop fails an assertion instead of hanging.
func TestZaloMessagesPageBudgetCountsTerminalPage(t *testing.T) {
	serve := func(nonempty int) *zmTransport {
		return &zmTransport{handler: func(n int, q zmReq, r *http.Request) (*http.Response, error) {
			if n >= 700 {
				return nil, errors.New("synthetic ceiling: traversal exceeded the budget")
			}
			if n >= nonempty {
				return zmReply(200, zmEmpty)
			}
			return zmReply(200, zmPage(false, zmMsg(fmt.Sprintf("m%04d", n), 1, int64(1000+n), "")))
		}}
	}
	// 499 nonempty pages plus an empty 500th page is complete.
	s := serve(499)
	msgs, err := zmFetch(t, s, "u-1")
	if err != nil || len(msgs) != 499 || len(s.requests()) != 500 {
		t.Fatalf("499+empty: msgs %d err %v requests %d", len(msgs), err, len(s.requests()))
	}
	// 500 nonempty pages never reach a terminal page: incomplete, and no page 501 is requested.
	s = serve(1000)
	msgs, err = zmFetch(t, s, "u-1")
	if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !strings.Contains(err.Error(), "page budget of 500") {
		t.Fatalf("500 nonempty: err %v", err)
	}
	if n := len(s.requests()); n != 500 {
		t.Fatalf("500 nonempty pages made %d requests, want exactly 500 (no page 501)", n)
	}
	if len(msgs) != 500 {
		t.Fatalf("diagnostic rows %d, want 500", len(msgs))
	}
}

// F02F-04: cancellation is checked before a request and after each response, including the
// terminal empty page, and keeps errors.Is for the context cause with a safe displayed error.
func TestZaloMessagesCancellationBarriers(t *testing.T) {
	t.Run("before the first request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s := zmScript([]string{zmMsg("a", 1, 1000, "")})
		captureZaloLog(t)
		_, err := zmAdapter(s).FetchMessages(ctx, "u-1", time.Time{})
		if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !errors.Is(err, context.Canceled) || len(s.requests()) != 0 {
			t.Fatalf("err %v requests %d", err, len(s.requests()))
		}
		assertZaloSafe(t, "pre-cancel", err)
	})
	cancelAt := func(offset int) (*zmTransport, context.Context) {
		ctx, cancel := context.WithCancel(context.Background())
		s := zmScript([]string{zmMsg("a", 1, 1000, ""), zmMsg("b", 1, 2000, "")}, []string{zmMsg("c", 1, 3000, "")})
		inner := s.handler
		s.handler = func(n int, q zmReq, r *http.Request) (*http.Response, error) {
			if q.offset == offset {
				cancel() // the response still arrives; the barrier must refuse to trust it
			}
			return inner(n, q, r)
		}
		return s, ctx
	}
	t.Run("at the terminal empty response", func(t *testing.T) {
		s, ctx := cancelAt(3)
		captureZaloLog(t)
		msgs, err := zmAdapter(s).FetchMessages(ctx, "u-1", time.Time{})
		if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !errors.Is(err, context.Canceled) {
			t.Fatalf("a cancelled terminal response must not succeed: msgs %d err %v", len(msgs), err)
		}
		assertZaloSafe(t, "terminal cancel", err)
	})
	t.Run("after a nonempty response", func(t *testing.T) {
		s, ctx := cancelAt(2)
		captureZaloLog(t)
		msgs, err := zmAdapter(s).FetchMessages(ctx, "u-1", time.Time{})
		if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !errors.Is(err, context.Canceled) {
			t.Fatalf("err %v", err)
		}
		if got := fmt.Sprint(s.offsets()); got != "[0 2]" {
			t.Fatalf("offsets %s: no request may follow the cancelled page", got)
		}
		if got := strings.Join(zmIDs(msgs), ","); got != "a,b" {
			t.Fatalf("diagnostic rows %q, want only the pre-cancel page", got)
		}
	})
}

// F02F-05: every redirect status is blocked before a destination request; the redirect is a
// failed (non-2xx) response with a safe error, on the first page and on a later page.
func TestZaloMessagesBlockRedirects(t *testing.T) {
	for _, code := range []int{301, 302, 307, 308} {
		for _, later := range []bool{false, true} {
			name := fmt.Sprintf("%d later=%v", code, later)
			t.Run(name, func(t *testing.T) {
				var destination int32
				s := &zmTransport{}
				s.handler = func(n int, q zmReq, r *http.Request) (*http.Response, error) {
					if q.host != "openapi.zalo.me" {
						atomic.AddInt32(&destination, 1)
						return zmReply(200, zmEmpty)
					}
					if later && q.offset == 0 {
						return zmReply(200, zmPage(false, zmMsg("a", 1, 1000, "")))
					}
					resp, _ := zmReply(code, "")
					resp.Header.Set("Location", "https://evil.example/steal?access_token="+zlAccess)
					return resp, nil
				}
				msgs, err := zmFetch(t, s, "u-1")
				if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !strings.Contains(err.Error(), fmt.Sprintf("http %d", code)) {
					t.Fatalf("err %v", err)
				}
				if atomic.LoadInt32(&destination) != 0 {
					t.Fatalf("redirect destination was requested")
				}
				want := 1
				if later {
					want = 2
				}
				if n := len(s.requests()); n != want {
					t.Fatalf("%d requests, want %d", n, want)
				}
				if later && strings.Join(zmIDs(msgs), ",") != "a" {
					t.Fatalf("diagnostic rows %v", zmIDs(msgs))
				}
				assertZaloSafe(t, name, err)
				if strings.Contains(err.Error(), "evil.example") {
					t.Fatalf("destination leaked: %v", err)
				}
			})
		}
	}
}

type zmCloseBody struct {
	io.Reader
	closed *int32
}

func (b zmCloseBody) Close() error { atomic.AddInt32(b.closed, 1); return nil }

type zmBrokenBody struct{ closed *int32 }

func (b zmBrokenBody) Read([]byte) (int, error) { return 0, errZMTransport }
func (b zmBrokenBody) Close() error             { atomic.AddInt32(b.closed, 1); return nil }

// F02F-05/06: transport, read, decode, HTTP, API and oversize failures on a later page are
// incomplete coverage with safe text, an errors.Is cause where one exists, a closed body and no
// secret or provider text in the error or the log.
func TestZaloMessagesLaterFailuresAreSafe(t *testing.T) {
	good := zmPage(false, zmMsg("a", 1, 1000, ""), zmMsg("b", 1, 2000, ""))
	type fail struct {
		name   string
		resp   func(closed *int32) (*http.Response, error)
		want   string
		cause  error
		closes bool
	}
	fails := []fail{
		{"transport error", func(*int32) (*http.Response, error) { return nil, errZMTransport }, "zalo api request failed", errZMTransport, false},
		{"read error", func(c *int32) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: zmBrokenBody{c}}, nil
		}, "zalo response read failed", errZMTransport, true},
		{"invalid JSON", func(c *int32) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: zmCloseBody{strings.NewReader("{" + zlMarker), c}}, nil
		}, "not a JSON object", nil, true},
		{"HTTP 500 with provider text", func(c *int32) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Header: http.Header{}, Body: zmCloseBody{strings.NewReader("provider says " + zlMarker + zlAccess), c}}, nil
		}, "http 500", nil, true},
		{"API error with provider text", func(c *int32) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: zmCloseBody{strings.NewReader(`{"error":-1234,"message":"provider says ` + zlMarker + `"}`), c}}, nil
		}, "zalo api error -1234", nil, true},
		{"oversize body", func(c *int32) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: zmCloseBody{strings.NewReader(`{"error":0,"data":[],"pad":"` + strings.Repeat("x", 8<<20) + `"}`), c}}, nil
		}, "8 MiB limit", nil, true},
	}
	for _, f := range fails {
		f := f
		t.Run(f.name, func(t *testing.T) {
			var closed int32
			s := &zmTransport{}
			s.handler = func(n int, q zmReq, r *http.Request) (*http.Response, error) {
				if q.offset == 0 {
					return zmReply(200, good)
				}
				return f.resp(&closed)
			}
			logs := captureZaloLog(t)
			msgs, err := zmAdapter(s).FetchMessages(context.Background(), "u-1", time.Time{})
			if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !strings.Contains(err.Error(), f.want) || !strings.Contains(err.Error(), "page 2") {
				t.Fatalf("err %v, want class + %q at page 2", err, f.want)
			}
			if f.cause != nil && !errors.Is(err, f.cause) {
				t.Fatalf("errors.Is lost the cause: %v", err)
			}
			if f.closes && atomic.LoadInt32(&closed) != 1 {
				t.Fatalf("response body closed %d times, want 1", closed)
			}
			if got := strings.Join(zmIDs(msgs), ","); got != "a,b" {
				t.Fatalf("diagnostic rows %q", got)
			}
			assertZaloSafe(t, f.name, err)
			assertZaloLogSafe(t, logs)
		})
	}
}

// F02F-05: an expired token refreshes once, persists the complete synthetic pair, then retries the
// same offset; later pages use the new token. Retries are requests but not logical pages.
func TestZaloMessagesRefreshPersistsThenRetriesSameOffset(t *testing.T) {
	var persisted [][2]string
	s := &zmTransport{}
	s.handler = func(n int, q zmReq, r *http.Request) (*http.Response, error) {
		switch {
		case q.host == "oauth.zaloapp.com":
			return zmReply(200, `{"error":0,"access_token":"`+zlNewAcc+`","refresh_token":"`+zlNewRef+`"}`)
		case q.token == zlAccess && q.offset == 2:
			return zmReply(200, `{"error":-216,"message":"expired"}`)
		case q.offset == 0:
			return zmReply(200, zmPage(false, zmMsg("a", 1, 1000, ""), zmMsg("b", 1, 2000, "")))
		case q.offset == 2:
			return zmReply(200, zmPage(true, zmMsg("c", 1, 3000, "")))
		case q.offset == 3:
			return zmReply(200, zmEmpty)
		}
		return nil, fmt.Errorf("unscripted %+v", q)
	}
	a := zmAdapter(s)
	a.SetTokenRefreshCallback(func(acc, ref string) error { persisted = append(persisted, [2]string{acc, ref}); return nil })
	captureZaloLog(t)
	msgs, err := a.FetchMessages(context.Background(), "u-1", time.Time{})
	if err != nil || strings.Join(zmIDs(msgs), ",") != "a,b,c" {
		t.Fatalf("msgs %v err %v", zmIDs(msgs), err)
	}
	if len(persisted) != 1 || persisted[0] != [2]string{zlNewAcc, zlNewRef} {
		t.Fatalf("persisted %v", persisted)
	}
	var seq []string
	for _, q := range s.requests() {
		seq = append(seq, fmt.Sprintf("%s:%d:%s", q.host, q.offset, q.token))
	}
	want := fmt.Sprint([]string{"openapi.zalo.me:0:" + zlAccess, "openapi.zalo.me:2:" + zlAccess, "oauth.zaloapp.com:-1:", "openapi.zalo.me:2:" + zlNewAcc, "openapi.zalo.me:3:" + zlNewAcc})
	if fmt.Sprint(seq) != want {
		t.Fatalf("request sequence %v\nwant %s", seq, want)
	}
	if got := fmt.Sprint(s.offsets()); got != "[0 2 2 3]" {
		t.Fatalf("offsets %s: the retry reuses its offset", got)
	}
}

// F02F-05: a second expiry, a failed persistence, an incomplete token pair and a failing refresh
// each stop after at most one refresh and one retry.
func TestZaloMessagesRefreshFailuresStop(t *testing.T) {
	cbErr := errors.New("persist failed " + zlNewAcc + " " + zlMarker)
	cases := []struct {
		name     string
		refresh  string
		callback func(string, string) error
		wantCall int
		wantReq  int
		wantIs   error
		wantText string
	}{
		{"second expiry", `{"error":0,"access_token":"` + zlNewAcc + `","refresh_token":"` + zlNewRef + `"}`, func(string, string) error { return nil }, 1, 3, nil, "zalo api error -216"},
		{"persistence fails", `{"error":0,"access_token":"` + zlNewAcc + `","refresh_token":"` + zlNewRef + `"}`, func(string, string) error { return cbErr }, 1, 2, cbErr, "persistence failed"},
		{"incomplete pair", `{"error":0,"access_token":"` + zlNewAcc + `"}`, func(string, string) error { return nil }, 0, 2, nil, "incomplete token pair"},
		{"provider refuses refresh", `{"error":-14014,"message":"provider says ` + zlMarker + `"}`, func(string, string) error { return nil }, 0, 2, nil, "refresh error -14014"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			s := &zmTransport{handler: func(n int, q zmReq, r *http.Request) (*http.Response, error) {
				if q.host == "oauth.zaloapp.com" {
					return zmReply(200, c.refresh)
				}
				return zmReply(200, `{"error":-216}`)
			}}
			a := zmAdapter(s)
			calls := 0
			a.SetTokenRefreshCallback(func(x, y string) error { calls++; return c.callback(x, y) })
			captureZaloLog(t)
			_, err := a.FetchMessages(context.Background(), "u-1", time.Time{})
			if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || !strings.Contains(err.Error(), c.wantText) {
				t.Fatalf("err %v, want %q", err, c.wantText)
			}
			if c.wantIs != nil && !errors.Is(err, c.wantIs) {
				t.Fatalf("errors.Is lost the callback cause: %v", err)
			}
			if calls != c.wantCall {
				t.Fatalf("persistence callback called %d times, want %d", calls, c.wantCall)
			}
			if n := len(s.requests()); n != c.wantReq {
				t.Fatalf("%d requests, want %d", n, c.wantReq)
			}
			assertZaloSafe(t, c.name, err)
			wantToken := zlAccess // replaced only after a complete pair was persisted successfully
			if c.name == "second expiry" {
				wantToken = zlNewAcc
			}
			if a.creds.AccessToken != wantToken {
				t.Fatalf("credentials %q, want %q", a.creds.AccessToken, wantToken)
			}
		})
	}
}

// F02F-06: a blank conversation ID fails locally with no authenticated request.
func TestZaloMessagesBlankConversationIDMakesNoRequest(t *testing.T) {
	for _, id := range []string{"", "   ", "\t\n"} {
		s := &zmTransport{handler: func(int, zmReq, *http.Request) (*http.Response, error) { return zmReply(200, zmEmpty) }}
		msgs, err := zmFetch(t, s, id)
		if !errors.Is(err, ErrZaloMessageCoverageIncomplete) || len(msgs) != 0 || len(s.requests()) != 0 {
			t.Fatalf("id %q: msgs %d err %v requests %d", id, len(msgs), err, len(s.requests()))
		}
	}
}

// F02F-06: a successful traversal and a failing one log only the safe summary line.
func TestZaloMessagesLogsStaySafe(t *testing.T) {
	logs := captureZaloLog(t)
	if _, err := zmAdapter(zmScript([]string{zmMsg("a", 1, 1000, `,"url":"https://example.invalid/secret"`)})).FetchMessages(context.Background(), "u-1", time.Time{}); err != nil {
		t.Fatal(err)
	}
	assertZaloLogSafe(t, logs)
	if !strings.Contains(logs.String(), "[zalo] API conversation: status=200") {
		t.Fatalf("expected the summary log line, got %q", logs.String())
	}
	if strings.Contains(logs.String(), "example.invalid") || strings.Contains(logs.String(), "u-1") {
		t.Fatalf("log leaks message content or conversation id: %q", logs.String())
	}
}
