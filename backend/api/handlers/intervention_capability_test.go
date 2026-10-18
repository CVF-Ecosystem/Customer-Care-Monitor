package handlers_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/ai"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/api/handlers"
	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/engine"
	"github.com/gin-gonic/gin"
)

type capabilityNeverCalled struct{}

func (capabilityNeverCalled) AnalyzeChatOnce(context.Context, string, string, int64) (ai.AIResponse, error) {
	panic("capability inspection must never call a provider")
}
func TestInterventionCapabilityHonestAndSecretFree(t *testing.T) {
	oldRunner, oldExecutor := handlers.InterventionRunner, engine.WaitDataExecutor
	defer func() { handlers.InterventionRunner, engine.WaitDataExecutor = oldRunner, oldExecutor }()
	configured := &engine.InterventionService{Transport: capabilityNeverCalled{}, Provider: "private-provider", Model: "private-model", ConfigRef: "private-config", PolicyVersion: "ccma.synthetic.intervention.v1", PromptVersion: "v1", Prompt: "private-prompt", MaxRequestBytes: 6000, Timeout: 10 * time.Second}
	wrongPolicy := *configured
	wrongPolicy.PolicyVersion = "unapproved"
	incomplete := *configured
	incomplete.Transport = nil
	for _, tc := range []struct {
		name              string
		runner, executor  *engine.InterventionService
		generation, rerun bool
	}{
		{"ordinary startup", nil, nil, false, false},
		{"partial", configured, nil, true, false},
		{"configured synthetic", configured, configured, true, true},
		{"wrong policy", &wrongPolicy, &wrongPolicy, false, false},
		{"missing transport", &incomplete, &incomplete, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handlers.InterventionRunner, engine.WaitDataExecutor = tc.runner, tc.executor
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			handlers.GetInterventionCapability(c)
			if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status/cache %d %v", w.Code, w.Header())
			}
			var got map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if len(got) != 3 || got["scope"] != "synthetic_only" || got["generation_configured"] != tc.generation || got["rerun_configured"] != tc.rerun {
				t.Fatalf("unexpected public capability: %v", got)
			}
		})
	}
}
