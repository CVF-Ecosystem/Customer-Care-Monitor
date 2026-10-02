package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
)

// CCMAI-RUNTIME-028 old-source probe (admission). It uses only the pre-F06 public surface
// (NewAnalyzer, providerOverride, RunJob) and the R025 fixture, so the same file compiles against the
// base source and fails there behaviorally: a second RunJob of a job whose first run is still
// working was admitted (second run row + second provider stream) instead of refused.

type probeBlockProvider struct {
	mu      sync.Mutex
	calls   int
	entered chan struct{}
	gate    chan struct{}
}

func (p *probeBlockProvider) AnalyzeChat(ctx context.Context, _ string, transcript string) (ai.AIResponse, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	select {
	case p.entered <- struct{}{}:
	default:
	}
	<-p.gate
	return (&incProvider{}).AnalyzeChat(ctx, "", transcript)
}

func (p *probeBlockProvider) AnalyzeChatBatch(ctx context.Context, s string, items []ai.BatchItem) (ai.AIResponse, error) {
	return (&incProvider{}).AnalyzeChatBatch(ctx, s, items)
}

func TestF06ProbeSecondRunOfAnOccupiedJobIsRefused(t *testing.T) {
	f := setupIncFixture(t, false, "qc_analysis")
	f.addConv(t, f.tenantID, f.channelID, "p", []time.Time{f.clock.Add(-3 * time.Hour)})
	f.addConv(t, f.tenantID, f.channelID, "q", []time.Time{f.clock.Add(-2 * time.Hour)})
	first := &probeBlockProvider{entered: make(chan struct{}, 1), gate: make(chan struct{})}
	a := NewAnalyzer(&config.Config{})
	a.providerOverride = first
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = a.RunJob(context.Background(), f.job(t))
	}()
	select {
	case <-first.entered:
	case <-time.After(30 * time.Second):
		t.Fatal("the first run never reached its provider")
	}
	defer func() {
		close(first.gate)
		<-done
	}()
	second := &incProvider{}
	b := NewAnalyzer(&config.Config{})
	b.providerOverride = second
	_, err := b.RunJob(context.Background(), f.job(t))
	var rows int64
	f.exec(t, "SELECT 1")
	rows = f.count(t, "job_runs")
	if err == nil || rows != 1 || second.callCount() != 0 {
		t.Fatalf("a second run of an occupied job must be refused: err=%v, %d run rows, %d provider calls by the second run", err, rows, second.callCount())
	}
}
