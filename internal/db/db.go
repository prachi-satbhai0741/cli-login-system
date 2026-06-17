package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yourname/cli-login-system/internal/models"
)

// DB wraps the sql.DB connection.
type DB struct {
	conn *sql.DB
}

// New opens (or creates) the SQLite database and runs migrations.
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	conn.SetMaxOpenConns(1) // SQLite supports single writer

	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

// migrate runs the embedded SQL schema.
func (d *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		username        TEXT UNIQUE NOT NULL,
		password_hash   TEXT NOT NULL,
		totp_secret     TEXT,
		totp_enabled    INTEGER DEFAULT 0,
		failed_attempts INTEGER DEFAULT 0,
		locked_until    DATETIME,
		last_login      DATETIME,
		created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS sessions (
		id          TEXT PRIMARY KEY,
		user_id     INTEGER NOT NULL,
		expires_at  DATETIME NOT NULL,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_users_username   ON users(username);
	`
	_, err := d.conn.Exec(schema)
	return err
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// ─── User queries ────────────────────────────────────────────────────────────

// CreateUser inserts a new user and returns their assigned ID.
func (d *DB) CreateUser(username, passwordHash string) (int64, error) {
	res, err := d.conn.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		username, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetUserByUsername fetches a user by username. Returns nil, nil if not found.
func (d *DB) GetUserByUsername(username string) (*models.User, error) {
	row := d.conn.QueryRow(
		`SELECT id, username, password_hash, COALESCE(totp_secret,''),
		        totp_enabled, failed_attempts, locked_until, last_login, created_at
		 FROM users WHERE username = ?`, username,
	)
	return scanUser(row)
}

// GetUserByID fetches a user by their primary key.
func (d *DB) GetUserByID(id int64) (*models.User, error) {
	row := d.conn.QueryRow(
		`SELECT id, username, password_hash, COALESCE(totp_secret,''),
		        totp_enabled, failed_attempts, locked_until, last_login, created_at
		 FROM users WHERE id = ?`, id,
	)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*models.User, error) {
	u := &models.User{}
	var totpEnabled int
	var lockedUntil, lastLogin sql.NullString

	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.TOTPSecret,
		&totpEnabled, &u.FailedAttempts, &lockedUntil, &lastLogin, &u.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	u.TOTPEnabled = totpEnabled == 1

	if lockedUntil.Valid {
		t, err := time.Parse("2006-01-02T15:04:05Z", lockedUntil.String)
		if err != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", lockedUntil.String)
		}
		u.LockedUntil = &t
	}
	if lastLogin.Valid {
		t, err := time.Parse("2006-01-02T15:04:05Z", lastLogin.String)
		if err != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", lastLogin.String)
		}
		u.LastLogin = &t
	}
	return u, nil
}

// RecordFailedAttempt increments failed_attempts. If >= maxAttempts, locks the account.
func (d *DB) RecordFailedAttempt(userID int64, maxAttempts int, lockDuration time.Duration) error {
	_, err := d.conn.Exec(
		`UPDATE users SET failed_attempts = failed_attempts + 1,
		 locked_until = CASE WHEN failed_attempts + 1 >= ? THEN datetime('now', ?) ELSE locked_until END
		 WHERE id = ?`,
		maxAttempts,
		fmt.Sprintf("+%d seconds", int(lockDuration.Seconds())),
		userID,
	)
	return err
}

// ClearFailedAttempts resets the failed attempt counter and lock after successful login.
func (d *DB) ClearFailedAttempts(userID int64) error {
	_, err := d.conn.Exec(
		`UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE id = ?`, userID,
	)
	return err
}

// UpdateLastLogin sets the last_login timestamp to now.
func (d *DB) UpdateLastLogin(userID int64) error {
	_, err := d.conn.Exec(
		`UPDATE users SET last_login = datetime('now') WHERE id = ?`, userID,
	)
	return err
}

// SetTOTPSecret stores the TOTP secret and enables 2FA for the user.
func (d *DB) SetTOTPSecret(userID int64, secret string) error {
	_, err := d.conn.Exec(
		`UPDATE users SET totp_secret = ?, totp_enabled = 1 WHERE id = ?`, secret, userID,
	)
	return err
}

// DisableTOTP removes the TOTP secret and disables 2FA.
func (d *DB) DisableTOTP(userID int64) error {
	_, err := d.conn.Exec(
		`UPDATE users SET totp_secret = NULL, totp_enabled = 0 WHERE id = ?`, userID,
	)
	return err
}

// UsernameExists returns true if the username is already taken.
func (d *DB) UsernameExists(username string) (bool, error) {
	var count int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&count)
	return count > 0, err
}

// ─── Session queries ─────────────────────────────────────────────────────────

// CreateSession inserts a new session record.
func (d *DB) CreateSession(id string, userID int64, expiresAt time.Time) error {
	_, err := d.conn.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		id, userID, expiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	)
	return err
}

// GetSession retrieves a session by ID. Returns nil, nil if not found.
func (d *DB) GetSession(id string) (*models.Session, error) {
	row := d.conn.QueryRow(
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?`, id,
	)
	s := &models.Session{}
	var expiresAt, createdAt string
	err := row.Scan(&s.ID, &s.UserID, &expiresAt, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.ExpiresAt, _ = time.Parse("2006-01-02T15:04:05Z", expiresAt)
	if s.ExpiresAt.IsZero() {
		s.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAt)
	}
	s.CreatedAt, _ = time.Parse("2006-01-02T15:04:05Z", createdAt)
	return s, nil
}

// DeleteSession removes a session (logout).
func (d *DB) DeleteSession(id string) error {
	_, err := d.conn.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteExpiredSessions removes all expired sessions from the database.
func (d *DB) DeleteExpiredSessions() error {
	_, err := d.conn.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`)
	return err
}
