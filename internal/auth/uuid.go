package auth

import "github.com/google/uuid"

// newUUID returns a new random UUID string.
func newUUID() string {
	return uuid.NewString()
}
