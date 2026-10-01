package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// CCMAI-RUNTIME-024 (F02-C): Zalo listrecentchat enumeration walks absolute row offsets to an
// explicit empty page, validates every physical row, dedupes customers by newest time and fails
// closed with token-free errors; token refresh keeps persistence-before-replacement. A synthetic
// RoundTripper stands in for openapi.zalo.me / oauth.zaloapp.com and records every request.

const (
	zlAccess  = "zl-ACCESS-R024"
	zlRefresh = "zl-REFRESH-R024"
	zlSecret  = "zl-APPSECRET-R024"
	zlNewAcc  = "zl-NEW-ACCESS-R024"
	zlNewRef  = "zl-NEW-REFRESH-R024"
	zlMarker  = "zl-BODY-MARKER-R024"
)

var zlBanned = []string{zlAccess, zlRefresh, zlSecret, zlNewAcc, zlNewRef, zlMarker, "openapi.zalo.me", "oauth.zaloapp.com", "access_token", "provider says"}

// captureZaloLog redirects the standard logger for one test and returns the captured text.
func captureZaloLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

func assertZaloLogSafe(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	for _, b := range zlBanned {
		if b != "openapi.zalo.me" && strings.Contains(buf.String(), b) {
			t.Fatalf("log leaks %q", b)
		}
	}
	if strings.Contains(buf.String(), "https://") {
		t.Fatalf("log contains a raw URL")
	}
}

func assertZaloSafe(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: want an error", label)
	}
	for _, b := range zlBanned {
		if strings.Contains(err.Error(), b) {
			t.Fatalf("%s: error text leaks %q: %v", label, b, err)
		}
	}
}

type zlRequest struct {
	method, host, path, token string
	offset, count             int
}

// zlServer serves listrecentchat pages by offset, optional refresh responses, and records requests.
type zlServer struct {
	mu        sync.Mutex
	requests  []zlRequest
	byOffset  map[int]func() (*http.Response, error)
	refresh   func() (*http.Response, error)
	afterPage func(offset int)
}

func zlResp(code int, body string) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

func (s *zlServer) RoundTrip(r *http.Request) (*http.Response, error) {
	req := zlRequest{method: r.Method, host: r.URL.Host, path: r.URL.Path, token: r.Header.Get("access_token"), offset: -1}
	if d := r.URL.Query().Get("data"); d != "" {
		var p struct{ Offset, Count int }
		_ = json.Unmarshal([]byte(d), &p)
		req.offset, req.count = p.Offset, p.Count
	}
	s.mu.Lock()
	s.requests = append(s.requests, req)
	s.mu.Unlock()
	var h func() (*http.Response, error)
	switch {
	case r.URL.Host == "oauth.zaloapp.com":
		h = s.refresh
	case r.URL.Host == "openapi.zalo.me" && r.URL.Path == "/v2.0/oa/listrecentchat":
		h = s.byOffset[req.offset]
	}
	if h == nil {
		return nil, fmt.Errorf("unscripted request %s %s offset %d", r.Method, r.URL.Path, req.offset)
	}
	resp, err := h()
	if resp != nil {
		resp.Request = r
	}
	if s.afterPage != nil && r.URL.Host == "openapi.zalo.me" {
		s.afterPage(req.offset)
	}
	return resp, err
}

func (s *zlServer) reqs() []zlRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]zlRequest(nil), s.requests...)
}

func newZLAdapter(s *zlServer) *ZaloOAAdapter {
	a := NewZaloOAAdapter(ZaloOACredentials{AppID: "app-r024", AppSecret: zlSecret, AccessToken: zlAccess, RefreshToken: zlRefresh, OAId: "oa-r024"})
	a.client = &http.Client{Transport: s}
	return a
}

type zlRow struct {
	src    int
	userID string
	name   string
	millis int64
}

func (r zlRow) json() string {
	if r.src == 0 { // OA sent last: customer is "to"
		return fmt.Sprintf(`{"src":0,"from_id":"oa-r024","from_display_name":"OA","to_id":%q,"to_display_name":%q,"time":%d,"message":"x"}`, r.userID, r.name, r.millis)
	}
	return fmt.Sprintf(`{"src":1,"from_id":%q,"from_display_name":%q,"to_id":"oa-r024","to_display_name":"OA","time":%d,"message":"x"}`, r.userID, r.name, r.millis)
}

