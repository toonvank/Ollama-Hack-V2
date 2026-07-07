package handlers

// pseudoModels are virtual model names exposed in /v1/models and /api/tags.
// best-abliterated is the OpenWebUI-friendly alias (no ":" separator).
var pseudoModels = []string{
	"smart:fastest", "smart:large", "smart:small", "smart:coding", "smart:cloud",
	"smart:abliterated", "best-abliterated",
}

// smartProfileConfig returns the SQL filter, UI description, and ranking clause
// for a smart pseudo-model profile. Ranking prioritizes time-to-first-chunk
// (reply latency), then token throughput as a tiebreaker.
func smartProfileConfig(profile string) (heuristic, description, rankingClause string) {
	rankingClause = "eam.max_connection_time ASC NULLS LAST, eam.token_per_second DESC NULLS LAST"

	switch profile {
	case "fastest":
		heuristic = "1=1"
		description = "Fastest to reply across all available models"
	case "large":
		heuristic = "(m.name ILIKE '%70b%' OR m.name ILIKE '%104b%' OR m.name ILIKE '%72b%')"
		description = "Large models (70B, 72B, 104B) with fastest reply time"
	case "small":
		heuristic = "(m.name ILIKE '%8b%' OR m.name ILIKE '%7b%' OR m.name ILIKE '%3b%' OR m.name ILIKE '%1.5b%')"
		description = "Small models (1.5B, 3B, 7B, 8B) with fastest reply time"
	case "coding":
		heuristic = "(m.name ILIKE '%code%' OR m.name ILIKE '%coder%')"
		description = "Code-specialized models with fastest reply time"
	case "cloud":
		heuristic = `(m.name ILIKE '%kimi%' OR m.name ILIKE '%glm%' OR m.name ILIKE '%deepseek%' 
		              OR m.name ILIKE '%gemma%' OR m.name ILIKE '%qwen%' OR m.name ILIKE '%ministral%' 
		              OR m.name ILIKE '%nemotron%' OR m.name ILIKE '%devstral%' OR m.name ILIKE '%minimax%' 
		              OR m.name ILIKE '%rnj%' OR m.name ILIKE '%gemini%' OR m.name ILIKE '%cogito%' 
		              OR m.name ILIKE '%mistral-large%' OR m.name ILIKE '%gpt-oss%')`
		description = "Frontier cloud-class models with fastest reply time"
	case "abliterated":
		heuristic = `(m.name ILIKE '%abliterated%' OR m.name ILIKE '%uncensored%' OR m.name ILIKE '%uncen%'
		              OR m.name ILIKE '%heretic%' OR m.name ILIKE '%heret%' OR m.name ILIKE '%josif%'
		              OR m.name ILIKE '%oym%' OR m.name ILIKE '%dolphin%')`
		description = "Abliterated/uncensored models — no refusals, fastest reply time"
	default:
		heuristic = "1=1"
		description = "Fastest to reply across all available models"
	}

	return heuristic, description, rankingClause
}