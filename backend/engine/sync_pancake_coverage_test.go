package engine

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-023 (F02-B): the real PancakeAdapter, driven by a synthetic pages.fm transport
// (installed as http.DefaultTransport; nothing leaves the machine), feeds the real SyncEngine on
// a disposable MySQL. The synthetic API honours since/until and the last_conversation_id cursor,
// pages 60 rows and ends with an empty page. Assertions read stored rows and the checkpoint.

const (
	pcEngPage  = "PCPAGE-R023"
	pcEngToken = "tok-ENG-R023-SECRET"
	pcEngStamp = "2006-01-02T15:04:05.000000"
)

type pcEngConv struct {
	id      string
	updated time.Time
}

type pcFake struct {
	mu         sync.Mutex
	convs      []pcEngConv
	failPage   int    // 1-based conversation page that fails (0 = none)
	failMode   string // "http500" | "malformed"
	convReqs   int
	msgReqs    int
	untils     []string
	otherHosts []string
	onMessage  func(convID string)
}

func (g *pcFake) add(id string, updated time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.convs = append(g.convs, pcEngConv{id: id, updated: updated})
}

func (g *pcFake) RoundTrip(r *http.Request) (*http.Response, error) {
	reply := func(code int, body string) (*http.Response, error) {
		return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}
	if r.URL.Host != "pages.fm" {
		g.mu.Lock()
		g.otherHosts = append(g.otherHosts, r.URL.Host)
		g.mu.Unlock()
		return nil, errors.New("unexpected host in synthetic pancake")
	}
	q := r.URL.Query()
	if q.Get("page_access_token") != pcEngToken {
		return reply(200, `{"success":false,"error_code":102,"message":"Invalid access_token"}`)
	}
	convPath := "/api/public_api/v2/pages/" + pcEngPage + "/conversations"
	msgPrefix := "/api/public_api/v1/pages/" + pcEngPage + "/conversations/"
	switch {
	case r.URL.Path == convPath:
		g.mu.Lock()
		g.convReqs++
		g.untils = append(g.untils, q.Get("until"))
		since, _ := strconv.ParseInt(q.Get("since"), 10, 64)
		until, _ := strconv.ParseInt(q.Get("until"), 10, 64)
		var window []pcEngConv
		for _, c := range g.convs {
			if c.updated.Unix() >= since && c.updated.Unix() <= until {
				window = append(window, c)
			}
		}
		start := 0
		if cur := q.Get("last_conversation_id"); cur != "" {
			start = -1
			for i, c := range window {
				if c.id == cur {
					start = i + 1
				}
			}
		}
		failPage, failMode := g.failPage, g.failMode
		g.mu.Unlock()
		if start < 0 {
			return reply(400, "unknown cursor")
		}
		page := start/60 + 1
		if page == failPage {
			if failMode == "malformed" {
				return reply(200, `{"success":true}`)
			}
			return reply(500, "synthetic server error "+pcEngToken)
		}
		var rows []string
		for i := start; i < start+60 && i < len(window); i++ {
			c := window[i]
			rows = append(rows, fmt.Sprintf(`{"id":%q,"type":"INBOX","updated_at":%q,"from":{"id":"u-%s","name":"Khach"}}`, c.id, c.updated.UTC().Format(pcEngStamp), c.id))
		}
		return reply(200, `{"success":true,"conversations":[`+strings.Join(rows, ",")+`]}`)
	case strings.HasPrefix(r.URL.Path, msgPrefix) && strings.HasSuffix(r.URL.Path, "/messages"):
		conv := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, msgPrefix), "/messages")
		g.mu.Lock()
		g.msgReqs++
		var updated time.Time
		for _, c := range g.convs {
			if c.id == conv {
				updated = c.updated
			}
		}
		hook := g.onMessage
		g.mu.Unlock()
		if hook != nil {
			hook(conv)
		}
		// Oldest first within the batch: an old message (before any since) ends paging.
		return reply(200, fmt.Sprintf(`{"success":true,"messages":[`+
			`{"id":"m_old_%s","type":"INBOX","original_message":"old","from":{"id":"c","name":"Khach"},"inserted_at":"2020-01-01T00:00:00.000000","attachments":[]},`+
			`{"id":"m_%s","type":"INBOX","original_message":"hello","from":{"id":"c","name":"Khach"},"inserted_at":%q,"attachments":[]}]}`,
			conv, conv, updated.Add(-time.Minute).UTC().Format(pcEngStamp)))
	}
	return reply(404, "unscripted synthetic pancake path")
}

