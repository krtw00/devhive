package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/iguchi/devhive/internal/db"
	"github.com/iguchi/devhive/internal/templates"
	"github.com/spf13/cobra"
)

func upCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [worker...]",
		Short: "Start workers from compose file",
		Long: `Start workers defined in .devhive.yaml compose file.

This command:
1. Reads .devhive.yaml from current directory
2. Creates/updates roles defined in the config
3. Creates a sprint if none exists
4. Registers all workers (or specified workers)
5. Creates git worktrees (default)

Examples:
  devhive up                    # Start all workers with worktrees
  devhive up perf-fe perf-be    # Start specific workers
  devhive up --no-worktree      # Start without creating worktrees`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			noWorktree, _ := cmd.Flags().GetBool("no-worktree")
			repoPath, _ := cmd.Flags().GetString("repo")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			createWorktrees := !noWorktree

			// Find compose file
			var err error
			if configFile == "" {
				configFile, err = FindComposeFile()
				if err != nil {
					return err
				}
			}

			// Load config
			config, err := LoadComposeFile(configFile)
			if err != nil {
				return err
			}

			configDir := filepath.Dir(configFile)
			fmt.Printf("Using compose file: %s\n", configFile)
			fmt.Printf("Project: %s\n\n", config.Project)

			if dryRun {
				fmt.Println("=== DRY RUN MODE ===\n")
			}


			// Step 1: Create/update roles
			fmt.Println("Registering roles...")
			for roleName, role := range config.Roles {
				// Skip if it only extends a builtin
				if role.Extends != "" && role.File == "" && role.Content == "" {
					fmt.Printf("  → %s (extends %s)\n", roleName, role.Extends)
					continue
				}

				// Get role file path
				roleFile := ""
				if role.File != "" {
					roleFile = role.File
					if !filepath.IsAbs(roleFile) {
						roleFile = filepath.Join(configDir, roleFile)
					}
				}

				if dryRun {
					fmt.Printf("  → Would create role: %s\n", roleName)
					continue
				}

				// Check if role exists
				existingRole, _ := database.GetRole(roleName)
				if existingRole != nil {
					// Update existing role
					err := database.UpdateRole(roleName, role.Description, roleFile, role.Args)
					if err != nil {
						fmt.Printf("  ⚠ Failed to update role %s: %v\n", roleName, err)
					} else {
						fmt.Printf("  ✓ Updated role: %s\n", roleName)
					}
				} else {
					// Create new role
					err := database.CreateRole(roleName, role.Description, roleFile, role.Args)
					if err != nil {
						fmt.Printf("  ⚠ Failed to create role %s: %v\n", roleName, err)
					} else {
						fmt.Printf("  ✓ Created role: %s\n", roleName)
					}
				}
			}
			fmt.Println()

			// Step 2: Ensure sprint exists
			sprint, err := database.GetActiveSprint()
			if err != nil {
				return err
			}

			sprintID := config.GenerateSprintID()
			if sprint == nil {
				if dryRun {
					fmt.Printf("Would create sprint: %s\n\n", sprintID)
				} else {
					if err := database.CreateSprint(sprintID, configFile, ""); err != nil {
						return fmt.Errorf("failed to create sprint: %w", err)
					}
					fmt.Printf("✓ Created sprint: %s\n\n", sprintID)
				}
			} else {
				sprintID = sprint.ID
				fmt.Printf("Using existing sprint: %s\n\n", sprintID)
			}

			// Step 3: Register workers
			fmt.Println("Registering workers...")
			workers := config.GetEffectiveWorkers(args)
			if len(workers) == 0 {
				fmt.Println("No workers to register")
				return nil
			}

			registeredCount := 0
			for workerName, worker := range workers {
				worktreePath := worker.Worktree

				// Resolve role name
				roleName := config.ResolveRole(worker.Role)

				// Validate role exists (either in DB or as builtin)
				if roleName != "" && !strings.HasPrefix(worker.Role, "@") {
					existingRole, _ := database.GetRole(roleName)
					if existingRole == nil && !templates.IsBuiltinRole(roleName) {
						fmt.Printf("  ⚠ Role not found: %s (skipping %s)\n", roleName, workerName)
						continue
					}
				}

				if dryRun {
					fmt.Printf("  → Would register: %s (branch: %s, role: %s)\n", workerName, worker.Branch, roleName)
					continue
				}

				// Create worktree if requested
				if createWorktrees && worktreePath == "" {
					wt, err := createGitWorktree(workerName, worker.Branch, repoPath)
					if err != nil {
						fmt.Printf("  ⚠ Failed to create worktree for %s: %v\n", workerName, err)
					} else {
						worktreePath = wt
						fmt.Printf("  ✓ Worktree: %s\n", wt)

						// Create .envrc for direnv
						if err := createWorkerEnvrc(wt, workerName); err != nil {
							fmt.Printf("    ⚠ Failed to create .envrc: %v\n", err)
						}
					}
				}

				// Register worker
				err := database.RegisterWorker(workerName, sprintID, worker.Branch, roleName, worktreePath)
				if err != nil {
					fmt.Printf("  ⚠ Failed to register %s: %v\n", workerName, err)
					continue
				}

				// Set task (from .devhive/tasks/<name>.md or inline)
				taskContent := GetTaskContent(configDir, workerName, worker.Task)
				if taskContent != "" {
					if err := database.UpdateWorkerTask(workerName, strings.TrimSpace(taskContent)); err != nil {
						fmt.Printf("  ⚠ Failed to set task for %s: %v\n", workerName, err)
					}
				}

				// Build output
				roleStr := ""
				if roleName != "" {
					roleStr = fmt.Sprintf(", role: %s", roleName)
				}
				fmt.Printf("  ✓ %s (branch: %s%s)\n", workerName, worker.Branch, roleStr)
				registeredCount++
			}

			fmt.Printf("\n✓ Registered %d workers\n", registeredCount)

			if createWorktrees && registeredCount > 0 {
				fmt.Println("\nTip: Run 'direnv allow' in each worktree to enable environment variables")
			}

			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "Compose file path (default: .devhive.yaml)")
	cmd.Flags().Bool("no-worktree", false, "Skip creating git worktrees")
	cmd.Flags().String("repo", "", "Git repository path (default: cwd)")
	cmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")

	return cmd
}

func downCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down [worker...]",
		Short: "Stop workers (mark as completed)",
		Long: `Mark workers as completed.

Examples:
  devhive down                  # Complete all workers
  devhive down perf-fe perf-be  # Complete specific workers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If specific workers provided, complete them
			if len(args) > 0 {
				for _, name := range args {
					if err := database.UpdateWorkerStatus(name, "completed", nil, nil); err != nil {
						fmt.Printf("⚠ Failed to complete %s: %v\n", name, err)
					} else {
						fmt.Printf("✓ Worker '%s' completed\n", name)
					}
				}
				return nil
			}

			// Complete all workers in active sprint
			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			if len(workers) == 0 {
				fmt.Println("No workers to complete")
				return nil
			}

			completeSprint, _ := cmd.Flags().GetBool("sprint")

			for _, w := range workers {
				if w.Status == "completed" {
					continue
				}
				if err := database.UpdateWorkerStatus(w.Name, "completed", nil, nil); err != nil {
					fmt.Printf("⚠ Failed to complete %s: %v\n", w.Name, err)
				} else {
					fmt.Printf("✓ Worker '%s' completed\n", w.Name)
				}
			}

			// Optionally complete the sprint too
			if completeSprint {
				sprintID, err := database.CompleteSprint()
				if err != nil {
					return err
				}
				fmt.Printf("✓ Sprint '%s' completed\n", sprintID)
			}

			return nil
		},
	}

	cmd.Flags().Bool("sprint", false, "Also complete the sprint")

	return cmd
}

func psCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List workers",
		Long: `List workers in the current sprint.

