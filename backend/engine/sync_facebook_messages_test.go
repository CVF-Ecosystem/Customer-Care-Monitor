package engine

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-031 (F02-E): the real FacebookAdapter message traversal, driven by the synthetic
// Graph transport of sync_facebook_coverage_test.go (installed as http.DefaultTransport; nothing
// leaves the machine), feeds the real SyncEngine on a disposable MySQL. Assertions read stored
// message IDs, the checkpoint, the recorded after-sync seam and the request log; they cover
// application semantics for the synthetic contract only, never live Graph behaviour or runtime AI
// governance. Storage/dispatch helpers are shared with sync_pancake_messages_test.go.

const (
	fbEngProviderToken = "PROVIDER-LINK-TOKEN-ENG-R031"
	fbEngMarker        = "R031-ENG-BODY-MARKER"
	fbEngFields        = "id,message,from,to,created_time,attachments,shares,sticker"
)

func fbEngTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05-0700") }

func fbEngMsg(id string, at time.Time) string {
	return fmt.Sprintf(`{"id":%q,"message":"tin %s","from":{"id":"u","name":"Khach"},"created_time":%q}`, id, id, fbEngTime(at))
}

// fbEngNext is a provider-style next link; it carries a provider token the adapter must replace.
func fbEngNext(conv, cursor string) string {
	return fmt.Sprintf("https://graph.facebook.com/v21.0/%s/messages?access_token=%s&after=%s&fields=%s&limit=100", conv, fbEngProviderToken, cursor, fbEngFields)
}

func fbEngMsgPage(next string, rows ...string) string {
	body := `{"data":[` + strings.Join(rows, ",") + `]`
	if next != "" {
		body += fmt.Sprintf(`,"paging":{"next":%q}`, next)
	}
	return body + `}`
}

func fbEngResp(r *http.Request, code int, body string) (*http.Response, error) {
	return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func (g *fbGraph) messageLogString() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.msgLog, ",")
}

