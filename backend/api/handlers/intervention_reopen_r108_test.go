package handlers_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/handlers"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const r108B06Reason = "Đã có thông tin bổ sung cần xác minh sau khi ghi kết quả; mở lại để tiếp tục theo dõi."

type r108SchemaMeta struct {
	Columns  []r108ColumnMeta  `json:"columns"`
	Indexes  []r108IndexMeta   `json:"indexes"`
	Triggers []r108TriggerMeta `json:"triggers"`
}

type r108ColumnMeta struct {
	Table        string  `gorm:"column:TABLE_NAME" json:"table"`
	Name         string  `gorm:"column:COLUMN_NAME" json:"name"`
	Type         string  `gorm:"column:COLUMN_TYPE" json:"type"`
	Nullable     string  `gorm:"column:IS_NULLABLE" json:"nullable"`
	Key          string  `gorm:"column:COLUMN_KEY" json:"key"`
	Extra        string  `gorm:"column:EXTRA" json:"extra"`
	CharacterSet *string `gorm:"column:CHARACTER_SET_NAME" json:"character_set,omitempty"`
	Collation    *string `gorm:"column:COLLATION_NAME" json:"collation,omitempty"`
}

type r108IndexMeta struct {
	Table     string `gorm:"column:TABLE_NAME" json:"table"`
	Name      string `gorm:"column:INDEX_NAME" json:"name"`
	NonUnique int    `gorm:"column:NON_UNIQUE" json:"non_unique"`
	Sequence  int    `gorm:"column:SEQ_IN_INDEX" json:"sequence"`
	Column    string `gorm:"column:COLUMN_NAME" json:"column"`
	Type      string `gorm:"column:INDEX_TYPE" json:"type"`
}

type r108TriggerMeta struct {
	Name      string `gorm:"column:TRIGGER_NAME" json:"name"`
	Table     string `gorm:"column:EVENT_OBJECT_TABLE" json:"table"`
	Timing    string `gorm:"column:ACTION_TIMING" json:"timing"`
	Operation string `gorm:"column:EVENT_MANIPULATION" json:"operation"`
}

type r108DBError struct {
	Captured    bool   `json:"captured"`
	Stage       string `json:"stage"`
	Class       string `json:"class"`
	MySQLNumber uint16 `json:"mysql_number"`
	SQLState    string `json:"sql_state"`
	Constraint  string `json:"constraint"`
}

type r108ErrorCapture struct {
	mu     sync.Mutex
	active bool
	result r108DBError
}

func (c *r108ErrorCapture) record(tx *gorm.DB, operation string) {
	if tx == nil || tx.Error == nil || errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active || c.result.Captured {
		return
	}
	result := r108DBError{Captured: true, Stage: r108Stage(operation, tx.Statement.Table), Class: "NON_MYSQL_DATABASE_ERROR", Constraint: "NOT_IDENTIFIED"}
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(tx.Error, &mysqlErr) {
		result.Class = "MYSQL_ERROR"
		result.MySQLNumber = mysqlErr.Number
		result.SQLState = string(mysqlErr.SQLState[:])
		result.Constraint = r108AllowedConstraint(mysqlErr.Message)
	}
	c.result = result
}

func (c *r108ErrorCapture) beginAction() {
	c.mu.Lock()
	c.active = true
	c.result = r108DBError{Class: "NOT_CAPTURED", Constraint: "NOT_IDENTIFIED"}
	c.mu.Unlock()
}

func (c *r108ErrorCapture) endAction() r108DBError {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active = false
	if !c.result.Captured {
		return r108DBError{Stage: "NOT_CAPTURED", Class: "NOT_CAPTURED", Constraint: "NOT_IDENTIFIED"}
	}
	return c.result
}

func r108Stage(operation, table string) string {
	switch table {
	case "messages":
		return "SOURCE_READ"
	case "channels":
		return "CHANNEL_LOCK_READ"
	case "conversations":
		return "CONVERSATION_LOCK_READ"
	case "user_tenants":
		return "MEMBERSHIP_READ"
	case "users":
		return "ASSIGNEE_READ"
	case "analysis_snapshots":
		return "SNAPSHOT_READ"
	case "intervention_cases":
		if operation == "update" {
			return "CASE_PROJECTION_UPDATE"
		}
		return "CASE_READ"
	case "intervention_proposals":
		return "PROPOSAL_READ"
	case "intervention_human_events":
		if operation == "create" {
			return "EVENT_INSERT"
		}
		return "EVENT_READ"
	default:
		return "OTHER_GORM_" + strings.ToUpper(operation)
	}
}

