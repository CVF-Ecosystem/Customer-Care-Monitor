package engine

import "testing"

func TestValidateAIResultRejectsIncompleteOrFabricatedEvidence(t *testing.T) {
	tests := []struct {
		name, kind, body string
		valid            bool
	}{
		{"valid pass", "qc_analysis", `{"verdict":"PASS","score":92,"review":"Good response","summary":"Resolved","violations":[]}`, true},
		{"missing verdict", "qc_analysis", `{"score":92,"review":"Good","summary":"Resolved","violations":[]}`, false},
		{"score outside range", "qc_analysis", `{"verdict":"PASS","score":120,"review":"Good","summary":"Resolved","violations":[]}`, false},
		{"fail without evidence", "qc_analysis", `{"verdict":"FAIL","score":20,"review":"Poor","summary":"Unresolved","violations":[]}`, false},
		{"valid failure", "qc_analysis", `{"verdict":"FAIL","score":20,"review":"Poor","summary":"Unresolved","violations":[{"severity":"NGHIEM_TRONG","rule":"Greeting","evidence":"Exact quote","explanation":"No greeting"}]}`, true},
		{"invalid confidence", "classification", `{"summary":"Question","tags":[{"rule_name":"Inquiry","confidence":1.5,"evidence":"Exact quote"}]}`, false},
		{"valid classification", "classification", `{"summary":"Question","tags":[{"rule_name":"Inquiry","confidence":0.7,"evidence":"Exact quote"}]}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAIResult(tc.kind, []byte(tc.body))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, got error=%v", tc.valid, err)
			}
		})
	}
}

func TestParseBatchResultsPreservesConversationBinding(t *testing.T) {
	tests := []struct {
		name  string
		ids   []string
		body  string
		valid bool
	}{
		{"matching ids", []string{"a", "b"}, `[{"conversation_id":"a"},{"conversation_id":"b"}]`, true},
		{"ordered without ids", []string{"a", "b"}, `[{},{}]`, true},
		{"swapped ids", []string{"a", "b"}, `[{"conversation_id":"b"},{"conversation_id":"a"}]`, false},
		{"missing result", []string{"a", "b"}, `[{}]`, false},
		{"extra result", []string{"a"}, `[{},{}]`, false},
		{"object for multiple chats", []string{"a", "b"}, `{}`, false},
		{"object for one chat", []string{"a"}, `{"conversation_id":"a"}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseBatchResults(tc.ids, []byte(tc.body))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, got error=%v", tc.valid, err)
			}
		})
	}
}
