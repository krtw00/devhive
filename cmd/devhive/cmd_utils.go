package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// mergeCmd merges a worker's branch into a target branch
func mergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge <worker> <target-branch>",
		Short: "Merge worker branch into target branch",
		Long: `Merge a worker's branch into a specified target branch.

Examples:
  devhive merge frontend develop    # Merge frontend branch into develop
  devhive merge backend staging     # Merge backend branch into staging
  devhive merge fe-auth main        # Merge fe-auth into main (with caution)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workerName := args[0]
			targetBranch := args[1]

			// Verify worker exists in DB
			worker, err := database.GetWorker(workerName)
			if err != nil || worker == nil {
				return fmt.Errorf("worker not found: %s", workerName)
			}

			// Load config to get branch
			configFile, err := FindComposeFile()
			if err != nil {
				return err
			}
			config, err := LoadComposeFile(configFile)
			if err != nil {
				return err
			}

			workerConfig, ok := config.Workers[workerName]
			if !ok {
				return fmt.Errorf("worker %s not found in config", workerName)
			}
			if workerConfig.Branch == "" {
				return fmt.Errorf("worker %s has no branch configured", workerName)
			}
			branch := workerConfig.Branch

			// Warning for main/master
			if targetBranch == "main" || targetBranch == "master" {
				fmt.Printf("⚠️  Warning: Merging into %s branch\n", targetBranch)
				fmt.Print("Continue? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Get project root
			cwd, _ := os.Getwd()

			// Checkout target branch
			fmt.Printf("Checking out %s...\n", targetBranch)
			if err := runGit(cwd, "checkout", targetBranch); err != nil {
				return fmt.Errorf("failed to checkout %s: %w", targetBranch, err)
			}

			// Pull latest
			fmt.Println("Pulling latest changes...")
			runGit(cwd, "pull") // Ignore error if no remote

			// Merge worker branch
			fmt.Printf("Merging %s into %s...\n", branch, targetBranch)
			noFF, _ := cmd.Flags().GetBool("no-ff")
			mergeArgs := []string{"merge", branch}
			if noFF {
				mergeArgs = append(mergeArgs, "--no-ff")
			}
			if err := runGit(cwd, mergeArgs...); err != nil {
				return fmt.Errorf("merge failed: %w\nResolve conflicts and commit manually", err)
			}

			fmt.Printf("✅ Successfully merged %s into %s\n", branch, targetBranch)

			// Log event
			database.LogEvent("branch_merged", workerName, fmt.Sprintf(`{"from":"%s","to":"%s"}`, branch, targetBranch))

			return nil
		},
	}

	cmd.Flags().Bool("no-ff", false, "Create merge commit even for fast-forward")

	return cmd
}

// progressCmd updates worker progress
func progressCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "progress <worker> <0-100>",
		Short: "Update worker progress",
		Long: `Update the progress percentage of a worker (0-100).

If defaults.auto_complete is true in .devhive.yaml, the worker will be
automatically marked as completed when progress reaches 100%.

Examples:
  devhive progress frontend 50    # Set frontend to 50%
  devhive progress backend 100    # Set backend to 100% (auto-complete if configured)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workerName := args[0]
			progress, err := strconv.Atoi(args[1])
			if err != nil || progress < 0 || progress > 100 {
				return fmt.Errorf("progress must be a number between 0 and 100")
			}

			if err := database.UpdateWorkerProgress(workerName, progress, ""); err != nil {
				return err
			}

			fmt.Printf("✅ %s progress: %d%%\n", workerName, progress)

			// Auto-complete if progress is 100% and auto_complete is enabled
			if progress == 100 {
				configFile, _ := FindComposeFile()
				if configFile != "" {
					config, err := LoadComposeFile(configFile)
					if err == nil && config.Defaults.AutoComplete {
						if err := database.UpdateWorkerStatus(workerName, "completed", nil); err != nil {
							fmt.Printf("⚠ Failed to auto-complete: %v\n", err)
						} else {
							fmt.Printf("✅ %s auto-completed\n", workerName)
						}
					}
				}
			}

			return nil
		},
	}
}

