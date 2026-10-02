package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

// CCMAI-RUNTIME-027 (F05): the analyzer's mode, candidate bounds, count cap, snapshot scope and
// checkpoint policy are separate decisions. A positive cap no longer implies "unanalyzed only";
// the mode is chosen by the entry point (and, over HTTP, by the strictly validated `mode`), never
// inferred from the cap or from flag combinations.
//
//	mode        candidates                                           cap     snapshot  last_run_at
//	ordinary    R025 source-version eligibility (all conversations)  none    full      scan start (checked)
//	test_run    last_message_at > run start - 7d, not yet evaluated  yes     full      preserved
//	conditional every conversation, optional Vietnam date bounds     yes     full      preserved
//	unanalyzed  not yet evaluated, all dates                         yes     full      preserved
//	since_last  last_message_at > newest evaluated one (else as unanalyzed)  yes  full  preserved
//
// "Evaluated" means a conversation_evaluation row of this tenant and job for the conversation
// (PASS, SKIP and classification evaluations count; finding-only rows, other jobs/tenants and
// orphan snapshots do not).

type analysisMode string

const (
	modeOrdinary   analysisMode = "ordinary"
	modeTestRun    analysisMode = "test_run"
	modeFull       analysisMode = "conditional"
	modeUnanalyzed analysisMode = "unanalyzed"
	modeSinceLast  analysisMode = "since_last"
)

// testRunWindow is the explicit look-back of a test run, applied to a clock captured once.
const testRunWindow = 7 * 24 * time.Hour

// ErrInvalidRunParameters is returned before any side effect (no run, activity, provider or
// configuration resolution) when an entry point receives an invalid cap or mode/date combination.
var ErrInvalidRunParameters = errors.New("invalid analyzer run parameters")

// runPlan is the resolved, validated execution plan.
type runPlan struct {
	mode  analysisMode
	limit int // conversation-count cap applied after eligibility; 0 = no cap
	dates pkg.BusinessRange
}

func (p runPlan) explicit() bool { return p.mode != modeOrdinary }

// ordinaryPlan is the R025 incremental scan. It has no cap.
func ordinaryPlan() runPlan { return runPlan{mode: modeOrdinary} }

func newPlan(mode analysisMode, limit int, dates pkg.BusinessRange) (runPlan, error) {
	if limit < 0 || (mode == modeTestRun && limit == 0) || (mode == modeOrdinary && limit != 0) {
		return runPlan{}, ErrInvalidRunParameters
	}
	if dates.Set() && mode != modeFull {
		return runPlan{}, ErrInvalidRunParameters
	}
	return runPlan{mode: mode, limit: limit, dates: dates}, nil
}

// ---- public entry points (names and signatures are unchanged) ----

// RunJobWithLimit runs a test run: a bounded number of recent, not yet evaluated conversations.
func (a *Analyzer) RunJobWithLimit(ctx context.Context, job models.Job, limit int) (*models.JobRun, error) {
	plan, err := newPlan(modeTestRun, limit, pkg.BusinessRange{})
	if err != nil {
		return nil, err
	}
	return a.execute(ctx, job, plan, nil)
}

// RunJob executes the ordinary incremental scan (R025): conversations whose source changed.
func (a *Analyzer) RunJob(ctx context.Context, job models.Job) (*models.JobRun, error) {
	return a.execute(ctx, job, ordinaryPlan(), nil)
}

// RunJobFull re-analyzes every conversation, including already evaluated ones. Use after rule
// changes. It never moves the ordinary checkpoint.
func (a *Analyzer) RunJobFull(ctx context.Context, job models.Job) (*models.JobRun, error) {
	plan, err := newPlan(modeFull, 0, pkg.BusinessRange{})
	if err != nil {
		return nil, err
	}
	return a.execute(ctx, job, plan, nil)
}

