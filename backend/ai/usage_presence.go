package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	"github.com/anthropics/anthropic-sdk-go"
	"google.golang.org/genai"
)

const (
	aiUsagePresenceSchemaVersion = "1"
	maxUsagePresenceRawBytes     = 8 << 20
)

// AIUsagePresenceStatus describes whether a provider response exposed a token count.
type AIUsagePresenceStatus string

const (
	AIUsagePresenceKnown       AIUsagePresenceStatus = "known"
	AIUsagePresenceAbsent      AIUsagePresenceStatus = "absent"
	AIUsagePresenceNull        AIUsagePresenceStatus = "null"
	AIUsagePresenceInvalid     AIUsagePresenceStatus = "invalid"
	AIUsagePresenceUnavailable AIUsagePresenceStatus = "unavailable"
)

// AIUsagePresenceSource identifies the fixed source used to classify token counts.
type AIUsagePresenceSource string

const (
	AIUsagePresenceSourceOpenAIWire        AIUsagePresenceSource = "openai_compatible_wire_json"
	AIUsagePresenceSourceAnthropicRawJSON  AIUsagePresenceSource = "anthropic_sdk_raw_json"
	AIUsagePresenceSourceGeminiHTTPBody    AIUsagePresenceSource = "gemini_sdk_http_body"
	AIUsagePresenceSourceGeminiUnavailable AIUsagePresenceSource = "gemini_sdk_http_body_unavailable"
)

// AIUsageCount carries a numeric value only when Status is known.
type AIUsageCount struct {
	Status AIUsagePresenceStatus `json:"status"`
	Value  *int64                `json:"value,omitempty"`
}

// AIUsagePresence records independently classified input/output counts.
type AIUsagePresence struct {
	SchemaVersion string                `json:"schema_version"`
	Source        AIUsagePresenceSource `json:"source"`
	InputTokens   AIUsageCount          `json:"input_tokens"`
	OutputTokens  AIUsageCount          `json:"output_tokens"`
	Complete      bool                  `json:"complete"`
	Diagnostic    string                `json:"diagnostic,omitempty"`
}

func openAICompatibleUsagePresence(raw []byte) *AIUsagePresence {
	return aiUsagePresenceFromJSON(raw, AIUsagePresenceSourceOpenAIWire, "usage", "prompt_tokens", "completion_tokens")
}

func anthropicUsagePresence(message *anthropic.Message) *AIUsagePresence {
	if message == nil {
		return unavailableAIUsagePresence(AIUsagePresenceSourceAnthropicRawJSON)
	}
	raw := message.RawJSON()
	if raw == "" {
		return unavailableAIUsagePresence(AIUsagePresenceSourceAnthropicRawJSON)
	}
	return aiUsagePresenceFromJSONString(raw, AIUsagePresenceSourceAnthropicRawJSON, "usage", "input_tokens", "output_tokens")
}

func geminiUsagePresence(response *genai.GenerateContentResponse) *AIUsagePresence {
	if response == nil || response.SDKHTTPResponse == nil || response.SDKHTTPResponse.Body == "" {
		return unavailableAIUsagePresence(AIUsagePresenceSourceGeminiUnavailable)
	}
	return aiUsagePresenceFromJSONString(response.SDKHTTPResponse.Body, AIUsagePresenceSourceGeminiHTTPBody, "usageMetadata", "promptTokenCount", "candidatesTokenCount")
}

func aiUsagePresenceFromJSONString(raw string, source AIUsagePresenceSource, containerKey, inputKey, outputKey string) *AIUsagePresence {
	if len(raw) > maxUsagePresenceRawBytes {
		return invalidAIUsagePresence(source, "raw_too_large")
	}
	return aiUsagePresenceFromJSON([]byte(raw), source, containerKey, inputKey, outputKey)
}

