package engine

import (
	"context"
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
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-022 (F02-A): the real FacebookAdapter, driven by a synthetic Graph transport
// (installed as http.DefaultTransport; nothing leaves the machine), feeds the real SyncEngine
// on a disposable MySQL. The assertions read stored rows and the stored checkpoint, not only
// call counts.

const (
	fbEngPage  = "PAGE-ENG-R022"
	fbEngToken = "TOKEN-ENG-R022-SECRET"
)

type fbGraph struct {
	mu          sync.Mutex
	total       int    // conversations in the window
	pageSize    int    // rows per page
	failPage    int    // 1-based conversations page that fails (0 = none)
	failMode    string // "graph-error" | "transport"
	convReqs    int
	messageReqs int
	otherHosts  []string
}

func (g *fbGraph) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "graph.facebook.com" {
		g.mu.Lock()
		g.otherHosts = append(g.otherHosts, r.URL.Host)
		g.mu.Unlock()
		return nil, errors.New("unexpected host in synthetic graph")
	}
	reply := func(body string) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}
	stamp := time.Now().UTC().Format("2006-01-02T15:04:05-0700")
	switch {
	case r.URL.Path == "/v21.0/"+fbEngPage+"/conversations":
		g.mu.Lock()
		g.convReqs++
		g.mu.Unlock()
		page := 1
		if after := r.URL.Query().Get("after"); after != "" {
			fmt.Sscanf(after, "p%d", &page)
		}
		if page == g.failPage {
			if g.failMode == "transport" {
				return nil, errors.New("connection reset by peer")
			}
			return reply(`{"error":{"message":"synthetic page failure","code":4}}`)
		}
		var rows []string
		for i := (page - 1) * g.pageSize; i < page*g.pageSize && i < g.total; i++ {
			rows = append(rows, fmt.Sprintf(`{"id":"t_%04d","updated_time":%q,"participants":{"data":[{"id":%q,"name":"Page"},{"id":"u%d","name":"Customer %d"}]}}`, i, stamp, fbEngPage, i, i))
		}
		next := ""
		if page*g.pageSize < g.total {
			next = fmt.Sprintf(`,"paging":{"next":"https://graph.facebook.com/v21.0/%s/conversations?access_token=%s&after=p%d&fields=id"}`, fbEngPage, fbEngToken, page+1)
		}
		return reply(`{"data":[` + strings.Join(rows, ",") + `]` + next + `}`)
	case strings.HasSuffix(r.URL.Path, "/messages"):
		g.mu.Lock()
		g.messageReqs++
		g.mu.Unlock()
		conv := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v21.0/"), "/messages")
		return reply(fmt.Sprintf(`{"data":[{"id":"m_%s","message":"hello %s","from":{"id":"u","name":"Customer"},"created_time":%q}]}`, conv, conv, stamp))
	}
	return nil, errors.New("unscripted synthetic graph path " + r.URL.Path)
}

