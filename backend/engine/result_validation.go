package engine

import (
	"encoding/json"
	"fmt"
	"strings"
)

// validateAIResult treats provider output as untrusted. A parsed JSON object
// alone is not evidence that the requested evaluation was produced.
func validateAIResult(jobType string, payload []byte) error {
	switch jobType {
	case "qc_analysis":
		var result struct {
			Verdict    *string `json:"verdict"`
			Score      *int    `json:"score"`
			Review     *string `json:"review"`
			Summary    *string `json:"summary"`
			Violations *[]struct {
				Severity    string `json:"severity"`
				Rule        string `json:"rule"`
				Evidence    string `json:"evidence"`
				Explanation string `json:"explanation"`
			} `json:"violations"`
		}
		if err := json.Unmarshal(payload, &result); err != nil {
			return fmt.Errorf("invalid QC JSON: %w", err)
		}
		if result.Verdict == nil || result.Score == nil || result.Review == nil ||
			result.Summary == nil || result.Violations == nil {
			return fmt.Errorf("QC result is missing required fields")
		}
		if *result.Verdict != "PASS" && *result.Verdict != "FAIL" && *result.Verdict != "SKIP" {
			return fmt.Errorf("invalid QC verdict")
		}
		if *result.Score < 0 || *result.Score > 100 || strings.TrimSpace(*result.Review) == "" ||
			strings.TrimSpace(*result.Summary) == "" {
			return fmt.Errorf("invalid QC score or explanation")
		}
		if *result.Verdict == "FAIL" && len(*result.Violations) == 0 {
			return fmt.Errorf("FAIL verdict requires a violation")
		}
		if *result.Verdict != "FAIL" && len(*result.Violations) != 0 {
			return fmt.Errorf("PASS or SKIP cannot contain violations")
		}
		for _, item := range *result.Violations {
			if (item.Severity != "NGHIEM_TRONG" && item.Severity != "CAN_CAI_THIEN") ||
				strings.TrimSpace(item.Rule) == "" || strings.TrimSpace(item.Evidence) == "" ||
				strings.TrimSpace(item.Explanation) == "" {
				return fmt.Errorf("invalid QC violation evidence")
			}
		}
		return nil
	case "classification":
		var result struct {
			Summary *string `json:"summary"`
			Tags    *[]struct {
				RuleName   string   `json:"rule_name"`
				Confidence *float64 `json:"confidence"`
				Evidence   string   `json:"evidence"`
			} `json:"tags"`
		}
		if err := json.Unmarshal(payload, &result); err != nil {
			return fmt.Errorf("invalid classification JSON: %w", err)
		}
		if result.Summary == nil || result.Tags == nil || strings.TrimSpace(*result.Summary) == "" {
			return fmt.Errorf("classification result is missing required fields")
		}
		for _, tag := range *result.Tags {
			if strings.TrimSpace(tag.RuleName) == "" || strings.TrimSpace(tag.Evidence) == "" ||
				tag.Confidence == nil || *tag.Confidence < 0 || *tag.Confidence > 1 {
				return fmt.Errorf("invalid classification tag evidence")
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported analysis type: %s", jobType)
	}
}

// parseBatchResults binds every response to the conversation sent at the same
// position. A model-supplied ID may confirm that binding, never override it.
func parseBatchResults(expectedIDs []string, payload []byte) ([]json.RawMessage, error) {
	var results []json.RawMessage
	if err := json.Unmarshal(payload, &results); err != nil {
		if len(expectedIDs) != 1 {
			return nil, fmt.Errorf("batch response must be a JSON array: %w", err)
		}
		results = []json.RawMessage{payload}
	}
	if len(results) != len(expectedIDs) {
		return nil, fmt.Errorf("batch returned %d results for %d conversations", len(results), len(expectedIDs))
	}
	for i, raw := range results {
		var identity struct {
			ConversationID *string `json:"conversation_id"`
		}
		if err := json.Unmarshal(raw, &identity); err != nil || len(raw) == 0 || raw[0] != '{' {
			return nil, fmt.Errorf("batch result %d must be a JSON object", i)
		}
		if identity.ConversationID != nil && *identity.ConversationID != expectedIDs[i] {
			return nil, fmt.Errorf("batch result %d has a mismatched conversation_id", i)
		}
	}
	return results, nil
}
