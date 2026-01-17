package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func workerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Worker management",
	}

	// worker register
	registerCmd := &cobra.Command{
		Use:   "register <name> <branch>",
		Short: "Register a worker",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workerName := args[0]
			branch := args[1]
			roleName, _ := cmd.Flags().GetString("role")
			worktree, _ := cmd.Flags().GetString("worktree")
			createWorktree, _ := cmd.Flags().GetBool("create-worktree")
			repoPath, _ := cmd.Flags().GetString("repo")

			sprint, err := database.GetActiveSprint()
			if err != nil || sprint == nil {
				return fmt.Errorf("no active sprint")
			}

			// Create worktree if requested
			if createWorktree {
				wt, err := createGitWorktree(workerName, branch, repoPath)
				if err != nil {
					return fmt.Errorf("failed to create worktree: %w", err)
				}
				worktree = wt
				fmt.Printf("✓ Worktree created at %s\n", worktree)
			}

			err = database.RegisterWorker(workerName, sprint.ID, branch, roleName, worktree)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' registered\n", workerName)
			return nil
		},
	}
	registerCmd.Flags().StringP("role", "r", "", "Role name (must exist in roles table)")
	registerCmd.Flags().StringP("worktree", "w", "", "Worktree path")
	registerCmd.Flags().BoolP("create-worktree", "c", false, "Create git worktree automatically")
	registerCmd.Flags().String("repo", "", "Git repository path (default: cwd)")

	// worker start
	startCmd := &cobra.Command{
		Use:   "start [name]",
		Short: "Mark worker as started",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(args, 0)
			if err != nil {
				return err
			}

			task, _ := cmd.Flags().GetString("task")
			var taskPtr *string
			if task != "" {
				taskPtr = &task
			}
			err = database.UpdateWorkerStatus(name, "working", taskPtr, nil)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' started\n", name)
			return nil
		},
	}
	startCmd.Flags().StringP("task", "t", "", "Current task description")

	// worker complete
	completeCmd := &cobra.Command{
		Use:   "complete [name]",
		Short: "Mark worker as completed",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(args, 0)
			if err != nil {
				return err
			}

			err = database.UpdateWorkerStatus(name, "completed", nil, nil)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' completed\n", name)
			return nil
		},
	}

	// worker status (update)
	statusUpdateCmd := &cobra.Command{
		Use:   "status [name] <status>",
		Short: "Update worker status",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name, status string
			if len(args) == 2 {
				name = args[0]
				status = args[1]
			} else {
				var err error
				name, err = getWorkerName(nil, 0)
				if err != nil {
					return err
				}
				status = args[0]
			}

			err := database.UpdateWorkerStatus(name, status, nil, nil)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' status updated to '%s'\n", name, status)
			return nil
		},
	}

	// worker show
	showCmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show worker details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(args, 0)
			if err != nil {
				return err
			}

			worker, err := database.GetWorker(name)
			if err != nil {
				return err
			}
			if worker == nil {
				return fmt.Errorf("worker not found: %s", name)
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				b, _ := json.MarshalIndent(worker, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			fmt.Printf("Worker: %s\n", worker.Name)
			if worker.RoleName != "" {
				fmt.Printf("Role: %s\n", worker.RoleName)
			}
			if worker.RoleFile != "" {
				fmt.Printf("Role File: %s\n", worker.RoleFile)
			}
			fmt.Printf("Branch: %s\n", worker.Branch)
			if worker.WorktreePath != "" {
				fmt.Printf("Worktree: %s\n", worker.WorktreePath)
			}
			fmt.Printf("Status: %s\n", statusIcon(worker.Status))
			fmt.Printf("Session: %s %s\n", sessionIcon(worker.SessionState), worker.SessionState)
			if worker.CurrentTask != "" {
				fmt.Printf("Task: %s\n", worker.CurrentTask)
			}
			if worker.LastCommit != "" {
				fmt.Printf("Last Commit: %s\n", worker.LastCommit)
			}
			fmt.Printf("Errors: %d\n", worker.ErrorCount)
			if worker.LastError != "" {
				fmt.Printf("Last Error: %s\n", worker.LastError)
			}
			fmt.Printf("Updated: %s\n", worker.UpdatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Unread Messages: %d\n", worker.UnreadMessages)

			return nil
		},
	}
	showCmd.Flags().Bool("json", false, "Output as JSON")

	// worker task
	taskCmd := &cobra.Command{
		Use:   "task <task>",
		Short: "Update current task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(nil, 0)
			if err != nil {
				return err
			}

			err = database.UpdateWorkerTask(name, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Task updated\n")
			return nil
		},
	}

	// worker error
	errorCmd := &cobra.Command{
		Use:   "error <message>",
		Short: "Report an error",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(nil, 0)
			if err != nil {
				return err
			}

			err = database.ReportWorkerError(name, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Error reported\n")
			return nil
		},
	}

	// worker session
	sessionCmd := &cobra.Command{
		Use:   "session <state>",
		Short: "Update session state (running|waiting_permission|idle|stopped)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(nil, 0)
			if err != nil {
				return err
			}

			state := args[0]
			validStates := map[string]bool{
				"running":            true,
				"waiting_permission": true,
				"idle":               true,
				"stopped":            true,
			}
			if !validStates[state] {
				return fmt.Errorf("invalid session state: %s (must be running|waiting_permission|idle|stopped)", state)
			}

			err = database.UpdateWorkerSessionState(name, state)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Session state updated to '%s'\n", state)
			return nil
		},
	}

	cmd.AddCommand(registerCmd, startCmd, completeCmd, statusUpdateCmd, showCmd, taskCmd, errorCmd, sessionCmd)
	return cmd
}
