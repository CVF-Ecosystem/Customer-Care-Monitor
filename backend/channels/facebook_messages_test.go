package channels

import (
	"context"
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

// CCMAI-RUNTIME-031 (F02-E): Facebook message traversal. A closed RoundTripper stands in for
// graph.facebook.com and records every request it receives (URL, method, host, token); no real
// Facebook endpoint or token is used. The tests assert the local safety contract (valid rows and
// paging, safe next links, blocked redirects, bounded budget, deterministic order), never live
// Graph behaviour.

const (
	fbmConv          = "t_100"
	fbmProviderToken = "PROVIDER-LINK-TOKEN-R031"
	fbmMarker        = "R031-BODY-MARKER"
	fbmFields        = "id,message,from,to,created_time,attachments,shares,sticker"
)

var fbmSince = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func fbmRow(id string, at time.Time) string {
	return fmt.Sprintf(`{"id":%q,"message":"tin %s","from":{"id":"u-1","name":"Khach"},"created_time":%q}`,
		id, id, at.UTC().Format(fbTestStamp))
}

// fbmNext builds a provider-style next link (it carries a provider token that must never be sent).
func fbmNext(conv, cursor string) string {
	return fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages?access_token=%s&after=%s&fields=%s&limit=100", conv, fbmProviderToken, cursor, fbmFields)
}

func fbmPage(next string, rows ...string) string {
	body := `{"data":[` + strings.Join(rows, ",") + `]`
	if next != "" {
		body += fmt.Sprintf(`,"paging":{"next":%q}`, next)
	}
	return body + `}`
}

// fbmTransport records requests and counts body opens/closes; handle decides each response.
type fbmTransport struct {
	mu     sync.Mutex
	reqs   []*url.URL
	meths  []string
	opened int
	closed int
	handle func(n int, r *http.Request) (*http.Response, error)
}

type fbmBody struct {
	io.Reader
	t *fbmTransport
}

func (b *fbmBody) Close() error {
	b.t.mu.Lock()
	b.t.closed++
	b.t.mu.Unlock()
	return nil
}

func (t *fbmTransport) respond(r *http.Request, code int, body string, hdr http.Header) (*http.Response, error) {
	t.mu.Lock()
	t.opened++
	t.mu.Unlock()
	if hdr == nil {
		hdr = http.Header{}
	}
	return &http.Response{StatusCode: code, Header: hdr, Body: &fbmBody{Reader: strings.NewReader(body), t: t}, Request: r}, nil
}

func (t *fbmTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mu.Lock()
	n := len(t.reqs)
	u := *r.URL
	t.reqs = append(t.reqs, &u)
	t.meths = append(t.meths, r.Method)
	t.mu.Unlock()
	return t.handle(n, r)
}

func (t *fbmTransport) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.reqs)
}

func (t *fbmTransport) countHost(host string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := 0
	for _, u := range t.reqs {
		if u.Host == host {
			n++
		}
	}
	return n
}

// fbmScript serves the given 200 bodies in request order; anything further is a transport error.
func fbmScript(pages ...string) *fbmTransport {
	t := &fbmTransport{}
	t.handle = func(n int, r *http.Request) (*http.Response, error) {
		if n >= len(pages) {
			return nil, fmt.Errorf("unscripted request %d to %s", n, r.URL.Host)
		}
		return t.respond(r, 200, pages[n], nil)
	}
	return t
}

func fbmAdapter(t *fbmTransport) *FacebookAdapter {
	a := NewFacebookAdapter(FacebookCredentials{PageID: fbTestPage, AccessToken: fbTestToken})
	a.client = &http.Client{Transport: t}
	return a
}

func fbmFetch(tr *fbmTransport, since time.Time) ([]SyncedMessage, error) {
	return fbmAdapter(tr).FetchMessages(context.Background(), fbmConv, since)
}

func fbmIDs(msgs []SyncedMessage) string {
	var ids []string
	for _, m := range msgs {
		ids = append(ids, m.ExternalID)
	}
	return strings.Join(ids, ",")
}

func fbmAssertSafe(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: want an error", label)
	}
	for _, banned := range []string{fbTestToken, fbmProviderToken, "access_token", fbmMarker, "evil.example", fbmConv, "graph.facebook.com"} {
		if strings.Contains(err.Error(), banned) {
			t.Fatalf("%s: error leaks %q: %v", label, banned, err)
		}
	}
}

func fbmAssertBodiesClosed(t *testing.T, label string, tr *fbmTransport) {
	t.Helper()
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.opened != tr.closed {
		t.Errorf("%s: %d response bodies opened, %d closed", label, tr.opened, tr.closed)
	}
}

