package auth

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/prachi-satbhai0741/cli-login-system/internal/db"
	"github.com/prachi-satbhai0741/cli-login-system/internal/models"
)

const (
	// BcryptCost is the work factor for bcrypt hashing (higher = slower = safer).
	BcryptCost = 12

	// MaxFailedAttempts before the account is temporarily locked.
	MaxFailedAttempts = 5

	// LockDuration is how long an account stays locked after too many failures.
	LockDuration = 15 * time.Minute
)

// ErrInvalidCredentials is returned when username or password is wrong.
var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrAccountLocked is returned when the account is locked due to failed attempts.
var ErrAccountLocked = errors.New("account locked")

// ErrUsernameTaken is returned when registering a username that already exists.
var ErrUsernameTaken = errors.New("username already taken")

// ErrTOTPRequired is returned when 2FA is enabled but no code was provided.
var ErrTOTPRequired = errors.New("TOTP code required")

// ErrTOTPInvalid is returned when the provided TOTP code is wrong.
var ErrTOTPInvalid = errors.New("invalid TOTP code")

// Service handles all authentication logic.
type Service struct {
	db *db.DB
}

// New creates a new auth Service.
func New(database *db.DB) *Service {
	return &Service{db: database}
}

// HashPassword creates a bcrypt hash of the plain-text password.
func HashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// CheckPassword returns true if the plain password matches the stored hash.
func CheckPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// Register creates a new user account.
// Returns ErrUsernameTaken if the username is already registered.
func (s *Service) Register(username, password string) error {
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > 32 {
		return errors.New("username must be 32 characters or fewer")
	}

	exists, err := s.db.UsernameExists(username)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if exists {
		return ErrUsernameTaken
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	_, err = s.db.CreateUser(username, hash)
	return err
}

// LoginResult holds the outcome of a successful login.
type LoginResult struct {
	User    *models.User
	Session *models.Session
}

// Login authenticates a user. If 2FA is enabled, totpCode must be provided.
// Returns ErrTOTPRequired if 2FA is enabled but totpCode is empty.
func (s *Service) Login(username, password, totpCode string, sessionTimeout time.Duration) (*LoginResult, error) {
	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	// Use generic error to avoid username enumeration
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check lockout before anything else
	if user.IsLocked() {
		remaining := user.LockRemaining().Round(time.Second)
		return nil, fmt.Errorf("%w: try again in %s", ErrAccountLocked, remaining)
	}

	// Verify password
	if !CheckPassword(password, user.PasswordHash) {
		if err := s.db.RecordFailedAttempt(user.ID, MaxFailedAttempts, LockDuration); err != nil {
			return nil, fmt.Errorf("record attempt: %w", err)
		}
		// Re-fetch to get updated attempt count
		user, _ = s.db.GetUserByUsername(username)
		if user != nil && user.IsLocked() {
			return nil, fmt.Errorf("%w: account locked for %s due to too many failed attempts",
				ErrAccountLocked, LockDuration)
		}
		return nil, ErrInvalidCredentials
	}

	// Verify TOTP if enabled
	if user.TOTPEnabled {
		if totpCode == "" {
			return nil, ErrTOTPRequired
		}
		// Validation is handled in the totp package; we call it via the caller
		// who passes a pre-validated flag by providing the code here.
		// Actual validation happens in the CLI layer before calling Login.
	}

	// All checks passed — clear failed attempts and create session
	if err := s.db.ClearFailedAttempts(user.ID); err != nil {
		return nil, err
	}
	if err := s.db.UpdateLastLogin(user.ID); err != nil {
		return nil, err
	}

	sess, err := createSession(s.db, user.ID, sessionTimeout)
	if err != nil {
		return nil, err
	}

	// Re-fetch user to get updated last_login
	user, err = s.db.GetUserByID(user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{User: user, Session: sess}, nil
}

// createSession makes a new session in the DB and returns it.
func createSession(database *db.DB, userID int64, timeout time.Duration) (*models.Session, error) {
	sessionID := newUUID()
	expiresAt := time.Now().Add(timeout)

	if err := database.CreateSession(sessionID, userID, expiresAt); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	sess, err := database.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("fetch session: %w", err)
	}
	return sess, nil
}
