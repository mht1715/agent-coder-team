CREATE TABLE IF NOT EXISTS project_profiles (
    project_id UUID PRIMARY KEY REFERENCES projects (id) ON DELETE CASCADE,
    primary_language VARCHAR(64) NOT NULL DEFAULT '',
    languages JSONB NOT NULL DEFAULT '[]'::jsonb,
    frameworks JSONB NOT NULL DEFAULT '[]'::jsonb,
    package_manager VARCHAR(64) NOT NULL DEFAULT '',
    build_command TEXT NOT NULL DEFAULT '',
    test_command TEXT NOT NULL DEFAULT '',
    format_command TEXT NOT NULL DEFAULT '',
    file_count INTEGER NOT NULL DEFAULT 0,
    top_level JSONB NOT NULL DEFAULT '[]'::jsonb,
    truncated BOOLEAN NOT NULL DEFAULT FALSE,
    detected_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);