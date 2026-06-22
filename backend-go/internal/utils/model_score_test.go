package utils

import "testing"

func TestExtractParamBillions(t *testing.T) {
	tests := []struct {
		name, tag string
		want      float64
	}{
		{"llama3", "70b", 70},
		{"qwen2.5", "1.5b", 1.5},
		{"mistral-7b-instruct", "latest", 7},
		{"some-model", "latest", 1.0},
		{"gpt-oss", "120b", 120},
	}
	for _, tt := range tests {
		got := ExtractParamBillions(tt.name, tt.tag)
		if got != tt.want {
			t.Errorf("ExtractParamBillions(%q, %q) = %v, want %v", tt.name, tt.tag, got, tt.want)
		}
	}
}

func TestCompositeScore(t *testing.T) {
	score := CompositeScore(100, 0.5, 7)
	if score == nil || *score <= 0 {
		t.Fatalf("expected positive score, got %v", score)
	}

	fast := CompositeScore(100, 0.1, 7)
	slow := CompositeScore(100, 2.0, 7)
	if fast == nil || slow == nil || *fast <= *slow {
		t.Errorf("faster reply should score higher: fast=%v slow=%v", *fast, *slow)
	}

	large := CompositeScore(50, 0.5, 70)
	small := CompositeScore(50, 0.5, 7)
	if large == nil || small == nil || *large <= *small {
		t.Errorf("larger params should score higher at equal speed/latency: large=%v small=%v", *large, *small)
	}

	if CompositeScore(0, 0.5, 7) != nil {
		t.Error("zero TPS should return nil")
	}
	if CompositeScore(100, 0, 7) != nil {
		t.Error("zero reply time should return nil")
	}
}

func TestCompositeScoreSQLNonEmpty(t *testing.T) {
	sql := ModelListCompositeScoreSQL()
	if len(sql) < 20 {
		t.Error("expected non-trivial SQL expression")
	}
}