package main

import (
	"fmt"
	"os"

	"github.com/iguchi/devhive/internal/db"
	"github.com/spf13/cobra"
)

var database *db.DB
var projectFlag string

func main() {
	rootCmd := &cobra.Command{
		Use:   "devhive",
		Short: "Parallel development coordination tool",
		Long: `DevHive - Docker-style parallel development coordination

Quick Start:
  1. devhive init      Initialize project (creates .devhive/)
  2. Edit .devhive.yaml to define workers
  3. devhive up        Start all workers (auto-creates worktrees)
  4. devhive ps        List running workers
  5. devhive logs -f   Follow event logs
  6. devhive down      Stop all workers`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip DB for commands that don't need it
			if cmd.Name() == "version" || cmd.Name() == "help" || cmd.Name() == "init" {
				return nil
			}

			// Set project name from flag (takes precedence over auto-detection)
			if projectFlag != "" {
				db.ProjectName = projectFlag
			}

			var err error
			database, err = db.Open("")
			return err
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if database != nil {
				database.Close()
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&projectFlag, "project", "P", "", "Project name (auto-detected from .devhive.yaml)")

	// Define command groups
	rootCmd.AddGroup(
		&cobra.Group{ID: "basic", Title: "Basic Commands:"},
		&cobra.Group{ID: "worker", Title: "Worker Management:"},
		&cobra.Group{ID: "utility", Title: "Utility Commands:"},
		&cobra.Group{ID: "comm", Title: "Communication:"},
	)

	// Basic commands
	rootCmd.AddCommand(withGroup(initCmd(), "basic"))
	rootCmd.AddCommand(withGroup(upCmd(), "basic"))
	rootCmd.AddCommand(withGroup(downCmd(), "basic"))
	rootCmd.AddCommand(withGroup(psCmd(), "basic"))
	rootCmd.AddCommand(withGroup(statusCmd(), "basic"))
	rootCmd.AddCommand(withGroup(logsCmd(), "basic"))
	rootCmd.AddCommand(withGroup(configCmd(), "basic"))

	// Worker management commands
	rootCmd.AddCommand(withGroup(startCmd(), "worker"))
	rootCmd.AddCommand(withGroup(stopCmd(), "worker"))
	rootCmd.AddCommand(withGroup(execCmd(), "worker"))
	rootCmd.AddCommand(withGroup(rmCmd(), "worker"))
	rootCmd.AddCommand(withGroup(rolesCmd(), "worker"))

	// Utility commands
	rootCmd.AddCommand(withGroup(progressCmd(), "utility"))
	rootCmd.AddCommand(withGroup(mergeCmd(), "utility"))
	rootCmd.AddCommand(withGroup(diffCmd(), "utility"))
	rootCmd.AddCommand(withGroup(noteCmd(), "utility"))
	rootCmd.AddCommand(withGroup(cleanCmd(), "utility"))

	// Communication commands
	rootCmd.AddCommand(withGroup(requestCmd(), "comm"))
	rootCmd.AddCommand(withGroup(reportCmd(), "comm"))
	rootCmd.AddCommand(withGroup(msgsCmd(), "comm"))
	rootCmd.AddCommand(withGroup(inboxCmd(), "comm"))
	rootCmd.AddCommand(withGroup(replyCmd(), "comm"))
	rootCmd.AddCommand(withGroup(broadcastCmd(), "comm"))

	// Other commands (no group - shown in "Additional Commands")
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(sessionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("devhive v0.7.0")
		},
	}
}

// withGroup sets the GroupID on a command and returns it
func withGroup(cmd *cobra.Command, groupID string) *cobra.Command {
	cmd.GroupID = groupID
	return cmd
}

