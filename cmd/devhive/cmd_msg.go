package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

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
