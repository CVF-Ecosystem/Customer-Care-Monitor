package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/pkg"
)

func exCollector() (*executionCollector, models.Conversation) {
	job := models.Job{ID: pkg.NewUUID(), TenantID: pkg.NewUUID()}
	run := models.JobRun{ID: pkg.NewUUID(), TenantID: job.TenantID, JobID: job.ID}
	return newExecutionCollector(job, run, ordinaryPlan()), models.Conversation{ID: pkg.NewUUID(), TenantID: job.TenantID}
}
func exReconcile(t *testing.T, r *executionReceipt) {
	t.Helper()
	if r.CallsBegun != r.ResponseReturned+r.ErrorReturned+r.Interrupted+r.InFlight || r.ItemCount != r.ItemsSaved+r.ItemsSaveFailed+r.ItemsNotPublished+r.ItemsPending {
		t.Fatalf("execution aggregates do not reconcile %+v", r)
	}
	for _, counts := range []map[string]int{r.UsageWrites, r.Parsing} {
		n := 0
		for _, v := range counts {
			if v < 0 {
				t.Fatal("negative count")
			}
			n += v
		}
		if n != r.CallsBegun {
			t.Fatal("dimension aggregate lost calls")
		}
	}
	retained := 0
	for i, c := range r.Calls {
		if c.Sequence != i+1 || c.ItemCount != c.ItemsSaved+c.ItemsSaveFailed+c.ItemsNotPublished+c.ItemsPending || c.OmittedMembers != c.ItemCount-len(c.MemberIDs) || c.MembersComplete != (c.OmittedMembers == 0) {
			t.Fatalf("call reconciliation %+v", c)
		}
		retained += len(c.MemberIDs)
	}
	if r.OmittedCalls != r.CallsBegun-len(r.Calls) || r.OmittedMembers != r.ItemCount-retained || r.EntriesComplete != (r.OmittedCalls == 0) || r.MembersComplete != (r.OmittedMembers == 0) {
		t.Fatal("omission reconciliation")
	}
}
func TestEXCollectorDimensionsAndFreeze(t *testing.T) {
	c, m := exCollector()
	c.begin("SINGLE", []models.Conversation{m})
	prefix := c.freeze()
	c.returned(nil)
	c.usageBegin()
	c.usageReturned(errors.New("raw-error-sentinel"))
	c.publication(true, nil)
	c.begin("BATCH", []models.Conversation{m, m, m})
	c.returned(nil)
	c.usageBegin()
	c.usageReturned(nil)
	c.parsed(nil)
	c.publication(true, nil)
	c.publication(true, errors.New("raw-save-sentinel"))
	c.publication(false, nil)
	c.begin("SINGLE", []models.Conversation{m})
	c.returned(errors.New("raw-provider-sentinel"))
	c.complete(false, false, 2)
	r := c.freeze()
	exReconcile(t, r)
	if r.CallsBegun != 3 || r.ResponseReturned != 2 || r.ErrorReturned != 1 || r.ItemsSaved != 2 || r.ItemsSaveFailed != 1 || r.ItemsNotPublished != 2 || r.ItemsPending != 0 || r.UsageWrites["WRITE_FAILED"] != 1 || r.StopReason != "EXECUTION_ERROR" {
		t.Fatalf("dimensions %+v", r)
	}
	if prefix.InFlight != 1 || prefix.Calls[0].Invocation != "IN_FLIGHT" || prefix.ItemsPending != 1 {
		t.Fatal("frozen prefix changed")
	}
	r.Calls[0].MemberIDs[0] = "mutated"
	r.UsageWrites["WRITE_FAILED"] = 99
	r.Parsing["ACCEPTED"] = 99
	fresh := c.freeze()
	if fresh.Calls[0].MemberIDs[0] != m.ID || fresh.UsageWrites["WRITE_FAILED"] != 1 || fresh.Parsing["ACCEPTED"] != 1 {
		t.Fatal("frozen copy aliases collector")
	}
}
func TestEXPanicUncertaintyAndReturnedBoundary(t *testing.T) {
	for _, stage := range []string{"provider", "usage", "publication"} {
		t.Run(stage, func(t *testing.T) {
			c, m := exCollector()
			c.begin("SINGLE", []models.Conversation{m})
			if stage != "provider" {
				c.returned(nil)
			}
			if stage == "usage" {
				c.usageBegin()
			}
			if stage == "publication" {
				c.usageBegin()
				c.usageReturned(nil)
			}
			c.panicStop()
			r := c.freeze()
			exReconcile(t, r)
			if r.StopReason != "PANIC" || r.ExecutionComplete || r.ItemsPending != 1 || r.InFlight != 0 {
				t.Fatalf("panic %+v", r)
			}
			if stage == "provider" {
				if r.Interrupted != 1 || r.ResponseReturned != 0 {
					t.Fatal("provider panic not interrupted")
				}
			} else if r.Interrupted != 0 || r.ResponseReturned != 1 {
				t.Fatal("returned response reclassified interrupted")
			}
			if stage == "usage" && r.Calls[0].UsageWrite != "WRITE_OUTCOME_UNKNOWN" {
				t.Fatal("in-flight usage write fabricated outcome")
			}
		})
	}
}
func TestEXBoundsPrivacyAndIndependentPreparationBytes(t *testing.T) {
	c, m := exCollector()
	job := models.Job{ID: c.receipt.JobID, TenantID: c.tenant}
	run := models.JobRun{ID: c.receipt.RunID, TenantID: job.TenantID, JobID: job.ID}
	prep := newPreparationCollector(job, run, ordinaryPlan())
	prep.selection(1)
	prep.start(m)
	prep.finish("PREPARED_FOR_INFERENCE")
	prep.complete()
	before, _ := json.Marshal(prep.freeze())
	for i := 0; i < 205; i++ {
		members := make([]models.Conversation, 205)
		for j := range members {
			members[j] = m
			members[j].ID = fmt.Sprintf("%08x-0000-0000-0000-%012x", i, j)
		}
		c.begin("BATCH", members)
		c.returned(nil)
		c.parsed(errors.New("prompt-transcript-response-secret-sentinel"))
	}
	r := c.freeze()
	exReconcile(t, r)
	if r.CallsBegun != 205 || r.ItemCount != 42025 || len(r.Calls) != 200 || len(r.Calls[0].MemberIDs) != 200 || r.OmittedCalls != 5 || r.OmittedMembers != 41825 {
		t.Fatalf("bounds %+v", r)
	}
	b, _ := json.Marshal(r)
	if len(b) > executionByteLimit || strings.Contains(string(b), "sentinel") {
		t.Fatal("byte bound/privacy")
	}
	c.byteLimit = 1500
	small := c.freeze()
	exReconcile(t, small)
	b, _ = json.Marshal(small)
	if len(b) > 1500 || len(small.Calls) >= 200 || small.CallsBegun != 205 || small.ItemCount != 42025 {
		t.Fatal("serializer aggregate bound")
	}
	composed := observedSummary(map[string]interface{}{"conversations_found": 1}, prep.freeze(), small)
	var env map[string]json.RawMessage
	json.Unmarshal([]byte(composed), &env)
	if string(env["source_preparation"]) != string(before) {
		t.Fatal("preparation receipt bytes changed")
	}
}
func TestEXInvalidMetadataNeverSerializesRawValues(t *testing.T) {
	sentinel := "https://secret.example/prompt-response-rules"
	job := models.Job{ID: sentinel, TenantID: sentinel}
	run := models.JobRun{ID: sentinel, JobID: sentinel, TenantID: sentinel}
	c := newExecutionCollector(job, run, runPlan{mode: analysisMode(sentinel)})
	c.begin("BATCH", []models.Conversation{{ID: sentinel, TenantID: sentinel}, {ID: pkg.NewUUID(), TenantID: "other"}})
	c.returned(errors.New(sentinel))
	c.stop(sentinel)
	r := c.freeze()
	exReconcile(t, r)
	b, _ := json.Marshal(r)
	if !r.MetadataIncomplete || strings.Contains(string(b), sentinel) || len(r.Calls[0].MemberIDs) != 0 || !r.Calls[0].MetadataIncomplete || r.OmittedMembers != 2 {
		t.Fatal("invalid metadata leak")
	}
	valid, m := exCollector()
	m.ID = strings.ToUpper(m.ID)
	valid.begin("SINGLE", []models.Conversation{m})
	if valid.freeze().Calls[0].MemberIDs[0] != strings.ToLower(m.ID) {
		t.Fatal("UUID normalization")
	}
}
