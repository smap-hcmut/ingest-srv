BEGIN;

SET search_path TO ingest, public;

ALTER TABLE ingest.external_tasks
    ADD COLUMN IF NOT EXISTS domain_type_code VARCHAR(50);

UPDATE ingest.external_tasks
SET domain_type_code = '_default'
WHERE domain_type_code IS NULL OR BTRIM(domain_type_code) = '';

ALTER TABLE ingest.external_tasks
    ALTER COLUMN domain_type_code SET DEFAULT '_default';

ALTER TABLE ingest.external_tasks
    ALTER COLUMN domain_type_code SET NOT NULL;

ALTER TABLE ingest.raw_batches
    ADD COLUMN IF NOT EXISTS domain_type_code VARCHAR(50);

UPDATE ingest.raw_batches
SET domain_type_code = '_default'
WHERE domain_type_code IS NULL OR BTRIM(domain_type_code) = '';

ALTER TABLE ingest.raw_batches
    ALTER COLUMN domain_type_code SET DEFAULT '_default';

ALTER TABLE ingest.raw_batches
    ALTER COLUMN domain_type_code SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_external_tasks_domain_type_created_desc
    ON ingest.external_tasks (domain_type_code, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_raw_batches_domain_type_received_desc
    ON ingest.raw_batches (domain_type_code, received_at DESC);

COMMIT;
