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
	err := db.CreateSprint("sprint-01")
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
	err = db.CreateSprint("sprint-02")
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

func TestWorkerOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create sprint first
	err := db.CreateSprint("sprint-01")
	if err != nil {
		t.Fatalf("CreateSprint failed: %v", err)
	}

	// Test RegisterWorker (simplified: only name and sprint_id)
	err = db.RegisterWorker("fe", "sprint-01")
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
	if worker.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", worker.Status)
	}
	if worker.SessionState != "stopped" {
		t.Errorf("Expected session state 'stopped', got '%s'", worker.SessionState)
	}

	// Test UpdateWorkerStatus
	err = db.UpdateWorkerStatus("fe", "working", nil)
	if err != nil {
		t.Fatalf("UpdateWorkerStatus failed: %v", err)
	}

	worker, _ = db.GetWorker("fe")
	if worker.Status != "working" {
		t.Errorf("Expected status 'working', got '%s'", worker.Status)
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

	// Test UpdateWorkerProgress
	err = db.UpdateWorkerProgress("fe", 50, "Building UI")
	if err != nil {
		t.Fatalf("UpdateWorkerProgress failed: %v", err)
	}

	worker, _ = db.GetWorker("fe")
	if worker.Progress != 50 {
		t.Errorf("Expected progress 50, got %d", worker.Progress)
	}
	if worker.Activity != "Building UI" {
		t.Errorf("Expected activity 'Building UI', got '%s'", worker.Activity)
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
	db.CreateSprint("sprint-01")
	db.RegisterWorker("fe", "sprint-01")
	db.RegisterWorker("be", "sprint-01")

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
	db.CreateSprint("sprint-01")
	db.RegisterWorker("fe", "sprint-01")
	db.UpdateWorkerStatus("fe", "working", nil)

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
	db.CreateSprint("sprint-01")
	db.RegisterWorker("fe", "sprint-01")

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
	err = db2.CreateSprint("sprint-01")
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

func TestWorkerUnreadMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	db.CreateSprint("sprint-01")
	db.RegisterWorker("fe", "sprint-01")
	db.RegisterWorker("be", "sprint-01")

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

	db.CreateSprint("sprint-01")

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