// F02E-01: initial request inventory, escaped conversation ID, configured token only, provider
// link token replaced, non-auth paging parameters preserved.
func TestFacebookMessageRequestInventory(t *testing.T) {
	const conv = "t_1/x y"
	first := fbmPage(fbmNext("t_1%2Fx%20y", "c1")+"&__paging_token=enc_xyz", fbmRow("a", fbmSince.Add(time.Hour)))
	tr := fbmScript(first, fbmPage("", fbmRow("b", fbmSince.Add(2*time.Hour))))
	got, err := fbmAdapter(tr).FetchMessages(context.Background(), conv, fbmSince)
	if err != nil {
		t.Fatalf("FetchMessages: %v", err)
	}
	if fbmIDs(got) != "a,b" || tr.count() != 2 {
		t.Fatalf("IDs %s over %d requests", fbmIDs(got), tr.count())
	}
	for i, u := range tr.reqs {
		if tr.meths[i] != http.MethodGet || u.Scheme != "https" || u.Host != "graph.facebook.com" || u.EscapedPath() != "/v21.0/t_1%2Fx%20y/messages" {
			t.Fatalf("request %d = %s %s", i, tr.meths[i], u)
		}
		q := u.Query()
		if toks := q["access_token"]; len(toks) != 1 || toks[0] != fbTestToken {
			t.Fatalf("request %d token = %v, want the configured token exactly once", i, toks)
		}
		if strings.Contains(u.String(), fbmProviderToken) {
			t.Fatalf("request %d transmitted the provider link token: %s", i, u)
		}
	}
	q0, q1 := tr.reqs[0].Query(), tr.reqs[1].Query()
	if q0.Get("fields") != fbmFields || q0.Get("limit") != "100" || q0.Has("after") {
		t.Fatalf("initial query = %v", q0)
	}
	if q1.Get("after") != "c1" || q1.Get("__paging_token") != "enc_xyz" || q1.Get("fields") != fbmFields {
		t.Fatalf("next query lost non-auth paging parameters: %v", q1)
	}
	fbmAssertBodiesClosed(t, "inventory", tr)
}

func TestFacebookBlankConversationIDSendsNothing(t *testing.T) {
	for _, id := range []string{"", "   "} {
		tr := fbmScript(fbmPage(""))
		_, err := fbmAdapter(tr).FetchMessages(context.Background(), id, fbmSince)
		if !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || tr.count() != 0 {
			t.Fatalf("blank conversation %q: err %v after %d requests", id, err, tr.count())
		}
	}
}

// Every unsafe next link is rejected BEFORE any further request; the host is never contacted and
// the token never leaves for it.
func TestFacebookUnsafeNextLinksAreRejectedBeforeAnyRequest(t *testing.T) {
	good := "/v21.0/" + fbmConv + "/messages"
	unsafe := map[string]string{
		"other host":                 "https://evil.example" + good + "?after=x",
		"host suffix trick":          "https://graph.facebook.com.evil.example" + good + "?after=x",
		"userinfo host trick":        "https://graph.facebook.com@evil.example" + good + "?after=x",
		"userinfo on graph host":     "https://user:pw@graph.facebook.com" + good + "?after=x",
		"http scheme":                "http://graph.facebook.com" + good + "?after=x",
		"ftp scheme":                 "ftp://graph.facebook.com" + good + "?after=x",
		"protocol relative":          "//graph.facebook.com" + good + "?after=x",
		"relative path":              good + "?after=x",
		"other port":                 "https://graph.facebook.com:444" + good + "?after=x",
		"other conversation":         "https://graph.facebook.com/v21.0/t_999/messages?after=x",
		"conversation list endpoint": "https://graph.facebook.com/v21.0/" + fbTestPage + "/conversations?after=x",
		"other version":              "https://graph.facebook.com/v22.0/" + fbmConv + "/messages?after=x",
		"no version":                 "https://graph.facebook.com/" + fbmConv + "/messages?after=x",
		"path traversal":             "https://graph.facebook.com/v21.0/" + fbmConv + "/../t_999/messages?after=x",
		"encoded traversal":          "https://graph.facebook.com/v21.0/" + fbmConv + "/%2e%2e/t_999/messages?after=x",
		"trailing segment":           "https://graph.facebook.com" + good + "/extra?after=x",
		"fragment":                   "https://graph.facebook.com" + good + "?after=x#frag",
		"malformed query":            "https://graph.facebook.com" + good + "?after=%zz",
		"malformed url":              "https://graph.facebook.com/%zz",
		"uppercase host":             "https://GRAPH.facebook.com" + good + "?after=x",
		"opaque":                     "https:graph.facebook.com" + good,
	}
	for name, link := range unsafe {
		tr := fbmScript(fbmPage(link, fbmRow("a", fbmSince.Add(time.Hour))), fbmPage("", fbmRow("zz", fbmSince.Add(time.Hour))))
		got, err := fbmFetch(tr, fbmSince)
		fbmAssertSafe(t, name, err)
		if !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || !strings.Contains(err.Error(), "unsafe next link") {
			t.Errorf("%s: err = %v, want the unsafe-link class", name, err)
		}
		if tr.count() != 1 || tr.countHost("graph.facebook.com") != 1 {
			t.Errorf("%s: %d requests; an unsafe link must not be followed", name, tr.count())
		}
		if len(got) != 1 {
			t.Errorf("%s: diagnostic rows = %d, want 1", name, len(got))
		}
	}
	// Positive controls: an explicit :443 and an encoded-but-equal path are the same endpoint.
	for name, link := range map[string]string{
		"default port":   "https://graph.facebook.com:443" + good + "?after=x",
		"encoded letter": "https://graph.facebook.com/v21.0/t%5F100/messages?after=x",
	} {
		tr := fbmScript(fbmPage(link, fbmRow("a", fbmSince.Add(time.Hour))), fbmPage("", fbmRow("b", fbmSince.Add(2*time.Hour))))
		got, err := fbmFetch(tr, fbmSince)
		if err != nil || fbmIDs(got) != "a,b" {
			t.Errorf("%s: %s, err %v", name, fbmIDs(got), err)
		}
	}
}

