package unit_tests

import (
	"strings"
	"testing"

	"pharma-platform/internal/handler"
)

func TestScoreCandidateCapsAt100AndIncludesExplanation(t *testing.T) {
	tokens := []string{"admin", "eagle", "123", "abc"}
	score, explanation := handler.ScoreCandidate(tokens, "Admin Eagle", "admin@eagle.test", "+251123", "ABC-123")

	if score != 100 {
		t.Fatalf("expected score to be capped at 100, got %d", score)
	}
	if len(explanation) == 0 {
		t.Fatalf("expected explanation entries")
	}
	joined := strings.Join(explanation, " ")
	if !strings.Contains(joined, "name matched") {
		t.Fatalf("expected name match explanation, got: %v", explanation)
	}
}

func TestScoreCandidateZeroForNoMatch(t *testing.T) {
	score, explanation := handler.ScoreCandidate([]string{"zzz"}, "Admin Eagle", "admin@eagle.test", "+251123", "ABC-123")
	if score != 0 {
		t.Fatalf("expected zero score for no match, got %d", score)
	}
	if len(explanation) != 0 {
		t.Fatalf("expected empty explanation for no match, got %v", explanation)
	}
}
