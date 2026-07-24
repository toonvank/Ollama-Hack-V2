package handlers

import (
	"strings"
	"testing"
)

func TestSmartCloudExcludesQwen25AndTiny(t *testing.T) {
	h, desc, rank := smartProfileConfig("cloud")
	if !strings.Contains(desc, "Kimi") || !strings.Contains(desc, "GLM") {
		t.Fatalf("cloud description should mention frontier families: %s", desc)
	}
	// Must exclude classic tiny qwen2.5 path that used to win pure-latency ranking
	if !strings.Contains(h, "qwen2.5") || !strings.Contains(h, "NOT ILIKE '%qwen2.5%'") {
		t.Fatalf("cloud heuristic must exclude qwen2.5: %s", h)
	}
	for _, fam := range []string{"kimi", "deepseek", "qwen3.6", "glm"} {
		if !strings.Contains(strings.ToLower(h), fam) {
			t.Fatalf("cloud heuristic must include family %s", fam)
		}
	}
	// Ranking must prefer :cloud / quality over pure latency alone
	if !strings.Contains(rank, "cloud") {
		t.Fatalf("cloud ranking should prefer cloud tags: %s", rank)
	}
}

func TestSmartFastestExcludesTestJunk(t *testing.T) {
	h, _, _ := smartProfileConfig("fastest")
	if !strings.Contains(h, "test_model") && !strings.Contains(h, "junkModelSQL") {
		// inlined junk filter
		if !strings.Contains(h, "NOT ILIKE 'test%'") {
			t.Fatalf("fastest must exclude test names: %s", h)
		}
	}
	if !strings.Contains(h, "NOT ILIKE 'test%'") {
		t.Fatalf("fastest must exclude test names: %s", h)
	}
}

func TestSmartAbliteratedPrefersLarger(t *testing.T) {
	_, _, rank := smartProfileConfig("abliterated")
	if !strings.Contains(rank, "70b") {
		t.Fatalf("abliterated ranking should prefer larger weights: %s", rank)
	}
}