// F02E-07: a complete multi-page traversal (an empty intermediate page with a next link, an
// overlapping row, the since boundary, ineligible older rows, newest-first order) stores exactly
// the eligible IDs; the success checkpoint advances, after-sync fires once, the sibling
// channel/tenant stay empty and a replay over the identical window adds nothing.
func TestFacebookSyncStoresExactEligibleMessageIDsAcrossPages(t *testing.T) {
	g := &fbGraph{total: 2, pageSize: 50}
	f, chID := fbEngFixture(t, g)
	previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, previous)
	since := previous.Add(-time.Hour) // the engine's window start
	at := func(d time.Duration) time.Time { return since.Add(d) }

	g.msgFn = func(conv string, r *http.Request) (*http.Response, error) {
		after := r.URL.Query().Get("after")
		if conv == "t_0001" {
			return fbEngResp(r, 200, fbEngMsgPage("", fbEngMsg("two_b", at(2*time.Hour)), fbEngMsg("two_a", at(time.Hour))))
		}
		switch after {
		case "":
			return fbEngResp(r, 200, fbEngMsgPage(fbEngNext(conv, "p1"), fbEngMsg("e4", at(4*time.Hour)), fbEngMsg("e3", at(3*time.Hour)), fbEngMsg("oldX", at(-2*time.Hour))))
		case "p1": // empty page that still carries a next link
			return fbEngResp(r, 200, fbEngMsgPage(fbEngNext(conv, "p2")))
		case "p2": // overlaps e3, adds e2, e1 and the since boundary, then terminal
			return fbEngResp(r, 200, fbEngMsgPage("", fbEngMsg("e3", at(3*time.Hour)), fbEngMsg("e2", at(2*time.Hour)), fbEngMsg("e1", at(time.Hour)), fbEngMsg("boundary", since), fbEngMsg("oldY", at(-3*time.Hour))))
		}
		return fbEngResp(r, 500, fbEngMarker)
	}

	before := time.Now().Truncate(time.Second)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := "boundary,e1,e2,e3,e4"
	if got := f.storedMessageIDs(t, chID, "t_0000"); strings.Join(got, ",") != want {
		t.Fatalf("t_0000 stored IDs = %v, want exactly %s", got, want)
	}
	if got := f.storedMessageIDs(t, chID, "t_0001"); strings.Join(got, ",") != "two_a,two_b" {
		t.Fatalf("t_0001 stored IDs = %v, want two_a,two_b", got)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 2 || msgs != 7 {
		t.Fatalf("stored %d conversations / %d messages, want 2 / 7", convs, msgs)
	}
	if log := g.messageLogString(); log != "t_0000?-,t_0000?p1,t_0000?p2,t_0001?-" {
		t.Fatalf("message request log = %s", log)
	}
	if g.badTokens != 0 || len(g.otherHosts) != 0 {
		t.Fatalf("message requests with a non-configured token: %d, unexpected hosts: %v", g.badTokens, g.otherHosts)
	}
	st := f.statusOf(t, chID)
	if st.Status != "success" || st.LastSyncAt == nil || st.LastSyncAt.Before(before) || !st.LastSyncAt.After(previous) {
		t.Fatalf("status %q checkpoint %v (previous %v, fetch started at or after %v)", st.Status, st.LastSyncAt, previous, before)
	}
	if f.triggerCount() != 1 || f.completedActivityCount(t) != 1 {
		t.Fatalf("after-sync fired %d times / completion activity %d, want 1 / 1", f.triggerCount(), f.completedActivityCount(t))
	}
	var siblings int64
	if err := db.DB.Raw("SELECT COUNT(*) FROM messages WHERE tenant_id = ? OR conversation_id IN (SELECT id FROM conversations WHERE channel_id IN (?, ?))", f.otherTenantID, f.chB, f.chX).Scan(&siblings).Error; err != nil || siblings != 0 {
		t.Fatalf("sibling channel/tenant messages = %d, err %v", siblings, err)
	}

	f.setCheckpoint(t, chID, previous)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := f.storedMessageIDs(t, chID, "t_0000"); strings.Join(got, ",") != want {
		t.Fatalf("replay changed t_0000 IDs to %v", got)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 2 || msgs != 7 {
		t.Fatalf("replay changed counts to %d / %d", convs, msgs)
	}
	f.assertNoDuplicateMessages(t)
}

// F02E-08: after valid rows, a later-page contract failure, a malformed row, an unsafe next link, an
// actual connection error and (separately labelled) an HTTP 500 all make the run partial. The
// failed conversation's diagnostic messages are not stored, the peer conversation is, the
// checkpoint stays, no after-sync analysis is dispatched and no completion is recorded; the
// retry stores everything once and a replay changes nothing.
func TestFacebookSyncMessageFailureIsPartialAndRetryCompletes(t *testing.T) {
	type mode struct {
		page2 func(r *http.Request) (*http.Response, error)
		log   string // request log of the first (failing) run
	}
	const failedLog = "t_0000?-,t_0000?p1,t_0001?-"
	modes := map[string]mode{
		"later-page contract failure (no data array)": {func(r *http.Request) (*http.Response, error) {
			return fbEngResp(r, 200, `{"paging":{}}`)
		}, failedLog},
		"malformed row on a later page": {func(r *http.Request) (*http.Response, error) {
			return fbEngResp(r, 200, `{"data":[{"id":"bad_x","created_time":"not-a-time"}]}`)
		}, failedLog},
		"unsafe next link on a later page": {func(r *http.Request) (*http.Response, error) {
			return fbEngResp(r, 200, fbEngMsgPage("https://evil.example/v21.0/t_0000/messages?after=p2", fbEngMsg("bad_n", time.Now())))
		}, failedLog},
		"actual connection error after valid rows": {func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("connection reset by peer " + fbEngToken)
		}, failedLog},
		"HTTP 500 (not a connection error) with a valid array": {func(r *http.Request) (*http.Response, error) {
			return fbEngResp(r, 500, fbEngMsgPage("", fbEngMsg("bad_h", time.Now())))
		}, failedLog},
		"redirect on a later page": {func(r *http.Request) (*http.Response, error) {
			resp, _ := fbEngResp(r, http.StatusFound, "")
			resp.Header.Set("Location", "https://evil.example/steal")
			return resp, nil
		}, failedLog},
	}
	for name, m := range modes {
		m := m
		t.Run(name, func(t *testing.T) {
			g := &fbGraph{total: 2, pageSize: 50}
			f, chID := fbEngFixture(t, g)
			previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
			f.setCheckpoint(t, chID, previous)
			recent := func(d time.Duration) time.Time { return time.Now().Add(d) }

			var mu sync.Mutex
			failing := true
			g.msgFn = func(conv string, r *http.Request) (*http.Response, error) {
				mu.Lock()
				fail := failing
				mu.Unlock()
				after := r.URL.Query().Get("after")
				if conv == "t_0001" {
					return fbEngResp(r, 200, fbEngMsgPage("", fbEngMsg("ok_1", recent(-8*time.Minute))))
				}
				switch {
				case after == "":
					return fbEngResp(r, 200, fbEngMsgPage(fbEngNext(conv, "p1"), fbEngMsg("bad_new", recent(-7*time.Minute)), fbEngMsg("bad_old", recent(-time.Hour))))
				case after == "p1" && fail:
					return m.page2(r)
				case after == "p1":
					return fbEngResp(r, 200, fbEngMsgPage("", fbEngMsg("bad_older", recent(-2*time.Hour))))
				}
				return fbEngResp(r, 500, fbEngMarker)
			}

			runErr := f.runSync(t, chID)
			t.Logf("partial run returned %v", runErr)
			st := f.statusOf(t, chID)
			if st.Status != "partial" {
				t.Fatalf("status = %q (err %q), want partial", st.Status, st.Error)
			}
			if st.LastSyncAt == nil || !st.LastSyncAt.Equal(previous) {
				t.Fatalf("checkpoint %v moved from %v after a partial run", st.LastSyncAt, previous)
			}
			if f.triggerCount() != 0 || f.completedActivityCount(t) != 0 {
				t.Fatalf("after-sync fired %d times / completion activity %d after a partial run, want 0 / 0", f.triggerCount(), f.completedActivityCount(t))
			}
			if got := f.storedMessageIDs(t, chID, "t_0000"); len(got) != 0 {
				t.Fatalf("failed conversation stored its diagnostic messages: %v", got)
			}
			if got := f.storedMessageIDs(t, chID, "t_0001"); strings.Join(got, ",") != "ok_1" {
				t.Fatalf("peer conversation stored %v, want ok_1", got)
			}
			if log := g.messageLogString(); log != m.log {
				t.Fatalf("message request log = %s, want %s", log, m.log)
			}
			if len(g.otherHosts) != 0 || g.badTokens != 0 {
				t.Fatalf("unexpected hosts %v or %d message requests with a non-configured token", g.otherHosts, g.badTokens)
			}
			for _, banned := range []string{fbEngToken, fbEngProviderToken, "access_token", "graph.facebook.com", "evil.example", fbEngMarker, "bad_x", "bad_n"} {
				if strings.Contains(st.Error, banned) {
					t.Fatalf("stored error leaks %q: %s", banned, st.Error)
				}
			}

			mu.Lock()
			failing = false
			mu.Unlock()
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("retry: %v", err)
			}
			st = f.statusOf(t, chID)
			if st.Status != "success" || st.LastSyncAt == nil || !st.LastSyncAt.After(previous) {
				t.Fatalf("after retry: status %q checkpoint %v", st.Status, st.LastSyncAt)
			}
			if got := f.storedMessageIDs(t, chID, "t_0000"); strings.Join(got, ",") != "bad_new,bad_old,bad_older" {
				t.Fatalf("retry stored %v for the formerly failed conversation", got)
			}
			if f.triggerCount() != 1 || f.completedActivityCount(t) != 1 {
				t.Fatalf("after retry: after-sync %d / completion %d, want 1 / 1", f.triggerCount(), f.completedActivityCount(t))
			}
			convs, msgs := f.storedCounts(t, chID)
			f.setCheckpoint(t, chID, previous)
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("replay: %v", err)
			}
			if c2, m2 := f.storedCounts(t, chID); c2 != convs || m2 != msgs {
				t.Fatalf("replay changed counts from %d / %d to %d / %d", convs, msgs, c2, m2)
			}
			f.assertNoDuplicateMessages(t)
		})
	}
}
