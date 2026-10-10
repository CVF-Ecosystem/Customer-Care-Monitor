package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"google.golang.org/genai"
)

func TestUsagePresencePositive(t *testing.T) {
	got := openAICompatibleUsagePresence([]byte("{\"usage\":{\"prompt_tokens\":9007199254740993,\"completion_tokens\":0}}"))
	assertAIUsageCount(t, "input", got.InputTokens, AIUsagePresenceKnown, int64Pointer(9007199254740993))
	assertAIUsageCount(t, "output", got.OutputTokens, AIUsagePresenceKnown, int64Pointer(0))
	if got.SchemaVersion != aiUsagePresenceSchemaVersion || got.Source != AIUsagePresenceSourceOpenAIWire || !got.Complete {
		t.Fatalf("unexpected presence metadata: %#v", got)
	}

	partial := openAICompatibleUsagePresence([]byte("{\"usage\":{\"prompt_tokens\":4}}"))
	assertAIUsageCount(t, "input", partial.InputTokens, AIUsagePresenceKnown, int64Pointer(4))
	assertAIUsageCount(t, "output", partial.OutputTokens, AIUsagePresenceAbsent, nil)
	if partial.Complete {
		t.Fatal("one known token count must not make the pair complete")
	}

	wrongContainer := openAICompatibleUsagePresence([]byte("{\"usage\":[]}"))
	assertAIUsageCount(t, "input", wrongContainer.InputTokens, AIUsagePresenceInvalid, nil)
	assertAIUsageCount(t, "output", wrongContainer.OutputTokens, AIUsagePresenceInvalid, nil)
}

func TestUsagePresenceM01AbsentNullDetector(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want AIUsagePresenceStatus
	}{
		{name: "missing_container", raw: "{}", want: AIUsagePresenceAbsent},
		{name: "missing_token", raw: "{\"usage\":{}}", want: AIUsagePresenceAbsent},
		{name: "null_container", raw: "{\"usage\":null}", want: AIUsagePresenceNull},
		{name: "null_token", raw: "{\"usage\":{\"prompt_tokens\":null}}", want: AIUsagePresenceNull},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := openAICompatibleUsagePresence([]byte(test.raw))
			assertAIUsageCount(t, "input", got.InputTokens, test.want, nil)
		})
	}
}

func TestUsagePresenceM01ExplicitZeroHealthyControl(t *testing.T) {
	got := openAICompatibleUsagePresence([]byte("{\"usage\":{\"prompt_tokens\":0,\"completion_tokens\":0}}"))
	assertAIUsageCount(t, "input", got.InputTokens, AIUsagePresenceKnown, int64Pointer(0))
	assertAIUsageCount(t, "output", got.OutputTokens, AIUsagePresenceKnown, int64Pointer(0))
	if !got.Complete {
		t.Fatal("explicit zero counts are known and complete")
	}
}

func TestUsagePresenceM02InvalidDetector(t *testing.T) {
	invalidValues := []string{"-1", "-0", "1.5", "1e2", "\"1\"", "true", "{}", "[]", "9223372036854775808"}
	for _, rawValue := range invalidValues {
		t.Run(rawValue, func(t *testing.T) {
			raw := []byte("{\"usage\":{\"prompt_tokens\":" + rawValue + ",\"completion_tokens\":7}}")
			got := openAICompatibleUsagePresence(raw)
			assertAIUsageCount(t, "input", got.InputTokens, AIUsagePresenceInvalid, nil)
			assertAIUsageCount(t, "output", got.OutputTokens, AIUsagePresenceKnown, int64Pointer(7))
		})
	}
}

func TestUsagePresenceM02ValidIntegerHealthyControl(t *testing.T) {
	raw := []byte("{\"usage\":{\"prompt_tokens\":9007199254740993,\"completion_tokens\":9223372036854775807}}")
	got := openAICompatibleUsagePresence(raw)
	assertAIUsageCount(t, "input", got.InputTokens, AIUsagePresenceKnown, int64Pointer(9007199254740993))
	assertAIUsageCount(t, "output", got.OutputTokens, AIUsagePresenceKnown, int64Pointer(9223372036854775807))
}

func TestUsagePresenceDuplicateRules(t *testing.T) {
	duplicateContainer := openAICompatibleUsagePresence([]byte("{\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2},\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":4}}"))
	assertAIUsageCount(t, "input", duplicateContainer.InputTokens, AIUsagePresenceInvalid, nil)
	assertAIUsageCount(t, "output", duplicateContainer.OutputTokens, AIUsagePresenceInvalid, nil)
	if duplicateContainer.Diagnostic != "duplicate_usage_container" {
		t.Fatalf("duplicate container diagnostic = %q", duplicateContainer.Diagnostic)
	}

	duplicateInput := openAICompatibleUsagePresence([]byte("{\"usage\":{\"prompt_tokens\":1,\"prompt_tokens\":2,\"completion_tokens\":5}}"))
	assertAIUsageCount(t, "input", duplicateInput.InputTokens, AIUsagePresenceInvalid, nil)
	assertAIUsageCount(t, "output", duplicateInput.OutputTokens, AIUsagePresenceKnown, int64Pointer(5))
}