Like 'docker ps', shows running workers by default.
Use -a to show all workers including completed ones.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			showAll, _ := cmd.Flags().GetBool("all")
			quiet, _ := cmd.Flags().GetBool("quiet")

			sprint, err := database.GetActiveSprint()
			if err != nil {
				return err
			}
			if sprint == nil {
				fmt.Println("No active sprint")
				return nil
			}

			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			// Filter workers
			var filtered []struct {
				Name    string
				Branch  string
				Role    string
				Status  string
				Session string
				Task    string
			}

			for _, w := range workers {
				// Skip completed workers unless -a flag
				if !showAll && w.Status == "completed" {
					continue
				}

				task := w.CurrentTask
				if len(task) > 30 {
					task = task[:27] + "..."
				}

				filtered = append(filtered, struct {
					Name    string
					Branch  string
					Role    string
					Status  string
					Session string
					Task    string
				}{
					Name:    w.Name,
					Branch:  w.Branch,
					Role:    w.RoleName,
					Status:  w.Status,
					Session: w.SessionState,
					Task:    task,
				})
			}

			if len(filtered) == 0 {
				if showAll {
					fmt.Println("No workers")
				} else {
					fmt.Println("No running workers")
				}
				return nil
			}

			// Quiet mode - just names
			if quiet {
				for _, w := range filtered {
					fmt.Println(w.Name)
				}
				return nil
			}

			// Table output (Docker ps style)
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tBRANCH\tROLE\tSTATUS\tSESSION\tTASK")
			for _, w := range filtered {
				statusStr := statusIcon(w.Status)
				sessionStr := fmt.Sprintf("%s %s", sessionIcon(w.Session), w.Session)
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
					w.Name, w.Branch, w.Role, statusStr, sessionStr, w.Task)
			}
			tw.Flush()

			return nil
		},
	}

	cmd.Flags().BoolP("all", "a", false, "Show all workers (including completed)")
	cmd.Flags().BoolP("quiet", "q", false, "Only display worker names")

	return cmd
}

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <worker> [worker...]",
		Short: "Start one or more workers",
		Long: `Start one or more workers (mark as working).

Like 'docker start', this resumes a stopped worker.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, _ := cmd.Flags().GetString("task")

			for _, name := range args {
				var taskPtr *string
				if task != "" {
					taskPtr = &task
				}

				if err := database.UpdateWorkerStatus(name, "working", taskPtr, nil); err != nil {
					fmt.Printf("Error: %s - %v\n", name, err)
					continue
				}
				fmt.Println(name)
			}
			return nil
		},
	}

	cmd.Flags().StringP("task", "t", "", "Set task description")

	return cmd
}

func stopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <worker> [worker...]",
		Short: "Stop one or more workers",
		Long: `Stop one or more workers (mark as completed or blocked).

