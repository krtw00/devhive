package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

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
			if cmd.Name() == "version" {
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
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(sprintCmd())
	rootCmd.AddCommand(roleCmd())
	rootCmd.AddCommand(workerCmd())
	rootCmd.AddCommand(msgCmd())
	rootCmd.AddCommand(eventsCmd())
	rootCmd.AddCommand(watchCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

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

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("devhive v0.3.0")
		},
	}
}

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

func statusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current status",
		RunE: func(cmd *cobra.Command, args []string) error {
			sprint, err := database.GetActiveSprint()
			if err != nil {
				return err
			}
			if sprint == nil {
				if jsonOutput {
					fmt.Println("{}")
				} else {
					fmt.Println("No active sprint")
				}
				return nil
			}

			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			if jsonOutput {
				output := map[string]interface{}{
					"project": db.GetProjectName(),
					"sprint":  sprint,
					"workers": workers,
				}
				b, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			// Show project name if set
			if project := db.GetProjectName(); project != "" {
				fmt.Printf("Project: %s\n", project)
			}
			fmt.Printf("Sprint: %s (started: %s)\n\n", sprint.ID, sprint.StartedAt.Format("2006-01-02 15:04"))

			if len(workers) == 0 {
				fmt.Println("No workers registered")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "WORKER\tROLE\tBRANCH\tSTATUS\tTASK\tMSGS")
			fmt.Fprintln(w, "------\t----\t------\t------\t----\t----")
			for _, worker := range workers {
				task := worker.CurrentTask
				if len(task) > 20 {
					task = task[:17] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
					worker.Name, worker.RoleName, worker.Branch, statusIcon(worker.Status),
					task, worker.UnreadMessages)
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

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

	cmd.AddCommand(completeCmd)
	return cmd
}

func roleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Role management",
	}

	// role create
	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description, _ := cmd.Flags().GetString("description")
			roleFile, _ := cmd.Flags().GetString("file")

			err := database.CreateRole(args[0], description, roleFile)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Role '%s' created\n", args[0])
			return nil
		},
	}
	createCmd.Flags().StringP("description", "d", "", "Role description")
	createCmd.Flags().StringP("file", "f", "", "Role definition file path")

	// role list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			roles, err := database.GetAllRoles()
			if err != nil {
				return err
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				b, _ := json.MarshalIndent(roles, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			if len(roles) == 0 {
				fmt.Println("No roles defined")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tFILE")
			fmt.Fprintln(w, "----\t-----------\t----")
			for _, role := range roles {
				desc := role.Description
				if len(desc) > 30 {
					desc = desc[:27] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", role.Name, desc, role.RoleFile)
			}
			w.Flush()
			return nil
		},
	}
	listCmd.Flags().Bool("json", false, "Output as JSON")

	// role show
	showCmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show role details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			role, err := database.GetRole(args[0])
			if err != nil {
				return err
			}
			if role == nil {
				return fmt.Errorf("role not found: %s", args[0])
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				b, _ := json.MarshalIndent(role, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			fmt.Printf("Name: %s\n", role.Name)
			if role.Description != "" {
				fmt.Printf("Description: %s\n", role.Description)
			}
			if role.RoleFile != "" {
				fmt.Printf("File: %s\n", role.RoleFile)
			}
			fmt.Printf("Created: %s\n", role.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
	showCmd.Flags().Bool("json", false, "Output as JSON")

	// role update
	updateCmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get current role first
			role, err := database.GetRole(args[0])
			if err != nil {
				return err
			}
			if role == nil {
				return fmt.Errorf("role not found: %s", args[0])
			}

			description := role.Description
			roleFile := role.RoleFile

			if cmd.Flags().Changed("description") {
				description, _ = cmd.Flags().GetString("description")
			}
			if cmd.Flags().Changed("file") {
				roleFile, _ = cmd.Flags().GetString("file")
			}

			err = database.UpdateRole(args[0], description, roleFile)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Role '%s' updated\n", args[0])
			return nil
		},
	}
	updateCmd.Flags().StringP("description", "d", "", "Role description")
	updateCmd.Flags().StringP("file", "f", "", "Role definition file path")

	// role delete
	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := database.DeleteRole(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Role '%s' deleted\n", args[0])
			return nil
		},
	}

	cmd.AddCommand(createCmd, listCmd, showCmd, updateCmd, deleteCmd)
	return cmd
}

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
			roleName, _ := cmd.Flags().GetString("role")
			worktree, _ := cmd.Flags().GetString("worktree")

			sprint, err := database.GetActiveSprint()
			if err != nil || sprint == nil {
				return fmt.Errorf("no active sprint")
			}

			err = database.RegisterWorker(args[0], sprint.ID, args[1], roleName, worktree)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' registered\n", args[0])
			return nil
		},
	}
	registerCmd.Flags().StringP("role", "r", "", "Role name (must exist in roles table)")
	registerCmd.Flags().StringP("worktree", "w", "", "Worktree path")

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

	cmd.AddCommand(registerCmd, startCmd, completeCmd, statusUpdateCmd, showCmd, taskCmd, errorCmd)
	return cmd
}

func msgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "msg",
		Short: "Message management",
	}

	// msg send
	sendCmd := &cobra.Command{
		Use:   "send <to> <message>",
		Short: "Send a message to a worker",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			from, err := getWorkerName(nil, 0)
			if err != nil {
				// Allow sending without being a worker (e.g., pm)
				from, _ = cmd.Flags().GetString("from")
				if from == "" {
					from = "pm"
				}
			}

			msgType, _ := cmd.Flags().GetString("type")
			subject, _ := cmd.Flags().GetString("subject")

			_, err = database.SendMessage(from, args[0], msgType, subject, args[1])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Message sent to '%s'\n", args[0])
			return nil
		},
	}
	sendCmd.Flags().StringP("from", "f", "", "Sender name (default: DEVHIVE_WORKER or 'pm')")
	sendCmd.Flags().StringP("type", "t", "info", "Message type")
	sendCmd.Flags().StringP("subject", "s", "", "Subject")

	// msg broadcast
	broadcastCmd := &cobra.Command{
		Use:   "broadcast <message>",
		Short: "Broadcast a message to all workers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			from, err := getWorkerName(nil, 0)
			if err != nil {
				from, _ = cmd.Flags().GetString("from")
				if from == "" {
					from = "pm"
				}
			}

			msgType, _ := cmd.Flags().GetString("type")
			subject, _ := cmd.Flags().GetString("subject")

			count, err := database.BroadcastMessage(from, msgType, subject, args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Message broadcast to %d workers\n", count)
			return nil
		},
	}
	broadcastCmd.Flags().StringP("from", "f", "", "Sender name (default: DEVHIVE_WORKER or 'pm')")
	broadcastCmd.Flags().StringP("type", "t", "info", "Message type")
	broadcastCmd.Flags().StringP("subject", "s", "", "Subject")

	// msg unread
	unreadCmd := &cobra.Command{
		Use:   "unread",
		Short: "Show unread messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(nil, 0)
			if err != nil {
				return err
			}

			messages, err := database.GetUnreadMessages(name)
			if err != nil {
				return err
			}

			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				b, _ := json.MarshalIndent(messages, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			if len(messages) == 0 {
				fmt.Println("No unread messages")
				return nil
			}

			for _, m := range messages {
				fmt.Printf("[%d] %s → you (%s)\n", m.ID, m.FromWorker, m.CreatedAt.Format("15:04"))
				if m.Subject != "" {
					fmt.Printf("    Subject: %s\n", m.Subject)
				}
				fmt.Printf("    %s\n\n", m.Content)
			}
			return nil
		},
	}
	unreadCmd.Flags().Bool("json", false, "Output as JSON")

	// msg read
	readCmd := &cobra.Command{
		Use:   "read <id|all>",
		Short: "Mark message(s) as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := getWorkerName(nil, 0)
			if err != nil {
				return err
			}

			if args[0] == "all" {
				count, err := database.MarkAllRead(name)
				if err != nil {
					return err
				}
				fmt.Printf("✓ Marked %d messages as read\n", count)
			} else {
				id, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid message ID: %s", args[0])
				}
				err = database.MarkMessageRead(id)
				if err != nil {
					return err
				}
				fmt.Printf("✓ Message #%d marked as read\n", id)
			}
			return nil
		},
	}

	cmd.AddCommand(sendCmd, broadcastCmd, unreadCmd, readCmd)
	return cmd
}

func eventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Show recent events",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			eventType, _ := cmd.Flags().GetString("type")
			worker, _ := cmd.Flags().GetString("worker")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			events, err := database.GetRecentEvents(limit, stringPtr(eventType), stringPtr(worker))
			if err != nil {
				return err
			}

			if jsonOutput {
				b, _ := json.MarshalIndent(events, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			if len(events) == 0 {
				fmt.Println("No events")
				return nil
			}

			for _, e := range events {
				workerStr := ""
				if e.Worker != "" {
					workerStr = fmt.Sprintf(" [%s]", e.Worker)
				}
				dataStr := ""
				if e.Data != "" && e.Data != "{}" {
					dataStr = " " + strings.ReplaceAll(e.Data, "\"", "")
				}
				fmt.Printf("%s %s%s%s\n", e.CreatedAt.Format("15:04:05"), e.EventType, workerStr, dataStr)
			}
			return nil
		},
	}

	cmd.Flags().IntP("limit", "l", 50, "Number of events to show")
	cmd.Flags().StringP("type", "t", "", "Filter by event type")
	cmd.Flags().StringP("worker", "w", "", "Filter by worker")
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func watchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch for state changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			interval, _ := cmd.Flags().GetInt("interval")
			filter, _ := cmd.Flags().GetString("filter")

			// Get current worker for message filtering
			currentWorker, _ := getWorkerName(nil, 0)

			// Get last event ID
			lastID, err := database.GetLastEventID()
			if err != nil {
				return err
			}

			fmt.Println("Watching for changes... (Ctrl+C to stop)")

			for {
				time.Sleep(time.Duration(interval) * time.Second)

				events, err := database.GetEventsSince(lastID, stringPtr(filter))
				if err != nil {
					continue
				}

				for _, e := range events {
					lastID = e.ID
					printEvent(e, currentWorker)
				}
			}
		},
	}

	cmd.Flags().IntP("interval", "i", 1, "Polling interval in seconds")
	cmd.Flags().StringP("filter", "f", "", "Filter events (message, worker)")
	return cmd
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

func printEvent(e db.Event, currentWorker string) {
	timestamp := e.CreatedAt.Format("15:04:05")

	switch e.EventType {
	case "message_sent":
		to := parseEventData(e.Data, "to")
		if currentWorker != "" && to != currentWorker {
			return // Skip messages not for us
		}
		fmt.Printf("[%s] message: %s -> %s\n", timestamp, e.Worker, to)

	case "message_broadcast":
		fmt.Printf("[%s] message: (broadcast) %s\n", timestamp, e.Worker)

	case "worker_status_changed":
		status := parseEventData(e.Data, "status")
		fmt.Printf("[%s] worker: %s -> %s\n", timestamp, e.Worker, status)

	case "worker_task_updated":
		task := parseEventData(e.Data, "task")
		fmt.Printf("[%s] task: %s: %s\n", timestamp, e.Worker, task)

	case "worker_error":
		msg := parseEventData(e.Data, "message")
		fmt.Printf("[%s] error: %s: %s\n", timestamp, e.Worker, msg)

	case "worker_registered":
		fmt.Printf("[%s] registered: %s\n", timestamp, e.Worker)

	case "sprint_created":
		sprintID := parseEventData(e.Data, "sprint_id")
		fmt.Printf("[%s] sprint created: %s\n", timestamp, sprintID)

	case "sprint_completed":
		sprintID := parseEventData(e.Data, "sprint_id")
		fmt.Printf("[%s] sprint completed: %s\n", timestamp, sprintID)

	default:
		workerStr := ""
		if e.Worker != "" {
			workerStr = fmt.Sprintf(" [%s]", e.Worker)
		}
		fmt.Printf("[%s] %s%s\n", timestamp, e.EventType, workerStr)
	}
}
