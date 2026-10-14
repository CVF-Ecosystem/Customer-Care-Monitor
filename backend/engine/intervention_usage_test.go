package engine

import (
	"errors"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
)

func TestR094UsagePresenceNeverTurnsUnknownIntoZero(t *testing.T) {
	zero := int64(0)
	seven := int64(7)
	cases := []struct {
		name          string
		presence      *ai.AIUsagePresence
		state         string
		input, output bool
	}{
		{name: "missing", state: "UNKNOWN"},
		{name: "one count", presence: &ai.AIUsagePresence{InputTokens: ai.AIUsageCount{Status: ai.AIUsagePresenceKnown, Value: &seven}, OutputTokens: ai.AIUsageCount{Status: ai.AIUsagePresenceAbsent}}, state: "PARTIAL", input: true},
		{name: "known zero", presence: &ai.AIUsagePresence{InputTokens: ai.AIUsageCount{Status: ai.AIUsagePresenceKnown, Value: &zero}, OutputTokens: ai.AIUsageCount{Status: ai.AIUsagePresenceKnown, Value: &seven}, Complete: true}, state: "KNOWN", input: true, output: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := usageFromResponse(ai.AIResponse{UsagePresence: tc.presence})
			if u.state != tc.state || (u.input != nil) != tc.input || (u.output != nil) != tc.output {
				t.Fatalf("state=%s input=%v output=%v", u.state, u.input, u.output)
			}
		})
	}
}

func TestR094PublicationErrorWhitelist(t *testing.T) {
	if safeInterventionPublicationCode(errors.New("driver detail including credential")) != "PERSIST_FAILED" {
		t.Fatal("driver error leaked")
	}
	if safeInterventionPublicationCode(errors.New("STALE_SOURCE")) != "STALE_SOURCE" {
		t.Fatal("stale code lost")
	}
}