// RunJobFullWithParams re-analyzes with optional Vietnam date bounds (conversation last message,
// inclusive dates) and an optional count cap. A cap never switches the mode to unanalyzed.
func (a *Analyzer) RunJobFullWithParams(ctx context.Context, job models.Job, dateFrom, dateTo string, maxConv int) (*models.JobRun, error) {
	dates, err := pkg.ParseBusinessRange(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	plan, err := newPlan(modeFull, maxConv, dates)
	if err != nil {
		return nil, err
	}
	return a.execute(ctx, job, plan, nil)
}

// RunJobUnanalyzed analyzes conversations not yet evaluated by this job, regardless of time.
func (a *Analyzer) RunJobUnanalyzed(ctx context.Context, job models.Job, maxConv int) (*models.JobRun, error) {
	plan, err := newPlan(modeUnanalyzed, maxConv, pkg.BusinessRange{})
	if err != nil {
		return nil, err
	}
	return a.execute(ctx, job, plan, nil)
}

// RunJobSinceLast analyzes conversations whose last message is newer than the newest conversation
// this job has evaluated in its current input channels (an event-time cursor: it does not cover
// older, equal-timestamp or backdated unevaluated conversations). Without an evaluated
// conversation it behaves as the unanalyzed mode.
func (a *Analyzer) RunJobSinceLast(ctx context.Context, job models.Job, maxConv int) (*models.JobRun, error) {
	plan, err := newPlan(modeSinceLast, maxConv, pkg.BusinessRange{})
	if err != nil {
		return nil, err
	}
	return a.execute(ctx, job, plan, nil)
}

// RunJobWithProvider runs with an injected AI provider (for testing without real API keys): a
// zero limit is the ordinary incremental scan, a positive limit a test run.
func (a *Analyzer) RunJobWithProvider(ctx context.Context, job models.Job, limit int, provider ai.AIProvider) (*models.JobRun, error) {
	mode := modeOrdinary
	if limit != 0 {
		mode = modeTestRun
	}
	plan, err := newPlan(mode, limit, pkg.BusinessRange{})
	if err != nil {
		return nil, err
	}
	return a.execute(ctx, job, plan, provider)
}

// RunReserved executes a run the caller already reserved with ReserveJobRun (the HTTP handlers do
// this before answering 202). It consumes the reservation exactly once and never reserves again.
// mode is one of "test_run", "unanalyzed", "since_last", "conditional"; an invalid combination
// closes the reservation through the checked finalizer and returns ErrInvalidRunParameters.
func (a *Analyzer) RunReserved(res *JobRunReservation, job models.Job, mode string, limit int, dateFrom, dateTo string) (*models.JobRun, error) {
	dates, err := pkg.ParseBusinessRange(dateFrom, dateTo)
	var plan runPlan
	if err == nil {
		var m analysisMode
		switch mode {
		case "test_run":
			m = modeTestRun
		case "unanalyzed":
			m = modeUnanalyzed
		case "since_last":
			m = modeSinceLast
		case "conditional":
			m = modeFull
		default:
			err = ErrInvalidRunParameters
		}
		if err == nil {
			plan, err = newPlan(m, limit, dates)
		}
	}
	if err != nil {
		if abortErr := res.Abort(job, "Tham số chạy không hợp lệ"); abortErr != nil {
			return nil, abortErr
		}
		return nil, err
	}
	return a.executeReserved(res, job, plan, nil)
}

// NewAnalyzerWithProvider returns an Analyzer that always uses the given provider. It exists so
// tests can route a synthetic provider through the real Analyzer from another package (the HTTP
// route test); production code never uses it.
func NewAnalyzerWithProvider(cfg *config.Config, provider ai.AIProvider) *Analyzer {
	a := NewAnalyzer(cfg)
	a.providerOverride = provider
	return a
}

// ---- candidate selection for the explicit modes ----

// evaluatedByJobSQL is true for a conversation that has a conversation_evaluation of this job
// (argument: job id) in this tenant, joined through job_runs with tenant and job identity.
const evaluatedByJobSQL = `EXISTS (
	SELECT 1 FROM job_results jr
	JOIN job_runs jrun ON jrun.id = jr.job_run_id
	WHERE jr.tenant_id = conversations.tenant_id
	  AND jr.conversation_id = conversations.id
	  AND jr.result_type = 'conversation_evaluation'
	  AND jrun.job_id = ?
	  AND jrun.tenant_id = conversations.tenant_id)`

// sinceLastAnchor returns the newest non-NULL last_message_at among conversations of the job's
// current input channels that this job has evaluated, or nil when there is none. A failed query
// is an error, never "no anchor".
func sinceLastAnchor(job models.Job, channelIDs []string) (*time.Time, error) {
	var anchor sql.NullTime
	err := db.DB.Model(&models.Conversation{}).
		Select("MAX(last_message_at)").
		Where("tenant_id = ? AND channel_id IN ?", job.TenantID, channelIDs).
		Where(evaluatedByJobSQL, job.ID).
		Scan(&anchor).Error
	if err != nil {
		return nil, fmt.Errorf("find since-last anchor: %w", err)
	}
	if !anchor.Valid {
		return nil, nil
	}
	return &anchor.Time, nil
}

// explicitCandidates returns the conversations an explicit mode selects: tenant- and
// input-channel-scoped, ordered by last_message_at then id (NULL first), the cap applied after
// eligibility. Time predicates use typed instants and strict/exclusive comparisons, so NULL
// last_message_at never qualifies for a bounded predicate. runStart is the clock captured once for
// the run (the test-run window must not move while the run works).
func explicitCandidates(job models.Job, channelIDs []string, plan runPlan, runStart time.Time) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if len(channelIDs) == 0 {
		return conversations, nil
	}
	q := db.DB.Where("tenant_id = ? AND channel_id IN ?", job.TenantID, channelIDs)

	switch plan.mode {
	case modeTestRun:
		q = q.Where("last_message_at > ?", runStart.Add(-testRunWindow)).
			Where("NOT "+evaluatedByJobSQL, job.ID)
	case modeFull:
		if plan.dates.From != nil {
			q = q.Where("last_message_at >= ?", *plan.dates.From)
		}
		if plan.dates.ToExclusive != nil {
			q = q.Where("last_message_at < ?", *plan.dates.ToExclusive)
		}
	case modeUnanalyzed:
		q = q.Where("NOT "+evaluatedByJobSQL, job.ID)
	case modeSinceLast:
		anchor, err := sinceLastAnchor(job, channelIDs)
		if err != nil {
			return nil, err
		}
		if anchor == nil {
			q = q.Where("NOT "+evaluatedByJobSQL, job.ID) // nothing evaluated yet: unanalyzed policy
		} else {
			q = q.Where("last_message_at > ?", *anchor)
		}
	default:
		return nil, ErrInvalidRunParameters
	}

	q = q.Order("last_message_at ASC, id ASC")
	if plan.limit > 0 {
		q = q.Limit(plan.limit)
	}
	if err := q.Find(&conversations).Error; err != nil {
		return nil, err
	}
	return conversations, nil
}

// prepareExplicit loads each candidate's full local snapshot (the same builder and format as the
// ordinary mode; no time cutoff, so a date bound never cuts the conversation's context). Empty
// source is skipped; snapshot errors are counted, never skipped silently. Cancellation stops it.
func prepareExplicit(ctx context.Context, candidates []models.Conversation) (prepared []preparedConversation, errorCount int, cancelled bool) {
	for _, conv := range candidates {
		if ctx.Err() != nil {
			return prepared, errorCount, true
		}
		snap, err := loadConversationSnapshot(conv, time.Time{})
		if err != nil {
			log.Printf("[analyzer] snapshot error for conversation %s: %v", conv.ID, err)
			errorCount++
			continue
		}
		if snap.Manifest.Coverage == coverageEmpty {
			continue
		}
		prepared = append(prepared, preparedConversation{Conv: conv, Snap: snap})
	}
	return prepared, errorCount, false
}
