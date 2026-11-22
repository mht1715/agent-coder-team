ALTER TABLE agent_runs
    DROP COLUMN IF EXISTS changed_files,
    DROP COLUMN IF EXISTS exit_code,
    DROP COLUMN IF EXISTS executor;

ALTER TABLE tasks
    DROP COLUMN IF EXISTS execution_image,
    DROP COLUMN IF EXISTS execution_command;
