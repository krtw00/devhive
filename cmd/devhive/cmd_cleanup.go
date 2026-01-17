package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/iguchi/devhive/internal/db"
	"github.com/spf13/cobra"
)

func cleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup old data and worktrees",
	}

	// cleanup events - remove old events
	eventsCmd := &cobra.Command{
		Use:   "events",
		Short: "Remove old events (keeps last N days)",
		RunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			count, err := database.CleanupOldEvents(days, dryRun)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Printf("Would delete %d events older than %d days\n", count, days)
			} else {
				fmt.Printf("✓ Deleted %d events older than %d days\n", count, days)
			}
			return nil
		},
	}
	eventsCmd.Flags().Int("days", 30, "Keep events from last N days")
	eventsCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")

	// cleanup messages - remove old read messages
	messagesCmd := &cobra.Command{
		Use:   "messages",
		Short: "Remove old read messages (keeps last N days)",
		RunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			count, err := database.CleanupOldMessages(days, dryRun)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Printf("Would delete %d read messages older than %d days\n", count, days)
			} else {
				fmt.Printf("✓ Deleted %d read messages older than %d days\n", count, days)
			}
			return nil
		},
	}
	messagesCmd.Flags().Int("days", 30, "Keep messages from last N days")
	messagesCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")

	// cleanup worktrees - remove worktrees for completed workers
	worktreesCmd := &cobra.Command{
		Use:   "worktrees",
		Short: "Remove worktrees for completed/removed workers",
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			force, _ := cmd.Flags().GetBool("force")

			// Get project worktrees directory
			home, _ := os.UserHomeDir()
			project := db.GetProjectName()
			var worktreesDir string
			if project != "" {
				worktreesDir = filepath.Join(home, ".devhive", "projects", project, "worktrees")
			} else {
				worktreesDir = filepath.Join(home, ".devhive", "worktrees")
			}

			// Check if directory exists
			if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
				fmt.Println("No worktrees directory found")
				return nil
			}

			// Get all registered workers
			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			workerNames := make(map[string]bool)
			completedWorkers := make(map[string]bool)
			for _, w := range workers {
				workerNames[w.Name] = true
				if w.Status == "completed" {
					completedWorkers[w.Name] = true
				}
			}

			// Scan worktrees directory
			entries, err := os.ReadDir(worktreesDir)
			if err != nil {
				return err
			}

			var toRemove []string
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				name := entry.Name()
				wtPath := filepath.Join(worktreesDir, name)

				// Check if worker exists
				if !workerNames[name] {
					toRemove = append(toRemove, wtPath)
					fmt.Printf("  %s (worker not found)\n", wtPath)
				} else if force && completedWorkers[name] {
					toRemove = append(toRemove, wtPath)
					fmt.Printf("  %s (worker completed)\n", wtPath)
				}
			}

			if len(toRemove) == 0 {
				fmt.Println("No worktrees to clean up")
				return nil
			}

			if dryRun {
				fmt.Printf("\nWould remove %d worktrees (use --dry-run=false to execute)\n", len(toRemove))
				return nil
			}

			// Remove worktrees
			for _, wtPath := range toRemove {
				// First try git worktree remove
				removeCmd := exec.Command("git", "worktree", "remove", wtPath, "--force")
				if err := removeCmd.Run(); err != nil {
					// Fall back to manual removal
					if err := os.RemoveAll(wtPath); err != nil {
						fmt.Printf("⚠ Failed to remove %s: %v\n", wtPath, err)
						continue
					}
				}
				fmt.Printf("✓ Removed %s\n", wtPath)
			}

			return nil
		},
	}
	worktreesCmd.Flags().Bool("dry-run", true, "Show what would be removed without removing")
	worktreesCmd.Flags().Bool("force", false, "Also remove worktrees for completed workers")

	// cleanup all - run all cleanup tasks
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Run all cleanup tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			fmt.Println("Cleaning up events...")
			eventCount, err := database.CleanupOldEvents(days, dryRun)
			if err != nil {
				fmt.Printf("⚠ Events cleanup failed: %v\n", err)
			} else if dryRun {
				fmt.Printf("  Would delete %d events\n", eventCount)
			} else {
				fmt.Printf("  ✓ Deleted %d events\n", eventCount)
			}

			fmt.Println("Cleaning up messages...")
			msgCount, err := database.CleanupOldMessages(days, dryRun)
			if err != nil {
				fmt.Printf("⚠ Messages cleanup failed: %v\n", err)
			} else if dryRun {
				fmt.Printf("  Would delete %d messages\n", msgCount)
			} else {
				fmt.Printf("  ✓ Deleted %d messages\n", msgCount)
			}

			return nil
		},
	}
	allCmd.Flags().Int("days", 30, "Keep data from last N days")
	allCmd.Flags().Bool("dry-run", false, "Show what would be deleted without deleting")

	cmd.AddCommand(eventsCmd, messagesCmd, worktreesCmd, allCmd)
	return cmd
}