func r108AllowedConstraint(message string) string {
	lower := strings.ToLower(message)
	for _, name := range []string{"uq_intervention_human_version", "uq_intervention_case", "uq_intervention_proposal_revision"} {
		if strings.Contains(lower, strings.ToLower(name)) {
			return name
		}
	}
	return "NOT_IDENTIFIED"
}

func r108CaptureSchema(t *testing.T) r108SchemaMeta {
	t.Helper()
	var meta r108SchemaMeta
	const tables = "'analysis_snapshots','channels','conversations','intervention_cases','intervention_human_events','intervention_proposals','messages','user_tenants','users'"
	if err := db.DB.Raw("SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, EXTRA, CHARACTER_SET_NAME, COLLATION_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME IN (" + tables + ") ORDER BY TABLE_NAME, ORDINAL_POSITION").Scan(&meta.Columns).Error; err != nil {
		t.Fatal("R108 schema column inventory failed")
	}
	if err := db.DB.Raw("SELECT TABLE_NAME, INDEX_NAME, NON_UNIQUE, SEQ_IN_INDEX, COLUMN_NAME, INDEX_TYPE FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME IN (" + tables + ") ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX").Scan(&meta.Indexes).Error; err != nil {
		t.Fatal("R108 schema index inventory failed")
	}
	if err := db.DB.Raw("SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE, ACTION_TIMING, EVENT_MANIPULATION FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = DATABASE() AND EVENT_OBJECT_TABLE IN (" + tables + ") ORDER BY EVENT_OBJECT_TABLE, TRIGGER_NAME").Scan(&meta.Triggers).Error; err != nil {
		t.Fatal("R108 schema trigger inventory failed")
	}
	return meta
}

type r108CaseSummary struct {
	Version         int    `json:"version"`
	Disposition     string `json:"disposition"`
	WorkState       string `json:"work_state"`
	HumanAssessment string `json:"human_assessment"`
	OutcomePresent  bool   `json:"outcome_present"`
	DueAt           string `json:"due_at,omitempty"`
}

type r108EventSummary struct {
	Version              int    `json:"version"`
	Action               string `json:"action"`
	ReasonMatchesB06     bool   `json:"reason_matches_b06"`
	BeforeOutcomePresent bool   `json:"before_outcome_present"`
	AfterOutcomePresent  bool   `json:"after_outcome_present"`
}

type r108Observation struct {
	Case                 r108CaseSummary              `json:"case"`
	EventCount           int                          `json:"event_count"`
	Events               []r108EventSummary           `json:"events"`
	ProposalCount        int64                        `json:"proposal_count"`
	PersistedStateSHA256 string                       `json:"persisted_case_and_events_sha256"`
	Proposal             *models.InterventionProposal `json:"-"`
}

func r108OutcomePresent(projection string) bool {
	var value map[string]any
	if json.Unmarshal([]byte(projection), &value) != nil {
		return false
	}
	outcome, _ := value["outcome_text"].(string)
	return outcome != ""
}

func r108Observe(t *testing.T, caseID string) r108Observation {
	t.Helper()
	var row models.InterventionCase
	if err := db.DB.Where("id = ?", caseID).Take(&row).Error; err != nil {
		t.Fatal("R108 case observation failed")
	}
	summary := r108Observation{Case: r108CaseSummary{
		Version: row.Version, Disposition: row.Disposition, WorkState: row.WorkState,
		HumanAssessment: row.HumanAssessment, OutcomePresent: row.OutcomeText != "",
	}}
	if row.DueAt != nil {
		summary.Case.DueAt = row.DueAt.UTC().Format(time.RFC3339)
	}
	var events []models.InterventionHumanEvent
	if err := db.DB.Where("case_id = ?", caseID).Order("case_version").Find(&events).Error; err != nil {
		t.Fatal("R108 event observation failed")
	}
	stateBytes, err := json.Marshal(struct {
		Case   models.InterventionCase
		Events []models.InterventionHumanEvent
	}{Case: row, Events: events})
	if err != nil {
		t.Fatal("R108 persisted-state digest failed")
	}
	stateDigest := sha256.Sum256(stateBytes)
	summary.PersistedStateSHA256 = hex.EncodeToString(stateDigest[:])
	summary.EventCount = len(events)
	for _, event := range events {
		summary.Events = append(summary.Events, r108EventSummary{
			Version: event.CaseVersion, Action: event.Action, ReasonMatchesB06: event.Action == "REOPEN" && event.Reason == r108B06Reason,
			BeforeOutcomePresent: r108OutcomePresent(event.Before), AfterOutcomePresent: r108OutcomePresent(event.After),
		})
	}
	var proposals []models.InterventionProposal
	if err := db.DB.Where("case_id = ?", caseID).Order("revision").Find(&proposals).Error; err != nil {
		t.Fatal("R108 proposal observation failed")
	}
	summary.ProposalCount = int64(len(proposals))
	if len(proposals) == 1 {
		proposal := proposals[0]
		summary.Proposal = &proposal
	}
	return summary
}

