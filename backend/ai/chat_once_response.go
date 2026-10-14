package ai

import (
	"errors"
	"net/url"
	"strings"
)

// chatOnceObservedResponseError is constructed only after a bounded HTTP 200
// response has been parsed and its model has matched the requested model.
// It deliberately carries no provider-controlled text.
type chatOnceObservedResponseError struct{}

func (chatOnceObservedResponseError) Error() string { return "observed_response_invalid" }

// IsObservedChatOnceResponse identifies a verified physical response whose
// semantic output is unusable. Other errors leave provider and usage unknown.
func IsObservedChatOnceResponse(err error) bool {
	var observed chatOnceObservedResponseError
	return errors.As(err, &observed)
}

// WithAlibabaQwen37OnceThinkingDisabled returns an immutable configuration for
// one-call analysis. Normal AnalyzeChat and AnalyzeChatBatch ignore this option.
func (p *OpenAICompatibleProvider) WithAlibabaQwen37OnceThinkingDisabled() (*OpenAICompatibleProvider, error) {
	if p == nil || p.maxTokens < 1 || p.maxTokens > 768 || !p.validAlibabaQwen37OnceEndpoint() {
		return nil, errors.New("once_thinking_option_invalid")
	}
	clone := *p
	clone.onceDisableThinking = true
	return &clone, nil
}

func (p *OpenAICompatibleProvider) validAlibabaQwen37OnceEndpoint() bool {
	if p.providerName != "openai" || p.model != "qwen3.7-flash" {
		return false
	}
	u, err := url.Parse(p.baseURL)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || u.RawPath != "" || u.Path != "/compatible-mode/v1" || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	host := u.Hostname()
	suffix := ".ap-southeast-1.maas.aliyuncs.com"
	if host == "" || host != strings.ToLower(host) || !strings.HasSuffix(host, suffix) || u.String() != p.baseURL {
		return false
	}
	prefix := strings.TrimSuffix(host, suffix)
	if prefix == "" || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(prefix, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

// AIUsagePresence is authoritative. The legacy flat int fields are filled only
// when the wire count is known and representable on this process architecture.
func chatOnceFlatKnownCount(count AIUsageCount) int {
	if count.Status != AIUsagePresenceKnown || count.Value == nil || *count.Value < 0 {
		return 0
	}
	value := *count.Value
	if int64(int(value)) != value {
		return 0
	}
	return int(value)
}
