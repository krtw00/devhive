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
	PaneID         *int
	Branch         string
	Issue          string
	WorktreePath   string
	Status         string
	CurrentTask    string
	LastCommit     string
	ErrorCount     int
	LastError      string
	UpdatedAt      time.Time
	PendingReviews int
	UnreadMessages int
}

// Review represents a review
type Review struct {
	ID          int
	Worker      string
	CommitHash  string
	Description string
	Status      string
	Reviewer    string
	Comment     string
	CreatedAt   time.Time
	ResolvedAt  *time.Time
	Branch      string
	Issue       string
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

// FileLock represents a file lock
type FileLock struct {
	FilePath string
	LockedBy string
	Reason   string
	LockedAt time.Time
}

// Event represents an event
type Event struct {
	ID        int
	EventType string
	Worker    string
	Data      string
	CreatedAt time.Time
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

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- Sprint Operations ---

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

// CompleteSprint completes the active sprint
func (db *DB) CompleteSprint() (string, error) {
	sprint, err := db.GetActiveSprint()
	if err != nil || sprint == nil {
		return "", err
	}

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

// --- Worker Operations ---

// RegisterWorker registers a worker
func (db *DB) RegisterWorker(name, sprintID, branch, issue string, paneID *int, worktreePath string) error {
	_, err := db.conn.Exec(`
		INSERT INTO workers (name, sprint_id, branch, issue, pane_id, worktree_path)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			sprint_id = excluded.sprint_id,
			branch = excluded.branch,
			issue = excluded.issue,
			pane_id = excluded.pane_id,
			worktree_path = excluded.worktree_path,
			status = 'pending',
			updated_at = CURRENT_TIMESTAMP
	`, name, sprintID, branch, nullString(issue), paneID, nullString(worktreePath))
	if err != nil {
		return err
	}

	return db.logEvent("worker_registered", name, map[string]interface{}{"branch": branch, "issue": issue})
}

// UpdateWorkerStatus updates worker status
func (db *DB) UpdateWorkerStatus(name, status string, currentTask, lastCommit, lastError *string) error {
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
	if lastError != nil {
		query += ", last_error = ?, error_count = error_count + 1"
		args = append(args, *lastError)
	}

	query += " WHERE name = ?"
	args = append(args, name)

	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("worker not found: %s", name)
	}

	return db.logEvent("worker_status_changed", name, map[string]interface{}{"status": status})
}

// GetWorker returns a worker by name
func (db *DB) GetWorker(name string) (*Worker, error) {
	row := db.conn.QueryRow(`
		SELECT name, sprint_id, pane_id, branch, COALESCE(issue, ''), COALESCE(worktree_path, ''),
		       status, COALESCE(current_task, ''), COALESCE(last_commit, ''), error_count,
		       COALESCE(last_error, ''), updated_at
		FROM workers WHERE name = ?
	`, name)

	var w Worker
	var paneID sql.NullInt64
	err := row.Scan(&w.Name, &w.SprintID, &paneID, &w.Branch, &w.Issue, &w.WorktreePath,
		&w.Status, &w.CurrentTask, &w.LastCommit, &w.ErrorCount, &w.LastError, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if paneID.Valid {
		p := int(paneID.Int64)
		w.PaneID = &p
	}
	return &w, nil
}

// GetAllWorkers returns all workers for active sprint
func (db *DB) GetAllWorkers() ([]Worker, error) {
	rows, err := db.conn.Query(`
		SELECT w.name, w.sprint_id, w.pane_id, w.branch, COALESCE(w.issue, ''),
		       COALESCE(w.worktree_path, ''), w.status, COALESCE(w.current_task, ''),
		       COALESCE(w.last_commit, ''), w.error_count, COALESCE(w.last_error, ''), w.updated_at,
		       (SELECT COUNT(*) FROM reviews r WHERE r.worker = w.name AND r.status = 'pending') as pending_reviews,
		       (SELECT COUNT(*) FROM messages m WHERE (m.to_worker = w.name OR m.to_worker IS NULL) AND m.read_at IS NULL) as unread_messages
		FROM workers w
		WHERE w.sprint_id = (SELECT id FROM sprints WHERE status = 'active' ORDER BY started_at DESC LIMIT 1)
		ORDER BY w.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []Worker
	for rows.Next() {
		var w Worker
		var paneID sql.NullInt64
		err := rows.Scan(&w.Name, &w.SprintID, &paneID, &w.Branch, &w.Issue, &w.WorktreePath,
			&w.Status, &w.CurrentTask, &w.LastCommit, &w.ErrorCount, &w.LastError, &w.UpdatedAt,
			&w.PendingReviews, &w.UnreadMessages)
		if err != nil {
			return nil, err
		}
		if paneID.Valid {
			p := int(paneID.Int64)
			w.PaneID = &p
		}
		workers = append(workers, w)
	}
	return workers, nil
}

// --- Review Operations ---

// RequestReview requests a review
func (db *DB) RequestReview(worker, commitHash, description string) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO reviews (worker, commit_hash, description)
		VALUES (?, ?, ?)
		ON CONFLICT(worker, commit_hash) DO UPDATE SET
			description = excluded.description,
			status = 'pending',
			created_at = CURRENT_TIMESTAMP
	`, worker, commitHash, nullString(description))
	if err != nil {
		return 0, err
	}

	// Update worker status
	db.conn.Exec(
		"UPDATE workers SET status = 'review_pending', last_commit = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?",
		commitHash, worker,
	)

	db.logEvent("review_requested", worker, map[string]interface{}{"commit": commitHash})

	return result.LastInsertId()
}

// ResolveReview resolves a review
func (db *DB) ResolveReview(reviewID int, status, reviewer, comment string) error {
	result, err := db.conn.Exec(`
		UPDATE reviews SET status = ?, reviewer = ?, comment = ?, resolved_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'pending'
	`, status, reviewer, nullString(comment), reviewID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("review not found or already resolved: %d", reviewID)
	}

	// Get worker and send notification
	var worker string
	db.conn.QueryRow("SELECT worker FROM reviews WHERE id = ?", reviewID).Scan(&worker)
	if worker != "" {
		msg := fmt.Sprintf("レビュー結果: %s", status)
		if comment != "" {
			msg += " - " + comment
		}
		db.SendMessage("system", &worker, "system", fmt.Sprintf("Review #%d", reviewID), msg)
		db.logEvent("review_resolved", worker, map[string]interface{}{"review_id": reviewID, "status": status})
	}

	return nil
}

// GetPendingReviews returns pending reviews
func (db *DB) GetPendingReviews() ([]Review, error) {
	rows, err := db.conn.Query(`
		SELECT r.id, r.worker, r.commit_hash, COALESCE(r.description, ''), r.status,
		       COALESCE(r.reviewer, ''), COALESCE(r.comment, ''), r.created_at, r.resolved_at,
		       w.branch, COALESCE(w.issue, '')
		FROM reviews r
		JOIN workers w ON r.worker = w.name
		WHERE r.status = 'pending'
		ORDER BY r.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []Review
	for rows.Next() {
		var r Review
		var resolvedAt sql.NullTime
		err := rows.Scan(&r.ID, &r.Worker, &r.CommitHash, &r.Description, &r.Status,
			&r.Reviewer, &r.Comment, &r.CreatedAt, &resolvedAt, &r.Branch, &r.Issue)
		if err != nil {
			return nil, err
		}
		if resolvedAt.Valid {
			r.ResolvedAt = &resolvedAt.Time
		}
		reviews = append(reviews, r)
	}
	return reviews, nil
}

// --- Message Operations ---

// SendMessage sends a message
func (db *DB) SendMessage(from string, to *string, msgType, subject, content string) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO messages (from_worker, to_worker, message_type, subject, content)
		VALUES (?, ?, ?, ?, ?)
	`, from, to, msgType, nullString(subject), content)
	if err != nil {
		return 0, err
	}

	toStr := "broadcast"
	if to != nil {
		toStr = *to
	}
	db.logEvent("message_sent", from, map[string]interface{}{"to": toStr, "type": msgType})

	return result.LastInsertId()
}

// GetUnreadMessages returns unread messages
func (db *DB) GetUnreadMessages(worker *string) ([]Message, error) {
	var rows *sql.Rows
	var err error

	if worker != nil {
		rows, err = db.conn.Query(`
			SELECT id, from_worker, COALESCE(to_worker, ''), message_type, COALESCE(subject, ''),
			       content, read_at, created_at
			FROM messages
			WHERE (to_worker = ? OR to_worker IS NULL) AND read_at IS NULL
			ORDER BY created_at ASC
		`, *worker)
	} else {
		rows, err = db.conn.Query(`
			SELECT id, from_worker, COALESCE(to_worker, ''), message_type, COALESCE(subject, ''),
			       content, read_at, created_at
			FROM messages WHERE read_at IS NULL ORDER BY created_at ASC
		`)
	}
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
		WHERE (to_worker = ? OR to_worker IS NULL) AND read_at IS NULL
	`, worker)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// --- File Lock Operations ---

// AcquireLock tries to acquire a file lock
func (db *DB) AcquireLock(filePath, worker, reason string) error {
	_, err := db.conn.Exec(
		"INSERT INTO file_locks (file_path, locked_by, reason) VALUES (?, ?, ?)",
		filePath, worker, nullString(reason),
	)
	if err != nil {
		// Check who has the lock
		var lockedBy string
		db.conn.QueryRow("SELECT locked_by FROM file_locks WHERE file_path = ?", filePath).Scan(&lockedBy)
		if lockedBy != "" {
			return fmt.Errorf("file already locked by: %s", lockedBy)
		}
		return err
	}

	db.logEvent("file_locked", worker, map[string]interface{}{"file": filePath})
	return nil
}

// ReleaseLock releases a file lock
func (db *DB) ReleaseLock(filePath, worker string) error {
	result, err := db.conn.Exec(
		"DELETE FROM file_locks WHERE file_path = ? AND locked_by = ?",
		filePath, worker,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("lock not found or not owned by you")
	}

	db.logEvent("file_unlocked", worker, map[string]interface{}{"file": filePath})
	return nil
}

// GetAllLocks returns all file locks
func (db *DB) GetAllLocks() ([]FileLock, error) {
	rows, err := db.conn.Query(`
		SELECT file_path, locked_by, COALESCE(reason, ''), locked_at
		FROM file_locks ORDER BY locked_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locks []FileLock
	for rows.Next() {
		var l FileLock
		err := rows.Scan(&l.FilePath, &l.LockedBy, &l.Reason, &l.LockedAt)
		if err != nil {
			return nil, err
		}
		locks = append(locks, l)
	}
	return locks, nil
}

// --- Event Operations ---

// GetRecentEvents returns recent events
func (db *DB) GetRecentEvents(limit int, eventType, worker *string) ([]Event, error) {
	query := "SELECT id, event_type, COALESCE(worker, ''), COALESCE(data, ''), created_at FROM events WHERE 1=1"
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

	var events []Event
	for rows.Next() {
		var e Event
		err := rows.Scan(&e.ID, &e.EventType, &e.Worker, &e.Data, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
