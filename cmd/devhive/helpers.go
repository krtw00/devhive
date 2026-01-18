package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// getWorkerName returns the worker name from args or environment variable
func getWorkerName(args []string, index int) (string, error) {
	if len(args) > index {
		return args[index], nil
	}
	if name := os.Getenv("DEVHIVE_WORKER"); name != "" {
		return name, nil
	}
	return "", fmt.Errorf("worker name required (set DEVHIVE_WORKER or provide as argument)")
}

// stringPtr returns a pointer to s if non-empty, otherwise nil
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// createGitWorktree creates a git worktree for the worker
// Returns the path to the created worktree
func createGitWorktree(workerName, branch, repoPath string) (string, error) {
	var err error

	// Determine repo path (project root)
	if repoPath == "" {
		repoPath, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	// Worktree path: <project>/.devhive/worktrees/<worker>
	worktreePath := filepath.Join(repoPath, ".devhive", "worktrees", workerName)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", err
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		// Already exists - return the path (useful for re-running up)
		return worktreePath, nil
	}

	// Check if branch exists locally
	checkCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", branch)
	branchExists := checkCmd.Run() == nil

	var cmd *exec.Cmd
	if branchExists {
		// Branch exists, create worktree
		cmd = exec.Command("git", "-C", repoPath, "worktree", "add", worktreePath, branch)
	} else {
		// Branch doesn't exist, create new branch
		cmd = exec.Command("git", "-C", repoPath, "worktree", "add", "-b", branch, worktreePath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add failed: %s\n%s", err, string(output))
	}

	return worktreePath, nil
}

// statusIcon returns an emoji icon for worker status
func statusIcon(status string) string {
	switch status {
	case "pending":
		return "⏳ pending"
	case "working":
		return "🔨 working"
	case "completed":
		return "✅ done"
	case "blocked":
		return "🚫 blocked"
	case "error":
		return "❌ error"
	default:
		return status
	}
}

// sessionIcon returns an icon for session state
func sessionIcon(state string) string {
	switch state {
	case "running":
		return "▶"
	case "waiting_permission":
		return "⏸"
	case "idle":
		return "○"
	case "stopped":
		return "■"
	default:
		return "?"
	}
}

// parseEventData parses JSON event data and returns the value for a key
func parseEventData(data, key string) string {
	var parsed map[string]interface{}
	json.Unmarshal([]byte(data), &parsed)
	if val, ok := parsed[key].(string); ok {
		return val
	}
	return ""
}

// createWorkerEnvrc creates a .envrc file in the worktree directory for direnv
func createWorkerEnvrc(worktreePath, workerName string) error {
	envrcPath := filepath.Join(worktreePath, ".envrc")
	content := fmt.Sprintf("export DEVHIVE_WORKER=%s\n", workerName)
	return os.WriteFile(envrcPath, []byte(content), 0644)
}

// runDirenvAllow runs 'direnv allow' in the specified directory
func runDirenvAllow(worktreePath string) error {
	// Run direnv allow with the .envrc path
	envrcPath := filepath.Join(worktreePath, ".envrc")
	cmd := exec.Command("direnv", "allow", envrcPath)
	cmd.Dir = worktreePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}
