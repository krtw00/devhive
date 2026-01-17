package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iguchi/devhive/internal/db"
	"github.com/spf13/cobra"
)

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

	case "worker_session_changed":
		state := parseEventData(e.Data, "session_state")
		icon := sessionIcon(state)
		fmt.Printf("[%s] session: %s -> %s %s\n", timestamp, e.Worker, icon, state)

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
