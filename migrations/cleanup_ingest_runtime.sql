-- Purpose:
-- Clean runtime data in ingest without dropping schema objects or defaults.
--
-- This script preserves:
-- - enum types
-- - tables / indexes / triggers
-- - crawl_mode_defaults seed data
--
-- Why this order:
-- - raw_batches depends on external_tasks and data_sources
-- - external_tasks depends on scheduled_jobs and data_sources
-- - scheduled_jobs depends on data_sources
-- - dryrun_results depends on data_sources
-- - crawl_mode_changes depends on data_sources
-- - crawl_targets depends on data_sources
-- - data_sources has a back-reference dryrun_last_result_id -> dryrun_results.id
--
-- The back-reference must be nulled first, otherwise deleting dryrun_results/data_sources
-- can fail with FK violations.

BEGIN;

SET search_path TO ingest;

-- -------------------------------------------------------------------
-- Break the data_sources <-> dryrun_results back-reference first
-- -------------------------------------------------------------------
UPDATE data_sources
SET dryrun_last_result_id = NULL
WHERE dryrun_last_result_id IS NOT NULL;

-- -------------------------------------------------------------------
-- Delete runtime tables from deepest child to root
-- -------------------------------------------------------------------
DELETE FROM raw_batches;
DELETE FROM external_tasks;
DELETE FROM scheduled_jobs;
DELETE FROM dryrun_results;
DELETE FROM crawl_mode_changes;
DELETE FROM crawl_targets;
DELETE FROM data_sources;

COMMIT;

-- -------------------------------------------------------------------
-- Optional inspection queries
-- -------------------------------------------------------------------
-- SELECT 'data_sources' AS table_name, COUNT(*) FROM ingest.data_sources
-- UNION ALL
-- SELECT 'crawl_targets', COUNT(*) FROM ingest.crawl_targets
-- UNION ALL
-- SELECT 'dryrun_results', COUNT(*) FROM ingest.dryrun_results
-- UNION ALL
-- SELECT 'scheduled_jobs', COUNT(*) FROM ingest.scheduled_jobs
-- UNION ALL
-- SELECT 'external_tasks', COUNT(*) FROM ingest.external_tasks
-- UNION ALL
-- SELECT 'raw_batches', COUNT(*) FROM ingest.raw_batches
-- UNION ALL
-- SELECT 'crawl_mode_changes', COUNT(*) FROM ingest.crawl_mode_changes
-- UNION ALL
-- SELECT 'crawl_mode_defaults', COUNT(*) FROM ingest.crawl_mode_defaults;