// cleanCmd removes completed workers and their worktrees
func cleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove completed workers and worktrees",
		Long: `Remove all completed workers from database and optionally delete their worktrees.

Examples:
  devhive clean           # Remove completed workers (keep worktrees)
  devhive clean --all     # Also delete worktrees and branches
  devhive clean --logs    # Also clear event logs
  devhive clean --all --logs  # Full cleanup for new sprint`,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			logs, _ := cmd.Flags().GetBool("logs")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			// Get completed workers
			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			var completed []string
			for _, w := range workers {
				if w.Status == "completed" {
					completed = append(completed, w.Name)
				}
			}

			if len(completed) == 0 {
				fmt.Println("No completed workers to clean.")
				return nil
			}

			fmt.Printf("Found %d completed worker(s):\n", len(completed))
			for _, name := range completed {
				fmt.Printf("  - %s\n", name)
			}

			if dryRun {
				fmt.Println("\n(dry-run mode, no changes made)")
				return nil
			}

			cwd, _ := os.Getwd()

			// Load config to get branch info if needed
			var config *ComposeConfig
			if all {
				configFile, _ := FindComposeFile()
				if configFile != "" {
					config, _ = LoadComposeFile(configFile)
				}
			}

			for _, name := range completed {
				if all {
					// Derive worktree path from convention
					worktreePath := filepath.Join(".devhive", "worktrees", name)
					if _, err := os.Stat(worktreePath); err == nil {
						fmt.Printf("Removing worktree: %s\n", worktreePath)
						runGit(cwd, "worktree", "remove", worktreePath, "--force")
					}

					// Delete branch (optional, only if merged)
					if config != nil {
						if workerConfig, ok := config.Workers[name]; ok && workerConfig.Branch != "" {
							fmt.Printf("Deleting branch: %s\n", workerConfig.Branch)
							runGit(cwd, "branch", "-d", workerConfig.Branch) // -d fails if not merged
						}
					}
				}

				// Remove from database
				if err := database.DeleteWorker(name); err != nil {
					fmt.Printf("Warning: failed to delete %s: %v\n", name, err)
				} else {
					fmt.Printf("✅ Removed %s\n", name)
				}
			}

			// Clear logs if requested
			if logs {
				count, err := database.ClearAllEvents()
				if err != nil {
					fmt.Printf("⚠ Failed to clear logs: %v\n", err)
				} else {
					fmt.Printf("✅ Cleared %d log events\n", count)
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("all", false, "Also delete worktrees and branches")
	cmd.Flags().Bool("logs", false, "Also clear event logs")
	cmd.Flags().Bool("dry-run", false, "Show what would be deleted without doing it")

	return cmd
}

// cleanupWorkers removes worktrees and branches for specified workers
func cleanupWorkers(workerNames []string) {
	cwd, _ := os.Getwd()

	// Load config to get branch info
	var config *ComposeConfig
	configFile, _ := FindComposeFile()
	if configFile != "" {
		config, _ = LoadComposeFile(configFile)
	}

	for _, name := range workerNames {
		// Derive worktree path from convention
		worktreePath := filepath.Join(".devhive", "worktrees", name)
		if _, err := os.Stat(worktreePath); err == nil {
			fmt.Printf("  Removing worktree: %s\n", worktreePath)
			runGit(cwd, "worktree", "remove", worktreePath, "--force")
		}

		// Delete branch (only if merged)
		if config != nil {
			if workerConfig, ok := config.Workers[name]; ok && workerConfig.Branch != "" {
				fmt.Printf("  Deleting branch: %s\n", workerConfig.Branch)
				runGit(cwd, "branch", "-d", workerConfig.Branch) // -d fails if not merged
			}
		}

		// Remove from database
		if err := database.DeleteWorker(name); err != nil {
			fmt.Printf("  ⚠ Failed to remove %s from DB: %v\n", name, err)
		} else {
			fmt.Printf("  ✓ Cleaned %s\n", name)
		}
	}
}

// noteCmd adds a note to worker's markdown file
func noteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "note <worker> <message>",
		Short: "Add a note to worker's log",
		Long: `Add a timestamped note to .devhive/workers/<worker>.md

Examples:
  devhive note frontend "Started auth implementation"
  devhive note backend "API endpoints complete, starting tests"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workerName := args[0]
			message := args[1]

			// Verify worker exists
			if _, err := database.GetWorker(workerName); err != nil {
				return fmt.Errorf("worker not found: %s", workerName)
			}

			cwd, _ := os.Getwd()
			notePath := filepath.Join(cwd, ".devhive", "workers", workerName+".md")

			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(notePath), 0755); err != nil {
				return err
			}

			// Create or append to file
			f, err := os.OpenFile(notePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()

			// Check if file is new
			info, _ := os.Stat(notePath)
			if info.Size() == 0 {
				// Write header
				fmt.Fprintf(f, "# %s Worker Notes\n\n", workerName)
			}

			// Add timestamped note
			timestamp := time.Now().Format("2006-01-02 15:04")
			fmt.Fprintf(f, "## %s\n\n%s\n\n", timestamp, message)

			fmt.Printf("✅ Note added to .devhive/workers/%s.md\n", workerName)
			return nil
		},
	}
}

// diffCmd shows diff for a worker's branch
func diffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [worker]",
		Short: "Show changes in worker branch",
		Long: `Show git diff for a worker's branch compared to base branch.

Examples:
  devhive diff frontend           # Show frontend changes vs main
  devhive diff backend --stat     # Show summary only
  devhive diff                    # Show all workers' changes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stat, _ := cmd.Flags().GetBool("stat")
			baseBranch, _ := cmd.Flags().GetString("base")

			cwd, _ := os.Getwd()

			// Load config to get branch info
			configFile, err := FindComposeFile()
			if err != nil {
				return err
			}
			config, err := LoadComposeFile(configFile)
			if err != nil {
				return err
			}

			if len(args) == 0 {
				// Show all workers from config
				for name, workerConfig := range config.Workers {
					if workerConfig.Branch == "" || workerConfig.Disabled {
						continue
					}
					fmt.Printf("\n=== %s (%s) ===\n", name, workerConfig.Branch)
					showDiff(cwd, baseBranch, workerConfig.Branch, stat)
				}
			} else {
				workerName := args[0]
				workerConfig, ok := config.Workers[workerName]
				if !ok {
					return fmt.Errorf("worker not found in config: %s", workerName)
				}
				if workerConfig.Branch == "" {
					return fmt.Errorf("worker %s has no branch configured", workerName)
				}
				showDiff(cwd, baseBranch, workerConfig.Branch, stat)
			}

			return nil
		},
	}

	cmd.Flags().Bool("stat", false, "Show diffstat only")
	cmd.Flags().String("base", "main", "Base branch to compare against")

	return cmd
}

