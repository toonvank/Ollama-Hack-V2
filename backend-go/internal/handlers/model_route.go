package handlers

import (
	"fmt"
	"log"
	"strings"

	"github.com/timlzh/ollama-hack/internal/services"
)

// routableEndpointSQL is the shared predicate for "this model can actually be
// raced right now" — eam available, endpoint available, health not in active
// cooldown, AND a successful measured generate (token_per_second > 0).
// Unmeasured / tags-only / OpenAI-list-only links must never enter the race pool
// (they produce 401/timeout junk and empty cloud-cascade fallbacks).
// Must stay in sync with bestEndpointsForModel and /v1/models.
const routableEndpointSQL = `
	m.enabled = TRUE
	AND eam.status = 'available'
	AND e.status = 'available'
	AND eam.token_per_second IS NOT NULL
	AND eam.token_per_second > 0
	AND (eh.disabled IS NOT TRUE OR eh.disabled_until IS NULL OR eh.disabled_until < NOW())
`

// modelListRoutableSQL selects distinct enabled models that have ≥1 routable endpoint.
const modelListRoutableSQL = `
	SELECT DISTINCT m.name, m.tag
	FROM ai_models m
	JOIN endpoint_ai_models eam ON eam.ai_model_id = m.id
	JOIN endpoints e ON e.id = eam.endpoint_id
	LEFT JOIN endpoint_health eh ON eh.url = e.url
	WHERE ` + routableEndpointSQL + `
	ORDER BY m.name, m.tag
`

// autoFallbackCandidates returns ordered model IDs to try after the requested
// model has zero routable endpoints. Pure function — no DB/IO.
//
// Goal: OpenCode/OpenChamber never hard-fail with
// "No available endpoint found for model X" when *any* proxy capacity remains.
func autoFallbackCandidates(name, tag string) []string {
	nameL := strings.ToLower(strings.TrimSpace(name))
	tagL := strings.ToLower(strings.TrimSpace(tag))
	requested := nameL + ":" + tagL

	var cands []string
	add := func(id string) {
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" || id == requested {
			return
		}
		for _, existing := range cands {
			if existing == id {
				return
			}
		}
		cands = append(cands, id)
	}

	// Cloud / frontier family → smart cloud profile first
	if strings.Contains(tagL, "cloud") || strings.HasSuffix(tagL, "-cloud") {
		add("smart:cloud")
	}

	// Coding models
	if strings.Contains(nameL, "coder") || strings.Contains(nameL, "code") ||
		strings.Contains(tagL, "coding") || strings.Contains(nameL, "codestral") {
		add("smart:coding")
	}

	// GLM family ladder (OpenCode defaults)
	if strings.HasPrefix(nameL, "glm") {
		for _, id := range []string{
			"glm-5.2:cloud", "glm-5.1:cloud", "glm-5:cloud",
			"glm-4.7:cloud", "glm-4.7-flash:latest", "smart:cloud",
		} {
			add(id)
		}
	}

	// Kimi / Mistral large / Qwen titans
	if strings.Contains(nameL, "kimi") || strings.Contains(nameL, "mistral-large") ||
		strings.Contains(nameL, "qwen3") || strings.Contains(nameL, "qwen2.5") {
		add("smart:cloud")
		if strings.Contains(nameL, "coder") {
			add("smart:coding")
		}
	}

	// Universal last-resort profiles (always try to answer)
	add("smart:coding")
	add("smart:cloud")
	add("smart:fastest")
	add("smart:small")

	return cands
}

