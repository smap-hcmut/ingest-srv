-- Add the per-project covering index missing from migration 001 so the
-- dashboard query
--   SELECT * FROM ingest.dryrun_results WHERE project_id = $1
--   ORDER BY created_at DESC LIMIT $2
-- stops seq-scanning once the table grows past a few thousand rows per
-- project.

CREATE INDEX IF NOT EXISTS idx_dryrun_results_project_created
    ON ingest.dryrun_results (project_id, created_at DESC);

ANALYZE ingest.dryrun_results;
