package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-024 (F02-C): the real ZaloOAAdapter, driven by a synthetic openapi.zalo.me /
// oauth.zaloapp.com transport (installed as http.DefaultTransport; nothing leaves the machine),
// feeds the real SyncEngine on a disposable MySQL. Assertions read stored rows, credentials and
// the checkpoint.

const (
	zlEngAccess  = "zl-eng-ACCESS-R024"
	zlEngRefresh = "zl-eng-REFRESH-R024"
	zlEngNewAcc  = "zl-eng-NEW-ACCESS-R024"
	zlEngNewRef  = "zl-eng-NEW-REFRESH-R024"
)

type zlEngConv struct {
	id string
	ms int64
}

type zlFake struct {
	mu          sync.Mutex
	convs       []zlEngConv
	validToken  string
	failPage    int    // 1-based list page that fails (0 = none)
	failMode    string // "http500" | "malformed"
	expirePage  int    // 1-based list page that answers -216 once
	expired     bool
	listReqs    int
	msgReqs     int
	refreshReqs int
	otherHosts  []string
	onMessage   func()

	// CCMAI-RUNTIME-032 (F02-F) message traversal fixtures. msgRows overrides the default
	// single-message history of a conversation (rows are served by physical offset, then an
	// explicit empty page). msgFail answers a message request with a failure instead; msgExpire
	// ("user@offset") answers -216 once for that request. msgLog records "user@offset".
	msgRows    map[string][]string
	msgFail    func(user string, offset int) (code int, body string, err error)
	msgExpire  string
	msgExpired bool
	msgLog     []string
}

func (g *zlFake) RoundTrip(r *http.Request) (*http.Response, error) {
	reply := func(code int, body string) (*http.Response, error) {
		return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}
	g.mu.Lock()
	switch r.URL.Host {
	case "oauth.zaloapp.com":
		g.refreshReqs++
		ok := r.URL.Query().Get("refresh_token") == zlEngRefresh
		if ok {
			g.validToken = zlEngNewAcc
		}
		g.mu.Unlock()
		if !ok {
			return reply(200, `{"error":-14014,"message":"bad refresh"}`)
		}
		return reply(200, `{"access_token":"`+zlEngNewAcc+`","refresh_token":"`+zlEngNewRef+`"}`)
	case "openapi.zalo.me":
	default:
		g.otherHosts = append(g.otherHosts, r.URL.Host)
		g.mu.Unlock()
		return nil, errors.New("unexpected host in synthetic zalo")
	}
	var p struct {
		Offset, Count int
		UserID        string `json:"user_id"`
	}
	_ = json.Unmarshal([]byte(r.URL.Query().Get("data")), &p)
	if r.Header.Get("access_token") != g.validToken {
		g.mu.Unlock()
		return reply(200, `{"error":-216,"message":"Access token is invalid"}`)
	}
	switch r.URL.Path {
	case "/v2.0/oa/listrecentchat":
		g.listReqs++
		page := p.Offset/10 + 1
		if page == g.expirePage && !g.expired {
			g.expired = true
			g.validToken = "" // the current token stops working until refreshed
			g.mu.Unlock()
			return reply(200, `{"error":-216,"message":"Access token is invalid"}`)
		}
		if page == g.failPage {
			mode := g.failMode
			g.mu.Unlock()
			if mode == "malformed" {
				return reply(200, `{"error":0}`)
			}
			return reply(500, "synthetic server error "+zlEngAccess)
		}
		var rows []string
		for i := p.Offset; i < p.Offset+p.Count && i < len(g.convs); i++ {
			c := g.convs[i]
			rows = append(rows, fmt.Sprintf(`{"src":1,"from_id":%q,"from_display_name":"Khach","to_id":"oa","time":%d,"message":"x"}`, c.id, c.ms))
		}
		g.mu.Unlock()
		return reply(200, `{"error":0,"message":"Success","data":[`+strings.Join(rows, ",")+`]}`)
	case "/v2.0/oa/conversation":
		g.msgReqs++
		g.msgLog = append(g.msgLog, fmt.Sprintf("%s@%d", p.UserID, p.Offset))
		hook := g.onMessage
		fail := g.msgFail
		var ms int64
		for _, c := range g.convs {
			if c.id == p.UserID {
				ms = c.ms
			}
		}
		if g.msgExpire == fmt.Sprintf("%s@%d", p.UserID, p.Offset) && !g.msgExpired {
			g.msgExpired = true
			g.validToken = "" // the current token stops working until refreshed
			g.mu.Unlock()
			return reply(200, `{"error":-216,"message":"Access token is invalid"}`)
		}
		// By default a conversation has one message at offset 0 and nothing after it; a custom
		// history is served by physical offset with an explicit empty page past its end.
		rows, custom := g.msgRows[p.UserID]
		if !custom {
			rows = []string{fmt.Sprintf(`{"message_id":"m_%s","src":1,"time":%d,"type":"text","message":"hello","from_display_name":"Khach"}`, p.UserID, ms)}
		}
		g.mu.Unlock()
		if hook != nil {
			hook()
		}
		if fail != nil {
			if code, body, err := fail(p.UserID, p.Offset); code != 0 || err != nil {
				if err != nil {
					return nil, err
				}
				return reply(code, body)
			}
		}
		var page []string
		for i := p.Offset; i < p.Offset+p.Count && i < len(rows); i++ {
			page = append(page, rows[i])
		}
		return reply(200, `{"error":0,"message":"Success","data":[`+strings.Join(page, ",")+`]}`)
	}
	g.mu.Unlock()
	return reply(404, "unscripted synthetic zalo path")
}

