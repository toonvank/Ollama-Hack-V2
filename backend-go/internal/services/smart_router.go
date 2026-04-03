package services

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"
)

// RoutingRule defines a category with keywords and preferred model
type RoutingRule struct {
	Category    string   `json:"category"`     // "coding", "creative", "analysis", "general"
	Keywords    []string `json:"keywords"`     // Keywords that trigger this category
	PreferModel string   `json:"prefer_model"` // e.g., "codellama", "mistral", "llama3"
}

// SmartRouter classifies prompts and routes to appropriate models
type SmartRouter struct {
	enabled bool
	rules   []RoutingRule
	mu      sync.RWMutex
}

// DefaultRoutingRules provides sensible defaults for prompt classification
var DefaultRoutingRules = []RoutingRule{
	{
		Category: "coding",
		Keywords: []string{
			"code", "function", "debug", "programming", "python", "javascript",
			"golang", "```", "error", "compile", "syntax", "variable", "class",
			"method", "api", "implement", "refactor", "bug", "fix", "test",
			"typescript", "java", "rust", "cpp", "c++", "html", "css", "sql",
			"algorithm", "data structure", "loop", "array", "object", "import",
			"package", "module", "library", "framework", "backend", "frontend",
		},
		PreferModel: "codellama",
	},
	{
		Category: "creative",
		Keywords: []string{
			"story", "creative", "imagine", "write a poem", "fiction", "novel",
			"character", "narrative", "plot", "dialogue", "screenplay", "lyrics",
			"song", "poetry", "fantasy", "adventure", "romance", "horror",
			"write me a", "create a story", "once upon", "tale", "myth",
		},
		PreferModel: "mistral",
	},
	{
		Category: "analysis",
		Keywords: []string{
			"analyze", "summarize", "explain", "compare", "contrast", "evaluate",
			"assess", "review", "critique", "breakdown", "interpret", "examine",
			"investigate", "research", "study", "report", "findings", "conclusion",
			"pros and cons", "advantages", "disadvantages", "impact", "effect",
		},
		PreferModel: "llama3",
	},
}

// NewSmartRouter creates a new smart router instance
func NewSmartRouter() *SmartRouter {
	sr := &SmartRouter{
		rules: DefaultRoutingRules,
	}

	// Check if smart routing is enabled via environment
	enabledStr := os.Getenv("SMART_ROUTING_ENABLED")
	sr.enabled = strings.ToLower(enabledStr) == "true" || enabledStr == "1"

	// Load custom rules from environment if provided
	rulesJSON := os.Getenv("SMART_ROUTING_RULES")
	if rulesJSON != "" {
		var customRules []RoutingRule
		if err := json.Unmarshal([]byte(rulesJSON), &customRules); err != nil {
			log.Printf("[smart-router] Failed to parse SMART_ROUTING_RULES: %v, using defaults", err)
		} else if len(customRules) > 0 {
			sr.rules = customRules
			log.Printf("[smart-router] Loaded %d custom routing rules", len(customRules))
		}
	}

	if sr.enabled {
		log.Printf("[smart-router] Smart routing ENABLED with %d rules", len(sr.rules))
	} else {
		log.Println("[smart-router] Smart routing DISABLED (set SMART_ROUTING_ENABLED=true to enable)")
	}

	return sr
}

// ClassificationResult holds the result of prompt classification
type ClassificationResult struct {
	Category      string  // The detected category
	PreferModel   string  // The preferred model for this category
	Confidence    float64 // Score indicating match strength (0.0 - 1.0)
	MatchedKeywords []string // Keywords that triggered the match
}

// IsEnabled returns whether smart routing is enabled
func (sr *SmartRouter) IsEnabled() bool {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.enabled
}

// SetEnabled enables or disables smart routing
func (sr *SmartRouter) SetEnabled(enabled bool) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.enabled = enabled
}

// Classify analyzes the prompt and returns the best matching category
func (sr *SmartRouter) Classify(prompt string) *ClassificationResult {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	if !sr.enabled || prompt == "" {
		return nil
	}

	lowerPrompt := strings.ToLower(prompt)
	
	var bestResult *ClassificationResult
	var bestScore float64

	for _, rule := range sr.rules {
		matchedKeywords := []string{}
		
		for _, keyword := range rule.Keywords {
			if strings.Contains(lowerPrompt, strings.ToLower(keyword)) {
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}

		if len(matchedKeywords) > 0 {
			// Calculate confidence based on keyword match ratio and count
			score := float64(len(matchedKeywords)) / float64(len(rule.Keywords))
			// Bonus for multiple matches
			if len(matchedKeywords) > 2 {
				score += 0.1 * float64(len(matchedKeywords)-2)
			}
			// Cap at 1.0
			if score > 1.0 {
				score = 1.0
			}

			if score > bestScore {
				bestScore = score
				bestResult = &ClassificationResult{
					Category:        rule.Category,
					PreferModel:     rule.PreferModel,
					Confidence:      score,
					MatchedKeywords: matchedKeywords,
				}
			}
		}
	}

	if bestResult != nil {
		log.Printf("[smart-router] Classified prompt as '%s' (confidence: %.2f, keywords: %v)",
			bestResult.Category, bestResult.Confidence, bestResult.MatchedKeywords)
	}

	return bestResult
}

// ClassifyMessages classifies based on chat messages (extracts text from all messages)
func (sr *SmartRouter) ClassifyMessages(messages []interface{}) *ClassificationResult {
	if !sr.IsEnabled() {
		return nil
	}

	var allContent strings.Builder
	
	for _, msg := range messages {
		if msgMap, ok := msg.(map[string]interface{}); ok {
			if content, ok := msgMap["content"].(string); ok {
				allContent.WriteString(content)
				allContent.WriteString(" ")
			}
		}
	}

	return sr.Classify(allContent.String())
}

// GetRules returns a copy of the current routing rules
func (sr *SmartRouter) GetRules() []RoutingRule {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	rules := make([]RoutingRule, len(sr.rules))
	copy(rules, sr.rules)
	return rules
}

// SetRules updates the routing rules
func (sr *SmartRouter) SetRules(rules []RoutingRule) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.rules = rules
}

// FormatRouteHeader creates the X-Smart-Route header value
func FormatRouteHeader(category, preferModel string) string {
	return category + "→" + preferModel
}