// F02E-02: malformed pages and rows fail; old/duplicate invalid rows beside fresh valid rows prove
// validation precedes dedupe and filtering.
func TestFacebookMessageMalformedPagesFailClosed(t *testing.T) {
	good := fbmRow("ok", fbmSince.Add(time.Hour))
	cases := map[string]struct {
		body     string
		coverage bool
	}{
		"missing data":          {`{}`, true},
		"null data":             {`{"data":null}`, true},
		"object data":           {`{"data":{"id":"x"}}`, true},
		"string data":           {`{"data":"[]"}`, true},
		"top-level null":        {`null`, true},
		"row is a string":       {`{"data":["x"]}`, true},
		"row is null":           {`{"data":[null]}`, true},
		"row is an array":       {`{"data":[[]]}`, true},
		"good row then null":    {`{"data":[` + good + `,null]}`, true},
		"malformed json":        {`{"data":[` + good + `,]}`, false},
		"top-level array":       {`[]`, false},
		"undecodable":           {`<html>` + fbmMarker + `</html>`, false},
		"graph error object":    {`{"error":{"message":"bad ` + fbTestToken + ` ` + fbmMarker + `","code":190}}`, false},
		"graph error with data": {`{"data":[` + good + `],"error":{"code":4}}`, false},
	}
	for name, c := range cases {
		tr := fbmScript(c.body)
		_, err := fbmFetch(tr, fbmSince)
		fbmAssertSafe(t, name, err)
		if c.coverage && !errors.Is(err, ErrFacebookMessageCoverageIncomplete) {
			t.Errorf("%s: err = %v, want message coverage incomplete", name, err)
		}
		if errors.Is(err, ErrFacebookCoverageIncomplete) {
			t.Errorf("%s: message failure reused the conversation error identity", name)
		}
		if tr.count() != 1 {
			t.Errorf("%s: %d requests, want 1", name, tr.count())
		}
		fbmAssertBodiesClosed(t, name, tr)
	}
}