func (g *zlFake) add(id string, ms int64, front bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if front {
		g.convs = append([]zlEngConv{{id, ms}}, g.convs...)
		return
	}
	g.convs = append(g.convs, zlEngConv{id, ms})
}

func zlEngFixture(t *testing.T, g *zlFake) (*sfFixture, string) {
	t.Helper()
	f := setupSFFixture(t)
	chID := f.chA
	f.setType(t, chID, "zalo_oa")
	creds, err := pkg.Encrypt([]byte(fmt.Sprintf(`{"app_id":"app","app_secret":"secret","access_token":%q,"refresh_token":%q,"oa_id":"oa"}`, zlEngAccess, zlEngRefresh)), sfKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Exec("UPDATE channels SET credentials_encrypted = ? WHERE id = ?", creds, chID).Error; err != nil {
		t.Fatal(err)
	}
	g.validToken = zlEngAccess
	newSyncAdapter = channels.NewAdapter // the fixture's cleanup restores the original
	orig := http.DefaultTransport
	http.DefaultTransport = g
	t.Cleanup(func() { http.DefaultTransport = orig })
	return f, chID
}

func zlConvs(g *zlFake, prefix string, n int, at time.Time) {
	for i := 0; i < n; i++ {
		g.add(fmt.Sprintf("%s_%04d", prefix, i), at.UnixMilli(), false)
	}
}

func (f *sfFixture) storedCreds(t *testing.T, id string) map[string]string {
	t.Helper()
	var ch models.Channel
	if err := db.DB.Take(&ch, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	plain, err := pkg.Decrypt(ch.CredentialsEncrypted, sfKey)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	if err := json.Unmarshal(plain, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// 105 conversations over 11 pages plus the empty page are all stored with their messages; the
// checkpoint is the fetch start, after-sync fires once.
func TestZaloSyncStoresEveryConversationBeyondTheOldLimit(t *testing.T) {
	g := &zlFake{}
	zlConvs(g, "zl", 105, time.Now().Add(-10*time.Minute))
	f, chID := zlEngFixture(t, g)
	previous := time.Now().Add(-72 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, previous)

	before := time.Now().Truncate(time.Second)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	after := time.Now()
	if convs, msgs := f.storedCounts(t, chID); convs != 105 || msgs != 105 {
		t.Fatalf("stored %d / %d, want 105 / 105", convs, msgs)
	}
	st := f.statusOf(t, chID)
	if st.Status != "success" || st.LastSyncAt == nil || st.LastSyncAt.Before(before) || st.LastSyncAt.After(after) {
		t.Fatalf("status %q checkpoint %v, want success within the run", st.Status, st.LastSyncAt)
	}
	// Each of the 105 conversations costs its one-row message page plus the explicit empty page.
	if g.listReqs != 12 || g.msgReqs != 210 || g.refreshReqs != 0 || len(g.otherHosts) != 0 {
		t.Fatalf("traffic: %d list pages, %d message fetches, %d refreshes, other hosts %v", g.listReqs, g.msgReqs, g.refreshReqs, g.otherHosts)
	}
	if f.triggerCount() != 1 {
		t.Fatalf("after-sync fired %d times, want 1", f.triggerCount())
	}
}

func TestZaloSyncPageFailureKeepsCheckpointAndRetryCompletesIdempotently(t *testing.T) {
	for _, mode := range []string{"http500", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			g := &zlFake{failPage: 3, failMode: mode}
			zlConvs(g, "rt", 35, time.Now().Add(-10*time.Minute))
			f, chID := zlEngFixture(t, g)
			previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
			f.setCheckpoint(t, chID, previous)

			if err := f.runSync(t, chID); err == nil {
				t.Fatal("a run whose page 3 failed must not report success")
			}
			st := f.statusOf(t, chID)
			if st.Status != "error" || st.LastSyncAt == nil || !st.LastSyncAt.Equal(previous) {
				t.Fatalf("after failure: status %q checkpoint %v, want error and unchanged %v", st.Status, st.LastSyncAt, previous)
			}
			if convs, msgs := f.storedCounts(t, chID); convs != 0 || msgs != 0 {
				t.Fatalf("partial window was processed: %d / %d", convs, msgs)
			}
			if f.triggerCount() != 0 {
				t.Fatalf("after-sync fired %d times after a failed window", f.triggerCount())
			}
			for _, banned := range []string{zlEngAccess, zlEngRefresh, "openapi.zalo.me", "synthetic server error"} {
				if strings.Contains(st.Error, banned) {
					t.Fatalf("stored error leaks %q: %s", banned, st.Error)
				}
			}
			if g.listReqs != 3 || g.msgReqs != 0 {
				t.Fatalf("traffic after failure: %d list pages, %d message fetches", g.listReqs, g.msgReqs)
			}

			g.mu.Lock()
			g.failPage = 0
			g.mu.Unlock()
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("retry: %v", err)
			}
			if convs, msgs := f.storedCounts(t, chID); convs != 35 || msgs != 35 {
				t.Fatalf("after retry: %d / %d, want 35 / 35", convs, msgs)
			}
			st = f.statusOf(t, chID)
			if st.Status != "success" || st.LastSyncAt == nil || !st.LastSyncAt.After(previous) {
				t.Fatalf("after retry: status %q checkpoint %v", st.Status, st.LastSyncAt)
			}
			if f.triggerCount() != 1 {
				t.Fatalf("after-sync fired %d times, want 1", f.triggerCount())
			}
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("replay: %v", err)
			}
			if convs, msgs := f.storedCounts(t, chID); convs != 35 || msgs != 35 {
				t.Fatalf("replay changed counts to %d / %d", convs, msgs)
			}
		})
	}
}

