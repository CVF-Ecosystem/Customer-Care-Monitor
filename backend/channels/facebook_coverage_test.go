package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// CCMAI-RUNTIME-022 (F02-A): FetchRecentConversations enumerates the whole `since` window
// across every page, fails closed on anything that could hide unvisited conversations and never
// puts the page access token into an error. A synthetic Graph transport serves the pages and
// records every request it receives; no real Facebook endpoint is contacted.

const (
	fbTestPage  = "PAGE-R022"
	fbTestToken = "TOKEN-R022-SECRET"
	fbTestStamp = "2006-01-02T15:04:05-0700"
)

type fbRow struct {
	id      string
	updated time.Time
}

func fbRowJSON(r fbRow) map[string]interface{} {
	return map[string]interface{}{
		"id": r.id, "updated_time": r.updated.UTC().Format(fbTestStamp),
		"participants": map[string]interface{}{"data": []interface{}{
			map[string]interface{}{"id": fbTestPage, "name": "Page"},
			map[string]interface{}{"id": "u-" + r.id, "name": "Customer " + r.id},
		}},
	}
}

func fbRows(prefix string, from, to int, updated time.Time) []fbRow {
	var out []fbRow
	for i := from; i < to; i++ {
		out = append(out, fbRow{id: fmt.Sprintf("%s%04d", prefix, i), updated: updated})
	}
	return out
}

// graphServer serves a scripted list of pages; each handler gets the 0-based request number
// and returns the response body (or a transport error).
type graphServer struct {
	mu       sync.Mutex
	requests []string
	pages    []func(n int) (body string, err error)
	onCall   func(n int)
}

func (g *graphServer) RoundTrip(r *http.Request) (*http.Response, error) {
	g.mu.Lock()
	n := len(g.requests)
	g.requests = append(g.requests, r.URL.String())
	g.mu.Unlock()
	if g.onCall != nil {
		g.onCall(n)
	}
	if n >= len(g.pages) {
		return nil, fmt.Errorf("unscripted request %d to %s", n, r.URL.Host)
	}
	body, err := g.pages[n](n)
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func (g *graphServer) count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.requests)
}

func (g *graphServer) hosts() map[string]bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := map[string]bool{}
	for _, raw := range g.requests {
		if u, err := url.Parse(raw); err == nil {
			out[u.Host] = true
		}
	}
	return out
}

func newFBTestAdapter(g *graphServer) *FacebookAdapter {
	a := NewFacebookAdapter(FacebookCredentials{PageID: fbTestPage, AccessToken: fbTestToken})
	a.client = &http.Client{Transport: g}
	return a
}

func fbNextURL(cursor string) string {
	return fmt.Sprintf("https://graph.facebook.com/v21.0/%s/conversations?access_token=%s&after=%s&fields=id", fbTestPage, fbTestToken, cursor)
}

func fbPageBody(rows []fbRow, next string) string {
	data := []interface{}{}
	for _, r := range rows {
		data = append(data, fbRowJSON(r))
	}
	page := map[string]interface{}{"data": data}
	if next != "" {
		page["paging"] = map[string]interface{}{"next": next}
	}
	b, _ := json.Marshal(page)
	return string(b)
}

func fbStatic(body string) func(int) (string, error) {
	return func(int) (string, error) { return body, nil }
}

func fbIDs(convs []SyncedConversation) []string {
	var out []string
	for _, c := range convs {
		out = append(out, c.ExternalID)
	}
	return out
}

// A window of 270 eligible conversations over four pages, with an older row ahead of newer
// eligible rows, a row exactly at `since`, a duplicate across a page boundary and a terminal
// page. The page order is deliberately not newest-first.
func fbWindowServer(since time.Time) (*graphServer, map[string]bool) {
	newer := since.Add(48 * time.Hour)
	want := map[string]bool{}
	add := func(rows []fbRow) []fbRow {
		for _, r := range rows {
			want[r.id] = true
		}
		return rows
	}
	page1 := add(fbRows("a", 0, 100, newer))
	old := fbRow{id: "old-1", updated: since.Add(-time.Second)} // before since: skipped, but not a stop signal
	page2 := append([]fbRow{old}, add(fbRows("b", 0, 100, newer))...)
	boundary := fbRow{id: "boundary", updated: since}                       // equal to since: included
	page3 := append(add([]fbRow{boundary}), fbRows("b", 99, 100, newer)...) // duplicate of b0099
	page3 = append(page3, add(fbRows("c", 0, 40, newer))...)
	page4 := add(fbRows("d", 0, 29, newer)) // terminal
	g := &graphServer{pages: []func(int) (string, error){
		fbStatic(fbPageBody(page1, fbNextURL("p2"))),
		fbStatic(fbPageBody(page2, fbNextURL("p3"))),
		fbStatic(fbPageBody(page3, fbNextURL("p4"))),
		fbStatic(fbPageBody(page4, "")),
	}}
	return g, want
}

