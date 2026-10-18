package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
)

const syntheticInterventionPolicy = "ccma.synthetic.intervention.v1"
const syntheticInterventionPromptVersion = "synthetic-activation-v1"
const syntheticInterventionPrompt = "Assess only this invented Pancake demonstration conversation. Identify a customer complaint or customer waiting for a response. Return concise evidence-backed JSON. Never send a message, classify an employee, or infer facts absent from the transcript."

var interventionModelPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,100}$`)
var interventionConfigRefPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,80}$`)

// An explicitly validated Alibaba compatible endpoint uses the existing
// one-call transport. Only its server-owned provider label is corrected; wire
// model, usage, response and error semantics remain those of the transport.
type interventionBrandedTransport struct {
	inner    InterventionTransport
	provider string
}

func (b interventionBrandedTransport) AnalyzeChatOnce(ctx context.Context, prompt, transcript string, maxBytes int64) (ai.AIResponse, error) {
	result, err := b.inner.AnalyzeChatOnce(ctx, prompt, transcript, maxBytes)
	if result.Provider != "" {
		result.Provider = b.provider
	}
	return result, err
}

// LoadSyntheticInterventionService validates the complete opt-in contract
// before touching a key file. A disabled mode never reads a credential.
// readKey is injected in tests; ordinary startup uses ReadSyntheticInterventionKeyFile.
func ReadSyntheticInterventionKeyFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, 8193))
}

func LoadSyntheticInterventionService(c *config.Config, readKey func(string) ([]byte, error)) (*InterventionService, error) {
	if c == nil {
		return nil, errors.New("intervention_config_invalid")
	}
	if c.InterventionMode == "disabled" || c.InterventionMode == "" {
		return nil, nil
	}
	if c.InterventionMode != "synthetic" || readKey == nil || !interventionModelPattern.MatchString(c.InterventionModel) || !interventionConfigRefPattern.MatchString(c.InterventionConfigRef) || !filepath.IsAbs(c.InterventionKeyFile) {
		return nil, errors.New("intervention_config_invalid")
	}
	maxBytes, e1 := strconv.ParseInt(c.InterventionRequestBytes, 10, 64)
	tokens, e2 := strconv.Atoi(c.InterventionOutputTokens)
	seconds, e3 := strconv.Atoi(c.InterventionTimeoutSecs)
	if e1 != nil || e2 != nil || e3 != nil || maxBytes < 512 || maxBytes > 6000 || tokens < 1 || tokens > 768 || seconds < 1 || seconds > 60 {
		return nil, errors.New("intervention_limits_invalid")
	}
	var transport InterventionTransport
	switch c.InterventionProvider {
	case "openai":
		if c.InterventionEndpoint != "https://api.openai.com/v1" {
			return nil, errors.New("intervention_endpoint_invalid")
		}
	case "alibaba_maas":
		if c.InterventionModel != "qwen3.7-flash" {
			return nil, errors.New("intervention_model_invalid")
		}
		if strings.HasSuffix(c.InterventionEndpoint, "/") {
			return nil, errors.New("intervention_endpoint_invalid")
		}
		// The existing helper rejects userinfo, query, fragment, non-443
		// ports, unapproved hosts and path variants.
		probe := ai.NewOpenAIProvider("validation-only", c.InterventionModel, tokens, c.InterventionEndpoint)
		if _, err := probe.WithAlibabaQwen37OnceThinkingDisabled(); err != nil {
			return nil, errors.New("intervention_endpoint_invalid")
		}
	default:
		return nil, errors.New("intervention_provider_invalid")
	}
	parsed, err := url.Parse(c.InterventionEndpoint)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("intervention_endpoint_invalid")
	}
	keyBytes, err := readKey(c.InterventionKeyFile)
	if err != nil || len(keyBytes) == 0 || len(keyBytes) > 8192 {
		return nil, errors.New("intervention_key_unavailable")
	}
	key := strings.TrimSpace(string(keyBytes))
	if key == "" || strings.ContainsAny(key, "\r\n\x00") {
		return nil, errors.New("intervention_key_invalid")
	}
	provider := ai.NewOpenAIProvider(key, c.InterventionModel, tokens, c.InterventionEndpoint)
	if c.InterventionProvider == "alibaba_maas" {
		provider, err = provider.WithAlibabaQwen37OnceThinkingDisabled()
		if err != nil {
			return nil, errors.New("intervention_endpoint_invalid")
		}
		transport = interventionBrandedTransport{inner: provider, provider: "alibaba_maas"}
	} else {
		transport = provider
	}
	// A changed provider/model/endpoint/limit/prompt produces a different grant
	// identity, even if an operator accidentally reuses the human config ref.
	identity := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%d|%s", c.InterventionConfigRef, c.InterventionProvider, c.InterventionModel, c.InterventionEndpoint, syntheticInterventionPromptVersion, maxBytes, tokens, seconds, syntheticInterventionPrompt)
	sum := sha256.Sum256([]byte(identity))
	return &InterventionService{
		Transport: transport,
		Provider:  c.InterventionProvider, Model: c.InterventionModel,
		ConfigRef:       c.InterventionConfigRef + ":" + hex.EncodeToString(sum[:8]),
		PolicyVersion:   syntheticInterventionPolicy,
		PromptVersion:   syntheticInterventionPromptVersion,
		Prompt:          syntheticInterventionPrompt,
		MaxRequestBytes: maxBytes,
		Timeout:         time.Duration(seconds) * time.Second,
	}, nil
}