func zlPageBody(rows []zlRow, nested bool) string {
	parts := make([]string, len(rows))
	for i, r := range rows {
		parts[i] = r.json()
	}
	arr := "[" + strings.Join(parts, ",") + "]"
	if nested {
		return `{"error":0,"message":"Success","data":{"total":999,"data":` + arr + `}}`
	}
	return `{"error":0,"message":"Success","data":` + arr + `}`
}

// zlScript serves pages back to back: page i is requested at the offset equal to the number of
// physical rows of pages before it. A final empty page is appended.
func zlScript(pages [][]zlRow) *zlServer {
	s := &zlServer{byOffset: map[int]func() (*http.Response, error){}}
	off := 0
	for i, rows := range pages {
		s.byOffset[off] = zlResp(200, zlPageBody(rows, i%2 == 1))
		off += len(rows)
	}
	s.byOffset[off] = zlResp(200, `{"error":0,"message":"Success","data":[]}`)
	return s
}

func zlMs(t time.Time) int64 { return t.UnixMilli() }

func TestZaloEnumeratesTheWholeWindowToAnEmptyPage(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	newer := since.Add(48 * time.Hour)
	bigA, bigB := "9007199254740993123", "18446744073709551617" // beyond float64/uint64 precision
	var all []zlRow
	want := map[string]int64{}
	for i := 0; i < 131; i++ {
		id := fmt.Sprintf("u%05d", i)
		src := i % 2
		all = append(all, zlRow{src: src, userID: id, name: "K" + id, millis: zlMs(newer) + int64(i)})
		want[id] = zlMs(newer) + int64(i)
	}
	all = append(all,
		zlRow{1, bigA, "big A", zlMs(newer)},
		zlRow{0, bigB, "big B", zlMs(newer)},
		zlRow{1, "old-1", "old", zlMs(since) - 1},             // older than since: skipped, not a stop signal
		zlRow{1, "boundary", "edge", zlMs(since)},             // exactly since: included
		zlRow{1, "u00003", "dup newer", zlMs(newer) + 10_000}, // duplicate, newer: replaces
		zlRow{0, "u00004", "dup older", zlMs(newer) - 10_000}, // duplicate, older: ignored
		zlRow{1, "u00005", "dup tie", want["u00005"]},         // duplicate, same time: first kept
	)
	want[bigA], want[bigB], want["boundary"] = zlMs(newer), zlMs(newer), zlMs(since)
	want["u00003"] = zlMs(newer) + 10_000
	// Pages of 10 except one short nonterminal page of 7 in the middle.
	var pages [][]zlRow
	for i := 0; i < len(all); {
		n := 10
		if len(pages) == 5 {
			n = 7
		}
		if i+n > len(all) {
			n = len(all) - i
		}
		pages = append(pages, all[i:i+n])
		i += n
	}
	s := zlScript(pages)
	got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0)
	if err != nil {
		t.Fatalf("enumeration failed: %v", err)
	}
	if len(want) != 134 {
		t.Fatalf("fixture drifted: %d eligible", len(want))
	}
	gotIDs := map[string]SyncedConversation{}
	for _, c := range got {
		if _, dup := gotIDs[c.ExternalID]; dup {
			t.Fatalf("customer %s returned twice", c.ExternalID)
		}
		gotIDs[c.ExternalID] = c
	}
	if len(gotIDs) != len(want) {
		t.Fatalf("got %d customers, want %d", len(gotIDs), len(want))
	}
	for id, ms := range want {
		c, ok := gotIDs[id]
		if !ok || c.LastMessageAt.UnixMilli() != ms || c.ExternalUserID != id {
			t.Fatalf("customer %s: %+v, want time %d", id, c, ms)
		}
	}
	if gotIDs["u00003"].CustomerName != "dup newer" || gotIDs["u00005"].CustomerName != "Ku00005" || gotIDs["u00004"].CustomerName != "Ku00004" {
		t.Fatalf("duplicate resolution wrong: %q %q %q", gotIDs["u00003"].CustomerName, gotIDs["u00005"].CustomerName, gotIDs["u00004"].CustomerName)
	}
	if gotIDs[bigB].CustomerName != "big B" || gotIDs["u00001"].CustomerName != "Ku00001" || gotIDs["u00000"].CustomerName != "Ku00000" {
		t.Fatal("src direction mapping wrong")
	}
	// Large time keeps exact metadata; IDs are exact strings.
	if b, _ := json.Marshal(gotIDs[bigA].Metadata); !strings.Contains(string(b), bigA) {
		t.Fatalf("metadata lost the exact id: %s", b)
	}
	// Offsets follow physical row counts (the short page included); count is always 10.
	var offsets []int
	for _, r := range s.reqs() {
		if r.host != "openapi.zalo.me" || r.method != http.MethodGet || r.count != 10 || r.token != zlAccess {
			t.Fatalf("unexpected request %+v", r)
		}
		offsets = append(offsets, r.offset)
	}
	wantOff, off := []int{}, 0
	for _, p := range pages {
		wantOff = append(wantOff, off)
		off += len(p)
	}
	wantOff = append(wantOff, off)
	if fmt.Sprint(offsets) != fmt.Sprint(wantOff) {
		t.Fatalf("offsets %v, want %v", offsets, wantOff)
	}
}

