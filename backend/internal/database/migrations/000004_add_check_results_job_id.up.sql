-- Store the originating job id so the same RabbitMQ message
-- cannot insert two CheckResult rows.

ALTER TABLE check_results
    ADD COLUMN job_id UUID;

CREATE UNIQUE INDEX idx_check_results_job_id
    ON check_results (job_id)
    WHERE job_id IS NOT NULL;
