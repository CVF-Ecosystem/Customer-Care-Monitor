// Package jobdispatch is the transport-independent job dispatch shared by the HTTP trigger and the
// MCP cqa_trigger_job tool (CCMAI-RUNTIME-044). It validates the configuration, reserves one owned
// run through the engine's ReserveJobRun, hands the reservation to a bounded background worker and
// aborts the reservation when the worker cannot be started.
//
// Authorization and the tenant-scoped job lookup stay in each transport; this package receives a job
// the caller has already looked up for its own tenant. It never reads a request context: the worker
// lifetime comes from the reservation (context.Background plus the bounded timeout), so a finished
// or cancelled request cannot stop an accepted run. The engine must not import this package.
package jobdispatch

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
)

// DefaultTimeout bounds one accepted run (the worker never uses a request context).
const DefaultTimeout = 30 * time.Minute

var (
	// ErrBusy: the tenant/job already has an owned execution or a stored running row.
	ErrBusy = engine.ErrJobBusy
	// ErrMissing: the tenant-scoped job vanished between the caller's lookup and the reservation.
	ErrMissing = engine.ErrJobMissing
	// ErrStartFailed: configuration, admission or worker start failed. It carries no detail.
	ErrStartFailed = engine.ErrJobAdmission
)

// Params are the run parameters handed unchanged to RunReserved. Mode is one of
// "test_run", "unanalyzed", "since_last", "conditional".
type Params struct {
	Mode             string
	DateFrom, DateTo string
	Limit            int
}

// StartFunc launches the worker for a reservation the caller holds. It consumes the reservation
// exactly once. A returned error means no worker will run it, and Dispatch aborts the reservation.
type StartFunc func(job models.Job, cfg *config.Config, p Params, res *engine.JobRunReservation) error

// Service is one dispatch path. Zero-value fields fall back to production behavior. The function
// fields exist so a transport or a test can intercept configuration loading and Analyzer creation
// before any real environment, provider or channel access.
type Service struct {
	// Label names the entry in server logs (no secrets), e.g. "trigger".
	Label string
	// LoadConfig loads and validates the configuration; nil means config.Load.
	LoadConfig func() (*config.Config, error)
	// Timeout bounds the run; zero means DefaultTimeout.
	Timeout time.Duration
	// Start launches the worker; nil means StartWorker with NewAnalyzer.
	Start StartFunc
	// NewAnalyzer builds the Analyzer used by the default Start; nil means engine.NewAnalyzer.
	NewAnalyzer func(*config.Config) *engine.Analyzer
}

// Default is the production service used by MCP. Tests may replace it and must restore it.
var Default = &Service{Label: "mcp trigger"}

// Dispatch admits and starts one run of an already tenant-verified job and returns the persisted
// reservation's run ID. Order: configuration, reservation, worker start. Errors are ErrBusy,
// ErrMissing or ErrStartFailed; nothing is reserved when the configuration is invalid, and a start
// failure aborts the reservation (a failed abort leaves the stored running row blocking, fail
// closed). Success means the run is reserved and its worker handed off, not that analysis finished.
func (s *Service) Dispatch(job models.Job, p Params) (string, error) {
	label := s.Label
	if label == "" {
		label = "trigger"
	}
	load := s.LoadConfig
	if load == nil {
		load = config.Load
	}
	// The error is not logged or returned because validation messages describe secret configuration.
	cfg, err := load()
	if err != nil || cfg == nil {
		log.Printf("[error] %s for job %s not admitted: configuration invalid", label, job.Name)
		return "", ErrStartFailed
	}

	timeout := s.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	res, err := engine.ReserveJobRun(context.Background(), job, timeout)
	switch {
	case err == nil:
	case errors.Is(err, ErrBusy):
		return "", ErrBusy
	case errors.Is(err, ErrMissing):
		return "", ErrMissing
	default:
		log.Printf("[error] job %s not admitted", job.ID)
		return "", ErrStartFailed
	}

	start := s.Start
	if start == nil {
		newAnalyzer := s.NewAnalyzer
		start = func(job models.Job, cfg *config.Config, p Params, res *engine.JobRunReservation) error {
			return StartWorker(label, newAnalyzer, job, cfg, p, res)
		}
	}
	if err := start(job, cfg, p, res); err != nil {
		AbortReservation(job, res)
		return "", ErrStartFailed
	}
	return res.RunID(), nil
}

// AbortReservation closes a reservation whose worker could not be started: the stored row is
// finalized as error (checked) or, if that cannot be recorded, stays running and keeps blocking.
func AbortReservation(job models.Job, res *engine.JobRunReservation) {
	if err := res.Abort(job, "Không khởi động được tiến trình chạy"); err != nil {
		log.Printf("[error] reserved run %s of job %s could not be closed: it keeps blocking admission", res.RunID(), job.ID)
	}
}

// StartWorker launches the bounded background worker for a reservation. The goroutine ends when
// RunReserved returns (its context is the reservation's, bounded by the timeout). A panic is
// recovered without logging its value and aborts the reservation. newAnalyzer nil means
// engine.NewAnalyzer.
func StartWorker(label string, newAnalyzer func(*config.Config) *engine.Analyzer, job models.Job, cfg *config.Config, p Params, res *engine.JobRunReservation) error {
	if newAnalyzer == nil {
		newAnalyzer = engine.NewAnalyzer
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[security] panic in %s goroutine for job %s", label, job.ID) // value not logged
				if err := res.Abort(job, "Lượt chạy dừng đột ngột; xem nhật ký máy chủ."); err != nil {
					log.Printf("[error] reserved run %s keeps blocking admission", res.RunID())
				}
			}
		}()
		analyzer := newAnalyzer(cfg)
		if _, err := analyzer.RunReserved(res, job, p.Mode, p.Limit, p.DateFrom, p.DateTo); err != nil {
			log.Printf("[%s] error for job %s: %v", label, job.Name, err)
		}
	}()
	return nil
}