func TestZaloShortPageIsNotTerminal(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	s := zlScript([][]zlRow{{{1, "a", "A", zlMs(since) + 1}}, {{1, "b", "B", zlMs(since) + 1}}})
	got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0)
	if err != nil || len(got) != 2 || len(s.reqs()) != 3 {
		t.Fatalf("short pages: %d rows, %d requests, err %v", len(got), len(s.reqs()), err)
	}
}

func TestZaloLimitThatWouldTruncateIsAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	var rows []zlRow
	for i := 0; i < 25; i++ {
		rows = append(rows, zlRow{1, fmt.Sprintf("l%02d", i), "L", zlMs(since) + 1})
	}
	s := zlScript([][]zlRow{rows[:10], rows[10:20], rows[20:]})
	got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 15)
	if !errors.Is(err, ErrZaloCoverageIncomplete) || len(got) != 15 {
		t.Fatalf("limit 15 over 25: %d rows, err %v", len(got), err)
	}
	s = zlScript([][]zlRow{rows[:10], rows[10:20], rows[20:]})
	if got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 25); err != nil || len(got) != 25 {
		t.Fatalf("limit equal to the window: %d rows, err %v", len(got), err)
	}
}

// Every malformed envelope, array or row fails the run, including rows that would be filtered.
func TestZaloMalformedPagesFailClosed(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ok := zlRow{1, "ok", "OK", zlMs(since) + 1}.json()
	old := zlMs(since) - 100
	cases := map[string]string{
		"not an object":         `[1,2]`,
		"null body":             `null`,
		"missing error code":    `{"data":[]}`,
		"string error code":     `{"error":"0","data":[]}`,
		"fractional error code": `{"error":0.5,"data":[]}`,
		"missing data":          `{"error":0}`,
		"null data":             `{"error":0,"data":null}`,
		"string data":           `{"error":0,"data":"[]"}`,
		"nested without array":  `{"error":0,"data":{"total":3}}`,
		"nested null array":     `{"error":0,"data":{"data":null}}`,
		"row not an object":     `{"error":0,"data":["x"]}`,
		"null row":              `{"error":0,"data":[null]}`,
		"src missing":           `{"error":0,"data":[{"from_id":"a","time":1790000000000}]}`,
		"src 2":                 `{"error":0,"data":[{"src":2,"from_id":"a","time":1790000000000}]}`,
		"src fractional":        `{"error":0,"data":[{"src":1.0,"from_id":"a","time":1790000000000}]}`,
		"src string":            `{"error":0,"data":[{"src":"1","from_id":"a","time":1790000000000}]}`,
		"customer id missing":   `{"error":0,"data":[{"src":0,"from_id":"oa","time":1790000000000}]}`,
		"customer id empty":     `{"error":0,"data":[{"src":1,"from_id":"","time":1790000000000}]}`,
		"customer id numeric":   `{"error":0,"data":[{"src":1,"from_id":123456789012345678,"time":1790000000000}]}`,
		"time missing":          `{"error":0,"data":[{"src":1,"from_id":"a"}]}`,
		"time string":           `{"error":0,"data":[{"src":1,"from_id":"a","time":"1790000000000"}]}`,
		"time fractional":       `{"error":0,"data":[{"src":1,"from_id":"a","time":1790000000000.5}]}`,
		"time zero":             `{"error":0,"data":[{"src":1,"from_id":"a","time":0}]}`,
		"time negative":         `{"error":0,"data":[{"src":1,"from_id":"a","time":-5}]}`,
		"time overflow":         `{"error":0,"data":[{"src":1,"from_id":"a","time":99999999999999999999}]}`,
		"time beyond year 9999": `{"error":0,"data":[{"src":1,"from_id":"a","time":253402300800000}]}`,
		"bad row that is old":   fmt.Sprintf(`{"error":0,"data":[{"src":7,"from_id":"a","time":%d}]}`, old),
		"good row then broken":  `{"error":0,"data":[` + ok + `,{"src":1,"time":1790000000000}]}`,
		"trailing garbage":      `{"error":0,"data":[]} x`,
	}
	for name, body := range cases {
		// Every later offset is a valid empty page, so a row that is wrongly accepted ends in a
		// success instead of an unrelated error on page 2.
		s := &zlServer{byOffset: map[int]func() (*http.Response, error){0: zlResp(200, body)}}
		for off := 1; off <= 3; off++ {
			s.byOffset[off] = zlResp(200, `{"error":0,"data":[]}`)
		}
		_, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0)
		if !errors.Is(err, ErrZaloCoverageIncomplete) {
			t.Errorf("%s: err = %v, want incomplete coverage", name, err)
			continue
		}
		if n := len(s.reqs()); n != 1 {
			t.Errorf("%s: failed after %d requests, want on page 1", name, n)
		}
		assertZaloSafe(t, name, err)
	}
	// Positive control for the same harness: a valid row then the empty page succeeds.
	s := &zlServer{byOffset: map[int]func() (*http.Response, error){0: zlResp(200, `{"error":0,"data":[`+ok+`]}`), 1: zlResp(200, `{"error":0,"data":[]}`)}}
	if got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0); err != nil || len(got) != 1 {
		t.Fatalf("positive control: %d rows, err %v", len(got), err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed " + zlAccess) }

// Later-page request failures are incomplete coverage with the 10 rows of page 1 kept for
// diagnostics, and no token, URL, body or provider text in the error. Positive control included.
func TestZaloLaterPageFailuresReturnSafeErrors(t *testing.T) {
	logs := captureZaloLog(t)
	defer func() { assertZaloLogSafe(t, logs) }()
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	var page1 []zlRow
	for i := 0; i < 10; i++ {
		page1 = append(page1, zlRow{1, fmt.Sprintf("p%02d", i), "P", zlMs(since) + 1})
	}
	failures := map[string]func() (*http.Response, error){
		"http 500":    zlResp(500, zlMarker),
		"http 429":    zlResp(429, `{"error":0,"data":[]}`),
		"undecodable": zlResp(200, "<html>"+zlMarker+" "+zlAccess+"</html>"),
		"api error":   zlResp(200, `{"error":-201,"message":"provider says `+zlAccess+` is wrong"}`),
		"oversized":   zlResp(200, `{"error":0,"data":[],"pad":"`+strings.Repeat("x", zaloMaxResponseBytes)+`"}`),
		"transport": func() (*http.Response, error) {
			return nil, &url.Error{Op: "Get", URL: "https://openapi.zalo.me/v2.0/oa/listrecentchat?access_token=" + zlAccess, Err: errors.New("reset " + zlSecret)}
		},
		"read failure": func() (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(errReader{})}, nil
		},
	}
	for name, page2 := range failures {
		s := &zlServer{byOffset: map[int]func() (*http.Response, error){0: zlResp(200, zlPageBody(page1, false)), 10: page2}}
		got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0)
		if !errors.Is(err, ErrZaloCoverageIncomplete) {
			t.Fatalf("%s: err %v, want incomplete coverage", name, err)
		}
		assertZaloSafe(t, name, err)
		if len(got) != 10 || len(s.reqs()) != 2 {
			t.Fatalf("%s: %d diagnostic rows, %d requests", name, len(got), len(s.reqs()))
		}
		if name == "oversized" && !strings.Contains(err.Error(), "8 MiB") {
			t.Fatalf("oversized body was not rejected by the size bound: %v", err)
		}
	}
	s := &zlServer{byOffset: map[int]func() (*http.Response, error){0: zlResp(200, zlPageBody(page1, false)), 10: zlResp(200, `{"error":0,"data":[]}`)}}
	if got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0); err != nil || len(got) != 10 {
		t.Fatalf("positive control: %d rows, err %v", len(got), err)
	}
}

