package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"github.com/prachi-satbhai0741/cli-login-system/internal/auth"
	"github.com/prachi-satbhai0741/cli-login-system/internal/db"
	"github.com/prachi-satbhai0741/cli-login-system/internal/models"
	"github.com/prachi-satbhai0741/cli-login-system/internal/session"
	"github.com/prachi-satbhai0741/cli-login-system/internal/totp"
)

const (
	appIssuer       = "CLI-Login-System"
	sessionTimeout  = 30 * time.Minute
	banner = `
╔══════════════════════════════════════╗
║       CLI Login System v1.0          ║
║   Type 'help' to see commands        ║
╚══════════════════════════════════════╝
`
)

// App is the top-level CLI application.
type App struct {
	db          *db.DB
	authSvc     *auth.Service
	sessionMgr  *session.Manager
	rl          *readline.Instance

	// State
	loggedIn    bool
	currentUser *models.User
	currentSess *models.Session
}

// New creates and configures the CLI application.
func New(database *db.DB) (*App, error) {
	app := &App{
		db:         database,
		authSvc:    auth.New(database),
		sessionMgr: session.New(database),
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "/tmp/cli-login-history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    app.completer(),
	})
	if err != nil {
		return nil, fmt.Errorf("init readline: %w", err)
	}
	app.rl = rl
	return app, nil
}

// Run starts the interactive CLI loop.
func (a *App) Run() {
	fmt.Print(banner)
	a.printHelp()
	fmt.Println()

	// Clean up stale sessions on startup
	_ = a.sessionMgr.CleanupExpired()

	for {
		// Refresh prompt based on login state
		if a.loggedIn {
			a.rl.SetPrompt(fmt.Sprintf("[%s]> ", a.currentUser.Username))
		} else {
			a.rl.SetPrompt("> ")
		}

		line, err := a.rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				a.handleExit()
			}
			break
		}

		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}

		a.dispatch(cmd)
	}
}

// dispatch routes a command string to the correct handler.
func (a *App) dispatch(cmd string) {
	// Commands available when NOT logged in
	if !a.loggedIn {
		switch cmd {
		case "register":
			a.handleRegister()
		case "login":
			a.handleLogin()
		case "help":
			a.printHelp()
		case "exit", "quit":
			a.handleExit()
		default:
			a.errorf("Unknown command: %q. Type 'help' for available commands.", cmd)
		}
		return
	}

	// Commands available when logged in
	// First validate the session is still active
	if a.currentSess != nil && a.currentSess.IsExpired() {
		a.infof("Your session has expired. Please login again.")
		a.loggedIn = false
		a.currentUser = nil
		a.currentSess = nil
		return
	}

	switch cmd {
	case "whoami":
		a.handleWhoami()
	case "enable-2fa":
		a.handleEnable2FA()
	case "disable-2fa":
		a.handleDisable2FA()
	case "logout":
		a.handleLogout()
	case "help":
		a.printHelp()
	case "exit", "quit":
		a.handleExit()
	default:
		a.errorf("Unknown command: %q. Type 'help' for available commands.", cmd)
	}
}

// ─── Command handlers ────────────────────────────────────────────────────────

func (a *App) handleRegister() {
	fmt.Println()
	a.infof("=== Register New Account ===")

	username, err := a.prompt("Username: ")
	if err != nil {
		return
	}
	username = strings.TrimSpace(username)

	password, err := a.promptPassword("Password (min 8 chars): ")
	if err != nil {
		return
	}

	confirm, err := a.promptPassword("Confirm password: ")
	if err != nil {
		return
	}

	if password != confirm {
		a.errorf("Passwords do not match.")
		return
	}

	if err := a.authSvc.Register(username, password); err != nil {
		switch {
		case errors.Is(err, auth.ErrUsernameTaken):
			a.errorf("Username %q is already taken. Please choose another.", username)
		default:
			a.errorf("Registration failed: %v", err)
		}
		return
	}

	a.successf("Account created successfully! You can now login with 'login'.")
	fmt.Println()
}

