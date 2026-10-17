#!/usr/bin/env sh
# Regenerates db/schema/schema.sql (the sqlc schema snapshot) by concatenating
# the up migrations in order. Run after adding or changing any migration.
# Usage: sh scripts/schema-snapshot.sh   (from the backend/ directory)
set -eu
cd "$(dirname "$0")/.."
out="db/schema/schema.sql"
{
  echo "-- GENERATED FILE. Do not edit by hand."
  echo "-- Canonical schema snapshot for sqlc, built from db/migrations/*.up.sql"
  echo "-- Regenerate with: sh scripts/schema-snapshot.sh"
  for f in db/migrations/*.up.sql; do
    echo
    echo "-- ---- $(basename "$f") ----"
    cat "$f"
  done
} > "$out"
echo "wrote $out"