func TestZaloRepeatedPageFailsClosed(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	page := []zlRow{{1, "r1", "R", zlMs(since) + 1}, {1, "r2", "R", zlMs(since) + 2}}
	// The provider ignores the offset and returns the same rows again.
	s := &zlServer{byOffset: map[int]func() (*http.Response, error){
		0: zlResp(200, zlPageBody(page, false)),
		2: zlResp(200, zlPageBody(page, true)), // same rows, other envelope form
		4: zlResp(200, `{"error":0,"data":[]}`),
	}}
	_, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0)
	if !errors.Is(err, ErrZaloCoverageIncomplete) || !strings.Contains(err.Error(), "repeats") || len(s.reqs()) != 2 {
		t.Fatalf("repeated page: err %v after %d requests", err, len(s.reqs()))
	}
	// Same customer with a new time on the next page is not a repeat.
	s = &zlServer{byOffset: map[int]func() (*http.Response, error){
		0: zlResp(200, zlPageBody(page, false)),
		2: zlResp(200, zlPageBody([]zlRow{{1, "r1", "R", zlMs(since) + 9}, {1, "r2", "R", zlMs(since) + 2}}, false)),
		4: zlResp(200, `{"error":0,"data":[]}`),
	}}
	if got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0); err != nil || len(got) != 2 || got[0].LastMessageAt.UnixMilli() != zlMs(since)+9 {
		t.Fatalf("moved row: %v, %v", got, err)
	}
}

func TestZaloPageBudgetIsAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	s := &zlServer{byOffset: map[int]func() (*http.Response, error){}}
	for i := 0; i < zaloMaxConversationPages+5; i++ {
		s.byOffset[i] = zlResp(200, zlPageBody([]zlRow{{1, fmt.Sprintf("n%04d", i), "N", zlMs(since) + 1}}, false))
	}
	got, err := newZLAdapter(s).FetchRecentConversations(context.Background(), since, 0)
	if !errors.Is(err, ErrZaloCoverageIncomplete) || !strings.Contains(err.Error(), "page budget") {
		t.Fatalf("endless pages: %v", err)
	}
	if len(s.reqs()) != zaloMaxConversationPages || len(got) != zaloMaxConversationPages {
		t.Fatalf("%d requests / %d rows, want the %d-page budget", len(s.reqs()), len(got), zaloMaxConversationPages)
	}
	// A window that ends exactly on the last budgeted page (499 rows + empty page) succeeds.
	s2 := &zlServer{byOffset: map[int]func() (*http.Response, error){}}
	for i := 0; i < zaloMaxConversationPages-1; i++ {
		s2.byOffset[i] = zlResp(200, zlPageBody([]zlRow{{1, fmt.Sprintf("m%04d", i), "M", zlMs(since) + 1}}, false))
	}
	s2.byOffset[zaloMaxConversationPages-1] = zlResp(200, `{"error":0,"data":[]}`)
	if got, err := newZLAdapter(s2).FetchRecentConversations(context.Background(), since, 0); err != nil || len(got) != zaloMaxConversationPages-1 {
		t.Fatalf("terminal page on the budget edge: %d rows, err %v", len(got), err)
	}
}

func TestZaloCancellationAtTheTerminalPageIsAnError(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	s := zlScript([][]zlRow{{{1, "c1", "C", zlMs(since) + 1}}})
	s.afterPage = func(offset int) {
		if offset == 1 { // the empty terminal page was just served
			cancel()
		}
	}
	got, err := newZLAdapter(s).FetchRecentConversations(ctx, since, 0)
	if !errors.Is(err, ErrZaloCoverageIncomplete) || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled at the terminal page: %d rows, err %v", len(got), err)
	}
	assertZaloSafe(t, "cancelled", err)
}

