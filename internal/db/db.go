package db

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

// DefaultDBPath returns the default database path
func DefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".devhive", "state.db")
}

// DB wraps the database connection
type DB struct {
	conn *sql.DB
}

// Open opens or creates the database
func Open(path string) (*DB, error) {
	if path == "" {
		path = DefaultDBPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.init(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) init() error {
	_, err := db.conn.Exec(schema)
	return err
}

// Close closes the database
func (db *DB) Close() error {
	return db.conn.Close()
}

// ============================================
// Data Structures
// ============================================

// Role represents a worker role
type Role struct {
	Name        string
	Description string
	RoleFile    string
	CreatedAt   time.Time
}

// Sprint represents a sprint
type Sprint struct {
	ID          string
	ConfigFile  string
	ProjectPath string
	Status      string
	StartedAt   time.Time
	CompletedAt *time.Time
}

// Worker represents a worker
type Worker struct {
	Name           string
	SprintID       string
	Branch         string
	RoleName       string // FK to roles.name
	RoleFile       string // From joined roles table
	WorktreePath   string
	Status         string
	CurrentTask    string
	LastCommit     string
	ErrorCount     int
	LastError      string
	UpdatedAt      time.Time
	UnreadMessages int
}

// Message represents a message
type Message struct {
	ID          int
	FromWorker  string
	ToWorker    string
	MessageType string
	Subject     string
	Content     string
	ReadAt      *time.Time
	CreatedAt   time.Time
}

// Event represents an event
type Event struct {
	ID        int
	EventType string
	Worker    string
	Data      string
	CreatedAt time.Time
}

// ============================================
// Helper Functions
// ============================================

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// logEvent logs an event
func (db *DB) logEvent(eventType, worker string, data map[string]interface{}) error {
	var dataJSON *string
	if data != nil {
		b, _ := json.Marshal(data)
		s := string(b)
		dataJSON = &s
	}
	_, err := db.conn.Exec(
		"INSERT INTO events (event_type, worker, data) VALUES (?, ?, ?)",
		eventType, nullString(worker), dataJSON,
	)
	return err
}

// checkRowsAffected verifies that at least one row was affected by an update/delete
func checkRowsAffected(result sql.Result, entityType, name string) error {
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("%s not found: %s", entityType, name)
	}
	return nil
}

// ============================================
// Role Operations
// ============================================

// CreateRole creates a new role
func (db *DB) CreateRole(name, description, roleFile string) error {
	_, err := db.conn.Exec(
		"INSERT INTO roles (name, description, role_file) VALUES (?, ?, ?)",
		name, nullString(description), nullString(roleFile),
	)
	return err
}

// GetRole returns a role by name
func (db *DB) GetRole(name string) (*Role, error) {
	row := db.conn.QueryRow(`
		SELECT name, COALESCE(description, ''), COALESCE(role_file, ''), created_at
		FROM roles WHERE name = ?
	`, name)

	var r Role
	err := row.Scan(&r.Name, &r.Description, &r.RoleFile, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetAllRoles returns all roles
func (db *DB) GetAllRoles() ([]Role, error) {
	rows, err := db.conn.Query(`
		SELECT name, COALESCE(description, ''), COALESCE(role_file, ''), created_at
		FROM roles ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		err := rows.Scan(&r.Name, &r.Description, &r.RoleFile, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

// UpdateRole updates a role
func (db *DB) UpdateRole(name, description, roleFile string) error {
	result, err := db.conn.Exec(`
		UPDATE roles SET description = ?, role_file = ? WHERE name = ?
	`, nullString(description), nullString(roleFile), name)
	if err != nil {
		return err
	}
	return checkRowsAffected(result, "role", name)
}

// DeleteRole deletes a role
func (db *DB) DeleteRole(name string) error {
	result, err := db.conn.Exec("DELETE FROM roles WHERE name = ?", name)
	if err != nil {
		return err
	}
	return checkRowsAffected(result, "role", name)
}

// ============================================
// Sprint Operations
// ============================================

// CreateSprint creates a new sprint
func (db *DB) CreateSprint(id, configFile, projectPath string) error {
	// Check for existing active sprint
	var existing string
	err := db.conn.QueryRow("SELECT id FROM sprints WHERE status = 'active' LIMIT 1").Scan(&existing)
	if err == nil {
		return fmt.Errorf("active sprint already exists: %s", existing)
	}

	_, err = db.conn.Exec(
		"INSERT INTO sprints (id, config_file, project_path) VALUES (?, ?, ?)",
		id, nullString(configFile), nullString(projectPath),
	)
	if err != nil {
		return err
	}

	return db.logEvent("sprint_created", "", map[string]interface{}{"sprint_id": id})
}

// GetActiveSprint returns the active sprint
func (db *DB) GetActiveSprint() (*Sprint, error) {
	row := db.conn.QueryRow(`
		SELECT id, COALESCE(config_file, ''), COALESCE(project_path, ''), status, started_at, completed_at
		FROM sprints WHERE status = 'active' ORDER BY started_at DESC LIMIT 1
	`)

	var s Sprint
	var completedAt sql.NullTime
	err := row.Scan(&s.ID, &s.ConfigFile, &s.ProjectPath, &s.Status, &s.StartedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		s.CompletedAt = &completedAt.Time
	}
	return &s, nil
}

// CompleteSprint completes the active sprint and all its workers
func (db *DB) CompleteSprint() (string, error) {
	sprint, err := db.GetActiveSprint()
	if err != nil || sprint == nil {
		return "", fmt.Errorf("no active sprint")
	}

	// Complete all workers
	_, err = db.conn.Exec(
		"UPDATE workers SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE sprint_id = ?",
		sprint.ID,
	)
	if err != nil {
		return "", err
	}

	// Complete sprint
	_, err = db.conn.Exec(
		"UPDATE sprints SET status = 'completed', completed_at = CURRENT_TIMESTAMP WHERE id = ?",
		sprint.ID,
	)
	if err != nil {
		return "", err
	}

	db.logEvent("sprint_completed", "", map[string]interface{}{"sprint_id": sprint.ID})
	return sprint.ID, nil
}

// ============================================
// Worker Operations
// ============================================

// workerSelectColumns defines the standard columns for worker queries
const workerSelectColumns = `
	w.name, w.sprint_id, w.branch, COALESCE(w.role_name, ''), COALESCE(r.role_file, ''),
	COALESCE(w.worktree_path, ''), w.status, COALESCE(w.current_task, ''),
	COALESCE(w.last_commit, ''), w.error_count, COALESCE(w.last_error, ''), w.updated_at,
	(SELECT COUNT(*) FROM messages m WHERE m.to_worker = w.name AND m.read_at IS NULL)`

// scanWorker scans a worker row into a Worker struct
func scanWorker(scanner interface{ Scan(...interface{}) error }) (Worker, error) {
	var w Worker
	err := scanner.Scan(&w.Name, &w.SprintID, &w.Branch, &w.RoleName, &w.RoleFile,
		&w.WorktreePath, &w.Status, &w.CurrentTask, &w.LastCommit, &w.ErrorCount,
		&w.LastError, &w.UpdatedAt, &w.UnreadMessages)
	return w, err
}

// RegisterWorker registers a worker
func (db *DB) RegisterWorker(name, sprintID, branch, roleName, worktreePath string) error {
	// Validate role exists if specified
	if roleName != "" {
		role, err := db.GetRole(roleName)
		if err != nil {
			return err
		}
		if role == nil {
			return fmt.Errorf("role not found: %s", roleName)
		}
	}

	_, err := db.conn.Exec(`
		INSERT INTO workers (name, sprint_id, branch, role_name, worktree_path)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			sprint_id = excluded.sprint_id,
			branch = excluded.branch,
			role_name = excluded.role_name,
			worktree_path = excluded.worktree_path,
			status = 'pending',
			updated_at = CURRENT_TIMESTAMP
	`, name, sprintID, branch, nullString(roleName), nullString(worktreePath))
	if err != nil {
		return err
	}

	return db.logEvent("worker_registered", name, map[string]interface{}{"branch": branch, "role": roleName})
}

// UpdateWorkerStatus updates worker status
func (db *DB) UpdateWorkerStatus(name, status string, currentTask, lastCommit *string) error {
	query := "UPDATE workers SET status = ?, updated_at = CURRENT_TIMESTAMP"
	args := []interface{}{status}

	if currentTask != nil {
		query += ", current_task = ?"
		args = append(args, *currentTask)
	}
	if lastCommit != nil {
		query += ", last_commit = ?"
		args = append(args, *lastCommit)
	}

	query += " WHERE name = ?"
	args = append(args, name)

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(result, "worker", name); err != nil {
		return err
	}
	return db.logEvent("worker_status_changed", name, map[string]interface{}{"status": status})
}

// UpdateWorkerTask updates the current task of a worker
func (db *DB) UpdateWorkerTask(name, task string) error {
	result, err := db.conn.Exec(
		"UPDATE workers SET current_task = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?",
		task, name,
	)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(result, "worker", name); err != nil {
		return err
	}
	return db.logEvent("worker_task_updated", name, map[string]interface{}{"task": task})
}

// ReportWorkerError reports an error and sets worker to error status
func (db *DB) ReportWorkerError(name, message string) error {
	result, err := db.conn.Exec(
		"UPDATE workers SET status = 'error', last_error = ?, error_count = error_count + 1, updated_at = CURRENT_TIMESTAMP WHERE name = ?",
		message, name,
	)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(result, "worker", name); err != nil {
		return err
	}
	return db.logEvent("worker_error", name, map[string]interface{}{"message": message})
}

// GetWorker returns a worker by name
func (db *DB) GetWorker(name string) (*Worker, error) {
	row := db.conn.QueryRow(`
		SELECT `+workerSelectColumns+`
		FROM workers w
		LEFT JOIN roles r ON w.role_name = r.name
		WHERE w.name = ?
	`, name)

	w, err := scanWorker(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetAllWorkers returns all workers for active sprint
func (db *DB) GetAllWorkers() ([]Worker, error) {
	rows, err := db.conn.Query(`
		SELECT `+workerSelectColumns+`
		FROM workers w
		LEFT JOIN roles r ON w.role_name = r.name
		WHERE w.sprint_id = (SELECT id FROM sprints WHERE status = 'active' ORDER BY started_at DESC LIMIT 1)
		ORDER BY w.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []Worker
	for rows.Next() {
		w, err := scanWorker(rows)
		if err != nil {
			return nil, err
		}
		workers = append(workers, w)
	}
	return workers, nil
}

// GetAllWorkerNames returns all worker names for active sprint
func (db *DB) GetAllWorkerNames() ([]string, error) {
	rows, err := db.conn.Query(`
		SELECT name FROM workers
		WHERE sprint_id = (SELECT id FROM sprints WHERE status = 'active' ORDER BY started_at DESC LIMIT 1)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

// ============================================
// Message Operations
// ============================================

// SendMessage sends a message to a specific worker
func (db *DB) SendMessage(from, to, msgType, subject, content string) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO messages (from_worker, to_worker, message_type, subject, content)
		VALUES (?, ?, ?, ?, ?)
	`, from, to, msgType, nullString(subject), content)
	if err != nil {
		return 0, err
	}

	db.logEvent("message_sent", from, map[string]interface{}{"to": to, "type": msgType})

	return result.LastInsertId()
}

// BroadcastMessage sends a message to all workers (expanded to individual messages)
func (db *DB) BroadcastMessage(from, msgType, subject, content string) (int, error) {
	workers, err := db.GetAllWorkerNames()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, worker := range workers {
		if worker == from {
			continue // Don't send to self
		}
		_, err := db.conn.Exec(`
			INSERT INTO messages (from_worker, to_worker, message_type, subject, content)
			VALUES (?, ?, ?, ?, ?)
		`, from, worker, msgType, nullString(subject), content)
		if err != nil {
			return count, err
		}
		count++
	}

	db.logEvent("message_broadcast", from, map[string]interface{}{"type": msgType, "count": count})

	return count, nil
}

// GetUnreadMessages returns unread messages for a worker
func (db *DB) GetUnreadMessages(worker string) ([]Message, error) {
	rows, err := db.conn.Query(`
		SELECT id, from_worker, to_worker, message_type, COALESCE(subject, ''),
		       content, read_at, created_at
		FROM messages
		WHERE to_worker = ? AND read_at IS NULL
		ORDER BY created_at ASC
	`, worker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var readAt sql.NullTime
		err := rows.Scan(&m.ID, &m.FromWorker, &m.ToWorker, &m.MessageType, &m.Subject,
			&m.Content, &readAt, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		if readAt.Valid {
			m.ReadAt = &readAt.Time
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// MarkMessageRead marks a message as read
func (db *DB) MarkMessageRead(id int) error {
	_, err := db.conn.Exec(
		"UPDATE messages SET read_at = CURRENT_TIMESTAMP WHERE id = ? AND read_at IS NULL",
		id,
	)
	return err
}

// MarkAllRead marks all messages for a worker as read
func (db *DB) MarkAllRead(worker string) (int64, error) {
	result, err := db.conn.Exec(`
		UPDATE messages SET read_at = CURRENT_TIMESTAMP
		WHERE to_worker = ? AND read_at IS NULL
	`, worker)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ============================================
// Event Operations
// ============================================

// eventSelectColumns defines the standard columns for event queries
const eventSelectColumns = "id, event_type, COALESCE(worker, ''), COALESCE(data, ''), created_at"

// scanEvents scans all rows into Event structs
func scanEvents(rows *sql.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.EventType, &e.Worker, &e.Data, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// GetRecentEvents returns recent events
func (db *DB) GetRecentEvents(limit int, eventType, worker *string) ([]Event, error) {
	query := "SELECT " + eventSelectColumns + " FROM events WHERE 1=1"
	args := []interface{}{}

	if eventType != nil {
		query += " AND event_type = ?"
		args = append(args, *eventType)
	}
	if worker != nil {
		query += " AND worker = ?"
		args = append(args, *worker)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

// GetEventsSince returns events since a given ID
func (db *DB) GetEventsSince(lastID int, eventType *string) ([]Event, error) {
	query := "SELECT " + eventSelectColumns + " FROM events WHERE id > ?"
	args := []interface{}{lastID}

	if eventType != nil {
		query += " AND event_type LIKE ?"
		args = append(args, *eventType+"%")
	}

	query += " ORDER BY created_at ASC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

// GetLastEventID returns the ID of the most recent event
func (db *DB) GetLastEventID() (int, error) {
	var id int
	err := db.conn.QueryRow("SELECT COALESCE(MAX(id), 0) FROM events").Scan(&id)
	return id, err
}