func TestFacebookMessageInvalidRowsAreNeverFilteredAway(t *testing.T) {
	old := fbmSince.Add(-48 * time.Hour).UTC().Format(fbTestStamp)
	fresh := fbmSince.Add(time.Hour).UTC().Format(fbTestStamp)
	cases := map[string]string{
		"missing id":             `{"created_time":"` + fresh + `"}`,
		"null id":                `{"id":null,"created_time":"` + fresh + `"}`,
		"numeric id":             `{"id":5,"created_time":"` + fresh + `"}`,
		"blank id":               `{"id":"   ","created_time":"` + fresh + `"}`,
		"blank id on an old row": `{"id":"","created_time":"` + old + `"}`,
		"missing created_time":   `{"id":"x"}`,
		"null created_time":      `{"id":"x","created_time":null}`,
		"numeric created_time":   `{"id":"x","created_time":1}`,
		"invalid created_time":   `{"id":"x","created_time":"yesterday"}`,
		"wrong layout":           `{"id":"x","created_time":"2026-09-02T00:00:00Z"}`,
		"zero created_time":      `{"id":"x","created_time":"0001-01-01T00:00:00+0000"}`,
	}
	for name, row := range cases {
		tr := fbmScript(`{"data":[` + row + `]}`)
		_, err := fbmFetch(tr, fbmSince)
		fbmAssertSafe(t, name, err)
		if !errors.Is(err, ErrFacebookMessageCoverageIncomplete) {
			t.Errorf("%s: err = %v, want message coverage incomplete", name, err)
		}
	}

	// The invalid REPEAT of a seen ID sits beside a NEW valid row and the page carries no next, so
	// dedupe-first code would finish successfully; only validate-before-dedupe reports it.
	tr := fbmScript(
		fbmPage(fbmNext(fbmConv, "c1"), fbmRow("m1", fbmSince.Add(time.Hour))),
		`{"data":[{"id":"m1","created_time":"not-a-time"},`+fbmRow("m2", fbmSince.Add(2*time.Hour))+`]}`)
	if _, err := fbmFetch(tr, fbmSince); !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || !strings.Contains(err.Error(), "valid id or created_time") {
		t.Fatalf("invalid repeat of a seen ID: err %v", err)
	}
	// An invalid OLD row (valid timestamp outside the window is fine; a broken one is not) beside
	// a fresh valid row on the same page.
	tr = fbmScript(`{"data":[` + fbmRow("fresh", fbmSince.Add(time.Hour)) + `,{"id":"older","created_time":"2020-13-45T99:00:00+0000"}]}`)
	if _, err := fbmFetch(tr, fbmSince); !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || !strings.Contains(err.Error(), "valid id or created_time") {
		t.Fatalf("invalid old row beside a fresh row: err %v", err)
	}
}

// F02E-03: terminal rules are Graph-specific. Empty and short pages with a safe next continue;
// absent paging or paging without next is terminal; malformed paging/next fails even on an empty
// or older-than-since page.
func TestFacebookMessageEmptyAndShortPagesWithNextContinue(t *testing.T) {
	tr := fbmScript(
		fbmPage(fbmNext(fbmConv, "c1")),                                        // empty, has next
		fbmPage(fbmNext(fbmConv, "c2"), fbmRow("e1", fbmSince.Add(time.Hour))), // short
		fbmPage(fbmNext(fbmConv, "c3")),                                        // empty again
		fbmPage("", fbmRow("e2", fbmSince.Add(2*time.Hour))),                   // nonempty terminal, no paging
	)
	got, err := fbmFetch(tr, fbmSince)
	if err != nil || fbmIDs(got) != "e1,e2" || tr.count() != 4 {
		t.Fatalf("empty/short pages with next: %s over %d requests, err %v", fbmIDs(got), tr.count(), err)
	}
	fbmAssertBodiesClosed(t, "continue", tr)
}

func TestFacebookMessageTerminalForms(t *testing.T) {
	row := fbmRow("a", fbmSince.Add(time.Hour))
	for name, body := range map[string]string{
		"empty data, no paging":      `{"data":[]}`,
		"nonempty data, no paging":   `{"data":[` + row + `]}`,
		"paging without next":        `{"data":[` + row + `],"paging":{"cursors":{"before":"b","after":"a"}}}`,
		"empty data, paging no next": `{"data":[],"paging":{}}`,
		"previous only (older link)": `{"data":[` + row + `],"paging":{"previous":"https://evil.example/x"}}`,
	} {
		tr := fbmScript(body)
		if _, err := fbmFetch(tr, fbmSince); err != nil || tr.count() != 1 {
			t.Errorf("%s: err %v after %d requests, want terminal success", name, err, tr.count())
		}
	}
	old := fbmRow("old", fbmSince.Add(-time.Hour))
	bad := map[string]string{
		"empty data, null paging":    `{"data":[],"paging":null}`,
		"empty data, string paging":  `{"data":[],"paging":"x"}`,
		"empty data, array paging":   `{"data":[],"paging":[]}`,
		"empty data, null next":      `{"data":[],"paging":{"next":null}}`,
		"empty data, numeric next":   `{"data":[],"paging":{"next":5}}`,
		"empty data, empty next":     `{"data":[],"paging":{"next":""}}`,
		"empty data, object next":    `{"data":[],"paging":{"next":{}}}`,
		"older page, null paging":    `{"data":[` + old + `],"paging":null}`,
		"older page, malformed next": `{"data":[` + old + `],"paging":{"next":7}}`,
		"nonempty page, null paging": `{"data":[` + row + `],"paging":null}`,
	}
	for name, body := range bad {
		tr := fbmScript(body, fbmPage("", fbmRow("zz", fbmSince.Add(time.Hour))))
		_, err := fbmFetch(tr, fbmSince)
		fbmAssertSafe(t, name, err)
		if !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || tr.count() != 1 {
			t.Errorf("%s: err %v after %d requests, want a paging failure and no second request", name, err, tr.count())
		}
	}
}