func (a *App) handleLogin() {
	fmt.Println()
	a.infof("=== Login ===")

	username, err := a.prompt("Username: ")
	if err != nil {
		return
	}
	username = strings.TrimSpace(username)

	password, err := a.promptPassword("Password: ")
	if err != nil {
		return
	}

	// First attempt without TOTP to check if it's required
	result, err := a.authSvc.Login(username, password, "", sessionTimeout)
	if err != nil {
		if errors.Is(err, auth.ErrTOTPRequired) {
			// 2FA is enabled — ask for the code
			code, promptErr := a.prompt("TOTP code (6 digits): ")
			if promptErr != nil {
				return
			}
			code = strings.TrimSpace(code)

			// Validate the TOTP code before calling Login again
			user, dbErr := a.db.GetUserByUsername(username)
			if dbErr != nil || user == nil {
				a.errorf("Login failed.")
				return
			}
			if !totp.Validate(code, user.TOTPSecret) {
				a.errorf("Invalid TOTP code.")
				return
			}

			// Re-call Login now that TOTP is verified (pass code to signal it's been checked)
			result, err = a.authSvc.Login(username, password, code, sessionTimeout)
			if err != nil {
				a.loginError(err)
				return
			}
		} else {
			a.loginError(err)
			return
		}
	}

	a.loggedIn = true
	a.currentUser = result.User
	a.currentSess = result.Session

	fmt.Println()
	a.successf("Login successful! Welcome back, %s.", a.currentUser.Username)
	a.displayUserDetails()
}

func (a *App) handleWhoami() {
	a.displayUserDetails()
}

func (a *App) handleEnable2FA() {
	fmt.Println()
	if a.currentUser.TOTPEnabled {
		a.errorf("2FA is already enabled on your account.")
		return
	}

	a.infof("=== Enable Two-Factor Authentication ===")

	secret, otpauthURL, err := totp.GenerateSecret(appIssuer, a.currentUser.Username)
	if err != nil {
		a.errorf("Failed to generate TOTP secret: %v", err)
		return
	}

	fmt.Println()
	fmt.Println("  Scan the QR code with Google Authenticator or similar app:")
	fmt.Println()
	fmt.Printf("  OTPAuth URL: %s\n", otpauthURL)
	fmt.Println()
	fmt.Printf("  Or enter this secret manually: %s\n", secret)
	fmt.Println()

	// Ask user to confirm with a code before saving
	code, err := a.prompt("Enter the 6-digit code from your app to confirm: ")
	if err != nil {
		return
	}
	code = strings.TrimSpace(code)

	if !totp.Validate(code, secret) {
		a.errorf("Code verification failed. 2FA was NOT enabled.")
		return
	}

	if err := a.db.SetTOTPSecret(a.currentUser.ID, secret); err != nil {
		a.errorf("Failed to save 2FA settings: %v", err)
		return
	}

	// Update local state
	a.currentUser.TOTPEnabled = true
	a.currentUser.TOTPSecret = secret

	a.successf("2FA enabled successfully! Future logins will require a TOTP code.")
	fmt.Println()
}

func (a *App) handleDisable2FA() {
	fmt.Println()
	if !a.currentUser.TOTPEnabled {
		a.errorf("2FA is not enabled on your account.")
		return
	}

	a.infof("=== Disable Two-Factor Authentication ===")

	// Require current TOTP code to disable
	code, err := a.prompt("Enter your current TOTP code to confirm: ")
	if err != nil {
		return
	}
	code = strings.TrimSpace(code)

	if !totp.Validate(code, a.currentUser.TOTPSecret) {
		a.errorf("Invalid TOTP code. 2FA was NOT disabled.")
		return
	}

	if err := a.db.DisableTOTP(a.currentUser.ID); err != nil {
		a.errorf("Failed to disable 2FA: %v", err)
		return
	}

	a.currentUser.TOTPEnabled = false
	a.currentUser.TOTPSecret = ""

	a.successf("2FA has been disabled.")
	fmt.Println()
}

func (a *App) handleLogout() {
	if a.currentSess != nil {
		if err := a.sessionMgr.Destroy(a.currentSess.ID); err != nil {
			a.errorf("Warning: could not remove session from DB: %v", err)
		}
	}
	username := a.currentUser.Username
	a.loggedIn = false
	a.currentUser = nil
	a.currentSess = nil
	a.successf("Goodbye, %s! You have been logged out.", username)
	fmt.Println()
}