// fbEngFixture turns channel chA into a Facebook channel whose credentials point at the synthetic
// page, restores the real adapter factory and installs the fake Graph transport.
func fbEngFixture(t *testing.T, g *fbGraph) (*sfFixture, string) {
	t.Helper()
	f := setupSFFixture(t)
	chID := f.chA
	f.setType(t, chID, "facebook")
	creds, err := pkg.Encrypt([]byte(fmt.Sprintf(`{"page_id":%q,"access_token":%q}`, fbEngPage, fbEngToken)), sfKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Exec("UPDATE channels SET credentials_encrypted = ? WHERE id = ?", creds, chID).Error; err != nil {
		t.Fatal(err)
	}
	newSyncAdapter = channels.NewAdapter // the fixture's cleanup restores the original
	orig := http.DefaultTransport
	http.DefaultTransport = g
	t.Cleanup(func() { http.DefaultTransport = orig })
	return f, chID
}

func (f *sfFixture) storedCounts(t *testing.T, channelID string) (conversations, messages int64) {
	t.Helper()
	if err := db.DB.Raw("SELECT COUNT(*) FROM conversations WHERE tenant_id = ? AND channel_id = ?", f.tenantID, channelID).Scan(&conversations).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Raw("SELECT COUNT(*) FROM messages WHERE tenant_id = ? AND conversation_id IN (SELECT id FROM conversations WHERE channel_id = ?)", f.tenantID, channelID).Scan(&messages).Error; err != nil {
		t.Fatal(err)
	}
	return
}

func (f *sfFixture) runSync(t *testing.T, id string) error {
	t.Helper()
	return NewSyncEngine(f.cfg).SyncReservedChannel(context.Background(), f.channel(t, id), f.mustReserve(t, f.tenantID, id))
}

func (f *sfFixture) setCheckpoint(t *testing.T, id string, at time.Time) {
	t.Helper()
	if err := db.DB.Exec("UPDATE channels SET last_sync_at = ? WHERE id = ?", at.UTC(), id).Error; err != nil {
		t.Fatal(err)
	}
}

// 130 eligible conversations over three Graph pages are all stored (more than the old 100 cap),
// the run succeeds, the checkpoint advances and after-sync fires once.
func TestFacebookSyncStoresEveryConversationBeyondTheOldLimit(t *testing.T) {
	g := &fbGraph{total: 130, pageSize: 50}
	f, chID := fbEngFixture(t, g)
	before := time.Now().Add(-3 * 24 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, before)

	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	convs, msgs := f.storedCounts(t, chID)
	if convs != 130 || msgs != 130 {
		t.Fatalf("stored %d conversations / %d messages, want 130 / 130", convs, msgs)
	}
	st := f.statusOf(t, chID)
	if st.Status != "success" || st.LastSyncAt == nil || !st.LastSyncAt.After(before) {
		t.Fatalf("status %q, checkpoint %v (was %v)", st.Status, st.LastSyncAt, before)
	}
	if g.convReqs != 3 || g.messageReqs != 130 || len(g.otherHosts) != 0 {
		t.Fatalf("graph traffic: %d conversation pages, %d message fetches, other hosts %v", g.convReqs, g.messageReqs, g.otherHosts)
	}
	if f.triggerCount() != 1 {
		t.Fatalf("after-sync fired %d times, want 1", f.triggerCount())
	}
}

// A later-page failure leaves the previous success checkpoint untouched, stores none of the
// partial rows as a success, records a non-success status and fires no after-sync job. The
// retry completes the window without duplicates and only then advances the checkpoint.
func TestFacebookSyncPageFailureKeepsCheckpointAndRetryCompletesIdempotently(t *testing.T) {
	for _, mode := range []string{"graph-error", "transport"} {
		t.Run(mode, func(t *testing.T) {
			g := &fbGraph{total: 130, pageSize: 50, failPage: 2, failMode: mode}
			f, chID := fbEngFixture(t, g)
			previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
			f.setCheckpoint(t, chID, previous)

			if err := f.runSync(t, chID); err == nil {
				t.Fatal("a run whose page 2 failed must not report success")
			}
			st := f.statusOf(t, chID)
			if st.Status != "error" || st.LastSyncAt == nil || !st.LastSyncAt.Equal(previous) {
				t.Fatalf("after failure: status %q, checkpoint %v, want error and unchanged %v", st.Status, st.LastSyncAt, previous)
			}
			if convs, msgs := f.storedCounts(t, chID); convs != 0 || msgs != 0 {
				t.Fatalf("partial window was processed: %d conversations / %d messages", convs, msgs)
			}
			if f.triggerCount() != 0 {
				t.Fatalf("after-sync fired %d times after a failed window", f.triggerCount())
			}
			for _, banned := range []string{fbEngToken, "access_token", "graph.facebook.com"} {
				if strings.Contains(st.Error, banned) {
					t.Fatalf("stored error leaks %q: %s", banned, st.Error)
				}
			}
			if g.convReqs != 2 || g.messageReqs != 0 {
				t.Fatalf("graph traffic after failure: %d conversation pages, %d message fetches", g.convReqs, g.messageReqs)
			}

			// Retry against the same previous checkpoint: the whole window, once.
			g.failPage = 0
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("retry: %v", err)
			}
			convs, msgs := f.storedCounts(t, chID)
			if convs != 130 || msgs != 130 {
				t.Fatalf("after retry: %d conversations / %d messages, want 130 / 130", convs, msgs)
			}
			st = f.statusOf(t, chID)
			if st.Status != "success" || st.LastSyncAt == nil || !st.LastSyncAt.After(previous) {
				t.Fatalf("after retry: status %q, checkpoint %v", st.Status, st.LastSyncAt)
			}
			if f.triggerCount() != 1 {
				t.Fatalf("after-sync fired %d times, want 1 (retry only)", f.triggerCount())
			}

			// Replaying the same window is idempotent (upserts, no duplicate rows).
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("replay: %v", err)
			}
			if c2, m2 := f.storedCounts(t, chID); c2 != 130 || m2 != 130 {
				t.Fatalf("replay changed counts to %d / %d", c2, m2)
			}
		})
	}
}

