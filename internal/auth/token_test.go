package auth

import (
	"testing"
	"time"
)

func TestTokenManagerIssueAndVerify(t *testing.T) {
	manager := NewTokenManager([]byte("0123456789abcdef0123456789abcdef"), "issuer", "audience", time.Minute)
	token, expiresAt, err := manager.Issue(42)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("Issue() expiry %v should be in the future", expiresAt)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Subject != "42" {
		t.Fatalf("subject = %q, want %q", claims.Subject, "42")
	}
	if _, err := NewTokenManager([]byte("another-secret-that-is-long-enough"), "issuer", "audience", time.Minute).Verify(token); err == nil {
		t.Fatal("Verify() accepted a token signed with a different secret")
	}
}
