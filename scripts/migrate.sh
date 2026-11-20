#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now());
SQL
for file in /migrations/*.up.sql; do
 version=$(basename "$file")
 applied=$(psql -At -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM schema_migrations WHERE version='$version'")
 if [ "$applied" = 0 ]; then
  { printf 'BEGIN;\n'; cat "$file"; printf ";\nINSERT INTO schema_migrations(version) VALUES ('%s');\nCOMMIT;\n" "$version"; } | psql -v ON_ERROR_STOP=1
 fi
done
