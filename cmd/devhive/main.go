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
  1. Create .devhive.yaml in your project root
  2. devhive up        Start all workers (auto-creates worktrees)
  3. devhive ps        List running workers
  4. devhive logs -f   Follow event logs
  5. devhive down      Stop all workers`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip DB for version command
			if cmd.Name() == "version" || cmd.Name() == "help" {
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

	// Commands
	rootCmd.AddCommand(versionCmd())

	// Docker-style commands
	rootCmd.AddCommand(upCmd())
	rootCmd.AddCommand(downCmd())
	rootCmd.AddCommand(psCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(logsCmd())
	rootCmd.AddCommand(rmCmd())
	rootCmd.AddCommand(execCmd())
	rootCmd.AddCommand(rolesCmd())
	rootCmd.AddCommand(configCmd())

	// Internal commands (for hooks)
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
			fmt.Println("devhive v0.6.0")
		},
	}
}

