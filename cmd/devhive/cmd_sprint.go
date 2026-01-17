package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/iguchi/devhive/internal/db"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	var configFile, projectPath string

	cmd := &cobra.Command{
		Use:   "init <sprint-id>",
		Short: "Initialize a new sprint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID := args[0]
			err := database.CreateSprint(sprintID, configFile, projectPath)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Sprint '%s' initialized\n", sprintID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
	cmd.Flags().StringVarP(&projectPath, "project", "p", "", "Project path")
	return cmd
}

func sprintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Sprint management",
	}

	completeCmd := &cobra.Command{
		Use:   "complete",
		Short: "Complete the active sprint",
		RunE: func(cmd *cobra.Command, args []string) error {
			sprintID, err := database.CompleteSprint()
			if err != nil {
				return err
			}
			fmt.Printf("✓ Sprint '%s' completed\n", sprintID)
			return nil
		},
	}

	// sprint setup - batch register workers from config
	setupCmd := &cobra.Command{
		Use:   "setup <config-file>",
		Short: "Setup workers from config file",
		Long: `Setup workers from a YAML/JSON config file.

Config file format (YAML):
  workers:
    - name: frontend
      branch: feat/ui
      role: frontend
    - name: backend
      branch: feat/api
      role: backend

Config file format (JSON):
  {"workers": [{"name": "frontend", "branch": "feat/ui", "role": "frontend"}]}`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := args[0]
			createWorktrees, _ := cmd.Flags().GetBool("create-worktrees")
			repoPath, _ := cmd.Flags().GetString("repo")

			sprint, err := database.GetActiveSprint()
			if err != nil || sprint == nil {
				return fmt.Errorf("no active sprint")
			}

			// Read config file
			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			// Parse config (support both JSON and simple format)
			workers, err := parseWorkersConfig(data)
			if err != nil {
				return err
			}

			// Register workers
			for _, w := range workers {
				worktreePath := w.Worktree

				// Create worktree if requested
				if createWorktrees && worktreePath == "" {
					wt, err := createGitWorktree(w.Name, w.Branch, repoPath)
					if err != nil {
						fmt.Printf("⚠ Failed to create worktree for %s: %v\n", w.Name, err)
					} else {
						worktreePath = wt
						fmt.Printf("✓ Worktree created: %s\n", wt)
					}
				}

				err := database.RegisterWorker(w.Name, sprint.ID, w.Branch, w.Role, worktreePath)
				if err != nil {
					fmt.Printf("⚠ Failed to register %s: %v\n", w.Name, err)
					continue
				}
				fmt.Printf("✓ Worker '%s' registered (branch: %s, role: %s)\n", w.Name, w.Branch, w.Role)
			}

			return nil
		},
	}
	setupCmd.Flags().BoolP("create-worktrees", "c", false, "Create git worktrees for each worker")
	setupCmd.Flags().String("repo", "", "Git repository path (default: cwd)")

	// sprint report - generate sprint report
	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Generate sprint report",
		RunE: func(cmd *cobra.Command, args []string) error {
			sprint, err := database.GetActiveSprint()
			if err != nil {
				return err
			}
			if sprint == nil {
				return fmt.Errorf("no active sprint")
			}

			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			events, err := database.GetRecentEvents(100, nil, nil)
			if err != nil {
				return err
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				report := map[string]interface{}{
					"sprint":  sprint,
					"workers": workers,
					"events":  events,
					"summary": generateSprintSummary(workers),
				}
				b, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			// Text report
			fmt.Println("═══════════════════════════════════════════════════════════")
			fmt.Printf("  Sprint Report: %s\n", sprint.ID)
			fmt.Println("═══════════════════════════════════════════════════════════")
			fmt.Printf("Started: %s\n", sprint.StartedAt.Format("2006-01-02 15:04:05"))
			if sprint.CompletedAt != nil {
				fmt.Printf("Completed: %s\n", sprint.CompletedAt.Format("2006-01-02 15:04:05"))
			}
			fmt.Printf("Status: %s\n", sprint.Status)
			fmt.Println()

			// Worker summary
			fmt.Println("Workers:")
			fmt.Println("───────────────────────────────────────────────────────────")
			summary := generateSprintSummary(workers)
			fmt.Printf("  Total: %d  |  Completed: %d  |  Working: %d  |  Pending: %d  |  Error: %d\n",
				summary["total"], summary["completed"], summary["working"], summary["pending"], summary["error"])
			fmt.Println()

			for _, w := range workers {
				statusStr := statusIcon(w.Status)
				sessionStr := fmt.Sprintf("%s %s", sessionIcon(w.SessionState), w.SessionState)
				fmt.Printf("  %s (%s)\n", w.Name, w.RoleName)
				fmt.Printf("    Branch: %s\n", w.Branch)
				fmt.Printf("    Status: %s | Session: %s\n", statusStr, sessionStr)
				if w.CurrentTask != "" {
					fmt.Printf("    Task: %s\n", w.CurrentTask)
				}
				if w.ErrorCount > 0 {
					fmt.Printf("    Errors: %d (Last: %s)\n", w.ErrorCount, w.LastError)
				}
				fmt.Println()
			}

			// Recent events summary
			fmt.Println("Recent Activity:")
			fmt.Println("───────────────────────────────────────────────────────────")
			eventCounts := make(map[string]int)
			for _, e := range events {
				eventCounts[e.EventType]++
			}
			for eventType, count := range eventCounts {
				fmt.Printf("  %s: %d\n", eventType, count)
			}

			return nil
		},
	}
	reportCmd.Flags().Bool("json", false, "Output as JSON")

	cmd.AddCommand(completeCmd, setupCmd, reportCmd)
	return cmd
}

// WorkerConfig represents a worker configuration from setup file
type WorkerConfig struct {
	Name     string `json:"name"`
	Branch   string `json:"branch"`
	Role     string `json:"role"`
	Worktree string `json:"worktree"`
}

// parseWorkersConfig parses worker configuration from JSON or simple format
func parseWorkersConfig(data []byte) ([]WorkerConfig, error) {
	// Try JSON format first
	var jsonConfig struct {
		Workers []WorkerConfig `json:"workers"`
	}
	if err := json.Unmarshal(data, &jsonConfig); err == nil && len(jsonConfig.Workers) > 0 {
		return jsonConfig.Workers, nil
	}

	// Try simple line format: name branch [role]
	var workers []WorkerConfig
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		w := WorkerConfig{
			Name:   parts[0],
			Branch: parts[1],
		}
		if len(parts) >= 3 {
			w.Role = parts[2]
		}
		workers = append(workers, w)
	}

	if len(workers) == 0 {
		return nil, fmt.Errorf("no workers found in config file")
	}

	return workers, nil
}

// generateSprintSummary generates summary statistics for a sprint
func generateSprintSummary(workers []db.Worker) map[string]int {
	summary := map[string]int{
		"total":     len(workers),
		"pending":   0,
		"working":   0,
		"completed": 0,
		"blocked":   0,
		"error":     0,
	}

	for _, w := range workers {
		switch w.Status {
		case "pending":
			summary["pending"]++
		case "working":
			summary["working"]++
		case "completed":
			summary["completed"]++
		case "blocked":
			summary["blocked"]++
		case "error":
			summary["error"]++
		}
	}

	return summary
}
