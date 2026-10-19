package engine_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/cli"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	"gorm.io/gorm/logger"
)

type noCallActivationTransport struct{ calls atomic.Int32 }

func (p *noCallActivationTransport) AnalyzeChatOnce(context.Context, string, string, int64) (ai.AIResponse, error) {
	p.calls.Add(1)
	return ai.AIResponse{}, nil
}

func TestSyntheticOperatorProvisionBudgetGrant(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN disposable MySQL required")
	}
	if err := db.Connect(dsn, false); err != nil {
		t.Fatal(err)
	}
	db.DB.Logger = logger.Default.LogMode(logger.Silent)
	if err := db.AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	tenant, owner, member := models.SingleWorkspaceID, pkg.NewUUID(), pkg.NewUUID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	t.Cleanup(func() {
		for _, model := range []any{&models.InterventionReceipt{}, &models.InterventionProposal{}, &models.InterventionAttempt{}, &models.InterventionExecution{}, &models.InterventionExecutionGrant{}, &models.InterventionBudget{}, &models.InterventionRerunRequest{}, &models.InterventionDataRevision{}, &models.InterventionHumanEvent{}, &models.InterventionCase{}, &models.AnalysisSnapshot{}, &models.JobRun{}, &models.Message{}, &models.Conversation{}, &models.Job{}, &models.Channel{}, &models.UserTenant{}} {
			if err := db.DB.Unscoped().Where("tenant_id = ?", tenant).Delete(model).Error; err != nil {
				t.Error(err)
			}
		}
		_ = db.DB.Unscoped().Where("id = ?", tenant).Delete(&models.Tenant{}).Error
		_ = db.DB.Unscoped().Where("id IN ?", []string{owner, member}).Delete(&models.User{}).Error
		db.Close()
	})
	for _, row := range []any{
		&models.Tenant{ID: tenant, Name: "R123 synthetic", Slug: "r123-" + tenant, Settings: "{}", CreatedAt: now, UpdatedAt: now},
		&models.User{ID: owner, Email: owner + "@example.invalid", Name: "R123 owner", PasswordHash: "synthetic", CreatedAt: now, UpdatedAt: now},
		&models.User{ID: member, Email: member + "@example.invalid", Name: "R123 member", PasswordHash: "synthetic", CreatedAt: now, UpdatedAt: now},
		&models.UserTenant{UserID: owner, TenantID: tenant, Role: "owner", Permissions: "{}"},
		&models.UserTenant{UserID: member, TenantID: tenant, Role: "member", Permissions: `{"intervention_cases":"rw","jobs":"rw"}`},
	} {
		if err := db.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	transport := &noCallActivationTransport{}
	service := &engine.InterventionService{Transport: transport, Provider: "openai", Model: "gpt-5", ConfigRef: "r123-synthetic", PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "synthetic-activation-v1", Prompt: "Synthetic only", MaxRequestBytes: 6000, Timeout: 5 * time.Second}
	ctx := context.Background()
	if _, err := cli.PrepareSyntheticInterventionDemo(ctx, db.DB, tenant, member, service); err == nil {
		t.Fatal("member prepared server demo")
	}
	ids, err := cli.PrepareSyntheticInterventionDemo(ctx, db.DB, tenant, owner, service)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := cli.PrepareSyntheticInterventionDemo(ctx, db.DB, tenant, owner, service)
	if err != nil || *replay != *ids {
		t.Fatalf("prepare replay changed identifiers: first=%+v replay=%+v err=%v", ids, replay, err)
	}
	if _, err := cli.SetSyntheticInterventionBudget(ctx, db.DB, tenant, member, ids.JobID, 2); err == nil {
		t.Fatal("member raised budget")
	}
	budget, err := cli.SetSyntheticInterventionBudget(ctx, db.DB, tenant, owner, ids.JobID, 2)
	if err != nil || budget.Cap != 2 || budget.Consumed != 0 {
		t.Fatalf("budget: %+v err=%v", budget, err)
	}
	if _, err := cli.SetSyntheticInterventionBudget(ctx, db.DB, tenant, owner, ids.JobID, 1); err == nil {
		t.Fatal("cap lowered")
	}
	if _, err := cli.SetSyntheticInterventionBudget(ctx, db.DB, tenant, owner, ids.JobID, 1001); err == nil {
		t.Fatal("unbounded cap")
	}
	if err := db.DB.Model(&models.InterventionBudget{}).Where("id = ?", budget.ID).Update("consumed", 1).Error; err != nil {
		t.Fatal(err)
	}
	raised, err := cli.SetSyntheticInterventionBudget(ctx, db.DB, tenant, owner, ids.JobID, 3)
	if err != nil || raised.Cap != 3 || raised.Consumed != 1 {
		t.Fatalf("raising cap reset consumption: %+v %v", raised, err)
	}
	wait, err := service.RunManual(ctx, tenant, owner, ids.JobID, ids.WaitConversation)
	if err != nil || wait.WorkState != "WAITING_ON_DATA" || transport.calls.Load() != 0 {
		t.Fatalf("WAIT_DATA invoked transport: case=%+v err=%v calls=%d", wait, err, transport.calls.Load())
	}
	if _, err := cli.CompleteSyntheticInterventionDemoData(ctx, db.DB, tenant, member); err == nil {
		t.Fatal("member completed fixture")
	}
	messageID, err := cli.CompleteSyntheticInterventionDemoData(ctx, db.DB, tenant, owner)
	if err != nil {
		t.Fatal(err)
	}
	replayedID, err := cli.CompleteSyntheticInterventionDemoData(ctx, db.DB, tenant, owner)
	if err != nil || replayedID != messageID {
		t.Fatalf("data completion replay: %q %q %v", messageID, replayedID, err)
	}
	var message models.Message
	if err := db.DB.Where("id = ?", messageID).Take(&message).Error; err != nil {
		t.Fatal(err)
	}
	revision, err := engine.CaptureInterventionDataRevision(ctx, tenant, owner, wait.ID, engine.DataRevisionInput{ExpectedVersion: wait.Version, ExpectedSnapshotID: *wait.SnapshotID, ExpectedSnapshotDigest: wait.SnapshotDigest, Reason: "Synthetic source completed"})
	if err != nil {
		t.Fatal(err)
	}
	request, _, err := engine.RequestInterventionRerun(ctx, tenant, owner, wait.ID, engine.RerunRequestInput{ExpectedVersion: wait.Version + 1, ExpectedSnapshotID: *wait.SnapshotID, ExpectedSnapshotDigest: wait.SnapshotDigest, DataRevisionID: revision.ID, DataRevisionDigest: revision.Digest, Reason: "Synthetic reassessment", RequestKey: "r123-operator-once"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().UTC().Add(10 * time.Minute).Truncate(time.Second)
	if _, _, err := engine.IssueSyntheticInterventionGrant(ctx, tenant, owner, owner, wait.ID, request.ID, "operator approval", time.Now().UTC().Add(25*time.Hour), service); err == nil {
		t.Fatal("out-of-window deadline accepted")
	}
	if _, _, err := engine.IssueSyntheticInterventionGrant(ctx, tenant, member, owner, wait.ID, request.ID, "operator approval", deadline, service); err == nil {
		t.Fatal("member issued grant")
	}
	if _, _, err := engine.IssueSyntheticInterventionGrant(ctx, tenant, owner, member, wait.ID, request.ID, "operator approval", deadline, service); err == nil {
		t.Fatal("grant actor differed from request")
	}
	if err := db.DB.Model(&models.Message{}).Where("id = ?", message.ID).Update("content", "New synthetic information").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := engine.IssueSyntheticInterventionGrant(ctx, tenant, owner, owner, wait.ID, request.ID, "operator approval", deadline, service); err == nil {
		t.Fatal("stale source granted")
	}
	if err := db.DB.Model(&models.Message{}).Where("id = ?", message.ID).Update("content", message.Content).Error; err != nil {
		t.Fatal(err)
	}
	grant, replayed, err := engine.IssueSyntheticInterventionGrant(ctx, tenant, owner, owner, wait.ID, request.ID, "operator approval", deadline, service)
	if err != nil || replayed || grant.State != "ISSUED" || grant.Provider != "openai" || grant.ConfigRef != service.ConfigRef {
		t.Fatalf("grant=%+v replay=%v err=%v", grant, replayed, err)
	}
	again, replayed, err := engine.IssueSyntheticInterventionGrant(ctx, tenant, owner, owner, wait.ID, request.ID, "operator approval", deadline, service)
	if err != nil || !replayed || again.ID != grant.ID {
		t.Fatalf("grant replay=%+v replay=%v err=%v", again, replayed, err)
	}
	if _, _, err := engine.IssueSyntheticInterventionGrant(ctx, tenant, owner, owner, wait.ID, request.ID, "changed approval", deadline, service); err == nil {
		t.Fatal("changed grant reused request")
	}
	if transport.calls.Load() != 0 {
		t.Fatalf("provider called during provision: %d", transport.calls.Load())
	}
	input := engine.WaitDataExecuteInput{ExpectedCaseVersion: request.CaseVersion, GrantID: grant.ID}
	for _, badTenant := range []string{"malformed", pkg.NewUUID()} {
		if _, _, err := engine.ClaimWaitDataExecution(ctx, badTenant, owner, wait.ID, request.ID, input, service); err == nil {
			t.Fatalf("invalid tenant %q admitted", badTenant)
		}
	}
	if transport.calls.Load() != 0 {
		t.Fatal("invalid tenant invoked transport")
	}
	claim, replayed, err := engine.ClaimWaitDataExecution(ctx, tenant, owner, wait.ID, request.ID, input, service)
	if err != nil || replayed || claim == nil || claim.ID == "" {
		t.Fatalf("single-workspace claim: %+v replay=%v err=%v", claim, replayed, err)
	}
	end := time.Now().Add(5 * time.Second)
	var execution models.InterventionExecution
	for time.Now().Before(end) {
		if err := db.DB.Where("id = ?", claim.ID).Take(&execution).Error; err == nil && execution.State == "PROVIDER_UNKNOWN" && !engine.JobRunActive(tenant, ids.JobID) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if execution.State != "PROVIDER_UNKNOWN" || engine.JobRunActive(tenant, ids.JobID) {
		t.Fatalf("stub execution did not settle: state=%s active=%v", execution.State, engine.JobRunActive(tenant, ids.JobID))
	}
	if got := transport.calls.Load(); got != 1 {
		t.Fatalf("expected one in-memory stub call, got %d", got)
	}
	var finalBudget models.InterventionBudget
	if err := db.DB.Where("id = ?", budget.ID).Take(&finalBudget).Error; err != nil || finalBudget.Consumed != 2 {
		t.Fatalf("claim consumption=%+v err=%v", finalBudget, err)
	}
}