// F02E-04: exact inclusive-window ID set; old rows never terminate; no assumption about page order.
func TestFacebookMessageSinceFiltersOutputWithoutEarlyStop(t *testing.T) {
	pages := func() *fbmTransport {
		return fbmScript(
			fbmPage(fbmNext(fbmConv, "c1"), fbmRow("oldA", fbmSince.Add(-time.Hour)), fbmRow("newA", fbmSince.Add(3*time.Hour))),
			fbmPage(fbmNext(fbmConv, "c2"), fbmRow("newB", fbmSince.Add(2*time.Hour)), fbmRow("oldB", fbmSince.Add(-2*time.Hour))),
			fbmPage("", fbmRow("boundary", fbmSince)),
		)
	}
	tr := pages()
	got, err := fbmFetch(tr, fbmSince)
	if err != nil || fbmIDs(got) != "boundary,newB,newA" || tr.count() != 3 {
		t.Fatalf("eligible IDs %s over %d requests, err %v; want boundary,newB,newA over 3", fbmIDs(got), tr.count(), err)
	}
	all, err := fbmFetch(pages(), time.Time{})
	if err != nil || fbmIDs(all) != "oldB,oldA,boundary,newB,newA" {
		t.Fatalf("zero since: %s, err %v", fbmIDs(all), err)
	}
}

// F02E-04: first-seen mapping, stable chronological order, tie order, mapping semantics retained.
func TestFacebookMessageDeduplicateOrderAndMapping(t *testing.T) {
	t1, t2, t3 := fbmSince.Add(time.Hour), fbmSince.Add(2*time.Hour), fbmSince.Add(3*time.Hour)
	fmtT := func(x time.Time) string { return x.UTC().Format(fbTestStamp) }
	first := `{"id":"dup","message":"FIRST","from":{"id":"u-1","name":"Khach"},"created_time":"` + fmtT(t2) + `"}`
	second := `{"id":"dup","message":"SECOND","from":{"id":"` + fbTestPage + `","name":"Page"},"created_time":"` + fmtT(t3) + `"}`
	photo := `{"id":"photo","from":{"id":"u-1","name":"Khach"},"created_time":"` + fmtT(t1) + `","attachments":{"data":[{"mime_type":"image/jpeg","name":"a.jpg","image_data":{"url":"https://cdn.example/a.jpg"}}]}}`
	sticker := `{"id":"stk","from":{"id":"u-1","name":"Khach"},"created_time":"` + fmtT(t1) + `","sticker":"https://cdn.example/s.png"}`
	agent := `{"id":"agent","message":"chao","from":{"id":"` + fbTestPage + `","name":"Page"},"created_time":"` + fmtT(fbmSince) + `"}`
	tr := fbmScript(
		fbmPage(fbmNext(fbmConv, "c1"), fbmRow("c3", t3), first, second, photo, sticker),
		fbmPage("", fbmRow("c3", t3), agent, fbmRow("c2b", t2)), // c3 overlaps the previous page
	)
	got, err := fbmFetch(tr, fbmSince)
	if err != nil {
		t.Fatalf("FetchMessages: %v", err)
	}
	if want := "agent,photo,stk,dup,c2b,c3"; fbmIDs(got) != want {
		t.Fatalf("order = %s, want %s", fbmIDs(got), want)
	}
	byID := map[string]SyncedMessage{}
	for _, m := range got {
		byID[m.ExternalID] = m
	}
	if m := byID["dup"]; m.Content != "FIRST" || m.SenderType != "customer" || !m.SentAt.Equal(t2) {
		t.Fatalf("duplicate must keep its first-seen mapping: %+v", m)
	}
	if m := byID["agent"]; m.SenderType != "agent" || m.Content != "chao" {
		t.Fatalf("agent mapping changed: %+v", m)
	}
	if m := byID["photo"]; m.ContentType != "attachment" || len(m.Attachments) != 1 || m.Attachments[0].URL != "https://cdn.example/a.jpg" || m.Attachments[0].Type != "image/jpeg" {
		t.Fatalf("attachment mapping changed: %+v", m)
	}
	if m := byID["stk"]; m.ContentType != "sticker" {
		t.Fatalf("sticker mapping changed: %+v", m)
	}
	if byID["photo"].RawData == nil || byID["photo"].RawData["id"] != "photo" {
		t.Fatalf("RawData no longer carries the provider row")
	}
	for i := 1; i < len(got); i++ {
		if got[i].SentAt.Before(got[i-1].SentAt) {
			t.Fatalf("not chronological at %d", i)
		}
	}
}

