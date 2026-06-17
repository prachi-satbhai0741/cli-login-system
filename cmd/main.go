package main

import (
	"fmt"
	"os"

	"github.com/yourname/cli-login-system/internal/cli"
	"github.com/yourname/cli-login-system/internal/db"
)

func main() {
	// Configuration via environment variables with sensible defaults
	dbPath := getEnv("DB_PATH", "/app/data/login.db")

	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not open database at %q: %v\n", dbPath, err)
		os.Exit(1)
	}
	defer database.Close()

	// Create and run the CLI application
	app, err := cli.New(database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not initialise CLI: %v\n", err)
		os.Exit(1)
	}

	app.Run()
}

// getEnv returns the value of an environment variable or a fallback default.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
