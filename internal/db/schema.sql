-- DevHive: Parallel Development State Management Schema (Normalized)

PRAGMA foreign_keys = ON;

-- ============================================
-- Master Tables
-- ============================================

-- Roles master table
CREATE TABLE IF NOT EXISTS roles (
    name TEXT PRIMARY KEY,
    description TEXT,
    role_file TEXT,
    args TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Message types master table
CREATE TABLE IF NOT EXISTS message_types (
    name TEXT PRIMARY KEY,
    description TEXT
);

-- Event types master table
CREATE TABLE IF NOT EXISTS event_types (
    name TEXT PRIMARY KEY,
    description TEXT
);

-- Insert default message types
INSERT OR IGNORE INTO message_types (name, description) VALUES
    ('info', 'General information'),
    ('warning', 'Warning message'),
    ('question', 'Question requiring response'),
    ('answer', 'Answer to a question'),
    ('system', 'System notification'),
    ('help', 'Help request'),
    ('review', 'Review request'),
    ('unblock', 'Unblock request'),
    ('clarify', 'Clarification request'),
    ('report', 'Progress report'),
    ('reply', 'Reply message'),
    ('broadcast', 'Broadcast message');

-- Insert default event types
INSERT OR IGNORE INTO event_types (name, description) VALUES
    ('sprint_created', 'Sprint was created'),
    ('sprint_completed', 'Sprint was completed'),
    ('sprint_loaded', 'Sprint was loaded from YAML'),
    ('worker_registered', 'Worker was registered'),
    ('worker_status_changed', 'Worker status changed'),
    ('worker_session_changed', 'Worker session state changed'),
    ('worker_task_updated', 'Worker task was updated'),
    ('worker_progress_updated', 'Worker progress was updated'),
    ('worker_error', 'Worker reported an error'),
    ('message_sent', 'Message was sent'),
    ('message_broadcast', 'Message was broadcast'),
    ('branch_merged', 'Branch was merged');

-- ============================================
-- Core Tables
-- ============================================

-- Sprints table
CREATE TABLE IF NOT EXISTS sprints (
    id TEXT PRIMARY KEY,
    config_file TEXT,
    project_path TEXT,
    status TEXT DEFAULT 'active' CHECK(status IN ('active', 'completed', 'aborted')),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Workers table
CREATE TABLE IF NOT EXISTS workers (
    name TEXT PRIMARY KEY,
    sprint_id TEXT NOT NULL,
    branch TEXT NOT NULL,
    role_name TEXT,
    worktree_path TEXT,
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'working', 'completed', 'blocked', 'error')),
    session_state TEXT DEFAULT 'stopped' CHECK(session_state IN ('running', 'waiting_permission', 'idle', 'stopped')),
    current_task TEXT,
    progress INTEGER DEFAULT 0 CHECK(progress >= 0 AND progress <= 100),
    activity TEXT,
    last_commit TEXT,
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sprint_id) REFERENCES sprints(id) ON DELETE CASCADE,
    FOREIGN KEY (role_name) REFERENCES roles(name) ON DELETE SET NULL
);

-- Messages table
-- Note: from_worker and to_worker do not have FK constraints to allow "pm" as sender/recipient
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_worker TEXT NOT NULL,
    to_worker TEXT NOT NULL,
    message_type TEXT DEFAULT 'info',
    subject TEXT,
    content TEXT NOT NULL,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_type) REFERENCES message_types(name) ON DELETE RESTRICT
);

-- Events table
-- Note: worker column does not have FK constraint to allow events from non-workers (e.g., pm)
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    worker TEXT,
    data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_type) REFERENCES event_types(name) ON DELETE RESTRICT
);

-- ============================================
-- Indexes
-- ============================================

CREATE INDEX IF NOT EXISTS idx_workers_sprint ON workers(sprint_id);
CREATE INDEX IF NOT EXISTS idx_workers_status ON workers(status);
CREATE INDEX IF NOT EXISTS idx_workers_role ON workers(role_name);
CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(to_worker);
CREATE INDEX IF NOT EXISTS idx_messages_from ON messages(from_worker);
CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(to_worker, read_at) WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_worker ON events(worker);
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at);
