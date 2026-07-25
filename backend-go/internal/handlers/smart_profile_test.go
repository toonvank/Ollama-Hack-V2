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
	if !strings.Contains(rank, ">= 30") || !strings.Contains(rank, ">= 7") {
		t.Fatalf("abliterated ranking should use broad parameter bands: %s", rank)
	}
	if !strings.Contains(rank, "regexp_match(m.name") || !strings.Contains(rank, "regexp_match(m.tag") {
		t.Fatalf("abliterated size detection should inspect both model name and tag: %s", rank)
	}
}

func TestSmartAbliteratedUsesStrongIntentSignals(t *testing.T) {
	heuristic, description, rank := smartProfileConfig("abliterated")

	for _, signal := range []string{"abliterated", "ablated", "heretic", "uncensored", "unfiltered", "unrestricted"} {
		if !strings.Contains(strings.ToLower(heuristic), signal) {
			t.Errorf("abliterated heuristic should recognize %q", signal)
		}
	}
	if strings.Contains(heuristic, "ILIKE '%uncen%'") {
		t.Fatalf("abliterated heuristic must not use the overly broad %%uncen%% match: %s", heuristic)
	}
	for _, excluded := range []string{"embed", "rerank", "guard", "moderation"} {
		if !strings.Contains(strings.ToLower(heuristic), "not ilike '%"+excluded+"%'") {
			t.Errorf("abliterated heuristic should exclude non-chat %q models", excluded)
		}
	}
	if !strings.Contains(description, "modern families first") {
		t.Fatalf("description should explain the quality-first behavior: %s", description)
	}

	explicit := strings.Index(rank, "WHEN m.name ILIKE '%abliterated%'")
	uncensored := strings.Index(rank, "WHEN m.name ILIKE '%uncensored%'")
	dolphin := strings.Index(rank, "WHEN m.name ILIKE '%dolphin%'")
	modern := strings.Index(rank, "WHEN m.name ILIKE '%qwen3%'")
	size := strings.Index(rank, ">= 30")
	latency := strings.LastIndex(rank, "eam.max_connection_time")
	if explicit < 0 || uncensored <= explicit || dolphin <= uncensored || modern <= dolphin || size <= modern || latency <= size {
		t.Fatalf("abliterated rank dimensions are not intent → generation → size → speed:\n%s", rank)
	}
}

func TestSmartAbliteratedRankingAppliesAcrossModels(t *testing.T) {
	_, _, endpointRank := smartProfileConfig("abliterated")
	modelRank := smartProfileModelRankingClause("abliterated", endpointRank)

	if strings.Contains(modelRank, "m.name") || strings.Contains(modelRank, "m.tag") ||
		strings.Contains(modelRank, "eam.") {
		t.Fatalf("outer model ranking contains unavailable inner aliases: %s", modelRank)
	}
	for _, expected := range []string{
		"ranked_models.name",
		"ranked_models.tag",
		"ranked_models.max_connection_time",
		"ranked_models.token_per_second",
	} {
		if !strings.Contains(modelRank, expected) {
			t.Errorf("outer abliterated ranking should contain %q", expected)
		}
	}
}

func TestOtherSmartProfilesKeepReplyTimeModelOrdering(t *testing.T) {
	want := "ranked_models.max_connection_time ASC NULLS LAST, ranked_models.token_per_second DESC NULLS LAST"
	for _, profile := range []string{"large", "cloud"} {
		_, _, endpointRank := smartProfileConfig(profile)
		modelRank := smartProfileModelRankingClause(profile, endpointRank)
		if modelRank != want {
			t.Errorf("%s behavior should remain stable; got %q, want %q", profile, modelRank, want)
		}
	}
}
