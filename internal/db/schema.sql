-- DevHive: Parallel Development State Management Schema

PRAGMA foreign_keys = ON;

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
    issue TEXT,
    worktree_path TEXT,
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'working', 'completed', 'blocked', 'error')),
    current_task TEXT,
    last_commit TEXT,
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sprint_id) REFERENCES sprints(id) ON DELETE CASCADE
);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_worker TEXT NOT NULL,
    to_worker TEXT,
    message_type TEXT DEFAULT 'info' CHECK(message_type IN ('info', 'warning', 'question', 'answer', 'system')),
    subject TEXT,
    content TEXT NOT NULL,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Events table
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,
    worker TEXT,
    data TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_workers_sprint ON workers(sprint_id);
CREATE INDEX IF NOT EXISTS idx_workers_status ON workers(status);
CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(to_worker);
CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(read_at) WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at);
