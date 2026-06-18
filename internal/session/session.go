package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/prachi-satbhai0741/cli-login-system/internal/db"
	"github.com/prachi-satbhai0741/cli-login-system/internal/models"
)

// ErrSessionExpired is returned when a session has passed its expiry time.
var ErrSessionExpired = errors.New("session expired, please login again")

// ErrSessionNotFound is returned when the session ID does not exist in the DB.
var ErrSessionNotFound = errors.New("session not found")

// Manager handles session validation and cleanup.
type Manager struct {
	db *db.DB
}

// New creates a new session Manager.
func New(database *db.DB) *Manager {
	return &Manager{db: database}
}

// Validate checks that the session exists and has not expired.
// Returns the session on success, or an error if it is missing / expired.
func (m *Manager) Validate(sessionID string) (*models.Session, error) {
	sess, err := m.db.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}
	if sess.IsExpired() {
		// Clean up expired session
		_ = m.db.DeleteSession(sessionID)
		return nil, ErrSessionExpired
	}
	return sess, nil
}

// Destroy removes a session from the database (logout).
func (m *Manager) Destroy(sessionID string) error {
	return m.db.DeleteSession(sessionID)
}

// CleanupExpired removes all expired sessions from the database.
func (m *Manager) CleanupExpired() error {
	return m.db.DeleteExpiredSessions()
}

// FormatExpiry returns a human-readable expiry string like "29m 45s".
func FormatExpiry(sess *models.Session) string {
	remaining := sess.TimeRemaining().Round(time.Second)
	if remaining <= 0 {
		return "expired"
	}
	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	s := int(remaining.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
