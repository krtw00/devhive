package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestDB creates a temporary database for testing
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "devhive-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestSprintOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test CreateSprint
	err := db.CreateSprint("sprint-01", "", "")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Test GetActiveSprint
	sprint, err := db.GetActiveSprint()
	if err != nil {
		t.Fatalf("GetActiveSprint failed: %v", err)
	}
	if sprint == nil {
		t.Fatal("Expected active sprint, got nil")
	}
	if sprint.ID != "sprint-01" {
		t.Errorf("Expected sprint ID 'sprint-01', got '%s'", sprint.ID)
	}
	if sprint.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", sprint.Status)
	}

	// Test duplicate sprint fails
	err = db.CreateSprint("sprint-02", "", "")
	if err == nil {
		t.Error("Expected error creating sprint while one is active")
	}

	// Test CompleteSprint
	sprintID, err := db.CompleteSprint()
	if err != nil {
		t.Fatalf("CompleteSprint failed: %v", err)
	}
	if sprintID != "sprint-01" {
		t.Errorf("Expected sprint ID 'sprint-01', got '%s'", sprintID)
	}

	// Verify sprint is completed
	sprint, err = db.GetActiveSprint()
	if err != nil {
		t.Fatalf("GetActiveSprint failed: %v", err)
	}
	if sprint != nil {
		t.Error("Expected no active sprint after completion")
	}
}

func TestRoleOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test CreateRole (use custom role name to avoid conflict with builtin roles)
	err := db.CreateRole("custom-frontend", "Custom Frontend developer", "roles/custom-frontend.md", "--model sonnet")
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Test GetRole
	role, err := db.GetRole("custom-frontend")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if role == nil {
		t.Fatal("Expected role, got nil")
	}
	if role.Name != "custom-frontend" {
		t.Errorf("Expected role name 'custom-frontend', got '%s'", role.Name)
	}
	if role.Description != "Custom Frontend developer" {
		t.Errorf("Expected description 'Custom Frontend developer', got '%s'", role.Description)
	}
	if role.Args != "--model sonnet" {
		t.Errorf("Expected args '--model sonnet', got '%s'", role.Args)
	}

	// Test GetAllRoles (only user-defined roles, no builtin roles in DB)
	roles, err := db.GetAllRoles()
	if err != nil {
		t.Fatalf("GetAllRoles failed: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(roles))
	}

	// Test UpdateRole
	err = db.UpdateRole("custom-frontend", "Senior Custom Frontend developer", "roles/custom-frontend-v2.md", "--model opus")
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}

	role, _ = db.GetRole("custom-frontend")
	if role.Description != "Senior Custom Frontend developer" {
		t.Errorf("Expected updated description, got '%s'", role.Description)
	}
	if role.Args != "--model opus" {
		t.Errorf("Expected updated args, got '%s'", role.Args)
	}

	// Test DeleteRole
	err = db.DeleteRole("custom-frontend")
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	role, _ = db.GetRole("custom-frontend")
	if role != nil {
		t.Error("Expected role to be deleted")
	}
}

func TestWorkerOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create sprint first
	err := db.CreateSprint("sprint-01", "", "")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Create role
	err = db.CreateRole("frontend", "Frontend", "", "")
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Test RegisterWorker
	err = db.RegisterWorker("fe", "sprint-01", "feat/ui", "frontend", "/path/to/worktree", "claude")
	if err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	// Test GetWorker
	worker, err := db.GetWorker("fe")
	if err != nil {
		t.Fatalf("GetWorker failed: %v", err)
	}
	if worker == nil {
		t.Fatal("Expected worker, got nil")
	}
	if worker.Name != "fe" {
		t.Errorf("Expected worker name 'fe', got '%s'", worker.Name)
	}
	if worker.Branch != "feat/ui" {
		t.Errorf("Expected branch 'feat/ui', got '%s'", worker.Branch)
	}
	if worker.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", worker.Status)
	}
	if worker.SessionState != "stopped" {
		t.Errorf("Expected session state 'stopped', got '%s'", worker.SessionState)
	}

	// Test UpdateWorkerStatus
	task := "Building UI"
	err = db.UpdateWorkerStatus("fe", "working", &task, nil)
	if err != nil {
		t.Fatalf("UpdateWorkerStatus failed: %v", err)
	}

	worker, _ = db.GetWorker("fe")
	if worker.Status != "working" {
		t.Errorf("Expected status 'working', got '%s'", worker.Status)
	}
	if worker.CurrentTask != "Building UI" {
		t.Errorf("Expected task 'Building UI', got '%s'", worker.CurrentTask)
	}

	// Test UpdateWorkerSessionState
	err = db.UpdateWorkerSessionState("fe", "running")
	if err != nil {
		t.Fatalf("UpdateWorkerSessionState failed: %v", err)
	}

	worker, _ = db.GetWorker("fe")
	if worker.SessionState != "running" {
		t.Errorf("Expected session state 'running', got '%s'", worker.SessionState)
	}

	// Test UpdateWorkerTask
	err = db.UpdateWorkerTask("fe", "Implementing buttons")
	if err != nil {
		t.Fatalf("UpdateWorkerTask failed: %v", err)
	}

	worker, _ = db.GetWorker("fe")
	if worker.CurrentTask != "Implementing buttons" {
		t.Errorf("Expected task 'Implementing buttons', got '%s'", worker.CurrentTask)
	}

	// Test ReportWorkerError
	err = db.ReportWorkerError("fe", "Build failed")
	if err != nil {
		t.Fatalf("ReportWorkerError failed: %v", err)
	}

	worker, _ = db.GetWorker("fe")
	if worker.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", worker.Status)
	}
	if worker.ErrorCount != 1 {
		t.Errorf("Expected error count 1, got %d", worker.ErrorCount)
	}
	if worker.LastError != "Build failed" {
		t.Errorf("Expected last error 'Build failed', got '%s'", worker.LastError)
	}

	// Test GetAllWorkers
	workers, err := db.GetAllWorkers()
	if err != nil {
		t.Fatalf("GetAllWorkers failed: %v", err)
	}
	if len(workers) != 1 {
		t.Errorf("Expected 1 worker, got %d", len(workers))
	}
}

func TestMessageOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup: create sprint and workers
	db.CreateSprint("sprint-01", "", "")
	db.RegisterWorker("fe", "sprint-01", "feat/ui", "", "", "")
	db.RegisterWorker("be", "sprint-01", "feat/api", "", "", "")

	// Test SendMessage
	msgID, err := db.SendMessage("fe", "be", "info", "API Update", "Please update the API")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msgID == 0 {
		t.Error("Expected non-zero message ID")
	}

	// Test GetUnreadMessages
	messages, err := db.GetUnreadMessages("be")
	if err != nil {
		t.Fatalf("GetUnreadMessages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("Expected 1 unread message, got %d", len(messages))
	}
	if messages[0].FromWorker != "fe" {
		t.Errorf("Expected from 'fe', got '%s'", messages[0].FromWorker)
	}
	if messages[0].Subject != "API Update" {
		t.Errorf("Expected subject 'API Update', got '%s'", messages[0].Subject)
	}

	// Test MarkMessageRead
	err = db.MarkMessageRead(int(msgID))
	if err != nil {
		t.Fatalf("MarkMessageRead failed: %v", err)
	}

	messages, _ = db.GetUnreadMessages("be")
	if len(messages) != 0 {
		t.Errorf("Expected 0 unread messages, got %d", len(messages))
	}

	// Test BroadcastMessage
	count, err := db.BroadcastMessage("pm", "info", "Meeting", "Team meeting at 3pm")
	if err != nil {
		t.Fatalf("BroadcastMessage failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected broadcast to 2 workers, got %d", count)
	}

	// Test MarkAllRead
	readCount, err := db.MarkAllRead("fe")
	if err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}
	if readCount != 1 {
		t.Errorf("Expected 1 message marked read, got %d", readCount)
	}
}

func TestEventOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Events are created by other operations
	db.CreateSprint("sprint-01", "", "")
	db.RegisterWorker("fe", "sprint-01", "feat/ui", "", "", "")
	db.UpdateWorkerStatus("fe", "working", nil, nil)

	// Test GetRecentEvents
	events, err := db.GetRecentEvents(10, nil, nil)
	if err != nil {
		t.Fatalf("GetRecentEvents failed: %v", err)
	}
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}

	// Test GetRecentEvents with filter
	workerFilter := "fe"
	events, err = db.GetRecentEvents(10, nil, &workerFilter)
	if err != nil {
		t.Fatalf("GetRecentEvents with filter failed: %v", err)
	}
	for _, e := range events {
		if e.Worker != "fe" && e.Worker != "" {
			t.Errorf("Expected worker 'fe', got '%s'", e.Worker)
		}
	}

	// Test GetLastEventID
	lastID, err := db.GetLastEventID()
	if err != nil {
		t.Fatalf("GetLastEventID failed: %v", err)
	}
	if lastID == 0 {
		t.Error("Expected non-zero last event ID")
	}

	// Test GetEventsSince
	events, err = db.GetEventsSince(0, nil)
	if err != nil {
		t.Fatalf("GetEventsSince failed: %v", err)
	}
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}
}

func TestCleanupOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create some events
	db.CreateSprint("sprint-01", "", "")
	db.RegisterWorker("fe", "sprint-01", "feat/ui", "", "", "")

	// Test CleanupOldEvents (dry run)
	count, err := db.CleanupOldEvents(30, true)
	if err != nil {
		t.Fatalf("CleanupOldEvents (dry run) failed: %v", err)
	}
	// Events are recent, so count should be 0
	if count != 0 {
		t.Logf("Note: Found %d events older than 30 days", count)
	}

	// Test CleanupOldMessages (dry run)
	count, err = db.CleanupOldMessages(30, true)
	if err != nil {
		t.Fatalf("CleanupOldMessages (dry run) failed: %v", err)
	}
	if count != 0 {
		t.Logf("Note: Found %d messages older than 30 days", count)
	}
}

func TestMigration(t *testing.T) {
	// Test that database migration works correctly
	tmpDir, err := os.MkdirTemp("", "devhive-migration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// First open creates the schema
	db1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db1.Close()

	// Second open should run migrations without error
	db2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Verify the database works
	err = db2.CreateSprint("sprint-01", "", "")
	if err != nil {
		t.Fatalf("Failed to create sprint after migration: %v", err)
	}
}

func TestProjectDetection(t *testing.T) {
	// Test DetectProject with explicit project name
	ProjectName = "test-project"
	project := DetectProject()
	if project != "test-project" {
		t.Errorf("Expected 'test-project', got '%s'", project)
	}

	// Reset
	ProjectName = ""

	// Test fallback to directory name
	project = DetectProject()
	if project == "" {
		t.Error("Expected non-empty project name from directory fallback")
	}
}

func TestWorkerWithRole(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create sprint and role
	db.CreateSprint("sprint-01", "", "")
	db.CreateRole("frontend", "Frontend dev", "roles/frontend.md", "--model sonnet")

	// Register worker with role
	err := db.RegisterWorker("fe", "sprint-01", "feat/ui", "frontend", "", "")
	if err != nil {
		t.Fatalf("RegisterWorker failed: %v", err)
	}

	// Verify role file is joined
	worker, err := db.GetWorker("fe")
	if err != nil {
		t.Fatalf("GetWorker failed: %v", err)
	}
	if worker.RoleName != "frontend" {
		t.Errorf("Expected role name 'frontend', got '%s'", worker.RoleName)
	}
	if worker.RoleFile != "roles/frontend.md" {
		t.Errorf("Expected role file 'roles/frontend.md', got '%s'", worker.RoleFile)
	}
}

func TestFreeFormRole(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.CreateSprint("sprint-01", "", "")

	// Any role name is allowed (no validation)
	err := db.RegisterWorker("fe", "sprint-01", "feat/ui", "any-role-name", "", "")
	if err != nil {
		t.Errorf("Expected no error for free-form role, got: %v", err)
	}

	worker, _ := db.GetWorker("fe")
	if worker.RoleName != "any-role-name" {
		t.Errorf("Expected role 'any-role-name', got '%s'", worker.RoleName)
	}
}

func TestWorkerUnreadMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.CreateSprint("sprint-01", "", "")
	db.RegisterWorker("fe", "sprint-01", "feat/ui", "", "", "")
	db.RegisterWorker("be", "sprint-01", "feat/api", "", "", "")

	// Send messages to fe
	db.SendMessage("be", "fe", "info", "", "Message 1")
	db.SendMessage("be", "fe", "info", "", "Message 2")
	db.SendMessage("be", "fe", "info", "", "Message 3")

	// Check unread count in worker info
	worker, _ := db.GetWorker("fe")
	if worker.UnreadMessages != 3 {
		t.Errorf("Expected 3 unread messages, got %d", worker.UnreadMessages)
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Test WAL mode allows concurrent reads
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.CreateSprint("sprint-01", "", "")

	// Simulate concurrent reads (basic test)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := db.GetActiveSprint()
			if err != nil {
				t.Errorf("Concurrent read failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}
}
