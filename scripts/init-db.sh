#!/bin/bash
set -e

# =====================================================
# Init DB: One-shot bootstrap for ingest-srv
# 1. Create role ingest_master
# 2. Run all migrations in order
# 3. Grant full access to ingest_master
# =====================================================

echo ">>> [1/3] Creating role ingest_master..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-'EOSQL'
    DO $$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'ingest_master') THEN
            CREATE ROLE ingest_master WITH LOGIN PASSWORD 'ingest_master_pwd';
        END IF;
    END
    $$;
    GRANT ALL PRIVILEGES ON DATABASE smap TO ingest_master;
    GRANT CREATE ON DATABASE smap TO ingest_master;
EOSQL

echo ">>> [2/3] Running migrations..."
for f in /migrations/*.sql; do
    echo "  → Applying $(basename "$f")..."
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$f"
done

echo ">>> [3/3] Granting access to ingest_master..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-'EOSQL'
    -- Schema ownership
    ALTER SCHEMA schema_ingest OWNER TO ingest_master;

    -- Table & sequence privileges
    GRANT USAGE ON SCHEMA schema_ingest TO ingest_master;
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA schema_ingest TO ingest_master;
    GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA schema_ingest TO ingest_master;

    -- Default privileges for future objects
    ALTER DEFAULT PRIVILEGES IN SCHEMA schema_ingest
        GRANT ALL ON TABLES TO ingest_master;
    ALTER DEFAULT PRIVILEGES IN SCHEMA schema_ingest
        GRANT USAGE, SELECT ON SEQUENCES TO ingest_master;
EOSQL

echo ">>> Done! ingest_master has full access to schema_ingest."
