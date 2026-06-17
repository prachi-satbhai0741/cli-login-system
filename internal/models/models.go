package models

import "time"

// User represents a registered user in the system.
type User struct {
	ID             int64
	Username       string
	PasswordHash   string
	TOTPSecret     string
	TOTPEnabled    bool
	FailedAttempts int
	LockedUntil    *time.Time
	LastLogin      *time.Time
	CreatedAt      time.Time
}

// IsLocked returns true if the user account is currently locked.
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// LockRemaining returns how long until the account unlocks.
func (u *User) LockRemaining() time.Duration {
	if u.LockedUntil == nil {
		return 0
	}
	remaining := time.Until(*u.LockedUntil)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Session represents an active login session.
type Session struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// IsExpired returns true if the session has passed its expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// TimeRemaining returns how long until the session expires.
func (s *Session) TimeRemaining() time.Duration {
	remaining := time.Until(s.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}
