package channels

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// CCMAI-RUNTIME-030 (F02-D): Pancake message traversal. A loopback httptest server (or a closed
// synthetic RoundTripper) stands in for pages.fm and records every request; no real Pancake
// endpoint or token is used. The tests assert the local traversal contract (explicit empty page,
// strict rows, physical-row offsets, bounded budget, deterministic order), never live behaviour.

const pmStamp = "2006-01-02T15:04:05.000000"

var pmSince = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func pmRow(id string, at time.Time) string {
	return fmt.Sprintf(`{"id":%q,"page_id":"1000001","type":"INBOX","original_message":"tin %s","from":{"id":"2000002","name":"Khach"},"inserted_at":%q}`,
		id, id, at.UTC().Format(pmStamp))
}

func pmPage(rows ...string) string {
	return `{"success":true,"messages":[` + strings.Join(rows, ",") + `]}`
}

type pmRequest struct {
	method, path, count string
	query               url.Values
}

// pmServer answers message requests by their current_count position ("" for the first request) and
// records every request. A script entry may be a fixed function or the dynamic fallback.
type pmServer struct {
	mu       sync.Mutex
	requests []pmRequest
	byPos    map[string]func(w http.ResponseWriter)
	dynamic  func(pos int, w http.ResponseWriter)
	srv      *httptest.Server
}

