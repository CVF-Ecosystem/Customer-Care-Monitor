package channels

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// CCMAI-RUNTIME-023 (F02-B): Pancake conversation enumeration walks the cursor chain to an
// explicit empty page with one fixed `until`, examines every row, and fails closed on anything
// that could hide unvisited conversations. A local httptest server stands in for Pancake and
// records every request; no real Pancake endpoint or token is used.

const (
	pcToken      = "tok-R023-SECRET"
	pcStamp      = "2006-01-02T15:04:05.000000"
	pcBodyMarker = "R023-BODY-MARKER"
)

func pcRow(id, typ string, updated time.Time) string {
	return fmt.Sprintf(`{"id":%q,"type":%q,"updated_at":%q,"from":{"id":"u-%s","name":"Khach %s"},"recent_phone_numbers":["0900000000"]}`,
		id, typ, updated.UTC().Format(pcStamp), id, id)
}

func pcRows(prefix string, from, to int, updated time.Time) []string {
	var out []string
	for i := from; i < to; i++ {
		out = append(out, pcRow(fmt.Sprintf("%s%04d", prefix, i), "INBOX", updated))
	}
	return out
}

func pcPage(rows []string) string {
	return `{"success":true,"conversations":[` + strings.Join(rows, ",") + `]}`
}

// pcServer answers conversation requests by cursor (last_conversation_id; "" for page 1) and
// records every query it receives.
type pcServer struct {
	mu      sync.Mutex
	queries []url.Values
	byCur   map[string]func(w http.ResponseWriter)
	srv     *httptest.Server
}

func newPCServer(t *testing.T, byCursor map[string]func(w http.ResponseWriter)) *pcServer {
	t.Helper()
	s := &pcServer{byCur: byCursor}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		s.mu.Lock()
		s.queries = append(s.queries, q)
		s.mu.Unlock()
		if r.URL.Path != "/v2/pages/"+pancakeTestPageID+"/conversations" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		h, ok := s.byCur[q.Get("last_conversation_id")]
		if !ok {
			http.Error(w, "unscripted cursor", http.StatusTeapot)
			return
		}
		h(w)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *pcServer) adapter() *PancakeAdapter {
	a := newTestPancakeAdapter(s.srv.URL)
	a.creds.PageAccessToken = pcToken
	return a
}

func (s *pcServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queries)
}

func (s *pcServer) cursors() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, q := range s.queries {
		out = append(out, q.Get("last_conversation_id"))
	}
	return out
}

func body(b string) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) { fmt.Fprint(w, b) }
}

func status(code int) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) { w.WriteHeader(code); fmt.Fprint(w, pcBodyMarker) }
}

func assertSafeError(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: want an error", label)
	}
	for _, banned := range []string{pcToken, "page_access_token", "last_conversation_id", pcBodyMarker, "127.0.0.1"} {
		if strings.Contains(err.Error(), banned) {
			t.Fatalf("%s: error leaks %q: %v", label, banned, err)
		}
	}
}

// The window: 143 eligible INBOX rows over three nonempty pages (the second one short) and an
// empty terminal page; COMMENT rows (one is the last physical row of page 1), an older row ahead
// of newer rows, a row exactly at `since` and a duplicate ID across pages.
func pcWindow(t *testing.T, since time.Time) (*pcServer, map[string]bool) {
	newer := since.Add(48 * time.Hour)
	want := map[string]bool{}
	add := func(rows []string, prefix string, from, to int) []string {
		for i := from; i < to; i++ {
			want[fmt.Sprintf("%s%04d", prefix, i)] = true
		}
		return rows
	}
	page1 := add(pcRows("a", 0, 59, newer), "a", 0, 59)
	page1 = append(page1, pcRow("comment-1", "COMMENT", newer)) // 60 rows; cursor is the COMMENT row
	page2 := []string{pcRow("old-1", "INBOX", since.Add(-time.Second))}
	page2 = append(page2, add(pcRows("b", 0, 43, newer), "b", 0, 43)...)
	page2 = append(page2, pcRow("boundary", "INBOX", since)) // equal to since: included
	want["boundary"] = true
	page3 := append(add(pcRows("c", 0, 40, newer), "c", 0, 40), pcRow("a0003", "INBOX", newer)) // duplicate
	s := newPCServer(t, map[string]func(http.ResponseWriter){
		"":          body(pcPage(page1)),
		"comment-1": body(pcPage(page2)), // 45 rows: short but not terminal
		"boundary":  body(pcPage(page3)),
		"a0003":     body(pcPage(nil)), // empty: terminal
	})
	return s, want
}