func TestFacebookEnumeratesEveryPageOfTheWindowExactlyOnce(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	g, want := fbWindowServer(since)
	got, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
	if err != nil {
		t.Fatalf("exhaustive enumeration failed: %v", err)
	}
	if len(want) != 270 || g.count() != 4 {
		t.Fatalf("fixture drifted: %d eligible, %d requests", len(want), g.count()) // positive observer
	}
	seen := map[string]int{}
	for _, id := range fbIDs(got) {
		seen[id]++
	}
	for id := range want {
		if seen[id] != 1 {
			t.Fatalf("conversation %s returned %d times, want exactly once", id, seen[id])
		}
	}
	if len(got) != len(want) || seen["old-1"] != 0 {
		t.Fatalf("got %d rows (old-1 seen %d), want exactly the %d eligible", len(got), seen["old-1"], len(want))
	}
	if h := g.hosts(); len(h) != 1 || !h["graph.facebook.com"] {
		t.Fatalf("unexpected hosts contacted: %v", h)
	}
}

// A positive limit that the window exceeds is an error, never a truncated success.
func TestFacebookLimitThatWouldTruncateIsAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	g, _ := fbWindowServer(since)
	got, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 100)
	if !errors.Is(err, ErrFacebookCoverageIncomplete) || len(got) != 100 {
		t.Fatalf("limit 100 over a 270-row window: %d rows, err %v", len(got), err)
	}
	// The same limit is fine when the window fits (the guard is not an unconditional error).
	g2 := &graphServer{pages: []func(int) (string, error){fbStatic(fbPageBody(fbRows("z", 0, 100, since.Add(time.Hour)), ""))}}
	if got, err := newFBTestAdapter(g2).FetchRecentConversations(context.Background(), since, 100); err != nil || len(got) != 100 {
		t.Fatalf("exactly 100 rows with limit 100: %d rows, err %v", len(got), err)
	}
}

func TestFacebookEmptyPagesAndTerminalRules(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	// An empty page that still has paging.next is not the end.
	g := &graphServer{pages: []func(int) (string, error){
		fbStatic(fbPageBody(nil, fbNextURL("p2"))),
		fbStatic(fbPageBody(fbRows("e", 0, 3, since.Add(time.Hour)), fbNextURL("p3"))),
		fbStatic(fbPageBody(nil, "")), // terminal empty page
	}}
	got, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
	if err != nil || len(got) != 3 || g.count() != 3 {
		t.Fatalf("empty page with next: %d rows, %d requests, err %v", len(got), g.count(), err)
	}
	// A terminal empty first page is a valid empty window.
	g = &graphServer{pages: []func(int) (string, error){fbStatic(fbPageBody(nil, ""))}}
	if got, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0); err != nil || len(got) != 0 {
		t.Fatalf("empty terminal page: %d rows, err %v", len(got), err)
	}
}