// A malformed page (not a transport failure) is incomplete coverage: error status, checkpoint kept.
func TestFacebookSyncMalformedPageIsNotASuccess(t *testing.T) {
	g := &fbGraph{total: 60, pageSize: 50}
	f, chID := fbEngFixture(t, g)
	previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, previous)
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"paging":{}}`)), Request: r}, nil
	})
	if err := f.runSync(t, chID); err == nil {
		t.Fatal("a page without a data array must not count as a complete window")
	}
	if st := f.statusOf(t, chID); st.Status != "error" || st.LastSyncAt == nil || !st.LastSyncAt.Equal(previous) {
		t.Fatalf("status %q checkpoint %v, want error with unchanged %v", st.Status, st.LastSyncAt, previous)
	}
	if f.triggerCount() != 0 {
		t.Fatal("after-sync fired for an incomplete window")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

// Facebook (R022) and Pancake (R023) ask for exhaustive coverage; Zalo keeps the shared limit
// of 100.
func TestConversationFetchLimitIsExhaustiveForFacebookAndPancake(t *testing.T) {
	f := setupSFFixture(t)
	limits := map[string]int{}
	var mu sync.Mutex
	var current string
	newSyncAdapter = func(string, []byte) (channels.ChannelAdapter, error) {
		return &limitProbeAdapter{record: func(limit int) {
			mu.Lock()
			limits[current] = limit
			mu.Unlock()
		}}, nil
	}
	for _, channelType := range []string{"facebook", "pancake", "zalo_oa"} {
		f.setType(t, f.chA, channelType)
		current = channelType
		if err := f.runSync(t, f.chA); err != nil {
			t.Fatalf("%s sync: %v", channelType, err)
		}
	}
	want := map[string]int{"facebook": 0, "pancake": 0, "zalo_oa": 100}
	for channelType, w := range want {
		if got, ok := limits[channelType]; !ok || got != w {
			t.Fatalf("%s received limit %d (seen %v), want %d", channelType, got, ok, w)
		}
	}
}

type limitProbeAdapter struct{ record func(limit int) }

func (a *limitProbeAdapter) FetchRecentConversations(_ context.Context, _ time.Time, limit int) ([]channels.SyncedConversation, error) {
	a.record(limit)
	return nil, nil
}
func (a *limitProbeAdapter) FetchMessages(context.Context, string, time.Time) ([]channels.SyncedMessage, error) {
	return nil, nil
}
func (a *limitProbeAdapter) HealthCheck(context.Context) error { return nil }
