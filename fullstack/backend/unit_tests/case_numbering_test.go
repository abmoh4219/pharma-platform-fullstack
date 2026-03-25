package unit_tests

import (
	"testing"
	"time"

	"pharma-platform/internal/handler"
)

func TestFormatCaseNumber(t *testing.T) {
	value := handler.FormatCaseNumber(time.Date(2026, time.March, 24, 10, 0, 0, 0, time.UTC), "Eagle Hospital", 7)
	expected := "20260324-EAGLEHOSPITA-000007"
	if value != expected {
		t.Fatalf("expected %s, got %s", expected, value)
	}
}

func TestNormalizeInstitutionPartFallback(t *testing.T) {
	value := handler.NormalizeInstitutionPart("   !!!   ")
	if value != "INST" {
		t.Fatalf("expected INST fallback, got %s", value)
	}
}
