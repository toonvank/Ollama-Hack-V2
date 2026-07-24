package handlers

import (
	"strings"
	"testing"
)

func TestAutoFallbackCandidates_CloudGLM(t *testing.T) {
	cands := autoFallbackCandidates("glm-5.2", "cloud")
	if len(cands) == 0 {
		t.Fatal("expected fallback candidates for glm-5.2:cloud")
	}
	// Must not include the requested model itself
	for _, c := range cands {
		if c == "glm-5.2:cloud" {
			t.Fatalf("candidates must not include the requested model: %v", cands)
		}
	}
	// Cloud models should prefer smart:cloud early
	foundCloud := false
	for i, c := range cands {
		if c == "smart:cloud" {
			foundCloud = true
			if i > 3 {
				t.Fatalf("smart:cloud should be near the front, got index %d in %v", i, cands)
			}
			break
		}
	}
	if !foundCloud {
		t.Fatalf("expected smart:cloud in candidates, got %v", cands)
	}
	// Always ends with universal profiles
	joined := strings.Join(cands, ",")
	for _, need := range []string{"smart:coding", "smart:fastest", "smart:small"} {
		if !strings.Contains(joined, need) {
			t.Fatalf("expected %s in candidates %v", need, cands)
		}
	}
}

func TestAutoFallbackCandidates_Coder(t *testing.T) {
	cands := autoFallbackCandidates("qwen3-coder-next", "max")
	found := false
	for _, c := range cands {
		if c == "smart:coding" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("coder model should fall back to smart:coding, got %v", cands)
	}
}

func TestAutoFallbackCandidates_Dedupe(t *testing.T) {
	cands := autoFallbackCandidates("glm-5.1", "cloud")
	seen := map[string]bool{}
	for _, c := range cands {
		if seen[c] {
			t.Fatalf("duplicate candidate %q in %v", c, cands)
		}
		seen[c] = true
	}
}

func TestParseModel(t *testing.T) {
	n, tag := parseModel("glm-5.2:cloud")
	if n != "glm-5.2" || tag != "cloud" {
		t.Fatalf("got %s %s", n, tag)
	}
	n, tag = parseModel("llama3")
	if n != "llama3" || tag != "latest" {
		t.Fatalf("default tag: got %s %s", n, tag)
	}
}

func TestRoutableSQLConstants(t *testing.T) {
	// Structural guard: catalog query must require endpoint + health readiness
	if !strings.Contains(routableEndpointSQL, "e.status = 'available'") {
		t.Fatal("routableEndpointSQL must require e.status=available")
	}
	if !strings.Contains(routableEndpointSQL, "eam.status = 'available'") {
		t.Fatal("routableEndpointSQL must require eam.status=available")
	}
	if !strings.Contains(modelListRoutableSQL, "JOIN endpoints e") {
		t.Fatal("modelListRoutableSQL must join endpoints")
	}
	if !strings.Contains(modelListRoutableSQL, routableEndpointSQL) {
		t.Fatal("modelListRoutableSQL must embed routableEndpointSQL")
	}
}
