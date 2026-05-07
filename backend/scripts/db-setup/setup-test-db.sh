#!/usr/bin/env bash
#
# Setup test database for the Go Budget backend.
#
# Creates (or recreates) a PostgreSQL database, applies schema, seeds reference
# data, and runs any pending Goose migrations.
#
# Usage:
#   ./setup-test-db.sh [options]
#
# Options:
#   --name NAME       Database name       (default: budgeter_test)
#   --user USER       PostgreSQL user      (default: postgres)
#   --password PASS   PostgreSQL password  (default: 123)
#   --host HOST       PostgreSQL host      (default: localhost)
#   --port PORT       PostgreSQL port      (default: 5432)
#   --keep            Do NOT drop existing database (default: drop and recreate)
#   --help            Show this help message

set -euo pipefail

# ------------------------------------------------------------------
# Defaults
# ------------------------------------------------------------------
DB_NAME="budgeter_test"
DB_USER="postgres"
DB_PASSWORD="123"
DB_HOST="localhost"
DB_PORT="5432"
DROP_EXISTING=true

# ------------------------------------------------------------------
# Parse arguments
# ------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --name)     DB_NAME="$2";     shift 2 ;;
        --user)     DB_USER="$2";     shift 2 ;;
        --password) DB_PASSWORD="$2"; shift 2 ;;
        --host)     DB_HOST="$2";     shift 2 ;;
        --port)     DB_PORT="$2";     shift 2 ;;
        --keep)     DROP_EXISTING=false; shift ;;
        --help)
            sed -n '2,/^$/p' "$0" | sed 's/^# \?//'
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# ------------------------------------------------------------------
# Resolve paths
# ------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_SQL="${SCRIPT_DIR}/schema.sql"
SEEDS_SQL="${SCRIPT_DIR}/seeds.sql"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MIGRATIONS_DIR="${BACKEND_DIR}/migrations"

for f in "$SCHEMA_SQL" "$SEEDS_SQL"; do
    if [[ ! -f "$f" ]]; then
        echo "ERROR: Required file not found: $f" >&2
        exit 1
    fi
done

# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------
export PGPASSWORD="$DB_PASSWORD"
PSQL_CONN=(-h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER")

run_psql() {
    psql "${PSQL_CONN[@]}" "$@"
}

run_psql_db() {
    psql "${PSQL_CONN[@]}" -d "$DB_NAME" "$@"
}

echo "=== Test Database Setup ==="
echo "  Database: ${DB_NAME}"
echo "  Host:     ${DB_HOST}:${DB_PORT}"
echo "  User:     ${DB_USER}"
echo ""

# ------------------------------------------------------------------
# Step 1: Create (or recreate) the database
# ------------------------------------------------------------------
DB_EXISTS=$(run_psql -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}';" 2>/dev/null || true)

if [[ "$DB_EXISTS" == "1" ]]; then
    if [[ "$DROP_EXISTING" == true ]]; then
        echo "[1/4] Dropping existing database '${DB_NAME}'..."
        # Terminate active connections
        run_psql -d postgres -c "
            SELECT pg_terminate_backend(pid)
            FROM pg_stat_activity
            WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();
        " > /dev/null 2>&1 || true
        run_psql -d postgres -c "DROP DATABASE \"${DB_NAME}\";"
        echo "[1/4] Creating database '${DB_NAME}'..."
        run_psql -d postgres -c "CREATE DATABASE \"${DB_NAME}\";"
    else
        echo "[1/4] Database '${DB_NAME}' already exists (--keep mode, skipping)"
    fi
else
    echo "[1/4] Creating database '${DB_NAME}'..."
    run_psql -d postgres -c "CREATE DATABASE \"${DB_NAME}\";"
fi

# ------------------------------------------------------------------
# Step 2: Apply schema
# ------------------------------------------------------------------
echo "[2/4] Applying schema..."
run_psql_db -f "$SCHEMA_SQL" > /dev/null

# ------------------------------------------------------------------
# Step 3: Apply seed data
# ------------------------------------------------------------------
echo "[3/4] Inserting seed data..."
run_psql_db -f "$SEEDS_SQL" > /dev/null

# ------------------------------------------------------------------
# Step 4: Run pending Goose migrations (for any new migrations beyond schema)
# ------------------------------------------------------------------
echo "[4/4] Running pending Goose migrations..."
if [[ -d "$MIGRATIONS_DIR" ]]; then
    DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
    (cd "$BACKEND_DIR" && DATABASE_URL="$DATABASE_URL" go run ./cmd/migrate up 2>&1) || {
        echo "WARNING: Goose migrations failed (may be OK if no pending migrations)" >&2
    }
else
    echo "  No migrations directory found, skipping"
fi

echo ""
echo "=== Done! Database '${DB_NAME}' is ready ==="
