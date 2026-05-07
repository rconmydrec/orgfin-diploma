// Command reconcile-balances is a one-shot CLI that detects accounts whose
// stored accounts.balance has drifted from a recalc-from-scratch result and
// inserts a single compensating is_adjustment=true transaction per
// divergent account so the next call to recalculateBalances lands on the
// user's currently trusted balance.
//
// Usage:
//
//	reconcile-balances [--dry-run] [--user-id=N] [--account-id=N] [--verbose]
//
// Build:
//
//	go build -o bin/reconcile-balances ./cmd/reconcile-balances
//
// The CLI builds its DSN through internal/config (same path the main app
// uses): DATABASE_URL is honored if set, otherwise the DSN is assembled
// from DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME. A .env file is loaded
// via godotenv if present. The CLI opens a *sql.DB and delegates to Run.
// It is NOT wired into the HTTP server, the Asynq worker, or any scheduled
// job — it is a manual one-shot command intended to fix legacy
// stored-vs-recalc drift introduced before adjustment-type transactions
// existed in the schema.
//
// Idempotency: a second invocation immediately after a successful first
// one inserts zero rows.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-budget/backend/internal/config"
	_ "github.com/lib/pq"
)

func main() {
	var opts Options
	flag.BoolVar(&opts.DryRun, "dry-run", false, "print the divergence table without writing any rows")
	flag.IntVar(&opts.UserID, "user-id", 0, "scope the run to a single user (0 = all users)")
	flag.IntVar(&opts.AccountID, "account-id", 0, "scope the run to a single account (0 = all accounts in scope)")
	flag.BoolVar(&opts.Verbose, "verbose", false, "print a short summary line in addition to the table/inserts")
	flag.Parse()

	// config.New() loads .env (if present), honors DATABASE_URL when set,
	// and otherwise builds the DSN from DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME.
	cfg := config.New()
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "database connection settings are required: set DATABASE_URL or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME")
		os.Exit(2)
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to ping database: %v\n", err)
		os.Exit(1)
	}

	if err := Run(db, opts, os.Stdout, time.Now().UTC()); err != nil {
		fmt.Fprintf(os.Stderr, "reconcile-balances failed: %v\n", err)
		os.Exit(1)
	}
}
