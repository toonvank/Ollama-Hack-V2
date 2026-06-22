package utils

import (
	"math"
	"regexp"
	"strconv"
)

var paramBillionsRe = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*b`)

// ExtractParamBillions guesses parameter count (in billions) from model name/tag.
// Returns 1.0 when no size hint is found (neutral default for scoring).
func ExtractParamBillions(name, tag string) float64 {
	for _, s := range []string{name, tag} {
		if m := paramBillionsRe.FindStringSubmatch(s); len(m) > 1 {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil && v > 0 {
				return v
			}
		}
	}
	return 1.0
}

// CompositeScore balances throughput, reply latency, and model size.
// Formula: tps * (1/reply_seconds) * ln(1 + param_billions)
// Higher is better. Unknown size uses 1B neutral weight.
func CompositeScore(tokenPerSecond, replySeconds, paramBillions float64) *float64 {
	if tokenPerSecond <= 0 || replySeconds <= 0 {
		return nil
	}
	if paramBillions <= 0 {
		paramBillions = 1.0
	}
	score := tokenPerSecond * (1.0 / replySeconds) * math.Log1p(paramBillions)
	return &score
}

// ParamBillionsSQL returns a SQL expression that extracts param size from name/tag columns.
func ParamBillionsSQL(nameCol, tagCol string) string {
	return `COALESCE(
		(regexp_match(` + nameCol + `, '(\d+(?:\.\d+)?)\s*b', 'i'))[1]::double precision,
		(regexp_match(` + tagCol + `, '(\d+(?:\.\d+)?)\s*b', 'i'))[1]::double precision,
		1.0
	)`
}

// CompositeScoreSQL builds a SQL expression for the composite score.
// Placeholders are raw SQL column/expression strings (not query args).
func CompositeScoreSQL(tpsExpr, replyExpr, paramExpr string) string {
	return `CASE
		WHEN ` + tpsExpr + ` IS NOT NULL
		 AND ` + replyExpr + ` IS NOT NULL
		 AND ` + replyExpr + ` > 0
		THEN ` + tpsExpr + ` * (1.0 / ` + replyExpr + `) * LN(1 + ` + paramExpr + `)
		ELSE NULL
	END`
}

// ModelListCompositeScoreSQL scores a grouped model row using best TPS and best reply.
func ModelListCompositeScoreSQL() string {
	param := ParamBillionsSQL("m.name", "m.tag")
	return CompositeScoreSQL("MAX(eam.token_per_second)", "MIN(eam.max_connection_time)", param)
}

// EndpointCompositeScoreSQL scores a single endpoint_ai_models row.
func EndpointCompositeScoreSQL() string {
	param := ParamBillionsSQL("m.name", "m.tag")
	return CompositeScoreSQL("eam.token_per_second", "eam.max_connection_time", param)
}