// F02E-05: cycle identity is canonical and independent of the token, order and encoding.
func TestFacebookMessageCyclesFailClosed(t *testing.T) {
	row := func(id string) string { return fbmRow(id, fbmSince.Add(time.Hour)) }
	base := "https://graph.facebook.com/v21.0/" + fbmConv + "/messages"
	cases := map[string]struct {
		pages    []string
		requests int
	}{
		"direct repeat": {[]string{
			fbmPage(fbmNext(fbmConv, "c1"), row("a")), fbmPage(fbmNext(fbmConv, "c1"), row("b")), fbmPage("", row("c")),
		}, 2},
		"alternating": {[]string{
			fbmPage(fbmNext(fbmConv, "c1"), row("a")), fbmPage(fbmNext(fbmConv, "c2"), row("b")), fbmPage(fbmNext(fbmConv, "c1"), row("c")), fbmPage("", row("d")),
		}, 3},
		"initial page identity (rotated token)": {[]string{
			fbmPage(base+"?access_token=ROTATED&fields="+url.QueryEscape(fbmFields)+"&limit=100", row("a")), fbmPage("", row("b")),
		}, 1},
		"reordered query and rotated token": {[]string{
			fbmPage(base+"?access_token=T1&after=c1&fields="+url.QueryEscape(fbmFields), row("a")),
			fbmPage(base+"?fields="+url.QueryEscape(fbmFields)+"&after=c1&access_token=T2", row("b")), fbmPage("", row("c")),
		}, 2},
		"percent-encoded query": {[]string{
			fbmPage(base+"?after=c1&access_token=T1", row("a")), fbmPage(base+"?after=%63%31&access_token=T2", row("b")), fbmPage("", row("c")),
		}, 2},
		"token only differs": {[]string{
			fbmPage(base+"?after=c1&access_token=T1", row("a")), fbmPage(base+"?after=c1&access_token=T2", row("b")), fbmPage("", row("c")),
		}, 2},
	}
	for name, c := range cases {
		tr := fbmScript(c.pages...)
		_, err := fbmFetch(tr, fbmSince)
		fbmAssertSafe(t, name, err)
		if !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || !strings.Contains(err.Error(), "repeats") {
			t.Errorf("%s: err = %v, want a repeated-cursor failure", name, err)
		}
		if tr.count() != c.requests {
			t.Errorf("%s: %d requests, want %d and no more", name, tr.count(), c.requests)
		}
	}
}

// Distinct cursors on empty or duplicate-only pages continue: message IDs alone are not progress proof.
func TestFacebookMessageDistinctCursorsContinueDespiteEmptyOrDuplicatePages(t *testing.T) {
	a := fbmRow("a", fbmSince.Add(time.Hour))
	tr := fbmScript(
		fbmPage(fbmNext(fbmConv, "c1"), a),
		fbmPage(fbmNext(fbmConv, "c2"), a), // duplicate-only, new cursor
		fbmPage(fbmNext(fbmConv, "c3")),    // empty, new cursor
		fbmPage("", fbmRow("b", fbmSince.Add(2*time.Hour))),
	)
	got, err := fbmFetch(tr, fbmSince)
	if err != nil || fbmIDs(got) != "a,b" || tr.count() != 4 {
		t.Fatalf("distinct cursors: %s over %d requests, err %v", fbmIDs(got), tr.count(), err)
	}
}

// pageChain serves distinct pages after=c0.., each with one row, up to `pages` pages; the last has
// a next link only when withNextAtEnd is set.
func fbmChain(pages int, withNextAtEnd bool) *fbmTransport {
	tr := &fbmTransport{}
	tr.handle = func(n int, r *http.Request) (*http.Response, error) {
		if n > fbMaxMessagePages+2 { // a runaway traversal must fail the test, not hang it
			return nil, errors.New("runaway traversal beyond the page budget")
		}
		next := ""
		if n < pages-1 || withNextAtEnd {
			next = fbmNext(fbmConv, fmt.Sprintf("c%d", n+1))
		}
		return tr.respond(r, 200, fbmPage(next, fbmRow(fmt.Sprintf("m%04d", n), fbmSince.Add(time.Duration(n+1)*time.Minute))), nil)
	}
	return tr
}