// -216 on page 2: one refresh, the persisted pair replaces the old one, and the same offset is
// retried with the new token.
func TestZaloExpiredTokenRefreshesPersistsAndRetriesTheSameOffset(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	var page1, page2 []zlRow
	for i := 0; i < 10; i++ {
		page1 = append(page1, zlRow{1, fmt.Sprintf("a%02d", i), "A", zlMs(since) + 1})
	}
	page2 = []zlRow{{0, "b00", "B", zlMs(since) + 1}}
	expiredOnce := false
	s := &zlServer{byOffset: map[int]func() (*http.Response, error){
		0: zlResp(200, zlPageBody(page1, false)),
		10: func() (*http.Response, error) {
			if !expiredOnce {
				expiredOnce = true
				return zlResp(200, `{"error":-216,"message":"Access token is invalid"}`)()
			}
			return zlResp(200, zlPageBody(page2, true))()
		},
		11: zlResp(200, `{"error":0,"data":[]}`),
	}, refresh: zlResp(200, `{"access_token":"`+zlNewAcc+`","refresh_token":"`+zlNewRef+`","expires_in":"90000"}`)}
	a := newZLAdapter(s)
	var persisted []string
	a.SetTokenRefreshCallback(func(acc, ref string) error {
		if a.creds.AccessToken != zlAccess {
			t.Fatal("in-memory token replaced before persistence")
		}
		persisted = append(persisted, acc+"|"+ref)
		return nil
	})
	got, err := a.FetchRecentConversations(context.Background(), since, 0)
	if err != nil || len(got) != 11 {
		t.Fatalf("refresh path: %d rows, err %v", len(got), err)
	}
	if len(persisted) != 1 || persisted[0] != zlNewAcc+"|"+zlNewRef || a.creds.AccessToken != zlNewAcc || a.creds.RefreshToken != zlNewRef {
		t.Fatalf("persist/replace order wrong: %v, creds %s/%s", persisted, a.creds.AccessToken, a.creds.RefreshToken)
	}
	var seq []string
	for _, r := range s.reqs() {
		seq = append(seq, fmt.Sprintf("%s:%d:%t", r.host, r.offset, r.token == zlNewAcc))
	}
	want := "openapi.zalo.me:0:false openapi.zalo.me:10:false oauth.zaloapp.com:-1:false openapi.zalo.me:10:true openapi.zalo.me:11:true"
	if strings.Join(seq, " ") != want {
		t.Fatalf("request sequence\n got %s\nwant %s", strings.Join(seq, " "), want)
	}
}

var errLostOwnership = errors.New("synthetic ownership lost")

