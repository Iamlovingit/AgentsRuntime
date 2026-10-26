package opencode

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVisionModelConfig(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		models := modelsMap([]string{"auto", "text-only"}, map[string]bool{"auto": enabled, "text-only": false})
		raw, _ := json.Marshal(models["auto"])
		if strings.Contains(string(raw), "image") != enabled {
			t.Fatal(string(raw))
		}
		raw, _ = json.Marshal(models["text-only"])
		if strings.Contains(string(raw), "image") {
			t.Fatal("text model advertised images")
		}
	}
	raw, _ := json.Marshal(modelsMap([]string{"legacy"}))
	if strings.Contains(string(raw), "capabilities") {
		t.Fatal("legacy config changed")
	}
}
