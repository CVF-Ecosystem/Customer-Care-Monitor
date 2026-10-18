package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatibleProvider nói chuyện với bất kỳ dịch vụ nào theo chuẩn
// /chat/completions của OpenAI — dùng được cho OpenAI, xAI (Grok), và các proxy
// như OpenRouter hay LiteLLM.
//
// Cố ý gọi HTTP trực tiếp thay vì thêm SDK: phần API dùng tới chỉ là một endpoint
// với vài trường, và mỗi SDK lại ràng buộc thêm một nhà cung cấp cụ thể.
type OpenAICompatibleProvider struct {
	providerName        string
	apiKey              string
	model               string
	baseURL             string
	maxTokens           int
	onceDisableThinking bool
}

const (
	openAIDefaultBaseURL = "https://api.openai.com/v1"
	xaiDefaultBaseURL    = "https://api.x.ai/v1"
	// Phản hồi chấm chat không bao giờ dài tới mức này; chặn để một phản hồi
	// bất thường không nuốt hết bộ nhớ.
	maxCompletionResponseSize = 8 << 20
)

// NewOpenAIProvider tạo provider cho OpenAI.
func NewOpenAIProvider(apiKey, model string, maxTokens int, baseURL string) *OpenAICompatibleProvider {
	if model == "" {
		model = "gpt-5"
	}
	if baseURL == "" {
		baseURL = openAIDefaultBaseURL
	}
	return newOpenAICompatible("openai", apiKey, model, maxTokens, baseURL)
}

// NewXAIProvider tạo provider cho xAI (Grok).
func NewXAIProvider(apiKey, model string, maxTokens int, baseURL string) *OpenAICompatibleProvider {
	if model == "" {
		model = "grok-4"
	}
	if baseURL == "" {
		baseURL = xaiDefaultBaseURL
	}
	return newOpenAICompatible("xai", apiKey, model, maxTokens, baseURL)
}