func TestFacebookMessagePageBudgetBoundary(t *testing.T) {
	tr := fbmChain(fbMaxMessagePages, false) // page 500 is terminal
	got, err := fbmFetch(tr, fbmSince)
	if err != nil || len(got) != fbMaxMessagePages || tr.count() != fbMaxMessagePages {
		t.Fatalf("terminal page 500: %d rows over %d requests, err %v", len(got), tr.count(), err)
	}
	if got[0].ExternalID != "m0000" || got[len(got)-1].ExternalID != fmt.Sprintf("m%04d", fbMaxMessagePages-1) {
		t.Fatalf("boundary order wrong: %s .. %s", got[0].ExternalID, got[len(got)-1].ExternalID)
	}

	tr = fbmChain(fbMaxMessagePages, true) // page 500 carries another next
	got, err = fbmFetch(tr, fbmSince)
	fbmAssertSafe(t, "budget", err)
	if !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || !strings.Contains(err.Error(), "page budget") {
		t.Fatalf("page 500 with another next: err %v", err)
	}
	if tr.count() != fbMaxMessagePages || len(got) != fbMaxMessagePages {
		t.Fatalf("budget: %d requests / %d diagnostic rows, want exactly %d and no request %d", tr.count(), len(got), fbMaxMessagePages, fbMaxMessagePages+1)
	}
}

// F02E-06: redirects are blocked before the destination is contacted; HTTP failures never succeed.
func TestFacebookMessageRedirectsAreNeverFollowed(t *testing.T) {
	targets := map[string]string{
		"cross-host redirect":      "https://evil.example/steal?access_token=" + fbTestToken,
		"same-host redirect":       "https://graph.facebook.com/v21.0/" + fbmConv + "/messages?after=x",
		"different-path same host": "https://graph.facebook.com/v21.0/other/messages",
		"scheme downgrade":         "http://graph.facebook.com/v21.0/" + fbmConv + "/messages",
	}
	for name, loc := range targets {
		for _, code := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
			tr := &fbmTransport{}
			tr.handle = func(n int, r *http.Request) (*http.Response, error) {
				// A redirect response that, if followed, would be a valid page (traps a lenient client).
				return tr.respond(r, code, fbmPage("", fbmRow("trap", fbmSince.Add(time.Hour))), http.Header{"Location": {loc}})
			}
			got, err := fbmFetch(tr, fbmSince)
			label := fmt.Sprintf("%s (%d)", name, code)
			fbmAssertSafe(t, label, err)
			if tr.count() != 1 || tr.countHost("evil.example") != 0 || len(got) != 0 {
				t.Errorf("%s: %d requests (evil %d), %d rows; the redirect must not be followed or its body trusted", label, tr.count(), tr.countHost("evil.example"), len(got))
			}
			fbmAssertBodiesClosed(t, label, tr)
		}
	}
	// A redirect on a LATER page is also stopped (the first page's rows stay diagnostic).
	tr := &fbmTransport{}
	tr.handle = func(n int, r *http.Request) (*http.Response, error) {
		if n == 0 {
			return tr.respond(r, 200, fbmPage(fbmNext(fbmConv, "c1"), fbmRow("a", fbmSince.Add(time.Hour))), nil)
		}
		return tr.respond(r, http.StatusFound, "", http.Header{"Location": {"https://evil.example/x"}})
	}
	got, err := fbmFetch(tr, fbmSince)
	fbmAssertSafe(t, "late redirect", err)
	if tr.count() != 2 || tr.countHost("evil.example") != 0 || len(got) != 1 {
		t.Fatalf("late redirect: %d requests, evil %d, %d rows", tr.count(), tr.countHost("evil.example"), len(got))
	}
}

