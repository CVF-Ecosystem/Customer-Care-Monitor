package engine

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"gorm.io/gorm"
)

func TestExecutionReadPureDTOBoundary(t *testing.T) {
	now := time.Now().UTC()
	request := "11111111-1111-4111-8111-111111111111"
	base := ExecutionReadiness{State: "AWAITING_OPERATOR_AUTHORITY", RequestID: request, ServerNow: now}
	b, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"grant_handle", "expires_at", "expected_case_version"} {
		if strings.Contains(string(b), forbidden) {
			t.Fatalf("non-ready response exposed %s", forbidden)
		}
	}
	version, expiry := 2, now.Add(time.Minute)
	base.State, base.ExpectedCaseVersion, base.ExpiresAt, base.GrantHandle = "READY", &version, &expiry, request
	b, err = json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"grant_handle", "expires_at", "expected_case_version"} {
		if !strings.Contains(string(b), required) {
			t.Fatalf("ready response omitted %s", required)
		}
	}
	status := ExecutionStatus{RequestID: request, RequestIntent: "AWAITING_EXECUTION_AUTHORITY", Phase: "NOT_STARTED", ObservedAt: now}
	b, err = json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "published_case_version") || strings.Contains(string(b), "proposal_available") || strings.Contains(string(b), "terminal_code") {
		t.Fatal("no-row response claimed outcome")
	}
}

func TestExecutionReadPureIdentityValidation(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	if err := executionReadIdentity(id, id, id, id); err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][4]string{{"bad", id, id, id}, {id, "bad", id, id}, {id, id, "bad", id}, {id, id, id, "bad"}} {
		if executionReadIdentity(ids[0], ids[1], ids[2], ids[3]) != ErrExecutionReadInvalid {
			t.Fatalf("accepted invalid identity: %v", ids)
		}
	}
}

func TestExecutionReadPureMissingVersusUnavailable(t *testing.T) {
	if executionReadDependency(gorm.ErrRecordNotFound) != nil {
		t.Fatal("known missing dependency should be ineligible")
	}
	if executionReadDependency(gorm.ErrInvalidDB) != ErrExecutionReadUnavailable {
		t.Fatal("database failure should be unavailable")
	}
}

func TestExecutionReadPureLineageRejectsChangedLinks(t *testing.T) {
	request := models.InterventionRerunRequest{ID: "request", TenantID: "tenant", CaseID: "case", JobID: "job", ConversationID: "conversation", DataRevisionID: "revision", ActorID: "actor", OriginalSnapshotID: "original", OriginalSnapshotDigest: "digest", CaseVersion: 2}
	row := models.InterventionExecution{ID: "execution", TenantID: "tenant", CaseID: "case", JobID: "job", ConversationID: "conversation", RequestID: "request", RevisionID: "revision", ActorID: "actor", OriginalSnapshotID: "original", OriginalSnapshotDigest: "digest", ExpectedCaseVersion: 2, ClaimedCaseVersion: 2, RunID: "run", SnapshotID: "snapshot", SnapshotDigest: "new-digest"}
	if !executionRowMatchesRequest(row, request) {
		t.Fatal("valid request link rejected")
	}
	mutations := []func(*models.InterventionExecution){
		func(r *models.InterventionExecution) { r.JobID = "other" }, func(r *models.InterventionExecution) { r.ConversationID = "other" },
		func(r *models.InterventionExecution) { r.ActorID = "other" }, func(r *models.InterventionExecution) { r.RevisionID = "other" },
		func(r *models.InterventionExecution) { r.OriginalSnapshotID = "other" }, func(r *models.InterventionExecution) { r.OriginalSnapshotDigest = "other" },
		func(r *models.InterventionExecution) { r.ExpectedCaseVersion = 3 },
	}
	for i, mutate := range mutations {
		changed := row
		mutate(&changed)
		if executionRowMatchesRequest(changed, request) {
			t.Fatalf("request mutation %d accepted", i)
		}
	}
	proposalID, revisionID, requestID := "proposal", "revision", "request"
	proposal := models.InterventionProposal{ID: proposalID, TenantID: "tenant", CaseID: "case", RunID: "run", SnapshotID: "snapshot", SnapshotDigest: "new-digest", Revision: 3}
	event := models.InterventionHumanEvent{TenantID: "tenant", CaseID: "case", ActorID: "actor", Action: "EXECUTE_RERUN", RerunRequestID: &requestID, ProposalID: &proposalID, DataRevisionID: &revisionID, CaseVersion: 3, SnapshotID: "snapshot", SnapshotDigest: "new-digest"}
	after := map[string]any{"execution_id": "execution", "request_id": "request", "revision_id": "revision", "run_id": "run", "snapshot_id": "snapshot", "snapshot_digest": "new-digest", "active_proposal_id": "proposal"}
	if !publishedLinksMatch(row, proposal, event, after) {
		t.Fatal("valid published links rejected")
	}
	changedEvent := event
	changedEvent.ActorID = "other"
	if publishedLinksMatch(row, proposal, changedEvent, after) {
		t.Fatal("foreign actor event accepted")
	}
	changedProposal := proposal
	changedProposal.RunID = "other"
	if publishedLinksMatch(row, changedProposal, event, after) {
		t.Fatal("unrelated proposal accepted")
	}
	changedAfter := map[string]any{}
	for k, v := range after {
		changedAfter[k] = v
	}
	changedAfter["execution_id"] = "other"
	if publishedLinksMatch(row, proposal, event, changedAfter) {
		t.Fatal("unrelated event projection accepted")
	}
}
