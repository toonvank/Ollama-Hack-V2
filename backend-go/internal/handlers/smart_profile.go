package handlers

import (
	"strings"

	"github.com/timlzh/ollama-hack/internal/utils"
)

// pseudoModels are virtual model names exposed in /v1/models and /api/tags.
// best-abliterated is the OpenWebUI-friendly alias (no ":" separator).
var pseudoModels = []string{
	"smart:fastest", "smart:large", "smart:small", "smart:coding", "smart:cloud",
	"smart:abliterated", "best-abliterated",
}

// junkModelSQL excludes probe/test/placeholder catalog noise that otherwise
// wins pure latency ranking (e.g. test_model:latest for smart:fastest).
const junkModelSQL = `
	m.name NOT ILIKE 'test%'
	AND m.name NOT ILIKE '%test_model%'
	AND m.name NOT ILIKE 'probe%'
	AND m.name NOT ILIKE 'model-b-%'
	AND m.name NOT ILIKE '%fake%'
	AND m.name NOT ILIKE '%nonexistent%'
	AND m.name NOT ILIKE '%.lhr.life%'
	AND m.tag NOT ILIKE '%probe%'
`

// tinyModelSQL matches tags that are too small to count as "cloud" / frontier.
const tinyModelSQL = `
	(
	  m.tag ILIKE '%0.5b%' OR m.tag ILIKE '%1b%' OR m.tag ILIKE '%1.5b%'
	  OR m.tag ILIKE '%1.7b%' OR m.tag ILIKE '%2b%' OR m.tag ILIKE '%3b%'
	  OR m.tag ILIKE '%4b%' OR m.tag = 'tiny' OR m.name ILIKE '%tinyllama%'
	  OR m.name ILIKE '%smollm%'
	)
`