func newOpenAICompatible(name, apiKey, model string, maxTokens int, baseURL string) *OpenAICompatibleProvider {
	if maxTokens <= 0 {
		maxTokens = 16384
	}
	return &OpenAICompatibleProvider{
		providerName: name,
		apiKey:       apiKey,
		model:        model,
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		maxTokens:    maxTokens,
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
			Refusal string `json:"refusal"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	// OpenAI trả error là object, xAI trả là chuỗi — nhận cả hai dạng.
	Error json.RawMessage `json:"error"`
}

// errorMessage rút thông báo lỗi ra khỏi phần error, bất kể nó là chuỗi hay object.
func (r chatCompletionResponse) errorMessage() string {
	if len(r.Error) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(r.Error, &asString); err == nil {
		return asString
	}
	var asObject struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(r.Error, &asObject); err == nil {
		return asObject.Message
	}
	return ""
}

func (p *OpenAICompatibleProvider) AnalyzeChat(ctx context.Context, systemPrompt string, chatTranscript string) (AIResponse, error) {
	return withRetry(ctx, p.providerName, func() (AIResponse, error) {
		body := map[string]any{
			"model": p.model,
			"messages": []chatMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: chatTranscript},
			},
		}
		// OpenAI đã đổi sang max_completion_tokens và từ chối max_tokens trên các
		// model đời mới; xAI và phần lớn proxy vẫn dùng tên cũ.
		if p.providerName == "openai" {
			body["max_completion_tokens"] = p.maxTokens
		} else {
			body["max_tokens"] = p.maxTokens
		}

		payload, err := json.Marshal(body)
		if err != nil {
			return AIResponse{}, fmt.Errorf("%s marshal request: %w", p.providerName, err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
		if err != nil {
			return AIResponse{}, fmt.Errorf("%s create request: %w", p.providerName, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := NewHTTPClientWithTimeout().Do(req)
		if err != nil {
			return AIResponse{}, fmt.Errorf("%s api error: %w", p.providerName, err)
		}
		defer resp.Body.Close()

		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxCompletionResponseSize))
		if err != nil {
			return AIResponse{}, fmt.Errorf("%s read response: %w", p.providerName, err)
		}

		var parsed chatCompletionResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			// Mã lỗi phải nằm trong thông báo để lớp thử lại nhận ra lỗi tạm thời.
			return AIResponse{}, fmt.Errorf("%s api error: mã %d, phản hồi không đọc được", p.providerName, resp.StatusCode)
		}

		if resp.StatusCode != http.StatusOK {
			return AIResponse{}, fmt.Errorf("%s api error: mã %d %s", p.providerName, resp.StatusCode, parsed.errorMessage())
		}

		if len(parsed.Choices) == 0 {
			return AIResponse{}, fmt.Errorf("%s api returned no choices", p.providerName)
		}
		choice := parsed.Choices[0]
		if choice.Message.Refusal != "" {
			return AIResponse{}, fmt.Errorf("%s từ chối yêu cầu: %s", p.providerName, choice.Message.Refusal)
		}
		if strings.TrimSpace(choice.Message.Content) == "" {
			return AIResponse{}, fmt.Errorf("%s api returned empty content", p.providerName)
		}

		return AIResponse{
			Content:       choice.Message.Content,
			Model:         p.model,
			Provider:      p.providerName,
			InputTokens:   parsed.Usage.PromptTokens,
			OutputTokens:  parsed.Usage.CompletionTokens,
			UsagePresence: openAICompatibleUsagePresence(raw),
		}, nil
	})
}

func (p *OpenAICompatibleProvider) AnalyzeChatBatch(ctx context.Context, systemPrompt string, items []BatchItem) (AIResponse, error) {
	batchPrompt := WrapBatchPrompt(systemPrompt, len(items))
	batchTranscript := FormatBatchTranscript(items)
	return p.AnalyzeChat(ctx, batchPrompt, batchTranscript)
}

// AnalyzeChatOnce is reserved for a caller that has durably consumed one
// physical-call slot. It never redirects, retries, or returns provider text in errors.
func (p *OpenAICompatibleProvider) AnalyzeChatOnce(ctx context.Context, systemPrompt, transcript string, maxRequestBytes int64) (AIResponse, error) {
	if p == nil {
		return AIResponse{}, fmt.Errorf("provider_unavailable")
	}
	if p.onceDisableThinking && !p.validAlibabaQwen37OnceEndpoint() {
		return AIResponse{}, fmt.Errorf("once_thinking_option_invalid")
	}
	if p.maxTokens < 1 || p.maxTokens > 768 {
		return AIResponse{}, fmt.Errorf("completion_limit_invalid")
	}
	if maxRequestBytes <= 0 || maxRequestBytes > 6000 {
		return AIResponse{}, fmt.Errorf("request_limit_invalid")
	}
	body := map[string]any{"model": p.model, "messages": []chatMessage{{Role: "system", Content: systemPrompt}, {Role: "user", Content: transcript}}, "max_completion_tokens": p.maxTokens}
	if p.onceDisableThinking {
		body["enable_thinking"] = false
	}
	payload, err := json.Marshal(body)
	if err != nil || int64(len(payload)) > maxRequestBytes {
		return AIResponse{}, fmt.Errorf("request_too_large")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return AIResponse{}, fmt.Errorf("request_invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	client := &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return AIResponse{}, fmt.Errorf("transport_unknown")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return AIResponse{}, fmt.Errorf("provider_status_%d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return AIResponse{}, fmt.Errorf("response_invalid")
	}
	fields, counts, objectOK := usageJSONObject(raw)
	if !objectOK || counts["model"] != 1 {
		return AIResponse{}, fmt.Errorf("response_invalid")
	}
	var wireModel string
	if json.Unmarshal(fields["model"], &wireModel) != nil || wireModel == "" {
		return AIResponse{}, fmt.Errorf("response_model_unknown")
	}
	if wireModel != p.model {
		return AIResponse{}, fmt.Errorf("response_model_mismatch")
	}
	observed := AIResponse{Model: wireModel, Provider: p.providerName, UsagePresence: openAICompatibleUsagePresence(raw)}
	observed.InputTokens = chatOnceFlatKnownCount(observed.UsagePresence.InputTokens)
	observed.OutputTokens = chatOnceFlatKnownCount(observed.UsagePresence.OutputTokens)
	var parsed chatCompletionResponse
	if json.Unmarshal(raw, &parsed) != nil {
		return observed, chatOnceObservedResponseError{}
	}
	providerError := bytes.TrimSpace(parsed.Error)
	if counts["choices"] != 1 || len(parsed.Choices) != 1 || (len(providerError) != 0 && !bytes.Equal(providerError, []byte("null"))) {
		return observed, chatOnceObservedResponseError{}
	}
	choice := parsed.Choices[0]
	if choice.Message.Refusal != "" || choice.FinishReason != "stop" || strings.TrimSpace(choice.Message.Content) == "" {
		return observed, chatOnceObservedResponseError{}
	}
	observed.Content = choice.Message.Content
	return observed, nil
}
