package engine

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// CCMAI-RUNTIME-030 (F02-D): the real PancakeAdapter message traversal, driven by the synthetic
// pages.fm transport of sync_pancake_coverage_test.go (nothing leaves the machine), feeds the real
// SyncEngine on a disposable MySQL. Assertions read stored message IDs, the checkpoint, the
// recorded after-sync seam and the request log; they cover application semantics for the
// synthetic contract only, never live Pancake behaviour or runtime AI governance.

const pmEngBodyMarker = "R030-BODY-MARKER"

func pmEngRow(id string, at time.Time) string {
	return fmt.Sprintf(`{"id":%q,"type":"INBOX","original_message":"tin %s","from":{"id":"c","name":"Khach"},"inserted_at":%q,"attachments":[]}`,
		id, id, at.UTC().Format(pcEngStamp))
}

func pmEngPage(rows ...string) string {
	return `{"success":true,"messages":[` + strings.Join(rows, ",") + `]}`
}

func (f *sfFixture) storedMessageIDs(t *testing.T, channelID, conversationExternalID string) []string {
	t.Helper()
	var ids []string
	err := db.DB.Raw(`SELECT m.external_message_id FROM messages m JOIN conversations c ON c.id = m.conversation_id
		WHERE m.tenant_id = ? AND c.tenant_id = ? AND c.channel_id = ? AND c.external_conversation_id = ?`,
		f.tenantID, f.tenantID, channelID, conversationExternalID).Scan(&ids).Error
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(ids)
	return ids
}

// No duplicate rows: every stored (conversation, external message) pair is unique.
func (f *sfFixture) assertNoDuplicateMessages(t *testing.T) {
	t.Helper()
	var total, distinct int64
	if err := db.DB.Raw("SELECT COUNT(*), COUNT(DISTINCT conversation_id, external_message_id) FROM messages WHERE tenant_id = ?", f.tenantID).Row().Scan(&total, &distinct); err != nil {
		t.Fatal(err)
	}
	if total != distinct {
		t.Fatalf("duplicate message rows: %d rows, %d distinct", total, distinct)
	}
}

func (f *sfFixture) completedActivityCount(t *testing.T) int64 {
	t.Helper()
	var n int64
	if err := db.DB.Raw("SELECT COUNT(*) FROM activity_logs WHERE tenant_id = ? AND action = 'sync.completed'", f.tenantID).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (g *pcFake) messageLog() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.msgLog, ",")
}

// F02D-08: a complete multi-page traversal stores exactly the eligible IDs (newest-first pages,
// an overlapping row, the since boundary, ineligible older rows), the checkpoint advances, after-sync
// fires once, other channel/tenant stay empty, and a replay of the same window adds nothing.
func TestPancakeSyncStoresExactEligibleMessageIDsAcrossPages(t *testing.T) {
	g := &pcFake{}
	g.add("mc_0001", time.Now().Add(-10*time.Minute))
	g.add("mc_0002", time.Now().Add(-9*time.Minute))
	f, chID := pcEngFixture(t, g)
	previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.setCheckpoint(t, chID, previous)
	since := previous.Add(-time.Hour) // the engine's window start

	at := func(d time.Duration) time.Time { return since.Add(d) }
	g.msgFn = func(conv string, cc int) (int, string) {
		if conv == "mc_0002" {
			if cc == 0 {
				return 200, pmEngPage(pmEngRow("two_b", at(2*time.Hour)), pmEngRow("two_a", at(time.Hour)))
			}
			return 200, pmEngPage()
		}
		switch cc {
		case 0: // newest first, with an ineligible older row
			return 200, pmEngPage(pmEngRow("e4", at(4*time.Hour)), pmEngRow("e3", at(3*time.Hour)), pmEngRow("oldX", at(-2*time.Hour)))
		case 3: // overlaps e3, adds e2 and e1
			return 200, pmEngPage(pmEngRow("e3", at(3*time.Hour)), pmEngRow("e2", at(2*time.Hour)), pmEngRow("e1", at(time.Hour)))
		case 6: // the exact since boundary then another ineligible row
			return 200, pmEngPage(pmEngRow("boundary", since), pmEngRow("oldY", at(-3*time.Hour)))
		case 8:
			return 200, pmEngPage()
		}
		return 500, pmEngBodyMarker
	}

	before := time.Now().Truncate(time.Second)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []string{"boundary", "e1", "e2", "e3", "e4"}
	if got := f.storedMessageIDs(t, chID, "mc_0001"); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("mc_0001 stored IDs = %v, want exactly %v", got, want)
	}
	if got := f.storedMessageIDs(t, chID, "mc_0002"); strings.Join(got, ",") != "two_a,two_b" {
		t.Fatalf("mc_0002 stored IDs = %v, want two_a,two_b", got)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 2 || msgs != 7 {
		t.Fatalf("stored %d conversations / %d messages, want 2 / 7", convs, msgs)
	}
	if log := g.messageLog(); log != "mc_0001@0,mc_0001@3,mc_0001@6,mc_0001@8,mc_0002@0,mc_0002@2" {
		t.Fatalf("message request log = %s", log)
	}
	st := f.statusOf(t, chID)
	if st.Status != "success" || st.LastSyncAt == nil || st.LastSyncAt.Before(before) {
		t.Fatalf("status %q checkpoint %v (fetch started at or after %v)", st.Status, st.LastSyncAt, before)
	}
	if f.triggerCount() != 1 || f.completedActivityCount(t) != 1 {
		t.Fatalf("after-sync fired %d times / completion activity %d, want 1 / 1", f.triggerCount(), f.completedActivityCount(t))
	}
	if len(g.otherHosts) != 0 {
		t.Fatalf("unexpected network hosts: %v", g.otherHosts)
	}
	// Tenant/channel isolation: nothing was written for the sibling channel or the other tenant.
	var siblings int64
	if err := db.DB.Raw("SELECT COUNT(*) FROM messages WHERE tenant_id = ? OR conversation_id IN (SELECT id FROM conversations WHERE channel_id IN (?, ?))", f.otherTenantID, f.chB, f.chX).Scan(&siblings).Error; err != nil || siblings != 0 {
		t.Fatalf("sibling channel/tenant messages = %d, err %v", siblings, err)
	}

	// Replay over the identical window: no new rows, no duplicates, same IDs.
	f.setCheckpoint(t, chID, previous)
	if err := f.runSync(t, chID); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := f.storedMessageIDs(t, chID, "mc_0001"); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("replay changed mc_0001 IDs to %v", got)
	}
	if convs, msgs := f.storedCounts(t, chID); convs != 2 || msgs != 7 {
		t.Fatalf("replay changed counts to %d / %d", convs, msgs)
	}
	f.assertNoDuplicateMessages(t)
}

