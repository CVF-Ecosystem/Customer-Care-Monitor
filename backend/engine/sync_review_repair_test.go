package engine

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

func TestChannelSyncDueThrottlesFailedAttempts(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	oldSuccess := now.Add(-48 * time.Hour)
	recentAttempt := now.Add(-5 * time.Minute)

	cases := []struct {
		name string
		ch   models.Channel
		want bool
	}{
		{"never synced", models.Channel{}, true},
		{"recent success", models.Channel{LastSyncStatus: "success", LastSyncAt: &recentAttempt, UpdatedAt: recentAttempt}, false},
		{"old success", models.Channel{LastSyncStatus: "success", LastSyncAt: &oldSuccess, UpdatedAt: oldSuccess}, true},
		{"recent partial after old success", models.Channel{LastSyncStatus: "partial", LastSyncAt: &oldSuccess, UpdatedAt: recentAttempt}, false},
		{"recent error never succeeded", models.Channel{LastSyncStatus: "error", UpdatedAt: recentAttempt}, false},
		{"old error", models.Channel{LastSyncStatus: "error", LastSyncAt: &oldSuccess, UpdatedAt: now.Add(-20 * time.Minute)}, true},
	}
	for _, tc := range cases {
		if got := channelSyncDue(tc.ch, 15, now); got != tc.want {
			t.Errorf("%s: channelSyncDue = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestS0VietnameseCorpusIsComplete(t *testing.T) {
	raw, err := os.ReadFile("testdata/s0_vietnamese_intervention_corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		CorpusVersion string `json:"corpus_version"`
		AuthorizedUse string `json:"authorized_use"`
		Cases         []struct {
			ID       string `json:"id"`
			Risk     string `json:"risk"`
			Expected string `json:"expected"`
			Reason   string `json:"reason"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.CorpusVersion != "s0-vi-intervention-v1" || corpus.AuthorizedUse != "synthetic_only" {
		t.Fatalf("unexpected corpus identity %q/%q", corpus.CorpusVersion, corpus.AuthorizedUse)
	}
	required := []string{"empty-chat", "missing-history", "duplicate-replay", "explicit-complaint", "negation", "sarcasm",
		"no-diacritics", "code-switch", "pii", "prompt-injection", "refund-claim", "stale-source"}
	seen := map[string]bool{}
	for _, c := range corpus.Cases {
		if seen[c.ID] {
			t.Fatalf("duplicate case id %q", c.ID)
		}
		seen[c.ID] = true
		if c.Expected != "ELIGIBLE" && c.Expected != "WAIT_DATA" {
			t.Errorf("case %s has unknown expected %q", c.ID, c.Expected)
		}
		if c.Reason == "" || c.Risk == "" {
			t.Errorf("case %s lacks reason or risk", c.ID)
		}
	}
	for _, id := range required {
		if !seen[id] {
			t.Errorf("required case %q missing", id)
		}
	}
	if len(corpus.Cases) != len(required) {
		t.Errorf("corpus has %d cases, want %d", len(corpus.Cases), len(required))
	}
}
