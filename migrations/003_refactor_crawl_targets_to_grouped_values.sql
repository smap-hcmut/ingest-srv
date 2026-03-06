BEGIN;

ALTER TABLE schema_ingest.crawl_targets
    ADD COLUMN values JSONB;

UPDATE schema_ingest.crawl_targets
SET values = CASE
    WHEN value IS NULL OR btrim(value) = '' THEN '[]'::jsonb
    ELSE jsonb_build_array(value)
END;

ALTER TABLE schema_ingest.crawl_targets
    ALTER COLUMN values SET NOT NULL;

ALTER TABLE schema_ingest.crawl_targets
    DROP COLUMN value;

COMMIT;