func TestFacebookMessageHTTPFailuresAndLateErrorsAreNeverSuccess(t *testing.T) {
	valid := fbmPage("", fbmRow("a", fbmSince.Add(time.Hour)))
	first := fbmPage(fbmNext(fbmConv, "c1"), fbmRow("p1", fbmSince.Add(time.Hour)), fbmRow("p2", fbmSince.Add(2*time.Hour)))
	resp := func(code int, body string) func(tr *fbmTransport, r *http.Request) (*http.Response, error) {
		return func(tr *fbmTransport, r *http.Request) (*http.Response, error) { return tr.respond(r, code, body, nil) }
	}
	late := map[string]func(tr *fbmTransport, r *http.Request) (*http.Response, error){
		"http 500 with a valid data array": resp(500, valid),
		"http 503 non-json":                resp(503, "<html>"+fbmMarker+"</html>"),
		"http 429 non-json":                resp(429, fbmMarker),
		"http 404 json":                    resp(404, `{"data":[]}`),
		"graph error object":               resp(200, `{"error":{"message":"`+fbmMarker+" "+fbTestToken+`","code":4}}`),
		"malformed json":                   resp(200, `{"data":[`),
		"missing data array":               resp(200, `{"paging":{}}`),
		"oversized response":               resp(200, fbmPage("")+strings.Repeat(" ", fbMaxMessageResponseBytes)),
		"connection error (round-trip)": func(tr *fbmTransport, r *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("connection reset by peer for %s", r.URL.String())
		},
	}
	for name, h := range late {
		h := h
		tr := &fbmTransport{}
		tr.handle = func(n int, r *http.Request) (*http.Response, error) {
			if n == 0 {
				return tr.respond(r, 200, first, nil)
			}
			return h(tr, r)
		}
		got, err := fbmFetch(tr, fbmSince)
		fbmAssertSafe(t, name, err)
		if len(got) != 2 || tr.count() != 2 {
			t.Errorf("%s: %d diagnostic rows over %d requests, want 2 over 2", name, len(got), tr.count())
		}
		if name == "oversized response" && !strings.Contains(err.Error(), "size budget") {
			t.Errorf("oversized: err %v", err)
		}
		fbmAssertBodiesClosed(t, name, tr)
	}
	// The same shapes on the very first request.
	for name, h := range late {
		h := h
		tr := &fbmTransport{}
		tr.handle = func(n int, r *http.Request) (*http.Response, error) { return h(tr, r) }
		if _, err := fbmFetch(tr, fbmSince); err == nil {
			t.Errorf("%s: first-request failure returned success", name)
		}
	}
}

// A body that fails mid-read is a read failure, not a terminal page.
type fbmBrokenBody struct{ read bool }

func (b *fbmBrokenBody) Read(p []byte) (int, error) {
	if !b.read {
		b.read = true
		return copy(p, `{"data":[`), nil
	}
	return 0, errors.New("stream broken " + fbTestToken)
}
func (b *fbmBrokenBody) Close() error { return nil }

func TestFacebookMessageBodyReadFailureIsNotSuccess(t *testing.T) {
	a := NewFacebookAdapter(FacebookCredentials{PageID: fbTestPage, AccessToken: fbTestToken})
	a.client = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: &fbmBrokenBody{}, Request: r}, nil
	})}
	_, err := a.FetchMessages(context.Background(), fbmConv, fbmSince)
	fbmAssertSafe(t, "broken body", err)
}

// Cancellation is recognizable and never success; orderings are fixed by in-transport hooks.
func TestFacebookMessageCancellationIsNeverSuccess(t *testing.T) {
	hooked := func(hook func(n int), body func(n int) string) (*fbmTransport, *int) {
		tr := &fbmTransport{}
		n := 0
		tr.handle = func(i int, r *http.Request) (*http.Response, error) {
			n++
			if hook != nil {
				hook(i)
			}
			return tr.respond(r, 200, body(i), nil)
		}
		return tr, &n
	}
	t.Run("before the first request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		tr, n := hooked(nil, func(int) string { return fbmPage("") })
		_, err := fbmAdapter(tr).FetchMessages(ctx, fbmConv, fbmSince)
		if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrFacebookMessageCoverageIncomplete) || *n != 0 {
			t.Fatalf("pre-cancelled: err %v after %d requests", err, *n)
		}
	})
	t.Run("on the terminal page", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		tr, _ := hooked(func(int) { cancel() }, func(int) string { return fbmPage("", fbmRow("a", fbmSince.Add(time.Hour))) })
		got, err := fbmAdapter(tr).FetchMessages(ctx, fbmConv, fbmSince)
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled terminal page returned success: %d rows, err %v", len(got), err)
		}
		fbmAssertBodiesClosed(t, "terminal cancel", tr)
	})
	t.Run("between pages", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		tr, n := hooked(func(i int) {
			if i == 0 {
				cancel()
			}
		}, func(i int) string { return fbmPage(fbmNext(fbmConv, "c1"), fbmRow("a", fbmSince.Add(time.Hour))) })
		_, err := fbmAdapter(tr).FetchMessages(ctx, fbmConv, fbmSince)
		if !errors.Is(err, context.Canceled) || *n != 1 {
			t.Fatalf("cancel after page 1: err %v after %d requests, want 1", err, *n)
		}
	})
}
