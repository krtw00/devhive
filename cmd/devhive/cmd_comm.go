package main

import (
	"fmt"
	"os"

	"github.com/iguchi/devhive/internal/db"
	"github.com/spf13/cobra"
)

// requestCmd allows workers to request help/review from PM
func requestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request <type> [message]",
		Short: "Request help from PM",
		Long: `Send a request to the project manager.

Request types:
  help      - Need assistance with a problem
  review    - Code is ready for review
  unblock   - Blocked and need intervention
  clarify   - Need clarification on requirements

Examples:
  devhive request help "Stuck on authentication flow"
  devhive request review "API endpoints complete"
  devhive request unblock "Waiting for backend API"
  devhive request clarify "What format for date fields?"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqType := args[0]

			// Validate request type
			validTypes := map[string]string{
				"help":    "🆘 Help Request",
				"review":  "👀 Review Request",
				"unblock": "🚫 Unblock Request",
				"clarify": "❓ Clarification Request",
			}

			subject, ok := validTypes[reqType]
			if !ok {
				return fmt.Errorf("invalid request type: %s (valid: help, review, unblock, clarify)", reqType)
			}

			// Get message
			message := ""
			if len(args) > 1 {
				message = args[1]
			}

			// Get worker name
			workerName, err := getWorkerName([]string{}, 0)
			if err != nil {
				return fmt.Errorf("worker name required: set DEVHIVE_WORKER or use --worker flag")
			}

			// Send message to PM
			_, err = database.SendMessage(workerName, "pm", reqType, subject, message)
			if err != nil {
				return err
			}

			// If unblock, also update worker status
			if reqType == "unblock" {
				database.UpdateWorkerStatus(workerName, "blocked", nil)
			}

			fmt.Printf("✅ %s sent to PM\n", subject)
			if message != "" {
				fmt.Printf("   Message: %s\n", message)
			}

			return nil
		},
	}

	return cmd
}

// reportCmd allows workers to send progress reports
func reportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report <message>",
		Short: "Send progress report to PM",
		Long: `Send a progress report or update to the project manager.

Examples:
  devhive report "Completed login form, starting registration"
  devhive report "Found bug in payment flow, investigating"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := args[0]

			// Get worker name
			workerName, err := getWorkerName([]string{}, 0)
			if err != nil {
				return fmt.Errorf("worker name required: set DEVHIVE_WORKER or use --worker flag")
			}

			// Send message to PM
			_, err = database.SendMessage(workerName, "pm", "report", "📋 Progress Report", message)
			if err != nil {
				return err
			}

			fmt.Println("✅ Progress report sent to PM")

			return nil
		},
	}
}

// inboxCmd shows messages for PM
func inboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Show messages from workers (PM view)",
		Long: `Display unread messages from workers.

Examples:
  devhive inbox           # Show unread messages
  devhive inbox --all     # Show all messages`,
		RunE: func(cmd *cobra.Command, args []string) error {
			messages, err := database.GetUnreadMessages("pm")
			if err != nil {
				return err
			}

			all, _ := cmd.Flags().GetBool("all")
			if !all {
				// Filter to unread only
				var unread []db.Message
				for _, m := range messages {
					if m.ReadAt == nil {
						unread = append(unread, m)
					}
				}
				messages = unread
			}

			if len(messages) == 0 {
				fmt.Println("No messages.")
				return nil
			}

			fmt.Printf("=== Inbox (%d messages) ===\n\n", len(messages))

			for _, m := range messages {
				icon := getMessageIcon(m.MessageType)
				readStatus := ""
				if m.ReadAt == nil {
					readStatus = " [NEW]"
				}
				fmt.Printf("%s %s from %s%s\n", icon, m.Subject, m.FromWorker, readStatus)
				if m.Content != "" {
					fmt.Printf("   %s\n", m.Content)
				}
				fmt.Printf("   (%s)\n\n", m.CreatedAt.Format("01/02 15:04"))
			}

			// Mark as read option
			markRead, _ := cmd.Flags().GetBool("mark-read")
			if markRead && len(messages) > 0 {
				database.MarkAllRead("pm")
				fmt.Println("Marked all as read.")
			}

			return nil
		},
	}

	cmd.Flags().BoolP("all", "a", false, "Show all messages including read")
	cmd.Flags().BoolP("mark-read", "r", false, "Mark all messages as read")

	return cmd
}

