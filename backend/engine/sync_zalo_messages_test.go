package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

// CCMAI-RUNTIME-032 (F02-F): the real ZaloOAAdapter message traversal, driven by the synthetic
// openapi.zalo.me transport of sync_zalo_coverage_test.go (installed as http.DefaultTransport;
// nothing leaves the machine), feeds the real SyncEngine on a disposable MySQL. Assertions read
// stored message rows, the checkpoint, credentials, the recorded after-sync seam and the request
// log; they cover application semantics for the synthetic contract only, never live Zalo
// behaviour or runtime AI governance. sync_files stays off, so no media is fetched.

const zmEngMarker = "R032-ENG-BODY-MARKER"

func zmEngRow(id string, src int, ms int64, extra string) string {
	return fmt.Sprintf(`{"message_id":%q,"src":%d,"time":%d,"type":"text","message":"tin %s","from_display_name":"Khach"%s}`, id, src, ms, id, extra)
}

// userLog returns the recorded "user@offset" requests of one conversation, in order.
func (g *zlFake) userLog(user string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	for _, e := range g.msgLog {
		if strings.HasPrefix(e, user+"@") {
			out = append(out, strings.TrimPrefix(e, user+"@"))
		}
	}
	return strings.Join(out, ",")
}

func (f *sfFixture) storedMessageField(t *testing.T, channelID, convExternalID, messageID, column string) string {
	t.Helper()
	var v string
	err := db.DB.Raw("SELECT m."+column+" FROM messages m JOIN conversations c ON c.id = m.conversation_id WHERE m.tenant_id = ? AND c.tenant_id = ? AND c.channel_id = ? AND c.external_conversation_id = ? AND m.external_message_id = ?",
		f.tenantID, f.tenantID, channelID, convExternalID, messageID).Scan(&v).Error
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// F02F-07: a conversation whose history spans many pages (more than 130 messages, a short last
// page, history far older than the sync window, large exact string IDs, attachment kinds) is
// stored exactly once with correct mappings; a peer conversation is stored too; the success
// checkpoint is the fetch start and after-sync fires once; sibling channels/tenants stay empty; a
// replay adds nothing and a later fetch with changed content updates in place.
func TestZaloSyncStoresFullMessageHistoryAcrossPages(t *testing.T) {
	g := &zlFake{}
	zlConvs(g, "zm_big", 1, time.Now().Add(-10*time.Minute))
	zlConvs(g, "zm_peer", 1, time.Now().Add(-9*time.Minute))
	big, peer := "zm_big_0000", "zm_peer_0000"
	old := time.Now().Add(-400 * 24 * time.Hour).UnixMilli() // far before any sync window
	recent := time.Now().Add(-time.Hour).UnixMilli()
	var rows []string
	var want []string
	for i := 0; i < 140; i++ {
		id := fmt.Sprintf("b%03d", i)
		ms := old + int64(i)
		if i >= 100 {
			ms = recent + int64(i)
		}
		rows = append(rows, zmEngRow(id, i%2, ms, ""))
		want = append(want, id)
	}
	rows = append(rows,
		zmEngRow("9007199254740993123", 1, old-1, ""), // beyond float64 precision
		`{"message_id":"photo1","src":0,"time":`+fmt.Sprint(recent+500)+`,"type":"photo","message":"","url":"https://example.invalid/p.jpg"}`,
		`{"message_id":"file1","src":1,"time":`+fmt.Sprint(recent+501)+`,"type":"file","links":[{"url":"https://example.invalid/f.pdf","name":"bao.pdf"}]}`,
	)
	want = append(want, "9007199254740993123", "photo1", "file1")
	sort.Strings(want)
	g.msgRows = map[string][]string{
		big:  rows,
		peer: {zmEngRow("p1", 1, recent, ""), zmEngRow("p2", 1, recent+1, ""), zmEngRow("p3", 0, recent+2, "")},
	}
	f, chID := zlEngFixture(t, g)
	previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, previous)

	before := time.Now().Truncate(time.Second)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got := f.storedMessageIDs(t, chID, big); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s stored %d IDs, want exactly the %d history IDs\n got %v\nwant %v", big, len(got), len(want), got, want)
	}
	if got := f.storedMessageIDs(t, chID, peer); strings.Join(got, ",") != "p1,p2,p3" {
		t.Fatalf("%s stored %v", peer, got)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 2 || msgs != 146 {
		t.Fatalf("stored %d conversations / %d messages, want 2 / 146", convs, msgs)
	}
	var wantOffsets []string
	for off := 0; off < 143; off += 10 {
		wantOffsets = append(wantOffsets, fmt.Sprint(off))
	}
	wantOffsets = append(wantOffsets, "143") // the explicit empty page, at the physical row count
	if got := g.userLog(big); got != strings.Join(wantOffsets, ",") {
		t.Fatalf("%s request offsets %s, want %s", big, got, strings.Join(wantOffsets, ","))
	}
	if got := g.userLog(peer); got != "0,3" {
		t.Fatalf("%s request offsets %s, want 0,3", peer, got)
	}

	// Mapping and attachments through the real store.
	if got := f.storedMessageField(t, chID, big, "photo1", "sender_type"); got != "agent" {
		t.Fatalf("photo1 sender_type %q", got)
	}
	if got := f.storedMessageField(t, chID, big, "photo1", "content_type"); got != "photo" {
		t.Fatalf("photo1 content_type %q", got)
	}
	// MySQL normalizes stored JSON text, so compare decoded attachments, not the raw string.
	wantAttachments := map[string][]map[string]string{
		"photo1": {{"type": "photo", "url": "https://example.invalid/p.jpg", "name": "photo-photo1"}},
		"file1":  {{"type": "file", "url": "https://example.invalid/f.pdf", "name": "bao.pdf"}},
	}
	for id, wantAtt := range wantAttachments {
		var got []map[string]string
		if err := json.Unmarshal([]byte(f.storedMessageField(t, chID, big, id, "attachments")), &got); err != nil || fmt.Sprint(got) != fmt.Sprint(wantAtt) {
			t.Fatalf("%s attachments %v (err %v), want %v", id, got, err, wantAtt)
		}
	}
	if got := f.storedMessageField(t, chID, big, "b000", "content"); got != "tin b000" {
		t.Fatalf("old message content %q: history before the sync window must be stored", got)
	}
	if got := f.storedMessageField(t, chID, big, "b000", "sender_type"); got != "agent" {
		t.Fatalf("b000 (src 0) sender_type %q", got)
	}
	if got := f.storedMessageField(t, chID, big, "b001", "sender_name"); got != "Khach" {
		t.Fatalf("b001 sender_name %q", got)
	}
	var stored models.Message
	if err := db.DB.Where("tenant_id = ? AND external_message_id = ?", f.tenantID, "b000").Take(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if d := stored.SentAt.UnixMilli() - old; d < -1000 || d > 1000 { // stored at second precision
		t.Fatalf("b000 sent_at %v, want the source time %d ms (within the stored second)", stored.SentAt, old)
	}

	st := f.statusOf(t, chID)
	if st.Status != "success" || st.LastSyncAt == nil || st.LastSyncAt.Before(before) || !st.LastSyncAt.After(previous) {
		t.Fatalf("status %q checkpoint %v (previous %v, fetch started at or after %v)", st.Status, st.LastSyncAt, previous, before)
	}
	if f.triggerCount() != 1 || f.completedActivityCount(t) != 1 {
		t.Fatalf("after-sync fired %d times / completion activity %d, want 1 / 1", f.triggerCount(), f.completedActivityCount(t))
	}
	if len(g.otherHosts) != 0 || g.refreshReqs != 0 {
		t.Fatalf("unexpected hosts %v or %d refreshes", g.otherHosts, g.refreshReqs)
	}
	var siblings int64
	if err := db.DB.Raw("SELECT COUNT(*) FROM messages WHERE tenant_id = ? OR conversation_id IN (SELECT id FROM conversations WHERE channel_id IN (?, ?))", f.otherTenantID, f.chB, f.chX).Scan(&siblings).Error; err != nil || siblings != 0 {
		t.Fatalf("sibling channel/tenant messages = %d, err %v", siblings, err)
	}

	// Replay over the same history adds nothing.
	f.setCheckpoint(t, chID, previous)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 2 || msgs != 146 {
		t.Fatalf("replay changed counts to %d / %d", convs, msgs)
	}
	f.assertNoDuplicateMessages(t)

	// A later successful fetch with changed content updates the stored row in place.
	g.mu.Lock()
	g.msgRows[big][0] = zmEngRow("b000", 0, old, "") // unchanged
	g.msgRows[big][1] = strings.Replace(g.msgRows[big][1], "tin b001", "tin b001 edited", 1)
	g.mu.Unlock()
	f.setCheckpoint(t, chID, previous)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("changed replay: %v", err)
	}
	if got := f.storedMessageField(t, chID, big, "b001", "content"); got != "tin b001 edited" {
		t.Fatalf("changed content not applied by the replay: %q", got)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 2 || msgs != 146 {
		t.Fatalf("changed replay changed counts to %d / %d", convs, msgs)
	}
	f.assertNoDuplicateMessages(t)
}

// F02F-08: after valid rows, a later-page malformed row, an actual connection error and (separately
// labelled) an HTTP 500 each make the run partial. The failed conversation's diagnostic first page
// is not stored, the peer conversation is, the checkpoint stays, no after-sync analysis runs and no
// completion is recorded; the retry stores everything once and a replay changes nothing.
func TestZaloSyncMessageFailureIsPartialAndRetryCompletes(t *testing.T) {
	type mode struct {
		fail func() (int, string, error)
	}
	modes := map[string]mode{
		"late malformed page": {func() (int, string, error) {
			return 200, `{"error":0,"data":[{"message_id":"bad_x","src":9,"time":1790000000000}]}`, nil
		}},
		"actual connection error": {func() (int, string, error) {
			return 0, "", errors.New("connection reset by peer " + zlEngAccess)
		}},
		"HTTP 500 (not a connection error)": {func() (int, string, error) {
			return 500, "synthetic server error " + zlEngAccess + " " + zmEngMarker, nil
		}},
	}
	for name, m := range modes {
		m := m
		t.Run(name, func(t *testing.T) {
			g := &zlFake{}
			zlConvs(g, "zm_bad", 1, time.Now().Add(-10*time.Minute))
			zlConvs(g, "zm_ok", 1, time.Now().Add(-9*time.Minute))
			bad, ok := "zm_bad_0000", "zm_ok_0000"
			recent := time.Now().Add(-time.Hour).UnixMilli()
			var badRows, wantBad []string
			for i := 0; i < 25; i++ {
				id := fmt.Sprintf("x%02d", i)
				badRows = append(badRows, zmEngRow(id, 1, recent+int64(i), ""))
				wantBad = append(wantBad, id)
			}
			g.msgRows = map[string][]string{bad: badRows, ok: {zmEngRow("ok1", 1, recent, "")}}
			var failing atomic.Bool
			failing.Store(true)
			g.msgFail = func(user string, offset int) (int, string, error) {
				if user == bad && offset == 10 && failing.Load() {
					return m.fail()
				}
				return 0, "", nil
			}
			f, chID := zlEngFixture(t, g)
			previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
			f.setCheckpoint(t, chID, previous)

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
			if got := f.storedMessageIDs(t, chID, bad); len(got) != 0 {
				t.Fatalf("failed conversation stored its diagnostic messages: %v", got)
			}
			if got := f.storedMessageIDs(t, chID, ok); strings.Join(got, ",") != "ok1" {
				t.Fatalf("peer conversation stored %v, want ok1", got)
			}
			if got := g.userLog(bad); got != "0,10" {
				t.Fatalf("failed conversation requests %s, want 0,10 and no request after the failure", got)
			}
			if got := g.userLog(ok); got != "0,1" {
				t.Fatalf("peer conversation requests %s, want 0,1", got)
			}
			for _, banned := range []string{zlEngAccess, zlEngRefresh, "openapi.zalo.me", "synthetic server error", zmEngMarker, "bad_x", "connection reset"} {
				if strings.Contains(st.Error, banned) {
					t.Fatalf("stored error leaks %q: %s", banned, st.Error)
				}
			}
			if !strings.Contains(st.Error, bad) {
				t.Fatalf("stored error does not identify the failed conversation: %s", st.Error)
			}

			failing.Store(false)
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("retry: %v", err)
			}
			st = f.statusOf(t, chID)
			if st.Status != "success" || st.LastSyncAt == nil || !st.LastSyncAt.After(previous) {
				t.Fatalf("after retry: status %q checkpoint %v", st.Status, st.LastSyncAt)
			}
			if got := f.storedMessageIDs(t, chID, bad); strings.Join(got, ",") != strings.Join(wantBad, ",") {
				t.Fatalf("retry stored %v for the formerly failed conversation", got)
			}
			if f.triggerCount() != 1 || f.completedActivityCount(t) != 1 {
				t.Fatalf("after retry: after-sync %d / completion %d, want 1 / 1", f.triggerCount(), f.completedActivityCount(t))
			}
			convs, msgs := f.storedCounts(t, chID)
			if convs != 2 || msgs != 26 {
				t.Fatalf("after retry counts %d / %d, want 2 / 26", convs, msgs)
			}
			f.setCheckpoint(t, chID, previous)
			if err := f.runSync(t, chID); err != nil {
				t.Fatalf("replay: %v", err)
			}
			if c2, m2 := f.storedCounts(t, chID); c2 != convs || m2 != msgs {
				t.Fatalf("replay changed counts from %d / %d to %d / %d", convs, msgs, c2, m2)
			}
			f.assertNoDuplicateMessages(t)
			if len(g.otherHosts) != 0 {
				t.Fatalf("unexpected hosts %v", g.otherHosts)
			}
		})
	}
}

