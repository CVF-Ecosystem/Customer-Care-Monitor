package engine

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSyncProgressFinalStatus(t *testing.T) {
	p := syncProgress{conversationsFetched: 2, conversationsSynced: 2, messagesSynced: 4}
	if got := p.finalStatus(); got != "success" {
		t.Fatalf("clean sync status = %q, want success", got)
	}

	p.fail("message", "m-2", errors.New("write failed"))
	if got := p.finalStatus(); got != "partial" {
		t.Fatalf("failed item status = %q, want partial", got)
	}
	message := p.errorMessage()
	if !strings.Contains(message, "message m-2: write failed") || !strings.Contains(message, "1 lỗi") {
		t.Fatalf("partial summary lacks actionable failure: %q", message)
	}
}

func TestSyncProgressErrorMessageIsBounded(t *testing.T) {
	p := syncProgress{conversationsFetched: 12}
	for i := 0; i < 12; i++ {
		p.fail("message", string(rune('a'+i)), errors.New(strings.Repeat("lỗi", 300)))
	}
	message := p.errorMessage()
	if !strings.Contains(message, "và 2 lỗi khác") {
		t.Fatalf("bounded summary should report hidden failures: %q", message)
	}
	if got := len([]rune(message)); got > maxSyncErrorMessageRunes+1 {
		t.Fatalf("summary has %d runes, want at most %d plus ellipsis", got, maxSyncErrorMessageRunes)
	}
}

func TestBuildSyncStatusUpdatesOnlyAdvancesSuccessfulCheckpoint(t *testing.T) {
	now := time.Date(2026, 9, 27, 7, 30, 0, 0, time.UTC)
	for _, status := range []string{"partial", "error"} {
		updates := buildSyncStatusUpdates(status, "failure", now)
		if _, ok := updates["last_sync_at"]; ok {
			t.Fatalf("%s status must not advance last_sync_at", status)
		}
	}

	updates := buildSyncStatusUpdates("success", "", now)
	if _, ok := updates["last_sync_at"]; !ok {
		t.Fatal("success status must advance last_sync_at")
	}
}