Like 'docker stop', this stops a running worker.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			block, _ := cmd.Flags().GetBool("block")

			status := "completed"
			if block {
				status = "blocked"
			}

			for _, name := range args {
				if err := database.UpdateWorkerStatus(name, status, nil, nil); err != nil {
					fmt.Printf("Error: %s - %v\n", name, err)
					continue
				}
				fmt.Println(name)
			}
			return nil
		},
	}

	cmd.Flags().BoolP("block", "b", false, "Mark as blocked instead of completed")

	return cmd
}

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [worker]",
		Short: "Fetch logs of a worker or all workers",
		Long: `View event logs for workers.

Like 'docker logs', shows the event history.
Without arguments, shows all events.
With a worker name, shows only that worker's events.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("tail")
			follow, _ := cmd.Flags().GetBool("follow")

			var workerPtr *string
			if len(args) > 0 {
				workerPtr = &args[0]
			}

			// If follow mode, use watch-like behavior
			if follow {
				fmt.Println("Following logs... (Ctrl+C to stop)")

				lastID, _ := database.GetLastEventID()

				for {
					time.Sleep(1 * time.Second)

					events, err := database.GetEventsSince(lastID, nil)
					if err != nil {
						continue
					}

					for _, e := range events {
						lastID = e.ID
						// Filter by worker if specified
						if workerPtr != nil && e.Worker != *workerPtr {
							continue
						}
						printLogLine(e)
					}
				}
			}

			// Non-follow mode
			events, err := database.GetRecentEvents(limit, nil, workerPtr)
			if err != nil {
				return err
			}

			if len(events) == 0 {
				fmt.Println("No logs")
				return nil
			}

			// Reverse to show oldest first (like docker logs)
			for i := len(events) - 1; i >= 0; i-- {
				printLogLine(events[i])
			}

			return nil
		},
	}

	cmd.Flags().IntP("tail", "n", 50, "Number of lines to show")
	cmd.Flags().BoolP("follow", "f", false, "Follow log output")

	return cmd
}

func printLogLine(e db.Event) {
	timestamp := e.CreatedAt.Format("2006-01-02 15:04:05")
	fmt.Printf("%s %s", timestamp, e.EventType)
	if e.Worker != "" {
		fmt.Printf(" [%s]", e.Worker)
	}
	if e.Data != "" && e.Data != "{}" {
		// Clean up JSON formatting
		data := strings.ReplaceAll(e.Data, "\"", "")
		data = strings.ReplaceAll(data, "{", "")
		data = strings.ReplaceAll(data, "}", "")
		fmt.Printf(" %s", data)
	}
	fmt.Println()
}

// createExecCommand creates an exec.Cmd for the given command
func createExecCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func rmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <worker> [worker...]",
		Short: "Remove one or more workers",
		Long: `Remove workers from the sprint.

Like 'docker rm', this removes stopped workers.
Use -f to force remove running workers.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			for _, name := range args {
				// Check worker status
				worker, err := database.GetWorker(name)
				if err != nil {
					fmt.Printf("Error: %s - %v\n", name, err)
					continue
				}
				if worker == nil {
					fmt.Printf("Error: worker not found - %s\n", name)
					continue
				}

				// Don't remove running workers unless forced
				if !force && (worker.Status == "working" || worker.SessionState == "running") {
					fmt.Printf("Error: cannot remove running worker %s (use -f to force)\n", name)
					continue
				}

				// Delete from database
				if err := database.DeleteWorker(name); err != nil {
					fmt.Printf("Error: %s - %v\n", name, err)
					continue
				}
				fmt.Println(name)
			}
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Force remove running workers")

	return cmd
}

func execCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <worker> <command> [args...]",
		Short: "Execute a command in a worker's worktree",
		Long: `Execute a command in a worker's worktree directory.

Like 'docker exec', runs a command in the worker's environment.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			workerName := args[0]
			command := args[1:]

			// Get worker info
			worker, err := database.GetWorker(workerName)
			if err != nil {
				return err
			}
			if worker == nil {
				return fmt.Errorf("worker not found: %s", workerName)
			}

			if worker.WorktreePath == "" {
				return fmt.Errorf("worker %s has no worktree", workerName)
			}

			// Check worktree exists
			if _, err := os.Stat(worker.WorktreePath); os.IsNotExist(err) {
				return fmt.Errorf("worktree does not exist: %s", worker.WorktreePath)
			}

			// Execute command
			execCmd := createExecCommand(command[0], command[1:]...)
			execCmd.Dir = worker.WorktreePath
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Stdin = os.Stdin

			// Set environment
			execCmd.Env = append(os.Environ(), fmt.Sprintf("DEVHIVE_WORKER=%s", workerName))

			return execCmd.Run()
		},
	}

	return cmd
}

func rolesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roles",
		Short: "List roles (like docker images)",
		Long: `List all available roles.