func pcEngFixture(t *testing.T, g *pcFake) (*sfFixture, string) {
	t.Helper()
	f := setupSFFixture(t) // chA is a pancake channel
	chID := f.chA
	creds, err := pkg.Encrypt([]byte(fmt.Sprintf(`{"page_id":%q,"page_access_token":%q}`, pcEngPage, pcEngToken)), sfKey)
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

func pcConvs(g *pcFake, prefix string, n int, updated time.Time) {
	for i := 0; i < n; i++ {
		g.add(fmt.Sprintf("%s_%04d", prefix, i), updated)
	}
}

func (f *sfFixture) storedPancakeIDs(t *testing.T, channelID string) []string {
	t.Helper()
	var ids []string
	if err := db.DB.Raw("SELECT external_conversation_id FROM conversations WHERE tenant_id = ? AND channel_id = ?", f.tenantID, channelID).Scan(&ids).Error; err != nil {
		t.Fatal(err)
	}
	sort.Strings(ids)
	return ids
}

// 105 eligible conversations (60 + 45 + empty page) are all stored with their messages, the run
// succeeds, and the checkpoint is the fetch start (not later than the fixed until).
func TestPancakeSyncStoresEveryConversationBeyondTheOldLimit(t *testing.T) {
	g := &pcFake{}
	pcConvs(g, "pc", 105, time.Now().Add(-10*time.Minute))
	f, chID := pcEngFixture(t, g)
	previous := time.Now().Add(-72 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, previous)

	before := time.Now().Truncate(time.Second)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	convs, msgs := f.storedCounts(t, chID)
	if convs != 105 || msgs != 105 {
		t.Fatalf("stored %d conversations / %d messages, want 105 / 105", convs, msgs)
	}
	st := f.statusOf(t, chID)
	if st.Status != "success" || st.LastSyncAt == nil || st.LastSyncAt.Before(before) {
		t.Fatalf("status %q checkpoint %v (fetch started at or after %v)", st.Status, st.LastSyncAt, before)
	}
	until, _ := strconv.ParseInt(g.untils[0], 10, 64)
	if st.LastSyncAt.Unix() > until {
		t.Fatalf("checkpoint %v is later than the fixed until %d", st.LastSyncAt, until)
	}
	if g.convReqs != 3 || g.msgReqs != 105 || len(g.otherHosts) != 0 {
		t.Fatalf("traffic: %d conversation pages, %d message fetches, other hosts %v", g.convReqs, g.msgReqs, g.otherHosts)
	}
	for _, u := range g.untils {
		if u != g.untils[0] {
			t.Fatalf("until changed between pages: %v", g.untils)
		}
	}
	if f.triggerCount() != 1 {
		t.Fatalf("after-sync fired %d times, want 1", f.triggerCount())
	}
}

// A failing or malformed later page leaves the previous checkpoint, stores nothing from the
// partial window and fires no after-sync job; the retry stores the whole window once, and a
// replay changes nothing.
func TestPancakeSyncPageFailureKeepsCheckpointAndRetryCompletesIdempotently(t *testing.T) {
	for _, mode := range []string{"http500", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			g := &pcFake{failPage: 2, failMode: mode}
			pcConvs(g, "rt", 70, time.Now().Add(-10*time.Minute))
			f, chID := pcEngFixture(t, g)
			previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
			f.setCheckpoint(t, chID, previous)

			if err := f.runSync(t, chID); err == nil {
				t.Fatal("a run whose page 2 failed must not report success")
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
			for _, banned := range []string{pcEngToken, "page_access_token", "pages.fm"} {
				if strings.Contains(st.Error, banned) {
					t.Fatalf("stored error leaks %q: %s", banned, st.Error)
				}
			}
			if g.convReqs != 2 || g.msgReqs != 0 {
				t.Fatalf("traffic after failure: %d pages, %d message fetches", g.convReqs, g.msgReqs)
			}

			g.mu.Lock()
			g.failPage = 0
			g.mu.Unlock()
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("retry: %v", err)
			}
			if convs, msgs := f.storedCounts(t, chID); convs != 70 || msgs != 70 {
				t.Fatalf("after retry: %d / %d, want 70 / 70", convs, msgs)
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
			if convs, msgs := f.storedCounts(t, chID); convs != 70 || msgs != 70 {
				t.Fatalf("replay changed counts to %d / %d", convs, msgs)
			}
		})
	}
}

