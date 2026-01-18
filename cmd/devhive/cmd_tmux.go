package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func tmuxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tmux [worker...]",
		Short: "Start workers in tmux panes",
		Long: `Start workers in a tmux session with split panes.

All workers are displayed in a single window with split panes.
Each pane runs the worker's configured command in its worktree.

Examples:
  devhive tmux                    # Start all workers in tmux
  devhive tmux frontend backend   # Start specific workers
  devhive tmux --session myapp    # Use custom session name
  devhive tmux --attach=false     # Create but don't attach`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName, _ := cmd.Flags().GetString("session")
			attach, _ := cmd.Flags().GetBool("attach")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			// Check tmux is available
			if _, err := exec.LookPath("tmux"); err != nil {
				return fmt.Errorf("tmux not found in PATH")
			}

			// Find and load compose file
			configFile, err := FindComposeFile()
			if err != nil {
				return err
			}
			config, err := LoadComposeFile(configFile)
			if err != nil {
				return err
			}

			// Get workers
			workers := config.GetEffectiveWorkers(args)
			if len(workers) == 0 {
				return fmt.Errorf("no workers to start")
			}

			// Sort worker names for consistent ordering
			workerNames := make([]string, 0, len(workers))
			for name := range workers {
				workerNames = append(workerNames, name)
			}
			sort.Strings(workerNames)

			// Default session name
			if sessionName == "" {
				sessionName = "devhive-" + config.Project
			}

			// Check if session already exists
			checkCmd := exec.Command("tmux", "has-session", "-t", sessionName)
			sessionExists := checkCmd.Run() == nil

			if sessionExists {
				if attach {
					fmt.Printf("Attaching to existing session: %s\n", sessionName)
					return tmuxAttach(sessionName)
				}
				return fmt.Errorf("session '%s' already exists (use -s to specify different name)", sessionName)
			}

			configDir := filepath.Dir(configFile)

			if dryRun {
				fmt.Println("=== DRY RUN ===")
				fmt.Printf("Would create tmux session: %s\n", sessionName)
				fmt.Printf("Workers (%d):\n", len(workerNames))
				for _, name := range workerNames {
					worker := workers[name]
					worktree := getWorktreePath(name, worker.Worktree)
					fmt.Printf("  %s: cd %s && %s\n", name, worktree, worker.GetFullCommand(name, config, configDir))
				}
				return nil
			}

			// Create tmux session with first worker
			firstName := workerNames[0]
			firstWorker := workers[firstName]
			firstWorktree := getWorktreePath(firstName, firstWorker.Worktree)

			// Ensure worktree exists
			if _, err := os.Stat(firstWorktree); os.IsNotExist(err) {
				return fmt.Errorf("worktree not found: %s (run 'devhive up' first)", firstWorktree)
			}

			// Create new session with first pane
			firstCmd := buildPaneCommand(firstName, firstWorker, firstWorktree, config, configDir)
			newSessionCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName,
				"-c", firstWorktree, "-n", "workers")
			if err := newSessionCmd.Run(); err != nil {
				return fmt.Errorf("failed to create tmux session: %w", err)
			}

			// Send command to first pane
			sendKeysCmd := exec.Command("tmux", "send-keys", "-t", sessionName+":workers", firstCmd, "Enter")
			sendKeysCmd.Run()

			// Rename first pane (set pane title via select-pane)
			exec.Command("tmux", "select-pane", "-t", sessionName+":workers.0", "-T", firstName).Run()

			fmt.Printf("✓ %s: %s\n", firstName, firstWorker.GetFullCommand(firstName, config, configDir))

			// Create additional panes for remaining workers
			for i := 1; i < len(workerNames); i++ {
				name := workerNames[i]
				worker := workers[name]
				worktree := getWorktreePath(name, worker.Worktree)

				// Ensure worktree exists
				if _, err := os.Stat(worktree); os.IsNotExist(err) {
					fmt.Printf("⚠ Skipping %s: worktree not found\n", name)
					continue
				}

				// Split pane
				splitCmd := exec.Command("tmux", "split-window", "-t", sessionName+":workers",
					"-c", worktree)
				if err := splitCmd.Run(); err != nil {
					fmt.Printf("⚠ Failed to create pane for %s: %v\n", name, err)
					continue
				}

				// Send command
				paneCmd := buildPaneCommand(name, worker, worktree, config, configDir)
				sendKeysCmd := exec.Command("tmux", "send-keys", "-t", sessionName+":workers", paneCmd, "Enter")
				sendKeysCmd.Run()

				// Set pane title
				exec.Command("tmux", "select-pane", "-t", sessionName+":workers", "-T", name).Run()

				fmt.Printf("✓ %s: %s\n", name, worker.GetFullCommand(name, config, configDir))
			}

			// Apply tiled layout for even distribution
			exec.Command("tmux", "select-layout", "-t", sessionName+":workers", "tiled").Run()

			// Enable pane border status to show names
			exec.Command("tmux", "set-option", "-t", sessionName, "pane-border-status", "top").Run()
			exec.Command("tmux", "set-option", "-t", sessionName, "pane-border-format", " #{pane_title} ").Run()

			fmt.Printf("\n✓ Created tmux session: %s (%d panes)\n", sessionName, len(workerNames))

			if attach {
				return tmuxAttach(sessionName)
			}

			fmt.Printf("\nTo attach: tmux attach -t %s\n", sessionName)
			return nil
		},
	}

	cmd.Flags().StringP("session", "s", "", "Tmux session name (default: devhive-<project>)")
	cmd.Flags().Bool("attach", true, "Attach to session after creation")
	cmd.Flags().Bool("dry-run", false, "Show what would be done")

	return cmd
}