func newPMServer(t *testing.T, byPos map[string]func(w http.ResponseWriter)) *pmServer {
	t.Helper()
	s := &pmServer{byPos: byPos}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		s.mu.Lock()
		s.requests = append(s.requests, pmRequest{method: r.Method, path: r.URL.EscapedPath(), count: q.Get("current_count"), query: q})
		dynamic := s.dynamic
		s.mu.Unlock()
		if q.Get("page_access_token") != pcToken {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		if h, ok := s.byPos[q.Get("current_count")]; ok {
			h(w)
			return
		}
		if dynamic != nil {
			pos, _ := strconv.Atoi(q.Get("current_count"))
			dynamic(pos, w)
			return
		}
		http.Error(w, "unscripted position", http.StatusTeapot)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *pmServer) adapter() *PancakeAdapter {
	a := newTestPancakeAdapter(s.srv.URL)
	a.creds.PageAccessToken = pcToken
	return a
}

func (s *pmServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func (s *pmServer) positions() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, r := range s.requests {
		out = append(out, r.count)
	}
	return strings.Join(out, ",")
}

func pmIDs(msgs []SyncedMessage) string {
	var ids []string
	for _, m := range msgs {
		ids = append(ids, m.ExternalID)
	}
	return strings.Join(ids, ",")
}

func pmFetch(t *testing.T, s *pmServer, since time.Time) ([]SyncedMessage, error) {
	t.Helper()
	return s.adapter().FetchMessages(context.Background(), "1000001_2000002", since)
}

// F02D-01: v1 GET, escaped IDs, token auth, initial omission, exact physical-row positions
// (duplicates and rows outside since counted), and a retry that reuses its position.
func TestPancakeMessageRequestInventory(t *testing.T) {
	var throttled sync.Once
	rows1 := pmPage(pmRow("m3", pmSince.Add(3*time.Hour)), pmRow("m2", pmSince.Add(2*time.Hour)), pmRow("old", pmSince.Add(-time.Hour)))
	rows2 := pmPage(pmRow("m2", pmSince.Add(2*time.Hour)), pmRow("m1", pmSince.Add(time.Hour)))
	s := newPMServer(t, map[string]func(http.ResponseWriter){
		"": body(rows1),
		"3": func(w http.ResponseWriter) {
			limited := false
			throttled.Do(func() { limited = true })
			if limited {
				w.WriteHeader(http.StatusTooManyRequests) // retried at the SAME position
				return
			}
			fmt.Fprint(w, rows2)
		},
		"5": body(pmPage()),
	})
	const conv = "page/1 conv?x"
	got, err := s.adapter().FetchMessages(context.Background(), conv, pmSince)
	if err != nil {
		t.Fatalf("FetchMessages: %v", err)
	}
	if pmIDs(got) != "m1,m2,m3" {
		t.Fatalf("returned IDs = %s, want m1,m2,m3 (old row and duplicate excluded)", pmIDs(got))
	}
	if p := s.positions(); p != ",3,3,5" {
		t.Fatalf("request positions = %q, want \",3,3,5\" (first omitted, physical rows 3 then 5, retry repeats 3)", p)
	}
	wantPath := "/v1/pages/" + pancakeTestPageID + "/conversations/page%2F1%20conv%3Fx/messages"
	for i, r := range s.requests {
		if r.method != http.MethodGet || r.path != wantPath || r.query.Get("page_access_token") != pcToken {
			t.Fatalf("request %d = %+v, want GET %s with the page token", i, r, wantPath)
		}
		if i == 0 && r.query.Has("current_count") {
			t.Fatalf("first request must omit current_count: %v", r.query)
		}
	}
}

// F02D-02: success only after a validated explicit empty array; every other shape fails.
func TestPancakeMessagesEmptyFirstPageIsSuccess(t *testing.T) {
	s := newPMServer(t, map[string]func(http.ResponseWriter){"": body(pmPage())})
	got, err := pmFetch(t, s, pmSince)
	if err != nil || len(got) != 0 || s.count() != 1 {
		t.Fatalf("explicit empty page: %d messages, %d requests, err %v", len(got), s.count(), err)
	}
}

func TestPancakeMessagesMalformedPagesFailClosed(t *testing.T) {
	good := pmRow("ok", pmSince.Add(time.Hour))
	cases := map[string]struct {
		body     string
		coverage bool
	}{
		"missing messages":          {`{"success":true}`, true},
		"null messages":             {`{"success":true,"messages":null}`, true},
		"object messages":           {`{"success":true,"messages":{"id":"x"}}`, true},
		"string messages":           {`{"success":true,"messages":"[]"}`, true},
		"number messages":           {`{"success":true,"messages":0}`, true},
		"top-level null":            {`null`, true},
		"row not an object":         {`{"success":true,"messages":["x"]}`, true},
		"null row":                  {`{"success":true,"messages":[null]}`, true},
		"good row then null row":    {`{"success":true,"messages":[` + good + `,null]}`, true},
		"malformed array":           {`{"success":true,"messages":[` + good + `,]}`, false},
		"top-level array":           {`[]`, false},
		"top-level string":          {`"ok"`, false},
		"malformed success":         {`{"success":"yes","messages":[]}`, false},
		"explicit provider failure": {`{"success":false,"error_code":105,"message":"bad page ` + pcToken + `"}`, false},
		"undecodable":               {`<html>` + pcBodyMarker + `</html>`, false},
	}
	for name, c := range cases {
		s := newPMServer(t, map[string]func(http.ResponseWriter){"": body(c.body)})
		_, err := pmFetch(t, s, pmSince)
		assertSafeError(t, name, err)
		if c.coverage && !errors.Is(err, ErrPancakeMessageCoverageIncomplete) {
			t.Errorf("%s: err = %v, want message coverage incomplete", name, err)
		}
		if errors.Is(err, ErrPancakeCoverageIncomplete) {
			t.Errorf("%s: message failure must not reuse the conversation error identity", name)
		}
		if s.count() != 1 {
			t.Errorf("%s: %d requests, want 1", name, s.count())
		}
	}
}

// Absent success stays compatible when the body is otherwise a valid explicit empty page.
func TestPancakeMessagesAbsentSuccessFieldIsCompatible(t *testing.T) {
	s := newPMServer(t, map[string]func(http.ResponseWriter){
		"":  body(`{"messages":[` + pmRow("a", pmSince.Add(time.Hour)) + `]}`),
		"1": body(`{"messages":[]}`),
	})
	got, err := pmFetch(t, s, pmSince)
	if err != nil || pmIDs(got) != "a" {
		t.Fatalf("absent success: %q, err %v", pmIDs(got), err)
	}
}

// F02D-03: every row is validated before dedupe or since filtering, including old/repeated rows.
func TestPancakeMessagesInvalidRowsAreNeverFilteredAway(t *testing.T) {
	old := pmSince.Add(-48 * time.Hour).UTC().Format(pmStamp)
	fresh := pmSince.Add(time.Hour).UTC().Format(pmStamp)
	cases := map[string]string{
		"missing id":                `{"inserted_at":"` + fresh + `"}`,
		"null id":                   `{"id":null,"inserted_at":"` + fresh + `"}`,
		"numeric id":                `{"id":5,"inserted_at":"` + fresh + `"}`,
		"blank id":                  `{"id":"   ","inserted_at":"` + fresh + `"}`,
		"blank id on an old row":    `{"id":"","inserted_at":"` + old + `"}`,
		"missing inserted_at":       `{"id":"x"}`,
		"null inserted_at":          `{"id":"x","inserted_at":null}`,
		"numeric inserted_at":       `{"id":"x","inserted_at":1}`,
		"invalid inserted_at":       `{"id":"x","inserted_at":"yesterday"}`,
		"impossible inserted_at":    `{"id":"x","inserted_at":"2020-13-45T99:00:00"}`,
		"zero inserted_at":          `{"id":"x","inserted_at":"0001-01-01T00:00:00"}`,
		"invalid time on an old id": `{"id":"x","inserted_at":"2020-13-45T99:00:00"}`,
		"wrong-typed attachments":   `{"id":"x","inserted_at":"` + fresh + `","attachments":"photo"}`,
	}
	for name, row := range cases {
		s := newPMServer(t, map[string]func(http.ResponseWriter){"": body(`{"success":true,"messages":[` + row + `]}`)})
		_, err := pmFetch(t, s, pmSince)
		assertSafeError(t, name, err)
		if !errors.Is(err, ErrPancakeMessageCoverageIncomplete) {
			t.Errorf("%s: err = %v, want message coverage incomplete", name, err)
		}
	}

	// An invalid row hidden behind a valid one on the same page, and an invalid REPEAT of a
	// previously seen ID on a later page, both fail instead of being deduplicated away.
	// The repeat sits beside a NEW valid row, so dedupe-first code would see progress and finish
	// successfully; only validate-before-dedupe reports the invalid row.
	valid := pmRow("m1", pmSince.Add(time.Hour))
	s := newPMServer(t, map[string]func(http.ResponseWriter){
		"":  body(pmPage(valid)),
		"1": body(`{"success":true,"messages":[{"id":"m1","inserted_at":"not-a-time"},` + pmRow("m2", pmSince.Add(2*time.Hour)) + `]}`),
		"3": body(pmPage()),
	})
	if _, err := pmFetch(t, s, pmSince); !errors.Is(err, ErrPancakeMessageCoverageIncomplete) || !strings.Contains(err.Error(), "valid id or inserted_at") {
		t.Fatalf("invalid repeat of a seen ID: err %v after %s", err, s.positions())
	}
	s = newPMServer(t, map[string]func(http.ResponseWriter){
		"":  body(pmPage(valid)),
		"1": body(`{"success":true,"messages":[{"id":"older","inserted_at":"2020-01-01T00:00:00"},{"id":"bad"}]}`),
		"3": body(pmPage()),
	})
	if _, err := pmFetch(t, s, pmSince); !errors.Is(err, ErrPancakeMessageCoverageIncomplete) {
		t.Fatalf("invalid old row after a filtered one: err %v", err)
	}
}

// F02D-04: since only filters output (inclusive); no early success on old rows or page order.
func TestPancakeMessagesSinceFiltersOutputWithoutEarlyStop(t *testing.T) {
	newest, mid := pmSince.Add(3*time.Hour), pmSince.Add(2*time.Hour)
	script := map[string]func(http.ResponseWriter){
		// an old row precedes eligible rows on page 1 and trails eligible ones on page 2
		"":  body(pmPage(pmRow("oldA", pmSince.Add(-time.Hour)), pmRow("newA", newest))),
		"2": body(pmPage(pmRow("newB", mid), pmRow("oldB", pmSince.Add(-2*time.Hour)))),
		"4": body(pmPage(pmRow("boundary", pmSince))), // exactly since: included
		"5": body(pmPage()),
	}
	s := newPMServer(t, script)
	got, err := pmFetch(t, s, pmSince)
	if err != nil {
		t.Fatalf("FetchMessages: %v", err)
	}
	if pmIDs(got) != "boundary,newB,newA" {
		t.Fatalf("eligible IDs = %s, want exactly boundary,newB,newA", pmIDs(got))
	}
	if s.positions() != ",2,4,5" {
		t.Fatalf("positions = %q, want all four pages walked (old rows are not terminal)", s.positions())
	}

	// A zero since keeps every row.
	s = newPMServer(t, script)
	all, err := pmFetch(t, s, time.Time{})
	if err != nil || pmIDs(all) != "oldB,oldA,boundary,newB,newA" {
		t.Fatalf("zero since: %s, err %v", pmIDs(all), err)
	}
}

// F02D-05: first-seen mapping, stable chronological order for newest-first, oldest-first and
// mixed pages, equal-time ties by encounter order, and unchanged content/sender/media mapping.
func TestPancakeMessagesDeduplicateAndOrderChronologically(t *testing.T) {
	t1, t2, t3 := pmSince.Add(time.Hour), pmSince.Add(2*time.Hour), pmSince.Add(3*time.Hour)
	first := `{"id":"dup","original_message":"FIRST","from":{"id":"2000002","name":"Khach"},"inserted_at":"` + t2.Format(pmStamp) + `"}`
	second := `{"id":"dup","original_message":"SECOND","from":{"id":"1000001","name":"Page","admin_name":"NV"},"inserted_at":"` + t3.Format(pmStamp) + `"}`
	s := newPMServer(t, map[string]func(http.ResponseWriter){
		// newest-first page with a within-page duplicate, then an oldest-first page that overlaps
		"":  body(pmPage(pmRow("c3", t3), first, second, pmRow("tieA", t1), pmRow("tieB", t1))),
		"5": body(pmPage(pmRow("c0", pmSince), pmRow("tieB", t1), pmRow("c2b", t2))),
		"8": body(pmPage()),
	})
	got, err := pmFetch(t, s, pmSince)
	if err != nil {
		t.Fatalf("FetchMessages: %v", err)
	}
	// c0 (T), tieA/tieB (t1, encounter order), then the t2 rows in encounter order (dup, c2b), c3 (t3)
	if want := "c0,tieA,tieB,dup,c2b,c3"; pmIDs(got) != want {
		t.Fatalf("order = %s, want %s", pmIDs(got), want)
	}
	for _, m := range got {
		if m.ExternalID == "dup" && (m.Content != "FIRST" || m.SenderType != "customer" || !m.SentAt.Equal(t2)) {
			t.Fatalf("duplicate must keep its first-seen mapping, got %+v", m)
		}
	}
	for i := 1; i < len(got); i++ {
		if got[i].SentAt.Before(got[i-1].SentAt) {
			t.Fatalf("not chronological at %d: %v then %v", i, got[i-1].SentAt, got[i].SentAt)
		}
	}
}

// Real-shaped mapping/redaction regression through the exact-terminal traversal: all seven
// fixture messages are returned once, chronologically, with the same mapping and no personal data.
func TestPancakeMessagesRealShapedPayloadKeepsMappingAndRedaction(t *testing.T) {
	s := newPMServer(t, map[string]func(http.ResponseWriter){"": body(pancakeMessagesFixture), "7": body(pmPage())})
	got, err := pmFetch(t, s, time.Time{})
	if err != nil {
		t.Fatalf("FetchMessages: %v", err)
	}
	if want := "m_text_customer,m_text_agent,m_sticker,m_escaped,m_photo,m_video,m_file"; pmIDs(got) != want {
		t.Fatalf("order = %s, want %s", pmIDs(got), want)
	}
	byID := map[string]SyncedMessage{}
	for _, m := range got {
		byID[m.ExternalID] = m
	}
	if m := byID["m_text_agent"]; m.SenderType != "agent" || m.SenderName != "Nhan Vien A" {
		t.Errorf("agent mapping changed: %+v", m)
	}
	if m := byID["m_sticker"]; m.ContentType != "sticker" || m.Content != "" {
		t.Errorf("sticker mapping changed: %+v", m)
	}
	if m := byID["m_file"]; m.ContentType != "attachment" || m.Attachments[0].Name != "Bao gia (1).pdf" {
		t.Errorf("file mapping changed: %+v", m)
	}
	raw := fmt.Sprint(byID["m_text_customer"].RawData)
	if strings.Contains(raw, "khach@example.com") || strings.Contains(raw, "0900000000") {
		t.Errorf("raw data leaks personal info: %s", raw)
	}
}

// F02D-06: nonprogress and cycles fail closed with no uncontrolled extra request.
func TestPancakeMessagesNonprogressFailsClosed(t *testing.T) {
	a, b := pmRow("a", pmSince.Add(time.Hour)), pmRow("b", pmSince.Add(2*time.Hour))
	cases := map[string]struct {
		script   map[string]func(http.ResponseWriter)
		requests int
	}{
		"duplicate-only replay": {map[string]func(http.ResponseWriter){"": body(pmPage(a, b)), "2": body(pmPage(a, b)), "4": body(pmPage())}, 2},
		"alternating replay":    {map[string]func(http.ResponseWriter){"": body(pmPage(a)), "1": body(pmPage(b)), "2": body(pmPage(a)), "3": body(pmPage())}, 3},
		"duplicate within page": {map[string]func(http.ResponseWriter){"": body(pmPage(a, a)), "2": body(pmPage(a)), "3": body(pmPage())}, 2},
	}
	for name, c := range cases {
		s := newPMServer(t, c.script)
		got, err := pmFetch(t, s, pmSince)
		assertSafeError(t, name, err)
		if !errors.Is(err, ErrPancakeMessageCoverageIncomplete) || !strings.Contains(err.Error(), "repeats") {
			t.Errorf("%s: err = %v", name, err)
		}
		if s.count() != c.requests {
			t.Errorf("%s: %d requests (%s), want %d and no more", name, s.count(), s.positions(), c.requests)
		}
		if len(got) == 0 {
			t.Errorf("%s: diagnostic rows missing", name)
		}
	}
}

// Overlap with fresh IDs continues; only the fixed page budget bounds it.
func TestPancakeMessagesOverlapWithNewIDsContinues(t *testing.T) {
	s := newPMServer(t, map[string]func(http.ResponseWriter){
		"":  body(pmPage(pmRow("a", pmSince.Add(3*time.Hour)), pmRow("b", pmSince.Add(2*time.Hour)))),
		"2": body(pmPage(pmRow("b", pmSince.Add(2*time.Hour)), pmRow("c", pmSince.Add(time.Hour)))), // b overlaps, c is new
		"4": body(pmPage()),
	})
	got, err := pmFetch(t, s, pmSince)
	if err != nil || pmIDs(got) != "c,b,a" || s.positions() != ",2,4" {
		t.Fatalf("overlap: %s over %s, err %v", pmIDs(got), s.positions(), err)
	}
}

func pmChain(rows int) func(pos int, w http.ResponseWriter) {
	return func(pos int, w http.ResponseWriter) {
		if pos >= rows {
			fmt.Fprint(w, pmPage())
			return
		}
		fmt.Fprint(w, pmPage(pmRow(fmt.Sprintf("n%04d", pos), pmSince.Add(time.Duration(pos+1)*time.Minute))))
	}
}

// 199 nonempty pages then an empty page 200 succeed; 200 nonempty pages fail without request 201.
func TestPancakeMessagesPageBudgetBoundary(t *testing.T) {
	s := newPMServer(t, nil)
	s.dynamic = pmChain(pancakeMaxPages - 1)
	got, err := pmFetch(t, s, pmSince)
	if err != nil || len(got) != pancakeMaxPages-1 || s.count() != pancakeMaxPages {
		t.Fatalf("199 nonempty + empty page 200: %d rows, %d requests, err %v", len(got), s.count(), err)
	}
	if got[0].ExternalID != "n0000" || got[len(got)-1].ExternalID != fmt.Sprintf("n%04d", pancakeMaxPages-2) {
		t.Fatalf("budget-boundary order wrong: first %s last %s", got[0].ExternalID, got[len(got)-1].ExternalID)
	}

	s = newPMServer(t, nil)
	s.dynamic = pmChain(pancakeMaxPages + 5)
	got, err = pmFetch(t, s, pmSince)
	assertSafeError(t, "200 nonempty pages", err)
	if !errors.Is(err, ErrPancakeMessageCoverageIncomplete) || !strings.Contains(err.Error(), "page budget") {
		t.Fatalf("budget exhaustion: err %v", err)
	}
	if s.count() != pancakeMaxPages || len(got) != pancakeMaxPages {
		t.Fatalf("budget exhaustion: %d requests / %d diagnostic rows, want exactly %d (no request %d)", s.count(), len(got), pancakeMaxPages, pancakeMaxPages+1)
	}
}

// Late transport/API/size failures after valid pages are errors; the valid rows are diagnostic.
func TestPancakeMessagesLaterPageFailuresAreNeverSuccess(t *testing.T) {
	first := body(pmPage(pmRow("p1", pmSince.Add(time.Hour)), pmRow("p2", pmSince.Add(2*time.Hour))))
	hijack := func(w http.ResponseWriter) {
		if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
			conn.Close()
		}
	}
	failures := map[string]func(http.ResponseWriter){
		"429 exhausted":   status(http.StatusTooManyRequests),
		"http 500":        status(http.StatusInternalServerError),
		"api error":       body(`{"success":false,"error_code":4,"message":"rate ` + pcToken + `"}`),
		"api echoes URL":  body(`{"success":false,"error_code":4,"message":"https://pages.fm/x?page_access_token=tok%2DR023%2DSECRET ` + pcBodyMarker + `"}`),
		"undecodable":     body("<html>" + pcBodyMarker + "</html>"),
		"missing array":   body(`{"success":true}`),
		"oversized":       body(pmPage() + strings.Repeat(" ", pancakeMaxResponseBytes)),
		"network failure": hijack,
	}
	for name, page2 := range failures {
		s := newPMServer(t, map[string]func(http.ResponseWriter){"": first, "2": page2, "5": body(pmPage())})
		got, err := pmFetch(t, s, pmSince)
		assertSafeError(t, name, err)
		if len(got) != 2 {
			t.Errorf("%s: diagnostic rows = %d, want 2", name, len(got))
		}
		if name == "429 exhausted" && s.count() != 1+pancakeMaxRetries+1 {
			t.Errorf("429: %d requests, want page 1 plus %d attempts", s.count(), pancakeMaxRetries+1)
		}
		if name == "oversized" && !strings.Contains(err.Error(), "size budget") {
			t.Errorf("oversized: err %v", err)
		}
	}
}

