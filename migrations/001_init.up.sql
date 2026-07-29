CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (char_length(trim(name)) BETWEEN 1 AND 120),
    email TEXT NOT NULL UNIQUE CHECK (email = lower(email)),
    password_hash TEXT NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (char_length(trim(name)) BETWEEN 1 AND 160),
    key TEXT NOT NULL UNIQUE CHECK (key ~ '^[A-Z][A-Z0-9]{1,9}$'),
    description TEXT NOT NULL DEFAULT '',
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    is_archived BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);

CREATE TABLE project_issue_counters (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    next_sequence BIGINT NOT NULL DEFAULT 1 CHECK (next_sequence > 0)
);

CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    sequence_number BIGINT NOT NULL CHECK (sequence_number > 0),
    identifier TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL CHECK (char_length(trim(title)) BETWEEN 1 AND 240),
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL CHECK (type IN ('task', 'bug', 'feature')),
    status TEXT NOT NULL CHECK (status IN ('backlog', 'todo', 'in_progress', 'done', 'cancelled')),
    priority TEXT NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    due_date DATE,
    position BIGINT NOT NULL DEFAULT 0 CHECK (position >= 0),
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, sequence_number)
);

CREATE TABLE issue_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action TEXT NOT NULL,
    field_name TEXT,
    old_value TEXT,
    new_value TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX project_members_user_idx ON project_members(user_id);
CREATE INDEX issues_project_status_position_idx ON issues(project_id, status, position) WHERE deleted_at IS NULL;
CREATE INDEX issues_project_updated_idx ON issues(project_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX issues_project_priority_idx ON issues(project_id, priority) WHERE deleted_at IS NULL;
CREATE INDEX issues_project_type_idx ON issues(project_id, type) WHERE deleted_at IS NULL;
CREATE INDEX issues_assignee_idx ON issues(assignee_id) WHERE deleted_at IS NULL;
CREATE INDEX issues_reporter_idx ON issues(reporter_id) WHERE deleted_at IS NULL;
CREATE INDEX issue_activities_issue_created_idx ON issue_activities(issue_id, created_at DESC);

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER projects_set_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER issues_set_updated_at BEFORE UPDATE ON issues FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE FUNCTION protect_project_owner_membership() RETURNS TRIGGER AS $$
DECLARE owner UUID;
BEGIN
    SELECT owner_id INTO owner FROM projects WHERE id = OLD.project_id;
    IF owner = OLD.user_id AND (TG_OP = 'DELETE' OR NEW.role <> 'owner') THEN
        RAISE EXCEPTION 'project owner membership cannot be removed or downgraded';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER project_owner_membership_guard
BEFORE UPDATE OR DELETE ON project_members
FOR EACH ROW EXECUTE FUNCTION protect_project_owner_membership();

CREATE OR REPLACE FUNCTION ensure_project_owner_membership() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO project_members(project_id, user_id, role) VALUES (NEW.id, NEW.owner_id, 'owner')
    ON CONFLICT (project_id, user_id) DO UPDATE SET role = 'owner';
    INSERT INTO project_issue_counters(project_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER project_owner_membership_insert
AFTER INSERT ON projects FOR EACH ROW EXECUTE FUNCTION ensure_project_owner_membership();