func TestPancakeEnumeratesTheWholeWindowToAnEmptyPage(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	s, want := pcWindow(t, since)
	before := time.Now().Unix()
	got, err := s.adapter().FetchRecentConversations(context.Background(), since, 0)
	if err != nil {
		t.Fatalf("exhaustive enumeration failed: %v", err)
	}
	if len(want) != 143 {
		t.Fatalf("fixture drifted: %d eligible", len(want))
	}
	seen := map[string]int{}
	for _, c := range got {
		seen[c.ExternalID]++
	}
	for id := range want {
		if seen[id] != 1 {
			t.Fatalf("conversation %s returned %d times, want once", id, seen[id])
		}
	}
	if len(got) != len(want) || seen["old-1"] != 0 || seen["comment-1"] != 0 {
		t.Fatalf("got %d rows (old-1 %d, comment-1 %d), want exactly the %d eligible", len(got), seen["old-1"], seen["comment-1"], len(want))
	}
	// Physical last-row cursor chain, including the filtered COMMENT row and the short page.
	if c := strings.Join(s.cursors(), ","); c != ",comment-1,boundary,a0003" {
		t.Fatalf("cursor chain = %q", c)
	}
	// One since and one fixed until on every request; type=INBOX kept.
	until := s.queries[0].Get("until")
	u, _ := strconv.ParseInt(until, 10, 64)
	if u < before || u > time.Now().Unix() {
		t.Fatalf("until %q is not the fetch-start time", until)
	}
	for i, q := range s.queries {
		if q.Get("since") != strconv.FormatInt(since.Unix(), 10) || q.Get("until") != until || q.Get("type") != "INBOX" {
			t.Fatalf("request %d filters changed: %v", i, q)
		}
	}
}

// A short nonempty page is not the end: the next page is requested.
func TestPancakeShortPageIsNotTerminal(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	s := newPCServer(t, map[string]func(http.ResponseWriter){
		"":   body(pcPage([]string{pcRow("x1", "INBOX", since.Add(time.Hour))})),
		"x1": body(pcPage([]string{pcRow("x2", "INBOX", since.Add(time.Hour))})),
		"x2": body(pcPage(nil)),
	})
	got, err := s.adapter().FetchRecentConversations(context.Background(), since, 0)
	if err != nil || len(got) != 2 || s.count() != 3 {
		t.Fatalf("short pages: %d rows, %d requests, err %v", len(got), s.count(), err)
	}
}

func TestPancakeLimitThatWouldTruncateIsAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	s, _ := pcWindow(t, since)
	got, err := s.adapter().FetchRecentConversations(context.Background(), since, 100)
	if !errors.Is(err, ErrPancakeCoverageIncomplete) || len(got) != 100 {
		t.Fatalf("limit 100 over 143 rows: %d rows, err %v", len(got), err)
	}
	s2 := newPCServer(t, map[string]func(http.ResponseWriter){
		"":      body(pcPage(pcRows("f", 0, 5, since.Add(time.Hour)))),
		"f0004": body(pcPage(nil)),
	})
	if got, err := s2.adapter().FetchRecentConversations(context.Background(), since, 5); err != nil || len(got) != 5 {
		t.Fatalf("limit equal to the window: %d rows, err %v", len(got), err)
	}
}

// Only an actual empty array terminates; every malformed page or row fails closed.
func TestPancakeMalformedPagesFailClosed(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	good := pcRow("ok", "INBOX", since.Add(time.Hour))
	cases := map[string]string{
		"missing conversations":    `{"success":true}`,
		"null conversations":       `{"success":true,"conversations":null}`,
		"object conversations":     `{"success":true,"conversations":{"id":"x"}}`,
		"string conversations":     `{"success":true,"conversations":"[]"}`,
		"row not an object":        `{"success":true,"conversations":["x"]}`,
		"null row":                 `{"success":true,"conversations":[null]}`,
		"empty id":                 `{"success":true,"conversations":[{"id":"","type":"INBOX","updated_at":"2026-09-02T00:00:00.000000"}]}`,
		"missing updated_at":       `{"success":true,"conversations":[{"id":"x","type":"INBOX"}]}`,
		"invalid updated_at":       `{"success":true,"conversations":[{"id":"x","type":"INBOX","updated_at":"yesterday"}]}`,
		"bad time on COMMENT row":  `{"success":true,"conversations":[{"id":"x","type":"COMMENT","updated_at":"nonsense"}]}`,
		"bad time on old row":      `{"success":true,"conversations":[{"id":"x","type":"INBOX","updated_at":"2020-13-45T99:00:00"}]}`,
		"numeric id":               `{"success":true,"conversations":[{"id":5,"type":"INBOX","updated_at":"2026-09-02T00:00:00.000000"}]}`,
		"good row then broken row": `{"success":true,"conversations":[` + good + `,{"id":"y"}]}`,
		"api success false":        `{"success":false,"error_code":105,"message":"bad page ` + pcToken + `"}`,
		"undecodable":              `<html>` + pcBodyMarker + `</html>`,
	}
	for name, b := range cases {
		s := newPCServer(t, map[string]func(http.ResponseWriter){"": body(b)})
		got, err := s.adapter().FetchRecentConversations(context.Background(), since, 0)
		assertSafeError(t, name, err)
		if name != "api success false" && name != "undecodable" && !errors.Is(err, ErrPancakeCoverageIncomplete) {
			t.Errorf("%s: err = %v, want incomplete coverage", name, err)
		}
		if name == "good row then broken row" && len(got) != 1 {
			t.Errorf("%s: diagnostic rows = %d, want 1", name, len(got))
		}
	}
}