// smartProfileConfig returns the SQL filter, UI description, and ranking clause
// for a smart pseudo-model profile.
//
// Ranking is profile-specific: cloud/large prefer real size/family quality first;
// fastest/small still optimize reply latency — but never on junk/test names.
func smartProfileConfig(profile string) (heuristic, description, rankingClause string) {
	// Default: reply latency, then throughput
	rankingClause = "eam.max_connection_time ASC NULLS LAST, eam.token_per_second DESC NULLS LAST"

	switch profile {
	case "fastest":
		// Popular small/fast families only — no test_model, no random HF hashes.
		heuristic = `(` + junkModelSQL + `)
			AND (
			  m.name ILIKE '%llama3.2%' OR m.name ILIKE '%llama3.1%'
			  OR m.name ILIKE '%phi3%' OR m.name ILIKE '%phi-3%'
			  OR m.name ILIKE '%gemma2%' OR m.name ILIKE '%gemma3%'
			  OR m.name ILIKE '%qwen2.5%' OR m.name ILIKE '%qwen3%'
			  OR m.name ILIKE '%mistral%' OR m.name ILIKE '%ministral%'
			)
			AND NOT (` + tinyModelSQL + ` AND m.name ILIKE '%0.5b%')
			AND m.name NOT ILIKE '%coder%' -- coding has its own profile
		`
		description = "Fastest real small/mid models (llama3.x, phi, gemma, qwen, mistral) — excludes test/junk names"

	case "large":
		heuristic = `(` + junkModelSQL + `)
			AND (
			  m.name ILIKE '%70b%' OR m.name ILIKE '%72b%' OR m.name ILIKE '%104b%'
			  OR m.tag ILIKE '%70b%' OR m.tag ILIKE '%72b%' OR m.tag ILIKE '%104b%'
			  OR m.tag ILIKE '%120b%' OR m.tag ILIKE '%123b%' OR m.tag ILIKE '%236b%'
			  OR m.tag ILIKE '%397b%' OR m.tag ILIKE '%480b%' OR m.tag ILIKE '%671b%'
			  OR m.name ILIKE '%kimi%' OR m.name ILIKE '%mistral-large%'
			)
		`
		// Prefer bigger labels, then speed
		rankingClause = `
			CASE
			  WHEN m.tag ILIKE '%671b%' OR m.tag ILIKE '%480b%' OR m.tag ILIKE '%397b%' THEN 0
			  WHEN m.tag ILIKE '%236b%' OR m.tag ILIKE '%123b%' OR m.tag ILIKE '%120b%' THEN 1
			  WHEN m.tag ILIKE '%104b%' OR m.tag ILIKE '%80b%' OR m.tag ILIKE '%72b%' OR m.tag ILIKE '%70b%' THEN 2
			  ELSE 3
			END ASC,
			eam.max_connection_time ASC NULLS LAST,
			eam.token_per_second DESC NULLS LAST`
		description = "Large models (70B+) with best reply time among big weights"

	case "small":
		heuristic = `(` + junkModelSQL + `)
			AND (
			  m.name ILIKE '%llama3.2%' OR m.name ILIKE '%phi3%' OR m.name ILIKE '%gemma2%'
			  OR m.name ILIKE '%qwen2.5%' OR m.name ILIKE '%qwen3%'
			  OR m.tag ILIKE '%7b%' OR m.tag ILIKE '%8b%' OR m.tag ILIKE '%3b%' OR m.tag ILIKE '%1.5b%'
			)
			AND m.name NOT ILIKE '%70b%' AND m.name NOT ILIKE '%72b%'
		`
		description = "Small models (≈1.5B–8B popular families) with fastest reply time"

	case "coding":
		heuristic = `(` + junkModelSQL + `)
			AND (m.name ILIKE '%code%' OR m.name ILIKE '%coder%' OR m.name ILIKE '%codestral%' OR m.name ILIKE '%devstral%')
			AND NOT (` + tinyModelSQL + `)
		`
		// Prefer substantial coders over 0.5b toys
		rankingClause = `
			CASE
			  WHEN m.tag ILIKE '%cloud%' OR m.tag ILIKE '%480b%' OR m.tag ILIKE '%236b%' OR m.tag ILIKE '%32b%' OR m.tag ILIKE '%30b%' THEN 0
			  WHEN m.tag ILIKE '%14b%' OR m.tag ILIKE '%7b%' OR m.tag ILIKE '%8b%' THEN 1
			  ELSE 2
			END ASC,
			eam.max_connection_time ASC NULLS LAST,
			eam.token_per_second DESC NULLS LAST`
		description = "Code-specialized models (prefer real coders, not tiny stubs)"

	case "cloud":
		// Frontier / cloud-class only — NOT qwen2.5:1.5b via broad %qwen% match.
		// Families: Kimi, DeepSeek, Qwen 3.5/3.6/3-next, GLM, plus known cloud tags
		// on other frontier lines (minimax, gpt-oss, nemotron-3, mistral-large, cogito).
		heuristic = `(` + junkModelSQL + `)
			AND NOT (` + tinyModelSQL + `)
			AND m.name NOT ILIKE '%qwen2.5%'
			AND m.name NOT ILIKE '%qwen2:%'
			AND m.name NOT ILIKE 'qwen2'
			AND (
			  -- Kimi line
			  m.name ILIKE '%kimi%'
			  -- DeepSeek line (size gated via NOT tiny + prefer cloud/large below)
			  OR m.name ILIKE '%deepseek%'
			  -- Qwen 3.x modern line (not 2.5)
			  OR m.name ILIKE '%qwen3.6%'
			  OR m.name ILIKE '%qwen3.5%'
			  OR m.name ILIKE '%qwen3-next%'
			  OR m.name ILIKE '%qwen3-coder%'
			  OR (m.name ILIKE 'qwen3%' AND (m.tag ILIKE '%cloud%' OR m.tag ILIKE '%30b%' OR m.tag ILIKE '%32b%' OR m.tag ILIKE '%72b%' OR m.tag ILIKE '%80b%' OR m.tag ILIKE '%235b%' OR m.tag ILIKE '%397b%' OR m.tag ILIKE '%480b%'))
			  -- GLM line
			  OR m.name ILIKE 'glm%'
			  OR m.name ILIKE '%/glm%'
			  OR m.name ILIKE '%glm-%'
			  OR m.name ILIKE '%glm4%'
			  -- Other frontier often served as :cloud
			  OR m.name ILIKE '%mistral-large%'
			  OR m.name ILIKE '%minimax%'
			  OR m.name ILIKE '%gpt-oss%'
			  OR m.name ILIKE '%nemotron-3%'
			  OR m.name ILIKE '%cogito%'
			  OR m.name ILIKE '%devstral%'
			  OR (m.tag ILIKE '%cloud%' AND (
			        m.name ILIKE '%gemma4%' OR m.name ILIKE '%gemma3%'
			        OR m.name ILIKE '%nemotron%' OR m.name ILIKE '%ministral%'
			      ))
			)
			-- Drop small local DeepSeek / plain qwen3 that aren't cloud/large
			AND NOT (
			  m.name ILIKE '%deepseek%'
			  AND m.tag NOT ILIKE '%cloud%'
			  AND m.tag NOT ILIKE '%70b%' AND m.tag NOT ILIKE '%32b%' AND m.tag NOT ILIKE '%30b%'
			  AND (m.tag ILIKE '%1.5b%' OR m.tag ILIKE '%7b%' OR m.tag ILIKE '%8b%' OR m.tag ILIKE '%14b%')
			)
			AND NOT (
			  (m.name = 'qwen3' OR m.name ILIKE 'qwen3:%')
			  AND m.tag NOT ILIKE '%cloud%'
			  AND m.tag NOT ILIKE '%30b%' AND m.tag NOT ILIKE '%32b%' AND m.tag NOT ILIKE '%72b%'
			)
		`
		// Quality-first: preferred frontier families, then other :cloud tags, then size
		rankingClause = `
			CASE
			  WHEN m.name ILIKE '%kimi%' THEN 0
			  WHEN m.name ILIKE '%glm-5%' OR m.name ILIKE '%glm-4.7%' OR m.name ILIKE 'glm-5%' THEN 0
			  WHEN m.name ILIKE '%qwen3.6%' OR m.name ILIKE '%qwen3.5%' OR m.name ILIKE '%qwen3-next%' THEN 0
			  WHEN m.name ILIKE '%deepseek%' AND (m.tag ILIKE '%cloud%' OR m.tag ILIKE '%70b%' OR m.tag = 'latest' OR m.tag ILIKE '%32b%') THEN 0
			  WHEN m.tag ILIKE '%cloud%' THEN 1
			  WHEN m.tag ILIKE '%70b%' OR m.tag ILIKE '%72b%' OR m.tag ILIKE '%80b%'
			    OR m.tag ILIKE '%120b%' OR m.tag ILIKE '%236b%' OR m.tag ILIKE '%397b%'
			    OR m.tag ILIKE '%480b%' OR m.tag ILIKE '%671b%' THEN 2
			  ELSE 3
			END ASC,
			eam.token_per_second DESC NULLS LAST,
			eam.max_connection_time ASC NULLS LAST`
		description = "Frontier cloud-class: Kimi, DeepSeek, Qwen 3.5/3.6, GLM, and :cloud titans — not tiny local toys"

	case "abliterated":
		paramBillions := utils.ParamBillionsSQL("m.name", "m.tag")
		heuristic = `(` + junkModelSQL + `)
			AND (
			  -- Strong, explicit de-alignment labels.
			  m.name ILIKE '%abliterated%' OR m.tag ILIKE '%abliterated%'
			  OR m.name ILIKE '%ablated%' OR m.tag ILIKE '%ablated%'
			  OR m.name ILIKE '%heretic%' OR m.tag ILIKE '%heretic%'
			  -- Other explicit unrestricted-model labels.
			  OR m.name ILIKE '%uncensored%' OR m.tag ILIKE '%uncensored%'
			  OR m.name ILIKE '%uncensor%' OR m.tag ILIKE '%uncensor%'
			  OR m.name ILIKE '%unfiltered%' OR m.tag ILIKE '%unfiltered%'
			  OR m.name ILIKE '%unrestricted%' OR m.tag ILIKE '%unrestricted%'
			  OR m.name ILIKE '%anti-censor%' OR m.tag ILIKE '%anti-censor%'
			  -- Compatibility fallbacks. These labels alone are weaker evidence.
			  OR m.name ILIKE '%dolphin%' OR m.tag ILIKE '%dolphin%'
			  OR m.name ILIKE '%wizard-vicuna%' OR m.tag ILIKE '%wizard-vicuna%'
			)
			AND NOT (` + tinyModelSQL + `)
			AND m.name NOT ILIKE '%embed%'
			AND m.name NOT ILIKE '%rerank%'
			AND m.name NOT ILIKE '%guard%'
			AND m.name NOT ILIKE '%moderation%'
		`
		// Intent confidence comes first: a genuinely abliterated model should not
		// lose to an arbitrary Dolphin/Wizard checkpoint. Within that tier, favor
		// current base-model generations, then broad size bands, then measured
		// responsiveness. Size is deliberately banded so a modern fast 32B can
		// beat a sluggish 70B instead of raw parameter count deciding everything.
		rankingClause = `
			CASE
			  WHEN m.name ILIKE '%abliterated%' OR m.tag ILIKE '%abliterated%'
			    OR m.name ILIKE '%ablated%' OR m.tag ILIKE '%ablated%'
			    OR m.name ILIKE '%heretic%' OR m.tag ILIKE '%heretic%' THEN 0
			  WHEN m.name ILIKE '%uncensored%' OR m.tag ILIKE '%uncensored%'
			    OR m.name ILIKE '%uncensor%' OR m.tag ILIKE '%uncensor%'
			    OR m.name ILIKE '%unfiltered%' OR m.tag ILIKE '%unfiltered%'
			    OR m.name ILIKE '%unrestricted%' OR m.tag ILIKE '%unrestricted%'
			    OR m.name ILIKE '%anti-censor%' OR m.tag ILIKE '%anti-censor%' THEN 1
			  WHEN m.name ILIKE '%dolphin%' OR m.tag ILIKE '%dolphin%' THEN 2
			  ELSE 3 -- legacy wizard-vicuna compatibility
			END ASC,
			CASE
			  WHEN m.name ILIKE '%qwen3%' OR m.name ILIKE '%llama4%'
			    OR m.name ILIKE '%gemma3%' OR m.name ILIKE '%deepseek-v3%'
			    OR m.name ILIKE '%mistral-small-3%' OR m.name ILIKE '%ministral-3%'
			    OR m.name ILIKE '%phi4%' THEN 0
			  WHEN m.name ILIKE '%qwen2.5%' OR m.name ILIKE '%llama3.3%'
			    OR m.name ILIKE '%llama3.2%' OR m.name ILIKE '%mistral-nemo%'
			    OR m.name ILIKE '%mixtral%' THEN 1
			  WHEN m.name ILIKE '%llama3.1%' OR m.name ILIKE '%llama3%' THEN 2
			  ELSE 3
			END ASC,
			CASE
			  WHEN ` + paramBillions + ` >= 30 THEN 0
			  WHEN ` + paramBillions + ` >= 20 THEN 1
			  WHEN ` + paramBillions + ` >= 12 THEN 2
			  WHEN ` + paramBillions + ` >= 7 THEN 3
			  ELSE 4
			END ASC,
			eam.max_connection_time ASC NULLS LAST,
			eam.token_per_second DESC NULLS LAST`
		description = "Abliterated/unrestricted chat models — explicit de-alignment and modern families first, then useful size and speed"

	default:
		// Unknown smart:* → same as fastest but still exclude junk
		heuristic = `(` + junkModelSQL + `)`
		description = "Fastest reply among non-junk catalog models"
	}

	return heuristic, description, rankingClause
}

// smartProfileModelRankingClause returns the ordering for the final list of
// distinct models. Most profiles retain their historical reply-time ordering.
// Abliterated needs its semantic/quality ranking here as well; otherwise the
// per-model DISTINCT query applies that ranking only between endpoints for the
// same model, where all semantic fields are identical.
func smartProfileModelRankingClause(profile, endpointRankingClause string) string {
	if profile != "abliterated" {
		return "ranked_models.max_connection_time ASC NULLS LAST, ranked_models.token_per_second DESC NULLS LAST"
	}

	return strings.NewReplacer(
		"m.name", "ranked_models.name",
		"m.tag", "ranked_models.tag",
		"eam.max_connection_time", "ranked_models.max_connection_time",
		"eam.token_per_second", "ranked_models.token_per_second",
	).Replace(endpointRankingClause)
}
