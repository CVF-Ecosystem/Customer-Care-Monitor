package engine

import (
	"encoding/json"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
	"strings"
	"testing"
)

func roBoundJob() (models.Job, models.JobRun) {
	j := models.Job{ID: "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA", TenantID: "BBBBBBBB-BBBB-BBBB-BBBB-BBBBBBBBBBBB", JobType: "qc_analysis", RulesContent: "R", SkipConditions: "S"}
	return j, models.JobRun{ID: "CCCCCCCC-CCCC-CCCC-CCCC-CCCCCCCCCCCC", JobID: j.ID, TenantID: j.TenantID}
}
func roHonest(t *testing.T, r *ruleObservationReceipt) {
	t.Helper()
	if r.Version != ruleObservationVersion || r.Scope != "analyzer_job_input_only" || r.RuleAuthority != "JOB_INPUT_OBSERVED" || r.PolicyVersionStatus != "NOT_AVAILABLE" || r.PermissionStatus != "NOT_OBSERVED_AT_ANALYZER" || r.WaitDataStatus != "NOT_IMPLEMENTED" {
		t.Fatalf("invented rule authority: %+v", r)
	}
}

// Constants independently generated with Python hashlib + struct.pack('>Q').
func TestROQCIndependentKnownVector(t *testing.T) {
	j, run := roBoundJob()
	r := newRuleObservation(j, run).freeze()
	roHonest(t, r)
	if r.Fingerprint != "92c65713689d82e31486696ed77e4540f15af5d6da34535bc535cb0d333f5942" {
		t.Fatal("QC effective RulesContent/SkipConditions known vector mismatch")
	}
	if r.TenantID != strings.ToLower(j.TenantID) || r.JobID != strings.ToLower(j.ID) || r.RunID != strings.ToLower(run.ID) || r.MetadataIncomplete || r.JobType != "qc_analysis" || r.FingerprintStatus != "OBSERVED" {
		t.Fatal("QC binding mismatch")
	}
	j.RulesConfig = "inactive"
	if newRuleObservation(j, run).freeze().Fingerprint != r.Fingerprint {
		t.Fatal("QC inactive field affected fingerprint")
	}
	j.SkipConditions = "changed"
	if newRuleObservation(j, run).freeze().Fingerprint == r.Fingerprint {
		t.Fatal("QC skip change absent")
	}
}
func TestROFramingUnicodeEmptyAndClassificationVectors(t *testing.T) {
	vectors := []struct{ kind, a, b, want string }{
		{"qc_analysis", "", "", "cd30574559f9a4524ea8ccd14055493b2444dffcf60c4aa33e5092c1fa9753bd"},
		{"classification", "", "", "9d96001a8cc8710d64df0467bc813e2a51a2cf9fa9616585fd013ffd05958be3"},
		{"classification", `{"a":1,"b":2}`, "", "9c1834f351909d856361a609093a80b7f04c2a84efd09e17381937f11a41c5dd"},
		{"classification", `{"b":2,"a":1}`, "", "c0cfd532b69e980e64e32c96cef9a269e92176fe8de6a7835026f761a011728c"},
		{"qc_analysis", "é 🐦", "không", "41b3bb539011bb96c36e32ff6964e6050d34f9f9e3d6320331e11a8165f5101c"},
		{"qc_analysis", "ab", "c", "f66ca4126e7b83a894f1af1d169d5c189a4b4f1daf02c83241faa11dfb7cfd1f"},
		{"qc_analysis", "a", "bc", "313f92e7b4ab09135ae80c5b9af0f0b2c5d40004d8669b9d692c5ab7d2241d61"},
		{"qc_analysis", " R", "S", "5bbd5f0dd881d66966c1b1fcfc174eff28c2cf79a3595c5f342f8beb7bc315f7"},
	}
	for _, v := range vectors {
		t.Run(v.want[:8], func(t *testing.T) {
			j, run := roBoundJob()
			j.JobType = v.kind
			j.RulesContent = v.a
			j.SkipConditions = v.b
			j.RulesConfig = v.a
			r := newRuleObservation(j, run).freeze()
			roHonest(t, r)
			if r.Fingerprint != v.want {
				t.Fatalf("independent byte-framed vector mismatch: %s", r.Fingerprint)
			}
			if v.kind == "classification" {
				j.RulesContent = "inactive"
				j.SkipConditions = "inactive"
				if newRuleObservation(j, run).freeze().Fingerprint != v.want {
					t.Fatal("classification inactive field affected digest")
				}
			}
		})
	}
	j, run := roBoundJob()
	j.JobType = "classification"
	j.RulesConfig = "invalid JSON"
	r := newRuleObservation(j, run).freeze()
	j.RulesConfig = "invalid JSON "
	if r.FingerprintStatus != "OBSERVED" || r.Fingerprint == newRuleObservation(j, run).freeze().Fingerprint {
		t.Fatal("raw invalid JSON normalized")
	}
}
func TestROInvalidBindingPriorityAndUnsupportedType(t *testing.T) {
	for _, which := range []string{"tenant", "job", "run", "foreign_tenant", "foreign_job", "run_tenant", "run_job"} {
		t.Run(which, func(t *testing.T) {
			j, run := roBoundJob()
			j.JobType = "secret unsupported"
			switch which {
			case "tenant":
				j.TenantID = "secret"
			case "job":
				j.ID = "secret"
			case "run":
				run.ID = "secret"
			case "foreign_tenant":
				run.TenantID = "DDDDDDDD-DDDD-DDDD-DDDD-DDDDDDDDDDDD"
			case "foreign_job":
				run.JobID = "DDDDDDDD-DDDD-DDDD-DDDD-DDDDDDDDDDDD"
			case "run_tenant":
				run.TenantID = "secret"
			case "run_job":
				run.JobID = "secret"
			}
			r := newRuleObservation(j, run).freeze()
			roHonest(t, r)
			if !r.MetadataIncomplete || r.FingerprintStatus != "METADATA_INVALID" || r.TenantID != "" || r.JobID != "" || r.RunID != "" || r.Fingerprint != "" || r.JobType != "" {
				t.Fatal("invalid metadata leaked binding/type/digest")
			}
		})
	}
	j, run := roBoundJob()
	j.JobType = "secret unsupported"
	r := newRuleObservation(j, run).freeze()
	if r.MetadataIncomplete || r.FingerprintStatus != "UNSUPPORTED_JOB_TYPE" || r.Fingerprint != "" || r.JobType != "" {
		t.Fatal("unsupported type was exposed or fabricated")
	}
}
func TestROPrivacyBoundAndIndependentFreezeComposition(t *testing.T) {
	j, run := roBoundJob()
	j.RulesContent = strings.Repeat("SECRET_RULE", 100000)
	j.SkipConditions = "SECRET_SKIP"
	j.Name = "SECRET_NAME"
	o := newRuleObservation(j, run)
	r := o.freeze()
	b, err := json.Marshal(r)
	if err != nil || len(b) > 2048 || strings.Contains(string(b), "SECRET") {
		t.Fatal("rule observation privacy or 2KiB limit")
	}
	r.Fingerprint = "altered"
	if o.freeze().Fingerprint == "altered" {
		t.Fatal("freeze aliased captured receipt")
	}
	prep := newPreparationCollector(j, run, runPlan{}).freeze()
	execution := newExecutionCollector(j, run, runPlan{}).freeze()
	before := observedSummary(map[string]interface{}{"count": 1}, prep, execution)
	after := observedSummary(map[string]interface{}{"count": 1}, prep, execution, o.freeze())
	var env map[string]json.RawMessage
	if err := json.Unmarshal([]byte(after), &env); err != nil {
		t.Fatal(err)
	}
	delete(env, "rule_observation")
	restored, _ := json.Marshal(env)
	var old interface{}
	var now interface{}
	json.Unmarshal([]byte(before), &old)
	json.Unmarshal(restored, &now)
	a, _ := json.Marshal(old)
	c, _ := json.Marshal(now)
	if string(a) != string(c) {
		t.Fatal("rule composition changed old receipts or scalars")
	}
}