// Every untrustworthy page is an incomplete-coverage error, not a quiet early stop.
func TestFacebookMalformedPagesFailClosed(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ok := fbRowJSON(fbRow{id: "good", updated: since.Add(time.Hour)})
	cases := map[string]string{
		"missing data":      `{"paging":{}}`,
		"data not an array": `{"data":{"id":"x"}}`,
		"row not an object": `{"data":["x"]}`,
		"missing id":        `{"data":[{"updated_time":"2026-09-02T00:00:00+0000"}]}`,
		"empty id":          `{"data":[{"id":"","updated_time":"2026-09-02T00:00:00+0000"}]}`,
		"missing time":      `{"data":[{"id":"x"}]}`,
		"invalid time":      `{"data":[{"id":"x","updated_time":"yesterday"}]}`,
		"next not a string": `{"data":[],"paging":{"next":5}}`,
		"malformed old row": `{"data":[{"id":"old","updated_time":"nonsense"}]}`,
		"null data":         `{"data":null}`,
	}
	goodThenBroken, _ := json.Marshal(map[string]interface{}{"data": []interface{}{ok, map[string]interface{}{"id": "bad"}}})
	cases["good row then broken"] = string(goodThenBroken)
	for name, body := range cases {
		g := &graphServer{pages: []func(int) (string, error){fbStatic(body)}}
		_, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
		if !errors.Is(err, ErrFacebookCoverageIncomplete) {
			t.Errorf("%s: err = %v, want incomplete coverage", name, err)
		}
	}
}

// A failure on a later page returns an error (with the rows seen so far for diagnostics);
// no error text contains the token, a URL or the response body.
func TestFacebookLaterPageFailuresReturnSafeErrors(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	first := fbStatic(fbPageBody(fbRows("p", 0, 5, since.Add(time.Hour)), fbNextURL("p2")))
	failures := map[string]func(int) (string, error){
		"graph error object": fbStatic(`{"error":{"message":"synthetic page failure","code":4}}`),
		"non-JSON body":      fbStatic("<html>" + fbTestToken + " gateway page</html>"),
		"transport error": func(int) (string, error) {
			return "", &url.Error{Op: "Get", URL: fbNextURL("p2"), Err: errors.New("connection reset by peer")}
		},
	}
	for name, page2 := range failures {
		g := &graphServer{pages: []func(int) (string, error){first, page2}}
		got, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
		if err == nil || g.count() != 2 {
			t.Fatalf("%s: err %v after %d requests, want an error on page 2", name, err, g.count())
		}
		if len(got) != 5 {
			t.Fatalf("%s: diagnostic rows = %d, want the 5 seen before the failure", name, len(got))
		}
		for _, banned := range []string{fbTestToken, "access_token", "graph.facebook.com", "gateway page"} {
			if strings.Contains(err.Error(), banned) {
				t.Fatalf("%s: error leaks %q: %v", name, banned, err)
			}
		}
	}
	// The sanitization keeps the real cause for diagnosis.
	g := &graphServer{pages: []func(int) (string, error){first, failures["transport error"]}}
	_, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
	if !strings.Contains(err.Error(), "connection reset by peer") {
		t.Fatalf("sanitized error lost its cause: %v", err)
	}
}

