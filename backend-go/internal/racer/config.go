package racer

import (
	"os"
	"strings"
)

func RelayEnabled() bool {
	return parseBoolEnv("RACER_RELAY_ENABLED")
}

func RaceEnabled() bool {
	return parseBoolEnv("RACER_RACE_ENABLED")
}

func BaseURL() string {
	if val := strings.TrimSpace(os.Getenv("RACER_URL")); val != "" {
		return strings.TrimRight(val, "/")
	}
	return "http://127.0.0.1:8787"
}

func parseBoolEnv(key string) bool {
	val := strings.TrimSpace(os.Getenv(key))
	switch strings.ToLower(val) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}