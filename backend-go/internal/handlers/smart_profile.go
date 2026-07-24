package handlers

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
		heuristic = `(` + junkModelSQL + `)
			AND (
			  m.name ILIKE '%abliterated%' OR m.name ILIKE '%uncensored%' OR m.name ILIKE '%uncen%'
			  OR m.name ILIKE '%heretic%' OR m.name ILIKE '%dolphin%' OR m.name ILIKE '%wizard-vicuna%'
			)
			AND NOT (` + tinyModelSQL + `)
		`
		// Prefer larger uncensored weights over whatever answers in 0.3s
		rankingClause = `
			CASE
			  WHEN m.tag ILIKE '%70b%' OR m.tag ILIKE '%72b%' OR m.tag ILIKE '%32b%' OR m.tag ILIKE '%34b%' THEN 0
			  WHEN m.tag ILIKE '%14b%' OR m.tag ILIKE '%13b%' OR m.tag ILIKE '%12b%' OR m.tag ILIKE '%9b%' THEN 1
			  WHEN m.tag ILIKE '%8b%' OR m.tag ILIKE '%7b%' THEN 2
			  ELSE 3
			END ASC,
			eam.token_per_second DESC NULLS LAST,
			eam.max_connection_time ASC NULLS LAST`
		description = "Abliterated/uncensored models — prefers larger weights, not the tiniest uncensored 7B"

	default:
		// Unknown smart:* → same as fastest but still exclude junk
		heuristic = `(` + junkModelSQL + `)`
		description = "Fastest reply among non-junk catalog models"
	}

	return heuristic, description, rankingClause
}
