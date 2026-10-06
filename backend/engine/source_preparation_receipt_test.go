package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

const spTenant = "11111111-1111-1111-1111-111111111111"
const spJob = "22222222-2222-2222-2222-222222222222"
const spRun = "33333333-3333-3333-3333-333333333333"
const spConv = "44444444-4444-4444-4444-444444444444"

func spCollector() *preparationCollector {
	return newPreparationCollector(models.Job{ID: spJob, TenantID: spTenant}, models.JobRun{ID: spRun, JobID: spJob, TenantID: spTenant}, ordinaryPlan())
}

func TestPreparationReceiptUnknownZeroAndPrivacy(t *testing.T) {
	c := spCollector()
	r := c.freeze()
	if r.Selected != nil || r.Unvisited != nil || r.SelectionStatus != "NOT_ATTEMPTED" {
		t.Fatalf("unobserved selection %+v", r)
	}
	c.selection(0)
	c.complete()
	r = c.freeze()
	if r.Selected == nil || *r.Selected != 0 || *r.Unvisited != 0 || !r.ScanComplete {
		t.Fatal("zero selection is not unknown")
	}
	unsafe := "SENSITIVE_BODY https://secret.invalid key=forbidden"
	c = newPreparationCollector(models.Job{ID: unsafe, TenantID: spTenant}, models.JobRun{ID: spRun, JobID: unsafe, TenantID: spTenant}, runPlan{mode: analysisMode(unsafe)})
	c.selection(1)
	conv := models.Conversation{ID: unsafe, TenantID: spTenant}
	c.start(conv)
	c.snapshot(conv, &conversationSnapshot{Manifest: snapshotManifest{TenantID: spTenant, ConversationID: unsafe, SchemaVersion: unsafe, Coverage: unsafe, CoverageReasons: []string{unsafe}}, Digest: unsafe})
	c.reference(preparationReference{State: unsafe, EvaluationID: unsafe})
	c.stop(unsafe)
	c.finish(unsafe)
	b, _ := json.Marshal(c.freeze())
	if strings.Contains(string(b), unsafe) || !c.freeze().MetadataIncomplete || !c.freeze().Entries[0].MetadataIncomplete {
		t.Fatalf("unsafe observation %s", b)
	}
	c = spCollector()
	c.selection(1)
	conv = models.Conversation{ID: spConv, TenantID: spTenant}
	c.start(conv)
	c.snapshot(conv, &conversationSnapshot{Manifest: snapshotManifest{TenantID: "foreign", ConversationID: spConv, SchemaVersion: snapshotSchemaV1, Coverage: coverageComplete}, Digest: strings.Repeat("a", 64)})
	c.finish("PREPARED_FOR_INFERENCE")
	e := c.freeze().Entries[0]
	if e.Digest != "" || e.Coverage != "" || e.SchemaVersion != "" || !e.MetadataIncomplete {
		t.Fatalf("foreign metadata retained %+v", e)
	}
}

func TestPreparationReceiptCountsCapsFreezeAndInterruption(t *testing.T) {
	c := spCollector()
	c.selection(250)
	for i := 0; i < 230; i++ {
		c.start(models.Conversation{ID: fmt.Sprintf("44444444-4444-4444-4444-%012d", i), TenantID: spTenant})
		c.finish("EMPTY_SOURCE")
	}
	c.start(models.Conversation{ID: spConv, TenantID: spTenant})
	c.interrupted()
	r := c.freeze()
	if r.Visited != 231 || *r.Unvisited != 19 || r.Counts["EMPTY_SOURCE"] != 230 || r.Counts["PREPARATION_INTERRUPTED"] != 1 || len(r.Entries) != 200 || r.OmittedEntries != 31 || r.ScanComplete || r.EntriesComplete {
		t.Fatalf("counts/cap %+v", r)
	}
	for i, e := range r.Entries {
		if e.ConversationID != fmt.Sprintf("44444444-4444-4444-4444-%012d", i) {
			t.Fatal("first200 original order lost")
		}
	}
	b, _ := json.Marshal(r)
	if len(b) > preparationByteLimit {
		t.Fatalf("envelope %d", len(b))
	}
	frozen, _ := json.Marshal(r)
	c.receipt.Counts["EMPTY_SOURCE"] = 999
	c.receipt.Entries[0].Outcome = "SNAPSHOT_ERROR"
	*c.receipt.Selected = 999
	again, _ := json.Marshal(r)
	if string(frozen) != string(again) {
		t.Fatal("frozen copy changed")
	}
	c.receipt.Counts["EMPTY_SOURCE"] = 230
	*c.receipt.Selected = 250
	c.byteLimit = 900
	r = c.freeze()
	b, _ = json.Marshal(r)
	if len(b) > 900 || len(r.Entries) >= 200 || r.Counts["EMPTY_SOURCE"] != 230 || r.OmittedEntries != r.Visited-len(r.Entries) {
		t.Fatalf("byte cap %d %+v", len(b), r)
	}
}

func TestPreparationReceiptMetadataAndSummaryCompatibility(t *testing.T) {
	c := spCollector()
	c.selection(1)
	conv := models.Conversation{ID: spConv, TenantID: spTenant}
	c.start(conv)
	c.snapshot(conv, &conversationSnapshot{Manifest: snapshotManifest{TenantID: spTenant, ConversationID: spConv, SchemaVersion: snapshotSchemaV1, Coverage: coveragePartial, CoverageReasons: []string{reasonAttachmentNotRepresented, reasonAttachmentNotRepresented, "INVALID"}}, Digest: strings.Repeat("A", 64)})
	c.reference(preparationReference{State: "VERIFIED", EvaluationID: spJob, RunID: spRun, SnapshotID: spConv})
	c.finish("PREPARED_FOR_INFERENCE")
	c.complete()
	r := c.freeze()
	c.receipt.Entries[0].CoverageReasons[0] = reasonNoMessages
	e := r.Entries[0]
	if e.CoverageReasons[0] != reasonAttachmentNotRepresented {
		t.Fatal("frozen reason slice changed")
	}
	if e.Digest != strings.Repeat("a", 64) || len(e.CoverageReasons) != 1 || e.Reference.State != "VERIFIED" || !e.MetadataIncomplete {
		t.Fatalf("metadata %+v", e)
	}
	original := map[string]interface{}{"conversations_found": float64(2), "conversations_errors": float64(1)}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(preparationSummary(map[string]interface{}{"conversations_found": 2, "conversations_errors": 1}, r)), &decoded); err != nil {
		t.Fatal(err)
	}
	delete(decoded, "source_preparation")
	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("scalars changed %+v", decoded)
	}
}
