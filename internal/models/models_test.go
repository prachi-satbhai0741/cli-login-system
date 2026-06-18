package models_test

import (
	"testing"
	"time"

	"github.com/prachi-satbhai0741/cli-login-system/internal/models"
)

func TestUserIsLocked(t *testing.T) {
	u := &models.User{}

	// No lock set
	if u.IsLocked() {
		t.Error("user with no lock should not be locked")
	}

	// Lock in the past (expired)
	past := time.Now().Add(-5 * time.Minute)
	u.LockedUntil = &past
	if u.IsLocked() {
		t.Error("user with lock in the past should not be locked")
	}

	// Lock in the future
	future := time.Now().Add(10 * time.Minute)
	u.LockedUntil = &future
	if !u.IsLocked() {
		t.Error("user with lock in the future should be locked")
	}
}

func TestSessionIsExpired(t *testing.T) {
	s := &models.Session{ExpiresAt: time.Now().Add(5 * time.Minute)}
	if s.IsExpired() {
		t.Error("session that expires in the future should not be expired")
	}

	s.ExpiresAt = time.Now().Add(-1 * time.Minute)
	if !s.IsExpired() {
		t.Error("session that expired in the past should be expired")
	}
}

func TestSessionTimeRemaining(t *testing.T) {
	s := &models.Session{ExpiresAt: time.Now().Add(10 * time.Minute)}
	rem := s.TimeRemaining()
	if rem <= 0 {
		t.Error("expected positive time remaining")
	}
	if rem > 10*time.Minute {
		t.Error("time remaining should not exceed the set duration")
	}
}