type r108ActionResult struct {
	Action     string          `json:"action"`
	ActorRole  string          `json:"actor_role"`
	Request    map[string]any  `json:"request_fields"`
	HTTPStatus int             `json:"http_status"`
	SafeError  string          `json:"safe_error,omitempty"`
	Before     r108Observation `json:"before"`
	After      r108Observation `json:"after"`
	DBError    r108DBError     `json:"db_error"`
}

type r108Report struct {
	SchemaBeforeMigration                r108SchemaMeta     `json:"schema_before_migration"`
	SchemaAfterMigration                 r108SchemaMeta     `json:"schema_after_migration"`
	Before                               r108Observation    `json:"before"`
	After                                r108Observation    `json:"after"`
	Actions                              []r108ActionResult `json:"actions"`
	DBError                              r108DBError        `json:"db_error"`
	ErrorCaptureCoverage                 string             `json:"error_capture_coverage"`
	ProposalComparisonAvailable          bool               `json:"proposal_comparison_available"`
	ProposalUnchanged                    bool               `json:"proposal_unchanged"`
	FailedActionStateComparisonAvailable bool               `json:"failed_action_state_comparison_available"`
	FailedActionStateUnchanged           bool               `json:"failed_action_state_unchanged"`
	ReopenDueMatches                     bool               `json:"reopen_due_matches_request"`
	PriorOutcomeInHistory                bool               `json:"prior_outcome_retained_in_history"`
	ReopenOutcomeCleared                 bool               `json:"reopen_outcome_cleared"`
	LifecycleAccepted                    bool               `json:"lifecycle_accepted"`
}

func r108ProposalComparison(before, after r108Observation) (bool, bool) {
	if before.Proposal == nil || after.Proposal == nil {
		return false, false
	}
	return true, *before.Proposal == *after.Proposal
}

func r108EventsEqual(left, right []r108EventSummary) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func r108StateUnchanged(before, after r108Observation) bool {
	available, proposalUnchanged := r108ProposalComparison(before, after)
	return available && proposalUnchanged && before.PersistedStateSHA256 != "" && before.PersistedStateSHA256 == after.PersistedStateSHA256 && before.Case == after.Case && before.EventCount == after.EventCount && r108EventsEqual(before.Events, after.Events)
}

func r108SafeHTTPError(body []byte) string {
	var response struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &response) != nil {
		return "UNCLASSIFIED"
	}
	switch response.Error {
	case "not_found", "permission_denied", "invalid_action", "case_conflict", "action_unavailable", "assignees_unavailable":
		return response.Error
	default:
		return "UNCLASSIFIED"
	}
}

func r108Emit(t *testing.T, report r108Report) {
	t.Helper()
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal("R108 diagnostic report serialization failed")
	}
	fmt.Printf("R108_DIAGNOSTIC_JSON=%s\n", body)
}