// F02D-07: a later-page contract failure or a transport failure after valid messages makes the
// run partial. The failed conversation's diagnostic messages are not stored, the peer conversation
// is, the checkpoint stays, no after-sync analysis is dispatched and no completion is recorded;
// the retry stores everything once and a replay changes nothing (F02D-08 failure -> retry).
func TestPancakeSyncMessageFailureIsPartialAndRetryCompletes(t *testing.T) {
	// Page 2 (position 2) of the failing conversation, per failure mode.
	modes := map[string]func() (int, string){
		"duplicate-only page": func() (int, string) {
			return 200, pmEngPage(pmEngRow("bad_new", time.Now().Add(-7*time.Minute)), pmEngRow("bad_old", time.Now().Add(-time.Hour)))
		},
		"missing messages array": func() (int, string) { return 200, `{"success":true}` },
		"transport failure":      func() (int, string) { return 500, pmEngBodyMarker + " " + pcEngToken },
	}
	for name, page2 := range modes {
		page2 := page2
		t.Run(name, func(t *testing.T) {
			g := &pcFake{}
			g.add("mf_bad", time.Now().Add(-10*time.Minute)) // provider order: the failing one first
			g.add("mf_ok", time.Now().Add(-9*time.Minute))
			f, chID := pcEngFixture(t, g)
			previous := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
			f.setCheckpoint(t, chID, previous)
			recent := func(d time.Duration) time.Time { return time.Now().Add(d) }

			failing := true
			mu := &g.mu
			g.msgFn = func(conv string, cc int) (int, string) {
				mu.Lock()
				fail := failing
				mu.Unlock()
				if conv == "mf_ok" {
					if cc == 0 {
						return 200, pmEngPage(pmEngRow("ok_1", recent(-8*time.Minute)))
					}
					return 200, pmEngPage()
				}
				switch {
				case cc == 0:
					return 200, pmEngPage(pmEngRow("bad_new", recent(-7*time.Minute)), pmEngRow("bad_old", recent(-time.Hour)))
				case cc == 2 && fail:
					return page2()
				case cc == 2:
					return 200, pmEngPage(pmEngRow("bad_older", recent(-2*time.Hour)))
				default:
					return 200, pmEngPage()
				}
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
			if got := f.storedMessageIDs(t, chID, "mf_bad"); len(got) != 0 {
				t.Fatalf("failed conversation stored its diagnostic messages: %v", got)
			}
			if got := f.storedMessageIDs(t, chID, "mf_ok"); strings.Join(got, ",") != "ok_1" {
				t.Fatalf("peer conversation stored %v, want ok_1", got)
			}
			if log := g.messageLog(); log != "mf_bad@0,mf_bad@2,mf_ok@0,mf_ok@1" {
				t.Fatalf("message request log = %s, want the failed conversation to stop after position 2 and the peer to complete", log)
			}
			if len(g.otherHosts) != 0 {
				t.Fatalf("unexpected network hosts: %v", g.otherHosts)
			}
			for _, banned := range []string{pcEngToken, "page_access_token", "pages.fm", pmEngBodyMarker, "bad_new"} {
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
			if got := f.storedMessageIDs(t, chID, "mf_bad"); strings.Join(got, ",") != "bad_new,bad_old,bad_older" {
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
