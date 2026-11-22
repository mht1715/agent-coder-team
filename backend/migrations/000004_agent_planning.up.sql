CREATE TABLE IF NOT EXISTS requirement_documents (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL UNIQUE REFERENCES projects (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content_md TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 1,
    generated_by VARCHAR(32) NOT NULL DEFAULT 'main_agent',
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_requirement_documents_project
    ON requirement_documents (project_id);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_key VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    acceptance_criteria TEXT NOT NULL DEFAULT '',
    agent_role VARCHAR(64) NOT NULL DEFAULT '',
    priority VARCHAR(16) NOT NULL DEFAULT 'medium',
    status VARCHAR(32) NOT NULL DEFAULT 'backlog',
    position INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, task_key)
);

CREATE INDEX IF NOT EXISTS idx_tasks_project_position
    ON tasks (project_id, position);

CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    depends_on_task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, depends_on_task_id),
    CHECK (task_id <> depends_on_task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_dependencies_depends_on
    ON task_dependencies (depends_on_task_id);

CREATE TABLE IF NOT EXISTS agent_runs (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_id UUID REFERENCES tasks (id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES conversations (id) ON DELETE SET NULL,
    kind VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    title VARCHAR(255) NOT NULL,
    input_summary TEXT NOT NULL DEFAULT '',
    output_summary TEXT NOT NULL DEFAULT '',
    logs JSONB NOT NULL DEFAULT '[]'::jsonb,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_project_created
    ON agent_runs (project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_runs_task_created
    ON agent_runs (task_id, created_at DESC);

CREATE TABLE IF NOT EXISTS approvals (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_id UUID REFERENCES tasks (id) ON DELETE SET NULL,
    run_id UUID REFERENCES agent_runs (id) ON DELETE SET NULL,
    kind VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    requested_by_type VARCHAR(32) NOT NULL DEFAULT 'main_agent',
    decided_by UUID REFERENCES users (id) ON DELETE SET NULL,
    decision_note TEXT NOT NULL DEFAULT '',
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_approvals_project_status
    ON approvals (project_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_approvals_task_status
    ON approvals (task_id, status);