func aiUsagePresenceFromJSON(raw []byte, source AIUsagePresenceSource, containerKey, inputKey, outputKey string) *AIUsagePresence {
	if len(raw) > maxUsagePresenceRawBytes {
		return invalidAIUsagePresence(source, "raw_too_large")
	}
	fields, counts, ok := usageJSONObject(raw)
	if !ok {
		return invalidAIUsagePresence(source, "invalid_json")
	}

	presence := newAIUsagePresence(source)
	if counts[containerKey] > 1 {
		presence.InputTokens = usageCountState(AIUsagePresenceInvalid)
		presence.OutputTokens = usageCountState(AIUsagePresenceInvalid)
		presence.Diagnostic = "duplicate_usage_container"
		return presence
	}

	usageRaw, found := fields[containerKey]
	if !found {
		presence.InputTokens = usageCountState(AIUsagePresenceAbsent)
		presence.OutputTokens = usageCountState(AIUsagePresenceAbsent)
		return presence
	}
	if bytes.Equal(bytes.TrimSpace(usageRaw), []byte("null")) {
		presence.InputTokens = usageCountState(AIUsagePresenceNull)
		presence.OutputTokens = usageCountState(AIUsagePresenceNull)
		return presence
	}

	usageFields, usageCounts, ok := usageJSONObject(usageRaw)
	if !ok {
		presence.InputTokens = usageCountState(AIUsagePresenceInvalid)
		presence.OutputTokens = usageCountState(AIUsagePresenceInvalid)
		presence.Diagnostic = "invalid_usage_container"
		return presence
	}

	presence.InputTokens = classifyAIUsageCount(usageFields[inputKey], usageCounts[inputKey] > 0, usageCounts[inputKey])
	presence.OutputTokens = classifyAIUsageCount(usageFields[outputKey], usageCounts[outputKey] > 0, usageCounts[outputKey])
	presence.Complete = presence.InputTokens.Status == AIUsagePresenceKnown && presence.OutputTokens.Status == AIUsagePresenceKnown
	return presence
}

func usageJSONObject(raw []byte) (map[string]json.RawMessage, map[string]int, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	first, err := decoder.Token()
	if err != nil {
		return nil, nil, false
	}
	delim, ok := first.(json.Delim)
	if !ok || delim != '{' {
		return nil, nil, false
	}

	fields := make(map[string]json.RawMessage)
	counts := make(map[string]int)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, nil, false
		}
		key, ok := token.(string)
		if !ok {
			return nil, nil, false
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, nil, false
		}
		counts[key]++
		if counts[key] == 1 {
			fields[key] = append(json.RawMessage(nil), value...)
		}
	}

	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, nil, false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, nil, false
	}
	return fields, counts, true
}

func classifyAIUsageCount(raw json.RawMessage, present bool, occurrences int) AIUsageCount {
	if occurrences > 1 {
		return invalidAIUsageCount()
	}
	if !present {
		return usageCountState(AIUsagePresenceAbsent)
	}
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return usageCountState(AIUsagePresenceNull)
	}
	if len(trimmed) == 0 || (len(trimmed) > 1 && trimmed[0] == '0') {
		return invalidAIUsageCount()
	}
	for _, digit := range trimmed {
		if digit < '0' || digit > '9' {
			return invalidAIUsageCount()
		}
	}
	value, err := strconv.ParseInt(string(trimmed), 10, 64)
	if err != nil || value < 0 {
		return invalidAIUsageCount()
	}
	return knownAIUsageCount(value)
}

func newAIUsagePresence(source AIUsagePresenceSource) *AIUsagePresence {
	return &AIUsagePresence{
		SchemaVersion: aiUsagePresenceSchemaVersion,
		Source:        source,
	}
}

func unavailableAIUsagePresence(source AIUsagePresenceSource) *AIUsagePresence {
	presence := newAIUsagePresence(source)
	presence.InputTokens = usageCountState(AIUsagePresenceUnavailable)
	presence.OutputTokens = usageCountState(AIUsagePresenceUnavailable)
	return presence
}

func invalidAIUsagePresence(source AIUsagePresenceSource, diagnostic string) *AIUsagePresence {
	presence := newAIUsagePresence(source)
	presence.InputTokens = usageCountState(AIUsagePresenceInvalid)
	presence.OutputTokens = usageCountState(AIUsagePresenceInvalid)
	presence.Diagnostic = diagnostic
	return presence
}

func usageCountState(status AIUsagePresenceStatus) AIUsageCount {
	return AIUsageCount{Status: status}
}

func invalidAIUsageCount() AIUsageCount {
	return usageCountState(AIUsagePresenceInvalid)
}

func knownAIUsageCount(value int64) AIUsageCount {
	return AIUsageCount{Status: AIUsagePresenceKnown, Value: &value}
}
