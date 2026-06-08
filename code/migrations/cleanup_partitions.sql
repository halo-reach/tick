-- Cleanup partitions older than 1 month for executions table
-- Run as a scheduled maintenance task

DO $$
DECLARE
    partition_name TEXT;
    cutoff_date DATE := NOW() - INTERVAL '1 month';
BEGIN
    FOR partition_name IN
        SELECT tablename FROM pg_tables
        WHERE tablename LIKE 'executions_%'
        AND tablename ~ 'executions_\d{4}_\d{2}'
    LOOP
        IF substring(partition_name FROM 'executions_(\d{4}_\d{2})')::DATE < cutoff_date THEN
            EXECUTE format('DROP TABLE IF EXISTS %I', partition_name);
            RAISE NOTICE 'Dropped partition: %', partition_name;
        END IF;
    END LOOP;
END $$;
