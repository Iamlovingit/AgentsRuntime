package hermes

import (
	"github.com/iamlovingit/clawmanager-agent/internal/gateway"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
)

func TestManagedVisionKeepsUnrelatedAuxiliaryConfiguration(t *testing.T) {
	base := "# user comment\nauxiliary:\n  compression:\n    model: keep-me\n  vision:\n    provider: openrouter\n    api_key: old-key\n    timeout: 42\n    extra_body:\n      model: unauthorized\nterminal:\n  timeout: 500\n"
	cfg := gateway.Config{LLMBaseURL: "https://gateway.example/llm", LLMVision: map[string]bool{"auto": true}}
	got, err := mergeManagedVisionYAML(base, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatal(err)
	}
	aux := parsed["auxiliary"].(map[string]any)
	vision := aux["vision"].(map[string]any)
	if vision["base_url"] != cfg.LLMBaseURL || vision["model"] != "auto" || vision["provider"] != "custom" || vision["api_key"] != "" {
		t.Fatal(vision)
	}
	if vision["timeout"] != 42 || !strings.Contains(got, "keep-me") || !strings.Contains(got, "# user comment") || !strings.Contains(got, "500") || strings.Contains(got, "old-key") || strings.Contains(got, "unauthorized") {
		t.Fatal(got)
	}
	again, err := mergeManagedVisionYAML(got, cfg)
	if err != nil || again != got {
		t.Fatal("not idempotent")
	}
	legacy, err := mergeManagedVisionYAML(base, gateway.Config{})
	if err != nil || legacy != base {
		t.Fatal("legacy changed")
	}
	if _, err := mergeManagedVisionYAML("auxiliary: wrong", cfg); err == nil {
		t.Fatal("invalid mapping overwritten")
	}
}