// resolveModelRoute tries the requested model, then configured + automatic
// fallbacks, until a routable endpoint set is found.
// Returns the route, the model id that was actually selected, and whether a
// fallback was used (for X-Model-Fallback header).
func (h *OllamaHandler) resolveModelRoute(name, tag string) (route *resolvedModelRoute, selected string, usedFallback bool, err error) {
	requested := fmt.Sprintf("%s:%s", name, tag)

	resolved, err := h.bestEndpointsForModel(name, tag, defaultEndpointRankMode())
	if err == nil && resolved != nil && len(resolved.URLs) > 0 {
		return resolved, fmt.Sprintf("%s:%s", resolved.Name, resolved.Tag), false, nil
	}

	// 1) Explicit APP_FALLBACK_MODELS map
	lookupKey := strings.ToLower(requested)
	if fallbackRaw, ok := h.fallbacks[lookupKey]; ok {
		if r, sel, ok := h.tryFallbackID(fallbackRaw); ok {
			log.Printf("[proxy] Model %s unavailable, APP_FALLBACK_MODELS → %s (resolved %s)", requested, fallbackRaw, sel)
			return r, sel, true, nil
		}
	}

	// 2) Same model name, different tags that are still routable
	if alt, altErr := h.bestAlternateTagForModel(name, tag); altErr == nil && alt != nil && len(alt.URLs) > 0 {
		sel := fmt.Sprintf("%s:%s", alt.Name, alt.Tag)
		log.Printf("[proxy] Model %s unavailable, alternate tag → %s", requested, sel)
		return alt, sel, true, nil
	}

	// 3) Automatic candidate ladder (smart profiles + related models)
	for _, cand := range autoFallbackCandidates(name, tag) {
		if r, sel, ok := h.tryFallbackID(cand); ok {
			log.Printf("[proxy] Model %s unavailable, auto-fallback → %s (via %s)", requested, sel, cand)
			return r, sel, true, nil
		}
	}

	if err != nil {
		return nil, requested, false, err
	}
	return nil, requested, false, fmt.Errorf("no available endpoint found for model %s", requested)
}

func (h *OllamaHandler) tryFallbackID(modelID string) (*resolvedModelRoute, string, bool) {
	cName, cTag := parseModel(modelID)
	if strings.EqualFold(cName, "smart") || strings.EqualFold(cName, "best-abliterated") {
		profile := cTag
		if strings.EqualFold(cName, "best-abliterated") {
			profile = "abliterated"
		}
		return h.routeFromSmartProfile(profile)
	}
	fbResolved, fbErr := h.bestEndpointsForModel(cName, cTag, defaultEndpointRankMode())
	if fbErr == nil && fbResolved != nil && len(fbResolved.URLs) > 0 {
		return fbResolved, fmt.Sprintf("%s:%s", fbResolved.Name, fbResolved.Tag), true
	}
	return nil, "", false
}

func (h *OllamaHandler) routeFromSmartProfile(profile string) (*resolvedModelRoute, string, bool) {
	candidates, err := h.resolveSmartModel(profile)
	if err != nil || len(candidates) == 0 {
		return nil, "", false
	}
	healthTracker := services.GetHealthTracker()
	for _, candidate := range candidates {
		urls := healthTracker.FilterHealthyEndpoints(candidate.urls)
		if len(urls) == 0 {
			continue
		}
		return &resolvedModelRoute{
			Name: candidate.name,
			Tag:  candidate.tag,
			URLs: urls,
		}, fmt.Sprintf("%s:%s", candidate.name, candidate.tag), true
	}
	return nil, "", false
}

// bestAlternateTagForModel finds another tag of the same model name that is routable.
func (h *OllamaHandler) bestAlternateTagForModel(modelName, excludeTag string) (*resolvedModelRoute, error) {
	type row struct {
		Name string `db:"name"`
		Tag  string `db:"tag"`
	}
	var rows []row
	err := h.db.Select(&rows, `
		SELECT m.name, m.tag
		FROM endpoint_ai_models eam
		JOIN endpoints e ON e.id = eam.endpoint_id
		JOIN ai_models m ON m.id = eam.ai_model_id
		LEFT JOIN endpoint_health eh ON eh.url = e.url
		WHERE LOWER(m.name) = LOWER($1)
		  AND LOWER(m.tag) <> LOWER($2)
		  AND `+routableEndpointSQL+`
		GROUP BY m.name, m.tag
		ORDER BY COUNT(*) DESC, MAX(eam.token_per_second) DESC NULLS LAST
		LIMIT 3
	`, modelName, excludeTag)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		resolved, rerr := h.bestEndpointsForModel(r.Name, r.Tag, defaultEndpointRankMode())
		if rerr == nil && resolved != nil && len(resolved.URLs) > 0 {
			return resolved, nil
		}
	}
	return nil, fmt.Errorf("no alternate tag for %s", modelName)
}
