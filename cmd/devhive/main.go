package main

import (
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

func main() {
	rootCmd := &cobra.Command{
		Use:   "devhive",
		Short: "Parallel development coordination tool",
		Long:  "DevHive - Manage parallel development with tmux, git worktree, and multiple Claude Code instances",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip DB for version command
			if cmd.Name() == "version" {
				return nil
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

	// Add commands
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(workerCmd())
	rootCmd.AddCommand(reviewCmd())
	rootCmd.AddCommand(msgCmd())
	rootCmd.AddCommand(lockCmd())
	rootCmd.AddCommand(eventsCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("devhive v0.1.0")
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
				fmt.Println("No active sprint")
				return nil
			}

			fmt.Printf("Sprint: %s (started: %s)\n\n", sprint.ID, sprint.StartedAt.Format("2006-01-02 15:04"))

			workers, err := database.GetAllWorkers()
			if err != nil {
				return err
			}

			if len(workers) == 0 {
				fmt.Println("No workers registered")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "WORKER\tBRANCH\tISSUE\tSTATUS\tCOMMIT\tREVIEWS\tMSGS")
			fmt.Fprintln(w, "------\t------\t-----\t------\t------\t-------\t----")
			for _, worker := range workers {
				commit := worker.LastCommit
				if len(commit) > 7 {
					commit = commit[:7]
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n",
					worker.Name, worker.Branch, worker.Issue, statusIcon(worker.Status),
					commit, worker.PendingReviews, worker.UnreadMessages)
			}
			w.Flush()

			// Show pending reviews
			reviews, _ := database.GetPendingReviews()
			if len(reviews) > 0 {
				fmt.Printf("\nPending Reviews: %d\n", len(reviews))
			}

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
	case "review_pending":
		return "👀 review"
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
			issue, _ := cmd.Flags().GetString("issue")
			paneStr, _ := cmd.Flags().GetString("pane")
			worktree, _ := cmd.Flags().GetString("worktree")

			sprint, err := database.GetActiveSprint()
			if err != nil || sprint == nil {
				return fmt.Errorf("no active sprint")
			}

			var paneID *int
			if paneStr != "" {
				p, _ := strconv.Atoi(paneStr)
				paneID = &p
			}

			err = database.RegisterWorker(args[0], sprint.ID, args[1], issue, paneID, worktree)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' registered\n", args[0])
			return nil
		},
	}
	registerCmd.Flags().StringP("issue", "i", "", "Issue number")
	registerCmd.Flags().StringP("pane", "p", "", "Tmux pane ID")
	registerCmd.Flags().StringP("worktree", "w", "", "Worktree path")

	// worker start
	startCmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Mark worker as started",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, _ := cmd.Flags().GetString("task")
			var taskPtr *string
			if task != "" {
				taskPtr = &task
			}
			err := database.UpdateWorkerStatus(args[0], "working", taskPtr, nil, nil)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' started\n", args[0])
			return nil
		},
	}
	startCmd.Flags().StringP("task", "t", "", "Current task description")

	// worker complete
	completeCmd := &cobra.Command{
		Use:   "complete <name>",
		Short: "Mark worker as completed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := database.UpdateWorkerStatus(args[0], "completed", nil, nil, nil)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' completed\n", args[0])
			return nil
		},
	}

	// worker status (update)
	statusCmd := &cobra.Command{
		Use:   "status <name> <status>",
		Short: "Update worker status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := database.UpdateWorkerStatus(args[0], args[1], nil, nil, nil)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worker '%s' status updated to '%s'\n", args[0], args[1])
			return nil
		},
	}

	cmd.AddCommand(registerCmd, startCmd, completeCmd, statusCmd)
	return cmd
}

func reviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review management",
	}

	// review request
	requestCmd := &cobra.Command{
		Use:   "request <commit>",
		Short: "Request a review",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, _ := cmd.Flags().GetString("worker")
			desc, _ := cmd.Flags().GetString("desc")

			if worker == "" {
				return fmt.Errorf("--worker is required")
			}

			id, err := database.RequestReview(worker, args[0], desc)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Review requested (ID: %d)\n", id)
			return nil
		},
	}
	requestCmd.Flags().StringP("worker", "w", "", "Worker name (required)")
	requestCmd.Flags().StringP("desc", "d", "", "Description")

	// review list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pending reviews",
		RunE: func(cmd *cobra.Command, args []string) error {
			reviews, err := database.GetPendingReviews()
			if err != nil {
				return err
			}

			if len(reviews) == 0 {
				fmt.Println("No pending reviews")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tWORKER\tCOMMIT\tBRANCH\tISSUE\tDESCRIPTION\tCREATED")
			fmt.Fprintln(w, "--\t------\t------\t------\t-----\t-----------\t-------")
			for _, r := range reviews {
				desc := r.Description
				if len(desc) > 30 {
					desc = desc[:27] + "..."
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
					r.ID, r.Worker, r.CommitHash[:7], r.Branch, r.Issue, desc,
					r.CreatedAt.Format("15:04"))
			}
			w.Flush()
			return nil
		},
	}

	// review ok
	okCmd := &cobra.Command{
		Use:   "ok <id> [comment]",
		Short: "Approve a review",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.Atoi(args[0])
			reviewer, _ := cmd.Flags().GetString("reviewer")
			comment := ""
			if len(args) > 1 {
				comment = args[1]
			}

			err := database.ResolveReview(id, "ok", reviewer, comment)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Review #%d approved\n", id)
			return nil
		},
	}
	okCmd.Flags().StringP("reviewer", "r", "senior", "Reviewer name")

	// review fix
	fixCmd := &cobra.Command{
		Use:   "fix <id> <comment>",
		Short: "Request fixes for a review",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.Atoi(args[0])
			reviewer, _ := cmd.Flags().GetString("reviewer")

			err := database.ResolveReview(id, "needs_fix", reviewer, args[1])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Review #%d marked as needs_fix\n", id)
			return nil
		},
	}
	fixCmd.Flags().StringP("reviewer", "r", "senior", "Reviewer name")

	cmd.AddCommand(requestCmd, listCmd, okCmd, fixCmd)
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
			from, _ := cmd.Flags().GetString("from")
			msgType, _ := cmd.Flags().GetString("type")
			subject, _ := cmd.Flags().GetString("subject")

			to := args[0]
			_, err := database.SendMessage(from, &to, msgType, subject, args[1])
			if err != nil {
				return err
			}
			fmt.Printf("✓ Message sent to '%s'\n", to)
			return nil
		},
	}
	sendCmd.Flags().StringP("from", "f", "pm", "Sender name")
	sendCmd.Flags().StringP("type", "t", "info", "Message type")
	sendCmd.Flags().StringP("subject", "s", "", "Subject")

	// msg broadcast
	broadcastCmd := &cobra.Command{
		Use:   "broadcast <message>",
		Short: "Broadcast a message to all workers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			from, _ := cmd.Flags().GetString("from")
			msgType, _ := cmd.Flags().GetString("type")
			subject, _ := cmd.Flags().GetString("subject")

			_, err := database.SendMessage(from, nil, msgType, subject, args[0])
			if err != nil {
				return err
			}
			fmt.Println("✓ Message broadcast to all workers")
			return nil
		},
	}
	broadcastCmd.Flags().StringP("from", "f", "pm", "Sender name")
	broadcastCmd.Flags().StringP("type", "t", "info", "Message type")
	broadcastCmd.Flags().StringP("subject", "s", "", "Subject")

	// msg unread
	unreadCmd := &cobra.Command{
		Use:   "unread [worker]",
		Short: "Show unread messages",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var worker *string
			if len(args) > 0 {
				worker = &args[0]
			}

			messages, err := database.GetUnreadMessages(worker)
			if err != nil {
				return err
			}

			if len(messages) == 0 {
				fmt.Println("No unread messages")
				return nil
			}

			for _, m := range messages {
				to := m.ToWorker
				if to == "" {
					to = "(broadcast)"
				}
				fmt.Printf("[%d] %s → %s (%s)\n", m.ID, m.FromWorker, to, m.CreatedAt.Format("15:04"))
				if m.Subject != "" {
					fmt.Printf("    Subject: %s\n", m.Subject)
				}
				fmt.Printf("    %s\n\n", m.Content)
			}
			return nil
		},
	}

	// msg read
	readCmd := &cobra.Command{
		Use:   "read <id|all>",
		Short: "Mark message(s) as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, _ := cmd.Flags().GetString("worker")

			if args[0] == "all" {
				if worker == "" {
					return fmt.Errorf("--worker is required for 'all'")
				}
				count, err := database.MarkAllRead(worker)
				if err != nil {
					return err
				}
				fmt.Printf("✓ Marked %d messages as read\n", count)
			} else {
				id, _ := strconv.Atoi(args[0])
				err := database.MarkMessageRead(id)
				if err != nil {
					return err
				}
				fmt.Printf("✓ Message #%d marked as read\n", id)
			}
			return nil
		},
	}
	readCmd.Flags().StringP("worker", "w", "", "Worker name (required for 'all')")

	cmd.AddCommand(sendCmd, broadcastCmd, unreadCmd, readCmd)
	return cmd
}

func lockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "File lock management",
	}

	// lock acquire (also available as just "lock <file>")
	acquireCmd := &cobra.Command{
		Use:   "acquire <file>",
		Short: "Acquire a file lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, _ := cmd.Flags().GetString("worker")
			reason, _ := cmd.Flags().GetString("reason")

			if worker == "" {
				return fmt.Errorf("--worker is required")
			}

			err := database.AcquireLock(args[0], worker, reason)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock acquired on '%s'\n", args[0])
			return nil
		},
	}
	acquireCmd.Flags().StringP("worker", "w", "", "Worker name (required)")
	acquireCmd.Flags().StringP("reason", "r", "", "Reason for lock")

	// lock release
	releaseCmd := &cobra.Command{
		Use:   "release <file>",
		Short: "Release a file lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			worker, _ := cmd.Flags().GetString("worker")
			if worker == "" {
				return fmt.Errorf("--worker is required")
			}

			err := database.ReleaseLock(args[0], worker)
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock released on '%s'\n", args[0])
			return nil
		},
	}
	releaseCmd.Flags().StringP("worker", "w", "", "Worker name (required)")

	// lock list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all locks",
		RunE: func(cmd *cobra.Command, args []string) error {
			locks, err := database.GetAllLocks()
			if err != nil {
				return err
			}

			if len(locks) == 0 {
				fmt.Println("No active locks")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "FILE\tLOCKED BY\tREASON\tSINCE")
			fmt.Fprintln(w, "----\t---------\t------\t-----")
			for _, l := range locks {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					l.FilePath, l.LockedBy, l.Reason, time.Since(l.LockedAt).Round(time.Second))
			}
			w.Flush()
			return nil
		},
	}

	cmd.AddCommand(acquireCmd, releaseCmd, listCmd)
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

			var eventTypePtr, workerPtr *string
			if eventType != "" {
				eventTypePtr = &eventType
			}
			if worker != "" {
				workerPtr = &worker
			}

			events, err := database.GetRecentEvents(limit, eventTypePtr, workerPtr)
			if err != nil {
				return err
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
	return cmd
}