// A later-page failure returns a safe error with the rows seen so far for diagnostics; a
// positive control on the same server shape succeeds.
func TestPancakeLaterPageFailuresReturnSafeErrors(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	first := body(pcPage(pcRows("p", 0, 5, since.Add(time.Hour))))
	hijack := func(w http.ResponseWriter) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}
	truncated := func(w http.ResponseWriter) {
		w.Header().Set("Content-Length", "500")
		fmt.Fprint(w, `{"success":true,"conversations":[`)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}
	failures := map[string]func(http.ResponseWriter){
		"429 exhausted":   status(http.StatusTooManyRequests),
		"http 500":        status(http.StatusInternalServerError),
		"api error":       body(`{"success":false,"error_code":4,"message":"rate ` + pcToken + `"}`),
		"undecodable":     body("<html>" + pcBodyMarker + "</html>"),
		"network failure": hijack,
		"read failure":    truncated,
	}
	for name, page2 := range failures {
		s := newPCServer(t, map[string]func(http.ResponseWriter){"": first, "p0004": page2, "after": body(pcPage(nil))})
		got, err := s.adapter().FetchRecentConversations(context.Background(), since, 0)
		assertSafeError(t, name, err)
		if len(got) != 5 {
			t.Fatalf("%s: diagnostic rows = %d, want 5", name, len(got))
		}
		if name == "429 exhausted" && s.count() != 1+pancakeMaxRetries+1 {
			t.Fatalf("429: %d requests, want page 1 plus %d attempts", s.count(), pancakeMaxRetries+1)
		}
	}
	// Positive control: the same first page followed by an empty page succeeds.
	s := newPCServer(t, map[string]func(http.ResponseWriter){"": first, "p0004": body(pcPage(nil))})
	if got, err := s.adapter().FetchRecentConversations(context.Background(), since, 0); err != nil || len(got) != 5 || s.count() != 2 {
		t.Fatalf("positive control: %d rows, %d requests, err %v", len(got), s.count(), err)
	}
}

// A transport that returns nested *url.Error values (each carrying the token URL) is scrubbed.
func TestPancakeNestedURLErrorsAreScrubbed(t *testing.T) {
	a := newTestPancakeAdapter("https://pancake.invalid")
	a.creds.PageAccessToken = pcToken
	a.client = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		inner := &url.Error{Op: "Get", URL: r.URL.String(), Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}}
		return nil, &url.Error{Op: "Get", URL: r.URL.String(), Err: inner}
	})}
	_, err := a.FetchRecentConversations(context.Background(), time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), 0)
	assertSafeError(t, "nested url error", err)
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("scrubbed error lost its cause: %v", err)
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPancakeRepeatedCursorFailsClosed(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	row := func(id string) string { return pcRow(id, "INBOX", since.Add(time.Hour)) }
	cases := map[string]map[string]func(http.ResponseWriter){
		// page 2 ends on the same row as page 1: the cursor does not progress
		"non-progressing": {"": body(pcPage([]string{row("r1")})), "r1": body(pcPage([]string{row("r2"), row("r1")}))},
		// r1 -> r2 -> r1 again
		"cycle": {"": body(pcPage([]string{row("r1")})), "r1": body(pcPage([]string{row("r2")})), "r2": body(pcPage([]string{row("r1")}))},
	}
	for name, script := range cases {
		s := newPCServer(t, script)
		_, err := s.adapter().FetchRecentConversations(context.Background(), since, 0)
		if !errors.Is(err, ErrPancakeCoverageIncomplete) || !strings.Contains(err.Error(), "repeats") {
			t.Fatalf("%s: err %v after %d requests", name, err, s.count())
		}
	}
}

func TestPancakePageBudgetIsAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	script := map[string]func(http.ResponseWriter){}
	cur := ""
	for i := 0; i < pancakeMaxPages+5; i++ {
		id := fmt.Sprintf("n%04d", i)
		script[cur] = body(pcPage([]string{pcRow(id, "INBOX", since.Add(time.Hour))}))
		cur = id
	}
	s := newPCServer(t, script)
	got, err := s.adapter().FetchRecentConversations(context.Background(), since, 0)
	if !errors.Is(err, ErrPancakeCoverageIncomplete) || !strings.Contains(err.Error(), "page budget") {
		t.Fatalf("endless chain: err %v", err)
	}
	if s.count() != pancakeMaxPages || len(got) != pancakeMaxPages {
		t.Fatalf("%d requests / %d rows, want exactly the %d-page budget", s.count(), len(got), pancakeMaxPages)
	}
}

func TestPancakeCancellationStopsWithAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	s := newPCServer(t, map[string]func(http.ResponseWriter){
		"": func(w http.ResponseWriter) {
			fmt.Fprint(w, pcPage(pcRows("k", 0, 3, since.Add(time.Hour))))
			cancel()
		},
		"k0002": body(pcPage(nil)),
	})
	got, err := s.adapter().FetchRecentConversations(ctx, since, 0)
	if err == nil {
		t.Fatalf("cancelled enumeration returned success with %d rows", len(got))
	}
	assertSafeError(t, "cancelled", err)
}
