package engine

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
	mysqlcfg "github.com/go-sql-driver/mysql"
)

func TestR116ExecuteBodyHashCanonicalAndChanged(t *testing.T) {
	a := WaitDataExecuteInput{ExpectedCaseVersion: 3, GrantID: "A98E0C20-0C48-4751-A8BC-96F6DB62C762"}
	b := WaitDataExecuteInput{ExpectedCaseVersion: 3, GrantID: "a98e0c20-0c48-4751-a8bc-96f6db62c762"}
	original := HashWaitDataExecute("A98E0C20-0C48-4751-A8BC-96F6DB62C762", "63B859FC-9B3A-46C8-867C-6C582B2109D3", "61E7C7E1-0BD0-4B75-B2F6-577EC51DD7E1", a)
	canonical := HashWaitDataExecute("a98e0c20-0c48-4751-a8bc-96f6db62c762", "63b859fc-9b3a-46c8-867c-6c582b2109d3", "61e7c7e1-0bd0-4b75-b2f6-577ec51dd7e1", b)
	if original != canonical || len(original) != 64 {
		t.Fatal("canonical UUID encoding changed")
	}
	b.ExpectedCaseVersion++
	if HashWaitDataExecute("a98e0c20-0c48-4751-a8bc-96f6db62c762", "63b859fc-9b3a-46c8-867c-6c582b2109d3", "61e7c7e1-0bd0-4b75-b2f6-577ec51dd7e1", b) == canonical {
		t.Fatal("changed version replay hash unchanged")
	}
}

type r116SyncTransport struct {
	started chan struct{}
	release chan struct{}
	messageID string
}

func (p *r116SyncTransport) AnalyzeChatOnce(ctx context.Context, prompt, transcript string, maxBytes int64) (ai.AIResponse, error) {
	p.started <- struct{}{}
	select {
	case <-p.release:
	case <-ctx.Done(): return ai.AIResponse{}, ctx.Err()
	}
	var digest, conv string
	for _, part := range strings.Fields(prompt) {
		part = strings.Trim(part, ",.")
		if len(part) == 64 { digest = part }
		if len(part) == 36 && strings.Count(part, "-") == 4 { conv = part }
	}
	content, _ := json.Marshal(map[string]any{"schema":"ccma.intervention.v1", "conversation_id":conv, "snapshot_digest":digest, "kind":"complaint", "explanation":"Khách khiếu nại", "next_action":"Liên hệ khách", "evidence_refs":[]map[string]string{{"message_id":p.messageID,"quote":"khiếu nại"}}})
	in, out := int64(20), int64(10)
	return ai.AIResponse{Provider:"alibaba_maas",Model:"qwen3.7-flash",Content:string(content),UsagePresence:&ai.AIUsagePresence{SchemaVersion:"1",Source:ai.AIUsagePresenceSourceOpenAIWire,InputTokens:ai.AIUsageCount{Status:ai.AIUsagePresenceKnown,Value:&in},OutputTokens:ai.AIUsageCount{Status:ai.AIUsagePresenceKnown,Value:&out},Complete:true}}, nil
}

