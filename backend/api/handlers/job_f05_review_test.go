package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
)

// Independent R027 review probes: SPEC admission contract, synthetic dispatch only.
func TestF05ReviewAdmissionMatchesSpec(t *testing.T) {
	cases := []struct {
		name, query string
		status      int
	}{
		{"zero is unlimited", "mode=unanalyzed&limit=0", http.StatusAccepted},
		{"empty is unlimited", "mode=unanalyzed&limit=", http.StatusAccepted},
		{"full conflict", "mode=unanalyzed&full=true", http.StatusBadRequest},
		{"invalid full", "mode=conditional&full=maybe", http.StatusBadRequest},
		{"signed plus cap", "mode=conditional&limit=%2B2", http.StatusBadRequest},
		{"valid false", "mode=since_last&full=false", http.StatusAccepted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db.Close()
			f := setupJobDispatchFixture(t)
			rec := f.callTrigger(f.tenantID, tc.query)
			if rec.Code != tc.status {
				t.Errorf("SPEC expects %d; got %d %s (config loads=%d, worker launches=%d)", tc.status, rec.Code, rec.Body.String(), f.cfgLoads, f.triggerLaunches)
			}
			if tc.status == http.StatusBadRequest {
				if !strings.Contains(rec.Body.String(), "invalid_run_parameters") || f.cfgLoads != 0 || f.triggerLaunches != 0 || f.jobRunCount(t) != 0 {
					t.Errorf("invalid input reached dispatch: response=%s config=%d worker=%d runs=%d", rec.Body.String(), f.cfgLoads, f.triggerLaunches, f.jobRunCount(t))
				}
			} else if f.triggerParams.maxConv != 0 || f.triggerLaunches != 1 {
				t.Errorf("unlimited request should dispatch cap 0 once: %+v launches=%d", f.triggerParams, f.triggerLaunches)
			}
		})
	}
}
