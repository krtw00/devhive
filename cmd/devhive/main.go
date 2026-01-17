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
		Long:  "DevHive - Manage parallel development state with SQLite",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip DB for version command
			if cmd.Name() == "version" || cmd.Name() == "help" {
				return nil
			}

			// Set project name from flag (takes precedence over env var)
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
	rootCmd.PersistentFlags().StringVarP(&projectFlag, "project", "P", "", "Project name (or set DEVHIVE_PROJECT)")

	// Add commands
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(helpCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(projectsCmd())
	rootCmd.AddCommand(sprintCmd())
	rootCmd.AddCommand(roleCmd())
	rootCmd.AddCommand(workerCmd())
	rootCmd.AddCommand(msgCmd())
	rootCmd.AddCommand(eventsCmd())
	rootCmd.AddCommand(watchCmd())
	rootCmd.AddCommand(cleanupCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("devhive v0.5.0")
		},
	}
}

func helpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help",
		Short: "Show help and common workflows",
		Run: func(cmd *cobra.Command, args []string) {
			help := `DevHive - Parallel Development Coordination Tool

QUICK START (PM):
  export DEVHIVE_PROJECT=myapp
  devhive init sprint-01
  devhive role create frontend -d "Frontend dev"
  devhive worker register fe feat/frontend --role frontend
  devhive status

QUICK START (Worker):
  export DEVHIVE_PROJECT=myapp
  export DEVHIVE_WORKER=fe
  devhive worker start --task "Implementing UI"
  devhive worker session running
  devhive msg unread
  devhive worker complete

COMMANDS:
  init <sprint-id>          Initialize a new sprint
  status                    Show current sprint status
  projects                  List all projects (cross-project view)

  sprint complete           Complete the active sprint
  sprint setup <file>       Batch register workers from config
  sprint report             Generate sprint report

  role create <name>        Create a role
  role list                 List all roles

  worker register <n> <b>   Register a worker
  worker start              Start working
  worker complete           Mark as completed
  worker task <desc>        Update current task
  worker error <msg>        Report an error
  worker session <state>    Update session state
  worker show               Show worker details

  msg send <to> <msg>       Send a message
  msg broadcast <msg>       Broadcast to all
  msg unread                Show unread messages
  msg read <id|all>         Mark as read

  events                    Show recent events
  watch                     Watch for changes

  cleanup events            Remove old events
  cleanup messages          Remove old messages
  cleanup worktrees         Remove unused worktrees
  cleanup all               Run all cleanup tasks

ENVIRONMENT VARIABLES:
  DEVHIVE_PROJECT           Project name (selects database)
  DEVHIVE_WORKER            Default worker name

SESSION STATES:
  running              ▶  Worker is actively working
  waiting_permission   ⏸  Waiting for user input/permission
  idle                 ○  Session open but idle
  stopped              ■  Session closed

For detailed help: devhive <command> --help
`
			fmt.Print(help)
		},
	}
}