// getWorktreePath returns the worktree path for a worker
func getWorktreePath(workerName, override string) string {
	if override != "" {
		return override
	}
	return filepath.Join(".devhive", "worktrees", workerName)
}

// buildPaneCommand builds the command to run in a pane
func buildPaneCommand(name string, worker ComposeWorker, worktree string, config *ComposeConfig, projectRoot string) string {
	// Set DEVHIVE_WORKER environment variable and run command
	cmd := worker.GetFullCommand(name, config, projectRoot)
	return fmt.Sprintf("export DEVHIVE_WORKER=%s && %s", name, cmd)
}

// tmuxAttach attaches to a tmux session
func tmuxAttach(sessionName string) error {
	// Check if we're already in tmux
	if os.Getenv("TMUX") != "" {
		// Switch client instead of attach
		switchCmd := exec.Command("tmux", "switch-client", "-t", sessionName)
		switchCmd.Stdin = os.Stdin
		switchCmd.Stdout = os.Stdout
		switchCmd.Stderr = os.Stderr
		return switchCmd.Run()
	}

	attachCmd := exec.Command("tmux", "attach", "-t", sessionName)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr
	return attachCmd.Run()
}

func tmuxKillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tmux-kill [session]",
		Short: "Kill tmux session",
		Long: `Kill the DevHive tmux session.

Examples:
  devhive tmux-kill                # Kill default session
  devhive tmux-kill myapp          # Kill specific session`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := ""
			if len(args) > 0 {
				sessionName = args[0]
			} else {
				// Try to find default session name
				configFile, err := FindComposeFile()
				if err == nil {
					config, err := LoadComposeFile(configFile)
					if err == nil {
						sessionName = "devhive-" + config.Project
					}
				}
			}

			if sessionName == "" {
				return fmt.Errorf("session name required")
			}

			killCmd := exec.Command("tmux", "kill-session", "-t", sessionName)
			if err := killCmd.Run(); err != nil {
				return fmt.Errorf("failed to kill session '%s': %w", sessionName, err)
			}

			fmt.Printf("✓ Killed session: %s\n", sessionName)
			return nil
		},
	}
}

func tmuxListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tmux-list",
		Short: "List DevHive tmux sessions",
		Long:  `List all tmux sessions that match DevHive naming convention.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			listCmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
			output, err := listCmd.Output()
			if err != nil {
				fmt.Println("No tmux sessions")
				return nil
			}

			sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
			devhiveSessions := []string{}
			for _, s := range sessions {
				if strings.HasPrefix(s, "devhive-") {
					devhiveSessions = append(devhiveSessions, s)
				}
			}

			if len(devhiveSessions) == 0 {
				fmt.Println("No DevHive tmux sessions")
				return nil
			}

			fmt.Println("DevHive tmux sessions:")
			for _, s := range devhiveSessions {
				fmt.Printf("  %s\n", s)
			}
			return nil
		},
	}
}