func TestR108ReopenDiagnostic(t *testing.T) {
	if os.Getenv("R108_DIAGNOSTIC") != "1" {
		t.Skip("NOT_RUN: R108_DIAGNOSTIC=1 required")
	}
	if handlers.InterventionRunner != nil {
		t.Fatal("R108 must not install an intervention transport")
	}
	dsn := os.Getenv("R101_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NOT_RUN: R101_MYSQL_DSN owned disposable MySQL required")
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Fatal("R108 pre-migration database connection failed")
	}
	beforeMigration := r108CaptureSchema(t)
	db.Close()

	f := m3FixtureStart(t)
	afterMigration := r108CaptureSchema(t)
	capture := &r108ErrorCapture{}
	queryHook := "r108:diagnostic_query"
	createHook := "r108:diagnostic_create"
	updateHook := "r108:diagnostic_update"
	if err := db.DB.Callback().Query().After("gorm:query").Register(queryHook, func(tx *gorm.DB) { capture.record(tx, "query") }); err != nil {
		t.Fatal("R108 query diagnostic hook registration failed")
	}
	if err := db.DB.Callback().Create().After("gorm:create").Register(createHook, func(tx *gorm.DB) { capture.record(tx, "create") }); err != nil {
		t.Fatal("R108 create diagnostic hook registration failed")
	}
	if err := db.DB.Callback().Update().After("gorm:update").Register(updateHook, func(tx *gorm.DB) { capture.record(tx, "update") }); err != nil {
		t.Fatal("R108 update diagnostic hook registration failed")
	}
	t.Cleanup(func() {
		_ = db.DB.Callback().Query().Remove(queryHook)
		_ = db.DB.Callback().Create().Remove(createHook)
		_ = db.DB.Callback().Update().Remove(updateHook)
	})

	report := r108Report{
		SchemaBeforeMigration: beforeMigration,
		SchemaAfterMigration:  afterMigration,
		ErrorCaptureCoverage:  "GORM query/create/update callbacks only; transaction begin/commit, raw SQL, and failures outside these callbacks may report NOT_CAPTURED",
	}
	report.Before = r108Observe(t, f.caseID)
	due := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	dueText := due.Format(time.RFC3339)
	steps := []struct {
		name, actor, role string
		version           int
		fields            map[string]any
		reason            string
	}{
		{name: "CORRECT", actor: f.member, role: "member", version: 1, fields: map[string]any{"explanation": "Điều chỉnh thông tin xử lý", "next_action": "Kiểm tra đơn hàng"}, reason: "R108 synthetic correction"},
		{name: "ASSIGN", actor: f.owner, role: "owner", version: 2, fields: map[string]any{"assignee_id": f.assignee, "due_at": dueText}, reason: "R108 synthetic assignment"},
		{name: "RESOLVE", actor: f.member, role: "member", version: 3, fields: map[string]any{"outcome": "Đã xác nhận và giao bộ phận xử lý"}, reason: "R108 synthetic resolution"},
		{name: "REOPEN", actor: f.owner, role: "owner", version: 4, fields: map[string]any{"due_at": dueText}, reason: r108B06Reason},
	}
	for _, step := range steps {
		beforeAction := r108Observe(t, f.caseID)
		request := f.action(step.actor, f.caseID, step.name, step.version, step.fields)
		request["reason"] = step.reason
		capture.beginAction()
		response := f.send("POST", f.tenant, step.actor, "/intervention-cases/"+f.caseID+"/actions", request)
		dbError := capture.endAction()
		afterAction := r108Observe(t, f.caseID)
		action := r108ActionResult{Action: step.name, ActorRole: step.role, Request: request, HTTPStatus: response.Code, Before: beforeAction, After: afterAction, DBError: dbError}
		if response.Code != http.StatusOK {
			action.SafeError = r108SafeHTTPError(response.Body.Bytes())
		}
		report.Actions = append(report.Actions, action)
		report.DBError = dbError
		if response.Code != http.StatusOK {
			report.After = afterAction
			report.ProposalComparisonAvailable, report.ProposalUnchanged = r108ProposalComparison(beforeAction, afterAction)
			report.FailedActionStateComparisonAvailable = true
			report.FailedActionStateUnchanged = r108StateUnchanged(beforeAction, afterAction)
			r108Emit(t, report)
			t.Fatalf("R108 action %s returned HTTP %d; see sanitized diagnostic", step.name, response.Code)
		}
	}

	report.After = r108Observe(t, f.caseID)
	report.ProposalComparisonAvailable, report.ProposalUnchanged = r108ProposalComparison(report.Before, report.After)
	report.PriorOutcomeInHistory = len(report.After.Events) == 4 && report.After.Events[3].BeforeOutcomePresent
	report.ReopenOutcomeCleared = report.After.Case.OutcomePresent == false && len(report.After.Events) == 4 && !report.After.Events[3].AfterOutcomePresent
	if report.After.Case.DueAt != "" {
		actualDue, err := time.Parse(time.RFC3339, report.After.Case.DueAt)
		report.ReopenDueMatches = err == nil && actualDue.Equal(due)
	}
	expectedActions := []string{"CORRECT", "ASSIGN", "RESOLVE", "REOPEN"}
	report.LifecycleAccepted = len(report.Actions) == 4 && report.After.Case.Version == 5 && report.After.Case.WorkState == "ASSIGNED" && report.After.Case.Disposition == "HUMAN_CORRECT" && report.ProposalUnchanged && report.PriorOutcomeInHistory && report.ReopenOutcomeCleared && report.ReopenDueMatches
	if len(report.After.Events) == len(expectedActions) {
		for index, action := range expectedActions {
			report.LifecycleAccepted = report.LifecycleAccepted && report.After.Events[index].Action == action && report.After.Events[index].Version == index+2
		}
		report.LifecycleAccepted = report.LifecycleAccepted && report.After.Events[3].ReasonMatchesB06
	} else {
		report.LifecycleAccepted = false
	}
	r108Emit(t, report)
	if !report.LifecycleAccepted {
		t.Fatal("R108 lifecycle acceptance failed; see sanitized diagnostic")
	}
}
