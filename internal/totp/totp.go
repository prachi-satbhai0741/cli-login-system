package totp

import (
	"fmt"

	gotp "github.com/pquerna/otp/totp"
)

// GenerateSecret creates a new TOTP secret key for the given user.
// Returns the secret string and the otpauth:// URL for QR code scanning.
func GenerateSecret(issuer, username string) (secret string, otpauthURL string, err error) {
	key, err := gotp.Generate(gotp.GenerateOpts{
		Issuer:      issuer,
		AccountName: username,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate TOTP key: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// Validate checks whether the given 6-digit code is valid for the secret.
// Uses a 30-second time window with ±1 step tolerance.
func Validate(code, secret string) bool {
	return gotp.Validate(code, secret)
}