func (a *App) handleExit() {
	if a.loggedIn {
		a.handleLogout()
	}
	fmt.Println("Bye!")
	a.rl.Close()
	os.Exit(0)
}

// ─── Display helpers ─────────────────────────────────────────────────────────

func (a *App) displayUserDetails() {
	u := a.currentUser
	s := a.currentSess

	mfaStatus := "disabled"
	if u.TOTPEnabled {
		mfaStatus = "enabled"
	}

	lastLogin := "never"
	if u.LastLogin != nil {
		lastLogin = u.LastLogin.Local().Format("2006-01-02 15:04:05")
	}

	sessionExpiry := "unknown"
	if s != nil {
		sessionExpiry = session.FormatExpiry(s)
	}

	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────┐")
	fmt.Printf("  │  Username   : %-22s│\n", u.Username)
	fmt.Printf("  │  Registered : %-22s│\n", u.CreatedAt.Local().Format("2006-01-02"))
	fmt.Printf("  │  2FA Status : %-22s│\n", mfaStatus)
	fmt.Printf("  │  Last login : %-22s│\n", lastLogin)
	fmt.Printf("  │  Session    : expires in %-12s│\n", sessionExpiry)
	fmt.Println("  └─────────────────────────────────────┘")
	fmt.Println()
}

func (a *App) printHelp() {
	fmt.Println()
	if !a.loggedIn {
		fmt.Println("  Available commands (not logged in):")
		fmt.Println("    register   — create a new account")
		fmt.Println("    login      — login with username and password")
		fmt.Println("    help       — show this help message")
		fmt.Println("    exit       — quit the program")
	} else {
		fmt.Println("  Available commands (logged in):")
		fmt.Println("    whoami     — show your account details")
		fmt.Println("    enable-2fa — enable TOTP two-factor authentication")
		fmt.Println("    disable-2fa— disable two-factor authentication")
		fmt.Println("    logout     — end your session")
		fmt.Println("    help       — show this help message")
		fmt.Println("    exit       — logout and quit")
	}
	fmt.Println()
}

// ─── Input helpers ────────────────────────────────────────────────────────────

// prompt reads a line with the given label.
func (a *App) prompt(label string) (string, error) {
	a.rl.SetPrompt(label)
	defer func() {
		if a.loggedIn && a.currentUser != nil {
			a.rl.SetPrompt(fmt.Sprintf("[%s]> ", a.currentUser.Username))
		} else {
			a.rl.SetPrompt("> ")
		}
	}()
	line, err := a.rl.Readline()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// promptPassword reads a password with echo disabled.
func (a *App) promptPassword(label string) (string, error) {
	pw, err := a.rl.ReadPassword(label)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// ─── Output helpers ──────────────────────────────────────────────────────────

func (a *App) errorf(format string, args ...any) {
	fmt.Printf("  ✗ "+format+"\n", args...)
}

func (a *App) successf(format string, args ...any) {
	fmt.Printf("  ✓ "+format+"\n", args...)
}

func (a *App) infof(format string, args ...any) {
	fmt.Printf("  "+format+"\n", args...)
}

func (a *App) loginError(err error) {
	switch {
	case errors.Is(err, auth.ErrAccountLocked):
		a.errorf("%v", err)
	case errors.Is(err, auth.ErrInvalidCredentials):
		a.errorf("Invalid username or password.")
	default:
		a.errorf("Login error: %v", err)
	}
}

// ─── Tab completion ───────────────────────────────────────────────────────────

func (a *App) completer() readline.AutoCompleter {
	loggedOutCmds := []readline.PrefixCompleterInterface{
		readline.PcItem("register"),
		readline.PcItem("login"),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	}
	loggedInCmds := []readline.PrefixCompleterInterface{
		readline.PcItem("whoami"),
		readline.PcItem("enable-2fa"),
		readline.PcItem("disable-2fa"),
		readline.PcItem("logout"),
		readline.PcItem("help"),
		readline.PcItem("exit"),
	}

	// Return a dynamic completer
	return readline.NewPrefixCompleter(append(loggedOutCmds, loggedInCmds...)...)
}