func TestUsagePresenceRawBoundsAndMalformedJSON(t *testing.T) {
	tooLarge := []byte(strings.Repeat(" ", maxUsagePresenceRawBytes+1))
	large := aiUsagePresenceFromJSON(tooLarge, AIUsagePresenceSourceOpenAIWire, "usage", "prompt_tokens", "completion_tokens")
	assertAIUsageCount(t, "input", large.InputTokens, AIUsagePresenceInvalid, nil)
	assertAIUsageCount(t, "output", large.OutputTokens, AIUsagePresenceInvalid, nil)
	if large.Diagnostic != "raw_too_large" {
		t.Fatalf("oversize diagnostic = %q", large.Diagnostic)
	}

	malformed := openAICompatibleUsagePresence([]byte("{\"usage\":"))
	assertAIUsageCount(t, "input", malformed.InputTokens, AIUsagePresenceInvalid, nil)
	assertAIUsageCount(t, "output", malformed.OutputTokens, AIUsagePresenceInvalid, nil)
	if malformed.Diagnostic != "invalid_json" {
		t.Fatalf("malformed diagnostic = %q", malformed.Diagnostic)
	}
}

func TestUsagePresenceSDKAdapters(t *testing.T) {
	anthropicUnavailable := anthropicUsagePresence(nil)
	assertAIUsageCount(t, "input", anthropicUnavailable.InputTokens, AIUsagePresenceUnavailable, nil)
	assertAIUsageCount(t, "output", anthropicUnavailable.OutputTokens, AIUsagePresenceUnavailable, nil)

	anthropicEmpty := anthropicUsagePresence(&anthropic.Message{})
	assertAIUsageCount(t, "input", anthropicEmpty.InputTokens, AIUsagePresenceUnavailable, nil)
	assertAIUsageCount(t, "output", anthropicEmpty.OutputTokens, AIUsagePresenceUnavailable, nil)

	var anthropicMessage anthropic.Message
	err := json.Unmarshal([]byte("{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-test\",\"content\":[{\"type\":\"text\",\"text\":\"ok\"}],\"stop_reason\":\"end_turn\",\"stop_sequence\":null,\"usage\":{\"input_tokens\":11,\"output_tokens\":12}}"), &anthropicMessage)
	if err != nil {
		t.Fatalf("decode local Anthropic SDK fixture: %v", err)
	}
	anthropicPresence := anthropicUsagePresence(&anthropicMessage)
	assertAIUsageCount(t, "input", anthropicPresence.InputTokens, AIUsagePresenceKnown, int64Pointer(11))
	assertAIUsageCount(t, "output", anthropicPresence.OutputTokens, AIUsagePresenceKnown, int64Pointer(12))
	if anthropicPresence.Source != AIUsagePresenceSourceAnthropicRawJSON || !anthropicPresence.Complete {
		t.Fatalf("unexpected Anthropic presence metadata: %#v", anthropicPresence)
	}

	geminiResponse := &genai.GenerateContentResponse{
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{PromptTokenCount: 31, CandidatesTokenCount: 7},
	}
	unavailable := geminiUsagePresence(geminiResponse)
	assertAIUsageCount(t, "input", unavailable.InputTokens, AIUsagePresenceUnavailable, nil)
	assertAIUsageCount(t, "output", unavailable.OutputTokens, AIUsagePresenceUnavailable, nil)
	if unavailable.Source != AIUsagePresenceSourceGeminiUnavailable || unavailable.Complete {
		t.Fatalf("typed Gemini defaults must stay unavailable: %#v", unavailable)
	}

	geminiResponse.SDKHTTPResponse = &genai.HTTPResponse{Body: "{\"usageMetadata\":{\"promptTokenCount\":31,\"candidatesTokenCount\":7}}"}
	geminiPresence := geminiUsagePresence(geminiResponse)
	assertAIUsageCount(t, "input", geminiPresence.InputTokens, AIUsagePresenceKnown, int64Pointer(31))
	assertAIUsageCount(t, "output", geminiPresence.OutputTokens, AIUsagePresenceKnown, int64Pointer(7))
	if geminiPresence.Source != AIUsagePresenceSourceGeminiHTTPBody || !geminiPresence.Complete {
		t.Fatalf("unexpected Gemini raw body metadata: %#v", geminiPresence)
	}

	geminiResponse.SDKHTTPResponse.Body = ""
	unavailableAgain := geminiUsagePresence(geminiResponse)
	assertAIUsageCount(t, "input", unavailableAgain.InputTokens, AIUsagePresenceUnavailable, nil)
	assertAIUsageCount(t, "output", unavailableAgain.OutputTokens, AIUsagePresenceUnavailable, nil)
}

func assertAIUsageCount(t *testing.T, name string, got AIUsageCount, status AIUsagePresenceStatus, value *int64) {
	t.Helper()
	if got.Status != status {
		t.Fatalf("%s status = %q, want %q", name, got.Status, status)
	}
	if value == nil {
		if got.Value != nil {
			t.Fatalf("%s unexpected value %d for status %q", name, *got.Value, status)
		}
		return
	}
	if got.Value == nil || *got.Value != *value {
		t.Fatalf("%s value = %v, want %d", name, got.Value, *value)
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