// An expired token mid-enumeration is refreshed once, the rotated pair is persisted to the
// channel's encrypted credentials by the owning run, the same page is retried and the run
// completes.
func TestZaloSyncRefreshesPersistsAndCompletes(t *testing.T) {
	g := &zlFake{expirePage: 2}
	zlConvs(g, "rf", 25, time.Now().Add(-10*time.Minute))
	f, chID := zlEngFixture(t, g)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync with refresh: %v", err)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 25 || msgs != 25 {
		t.Fatalf("stored %d / %d, want 25 / 25", convs, msgs)
	}
	if g.refreshReqs != 1 || g.listReqs != 5 { // pages 1, 2 (expired), 2 again, 3, empty
		t.Fatalf("traffic: %d refreshes, %d list requests", g.refreshReqs, g.listReqs)
	}
	creds := f.storedCreds(t, chID)
	if creds["access_token"] != zlEngNewAcc || creds["refresh_token"] != zlEngNewRef || creds["app_id"] != "app" {
		t.Fatal("rotated token pair was not persisted with the other credential fields intact")
	}
	if st := f.statusOf(t, chID); st.Status != "success" {
		t.Fatalf("status %q", st.Status)
	}
}

// A long successful run keeps the checkpoint at the fetch start while updated_at records the
// (simulated, two hours later) completion, so a conversation that moves to the front of the list
// during the run is inside the next run's window. With a completion-time checkpoint it would be
// filtered out as older than the next window.
func TestZaloCheckpointIsBoundedByFetchStart(t *testing.T) {
	g := &zlFake{}
	zlConvs(g, "dl", 3, time.Now().Add(-10*time.Minute))
	f, chID := zlEngFixture(t, g)
	f.setCheckpoint(t, chID, time.Now().Add(-48*time.Hour).UTC().Truncate(time.Second))

	var mu sync.Mutex
	elapsed := false
	origNow := syncNow
	syncNow = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		if elapsed {
			return time.Now().Add(2 * time.Hour)
		}
		return time.Now()
	}
	t.Cleanup(func() { syncNow = origNow })
	var once sync.Once
	g.onMessage = func() {
		once.Do(func() {
			time.Sleep(1100 * time.Millisecond)
			g.add("dl_late", time.Now().UnixMilli(), true)
			mu.Lock()
			elapsed = true
			mu.Unlock()
		})
	}

	before := time.Now().Truncate(time.Second)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("first run: %v", err)
	}
	st := f.statusOf(t, chID)
	if st.Status != "success" || st.LastSyncAt == nil || st.LastSyncAt.Before(before) || st.LastSyncAt.After(before.Add(time.Second)) {
		t.Fatalf("checkpoint %v, want the fetch start (%v)", st.LastSyncAt, before)
	}
	updatedAt, err := time.ParseInLocation("2006-01-02 15:04:05", st.UpdatedAt[:19], time.UTC)
	if err != nil || updatedAt.Sub(*st.LastSyncAt) < 119*time.Minute {
		t.Fatalf("updated_at %q should record the simulated completion (%v)", st.UpdatedAt, err)
	}
	if ids := f.storedPancakeIDs(t, chID); len(ids) != 3 {
		t.Fatalf("first run stored %v; the late conversation appeared after its pages", ids)
	}

	mu.Lock()
	elapsed = false
	mu.Unlock()
	g.mu.Lock()
	g.onMessage = nil
	g.mu.Unlock()
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("second run: %v", err)
	}
	ids := f.storedPancakeIDs(t, chID)
	if len(ids) != 4 || ids[3] != "dl_late" {
		t.Fatalf("second run stored %v; the conversation updated during the first run was missed", ids)
	}
}
