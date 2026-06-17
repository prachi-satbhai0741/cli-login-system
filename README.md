# CLI Login System

A secure command-line login system built in Go with user registration, password authentication, optional TOTP-based 2FA, session management, and account lockout. Runs in Docker with a persistent SQLite database.

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) 24+
- [Docker Compose](https://docs.docker.com/compose/install/) v2+

That's it — no Go installation required to run the project.

---

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/yourname/cli-login-system.git
cd cli-login-system

# 2. Build and run
docker compose run app
```

You will see the welcome banner and can start typing commands immediately.

---

## Usage

### Commands before login

| Command    | Description                        |
|------------|------------------------------------|
| `register` | Create a new user account          |
| `login`    | Login with username and password   |
| `help`     | Show available commands            |
| `exit`     | Quit the program                   |

### Commands after login

| Command       | Description                             |
|---------------|-----------------------------------------|
| `whoami`      | Show your account details               |
| `enable-2fa`  | Enable TOTP two-factor authentication   |
| `disable-2fa` | Disable two-factor authentication       |
| `logout`      | End your current session                |
| `help`        | Show available commands                 |
| `exit`        | Logout and quit                         |

---

## Setting up 2FA

1. Run `enable-2fa` after logging in
2. Copy the **OTPAuth URL** and paste it into a QR code generator (e.g. [qr-code-generator.com](https://www.qr-code-generator.com)), or enter the secret manually in your authenticator app
3. Scan the QR code or enter the secret in **Google Authenticator**, **Authy**, or any compatible app
4. Enter the 6-digit code shown in the app to confirm setup
5. All future logins will require this code

---

## Configuration

The following environment variables can be set in `docker-compose.yml`:

| Variable   | Default              | Description                      |
|------------|----------------------|----------------------------------|
| `DB_PATH`  | `/app/data/login.db` | Path to the SQLite database file |

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

- Passwords are hashed with **bcrypt at cost 12** — never stored in plain text
- **Generic error messages** are used on failed login to prevent username enumeration
- **Account lockout** triggers after 5 failed attempts; locked for 15 minutes
- **Sessions expire** after 30 minutes of inactivity
- All database queries use **parameterised statements** to prevent SQL injection
- TOTP confirmation is required both to **enable** and **disable** 2FA

---