// This native case uses the actual run-owned sync writer. Root alone opts it
// into the fresh loopback r116_fixture; it never opens a channel adapter.
func TestR116WaitDataNativeSyncWriterRace(t *testing.T) {
	if os.Getenv("CCMAI_R116_NATIVE") != "1" { t.Skip("NOT_RUN: native admission required") }
	dsn := os.Getenv("CCMAI_R116_MYSQL_DSN")
	cfg, err := mysqlcfg.ParseDSN(dsn)
	if err != nil || cfg.Net != "tcp" || !strings.HasPrefix(cfg.Addr, "127.0.0.1:") || cfg.DBName != "r116_fixture" { t.Fatal("owned loopback r116_fixture required") }
	if err := db.Connect(dsn, false); err != nil { t.Fatal(err) }
	defer func() { handle, err := db.DB.DB(); if err == nil { _ = handle.Close() } }()
	if err := db.AutoMigrate(); err != nil { t.Fatal(err) }
	tenant, actor, channelID, convID, jobID, caseID, originalRun := pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID(), pkg.NewUUID()
	syncRun := pkg.NewUUID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	defer func() {
		for _, model := range []any{&models.InterventionReceipt{}, &models.InterventionProposal{}, &models.InterventionAttempt{}, &models.InterventionExecution{}, &models.InterventionExecutionGrant{}, &models.InterventionBudget{}, &models.InterventionRerunRequest{}, &models.InterventionDataRevision{}, &models.InterventionHumanEvent{}, &models.InterventionCase{}, &models.AnalysisSnapshot{}, &models.JobRun{}, &models.Message{}, &models.Conversation{}, &models.Job{}, &models.Channel{}, &models.UserTenant{}} {
			if err := db.DB.Unscoped().Where("tenant_id = ?", tenant).Delete(model).Error; err != nil { t.Error(err) }
		}
		if err := db.DB.Unscoped().Where("id = ?", tenant).Delete(&models.Tenant{}).Error; err != nil { t.Error(err) }
		if err := db.DB.Unscoped().Where("id = ?", actor).Delete(&models.User{}).Error; err != nil { t.Error(err) }
	}()
	for _, row := range []any{
		&models.Tenant{ID:tenant,Name:"R116 sync fixture",Slug:"r116-sync-"+tenant,Settings:"{}",CreatedAt:now,UpdatedAt:now},
		&models.User{ID:actor,Email:actor+"@example.invalid",PasswordHash:"synthetic",Name:"R116 owner",CreatedAt:now,UpdatedAt:now},
		&models.UserTenant{UserID:actor,TenantID:tenant,Role:"owner",Permissions:"{}"},
		&models.Channel{ID:channelID,TenantID:tenant,ChannelType:"pancake",Name:"synthetic",ExternalID:channelID,CredentialsEncrypted:[]byte("synthetic"),IsDemoFixture:true,IsActive:true,LastSyncStatus:"syncing",SyncRunID:&syncRun,Metadata:"{}",CreatedAt:now,UpdatedAt:now},
		&models.Conversation{ID:convID,TenantID:tenant,ChannelID:channelID,ExternalConversationID:convID,Metadata:"{}",CreatedAt:now,UpdatedAt:now},
		&models.Job{ID:jobID,TenantID:tenant,Name:"R116 sync race",JobType:"intervention",InputChannelIDs:`["`+channelID+`"]`,RulesConfig:"{}",Outputs:"[]",ScheduleType:"manual",IsActive:true,CreatedAt:now,UpdatedAt:now},
		&models.JobRun{ID:originalRun,TenantID:tenant,JobID:jobID,StartedAt:now,FinishedAt:&now,Status:"error",Summary:"{}",CreatedAt:now},
	} { if err := db.DB.Create(row).Error; err != nil { t.Fatal(err) } }
	conv := models.Conversation{ID:convID,TenantID:tenant,ChannelID:channelID,ExternalConversationID:convID}
	initial, err := buildConversationSnapshot(conv, nil, 0)
	if err != nil { t.Fatal(err) }
	snap, err := initial.record(originalRun)
	if err != nil { t.Fatal(err) }
	if err := db.DB.Create(&snap).Error; err != nil { t.Fatal(err) }
	if err := db.DB.Create(&models.InterventionCase{ID:caseID,TenantID:tenant,JobID:jobID,RunID:originalRun,ConversationID:convID,UseCase:"pancake_complaint_wait",Version:1,SnapshotID:&snap.ID,SnapshotDigest:snap.Digest,Eligibility:"WAIT_DATA",Execution:"NOT_RUN",Disposition:"NOT_CREATED",WorkState:"WAITING_ON_DATA",ReasonCode:"SOURCE_INCOMPLETE",ActorID:actor,PolicyVersion:"ccma.synthetic.intervention.v1",PromptVersion:"r116-sync",SchemaVersion:"ccma.intervention.v1",CreatedAt:now,UpdatedAt:now}).Error; err != nil { t.Fatal(err) }
	reservation := SyncReservation{TenantID:tenant,ChannelID:channelID,RunID:syncRun}
	messageID := pkg.NewUUID()
	writer := &SyncEngine{}
	if err := writer.upsertMessage(reservation, convID, channels.SyncedMessage{ExternalID:messageID,SenderType:"customer",Content:"Tôi khiếu nại và chờ đơn hàng",ContentType:"text",SentAt:now,RawData:map[string]interface{}{}}); err != nil { t.Fatal(err) }
	var storedMessage models.Message
	if err := db.DB.Where("tenant_id = ? AND conversation_id = ? AND external_message_id = ?",tenant,convID,messageID).Take(&storedMessage).Error; err != nil { t.Fatal(err) }
	rev, err := CaptureInterventionDataRevision(context.Background(), tenant, actor, caseID, DataRevisionInput{ExpectedVersion:1,ExpectedSnapshotID:snap.ID,ExpectedSnapshotDigest:snap.Digest,Reason:"Đã nhận dữ liệu"})
	if err != nil { t.Fatal(err) }
	request, _, err := RequestInterventionRerun(context.Background(), tenant, actor, caseID, RerunRequestInput{ExpectedVersion:2,ExpectedSnapshotID:snap.ID,ExpectedSnapshotDigest:snap.Digest,DataRevisionID:rev.ID,DataRevisionDigest:rev.Digest,Reason:"Phân tích lại",RequestKey:"r116-sync-race"})
	if err != nil { t.Fatal(err) }
	grant := models.InterventionExecutionGrant{ID:pkg.NewUUID(),TenantID:tenant,JobID:jobID,CaseID:caseID,RequestID:request.ID,DataRevisionID:rev.ID,ActorID:actor,Provider:"alibaba_maas",Model:"qwen3.7-flash",PolicyVersion:"ccma.synthetic.intervention.v1",PromptVersion:"r116-sync",ConfigRef:"r116-sync",IssuerID:actor,EvidenceRef:"r116-synthetic",Deadline:now.Add(5*time.Minute),State:"ISSUED",CreatedAt:now}
	for _, row := range []any{&grant,&models.InterventionBudget{ID:pkg.NewUUID(),TenantID:tenant,JobID:jobID,Cap:1,CreatedAt:now}} { if err := db.DB.Create(row).Error; err != nil { t.Fatal(err) } }
	p := &r116SyncTransport{started:make(chan struct{},1),release:make(chan struct{}),messageID:storedMessage.ID}
	service := &InterventionService{Transport:p,Provider:grant.Provider,Model:grant.Model,ConfigRef:grant.ConfigRef,PolicyVersion:grant.PolicyVersion,PromptVersion:grant.PromptVersion,Prompt:"Synthetic complaint",MaxRequestBytes:6000,Timeout:5*time.Second}
	claim, replay, err := ClaimWaitDataExecution(context.Background(),tenant,actor,caseID,request.ID,WaitDataExecuteInput{ExpectedCaseVersion:3,GrantID:grant.ID},service)
	if err != nil || replay { t.Fatalf("admission: %v replay=%v",err,replay) }
	select { case <-p.started: case <-time.After(5*time.Second): t.Fatal("transport marker timeout") }
	// This is the supported sync writer, not a direct Message insert. It must
	// finish while the provider is held, proving no DB lock spans transport.
	writeDone := make(chan error,1)
	go func() { writeDone <- writer.upsertMessage(reservation,convID,channels.SyncedMessage{ExternalID:messageID,SenderType:"customer",Content:"Tôi khiếu nại, thông tin mới",ContentType:"text",SentAt:now,RawData:map[string]interface{}{}}) }()
	select { case err := <-writeDone: if err != nil { t.Fatal(err) }; case <-time.After(5*time.Second): t.Fatal("sync writer blocked by provider") }
	close(p.release)
	deadline := time.Now().Add(8*time.Second)
	var terminal models.InterventionExecution
	for time.Now().Before(deadline) { if db.DB.Where("id = ?",claim.ID).Take(&terminal).Error == nil && terminal.TerminalCode != "" { break }; time.Sleep(20*time.Millisecond) }
	if terminal.State != "STALE_SOURCE" { t.Fatalf("sync change published: %s",terminal.State) }
	var proposals int64
	if err := db.DB.Model(&models.InterventionProposal{}).Where("tenant_id = ?",tenant).Count(&proposals).Error; err != nil || proposals != 0 { t.Fatalf("proposal count %d: %v",proposals,err) }
	var budget models.InterventionBudget
	if err := db.DB.Where("tenant_id = ? AND job_id = ?",tenant,jobID).Take(&budget).Error; err != nil || budget.Consumed != 1 { t.Fatalf("spent usage %d: %v",budget.Consumed,err) }
}
