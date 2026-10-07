package engine

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/db/models"
)

const ruleObservationVersion = "ccmai.rule-observation.v1"

// Content identity of the captured Job only; this never grants authority or
// controls selection, cache reuse, permission, provider admission or WAIT_DATA.
type ruleObservationReceipt struct {
	Version             string `json:"version"`
	Scope               string `json:"scope"`
	TenantID            string `json:"tenant_id,omitempty"`
	JobID               string `json:"job_id,omitempty"`
	RunID               string `json:"run_id,omitempty"`
	JobType             string `json:"job_type,omitempty"`
	MetadataIncomplete  bool   `json:"metadata_incomplete"`
	RuleAuthority       string `json:"rule_authority"`
	PolicyVersionStatus string `json:"policy_version_status"`
	PermissionStatus    string `json:"permission_status"`
	WaitDataStatus      string `json:"wait_data_status"`
	FingerprintStatus   string `json:"fingerprint_status"`
	Fingerprint         string `json:"fingerprint,omitempty"`
}

type ruleObservation struct{ receipt ruleObservationReceipt }

func newRuleObservation(job models.Job, run models.JobRun) *ruleObservation {
	r := ruleObservationReceipt{Version: ruleObservationVersion, Scope: "analyzer_job_input_only", RuleAuthority: "JOB_INPUT_OBSERVED", PolicyVersionStatus: "NOT_AVAILABLE", PermissionStatus: "NOT_OBSERVED_AT_ANALYZER", WaitDataStatus: "NOT_IMPLEMENTED", FingerprintStatus: "METADATA_INVALID"}
	r.TenantID = preparationID(job.TenantID, &r.MetadataIncomplete)
	r.JobID = preparationID(job.ID, &r.MetadataIncomplete)
	r.RunID = preparationID(run.ID, &r.MetadataIncomplete)
	if r.MetadataIncomplete || run.TenantID != job.TenantID || run.JobID != job.ID {
		r.MetadataIncomplete = true
		r.TenantID, r.JobID, r.RunID = "", "", ""
		return &ruleObservation{receipt: r}
	}
	fields := []string{"ccmai.rule-input.v1", job.JobType}
	switch job.JobType {
	case "qc_analysis":
		fields = append(fields, job.RulesContent, job.SkipConditions)
	case "classification":
		fields = append(fields, job.RulesConfig)
	default:
		r.FingerprintStatus = "UNSUPPORTED_JOB_TYPE"
		return &ruleObservation{receipt: r}
	}
	r.JobType = job.JobType
	h := sha256.New()
	var length [8]byte
	for _, field := range fields {
		binary.BigEndian.PutUint64(length[:], uint64(len(field)))
		h.Write(length[:])
		h.Write([]byte(field))
	}
	r.Fingerprint = hex.EncodeToString(h.Sum(nil))
	r.FingerprintStatus = "OBSERVED"
	return &ruleObservation{receipt: r}
}

func (o *ruleObservation) freeze() *ruleObservationReceipt {
	if o == nil {
		return nil
	}
	r := o.receipt
	return &r
}