// Only a fresh Graph conversations URL over HTTPS is followed: a repeated cursor, a cross-host
// or non-HTTPS link and any other path stop the run before another request is made.
func TestFacebookUnsafeOrRepeatedPaginationFailsClosed(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	rows := fbRows("p", 0, 2, since.Add(time.Hour))
	good := "https://graph.facebook.com/v21.0/" + fbTestPage + "/conversations"
	cases := map[string]string{
		"cross host":      "https://evil.example/v21.0/" + fbTestPage + "/conversations?after=x",
		"http scheme":     "http://graph.facebook.com/v21.0/" + fbTestPage + "/conversations?after=x",
		"other page path": "https://graph.facebook.com/v21.0/OTHER/conversations?after=x",
		"other endpoint":  "https://graph.facebook.com/v21.0/" + fbTestPage + "/messages?after=x",
		"other version":   "https://graph.facebook.com/v1.0/" + fbTestPage + "/conversations?after=x",
		"userinfo":        "https://user:pw@graph.facebook.com/v21.0/" + fbTestPage + "/conversations?after=x",
		"odd port":        "https://graph.facebook.com:8443/v21.0/" + fbTestPage + "/conversations?after=x",
		"host lookalike":  "https://graph.facebook.com.evil.example/v21.0/" + fbTestPage + "/conversations?after=x",
		"relative":        "/v21.0/" + fbTestPage + "/conversations?after=x",
		"unparseable":     "https://graph.facebook.com/%zz",
		"first url again": good + "?fields=id,link,updated_time,participants&limit=100&access_token=" + fbTestToken,
		"first url bare":  good + "?limit=100&fields=id,link,updated_time,participants",
		"fragment":        good + "?after=x#frag",
	}
	for name, next := range cases {
		g := &graphServer{pages: []func(int) (string, error){
			fbStatic(fbPageBody(rows, next)),
			fbStatic(fbPageBody(rows, "")), // must never be requested
		}}
		_, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
		if !errors.Is(err, ErrFacebookCoverageIncomplete) || g.count() != 1 {
			t.Errorf("%s: err %v after %d requests, want incomplete coverage after 1", name, err, g.count())
		}
		if h := g.hosts(); len(h) != 1 || !h["graph.facebook.com"] {
			t.Errorf("%s: contacted %v", name, h)
		}
		if err != nil && strings.Contains(err.Error(), fbTestToken) {
			t.Errorf("%s: error leaks the token", name)
		}
	}

	// A cursor that repeats an earlier one (the same query, with or without a token) is a cycle.
	g := &graphServer{pages: []func(int) (string, error){
		fbStatic(fbPageBody(rows, fbNextURL("p2"))),
		fbStatic(fbPageBody(rows, good+"?fields=id&after=p2")), // same query as page 1's next, without a token
		fbStatic(fbPageBody(rows, "")),                         // must never be requested
	}}
	_, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
	if !errors.Is(err, ErrFacebookCoverageIncomplete) || !strings.Contains(err.Error(), "repeats") {
		t.Fatalf("cycle not detected: %v (requests %d)", err, g.count())
	}
	// Token-only differences must not hide a repeat.
	g = &graphServer{pages: []func(int) (string, error){
		fbStatic(fbPageBody(rows, fbNextURL("p2"))),
		fbStatic(fbPageBody(rows, strings.Replace(fbNextURL("p2"), fbTestToken, "OTHER-TOKEN", 1))),
		fbStatic(fbPageBody(rows, "")),
	}}
	_, err = newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
	if !errors.Is(err, ErrFacebookCoverageIncomplete) || g.count() != 2 {
		t.Fatalf("token-only difference hid a repeated cursor: %v after %d requests", err, g.count())
	}
}

// A chain of fresh cursors that never ends stops at the page budget with an error.
func TestFacebookPageBudgetIsAnErrorNotATruncatedSuccess(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	g := &graphServer{}
	for i := 0; i < fbMaxConversationPages+10; i++ {
		i := i
		g.pages = append(g.pages, func(int) (string, error) {
			return fbPageBody(fbRows(fmt.Sprintf("n%d-", i), 0, 1, since.Add(time.Hour)), fbNextURL(fmt.Sprintf("c%d", i+1))), nil
		})
	}
	got, err := newFBTestAdapter(g).FetchRecentConversations(context.Background(), since, 0)
	if !errors.Is(err, ErrFacebookCoverageIncomplete) || !strings.Contains(err.Error(), "page budget") {
		t.Fatalf("endless pagination: err %v", err)
	}
	if g.count() != fbMaxConversationPages || len(got) != fbMaxConversationPages {
		t.Fatalf("%d requests / %d rows, want exactly the %d-page budget", g.count(), len(got), fbMaxConversationPages)
	}
}

func TestFacebookCancellationStopsWithAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	g := &graphServer{pages: []func(int) (string, error){
		fbStatic(fbPageBody(fbRows("c", 0, 3, since.Add(time.Hour)), fbNextURL("p2"))),
		fbStatic(fbPageBody(nil, "")),
	}}
	g.onCall = func(n int) {
		if n == 0 {
			cancel() // cancelled after the first page was requested
		}
	}
	got, err := newFBTestAdapter(g).FetchRecentConversations(ctx, since, 0)
	if err == nil || g.count() > 2 {
		t.Fatalf("cancelled enumeration returned success (%d rows, %d requests)", len(got), g.count())
	}
	pre, preCancel := context.WithCancel(context.Background())
	preCancel()
	g2 := &graphServer{pages: []func(int) (string, error){fbStatic(fbPageBody(nil, ""))}}
	if _, err := newFBTestAdapter(g2).FetchRecentConversations(pre, since, 0); err == nil || g2.count() != 0 {
		t.Fatalf("pre-cancelled context: err %v, %d requests", err, g2.count())
	}
}
