DROP INDEX IF EXISTS idx_check_results_job_id;

ALTER TABLE check_results
    DROP COLUMN IF EXISTS job_id;
