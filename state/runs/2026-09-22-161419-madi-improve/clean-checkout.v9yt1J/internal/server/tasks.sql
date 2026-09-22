CREATE TABLE IF NOT EXISTS task_details (
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 task_id uuid NOT NULL,assignee_id uuid REFERENCES users(id) ON DELETE SET NULL,
 due_date date,status text NOT NULL DEFAULT 'todo' CHECK(status IN ('backlog','todo','doing','review','done')),
 priority text NOT NULL DEFAULT 'normal' CHECK(priority IN ('low','normal','high','urgent')),
 updated_by uuid NOT NULL REFERENCES users(id),updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(document_id,task_id)
);
CREATE INDEX IF NOT EXISTS task_due_idx ON task_details(assignee_id,due_date);
CREATE TABLE IF NOT EXISTS calendar_events (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id),owner_id uuid NOT NULL REFERENCES users(id),
 document_id uuid REFERENCES documents(id) ON DELETE CASCADE,title text NOT NULL,
 kind text NOT NULL CHECK(kind IN ('meeting','milestone')),start_date date NOT NULL,end_date date NOT NULL,
 visibility text NOT NULL CHECK(visibility IN ('private','workspace')),version integer NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),CHECK(end_date>=start_date)
);
CREATE INDEX IF NOT EXISTS calendar_dates_idx ON calendar_events(workspace_id,start_date,end_date);