func getMessageIcon(msgType string) string {
	icons := map[string]string{
		"help":    "🆘",
		"review":  "👀",
		"unblock": "🚫",
		"clarify": "❓",
		"report":  "📋",
		"info":    "ℹ️",
	}
	if icon, ok := icons[msgType]; ok {
		return icon
	}
	return "📨"
}

// replyCmd allows PM to reply to workers
func replyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reply <worker> <message>",
		Short: "Reply to a worker",
		Long: `Send a reply message to a worker.

Examples:
  devhive reply frontend "Approved, proceed with tests"
  devhive reply backend "Use ISO 8601 format for dates"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			toWorker := args[0]
			message := args[1]

			// Verify worker exists
			if _, err := database.GetWorker(toWorker); err != nil {
				return fmt.Errorf("worker not found: %s", toWorker)
			}

			// Send message from PM
			_, err := database.SendMessage("pm", toWorker, "reply", "💬 PM Reply", message)
			if err != nil {
				return err
			}

			fmt.Printf("✅ Reply sent to %s\n", toWorker)

			return nil
		},
	}
}

// msgsCmd shows messages for current worker
func msgsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "msgs",
		Short: "Show my messages (worker view)",
		Long: `Display messages for the current worker.

Examples:
  devhive msgs           # Show unread messages
  devhive msgs --all     # Show all messages`,
		RunE: func(cmd *cobra.Command, args []string) error {
			workerName := os.Getenv("DEVHIVE_WORKER")
			if workerName == "" {
				return fmt.Errorf("DEVHIVE_WORKER not set")
			}

			messages, err := database.GetUnreadMessages(workerName)
			if err != nil {
				return err
			}

			all, _ := cmd.Flags().GetBool("all")
			if !all {
				var unread []db.Message
				for _, m := range messages {
					if m.ReadAt == nil {
						unread = append(unread, m)
					}
				}
				messages = unread
			}

			if len(messages) == 0 {
				fmt.Println("No messages.")
				return nil
			}

			fmt.Printf("=== Messages for %s (%d) ===\n\n", workerName, len(messages))

			for _, m := range messages {
				readStatus := ""
				if m.ReadAt == nil {
					readStatus = " [NEW]"
				}
				fmt.Printf("💬 From %s%s\n", m.FromWorker, readStatus)
				fmt.Printf("   %s\n", m.Content)
				fmt.Printf("   (%s)\n\n", m.CreatedAt.Format("01/02 15:04"))
			}

			// Mark as read
			markRead, _ := cmd.Flags().GetBool("mark-read")
			if markRead {
				database.MarkAllRead(workerName)
				fmt.Println("Marked all as read.")
			}

			return nil
		},
	}

	cmd.Flags().BoolP("all", "a", false, "Show all messages including read")
	cmd.Flags().BoolP("mark-read", "r", false, "Mark all messages as read")

	return cmd
}

// broadcastCmd allows PM to send message to all workers
func broadcastCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "broadcast <message>",
		Short: "Send message to all workers",
		Long: `Broadcast a message to all active workers.

Examples:
  devhive broadcast "Sprint review at 3pm"
  devhive broadcast "Please rebase on latest main"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			message := args[0]

			count, err := database.BroadcastMessage("pm", "broadcast", "📢 Broadcast", message)
			if err != nil {
				return err
			}

			fmt.Printf("✅ Broadcast sent to %d worker(s)\n", count)

			return nil
		},
	}
}
