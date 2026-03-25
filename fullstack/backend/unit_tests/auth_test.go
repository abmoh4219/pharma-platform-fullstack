package unit_tests

import (
	"testing"
	"time"

	"pharma-platform/internal/security"
)

func TestIssueAndParseToken(t *testing.T) {
	secret := "unit-test-secret"
	issuedAt := time.Now().UTC()

	token, jti, expiresAt, err := security.IssueToken(secret, security.TokenInput{
		UserID:      100,
		Username:    "admin",
		Role:        "system_admin",
		ScopeID:     1,
		Institution: "EAGLE_HOSPITAL",
		Department:  "HQ",
		Team:        "CORE",
		ExpiryHours: 8,
	})
	if err != nil {
		t.Fatalf("expected token issue success: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}
	if jti == "" {
		t.Fatalf("expected non-empty jti")
	}
	if expiresAt.Before(issuedAt.Add(7 * time.Hour)) {
		t.Fatalf("expected expiration near 8 hours, got %s", expiresAt)
	}

	claims, err := security.ParseToken(secret, token)
	if err != nil {
		t.Fatalf("expected token parse success: %v", err)
	}
	if claims.Username != "admin" {
		t.Fatalf("unexpected username: %s", claims.Username)
	}
	if claims.Role != "system_admin" {
		t.Fatalf("unexpected role: %s", claims.Role)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	token, _, _, err := security.IssueToken("correct", security.TokenInput{
		UserID:      1,
		Username:    "admin",
		Role:        "system_admin",
		ScopeID:     1,
		Institution: "EAGLE_HOSPITAL",
		Department:  "HQ",
		Team:        "CORE",
		ExpiryHours: 8,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	if _, err := security.ParseToken("wrong", token); err == nil {
		t.Fatalf("expected parse failure with wrong secret")
	}
}