// A long successful run: the checkpoint stays at the fetch start while updated_at records the
// (simulated, two hours later) completion, so a conversation updated during the run is inside
// the next run's window. With a completion-time checkpoint the next window would start an hour
// after that update and miss it.
func TestPancakeCheckpointIsBoundedByFetchStart(t *testing.T) {
	g := &pcFake{}
	pcConvs(g, "dl", 3, time.Now().Add(-10*time.Minute))
	f, chID := pcEngFixture(t, g)
	previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, previous)

	var late sync.Once
	var lateAt time.Time
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
	g.onMessage = func(string) {
		late.Do(func() {
			// Processing has started: from now on the run "takes" two more hours, and the
			// provider receives an update the fixed until of this run excludes.
			time.Sleep(1100 * time.Millisecond)
			lateAt = time.Now()
			g.add("dl_late", lateAt)
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
	until, _ := strconv.ParseInt(g.untils[0], 10, 64)
	if st.Status != "success" || st.LastSyncAt == nil || st.LastSyncAt.Before(before) || st.LastSyncAt.Unix() > until {
		t.Fatalf("checkpoint %v, want within [fetch start %v, until %d]", st.LastSyncAt, before, until)
	}
	updatedAt, err := time.ParseInLocation("2006-01-02 15:04:05", st.UpdatedAt[:19], time.UTC)
	if err != nil {
		t.Fatalf("parse updated_at %q: %v", st.UpdatedAt, err)
	}
	if updatedAt.Sub(*st.LastSyncAt) < 119*time.Minute {
		t.Fatalf("updated_at %v should record the (simulated) completion, checkpoint %v", updatedAt, st.LastSyncAt)
	}
	if ids := f.storedPancakeIDs(t, chID); len(ids) != 3 {
		t.Fatalf("first run stored %v; the late update is after its until", ids)
	}

	mu.Lock()
	elapsed = false
	mu.Unlock()
	g.onMessage = nil
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("second run: %v", err)
	}
	ids := f.storedPancakeIDs(t, chID)
	if len(ids) != 4 || ids[3] != "dl_late" {
		t.Fatalf("second run stored %v; the conversation updated during the first run (%v) was missed", ids, lateAt)
	}
}

// Facebook and Pancake ask for exhaustive coverage, Zalo keeps 100; only Pancake bounds its
// success checkpoint by the fetch start.
func TestPancakeLimitAndCheckpointSelection(t *testing.T) {
	for channelType, want := range map[string]struct {
		limit   int
		bounded bool
	}{"facebook": {0, false}, "pancake": {0, true}, "zalo_oa": {100, false}, "unknown": {100, false}} {
		if got := conversationFetchLimit(channelType); got != want.limit {
			t.Errorf("%s limit = %d, want %d", channelType, got, want.limit)
		}
		if got := boundsCheckpointByFetchStart(channelType); got != want.bounded {
			t.Errorf("%s bounded checkpoint = %v, want %v", channelType, got, want.bounded)
		}
	}
}
