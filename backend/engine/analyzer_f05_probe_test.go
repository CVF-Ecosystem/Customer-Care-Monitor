package engine

import (
	"context"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
)

// CCMAI-RUNTIME-027 old-source probes. They use only the public entry points and the R025 fixture
// helpers, so the same file compiles against the pre-F05 source (base ebfa05b) and fails there
// (full+cap exclusion, UTC date parsing/strict lower bound, silent invalid dates, checkpoint
// contamination by an unlimited explicit run), and passes on the repaired source.

func probeAnalyzer(p *incProvider) *Analyzer {
	an := NewAnalyzer(&config.Config{})
	an.providerOverride = p
	return an
}

func TestF05ProbeFullWithCapReEvaluatesEvaluatedConversation(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	x := f.addConv(t, f.tenantID, f.channelID, "x", []time.Time{f.clock.Add(-2 * time.Hour)})
	p := &incProvider{}
	f.mustRun(t, p)
	run, err := probeAnalyzer(p).RunJobFullWithParams(context.Background(), f.job(t), "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := evaluatedIn(t, run.ID); !eqIDs(got, []string{x}) {
		t.Fatalf("full rerun with a cap must re-evaluate the evaluated conversation, got %v", got)
	}
}

func TestF05ProbeVietnamDayLowerBoundIsInclusive(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	at := func(s string) time.Time { v, _ := time.Parse(time.RFC3339, s); return v }
	edge := f.addConv(t, f.tenantID, f.channelID, "edge", []time.Time{at("2026-10-01T17:00:00Z")}) // VN 2026-10-02 00:00:00
	f.addConv(t, f.tenantID, f.channelID, "before", []time.Time{at("2026-10-01T16:59:59Z")})       // VN 2026-10-01 23:59:59
	run, err := probeAnalyzer(&incProvider{}).RunJobFullWithParams(context.Background(), f.job(t), "2026-10-02", "2026-10-02", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := evaluatedIn(t, run.ID); !eqIDs(got, []string{edge}) {
		t.Fatalf("the Vietnam day 2026-10-02 must select exactly its midnight conversation, got %v", got)
	}
}

func TestF05ProbeInvalidDatesAreRejectedWithoutARun(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	before := f.count(t, "job_runs")
	p := &incProvider{}
	if _, err := probeAnalyzer(p).RunJobFullWithParams(context.Background(), f.job(t), "not-a-date", "", 0); err == nil {
		t.Fatal("an invalid date must be an error, not silently ignored")
	}
	if f.count(t, "job_runs") != before || p.callCount() != 0 {
		t.Fatal("an invalid date must create no run and call no provider")
	}
}

func TestF05ProbeUnlimitedExplicitRunKeepsTheCheckpoint(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "a", []time.Time{f.clock.Add(-time.Hour)})
	sentinel := f.clock.Add(-48 * time.Hour).UTC().Truncate(time.Second)
	f.exec(t, "UPDATE jobs SET last_run_at = ? WHERE id = ?", sentinel, f.jobID)
	if _, err := probeAnalyzer(&incProvider{}).RunJobUnanalyzed(context.Background(), f.job(t), 0); err != nil {
		t.Fatal(err)
	}
	if cp := f.checkpoint(t); cp == nil || !cp.Equal(sentinel) {
		t.Fatalf("an unlimited explicit run moved the ordinary checkpoint to %v", cp)
	}
}
