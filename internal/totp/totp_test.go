package totp_test

import (
	"testing"

	"github.com/pquerna/otp/totp"
	apptotp "github.com/prachi-satbhai0741/cli-login-system/internal/totp"
)

func TestGenerateSecret(t *testing.T) {
	secret, url, err := apptotp.GenerateSecret("TestApp", "testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	if secret == "" {
		t.Error("expected non-empty secret")
	}
	if url == "" {
		t.Error("expected non-empty otpauth URL")
	}
}

func TestValidateCode(t *testing.T) {
	secret, _, err := apptotp.GenerateSecret("TestApp", "testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	// Generate a valid code using the same library
	code, err := totp.GenerateCode(secret, totp.Now())
	if err != nil {
		t.Fatalf("GenerateCode error: %v", err)
	}

	if !apptotp.Validate(code, secret) {
		t.Error("Validate should return true for a freshly-generated code")
	}
}

func TestValidateWrongCode(t *testing.T) {
	secret, _, err := apptotp.GenerateSecret("TestApp", "testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	if apptotp.Validate("000000", secret) {
		// Very unlikely but not impossible – just log it
		t.Log("Warning: 000000 happened to be valid (astronomically unlikely)")
	}
	if apptotp.Validate("abcdef", secret) {
		t.Error("Validate should return false for non-numeric code")
	}
}