// Transport causes (which can carry the token URL) are scrubbed for messages too.
func TestPancakeMessagesTransportCauseCannotEchoRequestURL(t *testing.T) {
	a := newTestPancakeAdapter("https://pancake.invalid")
	a.creds.PageAccessToken = pcToken
	a.client = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("proxy failed for %s", r.URL.String())
	})}
	_, err := a.FetchMessages(context.Background(), "c1", pmSince)
	assertSafeError(t, "transport cause", err)
}

// Cancellation is recognizable and never success; the orderings are fixed by in-line hooks, not
// by sleeps.
func TestPancakeMessagesCancellationIsNeverSuccess(t *testing.T) {
	synthetic := func(a *PancakeAdapter, hook func(n int), bodyFor func(n int) string) *int {
		n := 0
		a.client = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			n++
			if hook != nil {
				hook(n)
			}
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(bodyFor(n))), Request: r}, nil
		})}
		return &n
	}

	t.Run("before the first request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		a := newTestPancakeAdapter("https://pancake.invalid")
		n := synthetic(a, nil, func(int) string { return pmPage() })
		_, err := a.FetchMessages(ctx, "c1", pmSince)
		if !errors.Is(err, context.Canceled) || *n != 0 {
			t.Fatalf("pre-cancelled: err %v after %d requests", err, *n)
		}
	})

	t.Run("on the terminal empty page", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		a := newTestPancakeAdapter("https://pancake.invalid")
		synthetic(a, func(int) { cancel() }, func(int) string { return pmPage() })
		got, err := a.FetchMessages(ctx, "c1", pmSince)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled terminal page returned success: %d rows, err %v", len(got), err)
		}
	})

	t.Run("between pages", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		a := newTestPancakeAdapter("https://pancake.invalid")
		n := synthetic(a, func(n int) {
			if n == 1 {
				cancel()
			}
		}, func(n int) string { return pmPage(pmRow("x", pmSince.Add(time.Hour))) })
		_, err := a.FetchMessages(ctx, "c1", pmSince)
		if !errors.Is(err, context.Canceled) || *n != 1 {
			t.Fatalf("cancel after page 1: err %v after %d requests, want 1", err, *n)
		}
	})

	t.Run("during retry backoff", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		a := newTestPancakeAdapter("https://pancake.invalid")
		a.backoff = time.Hour // the backoff can never elapse; only the cancellation can end the wait
		n := 0
		a.client = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			n++
			cancel()
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
		})}
		_, err := a.FetchMessages(ctx, "c1", pmSince)
		if !errors.Is(err, context.Canceled) || n != 1 {
			t.Fatalf("cancel during retry: err %v after %d requests", err, n)
		}
	})
}