// F02F-05/08: an expired token in the middle of a message traversal is refreshed once; the rotated
// pair is persisted to the channel's encrypted credentials by the owning run and the same offset
// is retried, so the history completes without a gap or duplicate.
func TestZaloSyncMessageTokenRefreshPersistsAndRetriesSameOffset(t *testing.T) {
	g := &zlFake{msgExpire: "zm_rf_0000@10"}
	zlConvs(g, "zm_rf", 1, time.Now().Add(-10*time.Minute))
	user := "zm_rf_0000"
	recent := time.Now().Add(-time.Hour).UnixMilli()
	var rows, want []string
	for i := 0; i < 23; i++ {
		id := fmt.Sprintf("r%02d", i)
		rows = append(rows, zmEngRow(id, 1, recent+int64(i), ""))
		want = append(want, id)
	}
	g.msgRows = map[string][]string{user: rows}
	f, chID := zlEngFixture(t, g)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync with a message-page refresh: %v", err)
	}
	if got := f.storedMessageIDs(t, chID, user); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("stored %v, want %v", got, want)
	}
	if got := g.userLog(user); got != "0,10,10,20,23" { // the retry reuses offset 10
		t.Fatalf("offsets %s, want 0,10,10,20,23", got)
	}
	if g.refreshReqs != 1 {
		t.Fatalf("%d refreshes, want 1", g.refreshReqs)
	}
	creds := f.storedCreds(t, chID)
	if creds["access_token"] != zlEngNewAcc || creds["refresh_token"] != zlEngNewRef || creds["app_id"] != "app" {
		t.Fatal("rotated token pair was not persisted with the other credential fields intact")
	}
	if st := f.statusOf(t, chID); st.Status != "success" {
		t.Fatalf("status %q", st.Status)
	}
}
