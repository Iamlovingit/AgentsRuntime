package openclaw

import (
	"encoding/json"
	"github.com/iamlovingit/clawmanager-agent/internal/gateway"
	"strings"
	"testing"
)

func TestManagedVisionChangesInputOnly(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		doc := map[string]any{}
		mergeOpenClawLLMConfig(doc, gateway.Config{LLMModelIDs: []string{"auto", "text"}, LLMVision: map[string]bool{"auto": enabled, "text": false}})
		models := doc["models"].(map[string]any)["providers"].(map[string]any)["auto"].(map[string]any)["models"].([]any)
		raw, _ := json.Marshal(models[0].(map[string]any)["input"])
		if strings.Contains(string(raw), "image") != enabled {
			t.Fatal(string(raw))
		}
		raw, _ = json.Marshal(models[1].(map[string]any)["input"])
		if strings.Contains(string(raw), "image") {
			t.Fatal(string(raw))
		}
	}
}