// Every refresh failure stops before the retry, keeps the old pair and hides secrets.
func TestZaloRefreshFailuresStopAndKeepOldTokens(t *testing.T) {
	logs := captureZaloLog(t)
	defer func() { assertZaloLogSafe(t, logs) }()
	persistSecret := errors.New("persist failed with " + zlNewAcc)
	cases := map[string]struct {
		refresh  func() (*http.Response, error)
		callback error
		wantIs   error
		callsCB  bool
	}{
		"refresh http 500":       {refresh: zlResp(500, zlMarker)},
		"refresh undecodable":    {refresh: zlResp(200, "<html>"+zlMarker+"</html>")},
		"refresh provider error": {refresh: zlResp(200, `{"error":-14014,"message":"provider says `+zlRefresh+` invalid"}`)},
		"empty access token":     {refresh: zlResp(200, `{"access_token":"","refresh_token":"`+zlNewRef+`"}`)},
		"missing refresh token":  {refresh: zlResp(200, `{"access_token":"`+zlNewAcc+`"}`)},
		"refresh transport": {refresh: func() (*http.Response, error) {
			return nil, errors.New("dial failed for secret_key=" + zlSecret + " refresh_token=" + zlRefresh)
		}},
		"persistence failure": {refresh: zlResp(200, `{"access_token":"`+zlNewAcc+`","refresh_token":"`+zlNewRef+`"}`), callback: persistSecret, wantIs: persistSecret, callsCB: true},
		"ownership lost":      {refresh: zlResp(200, `{"access_token":"`+zlNewAcc+`","refresh_token":"`+zlNewRef+`"}`), callback: fmt.Errorf("wrapped: %w", errLostOwnership), wantIs: errLostOwnership, callsCB: true},
	}
	for name, c := range cases {
		s := &zlServer{byOffset: map[int]func() (*http.Response, error){0: zlResp(200, `{"error":-216}`)}, refresh: c.refresh}
		a := newZLAdapter(s)
		called := false
		a.SetTokenRefreshCallback(func(string, string) error { called = true; return c.callback })
		_, err := a.FetchRecentConversations(context.Background(), time.Time{}, 0)
		if !errors.Is(err, ErrZaloCoverageIncomplete) {
			t.Fatalf("%s: err %v", name, err)
		}
		assertZaloSafe(t, name, err)
		if c.wantIs != nil && !errors.Is(err, c.wantIs) {
			t.Fatalf("%s: errors.Is lost the callback cause: %v", name, err)
		}
		if called != c.callsCB {
			t.Fatalf("%s: callback called = %v, want %v", name, called, c.callsCB)
		}
		if a.creds.AccessToken != zlAccess || a.creds.RefreshToken != zlRefresh {
			t.Fatalf("%s: old tokens replaced", name)
		}
		if n := len(s.reqs()); n != 2 {
			t.Fatalf("%s: %d requests, want GET + refresh POST and no retry", name, n)
		}
	}
}

// A second -216 after a successful refresh is an error: at most one refresh per request.
func TestZaloSecondExpiryAfterRefreshIsAnError(t *testing.T) {
	s := &zlServer{byOffset: map[int]func() (*http.Response, error){0: zlResp(200, `{"error":-216}`)},
		refresh: zlResp(200, `{"access_token":"`+zlNewAcc+`","refresh_token":"`+zlNewRef+`"}`)}
	a := newZLAdapter(s)
	a.SetTokenRefreshCallback(func(string, string) error { return nil })
	_, err := a.FetchRecentConversations(context.Background(), time.Time{}, 0)
	if !errors.Is(err, ErrZaloCoverageIncomplete) || len(s.reqs()) != 3 {
		t.Fatalf("second expiry: err %v after %d requests", err, len(s.reqs()))
	}
	assertZaloSafe(t, "second expiry", err)
}

// FetchMessages keeps its mapping (float decoding, src, attachments) through the hardened helper.
func TestZaloFetchMessagesMappingUnchanged(t *testing.T) {
	body := `{"error":0,"data":[` +
		`{"message_id":"mid-1","src":1,"time":1790000000000,"type":"text","message":"xin chao","from_display_name":"Khach"},` +
		`{"message_id":"mid-2","src":0,"time":1790000001000,"type":"photo","message":"","url":"https://example.invalid/p.jpg"}]}`
	srv := &convTransport{body: body}
	a := NewZaloOAAdapter(ZaloOACredentials{AccessToken: zlAccess})
	a.client = &http.Client{Transport: srv}
	msgs, err := a.FetchMessages(context.Background(), "u1", time.Time{})
	if err != nil || len(msgs) != 2 {
		t.Fatalf("messages: %d, %v", len(msgs), err)
	}
	if msgs[0].SenderType != "customer" || msgs[0].SenderName != "Khach" || msgs[0].Content != "xin chao" || msgs[0].SentAt.UnixMilli() != 1790000000000 {
		t.Fatalf("customer message mapped wrong: %+v", msgs[0])
	}
	if msgs[1].SenderType != "agent" || msgs[1].ContentType != "photo" || len(msgs[1].Attachments) != 1 {
		t.Fatalf("agent attachment message mapped wrong: %+v", msgs[1])
	}
	if srv.calls != 1 {
		t.Fatalf("a short message page should end paging, got %d calls", srv.calls)
	}
}

type convTransport struct {
	body  string
	calls int
}

func (c *convTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.calls++
	if r.URL.Path != "/v2.0/oa/conversation" {
		return nil, errors.New("unexpected path")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(c.body)), Request: r}, nil
}