// statusCmd shows overall project status
func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show overall project status",
		Long: `Show a summary of all workers, their progress, and status.

This provides a quick overview of:
  - Worker count by status
  - Overall progress
  - Recent activity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			if len(workers) == 0 {
				fmt.Println("No workers registered.")
				return nil
			}

			// Load config to get branch info
			var config *ComposeConfig
			configFile, _ := FindComposeFile()
			if configFile != "" {
				config, _ = LoadComposeFile(configFile)
			}

			// Count by status
			counts := map[string]int{
				"pending":   0,
				"working":   0,
				"completed": 0,
				"blocked":   0,
				"error":     0,
			}
			totalProgress := 0

			fmt.Println("=== DevHive Status ===")
			fmt.Println()

			// Worker details
			fmt.Println("Workers:")
			for _, w := range workers {
				counts[w.Status]++
				totalProgress += w.Progress

				icon := statusIcon(w.Status)
				bar := progressBar(w.Progress, 10)
				fmt.Printf("  %-12s %s %s %3d%%", w.Name, icon, bar, w.Progress)

				// Show branch from config if available
				if config != nil {
					if workerConfig, ok := config.Workers[w.Name]; ok && workerConfig.Branch != "" {
						fmt.Printf("  (%s)", workerConfig.Branch)
					}
				}
				fmt.Println()
			}

			// Summary
			fmt.Println()
			fmt.Println("Summary:")
			fmt.Printf("  Total: %d workers\n", len(workers))
			if counts["working"] > 0 {
				fmt.Printf("  🔨 Working: %d\n", counts["working"])
			}
			if counts["completed"] > 0 {
				fmt.Printf("  ✅ Completed: %d\n", counts["completed"])
			}
			if counts["pending"] > 0 {
				fmt.Printf("  ⏳ Pending: %d\n", counts["pending"])
			}
			if counts["blocked"] > 0 {
				fmt.Printf("  🚫 Blocked: %d\n", counts["blocked"])
			}
			if counts["error"] > 0 {
				fmt.Printf("  ❌ Error: %d\n", counts["error"])
			}

			// Average progress
			avgProgress := totalProgress / len(workers)
			fmt.Printf("\n  Overall Progress: %s %d%%\n", progressBar(avgProgress, 20), avgProgress)

			return nil
		},
	}
}

// Helper functions

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func showDiff(dir, base, branch string, stat bool) {
	args := []string{"diff", base + "..." + branch}
	if stat {
		args = append(args, "--stat")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func progressBar(progress, width int) string {
	filled := progress * width / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return "[" + bar + "]"
}
