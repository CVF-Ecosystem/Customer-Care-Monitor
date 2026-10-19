package engine

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/config"
)

func startupFixtureConfig() *config.Config {
	keyPath := "/keys/intervention"
	if runtime.GOOS == "windows" {
		keyPath = `C:\keys\intervention`
	}
	return &config.Config{InterventionMode: "synthetic", InterventionProvider: "openai", InterventionModel: "gpt-5", InterventionEndpoint: "https://api.openai.com/v1", InterventionKeyFile: keyPath, InterventionConfigRef: "synthetic-operator", InterventionRequestBytes: "6000", InterventionOutputTokens: "128", InterventionTimeoutSecs: "15"}
}

func TestSyntheticInterventionStartupFailClosedBeforeKeyRead(t *testing.T) {
	reads := 0
	read := func(string) ([]byte, error) { reads++; return []byte("synthetic-key"), nil }
	c := startupFixtureConfig()
	c.InterventionMode = "disabled"
	if s, err := LoadSyntheticInterventionService(c, read); err != nil || s != nil || reads != 0 {
		t.Fatalf("disabled: service=%v err=%v reads=%d", s, err, reads)
	}
	c = startupFixtureConfig()
	c.InterventionEndpoint = "https://api.openai.com.evil.invalid/v1"
	if s, err := LoadSyntheticInterventionService(c, read); err == nil || s != nil || reads != 0 {
		t.Fatalf("bad host read key: err=%v reads=%d", err, reads)
	}
	c = startupFixtureConfig()
	c.InterventionOutputTokens = "769"
	if s, err := LoadSyntheticInterventionService(c, read); err == nil || s != nil || reads != 0 {
		t.Fatalf("bad cap read key: err=%v reads=%d", err, reads)
	}
	c = startupFixtureConfig()
	c.InterventionKeyFile = "relative/key"
	if s, err := LoadSyntheticInterventionService(c, read); err == nil || s != nil || reads != 0 {
		t.Fatalf("relative key read: err=%v reads=%d", err, reads)
	}
	c = startupFixtureConfig()
	c.InterventionProvider = "alibaba_maas"
	c.InterventionModel = "qwen3.7-flash"
	c.InterventionEndpoint = "https://tenant.ap-southeast-1.maas.aliyuncs.com:444/compatible-mode/v1"
	if s, err := LoadSyntheticInterventionService(c, read); err == nil || s != nil || reads != 0 {
		t.Fatalf("bad Alibaba port read: err=%v reads=%d", err, reads)
	}
	c = startupFixtureConfig()
	s, err := LoadSyntheticInterventionService(c, read)
	if err != nil || !s.SyntheticConfigured() || reads != 1 || s.Provider != "openai" {
		t.Fatalf("valid openai: service=%v err=%v reads=%d", s, err, reads)
	}
	c.InterventionModel = "gpt-5-mini"
	newService, err := LoadSyntheticInterventionService(c, read)
	if err != nil || newService.ConfigRef == s.ConfigRef {
		t.Fatalf("changed model reused config identity: %v", err)
	}
}

type onceObservedStub struct {
	response ai.AIResponse
	err      error
}

func (s onceObservedStub) AnalyzeChatOnce(context.Context, string, string, int64) (ai.AIResponse, error) {
	return s.response, s.err
}

func TestSyntheticAlibabaStartupPreservesOnceResponseAndError(t *testing.T) {
	c := startupFixtureConfig()
	c.InterventionProvider, c.InterventionModel, c.InterventionEndpoint = "alibaba_maas", "qwen3.7-flash", "https://tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1"
	s, err := LoadSyntheticInterventionService(c, func(string) ([]byte, error) { return []byte("synthetic-key\n"), nil })
	if err != nil || !s.SyntheticConfigured() || s.Provider != "alibaba_maas" {
		t.Fatalf("valid Alibaba rejected: %v", err)
	}
	marker := errors.New("observed-response")
	wrapped := interventionBrandedTransport{inner: onceObservedStub{response: ai.AIResponse{Provider: "openai", Model: "qwen3.7-flash", Content: "{}"}, err: marker}, provider: "alibaba_maas"}
	response, got := wrapped.AnalyzeChatOnce(context.Background(), "", "", 6000)
	if !errors.Is(got, marker) || response.Provider != "alibaba_maas" || response.Model != "qwen3.7-flash" || response.Content != "{}" {
		t.Fatalf("brand changed observation: %+v err=%v", response, got)
	}
	for _, invalid := range []string{"https://tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1?x=1", "https://user@tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1", "https://tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1/", "http://tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1"} {
		c.InterventionEndpoint = invalid
		if _, err := LoadSyntheticInterventionService(c, func(string) ([]byte, error) { t.Fatal("invalid endpoint read secret"); return nil, nil }); err == nil {
			t.Fatalf("accepted endpoint %q", invalid)
		}
	}
	c.InterventionEndpoint = "https://tenant.ap-southeast-1.maas.aliyuncs.com/compatible-mode/v1"
	if _, err := LoadSyntheticInterventionService(c, func(string) ([]byte, error) { return []byte(strings.Repeat("x", 8193)), nil }); err == nil {
		t.Fatal("oversize key accepted")
	}
}