Like 'docker images', shows available role templates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			showBuiltin, _ := cmd.Flags().GetBool("builtin")

			if showBuiltin {
				// Show builtin roles
				builtinRoles := templates.GetBuiltinRoles()

				tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "NAME\tDESCRIPTION\tTYPE")
				for _, role := range builtinRoles {
					fmt.Fprintf(tw, "@%s\t%s\tbuiltin\n", role.Name, role.Description)
				}
				tw.Flush()
				return nil
			}

			// Show user-defined roles
			roles, err := database.GetAllRoles()
			if err != nil {
				return err
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tDESCRIPTION\tFILE")

			// First show user-defined roles
			for _, role := range roles {
				desc := role.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n", role.Name, desc, role.RoleFile)
			}

			tw.Flush()

			if len(roles) == 0 {
				fmt.Println("No user-defined roles")
				fmt.Println("\nTip: Use 'devhive roles --builtin' to see built-in roles")
			}

			return nil
		},
	}

	cmd.Flags().BoolP("builtin", "b", false, "Show built-in roles only")

	return cmd
}

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session <state>",
		Short: "Update worker session state (for hooks)",
		Long: `Update the session state of the current worker.

Used by Claude Code hooks for automatic session tracking.
Valid states: running, waiting_permission, idle, stopped`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			state := args[0]

			// Validate state
			validStates := map[string]bool{
				"running":            true,
				"waiting_permission": true,
				"idle":               true,
				"stopped":            true,
			}
			if !validStates[state] {
				return fmt.Errorf("invalid state: %s (valid: running, waiting_permission, idle, stopped)", state)
			}

			// Get worker name from env
			workerName := os.Getenv("DEVHIVE_WORKER")
			if workerName == "" {
				return fmt.Errorf("DEVHIVE_WORKER not set")
			}

			return database.UpdateWorkerSessionState(workerName, state)
		},
	}

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show compose configuration",
		Long:  `Display the parsed compose configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile, _ := cmd.Flags().GetString("file")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			// Find compose file
			var err error
			if configFile == "" {
				configFile, err = FindComposeFile()
				if err != nil {
					return err
				}
			}

			// Load config
			config, err := LoadComposeFile(configFile)
			if err != nil {
				return err
			}

			if jsonOutput {
				b, _ := json.MarshalIndent(config, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			fmt.Printf("Compose file: %s\n", configFile)
			fmt.Printf("Version: %s\n", config.Version)
			fmt.Printf("Project: %s\n", config.Project)
			fmt.Println()

			// Defaults
			if config.Defaults.BaseBranch != "" || config.Defaults.Sprint != "" {
				fmt.Println("Defaults:")
				if config.Defaults.BaseBranch != "" {
					fmt.Printf("  base_branch: %s\n", config.Defaults.BaseBranch)
				}
				if config.Defaults.Sprint != "" {
					fmt.Printf("  sprint: %s\n", config.Defaults.Sprint)
				}
				fmt.Println()
			}

			// Roles
			if len(config.Roles) > 0 {
				fmt.Println("Roles:")
				for name, role := range config.Roles {
					desc := role.Description
					if desc == "" && role.Extends != "" {
						desc = fmt.Sprintf("extends %s", role.Extends)
					}
					fmt.Printf("  %s: %s\n", name, desc)
					if role.File != "" {
						fmt.Printf("    file: %s\n", role.File)
					}
				}
				fmt.Println()
			}

			// Workers
			if len(config.Workers) > 0 {
				fmt.Println("Workers:")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "  NAME\tBRANCH\tROLE\tDISABLED")
				fmt.Fprintln(w, "  ----\t------\t----\t--------")
				for name, worker := range config.Workers {
					disabled := ""
					if worker.Disabled {
						disabled = "yes"
					}
					fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", name, worker.Branch, worker.Role, disabled)
				}
				w.Flush()
			}

			return nil
		},
	}

	cmd.Flags().StringP("file", "f", "", "Compose file path")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}
