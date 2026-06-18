# CLI Login System

A secure command-line login system built in Go with user registration, bcrypt password hashing, optional TOTP-based 2FA, session management, and account lockout. Runs entirely in Docker with a persistent SQLite database — no local Go installation needed.

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) 24+
- [Docker Compose](https://docs.docker.com/compose/install/) v2+

That's it — no Go installation required to run the project.

---

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/prachi-satbhai0741/cli-login-system.git
cd cli-login-system

# 2. Build and run
docker compose run app
```

You will see the welcome banner and can start typing commands immediately.

---

## Usage

### Commands before login

| Command    | Description                      |
|------------|----------------------------------|
| `register` | Create a new user account        |
| `login`    | Login with username and password |
| `help`     | Show available commands          |
| `exit`     | Quit the program                 |

### Commands after login

| Command       | Description                           |
|---------------|---------------------------------------|
| `whoami`      | Show your account details             |
| `enable-2fa`  | Enable TOTP two-factor authentication |
| `disable-2fa` | Disable two-factor authentication     |
| `logout`      | End your current session              |
| `help`        | Show available commands               |
| `exit`        | Logout and quit                       |

---

## What is shown after login

After a successful login, the system automatically displays:

- Username
- Registration date
- 2FA status (enabled / disabled)
- Session expiration time (30 minutes from login)
- Last login time

---

## Setting up 2FA

1. Run `enable-2fa` after logging in
2. Copy the **OTPAuth URL** and paste it into a QR code generator (e.g. [qr-code-generator.com](https://www.qr-code-generator.com)), or enter the secret manually
3. Scan the QR code or enter the secret in **Google Authenticator**, **Authy**, or any TOTP-compatible app
4. Enter the 6-digit code shown in the app to confirm setup
5. All future logins will require this 6-digit code

To disable 2FA, run `disable-2fa` and confirm with your current TOTP code.

---

## Account Lockout

After **5 consecutive failed login attempts**, the account is locked for **15 minutes**. This resets automatically after the lockout period.

---

## Session Management

Sessions expire after **30 minutes of inactivity**. The remaining time is shown after login via `whoami`. Run `logout` to end the session immediately.

---

## Configuration

The following environment variables can be set in `docker-compose.yml`:

| Variable  | Default              | Description                      |
|-----------|----------------------|----------------------------------|
| `DB_PATH` | `/app/data/login.db` | Path to the SQLite database file |

---

## Database

- SQLite runs inside the container
- Data is persisted across container restarts via a Docker named volume (`db-data`)
- Schema is applied automatically on first run — no manual setup required
- The reference schema is at `migrations/001_init.sql`

---

## Running Tests

```bash
# Run all tests (requires Go installed locally)
go test ./...

# Or run tests inside Docker
docker compose run --entrypoint sh app -c "cd /build && go test ./..."
```

---

## Project Structure

```
cli-login-system/
├── cmd/
│   └── main.go              # Entry point
├── internal/
│   ├── auth/                # Password hashing, login, lockout logic
│   ├── cli/                 # Interactive prompt and command handlers
│   ├── db/                  # SQLite connection and all queries
│   ├── models/              # User and Session structs
│   ├── session/             # Session validation and cleanup
│   └── totp/                # TOTP secret generation and validation
├── migrations/
│   └── 001_init.sql         # Database schema (reference copy)
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## Security Notes

- Passwords hashed with **bcrypt at cost 12** — never stored in plain text
- **Generic error messages** on failed login to prevent username enumeration
- **Account lockout** after 5 failed attempts; locked for 15 minutes
- **Sessions expire** after 30 minutes of inactivity
- All queries use **parameterised statements** to prevent SQL injection
- TOTP confirmation required to both **enable and disable** 2FA
- SQLite runs with **WAL mode** for safe concurrent access

---
## License

MIT