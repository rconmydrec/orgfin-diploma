package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func main() {
	_ = godotenv.Load()

	var migrationsDir string
	flag.StringVar(&migrationsDir, "dir", "migrations", "directory with migration files")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: migrate [options] <command>")
		fmt.Println("Commands: up, down, status, create <name>")
		os.Exit(1)
	}

	command := args[0]

	// Skip gracefully if no migration files exist
	if hasMigrations, err := hasMigrationFiles(migrationsDir); err != nil {
		fmt.Printf("no migrations directory found (%s), skipping\n", migrationsDir)
		return
	} else if !hasMigrations {
		fmt.Printf("no migration files in %s, skipping\n", migrationsDir)
		return
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Println("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		fmt.Printf("failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		fmt.Printf("failed to set dialect: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			fmt.Printf("migration up failed: %v\n", err)
			os.Exit(1)
		}
	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			fmt.Printf("migration down failed: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			fmt.Printf("migration status failed: %v\n", err)
			os.Exit(1)
		}
	case "create":
		if len(args) < 2 {
			fmt.Println("Usage: migrate create <name>")
			os.Exit(1)
		}
		name := args[1]
		if err := goose.Create(db, migrationsDir, name, "sql"); err != nil {
			fmt.Printf("failed to create migration: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("unknown command: %s\n", command)
		os.Exit(1)
	}
}

func hasMigrationFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			matched, _ := filepath.Match("*.sql", e.Name())
			if matched {
				return true, nil
			}
		}
	}
	return false, nil
}
